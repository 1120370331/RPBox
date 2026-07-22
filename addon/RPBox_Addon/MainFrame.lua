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

local function BuildIdentityName(identity)
    if type(identity) ~= "table" then return nil end
    if identity.n and identity.n ~= "" then return identity.n end
    if identity.rpName and identity.rpName ~= "" then return identity.rpName end

    local firstName = identity.FN or ""
    local lastName = identity.LN or ""
    if firstName ~= "" and lastName ~= "" then return firstName .. " " .. lastName end
    if firstName ~= "" then return firstName end
    if lastName ~= "" then return lastName end
    return nil
end

local function GetInlineRecordIdentity(record)
    if type(record) ~= "table" then return nil end
    if type(record.snapshot) == "table" then return record.snapshot end
    if type(record.identity) == "table" then return record.identity end
    if record.sender and type(record.sender.snapshot) == "table" then return record.sender.snapshot end
    if record.sender and type(record.sender.trp3) == "table" then return record.sender.trp3 end
    if record.FN or record.LN or record.TI or record.IC or record.CH or record.n then return record end
    return nil
end

-- Historical identity precedence: inline record snapshot, immutable snapshot table,
-- legacy embedded/cache data, and finally the literal game ID.
local function ResolveRecordIdentity(record)
    local inlineIdentity = GetInlineRecordIdentity(record)
    if inlineIdentity then return inlineIdentity, "record" end

    if record.ps and ns.GetProfileSnapshot then
        local snapshot = ns.GetProfileSnapshot(record.ps)
        if snapshot then return snapshot, "snapshot" end
    end

    if record.ref then
        local cached = ns.GetCachedProfile(record.ref)
        if cached then return cached, "legacy-cache" end
    end
    return nil, "game"
end

local function ResolveListenerIdentity(listener)
    if type(listener) ~= "table" then return nil end
    if type(listener.snapshot) == "table" then return listener.snapshot end
    if type(listener.trp3) == "table" then return listener.trp3 end
    if listener.ps and ns.GetProfileSnapshot then
        local snapshot = ns.GetProfileSnapshot(listener.ps)
        if snapshot then return snapshot end
    end
    local profileID = listener.ref or listener.profileID
    if profileID then return ns.GetCachedProfile(profileID) end
    return nil
end

-- 获取内联图标字符串
local function GetInlineIcon(record)
    if not RPBox_Config.showIcon then return "" end

    local identity = ResolveRecordIdentity(record)
    if identity and identity.IC and identity.IC ~= "" then
        return format("|TInterface\\Icons\\%s:14:14|t ", identity.IC)
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

-- 获取显示名称（兼容新旧数据结构）
local function GetDisplayName(record)
    local senderID = record.s or (record.sender and record.sender.gameID)
    local identity = ResolveRecordIdentity(record)
    local displayName = BuildIdentityName(identity)
    local colorCode = identity and identity.CH or nil

    -- Do not query live TRP3 data here: it would rewrite historical identity at render time.
    if not displayName or displayName == "" then
        displayName = senderID and strsplit("-", senderID) or "未知"
    end

    return displayName, senderID, colorCode
end

-- 当前筛选条件
local currentFilter = {
    days = nil,  -- nil=全部, 0=今天, 3=3天内, 7=7天内, 30=30天内
    channels = {},
    speakers = {},
    listeners = {},
    search = "",
}

local LOG_VIEW_WINDOW_SIZE = 80
local LOG_MIN_PAGE_SIZE = 40
local LOG_MAX_LIVE_ROWS = 120
local LOG_SCAN_BATCH_SIZE = 250
local LOG_RENDER_BATCH_SIZE = 20
local LOG_LIVE_REFRESH_INTERVAL = 1
local RefreshLogContent
local RenderLogPage
local SchedulePendingLiveRefresh

local CHANNEL_FILTER_OPTIONS = {
    { value = "SAY", text = "说话" },
    { value = "YELL", text = "大喊" },
    { value = "EMOTE", text = "表情" },
    { value = "TEXT_EMOTE", text = "文字表情" },
    { value = "PARTY", text = "小队" },
    { value = "RAID", text = "团队" },
    { value = "WHISPER_IN", text = "收到密语" },
    { value = "WHISPER_OUT", text = "发送密语" },
    { value = "GUILD", text = "公会" },
    { value = "SYSTEM", text = "人物卡节点" },
}

local function NormalizeRecordChannel(channel)
    if channel == "CHAT_MSG_SAY" then return "SAY" end
    if channel == "CHAT_MSG_YELL" then return "YELL" end
    if channel == "CHAT_MSG_EMOTE" then return "EMOTE" end
    if channel == "CHAT_MSG_TEXT_EMOTE" then return "TEXT_EMOTE" end
    if channel == "CHAT_MSG_PARTY" or channel == "CHAT_MSG_PARTY_LEADER" then return "PARTY" end
    if channel == "CHAT_MSG_RAID" or channel == "CHAT_MSG_RAID_LEADER" then return "RAID" end
    if channel == "CHAT_MSG_WHISPER" then return "WHISPER_IN" end
    if channel == "CHAT_MSG_WHISPER_INFORM" then return "WHISPER_OUT" end
    if channel == "CHAT_MSG_GUILD" then return "GUILD" end
    return channel
end

local function CountSelected(values)
    local count = 0
    for _, selected in pairs(values or {}) do
        if selected then count = count + 1 end
    end
    return count
end

local function HasSelections(values)
    return CountSelected(values) > 0
end

local function SetMultiDropdownText(dropdown, emptyText, selectedValues)
    if not dropdown then return end
    local count = CountSelected(selectedValues)
    UIDropDownMenu_SetText(dropdown, count == 0 and emptyText or ("已选 " .. tostring(count) .. " 项"))
end

local function GetDatePresetText(days)
    if days == 0 then return "今天" end
    if days == 1 then return "24小时内" end
    if days == 3 then return "3天内" end
    if days == 7 then return "7天内" end
    if days == 30 then return "30天内" end
    return "全部时间"
end

local function GetIdentitySelectorKey(profileID, gameID)
    if profileID and profileID ~= "" then return "p:" .. tostring(profileID) end
    if gameID and gameID ~= "" then return "g:" .. tostring(gameID) end
    return nil
end

local function GetEndpointIdentity(endpoint)
    if type(endpoint) ~= "table" then return nil end
    if endpoint.ps and ns.GetProfileSnapshot then
        return ns.GetProfileSnapshot(endpoint.ps)
    end
    return nil
end

local function GetEndpointDisplayName(endpoint)
    if type(endpoint) ~= "table" then return nil end
    if endpoint.n and endpoint.n ~= "" then return endpoint.n end
    local snapshot = GetEndpointIdentity(endpoint)
    return BuildIdentityName(snapshot)
end

local function AddParticipantOption(optionsByKey, profileID, gameID, identity, endpoint)
    local key = GetIdentitySelectorKey(profileID, gameID)
    if not key then return end

    local displayName = endpoint and GetEndpointDisplayName(endpoint) or BuildIdentityName(identity)
    local profileName = endpoint and endpoint.pn or (identity and identity.pn)
    local literalID = profileID or gameID
    local label = displayName or (gameID and strsplit("-", gameID)) or tostring(literalID or "未知")
    if profileName and profileName ~= "" and profileName ~= label then
        label = label .. " · " .. profileName
    end
    if literalID and literalID ~= "" then
        label = label .. "  [" .. tostring(literalID) .. "]"
    end
    if not optionsByKey[key] then
        optionsByKey[key] = { value = key, text = label }
        return optionsByKey[key]
    end
    return nil
end

local function IndexParticipantOption(catalog, profileID, gameID, identity, endpoint)
    local option = AddParticipantOption(catalog.byKey, profileID, gameID, identity, endpoint)
    if option then catalog.list[#catalog.list + 1] = option end
end

local function IndexRecordParticipants(speakerOptions, listenerOptions, record)
    local senderID = record.s or (record.sender and record.sender.gameID)
    local identity = ResolveRecordIdentity(record)
    local profileID = record.ref or (identity and identity.ref)
    IndexParticipantOption(speakerOptions, profileID, senderID, identity)

    if record.mk == "S" and record.ev then
        local from = record.ev.from
        local to = record.ev.to
        if from then IndexParticipantOption(speakerOptions, from.ref, senderID, GetEndpointIdentity(from), from) end
        if to then IndexParticipantOption(speakerOptions, to.ref, senderID, GetEndpointIdentity(to), to) end
    end

    for _, listener in ipairs(record.listeners or {}) do
        local listenerIdentity = ResolveListenerIdentity(listener)
        local listenerProfileID = listener.ref or listener.profileID or (listenerIdentity and listenerIdentity.ref)
        IndexParticipantOption(listenerOptions, listenerProfileID, listener.gameID, listenerIdentity)
    end
end

local function BuildParticipantOptions(mode)
    if not MainFrame or not MainFrame.participantOptions then return {} end
    return MainFrame.participantOptions[mode] or {}
end

local function UpdateFilterSummary()
    if not MainFrame or not MainFrame.filterSummary then return end
    local parts = {}
    if currentFilter.days ~= nil then parts[#parts + 1] = GetDatePresetText(currentFilter.days) end

    local speakerCount = CountSelected(currentFilter.speakers)
    local listenerCount = CountSelected(currentFilter.listeners)
    local channelCount = CountSelected(currentFilter.channels)
    if speakerCount > 0 then parts[#parts + 1] = "发言者 " .. speakerCount end
    if listenerCount > 0 then parts[#parts + 1] = "视角 " .. listenerCount end
    if channelCount > 0 then parts[#parts + 1] = "频道 " .. channelCount end
    if currentFilter.search ~= "" then parts[#parts + 1] = "搜索“" .. currentFilter.search .. "”" end

    MainFrame.filterSummary:SetText(#parts > 0 and table.concat(parts, "\n") or "当前显示全部档案记录")
    if MainFrame.clearFilterBtn then
        MainFrame.clearFilterBtn:SetEnabled(#parts > 0)
    end
end

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
        { value = nil, text = "全部时间" },
        { value = 0, text = "今天" },
        { value = 1, text = "24小时内" },
        { value = 3, text = "3天内" },
        { value = 7, text = "7天内" },
        { value = 30, text = "30天内" },
    }

    UIDropDownMenu_Initialize(MainFrame.dateDropdown, function(self, level)
        for _, opt in ipairs(dayOptions) do
            local value = opt.value
            local optionText = opt.text
            local info = UIDropDownMenu_CreateInfo()
            info.text = optionText
            info.value = value
            info.checked = (currentFilter.days == value)
            info.func = function()
                currentFilter.days = value
                UIDropDownMenu_SetText(MainFrame.dateDropdown, optionText)
                UpdateFilterSummary()
                RefreshLogContent()
            end
            UIDropDownMenu_AddButton(info, level)
        end
    end)

    UIDropDownMenu_SetText(MainFrame.dateDropdown, GetDatePresetText(currentFilter.days))
end

-- 初始化频道下拉框
local function InitChannelDropdown()
    if not MainFrame or not MainFrame.channelDropdown then return end

    UIDropDownMenu_Initialize(MainFrame.channelDropdown, function(self, level)
        local allInfo = UIDropDownMenu_CreateInfo()
        allInfo.text = "全部频道"
        allInfo.checked = not HasSelections(currentFilter.channels)
        allInfo.func = function()
            wipe(currentFilter.channels)
            SetMultiDropdownText(MainFrame.channelDropdown, "全部频道", currentFilter.channels)
            UpdateFilterSummary()
            RefreshLogContent()
        end
        UIDropDownMenu_AddButton(allInfo, level)

        for _, opt in ipairs(CHANNEL_FILTER_OPTIONS) do
            local value = opt.value
            local info = UIDropDownMenu_CreateInfo()
            info.text = opt.text
            info.value = value
            info.isNotRadio = true
            info.keepShownOnClick = true
            info.checked = currentFilter.channels[value] == true
            info.func = function()
                currentFilter.channels[value] = not currentFilter.channels[value] or nil
                SetMultiDropdownText(MainFrame.channelDropdown, "全部频道", currentFilter.channels)
                UpdateFilterSummary()
                RefreshLogContent()
            end
            UIDropDownMenu_AddButton(info, level)
        end
    end)

    SetMultiDropdownText(MainFrame.channelDropdown, "全部频道", currentFilter.channels)
end

local function InitParticipantDropdown(dropdown, mode, selectedValues)
    if not dropdown then return end
    local options = BuildParticipantOptions(mode)
    local emptyText = mode == "speaker" and "全部发言者" or "全部视角"

    UIDropDownMenu_Initialize(dropdown, function(self, level)
        local allInfo = UIDropDownMenu_CreateInfo()
        allInfo.text = emptyText
        allInfo.checked = not HasSelections(selectedValues)
        allInfo.func = function()
            wipe(selectedValues)
            SetMultiDropdownText(dropdown, emptyText, selectedValues)
            UpdateFilterSummary()
            RefreshLogContent()
        end
        UIDropDownMenu_AddButton(allInfo, level)

        for _, option in ipairs(options) do
            local value = option.value
            local info = UIDropDownMenu_CreateInfo()
            info.text = option.text
            info.value = value
            info.isNotRadio = true
            info.keepShownOnClick = true
            info.checked = selectedValues[value] == true
            info.func = function()
                selectedValues[value] = not selectedValues[value] or nil
                SetMultiDropdownText(dropdown, emptyText, selectedValues)
                UpdateFilterSummary()
                RefreshLogContent()
            end
            UIDropDownMenu_AddButton(info, level)
        end
    end)

    SetMultiDropdownText(dropdown, emptyText, selectedValues)
end

local function GetRecordSpeakerKeys(record)
    local keys = {}
    local senderID = record.s or (record.sender and record.sender.gameID)
    local identity = ResolveRecordIdentity(record)
    local primary = GetIdentitySelectorKey(record.ref or (identity and identity.ref), senderID)
    if primary then keys[primary] = true end

    if record.mk == "S" and record.ev then
        if record.ev.from and record.ev.from.ref then keys["p:" .. tostring(record.ev.from.ref)] = true end
        if record.ev.to and record.ev.to.ref then keys["p:" .. tostring(record.ev.to.ref)] = true end
    end
    return keys
end

local function GetRecordListenerKeys(record)
    local keys = {}
    for _, listener in ipairs(record.listeners or {}) do
        local identity = ResolveListenerIdentity(listener)
        local key = GetIdentitySelectorKey(listener.ref or listener.profileID or (identity and identity.ref), listener.gameID)
        if key then keys[key] = true end
    end
    return keys
end

local function MatchesSelectedKeys(selectedValues, recordKeys)
    if not HasSelections(selectedValues) then return true end
    for key in pairs(recordKeys) do
        if selectedValues[key] then return true end
    end
    return false
end

local function AddSearchPart(parts, value)
    if value ~= nil and value ~= "" then parts[#parts + 1] = value end
end

local function AppendIdentitySearchParts(parts, identity)
    if type(identity) ~= "table" then return end
    AddSearchPart(parts, identity.n)
    AddSearchPart(parts, identity.pn)
    AddSearchPart(parts, identity.FN)
    AddSearchPart(parts, identity.LN)
    AddSearchPart(parts, identity.TI)
    AddSearchPart(parts, identity.ref)
    AddSearchPart(parts, identity.gameID)
end

local function RecordMatchesSearch(record, searchLower)
    if not searchLower or searchLower == "" then return true end
    local parts = {}
    AddSearchPart(parts, record.m or record.content)
    AddSearchPart(parts, record.s or (record.sender and record.sender.gameID))
    AddSearchPart(parts, record.npc)
    AddSearchPart(parts, record.ref)
    AddSearchPart(parts, record.id)
    AppendIdentitySearchParts(parts, ResolveRecordIdentity(record))

    for _, listener in ipairs(record.listeners or {}) do
        AddSearchPart(parts, listener.gameID)
        AddSearchPart(parts, listener.ref or listener.profileID)
        AppendIdentitySearchParts(parts, ResolveListenerIdentity(listener))
    end

    if record.ev then
        local endpoints = {}
        if record.ev.from then endpoints[#endpoints + 1] = record.ev.from end
        if record.ev.to then endpoints[#endpoints + 1] = record.ev.to end
        for _, endpoint in ipairs(endpoints) do
            if endpoint then
                AddSearchPart(parts, endpoint.ref)
                AddSearchPart(parts, endpoint.n)
                AddSearchPart(parts, endpoint.pn)
                AppendIdentitySearchParts(parts, GetEndpointIdentity(endpoint))
            end
        end
    end

    for _, value in ipairs(parts) do
        if value and tostring(value):lower():find(searchLower, 1, true) then return true end
    end
    return false
end

local function GetFilterMinTime()
    local now = time()
    if currentFilter.days ~= nil then
        if currentFilter.days == 0 then
            local today = date("*t", now)
            today.hour, today.min, today.sec = 0, 0, 0
            return time(today)
        end
        return now - (currentFilter.days * 24 * 60 * 60)
    end
    return nil
end

local function RecordMatchesCurrentFilter(record, minTime, searchLower)
    local timestamp = record.t or record.timestamp or 0
    if minTime and timestamp < minTime then return false end

    local channel = NormalizeRecordChannel(record.c or record.channel)
    if HasSelections(currentFilter.channels)
        and (not channel or currentFilter.channels[channel] ~= true) then
        return false
    end
    if not MatchesSelectedKeys(currentFilter.speakers, GetRecordSpeakerKeys(record)) then return false end
    if not MatchesSelectedKeys(currentFilter.listeners, GetRecordListenerKeys(record)) then return false end
    return RecordMatchesSearch(record, searchLower)
end

local function SortHourKeys(a, b)
    local numberA, numberB = tonumber(a), tonumber(b)
    if numberA and numberB and numberA ~= numberB then return numberA < numberB end
    return tostring(a) < tostring(b)
end

-- Buckets are small (dates x hours), while records remain in append order inside each hour.
-- This provides chronological playback without a one-frame sort over every matching record.
local function BuildChronologicalLogBuckets()
    local chatLog = RPBox_ChatLog or {}
    local dateKeys = {}
    for dateKey in pairs(chatLog) do dateKeys[#dateKeys + 1] = dateKey end
    table.sort(dateKeys, function(a, b) return tostring(a) < tostring(b) end)

    local buckets = {}
    local totalRecords = 0
    for _, dateKey in ipairs(dateKeys) do
        local hours = chatLog[dateKey]
        if type(hours) == "table" then
            local hourKeys = {}
            for hourKey in pairs(hours) do hourKeys[#hourKeys + 1] = hourKey end
            table.sort(hourKeys, SortHourKeys)
            for _, hourKey in ipairs(hourKeys) do
                local records = hours[hourKey]
                if type(records) == "table" and #records > 0 then
                    buckets[#buckets + 1] = records
                    totalRecords = totalRecords + #records
                end
            end
        end
    end
    return buckets, totalRecords
end

local function InvalidateLogRender()
    if not MainFrame then return end
    MainFrame.logRenderToken = (MainFrame.logRenderToken or 0) + 1
    MainFrame.logPageRenderToken = (MainFrame.logPageRenderToken or 0) + 1
    MainFrame.logLiveRefreshScheduleToken = (MainFrame.logLiveRefreshScheduleToken or 0) + 1
    MainFrame.logLiveRefreshScheduled = false
    MainFrame.logLiveRefreshPending = false
    MainFrame.logScan = nil
    MainFrame.logState = nil
end

local function GetLogContentWidth()
    if MainFrame and MainFrame.logScroll then
        local width = MainFrame.logScroll:GetWidth() or 0
        if width > 0 then
            return max(width - 22, 320)
        end
    end
    return 480
end

local function GetLogTextWidth()
    return max(GetLogContentWidth() - 4, 300)
end

local function UpdateLogLayoutWidth()
    if not MainFrame or not MainFrame.logContent then return end

    local contentWidth = GetLogContentWidth()
    MainFrame.logContent:SetWidth(contentWidth)

    local rows = MainFrame.logContent.rows or {}
    local textWidth = GetLogTextWidth()
    for i = 1, #rows do
        local row = rows[i]
        if row and row.text then
            row.text:SetWidth(textWidth)
        end
    end
end

local function EnsureLogRow(content, index)
    if index < 1 or index > LOG_MAX_LIVE_ROWS then return nil end
    content.rows = content.rows or {}

    local row = content.rows[index]
    if not row then
        row = CreateFrame("Frame", nil, content)
        row:SetHeight(20)
        content.rows[index] = row

        row.text = row:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        row.text:SetPoint("TOPLEFT", 0, 0)
        row.text:SetWidth(GetLogTextWidth())
        row.text:SetJustifyH("LEFT")
        row.text:SetWordWrap(true)
        if row.text.SetNonSpaceWrap then
            row.text:SetNonSpaceWrap(true)
        end
    end

    return row
end

local function FormatEventEndpoint(endpoint)
    if type(endpoint) ~= "table" then return "未记录" end
    local snapshot = GetEndpointIdentity(endpoint)
    local displayName = (endpoint.n and endpoint.n ~= "" and endpoint.n) or BuildIdentityName(snapshot)
    local profileName = endpoint.pn or (snapshot and snapshot.pn)
    local label = displayName
    if profileName and profileName ~= "" and profileName ~= displayName then
        label = label and (label .. " / " .. profileName) or profileName
    end
    if not label or label == "" then label = "未命名人物卡" end
    if endpoint.ref and endpoint.ref ~= "" then
        label = label .. " [" .. tostring(endpoint.ref) .. "]"
    end
    return label
end

local function BuildLogLineTexts(record)
    local timestamp = record.t or record.timestamp or 0
    local channel = record.c or record.channel or ""
    local msgContent = record.m or record.content or ""

    if msgContent:match("^|[^c]") then
        msgContent = msgContent:sub(2):match("^%s*(.*)") or msgContent
    end

    local timeStr = date("[%H:%M:%S]", timestamp)
    if record.mk == "S" and record.ev then
        local event = record.ev
        local fromLabel = FormatEventEndpoint(event.from)
        local toLabel = FormatEventEndpoint(event.to)
        local observed = event.certainty == "observed"
        local certaintyText = observed and "观测到" or "已记录"
        if event.kind == "profile_switch" then
            local lineText = format(
                "|cFF888888%s|r |cFF6EC6BE━━ %s人物卡切换 ━━|r\n|cFF95B8B4%s|r  |cFF6EC6BE→|r  |cFFFFFFFF%s|r",
                timeStr, certaintyText, fromLabel, toLabel
            )
            local plainText = format("%s [%s人物卡切换] %s -> %s", timeStr, certaintyText, fromLabel, toLabel)
            return lineText, plainText
        end

        local lineText = format(
            "|cFF888888%s|r |cFF6EC6BE◇ %s人物卡身份更新|r\n|cFF95B8B4%s|r  |cFF6EC6BE→|r  |cFFFFFFFF%s|r",
            timeStr, certaintyText, fromLabel, toLabel
        )
        local plainText = format("%s [%s人物卡身份更新] %s -> %s", timeStr, certaintyText, fromLabel, toLabel)
        return lineText, plainText
    end

    local displayName, _, colorCode = GetDisplayName(record)
    local channelColor = CHANNEL_COLORS[channel] or CHANNEL_COLORS["CHAT_MSG_" .. channel] or "FFFFFF"

    local nameColor = nil
    if colorCode then
        nameColor = colorCode:gsub("^#", "")
    end
    if not nameColor then
        nameColor = GetClassColor(record.cls)
    end
    if not nameColor then
        nameColor = "FFFFFF"
    end

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
        cleanMsg = cleanMsg:gsub("|T.-|t", "")
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

    return lineText, plainText
end

local function RenderLogRow(row, record)
    local lineText, plainText = BuildLogLineTexts(record)
    row.text:SetWidth(GetLogTextWidth())
    row.text:SetText(lineText)

    local textHeight = row.text:GetStringHeight() or 16
    row:SetHeight(textHeight + 4)
    return textHeight, plainText
end

local function GetLogPageSize()
    local size = tonumber(RPBox_Config.logViewWindowSize) or LOG_VIEW_WINDOW_SIZE
    if size < LOG_MIN_PAGE_SIZE then size = LOG_MIN_PAGE_SIZE end
    if size > LOG_MAX_LIVE_ROWS then size = LOG_MAX_LIVE_ROWS end
    return floor(size)
end

local function HideAllLogRows()
    if not MainFrame or not MainFrame.logContent then return end
    local rows = MainFrame.logContent.rows or {}
    for i = 1, #rows do
        if rows[i] then
            rows[i]:Hide()
        end
    end
    MainFrame.logShownRowCount = 0
end

local function SetLogNavigationState(state, scanning)
    if not MainFrame then return end
    local busy = scanning or (state and state.rendering)
    if MainFrame.prevPageBtn then
        MainFrame.prevPageBtn:SetEnabled(not busy and state ~= nil and state.page > 1)
    end
    if MainFrame.nextPageBtn then
        MainFrame.nextPageBtn:SetEnabled(not busy and state ~= nil and state.page < state.totalPages)
    end
    if MainFrame.latestPageBtn then
        MainFrame.latestPageBtn:SetEnabled(not busy and state ~= nil and state.page < state.totalPages)
    end
    if MainFrame.pageProgress then
        if scanning then
            MainFrame.pageProgress:SetText("筛选中…")
        elseif state then
            MainFrame.pageProgress:SetText(format("%d / %d 页", state.page, state.totalPages))
        else
            MainFrame.pageProgress:SetText("0 / 0 页")
        end
    end
end

local function UpdateLogScanStatus(scanned, totalRecords, matched)
    if not MainFrame or not MainFrame.statusText then return end
    MainFrame.statusText:SetText(format("正在筛选：已扫描 %d/%d 条，命中 %d 条", scanned, totalRecords, matched))
end

local function UpdateLogPageStatus(state)
    if not MainFrame or not MainFrame.statusText then return end
    if not state or state.totalMatched <= 0 then
        MainFrame.statusText:SetText("共 0 条记录")
        return
    end

    local text = format(
        "共 %d 条 · 当前 %d-%d · 第 %d/%d 页",
        state.totalMatched,
        state.startIndex,
        state.endIndex,
        state.page,
        state.totalPages
    )
    if state.rendering and state.loadedCount < state.displayCount then
        text = text .. format(" · 正在绘制 %d/%d", state.loadedCount, state.displayCount)
    end
    MainFrame.statusText:SetText(text)
end

local function ResetLogViewport()
    if not MainFrame or not MainFrame.logContent then return end
    HideAllLogRows()
    MainFrame.logContent:SetHeight(1)
    if MainFrame.logScroll then
        MainFrame.logScroll:SetVerticalScroll(0)
        MainFrame.logScroll:UpdateScrollChildRect()
    end
end

local function GetLiveRefreshClock()
    if type(GetTime) == "function" then return GetTime() end
    return time()
end

-- Live chat can arrive faster than a complete archive scan. Keep one dirty bit instead of
-- cancelling an active scan, and start at most one live-triggered refresh per interval.
SchedulePendingLiveRefresh = function()
    if not MainFrame or not MainFrame.logLiveRefreshPending then return end
    if not MainFrame:IsShown() or currentTab ~= "log" then return end
    if MainFrame.logScan or (MainFrame.logState and MainFrame.logState.rendering) then return end
    if MainFrame.logLiveRefreshScheduled then return end

    local now = GetLiveRefreshClock()
    local lastRefresh = MainFrame.logLastLiveRefreshAt or 0
    local delay = max(LOG_LIVE_REFRESH_INTERVAL - (now - lastRefresh), 0)
    MainFrame.logLiveRefreshScheduled = true
    MainFrame.logLiveRefreshScheduleToken = (MainFrame.logLiveRefreshScheduleToken or 0) + 1
    local scheduleToken = MainFrame.logLiveRefreshScheduleToken

    C_Timer.After(delay, function()
        if not MainFrame or scheduleToken ~= MainFrame.logLiveRefreshScheduleToken then return end
        MainFrame.logLiveRefreshScheduled = false
        if not MainFrame.logLiveRefreshPending then return end
        if not MainFrame:IsShown() or currentTab ~= "log" then return end
        if MainFrame.logScan or (MainFrame.logState and MainFrame.logState.rendering) then
            return
        end

        local state = MainFrame.logState
        local requestedPage = state and state.page or 1
        local followLatest = state and state.page == state.totalPages or false
        MainFrame.logLiveRefreshPending = false
        MainFrame.logLastLiveRefreshAt = GetLiveRefreshClock()
        RefreshLogContent({ requestedPage = requestedPage, followLatest = followLatest })
    end)
end

local function RefreshVisibleLogRows()
    if not MainFrame or not MainFrame.logState then return end
    local scrollOffset = MainFrame.logScroll and MainFrame.logScroll:GetVerticalScroll() or 0
    RenderLogPage(MainFrame.logState.page, scrollOffset)
end

RenderLogPage = function(page, restoreScrollOffset)
    if not MainFrame or not MainFrame.logState or not MainFrame.logContent then return end
    local state = MainFrame.logState
    if state.renderToken ~= MainFrame.logRenderToken then return end

    page = max(1, min(tonumber(page) or 1, state.totalPages))
    state.page = page
    state.startIndex = ((page - 1) * state.pageSize) + 1
    state.endIndex = min(state.startIndex + state.pageSize - 1, state.totalMatched)
    state.displayCount = max(state.endIndex - state.startIndex + 1, 0)
    state.loadedCount = 0
    state.yOffset = 0
    state.rendering = true

    MainFrame.logPageRenderToken = (MainFrame.logPageRenderToken or 0) + 1
    local pageRenderToken = MainFrame.logPageRenderToken
    ResetLogViewport()
    UpdateLogLayoutWidth()
    SetLogNavigationState(state, false)
    UpdateLogPageStatus(state)

    local function RenderBatch()
        if not MainFrame or not MainFrame.logState then return end
        if pageRenderToken ~= MainFrame.logPageRenderToken then return end
        if state.renderToken ~= MainFrame.logRenderToken or MainFrame.logState ~= state then return end

        local batchEnd = min(state.loadedCount + LOG_RENDER_BATCH_SIZE, state.displayCount)
        for rowIndex = state.loadedCount + 1, batchEnd do
            local row = EnsureLogRow(MainFrame.logContent, rowIndex)
            if not row then break end
            row:ClearAllPoints()
            row:SetPoint("TOPLEFT", 0, -state.yOffset)
            row:SetPoint("TOPRIGHT", 0, -state.yOffset)

            local recordIndex = state.startIndex + rowIndex - 1
            local textHeight = select(1, RenderLogRow(row, state.records[recordIndex]))
            row:Show()
            state.yOffset = state.yOffset + textHeight + 6
            state.loadedCount = rowIndex
        end

        MainFrame.logShownRowCount = state.loadedCount
        MainFrame.logContent:SetHeight(max(state.yOffset, 1))
        if MainFrame.logScroll then MainFrame.logScroll:UpdateScrollChildRect() end
        UpdateLogPageStatus(state)

        if state.loadedCount < state.displayCount then
            C_Timer.After(0, RenderBatch)
            return
        end

        state.rendering = false
        SetLogNavigationState(state, false)
        UpdateLogPageStatus(state)
        SchedulePendingLiveRefresh()
        if restoreScrollOffset and restoreScrollOffset > 0 and MainFrame.logScroll then
            C_Timer.After(0, function()
                if not MainFrame or pageRenderToken ~= MainFrame.logPageRenderToken then return end
                local range = MainFrame.logScroll:GetVerticalScrollRange() or 0
                MainFrame.logScroll:SetVerticalScroll(min(restoreScrollOffset, range))
            end)
        end
    end

    C_Timer.After(0, RenderBatch)
end

local function CompleteLogScan(scan)
    if not MainFrame or scan.renderToken ~= MainFrame.logRenderToken then return end
    MainFrame.logScan = nil
    MainFrame.participantOptions = {
        speaker = scan.speakerOptions.list,
        listener = scan.listenerOptions.list,
    }
    InitParticipantDropdown(MainFrame.speakerDropdown, "speaker", currentFilter.speakers)
    InitParticipantDropdown(MainFrame.listenerDropdown, "listener", currentFilter.listeners)

    local totalMatched = #scan.matches
    MainFrame.debugRecentRecords = {}
    for i = max(totalMatched - 4, 1), totalMatched do
        if scan.matches[i] then
            MainFrame.debugRecentRecords[#MainFrame.debugRecentRecords + 1] = scan.matches[i]
        end
    end
    if totalMatched <= 0 then
        MainFrame.logState = nil
        ResetLogViewport()
        SetLogNavigationState(nil, false)
        UpdateLogPageStatus(nil)
        SchedulePendingLiveRefresh()
        return
    end

    local pageSize = GetLogPageSize()
    local totalPages = max(1, math.ceil(totalMatched / pageSize))
    local requestedPage = scan.followLatest and totalPages or scan.requestedPage
    requestedPage = max(1, min(requestedPage or 1, totalPages))
    MainFrame.logState = {
        records = scan.matches,
        totalMatched = totalMatched,
        totalPages = totalPages,
        page = requestedPage,
        pageSize = pageSize,
        renderToken = scan.renderToken,
    }
    RenderLogPage(requestedPage)
end

local function StartLogScan(renderToken, requestedPage, followLatest)
    local buckets, totalRecords = BuildChronologicalLogBuckets()
    local scan = {
        renderToken = renderToken,
        requestedPage = requestedPage or 1,
        followLatest = followLatest == true,
        buckets = buckets,
        totalRecords = totalRecords,
        bucketIndex = 1,
        recordIndex = 1,
        scanned = 0,
        matches = {},
        minTime = GetFilterMinTime(),
        searchLower = currentFilter.search:lower(),
        speakerOptions = { byKey = {}, list = {} },
        listenerOptions = { byKey = {}, list = {} },
    }
    MainFrame.logScan = scan
    SetLogNavigationState(nil, true)
    UpdateLogScanStatus(0, totalRecords, 0)

    local function ScanBatch()
        if not MainFrame or scan.renderToken ~= MainFrame.logRenderToken then return end
        if MainFrame.logScan ~= scan or currentTab ~= "log" then return end

        local processed = 0
        while processed < LOG_SCAN_BATCH_SIZE and scan.bucketIndex <= #scan.buckets do
            local records = scan.buckets[scan.bucketIndex]
            local record = records[scan.recordIndex]
            if record then
                IndexRecordParticipants(scan.speakerOptions, scan.listenerOptions, record)
                if RecordMatchesCurrentFilter(record, scan.minTime, scan.searchLower) then
                    scan.matches[#scan.matches + 1] = record
                end
                scan.recordIndex = scan.recordIndex + 1
                scan.scanned = scan.scanned + 1
                processed = processed + 1
            else
                scan.bucketIndex = scan.bucketIndex + 1
                scan.recordIndex = 1
            end
        end

        UpdateLogScanStatus(scan.scanned, scan.totalRecords, #scan.matches)
        if scan.bucketIndex <= #scan.buckets then
            C_Timer.After(0, ScanBatch)
        else
            CompleteLogScan(scan)
        end
    end

    C_Timer.After(0, ScanBatch)
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
RefreshLogContent = function(options)
    if not MainFrame or not MainFrame.logContent then return end
    options = options or {}

    UpdateFilterSummary()
    local previousState = MainFrame.logState
    local requestedPage = options.requestedPage
        or (options.preservePage and previousState and previousState.page)
        or 1
    local followLatest = options.followLatest == true
    InvalidateLogRender()
    local renderToken = MainFrame.logRenderToken
    UpdateLogLayoutWidth()
    MainFrame.logContent.rows = MainFrame.logContent.rows or {}
    MainFrame.logPlainText = nil
    ResetLogViewport()
    StartLogScan(renderToken, requestedPage, followLatest)
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
    local records = MainFrame.debugRecentRecords or {}
    local firstRecord = 1
    local debugIndex = 0
    for i = firstRecord, #records do
        local record = records[i]
        debugIndex = debugIndex + 1
        table.insert(lines, "")
        table.insert(lines, format("[记录 %d]", debugIndex))

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

    -- 有界分页设置（同时存活的日志行永不超过硬上限）
    yOffset = yOffset + 15
    if not content.viewWindowSizeTitle then
        content.viewWindowSizeTitle = content:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    end
    content.viewWindowSizeTitle:SetPoint("TOPLEFT", 10, -yOffset)
    content.viewWindowSizeTitle:SetText("每页回放条数:")
    content.viewWindowSizeTitle:Show()

    if not content.viewWindowSizeBox then
        local eb = CreateFrame("EditBox", nil, content, "InputBoxTemplate")
        eb:SetSize(56, 20)
        eb:SetAutoFocus(false)
        eb:SetNumeric(true)
        content.viewWindowSizeBox = eb
    end

    local viewWindowSizeBox = content.viewWindowSizeBox
    viewWindowSizeBox:SetPoint("LEFT", content.viewWindowSizeTitle, "RIGHT", 6, 0)
    viewWindowSizeBox:SetText(tostring(GetLogPageSize()))
    viewWindowSizeBox:SetScript("OnEnterPressed", function(self)
        local value = tonumber(self:GetText()) or LOG_VIEW_WINDOW_SIZE
        if value < LOG_MIN_PAGE_SIZE then value = LOG_MIN_PAGE_SIZE end
        if value > LOG_MAX_LIVE_ROWS then value = LOG_MAX_LIVE_ROWS end
        RPBox_Config.logViewWindowSize = floor(value)
        self:SetText(tostring(RPBox_Config.logViewWindowSize))

        if MainFrame and MainFrame:IsShown() and currentTab == "log" then
            RefreshLogContent()
        end

        self:ClearFocus()
    end)
    viewWindowSizeBox:SetScript("OnEscapePressed", function(self)
        self:SetText(tostring(GetLogPageSize()))
        self:ClearFocus()
    end)
    viewWindowSizeBox:Show()

    if not content.viewWindowSizeHint then
        content.viewWindowSizeHint = content:CreateFontString(nil, "OVERLAY", "GameFontDisableSmall")
    end
    content.viewWindowSizeHint:SetPoint("LEFT", viewWindowSizeBox, "RIGHT", 8, 0)
    content.viewWindowSizeHint:SetText("建议 80（范围 40-120，硬上限 120）")
    content.viewWindowSizeHint:Show()
    yOffset = yOffset + 26

    content:SetHeight(yOffset + 20)
    MainFrame.statusText:SetText("设置")
end

-- 切换标签页
local function SwitchTab(tabName)
    if not MainFrame then return end
    if currentTab == "log" and tabName ~= "log" then
        InvalidateLogRender()
    end
    currentTab = tabName

    -- 隐藏所有内容
    if MainFrame.logScroll then MainFrame.logScroll:Hide() end
    if MainFrame.listScroll then MainFrame.listScroll:Hide() end
    if MainFrame.debugScroll then MainFrame.debugScroll:Hide() end
    if MainFrame.settingsScroll then MainFrame.settingsScroll:Hide() end
    if MainFrame.filterFrame then MainFrame.filterFrame:Hide() end
    if MainFrame.ledgerHeader then MainFrame.ledgerHeader:Hide() end

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
        MainFrame.ledgerHeader:Show()
        MainFrame.logScroll:Show()
        InitDateDropdown()
        InitChannelDropdown()
        InitParticipantDropdown(MainFrame.speakerDropdown, "speaker", currentFilter.speakers)
        InitParticipantDropdown(MainFrame.listenerDropdown, "listener", currentFilter.listeners)
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
    MainFrame:SetSize(780, 520)
    MainFrame:SetPoint("CENTER")
    MainFrame:SetMovable(true)
    MainFrame:EnableMouse(true)
    MainFrame:RegisterForDrag("LeftButton")
    MainFrame:SetScript("OnDragStart", MainFrame.StartMoving)
    MainFrame:SetScript("OnDragStop", MainFrame.StopMovingOrSizing)
    MainFrame:Hide()
    MainFrame:HookScript("OnHide", function()
        InvalidateLogRender()
    end)
    MainFrame:SetScript("OnSizeChanged", function()
        UpdateLogLayoutWidth()
        if MainFrame:IsShown() and currentTab == "log" and MainFrame.logState then
            RefreshVisibleLogRows()
        end
    end)

    -- 启用调整大小
    MainFrame:SetResizable(true)
    MainFrame:SetResizeBounds(680, 500, 1200, 900)
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

    -- 档案筛选侧栏：发言者与收听视角分开，避免把不同角色卡混成一个人。
    local filterFrame = CreateFrame("Frame", nil, MainFrame, "BackdropTemplate")
    filterFrame:SetPoint("TOPLEFT", 12, -58)
    filterFrame:SetPoint("BOTTOMLEFT", 12, 40)
    filterFrame:SetWidth(178)
    if filterFrame.SetBackdrop then
        filterFrame:SetBackdrop({
            bgFile = "Interface\\Buttons\\WHITE8X8",
            edgeFile = "Interface\\Buttons\\WHITE8X8",
            edgeSize = 1,
        })
        filterFrame:SetBackdropColor(0.035, 0.045, 0.06, 0.82)
        filterFrame:SetBackdropBorderColor(0.25, 0.31, 0.4, 0.8)
    end

    local archiveTitle = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    archiveTitle:SetPoint("TOPLEFT", 12, -10)
    archiveTitle:SetText("档案筛选")

    local archiveHint = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontDisableSmall")
    archiveHint:SetPoint("TOPLEFT", archiveTitle, "BOTTOMLEFT", 0, -3)
    archiveHint:SetText("多选条件取交集")

    local dateLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    dateLabel:SetPoint("TOPLEFT", 12, -50)
    dateLabel:SetText("时间范围")
    local dateDropdown = CreateFrame("Frame", "RPBoxDateDropdown", filterFrame, "UIDropDownMenuTemplate")
    dateDropdown:SetPoint("TOPLEFT", -4, -62)
    UIDropDownMenu_SetWidth(dateDropdown, 142)

    local speakerLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    speakerLabel:SetPoint("TOPLEFT", 12, -100)
    speakerLabel:SetText("发言者 / 人物卡")
    local speakerDropdown = CreateFrame("Frame", "RPBoxSpeakerDropdown", filterFrame, "UIDropDownMenuTemplate")
    speakerDropdown:SetPoint("TOPLEFT", -4, -112)
    UIDropDownMenu_SetWidth(speakerDropdown, 142)

    local listenerLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    listenerLabel:SetPoint("TOPLEFT", 12, -150)
    listenerLabel:SetText("收听者 / 记录视角")
    local listenerDropdown = CreateFrame("Frame", "RPBoxListenerDropdown", filterFrame, "UIDropDownMenuTemplate")
    listenerDropdown:SetPoint("TOPLEFT", -4, -162)
    UIDropDownMenu_SetWidth(listenerDropdown, 142)

    local channelLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    channelLabel:SetPoint("TOPLEFT", 12, -200)
    channelLabel:SetText("频道 / 节点")
    local channelDropdown = CreateFrame("Frame", "RPBoxChannelDropdown", filterFrame, "UIDropDownMenuTemplate")
    channelDropdown:SetPoint("TOPLEFT", -4, -212)
    UIDropDownMenu_SetWidth(channelDropdown, 142)

    local searchLabel = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    searchLabel:SetPoint("TOPLEFT", 12, -252)
    searchLabel:SetText("全文与历史姓名")
    local searchBox = CreateFrame("EditBox", nil, filterFrame, "InputBoxTemplate")
    searchBox:SetSize(150, 22)
    searchBox:SetPoint("TOPLEFT", 12, -268)
    searchBox:SetAutoFocus(false)
    searchBox:SetText(currentFilter.search)
    searchBox:SetScript("OnEnterPressed", function(self)
        currentFilter.search = strtrim(self:GetText() or "")
        self:SetText(currentFilter.search)
        UpdateFilterSummary()
        RefreshLogContent()
        self:ClearFocus()
    end)
    searchBox:SetScript("OnEscapePressed", function(self)
        self:SetText(currentFilter.search)
        self:ClearFocus()
    end)

    local summaryTitle = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    summaryTitle:SetPoint("TOPLEFT", 12, -306)
    summaryTitle:SetText("已启用条件")
    local filterSummary = filterFrame:CreateFontString(nil, "OVERLAY", "GameFontDisableSmall")
    filterSummary:SetPoint("TOPLEFT", summaryTitle, "BOTTOMLEFT", 0, -6)
    filterSummary:SetPoint("RIGHT", filterFrame, "RIGHT", -12, 0)
    filterSummary:SetJustifyH("LEFT")
    filterSummary:SetJustifyV("TOP")
    filterSummary:SetWordWrap(true)

    local clearFilterBtn = CreateFrame("Button", nil, filterFrame, "UIPanelButtonTemplate")
    clearFilterBtn:SetSize(150, 22)
    clearFilterBtn:SetPoint("BOTTOM", 0, 10)
    clearFilterBtn:SetText("清除全部筛选")
    clearFilterBtn:SetScript("OnClick", function()
        currentFilter.days = nil
        wipe(currentFilter.channels)
        wipe(currentFilter.speakers)
        wipe(currentFilter.listeners)
        currentFilter.search = ""
        searchBox:SetText("")
        InitDateDropdown()
        InitChannelDropdown()
        InitParticipantDropdown(speakerDropdown, "speaker", currentFilter.speakers)
        InitParticipantDropdown(listenerDropdown, "listener", currentFilter.listeners)
        UpdateFilterSummary()
        RefreshLogContent()
    end)

    MainFrame.filterFrame = filterFrame
    MainFrame.dateDropdown = dateDropdown
    MainFrame.speakerDropdown = speakerDropdown
    MainFrame.listenerDropdown = listenerDropdown
    MainFrame.channelDropdown = channelDropdown
    MainFrame.searchBox = searchBox
    MainFrame.filterSummary = filterSummary
    MainFrame.clearFilterBtn = clearFilterBtn

    local ledgerHeader = CreateFrame("Frame", nil, MainFrame)
    ledgerHeader:SetPoint("TOPLEFT", 205, -58)
    ledgerHeader:SetPoint("TOPRIGHT", -30, -58)
    ledgerHeader:SetHeight(24)
    local ledgerTitle = ledgerHeader:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    ledgerTitle:SetPoint("LEFT", 0, 0)
    ledgerTitle:SetText("时间账本 · 正序")

    local latestPageBtn = CreateFrame("Button", nil, ledgerHeader, "UIPanelButtonTemplate")
    latestPageBtn:SetSize(48, 20)
    latestPageBtn:SetPoint("RIGHT", 0, 0)
    latestPageBtn:SetText("最新")
    latestPageBtn:SetScript("OnClick", function()
        if MainFrame.logState then RenderLogPage(MainFrame.logState.totalPages) end
    end)

    local nextPageBtn = CreateFrame("Button", nil, ledgerHeader, "UIPanelButtonTemplate")
    nextPageBtn:SetSize(66, 20)
    nextPageBtn:SetPoint("RIGHT", latestPageBtn, "LEFT", -4, 0)
    nextPageBtn:SetText("继续 →")
    nextPageBtn:SetScript("OnClick", function()
        if MainFrame.logState then RenderLogPage(MainFrame.logState.page + 1) end
    end)

    local pageProgress = ledgerHeader:CreateFontString(nil, "OVERLAY", "GameFontDisableSmall")
    pageProgress:SetWidth(72)
    pageProgress:SetPoint("RIGHT", nextPageBtn, "LEFT", -4, 0)
    pageProgress:SetJustifyH("CENTER")
    pageProgress:SetText("0 / 0 页")

    local prevPageBtn = CreateFrame("Button", nil, ledgerHeader, "UIPanelButtonTemplate")
    prevPageBtn:SetSize(62, 20)
    prevPageBtn:SetPoint("RIGHT", pageProgress, "LEFT", -4, 0)
    prevPageBtn:SetText("← 较早")
    prevPageBtn:SetScript("OnClick", function()
        if MainFrame.logState then RenderLogPage(MainFrame.logState.page - 1) end
    end)

    MainFrame.ledgerHeader = ledgerHeader
    MainFrame.prevPageBtn = prevPageBtn
    MainFrame.nextPageBtn = nextPageBtn
    MainFrame.latestPageBtn = latestPageBtn
    MainFrame.pageProgress = pageProgress
    SetLogNavigationState(nil, false)

    -- 日志滚动框架
    local logScroll = CreateFrame("ScrollFrame", nil, MainFrame, "UIPanelScrollFrameTemplate")
    logScroll:SetPoint("TOPLEFT", 205, -84)
    logScroll:SetPoint("BOTTOMRIGHT", -30, 40)
    logScroll:SetScript("OnVerticalScroll", function(self, offset)
        if self._rpboxAdjustingScroll then return end

        local range = self:GetVerticalScrollRange() or 0
        local clamped = offset
        if offset < 0 then
            clamped = 0
        elseif offset > range then
            clamped = range
        end

        local current = self:GetVerticalScroll() or 0
        if abs(current - clamped) > 0.1 then
            self._rpboxAdjustingScroll = true
            self:SetVerticalScroll(clamped)
            self._rpboxAdjustingScroll = nil
        end
    end)

    local logContent = CreateFrame("Frame", nil, logScroll)
    logContent:SetSize(480, 1)
    logScroll:SetScrollChild(logContent)

    MainFrame.logScroll = logScroll
    MainFrame.logContent = logContent
    UpdateLogLayoutWidth()

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
        if currentTab == "log" then
            local state = MainFrame.logState
            local scan = MainFrame.logScan
            RefreshLogContent({
                requestedPage = state and state.page or (scan and scan.requestedPage) or 1,
                followLatest = (state and state.page == state.totalPages) or (scan and scan.followLatest) or false,
            })
        else
            SwitchTab(currentTab)
        end
    end)

    -- 复制按钮
    local copyBtn = CreateFrame("Button", nil, MainFrame, "UIPanelButtonTemplate")
    copyBtn:SetSize(60, 22)
    copyBtn:SetPoint("RIGHT", refreshBtn, "LEFT", -5, 0)
    copyBtn:SetText("复制")
    copyBtn:SetScript("OnClick", function()
        if not MainFrame.logState or not MainFrame.logState.records or MainFrame.logState.displayCount <= 0 then
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
            dialog.TitleText:SetText("复制当前页 (Ctrl+A 全选, Ctrl+C 复制)")

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
            local copyLines = {}
            local state = MainFrame.logState
            for i = state.startIndex, state.endIndex do
                local _, plainText = BuildLogLineTexts(state.records[i])
                copyLines[#copyLines + 1] = plainText
            end
            local text = table.concat(copyLines, "\n")
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
        if ns.ClearRecords then
            ns.ClearRecords(true, true)
        else
            RPBox_ChatLog = {}
            if ns.UpdateSyncState then ns.UpdateSyncState() end
        end
        if MainFrame and MainFrame:IsShown() and currentTab == "log" then
            RefreshLogContent()
        end
    end,
    timeout = 0,
    whileDead = true,
    hideOnEscape = true,
    preferredIndex = 3,
}

-- 打开主界面
local function ApplyArchivePreset(preset)
    if type(preset) ~= "table" then return end
    if preset.reset then
        currentFilter.days = nil
        wipe(currentFilter.channels)
        wipe(currentFilter.speakers)
        wipe(currentFilter.listeners)
        currentFilter.search = ""
    end

    if preset.datePreset == "all" then
        currentFilter.days = nil
    elseif type(preset.datePreset) == "number" then
        currentFilter.days = preset.datePreset
    end
    if type(preset.search) == "string" then currentFilter.search = preset.search end

    local selectionPresets = {
        { source = preset.channels, target = currentFilter.channels },
        { source = preset.speakers, target = currentFilter.speakers },
        { source = preset.listeners, target = currentFilter.listeners },
    }
    for _, selection in ipairs(selectionPresets) do
        if type(selection.source) == "table" then
            wipe(selection.target)
            for _, value in ipairs(selection.source) do selection.target[value] = true end
        end
    end
end

function ns.OpenMainFrame(preset)
    ApplyArchivePreset(preset)
    local frame = CreateMainFrame()
    if MainFrame.searchBox then MainFrame.searchBox:SetText(currentFilter.search) end
    SwitchTab("log")
    frame:Show()
end

-- 关闭主界面
function ns.CloseMainFrame()
    if MainFrame then
        InvalidateLogRender()
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
        MainFrame.logLiveRefreshPending = true
        SchedulePendingLiveRefresh()
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
