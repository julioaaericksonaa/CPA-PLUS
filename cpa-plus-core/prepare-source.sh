#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK_DIR="${CPA_PLUS_WORKDIR:-${ROOT_DIR}/.build}"
CLI_REPO="${CLI_UPSTREAM_REPO:-https://github.com/router-for-me/CLIProxyAPI.git}"
CLI_REF="${CLI_UPSTREAM_REF:-main}"
PLUS_REPO="${PLUS_UPSTREAM_REPO:-https://github.com/seakee/CPA-Manager-Plus.git}"
PLUS_REF="${PLUS_UPSTREAM_REF:-main}"
OUT_DIR="${CPA_PLUS_OUTPUT_SOURCE:-${WORK_DIR}/out/CLIProxyAPI}"
SKIP_LOCK="${CPA_PLUS_SKIP_LOCK:-0}"
SKIP_TESTS="${CPA_PLUS_SKIP_TESTS:-0}"
KEEP_WORK="${CPA_PLUS_KEEP_WORKDIR:-0}"

usage() {
  cat <<USAGE
Usage: $0 [options]

Prepare a materialized CPA-PLUS source tree from upstream CLIProxyAPI + CPA-Manager-Plus.

Options:
  --cli-ref REF       CLIProxyAPI ref/tag/branch to checkout. Default: ${CLI_REF}
  --plus-ref REF      CPA-Manager-Plus ref/tag/branch to checkout. Default: ${PLUS_REF}
  --cli-repo URL      CLIProxyAPI git repository. Default: ${CLI_REPO}
  --plus-repo URL     CPA-Manager-Plus git repository. Default: ${PLUS_REPO}
  --workdir DIR       Temporary/generated workspace. Default: ${WORK_DIR}
  --output DIR        Output source directory. Default: ${OUT_DIR}
  --skip-lock         Do not refresh package-lock.json if missing or stale.
  --skip-tests        Do not run focused Go tests after patching.
  --keep-workdir      Keep clone workspace between runs.
  -h, --help          Show this help.

Environment variables mirror the option names:
  CLI_UPSTREAM_REPO, CLI_UPSTREAM_REF, PLUS_UPSTREAM_REPO, PLUS_UPSTREAM_REF,
  CPA_PLUS_WORKDIR, CPA_PLUS_OUTPUT_SOURCE, CPA_PLUS_SKIP_LOCK, CPA_PLUS_SKIP_TESTS.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --cli-ref) CLI_REF="$2"; shift 2 ;;
    --plus-ref) PLUS_REF="$2"; shift 2 ;;
    --cli-repo) CLI_REPO="$2"; shift 2 ;;
    --plus-repo) PLUS_REPO="$2"; shift 2 ;;
    --workdir) WORK_DIR="$2"; OUT_DIR="${CPA_PLUS_OUTPUT_SOURCE:-$2/out/CLIProxyAPI}"; shift 2 ;;
    --output) OUT_DIR="$2"; shift 2 ;;
    --skip-lock) SKIP_LOCK=1; shift ;;
    --skip-tests) SKIP_TESTS=1; shift ;;
    --keep-workdir) KEEP_WORK=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

PATCH_DIR="${ROOT_DIR}/patches/cliproxyapi"
WEB_PATCHER="${ROOT_DIR}/cpa-plus-web/patch-plus-web-integrated.py"
CLONE_DIR="${WORK_DIR}/src/CLIProxyAPI"
PLUS_DIR="${WORK_DIR}/src/CPA-Manager-Plus"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing required command: $1" >&2; exit 127; }
}

require_cmd git
require_cmd python3
require_cmd rsync

merge_overlay_locales() {
  local overlay_dir="$1"
  local target_dir="$2"
  python3 - "$overlay_dir" "$target_dir" <<'PY'
import json
import sys
from pathlib import Path

overlay_dir = Path(sys.argv[1])
target_dir = Path(sys.argv[2])
overlay_locale_dir = overlay_dir / "src" / "i18n" / "locales"
target_locale_dir = target_dir / "src" / "i18n" / "locales"


def deep_merge(base, overlay):
    if isinstance(base, dict) and isinstance(overlay, dict):
        merged = dict(base)
        for key, value in overlay.items():
            merged[key] = deep_merge(merged.get(key), value)
        return merged
    return overlay


if not overlay_locale_dir.is_dir():
    raise SystemExit(0)

for overlay_file in sorted(overlay_locale_dir.glob("*.json")):
    target_file = target_locale_dir / overlay_file.name
    if target_file.exists():
        base = json.loads(target_file.read_text(encoding="utf-8"))
    else:
        base = {}
    overlay = json.loads(overlay_file.read_text(encoding="utf-8"))
    merged = deep_merge(base, overlay)
    target_file.parent.mkdir(parents=True, exist_ok=True)
    target_file.write_text(json.dumps(merged, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"merged overlay locale {overlay_file.name}")
PY
}

mkdir -p "${WORK_DIR}/src" "$(dirname "${OUT_DIR}")"

if [[ "${KEEP_WORK}" != "1" ]]; then
  rm -rf "${CLONE_DIR}" "${PLUS_DIR}" "${OUT_DIR}"
fi

if [[ ! -d "${CLONE_DIR}/.git" ]]; then
  echo "==> Cloning CLIProxyAPI ${CLI_REF}"
  git clone "${CLI_REPO}" "${CLONE_DIR}"
fi

git -C "${CLONE_DIR}" fetch --tags origin
git -C "${CLONE_DIR}" checkout --force "${CLI_REF}"
git -C "${CLONE_DIR}" reset --hard HEAD
git -C "${CLONE_DIR}" clean -fdx

if [[ ! -d "${PLUS_DIR}/.git" ]]; then
  echo "==> Cloning CPA-Manager-Plus ${PLUS_REF}"
  git clone "${PLUS_REPO}" "${PLUS_DIR}"
fi

git -C "${PLUS_DIR}" fetch --tags origin
git -C "${PLUS_DIR}" checkout --force "${PLUS_REF}"
git -C "${PLUS_DIR}" reset --hard HEAD
git -C "${PLUS_DIR}" clean -fdx

mkdir -p "$(dirname "${OUT_DIR}")"
rsync -a --delete \
  --exclude 'web/manager-plus' \
  "${CLONE_DIR}/" "${OUT_DIR}/"

git -C "${OUT_DIR}" reset --hard HEAD >/dev/null
git -C "${OUT_DIR}" clean -fdx >/dev/null

pushd "${OUT_DIR}" >/dev/null
for patch in "${PATCH_DIR}"/*.patch; do
  [[ -e "${patch}" ]] || continue
  echo "==> Applying patch $(basename "${patch}")"
  git apply --3way --whitespace=nowarn "${patch}"
done
popd >/dev/null

PLUS_WEB_SRC="${PLUS_DIR}/apps/web"
if [[ ! -d "${PLUS_WEB_SRC}" ]]; then
  echo "CPA-Manager-Plus web app not found at ${PLUS_WEB_SRC}" >&2
  exit 1
fi

mkdir -p "${OUT_DIR}/web/manager-plus"
rsync -a --delete \
  --exclude 'node_modules' \
  --exclude 'dist' \
  "${PLUS_WEB_SRC}/" "${OUT_DIR}/web/manager-plus/"

python3 "${WEB_PATCHER}" "${OUT_DIR}/web/manager-plus"

WEB_OVERLAY="${ROOT_DIR}/cpa-plus-web/overlay"
if [[ -d "${WEB_OVERLAY}" ]]; then
  rsync -a \
    --exclude 'src/i18n/locales/*.json' \
    "${WEB_OVERLAY}/" "${OUT_DIR}/web/manager-plus/"
  merge_overlay_locales "${WEB_OVERLAY}" "${OUT_DIR}/web/manager-plus"
fi

if [[ "${SKIP_LOCK}" != "1" && -f "${OUT_DIR}/web/manager-plus/package.json" && ! -f "${OUT_DIR}/web/manager-plus/package-lock.json" ]]; then
  require_cmd npm
  echo "==> Generating package-lock.json for Plus web"
  npm --prefix "${OUT_DIR}/web/manager-plus" install --package-lock-only --ignore-scripts
fi

cli_upstream_version() {
  git -C "${CLONE_DIR}" describe --tags --exact-match 2>/dev/null || \
    git -C "${CLONE_DIR}" describe --tags --abbrev=0 2>/dev/null || \
    printf '%s\n' "${CLI_REF}"
}

plus_upstream_version() {
  local exact nearest short
  short="$(git -C "${PLUS_DIR}" rev-parse --short=8 HEAD)"
  exact="$(git -C "${PLUS_DIR}" describe --tags --exact-match 2>/dev/null || true)"
  if [[ -n "${exact}" ]]; then
    printf '%s+%s\n' "${exact}" "${short}"
    return
  fi
  nearest="$(git -C "${PLUS_DIR}" describe --tags --abbrev=0 2>/dev/null || true)"
  if [[ -n "${nearest}" ]]; then
    printf '%s+%s\n' "${nearest}" "${short}"
  else
    printf '%s\n' "${short}"
  fi
}

cat > "${OUT_DIR}/.cpa-plus-auto-build.env" <<META
CPA_PLUS_BRANCH=main
CLI_UPSTREAM_REPO=${CLI_REPO}
CLI_UPSTREAM_REF=${CLI_REF}
CLI_UPSTREAM_VERSION=$(cli_upstream_version)
CLI_UPSTREAM_COMMIT=$(git -C "${CLONE_DIR}" rev-parse HEAD)
PLUS_UPSTREAM_REPO=${PLUS_REPO}
PLUS_UPSTREAM_REF=${PLUS_REF}
PLUS_UPSTREAM_VERSION=$(plus_upstream_version)
PLUS_UPSTREAM_COMMIT=$(git -C "${PLUS_DIR}" rev-parse HEAD)
PATCH_SOURCE_COMMIT=$(cat "${ROOT_DIR}/CPA_PLUS_SOURCE_COMMIT" 2>/dev/null || true)
PREPARED_AT_UTC=$(date -u +%Y-%m-%dT%H:%M:%SZ)
META

if [[ "${SKIP_TESTS}" != "1" ]]; then
  require_cmd go
  echo "==> Running focused CPA-PLUS Go tests"
  (
    cd "${OUT_DIR}"
    go test ./internal/config ./internal/managementasset ./internal/plusmanager/... ./internal/api ./internal/safemode -count=1
  )
fi

echo "==> CPA-PLUS source prepared at ${OUT_DIR}"
