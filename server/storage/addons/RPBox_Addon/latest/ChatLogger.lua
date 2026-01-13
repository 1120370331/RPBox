-- RPBox ChatLogger
-- 聊天记录采集模块（优化版数据结构）

RPBox.ChatLogger = {}
local ChatLogger = RPBox.ChatLogger

-- 监听的聊天频道
local CHAT_EVENTS = {
    "CHAT_MSG_SAY",
    "CHAT_MSG_YELL",
    "CHAT_MSG_EMOTE",
    "CHAT_MSG_TEXT_EMOTE",
    "CHAT_MSG_PARTY",
    "CHAT_MSG_PARTY_LEADER",
    "CHAT_MSG_RAID",
    "CHAT_MSG_RAID_LEADER",
}

-- 频道简写映射
local CHANNEL_SHORT = {
    ["CHAT_MSG_SAY"] = "SAY",
    ["CHAT_MSG_YELL"] = "YELL",
    ["CHAT_MSG_EMOTE"] = "EMOTE",
    ["CHAT_MSG_TEXT_EMOTE"] = "EMOTE",
    ["CHAT_MSG_PARTY"] = "PARTY",
    ["CHAT_MSG_PARTY_LEADER"] = "PARTY",
    ["CHAT_MSG_RAID"] = "RAID",
    ["CHAT_MSG_RAID_LEADER"] = "RAID",
    ["CHAT_MSG_WHISPER"] = "WHISPER",
    ["CHAT_MSG_WHISPER_INFORM"] = "WHISPER",
}

-- 记录上限
local MAX_RECORDS = 10000
local WARN_THRESHOLD = 9000

-- 自动白名单计时器
local targetTimer = nil

-- 初始化
function ChatLogger:Init()
    for _, event in ipairs(CHAT_EVENTS) do
        ChatFrame_AddMessageEventFilter(event, function(_, _, msg, sender, ...)
            self:OnChatMessage(event, msg, sender)
            return false
        end)
    end

    local frame = CreateFrame("Frame")
    frame:RegisterEvent("PLAYER_TARGET_CHANGED")
    frame:SetScript("OnEvent", function()
        self:OnTargetChanged()
    end)

    print("|cFF00FF00[RPBox]|r 聊天记录模块已启动")
end

-- 处理聊天消息
function ChatLogger:OnChatMessage(event, msg, sender)
    local unitID = sender
    local isFromSelf = (unitID == UnitName("player") .. "-" .. GetRealmName())

    if not self:ShouldRecord(unitID, isFromSelf) then
        return
    end

    -- 解析消息类型和内容
    local mark, npcName, content = self:ParseMessage(msg, event)

    -- 获取TRP3信息并缓存
    local profileID = self:CacheProfile(unitID)

    -- 保存记录（新格式）
    self:SaveRecord({
        t = time(),
        c = CHANNEL_SHORT[event] or event,
        m = content,
        mk = mark,
        s = unitID,
        ref = profileID,
        npc = npcName,
    })
end

-- 解析消息类型（P/N/B）
function ChatLogger:ParseMessage(msg, event)
    -- 检测 TRP3 NPC 语法: |NPC名字
    local npcName = msg:match("^|([^|]+)|")
    if npcName then
        local content = msg:gsub("^|[^|]+|%s*", "")
        return "N", npcName, content
    end

    -- 检测旁白（通常是纯表情且无说话者特征）
    if event == "CHAT_MSG_EMOTE" or event == "CHAT_MSG_TEXT_EMOTE" then
        -- 简单判断：如果消息以动作描述开头，视为旁白
        if msg:match("^[%*%[（(]") then
            return "B", nil, msg
        end
    end

    -- 默认为玩家消息
    return "P", nil, msg
end

-- 缓存角色卡并返回profileID
function ChatLogger:CacheProfile(unitID)
    if not TRP3_API or not TRP3_API.register then
        return nil
    end

    local character = TRP3_API.register.getUnitIDCharacter(unitID)
    if not character or not character.profileID then
        return nil
    end

    local profileID = character.profileID
    local profile = TRP3_API.register.getProfile(profileID)
    if not profile or not profile.player then
        return profileID
    end

    -- 缓存角色卡数据
    local chars = profile.player.characteristics or {}
    RPBox_ProfileCache[profileID] = {
        v = 1,
        FN = chars.FN,
        LN = chars.LN,
        TI = chars.TI,
        FT = chars.FT,
        RA = chars.RA,
        CL = chars.CL,
        AG = chars.AG,
        EC = chars.EC,
        HE = chars.HE,
        WE = chars.WE,
        BP = chars.BP,
        RE = chars.RE,
        IC = chars.IC,
        CH = chars.CH,
    }

    return profileID
end

-- 判断是否应该记录
function ChatLogger:ShouldRecord(unitID, isFromSelf)
    if isFromSelf then return true end
    if self:IsBlacklisted(unitID) then return false end
    if self:GetProfileID(unitID) then return true end
    if RPBox_Config.whitelist[unitID] then return true end
    return false
end

-- 获取profileID
function ChatLogger:GetProfileID(unitID)
    if not TRP3_API or not TRP3_API.register then
        return nil
    end
    local character = TRP3_API.register.getUnitIDCharacter(unitID)
    return character and character.profileID
end

-- 检查是否在黑名单
function ChatLogger:IsBlacklisted(unitID)
    if RPBox_Config.blacklist[unitID] then return true end
    if C_FriendList and C_FriendList.IsIgnored then
        local name = unitID:match("^([^-]+)")
        if C_FriendList.IsIgnored(name) then return true end
    end
    if TRP3_API and TRP3_API.register and TRP3_API.register.isIDIgnored then
        if TRP3_API.register.isIDIgnored(unitID) then return true end
    end
    return false
end

-- 保存聊天记录（新格式）
function ChatLogger:SaveRecord(record)
    local dateStr = date("%Y-%m-%d", record.t)
    local hourStr = date("%H", record.t)

    RPBox_ChatLog[dateStr] = RPBox_ChatLog[dateStr] or {}
    RPBox_ChatLog[dateStr][hourStr] = RPBox_ChatLog[dateStr][hourStr] or {}

    -- 清理nil字段
    if not record.npc then record.npc = nil end
    if not record.ref then record.ref = nil end

    table.insert(RPBox_ChatLog[dateStr][hourStr], record)
    RPBox:UpdateSyncState()
    self:CheckRecordLimit()
end

-- 检查记录上限
function ChatLogger:CheckRecordLimit()
    local count = RPBox:GetTotalRecordCount()
    if count >= WARN_THRESHOLD and not RPBox_Config.warnedThisSession then
        print("|cFFFFFF00[RPBox]|r 聊天记录已达 " .. count .. " 条")
        print("|cFFFFFF00[RPBox]|r 建议 /reload 后在客户端导出并清理")
        RPBox_Config.warnedThisSession = true
    end
end

-- 目标变化处理（自动白名单）
function ChatLogger:OnTargetChanged()
    if targetTimer then
        targetTimer:Cancel()
        targetTimer = nil
    end

    if not UnitExists("target") or not UnitIsPlayer("target") then
        return
    end

    local unitID = UnitName("target")
    local realm = GetRealmName()
    if unitID then
        unitID = unitID .. "-" .. realm
    end

    if not unitID or RPBox_Config.whitelist[unitID] then
        return
    end

    targetTimer = C_Timer.NewTimer(2, function()
        if UnitExists("target") then
            local currentTarget = UnitName("target") .. "-" .. realm
            if currentTarget == unitID then
                RPBox_Config.whitelist[unitID] = true
                print("|cFF00FF00[RPBox]|r " .. unitID .. " 已加入记录白名单")
            end
        end
        targetTimer = nil
    end)
end
