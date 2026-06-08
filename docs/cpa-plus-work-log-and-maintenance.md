# CPA-PLUS 整合工作日志、功能说明与维护文档

本文档记录本地 CPA-PLUS 整合版的改造内容、关键文件、功能入口、部署使用和高频上游同步维护方式。

## 1. 项目目标

把以下两个项目合并为一个本地维护项目：

- CLIProxyAPI：`https://github.com/router-for-me/CLIProxyAPI`
- CPA-Manager-Plus：`https://github.com/seakee/CPA-Manager-Plus`

最终形态：

- 一个 Git 工作区：当前仓库 `/root/code/cpa-plus-merge-study/CLIProxyAPI`
- 一个 Docker 镜像/容器
- 一个端口：`8317`
- 一个入口：`http://host:8317/management.html`
- 一个管理密钥：`remote-management.secret-key`
- Plus 数据持久化：`./data/usage.sqlite`

## 2. 当前本地提交记录

主要整合提交：

```text
0f4afa83 chore: add local upstream sync tooling
64c6de43 docs: update CPA-PLUS integration status
6591c0ef feat: complete integrated Plus monitoring flow
4c0668b7 feat: add integrated Plus usage collector
ea9161ef feat: add Plus usage analytics APIs
64f21b4d feat: add Plus API key aliases
40e31f68 fix: close integrated Plus model price flow
d2b8ee21 feat: add Plus model price store and API
e61a20c3 build: vendor Plus frontend into Docker build
0298e65b feat: add integrated Plus management routes
```

当前远程布局已改为只拉取上游，禁止 push：

```text
cli-upstream   https://github.com/router-for-me/CLIProxyAPI.git  fetch
cli-upstream   DISABLED                                          push
plus-upstream  https://github.com/seakee/CPA-Manager-Plus.git    fetch
plus-upstream  DISABLED                                          push
```

## 3. 主要修改位置

### 3.1 配置

- `internal/config/config.go`
- `internal/config/parse.go`
- `internal/config/plus_manager_test.go`
- `config.example.yaml`

新增配置块：

```yaml
plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
  collector-mode: auto
  poll-interval-ms: 1000
```

作用：控制集成 Plus Manager 的启用、SQLite 数据库位置、collector 是否启动和轮询间隔。

### 3.2 Docker 与前端嵌入

- `Dockerfile`
- `internal/managementasset/bundled.go`
- `internal/managementasset/bundled/management.html`
- `web/manager-plus/**`

Dockerfile 现在会：

1. 用 Node 构建 `web/manager-plus`。
2. 把构建结果复制成 `internal/managementasset/bundled/management.html`。
3. 用 Go 构建 CLIProxyAPI。
4. 最终镜像只暴露 `8317`。

### 3.3 服务路由与面板入口

- `internal/api/server.go`
- `internal/api/server_test.go`
- `internal/plusmanager/httpapi/router.go`
- `internal/plusmanager/httpapi/router_test.go`

关键入口：

```text
GET /management.html
GET /usage-service/info
GET /usage-service/config
PUT /usage-service/config
GET /v0/management/plus/status
GET /v0/management/plus/info
```

Plus 专用 API 统一挂载在：

```text
/v0/management/plus/*
```

### 3.4 Plus SQLite Store

- `internal/plusmanager/store/store.go`
- `internal/plusmanager/store/store_test.go`
- `internal/plusmanager/model/model_price.go`
- `internal/plusmanager/model/api_key_alias.go`
- `internal/plusmanager/model/usage.go`

SQLite 表包括：

- `model_prices`
- `api_key_aliases`
- `usage_events`

存储内容包括：

- 模型价格
- API key 别名
- 请求事件
- token 用量
- 请求成功/失败
- 延迟
- auth/account 快照字段

### 3.5 请求监控与 analytics

- `internal/plusmanager/httpapi/router.go`
- `internal/plusmanager/store/store.go`
- `internal/plusmanager/model/usage.go`

已实现：

```text
GET  /v0/management/plus/usage
GET  /v0/management/plus/usage/export
POST /v0/management/plus/usage/import
GET  /v0/management/plus/dashboard/summary
POST /v0/management/plus/monitoring/analytics
```

其中 `/usage` 返回前端可消费的结构：

```text
apis -> models -> details
```

### 3.6 API key aliases

- `internal/plusmanager/model/api_key_alias.go`
- `internal/plusmanager/store/store.go`
- `internal/plusmanager/httpapi/router.go`

接口：

```text
GET    /v0/management/plus/api-key-aliases
PUT    /v0/management/plus/api-key-aliases
DELETE /v0/management/plus/api-key-aliases/:apiKeyHash
```

### 3.7 模型价格与同步

- `internal/plusmanager/model/model_price.go`
- `internal/plusmanager/store/store.go`
- `internal/plusmanager/httpapi/router.go`

接口：

```text
GET  /v0/management/plus/model-prices
PUT  /v0/management/plus/model-prices
POST /v0/management/plus/model-prices/sync
```

说明：

- 支持前端的 `prompt/completion/cache/cacheRead/cacheCreation/source/sourceModelId` 字段。
- `PUT` 返回 `{ prices }`，与 Plus 前端契约一致。
- `sync` 从 LiteLLM 模型价格源同步。

### 3.8 Collector 生命周期

- `internal/plusmanager/collector/collector.go`
- `internal/plusmanager/collector/collector_test.go`
- `internal/plusmanager/collector/redisqueue.go`
- `internal/api/server.go`

接口：

```text
GET /v0/management/plus/collector/status
```

行为：

- 从 CLIProxyAPI 内部 usage queue 轮询请求事件。
- 写入 Plus SQLite store。
- 随 server 启停。
- 支持 Stop 后再次 Start，不复用已关闭 channel。
- collector 状态里的时间字段使用毫秒时间戳，适配前端。

### 3.9 Plus 前端适配

- `web/manager-plus/src/services/api/usageService.ts`

将 standalone Manager Server 路径迁移为 integrated 路径：

```text
/v0/management/plus/status
/v0/management/plus/usage
/v0/management/plus/model-prices
/v0/management/plus/model-prices/sync
/v0/management/plus/api-key-aliases
/v0/management/plus/usage/export
/v0/management/plus/usage/import
/v0/management/plus/dashboard/summary
/v0/management/plus/monitoring/analytics
```

保留兼容探测：

```text
/usage-service/info
/usage-service/config
```

## 4. 功能介绍

### 4.1 单入口管理面板

访问：

```text
http://host:8317/management.html
```

登录密钥：

```yaml
remote-management.secret-key
```

### 4.2 请求监控

Plus 面板可读取集成后端的 usage 数据，用于展示请求数、成功/失败、token 用量、模型分布和请求明细。

数据来源：

1. CLIProxyAPI 运行时产生 usage event。
2. integrated collector 从内部队列消费。
3. SQLite 存入 `./data/usage.sqlite`。
4. Plus 面板通过 `/v0/management/plus/*` 读取。

### 4.3 usage 导入/导出

接口：

```text
POST /v0/management/plus/usage/import
GET  /v0/management/plus/usage/export
```

用途：

- 迁移历史 usage 事件。
- 备份/恢复请求统计数据。
- 调试和离线分析。

### 4.4 API key aliases

接口：

```text
GET/PUT/DELETE /v0/management/plus/api-key-aliases
```

用途：给 API key hash 配置可读别名，改善面板可读性。

### 4.5 dashboard analytics

接口：

```text
GET  /v0/management/plus/dashboard/summary
POST /v0/management/plus/monitoring/analytics
```

用途：为 Plus dashboard 和 monitoring 页面提供摘要、事件分页、近期失败等数据。

### 4.6 模型价格

接口：

```text
GET/PUT /v0/management/plus/model-prices
POST    /v0/management/plus/model-prices/sync
```

用途：

- 手动维护模型价格。
- 从 LiteLLM 同步价格。
- 为费用估算提供价格表。

## 5. 部署使用

### 5.1 准备配置

```bash
cd /root/code/cpa-plus-merge-study/CLIProxyAPI
cp config.example.yaml config.yaml
```

修改：

```yaml
port: 8317

remote-management:
  allow-remote: true
  secret-key: "你的管理密钥"

plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
```

### 5.2 启动

```bash
docker compose up -d --build
```

### 5.3 访问

```text
http://服务器IP:8317/management.html
```

### 5.4 日常命令

查看状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f
```

重启：

```bash
docker compose restart
```

停止：

```bash
docker compose down
```

重新构建：

```bash
docker compose up -d --build
```

## 6. 数据与备份

重要路径：

```text
config.yaml        主配置，包含管理密钥，不要提交
.env               环境变量，不要提交
auths/             账号认证数据，不要提交
data/usage.sqlite  Plus 数据库，不要提交
logs/              日志，不要提交
```

备份：

```bash
tar -czf cpa-plus-backup-$(date +%F).tar.gz config.yaml auths data logs
```

恢复：

```bash
tar -xzf cpa-plus-backup-YYYY-MM-DD.tar.gz
```

## 7. 本地高频上游同步

详细文档：

```text
docs/local-upstream-sync.md
```

### 7.1 一键同步两个上游

```bash
./scripts/local-update-all.sh
```

脚本会：

1. 检查工作区是否干净。
2. fetch/merge CLIProxyAPI 上游。
3. clone/sync CPA-Manager-Plus `apps/web`。
4. 自动重新打 integrated API 路径补丁。
5. 刷新 package lock。
6. 运行 Go/前端测试和构建。
7. 清理 `node_modules`、`dist`。

### 7.2 只更新 CLIProxyAPI

```bash
./scripts/local-update-cli.sh
```

### 7.3 只更新 Plus 前端

```bash
./scripts/local-update-plus-web.sh
```

如果本地已有 Plus checkout，推荐用：

```bash
./scripts/local-update-plus-web.sh --source /root/code/cpa-plus-merge-study/CPA-Manager-Plus
```

### 7.4 Dry-run

```bash
ALLOW_DIRTY=1 ./scripts/local-update-all.sh --dry-run --skip-tests
```

## 8. 隐私与安全注意

不要执行：

```bash
git add .
```

推荐只添加明确文件：

```bash
git add internal web scripts docs README.md README_CN.md Dockerfile config.example.yaml docker-compose.yml
```

提交前检查：

```bash
git status --short
```

不应出现在待提交列表：

```text
config.yaml
.env
auths/
data/
logs/
*.sqlite
```

当前上游远程 push 已禁用：

```text
cli-upstream   DISABLED push
plus-upstream  DISABLED push
```

## 9. 已验证命令

整合完成时已验证：

```bash
go test ./internal/config ./internal/managementasset ./internal/plusmanager/... ./internal/api -count=1
go build ./cmd/server
npm --prefix web/manager-plus ci
npm --prefix web/manager-plus test
npm --prefix web/manager-plus run build
docker build -t cpa-plus-integrated:test .
```

运行态验证过：

```text
GET  /management.html
GET  /v0/management/plus/status
GET  /usage-service/config
PUT  /v0/management/plus/model-prices
GET  /v0/management/plus/model-prices
PUT  /v0/management/plus/api-key-aliases
GET  /v0/management/plus/api-key-aliases
POST /v0/management/plus/usage/import
GET  /v0/management/plus/usage
GET  /v0/management/plus/usage/export
GET  /v0/management/plus/dashboard/summary
POST /v0/management/plus/monitoring/analytics
GET  /v0/management/plus/collector/status
```

本地同步工具已验证：

```bash
./test/local_update_scripts_test.sh
ALLOW_DIRTY=1 ./scripts/local-update-all.sh --dry-run --skip-tests
```

## 10. 后续维护建议

- 高频更新时先运行 `./scripts/local-update-all.sh --dry-run --skip-tests` 看流程。
- 正式同步前确保 `git status --short` 干净。
- 同步后先本地构建，再部署。
- Plus 后端新功能不会自动完整迁移，需要按功能从 `CPA-Manager-Plus/apps/manager-server` 手工移植到 `internal/plusmanager`。
- 如果上游 Plus 前端大改，优先检查 `web/manager-plus/src/services/api/usageService.ts` 的 integrated 路径是否仍正确。
