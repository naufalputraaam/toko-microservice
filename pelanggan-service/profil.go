package main

// profil.go: lihat/ubah profil, CRUD alamat, dan daftar pelanggan (admin).

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
)

type Alamat struct {
	ID          int    `json:"id"`
	PelangganID int    `json:"pelanggan_id"`
	Label       string `json:"label"`
	Penerima    string `json:"penerima"`
	Telepon     string `json:"telepon"`
	Alamat      string `json:"alamat"`
	Kota        string `json:"kota"`
	KodePos     string `json:"kode_pos"`
}

const kolomAlamat = `id, pelanggan_id, label, penerima, telepon, alamat, kota, kode_pos`

func scanAlamat(row interface{ Scan(...any) error }) (Alamat, error) {
	var a Alamat
	err := row.Scan(&a.ID, &a.PelangganID, &a.Label, &a.Penerima, &a.Telepon, &a.Alamat, &a.Kota, &a.KodePos)
	return a, err
}

func (a *App) getProfil(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, currentUser(r))
}

func (a *App) updateProfil(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var in struct {
		Nama    *string `json:"nama"`
		Telepon *string `json:"telepon"`
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "body JSON tidak valid")
		return
	}
	if in.Nama != nil && *in.Nama != "" {
		u.Nama = *in.Nama
	}
	if in.Telepon != nil {
		u.Telepon = *in.Telepon
	}
	_, err := a.db.ExecContext(r.Context(), `UPDATE pelanggan SET nama=$1, telepon=$2 WHERE id=$3`, u.Nama, u.Telepon, u.ID)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, u)
}

func (a *App) listAlamat(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(),
		`SELECT `+kolomAlamat+` FROM alamat WHERE pelanggan_id=$1 ORDER BY id`, currentUser(r).ID)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()
	hasil := []Alamat{}
	for rows.Next() {
		al, err := scanAlamat(rows)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		hasil = append(hasil, al)
	}
	writeJSON(w, 200, hasil)
}

func (a *App) createAlamat(w http.ResponseWriter, r *http.Request) {
	var in Alamat
	if err := readJSON(r, &in); err != nil || in.Penerima == "" || in.Alamat == "" {
		writeError(w, 400, "penerima dan alamat wajib diisi")
		return
	}
	if in.Label == "" {
		in.Label = "Rumah"
	}
	al, err := scanAlamat(a.db.QueryRowContext(r.Context(),
		`INSERT INTO alamat (pelanggan_id, label, penerima, telepon, alamat, kota, kode_pos)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING `+kolomAlamat,
		currentUser(r).ID, in.Label, in.Penerima, in.Telepon, in.Alamat, in.Kota, in.KodePos))
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, al)
}

// getAlamat juga dipakai order-service saat checkout (token pelanggan diteruskan).
func (a *App) getAlamat(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	al, err := scanAlamat(a.db.QueryRowContext(r.Context(),
		`SELECT `+kolomAlamat+` FROM alamat WHERE id=$1 AND pelanggan_id=$2`, id, currentUser(r).ID))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "alamat tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, al)
}

func (a *App) updateAlamat(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var in Alamat
	if err := readJSON(r, &in); err != nil || in.Penerima == "" || in.Alamat == "" {
		writeError(w, 400, "penerima dan alamat wajib diisi")
		return
	}
	al, err := scanAlamat(a.db.QueryRowContext(r.Context(),
		`UPDATE alamat SET label=$1, penerima=$2, telepon=$3, alamat=$4, kota=$5, kode_pos=$6
		 WHERE id=$7 AND pelanggan_id=$8 RETURNING `+kolomAlamat,
		in.Label, in.Penerima, in.Telepon, in.Alamat, in.Kota, in.KodePos, id, currentUser(r).ID))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "alamat tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, al)
}

func (a *App) deleteAlamat(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	tag, err := a.db.ExecContext(r.Context(), `DELETE FROM alamat WHERE id=$1 AND pelanggan_id=$2`, id, currentUser(r).ID)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if n, _ := tag.RowsAffected(); n == 0 {
		writeError(w, 404, "alamat tidak ditemukan")
		return
	}
	w.WriteHeader(204)
}

func (a *App) listPelanggan(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(),
		`SELECT id, nama, email, telepon, role, created_at FROM pelanggan ORDER BY id`)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()
	hasil := []Pelanggan{}
	for rows.Next() {
		var p Pelanggan
		rows.Scan(&p.ID, &p.Nama, &p.Email, &p.Telepon, &p.Role, &p.CreatedAt)
		hasil = append(hasil, p)
	}
	writeJSON(w, 200, hasil)
}
