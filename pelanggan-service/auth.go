package main

// auth.go: registrasi, login (membuat token JWT), verifikasi token, dan middleware.

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// hashPassword menyimpan password sebagai PBKDF2-SHA256 + salt acak (tidak pernah plain text).
func hashPassword(pw string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	key, _ := pbkdf2.Key(sha256.New, pw, salt, 100000, 32)
	return "pbkdf2$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(key)
}

func checkPassword(hash, pw string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 3 {
		return false
	}
	salt, _ := hex.DecodeString(parts[1])
	want, _ := hex.DecodeString(parts[2])
	key, _ := pbkdf2.Key(sha256.New, pw, salt, 100000, 32)
	return subtle.ConstantTimeCompare(key, want) == 1
}

type Pelanggan struct {
	ID        int       `json:"id"`
	Nama      string    `json:"nama"`
	Email     string    `json:"email"`
	Telepon   string    `json:"telepon"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type ctxKey string

const userKey ctxKey = "user"

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nama     string `json:"nama"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Telepon  string `json:"telepon"`
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "body JSON tidak valid")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Nama == "" || !strings.Contains(in.Email, "@") || len(in.Password) < 6 {
		writeError(w, 400, "nama, email valid, dan password minimal 6 karakter wajib diisi")
		return
	}
	hash := hashPassword(in.Password)
	var p Pelanggan
	err := a.db.QueryRowContext(r.Context(),
		`INSERT INTO pelanggan (nama, email, password_hash, telepon) VALUES ($1,$2,$3,$4)
		 RETURNING id, nama, email, telepon, role, created_at`,
		in.Nama, in.Email, hash, in.Telepon).
		Scan(&p.ID, &p.Nama, &p.Email, &p.Telepon, &p.Role, &p.CreatedAt)
	if isDuplicate(err) {
		writeError(w, 409, "email sudah terdaftar")
		return
	}
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, p)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "body JSON tidak valid")
		return
	}
	var p Pelanggan
	var hash string
	err := a.db.QueryRowContext(r.Context(),
		`SELECT id, nama, email, telepon, role, created_at, password_hash FROM pelanggan WHERE email=$1`,
		strings.ToLower(strings.TrimSpace(in.Email))).
		Scan(&p.ID, &p.Nama, &p.Email, &p.Telepon, &p.Role, &p.CreatedAt, &hash)
	if err != nil || !checkPassword(hash, in.Password) {
		writeError(w, 401, "email atau password salah")
		return
	}
	// Token JWT berisi id, email, nama, role; berlaku 24 jam.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   p.ID,
		"email": p.Email,
		"nama":  p.Nama,
		"role":  p.Role,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString(a.jwtSecret)
	writeJSON(w, 200, map[string]any{"token": signed, "pelanggan": p})
}

// parseToken memeriksa header "Authorization: Bearer <token>".
func (a *App) parseToken(r *http.Request) (*Pelanggan, error) {
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if raw == "" {
		return nil, errors.New("token tidak ada")
	}
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return a.jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	id, _ := claims["sub"].(float64)
	var p Pelanggan
	err = a.db.QueryRow(
		`SELECT id, nama, email, telepon, role, created_at FROM pelanggan WHERE id=$1`, int(id)).
		Scan(&p.ID, &p.Nama, &p.Email, &p.Telepon, &p.Role, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// verify dipanggil service lain (katalog, order, pembayaran) untuk mengecek token.
func (a *App) verify(w http.ResponseWriter, r *http.Request) {
	p, err := a.parseToken(r)
	if err != nil {
		writeError(w, 401, "token tidak valid")
		return
	}
	writeJSON(w, 200, p)
}

// auth adalah middleware: handler hanya jalan jika token valid.
func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := a.parseToken(r)
		if err != nil {
			writeError(w, 401, "silakan login terlebih dahulu")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, p)))
	}
}

func (a *App) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r).Role != "admin" {
			writeError(w, 403, "khusus admin")
			return
		}
		next(w, r)
	}
}

func currentUser(r *http.Request) *Pelanggan {
	return r.Context().Value(userKey).(*Pelanggan)
}
