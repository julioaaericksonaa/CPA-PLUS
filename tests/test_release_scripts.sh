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
require("push:" not in text, "workflow must not run on pushes; local publish handles release uploads")
require("workflow_dispatch:" not in text, "workflow must not expose manual runs; schedule must be the only trigger")
require('cron: "0 13 * * *"' in text, "workflow must run once daily at 21:00 Asia/Shanghai")
require('${GITHUB_EVENT_NAME}' not in text, "workflow release decision must not depend on push/manual events")
require('should_release="${should_sync}"' in text, "workflow must release only when upstream changed")
require("Detect upstream changes" in text, "workflow must detect upstream changes before sync/build")
require("router-for-me/CLIProxyAPI.git" in text, "workflow must check CLIProxyAPI upstream")
require("seakee/CPA-Manager-Plus.git" in text, "workflow must check CPA-Manager-Plus upstream")
require(text.index("Detect upstream changes") < text.index("Setup Go"), "upstream detection must run before Go setup")
for marker in [
    "Setup Go",
    "Setup Node.js",
    "Install system dependencies",
    "Build Linux amd64 binary",
    "Prepare release metadata",
    "Refresh fixed latest release",
]:
    before = text[:text.index(marker)]
    after = text[text.index(marker): text.find("\n\n", text.index(marker)) if text.find("\n\n", text.index(marker)) != -1 else len(text)]
    require("steps.upstream.outputs.should_release == 'true'" in after, f"{marker} must be gated by release decision")
for marker in [
    "Sync upstream metadata and patch",
    "Commit refreshed patch and metadata",
]:
    after = text[text.index(marker): text.find("\n\n", text.index(marker)) if text.find("\n\n", text.index(marker)) != -1 else len(text)]
    require("steps.upstream.outputs.should_sync == 'true'" in after, f"{marker} must only run when upstream changed")
require(text.index("Commit refreshed patch and metadata") < text.index("Build Linux amd64 binary"), "workflow must commit synced upstream metadata before building the released binary")
PY
}

test_local_publish_script_recreates_latest_release() {
  local script="$ROOT_DIR/scripts/publish-latest-release.sh"
  [[ -f "$script" ]] || fail "local publish script must exist"
  grep -F '/root/.config/cpa-plus/publish.env' "$script" >/dev/null || \
    fail "local publish script must load publish.env"
  grep -F 'credential.helper=' "$script" >/dev/null || \
    fail "local publish script must disable cached git credentials"
  grep -F 'curl -fsS -X DELETE' "$script" >/dev/null || \
    fail "local publish script must delete latest release before recreating it"
  grep -F 'curl -fsS -X POST' "$script" >/dev/null || \
    fail "local publish script must recreate latest release"
  grep -F '"tag_name": "latest"' "$script" >/dev/null || \
    fail "local publish script must create the fixed latest release"
  if grep -F -- '--clobber' "$script" >/dev/null; then
    fail "local publish script must recreate latest instead of clobbering assets"
  fi
}

test_workflow_keeps_only_latest_release() {
  local wf="$ROOT_DIR/.github/workflows/auto-sync-release.yml"
  if grep -n "Publish versioned release" "$wf" >/dev/null; then
    fail "workflow must not publish versioned releases"
  fi
  if grep -n 'gh release create "${TAG}"' "$wf" >/dev/null; then
    fail "workflow must not create versioned releases"
  fi
  grep -F 'gh release delete latest --yes || true' "$wf" >/dev/null || fail "workflow must delete latest release before recreating it"
  grep -F 'gh release create latest' "$wf" >/dev/null || fail "workflow must recreate latest release"
  if grep -F 'gh release edit latest' "$wf" >/dev/null; then
    fail "workflow must recreate latest release instead of editing it in place"
  fi
  if grep -F -- '--clobber' "$wf" >/dev/null; then
    fail "workflow must recreate latest assets instead of clobbering them in place"
  fi
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

test_update_command_fetches_installer_from_main_by_default() {
  grep -F 'REF="${CPA_PLUS_INSTALL_REF:-main}"' "$ROOT_DIR/scripts/install-release.sh" >/dev/null || \
    fail "install-release.sh must default installer ref to main"
  grep -F 'REF="${CPA_PLUS_INSTALL_REF:-main}"' "$ROOT_DIR/scripts/update-cpa" >/dev/null || \
    fail "update-cpa must default installer ref to main"
  if grep -F 'CPA_PLUS_INSTALL_REF:-latest' "$ROOT_DIR/scripts/install-release.sh" "$ROOT_DIR/scripts/update-cpa" >/dev/null; then
    fail "update command must not default installer ref to moving latest tag"
  fi
}

test_plus_build_version_uses_tag_and_commit() {
  grep -F 'nearest="$(git -C "${PLUS_DIR}" describe --tags --abbrev=0' "$ROOT_DIR/cpa-plus-core/prepare-source.sh" >/dev/null || \
    fail "Plus build version must include nearest numeric tag"
  grep -F "printf '%s+%s\\n' \"\${nearest}\" \"\${short}\"" "$ROOT_DIR/cpa-plus-core/prepare-source.sh" >/dev/null || \
    fail "Plus build version must include short commit after nearest tag"
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

test_dashboard_exposes_cpa_plus_update_button() {
  local version_api="$ROOT_DIR/cpa-plus-web/overlay/src/services/api/version.ts"
  local card="$ROOT_DIR/cpa-plus-web/overlay/src/features/dashboard/components/VersionCard.tsx"
  grep -F "triggerCPAPlusUpdate: () => apiClient.post" "$version_api" >/dev/null || \
    fail "version API must expose fixed CPA-PLUS update trigger"
  grep -F "'/cpa-plus/update'" "$version_api" >/dev/null || \
    fail "CPA-PLUS update trigger must call /cpa-plus/update"
  grep -F "hasUpstreamUpdate" "$card" >/dev/null || \
    fail "VersionCard must hide update button unless an upstream update exists"
  grep -F "handleCPAPlusUpdate" "$card" >/dev/null || \
    fail "VersionCard must provide a CPA-PLUS update click handler"
  grep -F "data.status === 'release_not_ready'" "$card" >/dev/null || \
    fail "VersionCard must treat release_not_ready as a warning, not a successful update"
  grep -F "data.status === 'update_running'" "$card" >/dev/null || \
    fail "VersionCard must treat update_running as a non-error in-progress state"
  grep -F "data.status === 'started'" "$card" >/dev/null || \
    fail "VersionCard must only show success after the backend reports started"
  grep -F "showNotification(message, 'warning')" "$card" >/dev/null || \
    fail "VersionCard must show non-started update statuses as warnings"
  grep -F "dashboard.cpa_plus_update_now" "$card" >/dev/null || \
    fail "VersionCard must render localized update button text"
}

test_ai_provider_overlay_keeps_toolbar_i18n() {
  for locale in en zh-CN zh-TW; do
    local file="$ROOT_DIR/cpa-plus-web/overlay/src/i18n/locales/${locale}.json"
    python3 - "$file" "$locale" <<'PY'
import json
import sys

path, locale = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as fh:
    data = json.load(fh)

providers = data.get("ai_providers", {})
required = [
    "filter_all",
    "health_check_button",
    "add_config_button",
    "table_aria_label",
    "table_col_type",
    "table_col_identity",
    "table_col_models",
    "table_col_recent",
    "table_col_actions",
]
missing = [key for key in required if not providers.get(key)]
if missing:
    raise SystemExit(f"{locale} ai_providers missing toolbar translations: {', '.join(missing)}")
PY
  done
}

test_overlay_keeps_plugin_and_account_action_i18n() {
  for locale in en zh-CN zh-TW; do
    local file="$ROOT_DIR/cpa-plus-web/overlay/src/i18n/locales/${locale}.json"
    python3 - "$file" "$locale" <<'PY'
import json
import sys

path, locale = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as fh:
    data = json.load(fh)

checks = {
    "nav": ["plugins", "account_actions", "account_actions_short"],
    "plugin_management": [
        "global_disabled_hint",
        "tab_installed",
        "tab_store",
        "global_status",
        "global_disabled",
        "installed_count",
        "effective_count",
        "plugins_dir",
        "search_placeholder",
        "install_plugin",
        "refresh",
        "no_plugins",
        "no_plugins_desc",
    ],
    "account_actions": [
        "eyebrow",
        "title",
        "description",
        "pending_count",
        "visible_count",
        "filter_pending",
        "filter_all",
        "load_failed_title",
        "empty_title",
        "col_account",
        "col_operations",
    ],
}

missing = []
for namespace, keys in checks.items():
    values = data.get(namespace, {})
    for key in keys:
        if not isinstance(values, dict) or not values.get(key):
            missing.append(f"{namespace}.{key}")

if missing:
    raise SystemExit(f"{locale} missing translations: {', '.join(missing)}")
PY
  done
}

test_overlay_keeps_plugin_store_i18n() {
  for locale in en zh-CN zh-TW; do
    local file="$ROOT_DIR/cpa-plus-web/overlay/src/i18n/locales/${locale}.json"
    python3 - "$file" "$locale" <<'PY'
import json
import sys

path, locale = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as fh:
    data = json.load(fh)

checks = {
    "nav": ["plugin_store", "plugin_store_short"],
    "plugin_store": [
        "title",
        "description",
        "refresh",
        "global_status",
        "global_disabled",
        "plugins_dir",
        "sources",
        "stat_available",
        "search_placeholder",
        "manage",
        "install",
        "badge_not_installed",
        "badge_untrusted",
        "meta_version",
        "meta_author",
        "meta_license",
        "meta_source",
    ],
    "plugin_resource": [
        "title",
        "load_failed",
        "not_found",
        "unsupported_backend",
    ],
}

missing = []
for namespace, keys in checks.items():
    values = data.get(namespace, {})
    for key in keys:
        if not isinstance(values, dict) or not values.get(key):
            missing.append(f"{namespace}.{key}")

if missing:
    raise SystemExit(f"{locale} missing plugin store translations: {', '.join(missing)}")
PY
  done
}

test_plus_web_integrates_account_action_candidate_paths() {
  local patcher="$ROOT_DIR/cpa-plus-web/patch-plus-web-integrated.py"
  grep -F "'/v0/management/account-action-candidates'" "$patcher" >/dev/null || \
    fail "web patcher must rewrite account action candidate list path"
  grep -F "'/v0/management/plus/account-action-candidates'" "$patcher" >/dev/null || \
    fail "account action candidate list path must use integrated plus prefix"
  grep -F "/v0/management/plus/account-action-candidates" "$patcher" >/dev/null || \
    fail "web patcher must require integrated account action candidate paths"
}

test_patch_persists_cpa_plus_update_backend() {
  local patch="$ROOT_DIR/patches/cliproxyapi/0001-cpa-plus-integration.patch"
  grep -F 'mgmt.POST("/cpa-plus/update", s.mgmt.PostCPAPlusUpdate)' "$patch" >/dev/null || \
    fail "integration patch must register /cpa-plus/update backend route"
  grep -F "func (h *Handler) PostCPAPlusUpdate" "$patch" >/dev/null || \
    fail "integration patch must persist PostCPAPlusUpdate handler"
  grep -F 'cpaPlusUpdateCommand       = "/usr/local/bin/update-cpa"' "$patch" >/dev/null || \
    fail "integration patch must run the installed update-cpa command"
  grep -F "func cpaPlusSystemdRunCommand" "$patch" >/dev/null || \
    fail "integration patch must persist systemd-run update detachment"
  grep -F "systemd-run" "$patch" >/dev/null || \
    fail "integration patch must use systemd-run when available"
  grep -F "func cpaPlusReleaseNotReadyHTTPStatus" "$patch" >/dev/null || \
    fail "integration patch must persist non-error release-not-ready status"
  grep -F "return http.StatusOK" "$patch" >/dev/null || \
    fail "release-not-ready must use HTTP 200 so the browser does not report a failed POST"
  grep -F "CPA_PLUS_SYSTEMD=0" "$patch" >/dev/null || \
    fail "self-update must disable installer systemd mode to avoid restart loops"
  grep -F "release_not_ready" "$patch" >/dev/null || \
    fail "integration patch must report release_not_ready when latest Release is not refreshed yet"
  grep -F "90*time.Minute" "$patch" >/dev/null || \
    fail "self-update must allow slow release downloads to finish"
  grep -F -- "--property=RuntimeMaxSec=90min" "$patch" >/dev/null || \
    fail "systemd self-update unit must have a long runtime cap for slow downloads"
  grep -F 'cpaPlusSystemdRunProxySetenvArgs' "$patch" >/dev/null || \
    fail "systemd self-update unit must explicitly forward proxy env vars"
  grep -F "http_proxy" "$patch" >/dev/null || \
    fail "systemd self-update unit must forward http_proxy for GitHub downloads"
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
  grep -F 'WorkingDirectory=$(systemd_path_escape "$APP_DIR")' "$ROOT_DIR/scripts/install-release.sh" >/dev/null \
    || fail "install-release.sh systemd WorkingDirectory must use systemd path escaping"
  if grep -F 'WorkingDirectory=$(systemd_quote "$APP_DIR")' "$ROOT_DIR/scripts/install-release.sh" >/dev/null; then
    fail "install-release.sh systemd WorkingDirectory must not use shell-style quotes"
  fi
  grep -F 'ExecStart=' "$ROOT_DIR/scripts/install-release.sh" | grep -F '"' >/dev/null \
    || fail "install-release.sh systemd ExecStart must quote APP_DIR"
  grep -F 'WorkingDirectory=$(systemd_path_escape "$APP_DIR")' "$ROOT_DIR/scripts/install-systemd.sh" >/dev/null \
    || fail "install-systemd.sh systemd WorkingDirectory must use systemd path escaping"
  if grep -F 'WorkingDirectory=$(systemd_quote "$APP_DIR")' "$ROOT_DIR/scripts/install-systemd.sh" >/dev/null; then
    fail "install-systemd.sh systemd WorkingDirectory must not use shell-style quotes"
  fi
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

test_readme_is_public_repo_safe() {
  local readme="$ROOT_DIR/README.md"
  if grep -n "私有仓库" "$readme" >/dev/null; then
    fail "README must not describe this public repository as private"
  fi
  if grep -n "gh auth login" "$readme" >/dev/null; then
    fail "README public install path must not require gh auth login"
  fi
  grep -F "curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | bash" "$readme" >/dev/null || \
    fail "README must keep a public curl install command"
  if grep -nE "不要复制到公开 Issue|截图|PR" "$readme" >/dev/null; then
    fail "README should not include personal/internal privacy reminder wording"
  fi
  grep -F "secrets.env" "$ROOT_DIR/.gitignore" >/dev/null || \
    fail ".gitignore must ignore generated secrets.env"
  grep -F "config.yaml" "$ROOT_DIR/.gitignore" >/dev/null || \
    fail ".gitignore must ignore local config.yaml"
  for pattern in ".env.*" "auths/*" "data/" "logs/" "*.sqlite" "*.db" "*.pem" "*.p12" "id_rsa*" "token.txt" "cookie.txt" "*.bak.*"; do
    grep -F "$pattern" "$ROOT_DIR/.gitignore" >/dev/null || \
      fail ".gitignore must ignore privacy pattern ${pattern}"
  done
}

test_sanitized_examples_are_tracked() {
  local config_example="$ROOT_DIR/examples/config.yaml"
  local secrets_example="$ROOT_DIR/examples/secrets.env"
  [[ -f "$config_example" ]] || fail "repository must include sanitized examples/config.yaml"
  [[ -f "$secrets_example" ]] || fail "repository must include sanitized examples/secrets.env"
  grep -F "CHANGE_ME_MANAGEMENT_KEY" "$config_example" >/dev/null || \
    fail "example config must use a placeholder management key"
  grep -F "CHANGE_ME_CLIENT_API_KEY" "$config_example" >/dev/null || \
    fail "example config must use a placeholder client API key"
  grep -F "CPA_PLUS_MANAGEMENT_KEY=CHANGE_ME_MANAGEMENT_KEY" "$secrets_example" >/dev/null || \
    fail "example secrets.env must use placeholder management key"
  grep -F "CPA_PLUS_API_KEY=CHANGE_ME_CLIENT_API_KEY" "$secrets_example" >/dev/null || \
    fail "example secrets.env must use placeholder API key"
  if grep -REn '(sk-[A-Za-z0-9_-]{12,}|AIza[0-9A-Za-z_-]{20,}|ghp_[0-9A-Za-z_]+|github_pat_|xox[baprs]-|AKIA[0-9A-Z]{16})' "$config_example" "$secrets_example" >/dev/null; then
    fail "sanitized examples must not contain token-looking values"
  fi
}

main() {
  test_workflow_ignores_source_commit_for_change_detection
  test_workflow_detects_upstream_before_heavy_steps
  test_workflow_keeps_only_latest_release
  test_workflow_release_notes_include_refresh_time
  test_local_publish_script_recreates_latest_release
  test_update_command_fetches_installer_from_main_by_default
  test_plus_build_version_uses_tag_and_commit
  test_core_patch_excludes_non_runtime_files
  test_plus_version_checks_use_integrated_backend
  test_dashboard_exposes_cpa_plus_update_button
  test_ai_provider_overlay_keeps_toolbar_i18n
  test_overlay_keeps_plugin_and_account_action_i18n
  test_overlay_keeps_plugin_store_i18n
  test_plus_web_integrates_account_action_candidate_paths
  test_patch_persists_cpa_plus_update_backend
  test_generated_scripts_quote_single_quote_app_dir
  test_systemd_unit_quotes_paths_with_spaces
  test_generated_runtime_scripts_support_systemd_mode
  test_default_config_is_local_only_and_public_flag_is_explicit
  test_readme_quotes_gh_api_urls
  test_readme_is_public_repo_safe
  test_sanitized_examples_are_tracked
  echo "release script regression tests passed"
}

main "$@"
