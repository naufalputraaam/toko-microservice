package main

// pembayaran.go: inisiasi tagihan, simulasi pembayaran, sinkronisasi status, laporan.

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Pembayaran struct {
	ID          int        `json:"id"`
	OrderKode   string     `json:"order_kode"`
	PelangganID int        `json:"pelanggan_id"`
	Jumlah      int64      `json:"jumlah"`
	Metode      string     `json:"metode"`
	Status      string     `json:"status"`
	Tersinkron  bool       `json:"tersinkron"`
	SyncError   string     `json:"sync_error"`
	SyncAttempt int        `json:"sync_attempt"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DibayarAt   *time.Time `json:"dibayar_at"`
}

const selectBayar = `SELECT id, order_kode, pelanggan_id, jumlah, metode, status, tersinkron, sync_error,
	sync_attempt, created_at, updated_at, dibayar_at FROM pembayaran`

func scanBayar(row interface{ Scan(...any) error }) (Pembayaran, error) {
	var p Pembayaran
	err := row.Scan(&p.ID, &p.OrderKode, &p.PelangganID, &p.Jumlah, &p.Metode, &p.Status, &p.Tersinkron,
		&p.SyncError, &p.SyncAttempt, &p.CreatedAt, &p.UpdatedAt, &p.DibayarAt)
	return p, err
}

// ---------- Internal: dipanggil order-service ----------

// inisiasi membuat tagihan PENDING untuk sebuah pesanan. Idempotent per order_kode.
func (a *App) inisiasi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OrderKode   string `json:"order_kode"`
		PelangganID int    `json:"pelanggan_id"`
		Jumlah      int64  `json:"jumlah"`
	}
	if err := readJSON(r, &in); err != nil || in.OrderKode == "" || in.PelangganID <= 0 || in.Jumlah < 0 {
		writeError(w, 400, "order_kode, pelanggan_id, dan jumlah wajib diisi")
		return
	}
	a.db.ExecContext(r.Context(),
		`INSERT INTO pembayaran (order_kode, pelanggan_id, jumlah) VALUES ($1,$2,$3) ON CONFLICT (order_kode) DO NOTHING`,
		in.OrderKode, in.PelangganID, in.Jumlah)
	p, err := scanBayar(a.db.QueryRowContext(r.Context(), selectBayar+` WHERE order_kode=$1`, in.OrderKode))
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, p)
}

// batal membatalkan tagihan yang masih PENDING (pesanan dibatalkan pelanggan).
func (a *App) batal(w http.ResponseWriter, r *http.Request) {
	kode := r.PathValue("kode")
	p, err := scanBayar(a.db.QueryRowContext(r.Context(), selectBayar+` WHERE order_kode=$1`, kode))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "pembayaran tidak ditemukan")
		return
	}
	if p.Status == "BERHASIL" {
		writeError(w, 409, "pesanan sudah dibayar, tidak bisa dibatalkan")
		return
	}
	a.db.ExecContext(r.Context(), `UPDATE pembayaran SET status='DIBATALKAN', updated_at=now() WHERE id=$1 AND status='PENDING'`, p.ID)
	p, _ = scanBayar(a.db.QueryRowContext(r.Context(), selectBayar+` WHERE id=$1`, p.ID))
	writeJSON(w, 200, p)
}

// ---------- Pelanggan ----------

func (a *App) listPembayaranSaya(w http.ResponseWriter, r *http.Request) {
	a.list(w, r, `WHERE pelanggan_id=$1`, currentUser(r).ID)
}

func (a *App) getByOrder(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	p, err := scanBayar(a.db.QueryRowContext(r.Context(), selectBayar+` WHERE order_kode=$1`, r.PathValue("kode")))
	if err != nil || (p.PelangganID != u.ID && u.Role != "admin") {
		writeError(w, 404, "pembayaran tidak ditemukan")
		return
	}
	writeJSON(w, 200, p)
}

// bayar mensimulasikan pembayaran. Body: {"metode":"transfer","simulasi":"sukses"} (atau "gagal")
func (a *App) bayar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.Atoi(r.PathValue("id"))
	var in struct {
		Metode   string `json:"metode"`
		Simulasi string `json:"simulasi"`
	}
	readJSON(r, &in)
	if in.Metode == "" {
		in.Metode = "transfer"
	}
	statusBaru := "BERHASIL"
	if in.Simulasi == "gagal" {
		statusBaru = "GAGAL"
	}
	p, err := scanBayar(a.db.QueryRowContext(ctx, selectBayar+` WHERE id=$1`, id))
	if err != nil || p.PelangganID != currentUser(r).ID {
		writeError(w, 404, "pembayaran tidak ditemukan")
		return
	}
	if p.Status != "PENDING" {
		writeError(w, 409, "pembayaran sudah berstatus "+p.Status)
		return
	}
	// Simpan hasil pembayaran dulu (tandai belum tersinkron), baru kabari order-service.
	// "AND status='PENDING'" mencegah dua request bayar bersamaan sama-sama berhasil.
	res, err := a.db.ExecContext(ctx,
		`UPDATE pembayaran SET status=$1, metode=$2, tersinkron=false, updated_at=now(),
		 dibayar_at = CASE WHEN $3 THEN now() ELSE NULL END WHERE id=$4 AND status='PENDING'`,
		statusBaru, in.Metode, statusBaru == "BERHASIL", id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, 409, "pembayaran sudah diproses")
		return
	}
	a.sinkronkan(ctx, id)
	p, _ = scanBayar(a.db.QueryRowContext(ctx, selectBayar+` WHERE id=$1`, id))
	writeJSON(w, 200, p)
}

// ---------- Sinkronisasi status ke order-service ----------

// sinkronkan mengirim status pembayaran ke order-service.
// Jika order-service mati, tersinkron tetap false dan akan dicoba lagi oleh worker.
func (a *App) sinkronkan(ctx context.Context, id int) {
	p, err := scanBayar(a.db.QueryRowContext(ctx, selectBayar+` WHERE id=$1`, id))
	if err != nil || p.Tersinkron {
		return
	}
	statusOrder := map[string]string{"BERHASIL": "DIBAYAR", "GAGAL": "PEMBAYARAN_GAGAL"}[p.Status]
	if statusOrder == "" {
		a.db.ExecContext(ctx, `UPDATE pembayaran SET tersinkron=true WHERE id=$1`, id)
		return
	}
	code, err := callJSON(ctx, "PUT", orderURL+"/internal/pesanan/"+p.OrderKode+"/status", "",
		map[string]string{"status": statusOrder}, nil)
	if err == nil || code == 409 || code == 404 {
		// 409/404 tidak akan berubah walau dicoba ulang, jadi dianggap selesai (dicatat errornya).
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		a.db.ExecContext(ctx, `UPDATE pembayaran SET tersinkron=true, sync_error=$1, sync_attempt=sync_attempt+1 WHERE id=$2`, msg, id)
		return
	}
	log.Printf("sinkron %s gagal, akan dicoba lagi: %v", p.OrderKode, err)
	a.db.ExecContext(ctx, `UPDATE pembayaran SET sync_error=$1, sync_attempt=sync_attempt+1 WHERE id=$2`, err.Error(), id)
}

func (a *App) sinkronkanSemua() int {
	rows, err := a.db.Query(`SELECT id FROM pembayaran WHERE tersinkron=false ORDER BY id`)
	if err != nil {
		return 0
	}
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		a.sinkronkan(context.Background(), id)
	}
	return len(ids)
}

func (a *App) sinkronManual(w http.ResponseWriter, r *http.Request) {
	n := a.sinkronkanSemua()
	var sisa int
	a.db.QueryRow(`SELECT count(*) FROM pembayaran WHERE tersinkron=false`).Scan(&sisa)
	writeJSON(w, 200, map[string]int{"diproses": n, "belum_tersinkron": sisa})
}

// ---------- Admin ----------

func (a *App) list(w http.ResponseWriter, r *http.Request, where string, args ...any) {
	rows, err := a.db.QueryContext(r.Context(), selectBayar+" "+where+" ORDER BY id DESC", args...)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()
	hasil := []Pembayaran{}
	for rows.Next() {
		p, _ := scanBayar(rows)
		hasil = append(hasil, p)
	}
	writeJSON(w, 200, hasil)
}

func (a *App) listSemua(w http.ResponseWriter, r *http.Request) {
	a.list(w, r, `WHERE ($1 = '' OR status = $1)`, r.URL.Query().Get("status"))
}

// laporan: ringkasan per status, total pendapatan, dan pendapatan per hari (30 hari terakhir).
func (a *App) laporan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	type perStatus struct {
		Status string `json:"status"`
		Jumlah int    `json:"jumlah_transaksi"`
		Total  int64  `json:"total_nominal"`
	}
	type perHari struct {
		Tanggal    string `json:"tanggal"`
		Transaksi  int    `json:"transaksi"`
		Pendapatan int64  `json:"pendapatan"`
	}
	statuses := []perStatus{}
	rows, _ := a.db.QueryContext(ctx, `SELECT status, count(*), coalesce(sum(jumlah),0) FROM pembayaran GROUP BY status ORDER BY status`)
	for rows.Next() {
		var s perStatus
		rows.Scan(&s.Status, &s.Jumlah, &s.Total)
		statuses = append(statuses, s)
	}
	rows.Close()
	harian := []perHari{}
	rows, _ = a.db.QueryContext(ctx, `SELECT to_char(dibayar_at::date,'YYYY-MM-DD'), count(*), sum(jumlah) FROM pembayaran
		WHERE status='BERHASIL' AND dibayar_at > now() - interval '30 days' GROUP BY 1 ORDER BY 1`)
	for rows.Next() {
		var h perHari
		rows.Scan(&h.Tanggal, &h.Transaksi, &h.Pendapatan)
		harian = append(harian, h)
	}
	rows.Close()
	var pendapatan int64
	var belumSinkron int
	a.db.QueryRowContext(ctx, `SELECT coalesce(sum(jumlah),0) FROM pembayaran WHERE status='BERHASIL'`).Scan(&pendapatan)
	a.db.QueryRowContext(ctx, `SELECT count(*) FROM pembayaran WHERE tersinkron=false`).Scan(&belumSinkron)
	writeJSON(w, 200, map[string]any{
		"total_pendapatan": pendapatan,
		"belum_tersinkron": belumSinkron,
		"per_status":       statuses,
		"per_hari":         harian,
	})
}
