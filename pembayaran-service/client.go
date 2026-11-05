package main

// client.go: memanggil service lain lewat HTTP (komunikasi antar-service)
// dan mengecek token login ke pelanggan-service.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// User adalah data pengguna hasil verifikasi token dari pelanggan-service.
type User struct {
	ID    int    `json:"id"`
	Nama  string `json:"nama"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

// ErrServiceDown dipakai jika service tujuan tidak bisa dihubungi.
var ErrServiceDown = errors.New("service tidak dapat dihubungi")

// callJSON mengirim request JSON ke service lain dan membaca balasannya ke out.
// Mengembalikan status code HTTP dari service tujuan.
func callJSON(ctx context.Context, method, url, authHeader string, body, out any) (int, error) {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, &buf)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrServiceDown, err)
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode < 300 {
		json.NewDecoder(resp.Body).Decode(out)
	} else if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, errors.New(e.Error)
	}
	return resp.StatusCode, nil
}

type ctxKey string

const userKey ctxKey = "user"

// auth: middleware yang mengecek token ke pelanggan-service (GET /pelanggan/verify).
func auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" {
			writeError(w, 401, "silakan login terlebih dahulu")
			return
		}
		var u User
		status, err := callJSON(r.Context(), "GET", pelangganURL+"/pelanggan/verify", h, nil, &u)
		if errors.Is(err, ErrServiceDown) {
			writeError(w, 503, "pelanggan-service tidak dapat dihubungi")
			return
		}
		if status != 200 {
			writeError(w, 401, "token tidak valid")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, &u)))
	}
}

// adminOnly: hanya user dengan role admin yang boleh lanjut.
func adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return auth(func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r).Role != "admin" {
			writeError(w, 403, "khusus admin")
			return
		}
		next(w, r)
	})
}

func currentUser(r *http.Request) *User {
	return r.Context().Value(userKey).(*User)
}
