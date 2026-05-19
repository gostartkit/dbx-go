#!/bin/sh
set -eu

DIST_DIR="${DIST_DIR:-dist}"
VERSION="${VERSION:-v0.2.0}"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

build_target() {
  goos="$1"
  goarch="$2"
  name="dbx_${goos}_${goarch}"
  out_dir="${DIST_DIR}/${name}"

  mkdir -p "$out_dir"
  echo "Building ${name}"
  GOOS="$goos" GOARCH="$goarch" go build -o "${out_dir}/dbx" ./cmd/dbx
  tar -czf "${DIST_DIR}/${name}.tar.gz" -C "$out_dir" dbx
  rm -rf "$out_dir"
}

build_target linux amd64
build_target linux arm64
build_target darwin amd64
build_target darwin arm64

(
  cd "$DIST_DIR"
  shasum -a 256 ./*.tar.gz > checksums.txt
)

echo "Release artifacts written to ${DIST_DIR}"
