# Iron Backend — Trial and Activation additions

This update keeps the existing Express/Mongoose backend and adds a server-managed two-use trial plus single-use activation codes.

## Important routes

- `POST /api/auth/register` and `POST /api/register`: create an Iron account. Passwords are stored as Argon2id hashes.
- `POST /api/auth/login` and `POST /api/login`: authenticate an existing Iron account.
- `POST /api/auth/connect` and `POST /api/connect`: first connection creates an Iron account; later connections require the matching password.
- `GET /api/license/status`: authenticated user's license/trial status.
- `POST /api/license/trial/consume`: atomically consumes one of two trial uses. The server is the source of truth.
- `POST /api/license/activate` with `{ "code": "..." }`: redeems the user's single-use activation code.
- `GET /api/admin/licenses` with `x-admin-key`: lists license states without exposing activation-code hashes.
- `POST /api/admin/licenses/users/:userId/code` with `x-admin-key` and optional `{ "days": 15 }`: creates a new single-use code for an active Iron account. The plaintext code is returned only once; store/share it securely.
- Existing admin activation request approval/rejection routes remain available.

## Trial behavior

A new license begins with two trial uses. The server decrements the counter, and after the second use sets the license to `pending`. Further trial calls are denied until an admin issues a code and the customer redeems it. An active license is valid until `expiresAt`.

## Environment

Copy `.env` to `.env` for local development and replace all placeholders. Do not commit `.env` or share production secrets. The `ADMIN_API_KEY` protects admin routes; use HTTPS in production.

## Validation performed

- `node --check` on backend source files.
- `npm test` passed the existing 2 tests.

A live MongoDB integration test was not run in this workspace.
