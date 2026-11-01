package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"iron-v1-bot/internal/protocol"
	"iron-v1-bot/internal/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	maxConcurrentHandlers = 48
	databaseName          = "iron_v1_db"
	collectionName        = "telemetry"
	dbTimeout             = 5 * time.Second
)

type telemetryDocument struct {
	UserID     int64     `bson:"userId"`
	RawFrame   string    `bson:"rawFrame"`
	Event      string    `bson:"event,omitempty"`
	GameID     string    `bson:"gameId,omitempty"`
	ObservedAt time.Time `bson:"observedAt"`
	CreatedAt  time.Time `bson:"createdAt"`
}

type userSession struct {
	userID     int64
	store      *state.Store
	collection *mongo.Collection
}

func main() {
	// تحميل ملف .env افتراضياً دون إيقاف البرنامج إن لم يوجد، مع تسجيل تحذير عند حدوث خطأ آخر
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: error loading .env file: %v", err)
		}
	}

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	mongoURI := strings.TrimSpace(os.Getenv("MONGODB_URI"))
	if mongoURI == "" {
		log.Fatal("MONGODB_URI is required")
	}

	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatalf("MongoDB initialization failed: %v", err)
	}

	disconnect := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("MongoDB disconnect failed: %v", err)
		}
	}

	pingCtx, pingCancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	err = client.Ping(pingCtx, readpref.Primary())
	pingCancel()
	if err != nil {
		disconnect()
		log.Fatalf("MongoDB ping failed: %v", err)
	}

	collection := client.Database(databaseName).Collection(collectionName)

	// Optimize queries that retrieve the latest frame for a Telegram user.
	indexCtx, indexCancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	_, err = collection.Indexes().CreateOne(indexCtx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "createdAt", Value: -1},
		},
		Options: options.Index().SetName("user_latest_snapshot"),
	})
	indexCancel()
	if err != nil {
		disconnect()
		log.Fatalf("MongoDB index initialization failed: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		disconnect()
		log.Fatalf("Telegram initialization failed: %v", err)
	}
	bot.Debug = false

	allowed := parseAllowlist(os.Getenv("ALLOWED_USER_IDS"))
	stores := newUserStores(collection)

	updates := tgbotapi.NewUpdate(0)
	updates.Timeout = 30
	updateChan := bot.GetUpdatesChan(updates)

	sem := make(chan struct{}, maxConcurrentHandlers)
	var wg sync.WaitGroup

	log.Printf(
		"Iron v1 read-only observer started as @%s; MongoDB persistence enabled",
		bot.Self.UserName,
	)

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

				session := stores.For(m.From.ID)
				reply := handle(strings.TrimSpace(m.Text), session)
				if reply == "" {
					return
				}

				out := tgbotapi.NewMessage(m.Chat.ID, reply)
				out.ReplyToMessageID = m.MessageID
				out.AllowSendingWithoutReply = true

				if _, sendErr := bot.Send(out); sendErr != nil {
					log.Printf("Telegram send failed: %v", sendErr)
				}
			}(msg)

		default:
			_, _ = bot.Send(tgbotapi.NewMessage(
				msg.Chat.ID,
				"Iron v1 is busy; please retry shortly.",
			))
		}
	}

	bot.StopReceivingUpdates()
	wg.Wait()
	disconnect()
}

func handle(input string, session *userSession) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if strings.HasPrefix(input, "/start") ||
		strings.HasPrefix(input, "/help") {
		return "Iron v1 — read-only game-state observer\n" +
			"/current: show latest observed grid\n" +
			"/row N: show explicitly observed cells in row N\n" +
			"/parse <JSON>: parse and persist an authorized JSON frame/log\n\n" +
			"Unknown or unrevealed cells are never labelled safe, " +
			"and the bot does not predict gambling outcomes."
	}

	if input == "/current" || strings.HasPrefix(input, "/current@") {
		if err := session.refreshLatest(); err != nil {
			log.Printf("MongoDB current lookup failed for user %d: %v",
				session.userID, err)
			return "Could not load the latest frame from the database. Please retry."
		}
		return renderSnapshot(session.store)
	}

	if strings.HasPrefix(input, "/parse ") {
		raw := strings.TrimSpace(strings.TrimPrefix(input, "/parse "))
		if raw == "" {
			return "Usage: /parse <JSON>"
		}

		snapshot, err := protocol.ParseFrame([]byte(raw))
		if err != nil {
			return "Could not parse frame: " + err.Error()
		}

		if err := session.store.Replace(snapshot); err != nil {
			return "Frame rejected: " + err.Error()
		}

		doc := telemetryDocument{
			UserID:     session.userID,
			RawFrame:   raw,
			Event:      snapshot.Event,
			GameID:     snapshot.GameID,
			ObservedAt: snapshot.SeenAt,
			CreatedAt:  time.Now().UTC(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		_, err = session.collection.InsertOne(ctx, doc)
		cancel()
		if err != nil {
			log.Printf("MongoDB frame insert failed for user %d: %v",
				session.userID, err)
			return "Frame parsed, but database storage failed. " +
				"Please retry; the frame was not confirmed as saved."
		}

		return "Observed frame stored in MongoDB.\n" +
			renderSnapshot(session.store)
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

	if err := session.refreshLatest(); err != nil {
		log.Printf("MongoDB row lookup failed for user %d: %v",
			session.userID, err)
		return "Could not load the latest frame from the database. Please retry."
	}

	row, ok := session.store.Row(rowNo)
	if !ok {
		return "No observed data for row " + strconv.Itoa(rowNo) +
			". Import a JSON frame with /parse first."
	}

	return renderRow(row)
}

// refreshLatest reloads the most recently stored frame for this user.
// The original JSON is retained so the existing protocol parser remains
// the single source of truth for interpreting frames.
func (s *userSession) refreshLatest() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var doc telemetryDocument
	err := s.collection.FindOne(
		ctx,
		bson.D{{Key: "userId", Value: s.userID}},
		options.FindOne().SetSort(
			bson.D{{Key: "createdAt", Value: -1}},
		),
	).Decode(&doc)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return err
	}

	snapshot, err := protocol.ParseFrame([]byte(doc.RawFrame))
	if err != nil {
		return err
	}

	return s.store.Replace(snapshot)
}

func renderSnapshot(store *state.Store) string {
	snapshot, ok := store.Latest()
	if !ok {
		return "No observed game frame yet. Use /parse with an authorized " +
			"JSON frame/log; this bot does not connect to the betting service " +
			"or generate predictions."
	}

	var b strings.Builder
	b.WriteString("Latest observed grid (read-only)\n")

	if snapshot.Event != "" {
		b.WriteString("Event: ")
		b.WriteString(safeText(snapshot.Event))
		b.WriteByte('\n')
	}

	if snapshot.GameID != "" {
		b.WriteString("Game: ")
		b.WriteString(safeText(snapshot.GameID))
		b.WriteByte('\n')
	}

	b.WriteString("Observed: ")
	b.WriteString(snapshot.SeenAt.UTC().Format(time.RFC3339))
	b.WriteByte('\n')

	for _, row := range snapshot.Rows {
		b.WriteString(renderRow(row))
		b.WriteByte('\n')
	}

	b.WriteString(
		"Only source-reported statuses are shown; " +
			"blank/unrevealed cells are unknown, not safe picks.",
	)

	return strings.TrimSpace(b.String())
}

func renderRow(row protocol.Row) string {
	var b strings.Builder
	b.WriteString("Row ")
	b.WriteString(strconv.Itoa(row.Number))
	b.WriteString(": ")

	byCol := make(map[int]string, len(row.Cells))
	for _, cell := range row.Cells {
		byCol[cell.Column] = cell.Status
	}

	for col := 1; col <= state.Columns; col++ {
		if col > 1 {
			b.WriteString(" | ")
		}

		status, ok := byCol[col]
		if !ok {
			status = "unknown"
		}

		b.WriteString(strconv.Itoa(col))
		b.WriteByte('=')
		b.WriteString(safeText(status))
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

// userStores keeps a separate in-memory cache for each Telegram user.
// MongoDB documents are also scoped by userId.
type userStores struct {
	mu         sync.Mutex
	byUser     map[int64]*userSession
	collection *mongo.Collection
}

func newUserStores(collection *mongo.Collection) *userStores {
	return &userStores{
		byUser:     make(map[int64]*userSession),
		collection: collection,
	}
}

func (s *userStores) For(userID int64) *userSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.byUser[userID]
	if session == nil {
		session = &userSession{
			userID:     userID,
			store:      state.NewStore(),
			collection: s.collection,
		}
		s.byUser[userID] = session
	}

	return session
}