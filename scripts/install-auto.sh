#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export CPA_PLUS_IMAGE="${CPA_PLUS_IMAGE:-cpa-plus:auto}"
export CPA_PLUS_OUTPUT_SOURCE="${CPA_PLUS_OUTPUT_SOURCE:-${ROOT_DIR}/.build/out/CLIProxyAPI}"

"${ROOT_DIR}/cpa-plus-core/build-docker.sh" "$@"

if [[ ! -f "${ROOT_DIR}/config.yaml" ]]; then
  cp "${CPA_PLUS_OUTPUT_SOURCE}/config.example.yaml" "${ROOT_DIR}/config.yaml"
  cat <<MSG

Created ${ROOT_DIR}/config.yaml from generated config.example.yaml.
Edit remote-management.secret-key before exposing the service remotely.
MSG
fi

mkdir -p "${ROOT_DIR}/auths" "${ROOT_DIR}/data" "${ROOT_DIR}/logs"
docker compose -f "${ROOT_DIR}/compose.auto.yml" up -d
