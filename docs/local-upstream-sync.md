# CPA-PLUS 本地高频上游同步流程

本项目按“本地整合版”维护，不需要把代码推送到 GitHub。同步脚本只做 `fetch/clone/merge/rsync/test/build`，不会执行 `git push`。

## 远程布局

推荐本地 `.git/config` 中只保留两个上游远程：

```text
cli-upstream   https://github.com/router-for-me/CLIProxyAPI.git
plus-upstream  https://github.com/seakee/CPA-Manager-Plus.git
```

两个远程的 push URL 会被设置为 `DISABLED`，防止误推。

检查：

```bash
git remote -v
```

## 一键更新

从项目根目录运行：

```bash
./scripts/local-update-all.sh
```

它会依次执行：

1. 检查工作区是否干净。
2. 拉取并合并 `cli-upstream` 默认分支。
3. 拉取 `plus-upstream` 的 `apps/web`。
4. 同步到本项目 `web/manager-plus`。
5. 自动把 Plus 前端 API 路径改回集成版 `/v0/management/plus/*`。
6. 刷新 `web/manager-plus/package-lock.json`。
7. 运行 Go 测试、前端测试、前端构建。
8. 清理 `node_modules` 和 `dist` 构建产物。

成功后检查差异：

```bash
git status --short
git diff --stat
```

确认无误后提交：

```bash
git add scripts docs web/manager-plus internal README.md README_CN.md config.example.yaml Dockerfile docker-compose.yml
git commit -m "chore: sync upstream updates"
```

不要使用 `git add .`，避免误提交 `config.yaml`、`auths/`、`data/`、`logs/`。

## 只更新 CLIProxyAPI 主体

```bash
./scripts/local-update-cli.sh
```

常用选项：

```bash
./scripts/local-update-cli.sh --dry-run
./scripts/local-update-cli.sh --skip-tests
./scripts/local-update-cli.sh --branch main
```

## 只更新 Plus 前端

```bash
./scripts/local-update-plus-web.sh
```

如果本机已经有 CPA-Manager-Plus checkout，可以避免重新 clone：

```bash
./scripts/local-update-plus-web.sh --source /root/code/cpa-plus-merge-study/CPA-Manager-Plus
```

常用选项：

```bash
./scripts/local-update-plus-web.sh --dry-run
./scripts/local-update-plus-web.sh --skip-tests
./scripts/local-update-plus-web.sh --skip-lock
./scripts/local-update-plus-web.sh --branch main
```

## 同步后部署

```bash
docker compose up -d --build
```

访问：

```text
http://host:8317/management.html
```

## 冲突处理

如果 CLIProxyAPI merge 冲突：

```bash
git status
# 编辑冲突文件
git add <resolved-files>
git commit
```

如果 Plus 前端同步后测试失败：

1. 先看 `web/manager-plus/src/services/api/usageService.ts` 是否仍使用 `/v0/management/plus/*`。
2. 运行路径补丁：

```bash
./scripts/patch-plus-web-integrated.py web/manager-plus
```

3. 再跑：

```bash
npm --prefix web/manager-plus ci
npm --prefix web/manager-plus test
npm --prefix web/manager-plus run build
rm -rf web/manager-plus/node_modules web/manager-plus/dist
```

## 隐私文件保护

这些文件/目录不应提交：

```text
config.yaml
.env
auths/
data/
logs/
*.sqlite
```

提交前一定看：

```bash
git status --short
```
