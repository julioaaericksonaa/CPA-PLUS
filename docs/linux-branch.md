# CPA-PLUS linux 分支维护文档

`linux` 分支用于维护 Linux 二进制构建、安装、更新和自动运行。

## 默认端口文件

```text
PORT
```

默认：

```text
8318
```

修改：

```bash
echo 8319 > PORT
./scripts/install-linux.sh --skip-tests
```

## 常用命令

构建二进制：

```bash
./scripts/build-linux-binary.sh --skip-tests
```

安装到 `/root/apps/cliproxyapi-plus`：

```bash
./scripts/install-linux.sh --skip-tests
```

安装 systemd 自动运行：

```bash
sudo ./scripts/install-systemd.sh
```

更新并自动重启：

```bash
./scripts/update-linux.sh --skip-tests
```

## systemd 管理

```bash
systemctl status cliproxyapi-plus --no-pager
systemctl restart cliproxyapi-plus
journalctl -u cliproxyapi-plus -f
```

## 脚本管理

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
/root/apps/cliproxyapi-plus/stop.sh
/root/apps/cliproxyapi-plus/restart.sh
```

## 默认安装位置

```text
/root/apps/cliproxyapi-plus
```

## 与 main 分支区别

```text
main  维护 Docker 项目
linux 维护 Linux 二进制项目和 systemd 自动运行
```
