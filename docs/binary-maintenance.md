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

- 版本 Release：`${CLI_UPSTREAM_VERSION}-plus.${PLUS_UPSTREAM_VERSION}`，例如 `v7.1.59-plus.ba4993c6`。
- 固定 Release：`latest`，始终覆盖为最新成功构建。

## 本地维护

```bash
./scripts/ci-sync-upstream.sh
./scripts/build-linux-binary.sh --skip-tests
```

检查服务版本头：

```bash
curl -sS -D - -o /dev/null http://127.0.0.1:8317/v0/management/config | grep -Ei 'X-CPA|X-PLUS'
```
