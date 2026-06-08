#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=scripts/local-update-common.sh
source "$SCRIPT_DIR/local-update-common.sh"

DRY_RUN=0
SKIP_TESTS=0
BRANCH=""

usage() {
  cat <<'USAGE'
Usage: scripts/local-update-cli.sh [options]

Fetch and merge CLIProxyAPI upstream into the current local integration branch.
Remote used: cli-upstream (https://github.com/router-for-me/CLIProxyAPI.git), push disabled.

Options:
  --dry-run       Print commands without changing files
  --skip-tests    Skip Go verification after merge
  --branch NAME   Upstream branch to merge (default: remote HEAD, usually main)
  -h, --help      Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1 ;;
    --skip-tests) SKIP_TESTS=1 ;;
    --branch) BRANCH=${2:?--branch requires a value}; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

cd "$(repo_root)"
require_clean_worktree
ensure_cli_remote_layout
: "${BRANCH:=$(remote_default_branch cli-upstream main)}"
run_cmd git fetch cli-upstream
run_cmd git merge --no-edit "cli-upstream/$BRANCH"
if [[ "$SKIP_TESTS" != "1" ]]; then
  run_cmd docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine sh -c 'go test ./internal/config ./internal/managementasset ./internal/plusmanager/... ./internal/api -count=1 && go build ./cmd/server'
  run_cmd rm -f server
fi
log "CLI upstream sync complete from cli-upstream/$BRANCH"
