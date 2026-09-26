# Kontribusi ke Martis

Terima kasih sudah mau membantu! Martis adalah proyek iseng non-komersial, jadi kontribusi kecil pun sangat berarti.

## Melaporkan masalah

- **Bug**: buka [issue baru](https://github.com/wahyuakbarwibowo/martis/issues/new/choose) dengan template **Bug report**. Sertakan versi (`martis version`), OS, aplikasi yang dipakai (TUI atau desktop), dan langkah untuk mereproduksi.
- **Ide fitur**: pakai template **Feature request** dan jelaskan masalah yang ingin diselesaikan, bukan hanya solusinya.
- **Celah keamanan**: jangan buka issue publik. Laporkan secara privat lewat tab [Security → Report a vulnerability](https://github.com/wahyuakbarwibowo/martis/security/advisories/new).
- Cari dulu di [issue yang ada](https://github.com/wahyuakbarwibowo/martis/issues) agar tidak dobel.
- Jangan tempel token, password, atau URL internal di issue. Samarkan dulu.

## Menyiapkan lingkungan

Butuh Go sesuai versi di `go.mod`. Aplikasi desktop juga butuh CGO (Xcode Command Line Tools di macOS, `libgtk-3-dev` + `libwebkit2gtk-4.1-dev` di Linux).

```bash
git clone https://github.com/wahyuakbarwibowo/martis.git
cd martis
make run            # jalankan TUI
make desktop-run    # build dan buka aplikasi desktop
make test           # go test -race ./...
```

Struktur kode dan gaya penulisan ada di [`AGENTS.md`](AGENTS.md).

## Alur kerja

1. Fork repo lalu buat branch dari `main` dengan prefix sesuai jenis perubahan: `feat/…`, `fix/…`, `docs/…`, `refactor/…`, `chore/…`, `ci/…`.
2. Satu perubahan logis per commit. Pesan commit mengikuti [Conventional Commits](https://www.conventionalcommits.org/):

   ```text
   feat: add GraphQL body mode
   fix: keep URL row clickable on narrow terminals
   ```

   Subjek maksimal 72 karakter, kalimat perintah, tanpa titik di akhir. Isi body menjelaskan *kenapa*, bukan *apa*.
3. Sebelum membuka PR, jalankan:

   ```bash
   make fmt && make vet && make test
   ```

4. Buka pull request ke `main` dan isi template-nya. Sertakan screenshot untuk perubahan tampilan TUI atau desktop.

`main` dilindungi: semua perubahan masuk lewat pull request, dan force push tidak diizinkan.

## Hal yang perlu dijaga

- TUI harus tetap bisa dibuild dengan `CGO_ENABLED=0`; jangan impor Wails di luar `cmd/martis-desktop/`.
- Teks CLI yang terlihat pengguna ditulis dalam Bahasa Indonesia.
- Utamakan solusi ringan: pustaka standar dulu, dependensi baru hanya bila benar-benar perlu.
