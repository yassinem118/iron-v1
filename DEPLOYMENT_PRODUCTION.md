# Iron production deployment add-ons

These files add deployment infrastructure without changing existing business logic or UI. The supplied archive does **not** contain `mobile/android/`, so Android Gradle files below are examples to merge after generating the platform project with Flutter.

## 0. Prerequisites

- A Linux host with Docker Engine + Docker Compose plugin.
- A DNS A/AAAA record for `DOMAIN` pointing to the host; TCP 80 and 443 reachable.
- A production API domain you control and a verified official Melbet URL for the app's configured destination.
- Keep `.env.production`, private keys, signing files, and backups out of Git.

## 1. Create production environment

```sh
cp .env.production.example .env.production
chmod 600 .env.production
openssl rand -hex 32  # generate a unique value for each secret
```

Set `DOMAIN`, `JWT_SECRET`, `ADMIN_API_KEY`, Mongo root/app passwords, Telegram token, and numeric `ALLOWED_USER_IDS`. Use different random values for every secret. The example's placeholders are intentionally invalid.

Passwords use URL-safe hex secrets so the MongoDB URI does not need percent-encoding. If you use other characters, percent-encode username/password components before embedding them in a Mongo URI.

## 2. Build and issue Let's Encrypt certificate

The initial certificate must exist before the TLS Nginx container starts. Stop any other service binding port 80, then run the one-shot standalone Certbot container:

```sh
set -a
. ./.env.production
set +a
docker compose --env-file .env.production -f docker-compose.prod.yml --profile ops run --rm -p 80:80 certbot certonly --standalone \
  --email admin@${DOMAIN} --agree-tos --no-eff-email \
  -d "${DOMAIN}"
```

If Compose does not accept the one-off `-p` override in your version, temporarily add `ports: ["80:80"]` to the certbot service before running it, and remove that mapping after issuance. Use a real monitored mailbox for the certificate contact.

Then start the stack:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml logs --tail=100 backend nginx mongo bot
```

Check `https://DOMAIN/health`, admin UI at `https://DOMAIN/`, and backend logs. MongoDB is not published on the host and is reachable only on the private Docker network.

## 3. Renew certificate

The active Nginx config serves the ACME webroot challenge. Renewal uses the shared webroot volume:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml --profile ops run --rm certbot renew \
  --webroot -w /var/www/certbot
docker compose --env-file .env.production -f docker-compose.prod.yml exec nginx nginx -s reload
```

Schedule this command twice daily with systemd timer/cron and monitor renewal failures. The initial certificate is obtained with standalone; subsequent renewals use webroot. Ensure port 80 remains open for ACME challenge requests.

## 4. MongoDB details

- Mongo runs with authorization and no host-published port.
- The init script creates a database-scoped `readWrite` account for the backend.
- Mongo init scripts run only when the data directory is empty. Changing `.env.production` later does not rotate existing Mongo users; rotate credentials deliberately with `mongosh` and update the backend URI.
- Use encrypted off-host backups, test restore regularly, and restrict host SSH/firewall access. For higher assurance, use a managed MongoDB service with TLS and private networking.
- Compose secrets in `.env.production` are file-based environment injection, not a dedicated secrets manager. For production-critical workloads, use your platform's secret manager.

## 5. Backend / Nginx security notes

- The Node API is not exposed directly to the host; Nginx proxies `/api/` and `/health`.
- Nginx applies HTTPS, TLS 1.2+, HSTS, CSP, security headers, connection limits, and per-IP request limiting.
- The current CSP allows same-origin scripts and inline styles for compatibility with common React builds. Tighten it after verifying the built app; do not add `unsafe-eval`.
- Express already uses Helmet, JSON body limits, CORS, and route-level rate limits. If IP-based Express rate limiting is used behind Nginx, configure Express `trust proxy` narrowly for the actual proxy hop only; never trust arbitrary client-supplied `X-Forwarded-For`.
- Nginx request limiting is a basic abuse-control layer, not a volumetric DDoS mitigation service. Use a CDN/WAF or upstream provider protection for volumetric attacks.
- CORS is not authentication. Replace the shared `ADMIN_API_KEY` with SSO/MFA and role-based sessions before exposing an admin console publicly.
- Set `CORS_ORIGIN` to the exact admin origin. This stack serves admin and API on the same origin, so `/` is the Vite API base and no wildcard CORS is needed.
- Argon2id parameters are not automatically changed by environment variables unless application code reads them. Review the existing hash implementation and benchmark a target such as memory=64 MiB, iterations=3, parallelism=1 on production hardware.
- The backend should validate inputs using schemas, enforce account/license state on every protected request, return generic login errors, avoid logging credentials/tokens, and use constant-time comparisons for any manually checked shared secrets.

## 6. Flutter Android release signing

Generate the Android platform project if absent:

```sh
cd mobile
flutter create --platforms=android,ios .
```

Review the generated package/application ID before publishing. Create a release keystore (store it securely, outside source control):

```sh
keytool -genkeypair -v \
  -keystore "$HOME/iron-release.jks" \
  -alias iron-release \
  -keyalg RSA -keysize 4096 -validity 10000
```

Create `mobile/android/key.properties` (never commit it):

```properties
storePassword=YOUR_STORE_PASSWORD
keyPassword=YOUR_KEY_PASSWORD
keyAlias=iron-release
storeFile=/absolute/secure/path/iron-release.jks
```

Copy `mobile/android-signing.gradle.kts.example` into the generated `mobile/android/app/build.gradle.kts` by merging its `signingConfigs` and `buildTypes` sections into the existing `android {}` block. Copy `mobile/android-proguard-rules.pro.example` to `mobile/android/app/proguard-rules.pro`, then check that the generated Gradle file references it. Do not replace the generated Flutter plugin configuration.

Build:

```sh
flutter pub get
flutter test
flutter build apk --release \
  --dart-define=IRON_API_URL=https://iron.example.com \
  --dart-define=MELBET_URL=https://YOUR-VERIFIED-OFFICIAL-MELBET-URL/
```

Use the exact URL expected by the existing mobile implementation. If the code does not currently read `MELBET_URL` through `String.fromEnvironment`, this define has no effect until that existing URL configuration is wired to it. This deployment-only change intentionally does not rewrite app logic.

For Play Store delivery, consider `flutter build appbundle --release` and configure Play App Signing. Back up the keystore and passwords separately; losing the signing key can prevent updates to a sideloaded app.

## 7. Protect credentials during app launch

- Send only Iron credentials to the Iron HTTPS API. Never pass Iron credentials, JWTs, cookies, or authorization headers in a deep-link URL or Android intent extras.
- The launch intent should contain only the destination URL, and that URL should be a verified official URL. Let Android/iOS resolve the official universal/app link, or open the HTTPS website if the app is absent.
- Use certificate validation normally; do not disable TLS checks or bypass certificate pinning.
- Do not log login request bodies, passwords, bearer tokens, or deep-link query parameters.
- Validate ID length/character set and password length server-side; rate-limit login attempts and consider progressive lockouts and monitoring.
- Store any Iron session token using platform secure storage, set a short expiry, rotate/revoke refresh tokens, and avoid shared preferences for secrets.
- Android intent filters/app links must be verified against a domain you control. Do not claim a third-party app link is verified unless its asset links and app configuration support it.
- Keep the official destination URL free of user-specific tokens. Use HTTPS only.

## 8. R8 / reverse-engineering resistance

R8 minification and resource shrinking are enabled in the example release build. Obfuscation increases analysis cost but cannot make a client binary secret. Never embed server API keys, admin keys, database credentials, or signing passwords in Flutter assets or Dart defines. Dart `--obfuscate`/`--split-debug-info` can reduce symbol readability for supported release builds:

```sh
flutter build apk --release \
  --obfuscate --split-debug-info=build/symbols \
  --dart-define=IRON_API_URL=https://iron.example.com \
  --dart-define=MELBET_URL=https://YOUR-VERIFIED-OFFICIAL-MELBET-URL/
```

Securely archive the split-debug-info symbols for crash diagnosis. Test the release APK on real devices before distribution.

## 9. Alternative: systemd instead of Docker

The `deploy/systemd/` unit files are for bare-metal deployments only; do not run them alongside the Compose-managed copies. Create a dedicated `iron` user, install source under `/opt/iron`, store environment files under `/etc/iron/` with mode `0600`, then:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now iron-backend iron-bot
sudo systemctl status iron-backend iron-bot
```

Systemd unit paths and runtime user must match your host installation. Docker Compose restart policies are the intended process manager for the container deployment.

## 10. Release acceptance checklist

- [ ] `.env.production` is mode 600 and excluded from version control.
- [ ] DNS and ACME certificate issuance verified; renewal tested.
- [ ] Mongo has authentication, app least-privilege account, no public port, backup + restore test.
- [ ] API and admin are only reachable over HTTPS; admin access is protected with MFA/SSO or restricted network access.
- [ ] No secrets or credentials in logs, URLs, deep links, crash reports, or client bundle.
- [ ] Dependency scan, `npm test`, `go test ./...`, `flutter test`, and release-device smoke tests pass.
- [ ] APK signing key backup is tested and stored separately.
- [ ] Monitoring alerts for API health, container restarts, disk capacity, certificate expiry, backup failure, and authentication anomalies are enabled.
