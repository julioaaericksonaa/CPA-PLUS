# CPA-PLUS Integrated Deployment

CPA-PLUS is a CLIProxyAPI fork that embeds the CPA Manager Plus panel into the same
process and exposes integrated Plus extension APIs on the same `8317` port.

Current implementation status:

- Included: bundled Plus panel, unified CPA management-key login, integrated
  `/usage-service/info` detection, Plus status endpoints, and SQLite-backed
  model price persistence.
- In progress: request monitoring, usage import/export, API-key aliases,
  dashboard analytics, collector lifecycle, and server-side Codex inspection.
  Those Plus pages may still show unavailable states until their corresponding
  `/v0/management/plus/*` backends are ported.

## Access

- API base: `http://host:8317`
- Management panel: `http://host:8317/management.html`

## Required management key

Configure one key shared by the built-in Management API and CPA-PLUS panel:

```yaml
remote-management:
  allow-remote: true
  secret-key: "change-me"
```

## Persistent data

Mount both auth and Plus data:

```bash
-v ./auths:/root/.cli-proxy-api \
-v ./data:/CLIProxyAPI/data
```

Integrated Plus data is stored in `./data/usage.sqlite` by default. At this
stage the database is used for model price persistence; usage statistics will
use the same data path when the collector and usage-event store are ported.

With Docker Compose, the Plus data path is controlled by `CLI_PROXY_PLUS_DATA_PATH` and defaults to `./data` on the host.
