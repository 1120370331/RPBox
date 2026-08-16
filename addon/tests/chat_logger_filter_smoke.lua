-- Focused smoke test for the optional non-RP player filter in ChatLogger.lua.
--
-- Run from the repository root with:
--   npx --yes --package=fengari-node-cli fengari addon/tests/chat_logger_filter_smoke.lua

local function fail(message)
    error("[chat-logger-filter-smoke] " .. tostring(message), 2)
end

local function assert_equal(actual, expected, message)
    if actual ~= expected then
        fail((message or "values differ") .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual))
    end
end

local function new_frame()
    local frame = { scripts = {}, events = {} }
    function frame:RegisterEvent(event) self.events[event] = true end
    function frame:SetScript(event, callback) self.scripts[event] = callback end
    return frame
end

local chatEventFrame = nil
function CreateFrame()
    chatEventFrame = new_frame()
    return chatEventFrame
end

local now = 1700000000
function time() return now end
function date(pattern, timestamp) return os.date(pattern, timestamp) end
function GetTimePreciseSec() return now end

function GetRealmName() return "SmokeRealm" end
function IsInGuild() return false end
function GetNumGuildMembers() return 0 end
function GetGuildRosterInfo() return nil end
function GetPlayerInfoByGUID() return nil, nil end
function UnitClass() return "Tester", "MAGE" end

format = string.format
RPBox_Config = {
    enabled = true,
    channels = { SAY = true },
    whitelist = {},
    blacklist = {},
}
RPBox_ChatLog = {}

local knownRPPlayers = {
    ["Roleplayer-SmokeRealm"] = true,
}

local ns = {
    L = {},
    GetPlayerID = function() return "Tester-SmokeRealm" end,
    GetRemoteProfileContext = function(unitID)
        if not knownRPPlayers[unitID] then return nil end
        return {
            profileID = "profile:" .. unitID,
            profile = { characteristics = { FN = unitID:match("^([^-]+)") } },
            gameID = unitID,
        }
    end,
    GetSelfProfileContext = function() return nil end,
    CacheProfile = function() end,
    ObserveRemoteProfileIdentity = function() return "snapshot" end,
    IsBlacklisted = function(unitID) return RPBox_Config.blacklist[unitID] == true end,
    IsWhitelisted = function(unitID) return RPBox_Config.whitelist[unitID] == true end,
    ApplyRecordSchema = function() end,
    UpdateSyncState = function() end,
    TriggerOnNewMessage = function() end,
    GetTotalRecordCount = function()
        local count = 0
        for _, hours in pairs(RPBox_ChatLog) do
            for _, records in pairs(hours) do count = count + #records end
        end
        return count
    end,
}

local chunk, loadError = loadfile("addon/RPBox_Addon/ChatLogger.lua")
if not chunk then fail("could not load ChatLogger.lua: " .. tostring(loadError)) end
local ok, runtimeError = xpcall(function() chunk("RPBox_Addon", ns) end, debug.traceback)
if not ok then fail("ChatLogger.lua failed during load:\n" .. tostring(runtimeError)) end

local onEvent = chatEventFrame and chatEventFrame.scripts.OnEvent
if type(onEvent) ~= "function" then fail("chat event handler was not registered") end

local function record_count()
    return ns.GetTotalRecordCount()
end

local function send(sender, message)
    now = now + 1
    onEvent(chatEventFrame, "CHAT_MSG_SAY", message, sender)
end

-- Missing/false config is the default: ordinary players are recorded.
send("OrdinaryOne-SmokeRealm", "default-off")
assert_equal(record_count(), 1, "missing filter config should not block non-RP players")
RPBox_Config.ignoreNonRPPlayers = false
send("OrdinaryTwo-SmokeRealm", "explicitly-off")
assert_equal(record_count(), 2, "disabled filter should not block non-RP players")

-- Enabled config restores the former RP-card-only behavior.
RPBox_Config.ignoreNonRPPlayers = true
send("OrdinaryThree-SmokeRealm", "filtered")
assert_equal(record_count(), 2, "enabled filter should block a player without an RP card")
send("Roleplayer-SmokeRealm", "known-rp-player")
assert_equal(record_count(), 3, "enabled filter should retain a player with an RP card")

-- White/black lists keep their established precedence.
RPBox_Config.whitelist["Trusted-SmokeRealm"] = true
send("Trusted-SmokeRealm", "whitelisted")
assert_equal(record_count(), 4, "whitelist should override the non-RP filter")
RPBox_Config.blacklist["Roleplayer-SmokeRealm"] = true
send("Roleplayer-SmokeRealm", "blacklisted")
assert_equal(record_count(), 4, "blacklist should still block an RP player")

print("PASS chat logger non-RP filter smoke: default off, opt-in filter, white/black list precedence")
