// Pembayaran Service: inisiasi dan simulasi pembayaran, sinkronisasi status ke order-service, laporan.
// Database: pembayaran_db (PostgreSQL). Diakses lewat gateway di /pembayaran/...
package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"
)

type App struct{ db *sql.DB }

var (
	pelangganURL = env("PELANGGAN_URL", "http://pelanggan-service:8080")
	orderURL     = env("ORDER_URL", "http://order-service:8080")
)

const schema = `
CREATE TABLE IF NOT EXISTS pembayaran (
    id           SERIAL PRIMARY KEY,
    order_kode   VARCHAR(40) NOT NULL UNIQUE,
    pelanggan_id INT NOT NULL,
    jumlah       BIGINT NOT NULL CHECK (jumlah >= 0),
    metode       VARCHAR(30) NOT NULL DEFAULT '',
    status       VARCHAR(20) NOT NULL DEFAULT 'PENDING',  -- PENDING | BERHASIL | GAGAL | DIBATALKAN
    tersinkron   BOOLEAN NOT NULL DEFAULT true,          -- false = status belum terkirim ke order-service
    sync_error   TEXT NOT NULL DEFAULT '',
    sync_attempt INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    dibayar_at   TIMESTAMPTZ
);`

func main() {
	app := &App{db: connectDB(env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pembayaran_db?sslmode=disable"))}
	if _, err := app.db.Exec(schema); err != nil {
		log.Fatal("migrasi gagal: ", err)
	}

	// Worker latar belakang: setiap beberapa detik mencoba lagi sinkronisasi yang tertunda.
	interval, _ := time.ParseDuration(env("SYNC_INTERVAL", "10s"))
	go func() {
		for range time.Tick(interval) {
			app.sinkronkanSemua()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "service": "pembayaran"})
	})
	// Pelanggan (butuh login)
	mux.HandleFunc("GET /pembayaran", auth(app.listPembayaranSaya))
	mux.HandleFunc("GET /pembayaran/order/{kode}", auth(app.getByOrder))
	mux.HandleFunc("POST /pembayaran/{id}/bayar", auth(app.bayar))
	// Admin
	mux.HandleFunc("GET /pembayaran/admin/daftar", adminOnly(app.listSemua))
	mux.HandleFunc("GET /pembayaran/admin/laporan", adminOnly(app.laporan))
	mux.HandleFunc("POST /pembayaran/admin/sinkron", adminOnly(app.sinkronManual))
	// Internal (dipanggil order-service)
	mux.HandleFunc("POST /internal/pembayaran", app.inisiasi)
	mux.HandleFunc("POST /internal/pembayaran/{kode}/batal", app.batal)

	addr := env("ADDR", ":8080")
	log.Println("pembayaran-service berjalan di", addr)
	log.Fatal(http.ListenAndServe(addr, logRequest(mux)))
}
