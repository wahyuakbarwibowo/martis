#!/usr/bin/env bash
set -euo pipefail

# Martis TUI installer for macOS & Linux: downloads the latest release binary.
REPO="wahyuakbarwibowo/martis"
BINARY_NAME="martis"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Fallback ke ~/.local/bin jika /usr/local/bin tidak writable dan tanpa sudo
if [ ! -w "${INSTALL_DIR}" ] && ! command -v sudo &> /dev/null; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
fi

echo "⚡ Menyiapkan Martis TUI..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    darwin|linux) ;;
    *) echo "❌ OS tidak didukung: ${OS}"; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "❌ Arsitektur tidak didukung: $(uname -m)"; exit 1 ;;
esac

# Versi bisa dipaksa via VERSION=v0.4.0; default ambil rilis terbaru.
VERSION="${VERSION:-$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" | sed 's|.*/tag/||')}"
case "${VERSION}" in
    v*) ;;
    *) echo "❌ Gagal menentukan versi rilis terbaru"; exit 1 ;;
esac

ARCHIVE="${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "📥 Mengunduh ${ARCHIVE} (${VERSION})..."
curl -fsSL -o "${TMP_DIR}/${ARCHIVE}" "${BASE_URL}/${ARCHIVE}"
curl -fsSL -o "${TMP_DIR}/checksums.txt" "${BASE_URL}/checksums.txt"

echo "🔐 Memverifikasi checksum..."
EXPECTED="$(grep " ${ARCHIVE}\$" "${TMP_DIR}/checksums.txt" | cut -d' ' -f1)"
if command -v sha256sum &> /dev/null; then
    ACTUAL="$(sha256sum "${TMP_DIR}/${ARCHIVE}" | cut -d' ' -f1)"
else
    ACTUAL="$(shasum -a 256 "${TMP_DIR}/${ARCHIVE}" | cut -d' ' -f1)"
fi
if [ -z "${EXPECTED}" ] || [ "${EXPECTED}" != "${ACTUAL}" ]; then
    echo "❌ Checksum tidak cocok, instalasi dibatalkan"
    exit 1
fi

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}" "${BINARY_NAME}"
chmod +x "${TMP_DIR}/${BINARY_NAME}"

echo "📦 Memasang binary ke ${INSTALL_DIR}..."
if [ -w "${INSTALL_DIR}" ]; then
    mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
else
    echo "Memerlukan akses sudo untuk memasang ke ${INSTALL_DIR}:"
    sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo ""
echo "✅ Martis TUI ${VERSION} berhasil dipasang!"
echo "🚀 Jalankan langsung di terminal Anda:"
echo "   martis"
