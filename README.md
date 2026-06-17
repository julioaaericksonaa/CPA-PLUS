# CPA-PLUS

CPA-PLUS = CLIProxyAPI + CPA-Manager-Plus，一份 Linux 二进制、一个端口、一个 Plus 面板。

```text
http://127.0.0.1:8317/management.html
```

本仓库只维护 `main` 分支；GitHub Actions 每天北京时间 21:00 自动检测两个上游，任意上游有更新时自动合并、构建并刷新固定 `latest` Release。

---

## 公开安装

默认只监听本机 `127.0.0.1`，适合配合 SSH 隧道、反向代理或本机浏览器使用：

```bash
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | bash
```

如果你明确要公网直连 `http://host:8317/management.html`，安装时加 `--public`：

```bash
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | bash -s -- --public
```

脚本会自动完成：下载 `latest` Release 二进制、校验 sha256、生成本地默认配置、安装启动脚本、安装 `update-cpa`、启动服务。

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

`update-cpa` 默认从 `main` 拉取最新安装脚本、下载最新已发布的 `latest` Release，保留你的 `config.yaml`、`data/`、`logs/`，只替换二进制并重启服务。

> 日常使用主要看 Release，不需要自己本机构建：Actions 每天 21:00 自动检测上游、合并、构建、发布。

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

查看健康状态：

```bash
curl -sS http://127.0.0.1:8317/healthz
```

系统概览里的版本含义：

- 管理面板版本：构建时打包的 CPA-Manager-Plus 上游版本/commit。
- 服务端版本：构建时打包的 CLIProxyAPI 上游版本。
- 点刷新检查更新时，面板只请求本机管理接口；服务端再去查询上游，避免浏览器直连 GitHub API 触发 403。
- 如果检测到上游已更新，系统概览旁会出现“更新”按钮；按钮会先确认 `latest` Release 是否已经刷新，未刷新时提示等待每日 21:00 合并版发布。

---

## 本地配置与隐私

一键脚本首次安装会在本机生成这些文件：

```text
/root/apps/cliproxyapi-plus/config.yaml
/root/apps/cliproxyapi-plus/config.example.yaml
/root/apps/cliproxyapi-plus/secrets.env
```

仓库同时提供脱敏示例，便于了解配置结构：

```text
examples/config.yaml
examples/secrets.env
```

说明：

- `examples/config.yaml` 和 `examples/secrets.env` 只包含占位符，可作为参考模板。
- 安装脚本会在部署目录生成实际运行文件；仓库默认只跟踪脱敏示例，不跟踪本地运行配置和数据。
- 如需公开分享配置片段，请使用 `CHANGE_ME`、`example`、`redacted` 等占位符替换真实凭据。

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
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | bash -s -- --force-config
```

---

## 安装参数

```bash
# 指定端口
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | CPA_PLUS_PORT=8318 bash

# 指定目录
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | CPA_PLUS_APP_DIR=/root/apps/cpa-test bash

# 公网直连模式
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | bash -s -- --public

# 只安装不启动
curl -fsSL https://raw.githubusercontent.com/julioaaericksonaa/CPA-PLUS/main/scripts/install-release.sh | bash -s -- --no-start
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

该 Action 只按上面的定时每天运行一次；不会在 `push` 或手动触发时运行，避免 GitHub Actions 被频繁触发。

自动流程：

1. 先轻量检测两个上游 `main` 的最新 commit。
2. 如果 CLIProxyAPI 和 CPA-Manager-Plus 都没有更新，直接停止，不安装环境、不构建、不发布。
3. 如果任意一个上游有更新，再拉取上游、应用 CPA-PLUS 集成 patch。
4. 在 Ubuntu 22.04 构建 Linux amd64 二进制，降低 glibc 兼容性要求。
5. 提交更新后的上游元数据和 patch 到 `main`。
6. 删除并重建固定 Release：`latest`，让 GitHub Release 列表时间也刷新。
7. 删除所有旧 Release 和非 `latest` tag，仓库只保留最新版 `latest` Release。

说明：重建 `latest` Release 会让 GitHub 顶部和列表里的发布时间同步刷新；发布过程中可能存在几秒钟短暂空窗，如果安装时刚好遇到 404，稍等后重试即可。

日常使用只关心：

```text
https://github.com/julioaaericksonaa/CPA-PLUS/releases/tag/latest
```

Release 资产：

```text
CLIProxyAPI-linux-amd64
CLIProxyAPI-linux-amd64.sha256
```

本地维护者如果已经在本机重新拉取、构建并提交了新版二进制对应的代码，可以用本地发布脚本直接推送代码并重建固定 `latest` Release：

```bash
./scripts/publish-latest-release.sh
```

脚本会读取 `/root/.config/cpa-plus/publish.env` 中的 GitHub 发布凭据，禁用缓存凭据，推送当前 `HEAD` 到 `main`，然后删除并重建 `latest` Release。

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

## 更新链路信任说明

`update-cpa` 会执行本仓库 `main` 中的安装脚本，并校验 Release 二进制的 sha256。sha256 用来防下载损坏；仓库、分支和 Release 本身仍是信任根。公开使用时请固定从你信任的仓库地址安装，并在 fork 或二次分发前检查脚本内容。
