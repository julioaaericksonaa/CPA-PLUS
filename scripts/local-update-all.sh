#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/local-update-all.sh [options]

Run local-update-cli.sh and local-update-plus-web.sh in sequence for a full local
CPA-PLUS upstream refresh. No git push is performed.

Options:
  --dry-run     Print commands without changing files
  --skip-tests  Skip Go/npm verification in both steps
  -h, --help    Show this help
USAGE
}

DRY_ARGS=()
TEST_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_ARGS=(--dry-run) ;;
    --skip-tests) TEST_ARGS=(--skip-tests) ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
  shift
done

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
"$SCRIPT_DIR/local-update-cli.sh" "${DRY_ARGS[@]}" "${TEST_ARGS[@]}"
"$SCRIPT_DIR/local-update-plus-web.sh" "${DRY_ARGS[@]}" "${TEST_ARGS[@]}"
