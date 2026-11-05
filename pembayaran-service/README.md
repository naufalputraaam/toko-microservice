# Pembayaran Service (Peserta 4)

**Inisiasi tagihan**, **simulasi pembayaran** (sukses/gagal), **sinkronisasi status** ke order-service, dan **laporan**.
Database sendiri: `pembayaran_db` (tabel `pembayaran`).

## Isi folder

| File | Isi |
|---|---|
| `main.go` | Migrasi tabel, menjalankan **worker sinkronisasi** di latar belakang, daftar route |
| `pembayaran.go` | Inisiasi & batal (internal), bayar/simulasi, sinkronisasi, laporan admin |
| `client.go` | `callJSON` ke service lain, middleware `auth`/`adminOnly` |
| `util.go` | Fungsi bantu JSON & database |
| `Dockerfile` | Build image Docker |

## Poin penting untuk dijelaskan saat demo

- **Simpan dulu, kabari kemudian**: hasil bayar disimpan dengan `tersinkron=false`, baru dikirim ke order-service. Jika order-service mati, data pembayaran **tidak hilang**.
- **Worker retry**: goroutine `time.Tick(SYNC_INTERVAL)` mencari pembayaran yang `tersinkron=false` lalu mencoba mengirim lagi. Admin juga bisa menekan "Sinkronkan sekarang".
- **Cegah bayar dobel**: `UPDATE ... WHERE status='PENDING'`. Request kedua tidak mengubah apa pun dan dibalas 409.
- **Laporan**: total pendapatan, jumlah per status, pendapatan per hari (30 hari), jumlah yang belum tersinkron.

## Langkah commit bertahap

```powershell
git add pembayaran-service\go.mod pembayaran-service\go.sum pembayaran-service\util.go pembayaran-service\client.go pembayaran-service\main.go
git commit -m "pembayaran: struktur service, database, dan worker sinkronisasi"
git add pembayaran-service\pembayaran.go
git commit -m "pembayaran: inisiasi, simulasi bayar, sinkronisasi status, laporan"
git add pembayaran-service
git commit -m "pembayaran: Dockerfile dan dokumentasi"
git push -u origin feature/pembayaran
```
