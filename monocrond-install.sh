#!/usr/bin/env bash
set -e

REPO="Omotolani98/monocron"
APP="monocrond"
INSTALL_DIR="/usr/local/bin"
LOG_DIR="/var/log/monocron"
SERVICE_FILE="/etc/systemd/system/${APP}.service"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "🔍 Fetching latest release info..."
TAG=$(curl -s https://api.github.com/repos/${REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$TAG" ]; then
  echo "❌ Failed to fetch latest release."
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${APP}_${TAG#v}_${OS}_${ARCH}.tar.gz"

echo "⬇️ Downloading ${APP} ${TAG} for ${OS}/${ARCH}..."
curl -L "$URL" -o /tmp/${APP}.tar.gz

echo "📦 Extracting..."
tar -xzf /tmp/${APP}.tar.gz -C /tmp
chmod +x /tmp/${APP}
sudo mv /tmp/${APP} ${INSTALL_DIR}/${APP}

# --- Setup logs directory ---
echo "🪵 Setting up logs at ${LOG_DIR}..."
sudo mkdir -p "${LOG_DIR}"
sudo touch "${LOG_DIR}/${APP}.log}"
sudo chown -R root:root "${LOG_DIR}"

# --- Create systemd service ---
echo "⚙️ Creating systemd service..."
sudo tee "${SERVICE_FILE}" > /dev/null <<EOF
[Unit]
Description=Monocrond Daemon
After=network.target

[Service]
ExecStart=${INSTALL_DIR}/${APP}
Restart=always
RestartSec=5
StandardOutput=append:${LOG_DIR}/${APP}.log
StandardError=append:${LOG_DIR}/${APP}.log
User=root
WorkingDirectory=/root

[Install]
WantedBy=multi-user.target
EOF

echo "🚀 Starting ${APP} service..."
sudo systemctl daemon-reload
sudo systemctl enable ${APP}
sudo systemctl restart ${APP}

sleep 1
if systemctl is-active --quiet ${APP}; then
  echo "✅ ${APP} is running!"
  echo "Logs: ${LOG_DIR}/${APP}.log"
else
  echo "⚠️ ${APP} failed to start. Check logs:"
  echo "sudo journalctl -u ${APP} -e"
fi