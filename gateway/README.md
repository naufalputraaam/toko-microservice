# Gateway, Dashboard, Docker Compose, Seed & Tes (Peserta 5)

Folder `gateway/`, `tests/`, dan file `docker-compose.yml` di root.

| File | Isi |
|---|---|
| `gateway/default.conf` | Konfigurasi Nginx: routing `/katalog`, `/pelanggan`, `/order`, `/pembayaran` → service; blokir `/internal/`; JSON 503 jika service mati |
| `gateway/Dockerfile` | Image Nginx + file dashboard |
| `gateway/dashboard/index.html` | Dashboard **toko**: produk, keranjang, checkout, pesanan, bayar, akun & alamat |
| `gateway/dashboard/admin.html` | Dashboard **admin**: laporan, kelola pesanan, produk/kategori/stok, pembayaran, pelanggan |
| `gateway/dashboard/common.js`, `style.css` | Fungsi & tampilan bersama |
| `docker-compose.yml` | 4 database + 4 service + gateway + container `tests` |
| `tests/main.go` | `seed` (data contoh) dan `e2e` (39 skenario pengujian lewat gateway) |

## Poin penting untuk dijelaskan saat demo

- Klien cukup tahu **satu alamat** (`localhost:8080`); gateway yang tahu letak tiap service.
- `resolver 127.0.0.11` + `proxy_pass` dengan variabel membuat Nginx tetap jalan walau salah satu service mati.
- `depends_on: condition: service_healthy` membuat service menunggu database siap.
- Antar-container saling memanggil lewat **nama service** (DNS internal Docker), mis. `http://katalog-service:8080`.

## Langkah commit bertahap

```powershell
git add gateway\default.conf gateway\Dockerfile
git commit -m "gateway: konfigurasi Nginx API gateway"
git add gateway\dashboard
git commit -m "gateway: dashboard toko dan admin"
git add docker-compose.yml
git commit -m "docker compose: database per service, service, dan gateway"
git add tests gateway\README.md
git commit -m "tests: seed data dan tes end-to-end"
git push -u origin feature/gateway
```
