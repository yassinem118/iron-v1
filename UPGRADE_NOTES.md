# Iron Core Upgrade Notes

This archive preserves the original `admin/`, `backend/`, `mobile/`, and `bot/` layout and adds a backend `src/core/brainai/` module.

## Added backend modules

- `src/models/license.js`: trial and subscription state.
- `src/models/activationRequest.js`: renewal/activation queue.
- `src/middleware/license.js`: server-side active-license gate.
- `src/routes/license.js`: status, consume one trial attempt, and request activation.
- `src/routes/adminLicenses.js`: admin review queue and explicit approval/rejection.
- `src/models/outcome.js` and `src/core/brainai/calibrator.js`: user-scoped historical calibration.
- `src/routes/brainai.js`: authenticated endpoints for recording explicitly observed outcomes and viewing calibration.

## New endpoints

- `GET /api/license/status`
- `POST /api/license/trial/consume` — each successful trial call decrements the server-side counter; two total.
- `POST /api/license/activation/request`
- `GET /api/brainai/summary` — requires an active license.
- `POST /api/brainai/observations` — requires an active license; accepts only reported probability plus an already observed success/failure.
- `POST /api/brainai/calibrate` — returns historical calibration metrics.
- `GET /api/admin/licenses/requests` — requires `x-admin-key`.
- `POST /api/admin/licenses/requests/:id/approve` with `{"days":15}` — manually approve after separately verifying payment.
- `POST /api/admin/licenses/requests/:id/reject`

## Run and test

```sh
cd backend
cp .env.example .env
# Set unique random JWT_SECRET and ADMIN_API_KEY values.
npm install
npm test
npm run dev
```

The license record is created lazily on the first `/api/license/status` or trial call. Approval currently creates a 15-day subscription measured from the approval time. For a production billing system, add idempotent payment reconciliation, a durable outbox for notifications, admin identity/MFA and role permissions, and transactional updates for request/license transitions.

## Security and scope notes

- The supplied third-party authorization-like string is deliberately not embedded. Secrets should be rotated if they were real and exposed in a chat, repository, or log.
- The application must use Iron account credentials only. The login bridge does not collect or relay Melbet credentials.
- The existing Go WebSocket client keeps certificate and hostname verification enabled. Optional custom CA roots are additive trust, not a pinning bypass.
- Proxy configuration should be restricted to a developer-controlled test environment; don't log authorization headers, cookies, tokens, or raw session frames.
- `brainai` calibrates outcomes that have already been observed. It does not claim to predict hidden bomb/apple locations.
- Trial consumption is an API primitive. A production UI must call it immediately before each trial-only analysis action and must never decide access based solely on a local counter.
- This is a scaffold upgrade, not a security certification. Run tests, threat-model deployment, configure TLS, secrets management, backups, audit retention, CSRF/session protections, and external security review before production.
