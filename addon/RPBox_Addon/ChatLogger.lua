-- RPBox ChatLogger
-- 聊天记录采集模块

local ADDON_NAME, ns = ...
local L = ns.L or {}

-- 监听的聊天频道
local CHAT_EVENTS = {
    "CHAT_MSG_SAY",
    "CHAT_MSG_YELL",
    "CHAT_MSG_EMOTE",
    "CHAT_MSG_PARTY",
    "CHAT_MSG_PARTY_LEADER",
    "CHAT_MSG_RAID",
    "CHAT_MSG_RAID_LEADER",
    "CHAT_MSG_WHISPER",
    "CHAT_MSG_WHISPER_INFORM",
    "CHAT_MSG_GUILD",
}

-- 频道简写映射
local CHANNEL_SHORT = {
    CHAT_MSG_SAY = "SAY",
    CHAT_MSG_YELL = "YELL",
    CHAT_MSG_EMOTE = "EMOTE",
    CHAT_MSG_PARTY = "PARTY",
    CHAT_MSG_PARTY_LEADER = "PARTY",
    CHAT_MSG_RAID = "RAID",
    CHAT_MSG_RAID_LEADER = "RAID",
    CHAT_MSG_WHISPER = "WHISPER_IN",
    CHAT_MSG_WHISPER_INFORM = "WHISPER_OUT",
    CHAT_MSG_GUILD = "GUILD",
}

-- 获取 TRP3 角色信息并缓存
local function GetTRP3InfoAndCache(unitID)
    if not TRP3_API or not TRP3_API.register then return nil, nil end
    if not TRP3_API.register.isUnitIDKnown(unitID) then return nil, nil end

    local character = TRP3_API.register.getUnitIDCharacter(unitID)
    if not character or not character.profileID then return nil, nil end

    local profileID = character.profileID
    local profile = TRP3_API.register.getProfile(profileID)
    if not profile or not profile.player then return nil, nil end

    -- 缓存完整角色卡数据
    ns.CacheProfile(profileID, profile.player)

    return profileID, profile.player
end

-- 获取自己的 TRP3 信息并缓存
local function GetSelfTRP3InfoAndCache()
    if not TRP3_API or not TRP3_API.profile then return nil, nil end

    local profileID = TRP3_API.profile.getPlayerCurrentProfileID()
    local player = TRP3_API.profile.getData("player")
    if not player then return nil, nil end

    -- 缓存完整角色卡数据
    ns.CacheProfile(profileID, player)

    return profileID, player
end

-- 检查频道是否启用
local function IsChannelEnabled(channelShort)
    local channels = RPBox_Config and RPBox_Config.channels
    if not channels then return true end  -- 默认全部启用
    local enabled = channels[channelShort]
    if enabled == nil then return true end  -- 未配置的默认启用
    return enabled
end

-- 判断是否应该记录
local function ShouldRecord(unitID, isFromSelf, channelShort)
    -- 先检查频道是否启用
    if not IsChannelEnabled(channelShort) then return false end

    if isFromSelf then return true end
    if ns.IsBlacklisted(unitID) then return false end
    local profileID = GetTRP3InfoAndCache(unitID)
    if profileID then return true end
    if ns.IsWhitelisted(unitID) then return true end
    return false
end

-- 检查记录上限
local function CheckRecordLimit()
    local count = ns.GetTotalRecordCount()
    local threshold = RPBox_Config.warnThreshold or 9000

    if count >= threshold and not RPBox_Config.warnedThisSession then
        print(format(L["RECORD_WARNING"] or "|cFFFFFF00[RPBox]|r 聊天记录已达 %d 条", count))
        print("|cFFFFFF00[RPBox]|r 建议 /reload 后在客户端导出并清理")
        RPBox_Config.warnedThisSession = true
    end
end

-- 保存聊天记录
local function SaveChatLog(record)
    local timestamp = record.t or record.timestamp
    local dateStr = date("%Y-%m-%d", timestamp)
    local hourStr = date("%H", timestamp)

    RPBox_ChatLog[dateStr] = RPBox_ChatLog[dateStr] or {}
    RPBox_ChatLog[dateStr][hourStr] = RPBox_ChatLog[dateStr][hourStr] or {}

    table.insert(RPBox_ChatLog[dateStr][hourStr], record)

    -- 更新同步状态
    ns.UpdateSyncState()

    -- 触发新消息回调（用于自动刷新面板）
    ns.TriggerOnNewMessage()

    -- 检查记录上限
    CheckRecordLimit()
end

-- 解析 NPC/旁白消息
-- 返回: mk, npcName, message, npcType
local function ParseNPCMessage(content)
    if not content:match("^|") then return nil end
    -- 跳过 WoW 颜色代码 |cFFxxxxxx 开头的情况
    if content:match("^|c") then return nil end

    local text = content:sub(2):match("^%s*(.+)")
    if not text then return nil end

    -- 清理末尾的颜色代码 |r
    text = text:gsub("|r%s*$", "")

    -- 悄悄说
    local npcName, message = text:match("^(.-)%s*悄悄说%s*[：:]%s*(.*)$")
    if npcName and message then
        message = message:gsub("|r%s*$", "")
        return "N", npcName ~= "" and npcName or nil, message, "whisper"
    end
    -- 喊
    npcName, message = text:match("^(.-)%s*喊%s*[：:]%s*(.*)$")
    if npcName and message then
        message = message:gsub("|r%s*$", "")
        return "N", npcName ~= "" and npcName or nil, message, "yell"
    end
    -- 说
    npcName, message = text:match("^(.-)%s*说%s*[：:]%s*(.*)$")
    if npcName and message then
        message = message:gsub("|r%s*$", "")
        return "N", npcName ~= "" and npcName or nil, message, "say"
    end
    -- 旁白
    return "B", nil, text, nil
end

-- 聊天消息处理
local function OnChatMessage(self, event, msg, sender, ...)
    local playerID = ns.GetPlayerID()
    local senderID = sender

    if not senderID:find("-") then
        senderID = senderID .. "-" .. GetRealmName()
    end

    local isFromSelf = (senderID == playerID)
    local channelShort = CHANNEL_SHORT[event] or event

    if not ShouldRecord(senderID, isFromSelf, channelShort) then
        return false
    end

    -- 获取发送者GUID和职业
    local senderGUID = select(12, ...)
    local senderClass = nil
    if senderGUID then
        local _, classFilename = GetPlayerInfoByGUID(senderGUID)
        senderClass = classFilename
    end

    -- 获取 profileID
    local profileID
    if isFromSelf then
        profileID = GetSelfTRP3InfoAndCache()
        if not senderClass then
            local _, classFilename = UnitClass("player")
            senderClass = classFilename
        end
    else
        profileID = GetTRP3InfoAndCache(senderID)
    end

    -- 解析消息类型
    local mk, npcName, parsedMsg, npcType = ParseNPCMessage(msg)
    if not mk then
        mk = "P"  -- 普通玩家消息
    end

    -- 构建记录
    local record = {
        t = time(),
        c = CHANNEL_SHORT[event] or event,
        m = parsedMsg or msg,
        mk = mk,
        s = senderID,
        ref = profileID,
    }

    -- 保存职业信息
    if senderClass then
        record.cls = senderClass
    end

    -- NPC 消息添加 npc 和 nt 字段
    if mk == "N" then
        if npcName then
            local cleanName = npcName:gsub("^|%s*", "")
            if cleanName ~= "" then record.npc = cleanName end
        end
        if npcType then record.nt = npcType end
    end

    SaveChatLog(record)
    return false
end

-- 注册聊天事件监听
for _, event in ipairs(CHAT_EVENTS) do
    ChatFrame_AddMessageEventFilter(event, OnChatMessage)
end
