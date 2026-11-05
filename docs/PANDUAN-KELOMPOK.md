# Panduan Kelompok (untuk pemula)

Panduan ini menjelaskan langkah demi langkah: pasang aplikasi, buat repo GitHub, lalu **tiap anggota mengunggah bagiannya sendiri dari akun GitHub masing-masing**, supaya dosen bisa melihat siapa mengerjakan apa (tab *Commits*, *Pull requests*, dan *Insights → Contributors*).

## 0. Pembagian tugas

Isi nama dan username GitHub, lalu simpan di grup chat kelompok.

| Anggota | Username GitHub | Bagian | Folder | Branch |
|---|---|---|---|---|
| Anggota 1 | | Katalog Service | `katalog-service/` | `feature/katalog` |
| Anggota 2 | | Pelanggan Service | `pelanggan-service/` | `feature/pelanggan` |
| Anggota 3 | | Order Service | `order-service/` | `feature/order` |
| Anggota 4 (ketua, pembuat repo) | | Pembayaran Service **+** Gateway/Compose/Tes | `pembayaran-service/`, `gateway/`, `tests/`, `docker-compose.yml` | `feature/pembayaran`, `feature/gateway` |

Karena tabel dosen berisi 5 peran dan kelompok kita 4 orang, peran Peserta 5 dirangkap ketua. Kalau kelompok ingin pembagian lain, cukup ganti tabel ini.

**Aturan emas:** setiap orang **hanya mengubah folder miliknya sendiri**. Dengan begitu tidak akan ada konflik saat digabung.

---

## Cara cepat: skrip otomatis (setelah langkah 1–2 dan repo kosong dibuat di langkah 3.1–3.2)

Di folder hasil ekstrak zip ada `kirim-bagian.ps1`. Skrip ini melakukan langkah 3 dan 4 sekaligus: clone repo, menyalin folder bagian Anda, commit bertahap, lalu push ke branch Anda. Buka PowerShell **di folder hasil ekstrak zip**, lalu jalankan:

```powershell
# Ketua, sekali saja, PALING PERTAMA (setelah repo kosong dibuat di GitHub):
powershell -ExecutionPolicy Bypass -File .\kirim-bagian.ps1 -Repo https://github.com/USERNAME-KETUA/toko-microservice.git -Bagian awal

# Tiap anggota, sesuai bagiannya (katalog / pelanggan / order / pembayaran / gateway):
powershell -ExecutionPolicy Bypass -File .\kirim-bagian.ps1 -Repo https://github.com/USERNAME-KETUA/toko-microservice.git -Bagian katalog
```

Setelah skrip selesai, buka halaman repo di GitHub dan klik **Compare & pull request** (lihat langkah 4).

---

## 1. Pasang aplikasi (SEMUA anggota)

1. **Git**: unduh di https://git-scm.com/download/win, lalu install dengan pilihan default (Next terus).
2. **Docker Desktop**: unduh di https://www.docker.com/products/docker-desktop/, install, lalu restart laptop.
   - Saat pertama dibuka, pilih *Use WSL 2* (default). Jika muncul pesan "WSL 2 is not installed/incomplete", buka **PowerShell sebagai Administrator** dan jalankan `wsl --install` atau `wsl --update`, lalu restart.
   - Docker Desktop **harus terbuka** setiap kali mau menjalankan proyek. Tunggu sampai tulisan di kiri bawah "Engine running".
3. **VS Code** (opsional, untuk melihat kode): https://code.visualstudio.com
4. **Akun GitHub**: daftar di https://github.com jika belum punya.

Cek instalasi. Buka **PowerShell** (tekan tombol Windows, ketik "PowerShell"), lalu jalankan:

```powershell
git --version
docker --version
docker compose version
```

Ketiganya harus menampilkan nomor versi.

**Penting.** Atur identitas Git memakai **email yang sama dengan akun GitHub**. Kalau email berbeda, commit tidak tercatat atas nama Anda di GitHub.

```powershell
git config --global user.name  "Nama Lengkap"
git config --global user.email "email-akun-github@gmail.com"
```

---

## 2. Coba jalankan proyek lengkap (SEMUA anggota, sekali)

Ekstrak `tugas-microservice.zip`, misalnya ke `Documents\tugas-microservice`. Buka PowerShell di folder itu (di File Explorer: klik kanan area kosong → *Open in Terminal*), lalu:

```powershell
docker compose up -d --build
```

Pertama kali, proses ini butuh 3–5 menit karena Docker mengunduh image. Setelah selesai:

```powershell
docker compose ps                     # semua service harus "Up"
docker compose run --rm tests seed    # isi data contoh
docker compose run --rm tests e2e     # harus "Hasil: 39 lulus, 0 gagal"
```

Buka http://localhost:8080 (toko) dan http://localhost:8080/admin.html (admin: `admin@toko.local` / `admin123`).

Untuk mematikan: `docker compose down`.

> Folder hasil ekstrak zip ini **bukan** repo GitHub. Folder ini hanya "bahan" yang nanti disalin per bagian ke repo kelompok (langkah 4).

---

## 3. Ketua: membuat repo dan mengundang anggota

1. Buka https://github.com/new
   - Repository name: `toko-microservice`
   - Pilih **Public** (atau Private, asal dosen diundang juga)
   - **Jangan** centang "Add a README" (biarkan kosong)
   - Klik **Create repository**
2. Buka repo → **Settings** → **Collaborators** → **Add people**, lalu masukkan username GitHub ketiga anggota.
   Anggota harus **menerima undangan** yang masuk lewat email atau di https://github.com/notifications.
3. Unggah isi awal (README dan kontrak API) ke branch `main`:

```powershell
cd $HOME\Documents
git clone https://github.com/USERNAME-KETUA/toko-microservice.git
cd toko-microservice

# salin file awal dari folder zip
Copy-Item ..\tugas-microservice\README.md, ..\tugas-microservice\.gitignore, ..\tugas-microservice\.gitattributes .
Copy-Item ..\tugas-microservice\docs -Recurse .

git add .
git commit -m "Inisialisasi repo: README dan kontrak API"
git branch -M main
git push -u origin main
```

4. (Disarankan) **Settings → Branches → Add rule** untuk `main`: centang *Require a pull request before merging*. Dengan aturan ini, semua perubahan wajib lewat Pull Request.

---

## 4. Tiap anggota: kerjakan di branch sendiri

Contoh untuk **Anggota 1 (Katalog)**. Anggota lain cukup mengganti nama folder dan branch sesuai tabel di atas.

```powershell
cd $HOME\Documents
git clone https://github.com/USERNAME-KETUA/toko-microservice.git
cd toko-microservice

git checkout -b feature/katalog                  # buat branch sendiri

Copy-Item ..\tugas-microservice\katalog-service -Recurse .   # salin folder bagian Anda
```

Sebaiknya commit **bertahap**, tidak langsung semua sekaligus, supaya riwayat kerjanya jelas. Contoh untuk katalog di bawah; langkah commit untuk folder lain ada di bagian akhir `README.md` folder masing-masing.

```powershell
git add katalog-service\go.mod katalog-service\go.sum katalog-service\util.go katalog-service\client.go katalog-service\main.go
git commit -m "katalog: struktur service, koneksi database, dan routing"

git add katalog-service\produk.go
git commit -m "katalog: CRUD kategori dan produk"

git add katalog-service\stok.go
git commit -m "katalog: reserve, release, dan commit stok"

git add katalog-service
git commit -m "katalog: Dockerfile dan dokumentasi"

git push -u origin feature/katalog
```

Lalu buka repo di GitHub. Akan muncul tombol kuning **Compare & pull request**. Klik tombol itu, isi judul (misal "Katalog Service"), lalu **Create pull request**.

Minta satu anggota lain membuka PR tersebut, klik **Files changed → Review changes → Approve**, lalu **Merge pull request**.

Pengerjaan **Ketua** sama, tetapi dua kali:

- `feature/pembayaran` → folder `pembayaran-service`
- `feature/gateway` → folder `gateway`, `tests`, dan file `docker-compose.yml`

Sebelum membuat branch kedua, kembali ke main dulu: `git checkout main` lalu `git pull`.

### Urutan merge yang disarankan
`pelanggan` → `katalog` → `order` → `pembayaran` → `gateway`. Karena folder tiap orang berbeda, urutan lain juga tidak akan konflik.

---

## 5. Setelah semua di-merge (integrasi bersama)

Semua anggota menjalankan:

```powershell
cd $HOME\Documents\toko-microservice
git checkout main
git pull
docker compose up -d --build
docker compose run --rm tests seed
docker compose run --rm tests e2e
```

Lalu coba skenario kegagalan A–D di README untuk bahan laporan dan demo.

---

## 6. Wajib: pahami bagian sendiri

Saat demo, dosen biasanya bertanya ke **masing-masing** anggota. Setiap folder service punya `README.md` yang menjelaskan isi file dan alurnya. Baca, jalankan, dan coba ubah sedikit (misalnya pesan error) supaya Anda paham cara kerjanya.

Pertanyaan yang sering muncul:

- Kenapa tiap service punya database sendiri? Apa ruginya?
- Apa yang terjadi kalau katalog/pembayaran mati saat checkout? (jawaban: kompensasi)
- Bagaimana service lain tahu user sudah login? (jawaban: `/pelanggan/verify`)
- Kenapa endpoint `/internal` diblokir gateway?

---

## 7. Masalah umum

| Masalah | Solusi |
|---|---|
| `docker: command not found` / `error during connect` | Docker Desktop belum dibuka atau belum "Engine running" |
| `port is already allocated` (8080) | Ada aplikasi lain di port 8080. Tutup aplikasi itu, atau ubah `"8080:80"` jadi `"9090:80"` di docker-compose.yml |
| `WSL 2 installation is incomplete` | PowerShell (Admin): `wsl --update`, lalu restart |
| `Permission denied` saat push | Belum menerima undangan collaborator, atau belum login GitHub. Saat push pertama, Git akan membuka jendela login GitHub |
| Commit tidak muncul di profil GitHub | `git config user.email` berbeda dengan email akun GitHub |
| Tes e2e gagal setelah data diubah manual | Mulai dari data bersih: `docker compose down -v`, lalu ulangi langkah 5 |
| Build lama / gagal unduh | Pastikan internet stabil, lalu ulangi `docker compose up -d --build` |
