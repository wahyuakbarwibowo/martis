# 📌 Martis Roadmap & TODO

Rencana pengembangan dan daftar fitur yang perlu diimplementasikan untuk rilis berikutnya.

---

## 🎯 High Priority (v0.1.0)

### 1. Collections & Request History (Persistence)
- [x] **Request History**: Simpan 100 request terakhir ke JSON di `~/martis/history.json`.
- [x] **Collections / Workspaces**: Simpan request, buat folder, pilih, muat, atau hapus request tersimpan.
- [x] **Collection organization**: Rename folder/request (`r`), duplicate request (`y`), dan pindahkan request antar-folder (`m`).
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
- [x] Preset autentikasi untuk mode populer (F3; isi kredensial di tab Auth):
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
- [x] **Form-data file picker**: Pilih file lokal dengan Finder (macOS), picker internal (platform lain), mouse, atau `Ctrl+F`; mendukung banyak field/file dan cURL `-F field=@file;type=mime`.
- [x] **Form-data editor**: Kelola banyak field teks (`field=value`) dan file (`@field=/path`) langsung dari editor TUI.
- [x] **Export to cURL**: Tombol/shortcut untuk mengekspor konfigurasi request saat ini menjadi perintah cURL terminal.

---

## 🛠️ Low Priority & Polishing (v0.3.0+)

### 7. Custom Themes & UI Polish
- [x] Pilihan color scheme (Dracula, Catppuccin, Nord, Tokyo Night, Monokai).
- [x] Full mouse support (klik untuk fokus elemen/tab, pilih file, serta scroll sidebar dan response viewer).

### 8. Testing & Assertion Scripts
- [x] Status code & response assertion sederhana (e.g. `Status == 200`, `json.id != nil`).
- [x] Performance benchmark runner (mengirim N request berturut-turut untuk mengukur rata-rata latency).

### 9. Package Manager Distribution
- [x] Homebrew tap formula untuk macOS (`brew install wahyuakbarwibowo/tap/martis`).
- [x] Arch Linux AUR package (`PKGBUILD`).
- [ ] Scoop manifest untuk Windows (ditunda, fokus ke sistem Unix).
- [x] Chocolatey package template untuk Windows.

### 10. External Collection Import
- [x] Import Postman v2 dan OpenAPI 3 melalui `martis import <file>`.
- [x] Import environment Postman ke `~/martis/environments/*.env`.

### 11. Workflow & Response Tools (v0.4.0)
- [x] Simpan nilai response ke variabel environment (`set token = json.access_token`).
- [x] Collection runner CLI: `martis run [--env <nama>] <folder>` untuk CI.
- [x] Buka TUI langsung dengan `martis <url>` atau `martis curl ...`.
- [x] Filter response dengan JSON path (`json.data.0.id`).
- [x] Diff response dengan response sebelumnya (`Ctrl+D`).
- [x] Konfirmasi sebelum menampilkan response > 1 MB.
