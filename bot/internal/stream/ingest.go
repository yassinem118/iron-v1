package stream

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"iron-v1-bot/internal/protocol"
	"iron-v1-bot/internal/vision"
)

const maxIngestBytes = 64 * 1024
const maxImageRequestBytes = 6 << 20

type IngestEvent struct {
	Source  string          `json:"source,omitempty"`
	URL     string          `json:"url,omitempty"`
	Kind    string          `json:"kind,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

type IngestServer struct {
	Token   string
	Logger  *log.Logger
	OnFrame func([]byte) error
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
	mux.HandleFunc("/analyze-image", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorizedBearer(r, s.Token) {
			if s.Token == "" {
				http.Error(w, "image analysis authentication is not configured", http.StatusServiceUnavailable)
			} else {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxImageRequestBytes)
		var req vision.Request
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid image-analysis JSON", http.StatusBadRequest)
			return
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			http.Error(w, "request must contain exactly one JSON value", http.StatusBadRequest)
			return
		}
		imageBytes, err := base64.StdEncoding.DecodeString(req.ImageBase64)
		if err != nil {
			http.Error(w, "image_base64 must be valid standard base64", http.StatusBadRequest)
			return
		}
		result, err := vision.Analyze(imageBytes, req)
		if err != nil {
			http.Error(w, "image rejected: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
		// Never log the image, its base64 representation, or session identifiers.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
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
		if ev.Payload == nil || len(ev.Payload) == 0 {
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
			if err := s.OnFrame(ev.Payload); err != nil {
				http.Error(w, "frame rejected", http.StatusUnprocessableEntity)
				return
			}
		}
		if s.Logger != nil {
			s.Logger.Printf("ingested authorized frame kind=%q bytes=%d", safeKind(ev.Kind), len(ev.Payload))
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
