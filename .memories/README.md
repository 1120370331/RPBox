# .memories - RPBox 项目记忆系统

> 为 AI 助手和开发者提供结构化的项目知识库

## 目录结构

```
.memories/
├── README.md           # 本文件 - 记忆系统说明
├── modules/            # 模块文档
│   └── INDEX.md       # 模块索引（入口）
├── templates/          # 文档模板
│   └── module/
│       ├── README.md              # 模块导航模板
│       ├── PRD.md                 # 产品需求模板
│       └── FUNCTION.template.md   # 功能文档模板
└── scripts/            # 工具脚本
    ├── memories-lookup.sh   # Linux/Mac 速查
    └── memories-lookup.cmd  # Windows 速查
```

## 使用指南

### 对于 AI 助手

当需要了解项目某个模块时：
1. 首先查看 `modules/INDEX.md` 获取模块列表
2. 进入对应模块目录查看详细文档
3. 功能实现细节在 `functions/` 子目录

### 对于开发者

1. **新增模块**: 复制 `templates/module/` 到 `modules/{模块名}/`
2. **新增功能**: 复制 `FUNCTION.template.md` 并填写
3. **速查命令**: 运行 `scripts/memories-lookup.cmd`

## 与 CLAUDE.md 的关系

| 文件 | 用途 |
|------|------|
| `CLAUDE.md` | 项目概览、技术栈、代码规范 |
| `.memories/` | 详细的模块文档、功能实现、决策记录 |

## 命名规范

- 目录名：小写，连字符分隔 `user-auth`
- 文档名：大写，下划线分隔 `USER_LOGIN.md`
- 模板文件：`.template.md` 后缀
