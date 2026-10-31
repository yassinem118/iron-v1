# Iron Bot — stream telemetry refactor

This repository is a read-only observer for JSON event frames that the operator is authorized to inspect. It does not act as a transparent network proxy, bypass TLS, extract session credentials, reverse-engineer private encryption, or infer hidden betting outcomes.

## Architecture

- `cmd/ironstream`: lifecycle-managed WebSocket reader or authenticated local ingest HTTP service.
- `cmd/ironbot`: Telegram interface with bounded concurrent update handlers.
- `internal/stream`: explicit-endpoint WebSocket client and bearer-authenticated ingest service.
- `internal/protocol`: bounded JSON parser for source-reported rows/cells and common event wrappers.
- `internal/state`: concurrent snapshot store with validation and defensive copies.

The `internal/vision` and `internal/grid` packages have been removed. Game observations are modeled as protocol data, not as a computer-vision/grid subsystem. The legacy `iron_phase1.js` is retained unchanged as requested; this refactor does not execute or extend any TLS interception or decryption behavior in that file.

## Run

Set `IRON_STREAM_MODE=websocket` and `IRON_WS_URL` only for an endpoint you own or are explicitly authorized to monitor. Standard TLS certificate and hostname verification remain enabled. Optional `IRON_ROOT_CA_FILE` adds a trusted CA; it does not disable verification.

For local operator-forwarded frames, set `IRON_INGEST_TOKEN` and optionally `IRON_INGEST_ADDR` (default `127.0.0.1:8787`). POST an authorized JSON event to `/ingest` with `Authorization: Bearer <token>` and a JSON body containing `payload`. `/health` is available for liveness checks. Do not bind ingest to a public interface without a protected reverse proxy and firewall.

For Telegram bot mode, set `TELEGRAM_BOT_TOKEN` and optionally `ALLOWED_USER_IDS` as comma-separated numeric IDs.

## Parser behavior

`internal/protocol.ParseFrame` only extracts explicit JSON values from recognized `rows`/`grid` arrays and common `data`, `payload`, or `result` wrappers. Unknown, binary, encrypted, or undocumented payloads are not guessed or decrypted. A cell with `revealed:false` is omitted. Missing values remain unknown; they are never labeled safe.

## Performance and limits

- JSON input is capped at 64 KiB; ingest request bodies are bounded.
- WebSocket read limits and reconnect backoff are configured in the client.
- Telegram handlers use a bounded semaphore rather than unbounded goroutine creation.
- No sub-millisecond latency guarantee is claimed without target-hardware benchmarks and representative authorized payloads.

## Verification

Run `go test ./...`, `go vet ./...`, and `go test -race ./...` in an environment with module dependencies available. This working environment could run `go test ./internal/protocol ./internal/state`; full verification requires downloading the Telegram Bot API and Gorilla WebSocket modules, which was unavailable offline.

## Secrets

`.env` is retained from the supplied source tree. Treat it as private and do not publish this archive. The Docker build excludes `.env` from its build context.

## Go-only performance layer (2026-10-29)

The runtime remains Go-only. The WebSocket transport uses Gorilla WebSocket
with normal TLS certificate/hostname verification; the parser uses the Go
standard library JSON decoder to avoid an extra parser dependency until
benchmarks prove a replacement is beneficial. `internal/stream.Pipeline`
provides a bounded worker pool, frame-size limits, backpressure, drain-on-close,
and atomic accepted/dropped/handled/failed counters. It accepts only frames
provided by configured/authorized sources; it is not a raw network sniffer or
TLS interception proxy.

Suggested benchmark gate before changing JSON libraries: add representative,
redacted payload fixtures and compare `encoding/json` against `json-iterator`
using `go test -bench=. -benchmem ./internal/protocol`. Keep the standard
library unless the measured throughput/allocation improvement is material and
compatibility tests pass. No universal `<1ms` latency guarantee is made; measure
p50/p95/p99 on the target host and payload sizes.

### Validation

Run `gofmt -w .`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and
`go test -bench=. -benchmem ./...` in an environment with the dependencies in
`go.mod` available. The current build environment had no network access to
fetch the existing Telegram and Gorilla modules, so full repository validation
could not be completed here.
