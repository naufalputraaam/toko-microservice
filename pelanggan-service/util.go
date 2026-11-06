package main

// util.go berisi fungsi-fungsi bantu yang dipakai semua handler:
// membaca JSON dari request, menulis JSON ke response, dan format error.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lib/pq" // driver PostgreSQL
)

// env membaca environment variable, atau memakai nilai default jika kosong.
func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// readJSON membaca body request (format JSON) ke dalam struct dst.
func readJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// writeJSON menulis data sebagai JSON dengan status code tertentu.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError menulis error dengan format {"error": "pesan"}.
func writeError(w http.ResponseWriter, status int, pesan string) {
	writeJSON(w, status, map[string]string{"error": pesan})
}

// connectDB membuka koneksi ke PostgreSQL, mencoba ulang sampai database siap
// (saat docker compose up, database biasanya baru siap beberapa detik kemudian).
func connectDB(url string) *sql.DB {
	for i := 1; i <= 30; i++ {
		db, err := sql.Open("postgres", url)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			return db
		}
		log.Printf("menunggu database (percobaan %d): %v", i, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("database tidak tersedia")
	return nil
}

// logRequest mencatat setiap request yang masuk ke log container.
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

// isDuplicate bernilai true jika error berasal dari pelanggaran UNIQUE di PostgreSQL.
func isDuplicate(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
