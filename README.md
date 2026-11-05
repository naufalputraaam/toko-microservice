# Toko Mikro: Microservice dengan Go, PostgreSQL, Nginx, dan Docker

Tugas Kelompok Sistem Terdistribusi. Sistem toko online sederhana yang dipecah menjadi 4 service + API Gateway.

```
                      Browser / Postman
                             |
                 +-----------v------------+
                 |  API Gateway (Nginx)   |  :8080
                 |  + Dashboard toko/admin|
                 +-----------+------------+
      /katalog    /pelanggan |  /order       /pembayaran
   +-----------+-------------+-------------+--------------+
   v           v                           v              v
+---------+ +-----------+  HTTP  +----------+  HTTP  +------------+
| Katalog | | Pelanggan |<-------|  Order   |------->| Pembayaran |
| :8081   | | :8082     | verify | :8083    |<-------| :8084      |
+----+----+ +-----+-----+  token +----+-----+ status +-----+------+
     |  ^ reserve/release/commit stok  |                   |
     |  +------------------------------+                   |
     v            v                    v                   v
 katalog_db  pelanggan_db          order_db         pembayaran_db
 (Postgres)   (Postgres)          (Postgres)         (Postgres)
```

| Peran | Service | Folder | Branch |
|---|---|---|---|
| Peserta 1 | Katalog: produk, kategori, stok, reserve/release | `katalog-service/` | `feature/katalog` |
| Peserta 2 | Pelanggan: registrasi, login JWT, profil, alamat, verifikasi token | `pelanggan-service/` | `feature/pelanggan` |
| Peserta 3 | Order: keranjang, checkout dengan kompensasi, riwayat & status | `order-service/` | `feature/order` |
| Peserta 4 | Pembayaran: inisiasi & simulasi bayar, sinkronisasi status, laporan | `pembayaran-service/` | `feature/pembayaran` |
| Peserta 5 | Gateway, dashboard, Docker Compose, seed data, tes end-to-end | `gateway/`, `tests/`, `docker-compose.yml` | `feature/gateway` |

> Kelompok kami 4 orang, jadi peran Peserta 5 dirangkap oleh pembuat repo (lihat [docs/PANDUAN-KELOMPOK.md](docs/PANDUAN-KELOMPOK.md)).

## Menjalankan (Docker Desktop)

Pastikan Docker Desktop sudah terbuka (ikon paus di taskbar berwarna hijau/"running"). Lalu di folder repo:

```bash
docker compose up -d --build          # build & jalankan semua (pertama kali ±3-5 menit)
docker compose ps                     # semua harus "running"/"healthy"
docker compose run --rm tests seed    # isi data contoh
docker compose run --rm tests e2e     # jalankan 39 skenario pengujian
```

Buka di browser:

- Toko: http://localhost:8080 (akun demo `budi@toko.local` / `budi123`)
- Admin: http://localhost:8080/admin.html (`admin@toko.local` / `admin123`)

Perintah lain:

```bash
docker compose logs -f order-service       # lihat log satu service
docker compose stop pembayaran-service     # matikan satu service (uji kompensasi)
docker compose start pembayaran-service    # hidupkan lagi
docker compose down                        # matikan semua (data tetap tersimpan)
docker compose down -v                     # matikan + hapus semua data
```

## Konsep penting yang diterapkan

1. **Database per service**: tiap service punya PostgreSQL sendiri. Order tidak membaca tabel produk; ia bertanya ke katalog lewat HTTP.
2. **API Gateway**: Nginx meneruskan `/katalog`, `/pelanggan`, `/order`, `/pembayaran` ke service yang tepat dan **memblokir** `/internal/...` dari luar.
3. **Autentikasi terpusat**: pelanggan-service membuat token JWT. Service lain memvalidasi token dengan memanggil `GET /pelanggan/verify`.
4. **Saga + kompensasi (checkout)**: reserve stok → buat tagihan. Jika buat tagihan gagal, stok yang sudah di-reserve **dikembalikan** (release).
5. **Sinkronisasi status dengan retry**: jika order-service mati saat pelanggan membayar, pembayaran tetap tersimpan sebagai "belum tersinkron" dan worker mencoba lagi setiap 10 detik.
6. **Idempotent**: reserve/release/commit/sinkron aman dipanggil berulang kali.

Kontrak API lengkap: [docs/kontrak-api.md](docs/kontrak-api.md)

## Skenario uji kegagalan (untuk demo)

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| A | `docker compose stop pembayaran-service`, lalu checkout | 503, pesanan **GAGAL**, stok **kembali** seperti semula (kompensasi) |
| B | Checkout, `docker compose stop order-service`, bayar, lalu `docker compose start order-service` | Bayar tetap BERHASIL tapi "belum tersinkron"; ±10 detik setelah order hidup, pesanan otomatis jadi **DIBAYAR** |
| C | `docker compose stop katalog-service`, buka keranjang & checkout | Keranjang tetap tampil (detail "tidak tersedia"), checkout 503 |
| D | Matikan service apa pun, akses lewat gateway | Gateway membalas JSON `{"error":"service sedang tidak tersedia"}` (503), service lain tetap jalan |
