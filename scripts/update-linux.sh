#!/usr/bin/env bash
set -euo pipefail
APP_DIR="${CPA_PLUS_APP_DIR:-/root/apps/cliproxyapi-plus}"
"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/install-linux.sh" "$@"
"${APP_DIR}/restart.sh"
