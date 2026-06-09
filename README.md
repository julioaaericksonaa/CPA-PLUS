# CPA-PLUS

CPA-PLUS maintains a **single-port Linux binary** that integrates:

- CLIProxyAPI: `https://github.com/router-for-me/CLIProxyAPI`
- CPA-Manager-Plus: `https://github.com/seakee/CPA-Manager-Plus`

This repository is now binary-first. Docker-oriented source maintenance has been removed from the main line.

## Defaults

- Branch: `main`
- Port: `8317`
- Install dir: `/root/apps/cliproxyapi-plus`
- Panel: `http://host:8317/management.html`
- Binary: `dist/CLIProxyAPI-linux-amd64`

## Install

```bash
git clone https://github.com/julioaaericksonaa/CPA-PLUS.git
cd CPA-PLUS
./scripts/install-linux.sh --skip-tests
```

Edit local config:

```bash
nano /root/apps/cliproxyapi-plus/config.yaml
```

Set a management secret:

```yaml
remote-management:
  secret-key: "replace-with-your-strong-password"
```

Start:

```bash
/root/apps/cliproxyapi-plus/start-detached.sh
```

Open:

```text
http://host:8317/management.html
```

## Update

The installer writes a global update command:

```bash
update-cpa
```

It pulls this repository's `main` branch, syncs both upstream projects, rebuilds the Linux binary, installs it to `/root/apps/cliproxyapi-plus`, and restarts the service.

## Build manually

```bash
./scripts/build-linux-binary.sh --skip-tests
```

Output:

```text
dist/CLIProxyAPI-linux-amd64
```

## Auto Release

GitHub Actions workflow:

```text
.github/workflows/auto-sync-release.yml
```

It runs every two days and can also be triggered manually. When upstream changes are detected, it builds the Linux amd64 binary, commits refreshed metadata/patches to `main`, creates a versioned Release such as `v7.1.59-plus.ba4993c6`, and refreshes the fixed `latest` Release.

Enable write access for GitHub Actions:

```text
Settings → Actions → General → Workflow permissions → Read and write permissions
```

The workflow uses GitHub's built-in `GITHUB_TOKEN`; no PAT or private key should be committed.
