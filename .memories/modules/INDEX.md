# RPBox 模块索引

> 快速定位项目各功能模块的文档入口

## 模块列表

| 模块 | 状态 | 说明 | 文档 |
|------|------|------|------|
| character-sync | 🚧 开发中 | 人物卡备份同步 | [PRD2](../../PRD2-人物卡备份同步.md) |
| story-archive | 📋 规划中 | 剧情记录归档 | [PRD3](../../PRD3-剧情记录归档.md) |
| community | 📋 规划中 | 社区分享交流 | PRD4 |
| item-market | 📋 规划中 | TRP3道具市场 | PRD5 |
| wow-addon | 📋 规划中 | 配套WoW插件 | PRD6 |

## 状态说明

- ✅ 已完成
- 🚧 开发中
- 📋 规划中
- ⏸️ 暂停

## 快速导航

### 客户端 (client/)
- `src/views/` - 页面视图
- `src/components/` - 通用组件
- `src/stores/` - Pinia 状态管理
- `src/api/` - API 请求

### 服务端 (server/)
- `internal/api/` - HTTP 接口
- `internal/service/` - 业务逻辑
- `internal/model/` - 数据模型

## 添加新模块

1. 在本文件添加模块条目
2. 创建 `modules/{模块名}/` 目录
3. 从 `templates/module/` 复制模板
