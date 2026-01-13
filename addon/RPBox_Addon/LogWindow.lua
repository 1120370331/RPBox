-- RPBox LogWindow
-- 游戏内聊天回放窗口

local ADDON_NAME, ns = ...
local L = ns.L or {}

-- 窗口引用
local LogFrame = nil

-- 当前筛选条件
local currentFilter = {
    date = nil,      -- nil = 全部
    channel = nil,   -- nil = 全部
    search = "",
}

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

-- 获取 TRP3 显示名称
local function GetDisplayName(sender)
    if sender.trp3 then
        local name = sender.trp3.FN or ""
        if sender.trp3.LN then
            name = name .. " " .. sender.trp3.LN
        end
        if name ~= "" then return name end
    end
    return strsplit("-", sender.gameID)
end

-- 获取所有日期列表
local function GetDateList()
    local dates = {}
    for date in pairs(RPBox_ChatLog or {}) do
        table.insert(dates, date)
    end
    table.sort(dates, function(a, b) return a > b end)
    return dates
end

-- 获取筛选后的记录
local function GetFilteredRecords()
    local records = {}
    local chatLog = RPBox_ChatLog or {}

    for dateStr, hours in pairs(chatLog) do
        -- 日期筛选
        if not currentFilter.date or currentFilter.date == dateStr then
            for hourStr, hourRecords in pairs(hours) do
                for _, record in ipairs(hourRecords) do
                    -- 频道筛选
                    local channelMatch = not currentFilter.channel or record.channel == currentFilter.channel
                    -- 搜索筛选
                    local searchMatch = currentFilter.search == "" or
                        record.content:lower():find(currentFilter.search:lower(), 1, true)

                    if channelMatch and searchMatch then
                        table.insert(records, record)
                    end
                end
            end
        end
    end

    -- 按时间排序
    table.sort(records, function(a, b) return a.timestamp > b.timestamp end)
    return records
end

-- 创建窗口
local function CreateLogFrame()
    if LogFrame then return LogFrame end

    -- 主窗口
    LogFrame = CreateFrame("Frame", "RPBoxLogFrame", UIParent, "BasicFrameTemplateWithInset")
    LogFrame:SetSize(500, 400)
    LogFrame:SetPoint("CENTER")
    LogFrame:SetMovable(true)
    LogFrame:EnableMouse(true)
    LogFrame:RegisterForDrag("LeftButton")
    LogFrame:SetScript("OnDragStart", LogFrame.StartMoving)
    LogFrame:SetScript("OnDragStop", LogFrame.StopMovingOrSizing)
    LogFrame:Hide()

    LogFrame.TitleText:SetText("RPBox 聊天回放")

    -- 滚动框架
    local scrollFrame = CreateFrame("ScrollFrame", nil, LogFrame, "UIPanelScrollFrameTemplate")
    scrollFrame:SetPoint("TOPLEFT", 12, -60)
    scrollFrame:SetPoint("BOTTOMRIGHT", -30, 40)

    local content = CreateFrame("Frame", nil, scrollFrame)
    content:SetSize(440, 1)
    scrollFrame:SetScrollChild(content)

    LogFrame.scrollFrame = scrollFrame
    LogFrame.content = content

    -- 底部状态栏
    local statusText = LogFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    statusText:SetPoint("BOTTOMLEFT", 12, 12)
    LogFrame.statusText = statusText

    return LogFrame
end

-- 刷新内容显示
local function RefreshContent()
    if not LogFrame then return end

    local content = LogFrame.content
    -- 清空现有内容
    for _, child in pairs({content:GetChildren()}) do
        child:Hide()
    end

    local records = GetFilteredRecords()
    local yOffset = 0

    for i, record in ipairs(records) do
        if i > 100 then break end -- 限制显示数量

        local row = content.rows and content.rows[i]
        if not row then
            row = CreateFrame("Frame", nil, content)
            row:SetHeight(40)
            content.rows = content.rows or {}
            content.rows[i] = row
        end

        row:SetPoint("TOPLEFT", 0, -yOffset)
        row:SetPoint("TOPRIGHT", 0, -yOffset)

        -- 时间和发送者
        if not row.header then
            row.header = row:CreateFontString(nil, "OVERLAY", "GameFontNormal")
            row.header:SetPoint("TOPLEFT", 0, 0)
        end
        local timeStr = date("%H:%M", record.timestamp)
        local name = GetDisplayName(record.sender)
        local channel = CHANNEL_NAMES[record.channel] or ""
        row.header:SetText(format("|cFF00FF00%s|r [%s] |cFFFFD100%s|r", timeStr, channel, name))

        -- 内容
        if not row.text then
            row.text = row:CreateFontString(nil, "OVERLAY", "GameFontHighlight")
            row.text:SetPoint("TOPLEFT", 10, -16)
            row.text:SetWidth(420)
            row.text:SetJustifyH("LEFT")
        end
        row.text:SetText(record.content)

        row:Show()
        yOffset = yOffset + 44
    end

    content:SetHeight(math.max(yOffset, 1))
    LogFrame.statusText:SetText(format("共 %d 条记录", #records))
end

-- 打开回放窗口
function ns.OpenLogWindow(todayOnly)
    local frame = CreateLogFrame()

    if todayOnly then
        currentFilter.date = date("%Y-%m-%d")
    else
        currentFilter.date = nil
    end

    RefreshContent()
    frame:Show()
end
