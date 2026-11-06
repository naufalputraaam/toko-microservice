# Pelanggan Service (Peserta 2)

**Registrasi, login (JWT), profil, alamat**, dan **verifikasi token** untuk semua service lain.
Database sendiri: `pelanggan_db` (tabel `pelanggan`, `alamat`). Akun admin dibuat otomatis saat start (`ADMIN_EMAIL`/`ADMIN_PASSWORD`).

## Isi folder

| File | Isi |
|---|---|
| `main.go` | Migrasi tabel, buat akun admin, daftar route, jalankan server |
| `auth.go` | Register, login → membuat **token JWT**, `parseToken`, endpoint `/pelanggan/verify`, middleware `auth` & `adminOnly`, hash password |
| `profil.go` | Lihat/ubah profil, CRUD alamat, daftar pelanggan (admin) |
| `util.go` | Fungsi bantu JSON, koneksi database, deteksi data duplikat |
| `Dockerfile` | Build image Docker |

## Poin penting untuk dijelaskan saat demo

- **Password tidak disimpan asli**: di-hash dengan PBKDF2-SHA256 + salt acak.
- **JWT** berisi id, email, nama, role, dan masa berlaku 24 jam, lalu ditandatangani dengan `JWT_SECRET`.
- **Service lain tidak tahu `JWT_SECRET`**. Mereka memanggil `GET /pelanggan/verify` sambil membawa token, lalu service ini menjawab siapa user-nya. Jadi autentikasi terpusat di satu service.
- Konsekuensinya: kalau pelanggan-service mati, endpoint yang butuh login di service lain membalas 503.

## Uji mandiri

```bash
docker compose up -d --build pelanggan-service
curl -X POST http://localhost:8082/pelanggan/register -H "Content-Type: application/json" -d '{"nama":"Ani","email":"ani@mail.com","password":"rahasia1"}'
curl -X POST http://localhost:8082/pelanggan/login -H "Content-Type: application/json" -d '{"email":"ani@mail.com","password":"rahasia1"}'
```

## Langkah commit bertahap

```powershell
git add pelanggan-service\go.mod pelanggan-service\go.sum pelanggan-service\util.go pelanggan-service\main.go
git commit -m "pelanggan: struktur service, koneksi database, dan routing"
git add pelanggan-service\auth.go
git commit -m "pelanggan: registrasi, login JWT, dan verifikasi token"
git add pelanggan-service\profil.go
git commit -m "pelanggan: profil dan alamat"
git add pelanggan-service
git commit -m "pelanggan: Dockerfile dan dokumentasi"
git push -u origin feature/pelanggan
```
