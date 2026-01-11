# Octopus.lua vs limitDevice.lua 对比分析

## 概述对比

| 特性 | Octopus.lua | limitDevice.lua |
|------|-------------|-----------------|
| **定位** | 通用框架/库 | 完整应用（侏儒手机）|
| **代码行数** | ~1300 行 | ~1140 行 |
| **设计模式** | 模块化框架 | 单体应用 |
| **UI 方式** | TRP3 标记语言 | WoW 原生 Frame API |
| **变量存储** | `"c"` (Campaign) | `"o"` (Object) |
| **沙盒突破** | 无（纯沙盒内） | 有（args._G）|

---

## 1. 架构设计对比

### Octopus - 模块化框架

```
Octopus = {
    logic      -- 表操作工具库
    LOG        -- 日志系统
    Assests    -- 资源库（图片/音效/图标/颜色）
    GUI        -- GUI 构建器 + 菜单系统
    Data       -- 数据持久化层
    Sound      -- 音效播放
    Listener   -- 事件监听器
    Operands   -- 操作数获取
    RunScript  -- 脚本运行器
    BasicScripts -- 基础脚本集
}
```

### limitDevice - 单体应用

```
直接执行的脚本，包含：
├── UI 框架创建（phoneFrame）
├── 多页面系统（homePage, emotePage, musicPage...）
├── 表情系统（150+ 预设）
├── 音乐播放器
├── 骰子判定器
├── NPC 扫描仪
└── 各种工具函数
```

---

## 2. UI 构建方式对比

### Octopus - TRP3 标记语言

使用 TRP3-Extended 内置的标记语言构建 UI：

```lua
-- 构建菜单文本
local menutext = "{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}\n"
menutext = menutext.."{h1:c}你确定要：{col:670001} "..confirmtype.." {/col}吗？{/h1}\n"
menutext = menutext.."{h2:c}"..descripe.."{/h2}\n"
menutext = menutext.."{link*confirm*确定}"

-- 通过工作流显示
effect("run_workflow",args,"c","open_menu")
```

**支持的标记：**
- `{h1:c}...{/h1}` - 标题（c=居中）
- `{col:FF0000}...{/col}` - 颜色
- `{link*action*text}` - 可点击链接
- `{icon:name:size}` - 图标
- `{img:path:w:h}` - 图片

### limitDevice - WoW 原生 Frame API

直接使用 WoW API 创建 UI 元素：

```lua
-- 创建框架
local phoneFrame = args._G.CreateFrame("Frame", "GnomePhone", args._G.UIParent, "BackdropTemplate")
phoneFrame:SetSize(UI_WIDTH, UI_HEIGHT)
phoneFrame:SetFrameStrata("DIALOG")
phoneFrame:SetBackdrop({...})

-- 创建按钮
local btn = args._G.CreateFrame("Button", nil, parent, "UIPanelButtonTemplate")
btn:SetScript("OnClick", function() ... end)

-- 创建文本
local text = frame:CreateFontString(nil, "OVERLAY", "GameFontNormal")
text:SetText("内容")
```

### 对比总结

| 方面 | Octopus (标记语言) | limitDevice (Frame API) |
|------|-------------------|------------------------|
| 学习曲线 | 低 | 高 |
| 灵活性 | 受限于 TRP3 支持 | 完全自由 |
| 交互能力 | 基础（链接点击） | 完整（拖拽、滚动等）|
| 视觉效果 | 简单 | 丰富 |
| 安全性 | 高（沙盒内） | 低（需要 args._G）|

---

## 3. 数据存储对比

### Octopus - Campaign 变量 + 自定义序列化

```lua
-- 使用 "c" (Campaign) 存储
setVar(args, "c", "MenuText", text)
getVar(args, "c", "LatestMenu")

-- 自定义表序列化格式
-- 格式: {[key] = $value^, [key2] = $value2^}
Octopus.logic.tostring = function(tb)
    local str = "{"
    for i,v in pairs(tb) do
        if type(v) == "string" then
            str = str.."["..tostring(i).."] = ".."$"..v.."^"..","
        elseif type(v) == "table" then
            str = str.."["..tostring(i).."] = "..Octopus.logic.tostring(v)..","
        end
    end
    str = str.."}"
    return str
end

-- 反序列化
Octopus.logic.strtotable(tstring)

-- 高级数据层（文件夹概念）
Octopus.Data.save("folder", "key", value)
Octopus.Data.get_value("folder", "key")
```

### limitDevice - Object 变量 + 简单字符串

```lua
-- 使用 "o" (Object) 存储
setVar(args, "o", "GnomePhone_Position", "CENTER,UIParent,CENTER,0,0")
getVar(args, "o", "favorite_emotes")

-- 简单的逗号分隔存储
setVar(args, "o", "favorite_emotes", table.concat(FAVORITE_EMOTES, ","))

-- 读取时分割
local saved = getVar(args, "o", "favorite_emotes")
for i, id in ipairs({args._G.strsplit(",", saved)}) do
    FAVORITE_EMOTES[i] = tonumber(id)
end
```

### 存储方式对比

| 方面 | Octopus | limitDevice |
|------|---------|-------------|
| 存储源 | `"c"` Campaign | `"o"` Object |
| 生命周期 | 任务期间 | 道具永久 |
| 序列化 | 自定义格式 | 简单字符串 |
| 复杂数据 | 支持嵌套表 | 仅简单值 |
| 数据组织 | 文件夹结构 | 扁平键值 |

---

## 4. 事件系统对比

### Octopus - 监听器系统

```lua
-- 支持的监听类型
Octopus.Listener.types = {
    "OnMessage_say", "OnMessage_yell", "OnMessage_emote",
    "OnMessage_raid", "OnMessage_party", "OnMessage_raidwarning",
    "OnMessage_guild", "OnMessage_whisper", "OnPlayerStartMove", "Always"
}

-- 注册监听器
Octopus.Listener.open({
    type = "OnMessage_say",
    scriptname = "my_handler",
    scriptargs = "some_args",
    id = "unique_id"
})

-- 关闭监听器
Octopus.Listener.close({type = "OnMessage_say", id = "unique_id"})
```

### limitDevice - 直接事件绑定

```lua
-- 使用 WoW 原生事件
btn:SetScript("OnClick", function(self, button)
    if button == "LeftButton" then
        -- 处理左键点击
    elseif button == "RightButton" then
        -- 处理右键点击
    end
end)

btn:SetScript("OnEnter", function(self)
    -- 鼠标进入
end)

btn:SetScript("OnMouseDown", function(self, button)
    -- 鼠标按下
end)

-- 定时器
args._G.C_Timer.After(1, UpdateTime)
args._G.C_Timer.NewTicker(0.1, UpdateHabitDisplay)
```

---

## 5. 工作流调用对比

### Octopus

```lua
-- 调用 Campaign 级别的工作流
effect("run_workflow", args, "c", "open_menu")
effect("run_workflow", args, "c", "close_menu")
effect("run_workflow", args, "c", "refresh_operands")
```

### limitDevice

```lua
-- 调用 Object 级别的工作流
effect("run_workflow", args, "o", emote.workflowID)
effect("run_workflow", args, "o", "mus01")
effect("run_workflow", args, "o", "ask1")
```

---

## 6. 核心差异：沙盒突破

### Octopus - 纯沙盒内运行

```lua
-- 完全使用 TRP3-Extended 提供的 API
effect("text", args, "消息", "4")
effect("sound_id_self", args, nil, soundID)
setVar(args, "c", "key", "value")
getVar(args, "c", "key")

-- 不访问 WoW 全局环境
-- 不使用 args._G
```

### limitDevice - 突破沙盒

```lua
-- 通过 args._G 访问完整 WoW API
args._G.CreateFrame(...)
args._G.PlaySound(...)
args._G.C_Timer.After(...)
args._G.DoEmote(...)
args._G.SendChatMessage(...)
args._G.GetCursorPosition()
args._G.UnitName("target")
args._G.GetZoneText()
```

---

## 7. 代码风格对比

### Octopus - 面向对象风格

```lua
-- 清晰的命名空间
Octopus.GUI.Menu.open(text)
Octopus.GUI.Menu.close()
Octopus.Data.save("folder", "key", value)
Octopus.Sound.playSoundSelf(soundID)

-- 类型注解
---@param folder string
---@param name string|nil
---@param value any
save = function (folder, name, value)
```

### limitDevice - 过程式风格

```lua
-- 直接定义函数和变量
local function UpdateTime()
    local timeData = GetBlackTempleTime()
    locationText:SetText(timeData.location)
    args._G.C_Timer.After(1, UpdateTime)
end

-- 大量内联逻辑
btn:SetScript("OnClick", function()
    if isAnimating then return end
    isAnimating = true
    icon:SetAlpha(0.7)
    -- ... 更多逻辑
end)
```

---

## 8. 适用场景

### Octopus 适合

- 需要复杂数据管理的任务/战役
- 多脚本协作的大型项目
- 需要事件监听的交互系统
- 标准化的菜单/对话框
- 团队协作开发

### limitDevice 适合

- 需要精美 UI 的独立道具
- 需要完整 WoW API 的功能
- 单一用途的工具类道具
- 需要实时交互的应用
- 个人开发的特色道具

---

## 9. 架构图对比

### Octopus 架构

```
┌─────────────────────────────────────────────────────────┐
│                    TRP3-Extended                        │
│  ┌─────────────────────────────────────────────────┐   │
│  │              标准沙盒环境                        │   │
│  │  ┌───────────────────────────────────────────┐  │   │
│  │  │            Octopus 框架                   │  │   │
│  │  │  ┌─────────────────────────────────────┐  │  │   │
│  │  │  │         用户脚本                    │  │  │   │
│  │  │  │  Octopus.GUI.Menu.open(...)        │  │  │   │
│  │  │  │  Octopus.Data.save(...)            │  │  │   │
│  │  │  │  Octopus.Listener.open(...)        │  │  │   │
│  │  │  └─────────────────────────────────────┘  │  │   │
│  │  │                    │                      │  │   │
│  │  │                    ▼                      │  │   │
│  │  │  effect() / getVar() / setVar()          │  │   │
│  │  └───────────────────────────────────────────┘  │   │
│  │                       │                         │   │
│  │                       ▼                         │   │
│  │              TRP3 工作流系统                    │   │
│  │              Campaign 变量存储                  │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### limitDevice 架构

```
┌─────────────────────────────────────────────────────────┐
│                    TRP3-Extended                        │
│  ┌─────────────────────────────────────────────────┐   │
│  │              沙盒环境                            │   │
│  │  ┌───────────────────────────────────────────┐  │   │
│  │  │        limitDevice.lua 脚本               │  │   │
│  │  │                                           │  │   │
│  │  │  args._G ─────────────────────────────────┼──┼───┼──► WoW 全局 API
│  │  │                                           │  │   │    CreateFrame
│  │  │  getVar()/setVar() ───────────────────────┼──┼───┼──► Object 变量
│  │  │                                           │  │   │
│  │  │  effect("run_workflow",...) ──────────────┼──┼───┼──► TRP3 工作流
│  │  │                                           │  │   │
│  │  └───────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## 10. 总结

| 维度 | Octopus | limitDevice |
|------|---------|-------------|
| **哲学** | 框架优先，标准化 | 功能优先，自由度 |
| **安全性** | 高（纯沙盒） | 低（突破沙盒）|
| **可复用性** | 高（模块化设计） | 低（单体应用）|
| **UI 能力** | 基础（标记语言） | 完整（原生 API）|
| **学习成本** | 中（需学习框架） | 高（需懂 WoW API）|
| **维护性** | 好（结构清晰） | 一般（代码密集）|
| **适用范围** | 任务/战役系统 | 独立工具道具 |

**Octopus** 是一个"正统"的 TRP3-Extended 开发方式，遵循沙盒规则，提供了良好的抽象层。

**limitDevice** 是一个"黑科技"方式，通过 `args._G` 突破沙盒限制，实现了更强大的功能，但也带来了安全风险。

两者代表了 TRP3-Extended 脚本开发的两种极端：**规范 vs 自由**。
