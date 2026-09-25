#!/usr/bin/env bash
# Wraps a built martis-desktop binary into Martis.app, a .dmg, and an .app.tar.gz.
# Usage: scripts/package-macos.sh <binary> <version> <arch> <outdir>
set -euo pipefail

BIN="$1" VERSION="$2" ARCH="$3" OUT="$4"
ICON="$(dirname "$0")/../cmd/martis-desktop/icon.png"
WORK="$(mktemp -d)"
trap 'rm -rf "${WORK}"' EXIT
APP="${WORK}/Martis.app"

mkdir -p "${APP}/Contents/MacOS" "${APP}/Contents/Resources" "${OUT}"
cp "${BIN}" "${APP}/Contents/MacOS/martis-desktop"

# Build the .icns from the 1024px source icon.
ICONSET="${WORK}/icon.iconset"
mkdir -p "${ICONSET}"
for size in 16 32 128 256 512; do
    sips -z "${size}" "${size}" "${ICON}" --out "${ICONSET}/icon_${size}x${size}.png" >/dev/null
    sips -z "$((size * 2))" "$((size * 2))" "${ICON}" --out "${ICONSET}/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "${ICONSET}" -o "${APP}/Contents/Resources/Martis.icns"

cat > "${APP}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>Martis</string>
    <key>CFBundleDisplayName</key><string>Martis</string>
    <key>CFBundleIdentifier</key><string>com.github.wahyuakbarwibowo.martis</string>
    <key>CFBundleExecutable</key><string>martis-desktop</string>
    <key>CFBundleIconFile</key><string>Martis</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleShortVersionString</key><string>${VERSION}</string>
    <key>CFBundleVersion</key><string>${VERSION}</string>
    <key>LSApplicationCategoryType</key><string>public.app-category.developer-tools</string>
    <key>LSMinimumSystemVersion</key><string>11.0</string>
    <key>NSHighResolutionCapable</key><true/>
    <key>NSAppTransportSecurity</key><dict><key>NSAllowsArbitraryLoads</key><true/></dict>
</dict>
</plist>
PLIST

# Ad-hoc signature: no Apple Developer ID, but required for arm64 and keeps the bundle intact.
codesign --force --deep --sign - "${APP}"

tar -czf "${OUT}/Martis_${VERSION}_${ARCH}.app.tar.gz" -C "${WORK}" Martis.app

# The .dmg shows Martis.app next to an Applications shortcut for drag-to-install.
STAGE="${WORK}/dmg"
mkdir -p "${STAGE}"
cp -R "${APP}" "${STAGE}/"
ln -s /Applications "${STAGE}/Applications"
hdiutil create -quiet -volname "Martis" -srcfolder "${STAGE}" -ov -format UDZO "${OUT}/Martis_${VERSION}_${ARCH}.dmg"
