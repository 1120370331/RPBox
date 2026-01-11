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

## API 规范

```
基础路径: /api/v1
认证: Bearer Token (JWT)
请求头: Authorization: Bearer <token>

响应格式: { "code": 0, "message": "success", "data": {} }
错误响应: { "error": "error message" }
```

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

## PRD 文档

- PRD1: 项目介绍
- PRD2: 人物卡备份同步
- PRD3: 剧情记录归档
- PRD4: 社区分享交流
- PRD5: 道具市场
- PRD6: 配套WoW插件 (RPBox_Addon)
