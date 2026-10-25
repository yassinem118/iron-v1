# Iron v1 starter monorepo

This is a runnable starter scaffold, not a production certification. It intentionally handles **Iron v1 account credentials only**; it must not collect Melbet passwords or expose passwords in an admin table.

## Backend
1. Install Node.js 20+ and MongoDB.
2. `cd backend && cp .env.example .env`
3. Replace `JWT_SECRET` and `ADMIN_API_KEY` with long random secrets.
4. `npm install && npm run dev`
5. `GET http://localhost:8080/health`
6. Register with `POST /api/auth/register` JSON `{ "accountId":"demo_user", "password":"a-long-password-123" }`.

Admin requests require `x-admin-key: <ADMIN_API_KEY>`. Routes: `GET /api/admin/users`, `PATCH /api/admin/users/:id/status` with `{ "active": false }`.

## Admin UI
`cd admin && npm install && npm run dev`. Enter the admin API key in the dashboard. For deployment, serve behind HTTPS, set `VITE_API_URL`, and do not embed secrets in frontend build variables.

## Flutter
`cd mobile && flutter pub get && flutter run --dart-define=IRON_API_URL=http://10.0.2.2:8080` for Android emulator local development. Replace with HTTPS in deployment. The starter sign-in call does not yet retain a session or implement registration; complete session handling with a cookie-aware HTTP client or short-lived access token stored in platform secure storage before production.

## Telegram bot
`cd bot`, set `TELEGRAM_BOT_TOKEN` and optionally `ALLOWED_USER_IDS`, then `go mod tidy && go run ./cmd/ironbot`. The grid values in `main.go` are explicitly labelled demo mappings and are not predictions. Configure only values you have a legitimate, verified basis to use.

## Before production
- Use HTTPS end-to-end, secrets manager, managed MongoDB with TLS, least-privilege database user, backups, and network restrictions.
- Replace shared admin API key with OIDC/SSO, MFA, roles, and server-side session management.
- Add CSRF protections if using cookies across origins, password reset and email verification, account lockout policy, monitoring, structured logs, tests, CI, dependency scanning, and deployment health checks.
- Configure reverse-proxy trusted IP handling before enabling IP-based rate limits.
- Never log credentials, store third-party passwords, or display password hashes in the UI.

## Iron live diagnostics module (authorized endpoints only)

The `bot/cmd/ironstream` command adds two explicit modes. It is not a transparent MITM proxy and does not bypass certificate pinning, harvest live credentials, or replay undocumented game actions.

### Authenticated local frame ingestion

Run from `bot/` with a strong random secret:

```sh
export IRON_INGEST_ADDR=127.0.0.1:8787
export IRON_INGEST_TOKEN="$(openssl rand -hex 32)"
go run ./cmd/ironstream

POST sanitized, authorized JSON frames to /ingest with Authorization: Bearer $IRON_INGEST_TOKEN. The handler caps bodies at 64 KiB, validates supported frames, and never logs the raw payload or source URL. Keep it on loopback; use an authenticated TLS reverse proxy for remote access.

WebSocket observer
Bash
export IRON_STREAM_MODE=websocket
export IRON_WS_URL='wss://your-authorized-test-endpoint.example/ws'
# Optional additive PEM root CA bundle:
export IRON_ROOT_CA_FILE=/path/to/test-root-ca.pem
go run ./cmd/ironstream
The client retains normal TLS certificate and hostname verification, supports an additive CA bundle, caps frame size, sends Ping frames, handles Pong deadlines, and retries with exponential backoff. It logs only recognized frame metadata, never raw payloads or credentials. It does not generate actions or infer undocumented schemas.

Run go mod tidy, go test ./..., and go run ./cmd/ironbot in a network-enabled Go environment. First build requires downloading Go modules. A production-specific adapter requires authorized sample frames and a documented schema.

Production deployment
See DEPLOYMENT_PRODUCTION.md for the additive Docker Compose, Nginx/Let's Encrypt, MongoDB authorization, systemd alternative, and Android release-signing configuration. Copy .env.production.example to .env.production; never commit real secrets.