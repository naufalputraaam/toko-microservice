# Kontrak API

Disepakati bersama **sebelum** coding, supaya tiap anggota bisa bekerja paralel.

- Semua request/response berformat JSON. Error selalu `{"error": "pesan"}`.
- Endpoint bertanda 🔒 butuh header `Authorization: Bearer <token>` (didapat dari login).
- Endpoint bertanda 👑 khusus admin.
- Endpoint `/internal/...` hanya untuk komunikasi antar-service (diblokir gateway, 403 dari luar).
- Kode status: `200` OK, `201` dibuat, `204` dihapus, `400` data tidak valid, `401` belum login/token salah,
  `403` bukan admin, `404` tidak ditemukan, `409` konflik/duplikat/stok kurang, `503` service lain tidak tersedia.
- Semua service punya `GET /health`.

## 1. Pelanggan Service (Peserta 2) · port uji mandiri 8082

| Method | Path | Body | Keterangan |
|---|---|---|---|
| POST | `/pelanggan/register` | `nama, email, password, telepon` | 201 data pelanggan, 409 email terpakai |
| POST | `/pelanggan/login` | `email, password` | 200 `{token, pelanggan}`, 401 salah |
| GET | `/pelanggan/verify` 🔒 | | 200 `{id, nama, email, role}`, dipakai service lain |
| GET | `/pelanggan/profil` 🔒 | | profil sendiri |
| PUT | `/pelanggan/profil` 🔒 | `nama, telepon` | ubah profil |
| GET | `/pelanggan/alamat` 🔒 | | daftar alamat sendiri |
| POST | `/pelanggan/alamat` 🔒 | `label, penerima, telepon, alamat, kota, kode_pos` | 201 |
| GET/PUT/DELETE | `/pelanggan/alamat/{id}` 🔒 | | detail/ubah/hapus (204) |
| GET | `/pelanggan/admin/daftar` 👑 | | semua pelanggan |

Token JWT (HS256, berlaku 24 jam) berisi `sub` (id), `email`, `nama`, `role`.

## 2. Katalog Service (Peserta 1) · port 8081

| Method | Path | Body | Keterangan |
|---|---|---|---|
| GET | `/katalog/kategori` | | daftar kategori |
| POST/PUT/DELETE | `/katalog/kategori[/{id}]` 👑 | `nama, deskripsi` | 409 jika nama sudah ada |
| GET | `/katalog/produk?kategori_id=&q=` | | daftar + filter + cari |
| GET | `/katalog/produk/{id}` | | detail produk |
| POST/PUT | `/katalog/produk[/{id}]` 👑 | `nama, kategori_id, harga, stok, deskripsi, gambar_url` | |
| DELETE | `/katalog/produk/{id}` 👑 | | 204 |
| PATCH | `/katalog/produk/{id}/stok` 👑 | `{"tambah": 10}` (boleh negatif) | 400 jika stok jadi negatif |
| GET | `/katalog/admin/reservasi` 👑 | | 200 reservasi terakhir |
| POST | `/internal/stok/reserve` | `order_kode, items:[{produk_id, qty}]` | 200 `{items:[{produk_id,nama,harga,qty,subtotal}], total}`, 409 stok kurang |
| POST | `/internal/stok/release` | `order_kode` | kembalikan stok (kompensasi) |
| POST | `/internal/stok/commit` | `order_kode` | reservasi final (setelah dibayar) |

## 3. Order Service (Peserta 3) · port 8083

| Method | Path | Body | Keterangan |
|---|---|---|---|
| GET | `/order/keranjang` 🔒 | | `{items:[...], total}` (detail dari katalog) |
| POST | `/order/keranjang` 🔒 | `produk_id, qty` | 201, 404 jika produk tidak ada di katalog |
| PUT | `/order/keranjang/{produk_id}` 🔒 | `qty` | ubah jumlah |
| DELETE | `/order/keranjang/{produk_id}` 🔒 | | 204 |
| POST | `/order/checkout` 🔒 | `alamat_id` | 201 pesanan; 409 stok kurang; 503 service lain mati |
| GET | `/order/pesanan` 🔒 | | riwayat pesanan sendiri |
| GET | `/order/pesanan/{kode}` 🔒 | | detail + item + riwayat status |
| POST | `/order/pesanan/{kode}/batal` 🔒 | | hanya saat MENUNGGU_PEMBAYARAN |
| GET | `/order/admin/pesanan?status=` 👑 | | semua pesanan |
| PUT | `/order/admin/pesanan/{kode}/status` 👑 | `{"status":"DIKIRIM"}` / `SELESAI` | |
| PUT | `/internal/pesanan/{kode}/status` | `{"status":"DIBAYAR"}` / `PEMBAYARAN_GAGAL` | dipanggil pembayaran-service |

**Alur checkout (Saga):**

1. Ambil isi keranjang dan alamat (GET ke pelanggan-service, token diteruskan).
2. Simpan pesanan `DIBUAT`.
3. `POST katalog /internal/stok/reserve`. Gagal → pesanan `GAGAL` (409/503).
4. `POST pembayaran /internal/pembayaran`. Gagal → **kompensasi**: `POST katalog /internal/stok/release`, pesanan `GAGAL`.
5. Sukses → pesanan `MENUNGGU_PEMBAYARAN`, keranjang dikosongkan.

**Status:** `DIBUAT → MENUNGGU_PEMBAYARAN → DIBAYAR → DIKIRIM → SELESAI`, atau `GAGAL` / `DIBATALKAN`.

## 4. Pembayaran Service (Peserta 4) · port 8084

| Method | Path | Body | Keterangan |
|---|---|---|---|
| GET | `/pembayaran` 🔒 | | pembayaran milik sendiri |
| GET | `/pembayaran/order/{kode}` 🔒 | | tagihan untuk satu pesanan |
| POST | `/pembayaran/{id}/bayar` 🔒 | `metode, simulasi: "sukses"/"gagal"` | 200; 409 jika sudah diproses |
| GET | `/pembayaran/admin/daftar?status=` 👑 | | semua pembayaran |
| GET | `/pembayaran/admin/laporan` 👑 | | total pendapatan, per status, per hari |
| POST | `/pembayaran/admin/sinkron` 👑 | | paksa sinkronisasi sekarang |
| POST | `/internal/pembayaran` | `order_kode, pelanggan_id, jumlah` | 201 tagihan PENDING (idempotent) |
| POST | `/internal/pembayaran/{kode}/batal` | | 409 jika sudah BERHASIL |

**Sinkronisasi status:** setelah bayar, status dikirim ke `PUT order /internal/pesanan/{kode}/status`
(BERHASIL → `DIBAYAR`, GAGAL → `PEMBAYARAN_GAGAL`). Jika order-service mati, kolom `tersinkron=false`
dan worker latar belakang mencoba ulang setiap `SYNC_INTERVAL` (default 10 detik).

## 5. Gateway (Peserta 5) · port 8080

| Path | Diteruskan ke |
|---|---|
| `/katalog...` | katalog-service:8080 |
| `/pelanggan...` | pelanggan-service:8080 |
| `/order...` | order-service:8080 |
| `/pembayaran...` | pembayaran-service:8080 |
| `/internal/...` | **diblokir (403)** |
| `/`, `/admin.html` | dashboard statis |
