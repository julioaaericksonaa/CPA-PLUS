#!/usr/bin/env bash
# Shared helpers for local upstream sync scripts. Source this file; do not run directly.

set -euo pipefail

repo_root() {
  git rev-parse --show-toplevel
}

log() {
  printf '[local-update] %s\n' "$*" >&2
}

die() {
  printf '[local-update] ERROR: %s\n' "$*" >&2
  exit 1
}

run_cmd() {
  if [[ "${DRY_RUN:-0}" == "1" ]]; then
    printf '[dry-run]'
    printf ' %q' "$@"
    printf '\n'
  else
    "$@"
  fi
}

require_clean_worktree() {
  if [[ "${ALLOW_DIRTY:-0}" == "1" ]]; then
    log 'ALLOW_DIRTY=1 set; skipping clean worktree check'
    return
  fi
  local status
  status=$(git status --short)
  [[ -z "$status" ]] || die "worktree is not clean. Commit/stash first, or set ALLOW_DIRTY=1.\n$status"
}

ensure_fetch_remote() {
  local name=$1
  local url=$2
  if git remote get-url "$name" >/dev/null 2>&1; then
    return
  fi
  run_cmd git remote add "$name" "$url"
}

disable_remote_push() {
  local name=$1
  if git remote get-url "$name" >/dev/null 2>&1; then
    run_cmd git remote set-url --push "$name" DISABLED
  fi
}

remote_default_branch() {
  local remote=$1
  local fallback=${2:-main}
  local ref
  ref=$(git ls-remote --symref "$remote" HEAD 2>/dev/null | awk '/^ref:/ { sub("refs/heads/", "", $2); print $2; exit }' || true)
  printf '%s\n' "${ref:-$fallback}"
}

ensure_cli_remote_layout() {
  if ! git remote get-url cli-upstream >/dev/null 2>&1; then
    local origin_url=''
    origin_url=$(git remote get-url origin 2>/dev/null || true)
    if [[ "$origin_url" == *github.com/router-for-me/CLIProxyAPI* ]]; then
      run_cmd git remote rename origin cli-upstream
    else
      run_cmd git remote add cli-upstream https://github.com/router-for-me/CLIProxyAPI.git
    fi
  fi
  disable_remote_push cli-upstream
}

ensure_plus_remote_layout() {
  ensure_fetch_remote plus-upstream https://github.com/seakee/CPA-Manager-Plus.git
  disable_remote_push plus-upstream
}
