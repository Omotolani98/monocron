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

# Install systemd unit and create configuration/state directories when running as root.
if [ "$(id -u)" -eq 0 ]; then
  if ! id "$USER" &>/dev/null; then
    groupadd --system "$GROUP" || true
    useradd --system --no-create-home --shell /usr/sbin/nologin --gid "$GROUP" "$USER"
  fi

  mkdir -p /var/lib/monocron /etc/monocron
  cp "${TMP}/deploy/systemd/monocron-runner.service" /etc/systemd/system/

  if [ ! -f /etc/monocron/runner.env ]; then
    cat >/etc/monocron/runner.env <<'EOF'
# Set the controller URL before starting the runner.
MONOCRON_CONTROLLER_URL=https://ctrl.example.com
EOF
    chmod 600 /etc/monocron/runner.env
  fi

  chown -R "${USER}:${GROUP}" /var/lib/monocron /etc/monocron
  systemctl daemon-reload
  systemctl enable monocron-runner
fi

echo "${APP} installed to ${INSTALL_DIR}/${APP}"
echo ""
echo "1. Edit /etc/monocron/runner.env and set MONOCRON_CONTROLLER_URL."
echo "2. Generate an enrollment token from an operator machine:"
echo "     monocronctl runner token zone=home os=linux"
echo "3. Enroll this runner as the monocron user:"
echo "     sudo -u monocron monocronctl runner --state /var/lib/monocron/runner.state join '<token>'"
echo "4. Start the service:"
echo "     sudo systemctl start monocron-runner"
echo "5. Verify:"
echo "     systemctl status monocron-runner --no-pager -l"
