<div align="center">

<img src="assets/martis.png" alt="Martis MLBB" width="130" style="border-radius: 50%; box-shadow: 0 4px 12px rgba(125, 86, 244, 0.4);" />

# ⚡ Martis

**Ultra-lightweight, blazing-fast Terminal User Interface (TUI) REST client.**  
*"3,000 worlds, and not a single worthy API client... until now."*

[![Go Version](https://img.shields.io/github/go-mod/go-version/wahyuakbarwibowo/martis?color=00ADD8)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/wahyuakbarwibowo/martis?color=7D56F4)](https://github.com/wahyuakbarwibowo/martis/releases)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue)](https://github.com/wahyuakbarwibowo/martis/releases)

</div>

---

## 🌟 Highlights

- 📁 **Collections & Persistence**: Simpan request endpoint ke Collection lokal (`~/martis/collections.json`), muat ulang kapan saja, atau hapus request lama.
- ⚔️ **Ashura King of REST Clients**: Menaklukkan ribuan request endpoint dengan kecepatan instan dan memory footprint super hemat (< 20MB RAM).
- 🖥️ **Split-Screen Responsive**: Request Builder di panel kiri dan Response Viewer di panel kanan.
- ⌨️ **Keyboard First**: Navigasi intuitif menggunakan tombol `Tab`, `Shift+Tab`, dan shortcut tanpa perlu mouse.
- 📡 **Built-in Async Engine**: Eksekusi HTTP non-blocking menggunakan native goroutines dan Go standard library `net/http`.
- 🎨 **Beautiful Formatting**: Dilengkapi status code badges Lipgloss, pretty-printed JSON body, metadata latency, dan size metrics.
- 📦 **Cross-Platform**: Binary mandiri siap jalan untuk macOS (Apple Silicon & Intel), Linux, dan Windows.

---

## 🚀 Quick Install

### macOS / Linux (via Make)
```bash
git clone https://github.com/wahyuakbarwibowo/martis.git
cd martis
make install
```
Setelah terpasang, cukup jalankan:
```bash
martis
```

### Install via Shell Script
```bash
curl -fsSL https://raw.githubusercontent.com/wahyuakbarwibowo/martis/main/install.sh | bash
```

### Install via Package Manager

Release binaries are also distributed through package managers; Go is not required.

```bash
# Homebrew (macOS/Linux)
brew tap wahyuakbarwibowo/tap
brew install martis

# Arch Linux (AUR)
yay -S martis
```

For Windows Chocolatey, install the published package with `choco install martis -y`.

### Via Go Toolchain
```bash
go install github.com/wahyuakbarwibowo/martis@latest
```

### Pre-built Binary
Unduh binary mandiri langsung dari halaman [GitHub Releases](https://github.com/wahyuakbarwibowo/martis/releases).

---

## ⌨️ Keyboard Shortcuts

| Shortcut | Aksi |
|---|---|
| `Tab` / `Shift+Tab` | Pindah fokus antar elemen UI |
| `Ctrl+P` | **Buka modal Collections** (pilih / muat / hapus request tersimpan) |
| `Ctrl+N` | Buat folder collection |
| `Ctrl+E` | **Simpan request aktif ke Collection** |
| `Ctrl+T` | Ganti tab konfigurasi request (*Headers* / *Raw JSON* / *Form-Data*) |
| `Ctrl+I` | Impor cURL dari clipboard; jika clipboard kosong, buka editor impor |
| `Ctrl+F` | Buka Finder untuk memilih file lokal saat tab Form-Data aktif |
| `Ctrl+X` | Ekspor request aktif sebagai perintah cURL ke clipboard |
| `Ctrl+H` | Pilih request dari history |
| `Ctrl+G` | Pilih file environment `.env` |
| `F2` | Pilih tema |
| `F3` | Pilih preset autentikasi |
| `/` | Cari teks di response, atau filter JSON dengan `json.data.0.id` |
| `Ctrl+D` | Bandingkan (diff) response saat ini dengan response sebelumnya |
| `Ctrl+Y` / `Ctrl+O` | Salin response / simpan ke file |
| `Ctrl+B` | Jalankan benchmark request |
| `Ctrl+S` | Kirim HTTP Request |
| `Enter` | Kirim request (saat di URL bar / Send button) atau Konfirmasi modal |
| `d` / `Backspace` | Hapus request terpilih di sidebar atau modal Collection |
| `r` | Rename folder atau request terpilih di sidebar |
| `m` | Pindahkan request terpilih ke folder lain |
| `←` / `→` | Ganti HTTP Method (*GET*, *POST*, *PUT*, *DELETE*, *PATCH*, *HEAD*) |
| `↑` / `↓` / `j` / `k` | Scroll response viewer (saat fokus di viewport) atau navigasi Collection |
| `q` / `Ctrl+C` | Keluar dari aplikasi |

---

## 🛠️ CLI Commands

```bash
martis             # Buka antarmuka TUI
martis collections # Tampilkan daftar request di collection
martis import api.json # Impor Postman v2, OpenAPI 3, atau environment Postman
martis run --env prod Auth   # Jalankan semua request di folder "Auth" (exit 1 jika ada yang gagal, cocok untuk CI)
martis https://api.test/users  # Buka TUI dengan URL tersebut
martis curl -H 'X-A: b' https://api.test  # Buka TUI dengan request dari perintah cURL
martis version     # Tampilkan versi terpasang
martis update      # Cek & perbarui aplikasi dari upstream
martis help        # Tampilkan ringkasan bantuan
```

---

## 🗺️ Roadmap & Kontribusi
Pada tab Form-Data, isi satu baris per field (`name=value`) atau file (`@avatar=/path/to/avatar.png`). Impor cURL mendukung method, URL, headers, autentikasi dasar, body file (`--data-binary @file`), dan banyak field form-data/file (termasuk atribut MIME seperti `;type=image/png`). Pada macOS, tekan `Ctrl+F` atau klik editor file untuk membuka Finder dan memilih file lokal; platform lain memakai picker internal. File environment disimpan di `~/martis/environments/`; gunakan `{{variable}}` pada URL, header, atau body.

Tab Assertions menerima satu baris per aturan:

```text
Status == 200
json.data.id != nil
set token = json.access_token
```

Baris `set` menyimpan nilai dari response ke environment aktif, sehingga request berikutnya bisa memakai `{{token}}` (juga berlaku saat `martis run`). Response di atas 1 MB tidak langsung ditampilkan: tekan `Enter` di panel response untuk tetap menampilkan, `/` untuk filter, atau `Ctrl+O` untuk menyimpan ke file.

Kontribusi dan pull request selalu disambut dengan baik!

---

## 📄 Lisensi
Didistribusikan di bawah lisensi [MIT](LICENSE).
