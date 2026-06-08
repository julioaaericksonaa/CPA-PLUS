#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

fail() { echo "FAIL: $*" >&2; exit 1; }

for script in \
  scripts/local-update-cli.sh \
  scripts/local-update-plus-web.sh \
  scripts/local-update-all.sh \
  scripts/patch-plus-web-integrated.py; do
  [[ -x "$script" ]] || fail "$script must exist and be executable"
done

scripts/local-update-cli.sh --help | grep -q 'cli-upstream' || fail 'CLI updater help must mention cli-upstream'
scripts/local-update-plus-web.sh --help | grep -q 'plus-upstream' || fail 'Plus updater help must mention plus-upstream'
scripts/local-update-all.sh --help | grep -q 'local-update-cli' || fail 'All updater help must mention local-update-cli'

fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/src/services/api"
cat > "$fixture/src/services/api/usageService.ts" <<'TS'
axios.get(buildUrl(base, '/status'));
axios.get(buildUrl(base, '/v0/management/usage'));
axios.get(buildUrl(base, '/v0/management/model-prices'));
axios.put(buildUrl(base, '/v0/management/model-prices'));
axios.post(buildUrl(base, '/v0/management/model-prices/sync'));
axios.get(buildUrl(base, '/v0/management/api-key-aliases'));
axios.put(buildUrl(base, '/v0/management/api-key-aliases'));
axios.delete(buildUrl(base, `/v0/management/api-key-aliases/${hash}`));
axios.get(buildUrl(base, '/v0/management/usage/export'));
axios.post(buildUrl(base, '/v0/management/usage/import'));
axios.get(buildUrl(base, '/v0/management/dashboard/summary'));
axios.post(buildUrl(base, '/v0/management/monitoring/analytics'));
TS
scripts/patch-plus-web-integrated.py "$fixture"
patched=$(cat "$fixture/src/services/api/usageService.ts")
for path in \
  '/v0/management/plus/status' \
  '/v0/management/plus/usage' \
  '/v0/management/plus/model-prices' \
  '/v0/management/plus/model-prices/sync' \
  '/v0/management/plus/api-key-aliases' \
  '/v0/management/plus/usage/export' \
  '/v0/management/plus/usage/import' \
  '/v0/management/plus/dashboard/summary' \
  '/v0/management/plus/monitoring/analytics'; do
  grep -q "$path" <<<"$patched" || fail "patched usageService missing $path"
done

if grep -q "'/v0/management/usage'" <<<"$patched"; then
  fail 'old /v0/management/usage path survived transform'
fi

echo 'local update scripts contract OK'
