#!/usr/bin/env python3
"""Apply CPA-PLUS integrated API path transforms to CPA-Manager-Plus web files."""
from __future__ import annotations

import sys
from pathlib import Path

REPLACEMENTS = [
    ("buildUrl(base, '/status')", "buildUrl(base, '/v0/management/plus/status')"),
    ("'/v0/management/usage'", "'/v0/management/plus/usage'"),
    ("'/v0/management/model-prices'", "'/v0/management/plus/model-prices'"),
    ("'/v0/management/model-prices/sync'", "'/v0/management/plus/model-prices/sync'"),
    ("'/v0/management/api-key-aliases'", "'/v0/management/plus/api-key-aliases'"),
    ("`/v0/management/api-key-aliases/${encodeURIComponent(apiKeyHash)}`", "`/v0/management/plus/api-key-aliases/${encodeURIComponent(apiKeyHash)}`"),
    ("`/v0/management/api-key-aliases/${hash}`", "`/v0/management/plus/api-key-aliases/${hash}`"),
    ("'/v0/management/usage/export'", "'/v0/management/plus/usage/export'"),
    ("'/v0/management/usage/import'", "'/v0/management/plus/usage/import'"),
    ("'/v0/management/dashboard/summary'", "'/v0/management/plus/dashboard/summary'"),
    ("'/v0/management/monitoring/analytics'", "'/v0/management/plus/monitoring/analytics'"),
]

REQUIRED = [
    "/v0/management/plus/status",
    "/v0/management/plus/usage",
    "/v0/management/plus/model-prices",
    "/v0/management/plus/model-prices/sync",
    "/v0/management/plus/api-key-aliases",
    "/v0/management/plus/usage/export",
    "/v0/management/plus/usage/import",
    "/v0/management/plus/dashboard/summary",
    "/v0/management/plus/monitoring/analytics",
]

FORBIDDEN = [
    "'/v0/management/usage'",
    "'/v0/management/model-prices'",
    "'/v0/management/model-prices/sync'",
    "'/v0/management/api-key-aliases'",
    "`/v0/management/api-key-aliases/${encodeURIComponent(apiKeyHash)}`",
    "'/v0/management/usage/export'",
    "'/v0/management/usage/import'",
    "'/v0/management/dashboard/summary'",
    "'/v0/management/monitoring/analytics'",
]


def usage() -> int:
    print("Usage: scripts/patch-plus-web-integrated.py <web-root>")
    print("Example: scripts/patch-plus-web-integrated.py web/manager-plus")
    return 0


def main(argv: list[str]) -> int:
    if len(argv) != 2 or argv[1] in {"-h", "--help"}:
        return usage()
    root = Path(argv[1]).resolve()
    usage_service = root / "src" / "services" / "api" / "usageService.ts"
    if not usage_service.exists():
        print(f"missing {usage_service}", file=sys.stderr)
        return 1
    text = usage_service.read_text()
    for old, new in REPLACEMENTS:
        text = text.replace(old, new)
    missing = [needle for needle in REQUIRED if needle not in text]
    forbidden = [needle for needle in FORBIDDEN if needle in text]
    if missing or forbidden:
        if missing:
            print("missing integrated paths: " + ", ".join(missing), file=sys.stderr)
        if forbidden:
            print("old standalone paths remain: " + ", ".join(forbidden), file=sys.stderr)
        return 1
    usage_service.write_text(text)
    print(f"patched {usage_service}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
