#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${CPA_PLUS_IMAGE:-cpa-plus:auto}"
OUTPUT_SOURCE="${CPA_PLUS_OUTPUT_SOURCE:-${ROOT_DIR}/.build/out/CLIProxyAPI}"
NO_PREPARE=0

usage() {
  cat <<USAGE
Usage: $0 [options passed to prepare-source.sh]

Build Docker image from auto-generated CPA-PLUS source.

Environment:
  CPA_PLUS_IMAGE          Docker image tag. Default: cpa-plus:auto
  CPA_PLUS_OUTPUT_SOURCE  Prepared source directory. Default: .build/out/CLIProxyAPI

Special options:
  --no-prepare            Reuse existing prepared source and only run docker build.
  -h, --help              Show this help.
USAGE
}

ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-prepare) NO_PREPARE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) ARGS+=("$1"); shift ;;
  esac
done

if [[ "${NO_PREPARE}" != "1" ]]; then
  "${ROOT_DIR}/cpa-plus-core/prepare-source.sh" "${ARGS[@]}"
fi

if [[ ! -f "${OUTPUT_SOURCE}/Dockerfile" ]]; then
  echo "Prepared source does not contain Dockerfile: ${OUTPUT_SOURCE}" >&2
  exit 1
fi

echo "==> Building Docker image ${IMAGE} from ${OUTPUT_SOURCE}"
docker build -t "${IMAGE}" "${OUTPUT_SOURCE}"
echo "==> Built ${IMAGE}"
