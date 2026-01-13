-- RPBox Addon Core
-- 核心框架与初始化

local ADDON_NAME = "RPBox_Addon"
local VERSION = "1.0.0"

-- 创建全局命名空间
RPBox = RPBox or {}
RPBox.version = VERSION

-- 初始化 SavedVariables
RPBox_ChatLog = RPBox_ChatLog or {}
RPBox_ProfileCache = RPBox_ProfileCache or {}
RPBox_Config = RPBox_Config or {
    whitelist = {},
    blacklist = {},
    warnedThisSession = false,
}
RPBox_Sync = RPBox_Sync or {
    addon = {
        lastUpdate = 0,
        recordCount = 0,
        version = 1,
    },
    client = {},
}

-- 事件框架
local frame = CreateFrame("Frame")
frame:RegisterEvent("ADDON_LOADED")
frame:RegisterEvent("PLAYER_LOGOUT")

frame:SetScript("OnEvent", function(self, event, ...)
    if event == "ADDON_LOADED" then
        local name = ...
        if name == ADDON_NAME then
            RPBox:OnLoad()
        end
    elseif event == "PLAYER_LOGOUT" then
        RPBox:OnLogout()
    end
end)

-- 插件加载
function RPBox:OnLoad()
    -- 检查 TRP3 依赖
    if not TRP3_API then
        print("|cFFFF0000[RPBox]|r TotalRP3 未加载，插件功能受限")
        return
    end

    -- 处理客户端同步状态
    self:ProcessClientState()

    -- 初始化聊天记录模块
    if RPBox.ChatLogger then
        RPBox.ChatLogger:Init()
    end

    -- 注册斜杠命令
    self:RegisterCommands()

    print("|cFF00FF00[RPBox]|r v" .. VERSION .. " 已加载")
end

-- 退出时保存
function RPBox:OnLogout()
    -- 更新同步状态
    self:UpdateSyncState()
end

-- 处理客户端同步状态
function RPBox:ProcessClientState()
    local clientState = RPBox_Sync and RPBox_Sync.client
    if clientState and clientState.clearedBefore then
        -- 清理已被客户端处理的旧数据
        self:ClearRecordsBefore(clientState.clearedBefore)
    end
end

-- 清理指定时间之前的记录
function RPBox:ClearRecordsBefore(timestamp)
    local cleared = 0
    for date, hours in pairs(RPBox_ChatLog) do
        for hour, records in pairs(hours) do
            local newRecords = {}
            for _, record in ipairs(records) do
                if record.timestamp > timestamp then
                    table.insert(newRecords, record)
                else
                    cleared = cleared + 1
                end
            end
            if #newRecords > 0 then
                hours[hour] = newRecords
            else
                hours[hour] = nil
            end
        end
        -- 清理空日期
        local hasRecords = false
        for _ in pairs(hours) do
            hasRecords = true
            break
        end
        if not hasRecords then
            RPBox_ChatLog[date] = nil
        end
    end
    if cleared > 0 then
        print("|cFF00FF00[RPBox]|r 已清理 " .. cleared .. " 条已同步记录")
    end
end

-- 更新同步状态
function RPBox:UpdateSyncState()
    RPBox_Sync.addon = {
        lastUpdate = time(),
        recordCount = self:GetTotalRecordCount(),
        version = 1,
    }
end

-- 获取总记录数
function RPBox:GetTotalRecordCount()
    local count = 0
    for _, hours in pairs(RPBox_ChatLog) do
        for _, records in pairs(hours) do
            count = count + #records
        end
    end
    return count
end

-- 注册斜杠命令
function RPBox:RegisterCommands()
    SLASH_RPBOX1 = "/rpbox"
    SlashCmdList["RPBOX"] = function(msg)
        local cmd, arg = msg:match("^(%S*)%s*(.-)$")
        cmd = cmd:lower()

        if cmd == "status" or cmd == "" then
            self:ShowStatus()
        elseif cmd == "clear" then
            self:ClearCommand(arg)
        elseif cmd == "whitelist" then
            self:WhitelistCommand(arg)
        elseif cmd == "blacklist" then
            self:BlacklistCommand(arg)
        else
            print("|cFFFFFF00[RPBox]|r 命令:")
            print("  /rpbox status - 显示状态")
            print("  /rpbox clear - 清理已同步记录")
            print("  /rpbox whitelist add/remove 玩家名")
            print("  /rpbox blacklist add/remove 玩家名")
        end
    end
end

-- 显示状态
function RPBox:ShowStatus()
    local count = self:GetTotalRecordCount()
    print("|cFF00FF00[RPBox]|r 状态:")
    print("  版本: " .. VERSION)
    print("  记录数: " .. count)
    print("  白名单: " .. self:TableCount(RPBox_Config.whitelist) .. " 人")
    print("  黑名单: " .. self:TableCount(RPBox_Config.blacklist) .. " 人")
end

-- 清理命令
function RPBox:ClearCommand(arg)
    if arg == "all" then
        RPBox_ChatLog = {}
        print("|cFF00FF00[RPBox]|r 已清理全部记录")
    else
        print("|cFFFFFF00[RPBox]|r 使用 /rpbox clear all 清理全部记录")
    end
end

-- 白名单命令
function RPBox:WhitelistCommand(arg)
    local action, name = arg:match("^(%S+)%s+(.+)$")
    if action == "add" and name then
        RPBox_Config.whitelist[name] = true
        print("|cFF00FF00[RPBox]|r " .. name .. " 已加入白名单")
    elseif action == "remove" and name then
        RPBox_Config.whitelist[name] = nil
        print("|cFF00FF00[RPBox]|r " .. name .. " 已移出白名单")
    else
        print("|cFFFFFF00[RPBox]|r 用法: /rpbox whitelist add/remove 玩家名-服务器")
    end
end

-- 黑名单命令
function RPBox:BlacklistCommand(arg)
    local action, name = arg:match("^(%S+)%s+(.+)$")
    if action == "add" and name then
        RPBox_Config.blacklist[name] = true
        RPBox_Config.whitelist[name] = nil
        print("|cFF00FF00[RPBox]|r " .. name .. " 已加入黑名单")
    elseif action == "remove" and name then
        RPBox_Config.blacklist[name] = nil
        print("|cFF00FF00[RPBox]|r " .. name .. " 已移出黑名单")
    else
        print("|cFFFFFF00[RPBox]|r 用法: /rpbox blacklist add/remove 玩家名-服务器")
    end
end

-- 工具函数：计算表元素数量
function RPBox:TableCount(t)
    local count = 0
    for _ in pairs(t) do
        count = count + 1
    end
    return count
end
