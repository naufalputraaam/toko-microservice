// common.js: fungsi bersama untuk dashboard toko dan admin.
// Semua request lewat API Gateway (alamat yang sama dengan halaman ini).

const Auth = {
  get token() { try { return localStorage.getItem('token') || ''; } catch { return ''; } },
  get user() { try { return JSON.parse(localStorage.getItem('user') || 'null'); } catch { return null; } },
  simpan(token, user) {
    try { localStorage.setItem('token', token); localStorage.setItem('user', JSON.stringify(user)); } catch {}
  },
  keluar() { try { localStorage.removeItem('token'); localStorage.removeItem('user'); } catch {} },
};

async function api(method, path, body) {
  const opt = { method, headers: { 'Content-Type': 'application/json' } };
  if (Auth.token) opt.headers.Authorization = 'Bearer ' + Auth.token;
  if (body !== undefined) opt.body = JSON.stringify(body);
  const res = await fetch(path, opt);
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { error: text }; }
  if (!res.ok) {
    const err = new Error((data && data.error) || ('HTTP ' + res.status));
    err.status = res.status;
    throw err;
  }
  return data;
}

const rupiah = (n) => 'Rp ' + Number(n || 0).toLocaleString('id-ID');
const tgl = (s) => s ? new Date(s).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' }) : '-';
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const $ = (sel) => document.querySelector(sel);

function toast(pesan, jenis = 'ok') {
  const el = document.createElement('div');
  el.className = 'toast ' + jenis;
  el.textContent = pesan;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), 3500);
}

function badge(status) {
  return `<span class="badge s-${esc(status)}">${esc(String(status).replaceAll('_', ' '))}</span>`;
}

// Bungkus aksi tombol: tampilkan error sebagai toast.
async function aksi(fn) {
  try { await fn(); } catch (e) { toast(e.message, 'err'); }
}

// Cek status semua service lewat gateway (ditampilkan di header).
async function cekLayanan(el) {
  const svc = [['katalog', '/katalog/kategori'], ['pelanggan', '/pelanggan/verify'], ['order', '/order/keranjang'], ['pembayaran', '/pembayaran']];
  const hasil = await Promise.all(svc.map(async ([nama, p]) => {
    try {
      const r = await fetch(p); // tanpa token: cukup cek service merespons
      return [nama, r.status !== 503 && r.status !== 502 && r.status !== 504];
    } catch { return [nama, false]; }
  }));
  el.innerHTML = hasil.map(([n, ok]) => `<span class="dot ${ok ? 'up' : 'down'}" title="${n}: ${ok ? 'aktif' : 'tidak tersedia'}">${n}</span>`).join('');
}
