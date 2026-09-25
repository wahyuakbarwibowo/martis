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

# fetch <archive> <checksum file>: download an archive and verify it, leaving it in TMP_DIR.
fetch() {
    local archive="$1" sums="$2" expected
    echo "📥 Mengunduh ${archive} (${VERSION})..."
    curl -fsSL -o "${TMP_DIR}/${archive}" "${BASE_URL}/${archive}"
    curl -fsSL -o "${TMP_DIR}/${sums}" "${BASE_URL}/${sums}"
    echo "🔐 Memverifikasi checksum..."
    expected="$(grep " ${archive}\$" "${TMP_DIR}/${sums}" | cut -d' ' -f1)"
    if [ -z "${expected}" ] || [ "${expected}" != "$(sha256 "${TMP_DIR}/${archive}")" ]; then
        echo "❌ Checksum ${archive} tidak cocok, instalasi dibatalkan"
        exit 1
    fi
}

# place <source> <target>: move or link into INSTALL_DIR, using sudo only when needed.
place() {
    local cmd="$1" src="$2" dst="$3"
    if [ -w "${INSTALL_DIR}" ]; then
        ${cmd} "${src}" "${dst}"
    else
        echo "Memerlukan akses sudo untuk memasang ke ${INSTALL_DIR}:"
        sudo ${cmd} "${src}" "${dst}"
    fi
}

ARCHIVE="${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
fetch "${ARCHIVE}" checksums.txt
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}" "${BINARY_NAME}"
chmod +x "${TMP_DIR}/${BINARY_NAME}"
echo "📦 Memasang ${BINARY_NAME} ke ${INSTALL_DIR}..."
place mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

# Aplikasi desktop opsional: MARTIS_DESKTOP=1
DESKTOP_NOTE=""
if [ "${MARTIS_DESKTOP:-0}" = "1" ]; then
    if [ "${OS}" = darwin ]; then
        # Martis.app di ~/Applications agar muncul di Spotlight, Launchpad, dan Dock.
        APP_ARCHIVE="Martis_${VERSION#v}_${ARCH}.app.tar.gz"
        fetch "${APP_ARCHIVE}" "${APP_ARCHIVE}.sha256"
        APPS_DIR="${HOME}/Applications"
        mkdir -p "${APPS_DIR}"
        rm -rf "${APPS_DIR}/Martis.app"
        tar -xzf "${TMP_DIR}/${APP_ARCHIVE}" -C "${APPS_DIR}"
        echo "📦 Memasang Martis.app ke ${APPS_DIR}..."
        place "ln -sf" "${APPS_DIR}/Martis.app/Contents/MacOS/martis-desktop" "${INSTALL_DIR}/martis-desktop"
        DESKTOP_NOTE="Martis.app ada di ${APPS_DIR} (cari \"Martis\" di Spotlight)"
    else
        DESKTOP_ARCHIVE="${BINARY_NAME}-desktop_${VERSION#v}_${OS}_${ARCH}.tar.gz"
        fetch "${DESKTOP_ARCHIVE}" "${DESKTOP_ARCHIVE}.sha256"
        mkdir -p "${TMP_DIR}/desktop"
        tar -xzf "${TMP_DIR}/${DESKTOP_ARCHIVE}" -C "${TMP_DIR}/desktop"
        chmod +x "${TMP_DIR}/desktop/martis-desktop"
        echo "📦 Memasang martis-desktop ke ${INSTALL_DIR}..."
        place mv "${TMP_DIR}/desktop/martis-desktop" "${INSTALL_DIR}/martis-desktop"
        # Entri menu aplikasi (GNOME, KDE, dll.) untuk pengguna ini.
        SHARE="${XDG_DATA_HOME:-${HOME}/.local/share}"
        mkdir -p "${SHARE}/applications" "${SHARE}/icons"
        cp "${TMP_DIR}/desktop/martis.png" "${SHARE}/icons/martis.png"
        sed -e "s|^Exec=.*|Exec=${INSTALL_DIR}/martis-desktop|" -e "s|^Icon=.*|Icon=${SHARE}/icons/martis.png|" \
            "${TMP_DIR}/desktop/martis.desktop" > "${SHARE}/applications/martis.desktop"
        DESKTOP_NOTE="Martis ada di menu aplikasi"
    fi
fi

echo ""
echo "✅ Martis ${VERSION} berhasil dipasang!"
echo "🚀 Jalankan di terminal:"
echo "   martis            # TUI"
if [ -n "${DESKTOP_NOTE}" ]; then
    echo "   martis-desktop    # aplikasi desktop"
    echo "   ${DESKTOP_NOTE}"
fi
