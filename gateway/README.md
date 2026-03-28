# Gateway Nginx (COSL)

File contoh di folder ini dipakai sebagai referensi saat mengonfigurasi reverse proxy di VM gateway (mis. `~/cosl-proxy/nginx/conf.d/default.conf`).

## WebSocket (notifikasi `/api/notifications/ws`)

Browser membuka WebSocket ke **Nuxt BFF** (bukan langsung ke Go):

`ws(s)://<host>/api/notifications/ws`

Handler ada di Nitro (`cosl-test-lab-consignment`), lalu server mem-proxy ke backend Go. Karena itu **`location` yang mem-proxy `/api/` ke Nuxt** harus mendukung upgrade HTTP.

### Yang perlu ada di Nginx

Di blok `server` untuk app (biasanya `listen 80`), **di dalam** `location` yang mengarahkan `/api/` ke upstream Nuxt (port **3000**), tambahkan:

```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
proxy_read_timeout 86400;
```

**Tidak** perlu `location` terpisah bernama `/ws` — path WebSocket tetap `/api/notifications/ws`.

### Yang sering salah

- `/api/` di-proxy ke **Go (8000)** padahal notifikasi WS harus lewat **Nuxt (3000)**.
- Lupa header `Upgrade` / `Connection` → handshake WebSocket gagal (meskipun path benar).

### File referensi di repo

| File | Isi |
| --- | --- |
| [`nginx-cosl-bff.example.conf`](nginx-cosl-bff.example.conf) | Contoh `server` + `location /api/` ke Nuxt dengan header WS |
| [`nginx-cosl-bff-ws-map.snippet.conf`](nginx-cosl-bff-ws-map.snippet.conf) | `map $http_upgrade $connection_upgrade` — include sekali di `http { }` jika ingin memakai `Connection $connection_upgrade` (opsional, lebih rapi untuk trafik campuran) |
| [`nginx-auth.example.conf`](nginx-auth.example.conf) | Contoh site terpisah untuk host Zitadel |

### Setelah mengubah config

```bash
nginx -t && nginx -s reload
```

(atau restart container Nginx sesuai setup Anda.)
