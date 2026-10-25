package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"iron-v1-bot/internal/grid"
	"iron-v1-bot/internal/protocol"
)

const maxConcurrentHandlers = 48

func main() {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("telegram initialization failed: %v", err)
	}
	bot.Debug = false

	allowed := parseAllowlist(os.Getenv("ALLOWED_USER_IDS"))
	stores := newUserStores()
	updates := tgbotapi.NewUpdate(0)
	updates.Timeout = 30
	updateChan := bot.GetUpdatesChan(updates)
	sem := make(chan struct{}, maxConcurrentHandlers)
	var wg sync.WaitGroup
	log.Printf("Iron v1 read-only grid observer started as @%s", bot.Self.UserName)

	for update := range updateChan {
		if update.Message == nil || update.Message.From == nil {
			continue
		}
		msg := update.Message
		if len(allowed) > 0 && !allowed[msg.From.ID] {
			continue
		}
		select {
		case sem <- struct{}{}:
			wg.Add(1)
			go func(m *tgbotapi.Message) {
				defer wg.Done()
				defer func() { <-sem }()
				ctxText := strings.TrimSpace(m.Text)
				reply := handle(ctxText, stores.For(m.From.ID))
				if reply == "" {
					return
				}
				out := tgbotapi.NewMessage(m.Chat.ID, reply)
				out.ReplyToMessageID = m.MessageID
				out.AllowSendingWithoutReply = true
				if _, sendErr := bot.Send(out); sendErr != nil {
					log.Printf("telegram send failed: %v", sendErr)
				}
			}(msg)
		default:
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Iron v1 is busy; please retry shortly."))
		}
	}
	wg.Wait()
}

func handle(input string, store *grid.Store) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	if strings.HasPrefix(input, "/start") || strings.HasPrefix(input, "/help") {
		return "Iron v1 — read-only game-state observer\n/current: show latest observed grid\n/row N: show explicitly observed cells in row N\n/parse <JSON>: parse a JSON websocket frame/log you are authorized to inspect\n\nUnknown or unrevealed cells are never labelled safe, and the bot does not predict gambling outcomes."
	}
	if input == "/current" || strings.HasPrefix(input, "/current@") {
		return renderSnapshot(store)
	}
	if strings.HasPrefix(input, "/parse ") {
		raw := strings.TrimSpace(strings.TrimPrefix(input, "/parse "))
		snapshot, err := protocol.ParseFrame([]byte(raw))
		if err != nil {
			return "Could not parse frame: " + err.Error()
		}
		if err := store.Replace(snapshot); err != nil {
			return "Frame rejected: " + err.Error()
		}
		return "Observed frame stored.\n" + renderSnapshot(store)
	}
	if strings.HasPrefix(input, "/row ") {
		input = strings.TrimSpace(strings.TrimPrefix(input, "/row "))
	}
	if strings.HasPrefix(input, "/") {
		return "Unknown command. Use /help."
	}
	rowNo, err := strconv.Atoi(input)
	if err != nil || rowNo < 1 || rowNo > 100 {
		return "Use /row N where N is between 1 and 100."
	}
	row, ok := store.Row(rowNo)
	if !ok {
		return "No observed data for row " + strconv.Itoa(rowNo) + ". Import a JSON frame with /parse first."
	}
	return renderRow(row)
}

func renderSnapshot(store *grid.Store) string {
	snapshot, ok := store.Latest()
	if !ok {
		return "No observed game frame yet. Use /parse with an authorized JSON frame/log; this bot does not connect to the betting service or generate predictions."
	}
	var b strings.Builder
	b.WriteString("Latest observed grid (read-only)\n")
	if snapshot.Event != "" {
		b.WriteString("Event: " + safeText(snapshot.Event) + "\n")
	}
	if snapshot.GameID != "" {
		b.WriteString("Game: " + safeText(snapshot.GameID) + "\n")
	}
	b.WriteString("Observed: " + snapshot.SeenAt.UTC().Format(time.RFC3339) + "\n")
	for _, row := range snapshot.Rows {
		b.WriteString(renderRow(row) + "\n")
	}
	b.WriteString("Only source-reported statuses are shown; blank/unrevealed cells are unknown, not safe picks.")
	return strings.TrimSpace(b.String())
}

func renderRow(row grid.Row) string {
	var b strings.Builder
	b.WriteString("Row " + strconv.Itoa(row.Number) + ": ")
	byCol := make(map[int]string, len(row.Cells))
	for _, cell := range row.Cells {
		byCol[cell.Column] = cell.Status
	}
	for col := 1; col <= grid.Columns; col++ {
		if col > 1 {
			b.WriteString(" | ")
		}
		status, ok := byCol[col]
		if !ok {
			status = "unknown"
		}
		b.WriteString(strconv.Itoa(col) + "=" + safeText(status))
	}
	return b.String()
}

func safeText(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > 80 {
		return s[:80]
	}
	return s
}

func parseAllowlist(raw string) map[int64]bool {
	result := map[int64]bool{}
	for _, item := range strings.Split(raw, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(item), 10, 64)
		if err == nil && id > 0 {
			result[id] = true
		}
	}
	return result
}

// userStores isolates imported observations so one Telegram account cannot read
// another account's latest game snapshot.
type userStores struct {
	mu     sync.Mutex
	byUser map[int64]*grid.Store
}

func newUserStores() *userStores {
	return &userStores{byUser: make(map[int64]*grid.Store)}
}

func (s *userStores) For(userID int64) *grid.Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	store := s.byUser[userID]
	if store == nil {
		store = grid.NewStore()
		s.byUser[userID] = store
	}
	return store
}
