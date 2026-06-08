# CPA-PLUS Integrated Deployment

CPA-PLUS is a CLIProxyAPI fork that embeds the CPA Manager Plus panel into the same
process and exposes integrated Plus extension APIs on the same `8317` port.

Current implementation status:

- Included: bundled Plus panel, unified CPA management-key login, integrated
  `/usage-service/info` and `/usage-service/config` detection, Plus status
  endpoints, SQLite-backed model prices and LiteLLM sync, API-key aliases,
  request usage storage, usage import/export, dashboard summary, monitoring
  analytics, and the in-process usage collector lifecycle.
- The Plus pages use the same `remote-management.secret-key` and call
  `/v0/management/plus/*` APIs on the same origin/port. Server-side Codex
  inspection remains separate from the core CPA-PLUS monitoring flow.

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

Integrated Plus data is stored in `./data/usage.sqlite` by default. The same
SQLite database stores model prices, API-key aliases, and collected usage
events consumed by the Plus monitoring/dashboard pages.

With Docker Compose, the Plus data path is controlled by `CLI_PROXY_PLUS_DATA_PATH` and defaults to `./data` on the host.
