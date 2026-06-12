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

test_workflow_detects_upstream_before_heavy_steps() {
  python3 - "$ROOT_DIR/.github/workflows/auto-sync-release.yml" <<'PY'
from pathlib import Path
import sys

text = Path(sys.argv[1]).read_text()

def require(condition, message):
    if not condition:
        print(f"FAIL: {message}", file=sys.stderr)
        sys.exit(1)

require("force_release" not in text, "workflow must not expose force_release when no upstream changed")
require("Detect upstream changes" in text, "workflow must detect upstream changes before sync/build")
require("router-for-me/CLIProxyAPI.git" in text, "workflow must check CLIProxyAPI upstream")
require("seakee/CPA-Manager-Plus.git" in text, "workflow must check CPA-Manager-Plus upstream")
require(text.index("Detect upstream changes") < text.index("Setup Go"), "upstream detection must run before Go setup")
for marker in [
    "Setup Go",
    "Setup Node.js",
    "Install system dependencies",
    "Sync upstream metadata and patch",
    "Build Linux amd64 binary",
    "Commit refreshed patch and metadata",
    "Prepare release metadata",
    "Refresh fixed latest release",
]:
    before = text[:text.index(marker)]
    after = text[text.index(marker): text.find("\n\n", text.index(marker)) if text.find("\n\n", text.index(marker)) != -1 else len(text)]
    require("steps.upstream.outputs.should_sync == 'true'" in after, f"{marker} must be gated by upstream change")
PY
}

test_workflow_keeps_only_latest_release() {
  local wf="$ROOT_DIR/.github/workflows/auto-sync-release.yml"
  if grep -n "Publish versioned release" "$wf" >/dev/null; then
    fail "workflow must not publish versioned releases"
  fi
  if grep -n 'gh release create "${TAG}"' "$wf" >/dev/null; then
    fail "workflow must not create versioned releases"
  fi
  grep -F 'gh release create latest' "$wf" >/dev/null || fail "workflow must create latest release when missing"
  grep -F 'gh release delete "$tag" --yes --cleanup-tag' "$wf" >/dev/null || fail "workflow must delete old releases"
  grep -F 'git push origin ":refs/tags/${tag}"' "$wf" >/dev/null || fail "workflow must delete old non-latest tags"
}

test_workflow_release_notes_include_refresh_time() {
  local wf="$ROOT_DIR/.github/workflows/auto-sync-release.yml"
  grep -F 'updated_at="$(TZ=Asia/Shanghai date' "$wf" >/dev/null || \
    fail "workflow release notes must compute Asia/Shanghai refresh time"
  grep -F 'Updated at: ${updated_at}' "$wf" >/dev/null || \
    fail "workflow release notes must include the refresh time"
}

test_core_patch_excludes_non_runtime_files() {
  local patch="$ROOT_DIR/patches/cliproxyapi/0001-cpa-plus-integration.patch"
  for path in \
    ".github/workflows/" \
    "README.md" \
    "README_CN.md" \
    "README_JA.md" \
    "docs/" \
    "scripts/" \
    "test/local_update_scripts_test.sh"; do
    if grep -F "diff --git a/${path}" "$patch" >/dev/null; then
      fail "core patch must exclude non-runtime file ${path}"
    fi
  done
}

test_plus_version_checks_use_integrated_backend() {
  local version_api="$ROOT_DIR/cpa-plus-web/overlay/src/services/api/version.ts"
  if grep -F "api.github.com/repos/seakee/CPA-Manager-Plus" "$version_api" >/dev/null; then
    fail "Plus version check must not call GitHub API directly from the browser"
  fi
  if grep -F "import axios" "$version_api" >/dev/null; then
    fail "Plus version check must use integrated apiClient instead of direct axios"
  fi
  grep -F "checkManagerLatest: () => apiClient.get" "$version_api" >/dev/null || \
    fail "Plus version check must call the integrated management backend"
  grep -F "/manager-latest-version" "$version_api" >/dev/null || \
    fail "Plus version check must use /manager-latest-version"
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
  test_workflow_detects_upstream_before_heavy_steps
  test_workflow_keeps_only_latest_release
  test_workflow_release_notes_include_refresh_time
  test_core_patch_excludes_non_runtime_files
  test_plus_version_checks_use_integrated_backend
  test_generated_scripts_quote_single_quote_app_dir
  test_systemd_unit_quotes_paths_with_spaces
  test_generated_runtime_scripts_support_systemd_mode
  test_default_config_is_local_only_and_public_flag_is_explicit
  test_readme_quotes_gh_api_urls
  echo "release script regression tests passed"
}

main "$@"
