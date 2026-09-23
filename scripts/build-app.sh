#!/bin/bash
# 构建 macOS 应用包 dist/Gitra.app（双击即可启动 TUI）。
set -euo pipefail
cd "$(dirname "$0")/.."

APP="dist/Gitra.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

VERSION="${VERSION:-dev}"
SHORT_VERSION="${VERSION#v}"
case "$SHORT_VERSION" in
  [0-9]*) : ;;
  *) SHORT_VERSION="0.0.0" ;;
esac

echo "构建 gitra 二进制 ... (version=${VERSION}, 通用架构)"
LD_FLAGS="-s -w -X github.com/zhanhd/gitra/internal/version.Version=${VERSION}"
# 同时构建两种架构再合并，Intel 与 Apple Silicon 的 Mac 都能运行。
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$LD_FLAGS" -o "$APP/Contents/Resources/gitra.arm64" ./cmd/gitra
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$LD_FLAGS" -o "$APP/Contents/Resources/gitra.amd64" ./cmd/gitra
lipo -create -output "$APP/Contents/Resources/gitra" \
  "$APP/Contents/Resources/gitra.arm64" "$APP/Contents/Resources/gitra.amd64"
rm -f "$APP/Contents/Resources/gitra.arm64" "$APP/Contents/Resources/gitra.amd64"
chmod +x "$APP/Contents/Resources/gitra"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>gitra</string>
  <key>CFBundleDisplayName</key><string>gitra</string>
  <key>CFBundleIdentifier</key><string>dev.gitra.app</string>
  <key>CFBundleVersion</key><string>${SHORT_VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${SHORT_VERSION}</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleExecutable</key><string>GitraLauncher</string>
  <key>LSMinimumSystemVersion</key><string>12.0</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST

cat > "$APP/Contents/MacOS/GitraLauncher" <<'LAUNCH'
#!/bin/bash
# TUI 需要终端窗口：清掉浏览器下载带来的隔离标记，再让 Terminal 运行 app 内的 gitra。
DIR="$(cd "$(dirname "$0")/../Resources" && pwd)"
xattr -dr com.apple.quarantine "$DIR" 2>/dev/null || true
exec open -a Terminal "$DIR/gitra"
LAUNCH
chmod +x "$APP/Contents/MacOS/GitraLauncher"

# Ad-hoc 签名：macOS 不会再把它当成“已损坏”；没有付费证书时这是能做到的最好情况。
codesign --force --sign - "$APP/Contents/Resources/gitra"
codesign --force --sign - "$APP"
codesign --verify --deep --strict "$APP"

echo "已生成 ${APP}（可拖入「应用程序」文件夹）"
