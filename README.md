# CPA-PLUS Linux Binary Branch

`linux` 分支是 CPA-PLUS 的 **Linux 二进制维护分支**。

它不维护完整上游源码，也不以 Docker 为主；它维护 patch/overlay 构建流，用于从两个上游项目自动生成并编译 Linux 二进制。

## 分支定位

```text
main  = Docker 项目分支，适合容器部署和源码构建
linux = Linux 二进制项目分支，适合直接运行二进制、systemd 自启动
```

本分支目标：

- 生成单个 Linux 二进制：`dist/CLIProxyAPI-linux-amd64`
- 直接运行，不依赖 Docker
- 默认独立安装到：`/root/apps/cliproxyapi-plus`
- 默认端口由 `PORT` 文件维护，当前为 `8317`
- 面板入口：`http://host:8317/management.html`


## 最近更新：v7.1.54-plus.5

- 修复服务端 Codex 定时巡检不触发：Plus 启动后后台巡检 worker 会自动运行。
- 支持配置更新时自动启动/停止巡检 worker，按频率和按时间点配置均可由服务端后台执行。
- 优化 worker 停止流程，更新/重启服务时避免后台任务访问已关闭的 SQLite。
- 优化 SQLite 并发稳定性：单连接 + `busy_timeout=5000`，降低巡检、统计、面板并发访问时的锁冲突。
- 默认端口已统一为 `8317`，面板入口为 `http://host:8317/management.html`。

## 端口维护

仓库根目录有端口文件：

```text
PORT
```

默认内容：

```text
8317
```

修改默认安装端口：

```bash
echo 8317 > PORT
```

也可以安装时临时覆盖：

```bash
CPA_PLUS_PORT=8319 ./scripts/install-linux.sh --skip-tests
```

安装脚本会同步更新：

```text
/root/apps/cliproxyapi-plus/PORT
/root/apps/cliproxyapi-plus/config.yaml
/root/apps/cliproxyapi-plus/start-detached.sh
/root/apps/cliproxyapi-plus/stop.sh
/root/apps/cliproxyapi-plus/restart.sh
```

如果已经安装过，修改端口后重新执行安装或更新脚本：

```bash
./scripts/install-linux.sh --skip-tests
# 或
./scripts/update-linux.sh --skip-tests
```

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

默认安装到 `/root/apps/cliproxyapi-plus`，端口读取 `PORT` 文件：

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

手动启动：

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
```

访问：

```text
http://host:8317/management.html
```

## 自动运行 / 开机自启

安装二进制后，可安装 systemd 服务：

```bash
sudo ./scripts/install-systemd.sh
```

默认服务名：

```text
cliproxyapi-plus.service
```

查看状态：

```bash
systemctl status cliproxyapi-plus --no-pager
```

启动：

```bash
systemctl start cliproxyapi-plus
```

停止：

```bash
systemctl stop cliproxyapi-plus
```

重启：

```bash
systemctl restart cliproxyapi-plus
```

开机自启：

```bash
systemctl enable cliproxyapi-plus
```

取消开机自启：

```bash
systemctl disable cliproxyapi-plus
```

查看日志：

```bash
journalctl -u cliproxyapi-plus -f
```

如果你不用 systemd，也可以继续用脚本：

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
/root/apps/cliproxyapi-plus/stop.sh
/root/apps/cliproxyapi-plus/restart.sh
```

## 更新二进制项目

```bash
./scripts/update-linux.sh --skip-tests
```

如果已经安装了 systemd 服务，`update-linux.sh` 会自动：

1. 拉取上游。
2. 应用 CPA-PLUS patch。
3. 构建新 Linux 二进制。
4. 覆盖 `/root/apps/cliproxyapi-plus/cli-proxy-api`。
5. 执行 `systemctl restart cliproxyapi-plus`。

如果没有 systemd 服务，则调用：

```bash
/root/apps/cliproxyapi-plus/restart.sh
```

指定上游版本：

```bash
./scripts/update-linux.sh --cli-ref v7.1.54 --plus-ref main --skip-tests
```

自定义安装目录、服务名和端口：

```bash
CPA_PLUS_APP_DIR=/root/apps/cliproxyapi-plus \
CPA_PLUS_SERVICE_NAME=cliproxyapi-plus \
CPA_PLUS_PORT=8317 \
./scripts/install-linux.sh --skip-tests
```

安装 systemd 时也使用相同变量：

```bash
CPA_PLUS_APP_DIR=/root/apps/cliproxyapi-plus \
CPA_PLUS_SERVICE_NAME=cliproxyapi-plus \
sudo -E ./scripts/install-systemd.sh
```

## 运行维护

脚本方式启动：

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
```

脚本方式停止：

```bash
/root/apps/cliproxyapi-plus/stop.sh
```

脚本方式重启：

```bash
/root/apps/cliproxyapi-plus/restart.sh
```

应用日志：

```bash
tail -f /root/apps/cliproxyapi-plus/logs/main.log
```

systemd 日志：

```bash
journalctl -u cliproxyapi-plus -f
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
PORT
patches/cliproxyapi/0001-cpa-plus-integration.patch
cpa-plus-core/prepare-source.sh
cpa-plus-web/patch-plus-web-integrated.py
scripts/build-linux-binary.sh
scripts/install-linux.sh
scripts/update-linux.sh
scripts/install-systemd.sh
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
