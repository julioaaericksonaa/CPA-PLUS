#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export CPA_PLUS_IMAGE="${CPA_PLUS_IMAGE:-cpa-plus:auto}"
export CPA_PLUS_OUTPUT_SOURCE="${CPA_PLUS_OUTPUT_SOURCE:-${ROOT_DIR}/.build/out/CLIProxyAPI}"

"${ROOT_DIR}/cpa-plus-core/build-docker.sh" "$@"
docker compose -f "${ROOT_DIR}/compose.auto.yml" up -d
