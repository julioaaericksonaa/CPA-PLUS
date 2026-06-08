# CPA-PLUS linux 分支维护文档

`linux` 分支用于维护 Linux 二进制构建流。

## 常用命令

构建：

```bash
./scripts/build-linux-binary.sh --skip-tests
```

安装：

```bash
./scripts/install-linux.sh --skip-tests
```

更新并重启：

```bash
./scripts/update-linux.sh --skip-tests
```

## 默认安装位置

```text
/root/apps/cliproxyapi-plus
```

## 默认端口

```text
8318
```

## 与 main 分支区别

```text
main 维护 Docker 项目
linux 维护 Linux 二进制项目
```
