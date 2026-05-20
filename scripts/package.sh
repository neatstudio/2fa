#!/usr/bin/env sh
set -eu

APP="2fa"
VERSION="${VERSION:-$(tr -d '[:space:]' < VERSION)}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
DIST_DIR="dist"
MAIN="./cmd/2fa"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

build_one() {
  goos="$1"
  goarch="$2"
  ext=""
  if [ "$goos" = "windows" ]; then
    ext=".exe"
  fi
  name="$APP-$VERSION-$goos-$goarch"
  out="$DIST_DIR/$name/$APP$ext"
  mkdir -p "$DIST_DIR/$name"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
    -o "$out" "$MAIN"
  cp README.md "$DIST_DIR/$name/README.md"
  tar -C "$DIST_DIR" -czf "$DIST_DIR/$name.tar.gz" "$name"
  rm -rf "$DIST_DIR/$name"
}

build_one darwin amd64
build_one darwin arm64
build_one linux amd64
build_one linux arm64
build_one windows amd64

(
  cd "$DIST_DIR"
  shasum -a 256 *.tar.gz > SHA256SUMS
)

./scripts/release-notes.sh "$VERSION" > "$DIST_DIR/RELEASE_NOTES.md"

echo "Packaged $APP $VERSION into $DIST_DIR/"
