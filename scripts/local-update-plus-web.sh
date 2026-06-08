#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=scripts/local-update-common.sh
source "$SCRIPT_DIR/local-update-common.sh"

DRY_RUN=0
SKIP_TESTS=0
SKIP_LOCK=0
BRANCH=""
SOURCE_DIR=""

usage() {
  cat <<'USAGE'
Usage: scripts/local-update-plus-web.sh [options]

Sync CPA-Manager-Plus apps/web into web/manager-plus, then re-apply integrated
/v0/management/plus/* API path transforms. Remote used: plus-upstream
(https://github.com/seakee/CPA-Manager-Plus.git), push disabled.

Options:
  --dry-run          Print commands without changing files
  --skip-tests       Skip npm test/build after sync
  --skip-lock        Do not refresh package-lock.json
  --source PATH      Use an existing CPA-Manager-Plus checkout instead of cloning
  --branch NAME      Upstream branch to clone (default: remote HEAD, usually main)
  -h, --help         Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1 ;;
    --skip-tests) SKIP_TESTS=1 ;;
    --skip-lock) SKIP_LOCK=1 ;;
    --source) SOURCE_DIR=${2:?--source requires a value}; shift ;;
    --branch) BRANCH=${2:?--branch requires a value}; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

cd "$(repo_root)"
require_clean_worktree
ensure_plus_remote_layout
: "${BRANCH:=$(remote_default_branch plus-upstream main)}"

TMP_DIR=""
cleanup() {
  if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

if [[ -z "$SOURCE_DIR" ]]; then
  TMP_DIR=$(mktemp -d /tmp/cpa-manager-plus.XXXXXX)
  PLUS_URL=$(git remote get-url plus-upstream)
  run_cmd git clone --depth=1 --branch "$BRANCH" "$PLUS_URL" "$TMP_DIR"
  SOURCE_DIR="$TMP_DIR"
fi

WEB_SRC="$SOURCE_DIR/apps/web"
if [[ "$DRY_RUN" != "1" ]]; then
  [[ -d "$WEB_SRC" ]] || die "Plus web source not found: $WEB_SRC"
fi

run_cmd rsync -a --delete \
  --exclude node_modules \
  --exclude dist \
  --exclude .env \
  --exclude '.env.*' \
  --exclude package-lock.json \
  --exclude README.md \
  "$WEB_SRC/" web/manager-plus/

run_cmd "$SCRIPT_DIR/patch-plus-web-integrated.py" web/manager-plus

if [[ "$SKIP_LOCK" != "1" ]]; then
  run_cmd npm --prefix web/manager-plus install --package-lock-only
fi

if [[ "$SKIP_TESTS" != "1" ]]; then
  run_cmd npm --prefix web/manager-plus ci
  run_cmd npm --prefix web/manager-plus test
  run_cmd npm --prefix web/manager-plus run build
  run_cmd rm -rf web/manager-plus/node_modules web/manager-plus/dist
fi

log "Plus web sync complete from ${SOURCE_DIR} (${BRANCH})"
log "Review git diff, then commit if the sync is good."
