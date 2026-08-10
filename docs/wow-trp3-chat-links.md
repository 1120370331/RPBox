# WoW 与 TRP3 聊天链接机制

本文记录处理魔兽世界原版物品/技能链接、TRP3 道具链接，以及宏/插件组装聊天链接时需要理解的机制。

已验证修正：在真实聊天频道里，玩家直接键入 raw 字符串（例如 `|cffffffff|Hitem:6948::::::::80:64:::::::::|h[炉石]|h|r`）不会自动变成链接，通常会按普通文本显示源码。原版链接应通过客户端认可的路径插入或发送：Shift-Click、`ChatEdit_InsertLink()`、`C_Item.GetItemInfo()` / `C_Spell.GetSpellLink()` 生成后发送。

## 1. 原版 ItemLink 与 Skill/SpellLink 格式

魔兽聊天链接本质是带控制码的 hyperlink 字符串，不是单纯的 `[名称]` 文本。

通用结构：

```lua
|cffRRGGBB|HlinkType:payload|h[显示名称]|h|r
```

字段含义：

| 片段 | 含义 |
| --- | --- |
| `|cffRRGGBB` | 显示颜色，物品通常按品质着色 |
| `|H...|h...|h` | hyperlink 主体 |
| `linkType` | 链接类型，例如 `item`、`spell` |
| `payload` | 链接数据，原版由客户端/服务器识别 |
| `[显示名称]` | 聊天框中看到的文本 |
| `|r` | 结束颜色控制 |

### ItemLink

物品链接通常形如：

```lua
|cffffffff|Hitem:6948::::::::80:64:::::::::|h[炉石]|h|r
```

简化理解：

```lua
|cff品质颜色|Hitem:itemID:若干物品参数|h[物品名]|h|r
```

注意：

- `item:` payload 不只有 `itemID`，还可能包含附魔、宝石、随机词缀、等级、专精、难度、bonusID 等参数。
- 不建议手拼完整 item link，应通过 API 取客户端生成的 `itemLink`。
- 聊天框能渲染 `|Hitem:...|h[...]|h`，点击后默认进入 `SetItemRef()`，再由 `ItemRefTooltip:SetHyperlink()` 展示 tooltip。

### Skill/SpellLink

像 `[冲锋]` 这种技能链接通常是 `spell` hyperlink，不是 `item` hyperlink：

```lua
|cff71d5ff|Hspell:100|h[冲锋]|h|r
```

简化理解：

```lua
|cff71d5ff|Hspell:spellID|h[技能名]|h|r
```

说明：

- `spellID = 100` 是战士技能“冲锋”的常见 spellID。
- 技能链接一般比 item link 简单，payload 常见就是 `spell:spellID`。
- 图标不是 spell link 自带的；如果需要图标，要额外插入 texture escape：`|TiconFileID:16:16|t`。

## 2. 手敲 / 宏 / 代码组装原版链接

### 2.1 手敲的可行边界

不可行方式：

```text
直接在聊天框键入：
|cffffffff|Hitem:6948::::::::80:64:::::::::|h[炉石]|h|r
```

实际效果：这通常只是普通文本，不会被真实聊天频道重新解释成 hyperlink。

可行的“手工”方式：

- 物品：打开背包，Shift-Click 物品，客户端会把真实 item link 插入聊天输入框。
- 技能/法术：打开技能书，Shift-Click 技能，客户端会把真实 spell link 插入聊天输入框。
- 宏/插件：通过 API 拿到客户端生成的 link，再用 `ChatEdit_InsertLink()` 插入输入框，或用 `SendChatMessage()` / `C_ChatInfo.SendChatMessage()` 发送。

### 2.2 宏：生成并插入到聊天输入框

这种方式最接近 Shift-Click：先打开聊天输入框，再调用 `ChatEdit_InsertLink(link)` 插入真实链接，最后由玩家手动按 Enter 发送。

物品链接插入 `/say`：

```lua
/run local _,l=C_Item.GetItemInfo(6948); if l then ChatFrame_OpenChat("/say "); ChatEdit_InsertLink(l); end
```

技能链接插入 `/emote`：

```lua
/run local l=C_Spell.GetSpellLink(100); if l then ChatFrame_OpenChat("/emote 发动 "); ChatEdit_InsertLink(l); end
```

说明：

- `6948` 是炉石 itemID。
- `100` 是冲锋 spellID。
- 如果物品没有缓存，`C_Item.GetItemInfo(6948)` 可能返回 `nil`，宏不会插入。
- 这类宏只是准备聊天输入，不直接发送，适合验证链接是否被客户端按真实 link 处理。

### 2.3 宏：生成并直接发送

如果是玩家点击宏触发，可以直接发送 API 返回的 link。

兼容 `C_ChatInfo.SendChatMessage` 与旧式 `SendChatMessage`：

```lua
/run local send=C_ChatInfo and C_ChatInfo.SendChatMessage or SendChatMessage; local _,l=C_Item.GetItemInfo(6948); if l then send("我拿出 "..l,"SAY"); end
```

发送技能到 `/emote`：

```lua
/run local send=C_ChatInfo and C_ChatInfo.SendChatMessage or SendChatMessage; local l=C_Spell.GetSpellLink(100); if l then send("发动 "..l,"EMOTE"); end
```

发送到其他频道只需要替换第二个参数：

```lua
"SAY"
"YELL"
"EMOTE"
"PARTY"
"RAID"
"GUILD"
"WHISPER" -- 需要额外传目标玩家
"CHANNEL" -- 需要额外传频道号/频道名
```

示例：密语给某人：

```lua
/run local send=C_ChatInfo and C_ChatInfo.SendChatMessage or SendChatMessage; local _,l=C_Item.GetItemInfo(6948); if l then send("给你看 "..l,"WHISPER",nil,"玩家名-服务器"); end
```

### 2.4 插件代码：获取物品链接

推荐使用 `C_Item.GetItemInfo()` 返回的 `itemLink`。

```lua
local itemID = 6948 -- 炉石
local itemName, itemLink = C_Item.GetItemInfo(itemID)

if itemLink then
    SendChatMessage("我拿出 " .. itemLink, "SAY")
end
```

注意：

- 物品未缓存时，`C_Item.GetItemInfo(itemID)` 可能返回 `nil`。
- 如果要可靠处理未缓存物品，需要监听 `GET_ITEM_INFO_RECEIVED` 后重试，或使用 `Item:CreateFromItemID()` 的加载回调。
- 发真实聊天频道时，尽量使用 API 返回的链接，不要手拼。

未缓存物品的更稳写法：

```lua
local function SendItemLinkWhenLoaded(itemID, channel)
    local item = Item:CreateFromItemID(itemID)
    item:ContinueOnItemLoad(function()
        local itemLink = item:GetItemLink()
        if itemLink then
            local send = C_ChatInfo and C_ChatInfo.SendChatMessage or SendChatMessage
            send("我拿出 " .. itemLink, channel or "SAY")
        end
    end)
end

SendItemLinkWhenLoaded(6948, "SAY")
```

### 2.5 插件代码：获取技能/法术链接

推荐使用 `C_Spell.GetSpellLink()`。

```lua
local spellID = 100 -- 冲锋
local spellLink = C_Spell.GetSpellLink(spellID)

if spellLink then
    SendChatMessage("发动 " .. spellLink, "EMOTE")
end
```

### 2.6 插件 UI：手工组装 raw hyperlink

如果是在插件自己的回放窗口、自定义 FontString、ScrollingMessageFrame 或 HTML-like UI 中显示，可以手工组装 raw hyperlink：

```lua
local itemText = "|cffffffff|Hitem:6948::::::::80:64:::::::::|h[炉石]|h|r"
DEFAULT_CHAT_FRAME:AddMessage(itemText)
```

这个方法适合“本地 UI 渲染”，不等同于“真实聊天频道发送”。真实聊天频道不应依赖手拼 raw 字符串。

### 2.7 插件 UI：技能图标 + 链接

如果是在自定义窗口、日志回放或插件 UI 中显示，可以组合图标和链接：

```lua
local spellID = 100
local spellLink = C_Spell.GetSpellLink(spellID)
local spellTexture = C_Spell.GetSpellTexture(spellID)

if spellLink and spellTexture then
    local text = ("|T%d:16:16|t %s"):format(spellTexture, spellLink)
    DEFAULT_CHAT_FRAME:AddMessage(text)
end
```

说明：

- `|T...|t` 是贴图 escape，可在许多客户端 UI 文本区域显示图标。
- 发送到公共聊天频道时，不应假设服务器一定允许任意手拼 `|T...|t` 图标文本。
- 插件自己的日志回放 UI 可以解析 `|T...|t` 并渲染成图标。

## 3. 原版链接识别方式

插件需要把三类文本识别为“链接标记”，避免直接显示源码：

1. 原版 `item:` 链接。
2. 原版 `spell:` 链接。
3. TRP3 `[TRP3:名称:编号]` 文本链接。

### 3.1 识别原版 hyperlink

原版完整 hyperlink 常见格式：

```lua
|cffRRGGBB|Hitem:...|h[名称]|h|r
|cffRRGGBB|Hspell:...|h[名称]|h|r
```

Lua 识别示例：

```lua
local function StripBrackets(label)
    if not label then return "" end
    return label:gsub("^%[", ""):gsub("%]$", "")
end

local function ClassifyNativePayload(payload)
    local itemID = payload:match("^item:(%d+)")
    if itemID then
        return "item", tonumber(itemID)
    end

    local spellID = payload:match("^spell:(%d+)")
    if spellID then
        return "spell", tonumber(spellID)
    end

    return nil, nil
end

local function ExtractNativeLinks(text)
    local links = {}
    if not text or text == "" then return links end

    -- 带颜色的常见形式：|cff...|H...|h[...]|h|r
    for color, payload, label in text:gmatch("|c(%x%x%x%x%x%x%x%x)|H([^|]+)|h(%b[])|h|r") do
        local linkType, id = ClassifyNativePayload(payload)
        if linkType then
            links[#links + 1] = {
                type = linkType,
                id = id,
                label = StripBrackets(label),
                payload = payload,
                color = color,
            }
        end
    end

    -- 无颜色形式：|H...|h[...]|h
    for payload, label in text:gmatch("|H([^|]+)|h(%b[])|h") do
        local linkType, id = ClassifyNativePayload(payload)
        if linkType then
            links[#links + 1] = {
                type = linkType,
                id = id,
                label = StripBrackets(label),
                payload = payload,
            }
        end
    end

    return links
end
```

显示策略：

- 如果只是以文本回放：显示为 `[炉石]`、`[冲锋]`，不要显示 `|c...|H...|h...` 源码。
- 如果自己实现 tooltip：保留 `type/id/payload/raw`，悬停时用本地数据或 API 查询。
- 如果无法识别 payload 类型：保守降级为普通 `[显示名称]`。

### 3.2 清洗为纯文本显示

如果只需要“去源码、保留方括号文本”，可以这样处理：

```lua
local function RenderWowLinksAsPlainLabels(text)
    if not text or text == "" then return text end

    text = text:gsub("|c%x%x%x%x%x%x%x%x|H[^|]+|h(%b[])|h|r", "%1")
    text = text:gsub("|H[^|]+|h(%b[])|h", "%1")
    text = text:gsub("|T[^|]+|t", "")
    text = text:gsub("|c%x%x%x%x%x%x%x%x", "")
    text = text:gsub("|r", "")
    return text
end
```

示例：

```lua
RenderWowLinksAsPlainLabels("|cffffffff|Hitem:6948::::::::80:64:::::::::|h[炉石]|h|r")
-- => [炉石]

RenderWowLinksAsPlainLabels("|cff71d5ff|Hspell:100|h[冲锋]|h|r")
-- => [冲锋]
```

### 3.3 识别 TRP3 文本链接

TRP3 文本链接格式：

```text
[TRP3:海兽之血宝石:1]
```

Lua 识别示例：

```lua
local function ExtractTRP3Links(text)
    local links = {}
    if not text or text == "" then return links end

    for content in text:gmatch("%[TRP3:([^%]]+)%]") do
        local name, numericID = content:match("^(.*):(%d+)$")
        links[#links + 1] = {
            type = "trp3",
            label = name or content,
            id = numericID and tonumber(numericID) or nil,
            raw = "[TRP3:" .. content .. "]",
        }
    end

    return links
end
```

显示策略：

- 文本回放：显示 `[海兽之血宝石]` 或保留 `[TRP3:海兽之血宝石:1]`，但标记为 TRP3。
- 如果想贴近 TRP3 原生体验：显示为黄色/金色链接样式，但不要伪造 `|Haddon:totalrp3...|h`。
- 导入能力不能只靠这个文本恢复，仍需要 Extended 的完整 class/rootClass 数据。

## 4. TRP3 道具 Link 渲染原理

TRP3/Total RP 3 Extended 的道具链接不是原版 `item:` 链接。

TRP3 采用两段式机制：

1. 聊天中发送普通文本占位符：

```text
[TRP3:海兽之血宝石:1]
```

2. 接收方本地 TRP3 插件扫描聊天消息，匹配：

```lua
%[TRP3:([^%]]+)%]
```

3. TRP3 将匹配到的文本替换成本地 addon hyperlink：

```lua
|Haddon:totalrp3:发送者:海兽之血宝石:1|h[海兽之血宝石]|h
```

4. 用户点击链接时，TRP3 hook `SetItemRef()`，识别 `addon:totalrp3`，然后通过插件通信向原发送者请求链接数据。

5. 原发送者返回 tooltip 数据、按钮数据、模块 ID、版本号、数据大小等，接收方再显示 TRP3 自己的 tooltip。

核心点：

- `[TRP3:名称:编号]` 本身只是文本占位符。
- 真正可点击的 `|Haddon:totalrp3:...|h` 是接收方本地 TRP3 渲染出来的。
- tooltip 内容不是从 `[TRP3:名称:编号]` 直接解析出来的，而是点击后向发送者请求。
- 如果发送者本地没有对应 sent link 记录，接收方可能只能看到过期/未知链接提示。
- 如果接收方没有 TRP3，则只会看到普通文本 `[TRP3:名称:编号]`。

TRP3 支持扫描的常见聊天事件包括：

```lua
CHAT_MSG_SAY
CHAT_MSG_YELL
CHAT_MSG_EMOTE
CHAT_MSG_TEXT_EMOTE
CHAT_MSG_PARTY
CHAT_MSG_PARTY_LEADER
CHAT_MSG_RAID
CHAT_MSG_RAID_LEADER
CHAT_MSG_GUILD
CHAT_MSG_OFFICER
CHAT_MSG_WHISPER
CHAT_MSG_WHISPER_INFORM
CHAT_MSG_CHANNEL
```

### TRP3 Extended 道具导入

Extended 道具链接模块大致保存这些数据：

```lua
{
    class = CopyTable(itemData),
    rootClass = CopyTable(TRP3_API.extended.getClass(rootID)),
    fullID = fullID,
    rootID = rootID,
    slotInfo = slotInfo,
    canBeImported = canBeImported,
}
```

导入按钮出现的前提：

- 链接由 TRP3 Extended 的正式 `ItemsChatLinksModule:InsertLink(...)` 流程创建。
- 发送者选择允许导入，即 `canBeImported = true`。
- 接收方安装了对应 TRP3 Extended 模块。
- 点击时发送者仍能响应插件通信并提供完整 class 数据。

导入行为：

- 导入到数据库：复制 `rootClass`，写入 `TRP3_DB.exchange[rootID]`，重新注册对象。
- 导入到背包：除写入 exchange 外，还调用 `TRP3_API.inventory.addItem(...)` 添加物品实例。

对离线归档/回放的意义：

- 只记录 `[TRP3:名称:编号]` 不足以离线复原 tooltip 或导入。
- 若要支持离线查看/导入 TRP3 道具，需要在记录阶段额外保存 Extended 道具的 class/rootClass/fullID/rootID 等数据。
- 如果只是回放聊天，可以按文本显示 `[TRP3:名称:编号]`，或在前端模拟成可点击样式，但不能凭空生成真实 TRP3 导入数据。

## 5. TRP3 道具通过脚本发送带 `[]` 链接的信息

目标不是发送原版 `|H...|h` hyperlink，而是发送一句普通聊天文本，其中包含 TRP3 能识别的占位符：

```text
这是一颗 [TRP3:海兽之血宝石:1]
```

接收方安装 TRP3 时，会扫描 `[TRP3:...]` 并在本地渲染成可点击的 TRP3 链接。接收方没有 TRP3 时，它就是普通文本。

### 5.1 只发送文本占位符

最简单方式是直接通过 TRP3 脚本的发言效果，或 Lua 代码发送 `[TRP3:名称:编号]`：

```lua
C_ChatInfo.SendChatMessage("这是一颗 [TRP3:海兽之血宝石:1]", "SAY")
```

这种方式可以触发接收方 TRP3 的文本扫描，但它只保证“外观上像链接”。如果发送者本地没有登记这条 link 的完整数据，接收方点击后可能拿不到 tooltip/import 数据。

### 5.2 正确生成可请求数据的 TRP3 Extended 道具链接

TRP3 Extended 的 `ItemsChatLinksModule:InsertLink(...)` 会创建 `ChatLink` 并登记 sent link，但默认只是把 `[TRP3:...]` 插入聊天输入框。若脚本需要“自动发送一句带链接的话”，可以走同一套底层对象创建流程：

```lua
local function BuildTRP3ExtendedItemLinkText(fullID, rootID, slotInfo, canBeImported)
    if not TRP3_API
        or not TRP3_API.extended
        or not TRP3_API.extended.ItemsChatLinksModule
        or not TRP3_API.ChatLink then
        return nil
    end

    local module = TRP3_API.extended.ItemsChatLinksModule
    rootID = rootID or TRP3_API.extended.getRootClassID(fullID)

    local name, data = module:GetLinkData(fullID, rootID, slotInfo or {}, canBeImported == true)
    local link = TRP3_API.ChatLink(name, data, module:GetID())
    return link:GetText()
end

local linkText = BuildTRP3ExtendedItemLinkText(fullID, rootID, slotInfo, true)
if linkText then
    C_ChatInfo.SendChatMessage("这是一颗 " .. linkText, "SAY")
end
```

这里的关键是 `TRP3_API.ChatLink(name, data, module:GetID())`：

- 它会把本次发送的链接登记到 TRP3 的 sent link 管理器。
- `link:GetText()` 返回真正要发进聊天里的普通文本，形如 `[TRP3:海兽之血宝石:1]`。
- 接收方点击链接时，TRP3 会向发送者请求 tooltip/import 数据。

### 5.3 插入输入框但不自动发送

如果希望保持接近 TRP3 官方 Shift-Click 的行为，可以只插入聊天输入框：

```lua
TRP3_API.ChatLinks:OpenMakeImportablePrompt(TRP3_API.loc.CL_EXTENDED_ITEM, function(canBeImported)
    TRP3_API.extended.ItemsChatLinksModule:InsertLink(fullID, rootID, slotInfo or {}, canBeImported)
end)
```

如果要在链接前后加一句话，需要在 `InsertLink()` 前后操作当前聊天输入框，或直接使用 5.2 的底层 `ChatLink` 方式组装整句。

### 5.4 限制

- 纯手写 `[TRP3:名称:编号]` 可以被识别成 TRP3 链接样式，但不保证 tooltip/import 可用。
- 真正可点击并能取回数据的链接，需要发送者本地通过 TRP3 的 ChatLink 流程登记 sent link。
- `canBeImported = true` 只表示导入按钮允许显示；接收方仍需要安装 TRP3 Extended，且发送者仍需要在线并能响应插件通信。
- 不要发送 `|Haddon:totalrp3...|h`。TRP3 的设计是发送普通 `[TRP3:...]` 文本，由接收方本地转换成 addon hyperlink。
