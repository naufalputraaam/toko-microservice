// Pelanggan Service: registrasi, login JWT, profil, alamat, dan verifikasi token.
// Database: pelanggan_db (PostgreSQL). Diakses lewat gateway di /pelanggan/...
package main

import (
	"database/sql"
	"log"
	"net/http"
)

type App struct {
	db        *sql.DB
	jwtSecret []byte
}

const schema = `
CREATE TABLE IF NOT EXISTS pelanggan (
    id            SERIAL PRIMARY KEY,
    nama          VARCHAR(100) NOT NULL,
    email         VARCHAR(100) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    telepon       VARCHAR(20) NOT NULL DEFAULT '',
    role          VARCHAR(20) NOT NULL DEFAULT 'pelanggan',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS alamat (
    id           SERIAL PRIMARY KEY,
    pelanggan_id INT NOT NULL REFERENCES pelanggan(id) ON DELETE CASCADE,
    label        VARCHAR(50)  NOT NULL DEFAULT 'Rumah',
    penerima     VARCHAR(100) NOT NULL,
    telepon      VARCHAR(20)  NOT NULL DEFAULT '',
    alamat       TEXT         NOT NULL,
    kota         VARCHAR(100) NOT NULL DEFAULT '',
    kode_pos     VARCHAR(10)  NOT NULL DEFAULT ''
);`

func main() {
	app := &App{
		db:        connectDB(env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pelanggan_db?sslmode=disable")),
		jwtSecret: []byte(env("JWT_SECRET", "rahasia-ganti-saya")),
	}
	if _, err := app.db.Exec(schema); err != nil {
		log.Fatal("migrasi gagal: ", err)
	}
	app.seedAdmin()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "service": "pelanggan"})
	})
	// Publik
	mux.HandleFunc("POST /pelanggan/register", app.register)
	mux.HandleFunc("POST /pelanggan/login", app.login)
	// Dipakai service lain untuk mengecek token (Bearer)
	mux.HandleFunc("GET /pelanggan/verify", app.verify)
	// Butuh login
	mux.HandleFunc("GET /pelanggan/profil", app.auth(app.getProfil))
	mux.HandleFunc("PUT /pelanggan/profil", app.auth(app.updateProfil))
	mux.HandleFunc("GET /pelanggan/alamat", app.auth(app.listAlamat))
	mux.HandleFunc("POST /pelanggan/alamat", app.auth(app.createAlamat))
	mux.HandleFunc("GET /pelanggan/alamat/{id}", app.auth(app.getAlamat))
	mux.HandleFunc("PUT /pelanggan/alamat/{id}", app.auth(app.updateAlamat))
	mux.HandleFunc("DELETE /pelanggan/alamat/{id}", app.auth(app.deleteAlamat))
	// Khusus admin
	mux.HandleFunc("GET /pelanggan/admin/daftar", app.auth(app.adminOnly(app.listPelanggan)))

	addr := env("ADDR", ":8080")
	log.Println("pelanggan-service berjalan di", addr)
	log.Fatal(http.ListenAndServe(addr, logRequest(mux)))
}

// seedAdmin membuat akun admin otomatis saat service pertama kali jalan.
func (a *App) seedAdmin() {
	email := env("ADMIN_EMAIL", "admin@toko.local")
	hash := hashPassword(env("ADMIN_PASSWORD", "admin123"))
	_, err := a.db.Exec(
		`INSERT INTO pelanggan (nama, email, password_hash, role) VALUES ('Administrator', $1, $2, 'admin')
		 ON CONFLICT (email) DO NOTHING`, email, hash)
	if err != nil {
		log.Println("gagal membuat admin:", err)
	}
}
