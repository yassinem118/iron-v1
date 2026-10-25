# Iron v1 Telegram bot (read-only observer)

This bot parses **operator-supplied JSON frames/logs** and reports only explicitly observed grid cells. It does not connect to Melbet, replay undocumented websocket messages, bypass platform controls, or infer safe gambling picks from hidden state.

## Run

```sh
cp .env.example .env
export TELEGRAM_BOT_TOKEN='...'
export ALLOWED_USER_IDS='123456789' # recommended
 go mod tidy
go test ./...
go run ./cmd/ironbot
```

## Commands

- `/start` or `/help` — help.
- `/current` — latest imported observation.
- `/row 3` or `3` — show source-reported values for row 3.
- `/parse {"event":"board:update","data":{"grid":[...]}}` — parse one JSON frame. Telegram message limits apply; for large logs, extract a single frame first.

The parser currently supports JSON objects with `rows` or `grid` arrays, directly or under up to three `data`, `payload`, or `result` object wrappers. Rows can contain `cells`, `items`, or `values`; cells can use `column`, `col`, or `index` and `status`, `type`, `value`, or `state`. Cells explicitly marked `revealed:false` are omitted. This is a deliberately conservative generic adapter, not a claim about Melbet's undocumented schema.

## Integration notes

The `application` archive referenced in the request was not present among the files available in this workspace; only the earlier `iron-v1.zip` was available. Therefore no truthful mapping to `socket3` or any other proprietary socket module can be asserted yet. Once authorized sample frames and the relevant source files are supplied, add a narrowly scoped adapter under `internal/protocol/` and tests for the exact observed event schema. Do not include account passwords, session cookies, authorization headers, or live tokens in samples.

## Iron live diagnostics module (authorized endpoints only)

The `bot/cmd/ironstream` command adds two explicit modes. It does **not** act as a transparent MITM proxy, bypass certificate pinning, harvest live credentials, or replay undocumented game actions.

### 1) Ingest authorized proxy-exported frames

Run from `bot/`:

```sh
export IRON_INGEST_ADDR=127.0.0.1:8787
export IRON_INGEST_TOKEN="$(openssl rand -hex 32)"
go run ./cmd/ironstream
```

Post a sanitized frame exported from a test system (never include cookies, Authorization headers, passwords, or tokens):

```sh
curl -sS http://127.0.0.1:8787/ingest \
  -H "Authorization: Bearer $IRON_INGEST_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"source":"local-test","kind":"websocket-text","payload":{"event":"board:update","data":{"grid":[{"row":1,"cells":[{"column":1,"status":"revealed"}]}]}}}'
```

`/health` is a liveness endpoint. `/ingest` requires a bearer secret, caps request bodies at 64 KiB, parses only supported JSON frames, and never logs the payload or source URL. Keep the listener on loopback; if remote access is required, use an authenticated TLS reverse proxy.

### 2) Observe an authorized WebSocket test endpoint

```sh
export IRON_STREAM_MODE=websocket
export IRON_WS_URL='wss://your-authorized-test-endpoint.example/ws'
# Optional: export IRON_ROOT_CA_FILE=/path/to/your-test-root-ca.pem
go run ./cmd/ironstream
```

The client uses normal TLS certificate and hostname verification, supports an additive PEM CA bundle, limits frame size, sends protocol-level Ping frames, observes Pong, and reconnects with exponential backoff. It only reports recognized JSON frame metadata; it does not log raw frames or auth headers and does not synthesize client actions. Do not point it at services without authorization.

### Build and test

```sh
cd bot
go mod tidy
go test ./...
go run ./cmd/ironbot       # Telegram observer
```

The new WebSocket package uses `github.com/gorilla/websocket`; a network-enabled Go environment is required to download dependencies on first build. `IRON_WS_URL` must be a test endpoint with a documented schema before a production adapter can be claimed.

## Live diagnostics

See the root README section **Iron live diagnostics module (authorized endpoints only)** for authenticated local frame ingestion and the TLS-verifying WebSocket observer in `cmd/ironstream`. It intentionally does not bypass certificate pinning or collect session secrets.


## Screenshot geometry analysis (Iron v1)

The authenticated ingest service also exposes `POST /analyze-image`. It accepts a JSON body containing a base64-encoded PNG/JPEG/GIF screenshot, a session ID, and an explicitly selected normalized board rectangle and grid dimensions. It validates the image payload, caps the decoded image at 4 MiB, limits image dimensions to 4096×4096 and the requested grid to 400 cells, and returns pixel coordinates for each cell.

Example request shape:

```json
{
  "session_id": "local-test-session",
  "image_base64": "<base64-encoded-authorized-screenshot>",
  "rows": 5,
  "columns": 5,
  "board": { "x": 0.1, "y": 0.2, "width": 0.8, "height": 0.6 }
}
```

Send it to the bot's configured listener with the same bearer token used for `/ingest`:

```sh
curl --fail-with-body http://127.0.0.1:8787/analyze-image \
  -H "Authorization: Bearer $IRON_INGEST_TOKEN" \
  -H "Content-Type: application/json" \
  --data @authorized-image-request.json
```

This initial implementation performs safe image validation and deterministic cell-coordinate mapping for a caller-supplied board region. It does **not** automatically discover the game board, classify Apple & Fortune symbols, or infer concealed outcomes. It returns `insufficient_evidence`, unknown cell states, and no recommendations until a labeled, authorized dataset and validated classifier/model are available. This is intentional; screenshot geometry is not evidence that a hidden cell is safe.

The screenshot endpoint is an internal bot interface; the current Flutter app/backend do not call it. Integrating a user-facing upload flow would require a separately reviewed backend/mobile contract, outside the permitted `bot/` scope.

## Session isolation

Telegram observations are now stored per Telegram sender ID, so one user's `/parse`, `/current`, and `/row` commands do not expose another user's latest snapshot. Configure `ALLOWED_USER_IDS` in production; do not leave the allowlist empty for a private deployment.

## Verification note

The standard-library image-analysis and protocol/grid packages can be tested without fetching third-party dependencies. Full `go test ./...`, `go vet ./...`, and `go build ./...` also require downloading the dependencies listed in `go.mod`; if the build environment has no network access and no module cache, those full-module checks cannot complete.
