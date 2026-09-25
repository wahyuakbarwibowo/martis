#!/usr/bin/env bash
# Builds a .deb and a tarball (binary + .desktop entry + icon) for martis-desktop.
# Usage: scripts/package-linux.sh <binary> <version> <arch> <outdir>
set -euo pipefail

BIN="$1" VERSION="$2" ARCH="$3" OUT="$4"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "${WORK}"' EXIT
mkdir -p "${OUT}"

DESKTOP_ENTRY="[Desktop Entry]
Type=Application
Name=Martis
Comment=Lightweight REST client
Exec=martis-desktop
Icon=martis
Terminal=false
Categories=Development;Network;
Keywords=rest;http;api;client;"

# Plain tarball used by install.sh.
TAR="${WORK}/tar"
mkdir -p "${TAR}"
cp "${BIN}" "${TAR}/martis-desktop"
cp "${ROOT}/cmd/martis-desktop/icon.png" "${TAR}/martis.png"
printf '%s\n' "${DESKTOP_ENTRY}" > "${TAR}/martis.desktop"
cp "${ROOT}/LICENSE" "${ROOT}/README.md" "${TAR}/"
tar -czf "${OUT}/martis-desktop_${VERSION}_linux_${ARCH}.tar.gz" -C "${TAR}" .

# Debian package.
DEB="${WORK}/deb"
mkdir -p "${DEB}/DEBIAN" "${DEB}/usr/bin" "${DEB}/usr/share/applications" "${DEB}/usr/share/pixmaps"
install -m 0755 "${BIN}" "${DEB}/usr/bin/martis-desktop"
printf '%s\n' "${DESKTOP_ENTRY}" > "${DEB}/usr/share/applications/martis.desktop"
cp "${ROOT}/cmd/martis-desktop/icon.png" "${DEB}/usr/share/pixmaps/martis.png"
cat > "${DEB}/DEBIAN/control" <<CONTROL
Package: martis
Version: ${VERSION}
Architecture: ${ARCH}
Maintainer: Martis <martis@users.noreply.github.com>
Depends: libgtk-3-0, libwebkit2gtk-4.1-0
Section: devel
Priority: optional
Homepage: https://github.com/wahyuakbarwibowo/martis
Description: Lightweight REST client
 Martis desktop app on the system webview.
CONTROL
dpkg-deb --root-owner-group --build "${DEB}" "${OUT}/martis-desktop_${VERSION}_${ARCH}.deb" >/dev/null
