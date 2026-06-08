#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_REPO="${1:-/root/code/cpa-plus-merge-study/CLIProxyAPI}"
BASE_REF="${2:-cli-upstream/main}"
OUT_PATCH="${ROOT_DIR}/patches/cliproxyapi/0001-cpa-plus-integration.patch"

if [[ ! -d "${SOURCE_REPO}/.git" ]]; then
  echo "Usage: $0 /path/to/integrated/CLIProxyAPI [base-ref]" >&2
  exit 2
fi

mkdir -p "$(dirname "${OUT_PATCH}")"
(
  cd "${SOURCE_REPO}"
  git diff --binary "${BASE_REF}..HEAD" -- . \
    ':(exclude)web/manager-plus/**' \
    ':(exclude)docs/superpowers/**'
) > "${OUT_PATCH}"

git -C "${SOURCE_REPO}" rev-parse HEAD > "${ROOT_DIR}/CPA_PLUS_SOURCE_COMMIT"
git -C "${SOURCE_REPO}" rev-parse "${BASE_REF}" > "${ROOT_DIR}/CLI_UPSTREAM_BASE"

echo "Regenerated ${OUT_PATCH}"
