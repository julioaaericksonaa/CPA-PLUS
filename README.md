# CPA-PLUS Auto Branch

这个 `auto` 分支采用 **CLIProxyAPI-Pro 风格的 patch/overlay 维护方式**：

- 不在分支里保存完整 CLIProxyAPI 源码。
- 不在分支里保存完整 CPA-Manager-Plus 前端源码。
- 只保存 CPA-PLUS 自己的补丁、前端适配脚本、构建脚本和部署文件。
- 构建时自动拉取两个上游，应用补丁，生成完整可运行源码，再构建 Docker 镜像。

最终运行形态仍然是：

```text
单 Docker
单端口 8317
完整 Plus 面板：http://host:8317/management.html
Plus API：/v0/management/plus/*
```

## 目录说明

```text
patches/cliproxyapi/0001-cpa-plus-integration.patch
  CPA-PLUS 后端、Dockerfile、配置、README、Plus API、SQLite store、collector 等核心补丁。

cpa-plus-web/patch-plus-web-integrated.py
  将上游 CPA-Manager-Plus 前端改成单端口 integrated API 路径。

cpa-plus-core/prepare-source.sh
  拉取两个上游并生成完整 CPA-PLUS 源码。

cpa-plus-core/build-docker.sh
  生成源码并构建 Docker 镜像。

scripts/install-auto.sh
  首次安装：构建镜像、生成 config.yaml、启动 compose。

scripts/update-auto.sh
  后续更新：重新拉取上游、应用补丁、构建镜像、重启容器。

compose.auto.yml
  auto 分支运行用 Docker Compose 文件。
```

## 首次安装

```bash
git clone -b auto https://github.com/julioaaericksonaa/CPA-PLUS.git CPA-PLUS-auto
cd CPA-PLUS-auto
./scripts/install-auto.sh
```

脚本会：

1. 拉取 `router-for-me/CLIProxyAPI`。
2. 拉取 `seakee/CPA-Manager-Plus`。
3. 应用 CPA-PLUS core patch。
4. 同步 Plus 前端到 `web/manager-plus`。
5. 自动改写前端 API 路径为 `/v0/management/plus/*`。
6. 构建本地镜像 `cpa-plus:auto`。
7. 如果没有 `config.yaml`，自动从生成的 `config.example.yaml` 复制一份。
8. 使用 `compose.auto.yml` 启动容器。

首次启动后请修改：

```yaml
remote-management:
  allow-remote: true
  secret-key: "换成你自己的强密码"
```

然后重启：

```bash
docker compose -f compose.auto.yml restart cpa-plus
```

访问：

```text
http://host:8317/management.html
```

## 后续更新

同步两个上游并重建：

```bash
./scripts/update-auto.sh
```

指定上游版本：

```bash
./scripts/update-auto.sh --cli-ref v7.1.54 --plus-ref main
```

只准备源码，不构建 Docker：

```bash
./cpa-plus-core/prepare-source.sh --skip-tests
```

生成后的完整源码位置：

```text
.build/out/CLIProxyAPI
```

如果想手动检查生成结果：

```bash
cd .build/out/CLIProxyAPI
git status --short
go test ./internal/config ./internal/managementasset ./internal/plusmanager/... ./internal/api ./internal/safemode -count=1
docker build -t cpa-plus:auto .
```

## 更新 CPA-PLUS 自身补丁

如果你在 `main` 完整源码分支继续开发了 CPA-PLUS 功能，可以回到本分支重新生成 core patch：

```bash
./scripts/regenerate-core-patch.sh /root/code/cpa-plus-merge-study/CLIProxyAPI cli-upstream/main
```

然后验证：

```bash
./cpa-plus-core/prepare-source.sh --skip-tests
```

确认没问题后提交 `auto` 分支。

## 隐私安全

不要提交这些内容：

```text
config.yaml
.env
auths/  # 除 auths/.gitkeep
data/
logs/
.build/
*.sqlite
*.db
*.key
*.pem
.codex/
.claude/
.gemini/
```

提交前检查：

```bash
git status --short
git diff --cached --name-only
```

## 和 main 分支的区别

```text
main 分支：完整源码整合版，适合直接开发和调试。
auto 分支：patch/overlay 自动构建版，适合高频同步上游和自动发布。
```

建议维护方式：

1. 平时部署/更新用 `auto` 分支。
2. 需要开发新功能时在 `main` 分支改。
3. main 验证通过后，用 `scripts/regenerate-core-patch.sh` 刷新 auto 分支补丁。
4. auto 分支跑 `prepare-source.sh` / `build-docker.sh` 验证。
