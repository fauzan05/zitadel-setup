# Catatan Setup Zitadel (Docker Compose)

## Apa itu Zitadel?

**Zitadel** adalah platform **Identity & Access Management (IAM)** open-source — mirip Auth0, Keycloak, atau Firebase Auth. Fungsinya:

- Login / Register user (OAuth2, OIDC, SAML)
- Manajemen user, roles, dan permissions
- Multi-tenant (organisasi)
- Single Sign-On (SSO)
- API authentication (service accounts, personal access tokens)

---

## Arsitektur Stack

```
User → Traefik (reverse proxy, port 8080)
         ├── /                    → zitadel-login (halaman login)
         ├── /ui/v2/login/*       → zitadel-login
         ├── /api/*               → zitadel-api (strip prefix /api)
         └── path lainnya         → zitadel-api (admin console, OIDC, dll)
```

### Daftar Service

| Service            | Image                              | Fungsi                                                        |
| ------------------ | ---------------------------------- | ------------------------------------------------------------- |
| **proxy**          | Traefik                            | Reverse proxy — menerima traffic HTTP dan route ke service     |
| **zitadel-api**    | ghcr.io/zitadel/zitadel            | Backend utama — API, gRPC, admin console, OIDC endpoints      |
| **zitadel-login**  | ghcr.io/zitadel/zitadel-login      | UI login v2 (Next.js) — halaman login/register yang user lihat |
| **postgres**       | PostgreSQL                         | Database utama Zitadel                                        |
| **redis**          | Redis (opsional, profile `cache`)  | Cache untuk mempercepat performa                              |
| **otel-collector** | OTEL Collector (opsional, profile `observability`) | Tracing/monitoring                          |

---

## Penjelasan .env

### Domain & URL

```env
ZITADEL_DOMAIN=localhost          # Domain yang diakses user
PROXY_HTTP_PUBLISHED_PORT=8080    # Port yang di-expose ke host machine
ZITADEL_EXTERNALPORT=8080         # Port yang "dilihat" user dari luar
ZITADEL_EXTERNALSECURE=false      # false = HTTP, true = HTTPS
ZITADEL_PUBLIC_SCHEME=http        # http atau https
```

### Security / Bootstrap

```env
ZITADEL_MASTERKEY=MasterkeyNeedsToHave32Characters  # Encryption key (HARUS tepat 32 karakter!)
LOGIN_CLIENT_PAT_EXPIRATION=2099-01-01T00:00:00Z     # Masa berlaku token login-client internal
```

> **PENTING:** `ZITADEL_MASTERKEY` digunakan untuk mengenkripsi data sensitif di database. Sekali di-set dan data sudah ada, **JANGAN diubah** karena data lama tidak bisa di-decrypt.

### Pinned Image Tags

```env
ZITADEL_VERSION=v4.11.0                        # Versi Zitadel (API + Login)
TRAEFIK_IMAGE=traefik:v3.6.8                    # Versi Traefik proxy
POSTGRES_IMAGE=postgres:17.2-alpine             # Versi PostgreSQL
REDIS_IMAGE=redis:7.4.2-alpine                  # Versi Redis
OTEL_COLLECTOR_IMAGE=otel/opentelemetry-collector-contrib:0.114.0
```

### Proxy Settings (Traefik)

```env
TRAEFIK_DASHBOARD_ENABLED=false       # Traefik dashboard (false untuk production)
TRAEFIK_LOG_LEVEL=INFO                # Level log: DEBUG, INFO, WARN, ERROR
TRAEFIK_ACCESSLOG_ENABLED=true        # Log setiap request masuk
TRAEFIK_TRUSTED_IPS=10.0.0.0/8,...    # IP range yang dipercaya (untuk X-Forwarded-* headers)
LETSENCRYPT_EMAIL=ops@example.com     # Email untuk sertifikat Let's Encrypt
```

### Database (PostgreSQL)

```env
POSTGRES_DB=zitadel                    # Nama database
POSTGRES_ADMIN_USER=postgres           # User admin postgres
POSTGRES_ADMIN_PASSWORD=postgres       # Password admin ⚠️ GANTI untuk production!
POSTGRES_ZITADEL_USER=zitadel         # User khusus untuk app Zitadel
POSTGRES_ZITADEL_PASSWORD=zitadel     # Password user Zitadel ⚠️ GANTI untuk production!
```

### Redis Cache (Opsional)

```env
ZITADEL_CACHES_CONNECTORS_REDIS_ENABLED=false   # false = tidak pakai Redis
ZITADEL_CACHES_CONNECTORS_REDIS_ADDR=redis:6379
ZITADEL_CACHES_INSTANCE_CONNECTOR=               # Kosong = default (tanpa cache)
ZITADEL_CACHES_MILESTONES_CONNECTOR=
ZITADEL_CACHES_ORGANIZATION_CONNECTOR=
```

Untuk mengaktifkan Redis, ubah `ENABLED=true`, isi connector dengan `redis`, dan jalankan dengan `--profile cache`.

### OTEL Tracing (Opsional)

```env
ZITADEL_INSTRUMENTATION_TRACE_EXPORTER_TYPE=none   # none = mati, otel = aktif
ZITADEL_INSTRUMENTATION_TRACE_EXPORTER_ENDPOINT=otel-collector:4317
```

Untuk mengaktifkan, ubah `TYPE=otel` dan jalankan dengan `--profile observability`.

---

## Setup untuk LOCAL Development

### Prasyarat

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) terinstal dan running
- Port 8080 tidak dipakai aplikasi lain

### Langkah

```powershell
# 1. Masuk ke folder project
cd c:\Users\fauza\Documents\Coding\zitadel

# 2. Jalankan semua service
docker compose up -d --wait

# 3. Buka browser
#    http://localhost:8080
```

`.env` default sudah siap pakai untuk lokal — **tidak perlu diubah apapun**.

### Login Pertama Kali

Saat `start-from-init`, Zitadel membuat instance pertama beserta admin user. Cek log untuk melihat credential:

```powershell
docker compose logs zitadel-api | Select-String -Pattern "username|password"
```

Atau cek di admin console: `http://localhost:8080/ui/console`

---

## Setup untuk SERVER (Production)

### Yang WAJIB Diubah

| Variable                     | Nilai Lokal        | Ubah ke (Production)                    |
| ---------------------------- | ------------------ | --------------------------------------- |
| `ZITADEL_DOMAIN`             | `localhost`        | Domain asli, misal `auth.example.com`   |
| `ZITADEL_MASTERKEY`          | `Masterkey...`     | String random tepat 32 karakter         |
| `ZITADEL_EXTERNALSECURE`     | `false`            | `true`                                  |
| `ZITADEL_PUBLIC_SCHEME`      | `http`             | `https`                                 |
| `ZITADEL_EXTERNALPORT`       | `8080`             | `443`                                   |
| `PROXY_HTTP_PUBLISHED_PORT`  | `8080`             | `80` atau `443`                         |
| `POSTGRES_ADMIN_PASSWORD`    | `postgres`         | Password kuat & unik                    |
| `POSTGRES_ZITADEL_PASSWORD`  | `zitadel`          | Password kuat & unik                    |
| `LETSENCRYPT_EMAIL`          | `ops@example.com`  | Email asli kamu                         |

### Contoh .env Production

```env
ZITADEL_DOMAIN=auth.yourdomain.com
PROXY_HTTP_PUBLISHED_PORT=80
ZITADEL_EXTERNALPORT=443
ZITADEL_EXTERNALSECURE=true
ZITADEL_PUBLIC_SCHEME=https
ZITADEL_MASTERKEY=abcdefghij1234567890abcdefghij12
POSTGRES_ADMIN_PASSWORD=SuperSecurePassword123!
POSTGRES_ZITADEL_PASSWORD=AnotherSecurePass456!
LETSENCRYPT_EMAIL=admin@yourdomain.com
```

### Catatan TLS/HTTPS

Untuk HTTPS di production, kamu perlu salah satu:

1. **TLS overlay compose** — file tambahan seperti `docker-compose.tls.yml` yang mengkonfigurasi Traefik dengan Let's Encrypt
2. **External reverse proxy** — Nginx/Caddy/Cloudflare di depan Traefik yang handle SSL termination

---

## Command Penting

### Operasi Dasar

```powershell
# Jalankan stack
docker compose up -d --wait

# Lihat status semua service
docker compose ps

# Lihat log (semua)
docker compose logs -f

# Lihat log service tertentu
docker compose logs -f zitadel-api
docker compose logs -f zitadel-login
docker compose logs -f postgres

# Stop semua
docker compose down

# Stop + HAPUS semua data (reset total)
docker compose down -v
```

### Dengan Profile Opsional

```powershell
# Dengan Redis cache
docker compose --profile cache up -d --wait

# Dengan observability (OTEL tracing)
docker compose --profile observability up -d --wait

# Keduanya sekaligus
docker compose --profile cache --profile observability up -d --wait
```

### Upgrade Versi

```powershell
# 1. Edit ZITADEL_VERSION di .env (contoh: v4.12.0)

# 2. Pull image baru
docker compose pull

# 3. Restart
docker compose up -d --wait
```

### Debugging

```powershell
# Cek health status
docker compose ps

# Masuk ke container postgres (untuk cek database langsung)
docker compose exec postgres psql -U postgres -d zitadel

# Restart satu service saja
docker compose restart zitadel-api

# Rebuild tanpa cache
docker compose up -d --force-recreate
```

---

## Troubleshooting

### Service tidak healthy

```powershell
# Cek log untuk error
docker compose logs zitadel-api --tail 50
```

Penyebab umum:
- PostgreSQL belum ready → tunggu atau restart
- Masterkey berubah setelah data dibuat → kembalikan masterkey lama
- Port 8080 sudah dipakai → ubah `PROXY_HTTP_PUBLISHED_PORT`

### Tidak bisa login

- Pastikan `ZITADEL_DOMAIN` sesuai dengan URL yang kamu akses di browser
- Pastikan `ZITADEL_EXTERNALPORT` sesuai dengan port yang kamu akses
- Cek log `zitadel-login` untuk error token

### Reset total (mulai dari awal)

```powershell
docker compose down -v
docker compose up -d --wait
```

> **Peringatan:** `docker compose down -v` menghapus semua data termasuk database dan bootstrap token!

---

## Referensi

- Dokumentasi resmi: https://zitadel.com/docs
- Self-hosting guide: https://zitadel.com/docs/self-hosting/deploy/compose
- API docs: https://zitadel.com/docs/apis/introduction
- GitHub: https://github.com/zitadel/zitadel
