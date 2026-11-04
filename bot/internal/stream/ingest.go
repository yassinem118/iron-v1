package stream

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"iron-v1-bot/internal/protocol"
)

const maxIngestBytes = 64 * 1024

type IngestEvent struct {
	UserID  int64           `json:"userId,omitempty"` // تم إضافة معرف الزبون لتوجيه الإطار لعزله
	Source  string          `json:"source,omitempty"`
	URL     string          `json:"url,omitempty"`
	Kind    string          `json:"kind,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

type IngestServer struct {
	Token   string
	Logger  *log.Logger
	OnFrame func(userID int64, payload []byte) error // تحديث الدالة لاستقبال userId مع الـ payload
}

// Handler accepts operator-forwarded, authorized frame samples. It is not a
// transparent proxy and deliberately does not capture credentials or traffic.
func (s *IngestServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.Token == "" {
			http.Error(w, "ingest authentication is not configured", http.StatusServiceUnavailable)
			return
		}
		if !authorizedBearer(r, s.Token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxIngestBytes)
		var ev IngestEvent
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&ev); err != nil {
			http.Error(w, "invalid JSON event", http.StatusBadRequest)
			return
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			http.Error(w, "request must contain exactly one JSON value", http.StatusBadRequest)
			return
		}
		if len(ev.Payload) == 0 {
			http.Error(w, "payload is required", http.StatusBadRequest)
			return
		}
		// Parse before forwarding, and never log the payload or source URL (which
		// may contain session material).
		if _, err := protocol.ParseFrame(ev.Payload); err != nil {
			http.Error(w, "payload is not a supported observed JSON frame", http.StatusUnprocessableEntity)
			return
		}
		if s.OnFrame != nil {
			if err := s.OnFrame(ev.UserID, ev.Payload); err != nil {
				http.Error(w, "frame rejected", http.StatusUnprocessableEntity)
				return
			}
		}
		if s.Logger != nil {
			s.Logger.Printf("ingested authorized frame userId=%d kind=%q bytes=%d", ev.UserID, safeKind(ev.Kind), len(ev.Payload))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"accepted":true}`)
	})
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
func safeKind(v string) string {
	v = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return -1
		}
		return r
	}, v)
	if len(v) > 40 {
		return v[:40]
	}
	return v
}

func authorizedBearer(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	got := strings.TrimPrefix(header, prefix)
	return len(got) == len(token) && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
 }