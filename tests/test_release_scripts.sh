#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIRS=()
cleanup() {
  local dir
  for dir in "${TMP_DIRS[@]:-}"; do
    rm -rf "$dir"
  done
}
trap cleanup EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

make_fake_gh() {
  local bin_dir="$1"
  local release_dir="$2"
  mkdir -p "$bin_dir" "$release_dir"
  printf '#!/usr/bin/env bash\necho fake-cpa-plus\n' > "$release_dir/CLIProxyAPI-linux-amd64"
  chmod 755 "$release_dir/CLIProxyAPI-linux-amd64"
  sha256sum "$release_dir/CLIProxyAPI-linux-amd64" > "$release_dir/CLIProxyAPI-linux-amd64.sha256"
  cat > "$bin_dir/gh" <<'GH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "auth" && "${2:-}" == "status" ]]; then
  exit 0
fi
if [[ "${1:-}" == "release" && "${2:-}" == "download" ]]; then
  dir=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dir) dir="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  mkdir -p "$dir"
  cp "$FAKE_RELEASE_DIR/CLIProxyAPI-linux-amd64" "$dir/CLIProxyAPI-linux-amd64"
  cp "$FAKE_RELEASE_DIR/CLIProxyAPI-linux-amd64.sha256" "$dir/CLIProxyAPI-linux-amd64.sha256"
  exit 0
fi
echo "fake gh unsupported: $*" >&2
exit 1
GH
  chmod 755 "$bin_dir/gh"
}

test_workflow_ignores_source_commit_for_change_detection() {
  if grep -n "git diff --quiet -- .*CPA_PLUS_SOURCE_COMMIT" "$ROOT_DIR/.github/workflows/auto-sync-release.yml" >/dev/null; then
    fail "workflow change detection must not include CPA_PLUS_SOURCE_COMMIT"
  fi
  if grep -n "git add .*CPA_PLUS_SOURCE_COMMIT" "$ROOT_DIR/.github/workflows/auto-sync-release.yml" >/dev/null; then
    fail "workflow commit step must not add CPA_PLUS_SOURCE_COMMIT"
  fi
}

test_generated_scripts_quote_single_quote_app_dir() {
  local tmp app
  tmp="$(mktemp -d)"
  TMP_DIRS+=("$tmp")
  make_fake_gh "$tmp/bin" "$tmp/release"
  app="$tmp/app with ' quote"
  PATH="$tmp/bin:$PATH" \
  FAKE_RELEASE_DIR="$tmp/release" \
  CPA_PLUS_APP_DIR="$app" \
  CPA_PLUS_PORT=9909 \
  CPA_PLUS_INSTALL_UPDATE_CMD=0 \
    "$ROOT_DIR/scripts/install-release.sh" --no-start >/dev/null
  bash -n "$app/start-detached.sh"
  bash -n "$app/stop.sh"
  bash -n "$app/restart.sh"
  bash -n "$app/status.sh"
  bash -n "$app/secrets.env"
}

test_systemd_unit_quotes_paths_with_spaces() {
  grep -F 'WorkingDirectory=' "$ROOT_DIR/scripts/install-release.sh" | grep -F '"' >/dev/null \
    || fail "install-release.sh systemd WorkingDirectory must quote APP_DIR"
  grep -F 'ExecStart=' "$ROOT_DIR/scripts/install-release.sh" | grep -F '"' >/dev/null \
    || fail "install-release.sh systemd ExecStart must quote APP_DIR"
  grep -F 'WorkingDirectory=' "$ROOT_DIR/scripts/install-systemd.sh" | grep -F '"' >/dev/null \
    || fail "install-systemd.sh systemd WorkingDirectory must quote APP_DIR"
  grep -F 'ExecStart=' "$ROOT_DIR/scripts/install-systemd.sh" | grep -F '"' >/dev/null \
    || fail "install-systemd.sh systemd ExecStart must quote APP_DIR"
}

test_generated_runtime_scripts_support_systemd_mode() {
  local tmp app
  tmp="$(mktemp -d)"
  TMP_DIRS+=("$tmp")
  make_fake_gh "$tmp/bin" "$tmp/release"
  app="$tmp/app"
  PATH="$tmp/bin:$PATH" \
  FAKE_RELEASE_DIR="$tmp/release" \
  CPA_PLUS_APP_DIR="$app" \
  CPA_PLUS_PORT=9909 \
  CPA_PLUS_INSTALL_UPDATE_CMD=0 \
    "$ROOT_DIR/scripts/install-release.sh" --no-start >/dev/null
  [[ -f "$app/runtime-mode" ]] || fail "installer must write runtime-mode"
  grep -F 'runtime-mode' "$app/start-detached.sh" >/dev/null || fail "start script must inspect runtime-mode"
  grep -F 'systemctl start' "$app/start-detached.sh" >/dev/null || fail "start script must delegate systemd mode to systemctl"
  grep -F 'systemctl stop' "$app/stop.sh" >/dev/null || fail "stop script must delegate systemd mode to systemctl"
  grep -F 'systemctl restart' "$app/restart.sh" >/dev/null || fail "restart script must delegate systemd mode to systemctl"
  grep -F 'systemctl status' "$app/status.sh" >/dev/null || fail "status script must delegate systemd mode to systemctl"
}

test_default_config_is_local_only_and_public_flag_is_explicit() {
  local tmp app public_app out management_key
  tmp="$(mktemp -d)"
  TMP_DIRS+=("$tmp")
  make_fake_gh "$tmp/bin" "$tmp/release"
  app="$tmp/local-app"
  PATH="$tmp/bin:$PATH" \
  FAKE_RELEASE_DIR="$tmp/release" \
  CPA_PLUS_APP_DIR="$app" \
  CPA_PLUS_PORT=9909 \
  CPA_PLUS_INSTALL_UPDATE_CMD=0 \
    "$ROOT_DIR/scripts/install-release.sh" --no-start >"$tmp/local.out"
  grep -F 'host: "127.0.0.1"' "$app/config.yaml" >/dev/null || fail "default config must bind localhost"
  grep -F '  allow-remote: false' "$app/config.yaml" >/dev/null || fail "default config must disallow remote management"
  # shellcheck disable=SC1091
  source "$app/secrets.env"
  management_key="${CPA_PLUS_MANAGEMENT_KEY:-}"
  [[ -n "$management_key" ]] || fail "secrets.env must contain management key"
  if grep -F "$management_key" "$tmp/local.out" >/dev/null; then
    fail "installer output must not print raw management key"
  fi

  PATH="$tmp/bin:$PATH" \
  FAKE_RELEASE_DIR="$tmp/release" \
  CPA_PLUS_APP_DIR="$app" \
  CPA_PLUS_PORT=9909 \
  CPA_PLUS_INSTALL_UPDATE_CMD=0 \
    "$ROOT_DIR/scripts/install-release.sh" --no-start --force-config >/dev/null
  ls "$app"/config.yaml.bak.* >/dev/null 2>&1 || fail "--force-config must backup config.yaml"
  ls "$app"/secrets.env.bak.* >/dev/null 2>&1 || fail "--force-config must backup secrets.env"

  public_app="$tmp/public-app"
  PATH="$tmp/bin:$PATH" \
  FAKE_RELEASE_DIR="$tmp/release" \
  CPA_PLUS_APP_DIR="$public_app" \
  CPA_PLUS_PORT=9910 \
  CPA_PLUS_INSTALL_UPDATE_CMD=0 \
    "$ROOT_DIR/scripts/install-release.sh" --no-start --public >/dev/null
  grep -F 'host: ""' "$public_app/config.yaml" >/dev/null || fail "--public config must bind all interfaces"
  grep -F '  allow-remote: true' "$public_app/config.yaml" >/dev/null || fail "--public config must allow remote management"
}

test_readme_quotes_gh_api_urls() {
  if grep -nE 'gh api [^"[:space:]]*\?ref=main' "$ROOT_DIR/README.md" >/dev/null; then
    fail "README gh api URLs with ?ref=main must be quoted"
  fi
}

main() {
  test_workflow_ignores_source_commit_for_change_detection
  test_generated_scripts_quote_single_quote_app_dir
  test_systemd_unit_quotes_paths_with_spaces
  test_generated_runtime_scripts_support_systemd_mode
  test_default_config_is_local_only_and_public_flag_is_explicit
  test_readme_quotes_gh_api_urls
  echo "release script regression tests passed"
}

main "$@"
