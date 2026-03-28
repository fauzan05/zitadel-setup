# Checklist deploy Zitadel di VM (VM2 + gateway VM3)

Dokumen ini mewujudkan langkah dari rencana deploy VM: URL konsisten, `.env`, HTTPS lewat gateway, firewall, SMTP opsional, OIDC, dan `lark-proxy`.

## 1. Tentukan alamat yang dipakai user (issuer)

- Putuskan hostname publik untuk Zitadel, misalnya `auth.perusahaan.com` (disarankan **subdomain terpisah** dari app).
- DNS internal atau publik: buat **A record** ke IP **VM3 (gateway)** — bukan langsung ke VM2, jika user hanya lewat Nginx.
- Nilai **`ZITADEL_DOMAIN`** di `.env` VM2 harus **sama persis** dengan hostname yang diketik di browser (tanpa `http://`, tanpa path).
- **Port luar** yang “dilihat” Zitadel untuk redirect OIDC:
  - User pakai `https://auth...` (port 443) → set `ZITADEL_EXTERNALPORT=443`, `ZITADEL_EXTERNALSECURE=true`, `ZITADEL_PUBLIC_SCHEME=https`.
  - Uji internal HTTP saja ke VM2 → sesuaikan port publish Traefik dan `ZITADEL_EXTERNALPORT` (mis. sama dengan `PROXY_HTTP_PUBLISHED_PORT`).

## 2. Selaraskan `.env` sebelum bootstrap pertama (idealnya)

- Salin `.env.example` → `.env` di server, lalu isi domain, port, skema, dan password kuat.
- **`ZITADEL_MASTERKEY`**: tepat 32 karakter; **jangan ubah** setelah data terisi.
- Setelah instance jalan, mengganti domain/issuer mengikuti prosedur Zitadel (bukan sekadar edit `.env`).

## 3. HTTPS (TLS terminate di VM3)

- User → **HTTPS ke VM3 (Nginx + Certbot)** → **HTTP ke VM2** (mis. `http://10.184.131.67:8080`) adalah pola yang umum.
- Di Nginx: set `proxy_set_header Host $host;`, `X-Forwarded-For`, dan **`X-Forwarded-Proto $scheme`** (harus `https` untuk user).
- Di `.env` VM2: Zitadel harus dikonfigurasi seolah user memakai **https** (lihat poin 1).
- Traefik di VM2 sekarang mempercayai **`X-Forwarded-*`** dari IP di **`TRAEFIK_TRUSTED_IPS`** (gateway biasanya di rentang privat — sesuaikan jika perlu).

Contoh blok server terpisah untuk hostname auth: lihat [`gateway/nginx-auth.example.conf`](gateway/nginx-auth.example.conf).

## 4. Firewall dan rahasia

- **VM2**: izinkan inbound (TCP) ke port publish Traefik (mis. **8080**) hanya dari **VM3** (dan dari **VM1** jika backend memanggil Zitadel lewat IP internal).
- **VM4 (Postgres app)**: jika dipakai terpisah, batasi `5432` hanya dari klien yang berhak (bukan topik stack ini jika Postgres Zitadel tetap di compose VM2).
- Jangan commit **`.env`**, **`login-client.pat`**, atau token Lark — lihat `.gitignore`.

## 5. Email (SMTP)

- **Tidak wajib** di `.env`. Umumnya diisi lewat **Console Zitadel** → pengaturan instance / SMTP setelah login admin.
- Tanpa SMTP, fitur yang mengandalkan email (mis. reset password lewat email) bisa tidak jalan.

## 6. OIDC dan integrasi aplikasi (COSL)

- Di Console Zitadel: buat **Application** (OIDC), catat **issuer** (biasanya `https://auth.../`), client id/secret, redirect URI sesuai app Anda.
- Samakan konfigurasi di backend/frontend (mis. `config.json` / env di repo COSL backend Anda) dengan issuer dan redirect URI tersebut.

## 7. `lark-proxy`

- Service ini (Go) memproses alur token/userinfo Lark (port **4000** di dalam container). Endpoint dan env: [lark-proxy/README.md](lark-proxy/README.md).
- Secara default **tidak** dipublish ke host; akses dari stack Docker atau tambahkan `ports` di `docker-compose.yml` jika harus dicapai dari luar (batasi firewall).
- Simpan **client secret** / token Lark di env terpisah dan jangan commit.

## Verifikasi cepat

```bash
docker compose --env-file .env -f docker-compose.yml ps
docker compose --env-file .env -f docker-compose.yml logs zitadel-api --tail 80
```

Buka URL sesuai `ZITADEL_DOMAIN` (lewat gateway) dan uji login / console.
