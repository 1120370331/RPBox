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
    CHAT_MSG_WHISPER = "收到密语",
    CHAT_MSG_WHISPER_INFORM = "发送密语",
    WHISPER_IN = "收到密语",
    WHISPER_OUT = "发送密语",
    CHAT_MSG_GUILD = "公会",
    GUILD = "公会",
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
    WHISPER_IN = "FF80FF",
    WHISPER_OUT = "FF80FF",
    CHAT_MSG_GUILD = "40FF40",         -- 绿色
    GUILD = "40FF40",
}

-- TRP3 NPC/旁白颜色
local NPC_SAY_COLOR = "FFFFFF"      -- 白色 (说)
local NPC_WHISPER_COLOR = "CC99FF"  -- 淡紫色 (悄悄说)
local NPC_YELL_COLOR = "FF4040"     -- 红色 (喊)
local NPC_EMOTE_COLOR = "FF8040"    -- 橙色 (旁白/动作)

local function StripInvalidLeadingBytes(text)
    if not text or text == "" then return text end

    -- Drop UTF-8 replacement chars (U+FFFD) if any
    while text:sub(1, 3) == "\239\191\189" do
        text = text:sub(4)
    end

    -- Drop orphaned UTF-8 continuation bytes (0x80-0xBF)
    while true do
        local b = text:byte(1)
        if not b or b < 0x80 or b > 0xBF then break end
        text = text:sub(2)
    end

    return text:gsub("^%s+", "")
end

-- 获取职业颜色（十六进制字符串）
local function GetClassColor(classFilename)
    if not classFilename then return nil end
    local classColor = RAID_CLASS_COLORS[classFilename]
    if classColor then
        return format("%02X%02X%02X", classColor.r * 255, classColor.g * 255, classColor.b * 255)
    end
    return nil
end

-- 获取内联图标字符串
local function GetInlineIcon(record)
    if not RPBox_Config.showIcon then return "" end

    -- 优先使用TRP3头像
    if record.ref then
        local profile = ns.GetCachedProfile(record.ref)
        if profile and profile.IC then
            return format("|TInterface\\Icons\\%s:14:14|t ", profile.IC)
        end
    end
    -- 使用职业图标
    if record.cls then
        local coords = CLASS_ICON_TCOORDS and CLASS_ICON_TCOORDS[record.cls]
        if coords then
            return format("|TInterface\\GLUES\\CHARACTERCREATE\\UI-CHARACTERCREATE-CLASSES:14:14:0:0:64:64:%d:%d:%d:%d|t ",
                coords[1]*64, coords[2]*64, coords[3]*64, coords[4]*64)
        end
    end
    return ""
end

-- 解析 TRP3 NPC 对话格式
-- 格式: | NPC名字 说话方式 内容
local function ParseNPCMessage(content)
    if not content:match("^|") then
        return nil
    end
    -- 跳过 WoW 颜色代码 |cFFxxxxxx 开头的情况
    if content:match("^|c") then return nil end

    local text = content:gsub("^|+", ""):match("^%s*(.+)") -- 移除开头 | 并清理前导空格
    if not text then return nil end

    -- 清理末尾的颜色代码 |r
    text = text:gsub("|r%s*$", "")

    -- 尝试匹配不同的说话方式
    local npcName, message

    -- 悄悄说：
    npcName, message = text:match("^(.-)%s*悄悄说%s*[：:]%s*(.*)$")
    if npcName and message then
        message = message:gsub("|r%s*$", "")
        message = StripInvalidLeadingBytes(message)
        return { name = npcName, type = "whisper", message = message, color = NPC_WHISPER_COLOR }
    end

    -- 喊:
    npcName, message = text:match("^(.-)%s*喊%s*[：:]%s*(.*)$")
    if npcName and message then
        message = message:gsub("|r%s*$", "")
        return { name = npcName, type = "yell", message = message, color = NPC_YELL_COLOR }
    end

    -- 说:
    npcName, message = text:match("^(.-)%s*说%s*[：:]%s*(.*)$")
    if npcName and message then
        message = message:gsub("|r%s*$", "")
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
                if name ~= "" then
                    name = name .. " " .. cached.LN
                else
                    name = cached.LN
                end
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
                if name ~= "" then
                    name = name .. " " .. trp3.LN
                else
                    name = trp3.LN
                end
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
    days = nil,  -- nil=全部, 0=今天, 3=3天内, 7=7天内, 30=30天内
    channel = nil,
    search = "",
}

-- 获取可用的日期列表
local function GetAvailableDates()
    local dates = {}
    local chatLog = RPBox_ChatLog or {}
    for dateStr, _ in pairs(chatLog) do
        table.insert(dates, dateStr)
    end
    table.sort(dates, function(a, b) return a > b end)  -- 降序，最新的在前
    return dates
end

-- 初始化日期下拉框（改为天数范围选择）
local function InitDateDropdown()
    if not MainFrame or not MainFrame.dateDropdown then return end

    local dayOptions = {
        { value = nil, text = "全部" },
        { value = 0, text = "今天" },
        { value = 3, text = "3天内" },
        { value = 7, text = "7天内" },
        { value = 30, text = "30天内" },
    }

    UIDropDownMenu_Initialize(MainFrame.dateDropdown, function(self, level)
        for _, opt in ipairs(dayOptions) do
            local info = UIDropDownMenu_CreateInfo()
            info.text = opt.text
            info.value = opt.value
            info.checked = (currentFilter.days == opt.value)
            info.func = function()
                currentFilter.days = opt.value
                UIDropDownMenu_SetText(MainFrame.dateDropdown, opt.text)
                RefreshLogContent()
            end
            UIDropDownMenu_AddButton(info, level)
        end
    end)

    UIDropDownMenu_SetText(MainFrame.dateDropdown, "全部")
end

-- 初始化频道下拉框
local function InitChannelDropdown()
    if not MainFrame or not MainFrame.channelDropdown then return end

    local channelOptions = {
        { value = nil, text = "全部" },
        { value = "SAY", text = "说话" },
        { value = "YELL", text = "大喊" },
        { value = "EMOTE", text = "表情" },
        { value = "PARTY", text = "小队" },
        { value = "RAID", text = "团队" },
        { value = "WHISPER_IN", text = "收到密语" },
        { value = "WHISPER_OUT", text = "发送密语" },
        { value = "GUILD", text = "公会" },
    }

    UIDropDownMenu_Initialize(MainFrame.channelDropdown, function(self, level)
        for _, opt in ipairs(channelOptions) do
            local info = UIDropDownMenu_CreateInfo()
            info.text = opt.text
            info.value = opt.value
            info.checked = (currentFilter.channel == opt.value)
            info.func = function()
                currentFilter.channel = opt.value
                UIDropDownMenu_SetText(MainFrame.channelDropdown, opt.text)
                RefreshLogContent()
            end
            UIDropDownMenu_AddButton(info, level)
        end
    end)

    UIDropDownMenu_SetText(MainFrame.channelDropdown, "全部")
end

-- 获取筛选后的记录
local function GetFilteredRecords()
    local records = {}
    local chatLog = RPBox_ChatLog or {}
    local now = time()

    -- 计算时间范围
    local minTime = nil
    if currentFilter.days ~= nil then
        if currentFilter.days == 0 then
            -- 今天：从今天0点开始
            local today = date("*t", now)
            today.hour, today.min, today.sec = 0, 0, 0
            minTime = time(today)
        else
            -- x天内
            minTime = now - (currentFilter.days * 24 * 60 * 60)
        end
    end

    for dateStr, hours in pairs(chatLog) do
        for hourStr, hourRecords in pairs(hours) do
            for _, record in ipairs(hourRecords) do
                local timestamp = record.t or record.timestamp or 0
                local channel = record.c or record.channel
                local content = record.m or record.content

                -- 时间筛选
                local timeMatch = (minTime == nil) or (timestamp >= minTime)
                -- 频道筛选
                local channelMatch = not currentFilter.channel or channel == currentFilter.channel
                -- 搜索筛选（搜索内容、发送者、NPC名）
                local searchMatch = true
                if currentFilter.search ~= "" then
                    local searchLower = currentFilter.search:lower()
                    local sender = record.s or record.sender
                    local senderName = sender
                    if type(sender) == "table" then
                        senderName = sender.name or sender.gameID or ""
                    end
                    local npcName = record.npc or ""

                    local contentMatch = content and content:lower():find(searchLower, 1, true)
                    local senderMatch = senderName and tostring(senderName):lower():find(searchLower, 1, true)
                    local npcMatch = npcName and npcName:lower():find(searchLower, 1, true)

                    searchMatch = contentMatch or senderMatch or npcMatch
                end

                if timeMatch and channelMatch and searchMatch then
                    table.insert(records, record)
                end
            end
        end
    end

    table.sort(records, function(a, b)
        local ta = a.t or a.timestamp or 0
        local tb = b.t or b.timestamp or 0
        return ta > tb  -- 降序：最新的在前
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

    -- 保存纯文本用于复制
    MainFrame.logPlainText = {}

    for i, record in ipairs(records) do
        if i > (RPBox_Config.maxRecords or 10000) then break end

        local row = content.rows[i]
        if not row then
            row = CreateFrame("Frame", nil, content)
            row:SetHeight(20)
            content.rows[i] = row

            row.text = row:CreateFontString(nil, "OVERLAY", "GameFontNormal")
            row.text:SetPoint("TOPLEFT", 0, 0)
            row.text:SetWidth(500)
            row.text:SetJustifyH("LEFT")
            row.text:SetWordWrap(true)
        end

        row:SetPoint("TOPLEFT", 0, -yOffset)
        row:SetPoint("TOPRIGHT", 0, -yOffset)

        -- 兼容新旧字段
        local timestamp = record.t or record.timestamp or 0
        local channel = record.c or record.channel or ""
        local msgContent = record.m or record.content or ""

        -- 清理开头的 | 标记和空格
        if msgContent:match("^|[^c]") then
            msgContent = msgContent:sub(2):match("^%s*(.*)") or msgContent
        end

        local timeStr = date("[%H:%M:%S]", timestamp)
        local displayName, gameID, colorCode = GetDisplayName(record)
        local channelColor = CHANNEL_COLORS[channel] or CHANNEL_COLORS["CHAT_MSG_" .. channel] or "FFFFFF"

        -- 名字颜色优先级：TRP3自定义颜色 > 职业色 > 默认白色
        local nameColor = nil
        if colorCode then
            nameColor = colorCode:gsub("^#", "")
        end
        if not nameColor then
            nameColor = GetClassColor(record.cls)
        end
        if not nameColor then
            nameColor = "FFFFFF"  -- 默认白色
        end

        -- 检测 NPC 对话
        local npcData = nil
        local mk = record.mk

        if mk == "N" then
            local npcColor = NPC_SAY_COLOR
            local npcSpeechType = record.nt or "say"
            if npcSpeechType == "whisper" then
                npcColor = NPC_WHISPER_COLOR
            elseif npcSpeechType == "yell" then
                npcColor = NPC_YELL_COLOR
            end
            local cleanNpcName = record.npc
            if cleanNpcName then
                cleanNpcName = cleanNpcName:gsub("^|%s*", "")
            end
            local cleanMsg = msgContent
            -- 清理消息中的纹理代码 |Txxx|t
            cleanMsg = cleanMsg:gsub("|T.-|t", "")
            -- 清理开头的空格
            cleanMsg = cleanMsg:gsub("^%s+", "")
            if msgContent:match("^|[^c]") then
                local parsed = ParseNPCMessage(msgContent)
                if parsed then cleanMsg = parsed.message end
            end
            if npcSpeechType == "whisper" then
                cleanMsg = StripInvalidLeadingBytes(cleanMsg)
            end
            npcData = { name = cleanNpcName, type = npcSpeechType, message = cleanMsg, color = npcColor }
        elseif mk == "B" then
            local cleanMsg = msgContent
            if msgContent:match("^|[^c]") then
                local parsed = ParseNPCMessage(msgContent)
                if parsed then cleanMsg = parsed.message end
            end
            npcData = { name = nil, type = "emote", message = cleanMsg, color = NPC_EMOTE_COLOR }
        elseif not mk then
            npcData = ParseNPCMessage(msgContent)
        end

        -- 构建显示文本（带颜色）和纯文本（用于复制）
        local lineText, plainText
        local icon = GetInlineIcon(record)
        local senderTag = format("|cFF666666[来自%s]|r", displayName)
        local plainSenderTag = format("[来自%s]", displayName)

        if npcData then
            if npcData.name and npcData.name ~= "" then
                local npcColor = "|cFF" .. npcData.color
                if npcData.type == "whisper" then
                    lineText = format("|cFF888888%s|r %s[%s]|r %s悄悄说：%s|r %s",
                        timeStr, npcColor, npcData.name, npcColor, npcData.message, senderTag)
                    plainText = format("%s [%s] 悄悄说：%s %s",
                        timeStr, npcData.name, npcData.message, plainSenderTag)
                elseif npcData.type == "yell" then
                    lineText = format("|cFF888888%s|r %s[%s]|r 大喊：%s %s",
                        timeStr, npcColor, npcData.name, npcData.message, senderTag)
                    plainText = format("%s [%s] 大喊：%s %s",
                        timeStr, npcData.name, npcData.message, plainSenderTag)
                elseif npcData.type == "say" then
                    lineText = format("|cFF888888%s|r %s[%s]|r 说：%s %s",
                        timeStr, npcColor, npcData.name, npcData.message, senderTag)
                    plainText = format("%s [%s] 说：%s %s",
                        timeStr, npcData.name, npcData.message, plainSenderTag)
                else
                    lineText = format("|cFF888888%s|r %s[%s] %s|r %s",
                        timeStr, npcColor, npcData.name, npcData.message, senderTag)
                    plainText = format("%s [%s] %s %s",
                        timeStr, npcData.name, npcData.message, plainSenderTag)
                end
            else
                local npcColor = "|cFF" .. npcData.color
                lineText = format("|cFF888888%s|r %s%s|r %s",
                    timeStr, npcColor, npcData.message, senderTag)
                plainText = format("%s %s %s", timeStr, npcData.message, plainSenderTag)
            end
        elseif channel == "CHAT_MSG_EMOTE" or channel == "EMOTE" then
            lineText = format("|cFF888888%s|r |cFF%s[%s]|r%s |cFF%s%s|r",
                timeStr, nameColor, displayName, icon, channelColor, msgContent)
            plainText = format("%s [%s] %s", timeStr, displayName, msgContent)
        elseif channel == "TEXT_EMOTE" or channel == "CHAT_MSG_TEXT_EMOTE" then
            lineText = format("|cFF888888%s|r |cFF%s[%s]|r%s |cFF%s%s|r",
                timeStr, nameColor, displayName, icon, channelColor, msgContent)
            plainText = format("%s [%s] %s", timeStr, displayName, msgContent)
        elseif channel == "CHAT_MSG_YELL" or channel == "YELL" then
            lineText = format("|cFF888888%s|r |cFF%s[%s]|r%s 大喊：|cFF%s%s|r",
                timeStr, nameColor, displayName, icon, channelColor, msgContent)
            plainText = format("%s [%s] 大喊：%s", timeStr, displayName, msgContent)
        elseif channel == "WHISPER_IN" or channel == "CHAT_MSG_WHISPER" then
            lineText = format("|cFF888888%s|r |cFF%s[%s]|r%s 悄悄地说：|cFF%s%s|r",
                timeStr, nameColor, displayName, icon, channelColor, msgContent)
            plainText = format("%s [%s] 悄悄地说：%s", timeStr, displayName, msgContent)
        elseif channel == "WHISPER_OUT" or channel == "CHAT_MSG_WHISPER_INFORM" then
            lineText = format("|cFF888888%s|r 你悄悄地对 |cFF%s[%s]|r%s 说：|cFF%s%s|r",
                timeStr, nameColor, displayName, icon, channelColor, msgContent)
            plainText = format("%s 你悄悄地对 [%s] 说：%s", timeStr, displayName, msgContent)
        elseif channel == "GUILD" or channel == "CHAT_MSG_GUILD" then
            lineText = format("|cFF888888%s|r |cFF40FF40[公会]|r|cFF%s[%s]|r%s 说：|cFF40FF40%s|r",
                timeStr, nameColor, displayName, icon, msgContent)
            plainText = format("%s [公会][%s] 说：%s", timeStr, displayName, msgContent)
        else
            lineText = format("|cFF888888%s|r |cFF%s[%s]|r%s 说：|cFF%s%s|r",
                timeStr, nameColor, displayName, icon, channelColor, msgContent)
            plainText = format("%s [%s] 说：%s", timeStr, displayName, msgContent)
        end

        row.text:SetText(lineText)
        table.insert(MainFrame.logPlainText, plainText)

        local textHeight = row.text:GetStringHeight() or 16
        row:SetHeight(textHeight + 4)
        row:Show()
        yOffset = yOffset + textHeight + 6
    end

    content:SetHeight(math.max(yOffset, 1))
    MainFrame.statusText:SetText(format("共 %d 条记录", #records))

    -- 强制刷新 ScrollFrame 显示
    if MainFrame.logScrollFrame then
        MainFrame.logScrollFrame:UpdateScrollChildRect()
        -- 重置滚动位置到顶部
        MainFrame.logScrollFrame:SetVerticalScroll(0)
    end
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

        -- 兼容新旧字段
        local senderID = record.s or (record.sender and record.sender.gameID) or "unknown"
        local channel = record.c or record.channel or ""
        local content = record.m or record.content or ""
        local mk = record.mk
        local nt = record.nt
        local npc = record.npc

        table.insert(lines, "  senderID: " .. tostring(senderID))
        table.insert(lines, "  channel: " .. tostring(channel))
        table.insert(lines, "  mk: " .. tostring(mk))
        table.insert(lines, "  nt: " .. tostring(nt))
        table.insert(lines, "  npc: " .. tostring(npc))
        table.insert(lines, "  content (原始): [" .. tostring(content) .. "]")

        -- 检查是否以 | 开头
        local startsWithPipe = content:match("^|") and "是" or "否"
        local startsWithPipeNotC = content:match("^|[^c]") and "是" or "否"
        table.insert(lines, "  以|开头: " .. startsWithPipe)
        table.insert(lines, "  以|[^c]开头: " .. startsWithPipeNotC)

        -- 保存的 TRP3 数据
        if record.sender and record.sender.trp3 then
            table.insert(lines, "  [保存的TRP3数据(旧结构)]")
            table.insert(lines, "    FN: " .. tostring(record.sender.trp3.FN))
        elseif record.ref then
            table.insert(lines, "  [ProfileCache ref]: " .. tostring(record.ref))
            local cached = ns.GetCachedProfile(record.ref)
            if cached then
                table.insert(lines, "    FN: " .. tostring(cached.FN))
            end
        else
            table.insert(lines, "  [TRP3数据] 无")
        end

        -- NPC 解析结果
        local npcData = ParseNPCMessage(content)
        if npcData then
            table.insert(lines, "  [NPC解析结果]")
            table.insert(lines, "    type: " .. tostring(npcData.type))
            table.insert(lines, "    name: [" .. tostring(npcData.name) .. "]")
            table.insert(lines, "    message: [" .. tostring(npcData.message) .. "]")
        else
            table.insert(lines, "  [NPC解析结果] 返回nil")
        end
    end

    table.insert(lines, "")
    table.insert(lines, "--- 提示 ---")
    table.insert(lines, "可全选复制 (Ctrl+A, Ctrl+C)")

    MainFrame.debugEdit:SetText(table.concat(lines, "\n"))
    MainFrame.statusText:SetText("调试信息已生成")
end

-- 频道配置列表
local CHANNEL_CONFIG = {
    { key = "SAY", name = "说话" },
    { key = "YELL", name = "大喊" },
    { key = "EMOTE", name = "表情" },
    { key = "PARTY", name = "小队" },
    { key = "RAID", name = "团队" },
    { key = "WHISPER_IN", name = "收到密语" },
    { key = "WHISPER_OUT", name = "发送密语" },
    { key = "GUILD", name = "公会" },
}

-- 刷新设置内容
local function RefreshSettingsContent()
    if not MainFrame or not MainFrame.settingsContent then return end

    local content = MainFrame.settingsContent
    -- 清空
    for _, child in pairs({content:GetChildren()}) do
        child:Hide()
    end

    local yOffset = 0

    -- 功能开关区域
    local enableTitle = content:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    enableTitle:SetPoint("TOPLEFT", 5, -yOffset)
    enableTitle:SetText("功能开关")
    yOffset = yOffset + 30

    -- 总开关
    if not content.enabledCb then
        content.enabledCb = CreateFrame("CheckButton", nil, content, "UICheckButtonTemplate")
        content.enabledCb.text = content.enabledCb:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        content.enabledCb.text:SetPoint("LEFT", content.enabledCb, "RIGHT", 2, 0)
    end
    content.enabledCb:SetPoint("TOPLEFT", 10, -yOffset)
    content.enabledCb.text:SetText("开启聊天记录功能")
    content.enabledCb:SetChecked(RPBox_Config.enabled ~= false)
    content.enabledCb:SetScript("OnClick", function(self)
        RPBox_Config.enabled = self:GetChecked()
    end)
    content.enabledCb:Show()
    yOffset = yOffset + 26

    -- 频道监听设置标题
    yOffset = yOffset + 15
    local title = content:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    title:SetPoint("TOPLEFT", 5, -yOffset)
    title:SetText("频道监听设置")
    yOffset = yOffset + 25

    -- 频道复选框
    content.checkboxes = content.checkboxes or {}
    for i, channelInfo in ipairs(CHANNEL_CONFIG) do
        local cb = content.checkboxes[i]
        if not cb then
            cb = CreateFrame("CheckButton", nil, content, "UICheckButtonTemplate")
            cb.text = cb:CreateFontString(nil, "OVERLAY", "GameFontNormal")
            cb.text:SetPoint("LEFT", cb, "RIGHT", 2, 0)
            content.checkboxes[i] = cb
        end

        cb:SetPoint("TOPLEFT", 10, -yOffset)
        cb.text:SetText(channelInfo.name)
        cb.channelKey = channelInfo.key

        -- 读取当前配置
        local channels = RPBox_Config and RPBox_Config.channels or {}
        local enabled = channels[channelInfo.key]
        if enabled == nil then enabled = true end
        cb:SetChecked(enabled)

        -- 点击事件
        cb:SetScript("OnClick", function(self)
            RPBox_Config.channels = RPBox_Config.channels or {}
            RPBox_Config.channels[self.channelKey] = self:GetChecked()
        end)

        cb:Show()
        yOffset = yOffset + 26
    end

    -- 屏蔽设置标题
    yOffset = yOffset + 15
    local filterTitle = content:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    filterTitle:SetPoint("TOPLEFT", 5, -yOffset)
    filterTitle:SetText("屏蔽设置")
    yOffset = yOffset + 25

    -- 屏蔽自己复选框
    if not content.ignoreSelfCb then
        content.ignoreSelfCb = CreateFrame("CheckButton", nil, content, "UICheckButtonTemplate")
        content.ignoreSelfCb.text = content.ignoreSelfCb:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        content.ignoreSelfCb.text:SetPoint("LEFT", content.ignoreSelfCb, "RIGHT", 2, 0)
    end
    content.ignoreSelfCb:SetPoint("TOPLEFT", 10, -yOffset)
    content.ignoreSelfCb.text:SetText("屏蔽自己的消息")
    content.ignoreSelfCb:SetChecked(RPBox_Config.ignoreSelf == true)
    content.ignoreSelfCb:SetScript("OnClick", function(self)
        RPBox_Config.ignoreSelf = self:GetChecked()
    end)
    content.ignoreSelfCb:Show()
    yOffset = yOffset + 26

    -- 只接受公会成员复选框
    if not content.guildOnlyCb then
        content.guildOnlyCb = CreateFrame("CheckButton", nil, content, "UICheckButtonTemplate")
        content.guildOnlyCb.text = content.guildOnlyCb:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        content.guildOnlyCb.text:SetPoint("LEFT", content.guildOnlyCb, "RIGHT", 2, 0)
    end
    content.guildOnlyCb:SetPoint("TOPLEFT", 10, -yOffset)
    content.guildOnlyCb.text:SetText("只接受公会成员的消息")
    content.guildOnlyCb:SetChecked(RPBox_Config.guildOnly == true)
    content.guildOnlyCb:SetScript("OnClick", function(self)
        RPBox_Config.guildOnly = self:GetChecked()
    end)
    content.guildOnlyCb:Show()
    yOffset = yOffset + 26

    -- 显示设置标题
    yOffset = yOffset + 15
    local displayTitle = content:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    displayTitle:SetPoint("TOPLEFT", 5, -yOffset)
    displayTitle:SetText("显示设置")
    yOffset = yOffset + 25

    -- 显示图标复选框
    if not content.showIconCb then
        content.showIconCb = CreateFrame("CheckButton", nil, content, "UICheckButtonTemplate")
        content.showIconCb.text = content.showIconCb:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        content.showIconCb.text:SetPoint("LEFT", content.showIconCb, "RIGHT", 2, 0)
    end
    content.showIconCb:SetPoint("TOPLEFT", 10, -yOffset)
    content.showIconCb.text:SetText("在记录中显示头像图标")
    content.showIconCb:SetChecked(RPBox_Config.showIcon ~= false)
    content.showIconCb:SetScript("OnClick", function(self)
        RPBox_Config.showIcon = self:GetChecked()
    end)
    content.showIconCb:Show()
    yOffset = yOffset + 26

    content:SetHeight(yOffset + 20)
    MainFrame.statusText:SetText("设置")
end

-- 切换标签页
local function SwitchTab(tabName)
    if not MainFrame then return end
    currentTab = tabName

    -- 隐藏所有内容
    if MainFrame.logScroll then MainFrame.logScroll:Hide() end
    if MainFrame.listScroll then MainFrame.listScroll:Hide() end
    if MainFrame.debugScroll then MainFrame.debugScroll:Hide() end
    if MainFrame.settingsScroll then MainFrame.settingsScroll:Hide() end
    if MainFrame.filterFrame then MainFrame.filterFrame:Hide() end

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
        MainFrame.filterFrame:Show()
        MainFrame.logScroll:Show()
        InitDateDropdown()
        InitChannelDropdown()
        RefreshLogContent()
    elseif tabName == "whitelist" or tabName == "blacklist" then
        MainFrame.listScroll:Show()
        RefreshListContent(tabName)
    elseif tabName == "debug" then
        MainFrame.debugScroll:Show()
        RefreshDebugContent()
    elseif tabName == "settings" then
        MainFrame.settingsScroll:Show()
        RefreshSettingsContent()
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

    -- 启用调整大小
    MainFrame:SetResizable(true)
    MainFrame:SetResizeBounds(400, 300, 1200, 900)
    MainFrame:SetClampedToScreen(true)

    -- 创建调整大小按钮
    local resizeButton = CreateFrame("Button", nil, MainFrame)
    resizeButton:SetSize(16, 16)
    resizeButton:SetPoint("BOTTOMRIGHT", -5, 5)
    resizeButton:SetNormalTexture("Interface\\ChatFrame\\UI-ChatIM-SizeGrabber-Up")
    resizeButton:SetHighlightTexture("Interface\\ChatFrame\\UI-ChatIM-SizeGrabber-Highlight")
    resizeButton:SetPushedTexture("Interface\\ChatFrame\\UI-ChatIM-SizeGrabber-Down")
    resizeButton:SetScript("OnMouseDown", function(self, button)
        MainFrame:StartSizing("BOTTOMRIGHT")
    end)
    resizeButton:SetScript("OnMouseUp", function(self, button)
        MainFrame:StopMovingOrSizing()
    end)
    MainFrame.resizeButton = resizeButton

    MainFrame.TitleText:SetText("RPBox")

    -- 标签按钮
    MainFrame.tabButtons = {}
    local tabLog = CreateTabButton(MainFrame, "聊天记录", "log", 0)
    local tabWhite = CreateTabButton(MainFrame, "白名单", "whitelist", 85)
    local tabBlack = CreateTabButton(MainFrame, "黑名单", "blacklist", 170)
    local tabSettings = CreateTabButton(MainFrame, "设置", "settings", 255)
    local tabDebug = CreateTabButton(MainFrame, "调试", "debug", 340)

    table.insert(MainFrame.tabButtons, tabLog)
    table.insert(MainFrame.tabButtons, tabWhite)
    table.insert(MainFrame.tabButtons, tabBlack)
    table.insert(MainFrame.tabButtons, tabSettings)
    table.insert(MainFrame.tabButtons, tabDebug)

    for _, btn in pairs(MainFrame.tabButtons) do
        btn:SetScript("OnClick", function(self)
            SwitchTab(self.tabName)
        end)
    end

    -- 日志筛选栏
    local filterFrame = CreateFrame("Frame", nil, MainFrame)
    filterFrame:SetPoint("TOPLEFT", 12, -58)
    filterFrame:SetPoint("TOPRIGHT", -30, -58)
    filterFrame:SetHeight(28)

    -- 日期下拉框标签
    local dateLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    dateLabel:SetPoint("LEFT", 0, 0)
    dateLabel:SetText("日期:")

    -- 日期下拉框
    local dateDropdown = CreateFrame("Frame", "RPBoxDateDropdown", filterFrame, "UIDropDownMenuTemplate")
    dateDropdown:SetPoint("LEFT", dateLabel, "RIGHT", -10, -2)
    UIDropDownMenu_SetWidth(dateDropdown, 100)

    MainFrame.filterFrame = filterFrame
    MainFrame.dateDropdown = dateDropdown

    -- 频道下拉框标签
    local channelLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    channelLabel:SetPoint("LEFT", dateDropdown, "RIGHT", 10, 2)
    channelLabel:SetText("频道:")

    -- 频道下拉框
    local channelDropdown = CreateFrame("Frame", "RPBoxChannelDropdown", filterFrame, "UIDropDownMenuTemplate")
    channelDropdown:SetPoint("LEFT", channelLabel, "RIGHT", -10, -2)
    UIDropDownMenu_SetWidth(channelDropdown, 80)

    MainFrame.channelDropdown = channelDropdown

    -- 搜索框标签
    local searchLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    searchLabel:SetPoint("LEFT", channelDropdown, "RIGHT", 10, 2)
    searchLabel:SetText("搜索:")

    -- 搜索框
    local searchBox = CreateFrame("EditBox", nil, filterFrame, "InputBoxTemplate")
    searchBox:SetSize(100, 20)
    searchBox:SetPoint("LEFT", searchLabel, "RIGHT", 5, 0)
    searchBox:SetAutoFocus(false)
    searchBox:SetScript("OnEnterPressed", function(self)
        currentFilter.search = self:GetText()
        RefreshLogContent()
        self:ClearFocus()
    end)
    searchBox:SetScript("OnEscapePressed", function(self)
        self:ClearFocus()
    end)

    MainFrame.searchBox = searchBox

    -- 日志滚动框架
    local logScroll = CreateFrame("ScrollFrame", nil, MainFrame, "UIPanelScrollFrameTemplate")
    logScroll:SetPoint("TOPLEFT", 12, -88)
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

    -- 设置滚动框架
    local settingsScroll = CreateFrame("ScrollFrame", nil, MainFrame, "UIPanelScrollFrameTemplate")
    settingsScroll:SetPoint("TOPLEFT", 12, -60)
    settingsScroll:SetPoint("BOTTOMRIGHT", -30, 40)
    settingsScroll:Hide()

    local settingsContent = CreateFrame("Frame", nil, settingsScroll)
    settingsContent:SetSize(480, 300)
    settingsScroll:SetScrollChild(settingsContent)

    MainFrame.settingsScroll = settingsScroll
    MainFrame.settingsContent = settingsContent

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

    -- 复制按钮
    local copyBtn = CreateFrame("Button", nil, MainFrame, "UIPanelButtonTemplate")
    copyBtn:SetSize(60, 22)
    copyBtn:SetPoint("RIGHT", refreshBtn, "LEFT", -5, 0)
    copyBtn:SetText("复制")
    copyBtn:SetScript("OnClick", function()
        if not MainFrame.logPlainText or #MainFrame.logPlainText == 0 then
            print("|cFFFF0000[RPBox]|r 没有可复制的记录，请先筛选或刷新日志")
            return
        end

        -- 创建对话框（如果不存在）
        if not MainFrame.copyDialog then
            local dialog = CreateFrame("Frame", "RPBoxCopyDialog", UIParent, "BasicFrameTemplateWithInset")
            dialog:SetSize(450, 350)
            dialog:SetPoint("CENTER")
            dialog:SetMovable(true)
            dialog:EnableMouse(true)
            dialog:RegisterForDrag("LeftButton")
            dialog:SetScript("OnDragStart", dialog.StartMoving)
            dialog:SetScript("OnDragStop", dialog.StopMovingOrSizing)
            dialog:SetFrameStrata("DIALOG")
            dialog.TitleText:SetText("复制日志 (Ctrl+A 全选, Ctrl+C 复制)")

            -- 设置关闭按钮
            dialog.CloseButton:SetScript("OnClick", function()
                dialog.editBox:ClearFocus()
                dialog:Hide()
            end)

            local scroll = CreateFrame("ScrollFrame", nil, dialog, "UIPanelScrollFrameTemplate")
            scroll:SetPoint("TOPLEFT", 10, -30)
            scroll:SetPoint("BOTTOMRIGHT", -30, 10)

            local editBox = CreateFrame("EditBox", nil, scroll)
            editBox:SetMultiLine(true)
            editBox:SetFontObject(GameFontHighlightSmall)
            editBox:SetWidth(390)
            editBox:SetAutoFocus(false)
            editBox:EnableMouse(true)
            editBox:SetScript("OnEscapePressed", function(self)
                self:ClearFocus()
                dialog:Hide()
            end)
            scroll:SetScrollChild(editBox)

            dialog.editBox = editBox
            MainFrame.copyDialog = dialog

            -- 确保初始状态是隐藏的
            dialog:Hide()
        end

        -- 切换显示/隐藏
        if MainFrame.copyDialog:IsShown() then
            MainFrame.copyDialog.editBox:ClearFocus()
            MainFrame.copyDialog:Hide()
        else
            -- 更新内容并显示
            local text = table.concat(MainFrame.logPlainText, "\n")
            MainFrame.copyDialog.editBox:SetText(text)
            MainFrame.copyDialog.editBox:SetHeight(300)
            MainFrame.copyDialog:Show()
            MainFrame.copyDialog.editBox:HighlightText()
            MainFrame.copyDialog.editBox:SetFocus()
        end
    end)
    MainFrame.copyBtn = copyBtn

    -- 导出按钮 (reload)
    local exportBtn = CreateFrame("Button", nil, MainFrame, "UIPanelButtonTemplate")
    exportBtn:SetSize(60, 22)
    exportBtn:SetPoint("RIGHT", copyBtn, "LEFT", -5, 0)
    exportBtn:SetText("导出")
    exportBtn:SetScript("OnClick", function()
        ReloadUI()
    end)
    MainFrame.exportBtn = exportBtn

    -- 清空按钮
    local clearBtn = CreateFrame("Button", nil, MainFrame, "UIPanelButtonTemplate")
    clearBtn:SetSize(60, 22)
    clearBtn:SetPoint("RIGHT", exportBtn, "LEFT", -5, 0)
    clearBtn:SetText("清空")
    clearBtn:SetScript("OnClick", function()
        StaticPopup_Show("RPBOX_CLEAR_LOG_CONFIRM")
    end)
    MainFrame.clearBtn = clearBtn

    return MainFrame
end

-- 清空日志确认弹窗
StaticPopupDialogs["RPBOX_CLEAR_LOG_CONFIRM"] = {
    text = "确定要清空所有聊天记录吗？\n此操作不可撤销！",
    button1 = "确定",
    button2 = "取消",
    OnAccept = function()
        RPBox_ChatLog = {}
        if MainFrame and MainFrame:IsShown() and currentTab == "log" then
            RefreshLogContent()
        end
        print("|cFF00FF00[RPBox]|r 聊天记录已清空")
    end,
    timeout = 0,
    whileDead = true,
    hideOnEscape = true,
    preferredIndex = 3,
}

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

-- 注册新消息回调，自动刷新日志面板
ns.RegisterOnNewMessage(function()
    if MainFrame and MainFrame:IsShown() and currentTab == "log" then
        RefreshLogContent()
    end
end)

-- 注册名单变更回调，自动刷新名单面板
ns.RegisterOnListChange(function()
    if MainFrame and MainFrame:IsShown() then
        if currentTab == "whitelist" or currentTab == "blacklist" then
            RefreshListContent(currentTab)
        end
    end
end)
