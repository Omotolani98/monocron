#!/usr/bin/env bash
set -e

REPO="Omotolani98/monocron"
APP="monocron-runner"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
USER="${MONOCRON_USER:-monocron}"
GROUP="${MONOCRON_GROUP:-monocron}"

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

URL="https://github.com/${REPO}/releases/download/${TAG}/${APP}_${TAG}_${OS}_${ARCH}.tar.gz"
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

# Create user and state directory if running as root
if [ "$(id -u)" -eq 0 ]; then
  if ! id "$USER" &>/dev/null; then
    groupadd --system "$GROUP" || true
    useradd --system --no-create-home --shell /usr/sbin/nologin --gid "$GROUP" "$USER"
  fi
  mkdir -p /var/lib/monocron
  chown -R "${USER}:${GROUP}" /var/lib/monocron
fi

echo "${APP} installed to ${INSTALL_DIR}/${APP}"
echo "Join the runner before starting:"
echo "  monocronctl runner join <token>"
echo "Then start the service:"
echo "  sudo systemctl enable --now monocron-runner"
