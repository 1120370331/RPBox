# Server-side SSL/TLS To-Do (PostgreSQL)

Scope: actions that must be done on the database server to enable SSL/TLS.

| ID | Task | Importance | Notes |
| --- | --- | --- | --- |
| DB-SSL-001 | Generate or obtain server certificate and private key | Critical | Use CA-signed cert in production; self-signed only for internal/dev. |
| DB-SSL-002 | Set key permissions and ownership | Critical | `server.key` must be 600 and owned by the Postgres user. |
| DB-SSL-003 | Enable SSL in `postgresql.conf` | Critical | Set `ssl = on`, `ssl_cert_file`, `ssl_key_file`. |
| DB-SSL-004 | Enforce SSL in `pg_hba.conf` | Critical | Use `hostssl` rules; avoid `host` rules that allow non-SSL. |
| DB-SSL-005 | Reload/restart PostgreSQL to apply changes | High | Use `pg_ctl reload` or service restart if needed. |
| DB-SSL-006 | Verify SSL is in use | High | Check `pg_stat_ssl` for active connections. |
| DB-SSL-007 | Record cert paths and rotation plan | Medium | Track expiration; schedule rotation before expiry. |
| DB-SSL-008 | Backup config files before changes | Medium | Keep copies of `postgresql.conf` and `pg_hba.conf`. |
| DB-SSL-009 | Validate firewall/NAT paths for SSL traffic | Medium | Ensure 5432 is reachable only from trusted networks. |
| DB-SSL-010 | Enable logging for SSL errors | Low | Helps diagnose client connection issues. |

# Server-side Security To-Do (CORS + Rate Limit)

Scope: actions that must be done on the production API server to enforce CORS and rate limits.

| ID | Task | Importance | Notes |
| --- | --- | --- | --- |
| API-CORS-001 | Update production CORS allowlist | Critical | Set `cors.allowed_origins` in `config.local.yaml` (or production config) to include `https://totalrpbox.com`, `https://www.totalrpbox.com`, `https://ksxvodevhonx.sealosbja.site`, `tauri://localhost`, `https://tauri.localhost`. |
| API-CORS-002 | Remove temporary domain after DNS cutover | Medium | Drop `https://ksxvodevhonx.sealosbja.site` once `totalrpbox.com` is live. |
| API-CORS-003 | Deploy config and restart API service | High | CORS config is loaded at startup; restart is required. |
| API-CORS-004 | Validate CORS behavior | High | From allowed origin, verify `Access-Control-Allow-Origin` and credentialed requests; from non-allowed origin, verify browser blocks. |
| API-RATE-001 | Set production rate limits | High | Tune `rate_limit.global/auth/api` in `config.local.yaml` based on expected traffic and auth abuse tolerance. |
| API-RATE-002 | Ensure real client IP is preserved | High | If behind a reverse proxy, confirm `X-Forwarded-For` / `X-Real-IP` are passed through; rate limiting depends on real client IP. |
| API-RATE-003 | Validate 429 responses | High | Use a small burst test on `/api/v1/auth/login` and general endpoints to confirm `429 Too Many Requests`. |
| API-RATE-004 | Monitor and adjust limits | Medium | Track 429 counts and adjust limits to reduce false positives. |

# Server-side HTTPS/Security To-Do (App Server)

Scope: actions that must be done on the production application server and reverse proxy to enforce HTTPS and security headers.

| ID | Task | Importance | Notes |
| --- | --- | --- | --- |
| APP-SEC-001 | Deploy backend build with S7/S8 fixes | Critical | Roll out new server binary/image and restart service. |
| APP-SEC-002 | Set server mode to release | Critical | Ensure `server.mode=release` so HTTPS redirect middleware is active. |
| APP-SEC-003 | Configure TLS termination and HTTP->HTTPS redirect | Critical | Use Nginx/Caddy; port 80 must 301 to 443. |
| APP-SEC-004 | Forward `X-Forwarded-Proto` and `Host` | High | Required for HTTPS detection and correct redirect URLs. |
| APP-SEC-005 | Install/renew CA-signed certificate | High | Automate renewal (e.g., certbot/ACME). |
| APP-SEC-006 | Validate security headers and redirect | High | `curl -I` and securityheaders.com should show HSTS and 301. |
| APP-SEC-007 | Restrict inbound ports | Medium | Allow 443 (and 80 for redirect); block direct access to app port. |

# Server-side Secrets To-Do (Production Config)

Scope: actions that must be done on the production server to remove plaintext secrets and use env/secret storage.

| ID | Task | Importance | Notes |
| --- | --- | --- | --- |
| APP-SECRET-001 | Provision production secrets in env/secret manager | Critical | Set `DATABASE_PASSWORD`, `JWT_SECRET`, `SMTP_PASSWORD`, `REDIS_PASSWORD`, and OSS keys if enabled (`OSS_ACCESS_KEY_ID`, `OSS_ACCESS_KEY_SECRET`). |
| APP-SECRET-002 | Create `config.local.yaml` or environment file | Critical | Keep it out of VCS; override only sensitive fields and production-only values. |
| APP-SECRET-003 | Remove plaintext secrets from deployed `config.yaml` | High | Ensure runtime values are injected via env or `config.local.yaml`. |
| APP-SECRET-004 | Lock down secret file permissions | High | `chmod 600` and owned by the service user. |
| APP-SECRET-005 | Rotate default/leaked credentials | High | Rotate DB, SMTP, JWT; expect user re-login after JWT rotation. |
| APP-SECRET-006 | Restart API service and verify config load | High | Confirm login, email sending, Redis connection, and DB auth succeed. |
