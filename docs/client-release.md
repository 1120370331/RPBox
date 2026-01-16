# RPBox 客户端发布指南

本文档介绍如何构建、发布客户端更新，以及自动更新机制的工作原理。

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
