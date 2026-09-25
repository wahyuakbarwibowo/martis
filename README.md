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
| `Ctrl+T` | Ganti tab konfigurasi request (*Headers* / *Raw JSON* / *Form-Data*) |
| `Ctrl+S` | Kirim HTTP Request |
| `Enter` | Kirim request (saat di URL bar / Send button) |
| `←` / `→` | Ganti HTTP Method (*GET*, *POST*, *PUT*, *DELETE*, *PATCH*, *HEAD*) |
| `↑` / `↓` / `j` / `k` | Scroll response viewer (saat fokus di viewport) |
| `q` / `Ctrl+C` | Keluar dari aplikasi |

---

## 🛠️ CLI Commands

```bash
martis           # Buka antarmuka TUI
martis version   # Tampilkan versi terpasang
martis update    # Cek & perbarui aplikasi dari upstream
martis help      # Tampilkan ringkasan bantuan
```

---

## 🗺️ Roadmap & Kontribusi
Lihat [TODO.md](TODO.md) untuk melihat daftar rencana fitur berikutnya (Collections, Environments, Auth Presets, cURL import/export). Kontribusi dan pull request selalu disambut dengan baik!

---

## 📄 Lisensi
Didistribusikan di bawah lisensi [MIT](LICENSE).
