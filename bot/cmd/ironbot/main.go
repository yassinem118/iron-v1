package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
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
	authCollectionName    = "users_auth" // مخصص فقط لبيانات المصادقة وتسجيل الدخول
	dbTimeout             = 5 * time.Second
)

// authDocument مخصص لتخزين بيانات تسجيل الدخول والمصادقة للزبائن فقط في قاعدة البيانات
type authDocument struct {
	UserID       int64     `bson:"userId" json:"userId"`
	Username     string    `bson:"username,omitempty" json:"username"`
	SessionToken string    `bson:"sessionToken,omitempty" json:"sessionToken"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
}

type userSession struct {
	userID       int64
	store        *state.Store
	authCollection *mongo.Collection
}

func main() {
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

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Ping(pingCtx, readpref.Primary())
	pingCancel()
	if err != nil {
		disconnect()
		log.Fatalf("MongoDB ping failed: %v", err)
	}

	authCollection := client.Database(databaseName).Collection(authCollectionName)

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		disconnect()
		log.Fatalf("Telegram initialization failed: %v", err)
	}
	bot.Debug = false

	allowed := parseAllowlist(os.Getenv("ALLOWED_USER_IDS"))
	stores := newUserStores(authCollection)

	// بدء تشغيل خادم HTTP الخاص بالـ Dashboard والربط مع العميل (Dashboard API & Ingestion)
	go startDashboardServer(stores)

	updates := tgbotapi.NewUpdate(0)
	updates.Timeout = 30
	updateChan := bot.GetUpdatesChan(updates)

	sem := make(chan struct{}, maxConcurrentHandlers)
	var wg sync.WaitGroup

	log.Printf("Iron v1 Backend & Bot started as @%s; Auth persistence in MongoDB enabled", bot.Self.UserName)

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

				session := stores.For(m.From.ID, m.From.UserName)
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

// startDashboardServer تشغيل خادم HTTP لتزويد لوحة التحكم (Dashboard) بنتائج كل زبون معزولة
func startDashboardServer(stores *userStores) {
	mux := http.NewServeMux()

	// نقطة نهاية لجلب حالة اللعبة الخاصة بالزبون للـ Dashboard
	mux.HandleFunc("/api/dashboard/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userIDStr := r.URL.Query().Get("userId")
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil || userID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"valid userId is required"}`))
			return
		}

		session := stores.For(userID, "")
		snapshot, ok := session.store.Latest()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"no active game state found for user"}`))
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"userId":  userID,
			"gameId":  snapshot.GameID,
			"event":   snapshot.Event,
			"seenAt":  snapshot.SeenAt,
			"rows":    snapshot.Rows,
		})
	})

	// نقطة نهاية لاستقبال إطارات اللعبة المباشرة الموجهة لكل زبون (تحديث فوري للـ Dashboard)
	mux.HandleFunc("/api/dashboard/parse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			UserID int64  `json:"userId"`
			Raw    string `json:"raw"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 || req.Raw == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid payload or missing userId"}`))
			return
		}

		session := stores.For(req.UserID, "")
		snapshot, err := protocol.ParseFrame([]byte(req.Raw))
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		if err := session.store.Replace(snapshot); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		_, _ = w.Write([]byte(`{"success":true,"message":"frame updated for user successfully"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Dashboard HTTP server running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Dashboard server failed: %v", err)
	}
}

func handle(input string, session *userSession) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if strings.HasPrefix(input, "/start") || strings.HasPrefix(input, "/help") {
		return "Iron v1 — Isolated User Game Observer\n" +
			"/current: show your latest observed grid\n" +
			"/row N: show your explicitly observed cells in row N\n" +
			"/parse <JSON>: parse and load your frame"
	}

	if input == "/current" || strings.HasPrefix(input, "/current@") {
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

		return "Observed frame updated successfully for your session.\n" + renderSnapshot(session.store)
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

	row, ok := session.store.Row(rowNo)
	if !ok {
		return "No observed data for row " + strconv.Itoa(rowNo)
	}

	return renderRow(row)
}

func renderSnapshot(store *state.Store) string {
	snapshot, ok := store.Latest()
	if !ok {
		return "No observed game frame yet for your session."
	}

	var b strings.Builder
	b.WriteString("Your Latest observed grid (Isolated)\n")

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

	for _, row := range snapshot.Rows {
		b.WriteString(renderRow(row))
		b.WriteByte('\n')
	}

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

type userStores struct {
	mu             sync.Mutex
	byUser         map[int64]*userSession
	authCollection *mongo.Collection
}

func newUserStores(authCollection *mongo.Collection) *userStores {
     return &userStores{
         byUser:         make(map[int64]*userSession),
         authCollection: authCollection,
     }
}

// For تقوم بإنشاء أو جلب الجلسة المعزولة الخاصة بالزبون وتخزين بيانات المصادقة في القاعدة
func (s *userStores) For(userID int64, username string) *userSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.byUser[userID]
	if session == nil {
		session = &userSession{
			userID:         userID,
			store:          state.NewStore(),
			authCollection: s.authCollection,
		}
		s.byUser[userID] = session

		// حفظ بيانات المصادقة فقط في MongoDB بشكل غير متزامن
		go func(uid int64, uname string) {
			ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
			defer cancel()
			
			filter := bson.D{{Key: "userId", Value: uid}}
			update := bson.D{{Key: "$set", Value: bson.D{
				{Key: "username", Value: uname},
				{Key: "updatedAt", Value: time.Now().UTC()},
			}}}
			opts := options.Update().SetUpsert(true)
			_, _ = s.authCollection.UpdateOne(ctx, filter, update, opts)
		}(userID, username)
	}

	return session
}