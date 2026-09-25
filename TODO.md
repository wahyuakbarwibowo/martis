# 📌 Martis Roadmap & TODO

Rencana pengembangan dan daftar fitur yang perlu diimplementasikan untuk rilis berikutnya.

---

## 🎯 High Priority (v0.1.0)

### 1. Collections & Request History (Persistence)
- [x] **Request History**: Simpan riwayat request terakhir ke local storage (SQLite embedded atau file JSON di `~/.config/martis/history.json`).
- [x] **Collections / Workspaces**: Simpan dan kelola kumpulan request tersimpan dengan folder/kategori mirip Postman.
- [x] **Quick History Picker**: Shortcut (misal `Ctrl+H`) untuk memilih dan memuat kembali request sebelumnya.

### 2. Environment Variables & Interpolation
- [x] Dukungan file environment (e.g. `local.env`, `prod.env` atau syntax `{{base_url}}`).
- [x] Parser otomatis untuk menggantikan variabel seperti `{{base_url}}/users` dan `{{token}}` saat eksekusi.
- [x] UI picker untuk beralih active environment secara instan.

### 3. Header & Query Params Manager
- [x] Multiple Key-Value pairs untuk Headers dengan tombol Add/Delete row dinamis.
- [x] Tab khusus **Query Parameters** yang otomatis tersinkronisasi dua arah dengan endpoint URL string (`?page=1&limit=10`).

---

## ⚡ Medium Priority (v0.2.0)

### 4. Authentication Helper
- [x] Preset form untuk mode autentikasi populer:
  - **Bearer Token** (input token instan).
  - **Basic Auth** (Username + Password dengan auto base64 encode).
  - **API Key** (Key-Value dengan opsi `Header` atau `Query Params`).
  - **OAuth 2.0** token helper (client credentials).

### 5. Response Viewer Enhancement
- [x] **Tab Response**: Pisahkan view antara **Body**, **Headers**, dan **Cookies**.
- [x] **Search / Find dalam Body**: Fitur pencarian teks (`/`) di dalam viewport JSON response.
- [x] **Copy to Clipboard**: Shortcut untuk menyalin response body atau header ke sistem clipboard.
- [x] **Save Response to File**: Shortcut untuk mengekspor payload response ke file lokal (misal: `response.json`).

### 6. cURL Integration
- [x] **Import from cURL**: Fitur paste perintah `curl -X POST ...` dan otomatis mengisi URL, headers, dan body.
- [x] **Export to cURL**: Tombol/shortcut untuk mengekspor konfigurasi request saat ini menjadi perintah cURL terminal.

---

## 🛠️ Low Priority & Polishing (v0.3.0+)

### 7. Custom Themes & UI Polish
- [x] Pilihan color scheme (Dracula, Catppuccin, Nord, Tokyo Night, Monokai).
- [x] Full mouse support (klik untuk fokus elemen atau ganti tab).

### 8. Testing & Assertion Scripts
- [x] Status code & response assertion sederhana (e.g. `Status == 200`, `json.id != nil`).
- [x] Performance benchmark runner (mengirim N request berturut-turut untuk mengukur rata-rata latency).

### 9. Package Manager Distribution
- [x] Homebrew tap formula untuk macOS (`brew install wahyuakbarwibowo/tap/martis`).
- [x] Arch Linux AUR package (`PKGBUILD`).
- [x] Scoop manifest untuk Windows (`scoop install martis`).
