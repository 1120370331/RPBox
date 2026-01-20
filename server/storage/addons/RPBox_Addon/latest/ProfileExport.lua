-- RPBox ProfileExport
-- 人物卡导出模块

local ADDON_NAME, ns = ...
local L = ns.L or {}

-- 导出人物卡
function ns.ExportProfile(unit)
    if not TRP3_API then
        print(L["NO_TRP3"] or "|cFFFF0000[RPBox]|r 未检测到 TotalRP3")
        return
    end

    local unitID = ns.GetUnitID(unit)
    if not unitID then
        print("|cFFFF0000[RPBox]|r 无效的目标")
        return
    end

    local profileData
    if unit == "player" then
        profileData = TRP3_API.profile.getData("player")
    else
        local character = TRP3_API.register.getUnitIDCharacter(unitID)
        if character and character.profileID then
            profileData = TRP3_API.register.getProfile(character.profileID)
        end
    end

    if not profileData then
        print("|cFFFF0000[RPBox]|r 无法获取人物卡数据")
        return
    end

    -- 保存到导出数据
    RPBox_ProfileExport.lastExport = time()
    RPBox_ProfileExport.profiles = RPBox_ProfileExport.profiles or {}
    RPBox_ProfileExport.profiles[unitID] = profileData

    print(format(L["EXPORT_DONE"] or "|cFF00FF00[RPBox]|r %s 的人物卡已导出", unitID))
end
