# CPA-PLUS

CPA-PLUS 是一个本地维护的 **CLIProxyAPI + CPA-Manager-Plus 整合版**。

它把 CLI Proxy API 主服务、完整 Plus Web 管理面板、请求统计持久化与 Plus 扩展 API 合并到同一个项目中，目标是减少部署和维护成本：

- 一个项目
- 一个 Docker 镜像/容器
- 一个端口：`8317`
- 一个入口：`http://host:8317/management.html`
- 一个管理密钥：`remote-management.secret-key`

> 本仓库适合私有自用部署。请不要提交真实 `config.yaml`、`.env`、`auths/`、`data/`、`logs/`、数据库、Token、Cookie 或 Codex/Claude/Gemini 等本地配置。

## 项目来源

CPA-PLUS 基于以下两个项目整合维护：

- CLIProxyAPI：<https://github.com/router-for-me/CLIProxyAPI>
- CPA-Manager-Plus：<https://github.com/seakee/CPA-Manager-Plus>

本仓库保留原项目能力，并新增单端口集成、Plus 前端嵌入、Plus API、SQLite 持久化、usage collector 和高频上游同步脚本。

## 功能介绍

### CLIProxyAPI 主能力

- OpenAI / Gemini / Claude / Codex / Grok 兼容 API 代理
- OAuth 登录与多账号轮询
- OpenAI-compatible 上游配置
- 流式、非流式、多模态、工具调用等能力
- Management API 与本地配置热更新能力

### CPA-PLUS 集成能力

- 内置完整 Plus 管理面板：`/management.html`
- Plus API 统一挂载：`/v0/management/plus/*`
- 共用 CLIProxyAPI 管理密钥：`remote-management.secret-key`
- SQLite 持久化：默认 `./data/usage.sqlite`
- 请求监控、用量统计、失败请求、延迟、模型、账号/渠道维度分析
- usage 导入/导出
- API key alias 管理
- 模型价格管理与 LiteLLM 价格同步
- 内置 usage collector，随主服务启动/停止
- Docker 构建时自动打包 Plus 前端到 Go 服务中

## 快速运行

### 1. 准备配置

```bash
cp config.example.yaml config.yaml
```

至少修改：

```yaml
port: 8317

remote-management:
  allow-remote: true
  secret-key: "请改成你自己的强密码"

plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
  collector-mode: auto
  poll-interval-ms: 1000
```

如果只允许本机访问管理端，可设置：

```yaml
remote-management:
  allow-remote: false
```

### 2. 启动 Docker

```bash
docker compose up -d --build
```

默认挂载：

```text
./config.yaml -> /CLIProxyAPI/config.yaml
./auths       -> /root/.cli-proxy-api
./logs        -> /CLIProxyAPI/logs
./data        -> /CLIProxyAPI/data
```

### 3. 访问面板

```text
http://host:8317/management.html
```

登录时使用 `config.yaml` 中的：

```text
remote-management.secret-key
```

### 4. API 调用

主 API 与原 CLIProxyAPI 保持一致，基础地址为：

```text
http://host:8317
```

Plus 集成接口在：

```text
/v0/management/plus/*
```

## 常用运维命令

启动/重建：

```bash
docker compose up -d --build
```

查看容器：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f cli-proxy-api
```

停止：

```bash
docker compose down
```

重启：

```bash
docker compose restart cli-proxy-api
```

## 数据、备份与恢复

建议重点备份：

```text
config.yaml
.env                         # 如果你实际使用了 .env
auths/
data/usage.sqlite
```

备份示例：

```bash
mkdir -p backups
cp config.yaml backups/config.yaml.$(date +%Y%m%d%H%M%S)
tar -czf backups/auths.$(date +%Y%m%d%H%M%S).tgz auths
tar -czf backups/data.$(date +%Y%m%d%H%M%S).tgz data
```

恢复时停止容器，替换对应文件/目录，再重新启动：

```bash
docker compose down
# restore config.yaml auths/ data/
docker compose up -d --build
```

## 高频上游同步方案

本项目包含本地同步脚本，用于快速吸收两个上游项目更新。

一键同步两个上游：

```bash
./scripts/local-update-all.sh
```

只同步 CLIProxyAPI：

```bash
./scripts/local-update-cli.sh
```

只同步 CPA-Manager-Plus 前端：

```bash
./scripts/local-update-plus-web.sh
```

同步后建议检查并提交：

```bash
git status --short
git diff --stat
git add README.md README_CN.md Dockerfile config.example.yaml docker-compose.yml internal web scripts docs
git commit -m "chore: sync upstream updates"
docker compose up -d --build
```

不要使用 `git add .`，避免误提交隐私文件。

详细流程见：

- `docs/local-upstream-sync.md`
- `docs/cpa-plus-work-log-and-maintenance.md`

## 隐私安全清单

这些内容不要提交到 GitHub：

```text
config.yaml
.env
.env.*.local
auths/                 # 只保留 auths/.gitkeep
data/
logs/
*.sqlite
*.db
*.key
*.pem
.codex/
.claude/
.gemini/
```

提交前建议执行：

```bash
git status --short
git diff --cached --name-only
```

如需发布到私有仓库，建议使用无历史快照方式，而不是直接推送包含旧历史的完整仓库。

## 重要文档

- 整合部署说明：`docs/cpa-plus-integrated.md`
- 工作日志、功能说明与维护文档：`docs/cpa-plus-work-log-and-maintenance.md`
- 本地上游同步流程：`docs/local-upstream-sync.md`
- SDK 使用：`docs/sdk-usage.md`
- SDK 进阶：`docs/sdk-advanced.md`
- SDK 访问控制：`docs/sdk-access.md`
- SDK Watcher：`docs/sdk-watcher.md`

## 本仓库维护原则

- 本地运行优先，私有仓库只保存可公开给自己的源码快照
- 上游同步优先使用脚本，不手工大范围复制
- 每次同步后先测试/构建，再部署
- 推送前先检查敏感文件和 staged 文件
- 保留 `cli-upstream` / `plus-upstream` 为只拉取远程，禁止误推上游

## License

本仓库基于原 CLIProxyAPI 与 CPA-Manager-Plus 改造维护，相关许可请参考仓库中的 `LICENSE` 以及对应上游项目许可。
