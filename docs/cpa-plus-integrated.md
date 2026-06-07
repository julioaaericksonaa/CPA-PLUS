# CPA-PLUS Integrated Deployment

CPA-PLUS is a CLIProxyAPI fork with CPA Manager Plus integrated into the same process.

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

Plus statistics are stored in `./data/usage.sqlite` by default.

With Docker Compose, the Plus data path is controlled by `CLI_PROXY_PLUS_DATA_PATH` and defaults to `./data` on the host.
