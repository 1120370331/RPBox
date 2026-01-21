# CLAUDE.md - RPBox 项目规范

## 项目愿景

> **打破封闭的 RP 生态，迈向开放、互联的新时代。**

长久以来，魔兽世界 RP 社区的创作者们在游戏内的沙盒中艰难探索——数据无法云端同步、剧情记录随风消散、优秀作品难以传播。

RPBox 不在沙盒内寻找漏洞，而是在沙盒之外构建基础设施。连接游戏内外，让数据自由流动，让创作不再孤立。

**这不仅是一个工具，更是 RP 生态走向开放与互联的基石。**

## 项目概述

RPBox 是一个服务于魔兽世界 RP 玩家的工具箱，主要解决 TotalRP3 插件用户的痛点：
- 人物卡跨设备备份同步
- RP剧情记录归档
- 社区分享交流
- TRP3道具市场
- AI 辅助创作

## 技术栈

| 层级 | 技术 |
|------|------|
| 桌面客户端 | Tauri 2.0 + Vue 3 + TypeScript + Pinia |
| 后端服务 | Go 1.21 + Gin + GORM |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 搜索 | MeiliSearch |
| WoW插件 | Lua (配套插件 RPBox_ChatLogger) |

## 目录结构

```
RPBox/
├── client/                 # Tauri 桌面客户端
│   ├── src/               # Vue 前端源码
│   │   ├── api/          # API 请求模块
│   │   ├── components/   # 通用组件
│   │   ├── stores/       # Pinia 状态管理
│   │   └── views/        # 页面视图
│   └── src-tauri/        # Rust 后端
│
├── server/                # Go 后端服务
│   ├── cmd/server/       # 入口
│   ├── internal/
│   │   ├── api/         # HTTP 接口
│   │   ├── config/      # 配置
│   │   ├── database/    # 数据库
│   │   ├── middleware/  # 中间件
│   │   ├── model/       # 数据模型
│   │   └── service/     # 业务逻辑
│   └── pkg/             # 公共包
│       ├── auth/        # 认证
│       └── storage/     # 存储
│
├── shared/               # 共享定义
│   └── proto/           # API 类型定义
│
├── refs/                 # 参考仓库
│   ├── Total-RP-3/
│   └── Total-RP-3-Extended/
│
└── docs/                 # 项目文档
```

## 代码规范

### Go 后端

```go
// 文件命名：小写下划线 user_service.go
// 包命名：小写单词 package service
// 接口命名：动词+er  type Reader interface

// 错误处理：始终检查错误
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// 注释：公开函数必须有注释
// CreateUser creates a new user in the database.
func CreateUser(u *User) error {}
```

### TypeScript 前端

```typescript
// 文件命名：PascalCase 组件，camelCase 工具
// UserProfile.vue, useAuth.ts, request.ts

// 类型定义：interface 优先于 type
interface User {
  id: number
  username: string
}

// 组件：使用 <script setup> 语法
<script setup lang="ts">
import { ref } from 'vue'
const count = ref(0)
</script>
```

### UI 规范

- **弹窗**: 使用内置弹窗组件 `RDialog`，不要使用浏览器原生 `alert`/`confirm`
- **消息提示**: 使用内置 `RToast` 组件
- **确认操作**: 危险操作（删除、解散等）必须使用确认弹窗

## 文件编辑规范

### JSON 文件编辑

**⚠️ 关键注意事项**：编辑 JSON 文件时必须使用标准 ASCII 引号，避免使用 Unicode 曲引号。

**常见问题**：

使用中文输入法编辑 JSON 文件时，输入法可能自动将引号转换为 Unicode 曲引号（""），导致 JSON 解析失败。

**错误示例**：
```json
{
  "changelog": "新增功能：自动替换"你"等代词"
}
```
上面的 `"你"` 使用了中文曲引号（U+201C 和 U+201D），会导致 JSON 解析错误。

**正确示例**：
```json
{
  "changelog": "新增功能：自动替换你等代词"
}
```
或者使用转义：
```json
{
  "changelog": "新增功能：自动替换\"你\"等代词"
}
```

**最佳实践**：

1. **编辑前检查输入法**：编辑 JSON 文件前切换到英文输入法
2. **使用专业编辑器**：使用支持 JSON 语法检查的编辑器（VS Code、Sublime Text 等）
3. **编辑后验证**：使用 `jq` 或在线工具验证 JSON 格式
   ```bash
   # 验证 JSON 文件格式
   jq . manifest.json
   ```
4. **修改后测试**：修改配置文件后立即测试相关 API 是否正常工作

**影响范围**：

以下文件编辑时需要特别注意：
- `server/storage/addons/RPBox_Addon/manifest.json` - 插件更新配置
- `client/src-tauri/tauri.conf.json` - Tauri 配置
- `package.json` - NPM 配置
- 所有 `.json` 配置文件

## API 规范

```
基础路径: /api/v1
认证: Bearer Token (JWT)
请求头: Authorization: Bearer <token>

响应格式: { "code": 0, "message": "success", "data": {} }
错误响应: { "error": "error message" }
```

## 图片缓存机制

- 列表/卡片接口不返回 base64 图片，统一使用 `/api/v1/images/:type/:id` 获取。
- 前端通过 `getImageUrl` 拼接 `w`/`q`/`v`/`cv`，`v` 来自 `*_updated_at`（没有时用 `updated_at`）。
- 后端在图片变更时必须更新对应字段：`preview_image_updated_at`/`cover_image_updated_at`/`banner_updated_at`。
- 图片接口需支持 ETag；带 `v` 的请求返回长缓存（immutable），不带 `v` 的请求使用短缓存。
- 清除缓存只通过提升 `cv`（客户端 cache version）实现，不做通用 API 缓存。

## Git 规范

```bash
# 分支命名
main          # 主分支
feat/xxx      # 功能分支
fix/xxx       # 修复分支

# 提交信息
feat: 添加用户登录功能
fix: 修复人物卡同步失败
docs: 更新文档
```

## TRP3 数据结构

```
WTF/Account/{账号}/SavedVariables/
├── TRP3_Profiles.lua      # 人物卡配置
├── TRP3_Characters.lua    # 角色绑定
├── TRP3_Companions.lua    # 伙伴数据
├── TRP3_Tools_DB.lua      # 道具数据库 (Extended)
└── TRP3_Extended_ImpExport.lua  # 导入导出
```

## 开发命令

```bash
# 客户端
cd client && npm install && npm run tauri dev

# 服务端
cd server && go mod tidy && go run cmd/server/main.go

# 一键启动
.\dev.bat  或  .\dev.ps1
```

## 用户角色管理

### 角色层级

| 角色 | 权限 | 设置方式 |
|------|------|----------|
| `user` | 普通用户，默认角色 | 注册时自动分配 |
| `moderator` | 版主，可审核帖子/道具 | 管理员通过 API 设置 |
| `admin` | 超级管理员，最高权限 | 仅通过后台脚本设置 |

### 设置超级管理员

超级管理员只能通过后台脚本设置，不能通过 API 设置：

```bash
cd server
go run cmd/setadmin/main.go <用户名>

# 示例
go run cmd/setadmin/main.go admin
```

### 管理员 API

管理员可通过 API 管理版主（不能设置 admin 角色）：

```
GET  /api/v1/admin/users           # 获取用户列表
PUT  /api/v1/admin/users/:id/role  # 设置用户角色 (仅 user/moderator)
```

## 客户端自动更新

### 配置文件

更新配置位于 `client/src-tauri/tauri.conf.json`：

```json
"plugins": {
  "updater": {
    "endpoints": ["https://api.rpbox.app/api/v1/updater/{{target}}/{{arch}}/{{current_version}}"],
    "pubkey": "公钥内容",
    "dangerousInsecureTransportProtocol": true  // 仅开发环境需要
  }
}
```

### ⚠️ 重要注意事项

**HTTPS 强制要求**：Tauri updater 在 release 模式下默认要求 HTTPS 协议。如果使用 HTTP 端点，必须添加 `dangerousInsecureTransportProtocol: true`，否则应用启动时会崩溃：

```
error: The configured updater endpoint must use a secure protocol like `https`.
```

**生产环境**：正式发布时应使用 HTTPS 端点并移除 `dangerousInsecureTransportProtocol` 配置。

### 构建和签名

```bash
# 生成签名密钥（仅首次）
cd client
npx tauri signer generate -w .tauri/rpbox.key

# 构建并签名
npx tauri signer sign -k "密钥内容" -p "密码" "安装包路径"
```

### 服务端配置

`server/config.yaml`：

```yaml
updater:
  latest_version: "0.2.0"
  base_url: "http://localhost:8080/releases"
  release_notes: "更新说明"
```

更新包放置于 `server/releases/{version}/` 目录。

### 故障排查指南

#### 问题1：客户端检测不到更新

**症状**：点击"检查更新"后显示"当前已是最新版本"，但实际上服务器有新版本。

**可能原因和解决方案**：

1. **版本号相同**
   - 检查：客户端版本号 = 服务器 `latest_version`
   - 解决：确保客户端版本号小于服务器配置的版本号
   - 开发模式：`tauri.conf.json` 中的版本号会被使用

2. **base_url 配置错误或缺失**
   - 检查：`server/config.yaml` 中是否配置了 `base_url`
   - 问题：如果未配置，会使用默认值 `https://api.rpbox.app/releases`（可能无法解析）
   - 解决：在 `config.yaml` 中添加：
   ```yaml
   updater:
     base_url: "https://your-domain.com/releases"
   ```
   - 验证：`curl https://your-domain.com/releases/0.1.6/RPBox_0.1.6_x64-setup.exe` 应该能访问

#### 问题2：检查更新失败，提示日期格式错误

**症状**：点击"检查更新"后显示错误：`invalid value for 'pub_date': the 'separator' component could not be parsed`

**原因**：`pub_date` 格式不正确。Tauri updater 要求完整的 ISO 8601 格式（包含时间和时区）。

**错误配置**：
```yaml
updater:
  pub_date: "2026-01-17"  # ❌ 只有日期，缺少时间
```

**正确配置**：
```yaml
updater:
  pub_date: "2026-01-17T12:00:00Z"  # ✅ 完整的 ISO 8601 格式
```

**修复步骤**：
1. 编辑 `server/config.yaml`，将 `pub_date` 改为完整格式
2. 重启服务：`sudo supervisorctl restart rpbox`
3. 验证：`curl http://localhost:8081/api/v1/updater/windows/x86_64/0.1.0` 查看返回的 `pub_date` 格式

#### 问题3：GitHub Actions 部署文件到错误位置

**症状**：GitHub Actions 构建成功，但服务器上找不到更新包文件。

**原因**：`RELEASE_PATH` Secret 配置错误，文件上传到了错误的目录。

**检查方法**：
```bash
# 正确位置（应该在这里）
ls /home/devbox/RPBox/server/releases/0.1.6/

# 错误位置（如果在这里就是配置错了）
ls /home/devbox/RPBox/releases/0.1.6/
```

**修复步骤**：
1. 在 GitHub 仓库中检查 Secrets 配置：`Settings → Secrets and variables → Actions`
2. 确认 `RELEASE_PATH` = `/home/devbox/RPBox/server/releases`（注意是 `server/releases`）
3. 如果配置错误，修改后重新推送 tag 触发构建

#### 问题4：更新后客户端无法连接后端，看不到数据

**症状**：通过自动更新安装新版本后，客户端无法登录或看不到社区帖子等数据。

**原因**：GitHub Actions 构建时缺少 `VITE_API_BASE` 环境变量，导致客户端使用默认的 `localhost` 地址。

**检查方法**：
- 安装新版本后，尝试登录或访问社区功能
- 如果提示连接失败或看不到数据，说明API地址配置错误

**修复步骤**：
1. 编辑 `.github/workflows/release-client.yml`
2. 在 "Build Tauri app" 步骤的 `env` 中添加：
   ```yaml
   env:
     TAURI_SIGNING_PRIVATE_KEY: ${{ secrets.TAURI_SIGNING_PRIVATE_KEY }}
     TAURI_SIGNING_PRIVATE_KEY_PASSWORD: ${{ secrets.TAURI_SIGNING_PRIVATE_KEY_PASSWORD }}
     VITE_API_BASE: https://your-domain.com/api/v1  # 添加这一行
   ```
3. 提交并推送修改
4. 重新发布新版本

#### 调试方法

**在开发模式下测试更新功能**：

1. **修改版本号进行测试**：
   ```bash
   # 将 tauri.conf.json 中的版本号改为低于服务器的版本
   # 例如：服务器是 0.1.6，改为 0.1.5
   cd client
   npm run tauri dev
   ```

2. **查看详细日志**：
   - 打开浏览器开发者工具（F12）
   - 查看 Console 标签页
   - 所有更新相关的日志都有 `[Updater]` 前缀
   - 关键日志：
     - `[Updater] 开始检查更新...`
     - `[Updater] 当前配置的 endpoint: ...`
     - `[Updater] 检查结果: ...`
     - `[Updater] 发现新版本: ...` 或 `[Updater] 当前已是最新版本`

3. **手动测试 API**：
   ```bash
   # 测试更新检测 API
   curl https://your-domain.com/api/v1/updater/windows/x86_64/0.1.0

   # 测试更新包是否可访问
   curl -I https://your-domain.com/releases/0.1.6/RPBox_0.1.6_x64-setup.exe
   ```

**最佳实践**：

1. **发布新版本前的检查清单**：
   - ✅ 更新三个文件的版本号：`tauri.conf.json`, `Cargo.toml`, `package.json`
   - ✅ 确认 `server/config.yaml` 中的 `base_url` 配置正确
   - ✅ 确认 `pub_date` 使用完整的 ISO 8601 格式
   - ✅ 确认 GitHub Secrets 中的 `RELEASE_PATH` 正确
   - ✅ 在开发模式下测试更新检测功能

2. **服务器配置模板**：
   ```yaml
   updater:
     latest_version: "0.1.6"
     base_url: "https://your-domain.com/releases"
     release_notes: "版本更新说明"
     pub_date: "2026-01-17T12:00:00Z"
   ```

## 插件自动更新

### 更新流程

客户端会自动检测 RPBox_Addon 插件的新版本，并提供一键安装功能。

### 发布新版本插件（CI）

只需要推送 tag，CI 会自动完成版本号更新、打包、上传和 manifest 更新。

**步骤 1：准备发布说明（可选）**

新增 `addon/release-notes/<version>.txt`，内容为本次更新说明。

**步骤 2：推送 tag**

```bash
git tag addon-v1.0.7
git push origin addon-v1.0.7
```

**CI 自动完成**

- 更新 `RPBox_Addon.toc` 的 `## Version`
- 打包并上传 `versions/<version>.zip` 与 `latest.zip`
- 更新服务器 `manifest.json`
- 校验包内版本号与 tag 一致，失败会中断发布

### 重要说明

- **覆盖安装**：插件安装采用覆盖模式，不会删除旧文件，避免文件锁定问题
- **版本检测**：客户端通过读取 `.toc` 文件的 `## Version:` 字段来检测版本
- **存储路径**：插件存储在 `server/storage/addons/RPBox_Addon/` 目录
- **下载优先级**：服务端优先使用 `versions/<version>.zip`，只有在版本包缺失时回退到 `latest.zip`
- **manifest 维护**：`server/storage/addons/RPBox_Addon/manifest.json` 作为仓库参考，生产环境由 CI 自动更新

## CI/CD 流程

### 自动化流程

| 流程 | 触发条件 | 配置文件 |
|------|----------|----------|
| CI 构建测试 | push 到 main/master | `.github/workflows/ci.yml` |
| 客户端发布 | 推送 `v*` tag | `.github/workflows/release-client.yml` |
| 插件发布 | 推送 `addon-v*` tag | `.github/workflows/release-addon.yml` |

### 发布命令

```bash
# 发布客户端 v0.2.0
git tag v0.2.0 && git push --tags

# 发布插件 v1.1.0
git tag addon-v1.1.0 && git push --tags
```

### 手动发布脚本

```powershell
# 客户端发布
.\scripts\release.ps1 -Version "0.2.0" -Notes "更新说明"

# 插件发布
.\scripts\release-addon.ps1 -Version "1.1.0" -Changelog "更新说明"
```

> 插件发布脚本仅用于紧急手动发布，正常发版请使用 tag 触发 CI。

### GitHub Secrets 配置

在仓库 **Settings → Secrets and variables → Actions** 中配置：

| Secret | 用途 |
|--------|------|
| `TAURI_SIGNING_PRIVATE_KEY` | Tauri 签名私钥 |
| `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` | 签名密钥密码 |
| `SSH_PRIVATE_KEY` | 部署用 SSH 私钥 |
| `SERVER_HOST` | 服务器地址 |
| `SERVER_USER` | SSH 用户名 |
| `RELEASE_PATH` | 客户端发布目录 |
| `ADDON_PATH` | 插件发布目录 |

## PRD 文档

- PRD1: 项目介绍
- PRD2: 人物卡备份同步
- **PRD3: 剧情记录归档 ⭐ 核心功能**
- PRD4: 社区分享交流
- PRD5: 道具市场
- PRD6: 配套WoW插件 (RPBox_Addon)

## 任务卡系统

本项目使用分散式任务卡系统管理任务。

### 使用方式

- 任务卡按需创建，在需要的模块下建立 `tasks/` 目录
- 任务 ID 格式：`[模块前缀]-[4位序号]`（如 `SYNC-0001`）
- 新任务添加到 `TASK_LIST.md` 表格顶部

### 已有模块前缀

| 模块 | 前缀 | 路径 |
|------|------|------|
| 人物卡同步 | SYNC | `client/src/views/sync/` |
| 剧情归档 | STORY | `client/src/views/story/` |
| 社区分享 | COMM | `client/src/views/community/` |
| 道具市场 | ITEM | `client/src/views/market/` |
| 服务端 | SRV | `server/` |

### 状态与类型

- **状态**: `TODO` 待处理 | `WIP` 进行中 | `DONE` 已完成 | `CANCEL` 已取消
- **类型**: `DEV` 开发 | `FIX` 修复 | `OPS` 运维 | `DOC` 文档 | `REF` 重构 | `TEST` 测试
- **优先级**: `P0` 紧急 | `P1` 高 | `P2` 中 | `P3` 低
