# CPA-PLUS

CPA-PLUS 是把以下两个上游项目组合成 **单端口 Linux 二进制** 的维护仓库：

- CLIProxyAPI：`https://github.com/router-for-me/CLIProxyAPI`
- CPA-Manager-Plus：`https://github.com/seakee/CPA-Manager-Plus`

本仓库现在只维护二进制构建流，不再以 Docker 项目为主线。

## 项目定位

- 默认分支：`main`
- 默认端口：`8317`
- 默认安装目录：`/root/apps/cliproxyapi-plus`
- 面板入口：`http://host:8317/management.html`
- 生成二进制：`dist/CLIProxyAPI-linux-amd64`

## 快速安装

```bash
git clone https://github.com/julioaaericksonaa/CPA-PLUS.git
cd CPA-PLUS
./scripts/install-linux.sh --skip-tests
```

安装后编辑本机配置：

```bash
nano /root/apps/cliproxyapi-plus/config.yaml
```

至少设置管理密钥：

```yaml
remote-management:
  secret-key: "换成你自己的强密码"
```

启动：

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
```

访问：

```text
http://host:8317/management.html
```

## systemd 开机自启

```bash
sudo ./scripts/install-systemd.sh
```

常用命令：

```bash
systemctl status cliproxyapi-plus --no-pager
systemctl restart cliproxyapi-plus
journalctl -u cliproxyapi-plus -f
```

## 一键更新

安装脚本会写入全局命令：

```bash
update-cpa
```

它会执行：

1. 拉取本仓库 `main` 分支最新维护脚本。
2. 同步 CLIProxyAPI 和 CPA-Manager-Plus 上游。
3. 应用 CPA-PLUS patch/overlay。
4. 重建 Linux 二进制。
5. 覆盖安装目录中的二进制并重启服务。

默认参数：

```text
CPA_PLUS_REPO_DIR=/root/.config/superpowers/worktrees/CLIProxyAPI/auto
CPA_PLUS_APP_DIR=/root/apps/cliproxyapi-plus
CPA_PLUS_PORT=8317
CPA_PLUS_SKIP_TESTS=1
```

## 手动构建

```bash
./scripts/build-linux-binary.sh --skip-tests
```

生成：

```text
dist/CLIProxyAPI-linux-amd64
```

完整临时源码：

```text
.build/out/CLIProxyAPI
```

## 自动 Release

仓库包含 GitHub Actions workflow：

```text
.github/workflows/auto-sync-release.yml
```

计划任务：每 2 天自动运行一次，也可以在 GitHub Actions 页面手动运行。

自动流程：

1. 同步两个上游 `main`。
2. 尝试自动刷新 CPA-PLUS patch 和上游基线文件。
3. 构建 Linux amd64 二进制。
4. 上游有变化时提交到 `main`。
5. 发布版本 Release，例如：
   `v7.1.59-plus.ba4993c6`
6. 同时更新固定 `latest` Release。

需要在 GitHub 仓库开启：

```text
Settings → Actions → General → Workflow permissions → Read and write permissions
```

workflow 使用 GitHub 自动提供的 `GITHUB_TOKEN`，不需要提交你的 PAT 或 Key。

如果上游冲突、patch 失败或构建失败，workflow 会失败并停止发布，不会覆盖已有 Release。

## 上游版本显示

运行中的服务会在管理接口响应头中暴露上游版本：

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

## 目录说明

```text
cpa-plus-core/              # 从 CLIProxyAPI 上游生成集成源码
cpa-plus-web/               # Plus 面板 patch/overlay
patches/cliproxyapi/        # CPA-PLUS 集成 patch
scripts/build-linux-binary.sh
scripts/install-linux.sh
scripts/update-linux.sh
scripts/update-cpa
scripts/ci-sync-upstream.sh # GitHub Actions 自动同步辅助脚本
```

## 隐私注意

不要提交以下文件：

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

这些已在 `.gitignore` 中排除。
