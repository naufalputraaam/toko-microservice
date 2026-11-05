// Program seed data dan tes end-to-end. Semua request lewat API Gateway.
//
//	docker compose run --rm tests seed   -> isi data contoh (kategori, produk, akun demo)
//	docker compose run --rm tests e2e    -> jalankan semua skenario pengujian
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	baseURL    = env("BASE_URL", "http://localhost:8080")
	adminEmail = env("ADMIN_EMAIL", "admin@toko.local")
	adminPass  = env("ADMIN_PASSWORD", "admin123")
	client     = &http.Client{Timeout: 10 * time.Second}
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

type M = map[string]any

// call mengirim request dan mengembalikan status code + body (sudah di-decode).
func call(method, path, token string, body any) (int, M, []any) {
	var buf io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, baseURL+path, buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, M{"error": err.Error()}, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var obj M
	var arr []any
	if len(raw) > 0 && raw[0] == '[' {
		json.Unmarshal(raw, &arr)
	} else {
		json.Unmarshal(raw, &obj)
	}
	return resp.StatusCode, obj, arr
}

func login(email, pass string) string {
	_, r, _ := call("POST", "/pelanggan/login", "", M{"email": email, "password": pass})
	t, _ := r["token"].(string)
	return t
}

// tungguGateway menunggu sampai semua service siap (maks. ~60 detik).
func tungguGateway() {
	for i := 0; i < 30; i++ {
		ok := true
		for _, p := range []string{"/katalog/kategori", "/pelanggan/verify", "/order/keranjang", "/pembayaran"} {
			if s, _, _ := call("GET", p, "", nil); s == 0 || s >= 500 {
				ok = false
			}
		}
		if ok {
			return
		}
		fmt.Println("menunggu semua service siap...")
		time.Sleep(2 * time.Second)
	}
}

func main() {
	mode := "e2e"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	tungguGateway()
	switch mode {
	case "seed":
		seed()
	case "e2e":
		os.Exit(e2e())
	default:
		fmt.Println("pakai: tests seed | tests e2e")
		os.Exit(2)
	}
}

// ================= SEED =================

func seed() {
	admin := login(adminEmail, adminPass)
	if admin == "" {
		fmt.Println("gagal login admin")
		os.Exit(1)
	}
	data := map[string][]M{
		"Elektronik": {
			{"nama": "Mouse Wireless", "harga": 85000, "stok": 40, "deskripsi": "Mouse 2.4GHz, baterai AA"},
			{"nama": "Keyboard Mekanik", "harga": 450000, "stok": 15, "deskripsi": "Switch blue, 87 tombol"},
			{"nama": "Headset Gaming", "harga": 320000, "stok": 10, "deskripsi": "Mic noise cancelling"},
			{"nama": "Flashdisk 64GB", "harga": 75000, "stok": 3, "deskripsi": "USB 3.1"},
		},
		"Buku": {
			{"nama": "Belajar Golang", "harga": 120000, "stok": 25, "deskripsi": "Dasar sampai REST API"},
			{"nama": "Sistem Terdistribusi", "harga": 150000, "stok": 12, "deskripsi": "Konsep dan praktik"},
		},
		"Alat Tulis": {
			{"nama": "Pulpen Gel (12 pcs)", "harga": 36000, "stok": 100, "deskripsi": "Tinta hitam 0.5mm"},
			{"nama": "Buku Catatan A5", "harga": 18000, "stok": 60, "deskripsi": "80 lembar bergaris"},
		},
	}
	_, _, produkAda := call("GET", "/katalog/produk", "", nil)
	sudah := map[string]bool{}
	for _, p := range produkAda {
		sudah[p.(M)["nama"].(string)] = true
	}
	for nama, produk := range data {
		s, k, _ := call("POST", "/katalog/kategori", admin, M{"nama": nama})
		var katID any
		if s == 201 {
			katID = k["id"]
		} else { // sudah ada, cari id-nya
			_, _, list := call("GET", "/katalog/kategori", "", nil)
			for _, x := range list {
				if x.(M)["nama"] == nama {
					katID = x.(M)["id"]
				}
			}
		}
		for _, p := range produk {
			if sudah[p["nama"].(string)] {
				continue
			}
			p["kategori_id"] = katID
			s, _, _ := call("POST", "/katalog/produk", admin, p)
			fmt.Printf("produk %-22s -> %d\n", p["nama"], s)
		}
	}
	// Akun pelanggan demo + alamat
	s, _, _ := call("POST", "/pelanggan/register", "", M{"nama": "Budi Santoso", "email": "budi@toko.local", "password": "budi123", "telepon": "081234567890"})
	if s == 201 {
		budi := login("budi@toko.local", "budi123")
		call("POST", "/pelanggan/alamat", budi, M{"label": "Kos", "penerima": "Budi Santoso", "telepon": "081234567890",
			"alamat": "Jl. Salemba Raya No. 10", "kota": "Jakarta Pusat", "kode_pos": "10430"})
	}
	fmt.Println("\nSeed selesai.")
	fmt.Println("  Admin     :", adminEmail, "/", adminPass)
	fmt.Println("  Pelanggan : budi@toko.local / budi123")
}

// ================= E2E =================

var lulus, gagal int

func cek(nama string, ok bool, detail ...any) {
	if ok {
		lulus++
		fmt.Println("  [LULUS]", nama)
	} else {
		gagal++
		fmt.Println("  [GAGAL]", nama, fmt.Sprint(detail...))
	}
}

func stokProduk(id any) float64 {
	_, p, _ := call("GET", fmt.Sprintf("/katalog/produk/%v", id), "", nil)
	f, _ := p["stok"].(float64)
	return f
}

func statusPesanan(token, kode string) string {
	_, p, _ := call("GET", "/order/pesanan/"+kode, token, nil)
	s, _ := p["status"].(string)
	return s
}

func e2e() int {
	uniq := fmt.Sprint(time.Now().UnixNano())
	fmt.Println("Tes end-to-end lewat", baseURL)

	fmt.Println("\n1. Gateway & keamanan")
	s, _, _ := call("GET", "/health", "", nil)
	cek("gateway /health = 200", s == 200, s)
	s, _, _ = call("POST", "/internal/stok/reserve", "", M{})
	cek("endpoint /internal diblokir gateway (403)", s == 403, s)
	s, _, _ = call("GET", "/order/keranjang", "", nil)
	cek("akses keranjang tanpa token = 401", s == 401, s)

	fmt.Println("\n2. Pelanggan service")
	email := "tes" + uniq + "@mail.com"
	s, p, _ := call("POST", "/pelanggan/register", "", M{"nama": "Tester", "email": email, "password": "rahasia1"})
	cek("registrasi = 201", s == 201, s, p)
	s, _, _ = call("POST", "/pelanggan/register", "", M{"nama": "Tester", "email": email, "password": "rahasia1"})
	cek("email duplikat = 409", s == 409, s)
	s, _, _ = call("POST", "/pelanggan/register", "", M{"nama": "X", "email": "bukan-email", "password": "1"})
	cek("data tidak valid = 400", s == 400, s)
	s, _, _ = call("POST", "/pelanggan/login", "", M{"email": email, "password": "salah"})
	cek("login password salah = 401", s == 401, s)
	user := login(email, "rahasia1")
	cek("login berhasil dapat token JWT", user != "")
	s, p, _ = call("GET", "/pelanggan/verify", user, nil)
	cek("verifikasi token = 200", s == 200 && p["email"] == email, s)
	s, p, _ = call("PUT", "/pelanggan/profil", user, M{"telepon": "0811"})
	cek("ubah profil", s == 200 && p["telepon"] == "0811", s)
	s, al, _ := call("POST", "/pelanggan/alamat", user, M{"penerima": "Tester", "alamat": "Jl. Uji 1", "kota": "Bogor"})
	cek("tambah alamat = 201", s == 201, s)
	alamatID := al["id"]

	fmt.Println("\n3. Katalog service")
	admin := login(adminEmail, adminPass)
	cek("login admin", admin != "")
	s, _, _ = call("POST", "/katalog/produk", user, M{"nama": "X", "harga": 1})
	cek("pelanggan biasa tidak boleh tambah produk (403)", s == 403, s)
	s, k, _ := call("POST", "/katalog/kategori", admin, M{"nama": "Uji " + uniq})
	cek("admin tambah kategori = 201", s == 201, s)
	s, pr, _ := call("POST", "/katalog/produk", admin, M{"nama": "Produk Uji " + uniq, "harga": 10000, "stok": 5, "kategori_id": k["id"]})
	cek("admin tambah produk (stok 5) = 201", s == 201, s, pr)
	pid := pr["id"]
	s, _, list := call("GET", fmt.Sprintf("/katalog/produk?kategori_id=%v", k["id"]), "", nil)
	cek("filter produk per kategori", s == 200 && len(list) == 1, s)
	s, _, _ = call("POST", "/katalog/produk", admin, M{"nama": "Minus", "harga": -1})
	cek("harga negatif ditolak (400)", s == 400, s)

	fmt.Println("\n4. Keranjang & checkout -> bayar sukses")
	s, _, _ = call("POST", "/order/keranjang", user, M{"produk_id": 999999, "qty": 1})
	cek("produk tidak ada di katalog = 404", s == 404, s)
	s, kr, _ := call("POST", "/order/keranjang", user, M{"produk_id": pid, "qty": 2})
	cek("tambah ke keranjang = 201, total 20000", s == 201 && kr["total"] == float64(20000), s, kr)
	s, ps, _ := call("POST", "/order/checkout", user, M{"alamat_id": alamatID})
	kode, _ := ps["kode"].(string)
	cek("checkout = 201 MENUNGGU_PEMBAYARAN", s == 201 && ps["status"] == "MENUNGGU_PEMBAYARAN", s, ps)
	cek("stok berkurang 5 -> 3 (reserve)", stokProduk(pid) == 3, stokProduk(pid))
	_, kr, _ = call("GET", "/order/keranjang", user, nil)
	cek("keranjang kosong setelah checkout", len(kr["items"].([]any)) == 0)
	_, by, _ := call("GET", "/pembayaran/order/"+kode, user, nil)
	cek("tagihan dibuat di pembayaran-service (PENDING)", by["status"] == "PENDING" && by["jumlah"] == float64(20000), by)
	s, by, _ = call("POST", fmt.Sprintf("/pembayaran/%v/bayar", by["id"]), user, M{"metode": "transfer", "simulasi": "sukses"})
	cek("simulasi bayar sukses = BERHASIL & tersinkron", s == 200 && by["status"] == "BERHASIL" && by["tersinkron"] == true, s, by)
	cek("status pesanan jadi DIBAYAR", statusPesanan(user, kode) == "DIBAYAR", statusPesanan(user, kode))
	s, _, _ = call("POST", fmt.Sprintf("/pembayaran/%v/bayar", by["id"]), user, M{})
	cek("bayar dua kali ditolak (409)", s == 409, s)
	s, _, _ = call("PUT", "/order/admin/pesanan/"+kode+"/status", admin, M{"status": "SELESAI"})
	cek("lompat ke SELESAI sebelum DIKIRIM ditolak (409)", s == 409, s)
	call("PUT", "/order/admin/pesanan/"+kode+"/status", admin, M{"status": "DIKIRIM"})
	s, _, _ = call("PUT", "/order/admin/pesanan/"+kode+"/status", admin, M{"status": "SELESAI"})
	cek("admin: DIKIRIM -> SELESAI", s == 200 && statusPesanan(user, kode) == "SELESAI", s)

	fmt.Println("\n5. Stok tidak cukup")
	call("POST", "/order/keranjang", user, M{"produk_id": pid, "qty": 10})
	s, ps, _ = call("POST", "/order/checkout", user, M{"alamat_id": alamatID})
	cek("checkout melebihi stok = 409", s == 409, s, ps)
	cek("stok tetap 3", stokProduk(pid) == 3, stokProduk(pid))
	_, _, riw := call("GET", "/order/pesanan", user, nil)
	cek("pesanan tercatat GAGAL", len(riw) > 0 && riw[0].(M)["status"] == "GAGAL")
	call("PUT", fmt.Sprintf("/order/keranjang/%v", pid), user, M{"qty": 1})

	fmt.Println("\n6. Pembayaran gagal -> kompensasi (stok dikembalikan)")
	s, ps, _ = call("POST", "/order/checkout", user, M{"alamat_id": alamatID})
	kode2, _ := ps["kode"].(string)
	cek("checkout qty 1, stok 3 -> 2", s == 201 && stokProduk(pid) == 2, s)
	_, by, _ = call("GET", "/pembayaran/order/"+kode2, user, nil)
	call("POST", fmt.Sprintf("/pembayaran/%v/bayar", by["id"]), user, M{"simulasi": "gagal"})
	cek("pesanan DIBATALKAN", statusPesanan(user, kode2) == "DIBATALKAN", statusPesanan(user, kode2))
	cek("stok kembali 3", stokProduk(pid) == 3, stokProduk(pid))

	fmt.Println("\n7. Pelanggan membatalkan pesanan")
	call("POST", "/order/keranjang", user, M{"produk_id": pid, "qty": 2})
	_, ps, _ = call("POST", "/order/checkout", user, M{"alamat_id": alamatID})
	kode3, _ := ps["kode"].(string)
	cek("checkout qty 2, stok 3 -> 1", stokProduk(pid) == 1, stokProduk(pid))
	s, _, _ = call("POST", "/order/pesanan/"+kode3+"/batal", user, nil)
	cek("batalkan = 200, stok kembali 3", s == 200 && stokProduk(pid) == 3, s, stokProduk(pid))
	_, by, _ = call("GET", "/pembayaran/order/"+kode3, user, nil)
	cek("tagihan ikut DIBATALKAN", by["status"] == "DIBATALKAN", by["status"])

	fmt.Println("\n8. Laporan admin")
	s, lap, _ := call("GET", "/pembayaran/admin/laporan", admin, nil)
	cek("laporan pembayaran = 200", s == 200 && lap["total_pendapatan"] != nil, s)
	s, _, _ = call("GET", "/pembayaran/admin/laporan", user, nil)
	cek("laporan ditolak untuk pelanggan biasa (403)", s == 403, s)

	fmt.Printf("\nHasil: %d lulus, %d gagal\n", lulus, gagal)
	if gagal > 0 {
		return 1
	}
	return 0
}
