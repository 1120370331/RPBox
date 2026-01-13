-- RPBox MainFrame
-- 统一管理界面

local ADDON_NAME, ns = ...
local L = ns.L or {}

-- 主窗口引用
local MainFrame = nil
local currentTab = "log"

-- 频道名称映射
local CHANNEL_NAMES = {
    CHAT_MSG_SAY = "说",
    CHAT_MSG_YELL = "喊",
    CHAT_MSG_EMOTE = "表情",
    CHAT_MSG_PARTY = "小队",
    CHAT_MSG_PARTY_LEADER = "小队",
    CHAT_MSG_RAID = "团队",
    CHAT_MSG_RAID_LEADER = "团队",
    CHAT_MSG_WHISPER = "密语",
    CHAT_MSG_WHISPER_INFORM = "密语",
}

-- WoW 原生频道颜色
local CHANNEL_COLORS = {
    CHAT_MSG_SAY = "FFFFFF",           -- 白色
    CHAT_MSG_YELL = "FF4040",          -- 红色
    CHAT_MSG_EMOTE = "FF8040",         -- 橙色
    CHAT_MSG_PARTY = "AAAAFF",         -- 蓝色
    CHAT_MSG_PARTY_LEADER = "AAAAFF",
    CHAT_MSG_RAID = "FF7F00",          -- 橙色
    CHAT_MSG_RAID_LEADER = "FF7F00",
    CHAT_MSG_WHISPER = "FF80FF",       -- 粉色
    CHAT_MSG_WHISPER_INFORM = "FF80FF",
}

-- TRP3 NPC/旁白颜色
local NPC_SAY_COLOR = "FFFFFF"      -- 白色 (说)
local NPC_WHISPER_COLOR = "CC99FF"  -- 淡紫色 (悄悄说)
local NPC_YELL_COLOR = "FF4040"     -- 红色 (喊)
local NPC_EMOTE_COLOR = "FF8040"    -- 橙色 (旁白/动作)

-- 解析 TRP3 NPC 对话格式
-- 格式: | NPC名字 说话方式 内容
local function ParseNPCMessage(content)
    if not content:match("^|") then
        return nil
    end

    local text = content:sub(2):match("^%s*(.+)") -- 移除 | 和前导空格
    if not text then return nil end

    -- 尝试匹配不同的说话方式
    local npcName, speechType, message

    -- 悄悄说：
    npcName, message = text:match("^(.-)%s*悄悄说[：:]%s*(.*)$")
    if npcName and message then
        return { name = npcName, type = "whisper", message = message, color = NPC_WHISPER_COLOR }
    end

    -- 喊:
    npcName, message = text:match("^(.-)%s*喊[：:]%s*(.*)$")
    if npcName and message then
        return { name = npcName, type = "yell", message = message, color = NPC_YELL_COLOR }
    end

    -- 说:
    npcName, message = text:match("^(.-)%s*说[：:]%s*(.*)$")
    if npcName and message then
        return { name = npcName, type = "say", message = message, color = NPC_SAY_COLOR }
    end

    -- 没有匹配到说话方式，视为旁白/动作
    return { name = nil, type = "emote", message = text, color = NPC_EMOTE_COLOR }
end

-- 实时获取 TRP3 信息（显示时查询）
local function GetTRP3InfoRealtime(unitID)
    if not TRP3_API or not TRP3_API.register then return nil end
    if not TRP3_API.register.isUnitIDKnown(unitID) then return nil end

    local character = TRP3_API.register.getUnitIDCharacter(unitID)
    if not character or not character.profileID then return nil end

    local profile = TRP3_API.register.getProfile(character.profileID)
    if not profile or not profile.player then return nil end

    local char = profile.player.characteristics or {}
    local rpName = nil
    if TRP3_API.register.getCompleteName then
        rpName = TRP3_API.register.getCompleteName(char, unitID, true)
    end

    return {
        rpName = rpName,
        CH = char.CH,
    }
end

-- 获取显示名称（兼容新旧数据结构）
local function GetDisplayName(record)
    local displayName = nil
    local colorCode = nil
    local senderID = record.s or (record.sender and record.sender.gameID)

    -- 1. 先尝试实时获取 TRP3 数据
    if senderID then
        local realtimeTRP3 = GetTRP3InfoRealtime(senderID)
        if realtimeTRP3 then
            if realtimeTRP3.rpName and realtimeTRP3.rpName ~= "" then
                displayName = realtimeTRP3.rpName
            end
            if realtimeTRP3.CH and realtimeTRP3.CH ~= "" then
                colorCode = realtimeTRP3.CH
            end
        end
    end

    -- 2. 从 ProfileCache 获取（新结构）
    if not displayName and record.ref then
        local cached = ns.GetCachedProfile(record.ref)
        if cached then
            local name = cached.FN or ""
            if cached.LN and cached.LN ~= "" then
                name = name .. " " .. cached.LN
            end
            if name ~= "" then
                displayName = name
            end
            if not colorCode and cached.CH then
                colorCode = cached.CH
            end
        end
    end

    -- 3. 旧结构兼容
    if not displayName and record.sender and record.sender.trp3 then
        local trp3 = record.sender.trp3
        if trp3.rpName and trp3.rpName ~= "" then
            displayName = trp3.rpName
        else
            local name = trp3.FN or ""
            if trp3.LN and trp3.LN ~= "" then
                name = name .. " " .. trp3.LN
            end
            if name ~= "" then
                displayName = name
            end
        end
        if not colorCode and trp3.CH then
            colorCode = trp3.CH
        end
    end

    -- 4. 回退到游戏名
    if not displayName or displayName == "" then
        displayName = senderID and strsplit("-", senderID) or "未知"
    end

    return displayName, senderID, colorCode
end

-- 当前筛选条件
local currentFilter = {
    date = nil,
    channel = nil,
    search = "",
}

-- 获取筛选后的记录
local function GetFilteredRecords()
    local records = {}
    local chatLog = RPBox_ChatLog or {}

    for dateStr, hours in pairs(chatLog) do
        if not currentFilter.date or currentFilter.date == dateStr then
            for hourStr, hourRecords in pairs(hours) do
                for _, record in ipairs(hourRecords) do
                    -- 兼容新旧字段
                    local channel = record.c or record.channel
                    local content = record.m or record.content
                    local channelMatch = not currentFilter.channel or channel == currentFilter.channel
                    local searchMatch = currentFilter.search == "" or
                        (content and content:lower():find(currentFilter.search:lower(), 1, true))

                    if channelMatch and searchMatch then
                        table.insert(records, record)
                    end
                end
            end
        end
    end

    table.sort(records, function(a, b)
        local ta = a.t or a.timestamp or 0
        local tb = b.t or b.timestamp or 0
        return ta > tb
    end)
    return records
end

-- 创建标签按钮
local function CreateTabButton(parent, text, tabName, xOffset)
    local btn = CreateFrame("Button", nil, parent, "UIPanelButtonTemplate")
    btn:SetSize(80, 24)
    btn:SetPoint("TOPLEFT", 12 + xOffset, -30)
    btn:SetText(text)
    btn.tabName = tabName
    return btn
end

-- 刷新日志内容
local function RefreshLogContent()
    if not MainFrame or not MainFrame.logContent then return end

    local content = MainFrame.logContent
    -- 清空
    for _, child in pairs({content:GetChildren()}) do
        child:Hide()
    end

    local records = GetFilteredRecords()
    local yOffset = 0
    content.rows = content.rows or {}

    for i, record in ipairs(records) do
        if i > 100 then break end

        local row = content.rows[i]
        if not row then
            row = CreateFrame("Frame", nil, content)
            row:SetHeight(20)
            content.rows[i] = row

            row.text = row:CreateFontString(nil, "OVERLAY", "GameFontNormal")
            row.text:SetPoint("TOPLEFT", 0, 0)
            row.text:SetWidth(480)
            row.text:SetJustifyH("LEFT")
            row.text:SetWordWrap(true)
        end

        row:SetPoint("TOPLEFT", 0, -yOffset)
        row:SetPoint("TOPRIGHT", 0, -yOffset)

        -- 兼容新旧字段
        local timestamp = record.t or record.timestamp or 0
        local channel = record.c or record.channel or ""
        local msgContent = record.m or record.content or ""

        local timeStr = date("%H:%M", timestamp)
        local displayName, gameID, colorCode = GetDisplayName(record)
        local channelColor = CHANNEL_COLORS[channel] or CHANNEL_COLORS["CHAT_MSG_" .. channel] or "FFFFFF"

        -- 名字颜色：优先 TRP3 自定义颜色
        local nameColor = colorCode and colorCode:gsub("^#", "") or "FFD100"

        -- 检测 TRP3 NPC 对话语法（兼容新旧结构）
        local npcData = nil
        local mk = record.mk

        -- 新结构：直接使用 mk 标记
        if mk == "N" then
            npcData = { name = record.npc, type = "npc", message = msgContent, color = NPC_SAY_COLOR }
        elseif mk == "B" then
            npcData = { name = nil, type = "emote", message = msgContent, color = NPC_EMOTE_COLOR }
        elseif not mk then
            -- 旧结构：解析消息内容
            npcData = ParseNPCMessage(msgContent)
        end

        -- 构建显示文本
        local lineText
        if npcData then
            -- NPC 对话 (标注原始发送者)
            local senderTag = format("|cFF666666[%s]|r ", displayName)
            if npcData.name and npcData.name ~= "" then
                local npcColor = "|cFF" .. npcData.color
                if npcData.type == "whisper" then
                    lineText = format("|cFF888888%s|r %s%s%s 悄悄说：%s|r",
                        timeStr, senderTag, npcColor, npcData.name, npcData.message)
                elseif npcData.type == "yell" then
                    lineText = format("|cFF888888%s|r %s%s%s 大喊：%s|r",
                        timeStr, senderTag, npcColor, npcData.name, npcData.message)
                elseif npcData.type == "say" then
                    lineText = format("|cFF888888%s|r %s%s%s 说：%s|r",
                        timeStr, senderTag, npcColor, npcData.name, npcData.message)
                else
                    lineText = format("|cFF888888%s|r %s%s%s %s|r",
                        timeStr, senderTag, npcColor, npcData.name, npcData.message)
                end
            else
                -- 无名字的旁白
                local npcColor = "|cFF" .. npcData.color
                lineText = format("|cFF888888%s|r %s%s%s|r",
                    timeStr, senderTag, npcColor, npcData.message)
            end
        elseif channel == "CHAT_MSG_EMOTE" or channel == "EMOTE" then
            -- 表情
            lineText = format("|cFF888888%s|r |cFF%s%s %s|r",
                timeStr, channelColor, displayName, msgContent)
        elseif channel == "CHAT_MSG_YELL" or channel == "YELL" then
            -- 大喊
            lineText = format("|cFF888888%s|r |cFF%s%s|r |cFF%s大喊：%s|r",
                timeStr, nameColor, displayName, channelColor, msgContent)
        else
            -- 普通说话
            lineText = format("|cFF888888%s|r |cFF%s%s|r 说：|cFF%s%s|r",
                timeStr, nameColor, displayName, channelColor, msgContent)
        end

        row.text:SetText(lineText)

        -- 动态计算行高
        local textHeight = row.text:GetStringHeight() or 16
        row:SetHeight(textHeight + 4)

        row:Show()
        yOffset = yOffset + textHeight + 6
    end

    content:SetHeight(math.max(yOffset, 1))
    MainFrame.statusText:SetText(format("共 %d 条记录", #records))
end

-- 刷新名单内容
local function RefreshListContent(listType)
    if not MainFrame or not MainFrame.listContent then return end

    local content = MainFrame.listContent
    for _, child in pairs({content:GetChildren()}) do
        child:Hide()
    end

    local list = listType == "whitelist" and RPBox_Config.whitelist or RPBox_Config.blacklist
    local yOffset = 0
    content.rows = content.rows or {}
    local i = 0

    for unitID, _ in pairs(list or {}) do
        i = i + 1
        local row = content.rows[i]
        if not row then
            row = CreateFrame("Frame", nil, content)
            row:SetHeight(24)
            content.rows[i] = row

            row.text = row:CreateFontString(nil, "OVERLAY", "GameFontNormal")
            row.text:SetPoint("LEFT", 5, 0)

            row.removeBtn = CreateFrame("Button", nil, row, "UIPanelButtonTemplate")
            row.removeBtn:SetSize(50, 20)
            row.removeBtn:SetPoint("RIGHT", -5, 0)
            row.removeBtn:SetText("移除")
        end

        row:SetPoint("TOPLEFT", 0, -yOffset)
        row:SetPoint("TOPRIGHT", 0, -yOffset)
        row.text:SetText(unitID)
        row.removeBtn:SetScript("OnClick", function()
            if listType == "whitelist" then
                ns.RemoveFromWhitelist(unitID)
            else
                ns.RemoveFromBlacklist(unitID)
            end
            RefreshListContent(listType)
        end)

        row:Show()
        yOffset = yOffset + 26
    end

    content:SetHeight(math.max(yOffset, 1))

    local count = 0
    for _ in pairs(list or {}) do count = count + 1 end
    MainFrame.statusText:SetText(format("%s: %d 人", listType == "whitelist" and "白名单" or "黑名单", count))
end

-- 生成调试信息
local function RefreshDebugContent()
    if not MainFrame or not MainFrame.debugEdit then return end

    local lines = {}
    table.insert(lines, "=== RPBox 调试日志 ===")
    table.insert(lines, "时间: " .. date("%Y-%m-%d %H:%M:%S"))
    table.insert(lines, "")

    -- TRP3 API 状态
    table.insert(lines, "--- TRP3 API 状态 ---")
    table.insert(lines, "TRP3_API: " .. (TRP3_API and "存在" or "不存在"))
    if TRP3_API then
        table.insert(lines, "TRP3_API.register: " .. (TRP3_API.register and "存在" or "不存在"))
        table.insert(lines, "isUnitIDKnown: " .. (TRP3_API.register and TRP3_API.register.isUnitIDKnown and "存在" or "不存在"))
        table.insert(lines, "getCompleteName: " .. (TRP3_API.register and TRP3_API.register.getCompleteName and "存在" or "不存在"))
    end
    table.insert(lines, "")

    -- 当前玩家信息
    table.insert(lines, "--- 当前玩家信息 ---")
    local playerID = ns.GetPlayerID()
    table.insert(lines, "playerID: " .. tostring(playerID))
    table.insert(lines, "GetRealmName(): " .. tostring(GetRealmName()))
    -- 测试自己的 TRP3 数据
    if TRP3_API and TRP3_API.profile then
        local player = TRP3_API.profile.getData("player")
        if player and player.characteristics then
            local char = player.characteristics
            table.insert(lines, "自己的TRP3 FN: " .. tostring(char.FN))
            table.insert(lines, "自己的TRP3 LN: " .. tostring(char.LN))
            table.insert(lines, "自己的TRP3 CH: " .. tostring(char.CH))
        else
            table.insert(lines, "自己的TRP3数据: 无法获取")
        end
    end
    table.insert(lines, "")

    -- 最近5条记录的详细信息
    table.insert(lines, "--- 最近5条记录详情 ---")
    local records = GetFilteredRecords()
    for i = 1, math.min(5, #records) do
        local record = records[i]
        table.insert(lines, "")
        table.insert(lines, format("[记录 %d]", i))
        table.insert(lines, "  gameID: " .. tostring(record.sender.gameID))
        table.insert(lines, "  channel: " .. tostring(record.channel))
        table.insert(lines, "  content: " .. tostring(record.content):sub(1, 50))

        -- 保存的 TRP3 数据
        if record.sender.trp3 then
            table.insert(lines, "  [保存的TRP3数据]")
            table.insert(lines, "    FN: " .. tostring(record.sender.trp3.FN))
            table.insert(lines, "    LN: " .. tostring(record.sender.trp3.LN))
            table.insert(lines, "    rpName: " .. tostring(record.sender.trp3.rpName))
            table.insert(lines, "    CH: " .. tostring(record.sender.trp3.CH))
        else
            table.insert(lines, "  [保存的TRP3数据] 无")
        end

        -- 实时查询 TRP3
        local realtimeTRP3 = GetTRP3InfoRealtime(record.sender.gameID)
        if realtimeTRP3 then
            table.insert(lines, "  [实时TRP3查询]")
            table.insert(lines, "    rpName: " .. tostring(realtimeTRP3.rpName))
            table.insert(lines, "    CH: " .. tostring(realtimeTRP3.CH))
        else
            table.insert(lines, "  [实时TRP3查询] 无数据")
            -- 详细诊断
            if TRP3_API and TRP3_API.register then
                local known = TRP3_API.register.isUnitIDKnown(record.sender.gameID)
                table.insert(lines, "    isUnitIDKnown: " .. tostring(known))
            end
        end

        -- NPC 解析结果
        local npcData = ParseNPCMessage(record.content)
        if npcData then
            table.insert(lines, "  [NPC解析结果]")
            table.insert(lines, "    type: " .. tostring(npcData.type))
            table.insert(lines, "    name: " .. tostring(npcData.name))
            table.insert(lines, "    message: " .. tostring(npcData.message))
            table.insert(lines, "    color: " .. tostring(npcData.color))
        else
            table.insert(lines, "  [NPC解析结果] 非NPC语法")
        end

        -- 最终显示结果
        local displayName, gameID, colorCode = GetDisplayName(record.sender)
        table.insert(lines, "  [最终显示]")
        table.insert(lines, "    displayName: " .. tostring(displayName))
        table.insert(lines, "    colorCode: " .. tostring(colorCode))
    end

    table.insert(lines, "")
    table.insert(lines, "--- 提示 ---")
    table.insert(lines, "可全选复制 (Ctrl+A, Ctrl+C)")

    MainFrame.debugEdit:SetText(table.concat(lines, "\n"))
    MainFrame.statusText:SetText("调试信息已生成")
end

-- 切换标签页
local function SwitchTab(tabName)
    if not MainFrame then return end
    currentTab = tabName

    -- 隐藏所有内容
    if MainFrame.logScroll then MainFrame.logScroll:Hide() end
    if MainFrame.listScroll then MainFrame.listScroll:Hide() end
    if MainFrame.debugScroll then MainFrame.debugScroll:Hide() end

    -- 更新按钮状态
    for _, btn in pairs(MainFrame.tabButtons or {}) do
        if btn.tabName == tabName then
            btn:SetEnabled(false)
        else
            btn:SetEnabled(true)
        end
    end

    -- 显示对应内容
    if tabName == "log" then
        MainFrame.logScroll:Show()
        RefreshLogContent()
    elseif tabName == "whitelist" or tabName == "blacklist" then
        MainFrame.listScroll:Show()
        RefreshListContent(tabName)
    elseif tabName == "debug" then
        MainFrame.debugScroll:Show()
        RefreshDebugContent()
    end
end

-- 创建主窗口
local function CreateMainFrame()
    if MainFrame then return MainFrame end

    -- 主窗口
    MainFrame = CreateFrame("Frame", "RPBoxMainFrame", UIParent, "BasicFrameTemplateWithInset")
    MainFrame:SetSize(550, 450)
    MainFrame:SetPoint("CENTER")
    MainFrame:SetMovable(true)
    MainFrame:EnableMouse(true)
    MainFrame:RegisterForDrag("LeftButton")
    MainFrame:SetScript("OnDragStart", MainFrame.StartMoving)
    MainFrame:SetScript("OnDragStop", MainFrame.StopMovingOrSizing)
    MainFrame:Hide()

    MainFrame.TitleText:SetText("RPBox")

    -- 标签按钮
    MainFrame.tabButtons = {}
    local tabLog = CreateTabButton(MainFrame, "聊天记录", "log", 0)
    local tabWhite = CreateTabButton(MainFrame, "白名单", "whitelist", 85)
    local tabBlack = CreateTabButton(MainFrame, "黑名单", "blacklist", 170)
    local tabDebug = CreateTabButton(MainFrame, "调试", "debug", 255)

    table.insert(MainFrame.tabButtons, tabLog)
    table.insert(MainFrame.tabButtons, tabWhite)
    table.insert(MainFrame.tabButtons, tabBlack)
    table.insert(MainFrame.tabButtons, tabDebug)

    for _, btn in pairs(MainFrame.tabButtons) do
        btn:SetScript("OnClick", function(self)
            SwitchTab(self.tabName)
        end)
    end

    -- 日志滚动框架
    local logScroll = CreateFrame("ScrollFrame", nil, MainFrame, "UIPanelScrollFrameTemplate")
    logScroll:SetPoint("TOPLEFT", 12, -60)
    logScroll:SetPoint("BOTTOMRIGHT", -30, 40)

    local logContent = CreateFrame("Frame", nil, logScroll)
    logContent:SetSize(480, 1)
    logScroll:SetScrollChild(logContent)

    MainFrame.logScroll = logScroll
    MainFrame.logContent = logContent

    -- 名单滚动框架
    local listScroll = CreateFrame("ScrollFrame", nil, MainFrame, "UIPanelScrollFrameTemplate")
    listScroll:SetPoint("TOPLEFT", 12, -60)
    listScroll:SetPoint("BOTTOMRIGHT", -30, 40)
    listScroll:Hide()

    local listContent = CreateFrame("Frame", nil, listScroll)
    listContent:SetSize(480, 1)
    listScroll:SetScrollChild(listContent)

    MainFrame.listScroll = listScroll
    MainFrame.listContent = listContent

    -- 调试滚动框架（带可复制的EditBox）
    local debugScroll = CreateFrame("ScrollFrame", nil, MainFrame, "UIPanelScrollFrameTemplate")
    debugScroll:SetPoint("TOPLEFT", 12, -60)
    debugScroll:SetPoint("BOTTOMRIGHT", -30, 40)
    debugScroll:Hide()

    local debugEdit = CreateFrame("EditBox", nil, debugScroll)
    debugEdit:SetMultiLine(true)
    debugEdit:SetFontObject(GameFontHighlightSmall)
    debugEdit:SetWidth(480)
    debugEdit:SetAutoFocus(false)
    debugEdit:EnableMouse(true)
    debugEdit:SetScript("OnEscapePressed", function(self) self:ClearFocus() end)
    debugScroll:SetScrollChild(debugEdit)

    MainFrame.debugScroll = debugScroll
    MainFrame.debugEdit = debugEdit

    -- 底部状态栏
    local statusText = MainFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    statusText:SetPoint("BOTTOMLEFT", 12, 12)
    MainFrame.statusText = statusText

    -- 刷新按钮
    local refreshBtn = CreateFrame("Button", nil, MainFrame, "UIPanelButtonTemplate")
    refreshBtn:SetSize(60, 22)
    refreshBtn:SetPoint("BOTTOMRIGHT", -35, 8)
    refreshBtn:SetText("刷新")
    refreshBtn:SetScript("OnClick", function()
        SwitchTab(currentTab)
    end)

    return MainFrame
end

-- 打开主界面
function ns.OpenMainFrame()
    local frame = CreateMainFrame()
    SwitchTab("log")
    frame:Show()
end

-- 关闭主界面
function ns.CloseMainFrame()
    if MainFrame then
        MainFrame:Hide()
    end
end

-- 切换主界面
function ns.ToggleMainFrame()
    if MainFrame and MainFrame:IsShown() then
        ns.CloseMainFrame()
    else
        ns.OpenMainFrame()
    end
end
