# PRD3: RP剧情记录归档

> **⭐ 核心功能 - 本模块是 RPBox 的核心差异化功能，优先级最高**

## 1. 概述

### 1.1 背景
魔兽世界的RP活动包含大量玩家创作的文字对话和场景描述，这些内容是玩家的心血结晶。然而：
- 游戏内聊天记录有限，重启后丢失
- 现有聊天记录插件（如Prat、WIM）仅做短期存储
- 缺乏按剧情线、活动归档的能力

### 1.2 目标
提供RP剧情的长期归档方案，支持：
- 自动采集游戏内RP对话
- 按活动/剧情线组织归档
- 支持后期编辑和整理
- AI辅助生成剧情总结

## 2. 用户故事

| 编号 | 故事 | 优先级 |
|------|------|--------|
| US-01 | 我想自动保存RP活动中的所有对话 | P0 |
| US-02 | 我想按活动/剧情线整理我的RP记录 | P0 |
| US-03 | 我想编辑和补充剧情记录 | P1 |
| US-04 | 我想用AI生成剧情摘要 | P2 |

## 3. 功能需求

### 3.1 聊天采集 (P0)

> **前置依赖**：配套 WoW 插件，详见 [PRD6-配套WoW插件.md](PRD6-配套WoW插件.md)

**插件自动安装：**

客户端首次进入"剧情故事"标签时，检测并引导安装插件。

```
┌─ 安装 RPBox 插件 ────────────────────────────┐
│                                              │
│  📦 需要安装配套插件才能使用剧情记录功能       │
│                                              │
│  WoW 安装目录:                               │
│  [C:\Program Files\World of Warcraft ▼] [浏览]│
│                                              │
│  检测到版本: _retail_ / _classic_            │
│                                              │
│  [安装插件]                    [稍后再说]     │
└──────────────────────────────────────────────┘
```

**安装流程：**
1. 用户选择 WoW 安装目录（或自动检测）
2. 客户端将插件复制到 `Interface/AddOns/RPBox_Addon/`
3. 提示用户重启游戏或 `/reload`

**插件更新：**
- 客户端启动时检查插件版本
- 有新版本时提示更新
- 自动覆盖旧文件

**版本检查流程：**
```
客户端启动
    ↓
GET /api/v1/addon/latest → 获取最新版本号
    ↓
对比本地已安装版本（读取 .toc 文件）
    ↓
有新版本 → 提示更新
    ↓
用户确认 → 下载并覆盖安装
    ↓
提示用户 /reload
```

**相关 API：**
| 接口 | 说明 |
|------|------|
| `GET /api/v1/addon/manifest` | 获取版本清单 |
| `GET /api/v1/addon/latest` | 获取最新版本号 |
| `GET /api/v1/addon/download/{version}` | 下载指定版本 |

**客户端功能：**
- 读取并解析 `RPBox_ChatLog` SavedVariables
- 导入到本地数据库
- 支持手动/自动同步

**多账号支持：**

一个用户可能有多个 WoW 账号，数据分布在不同子目录：

```
WTF/Account/
├── ACCOUNT1/SavedVariables/RPBox_ChatLog.lua
├── ACCOUNT2/SavedVariables/RPBox_ChatLog.lua
└── ACCOUNT3/SavedVariables/RPBox_ChatLog.lua
```

**客户端处理：**
- 扫描 WoW 目录下所有账号子目录
- 分别监控每个账号的 SavedVariables 变化
- 所有账号数据统一归到当前登录用户
- 待归档池中标记数据来源账号

**账号选择界面：**
```
┌─ 选择要同步的账号 ─────────────────────────┐
│                                            │
│  ☑ ACCOUNT1  (最后更新: 2024-01-11 22:00)  │
│  ☑ ACCOUNT2  (最后更新: 2024-01-10 20:30)  │
│  ☐ ACCOUNT3  (无新数据)                    │
│                                            │
│  [全选]  [同步选中账号]                     │
└────────────────────────────────────────────┘
```

**数据同步时机：**

WoW 插件数据存储在 SavedVariables 文件中，该文件仅在以下时机写入：
- 退出游戏时
- `/reload` 重载 UI 时
- 登出角色时

客户端解析策略：
- **启动时自动检测**：客户端启动时检查是否有新数据
- **手动触发同步**：用户点击"同步"按钮时读取

**用户操作流程：**
1. 游戏内 RP 完毕
2. 游戏内执行 `/reload` 或退出游戏
3. 打开 RPBox 客户端，点击"同步"或客户端自动检测新数据

### 3.2 数据生命周期 (P0)

采用类似 Git 的三区设计，分离"同步"、"归档"、"清理"操作：

```
┌─────────────┐    同步    ┌─────────────┐    归档    ┌─────────────┐
│  插件端     │ ────────> │  待归档池   │ ────────> │   剧情      │
│ (工作区)    │           │ (暂存区)    │           │ (已提交)    │
└─────────────┘           └─────────────┘           └─────────────┘
      ↑                         │
      │        清理             │ 可部分归档
      └─────────────────────────┘ 可暂不处理
```

**三个独立操作：**

| 操作 | 含义 | 说明 |
|------|------|------|
| 同步 | 插件 → 待归档池 | 只导入，不删除插件端数据 |
| 归档 | 待归档池 → 剧情 | 可选择部分内容，可关联多个剧情 |
| 清理 | 删除插件端旧数据 | 用户手动触发，确认数据已安全 |

**设计理由：**
- RP 活动可能跨多天/多次游戏会话
- 中途 reload/退出不代表活动结束
- 用户需要灵活整理，不急于立即归档

### 3.3 待归档池 (P0)

**待归档池界面：**
```
┌─────────────────────────────────────────────────────┐
│  待归档池 (156条)                    [同步] [清理]  │
├─────────────────────────────────────────────────────┤
│  ▼ 2024-01-11 (80条)                       [☐ 全选] │
│    ▼ 20:00 - 20:59  (32条)                 [☐ 全选] │
│      ☑ [艾琳] 说: 今晚的月色真美...                  │
│      ☑ [索林·铁炉] 表情: 点燃了篝火                  │
│    ▼ 21:00 - 21:59  (48条)                 [☐ 全选] │
│      ☑ [神秘旅人] 说: 我有一个秘密...                │
│                                                     │
│  ▼ 2024-01-10 (76条)                       [☐ 全选] │
│    ...                                              │
├─────────────────────────────────────────────────────┤
│  归档到: [选择剧情 ▼]  [+ 新建剧情]                  │
│  已选 28 条对话              [归档选中内容]          │
└─────────────────────────────────────────────────────┘
```

**功能点：**
- 按日期 → 小时两级折叠
- 勾选框批量选择
- 支持按小时/按天一键全选
- 可多次部分归档到不同剧情
- 已归档内容标记但不删除（可重复归档到其他剧情）

### 3.4 剧情管理 (P0)

**剧情列表：**
- 创建/编辑/删除剧情
- 剧情可包含多个归档片段
- 支持从待归档池追加内容
- 已归档内容可移动到其他剧情

### 3.5 内容编辑 (P1)
- 富文本编辑器
- 添加场景描述、旁白
- 插入图片/截图
- 标记重要对话

### 3.6 分享与回放 (P1)

**剧情回放链接：**

生成可分享的链接，任何人点开即可查看剧情回放。

```
https://rpbox.cc/story/{story_id}
```

**回放页面功能：**
- 无需登录即可访问
- 支持静态浏览（滚动查看全部对话）
- 支持播放模式（逐条展示，模拟 RP 过程）
- 显示角色头像图标
- 可调节播放速度

**回放页界面：**
```
┌─ 暴风城酒馆奇遇 ─────────────────────────────┐
│  作者: 艾琳  |  2024-01-11  |  参与: 3人      │
├──────────────────────────────────────────────┤
│                                              │
│  [图标] 艾琳·风语者                           │
│     今晚的月色真美...                         │
│                                    ← 逐条显示 │
│  [图标] 索林·铁炉  *表情*                     │
│     点燃了篝火                                │
│                                              │
├──────────────────────────────────────────────┤
│  [|◀] [▶ 播放] [▶|]     速度: [1x ▼]         │
└──────────────────────────────────────────────┘
```

**分享方式：**
- 复制链接（用于 QQ 群等外部分享）
- 嵌入战报帖子（社区内引用）

### 3.7 AI摘要 (P2)

**功能：**
- 自动生成剧情摘要
- 提取关键人物和事件
- 生成时间线

**AI 服务配置：**

| 配置项 | 值 |
|--------|-----|
| 服务商 | whatai.cc |
| 模型 | gemini-3-flash-preview-nothinking |
| 接口 | OpenAI 兼容格式 |

**服务端调用示例：**
```go
// POST https://whatai.cc/v1/chat/completions
type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

func GenerateSummary(storyContent string) string {
    req := ChatRequest{
        Model: "gemini-3-flash-preview-nothinking",
        Messages: []Message{
            {
                Role:    "system",
                Content: "你是一个RP剧情整理助手，请根据以下对话记录生成简洁的剧情摘要，提取关键人物和事件。",
            },
            {
                Role:    "user",
                Content: storyContent,
            },
        },
    }
    // 发送请求，返回摘要
}
```

**请求头：**
```
Authorization: Bearer {API_KEY}
Content-Type: application/json
```

**响应格式：**
```json
{
    "id": "chatcmpl-123",
    "object": "chat.completion",
    "created": 1677652288,
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "摘要内容..."
            },
            "finish_reason": "stop"
        }
    ],
    "usage": {
        "prompt_tokens": 100,
        "completion_tokens": 50,
        "total_tokens": 150
    }
}
```

### 3.8 图标服务 (P0)

TRP3 人物卡包含角色图标（如 `inv_misc_book_09`），需要在客户端正确渲染以增强代入感。

**方案：服务端代理缓存**

```
客户端请求图标 → 服务端
                  ↓
            检查本地缓存
                  ↓
        有 → 直接返回
        无 → 从 Wowhead 拉取
                  ↓
            永久存储到服务端
                  ↓
            返回客户端
```

**服务端 API：**
```
GET /api/v1/icons/{icon_name}
示例: GET /api/v1/icons/inv_misc_book_09
返回: 图片二进制 (image/jpeg)
```

**服务端逻辑：**
```go
func GetIcon(iconName string) {
    // 1. 检查本地缓存
    cachePath := fmt.Sprintf("./icons/%s.jpg", iconName)
    if exists(cachePath) {
        return readFile(cachePath)
    }

    // 2. 从 Wowhead 拉取
    url := fmt.Sprintf("https://wow.zamimg.com/images/wow/icons/large/%s.jpg", iconName)
    data := fetch(url)

    // 3. 永久存储
    saveFile(cachePath, data)

    return data
}
```

**客户端渲染：**
```vue
<img :src="`${API_BASE}/icons/${sender.trp3.IC}`" />
```

**优点：**
- 服务端可部署在能访问 Wowhead 的位置
- 一次拉取，永久缓存
- 国内用户直接从服务端获取，速度快

## 4. 数据结构

```typescript
// 待归档池中的聊天记录
interface StagedChatLog {
  id: string;
  timestamp: Date;
  channel: string;           // SAY / EMOTE / PARTY 等
  sender: {
    gameID: string;          // 玩家名-服务器
    trp3?: {
      FN: string;            // 名
      LN: string;            // 姓
      TI: string;            // 头衔
      IC: string;            // 图标
    };
  };
  content: string;
  archivedTo: string[];      // 已归档到哪些剧情（可多个）
}

// 剧情
interface Story {
  id: string;
  userId: number;
  title: string;
  description: string;
  participants: string[];    // 参与角色
  tags: string[];
  startTime: Date;
  endTime: Date;
  status: 'draft' | 'published';
}

// 剧情条目
interface StoryEntry {
  id: string;
  storyId: string;
  sourceId?: string;         // 来源聊天记录ID（可追溯）
  type: 'dialogue' | 'narration' | 'image';
  speaker?: string;
  content: string;
  timestamp: Date;
}
```

## 5. 里程碑

| 阶段 | 功能 |
|------|------|
| M1 | 聊天日志解析 + 待归档池 |
| M2 | 剧情创建 + 归档管理 |
| M3 | 插件端清理联动 |
| M4 | 分享链接 + 回放页面 |
| M5 | 图标服务 |
| M6 | 富文本编辑 |
| M7 | AI摘要生成 |

## 6. 工作步骤

### 阶段一：基础设施（M1 前置）

**插件端：**
1. [ ] 创建 RPBox_Addon 插件骨架（.toc + Core.lua）
2. [ ] 实现聊天事件监听（SAY/EMOTE/PARTY等）
3. [ ] 实现 TRP3 信息获取
4. [ ] 实现过滤规则（TRP3 → 白名单）
5. [ ] 实现白名单自动添加（选中2秒）
6. [ ] 实现黑名单同步（WoW + TRP3）
7. [ ] 实现 SavedVariables 数据结构

**服务端：**
8. [ ] 插件版本管理 API（manifest/latest/download）
9. [ ] 插件文件存储结构

**客户端：**
10. [ ] WoW 目录选择/自动检测
11. [ ] 插件安装功能
12. [ ] 插件版本检查与更新

### 阶段二：数据同步（M1）

**插件端：**
1. [ ] 实现 RPBox_Sync 双向状态文件
2. [ ] 实现记录上限检查（9k警告/10k上限）

**客户端：**
3. [ ] SavedVariables 解析器（Lua → JSON）
4. [ ] 多账号目录扫描
5. [ ] 待归档池本地数据库设计
6. [ ] 待归档池 UI（日期→小时折叠）
7. [ ] 同步功能（插件 → 待归档池）

### 阶段三：剧情管理（M2）

**客户端：**
1. [ ] 剧情 CRUD API 对接
2. [ ] 剧情列表 UI
3. [ ] 归档操作（待归档池 → 剧情）
4. [ ] 支持部分归档、多剧情归档

**服务端：**
5. [ ] 剧情表设计（Story/StoryEntry）
6. [ ] 剧情 CRUD API

### 阶段四：清理联动（M3）

**客户端：**
1. [ ] 清理确认 UI
2. [ ] 写回 RPBox_Sync.client.clearedBefore

**插件端：**
3. [ ] 启动时读取 client 状态并清理

### 阶段五：分享与回放（M4）

**服务端：**
1. [ ] 剧情公开/分享状态字段
2. [ ] 公开剧情 API（无需登录）

**前端（Web）：**
3. [ ] 回放页面（/story/{id}）
4. [ ] 静态浏览模式
5. [ ] 播放模式（逐条展示）
6. [ ] 播放速度控制

### 阶段六：图标服务（M5）

**服务端：**
1. [ ] 图标缓存目录结构
2. [ ] 图标 API（/icons/{name}）
3. [ ] Wowhead 拉取逻辑

**客户端/前端：**
4. [ ] 图标渲染组件

### 阶段七：富文本编辑（M6）

**客户端：**
1. [ ] 富文本编辑器集成
2. [ ] 添加旁白/场景描述
3. [ ] 图片上传与插入

### 阶段八：AI摘要（M7）

**服务端：**
1. [ ] AI 服务配置（whatai.cc）
2. [ ] 摘要生成 API

**客户端：**
3. [ ] 摘要生成 UI
4. [ ] 摘要展示与编辑
