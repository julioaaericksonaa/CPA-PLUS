# CPA-PLUS Integrated Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a CLIProxyAPI fork that serves the CPA Manager Plus panel and integrated Plus APIs from the same `8317` process.

**Architecture:** Add bounded integration seams first: Plus config, bundled panel serving, and route registration. Then port Manager Plus backend pieces into `internal/plusmanager` and wire them into CLIProxyAPI lifecycle. Keep existing CPA Management API stable and put new endpoints under `/v0/management/plus/*`.

**Tech Stack:** Go 1.26, Gin, YAML config, SQLite-compatible Manager Plus store code, Node 22/Vite/React for the panel, Docker multi-stage build.

---

## Parallelization Strategy

Use separate worktrees/branches for independent slices, then integrate in order:

1. `feature/cpa-plus-config-panel`: config + panel serving seam.
2. `feature/cpa-plus-frontend`: vendored frontend integrated mode + build output.
3. `feature/cpa-plus-backend-core`: backend store/models/services under `internal/plusmanager`.
4. `feature/cpa-plus-routes-collector`: route registration + collector lifecycle.
5. `feature/cpa-plus-docker-docs`: Docker and deployment docs.

Do not let two agents edit the same files at the same time. Expected write sets are listed per task.

---

### Task 1: Add Plus Manager config defaults

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/parse.go`
- Create: `internal/config/plus_manager_test.go`
- Modify: `config.example.yaml`

- [ ] **Step 1: Write failing config tests**

Create `internal/config/plus_manager_test.go`:

```go
package config

import "testing"

func TestParseConfigBytesPlusManagerDefaults(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`remote-management:
  secret-key: test-key
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if !cfg.PlusManager.Enabled {
		t.Fatalf("PlusManager.Enabled = false, want true")
	}
	if cfg.PlusManager.DataDir != "./data" {
		t.Fatalf("PlusManager.DataDir = %q, want ./data", cfg.PlusManager.DataDir)
	}
	if cfg.PlusManager.DBPath != "./data/usage.sqlite" {
		t.Fatalf("PlusManager.DBPath = %q, want ./data/usage.sqlite", cfg.PlusManager.DBPath)
	}
	if !cfg.PlusManager.CollectorEnabled {
		t.Fatalf("PlusManager.CollectorEnabled = false, want true")
	}
	if cfg.PlusManager.CollectorMode != "auto" {
		t.Fatalf("PlusManager.CollectorMode = %q, want auto", cfg.PlusManager.CollectorMode)
	}
	if cfg.PlusManager.PollIntervalMs != 1000 {
		t.Fatalf("PlusManager.PollIntervalMs = %d, want 1000", cfg.PlusManager.PollIntervalMs)
	}
}

func TestParseConfigBytesPlusManagerOverrides(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`plus-manager:
  enabled: false
  data-dir: /var/lib/cpa-plus
  db-path: /var/lib/cpa-plus/custom.sqlite
  collector-enabled: false
  collector-mode: http
  poll-interval-ms: 2500
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if cfg.PlusManager.Enabled {
		t.Fatalf("PlusManager.Enabled = true, want false")
	}
	if cfg.PlusManager.DataDir != "/var/lib/cpa-plus" {
		t.Fatalf("PlusManager.DataDir = %q", cfg.PlusManager.DataDir)
	}
	if cfg.PlusManager.DBPath != "/var/lib/cpa-plus/custom.sqlite" {
		t.Fatalf("PlusManager.DBPath = %q", cfg.PlusManager.DBPath)
	}
	if cfg.PlusManager.CollectorEnabled {
		t.Fatalf("PlusManager.CollectorEnabled = true, want false")
	}
	if cfg.PlusManager.CollectorMode != "http" {
		t.Fatalf("PlusManager.CollectorMode = %q, want http", cfg.PlusManager.CollectorMode)
	}
	if cfg.PlusManager.PollIntervalMs != 2500 {
		t.Fatalf("PlusManager.PollIntervalMs = %d, want 2500", cfg.PlusManager.PollIntervalMs)
	}
}

func TestParseConfigBytesPlusManagerDBPathFallsBackToDataDir(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`plus-manager:
  data-dir: /tmp/cpa-plus
  db-path: ""
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if cfg.PlusManager.DBPath != "/tmp/cpa-plus/usage.sqlite" {
		t.Fatalf("PlusManager.DBPath = %q, want /tmp/cpa-plus/usage.sqlite", cfg.PlusManager.DBPath)
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/config -run PlusManager -count=1
```

Expected: FAIL because `Config.PlusManager` does not exist.

- [ ] **Step 3: Implement config struct and normalization**

In `internal/config/config.go`, add to `Config`:

```go
	// PlusManager configures the integrated CPA Manager Plus functionality.
	PlusManager PlusManagerConfig `yaml:"plus-manager" json:"plus-manager"`
```

Add near other config structs:

```go
// PlusManagerConfig controls integrated CPA Manager Plus behavior.
type PlusManagerConfig struct {
	Enabled          bool   `yaml:"enabled" json:"enabled"`
	DataDir          string `yaml:"data-dir" json:"data-dir"`
	DBPath           string `yaml:"db-path" json:"db-path"`
	CollectorEnabled bool   `yaml:"collector-enabled" json:"collector-enabled"`
	CollectorMode    string `yaml:"collector-mode" json:"collector-mode"`
	PollIntervalMs   int    `yaml:"poll-interval-ms" json:"poll-interval-ms"`
}

func (cfg *Config) normalizePlusManagerConfig() {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.PlusManager.DataDir) == "" {
		cfg.PlusManager.DataDir = "./data"
	}
	if strings.TrimSpace(cfg.PlusManager.DBPath) == "" {
		cfg.PlusManager.DBPath = strings.TrimRight(cfg.PlusManager.DataDir, "/\\") + "/usage.sqlite"
	}
	if strings.TrimSpace(cfg.PlusManager.CollectorMode) == "" {
		cfg.PlusManager.CollectorMode = "auto"
	}
	if cfg.PlusManager.PollIntervalMs <= 0 {
		cfg.PlusManager.PollIntervalMs = 1000
	}
}
```

Set defaults before YAML unmarshal in both `LoadConfigOptional` and `ParseConfigBytes`:

```go
cfg.PlusManager.Enabled = true
cfg.PlusManager.DataDir = "./data"
cfg.PlusManager.DBPath = "./data/usage.sqlite"
cfg.PlusManager.CollectorEnabled = true
cfg.PlusManager.CollectorMode = "auto"
cfg.PlusManager.PollIntervalMs = 1000
```

Call after unmarshal in both paths:

```go
cfg.normalizePlusManagerConfig()
```

- [ ] **Step 4: Update example config**

Add to `config.example.yaml`:

```yaml
plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
  collector-mode: auto
  poll-interval-ms: 1000
```

- [ ] **Step 5: Run tests and verify GREEN**

Run:

```bash
go test ./internal/config -run PlusManager -count=1
go test ./internal/config -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/parse.go internal/config/plus_manager_test.go config.example.yaml
git commit -m "feat: add integrated Plus manager config"
```

---

### Task 2: Add bundled management panel provider

**Files:**
- Modify: `internal/managementasset/updater.go`
- Create: `internal/managementasset/bundled/management.html`
- Create: `internal/managementasset/bundled.go`
- Create: `internal/managementasset/bundled_test.go`
- Modify: `internal/api/server.go`

- [ ] **Step 1: Write failing bundled asset tests**

Create `internal/managementasset/bundled_test.go`:

```go
package managementasset

import (
	"strings"
	"testing"
)

func TestBundledManagementHTMLAvailable(t *testing.T) {
	html := BundledManagementHTML()
	if !strings.Contains(html, "CPA Manager Plus") {
		t.Fatalf("BundledManagementHTML() missing CPA Manager Plus marker")
	}
}

func TestHasBundledManagementHTML(t *testing.T) {
	if !HasBundledManagementHTML() {
		t.Fatalf("HasBundledManagementHTML() = false, want true")
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/managementasset -run Bundled -count=1
```

Expected: FAIL because bundled helpers do not exist.

- [ ] **Step 3: Add embedded placeholder panel**

Create `internal/managementasset/bundled/management.html`:

```html
<!doctype html><html><head><meta charset="utf-8"><title>CPA Manager Plus</title></head><body><div id="root">CPA Manager Plus integrated panel placeholder</div></body></html>
```

Create `internal/managementasset/bundled.go`:

```go
package managementasset

import (
	_ "embed"
	"strings"
)

//go:embed bundled/management.html
var bundledManagementHTML string

func BundledManagementHTML() string {
	return bundledManagementHTML
}

func HasBundledManagementHTML() bool {
	return strings.TrimSpace(bundledManagementHTML) != ""
}
```

- [ ] **Step 4: Serve bundled panel before remote/local asset**

In `internal/api/server.go`, update `serveManagementControlPanel` before resolving `managementasset.FilePath`:

```go
	if managementasset.HasBundledManagementHTML() {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, managementasset.BundledManagementHTML())
		return
	}
```

Keep existing file/updater fallback after this block.

- [ ] **Step 5: Run focused tests**

Run:

```bash
go test ./internal/managementasset -run Bundled -count=1
go test ./internal/api -run Management -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/managementasset/bundled.go internal/managementasset/bundled/management.html internal/managementasset/bundled_test.go internal/api/server.go
git commit -m "feat: serve bundled Plus management panel"
```

---

### Task 3: Add Plus route registration seam

**Files:**
- Create: `internal/plusmanager/httpapi/router.go`
- Create: `internal/plusmanager/httpapi/router_test.go`
- Modify: `internal/api/server.go`

- [ ] **Step 1: Write failing route tests**

Create `internal/plusmanager/httpapi/router_test.go`:

```go
package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: true})

	req := httptest.NewRequest(http.MethodGet, "/v0/management/plus/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterRoutesDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/v0/management/plus"), Options{Enabled: false})

	req := httptest.NewRequest(http.MethodGet, "/v0/management/plus/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want 404", w.Code)
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/plusmanager/httpapi -run RegisterRoutes -count=1
```

Expected: FAIL because package/functions do not exist.

- [ ] **Step 3: Implement route seam**

Create `internal/plusmanager/httpapi/router.go`:

```go
package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Options struct {
	Enabled bool
}

func RegisterRoutes(group *gin.RouterGroup, opts Options) {
	if group == nil || !opts.Enabled {
		return
	}
	group.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"mode":   "integrated",
		})
	})
	group.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"integrated": true,
			"setupRequired": false,
		})
	})
}
```

- [ ] **Step 4: Wire route seam into CLIProxyAPI server**

In `internal/api/server.go`, import:

```go
plushttpapi "github.com/router-for-me/CLIProxyAPI/v7/internal/plusmanager/httpapi"
```

Inside `registerManagementRoutes`, after `mgmt := s.engine.Group("/v0/management")` and existing auth middleware is applied, add:

```go
	plushttpapi.RegisterRoutes(mgmt.Group("/plus"), plushttpapi.Options{Enabled: s.cfg != nil && s.cfg.PlusManager.Enabled})
```

Use the exact local structure of `registerManagementRoutes`; do not duplicate auth middleware.

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/plusmanager/httpapi -count=1
go test ./internal/api -run Management -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/plusmanager/httpapi/router.go internal/plusmanager/httpapi/router_test.go internal/api/server.go
git commit -m "feat: add integrated Plus management routes"
```

---

### Task 4: Vendor Manager Plus frontend as build input

**Files:**
- Create/modify: `web/manager-plus/**`
- Modify: `Dockerfile`
- Create: `web/manager-plus/README.md`

- [ ] **Step 1: Copy frontend source**

Copy `CPA-Manager-Plus/apps/web` into CLIProxyAPI as `web/manager-plus`, preserving source, package files, and Vite config. Do not copy `node_modules` or build output.

- [ ] **Step 2: Add integrated-mode build marker**

Add `web/manager-plus/README.md`:

```markdown
# Integrated CPA Manager Plus frontend

This directory vendors the CPA Manager Plus frontend for the CPA-PLUS integrated fork.
Build output is embedded into CLIProxyAPI as `internal/managementasset/bundled/management.html`.
Integrated mode uses same-origin APIs:

- CPA Management API: `/v0/management/*`
- Plus APIs: `/v0/management/plus/*`
```

- [ ] **Step 3: Update Dockerfile frontend stage**

Add a Node build stage before Go build:

```dockerfile
FROM --platform=$BUILDPLATFORM node:22-alpine AS web-build
WORKDIR /web
COPY web/manager-plus/package*.json ./
RUN npm ci
COPY web/manager-plus ./
RUN npm run build
```

In the Go build stage, copy the built panel into the embedded path before `go build`:

```dockerfile
COPY --from=web-build /web/dist/index.html ./internal/managementasset/bundled/management.html
```

Keep existing Go build behavior and runtime image behavior.

- [ ] **Step 4: Verify Dockerfile syntax and frontend install path**

Run:

```bash
test -f web/manager-plus/package.json
test -f web/manager-plus/vite.config.ts
grep -q "web-build" Dockerfile
grep -q "internal/managementasset/bundled/management.html" Dockerfile
```

Expected: all commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add web/manager-plus Dockerfile
git commit -m "build: vendor Plus frontend into Docker build"
```

---

### Task 5: Port minimal Plus store and model-price APIs

**Files:**
- Create: `internal/plusmanager/store/store.go`
- Create: `internal/plusmanager/store/store_test.go`
- Create: `internal/plusmanager/model/model_price.go`
- Modify: `internal/plusmanager/httpapi/router.go`
- Modify: `internal/plusmanager/httpapi/router_test.go`
- Modify: `go.mod`, `go.sum` if SQLite dependency is required

- [ ] **Step 1: Write failing store tests**

Create `internal/plusmanager/store/store_test.go`:

```go
package store

import (
	"path/filepath"
	"testing"
)

func TestModelPricesPersist(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "usage.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	prices := []ModelPrice{{Model: "gpt-test", InputPerMTok: 1.25, OutputPerMTok: 5.5}}
	if err := s.ReplaceModelPrices(prices); err != nil {
		t.Fatalf("ReplaceModelPrices() error = %v", err)
	}
	got, err := s.ListModelPrices()
	if err != nil {
		t.Fatalf("ListModelPrices() error = %v", err)
	}
	if len(got) != 1 || got[0].Model != "gpt-test" || got[0].InputPerMTok != 1.25 || got[0].OutputPerMTok != 5.5 {
		t.Fatalf("ListModelPrices() = %#v", got)
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/plusmanager/store -run ModelPricesPersist -count=1
```

Expected: FAIL because store does not exist.

- [ ] **Step 3: Implement minimal SQLite-backed store**

Port the smallest working subset from CPA Manager Plus model-price repository. If adding a SQLite driver is necessary, prefer the driver already used by CPA Manager Plus. Expose this API:

```go
package store

type ModelPrice struct {
	Model         string  `json:"model"`
	InputPerMTok  float64 `json:"inputPerMTok"`
	OutputPerMTok float64 `json:"outputPerMTok"`
}

type Store struct { /* private db fields */ }

func Open(dbPath string) (*Store, error)
func (s *Store) Close() error
func (s *Store) ListModelPrices() ([]ModelPrice, error)
func (s *Store) ReplaceModelPrices([]ModelPrice) error
```

Create a `model_prices` table with `model` as primary key and numeric input/output price columns.

- [ ] **Step 4: Add route-level model-price tests**

Extend `internal/plusmanager/httpapi/router_test.go` with a test that registers a temp store and verifies `GET /model-prices` returns JSON array and `PUT /model-prices` persists a replacement.

- [ ] **Step 5: Implement route handlers**

Extend `Options`:

```go
type Options struct {
	Enabled bool
	Store   ModelPriceStore
}

type ModelPriceStore interface {
	ListModelPrices() ([]store.ModelPrice, error)
	ReplaceModelPrices([]store.ModelPrice) error
}
```

Register:

```text
GET /model-prices
PUT /model-prices
```

Return `503` when store is nil, `500` on store errors, `400` on invalid JSON.

- [ ] **Step 6: Run tests**

Run:

```bash
go test ./internal/plusmanager/store -count=1
go test ./internal/plusmanager/httpapi -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/plusmanager/store internal/plusmanager/model internal/plusmanager/httpapi go.mod go.sum
git commit -m "feat: add Plus model price store and API"
```

---

### Task 6: Add Docker and compose docs for single-port deployment

**Files:**
- Modify: `docker-compose.yml`
- Modify: `README_CN.md`
- Modify: `README.md`
- Create: `docs/cpa-plus-integrated.md`

- [ ] **Step 1: Update compose volume example**

In `docker-compose.yml`, ensure the service has data persistence:

```yaml
    volumes:
      - ${CLI_PROXY_CONFIG_PATH:-./config.yaml}:/CLIProxyAPI/config.yaml
      - ${CLI_PROXY_AUTH_PATH:-./auths}:/root/.cli-proxy-api
      - ${CLI_PROXY_LOG_PATH:-./logs}:/CLIProxyAPI/logs
      - ${CLI_PROXY_PLUS_DATA_PATH:-./data}:/CLIProxyAPI/data
```

- [ ] **Step 2: Add integrated deployment doc**

Create `docs/cpa-plus-integrated.md` with:

```markdown
# CPA-PLUS Integrated Deployment

CPA-PLUS is a CLIProxyAPI fork with CPA Manager Plus integrated into the same process.

## Access

- API base: `http://host:8317`
- Management panel: `http://host:8317/management.html`

## Required management key

Configure one key:

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
```

- [ ] **Step 3: Link docs from READMEs**

Add a short CPA-PLUS integrated note near Docker deployment sections in `README.md` and `README_CN.md`, linking `docs/cpa-plus-integrated.md`.

- [ ] **Step 4: Verify docs references**

Run:

```bash
grep -q "CPA-PLUS Integrated Deployment" docs/cpa-plus-integrated.md
grep -q "CLI_PROXY_PLUS_DATA_PATH" docker-compose.yml
grep -q "cpa-plus-integrated" README.md
grep -q "cpa-plus-integrated" README_CN.md
```

Expected: all commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml README.md README_CN.md docs/cpa-plus-integrated.md
git commit -m "docs: document single-port CPA Plus deployment"
```

---

## Integration Checkpoint

After Tasks 1-6 are integrated, run:

```bash
go test ./internal/config ./internal/managementasset ./internal/plusmanager/... ./internal/api -count=1
go test ./cmd/server ./sdk/cliproxy -count=1
```

Then build:

```bash
go build ./cmd/server
```

Expected: all tests pass and build succeeds.

## Later Phase Tasks

The first parallel implementation intentionally stops at a working seam plus a minimal persisted Plus API. Follow-up plans should port these larger pieces from CPA Manager Plus after the integration seam is green:

- usage event schema, import/export, and aggregation APIs
- HTTP/RESP collector worker lifecycle
- API key aliases
- dashboard summary APIs
- server-side Codex inspection
- full frontend integrated-mode route remapping and feature restoration
