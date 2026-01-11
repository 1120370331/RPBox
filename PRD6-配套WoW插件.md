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

**数据结构：**
```lua
RPBox_ChatLog = {
    ["2024-01-11"] = {
        ["20"] = {  -- 20:00-20:59
            {
                timestamp = 1704960000,
                channel = "CHAT_MSG_SAY",
                sender = {
                    gameID = "玩家名-服务器",
                    trp3 = {
                        FN = "艾琳",      -- 名
                        LN = "风语者",    -- 姓
                        TI = "游荡的诗人", -- 头衔
                        RA = "暗夜精灵",  -- 种族
                        CL = "德鲁伊",    -- 职业
                        IC = "inv_misc_book_09", -- 图标
                    }
                },
                content = "今晚的月色真美...",
            },
        },
    },
}
```

**注意事项：**
- 若目标无TRP3数据，trp3字段为nil，仅保留gameID
- 首次遇到新角色时缓存其TRP3信息，避免重复查询

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

## 3. 技术实现

### 3.1 依赖
- TotalRP3 (必需)
- TotalRP3 Extended (可选)

### 3.2 SavedVariables
```toc
## SavedVariables: RPBox_ChatLog, RPBox_ProfileExport, RPBox_Config
```

### 3.3 TRP3 API 调用
```lua
-- 获取人物卡数据
TRP3_API.profile.getData("player")
-- 监听聊天
ChatFrame_AddMessageEventFilter("CHAT_MSG_SAY", handler)
```

## 4. 里程碑

| 阶段 | 功能 |
|------|------|
| M1 | ChatLogger 聊天记录 |
| M2 | TRP3 人物卡关联 |
| M3 | ProfileExport 导出 |
| M4 | ItemSync 道具标记 |
