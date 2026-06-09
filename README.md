# CPA-PLUS

CPA-PLUS 是一个 **单端口、二进制优先** 的 CLIProxyAPI + CPA-Manager-Plus 集成项目。

它把两个上游项目合并成一个 Linux 可执行文件：

- CLIProxyAPI：`https://github.com/router-for-me/CLIProxyAPI`
- CPA-Manager-Plus：`https://github.com/seakee/CPA-Manager-Plus`

运行后只需要访问一个端口：

```text
http://host:8317/management.html
```

本仓库现在只维护 `main` 分支，`main` 就是二进制维护主线；旧的 Docker 主线和 `linux` 分支已经移除。

---

## 1. 默认约定

| 项目 | 默认值 |
|---|---|
| 维护分支 | `main` |
| 服务端口 | `8317` |
| 运行目录 | `/root/apps/cliproxyapi-plus` |
| 主程序 | `/root/apps/cliproxyapi-plus/cli-proxy-api` |
| 配置文件 | `/root/apps/cliproxyapi-plus/config.yaml` |
| 面板地址 | `http://host:8317/management.html` |
| 数据目录 | `/root/apps/cliproxyapi-plus/data` |
| 日志目录 | `/root/apps/cliproxyapi-plus/logs` |
| 最新版 Release | `https://github.com/julioaaericksonaa/CPA-PLUS/releases/tag/latest` |
| 本地维护 checkout | `/root/.config/superpowers/worktrees/CLIProxyAPI/main` |

---

## 2. 快速运行：直接使用 Release 二进制（推荐）

最快的运行方式应该直接使用本仓库发布的 Release 二进制，不需要 clone 仓库，也不需要在本机编译。

固定最新版下载地址：

```text
https://github.com/julioaaericksonaa/CPA-PLUS/releases/download/latest/CLIProxyAPI-linux-amd64
```

### 2.1 公共仓库或已授权环境

```bash
APP=/root/apps/cliproxyapi-plus
mkdir -p "$APP"/{data,logs}
cd "$APP"

curl -fL -o cli-proxy-api \
  https://github.com/julioaaericksonaa/CPA-PLUS/releases/download/latest/CLIProxyAPI-linux-amd64

curl -fL -o cli-proxy-api.sha256 \
  https://github.com/julioaaericksonaa/CPA-PLUS/releases/download/latest/CLIProxyAPI-linux-amd64.sha256

expected="$(awk '{print $1}' cli-proxy-api.sha256)"
actual="$(sha256sum cli-proxy-api | awk '{print $1}')"
[ "$expected" = "$actual" ] || { echo "sha256 mismatch"; exit 1; }

chmod 755 cli-proxy-api
```

### 2.2 私有仓库推荐方式

如果仓库保持私有，直接 `curl` 可能因为未登录而下载失败。推荐使用 GitHub CLI：

```bash
APP=/root/apps/cliproxyapi-plus
mkdir -p "$APP"/{data,logs}
cd "$APP"

gh auth login
gh release download latest \
  --repo julioaaericksonaa/CPA-PLUS \
  --pattern CLIProxyAPI-linux-amd64 \
  --pattern CLIProxyAPI-linux-amd64.sha256 \
  --dir .

mv -f CLIProxyAPI-linux-amd64 cli-proxy-api

expected="$(awk '{print $1}' CLIProxyAPI-linux-amd64.sha256)"
actual="$(sha256sum cli-proxy-api | awk '{print $1}')"
[ "$expected" = "$actual" ] || { echo "sha256 mismatch"; exit 1; }

chmod 755 cli-proxy-api
```

### 2.3 创建最小配置

```bash
cat > /root/apps/cliproxyapi-plus/config.yaml <<'YAML'
host: ""
port: 8317

remote-management:
  allow-remote: false
  secret-key: "换成你自己的强密码"
  disable-control-panel: false
  disable-auto-update-panel: true

auth-dir: "~/.cli-proxy-api"

plus-manager:
  enabled: true
  data-dir: ./data
  db-path: ./data/usage.sqlite
  collector-enabled: true
  collector-mode: auto
  poll-interval-ms: 1000

# 如需让客户端调用模型，请按 CLIProxyAPI 上游格式补充 api-keys、providers、openai 等配置。
# api-keys:
#   - "your-api-key"
YAML
```

如果你要从服务器外部访问管理接口，请把 `remote-management.allow-remote` 改成 `true`，并确保 `secret-key` 足够强。

### 2.4 启动

前台启动：

```bash
cd /root/apps/cliproxyapi-plus
./cli-proxy-api -config ./config.yaml
```

后台启动：

```bash
cd /root/apps/cliproxyapi-plus
nohup ./cli-proxy-api -config ./config.yaml > logs/cliproxyapi-plus.nohup.log 2>&1 &
echo $! > cliproxyapi-plus.pid
```

访问面板：

```text
http://host:8317/management.html
```

如果浏览器显示旧内容，按 `Ctrl + F5` 强制刷新。

> 说明：这种快速方式只依赖 Release 二进制，不会自动安装 `start-detached.sh`、`restart.sh`、`systemd` 服务和 `update-cpa`。如果是长期维护机器，建议继续执行下一节的完整安装。

---

## 3. 完整安装：安装脚本 + `update-cpa`

如果你希望本机具备启动脚本、重启脚本、systemd 支持和 `update-cpa` 更新命令，再 clone 仓库执行安装脚本：

```bash
git clone https://github.com/julioaaericksonaa/CPA-PLUS.git
cd CPA-PLUS
./scripts/install-linux.sh --skip-tests
```

安装完成后会生成：

```text
/root/apps/cliproxyapi-plus/cli-proxy-api
/root/apps/cliproxyapi-plus/config.yaml
/root/apps/cliproxyapi-plus/start-detached.sh
/root/apps/cliproxyapi-plus/stop.sh
/root/apps/cliproxyapi-plus/restart.sh
/usr/local/bin/update-cpa
```

然后编辑配置：

```bash
nano /root/apps/cliproxyapi-plus/config.yaml
```

至少设置管理密钥：

```yaml
remote-management:
  secret-key: "换成你自己的强密码"
```

启动服务：

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
```

访问面板：

```text
http://host:8317/management.html
```

如果浏览器显示旧内容，按 `Ctrl + F5` 强制刷新。

---

## 4. 常用运行命令

脚本方式：

```bash
/root/apps/cliproxyapi-plus/start-detached.sh   # 后台启动
/root/apps/cliproxyapi-plus/stop.sh             # 停止
/root/apps/cliproxyapi-plus/restart.sh          # 重启
```

查看是否监听 8317：

```bash
ss -ltnp | grep ':8317 '
```

查看启动日志：

```bash
tail -f /root/apps/cliproxyapi-plus/logs/cliproxyapi-plus.nohup.log
```

查看服务健康状态：

```bash
curl -sS http://127.0.0.1:8317/usage-service/info
```

---

## 5. systemd 开机自启

如果希望开机自启，先完成安装，然后执行：

```bash
cd /root/.config/superpowers/worktrees/CLIProxyAPI/main
sudo ./scripts/install-systemd.sh
```

常用命令：

```bash
systemctl status cliproxyapi-plus --no-pager
systemctl start cliproxyapi-plus
systemctl stop cliproxyapi-plus
systemctl restart cliproxyapi-plus
systemctl enable cliproxyapi-plus
systemctl disable cliproxyapi-plus
journalctl -u cliproxyapi-plus -f
```

---

## 6. 日常更新

### 6.1 推荐更新方式：使用已发布 Release

本仓库的 GitHub Actions 会每天北京时间 21:00 自动同步两个上游、合并 CPA-PLUS 集成内容、构建二进制并刷新 `latest` Release。

普通使用时，更新就围绕 `latest` Release 做一件事：

```text
下载 latest 二进制 → 校验 sha256 → 替换本地 cli-proxy-api → 重启服务
```

公共仓库或已授权 `curl` 环境：

```bash
APP=/root/apps/cliproxyapi-plus
cd "$APP"

curl -fL -o cli-proxy-api.new \
  https://github.com/julioaaericksonaa/CPA-PLUS/releases/download/latest/CLIProxyAPI-linux-amd64

curl -fL -o cli-proxy-api.sha256 \
  https://github.com/julioaaericksonaa/CPA-PLUS/releases/download/latest/CLIProxyAPI-linux-amd64.sha256

expected="$(awk '{print $1}' cli-proxy-api.sha256)"
actual="$(sha256sum cli-proxy-api.new | awk '{print $1}')"
[ "$expected" = "$actual" ] || { echo "sha256 mismatch"; rm -f cli-proxy-api.new; exit 1; }

chmod 755 cli-proxy-api.new
mv -f cli-proxy-api.new cli-proxy-api

./restart.sh 2>/dev/null || {
  kill "$(cat cliproxyapi-plus.pid)" 2>/dev/null || true
  nohup ./cli-proxy-api -config ./config.yaml > logs/cliproxyapi-plus.nohup.log 2>&1 &
  echo $! > cliproxyapi-plus.pid
}
```

私有仓库推荐使用 GitHub CLI 下载 Release：

```bash
APP=/root/apps/cliproxyapi-plus
cd "$APP"

gh auth login
rm -rf /tmp/cpa-plus-release
mkdir -p /tmp/cpa-plus-release
gh release download latest \
  --repo julioaaericksonaa/CPA-PLUS \
  --pattern CLIProxyAPI-linux-amd64 \
  --pattern CLIProxyAPI-linux-amd64.sha256 \
  --dir /tmp/cpa-plus-release

cp /tmp/cpa-plus-release/CLIProxyAPI-linux-amd64 "$APP/cli-proxy-api.new"
cp /tmp/cpa-plus-release/CLIProxyAPI-linux-amd64.sha256 "$APP/cli-proxy-api.sha256"

expected="$(awk '{print $1}' cli-proxy-api.sha256)"
actual="$(sha256sum cli-proxy-api.new | awk '{print $1}')"
[ "$expected" = "$actual" ] || { echo "sha256 mismatch"; rm -f cli-proxy-api.new; exit 1; }

chmod 755 cli-proxy-api.new
mv -f cli-proxy-api.new cli-proxy-api

./restart.sh 2>/dev/null || {
  kill "$(cat cliproxyapi-plus.pid)" 2>/dev/null || true
  nohup ./cli-proxy-api -config ./config.yaml > logs/cliproxyapi-plus.nohup.log 2>&1 &
  echo $! > cliproxyapi-plus.pid
}
```

> 推荐节奏：面板看到上游有新版本后，等待当天 21:00 后的 Action 发布 `latest`，然后执行上面的 Release 更新命令。也可以在 GitHub Actions 页面手动 Run workflow 立即发布。

### 6.2 本地维护机可选：`update-cpa`

`update-cpa` 是高级维护命令，适合你想在本机立即同步两个上游并重新构建，而不是等待 GitHub Release。

```bash
update-cpa
```

它会在本机自动完成：

1. 拉取本仓库 `main` 分支最新维护脚本。
2. 同步 CLIProxyAPI 上游。
3. 同步 CPA-Manager-Plus 上游。
4. 应用 CPA-PLUS 集成 patch 和面板 overlay。
5. 构建新的 Linux 二进制。
6. 替换 `/root/apps/cliproxyapi-plus/cli-proxy-api`。
7. 自动重启服务。

日常普通更新推荐优先使用 `latest` Release；`update-cpa` 只作为本地快速构建或排障补充。

默认环境变量：

```text
CPA_PLUS_REPO_DIR=/root/.config/superpowers/worktrees/CLIProxyAPI/main
CPA_PLUS_APP_DIR=/root/apps/cliproxyapi-plus
CPA_PLUS_PORT=8317
CPA_PLUS_SKIP_TESTS=1
```

临时指定端口示例：

```bash
CPA_PLUS_PORT=8318 update-cpa
```

临时指定安装目录示例：

```bash
CPA_PLUS_APP_DIR=/root/apps/cliproxyapi-plus-test update-cpa
```

> 注意：`update-cpa` 会保留你现有的 `config.yaml`、数据和日志，不会覆盖你的密钥配置。

---

## 7. 手动构建二进制

如果只想构建，不安装：

```bash
cd /root/.config/superpowers/worktrees/CLIProxyAPI/main
./scripts/build-linux-binary.sh --skip-tests
```

生成文件：

```text
dist/CLIProxyAPI-linux-amd64
```

完整临时源码目录：

```text
.build/out/CLIProxyAPI
```

这个目录是自动生成的，不需要手动维护。

---

## 8. 自动同步和自动 Release

仓库内置 GitHub Actions：

```text
.github/workflows/auto-sync-release.yml
```

它会每天北京时间 21:00 自动运行一次：

```text
cron: 0 13 * * *
```

说明：GitHub Actions 的 cron 使用 UTC 时间，`13:00 UTC` 对应北京时间 `21:00`。

也可以手动触发：

```text
GitHub 仓库 → Actions → Auto sync upstream and release binary → Run workflow
```

自动流程：

1. 拉取两个上游最新 `main`。
2. 尝试重新应用并刷新 CPA-PLUS patch。
3. 构建 Linux amd64 二进制。
4. 上游有变化时提交到本仓库 `main`。
5. 发布版本 Release，例如：

```text
v7.1.59-plus.ba4993c6
```

6. 同时刷新固定的 `latest` Release。

日常使用只需要关注这个固定 Release：

```text
https://github.com/julioaaericksonaa/CPA-PLUS/releases/tag/latest
```

每次 `latest` 刷新后，本地服务器只需要下载 Release 里的：

```text
CLIProxyAPI-linux-amd64
CLIProxyAPI-linux-amd64.sha256
```

然后替换 `/root/apps/cliproxyapi-plus/cli-proxy-api` 并重启。

GitHub 仓库需要开启：

```text
Settings → Actions → General → Workflow permissions → Read and write permissions
```

workflow 使用 GitHub 自动提供的 `GITHUB_TOKEN`。不要把个人 PAT、密钥、配置文件提交到仓库。

如果上游发生冲突或构建失败，workflow 会失败并停止发布，不会覆盖已有 Release。

---

## 9. 如何判断是否需要更新

面板的系统概览会显示两个上游版本：

- CLIProxyAPI 上游版本
- Plus 上游版本

也可以用命令查看响应头：

```bash
curl -sS -D - -o /dev/null http://127.0.0.1:8317/v0/management/config | grep -Ei 'X-CPA|X-PLUS'
```

示例：

```text
X-Cpa-Upstream-Version: v7.1.59
X-Cpa-Upstream-Commit: 44ea9abc
X-Plus-Upstream-Version: ba4993c6
X-Plus-Upstream-Commit: ba4993c6
```

如果面板提示上游有新版本，推荐等待或手动触发 GitHub Actions 发布新的 `latest` Release，然后按第 6.1 节下载 Release 二进制更新。

---

## 10. 目录说明

仓库目录：

```text
cpa-plus-core/              # 从 CLIProxyAPI 上游生成集成源码
cpa-plus-web/               # CPA-Manager-Plus 面板 patch/overlay
patches/cliproxyapi/        # CPA-PLUS 集成 patch
scripts/build-linux-binary.sh
scripts/install-linux.sh
scripts/install-systemd.sh
scripts/update-linux.sh
scripts/update-cpa
scripts/ci-sync-upstream.sh # GitHub Actions 自动同步辅助脚本
```

运行目录：

```text
/root/apps/cliproxyapi-plus/
├── cli-proxy-api
├── config.yaml
├── config.example.yaml
├── data/
├── logs/
├── start-detached.sh
├── stop.sh
└── restart.sh
```

---

## 11. 隐私和安全

不要提交这些文件：

```text
config.yaml
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

这些已经在 `.gitignore` 中排除。

本机用于推送 GitHub 的 PAT 可以放在：

```text
/root/.config/cpa-plus/publish.env
```

这个文件只留在本机，不要提交到仓库。

---

## 12. 常见问题

### 端口被占用

```bash
ss -ltnp | grep ':8317 '
```

如果确实需要换端口，优先修改运行目录里的配置：

```bash
nano /root/apps/cliproxyapi-plus/config.yaml
```

把 `port: 8317` 改成其他端口，例如 `port: 8318`，然后重启服务。

### 服务启动失败

先看日志：

```bash
tail -n 100 /root/apps/cliproxyapi-plus/logs/cliproxyapi-plus.nohup.log
```

常见原因：

- `config.yaml` 缩进错误。
- `remote-management.secret-key` 未设置。
- 端口被其他进程占用。
- Release 下载不完整或校验失败，需要重新下载 `CLIProxyAPI-linux-amd64` 和 `.sha256`。
- 如果是自动发布失败，需要查看 GitHub Actions 日志。

### 面板显示旧版本

浏览器缓存导致，按：

```text
Ctrl + F5
```

### 自动 Release 失败

进入 GitHub：

```text
Actions → Auto sync upstream and release binary
```

查看失败日志。通常是上游代码冲突、依赖构建失败或 GitHub Actions 权限未开启写入。
