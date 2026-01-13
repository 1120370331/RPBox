# PRD6: 配套WoW插件 - RPBox_Addon

## 1. 概述

RPBox 桌面客户端需要配套的魔兽世界插件来实现部分核心功能。本文档统一描述插件需求。

### 1.1 插件定位
- 轻量级辅助插件
- 依赖 TotalRP3 插件
- 数据通过 SavedVariables 与桌面客户端交互

### 1.2 核心功能
| 模块 | 功能 | 关联PRD |
|------|------|---------|
| ChatLogger | RP聊天记录采集 | PRD3 |
| ProfileExport | 人物卡快速导出 | PRD2 |
| ItemSync | 道具数据标记 | PRD5 |

## 2. 模块详情

### 2.1 ChatLogger - 聊天记录采集

**功能描述：**
监听RP相关聊天频道，关联TRP3人物卡信息，保存结构化数据。

**TRP3 数据获取方案：**

```lua
-- 核心API
-- 1. 获取单位的角色ID
local unitID = Utils.str.getUnitID("target")  -- 返回 "玩家名-服务器"

-- 2. 获取角色绑定的profileID
local character = TRP3_API.register.getUnitIDCharacter(unitID)
local profileID = character and character.profileID

-- 3. 获取profile数据
local profile = TRP3_API.register.getProfile(profileID)
local characteristics = profile.player.characteristics

-- 4. 提取RP角色信息
local rpName = {
    FN = characteristics.FN,  -- 名 (First Name)
    LN = characteristics.LN,  -- 姓 (Last Name)
    TI = characteristics.TI,  -- 头衔 (Title)
    RA = characteristics.RA,  -- 种族 (Race)
    CL = characteristics.CL,  -- 职业 (Class)
    IC = characteristics.IC,  -- 图标 (Icon)
}
```

**聊天事件监听：**
```lua
-- 监听聊天消息
local function OnChatMessage(self, event, msg, sender, ...)
    local unitID = sender  -- "玩家名-服务器"
    local trp3Info = GetTRP3Info(unitID)  -- 获取TRP3角色卡

    SaveChatLog({
        timestamp = time(),
        channel = event,  -- CHAT_MSG_SAY / CHAT_MSG_EMOTE 等
        sender = {
            gameID = unitID,
            trp3 = trp3Info,
        },
        content = msg,
    })
end

ChatFrame_AddMessageEventFilter("CHAT_MSG_SAY", OnChatMessage)
ChatFrame_AddMessageEventFilter("CHAT_MSG_YELL", OnChatMessage)
ChatFrame_AddMessageEventFilter("CHAT_MSG_EMOTE", OnChatMessage)
```

**监听频道：**
- SAY (说)
- YELL (喊)
- EMOTE (表情)
- PARTY (小队)
- RAID (团队)
- WHISPER (密语) - 可选
- TRP3自定义频道

**数据结构（优化版）：**

为减少数据冗余，采用分离存储策略：
- 聊天记录只存储关联字段
- TRP3 角色卡数据独立存储
- 支持同一玩家切换不同人物卡

**1. 消息标记类型：**
```lua
-- 标记类型
"P"  -- Player: 玩家消息，关联 TRP3 角色卡
"N"  -- NPC: 临时 NPC 消息（TRP3 | 语法）
"B"  -- Background: 旁白/背景描述
```

**2. 聊天记录（精简）：**
```lua
RPBox_ChatLog = {
    ["2024-01-11"] = {
        ["20"] = {  -- 20:00-20:59
            -- 玩家消息
            {
                t = 1704960000,           -- timestamp
                c = "SAY",                -- channel (简写)
                m = "今晚的月色真美...",   -- message
                mk = "P",                 -- mark: Player
                s = "玩家名-服务器",       -- sender
                ref = "profileID_001",    -- 关联的角色卡ID
            },
            -- NPC 消息 (TRP3 | 语法)
            {
                t = 1704960060,
                c = "EMOTE",
                m = "你好",
                mk = "N",                 -- mark: NPC
                s = "玩家名-服务器",       -- 创建者
                ref = "profileID_001",    -- 创建者的角色卡
                npc = "恩佐斯",            -- NPC 名字
            },
            -- 旁白消息
            {
                t = 1704960120,
                c = "EMOTE",
                m = "山谷染成红色",
                mk = "B",                 -- mark: Background
                s = "玩家名-服务器",       -- 创建者
                ref = "profileID_001",    -- 创建者的角色卡
            },
        },
    },
}
```

**3. TRP3 角色卡缓存（完整数据）：**
```lua
RPBox_ProfileCache = {
    ["profileID_001"] = {
        -- 基本特征 (characteristics)
        v = 1,                        -- 版本
        FN = "艾琳",                   -- 名 (First Name)
        LN = "风语者",                 -- 姓 (Last Name)
        TI = "游荡的诗人",             -- 头衔 (Title)
        FT = "月神的使者",             -- 全称 (Full Title)
        RA = "暗夜精灵",               -- 种族 (Race)
        CL = "德鲁伊",                 -- 职业 (Class)
        AG = "10000",                 -- 年龄 (Age)
        EC = "银色",                   -- 眼睛颜色 (Eye Color)
        HE = "185cm",                 -- 身高 (Height)
        WE = "60kg",                  -- 体重 (Weight)
        BP = "达纳苏斯",               -- 出生地 (Birthplace)
        RE = "暴风城",                 -- 住所 (Residence)
        RS = "单身",                   -- 关系状态 (Relationship Status)
        IC = "inv_misc_book_09",      -- 图标 (Icon)
        CH = "3ba68d",                -- 自定义颜色 (Custom Hex)
        MI = {},                      -- 杂项信息 (Miscellaneous Info)
        PS = {},                      -- 心理特征 (Psycho traits)
    },
}
```

**4. 角色卡扩展字段（misc + about）：**
```lua
-- 继续 RPBox_ProfileCache["profileID_001"]
{
    -- 杂项 (misc)
    misc = {
        v = 1,
        PE = {},                      -- 第一印象 (Peek/First Glance)
        ST = {},                      -- 当前状态 (Status)
    },
    -- 关于 (about)
    about = {
        v = 1,
        TE = 1,                       -- 模板类型
        T1 = {},                      -- 模板1 (纯文本)
        T2 = {},                      -- 模板2 (分段)
        T3 = {                        -- 模板3 (结构化)
            PH = {},                  -- 外貌 (Physical)
            PS = {},                  -- 性格 (Personality)
            HI = {},                  -- 历史 (History)
        },
    },
}
```

**5. 频道简写映射：**
```lua
-- 频道简写
SAY     -- CHAT_MSG_SAY
YELL    -- CHAT_MSG_YELL
EMOTE   -- CHAT_MSG_EMOTE
PARTY   -- CHAT_MSG_PARTY / CHAT_MSG_PARTY_LEADER
RAID    -- CHAT_MSG_RAID / CHAT_MSG_RAID_LEADER
WHISPER -- CHAT_MSG_WHISPER / CHAT_MSG_WHISPER_INFORM
```

**记录过滤规则：**

| 条件 | 是否记录 |
|------|----------|
| 自己发送的消息 | ✅ 无条件记录 |
| 对方有 TRP3 信息 | ✅ 记录 |
| 对方无 TRP3 但在白名单 | ✅ 记录 |
| 对方无 TRP3 且不在白名单 | ❌ 不记录 |
| 对方在黑名单 | ❌ 不记录 |

```lua
local function ShouldRecord(sender, isFromSelf)
    -- 1. 自己 → 记录
    if isFromSelf then return true end
    -- 2. 黑名单 → 不记录
    if IsBlacklisted(sender) then return false end
    -- 3. 有 TRP3 → 记录（RP玩家标志）
    if GetCurrentTRP3Info(sender) then return true end
    -- 4. 白名单 → 记录（没TRP3但确实在RP的人）
    if RPBox_Config.whitelist[sender] then return true end
    -- 5. 其他 → 不记录
    return false
end
```

**TRP3 数据获取策略：**
- 每条消息实时获取 TRP3 信息，不缓存
- 原因：玩家可能在 RP 过程中临时更换人物卡
- 性能影响小：TRP3 数据本地已缓存，查询开销很小

**白名单管理：**

添加方式：
- 选中目标超过 2 秒自动添加，并提示"已加入记录白名单"
- 命令：`/rpbox whitelist add 玩家名-服务器`

移除方式：
- 拉黑时自动移除
- 命令：`/rpbox whitelist remove 玩家名-服务器`

```lua
-- 自动白名单：选中目标超过2秒
local targetTimer = nil

local function OnTargetChanged()
    if targetTimer then targetTimer:Cancel() end

    local unitID = GetUnitID("target")
    if not unitID or RPBox_Config.whitelist[unitID] then return end

    targetTimer = C_Timer.NewTimer(2, function()
        RPBox_Config.whitelist[unitID] = true
        print("|cFF00FF00[RPBox]|r " .. unitID .. " 已加入记录白名单")
    end)
end
```

**黑名单同步：**

黑名单来源（任一触发即生效）：
- WoW 原生拉黑列表
- TRP3 拉黑/屏蔽
- RPBox 命令：`/rpbox blacklist add 玩家名-服务器`

```lua
local function IsBlacklisted(unitID)
    -- RPBox 黑名单
    if RPBox_Config.blacklist[unitID] then return true end
    -- WoW 原生拉黑
    if C_FriendList.IsIgnored(unitID) then return true end
    -- TRP3 拉黑
    local relation = TRP3_API.register.relation.getRelation(unitID)
    if relation == TRP3_API.register.relation.NONE then return true end
    return false
end
```

### 2.2 ProfileExport - 人物卡快速导出

**功能描述：**
提供游戏内命令，快速导出当前角色或目标的TRP3人物卡数据。

**命令：**
```
/rpbox export        -- 导出自己的人物卡
/rpbox export target -- 导出目标的人物卡
/rpbox sync          -- 标记数据待同步
```

**数据结构：**
```lua
RPBox_ProfileExport = {
    lastExport = 1704960000,
    profiles = {
        ["角色名-服务器"] = {
            -- TRP3完整人物卡数据
        },
    },
}
```

### 2.3 ItemSync - 道具数据标记

**功能描述：**
标记玩家创建的TRP3 Extended道具，便于导出分享。

**命令：**
```
/rpbox item mark    -- 标记当前道具待分享
/rpbox item list    -- 列出已标记道具
```

### 2.4 游戏内 UI - 聊天回放

**功能描述：**
提供简单的游戏内界面，让玩家在 RP 过程中快速回顾聊天记录。

**打开方式：**
```
/rpbox log          -- 打开回放窗口
/rpbox log today    -- 只看今天
```

**界面设计：**
```
┌─ RPBox 聊天回放 ──────────────────── [×]─┐
│  [今天 ▼]  [全部频道 ▼]  [搜索...]       │
├──────────────────────────────────────────┤
│  20:32 [艾琳·风语者]                      │
│     今晚的月色真美...                     │
│                                          │
│  20:33 [索林] *表情*                      │
│     点燃了篝火                            │
│                                          │
│  20:35 [艾琳·风语者]                      │
│     你听说过银月城的传说吗                │
│  ...                                     │
├──────────────────────────────────────────┤
│  共 48 条记录                             │
└──────────────────────────────────────────┘
```

**功能点：**
- 按日期筛选（下拉菜单）
- 按频道筛选（下拉菜单）
- 关键词搜索
- 滚动浏览

### 2.5 记录上限管理

**存储限制：**
- 默认上限：10000 条记录
- 警告阈值：9000 条

**提示机制：**
```lua
local MAX_RECORDS = 10000
local WARN_THRESHOLD = 9000

local function CheckRecordLimit()
    local count = GetTotalRecordCount()
    if count >= WARN_THRESHOLD and not RPBox_Config.warnedThisSession then
        print("|cFFFFFF00[RPBox]|r 聊天记录已达 " .. count .. " 条")
        print("|cFFFFFF00[RPBox]|r 建议 /reload 后在客户端导出并清理")
        RPBox_Config.warnedThisSession = true
    end
end
```

**清理命令：**
```
/rpbox clear        -- 清理已导出的记录
/rpbox clear all    -- 清理全部（需二次确认）
```

## 3. 技术实现

### 3.1 依赖
- TotalRP3 (必需)
- TotalRP3 Extended (可选)

### 3.2 SavedVariables
```toc
## SavedVariables: RPBox_ChatLog, RPBox_ProfileExport, RPBox_Config, RPBox_Sync
```

### 3.3 插件与客户端双向状态同步

由于 WoW 插件无法直接与外部程序通信，采用双向状态文件实现同步。

**状态文件：**
```
WTF/Account/{账号}/SavedVariables/RPBox_Sync.lua
```

**数据结构：**
```lua
RPBox_Sync = {
    -- 插件写入（每次记录新消息后更新）
    addon = {
        lastUpdate = 1704960000,    -- 最后记录时间戳
        recordCount = 156,          -- 当前记录条数
        version = 1,                -- 数据格式版本
    },

    -- 客户端写入（同步/清理后更新）
    client = {
        lastSync = 1704955000,      -- 上次同步时间戳
        syncedCount = 120,          -- 已同步条数
        clearedBefore = 1704900000, -- 已清理到此时间之前
    },
}
```

**插件侧逻辑：**
```lua
-- 启动时读取客户端状态
local function OnAddonLoaded()
    local clientState = RPBox_Sync and RPBox_Sync.client
    if clientState and clientState.clearedBefore then
        -- 清理已被客户端处理的旧数据
        ClearRecordsBefore(clientState.clearedBefore)
    end
end

-- 记录新消息后更新状态
local function UpdateAddonState()
    RPBox_Sync.addon = {
        lastUpdate = time(),
        recordCount = GetTotalRecordCount(),
        version = 1,
    }
end
```

**客户端侧逻辑：**
```
1. 监控 RPBox_Sync.lua 文件修改时间
2. 读取并对比：addon.lastUpdate > client.lastSync → 有新数据
3. 同步完成后写入 client.lastSync = 当前时间
4. 清理完成后写入 client.clearedBefore = 清理截止时间
```

**交互流程图：**
```
┌─────────────┐                      ┌─────────────┐
│   WoW 插件   │                      │  RPBox 客户端 │
└──────┬──────┘                      └──────┬──────┘
       │                                    │
       │  记录聊天消息                        │
       │  更新 addon.lastUpdate              │
       │                                    │
       │         /reload 或退出游戏          │
       │ ─────────────────────────────────> │
       │         文件写入磁盘                 │
       │                                    │
       │                                    │  检测文件变化
       │                                    │  读取新数据
       │                                    │  更新 client.lastSync
       │                                    │
       │         下次启动游戏                 │
       │ <───────────────────────────────── │
       │         读取 client 状态            │
       │         清理已处理数据               │
       │                                    │
```

### 3.4 TRP3 API 调用
```lua
-- 获取人物卡数据
TRP3_API.profile.getData("player")
-- 监听聊天
ChatFrame_AddMessageEventFilter("CHAT_MSG_SAY", handler)
```

### 3.5 服务端插件版本管理

**存储结构：**
```
/storage/addons/
├── RPBox_Addon/
│   ├── latest/              # 最新版本文件
│   │   ├── RPBox_Addon.toc
│   │   ├── Core.lua
│   │   ├── ChatLogger.lua
│   │   └── ...
│   ├── versions/            # 历史版本归档
│   │   ├── 1.0.0.zip
│   │   ├── 1.0.1.zip
│   │   └── 1.1.0.zip
│   └── manifest.json        # 版本清单
```

**版本清单 manifest.json：**
```json
{
  "name": "RPBox_Addon",
  "latest": "1.1.0",
  "versions": [
    {
      "version": "1.1.0",
      "releaseDate": "2024-01-15",
      "minClientVersion": "1.0.0",
      "changelog": "新增游戏内回放UI",
      "downloadUrl": "/api/v1/addon/download/1.1.0"
    }
  ]
}
```

**服务端 API：**

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/addon/manifest` | GET | 获取完整版本清单 |
| `/api/v1/addon/latest` | GET | 获取最新版本号 |
| `/api/v1/addon/download/{version}` | GET | 下载指定版本 zip |

**API 响应示例：**

```go
// GET /api/v1/addon/latest
type LatestResponse struct {
    Version string `json:"version"`
    Url     string `json:"downloadUrl"`
}

// GET /api/v1/addon/manifest
type ManifestResponse struct {
    Name     string        `json:"name"`
    Latest   string        `json:"latest"`
    Versions []VersionInfo `json:"versions"`
}
```

## 4. 里程碑

| 阶段 | 功能 |
|------|------|
| M1 | ChatLogger 基础聊天记录 |
| M2 | TRP3 人物卡关联 + 过滤规则 |
| M3 | 白名单/黑名单机制 |
| M4 | 游戏内回放 UI |
| M5 | ProfileExport 导出 |
| M6 | ItemSync 道具标记 |

## 5. 工作步骤

### 阶段一：插件骨架（M1）

1. [ ] 创建插件目录结构
   ```
   RPBox_Addon/
   ├── RPBox_Addon.toc
   ├── Core.lua
   ├── ChatLogger.lua
   └── Locales/
   ```
2. [ ] 编写 .toc 文件（依赖 TRP3）
3. [ ] 实现插件加载框架
4. [ ] 实现斜杠命令 `/rpbox`

### 阶段二：聊天记录（M1）

1. [ ] 注册聊天事件监听
2. [ ] 实现 SaveChatLog 函数
3. [ ] 实现按日期→小时的数据结构
4. [ ] 测试各频道消息捕获

### 阶段三：TRP3 集成（M2）

1. [ ] 实现 GetCurrentTRP3Info 函数
2. [ ] 实现 ShouldRecord 过滤逻辑
3. [ ] 测试 TRP3 数据获取
4. [ ] 处理无 TRP3 数据的情况

### 阶段四：白名单/黑名单（M3）

1. [ ] 实现白名单数据结构
2. [ ] 实现选中目标2秒自动添加
3. [ ] 实现黑名单检查（WoW + TRP3 + RPBox）
4. [ ] 实现白名单/黑名单命令

### 阶段五：双向状态同步

1. [ ] 实现 RPBox_Sync 数据结构
2. [ ] 实现 addon 状态更新
3. [ ] 实现启动时读取 client 状态
4. [ ] 实现根据 clearedBefore 清理数据

### 阶段六：记录上限管理

1. [ ] 实现记录计数
2. [ ] 实现 9000 条警告提示
3. [ ] 实现清理命令 `/rpbox clear`

### 阶段七：游戏内回放 UI（M4）

1. [ ] 创建回放窗口框架
2. [ ] 实现日期/频道筛选
3. [ ] 实现搜索功能
4. [ ] 实现滚动浏览

### 阶段八：ProfileExport（M5）

1. [ ] 实现 `/rpbox export` 命令
2. [ ] 实现 TRP3 人物卡数据提取
3. [ ] 实现 RPBox_ProfileExport 数据结构

### 阶段九：ItemSync（M6）

1. [ ] 实现 `/rpbox item mark` 命令
2. [ ] 实现 TRP3 Extended 道具标记
3. [ ] 实现已标记道具列表
