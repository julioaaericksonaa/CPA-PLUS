#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${CPA_PLUS_OUTPUT_SOURCE:-${ROOT_DIR}/.build/out/CLIProxyAPI}"
DIST_DIR="${CPA_PLUS_DIST_DIR:-${ROOT_DIR}/dist}"
BINARY_NAME="${CPA_PLUS_BINARY_NAME:-CLIProxyAPI-linux-amd64}"
SKIP_TESTS="${CPA_PLUS_SKIP_TESTS:-0}"
VERSION="${CPA_PLUS_VERSION:-v7.1.54-plus.2}"
COMMIT_SUFFIX="${CPA_PLUS_COMMIT_SUFFIX:-plus}"

"${ROOT_DIR}/cpa-plus-core/prepare-source.sh" "$@"

META_FILE="${OUT_DIR}/.cpa-plus-auto-build.env"
if [[ -f "${META_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${META_FILE}"
fi
CLI_UPSTREAM_VERSION="${CLI_UPSTREAM_VERSION:-${VERSION}}"
CLI_UPSTREAM_COMMIT="${CLI_UPSTREAM_COMMIT:-$(git -C "${OUT_DIR}" rev-parse HEAD 2>/dev/null || echo none)}"
PLUS_UPSTREAM_VERSION="${PLUS_UPSTREAM_VERSION:-dev}"
PLUS_UPSTREAM_COMMIT="${PLUS_UPSTREAM_COMMIT:-none}"

if [[ ! -d "${OUT_DIR}/web/manager-plus" ]]; then
  echo "missing Plus web source: ${OUT_DIR}/web/manager-plus" >&2
  exit 1
fi

npm --prefix "${OUT_DIR}/web/manager-plus" ci
VERSION="${PLUS_UPSTREAM_VERSION}" npm --prefix "${OUT_DIR}/web/manager-plus" run build
cp "${OUT_DIR}/web/manager-plus/dist/index.html" "${OUT_DIR}/internal/managementasset/bundled/management.html"

if [[ "${SKIP_TESTS}" != "1" ]]; then
  (
    cd "${OUT_DIR}"
    go test ./internal/config ./internal/managementasset ./internal/plusmanager/... ./internal/api ./internal/safemode -count=1
  )
fi

mkdir -p "${DIST_DIR}"
(
  cd "${OUT_DIR}"
  CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildvcs=false \
    -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo none)-${COMMIT_SUFFIX}' -X 'main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'main.CLIUpstreamVersion=${CLI_UPSTREAM_VERSION}' -X 'main.CLIUpstreamCommit=${CLI_UPSTREAM_COMMIT}' -X 'main.PlusUpstreamVersion=${PLUS_UPSTREAM_VERSION}' -X 'main.PlusUpstreamCommit=${PLUS_UPSTREAM_COMMIT}'" \
    -o "${DIST_DIR}/${BINARY_NAME}" ./cmd/server/
)
chmod 755 "${DIST_DIR}/${BINARY_NAME}"
echo "Built ${DIST_DIR}/${BINARY_NAME}"
