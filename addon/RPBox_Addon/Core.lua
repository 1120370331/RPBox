-- RPBox Addon Core
-- 插件核心框架

local ADDON_NAME, ns = ...

-- 版本信息
ns.VERSION = "1.0.0"

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
    maxRecords = 10000,
    warnThreshold = 9000,
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
    RPBox_ProfileExport = RPBox_ProfileExport or {}
    RPBox_Sync = RPBox_Sync or { addon = {}, client = {} }
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

    if not TRP3_API or not TRP3_API.register then
        -- print("|cFFFF0000[RPBox]|r UpdateProfileCache 失败: TRP3 API不可用")
        return
    end

    -- 检查是否已知该玩家
    if not TRP3_API.register.isUnitIDKnown(unitID) then
        -- print("|cFFFFFF00[RPBox]|r UpdateProfileCache 跳过: TRP3尚未知晓玩家 " .. unitID)
        return
    end

    local character = TRP3_API.register.getUnitIDCharacter(unitID)
    if not character or not character.profileID then
        -- print("|cFFFF0000[RPBox]|r UpdateProfileCache 失败: 无法获取角色信息")
        return
    end

    local profileID = character.profileID
    -- print("|cFF00FF00[RPBox]|r 获取到profileID: " .. tostring(profileID))

    local profile = TRP3_API.register.getProfile(profileID)
    if not profile or not (profile.characteristics or profile.about or profile.misc) then
        -- print("|cFFFF0000[RPBox]|r UpdateProfileCache 失败: 无法获取profile数据")
        return
    end

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
        version = 1,
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
            TRP3_Addon.RegisterCallback(ns, "REGISTER_DATA_UPDATED", function(_, unitID, hasProfile)
                -- 当TRP3收到新的人物卡数据时，自动更新缓存
                -- print("|cFF00FF00[RPBox]|r TRP3事件触发: unitID=" .. tostring(unitID) .. ", hasProfile=" .. tostring(hasProfile))
                if hasProfile and unitID then
                    ns.UpdateProfileCache(unitID)
                end
            end)
            -- print("|cFF00FF00[RPBox]|r TRP3事件监听注册成功！")

            -- 批量导入 TRP3 已有的人物卡数据
            C_Timer.After(1, function()
                ns.ImportAllTRP3Profiles()
            end)
        else
            print("|cFFFFFF00[RPBox]|r 警告: TRP3未加载或不支持事件监听")
        end
    elseif event == "PLAYER_TARGET_CHANGED" then
        OnTargetChanged()
    end
end)
