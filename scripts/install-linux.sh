#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="${CPA_PLUS_APP_DIR:-/root/apps/cliproxyapi-plus}"
PORT="${CPA_PLUS_PORT:-$(cat "${ROOT_DIR}/PORT" 2>/dev/null || echo 8318)}"
DIST_DIR="${CPA_PLUS_DIST_DIR:-${ROOT_DIR}/dist}"
BINARY_NAME="${CPA_PLUS_BINARY_NAME:-CLIProxyAPI-linux-amd64}"
SOURCE_DIR="${CPA_PLUS_OUTPUT_SOURCE:-${ROOT_DIR}/.build/out/CLIProxyAPI}"

"${ROOT_DIR}/scripts/build-linux-binary.sh" "$@"

mkdir -p "${APP_DIR}/data" "${APP_DIR}/logs" "${APP_DIR}/static"
printf "%s\n" "${PORT}" > "${APP_DIR}/PORT"
cp "${DIST_DIR}/${BINARY_NAME}" "${APP_DIR}/cli-proxy-api"
chmod 755 "${APP_DIR}/cli-proxy-api"
cp "${SOURCE_DIR}/config.example.yaml" "${APP_DIR}/config.example.yaml"

if [[ ! -f "${APP_DIR}/config.yaml" ]]; then
  cp "${APP_DIR}/config.example.yaml" "${APP_DIR}/config.yaml"
fi

python3 - "$APP_DIR/config.yaml" "$PORT" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1])
port=sys.argv[2]
lines=p.read_text(errors='replace').splitlines()
out=[]
in_rm=False
plus_seen=False
for line in lines:
    stripped=line.strip()
    if line.startswith('port:'):
        out.append(f'port: {port}')
        continue
    if stripped.startswith('remote-management:'):
        in_rm=True
        out.append(line)
        continue
    if in_rm and line and not line.startswith((' ', '\t')) and not stripped.startswith('#'):
        out.append('  disable-auto-update-panel: true')
        in_rm=False
    if in_rm:
        if stripped.startswith('# disable-auto-update-panel:') or stripped.startswith('disable-auto-update-panel:'):
            out.append('  disable-auto-update-panel: true')
            continue
    if stripped.startswith('plus-manager:'):
        plus_seen=True
    out.append(line)
if in_rm:
    out.append('  disable-auto-update-panel: true')
text='\n'.join(out).rstrip()
if not plus_seen:
    text += '''

plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
  collector-mode: auto
  poll-interval-ms: 1000
'''
p.write_text(text+'\n')
PY

cat > "${APP_DIR}/start-detached.sh" <<EOF2
#!/usr/bin/env bash
set -euo pipefail
APP=${APP_DIR}
cd "\$APP"
setsid nohup "\$APP/cli-proxy-api" -config "\$APP/config.yaml" > "\$APP/logs/cliproxyapi-plus.nohup.log" 2>&1 < /dev/null &
echo \$! > "\$APP/cliproxyapi-plus.pid"
sleep 1
LISTEN_PID=\$(ss -ltnp 2>/dev/null | awk '/:${PORT} / {print}' | sed -n 's/.*pid=\\([0-9]*\\).*/\\1/p' | head -n1 || true)
if [[ -n "\$LISTEN_PID" ]]; then echo "\$LISTEN_PID" > "\$APP/cliproxyapi-plus.pid"; fi
EOF2

cat > "${APP_DIR}/stop.sh" <<EOF2
#!/usr/bin/env bash
set -euo pipefail
PID_FILE=${APP_DIR}/cliproxyapi-plus.pid
if [[ -f "\$PID_FILE" ]]; then
  pid=\$(cat "\$PID_FILE")
  if [[ -n "\$pid" ]] && kill -0 "\$pid" 2>/dev/null; then
    kill "\$pid"
    exit 0
  fi
fi
pids=\$(ss -ltnp 2>/dev/null | awk '/:${PORT} / {print}' | sed -n 's/.*pid=\\([0-9]*\\).*/\\1/p' | sort -u)
for pid in \$pids; do kill "\$pid" 2>/dev/null || true; done
EOF2

cat > "${APP_DIR}/restart.sh" <<EOF2
#!/usr/bin/env bash
set -euo pipefail
APP=${APP_DIR}
"\$APP/stop.sh" || true
sleep 1
"\$APP/start-detached.sh"
ss -ltnp 2>/dev/null | grep ':${PORT} ' || true
EOF2

chmod 755 "${APP_DIR}/start-detached.sh" "${APP_DIR}/stop.sh" "${APP_DIR}/restart.sh"
echo "Installed CPA-PLUS binary project to ${APP_DIR} on port ${PORT}"
