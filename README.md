# Martis TUI - Ultra-Light REST Client

Alternatif Postman berbasis Terminal User Interface (TUI) yang ultra-ringan, responsif, dan hemat memori, dibangun menggunakan Go dan Charmbracelet stack.

## Tech Stack
- **Go (Golang)**: Standar `net/http` untuk client HTTP non-blocking.
- **TUI Engine**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **UI Components**: [Bubbles](https://github.com/charmbracelet/bubbles) (`textinput`, `textarea`, `viewport`, `spinner`)

## Fitur Utama
1. **Split-View Responsive**: Request Builder di sisi kiri dan Response Viewer di sisi kanan.
2. **Method Selector**: Beralih antara `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`.
3. **Tab Configuration**:
   - **Headers**: Custom key-value header & Authorization token/credentials.
   - **Body (Raw JSON)**: Multiline textarea dengan pretty format.
   - **Body (Form-Data / Upload)**: Field name dan local file path untuk multipart testing.
4. **Asynchronous Request**: Eksekusi HTTP non-blocking dengan animasi spinner loading dan timeout protection.
5. **Response Metrics & Pretty-Print**:
   - Status badge dengan kode warna (Hijau 2xx, Kuning 3xx, Merah 4xx/5xx).
   - Latency timer (ms) dan payload content-length.
   - Response headers & pretty-printed JSON response viewport dengan scrolling.

## Cara Menjalankan

```bash
# Jalankan langsung
go run main.go

# Atau build binary
go build -o martis-tui .
./martis-tui
```

## Navigasi Keyboard
- `Tab` / `Shift+Tab`: Pindah fokus antar elemen (Method -> URL -> Config Tabs -> Inputs -> Send -> Response Viewport).
- `Ctrl+T`: Pindah antar tab konfigurasi dengan cepat (Headers / Raw JSON / Form-Data).
- `Ctrl+S` atau `Enter` di URL/Send: Kirim HTTP request.
- `↑` / `↓` / `j` / `k`: Navigasi dan scroll viewport response saat aktif.
- `q` (saat di Response Viewport) atau `Ctrl+C`: Keluar dari aplikasi.

---

## 🚀 Instalasi & Release (Ready to Install)

### 1. Install ke Sistem via Make (Rekomendasi Lokal)
Mengompilasi binary release teroptimasi dan memasangnya langsung ke `/usr/local/bin/martis`:
```bash
make install
```
Setelah itu, Martis dapat langsung dipanggil dari mana saja:
```bash
martis
```
Untuk menghapus:
```bash
make uninstall
```

### 2. Install via Script (`install.sh`)
```bash
./install.sh
```

### 3. Cross-Compile Multi-Platform Release
Untuk membuat binary release untuk Linux, macOS, dan Windows sekaligus:
```bash
make release-all
```
File output akan tersedia di direktori `dist/`:
- `dist/martis-darwin-arm64` (macOS Apple Silicon M1/M2/M3)
- `dist/martis-darwin-amd64` (macOS Intel)
- `dist/martis-linux-amd64` (Linux x86_64)
- `dist/martis-linux-arm64` (Linux ARM64)
- `dist/martis-windows-amd64.exe` (Windows x64)

### 4. CI/CD Otomatis (GitHub Actions & GoReleaser)
Telah disediakan konfigurasi:
- [.goreleaser.yaml](.goreleaser.yaml)
- [.github/workflows/release.yml](.github/workflows/release.yml)

Saat Anda membuat git tag baru (misal: `git tag v1.0.0 && git push origin v1.0.0`), GitHub Actions akan secara otomatis mengompilasi dan mempublikasikan tar.gz/zip beserta checksums ke halaman GitHub Releases.
