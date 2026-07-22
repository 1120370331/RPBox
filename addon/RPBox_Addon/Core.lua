-- RPBox Addon Core
-- 插件核心框架

local ADDON_NAME, ns = ...

-- 版本信息
ns.VERSION = "1.0.13"

-- 公开 API
RPBox_API = ns

-- 本地化
local L = RPBox_L or {}
ns.L = L

-- 默认配置
local DEFAULT_CONFIG = {
    -- 功能开关
    enabled = true,  -- 是否开启记录功能
    -- 名单
    whitelist = {},
    blacklist = {},
    warnedThisSession = false,
    warnThreshold = 9000,
    logViewWindowSize = 80,
    -- 监听的频道配置
    channels = {
        SAY = true,
        YELL = true,
        EMOTE = true,
        PARTY = false,
        RAID = false,
        WHISPER_IN = false,
        WHISPER_OUT = false,
        GUILD = false,
    },
    -- 显示设置
    showIcon = true,  -- 是否显示头像图标
    ignoreSelf = false,  -- 是否屏蔽自己的消息
    guildOnly = false,  -- 是否只接受公会成员的消息
}

local RECORD_SCHEMA_VERSION = 2

local function NewInstallID()
    local playerID = ns.GetPlayerID and ns.GetPlayerID() or nil
    local safePlayerID = tostring(playerID or "account"):gsub("[^%w]", "")
    return table.concat({ tostring(time()), safePlayerID, tostring(math.random(100000, 999999)) }, "-")
end

local function InitRecordState()
    RPBox_RecordState = RPBox_RecordState or {}
    if not RPBox_RecordState.installID or RPBox_RecordState.installID == "" then
        RPBox_RecordState.installID = NewInstallID()
    end

    RPBox_RecordState.sessionCounter = (tonumber(RPBox_RecordState.sessionCounter) or 0) + 1
    RPBox_RecordState.sessionID = RPBox_RecordState.installID .. "." .. tostring(RPBox_RecordState.sessionCounter)
    RPBox_RecordState.recordSequence = 0
    RPBox_RecordState.snapshotSequence = tonumber(RPBox_RecordState.snapshotSequence) or 0
    RPBox_RecordState.snapshotIndex = RPBox_RecordState.snapshotIndex or {}
    RPBox_RecordState.observedProfiles = RPBox_RecordState.observedProfiles or {}
end

-- 初始化 SavedVariables
local function InitSavedVariables()
    RPBox_Config = RPBox_Config or {}
    for k, v in pairs(DEFAULT_CONFIG) do
        if RPBox_Config[k] == nil then
            RPBox_Config[k] = v
        end
    end

    RPBox_ChatLog = RPBox_ChatLog or {}
    RPBox_ProfileCache = RPBox_ProfileCache or {}
    RPBox_ProfileSnapshots = RPBox_ProfileSnapshots or {}
    RPBox_ProfileExport = RPBox_ProfileExport or {}
    RPBox_Sync = RPBox_Sync or { addon = {}, client = {} }
    InitRecordState()
end

-- 获取单位ID (玩家名-服务器)
function ns.GetUnitID(unit)
    local name, realm = UnitName(unit)
    if not name then return nil end
    realm = realm or GetRealmName()
    return name .. "-" .. realm
end

-- 获取玩家自己的单位ID
function ns.GetPlayerID()
    return ns.GetUnitID("player")
end

local function NormalizeIdentityValue(value)
    if value == nil then return "" end
    return tostring(value)
end

local function GetProfilePlayerData(profileData)
    if type(profileData) ~= "table" then return nil end
    if type(profileData.player) == "table" then
        return profileData.player
    end
    return profileData
end

local function GetIdentityName(characteristics)
    if type(characteristics) ~= "table" then return "" end
    local firstName = NormalizeIdentityValue(characteristics.FN)
    local lastName = NormalizeIdentityValue(characteristics.LN)
    if firstName ~= "" and lastName ~= "" then
        return firstName .. " " .. lastName
    end
    return firstName ~= "" and firstName or lastName
end

local function BuildIdentitySignature(profileID, gameID, characteristics, profileName)
    characteristics = characteristics or {}
    return table.concat({
        NormalizeIdentityValue(profileID),
        NormalizeIdentityValue(gameID),
        NormalizeIdentityValue(profileName),
        NormalizeIdentityValue(characteristics.FN),
        NormalizeIdentityValue(characteristics.LN),
        NormalizeIdentityValue(characteristics.TI),
        NormalizeIdentityValue(characteristics.IC),
        NormalizeIdentityValue(characteristics.CH),
    }, "\31")
end

local function GetSnapshotSubjectKey(profileID, gameID)
    if profileID and profileID ~= "" then
        return "profile:" .. tostring(profileID) .. "\31game:" .. tostring(gameID or "unknown")
    end
    return "game:" .. tostring(gameID or "unknown")
end

local function BuildProfileIdentitySnapshot(profileID, profileData, gameID, profileName)
    local playerData = GetProfilePlayerData(profileData) or {}
    local characteristics = playerData.characteristics or {}
    if not profileName and type(profileData) == "table" then
        profileName = profileData.profileName
    end

    return {
        ref = profileID,
        gameID = gameID,
        n = GetIdentityName(characteristics),
        pn = profileName,
        FN = characteristics.FN,
        LN = characteristics.LN,
        TI = characteristics.TI,
        IC = characteristics.IC,
        CH = characteristics.CH,
        at = time(),
    }, BuildIdentitySignature(profileID, gameID, characteristics, profileName)
end

-- GetProfileSnapshot returns one immutable identity observation captured with a record.
function ns.GetProfileSnapshot(snapshotKey)
    if not snapshotKey or not RPBox_ProfileSnapshots then return nil end
    return RPBox_ProfileSnapshots[snapshotKey]
end

-- GetSnapshotSignature returns the compact identity fields used to detect a real change.
function ns.GetSnapshotSignature(snapshot)
    if type(snapshot) ~= "table" then return nil end
    return BuildIdentitySignature(snapshot.ref, snapshot.gameID, snapshot, snapshot.pn)
end

-- CaptureProfileSnapshot stores a new immutable identity only when the observed identity changes.
function ns.CaptureProfileSnapshot(profileID, profileData, gameID, profileName)
    if not RPBox_RecordState or not RPBox_ProfileSnapshots then return nil, nil end

    local subjectKey = GetSnapshotSubjectKey(profileID, gameID)
    local snapshot, signature = BuildProfileIdentitySnapshot(profileID, profileData, gameID, profileName)
    local index = RPBox_RecordState.snapshotIndex[subjectKey]
    if index and index.signature == signature and index.key then
        local existing = RPBox_ProfileSnapshots[index.key]
        if existing then
            return index.key, existing
        end
    end

    local previousRevision = index and tonumber(index.rev) or 0
    RPBox_RecordState.snapshotSequence = (tonumber(RPBox_RecordState.snapshotSequence) or 0) + 1
    local snapshotKey = RPBox_RecordState.sessionID .. ".p" .. tostring(RPBox_RecordState.snapshotSequence)
    snapshot.rev = previousRevision + 1

    -- Never mutate an existing key: records already referring to it are history.
    RPBox_ProfileSnapshots[snapshotKey] = snapshot
    RPBox_RecordState.snapshotIndex[subjectKey] = {
        key = snapshotKey,
        signature = signature,
        rev = snapshot.rev,
    }
    return snapshotKey, snapshot
end

-- MakeSnapshotEndpoint copies literal event endpoint data so later cache updates cannot rewrite it.
function ns.MakeSnapshotEndpoint(profileID, snapshotKey, snapshot)
    snapshot = snapshot or ns.GetProfileSnapshot(snapshotKey) or {}
    return {
        ref = profileID or snapshot.ref,
        ps = snapshotKey,
        n = snapshot.n or "",
        pn = snapshot.pn,
    }
end

-- ApplyRecordSchema assigns a collision-resistant session/sequence identity to a new record.
function ns.ApplyRecordSchema(record)
    if type(record) ~= "table" then return record end
    if not RPBox_RecordState then return record end
    if record.sv == RECORD_SCHEMA_VERSION and record.id and record.sid and record.seq then
        return record
    end

    RPBox_RecordState.recordSequence = (tonumber(RPBox_RecordState.recordSequence) or 0) + 1
    local sequence = RPBox_RecordState.recordSequence
    record.sv = RECORD_SCHEMA_VERSION
    record.sid = RPBox_RecordState.sessionID
    record.seq = sequence
    record.id = record.sid .. ".r" .. tostring(sequence)
    return record
end

-- GetSelfProfileContext returns the active local profile and its management label.
function ns.GetSelfProfileContext()
    if not TRP3_API or not TRP3_API.profile then return nil end
    if not TRP3_API.profile.getPlayerCurrentProfileID then return nil end

    local profileID = TRP3_API.profile.getPlayerCurrentProfileID()
    if not profileID then return nil end

    local rootProfile = nil
    if TRP3_API.profile.getPlayerCurrentProfile then
        rootProfile = TRP3_API.profile.getPlayerCurrentProfile()
    end
    local playerData = rootProfile and rootProfile.player or nil
    if not playerData and TRP3_API.profile.getData then
        playerData = TRP3_API.profile.getData("player")
    end

    return {
        profileID = profileID,
        profile = playerData or {},
        root = rootProfile,
        profileName = rootProfile and rootProfile.profileName or nil,
        gameID = ns.GetPlayerID(),
    }
end

-- GetRemoteProfileContext safely reads the currently observed card for a unit.
function ns.GetRemoteProfileContext(unitID)
    if not unitID or not TRP3_API or not TRP3_API.register then return nil end
    if not TRP3_API.register.isUnitIDKnown or not TRP3_API.register.isUnitIDKnown(unitID) then return nil end

    local okCharacter, character = pcall(TRP3_API.register.getUnitIDCharacter, unitID)
    if not okCharacter or not character or not character.profileID then return nil end

    local profileID = character.profileID
    local profile = nil
    if TRP3_API.register.getProfileOrNil then
        profile = TRP3_API.register.getProfileOrNil(profileID)
    elseif TRP3_API.register.getProfile then
        local okProfile, result = pcall(TRP3_API.register.getProfile, profileID)
        if okProfile then profile = result end
    end
    if not profile then return nil end

    return {
        profileID = profileID,
        profile = profile,
        gameID = unitID,
    }
end

-- 检查是否在黑名单
function ns.IsBlacklisted(unitID)
    if not unitID then return false end

    -- RPBox 黑名单
    if RPBox_Config.blacklist[unitID] then return true end

    -- WoW 原生拉黑
    local name = strsplit("-", unitID)
    if C_FriendList and C_FriendList.IsIgnored(name) then return true end

    -- TRP3 拉黑检查
    if TRP3_API and TRP3_API.register and TRP3_API.register.relation then
        local relation = TRP3_API.register.relation.getRelation(unitID)
        if relation == TRP3_API.register.relation.NONE then return true end
    end

    return false
end

-- 检查是否在白名单
function ns.IsWhitelisted(unitID)
    return unitID and RPBox_Config.whitelist[unitID] == true
end

-- 添加到白名单
function ns.AddToWhitelist(unitID)
    if not unitID then return end
    -- 排斥自己
    local playerName = UnitName("player")
    if unitID == playerName then
        print("[RPBox] 不能将自己添加到白名单")
        return
    end
    RPBox_Config.whitelist[unitID] = true
    RPBox_Config.blacklist[unitID] = nil
    print(format(L["WHITELIST_ADDED"] or "[RPBox] %s 已加入白名单", unitID))
    ns.TriggerOnListChange()
end

-- 添加到黑名单
function ns.AddToBlacklist(unitID)
    if not unitID then return end
    RPBox_Config.blacklist[unitID] = true
    RPBox_Config.whitelist[unitID] = nil
    print(format(L["BLACKLIST_ADDED"] or "[RPBox] %s 已加入黑名单", unitID))
    ns.TriggerOnListChange()
end

-- 从白名单移除
function ns.RemoveFromWhitelist(unitID)
    if unitID then
        RPBox_Config.whitelist[unitID] = nil
        ns.TriggerOnListChange()
    end
end

-- 从黑名单移除
function ns.RemoveFromBlacklist(unitID)
    if unitID then
        RPBox_Config.blacklist[unitID] = nil
        ns.TriggerOnListChange()
    end
end

-- 缓存角色卡数据
function ns.CacheProfile(profileID, playerData)
    if not profileID or not playerData then return end

    playerData = GetProfilePlayerData(playerData)
    if not playerData then return end

    local cache = {
        -- characteristics
        v = playerData.characteristics and playerData.characteristics.v,
        FN = playerData.characteristics and playerData.characteristics.FN,
        LN = playerData.characteristics and playerData.characteristics.LN,
        TI = playerData.characteristics and playerData.characteristics.TI,
        FT = playerData.characteristics and playerData.characteristics.FT,
        RA = playerData.characteristics and playerData.characteristics.RA,
        CL = playerData.characteristics and playerData.characteristics.CL,
        AG = playerData.characteristics and playerData.characteristics.AG,
        EC = playerData.characteristics and playerData.characteristics.EC,
        HE = playerData.characteristics and playerData.characteristics.HE,
        WE = playerData.characteristics and playerData.characteristics.WE,
        BP = playerData.characteristics and playerData.characteristics.BP,
        RE = playerData.characteristics and playerData.characteristics.RE,
        RS = playerData.characteristics and playerData.characteristics.RS,
        IC = playerData.characteristics and playerData.characteristics.IC,
        CH = playerData.characteristics and playerData.characteristics.CH,
        MI = playerData.characteristics and playerData.characteristics.MI,
        PS = playerData.characteristics and playerData.characteristics.PS,
        -- misc
        misc = playerData.misc,
        -- about
        about = playerData.about,
    }

    RPBox_ProfileCache[profileID] = cache
end

-- 获取缓存的角色卡
function ns.GetCachedProfile(profileID)
    return profileID and RPBox_ProfileCache[profileID]
end

-- 更新指定玩家的角色卡缓存（响应TRP3事件）
function ns.UpdateProfileCache(unitID)
    -- print("|cFF00FF00[RPBox]|r UpdateProfileCache 开始: unitID=" .. tostring(unitID))

    if not unitID then
        -- print("|cFFFF0000[RPBox]|r UpdateProfileCache 失败: unitID为空")
        return
    end

    local context = ns.GetRemoteProfileContext(unitID)
    if not context then
        -- print("|cFFFF0000[RPBox]|r UpdateProfileCache 失败: 无法获取profile数据")
        return
    end

    local profileID = context.profileID
    local profile = context.profile

    -- 更新缓存
    ns.CacheProfile(profileID, profile)

    -- 提取角色名用于显示
    local charName = "未知"
    if profile.characteristics then
        local fn = profile.characteristics.FN or ""
        local ln = profile.characteristics.LN or ""
        if fn ~= "" then
            charName = ln ~= "" and (fn .. " " .. ln) or fn
        end
    end

    -- print("|cFF00FF00[RPBox]|r ✓ 成功缓存人物卡: " .. charName .. " (profileID: " .. profileID .. ")")
end

-- 批量导入 TRP3 所有人物卡
function ns.ImportAllTRP3Profiles()
    print("|cFF00FF00[RPBox]|r 开始导入 TRP3 人物卡数据...")

    if not TRP3_API or not TRP3_API.register or not TRP3_API.register.getProfileList then
        print("|cFFFF0000[RPBox]|r 错误: TRP3 API 不可用")
        return
    end

    local profiles = TRP3_API.register.getProfileList()
    if not profiles then
        print("|cFFFF0000[RPBox]|r 错误: 无法获取 TRP3 人物卡列表")
        return
    end

    local count = 0
    local skipped = 0

    for profileID, profileData in pairs(profiles) do
        -- TRP3_Register.profiles 中的 profile 结构是直接包含 characteristics, about, misc 等字段
        -- 而不是 profile.player.characteristics
        if profileData and (profileData.characteristics or profileData.about or profileData.misc) then
            ns.CacheProfile(profileID, profileData)
            count = count + 1
        else
            skipped = skipped + 1
        end
    end

    print("|cFF00FF00[RPBox]|r ========== 导入完成 ==========")
    print("|cFF00FF00[RPBox]|r 成功导入: " .. count .. " 个人物卡")
    if skipped > 0 then
        print("|cFFFFFF00[RPBox]|r 跳过: " .. skipped .. " 个无效数据")
    end
end

local localProfileBaseline = nil

local function PersistProfileObservation(observation)
    if not observation then return nil end
    if observation.ps and ns.GetProfileSnapshot(observation.ps) then return observation end

    local identity = observation.snapshot or {}
    local profileData = {
        characteristics = {
            FN = identity.FN,
            LN = identity.LN,
            TI = identity.TI,
            IC = identity.IC,
            CH = identity.CH,
        },
    }
    local snapshotKey, snapshot = ns.CaptureProfileSnapshot(
        observation.ref,
        profileData,
        identity.gameID,
        identity.pn
    )
    if not snapshotKey or not snapshot then return observation end

    observation.ps = snapshotKey
    observation.snapshot = snapshot
    observation.signature = ns.GetSnapshotSignature(snapshot)
    observation.endpoint = ns.MakeSnapshotEndpoint(observation.ref, snapshotKey, snapshot)
    return observation
end

local function CaptureSelfObservation(persistSnapshot)
    local context = ns.GetSelfProfileContext()
    if not context then return nil end

    ns.CacheProfile(context.profileID, context.profile)
    local snapshot, signature = BuildProfileIdentitySnapshot(
        context.profileID,
        context.root or context.profile,
        context.gameID,
        context.profileName
    )
    local observation = {
        ref = context.profileID,
        snapshot = snapshot,
        signature = signature,
        endpoint = ns.MakeSnapshotEndpoint(context.profileID, nil, snapshot),
    }
    if persistSnapshot ~= false then
        PersistProfileObservation(observation)
    end
    return observation
end

local function EmitProfileTimelineEvent(kind, certainty, fromEndpoint, toEndpoint, actorGameID)
    if ns.AppendProfileTimelineEvent then
        ns.AppendProfileTimelineEvent(kind, certainty, fromEndpoint, toEndpoint, actorGameID)
    end
end

-- ObserveRemoteProfileIdentity captures remote changes as observations, without claiming when they happened.
function ns.ObserveRemoteProfileIdentity(unitID, profileID, profileData, emitChange)
    if not unitID or not profileID or not profileData then return nil, nil end
    local playerID = ns.GetPlayerID()
    if playerID and unitID == playerID then
        return ns.CaptureProfileSnapshot(profileID, profileData, unitID)
    end

    local snapshotKey, snapshot = ns.CaptureProfileSnapshot(profileID, profileData, unitID)
    if not snapshotKey or not snapshot or not RPBox_RecordState then return snapshotKey, snapshot end

    local previousKey = RPBox_RecordState.observedProfiles[unitID]
    RPBox_RecordState.observedProfiles[unitID] = snapshotKey
    if previousKey and previousKey ~= snapshotKey and emitChange ~= false then
        local previousSnapshot = ns.GetProfileSnapshot(previousKey)
        if previousSnapshot then
            local kind = previousSnapshot.ref ~= snapshot.ref and "profile_switch" or "profile_update"
            EmitProfileTimelineEvent(
                kind,
                "observed",
                ns.MakeSnapshotEndpoint(previousSnapshot.ref, previousKey, previousSnapshot),
                ns.MakeSnapshotEndpoint(snapshot.ref, snapshotKey, snapshot),
                unitID
            )
        end
    end

    return snapshotKey, snapshot
end

local function SeedLocalProfileBaseline()
    if localProfileBaseline then return end
    localProfileBaseline = CaptureSelfObservation()
end

local function OnLocalProfilesLoaded()
    local current = CaptureSelfObservation(false)
    if not current then return end
    if not localProfileBaseline then
        localProfileBaseline = PersistProfileObservation(current)
        return
    end

    local previous = localProfileBaseline
    if previous.ref ~= current.ref then
        -- REGISTER_PROFILES_LOADED is the authoritative local switch signal.
        PersistProfileObservation(previous)
        PersistProfileObservation(current)
        EmitProfileTimelineEvent("profile_switch", "exact", previous.endpoint, current.endpoint, ns.GetPlayerID())
    elseif previous.signature ~= current.signature then
        -- Same-ID reloads only establish that a changed identity is now observed.
        PersistProfileObservation(previous)
        PersistProfileObservation(current)
        EmitProfileTimelineEvent("profile_update", "observed", previous.endpoint, current.endpoint, ns.GetPlayerID())
    else
        current = previous
    end
    localProfileBaseline = current
end

local function OnLocalProfileDataUpdated()
    local current = CaptureSelfObservation(false)
    if not current then return end
    if not localProfileBaseline then
        localProfileBaseline = PersistProfileObservation(current)
        return
    end

    local previous = localProfileBaseline
    if previous.ref == current.ref and previous.signature ~= current.signature then
        PersistProfileObservation(previous)
        PersistProfileObservation(current)
        EmitProfileTimelineEvent("profile_update", "exact", previous.endpoint, current.endpoint, ns.GetPlayerID())
    elseif previous.ref == current.ref then
        current = previous
    end
    -- A differing ID is handled by REGISTER_PROFILES_LOADED; seeding here prevents its duplicate update event.
    localProfileBaseline = current
end

local function MarkSnapshotReference(reachable, snapshotKey)
    if snapshotKey and RPBox_ProfileSnapshots and RPBox_ProfileSnapshots[snapshotKey] then
        reachable[snapshotKey] = true
    end
end

local function MarkRecordSnapshotReferences(reachable, record)
    if type(record) ~= "table" then return end
    MarkSnapshotReference(reachable, record.ps)
    if type(record.sender) == "table" then
        MarkSnapshotReference(reachable, record.sender.ps)
    end
    for _, listener in ipairs(record.listeners or {}) do
        if type(listener) == "table" then
            MarkSnapshotReference(reachable, listener.ps)
        end
    end
    if type(record.ev) == "table" then
        if type(record.ev.from) == "table" then
            MarkSnapshotReference(reachable, record.ev.from.ps)
        end
        if type(record.ev.to) == "table" then
            MarkSnapshotReference(reachable, record.ev.to.ps)
        end
    end
end

local function IsNewerSnapshot(candidateKey, candidate, indexed)
    if not indexed then return true end
    local candidateRevision = tonumber(candidate.rev) or 0
    local indexedRevision = tonumber(indexed.rev) or 0
    if candidateRevision ~= indexedRevision then return candidateRevision > indexedRevision end

    local candidateTime = tonumber(candidate.at) or 0
    local indexedSnapshot = RPBox_ProfileSnapshots[indexed.key]
    local indexedTime = indexedSnapshot and tonumber(indexedSnapshot.at) or 0
    if candidateTime ~= indexedTime then return candidateTime > indexedTime end
    return tostring(candidateKey) > tostring(indexed.key or "")
end

local function RebuildSnapshotIndex()
    if not RPBox_RecordState then return end
    local rebuilt = {}
    for snapshotKey, snapshot in pairs(RPBox_ProfileSnapshots or {}) do
        if type(snapshot) == "table" then
            local subjectKey = GetSnapshotSubjectKey(snapshot.ref, snapshot.gameID)
            if IsNewerSnapshot(snapshotKey, snapshot, rebuilt[subjectKey]) then
                rebuilt[subjectKey] = {
                    key = snapshotKey,
                    signature = ns.GetSnapshotSignature(snapshot),
                    rev = tonumber(snapshot.rev) or 0,
                }
            end
        end
    end
    RPBox_RecordState.snapshotIndex = rebuilt
end

-- PruneProfileSnapshots keeps record/event references plus live observation baselines.
-- The latter prevent a partial retention cleanup from fabricating a new remote/local change.
function ns.PruneProfileSnapshots(keepObservationBaselines)
    if not RPBox_ProfileSnapshots or not RPBox_RecordState then return 0 end
    local reachable = {}

    for _, hours in pairs(RPBox_ChatLog or {}) do
        for _, records in pairs(hours) do
            for _, record in ipairs(records) do
                MarkRecordSnapshotReferences(reachable, record)
            end
        end
    end

    if keepObservationBaselines ~= false then
        if localProfileBaseline and localProfileBaseline.ps then
            MarkSnapshotReference(reachable, localProfileBaseline.ps)
        end
        for _, snapshotKey in pairs(RPBox_RecordState.observedProfiles or {}) do
            MarkSnapshotReference(reachable, snapshotKey)
        end
    end

    local retained = {}
    local removed = 0
    for snapshotKey, snapshot in pairs(RPBox_ProfileSnapshots) do
        if reachable[snapshotKey] then
            retained[snapshotKey] = snapshot
        else
            removed = removed + 1
        end
    end
    RPBox_ProfileSnapshots = retained

    -- Drop any pre-existing dangling observer entry; valid live observers were marked above.
    local staleObservers = {}
    for unitID, snapshotKey in pairs(RPBox_RecordState.observedProfiles or {}) do
        if not RPBox_ProfileSnapshots[snapshotKey] then
            staleObservers[#staleObservers + 1] = unitID
        end
    end
    for _, unitID in ipairs(staleObservers) do
        RPBox_RecordState.observedProfiles[unitID] = nil
    end
    if localProfileBaseline and localProfileBaseline.ps
        and not RPBox_ProfileSnapshots[localProfileBaseline.ps] then
        localProfileBaseline.ps = nil
        localProfileBaseline.endpoint = ns.MakeSnapshotEndpoint(
            localProfileBaseline.ref,
            nil,
            localProfileBaseline.snapshot
        )
    end

    RebuildSnapshotIndex()
    return removed
end

local function ResetProfileHistoryState()
    RPBox_ProfileSnapshots = {}
    if RPBox_RecordState then
        -- Keep install/session/sequence fields so record and snapshot IDs are never reused.
        RPBox_RecordState.snapshotIndex = {}
        RPBox_RecordState.observedProfiles = {}
    end

    -- This fresh in-memory baseline is not history and owns no snapshot key. It lets the
    -- next local switch materialize a valid before-snapshot without crossing the clear boundary.
    localProfileBaseline = CaptureSelfObservation(false)
end

-- 显示缓存统计信息
function ns.ShowCacheStats()
    local count = 0
    for _ in pairs(RPBox_ProfileCache) do
        count = count + 1
    end

    print("|cFF00FF00[RPBox]|r ========== 人物卡缓存统计 ==========")
    print("|cFF00FF00[RPBox]|r 已缓存人物卡数量: " .. count)
    print("|cFF00FF00[RPBox]|r 使用 /rpbox cache list 查看详细列表")
end

-- 列出所有已缓存的人物卡
function ns.ListCachedProfiles()
    local profiles = {}

    -- 收集所有人物卡信息
    for profileID, data in pairs(RPBox_ProfileCache) do
        local charName = "未知"
        local fn = data.FN or ""
        local ln = data.LN or ""

        if fn ~= "" then
            charName = ln ~= "" and (fn .. " " .. ln) or fn
        end

        table.insert(profiles, {
            id = profileID,
            name = charName,
            title = data.TI or "",
            race = data.RA or "",
            class = data.CL or "",
        })
    end

    -- 按名字排序
    table.sort(profiles, function(a, b) return a.name < b.name end)

    print("|cFF00FF00[RPBox]|r ========== 已缓存的人物卡 (" .. #profiles .. ") ==========")

    if #profiles == 0 then
        print("|cFFFFFF00[RPBox]|r 暂无缓存的人物卡")
        print("|cFFFFFF00[RPBox]|r 提示: 当您与其他RP玩家互动时，系统会自动缓存他们的人物卡")
        return
    end

    for i, profile in ipairs(profiles) do
        local info = profile.name
        if profile.title ~= "" then
            info = info .. " <" .. profile.title .. ">"
        end
        if profile.race ~= "" or profile.class ~= "" then
            local extra = {}
            if profile.race ~= "" then table.insert(extra, profile.race) end
            if profile.class ~= "" then table.insert(extra, profile.class) end
            info = info .. " (" .. table.concat(extra, ", ") .. ")"
        end

        print(string.format("|cFF00FF00[RPBox]|r %d. %s", i, info))
        print(string.format("    ProfileID: |cFFAAAAAA%s|r", profile.id))
    end

    print("|cFF00FF00[RPBox]|r =====================================")
end

-- 显示帮助信息
local function ShowHelp()
    print(L["HELP_TITLE"] or "|cFF00FF00[RPBox]|r 命令帮助:")
    print("  /rpbox whitelist add/remove 玩家名-服务器")
    print("  /rpbox blacklist add/remove 玩家名-服务器")
    print("  /rpbox export [target] - 导出人物卡")
    print("  /rpbox clear [all] - 清理记录")
    print("  /rpbox log [today] - 打开回放窗口")
    print("  /rpbox item mark/list - 道具标记")
    print("  /rpbox cache list - 列出已缓存的人物卡")
    print("  /rpbox cache stats - 显示缓存统计")
    print("  /rpbox cache import - 导入TRP3所有人物卡")
end

-- 斜杠命令处理
local function HandleSlashCommand(msg)
    local args = {}
    for word in msg:gmatch("%S+") do
        table.insert(args, word)
    end

    local cmd = args[1] and args[1]:lower() or ""
    local subcmd = args[2] and args[2]:lower() or ""
    local param = args[3] or ""

    -- 无参数：打开主界面
    if cmd == "" then
        ns.OpenMainFrame()
        return
    end

    -- help：显示帮助
    if cmd == "help" then
        ShowHelp()
        return
    end

    if cmd == "whitelist" then
        local targetName = param
        if targetName == "" and UnitExists("target") and UnitIsPlayer("target") then
            targetName = UnitName("target")
        end
        if subcmd == "add" and targetName ~= "" then
            ns.AddToWhitelist(targetName)
        elseif subcmd == "remove" and targetName ~= "" then
            ns.RemoveFromWhitelist(targetName)
            print("|cFF00FF00[RPBox]|r " .. targetName .. " 已从白名单移除")
        elseif targetName == "" then
            print("[RPBox] 请指定玩家名或选中一个玩家目标")
        end
    elseif cmd == "blacklist" then
        local targetName = param
        if targetName == "" and UnitExists("target") and UnitIsPlayer("target") then
            targetName = UnitName("target")
        end
        if subcmd == "add" and targetName ~= "" then
            ns.AddToBlacklist(targetName)
        elseif subcmd == "remove" and targetName ~= "" then
            ns.RemoveFromBlacklist(targetName)
            print("|cFF00FF00[RPBox]|r " .. targetName .. " 已从黑名单移除")
        elseif targetName == "" then
            print("[RPBox] 请指定玩家名或选中一个玩家目标")
        end
    elseif cmd == "export" then
        ns.ExportProfile(subcmd == "target" and "target" or "player")
    elseif cmd == "clear" then
        ns.ClearRecords(subcmd == "all", args[3] == "confirm")
    elseif cmd == "log" then
        ns.OpenLogWindow(subcmd == "today")
    elseif cmd == "item" then
        if subcmd == "mark" then
            ns.MarkItem(param)
        elseif subcmd == "list" then
            ns.ListMarkedItems()
        else
            print("|cFF00FF00[RPBox]|r 用法: /rpbox item mark/list")
        end
    elseif cmd == "cache" or cmd == "profiles" then
        if subcmd == "list" or subcmd == "" then
            ns.ListCachedProfiles()
        elseif subcmd == "stats" then
            ns.ShowCacheStats()
        elseif subcmd == "import" then
            ns.ImportAllTRP3Profiles()
        else
            print("|cFF00FF00[RPBox]|r 用法: /rpbox cache list/stats/import")
        end
    else
        print("|cFFFF0000[RPBox]|r 未知命令，输入 /rpbox help 查看帮助")
    end
end

-- 注册斜杠命令
SLASH_RPBOX1 = "/rpbox"
SlashCmdList["RPBOX"] = HandleSlashCommand

-- 目标选中计时器（自动白名单）
local targetTimer = nil

local function OnTargetChanged()
    if targetTimer then
        targetTimer:Cancel()
        targetTimer = nil
    end

    -- 只处理玩家目标，忽略 NPC/怪物
    if not UnitIsPlayer("target") then return end

    local unitID = ns.GetUnitID("target")
    if not unitID or ns.IsWhitelisted(unitID) then return end

    targetTimer = C_Timer.NewTimer(2, function()
        ns.AddToWhitelist(unitID)
        targetTimer = nil
    end)
end

-- 更新同步状态
function ns.UpdateSyncState()
    RPBox_Sync.addon = {
        lastUpdate = time(),
        recordCount = ns.GetTotalRecordCount(),
        version = RECORD_SCHEMA_VERSION,
    }
end

-- 新消息回调列表
local onNewMessageCallbacks = {}

-- 注册新消息回调
function ns.RegisterOnNewMessage(callback)
    table.insert(onNewMessageCallbacks, callback)
end

-- 触发新消息回调
function ns.TriggerOnNewMessage()
    for _, callback in ipairs(onNewMessageCallbacks) do
        pcall(callback)
    end
end

-- 名单变更回调列表
local onListChangeCallbacks = {}

-- 注册名单变更回调
function ns.RegisterOnListChange(callback)
    table.insert(onListChangeCallbacks, callback)
end

-- 触发名单变更回调
function ns.TriggerOnListChange()
    for _, callback in ipairs(onListChangeCallbacks) do
        pcall(callback)
    end
end

-- 获取总记录数
function ns.GetTotalRecordCount()
    local count = 0
    for date, hours in pairs(RPBox_ChatLog) do
        for hour, records in pairs(hours) do
            count = count + #records
        end
    end
    return count
end

-- 根据客户端状态清理旧数据
local function CleanupFromClientState()
    local clientState = RPBox_Sync and RPBox_Sync.client
    if clientState and clientState.clearedBefore then
        ns.ClearRecordsBefore(clientState.clearedBefore)
    end
end

-- 清理指定时间之前的记录
function ns.ClearRecordsBefore(timestamp)
    local cleared = 0
    for date, hours in pairs(RPBox_ChatLog) do
        for hour, records in pairs(hours) do
            local newRecords = {}
            for _, record in ipairs(records) do
                local recordTime = record.t or record.timestamp or 0
                if recordTime >= timestamp then
                    table.insert(newRecords, record)
                else
                    cleared = cleared + 1
                end
            end
            if #newRecords > 0 then
                RPBox_ChatLog[date][hour] = newRecords
            else
                RPBox_ChatLog[date][hour] = nil
            end
        end
        -- 清理空日期
        if not next(RPBox_ChatLog[date]) then
            RPBox_ChatLog[date] = nil
        end
    end
    ns.PruneProfileSnapshots(true)
    return cleared
end

-- 清理记录命令
function ns.ClearRecords(clearAll, confirmed)
    if clearAll then
        if not confirmed then
            print(L["CLEAR_CONFIRM"] or "|cFFFFFF00[RPBox]|r 输入 /rpbox clear all confirm 确认")
            return
        end
        local count = ns.GetTotalRecordCount()
        RPBox_ChatLog = {}
        ResetProfileHistoryState()
        print(format(L["CLEAR_DONE"] or "[RPBox] 已清理 %d 条记录", count))
    else
        -- 只清理已同步的
        local clientState = RPBox_Sync and RPBox_Sync.client
        if clientState and clientState.lastSync then
            local cleared = ns.ClearRecordsBefore(clientState.lastSync)
            print(format(L["CLEAR_DONE"] or "[RPBox] 已清理 %d 条记录", cleared))
        end
    end
    ns.UpdateSyncState()
end

-- 事件框架
local frame = CreateFrame("Frame")
frame:RegisterEvent("ADDON_LOADED")
frame:RegisterEvent("PLAYER_TARGET_CHANGED")

frame:SetScript("OnEvent", function(self, event, arg1)
    if event == "ADDON_LOADED" and arg1 == ADDON_NAME then
        InitSavedVariables()
        CleanupFromClientState()
        print(L["ADDON_LOADED"] or "|cFF00FF00[RPBox]|r 插件已加载")

        -- 注册TRP3事件监听
        if TRP3_Addon and TRP3_Addon.RegisterCallback then
            -- print("|cFF00FF00[RPBox]|r 正在注册TRP3事件监听...")
            TRP3_Addon.RegisterCallback(ns, "REGISTER_PROFILES_LOADED", function()
                OnLocalProfilesLoaded()
            end)

            TRP3_Addon.RegisterCallback(ns, "REGISTER_DATA_UPDATED", function(_, unitID, profileID)
                local playerID = ns.GetPlayerID()
                local selfContext = ns.GetSelfProfileContext()
                local isLocalUpdate = (unitID and playerID and unitID == playerID)
                    or (not unitID and selfContext and profileID == selfContext.profileID)

                if isLocalUpdate then
                    OnLocalProfileDataUpdated()
                    return
                end

                if unitID then
                    local context = ns.GetRemoteProfileContext(unitID)
                    if context then
                        ns.ObserveRemoteProfileIdentity(unitID, context.profileID, context.profile, true)
                        ns.CacheProfile(context.profileID, context.profile)
                    end
                end
            end)
            -- print("|cFF00FF00[RPBox]|r TRP3事件监听注册成功！")

            -- 批量导入 TRP3 已有的人物卡数据
            C_Timer.After(0, SeedLocalProfileBaseline)
            C_Timer.After(1, function()
                SeedLocalProfileBaseline()
                ns.ImportAllTRP3Profiles()
            end)
        else
            print("|cFFFFFF00[RPBox]|r 警告: TRP3未加载或不支持事件监听")
        end
    elseif event == "PLAYER_TARGET_CHANGED" then
        OnTargetChanged()
    end
end)
