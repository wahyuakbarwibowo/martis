#!/usr/bin/env bash
set -e

# Martis TUI Installer Script for macOS & Unix-like systems
BINARY_NAME="martis"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Fallback ke ~/.local/bin jika /usr/local/bin tidak writable dan tanpa sudo
if [ ! -w "${INSTALL_DIR}" ] && ! command -v sudo &> /dev/null; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
fi

echo "⚡ Menyiapkan Martis TUI untuk macOS/Unix..."

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Error: 'go' tidak ditemukan. Pastikan Golang sudah terpasang di sistem Anda."
    exit 1
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${ARCH}" in
    x86_64)  GOARCH="amd64" ;;
    arm64|aarch64) GOARCH="arm64" ;;
    *)       GOARCH="amd64" ;;
esac

echo "Detected OS: ${OS} (${GOARCH})"

# Build binary optimized for current Unix/macOS host
echo "🔨 Compiling binary release..."
CGO_ENABLED=0 GOOS="${OS}" GOARCH="${GOARCH}" go build -ldflags="-s -w" -o "${BINARY_NAME}" .

# Install to system path
echo "📦 Memasang binary ke ${INSTALL_DIR}..."
if [ -w "${INSTALL_DIR}" ]; then
    mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
else
    echo "Memerlukan akses sudo untuk memasang ke ${INSTALL_DIR}:"
    sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
fi

chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo ""
echo "✅ Martis TUI berhasil dipasang!"
echo "🚀 Jalankan langsung di terminal Anda:"
echo "   martis"
