#!/usr/bin/env bash
set -euo pipefail

# Martis installer for macOS & Linux: downloads the latest release binary.
# MARTIS_DESKTOP=1 also installs the desktop app.
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

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

sha256() {
    if command -v sha256sum &> /dev/null; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

# install_binary <name> <archive> <checksum file>: download, verify, and install one binary.
install_binary() {
    local name="$1" archive="$2" sums="$3"
    echo "📥 Mengunduh ${archive} (${VERSION})..."
    curl -fsSL -o "${TMP_DIR}/${archive}" "${BASE_URL}/${archive}"
    curl -fsSL -o "${TMP_DIR}/${sums}" "${BASE_URL}/${sums}"

    echo "🔐 Memverifikasi checksum..."
    local expected
    expected="$(grep " ${archive}\$" "${TMP_DIR}/${sums}" | cut -d' ' -f1)"
    if [ -z "${expected}" ] || [ "${expected}" != "$(sha256 "${TMP_DIR}/${archive}")" ]; then
        echo "❌ Checksum ${archive} tidak cocok, instalasi dibatalkan"
        exit 1
    fi

    tar -xzf "${TMP_DIR}/${archive}" -C "${TMP_DIR}" "${name}"
    chmod +x "${TMP_DIR}/${name}"
    echo "📦 Memasang ${name} ke ${INSTALL_DIR}..."
    if [ -w "${INSTALL_DIR}" ]; then
        mv "${TMP_DIR}/${name}" "${INSTALL_DIR}/${name}"
    else
        echo "Memerlukan akses sudo untuk memasang ke ${INSTALL_DIR}:"
        sudo mv "${TMP_DIR}/${name}" "${INSTALL_DIR}/${name}"
    fi
}

install_binary "${BINARY_NAME}" "${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.tar.gz" checksums.txt

# Aplikasi desktop opsional: MARTIS_DESKTOP=1
if [ "${MARTIS_DESKTOP:-0}" = "1" ]; then
    DESKTOP_ARCHIVE="${BINARY_NAME}-desktop_${VERSION#v}_${OS}_${ARCH}.tar.gz"
    install_binary "${BINARY_NAME}-desktop" "${DESKTOP_ARCHIVE}" "${DESKTOP_ARCHIVE}.sha256"
fi

echo ""
echo "✅ Martis ${VERSION} berhasil dipasang!"
echo "🚀 Jalankan di terminal:"
echo "   martis            # TUI"
if [ "${MARTIS_DESKTOP:-0}" = "1" ]; then
    echo "   martis-desktop    # aplikasi desktop"
fi
