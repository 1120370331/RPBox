# 贡献指南

感谢你对 RPBox 项目的关注！我们欢迎各种形式的贡献。

## 贡献方式

- 🐛 报告 Bug
- 💡 提出新功能建议
- 📝 改进文档
- 💻 提交代码
- 🌐 翻译

## 开发环境

### 前置要求

- Node.js 18+
- Go 1.21+
- PostgreSQL 14+
- Rust (Tauri 需要)

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/your-repo/RPBox.git
cd RPBox

# 客户端开发
cd client
npm install
npm run tauri dev

# 服务端开发
cd server
cp config.example.yaml config.yaml
go run cmd/server/main.go
```

## 代码规范

### Go 后端

- 文件命名：小写下划线 `user_service.go`
- 包命名：小写单词 `package service`
- 错误处理：始终检查错误并使用 `fmt.Errorf` 包装
- 注释：公开函数必须有注释

### TypeScript 前端

- 文件命名：PascalCase 组件，camelCase 工具
- 类型定义：优先使用 `interface`
- 组件：使用 `<script setup>` 语法

详细规范请参考 [CLAUDE.md](./CLAUDE.md)

## 提交规范

使用语义化提交信息：

```
feat: 添加用户登录功能
fix: 修复人物卡同步失败
docs: 更新文档
style: 代码格式化
refactor: 重构认证模块
test: 添加单元测试
chore: 更新依赖
```

## Pull Request 流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feat/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feat/amazing-feature`)
5. 创建 Pull Request

### PR 要求

- 清晰描述改动内容和原因
- 关联相关 Issue（如有）
- 确保代码通过测试
- 遵循项目代码规范

## 报告问题

提交 Issue 时请包含：

- 问题描述
- 复现步骤
- 预期行为
- 实际行为
- 环境信息（操作系统、版本等）
- 截图或日志（如有）

## 开源协议

贡献代码即表示你同意：

- 客户端/插件代码使用 MIT License
- 服务端代码使用 AGPL-3.0 License

## 行为准则

- 尊重所有贡献者
- 保持友好和专业
- 接受建设性批评
- 关注项目最佳利益

## 联系方式

- GitHub Issues: 技术问题和功能建议
- Discussions: 一般讨论和问答

感谢你的贡献！🎉
