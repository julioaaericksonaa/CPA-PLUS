# CPA-PLUS

CPA-PLUS = CLIProxyAPI + CPA-Manager-Plus，一份 Linux 二进制、一个端口、一个 Plus 面板。

```text
http://127.0.0.1:8317/management.html
```

本仓库只维护 `main` 分支；GitHub Actions 每天北京时间 21:00 自动同步两个上游并刷新 `latest` Release。

---

## 一句话安装

私有仓库推荐先登录 GitHub CLI，然后一条命令安装并启动（默认只监听本机 127.0.0.1）：

```bash
gh auth login && gh api "repos/julioaaericksonaa/CPA-PLUS/contents/scripts/install-release.sh?ref=main" --jq .content | base64 -d | bash
```

如果你明确要公网直连 `http://host:8317/management.html`，加 `--public`：

```bash
gh auth login && gh api "repos/julioaaericksonaa/CPA-PLUS/contents/scripts/install-release.sh?ref=main" --jq .content | base64 -d | bash -s -- --public
```

如果仓库改为公开，也可以：

```bash
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | bash
```

脚本会自动完成：下载 `latest` Release 二进制、校验 sha256、生成完整默认配置、安装启动脚本、安装 `update-cpa`、启动服务。

安装完成后只需要记住：

```text
面板：http://127.0.0.1:8317/management.html（公网模式为 http://host:8317/management.html）
配置：/root/apps/cliproxyapi-plus/config.yaml
密钥：/root/apps/cliproxyapi-plus/secrets.env
更新：update-cpa
```

---

## 一句话更新

```bash
update-cpa
```

`update-cpa` 默认从 `latest` tag 拉取安装脚本、下载最新已发布的 `latest` Release，保留你的 `config.yaml`、`data/`、`logs/`，只替换二进制并重启服务。

> 以后主要看 Release，不需要自己本机构建：Actions 每天 21:00 自动拉取上游、合并、构建、发布。

---

## 常用命令

```bash
/root/apps/cliproxyapi-plus/start-detached.sh   # 启动
/root/apps/cliproxyapi-plus/stop.sh             # 停止
/root/apps/cliproxyapi-plus/restart.sh          # 重启
/root/apps/cliproxyapi-plus/status.sh           # 状态
update-cpa                                      # 更新到最新 Release
```

查看日志：

```bash
tail -f /root/apps/cliproxyapi-plus/logs/cliproxyapi-plus.nohup.log
```

查看版本响应头：

```bash
curl -sS -D - -o /dev/null http://127.0.0.1:8317/v0/management/config | grep -Ei 'X-CPA|X-PLUS'
```

系统概览里的版本含义：

- 管理面板版本：构建时打包的 CPA-Manager-Plus 上游版本/commit。
- 服务端版本：构建时打包的 CLIProxyAPI 上游版本。
- 点刷新检查更新时，面板只请求本机管理接口；服务端再去查询上游，避免浏览器直连 GitHub API 触发 403。

---

## 默认配置说明

一键脚本首次安装会生成较完整的默认配置：

```text
/root/apps/cliproxyapi-plus/config.yaml
/root/apps/cliproxyapi-plus/config.example.yaml
/root/apps/cliproxyapi-plus/secrets.env
```

默认配置包含：

- `host: "127.0.0.1"` 和 `remote-management.allow-remote: false`：默认只允许本机访问；公网直连请用 `--public` 或手动改配置。
- `remote-management`：Plus 面板管理密钥，默认自动生成强密钥。
- `api-keys`：客户端调用本代理用的 API key，默认自动生成。
- `plus-manager`：统计、巡检、历史记录默认开启。
- `routing`、`quota-exceeded`、`retry`、`logs`、`plugins` 等常用项。
- OpenAI、Gemini、Claude、Codex Provider 示例，默认注释，不含真实 key。

修改配置：

```bash
nano /root/apps/cliproxyapi-plus/config.yaml
/root/apps/cliproxyapi-plus/restart.sh
```

重新生成配置会备份旧配置：

```bash
gh api "repos/julioaaericksonaa/CPA-PLUS/contents/scripts/install-release.sh?ref=main" --jq .content | base64 -d | bash -s -- --force-config
```

---

## 安装参数

```bash
# 指定端口
gh api "repos/julioaaericksonaa/CPA-PLUS/contents/scripts/install-release.sh?ref=main" --jq .content | base64 -d | CPA_PLUS_PORT=8318 bash

# 指定目录
gh api "repos/julioaaericksonaa/CPA-PLUS/contents/scripts/install-release.sh?ref=main" --jq .content | base64 -d | CPA_PLUS_APP_DIR=/root/apps/cpa-test bash

# 公网直连模式
gh api "repos/julioaaericksonaa/CPA-PLUS/contents/scripts/install-release.sh?ref=main" --jq .content | base64 -d | bash -s -- --public

# 只安装不启动
gh api "repos/julioaaericksonaa/CPA-PLUS/contents/scripts/install-release.sh?ref=main" --jq .content | base64 -d | bash -s -- --no-start
```

脚本参数也可以这样写：

```bash
bash scripts/install-release.sh --port 8318 --app-dir /root/apps/cpa-test --public
```

---

## 自动 Release

工作流：

```text
.github/workflows/auto-sync-release.yml
```

运行时间：

```text
每天北京时间 21:00
cron: 0 13 * * *
```

自动流程：

1. 先轻量检测两个上游 `main` 的最新 commit。
2. 如果 CLIProxyAPI 和 CPA-Manager-Plus 都没有更新，直接停止，不安装环境、不构建、不发布。
3. 如果任意一个上游有更新，再拉取上游、应用 CPA-PLUS 集成 patch。
4. 在 Ubuntu 22.04 构建 Linux amd64 二进制，降低 glibc 兼容性要求。
5. 提交更新后的上游元数据和 patch 到 `main`。
6. 只刷新固定 Release：`latest`。
7. 删除所有旧 Release 和非 `latest` tag，仓库只保留最新版 `latest` Release。

说明：GitHub Release 顶部的 “released ... ago” 是 `latest` 这个 Release 首次创建时间；CPA-PLUS 会在 Release notes 中写入 `Updated at`，以它作为最新二进制发布时间。

日常使用只关心：

```text
https://github.com/julioaaericksonaa/CPA-PLUS/releases/tag/latest
```

Release 资产：

```text
CLIProxyAPI-linux-amd64
CLIProxyAPI-linux-amd64.sha256
```

---

## 本地源码构建（高级维护）

普通使用不要走这里，直接用 Release。

```bash
git clone https://github.com/julioaaericksonaa/CPA-PLUS.git
cd CPA-PLUS
./scripts/build-linux-binary.sh --skip-tests
```

安装本地构建结果：

```bash
./scripts/install-linux.sh --skip-tests
```

---

## 隐私提醒

不要提交这些本机文件：

```text
config.yaml
secrets.env
auths/*
data/*
logs/*
.env*
*.key
*.pem
.codex/
.claude/
.gemini/
```

仓库里的配置模板只放占位符，不放真实 key。


## 更新链路信任说明

`update-cpa` 会执行本仓库 `latest` tag 中的安装脚本，并校验 Release 二进制的 sha256。sha256 用来防下载损坏；仓库和 Release 本身仍是信任根，请只在你控制的私有仓库中使用。
