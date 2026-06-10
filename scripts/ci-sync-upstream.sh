#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${CPA_PLUS_OUTPUT_SOURCE:-${ROOT_DIR}/.build/out/CLIProxyAPI}"
PATCH_FILE="${ROOT_DIR}/patches/cliproxyapi/0001-cpa-plus-integration.patch"
META_FILE="${OUT_DIR}/.cpa-plus-auto-build.env"

"${ROOT_DIR}/cpa-plus-core/prepare-source.sh" --skip-tests "$@"

if [[ ! -f "${META_FILE}" ]]; then
  echo "missing build metadata: ${META_FILE}" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "${META_FILE}"

mkdir -p "$(dirname "${PATCH_FILE}")"
(
  cd "${OUT_DIR}"
  git diff --binary --cached HEAD -- . \
    ':(exclude).github/**' \
    ':(exclude)README.md' \
    ':(exclude)README_CN.md' \
    ':(exclude)README_JA.md' \
    ':(exclude)Dockerfile' \
    ':(exclude)docker-compose.yml' \
    ':(exclude)docs/**' \
    ':(exclude)scripts/**' \
    ':(exclude)test/**' \
    ':(exclude)web/manager-plus/**' \
    ':(exclude)docs/superpowers/**'
) > "${PATCH_FILE}"

printf '%s\n' "${CLI_UPSTREAM_COMMIT}" > "${ROOT_DIR}/CLI_UPSTREAM_BASE"
printf '%s\n' "${PLUS_UPSTREAM_COMMIT}" > "${ROOT_DIR}/PLUS_UPSTREAM_BASE"
# In binary-only mode there is no separate integrated source branch. Keep the
# previous maintenance commit as patch provenance for generated build metadata.
git -C "${ROOT_DIR}" rev-parse HEAD > "${ROOT_DIR}/CPA_PLUS_SOURCE_COMMIT"

cat <<SYNC
CLI_UPSTREAM_VERSION=${CLI_UPSTREAM_VERSION}
CLI_UPSTREAM_COMMIT=${CLI_UPSTREAM_COMMIT}
PLUS_UPSTREAM_VERSION=${PLUS_UPSTREAM_VERSION}
PLUS_UPSTREAM_COMMIT=${PLUS_UPSTREAM_COMMIT}
PATCH_FILE=${PATCH_FILE}
SYNC

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  {
    echo "cli_version=${CLI_UPSTREAM_VERSION}"
    echo "cli_commit=${CLI_UPSTREAM_COMMIT}"
    echo "plus_version=${PLUS_UPSTREAM_VERSION}"
    echo "plus_commit=${PLUS_UPSTREAM_COMMIT}"
  } >> "${GITHUB_OUTPUT}"
fi
