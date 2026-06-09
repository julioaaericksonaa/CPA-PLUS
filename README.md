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
| 本地维护 checkout | `/root/.config/superpowers/worktrees/CLIProxyAPI/main` |

---

## 2. 快速安装运行

首次安装推荐直接从仓库构建并安装：

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

## 3. 常用运行命令

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

## 4. systemd 开机自启

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

## 5. 日常更新：只用 `update-cpa`

以后需要更新时，直接运行：

```bash
update-cpa
```

它会自动完成：

1. 拉取本仓库 `main` 分支最新维护脚本。
2. 同步 CLIProxyAPI 上游。
3. 同步 CPA-Manager-Plus 上游。
4. 应用 CPA-PLUS 集成 patch 和面板 overlay。
5. 构建新的 Linux 二进制。
6. 替换 `/root/apps/cliproxyapi-plus/cli-proxy-api`。
7. 自动重启服务。

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

## 6. 手动构建二进制

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

## 7. 自动同步和自动 Release

仓库内置 GitHub Actions：

```text
.github/workflows/auto-sync-release.yml
```

它会每 2 天自动运行一次，也可以手动触发：

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

GitHub 仓库需要开启：

```text
Settings → Actions → General → Workflow permissions → Read and write permissions
```

workflow 使用 GitHub 自动提供的 `GITHUB_TOKEN`。不要把个人 PAT、密钥、配置文件提交到仓库。

如果上游发生冲突或构建失败，workflow 会失败并停止发布，不会覆盖已有 Release。

---

## 8. 如何判断是否需要更新

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

如果面板提示上游有新版本，运行：

```bash
update-cpa
```

---

## 9. 目录说明

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

## 10. 隐私和安全

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

## 11. 常见问题

### 端口被占用

```bash
ss -ltnp | grep ':8317 '
```

如果确实需要换端口：

```bash
CPA_PLUS_PORT=8318 update-cpa
```

### 服务启动失败

先看日志：

```bash
tail -n 100 /root/apps/cliproxyapi-plus/logs/cliproxyapi-plus.nohup.log
```

常见原因：

- `config.yaml` 缩进错误。
- `remote-management.secret-key` 未设置。
- 端口被其他进程占用。
- 上游变更导致构建失败，需要查看 `update-cpa` 输出或 GitHub Actions 日志。

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
