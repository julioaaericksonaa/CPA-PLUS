#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${CPA_PLUS_PUBLISH_ENV:-/root/.config/cpa-plus/publish.env}"
REPO="${CPA_PLUS_REPO:-julioaaericksonaa/CPA-PLUS}"
BINARY="${CPA_PLUS_BINARY:-${ROOT_DIR}/dist/CLIProxyAPI-linux-amd64}"
SHA_FILE="${CPA_PLUS_SHA_FILE:-${BINARY}.sha256}"
RELEASE_NOTES="${CPA_PLUS_RELEASE_NOTES:-${ROOT_DIR}/dist/release-notes.md}"
TARGET_REF="${CPA_PLUS_TARGET_REF:-main}"
SKIP_PUSH="${CPA_PLUS_SKIP_PUSH:-0}"

log() { printf '==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

[[ -f "$ENV_FILE" ]] || fail "missing publish env: ${ENV_FILE}"
# shellcheck disable=SC1090
set -a
source "$ENV_FILE"
set +a

[[ -n "${GITHUB_TOKEN:-}" ]] || fail "GITHUB_TOKEN is not set in ${ENV_FILE}"
GITHUB_USERNAME="${GITHUB_USERNAME:-x-access-token}"

command -v git >/dev/null 2>&1 || fail "git is required"
command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"
command -v sha256sum >/dev/null 2>&1 || fail "sha256sum is required"

cd "$ROOT_DIR"
[[ -x "$BINARY" ]] || fail "missing executable binary: ${BINARY}"
mkdir -p "$(dirname "$SHA_FILE")" "$(dirname "$RELEASE_NOTES")"
sha256sum "$BINARY" > "$SHA_FILE"

if ! git diff --quiet || ! git diff --cached --quiet; then
  fail "working tree has uncommitted changes; commit before publishing latest release"
fi

askpass="$(mktemp)"
cleanup() { rm -f "$askpass"; }
trap cleanup EXIT
cat > "$askpass" <<'ASKPASS'
#!/usr/bin/env sh
case "$1" in
  *Username*) printf '%s\n' "${GITHUB_USERNAME:-x-access-token}" ;;
  *Password*) printf '%s\n' "$GITHUB_TOKEN" ;;
  *) printf '\n' ;;
esac
ASKPASS
chmod 700 "$askpass"

git_auth=(env GIT_TERMINAL_PROMPT=0 GIT_ASKPASS="$askpass" git -c credential.helper=)

if [[ "$SKIP_PUSH" != "1" ]]; then
  log "Pushing HEAD to origin/${TARGET_REF} without cached credentials"
  "${git_auth[@]}" push origin "HEAD:${TARGET_REF}"
fi

log "Updating latest tag"
"${git_auth[@]}" tag -f latest HEAD
"${git_auth[@]}" push origin refs/tags/latest --force

updated_at="$(TZ=Asia/Shanghai date '+%Y-%m-%d %H:%M:%S %Z')"
commit="$(git rev-parse HEAD)"
cat > "$RELEASE_NOTES" <<NOTES
CPA-PLUS latest binary release

Updated at: ${updated_at}

- Commit: ${commit}
- Binary: CLIProxyAPI-linux-amd64
- Release source: local publish script

Download and run on Linux amd64. The bundled Plus panel is served at /management.html.
NOTES

api_base="https://api.github.com/repos/${REPO}"
auth_headers=(
  -H "Authorization: Bearer ${GITHUB_TOKEN}"
  -H "Accept: application/vnd.github+json"
  -H "X-GitHub-Api-Version: 2022-11-28"
)

log "Deleting existing latest release if present"
release_json="$(curl -fsS "${auth_headers[@]}" "${api_base}/releases/tags/latest" 2>/dev/null || true)"
release_id="$(python3 -c 'import json,sys; data=sys.stdin.read().strip(); print(json.loads(data).get("id","") if data else "")' <<<"$release_json")"
if [[ -n "$release_id" ]]; then
  curl -fsS -X DELETE "${auth_headers[@]}" "${api_base}/releases/${release_id}" >/dev/null
fi

log "Creating latest release"
create_payload="$(python3 - "$commit" "$RELEASE_NOTES" <<'PY'
import json
import pathlib
import sys

commit = sys.argv[1]
notes = pathlib.Path(sys.argv[2]).read_text()
print(json.dumps({
    "tag_name": "latest",
    "target_commitish": commit,
    "name": "CPA-PLUS latest",
    "body": notes,
    "draft": False,
    "prerelease": False,
    "make_latest": "true",
}))
PY
)"
created_json="$(curl -fsS -X POST "${auth_headers[@]}" -d "$create_payload" "${api_base}/releases")"
upload_url="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["upload_url"].split("{",1)[0])' <<<"$created_json")"

upload_asset() {
  local path="$1"
  local name
  name="$(basename "$path")"
  log "Uploading ${name}"
  curl -fsS -X POST \
    "${auth_headers[@]}" \
    -H "Content-Type: application/octet-stream" \
    --data-binary "@${path}" \
    "${upload_url}?name=${name}" >/dev/null
}

upload_asset "$BINARY"
upload_asset "$SHA_FILE"

log "Published latest release for ${commit}"
