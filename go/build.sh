#!/usr/bin/env bash
# Build script for the WOS Gift-Code client (Go).
#
# Usage:
#   ./build.sh            Build one binary for the current platform (-> dist/)
#   ./build.sh all        Build binaries for all target platforms
#   ./build.sh clean      Remove the dist/ directory
set -euo pipefail

cd "$(dirname "$0")"

APP="wos-client"
DIST="../dist"

# Find Go in PATH or common install locations.
GO="${GO:-go}"
if ! command -v "$GO" >/dev/null 2>&1; then
  for p in /usr/local/go/bin/go /opt/homebrew/bin/go "$HOME/go/bin/go"; do
    if [ -x "$p" ]; then GO="$p"; break; fi
  done
fi
if ! command -v "$GO" >/dev/null 2>&1; then
  echo "Error: 'go' was not found. Please install Go or set GO=..." >&2
  exit 1
fi

# Version info for -ldflags (optional).
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
LDFLAGS="-s -w -X main.version=${VERSION}"

# Target platforms for 'all' mode.
PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

build_one() {
  local goos="$1" goarch="$2"
  local out="${DIST}/${APP}-${goos}-${goarch}"
  [ "$goos" = "windows" ] && out="${out}.exe"
  echo "==> ${goos}/${goarch}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    "$GO" build -trimpath -ldflags "$LDFLAGS" -o "$out" .
}

case "${1:-current}" in
  clean)
    rm -rf "$DIST"
    echo "dist/ removed."
    ;;
  one)
    if [ "$#" -ne 3 ]; then
      echo "Usage: $0 one <goos> <goarch>" >&2
      exit 1
    fi
    mkdir -p "$DIST"
    build_one "$2" "$3"
    ;;
  all)
    mkdir -p "$DIST"
    for platform in "${PLATFORMS[@]}"; do
      build_one "${platform%/*}" "${platform#*/}"
    done
    echo "Done. Binaries in ${DIST}/"
    ;;
  current)
    mkdir -p "$DIST"
    goos="$("$GO" env GOOS)"
    goarch="$("$GO" env GOARCH)"
    build_one "$goos" "$goarch"
    echo "Done: ${DIST}/${APP}-${goos}-${goarch}"
    ;;
  *)
    echo "Unknown command: $1" >&2
    echo "Usage: $0 [current|all|one <goos> <goarch>|clean]" >&2
    exit 1
    ;;
esac

