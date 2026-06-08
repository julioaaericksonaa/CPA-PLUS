# CPA-PLUS auto 分支维护文档

## 目标

`auto` 分支把 CPA-PLUS 从“完整源码 fork”改成“patch/overlay 自动构建流”。它的目标是接近 `CLIProxyAPI-Pro` 的维护体验：上游更新频繁时，不手工搬代码，而是自动拉取上游、应用本项目补丁并构建镜像。

## 构建流程

```text
router-for-me/CLIProxyAPI       -> git clone/checkout -> 应用 patches/cliproxyapi/*.patch
seakee/CPA-Manager-Plus apps/web -> rsync              -> cpa-plus-web/patch-plus-web-integrated.py
生成完整源码                    -> .build/out/CLIProxyAPI
Docker build                    -> cpa-plus:auto
Compose run                     -> 8317/management.html
```

## 常用命令

首次安装：

```bash
./scripts/install-auto.sh
```

日常更新：

```bash
./scripts/update-auto.sh
```

指定 CLIProxyAPI tag：

```bash
./scripts/update-auto.sh --cli-ref v7.1.54
```

指定 Plus 分支：

```bash
./scripts/update-auto.sh --plus-ref main
```

只生成源码：

```bash
./cpa-plus-core/prepare-source.sh --skip-tests
```

只用已有源码构建镜像：

```bash
./cpa-plus-core/build-docker.sh --no-prepare
```

## patch 冲突处理

如果 `prepare-source.sh` 在 `git apply --3way` 失败：

1. 查看失败位置。
2. 到 `.build/src/CLIProxyAPI` 或 `.build/out/CLIProxyAPI` 对比上游改动。
3. 在 `main` 完整源码分支合并上游并修复 CPA-PLUS 功能。
4. 运行：

```bash
./scripts/regenerate-core-patch.sh /root/code/cpa-plus-merge-study/CLIProxyAPI cli-upstream/main
```

5. 回到 auto 分支重新验证：

```bash
./cpa-plus-core/prepare-source.sh
```

## 发布建议

如果以后要启用 GitHub Actions 自动发镜像，可以使用 `.github/workflows/auto-build.yml`。注意：推送 workflow 文件需要 GitHub PAT 带 `workflow` 权限。

本地私有使用时，不需要 GitHub Actions，直接运行：

```bash
./scripts/update-auto.sh
```
