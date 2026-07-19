#!/usr/bin/env bash
set -e

REPO="Omotolani98/monocron"
APP="monocron-controller"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
  echo "Failed to fetch latest release."
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${APP}_${TAG#v}_${OS}_${ARCH}.tar.gz"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${APP} ${TAG} for ${OS}/${ARCH}..."
curl -fsSL "$URL" -o "${TMP}/${APP}.tar.gz"

echo "Extracting..."
tar -xzf "${TMP}/${APP}.tar.gz" -C "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/${APP}" "${INSTALL_DIR}/${APP}"
else
  echo "Installing to ${INSTALL_DIR} requires sudo..."
  sudo mv "${TMP}/${APP}" "${INSTALL_DIR}/${APP}"
fi

chmod +x "${INSTALL_DIR}/${APP}"

echo "${APP} installed to ${INSTALL_DIR}/${APP}"
echo "Set MONOCRON_DATABASE_URL and run the controller:"
echo "  export MONOCRON_DATABASE_URL=postgres://user:pass@localhost/monocron?sslmode=disable"
echo "  ${INSTALL_DIR}/${APP}"
