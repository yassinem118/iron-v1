# IRON Mobile — Dashboard update

This update preserves the existing Flutter project and adds a live rounds dashboard after a successful IRON login.

## Configuration

Configure endpoints at build time; do not ship an unencrypted HTTP backend in production.

```bash
flutter build apk --release \
  --dart-define=IRON_API_URL=https://YOUR_API_HOST \
  --dart-define=IRON_ROUNDS_PATH=/api/rounds \
  --dart-define=MELBET_URL=https://melbet.com/
```

The dashboard polls `GET /api/rounds` every five seconds. The response may be either a JSON list or an object with `rounds`, `results`, or `data` containing a list. Each record should contain `round` (or `roundNumber`), `row` (or `level`), `cell` (or `column`), and optionally `createdAt` (ISO-8601), `id`, and `status`. Example record for integration testing only:

```json
{ "round": 1, "row": 1, "cell": 3, "createdAt": "2026-10-31T14:00:00Z", "status": "confirmed" }
```

The example is schema documentation, not a real game result. The backend must authenticate access and return only confirmed, authorized records. If the backend route or response shape differs, adjust `IRON_ROUNDS_PATH` or the parser in `lib/screens/rounds_dashboard.dart`.

## Notes

- On successful IRON login, the dashboard opens and attempts to launch the configured Melbet destination. Returning to IRON refreshes the round list.
- The dashboard displays server-returned records; it does not predict hidden cells.
- The previous login API URL is retained for compatibility with the current project. Verify it and move it to a build-time configuration before production.
- Use HTTPS and ensure the backend does not expose records across different customers.
