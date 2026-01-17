# RPBox

> 打破封闭的 RP 生态，迈向开放、互联的新时代。

RPBox 是一个服务于魔兽世界 RP 玩家的工具箱，提供人物卡备份同步、剧情记录归档、社区分享交流等功能。

## 功能特性

- 📦 **人物卡备份同步** - TotalRP3 人物卡跨设备云端同步
- 📝 **剧情记录归档** - 自动记录并归档 RP 聊天记录
- 🌐 **社区分享交流** - 分享人物卡、剧情和创作
- 🛒 **道具市场** - TRP3 Extended 道具交易平台
- 🤖 **AI 辅助创作** - 智能辅助人物卡和剧情创作

## 技术栈

- **客户端**: Tauri 2.0 + Vue 3 + TypeScript
- **服务端**: Go + Gin + PostgreSQL
- **插件**: Lua (WoW Addon)

## 快速开始

### 客户端开发

\`\`\`bash
cd client
npm install
npm run tauri dev
\`\`\`

### 服务端开发

\`\`\`bash
cd server
cp config.example.yaml config.yaml  # 编辑配置文件
go run cmd/server/main.go
\`\`\`

详细文档请参考 [CLAUDE.md](./CLAUDE.md)

## 开源协议

本项目采用分层开源策略：

- **客户端** (\`client/\`) - [MIT License](./client/LICENSE)
- **服务端** (\`server/\`) - [AGPL-3.0 License](./server/LICENSE)
- **插件** (\`addon/\`) - [MIT License](./addon/LICENSE)

### 为什么使用不同的协议？

- **MIT** (客户端/插件) - 最大化开放性，鼓励社区贡献和二次开发
- **AGPL-3.0** (服务端) - 保护服务端代码，要求修改后的网络服务也必须开源

## 贡献指南

欢迎贡献代码、报告问题或提出建议！请查看 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解详情。

## 联系方式

- 问题反馈: [GitHub Issues](https://github.com/your-repo/RPBox/issues)
- 项目文档: [CLAUDE.md](./CLAUDE.md)

## 致谢

感谢 [Total RP 3](https://github.com/Total-RP/Total-RP-3) 项目为 RP 社区做出的贡献。

---

**RPBox** - 让 RP 创作更自由 ✨
