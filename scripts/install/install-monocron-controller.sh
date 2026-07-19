#!/usr/bin/env bash
set -e

REPO="Omotolani98/monocron"
APP="monocron-controller"
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

# Install systemd unit and environment file when running as root.
if [ "$(id -u)" -eq 0 ]; then
  if ! id "$USER" &>/dev/null; then
    groupadd --system "$GROUP" || true
    useradd --system --no-create-home --shell /usr/sbin/nologin --gid "$GROUP" "$USER"
  fi

  mkdir -p /etc/monocron /var/lib/monocron
  cp "${TMP}/deploy/systemd/monocron-controller.service" /etc/systemd/system/

  if [ ! -f /etc/monocron/controller.env ]; then
    cp "${TMP}/deploy/systemd/controller.env" /etc/monocron/controller.env
    chmod 600 /etc/monocron/controller.env
  fi

  chown -R "${USER}:${GROUP}" /etc/monocron /var/lib/monocron
  systemctl daemon-reload
  systemctl enable monocron-controller
fi

echo "${APP} installed to ${INSTALL_DIR}/${APP}"
echo ""
echo "1. Edit /etc/monocron/controller.env and set MONOCRON_DATABASE_URL."
echo "2. Start the controller:"
echo "     sudo systemctl start monocron-controller"
echo "3. Verify:"
echo "     systemctl status monocron-controller --no-pager -l"
echo "     curl http://127.0.0.1:8080/api/v1/health"
