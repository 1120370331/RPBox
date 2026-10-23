# RPBox 客户端发布指南

本文档介绍如何构建、发布客户端更新，以及自动更新机制的工作原理。

## 推荐：按需使用本机加速 Actions

`Release Client` 支持 `builder=local`。本机完成签名 NSIS 构建和直接上传，Actions 保留日志、签名密钥和安装包归档；版本元数据和下载页仍按原有 stable/beta 渠道更新。普通 tag 发布、`builder=hosted` 继续使用 GitHub Windows。

已观察到的基线：[2026-10-22 的发布](https://github.com/1120370331/RPBox/actions/runs/39779144074) 总计约 21 分 50 秒，Tauri 构建 7 分 34 秒，服务器上传 12 分 46 秒。新的流程只检出客户端、共享源码和脚本，跳过手机截图、营销资源和参考仓库；去掉 MSI、整目录归档和 Ubuntu 中转，将上传合为一次 SCP，本机 Rust 产物持续缓存。实际提速以运行记录为准，第一次冷构建仍需下载依赖。

前提：本次 workflow 和脚本已推送到要构建的分支；本机安装 PowerShell 7、Git、Node、stable MSVC Rust、Visual Studio C++ 工具链、Windows SDK 和 OpenSSH，`gh` 已登录且可管理本仓库 Runner。

```powershell
# 第一次安装；下载官方 Runner 并核对 GitHub 返回的 SHA256
pwsh -NoProfile -File scripts/start-local-runner.ps1 -InstallOnly

# 每次发包前，在终端 A 启动（前台运行，一份任务完成后自动注销退出）
pwsh -NoProfile -File scripts/start-local-runner.ps1

# 终端 B：构建、归档并验证临时上传，不发布更新、不更新下载页
gh workflow run release-client.yml --ref main -f version=1.0.7-1 -f channel=beta -f builder=local -F build_only=true

# 正式发布测试版；版本应为尚未发布的新版本
gh workflow run release-client.yml --ref main -f version=1.0.7-1 -f channel=beta -f builder=local -F build_only=false
```

每份任务都要重新启动 Runner；正式版使用 `channel=stable` 和正式版本号。`--ref` 指定的是要构建的**远端分支/tag**，本地未提交文件不会被打包。不要同时推同一版本 tag 再手动发布，以免重复运行。本机未开机时选择 `builder=hosted`，本地任务不会自动转到云端；已经排队的本地任务需取消后重新选择 hosted。

Runner 默认安装在 `%LOCALAPPDATA%\RPBox\actions-runner`，工作副本在 `_work`，编译缓存位于 `_work\_rpbox-cache\cargo-target`，均独立于开发仓库。通过 Git alternates 只读复用开发仓库已提交的对象，减少首次下载；运行期间请保留该开发仓库。checkout 仍按远端 SHA 清理并检出工作副本，避免旧源码或未提交文件混入。此流程只用于本人审核过的发布源码，不配置 PR 触发器；ephemeral 代表只接一份任务，并非虚拟机隔离。可按需先执行 `rustup update stable`，升级编译器后首次构建会重建相关缓存。

安装和启动时还会按 Action 的实际 commit SHA 缓存官方下载包到 `action-cache`，供 Runner 的 `ACTIONS_RUNNER_ACTION_ARCHIVE_CACHE` 使用，避免每次初始化重新下载。Action 标签升级后会按新 SHA 补充缓存，远端 workflow 使用其他版本时仍由 Runner 正常下载。仓库 `.gitmodules` 补齐已有的两个参考子模块定义；桌面构建不初始化或下载它们。

若终端没有显式代理环境变量，启动脚本会将 Windows 当前系统代理传给 Runner 的 Git/Node 子进程，退出时恢复环境；不修改系统代理设置。使用系统代理时须在发包期间保持代理软件运行。

NSIS 工具独立缓存在 Cargo target 的 `.tauri` 目录，不与日常开发使用的其他 Tauri CLI 版本混用。准备步骤按已安装 CLI 的官方 release 源码获取工具 URL 和校验值，用 PowerShell 下载并核验，避开旧 Tauri 下载器与 Windows 代理的兼容问题；工具下载失败会在编译之前报错。

签名和部署使用现有 GitHub Secrets，脚本不读取开发目录 `.env`，SSH 私钥只在上传期间写入受限临时目录并在结束时删除。Runner 使用独立 `ci.gitconfig`，不继承开发账号的 Git URL 改写和凭据助手；不传入启动终端的 GitHub Token。可选 `SSH_KNOWN_HOSTS` Secret 固定服务器主机公钥；未设置时沿用原流程的 `ssh-keyscan`。不修改本机的 Git/SSH 配置或 `known_hosts`。

上传前后都校验 SHA256，只有安装包、签名和版本元数据就绪后才切换 `latest.json` / `latest-beta.json`。失败时旧的渠道指针保留，服务器可能留下 `.upload-*` 排错目录。发布互斥执行，避免两个版本同时修改指针。`build_only=true` 适合验证新 Runner：使用正式签名密钥并上传到独立临时目录，校验后清理，不创建正式版本目录、不切换渠道指针、不更新下载页。

脚本验证：`pwsh -NoProfile -File scripts/tests/client-release.tests.ps1`。此验证用临时目录及本机 Git Bash 检查远端发布命令，拦截 SSH/SCP，不连接生产环境。

以下旧手动发布脚本说明保留供历史维护参考，新发布优先使用上述 Actions 流程。

## 目录

- [前置准备](#前置准备)
- [发布新版本](#发布新版本)
- [服务器配置](#服务器配置)
- [自动更新原理](#自动更新原理)
- [故障排查](#故障排查)

---

## 前置准备

### 1. 签名密钥

客户端更新需要签名验证，确保更新包来源可信。

**密钥文件位置：**
- 私钥：`client/.tauri/rpbox.key`（保密）
- 公钥：`client/.tauri/rpbox.key.pub`

**如果需要重新生成：**
```powershell
cd client
npx tauri signer generate -w .tauri/rpbox.key
```

### 2. 环境变量

确保 `client/.env` 包含签名配置：

```env
TAURI_SIGNING_PRIVATE_KEY=<私钥内容>
TAURI_SIGNING_PRIVATE_KEY_PASSWORD=<密码>
```

### 3. SSH 访问

确保本地可以 SSH 连接到发布服务器：

```powershell
ssh root@your-server.com
```

---

## 发布新版本

### 使用发布脚本

```powershell
# 基本用法
.\scripts\release.ps1 -Version "0.2.0"

# 带更新说明
.\scripts\release.ps1 -Version "0.2.0" -Notes "修复了登录问题，优化了性能"

# 指定服务器
.\scripts\release.ps1 -Version "0.2.0" -SSHHost "api.rpbox.app" -SSHUser "root"
```

### 脚本执行流程

| 步骤 | 说明 |
|------|------|
| 1 | 加载 `.env` 签名密钥 |
| 2 | 更新版本号（tauri.conf.json、Cargo.toml、package.json） |
| 3 | 执行 `npm run tauri build` 构建签名安装包 |
| 4 | 收集产物到 `releases/{version}/` |
| 5 | SSH 上传到服务器 |
| 6 | 提示更新服务器配置 |

### 构建产物

构建完成后，`releases/{version}/` 目录包含：

```
releases/0.2.0/
├── RPBox_0.2.0_x64-setup.nsis.zip      # Windows 安装包
├── RPBox_0.2.0_x64-setup.nsis.zip.sig  # 签名文件
└── update.json                          # 更新信息
```

---

## 服务器配置

### 1. 目录结构

在服务器上创建以下目录：

```bash
mkdir -p /var/www/rpbox/releases
mkdir -p /var/www/rpbox/signatures
```

### 2. 更新 config.yaml

发布后需要更新服务器配置：

```yaml
updater:
  latest_version: "0.2.0"
  base_url: "https://api.rpbox.app/releases"
  release_notes: "修复了登录问题，优化了性能"
  pub_date: "2025-01-15T10:00:00Z"
  signature_dir: "/var/www/rpbox/signatures"
```

### 3. 重启服务

```bash
systemctl restart rpbox
```

---

## 自动更新原理

### 更新流程

```
客户端启动
    │
    ▼
请求 /api/v1/updater/{target}/{arch}/{current_version}
    │
    ▼
服务器比较版本号
    │
    ├─ 相同 → 返回 204 No Content
    │
    └─ 有新版本 → 返回更新信息 JSON
                    │
                    ▼
              客户端下载更新包
                    │
                    ▼
              验证签名（用内置公钥）
                    │
                    ▼
              安装并重启
```

### API 响应格式

**有更新时返回：**

```json
{
  "version": "0.2.0",
  "notes": "更新说明",
  "pub_date": "2025-01-15T10:00:00Z",
  "url": "https://api.rpbox.app/releases/0.2.0/RPBox_0.2.0_x64-setup.nsis.zip",
  "signature": "dW50cnVzdGVkIGNvbW1lbnQ6..."
}
```

**无更新时返回：** `204 No Content`

---

## 故障排查

### 客户端报权限错误

```
updater.check not allowed
```

**解决：** 检查 `client/src-tauri/capabilities/default.json` 是否包含：

```json
{
  "permissions": [
    "updater:default",
    "process:allow-restart",
    "process:allow-exit"
  ]
}
```

### 签名验证失败

**原因：** 公钥不匹配或签名文件损坏

**解决：**
1. 确认 `tauri.conf.json` 中的 `pubkey` 与 `.tauri/rpbox.key.pub` 内容一致
2. 重新构建并上传签名文件

### 检测不到更新

**检查：**
1. 服务器 `config.yaml` 中 `latest_version` 是否已更新
2. 客户端当前版本是否低于服务器版本
3. API 是否正常响应：`curl https://api.rpbox.app/api/v1/updater/windows/x86_64/0.1.0`

### 下载失败

**检查：**
1. 更新包 URL 是否可访问
2. 服务器是否正确配置了静态文件服务
3. 防火墙是否放行

---

## 相关文件

| 文件 | 说明 |
|------|------|
| `client/.env` | 签名密钥配置 |
| `client/.tauri/rpbox.key` | 私钥（保密） |
| `client/.tauri/rpbox.key.pub` | 公钥 |
| `client/src-tauri/tauri.conf.json` | Tauri 配置（含公钥） |
| `client/src-tauri/capabilities/default.json` | 权限配置 |
| `client/src/composables/useUpdater.ts` | 前端更新逻辑 |
| `server/internal/api/updater.go` | 后端更新 API |
| `scripts/release.ps1` | 发布脚本 |
