package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"iron-v1-bot/internal/grid"
	"iron-v1-bot/internal/protocol"
	"iron-v1-bot/internal/stream"
)

func main() {
	logger := log.New(os.Stdout, "iron-stream ", log.LstdFlags|log.LUTC)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("IRON_STREAM_MODE")))
	if mode == "websocket" {
		client := stream.Client{URL: os.Getenv("IRON_WS_URL"), RootCAFile: os.Getenv("IRON_ROOT_CA_FILE"), OnMessage: func(kind int, payload []byte) error {
			if kind != 1 {
				logger.Printf("ignored non-text frame type=%d bytes=%d", kind, len(payload))
				return nil
			}
			snapshot, err := protocol.ParseFrame(payload)
			if err != nil {
				logger.Printf("unrecognized text frame bytes=%d", len(payload))
				return nil
			}
			logger.Printf("observed frame event=%q rows=%d game_id_present=%t", snapshot.Event, len(snapshot.Rows), snapshot.GameID != "")
			return nil
		}}
		if err := client.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("client stopped: %v", err)
			os.Exit(1)
		}
		return
	}
	addr := strings.TrimSpace(os.Getenv("IRON_INGEST_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:8787"
	}
	token := strings.TrimSpace(os.Getenv("IRON_INGEST_TOKEN"))
	if token == "" {
		logger.Fatal("IRON_INGEST_TOKEN is required for ingest mode")
	}
	store := grid.NewStore()
	srv := &stream.IngestServer{Token: token, Logger: logger, OnFrame: func(raw []byte) error {
		snap, err := protocol.ParseFrame(raw)
		if err != nil {
			return err
		}
		return store.Replace(snap)
	}}
	httpSrv := &http.Server{Addr: addr, Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()
	logger.Printf("authorized-frame ingest listening on %s; payloads and secrets are not logged", addr)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal(err)
	}
}
