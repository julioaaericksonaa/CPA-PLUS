# CPA-PLUS Linux Binary Branch

`linux` 分支是 CPA-PLUS 的 **Linux 二进制维护分支**。

它不维护完整上游源码，也不以 Docker 为主；它维护 patch/overlay 构建流，用于从两个上游项目自动生成并编译 Linux 二进制。

## 分支定位

```text
main  = Docker 项目分支，适合容器部署和源码构建
linux = Linux 二进制项目分支，适合直接运行二进制
```

本分支目标：

- 生成单个 Linux 二进制：`dist/CLIProxyAPI-linux-amd64`
- 直接运行，不依赖 Docker
- 默认独立安装到：`/root/apps/cliproxyapi-plus`
- 默认端口：`8318`
- 面板入口：`http://host:8318/management.html`

## 构建二进制

```bash
git clone -b linux https://github.com/julioaaericksonaa/CPA-PLUS.git CPA-PLUS-linux
cd CPA-PLUS-linux
./scripts/build-linux-binary.sh --skip-tests
```

生成文件：

```text
dist/CLIProxyAPI-linux-amd64
```

完整生成源码位于：

```text
.build/out/CLIProxyAPI
```

## 安装到本机

默认安装到 `/root/apps/cliproxyapi-plus`，端口 `8318`：

```bash
./scripts/install-linux.sh --skip-tests
```

安装后编辑：

```text
/root/apps/cliproxyapi-plus/config.yaml
```

至少设置：

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
http://host:8318/management.html
```

## 更新二进制项目

```bash
./scripts/update-linux.sh --skip-tests
```

指定上游版本：

```bash
./scripts/update-linux.sh --cli-ref v7.1.54 --plus-ref main --skip-tests
```

自定义安装目录和端口：

```bash
CPA_PLUS_APP_DIR=/root/apps/cliproxyapi-plus \
CPA_PLUS_PORT=8318 \
./scripts/install-linux.sh --skip-tests
```

## 运行维护

启动：

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
```

停止：

```bash
/root/apps/cliproxyapi-plus/stop.sh
```

重启：

```bash
/root/apps/cliproxyapi-plus/restart.sh
```

日志：

```bash
tail -f /root/apps/cliproxyapi-plus/logs/main.log
```

数据：

```text
/root/apps/cliproxyapi-plus/data/usage.sqlite
```

## 构建流程

```text
router-for-me/CLIProxyAPI
  -> cpa-plus-core/prepare-source.sh
  -> patches/cliproxyapi/*.patch

seakee/CPA-Manager-Plus apps/web
  -> cpa-plus-web/patch-plus-web-integrated.py
  -> npm build
  -> embedded management.html

Go build
  -> dist/CLIProxyAPI-linux-amd64
```

## 重要文件

```text
patches/cliproxyapi/0001-cpa-plus-integration.patch
cpa-plus-core/prepare-source.sh
cpa-plus-web/patch-plus-web-integrated.py
scripts/build-linux-binary.sh
scripts/install-linux.sh
scripts/update-linux.sh
```

## 隐私安全

不要提交：

```text
config.yaml
.env
auths/  # 除 auths/.gitkeep
data/
logs/
.build/
dist/
*.sqlite
*.db
*.key
*.pem
.codex/
.claude/
.gemini/
```

## 来源

CPA-PLUS 基于以下项目整合维护：

- CLIProxyAPI：https://github.com/router-for-me/CLIProxyAPI
- CPA-Manager-Plus：https://github.com/seakee/CPA-Manager-Plus
