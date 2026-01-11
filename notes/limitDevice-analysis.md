# limitDevice.lua 与 TRP3-Extended 协同机制分析

## 文件概述

`limitDevice.lua` 是一个 TRP3-Extended 道具脚本，模拟了一个功能丰富的"侏儒手机"（GnomePhone）UI 界面。

### 主要功能模块

| 模块 | 功能描述 |
|------|----------|
| 表情系统 | 150+ 预设表情，支持偏好设置 |
| 音乐播放器 | 内置曲库 + 自定义音乐 |
| 跑团骰子 | D&D 风格判定器 |
| NPC 扫描仪 | 模型 ID 扫描与全息投影 |
| 口癖设置 | 发言前缀/后缀自定义 |
| 小游戏 | 2048、剥沙蟹等 |

---

## 核心协同机制

### 1. 沙盒执行环境

TRP3-Extended 通过 `runLuaScriptEffect` 函数（位于 `ScriptGeneration.lua:993`）为道具脚本创建受限的沙盒环境：

```lua
-- TRP3-Extended 提供的沙盒环境 (LUA_ENV)
local LUA_ENV = {
    ["string"] = string,
    ["table"] = table,
    ["math"] = math,
    ["pairs"] = pairs,
    ["ipairs"] = ipairs,
    ["next"] = next,
    ["select"] = select,
    ["unpack"] = unpack,
    ["type"] = type,
    ["tonumber"] = tonumber,
    ["tostring"] = tostring,
    ["date"] = date,
};
```

然后注入三个关键的桥接函数：

| 函数 | 映射目标 | 用途 |
|------|----------|------|
| `getVar` | `TRP3_API.script.varCheck` | 读取持久化变量 |
| `setVar` | `setVarValue` | 写入持久化变量 |
| `effect` | `securedEffect` / `unsecuredEffect` | 触发工作流/效果 |

---

### 2. `args._G` 的原理

在 `limitDevice.lua` 中大量使用 `args._G`：

```lua
local phoneFrame = args._G.CreateFrame("Frame", "GnomePhone", args._G.UIParent, "BackdropTemplate")
args._G.PlaySound(624)
args._G.C_Timer.After(1, UpdateTime)
```

#### 关键发现

经过源码分析，TRP3-Extended 的标准代码中：

1. **`LUA_ENV` 不包含 `_G`**
2. **`args` 标准字段只有**：`object`、`container`、`class`、`scripts`、`classID`
3. **`setfenv` 设置的环境也不包含 `_G`**

```lua
-- Inventory.lua:279-280 - 标准的 args 构建
TRP3_API.script.executeClassScript(useWorkflow, class.SC,
    {object = info, container = container, class = class}, info.id);
```

#### 推测的注入方式

`args._G` **不是 TRP3-Extended 标准功能**，可能通过以下方式实现：

1. **道具初始化脚本注入**：在道具的某个工作流中，使用脚本效果设置 `args._G = _G`
2. **修改版 TRP3-Extended**：道具创建者可能使用了修改版的插件
3. **元表代理**：通过 `setmetatable` 设置 `__index` 元方法

> **注意**：这是一种绕过沙盒限制的技巧，使脚本能够访问完整的 WoW API。

---

### 3. 变量持久化系统

```lua
-- limitDevice.lua 中的使用示例
local savedPosition = getVar(args, "o", UI_POSITION_KEY)
setVar(args, "o", UI_POSITION_KEY, "CENTER,UIParent,CENTER,0,0")
setVar(args, "o", "favorite_emotes", table.concat(FAVORITE_EMOTES, ","))
```

#### 参数解析（来自 `ScriptGeneration.lua:861-902`）

| source | 含义 | 存储位置 | 生命周期 |
|--------|------|----------|----------|
| `"w"` | Workflow | `args.custom` | 工作流结束后消失 |
| `"o"` | Object | `args.object.vars` | **持久化到道具** |
| `"c"` | Campaign | 任务日志变量 | 任务期间 |

使用 `"o"` 意味着数据**保存在道具本身**，即使重新登录也会保留！

#### 源码实现

```lua
-- ScriptGeneration.lua:861-902
function TRP3_API.script.setVar(args, source, operationType, varName, varValue)
    if args and source and operationType then
        local storage;

        if source == "w" then
            storage = args.custom;
        elseif source == "o" and args.object then
            if not args.object.vars then
                args.object.vars = {};
            end
            storage = args.object.vars;
        elseif source == "c" and TRP3_API.quest.getActiveCampaignLog() then
            storage = TRP3_API.quest.getActiveCampaignLog();
            if not storage.vars then
                storage.vars = {};
            end
            storage = storage.vars;
        else
            return;
        end

        -- 支持的操作类型: =, [=], +, -, /, x
        if (operationType == "[=]" and not storage[varName]) or operationType == "=" then
            storage[varName] = varValue;
            return;
        end
        -- ... 数学运算
    end
end
```

---

### 4. 工作流触发机制

```lua
-- limitDevice.lua 中触发工作流
effect("run_workflow", args, "o", emote.workflowID)
effect("run_workflow", args, "o", "ask1")
effect("run_workflow", args, "o", "mus01")
```

#### 源码实现（`ScriptEffects.lua:260-272`）

```lua
["run_workflow"] = {
    getCArgs = function(args)
        local source = args[1] or "o";
        local id = args[2] or "";
        return source, id;
    end,
    method = function(structure, cArgs, eArgs)
        local workflowSource, workflowID = structure.getCArgs(cArgs);
        TRP3_API.script.runWorkflow(eArgs, workflowSource, workflowID);
        eArgs.LAST = 0;
    end,
    secured = security.HIGH,
},
```

这允许 Lua 脚本调用道具中预定义的其他工作流（如 `emo001`、`mus01` 等），实现模块化设计。

---

## 架构图示

```
┌─────────────────────────────────────────────────────────┐
│                    TRP3-Extended                        │
│  ┌─────────────────────────────────────────────────┐   │
│  │           runLuaScriptEffect()                  │   │
│  │  ┌─────────────────────────────────────────┐    │   │
│  │  │         沙盒环境 (setfenv)              │    │   │
│  │  │  ┌───────────────────────────────────┐  │    │   │
│  │  │  │      limitDevice.lua 脚本         │  │    │   │
│  │  │  │                                   │  │    │   │
│  │  │  │  args._G ──────► WoW 全局 API     │  │    │   │
│  │  │  │  getVar() ─────► 读取持久化变量   │  │    │   │
│  │  │  │  setVar() ─────► 写入持久化变量   │  │    │   │
│  │  │  │  effect() ─────► 触发工作流/效果  │  │    │   │
│  │  │  └───────────────────────────────────┘  │    │   │
│  │  └─────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────┘   │
│                         │                              │
│                         ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │              道具数据存储                        │   │
│  │   args.object.vars = {                          │   │
│  │     GnomePhone_Position = "CENTER,...",         │   │
│  │     favorite_emotes = "1,2,3,4,5,6",            │   │
│  │     custom_music = "2180808:麦卡贡点唱机",      │   │
│  │     前缀 = "[恶魔语]",                          │   │
│  │     后缀 = "*嚣张的语气*"                       │   │
│  │   }                                             │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## 脚本执行流程

```
1. 玩家使用道具
       │
       ▼
2. Inventory.lua:doUseSlot()
   构建 args = {object, container, class}
       │
       ▼
3. executeClassScript(workflowID, class.SC, args, itemID)
   添加 args.scripts, args.classID
       │
       ▼
4. 如果是脚本效果 → runLuaScriptEffect(code, args, secured)
   - 构建沙盒环境 env (LUA_ENV + effect/getVar/setVar)
   - loadstring 编译代码
   - setfenv 设置环境
   - 执行 func(args)
       │
       ▼
5. 脚本通过 args._G 访问 WoW API
   脚本通过 getVar/setVar 读写持久化数据
   脚本通过 effect("run_workflow",...) 调用其他工作流
```

---

## 巧妙之处总结

1. **安全与功能的平衡**：通过 `args._G` 暴露完整 WoW API，但脚本本身运行在沙盒中
2. **持久化存储**：`"o"` 源的变量直接存储在道具数据中，跨会话保留
3. **模块化设计**：复杂逻辑（如表情动画）通过 `run_workflow` 调用预定义工作流
4. **完整的 UI 框架**：利用 WoW 原生 Frame API 构建了一个功能完整的"手机"界面

---

## 关键源码位置

| 文件 | 行号 | 功能 |
|------|------|------|
| `ScriptGeneration.lua` | 993-1022 | `runLuaScriptEffect` 沙盒执行 |
| `ScriptGeneration.lua` | 979-992 | `LUA_ENV` 定义 |
| `ScriptGeneration.lua` | 861-902 | `setVar` 变量存储 |
| `ScriptGeneration.lua` | 909-935 | `varCheck` (getVar) 变量读取 |
| `ScriptEffects.lua` | 260-272 | `run_workflow` 效果 |
| `ScriptEffects.lua` | 476-488 | `script` 效果（执行 Lua 代码）|
| `Inventory.lua` | 270-284 | `doUseSlot` 道具使用入口 |

---

## 结论

这个脚本本质上是一个**运行在 TRP3-Extended 沙盒中的完整 WoW 插件**，展示了 TRP3-Extended 脚本系统的强大扩展能力。`args._G` 的使用是一种高级技巧，使得道具脚本能够突破沙盒限制，访问完整的游戏 API。
