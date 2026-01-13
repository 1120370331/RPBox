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
    whitelist = {},
    blacklist = {},
    warnedThisSession = false,
    maxRecords = 10000,
    warnThreshold = 9000,
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
    RPBox_Config.whitelist[unitID] = true
    RPBox_Config.blacklist[unitID] = nil
    print(format(L["WHITELIST_ADDED"] or "[RPBox] %s 已加入白名单", unitID))
end

-- 添加到黑名单
function ns.AddToBlacklist(unitID)
    if not unitID then return end
    RPBox_Config.blacklist[unitID] = true
    RPBox_Config.whitelist[unitID] = nil
    print(format(L["BLACKLIST_ADDED"] or "[RPBox] %s 已加入黑名单", unitID))
end

-- 从白名单移除
function ns.RemoveFromWhitelist(unitID)
    if unitID then
        RPBox_Config.whitelist[unitID] = nil
    end
end

-- 从黑名单移除
function ns.RemoveFromBlacklist(unitID)
    if unitID then
        RPBox_Config.blacklist[unitID] = nil
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

-- 显示帮助信息
local function ShowHelp()
    print(L["HELP_TITLE"] or "|cFF00FF00[RPBox]|r 命令帮助:")
    print("  /rpbox whitelist add/remove 玩家名-服务器")
    print("  /rpbox blacklist add/remove 玩家名-服务器")
    print("  /rpbox export [target] - 导出人物卡")
    print("  /rpbox clear [all] - 清理记录")
    print("  /rpbox log [today] - 打开回放窗口")
    print("  /rpbox item mark/list - 道具标记")
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
        if subcmd == "add" and param ~= "" then
            ns.AddToWhitelist(param)
        elseif subcmd == "remove" and param ~= "" then
            ns.RemoveFromWhitelist(param)
            print("|cFF00FF00[RPBox]|r " .. param .. " 已从白名单移除")
        end
    elseif cmd == "blacklist" then
        if subcmd == "add" and param ~= "" then
            ns.AddToBlacklist(param)
        elseif subcmd == "remove" and param ~= "" then
            ns.RemoveFromBlacklist(param)
            print("|cFF00FF00[RPBox]|r " .. param .. " 已从黑名单移除")
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
                if record.timestamp >= timestamp then
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
    elseif event == "PLAYER_TARGET_CHANGED" then
        OnTargetChanged()
    end
end)
