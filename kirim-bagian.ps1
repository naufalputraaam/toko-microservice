# kirim-bagian.ps1 : menyalin bagian Anda ke repo kelompok, commit bertahap, lalu push ke branch sendiri.
#
# Cara pakai (PowerShell, di dalam folder hasil ekstrak zip ini):
#   powershell -ExecutionPolicy Bypass -File .\kirim-bagian.ps1 -Repo https://github.com/USERNAME/toko-microservice.git -Bagian katalog
#
# Pilihan -Bagian:
#   awal        (ketua, PALING PERTAMA) README + docs ke branch main
#   katalog | pelanggan | order | pembayaran | gateway
#
# Syarat: Git sudah terpasang, sudah 'git config --global user.name/user.email',
#         dan Anda sudah menerima undangan collaborator repo.

param(
    [Parameter(Mandatory = $true)][string]$Repo,
    [Parameter(Mandatory = $true)][ValidateSet("awal", "katalog", "pelanggan", "order", "pembayaran", "gateway")][string]$Bagian
)

$ErrorActionPreference = "Continue"   # error ditangani manual lewat $LASTEXITCODE
$Sumber = $PSScriptRoot                                   # folder hasil ekstrak zip
$Nama = [IO.Path]::GetFileNameWithoutExtension($Repo.TrimEnd('/'))
$Tujuan = Join-Path ([Environment]::GetFolderPath("MyDocuments")) $Nama

function Jalan([string]$cmd) {
    Write-Host "> $cmd" -ForegroundColor DarkGray
    Invoke-Expression $cmd
    if ($LASTEXITCODE -ne 0) { throw "Perintah gagal: $cmd" }
}

function Commit([string[]]$File, [string]$Pesan) {
    foreach ($f in $File) { Jalan "git add -- `"$f`"" }
    git diff --cached --quiet
    if ($LASTEXITCODE -ne 0) { Jalan "git commit -m `"$Pesan`"" }
    else { Write-Host "  (tidak ada perubahan untuk: $Pesan)" -ForegroundColor Yellow }
}

# 0. Cek Git & identitas
if (-not (Get-Command git -ErrorAction SilentlyContinue)) { throw "Git belum terpasang. Install dari https://git-scm.com/download/win" }
$email = git config --global user.email
if (-not $email) { throw "Atur identitas dulu:  git config --global user.name ""Nama""  dan  git config --global user.email ""email-github@...""" }
Write-Host "Commit akan tercatat atas: $(git config --global user.name) <$email>" -ForegroundColor Cyan

# 1. Clone repo (atau perbarui jika sudah ada)
if (-not (Test-Path (Join-Path $Tujuan ".git"))) {
    Jalan "git clone `"$Repo`" `"$Tujuan`""
}
Set-Location $Tujuan
git fetch origin 2>$null

# 2. Pindah ke branch yang benar
if ($Bagian -eq "awal") {
    $Branch = "main"
    git rev-parse --verify origin/main 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) { Jalan "git checkout main"; Jalan "git pull origin main" }
    else { git symbolic-ref HEAD refs/heads/main }   # repo masih kosong
} else {
    $Branch = "feature/$Bagian"
    git rev-parse --verify origin/main 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Branch main belum ada. Ketua harus menjalankan -Bagian awal terlebih dahulu." }
    Jalan "git checkout main"
    Jalan "git pull origin main"
    Jalan "git checkout -B $Branch"
}

# 3. Salin file bagian ini dari folder zip
function Salin([string]$rel) {
    $src = Join-Path $Sumber $rel
    $dst = Join-Path $Tujuan $rel
    if (Test-Path $src -PathType Container) {
        New-Item -ItemType Directory -Force -Path $dst | Out-Null
        Copy-Item -Path (Join-Path $src "*") -Destination $dst -Recurse -Force
        Get-ChildItem -Path $src -Force -Filter ".*" | Copy-Item -Destination $dst -Force
    } else {
        New-Item -ItemType Directory -Force -Path (Split-Path $dst) | Out-Null
        Copy-Item -Path $src -Destination $dst -Force
    }
}

switch ($Bagian) {
    "awal" {
        Salin "README.md"; Salin ".gitignore"; Salin ".gitattributes"; Salin "docs"; Salin "kirim-bagian.ps1"
        Commit @("README.md", ".gitignore", ".gitattributes") "Inisialisasi repo: README"
        Commit @("docs") "Kontrak API dan panduan kelompok"
        Commit @("kirim-bagian.ps1") "Skrip bantu untuk anggota"
    }
    "katalog" {
        Salin "katalog-service"; $d = "katalog-service"
        Commit @("$d/go.mod", "$d/go.sum", "$d/util.go", "$d/client.go", "$d/main.go") "katalog: struktur service, koneksi database, dan routing"
        Commit @("$d/produk.go") "katalog: CRUD kategori dan produk"
        Commit @("$d/stok.go") "katalog: reserve, release, dan commit stok"
        Commit @($d) "katalog: Dockerfile dan dokumentasi"
    }
    "pelanggan" {
        Salin "pelanggan-service"; $d = "pelanggan-service"
        Commit @("$d/go.mod", "$d/go.sum", "$d/util.go", "$d/main.go") "pelanggan: struktur service, koneksi database, dan routing"
        Commit @("$d/auth.go") "pelanggan: registrasi, login JWT, dan verifikasi token"
        Commit @("$d/profil.go") "pelanggan: profil dan alamat"
        Commit @($d) "pelanggan: Dockerfile dan dokumentasi"
    }
    "order" {
        Salin "order-service"; $d = "order-service"
        Commit @("$d/go.mod", "$d/go.sum", "$d/util.go", "$d/client.go", "$d/main.go") "order: struktur service, koneksi database, dan routing"
        Commit @("$d/keranjang.go") "order: fitur keranjang"
        Commit @("$d/pesanan.go") "order: checkout saga dengan kompensasi, riwayat dan status pesanan"
        Commit @($d) "order: Dockerfile dan dokumentasi"
    }
    "pembayaran" {
        Salin "pembayaran-service"; $d = "pembayaran-service"
        Commit @("$d/go.mod", "$d/go.sum", "$d/util.go", "$d/client.go", "$d/main.go") "pembayaran: struktur service, database, dan worker sinkronisasi"
        Commit @("$d/pembayaran.go") "pembayaran: inisiasi, simulasi bayar, sinkronisasi status, laporan"
        Commit @($d) "pembayaran: Dockerfile dan dokumentasi"
    }
    "gateway" {
        Salin "gateway"; Salin "tests"; Salin "docker-compose.yml"
        Commit @("gateway/default.conf", "gateway/Dockerfile") "gateway: konfigurasi Nginx API gateway"
        Commit @("gateway/dashboard") "gateway: dashboard toko dan admin"
        Commit @("docker-compose.yml") "docker compose: database per service, service, dan gateway"
        Commit @("tests", "gateway") "tests: seed data dan tes end-to-end"
    }
}

# 4. Push
Jalan "git push -u origin $Branch"

Write-Host ""
Write-Host "SELESAI. Bagian '$Bagian' sudah di-push ke branch '$Branch'." -ForegroundColor Green
if ($Bagian -ne "awal") {
    $web = $Repo -replace '\.git$', ''
    Write-Host "Langkah berikutnya: buka $web lalu klik tombol 'Compare & pull request'." -ForegroundColor Green
}
