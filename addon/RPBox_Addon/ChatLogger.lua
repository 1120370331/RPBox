-- RPBox ChatLogger
-- 聊天记录采集模块

local ADDON_NAME, ns = ...
local L = ns.L or {}

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
    "CHAT_MSG_WHISPER",
    "CHAT_MSG_WHISPER_INFORM",
    "CHAT_MSG_GUILD",
}

-- 频道简写映射
local CHANNEL_SHORT = {
    CHAT_MSG_SAY = "SAY",
    CHAT_MSG_YELL = "YELL",
    CHAT_MSG_EMOTE = "EMOTE",
    CHAT_MSG_TEXT_EMOTE = "TEXT_EMOTE",
    CHAT_MSG_PARTY = "PARTY",
    CHAT_MSG_PARTY_LEADER = "PARTY",
    CHAT_MSG_RAID = "RAID",
    CHAT_MSG_RAID_LEADER = "RAID",
    CHAT_MSG_WHISPER = "WHISPER_IN",
    CHAT_MSG_WHISPER_INFORM = "WHISPER_OUT",
    CHAT_MSG_GUILD = "GUILD",
}

-- 去重缓存（基于 chat lineID / 消息指纹）
local MESSAGE_DEDUPE_WINDOW = 5
local messageCache = {}

local function NormalizeMessageForSignature(msg)
    if not msg or msg == "" then return "" end
    return msg:gsub("^%s+", ""):gsub("%s+$", "")
end

local function CleanupMessageCache(now)
    for signature, cachedTime in pairs(messageCache) do
        if (now - cachedTime) > MESSAGE_DEDUPE_WINDOW then
            messageCache[signature] = nil
        end
    end
end

local function BuildMessageSignature(sender, channelShort, msg, lineID)
    if lineID and lineID ~= 0 and tostring(lineID) ~= "" then
        return "line:" .. tostring(lineID)
    end
    return table.concat({
        sender or "",
        channelShort or "",
        NormalizeMessageForSignature(msg),
    }, "\31")
end

-- 检查消息是否重复
local function IsDuplicateMessage(sender, channelShort, msg, lineID)
    local now = GetTimePreciseSec and GetTimePreciseSec() or time()
    CleanupMessageCache(now)

    local signature = BuildMessageSignature(sender, channelShort, msg, lineID)
    if messageCache[signature] then
        return true
    end

    messageCache[signature] = now
    return false
end

-- 获取 TRP3 角色信息并缓存
local function GetTRP3InfoAndCache(unitID)
    local context = ns.GetRemoteProfileContext and ns.GetRemoteProfileContext(unitID) or nil
    if not context then return nil, nil, nil end

    -- 缓存完整角色卡数据
    ns.CacheProfile(context.profileID, context.profile)
    local snapshotKey = nil
    if ns.ObserveRemoteProfileIdentity then
        snapshotKey = ns.ObserveRemoteProfileIdentity(unitID, context.profileID, context.profile, true)
    elseif ns.CaptureProfileSnapshot then
        snapshotKey = ns.CaptureProfileSnapshot(context.profileID, context.profile, unitID)
    end

    return context.profileID, context.profile, snapshotKey
end

-- 获取自己的 TRP3 信息并缓存
local function GetSelfTRP3InfoAndCache()
    local context = ns.GetSelfProfileContext and ns.GetSelfProfileContext() or nil
    if not context then return nil, nil, nil end

    -- 缓存完整角色卡数据
    ns.CacheProfile(context.profileID, context.profile)
    local snapshotKey = nil
    if ns.CaptureProfileSnapshot then
        snapshotKey = ns.CaptureProfileSnapshot(
            context.profileID,
            context.root or context.profile,
            context.gameID,
            context.profileName
        )
    end

    return context.profileID, context.profile, snapshotKey
end

-- 获取角色的 TRP3 显示名称（纯文本，不含格式代码）
local function GetTRP3DisplayName(unitID, isSelf)
    local profileID, profile

    if isSelf then
        profileID, profile = GetSelfTRP3InfoAndCache()
    else
        profileID, profile = GetTRP3InfoAndCache(unitID)
    end

    if not profile or not profile.characteristics then
        return nil
    end

    -- 只返回纯文本名字
    local name = profile.characteristics.FN or unitID:match("^([^-]+)")
    return name
end

-- 转义 Lua 模式中的特殊字符
local function EscapePattern(str)
    return str:gsub("([%^%$%(%)%%%.%[%]%*%+%-%?])", "%%%1")
end

-- 替换表情消息中的角色名为 TRP3 人物卡名称
local function ReplaceEmoteNames(msg, senderID, listenerID)
    if not msg or msg == "" then return msg end

    -- 获取游戏角色名（不带服务器）
    local senderName = senderID:match("^([^-]+)")
    local listenerName = listenerID:match("^([^-]+)")

    -- 判断是否是自己发送的消息
    local isFromSelf = (senderID == listenerID)

    -- 获取 TRP3 显示名称
    local senderTRP3Name = GetTRP3DisplayName(senderID, isFromSelf)
    local listenerTRP3Name = GetTRP3DisplayName(listenerID, true)

    -- 如果是自己发送的消息，替换开头的"你"
    if isFromSelf and senderTRP3Name then
        msg = msg:gsub("^你", senderTRP3Name)
    end

    -- 替换发送者名字（通常在消息开头）
    if senderTRP3Name and senderName and not isFromSelf then
        local escapedSenderName = EscapePattern(senderName)
        msg = msg:gsub("^" .. escapedSenderName, senderTRP3Name)
    end

    -- 替换"对你"、"向你"等包含"你"的常见表情短语（仅当不是自己发送时）
    if listenerTRP3Name and not isFromSelf then
        msg = msg:gsub("对你", "对" .. listenerTRP3Name)
        msg = msg:gsub("向你", "向" .. listenerTRP3Name)
        msg = msg:gsub("给你", "给" .. listenerTRP3Name)
        msg = msg:gsub("朝你", "朝" .. listenerTRP3Name)
        msg = msg:gsub("跟你", "跟" .. listenerTRP3Name)
        msg = msg:gsub("和你", "和" .. listenerTRP3Name)
    end

    return msg
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
    if not unitID or unitID == "" then return false end

    -- 检查总开关
    if RPBox_Config.enabled == false then return false end

    -- 先检查频道是否启用
    if not IsChannelEnabled(channelShort) then return false end

    -- 检查是否屏蔽自己
    if isFromSelf then
        if RPBox_Config.ignoreSelf then return false end
        return true
    end

    -- 检查是否只接受公会成员
    if RPBox_Config.guildOnly then
        if not IsInGuild() then return false end
        local isGuildMember = false
        for i = 1, GetNumGuildMembers() do
            local name = GetGuildRosterInfo(i)
            if name then
                local shortName = name:match("^([^-]+)")
                if shortName == unitID or name == unitID then
                    isGuildMember = true
                    break
                end
            end
        end
        if not isGuildMember then return false end
    end

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
    if not record.t and not record.timestamp then
        record.t = time()
    end
    if ns.ApplyRecordSchema then
        ns.ApplyRecordSchema(record)
    end

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

ns.SaveRecord = SaveChatLog

local function CopyEventEndpoint(endpoint)
    endpoint = endpoint or {}
    return {
        ref = endpoint.ref,
        ps = endpoint.ps,
        n = endpoint.n or "",
        pn = endpoint.pn,
    }
end

local function BuildCurrentListener()
    local context = ns.GetSelfProfileContext and ns.GetSelfProfileContext() or nil
    if not context then return nil end

    local snapshotKey = nil
    if ns.CaptureProfileSnapshot then
        snapshotKey = ns.CaptureProfileSnapshot(
            context.profileID,
            context.root or context.profile,
            context.gameID,
            context.profileName
        )
    end
    return {
        gameID = context.gameID,
        profileID = context.profileID,
        ref = context.profileID,
        ps = snapshotKey,
    }
end

-- AppendProfileTimelineEvent writes profile transitions into the same chronological ledger.
function ns.AppendProfileTimelineEvent(kind, certainty, fromEndpoint, toEndpoint, actorGameID)
    if kind ~= "profile_switch" and kind ~= "profile_update" then return end

    local fromCopy = CopyEventEndpoint(fromEndpoint)
    local toCopy = CopyEventEndpoint(toEndpoint)
    local verb = kind == "profile_switch" and "人物卡切换" or "人物卡更新"
    local fromLabel = fromCopy.n ~= "" and fromCopy.n or tostring(fromCopy.ref or "未知")
    local toLabel = toCopy.n ~= "" and toCopy.n or tostring(toCopy.ref or "未知")
    local record = {
        t = time(),
        c = "SYSTEM",
        m = verb .. "：" .. fromLabel .. " -> " .. toLabel,
        mk = "S",
        s = actorGameID or ns.GetPlayerID(),
        ref = toCopy.ref or fromCopy.ref,
        ps = toCopy.ps or fromCopy.ps,
        ev = {
            kind = kind,
            certainty = certainty == "exact" and "exact" or "observed",
            from = fromCopy,
            to = toCopy,
        },
    }

    local listener = BuildCurrentListener()
    if listener then
        record.listeners = { listener }
    end
    SaveChatLog(record)
end

-- 解析 NPC/旁白消息
-- 返回: mk, npcName, message, npcType
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

local function ParseNPCMessage(content)
    if not content:match("^|") then return nil end
    -- 跳过 WoW 颜色代码 |cFFxxxxxx 开头的情况
    if content:match("^|c") then return nil end

    local text = content:gsub("^|+", ""):match("^%s*(.+)")
    if not text then return nil end

    -- 清理末尾的颜色代码 |r
    text = text:gsub("|r%s*$", "")

    -- 悄悄说
    local npcName, message = text:match("^(.-)%s*悄悄说%s*[：:]%s*(.*)$")
    if npcName and message then
        message = message:gsub("|r%s*$", "")
        message = StripInvalidLeadingBytes(message)
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
    local senderID = sender or ""

    if senderID ~= "" and not senderID:find("-") then
        senderID = senderID .. "-" .. GetRealmName()
    end

    local isFromSelf = (senderID == playerID)
    local channelShort = CHANNEL_SHORT[event] or event
    local lineID = select(9, ...)

    if not ShouldRecord(senderID, isFromSelf, channelShort) then
        return false
    end

    -- 用 lineID 优先去重；没有 lineID 时退回到 sender + channel + 文本指纹
    if IsDuplicateMessage(senderID, channelShort, msg, lineID) then
        return false
    end

    -- 获取发送者GUID和职业
    local senderGUID = select(10, ...)
    local senderClass = nil
    if senderGUID then
        local _, classFilename = GetPlayerInfoByGUID(senderGUID)
        senderClass = classFilename
    end

    -- 获取 profileID
    local profileID
    local senderProfileSnapshot = nil
    if isFromSelf then
        profileID, _, senderProfileSnapshot = GetSelfTRP3InfoAndCache()
        if not senderClass then
            local _, classFilename = UnitClass("player")
            senderClass = classFilename
        end
    else
        profileID, _, senderProfileSnapshot = GetTRP3InfoAndCache(senderID)
    end

    -- 解析消息类型
    local mk, npcName, parsedMsg, npcType = ParseNPCMessage(msg)
    if not mk then
        mk = "P"  -- 普通玩家消息
    end

    -- 如果是表情消息，替换角色名为 TRP3 人物卡名称
    if event == "CHAT_MSG_EMOTE" or event == "CHAT_MSG_TEXT_EMOTE" then
        local originalMsg = parsedMsg or msg
        local replacedMsg = ReplaceEmoteNames(originalMsg, senderID, playerID)
        if replacedMsg then
            parsedMsg = replacedMsg
        end
    end

    local recordMessage = parsedMsg or msg

    -- 构建记录
    local record = {
        t = time(),
        c = CHANNEL_SHORT[event] or event,
        m = recordMessage,
        mk = mk,
        s = senderID,
        ref = profileID,
        ps = senderProfileSnapshot,
    }

    if ns.ExtractChatLinks then
        local links = ns.ExtractChatLinks(recordMessage)
        if links and #links > 0 then
            record.lk = links
        end
    end

    -- 添加收听者信息（当前登录的角色）
    local listenerContext = ns.GetSelfProfileContext and ns.GetSelfProfileContext() or nil
    if listenerContext and listenerContext.gameID then
        local listenerSnapshot = nil
        if isFromSelf and listenerContext.profileID == profileID then
            listenerSnapshot = senderProfileSnapshot
        elseif ns.CaptureProfileSnapshot then
            listenerSnapshot = ns.CaptureProfileSnapshot(
                listenerContext.profileID,
                listenerContext.root or listenerContext.profile,
                listenerContext.gameID,
                listenerContext.profileName
            )
        end
        record.listeners = {
            {
                gameID = listenerContext.gameID,
                profileID = listenerContext.profileID,
                ref = listenerContext.profileID,
                ps = listenerSnapshot,
            }
        }
    end

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
-- 使用独立事件帧，避免多聊天窗口同时挂过滤器时重复记录同一条消息
local chatEventFrame = CreateFrame("Frame")
for _, event in ipairs(CHAT_EVENTS) do
    chatEventFrame:RegisterEvent(event)
end

chatEventFrame:SetScript("OnEvent", function(self, event, ...)
    OnChatMessage(self, event, ...)
end)
