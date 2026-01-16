# RPBox

> **打破封闭的 RP 生态，迈向开放、互联的新时代。**

魔兽世界 RP 社区的创作者们，长久以来在游戏内的沙盒中艰难探索。人物卡无法云端同步、剧情记录随风消散、优秀道具难以传播。

RPBox 不在沙盒内寻找漏洞，而是在沙盒之外构建基础设施——连接游戏内外，让数据自由流动，让创作不再孤立。

**这不仅是一个工具，更是 RP 生态走向开放与互联的基石。**

——Claude Opus 4.5

---

## 功能特性

- **人物卡云同步** - 跨设备备份和管理 TRP3 人物卡数据
- **剧情记录归档** - 自动采集和长期保存 RP 对话记录
- **社区分享平台** - 按公会/人物/剧情线归档，便捷检索
- **道具市场** - TRP3 Extended 道具分享和一键导入
- **AI 辅助** - 剧情生成、内容总结等智能功能
- **自动更新** - 客户端支持检测和安装新版本

## 项目结构

```
RPBox/
├── client/          # Tauri + Vue3 桌面客户端
├── server/          # Go 后端服务
├── shared/          # 共享协议定义
└── docs/            # 项目文档
```

## 环境要求

- Node.js >= 18
- Rust >= 1.70
- Go >= 1.21

## 快速开始

### 安装依赖

```bash
# 安装 Rust
winget install Rustlang.Rustup

# 安装 Go
winget install GoLang.Go

# 重启终端后验证
rustc --version
go version
```

### 启动开发

```bash
# 客户端
cd client
npm install
npm run tauri dev

# 服务端
cd server
go mod tidy
go run cmd/server/main.go
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 桌面客户端 | Tauri 2.0 + Vue 3 + TypeScript |
| 后端服务 | Go + Gin |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 搜索 | MeiliSearch |

## License

MIT

## 自动更新注意事项

Tauri updater 在 release 模式下**强制要求 HTTPS**。本地开发测试时需在 `tauri.conf.json` 中添加：

```json
"dangerousInsecureTransportProtocol": true
```

否则使用 HTTP 端点的应用将无法启动。详见 `CLAUDE.md` 中的完整说明。
