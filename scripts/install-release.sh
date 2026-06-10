#!/usr/bin/env bash
set -euo pipefail

REPO="${CPA_PLUS_REPO:-julioaaericksonaa/CPA-PLUS}"
REF="${CPA_PLUS_INSTALL_REF:-latest}"
RELEASE="${CPA_PLUS_RELEASE:-latest}"
APP_DIR="${CPA_PLUS_APP_DIR:-/root/apps/cliproxyapi-plus}"
PORT="${CPA_PLUS_PORT:-8317}"
BINARY_NAME="${CPA_PLUS_BINARY_NAME:-CLIProxyAPI-linux-amd64}"
START_SERVICE="${CPA_PLUS_START:-1}"
FORCE_CONFIG="${CPA_PLUS_FORCE_CONFIG:-0}"
PUBLIC_ACCESS="${CPA_PLUS_PUBLIC:-0}"
INSTALL_SYSTEMD="${CPA_PLUS_SYSTEMD:-auto}"
INSTALL_UPDATE_CMD="${CPA_PLUS_INSTALL_UPDATE_CMD:-1}"
SERVICE_NAME="${CPA_PLUS_SERVICE_NAME:-cliproxyapi-plus}"

usage() {
  cat <<USAGE
Usage: install-release.sh [options]

Install or update CPA-PLUS from GitHub Release.

Options:
  --app-dir DIR       Install directory. Default: ${APP_DIR}
  --port PORT         Service port for new config. Default: ${PORT}
  --release TAG       Release tag. Default: ${RELEASE}
  --repo OWNER/REPO   GitHub repo. Default: ${REPO}
  --no-start          Download/install only, do not start service
  --public            Bind 0.0.0.0 and allow remote management
  --force-config      Regenerate config.yaml and backup old config
  -h, --help          Show this help

Environment:
  CPA_PLUS_MANAGEMENT_KEY   Management panel key; generated when empty
  CPA_PLUS_API_KEY          Client API key; generated when empty
  CPA_PLUS_PUBLIC           1 to bind all interfaces. Default: 0
  CPA_PLUS_SYSTEMD          auto, 1, or 0. Default: auto
  CPA_PLUS_INSTALL_UPDATE_CMD 1 or 0. Default: 1
USAGE
}

require_value() {
  local opt="$1"
  local value="${2:-}"
  if [[ -z "$value" || "$value" == --* ]]; then
    echo "${opt} requires a value" >&2
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --app-dir) require_value "$1" "${2:-}"; APP_DIR="$2"; shift 2 ;;
    --port) require_value "$1" "${2:-}"; PORT="$2"; shift 2 ;;
    --release) require_value "$1" "${2:-}"; RELEASE="$2"; shift 2 ;;
    --repo) require_value "$1" "${2:-}"; REPO="$2"; shift 2 ;;
    --no-start) START_SERVICE=0; shift ;;
    --public|--allow-remote) PUBLIC_ACCESS=1; shift ;;
    --force-config) FORCE_CONFIG=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
  esac
done

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mWARN:\033[0m %s\n' "$*" >&2; }

rand_hex() {
  local bytes="${1:-24}"
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$bytes"
  else
    head -c "$bytes" /dev/urandom | od -An -tx1 | tr -d ' \n'
  fi
}

shell_quote() {
  printf "%q" "$1"
}

systemd_quote() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//%/%%}"
  printf '"%s"' "$value"
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing command: $1" >&2; exit 1; }
}

need_cmd awk
need_cmd sha256sum
need_cmd chmod
need_cmd mkdir
need_cmd mv
need_cmd cp
need_cmd install
need_cmd date

if [[ ! "$PORT" =~ ^[0-9]+$ ]]; then
  echo "Invalid port: $PORT" >&2
  exit 1
fi
if (( PORT < 1 || PORT > 65535 )); then
  echo "Invalid port range: $PORT" >&2
  exit 1
fi
if [[ "$PUBLIC_ACCESS" != "0" && "$PUBLIC_ACCESS" != "1" ]]; then
  echo "Invalid CPA_PLUS_PUBLIC: $PUBLIC_ACCESS (expected 0 or 1)" >&2
  exit 1
fi

panel_url() {
  if [[ "$PUBLIC_ACCESS" == "1" ]]; then
    printf 'http://host:%s/management.html' "$PORT"
  else
    printf 'http://127.0.0.1:%s/management.html' "$PORT"
  fi
}

download_release() {
  local tmp="$1"
  mkdir -p "$tmp"

  if command -v gh >/dev/null 2>&1 && gh auth status --hostname github.com >/dev/null 2>&1; then
    log "Downloading ${REPO}@${RELEASE} by GitHub CLI"
    gh release download "$RELEASE" \
      --repo "$REPO" \
      --pattern "$BINARY_NAME" \
      --pattern "$BINARY_NAME.sha256" \
      --dir "$tmp" \
      --clobber
    return
  fi

  need_cmd curl
  log "Downloading ${REPO}@${RELEASE} by curl"
  local base="https://github.com/${REPO}/releases/download/${RELEASE}"
  curl -fL -o "$tmp/$BINARY_NAME" "$base/$BINARY_NAME"
  curl -fL -o "$tmp/$BINARY_NAME.sha256" "$base/$BINARY_NAME.sha256"
}

write_config_template() {
  local path="$1"
  local management_key="$2"
  local api_key="$3"
  local bind_host="127.0.0.1"
  local allow_remote="false"
  if [[ "$PUBLIC_ACCESS" == "1" ]]; then
    bind_host=""
    allow_remote="true"
  fi

  cat > "$path" <<YAML
# CPA-PLUS 默认配置：单端口 API + Plus 面板
# 面板：$(panel_url)
# 首次安装会自动生成 management key 和 api key，并保存到 ${APP_DIR}/secrets.env
# 默认仅监听 127.0.0.1；如需公网直连安装时加 --public。

host: "${bind_host}"
port: ${PORT}

tls:
  enable: false
  cert: ""
  key: ""

remote-management:
  # true：允许非 localhost 访问管理接口；公网直连时请务必保管好 secret-key。
  allow-remote: ${allow_remote}
  secret-key: "${management_key}"
  disable-control-panel: false
  # CPA-PLUS 已内置 Plus 面板，关闭上游旧面板自动更新。
  disable-auto-update-panel: true
  panel-github-repository: "https://github.com/router-for-me/Cli-Proxy-API-Management-Center"

auth-dir: "~/.cli-proxy-api"

# 客户端调用本代理时使用的 API Key，可自行增加多条。
api-keys:
  - "${api_key}"

# Plus Manager：请求统计、巡检、历史记录。
plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
  collector-mode: auto
  poll-interval-ms: 1000

# 常用运行参数
debug: false
commercial-mode: false
logging-to-file: false
logs-max-total-size-mb: 1024
error-logs-max-files: 10
usage-statistics-enabled: false
redis-usage-queue-retention-seconds: 60
proxy-url: ""
request-retry: 3
max-retry-credentials: 0
max-retry-interval: 30
disable-cooling: false
force-model-prefix: false
passthrough-headers: false
ws-auth: true
enable-gemini-cli-endpoint: false
nonstream-keepalive-interval: 0

routing:
  strategy: "round-robin"
  session-affinity: false
  session-affinity-ttl: "1h"

quota-exceeded:
  switch-project: true
  switch-preview-model: true
  antigravity-credits: true

pprof:
  enable: false
  addr: "127.0.0.1:8316"

plugins:
  enabled: false
  dir: "plugins"
  configs: {}

codex:
  identity-confuse: false

# ===== 模型 Provider 示例：按需取消注释并填入自己的 Key =====

# OpenAI-compatible 示例：
# openai-compatibility:
#   - name: "openai"
#     disabled: false
#     prefix: "openai"
#     base-url: "https://api.openai.com/v1"
#     api-key-entries:
#       - api-key: "sk-your-openai-key"
#     models:
#       - name: "gpt-4o-mini"
#         alias: "gpt-mini"

# Gemini 示例：
# gemini-api-key:
#   - api-key: "AIza-your-gemini-key"
#     prefix: "gemini"
#     base-url: "https://generativelanguage.googleapis.com"
#     models:
#       - name: "gemini-2.5-flash"
#         alias: "gemini-flash"

# Claude 示例：
# claude-api-key:
#   - api-key: "sk-ant-your-claude-key"
#     prefix: "claude"
#     models:
#       - name: "claude-sonnet-4-5"
#         alias: "claude-sonnet"

# Codex 示例：
# codex-api-key:
#   - api-key: "sk-your-codex-key"
#     prefix: "codex"
#     models:
#       - name: "gpt-5-codex"
#         alias: "codex-latest"
YAML
}

write_config_example() {
  local path="$1"
  write_config_template "$path" "CHANGE_ME_MANAGEMENT_KEY" "CHANGE_ME_CLIENT_API_KEY"
}

write_runtime_scripts() {
  mkdir -p "$APP_DIR/logs"

  cat > "$APP_DIR/start-detached.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
APP=$(shell_quote "$APP_DIR")
SERVICE=$(shell_quote "$SERVICE_NAME")
cd "\$APP"
mkdir -p "\$APP/logs" "\$APP/data"
if [[ "\$(cat "\$APP/runtime-mode" 2>/dev/null || true)" == "systemd" ]]; then
  systemctl start "\$SERVICE"
  echo "CPA-PLUS started by systemd: $(panel_url)"
  exit 0
fi
if [[ -f "\$APP/cliproxyapi-plus.pid" ]]; then
  old_pid=\$(cat "\$APP/cliproxyapi-plus.pid" 2>/dev/null || true)
  if [[ -n "\$old_pid" ]] && kill -0 "\$old_pid" 2>/dev/null; then
    echo "CPA-PLUS already running: \$old_pid"
    exit 0
  fi
fi
setsid nohup "\$APP/cli-proxy-api" -config "\$APP/config.yaml" > "\$APP/logs/cliproxyapi-plus.nohup.log" 2>&1 < /dev/null &
echo \$! > "\$APP/cliproxyapi-plus.pid"
sleep 1
LISTEN_PID=\$(ss -ltnp 2>/dev/null | awk '/:${PORT} / {print}' | sed -n 's/.*pid=\\([0-9]*\\).*/\\1/p' | head -n1 || true)
if [[ -n "\$LISTEN_PID" ]]; then echo "\$LISTEN_PID" > "\$APP/cliproxyapi-plus.pid"; fi
echo "CPA-PLUS started: $(panel_url)"
EOF

  cat > "$APP_DIR/stop.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
APP=$(shell_quote "$APP_DIR")
SERVICE=$(shell_quote "$SERVICE_NAME")
if [[ "\$(cat "\$APP/runtime-mode" 2>/dev/null || true)" == "systemd" ]]; then
  systemctl stop "\$SERVICE"
  echo "CPA-PLUS stopped by systemd"
  exit 0
fi
PID_FILE="\$APP/cliproxyapi-plus.pid"
if [[ -f "\$PID_FILE" ]]; then
  pid=\$(cat "\$PID_FILE" 2>/dev/null || true)
  if [[ -n "\$pid" ]] && kill -0 "\$pid" 2>/dev/null; then
    kill "\$pid" 2>/dev/null || true
    rm -f "\$PID_FILE"
    echo "CPA-PLUS stopped: \$pid"
    exit 0
  fi
fi
pids=\$(ss -ltnp 2>/dev/null | awk '/:${PORT} / {print}' | sed -n 's/.*pid=\\([0-9]*\\).*/\\1/p' | sort -u || true)
for pid in \$pids; do kill "\$pid" 2>/dev/null || true; done
rm -f "\$PID_FILE"
echo "CPA-PLUS stopped"
EOF

  cat > "$APP_DIR/restart.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
APP=$(shell_quote "$APP_DIR")
SERVICE=$(shell_quote "$SERVICE_NAME")
if [[ "\$(cat "\$APP/runtime-mode" 2>/dev/null || true)" == "systemd" ]]; then
  systemctl restart "\$SERVICE"
  echo "CPA-PLUS restarted by systemd: $(panel_url)"
  exit 0
fi
"\$APP/stop.sh" || true
sleep 1
"\$APP/start-detached.sh"
EOF

  cat > "$APP_DIR/status.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
APP=$(shell_quote "$APP_DIR")
SERVICE=$(shell_quote "$SERVICE_NAME")
if [[ "\$(cat "\$APP/runtime-mode" 2>/dev/null || true)" == "systemd" ]]; then
  systemctl status "\$SERVICE" --no-pager
  exit \$?
fi
if [[ -f "\$APP/cliproxyapi-plus.pid" ]]; then
  pid=\$(cat "\$APP/cliproxyapi-plus.pid" 2>/dev/null || true)
  if [[ -n "\$pid" ]] && kill -0 "\$pid" 2>/dev/null; then
    echo "running pid=\$pid"
    exit 0
  fi
fi
ss -ltnp 2>/dev/null | grep ':${PORT} ' || { echo "not running"; exit 1; }
EOF

  chmod 755 "$APP_DIR/start-detached.sh" "$APP_DIR/stop.sh" "$APP_DIR/restart.sh" "$APP_DIR/status.sh"
}

systemd_usable() {
  command -v systemctl >/dev/null 2>&1 || return 1
  [[ -d /run/systemd/system ]] || return 1
  systemctl list-units >/dev/null 2>&1 || return 1
}

install_systemd_service() {
  [[ $(id -u) -eq 0 ]] || return 1
  mkdir -p /etc/systemd/system
  cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=CPA-PLUS single-port service
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
EOF
  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME" >/dev/null
}

write_update_command() {
  if [[ -w /usr/local/bin ]]; then
    cat > /usr/local/bin/update-cpa <<EOF
#!/usr/bin/env bash
set -euo pipefail
export CPA_PLUS_REPO="\${CPA_PLUS_REPO:-${REPO}}"
export CPA_PLUS_INSTALL_REF="\${CPA_PLUS_INSTALL_REF:-${REF}}"
export CPA_PLUS_APP_DIR="\${CPA_PLUS_APP_DIR:-${APP_DIR}}"
export CPA_PLUS_PORT="\${CPA_PLUS_PORT:-${PORT}}"
export CPA_PLUS_RELEASE="\${CPA_PLUS_RELEASE:-latest}"
if command -v gh >/dev/null 2>&1 && gh auth status --hostname github.com >/dev/null 2>&1; then
  gh api "repos/\${CPA_PLUS_REPO}/contents/scripts/install-release.sh?ref=\${CPA_PLUS_INSTALL_REF}" --jq .content | base64 -d | bash -s -- "\$@"
else
  if ! command -v curl >/dev/null 2>&1; then
    echo "Missing curl. Install curl or login with GitHub CLI: gh auth login" >&2
    exit 1
  fi
  curl -fsSL "https://raw.githubusercontent.com/\${CPA_PLUS_REPO}/\${CPA_PLUS_INSTALL_REF}/scripts/install-release.sh" | bash -s -- "\$@"
fi
EOF
    chmod 755 /usr/local/bin/update-cpa
  else
    warn "/usr/local/bin is not writable; skipped installing update-cpa"
  fi
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

mkdir -p "$APP_DIR/data" "$APP_DIR/logs"
log "Installing CPA-PLUS to ${APP_DIR}"

download_release "$TMP_DIR"

if [[ ! -f "$TMP_DIR/$BINARY_NAME" || ! -f "$TMP_DIR/$BINARY_NAME.sha256" ]]; then
  echo "Release asset missing: $BINARY_NAME or $BINARY_NAME.sha256" >&2
  exit 1
fi

expected="$(awk '{print $1}' "$TMP_DIR/$BINARY_NAME.sha256")"
actual="$(sha256sum "$TMP_DIR/$BINARY_NAME" | awk '{print $1}')"
if [[ "$expected" != "$actual" ]]; then
  echo "sha256 mismatch" >&2
  echo "expected=$expected" >&2
  echo "actual=$actual" >&2
  exit 1
fi
log "Binary checksum OK"

if [[ -f "$APP_DIR/cli-proxy-api" ]]; then
  cp -p "$APP_DIR/cli-proxy-api" "$APP_DIR/cli-proxy-api.bak.$(date +%Y%m%d%H%M%S)"
fi
install -m 755 "$TMP_DIR/$BINARY_NAME" "$APP_DIR/cli-proxy-api"
printf '%s\n' "$PORT" > "$APP_DIR/PORT"
cp -f "$TMP_DIR/$BINARY_NAME.sha256" "$APP_DIR/cli-proxy-api.sha256"

write_config_example "$APP_DIR/config.example.yaml"
CONFIG_CREATED=0
if [[ "$FORCE_CONFIG" == "1" && -f "$APP_DIR/config.yaml" ]]; then
  backup_ts="$(date +%Y%m%d%H%M%S)"
  cp -p "$APP_DIR/config.yaml" "$APP_DIR/config.yaml.bak.${backup_ts}"
  if [[ -f "$APP_DIR/secrets.env" ]]; then
    cp -p "$APP_DIR/secrets.env" "$APP_DIR/secrets.env.bak.${backup_ts}"
  fi
  rm -f "$APP_DIR/config.yaml"
fi

if [[ ! -f "$APP_DIR/config.yaml" ]]; then
  MANAGEMENT_KEY="${CPA_PLUS_MANAGEMENT_KEY:-$(rand_hex 24)}"
  API_KEY="${CPA_PLUS_API_KEY:-cpa-$(rand_hex 18)}"
  write_config_template "$APP_DIR/config.yaml" "$MANAGEMENT_KEY" "$API_KEY"
  chmod 600 "$APP_DIR/config.yaml"
  cat > "$APP_DIR/secrets.env" <<EOF
CPA_PLUS_MANAGEMENT_KEY=$(shell_quote "$MANAGEMENT_KEY")
CPA_PLUS_API_KEY=$(shell_quote "$API_KEY")
CPA_PLUS_PANEL_URL=$(shell_quote "$(panel_url)")
EOF
  chmod 600 "$APP_DIR/secrets.env"
  CONFIG_CREATED=1
else
  warn "Existing config.yaml kept. Use --force-config to regenerate it."
fi

write_runtime_scripts
printf 'background\n' > "$APP_DIR/runtime-mode"
if [[ "$INSTALL_UPDATE_CMD" == "1" ]]; then
  write_update_command
fi

if [[ "$START_SERVICE" == "1" ]]; then
  if [[ "$INSTALL_SYSTEMD" == "1" || "$INSTALL_SYSTEMD" == "auto" ]]; then
    if systemd_usable && install_systemd_service; then
      printf 'systemd\n' > "$APP_DIR/runtime-mode"
      "$APP_DIR/stop.sh" >/dev/null 2>&1 || true
      log "Starting by systemd"
      systemctl restart "$SERVICE_NAME"
    elif [[ "$INSTALL_SYSTEMD" == "1" ]]; then
      echo "systemd is not usable on this host" >&2
      exit 1
    else
      log "Starting in background"
      "$APP_DIR/restart.sh"
    fi
  else
    log "Starting in background"
    "$APP_DIR/restart.sh"
  fi
fi

echo
echo "CPA-PLUS ready"
echo "Panel:  $(panel_url)"
echo "Config: ${APP_DIR}/config.yaml"
echo "Update: update-cpa"
echo "Logs:   ${APP_DIR}/logs/cliproxyapi-plus.nohup.log"
if [[ "$CONFIG_CREATED" == "1" ]]; then
  echo "Keys:   ${APP_DIR}/secrets.env"
  echo "Management key saved in ${APP_DIR}/secrets.env"
fi
