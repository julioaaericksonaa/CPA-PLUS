# CPA-PLUS 二进制维护说明

本仓库 `main` 分支只维护二进制构建流：patch、overlay、构建脚本、安装脚本和自动 Release workflow。

## 维护模型

1. `cpa-plus-core/prepare-source.sh` 克隆 CLIProxyAPI 上游。
2. 应用 `patches/cliproxyapi/0001-cpa-plus-integration.patch`。
3. 克隆 CPA-Manager-Plus 上游并复制 `apps/web` 到 `web/manager-plus`。
4. 执行 `cpa-plus-web/patch-plus-web-integrated.py` 和 overlay。
5. 构建单文件 Plus 面板并嵌入 CLIProxyAPI。
6. 构建 Linux amd64 二进制。

## 自动同步

`scripts/ci-sync-upstream.sh` 用于 CI：

- 准备最新上游源码。
- 如果 patch 可以应用，重新生成适配最新上游基线的 patch。
- 更新 `CLI_UPSTREAM_BASE` 和 `PLUS_UPSTREAM_BASE`。

如果 patch 冲突，脚本失败，GitHub Actions 不会发布 Release。

## 发布策略

- GitHub Actions 只在每天北京时间 21:00 执行一次。
- Action 先检测 CLIProxyAPI 和 CPA-Manager-Plus 两个上游；任意上游有更新才同步、构建并重建固定 `latest` Release。
- 如果两个上游都没有更新，Action 不安装构建环境、不构建、不发布。
- 本地维护者手动提交代码时，如果已经在本机重新拉取并构建出新版二进制，使用 `scripts/publish-latest-release.sh` 直接推送代码并删除重建 `latest` Release。
- 仓库不维护版本号 Release；固定 Release：`latest`。

## 本地维护

```bash
./scripts/ci-sync-upstream.sh
./scripts/build-linux-binary.sh --skip-tests
./scripts/publish-latest-release.sh
```

检查服务版本头：

```bash
curl -sS -D - -o /dev/null http://127.0.0.1:8317/v0/management/config | grep -Ei 'X-CPA|X-PLUS'
```
