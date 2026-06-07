# CPA-PLUS Integrated Design

## Outcome

Fork `CLIProxyAPI` as the primary project and integrate `CPA-Manager-Plus` into it. The final distribution runs as one Go binary, one Docker image, and one HTTP port (`8317`). Users open `http://host:8317/management.html` to access the full Plus management panel while the normal CPA API remains available on the same server.

## Goals

- Maintain one project and one deployment artifact.
- Serve the full CPA Manager Plus panel from CLIProxyAPI's existing `/management.html` route.
- Keep the original CPA API and Management API behavior intact.
- Use one management credential: `remote-management.secret-key`.
- Add Plus persistence and usage analytics in-process, backed by SQLite.
- Avoid an independent Manager Server process and avoid a second Docker service.

## Non-goals for the first implementation

- Do not keep the old `cmp_admin_...` Manager Server admin-key setup flow.
- Do not require users to enter or store a separate CPA base URL in the integrated panel.
- Do not redesign CPA provider/auth/model behavior.
- Do not optimize collector internals before the integrated HTTP/queue path works.

## Runtime Architecture

```text
Browser
  -> CLIProxyAPI :8317
      -> /management.html
          serves bundled CPA Manager Plus single-file panel

      -> /v0/management/*
          existing CPA Management API

      -> /v0/management/plus/*
          integrated Plus APIs:
          - service status/info
          - usage summaries and events
          - model prices
          - API key aliases
          - import/export
          - collector status/config

      -> Plus collector worker
          consumes CPA usage queue through same-process/localhost-compatible paths
          writes SQLite

      -> SQLite
          default: ./data/usage.sqlite inside the CLIProxyAPI working directory
```

## Frontend Integration

Copy or vendor the CPA Manager Plus frontend into the CLIProxyAPI fork under `web/manager-plus/` or an equivalent clearly bounded directory. The Docker build compiles it with Node/Vite and emits one `management.html` artifact.

CLIProxyAPI should prefer the bundled Plus panel for `/management.html`. The integrated fork should default `remote-management.disable-auto-update-panel` semantics to avoid replacing the Plus panel with the older remote control panel. If the upstream auto-updater remains available, it must not overwrite the bundled Plus panel unless the user explicitly opts into external panel updates.

The frontend adds an integrated mode. In integrated mode:

- API calls are same-origin.
- CPA management calls continue to use `/v0/management/*`.
- Plus-only calls use `/v0/management/plus/*`.
- The setup flow for standalone Manager Server is bypassed.
- Login asks for only the CPA management key.

## Backend Module Integration

Add a Plus module to the CLIProxyAPI fork, for example:

```text
internal/plusmanager/
  app/
  collector/
  config/
  httpapi/
  model/
  service/
  store/
  usage/
  worker/
```

Source material comes from `CPA-Manager-Plus/apps/manager-server/internal/*`, but it must be adapted for CLIProxyAPI:

- Remove independent `HTTP_ADDR` server startup.
- Remove generated Manager Server admin key handling.
- Remove first-run setup that binds an external CPA URL.
- Remove encrypted storage of a separate CPA Management Key.
- Reuse CLIProxyAPI's current configuration, logger, lifecycle, and management auth.
- Register routes through CLIProxyAPI's existing Gin server extension points.

## Authentication

All integrated Plus endpoints use the same authorization model as CPA Management API. The single credential is configured as:

```yaml
remote-management:
  allow-remote: true
  secret-key: "your-management-key"
```

The same key authorizes:

- CPA configuration management
- provider/account/API-key management
- request statistics
- model prices
- API-key aliases
- import/export
- collector controls

No `cmp_admin_...` key is generated or required in integrated mode.

## Route Design

Keep existing CPA routes stable. Add Plus-only API routes under a collision-safe prefix:

```text
GET  /v0/management/plus/status
GET  /v0/management/plus/info
GET  /v0/management/plus/usage/summary
GET  /v0/management/plus/usage/events
GET  /v0/management/plus/model-prices
PUT  /v0/management/plus/model-prices
GET  /v0/management/plus/api-key-aliases
PUT  /v0/management/plus/api-key-aliases
POST /v0/management/plus/usage/import
GET  /v0/management/plus/usage/export
GET  /v0/management/plus/collector/status
PUT  /v0/management/plus/collector/config
```

Compatibility wrappers may be added only when the frontend migration needs them, but new integrated code should use `/v0/management/plus/*`.

## Configuration and Storage

Add a top-level integrated Plus configuration block:

```yaml
plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
  collector-mode: auto
  poll-interval-ms: 1000
```

Default behavior:

- `plus-manager.enabled` defaults to `true` in the integrated fork.
- If `db-path` is empty, resolve it under `data-dir` as `usage.sqlite`.
- If both paths are empty, use `./data/usage.sqlite` relative to the CLIProxyAPI working directory.
- Docker examples mount `./data:/CLIProxyAPI/data` to persist statistics.

## Usage Collection

The first implementation uses the existing CPA usage queue behavior instead of inventing a new event bus. The collector should work with CPA's current `usage-statistics-enabled` and `usage-queue` semantics.

Initial priority:

1. Ensure the collector can consume usage events from the same running CPA process through the existing management queue contract.
2. Preserve HTTP/RESP/PubSub collector compatibility where feasible.
3. Automatically enable or clearly require `usage-statistics-enabled: true` when `plus-manager.collector-enabled` is true.

A later optimization may connect the collector directly to an internal usage plugin or event stream. That is not required for the first full integration.

## Docker Build

The Dockerfile becomes a multi-stage build:

1. Node stage builds the Plus frontend into `management.html`.
2. Go stage builds CLIProxyAPI with the bundled panel and Plus backend module.
3. Runtime stage copies the single binary and exposes `8317`.

Expected deployment:

```bash
docker run -d \
  --name cpa-plus \
  --restart unless-stopped \
  -p 8317:8317 \
  -v ./config.yaml:/CLIProxyAPI/config.yaml \
  -v ./auths:/root/.cli-proxy-api \
  -v ./logs:/CLIProxyAPI/logs \
  -v ./data:/CLIProxyAPI/data \
  yourname/cpa-plus:latest
```

## Phased Implementation

### Phase 1: Bundled Plus panel

- Add a bundled management panel provider.
- Build CPA Manager Plus frontend during Docker builds.
- Serve the bundled panel at `/management.html`.
- Add integrated-mode frontend config.
- Hide or disable standalone Manager Server-only features until backend routes exist.

### Phase 2: Integrated Plus backend

- Port SQLite store and migrations.
- Add Plus config parsing/defaults.
- Register `/v0/management/plus/*` routes in the CLIProxyAPI server.
- Port usage summaries, events, model prices, aliases, and import/export.
- Start the collector worker with the CLIProxyAPI lifecycle.

### Phase 3: Hardening and optimization

- Add direct in-process usage event integration if useful.
- Add hot-reload behavior for Plus config.
- Expand tests and release workflow.
- Document migration from two-container deployment.

## Testing Strategy

- Unit-test config parsing and default path resolution.
- Unit-test bundled panel selection so `/management.html` does not unexpectedly use the remote updater in integrated mode.
- Unit-test Plus route registration under `/v0/management/plus/*` without changing existing `/v0/management/*` behavior.
- Unit-test SQLite store migration and usage aggregation using temporary databases.
- Integration-test a minimal request event flowing into the collector and appearing in usage summary APIs.
- Build-test the Docker multi-stage pipeline or equivalent local build commands.

## Risks

- CPA Manager Plus imports must be rewritten from the original module path.
- Frontend mode detection currently distinguishes standalone Manager Server and CPA control-panel modes; integrated mode must be explicit to avoid broken setup screens.
- Route/auth behavior must not weaken CPA Management security.
- SQLite data must be persisted through Docker volume examples.
- Long-lived divergence from both upstreams is possible; preserve clear module boundaries to ease future rebases.
