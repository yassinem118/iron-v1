# Iron Control Admin

## Development
Copy `.env.example` to `.env.local`, set `VITE_API_URL` to the backend base URL (for example `http://localhost:3000`), run `npm ci`, then `npm run dev`.

## Production
Set `VITE_API_URL` to the HTTPS backend URL and run `npm ci && npm run build`. Serve `dist/` behind HTTPS. Configure backend CORS to allow this admin origin and set the matching `ADMIN_API_KEY` on the server.

## Backend endpoints used
- `GET /api/admin/users?limit=100`
- `POST /api/admin/users`
- `PATCH /api/admin/users/:id`
- `PATCH /api/admin/users/:id/status`
- `GET /api/admin/licenses`
- `POST /api/admin/licenses/users/:userId/code`
- `GET /api/admin/licenses/requests`
- `POST /api/admin/licenses/requests/:id/approve`
- `POST /api/admin/licenses/requests/:id/reject`

All admin calls use the `x-admin-key` header. This panel manages Iron accounts and licenses only. Do not collect Melbet passwords in Iron; customers should authenticate directly in the official Melbet app. Iron may open a user-initiated deep link such as `melbet://app/` if the installed app supports it, but this panel cannot guarantee third-party app availability or deep-link behavior.
