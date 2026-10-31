# IRON deployment update — integration notes

This is an infrastructure/configuration update only. It is designed to sit beside the current `backend/`, `admin/`, `bot/`, and `mobile/` source directories; those source trees are not duplicated in this archive.

## What changed
- Compose builds the current Node backend and Go bot from their existing directories.
- The React/Vite admin is built into the Nginx image with `VITE_API_URL=/`, so it calls same-origin `/api/...` routes rather than a stale localhost or separate example API host.
- Nginx provides TLS termination, HTTP-to-HTTPS redirect, security headers, basic request/connection limits, `/api/` proxying and SPA fallback.
- MongoDB is not published to the host. A first-run init script creates a least-privilege application user.
- Existing `.env.production` values were retained as supplied. Review that file locally and rotate any credential that was real and previously shared. Do not commit it.

## Expected repository layout
```
.
├── admin/                 # current admin-updated source
├── backend/               # current backend-updated source
├── bot/                   # current Go bot source
├── mobile/                # current Flutter source
├── deploy/mongo-init/01-create-app-user.sh
├── nginx/Dockerfile
├── nginx/default.conf.template
├── docker-compose.prod.yml
└── .env.production
```

## Important compatibility notes
- The admin UI calls `/api/admin/users`, `/api/admin/licenses`, and `/api/admin/licenses/...`; keep those route paths aligned with the current backend.
- The compose `backend` service expects `GET /health` to return a success status.
- The bot container currently runs `cmd/ironbot` as specified by the supplied bot Dockerfile. The separate `cmd/ironstream` observer is not launched by this compose file; deploy it separately only if you have an authorized endpoint and a documented data schema.
- The current bot and backend packages are not changed by this archive. No claim is made that the bot sends observations into the backend automatically.
- `CORS_ORIGIN` is set to `https://${DOMAIN}` because the admin and API are served on the same origin.
- A Let’s Encrypt certificate for `${DOMAIN}` must exist before starting the HTTPS Nginx container. See `DEPLOYMENT_PRODUCTION.md`.
- This archive contains production configuration, not a complete source monorepo. Extract it into the repository root and merge/replace only the matching infrastructure files.

## Validation status
Archive/file consistency was checked. Container builds and end-to-end production startup were not run in this environment, so execute `docker compose --env-file .env.production -f docker-compose.prod.yml config` and the acceptance checklist in `DEPLOYMENT_PRODUCTION.md` on your deployment host.
