-- RPBox ItemSync
-- 道具标记模块

local ADDON_NAME, ns = ...
local L = ns.L or {}

-- 已标记的道具
ns.MarkedItems = {}

-- 标记道具
function ns.MarkItem(itemID)
    if not itemID or itemID == "" then
        print("|cFFFF0000[RPBox]|r 请指定道具ID")
        return
    end

    if not TRP3_API or not TRP3_API.extended then
        print("|cFFFF0000[RPBox]|r 需要 TotalRP3 Extended")
        return
    end

    ns.MarkedItems[itemID] = {
        markedAt = time(),
        itemID = itemID,
    }
    print("|cFF00FF00[RPBox]|r 道具已标记: " .. itemID)
end

-- 列出已标记道具
function ns.ListMarkedItems()
    local count = 0
    for id, info in pairs(ns.MarkedItems) do
        print(format("  - %s (标记于 %s)", id, date("%Y-%m-%d %H:%M", info.markedAt)))
        count = count + 1
    end
    if count == 0 then
        print("|cFF00FF00[RPBox]|r 暂无标记的道具")
    else
        print(format("|cFF00FF00[RPBox]|r 共 %d 个标记道具", count))
    end
end
