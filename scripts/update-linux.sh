#!/usr/bin/env bash
set -euo pipefail
APP_DIR="${CPA_PLUS_APP_DIR:-/root/apps/cliproxyapi-plus}"
SERVICE_NAME="${CPA_PLUS_SERVICE_NAME:-cliproxyapi-plus}"
"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/install-linux.sh" "$@"
if command -v systemctl >/dev/null 2>&1 && \
  systemctl list-unit-files --no-legend "${SERVICE_NAME}.service" 2>/dev/null | grep -q "^${SERVICE_NAME}\.service"; then
  systemctl restart "${SERVICE_NAME}.service"
else
  "${APP_DIR}/restart.sh"
fi
