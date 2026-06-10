#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${CPA_PLUS_APP_DIR:-/root/apps/cliproxyapi-plus}"
SERVICE_NAME="${CPA_PLUS_SERVICE_NAME:-cliproxyapi-plus}"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

systemd_quote() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//%/%%}"
  printf '"%s"' "$value"
}

if [[ "$(id -u)" != "0" ]]; then
  echo "install-systemd.sh must be run as root" >&2
  exit 1
fi

if [[ ! -x "${APP_DIR}/cli-proxy-api" ]]; then
  echo "missing binary: ${APP_DIR}/cli-proxy-api" >&2
  echo "run scripts/install-linux.sh first" >&2
  exit 1
fi

cat > "${SERVICE_FILE}" <<SERVICE
[Unit]
Description=CPA-PLUS Linux Binary Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$(systemd_quote "$APP_DIR")
ExecStart=$(systemd_quote "$APP_DIR/cli-proxy-api") -config $(systemd_quote "$APP_DIR/config.yaml")
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl restart "${SERVICE_NAME}.service"
systemctl --no-pager --full status "${SERVICE_NAME}.service" || true

echo "Installed and started ${SERVICE_NAME}.service"
