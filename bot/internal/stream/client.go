package stream

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

// Client connects only to an explicitly configured endpoint. TLS certificate
// and hostname verification remain enabled. Custom roots are additive trust,
// not a pinning bypass.
type Client struct {
	URL          string
	RootCAFile   string
	Subprotocols []string
	DialTimeout  time.Duration
	ReadLimit    int64
	OnMessage    func(messageType int, payload []byte) error
}

func (c Client) Run(ctx context.Context) error {
	if c.URL == "" {
		return errors.New("IRON_WS_URL is required")
	}
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := c.runOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "websocket session ended: %v; retry in %s\n", err, backoff)
		}
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (c Client) runOnce(ctx context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: c.DialTimeout, Subprotocols: c.Subprotocols, EnableCompression: false}
	if dialer.HandshakeTimeout == 0 {
		dialer.HandshakeTimeout = 10 * time.Second
	}
	if len(c.RootCAFile) > 0 {
		pem, err := os.ReadFile(c.RootCAFile)
		if err != nil {
			return fmt.Errorf("read custom CA: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return errors.New("custom CA file contains no valid certificates")
		}
		dialer.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12} // #nosec G402: verification remains enabled
	}
	conn, resp, err := dialer.DialContext(ctx, c.URL, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket handshake HTTP %d", resp.StatusCode)
		}
		return err
	}
	defer conn.Close()
	limit := c.ReadLimit
	if limit <= 0 {
		limit = 64 * 1024
	}
	conn.SetReadLimit(limit)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutdown"), time.Now().Add(time.Second))
				return
			case <-done:
				return
			case <-ticker.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			}
		}
	}()
	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if c.OnMessage != nil {
			if err := c.OnMessage(kind, data); err != nil {
				return fmt.Errorf("message rejected: %w", err)
			}
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
}
