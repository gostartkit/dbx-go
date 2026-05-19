#!/bin/sh
set -eu

REPO="OWNER/dbx"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$arch" in
  x86_64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

case "$os" in
  linux|darwin) ;;
  *)
    echo "Unsupported operating system: $os" >&2
    exit 1
    ;;
esac

if [ -z "$INSTALL_DIR" ]; then
  if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
fi

artifact="dbx_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases"
if [ "$VERSION" = "latest" ]; then
  url="${base_url}/latest/download/${artifact}"
else
  url="${base_url}/download/${VERSION}/${artifact}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

echo "Downloading ${url}"
curl -fsSL "$url" -o "$tmp_dir/$artifact"
tar -xzf "$tmp_dir/$artifact" -C "$tmp_dir"

binary="$tmp_dir/dbx"
if [ ! -f "$binary" ]; then
  echo "Downloaded archive did not contain dbx binary." >&2
  exit 1
fi

target="${INSTALL_DIR}/dbx"
echo "Installing to ${target}"
install "$binary" "$target"

echo
echo "Installed dbx to ${target}"
if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
  echo "Ensure ${INSTALL_DIR} is on your PATH."
fi
