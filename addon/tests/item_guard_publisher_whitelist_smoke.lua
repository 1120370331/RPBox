-- Smoke test for RPBox_Addon/ItemGuardPublisherWhitelist.lua.

local function fail(message) error("[publisher-whitelist-smoke] " .. tostring(message), 2) end
local function assert_true(value, message) if not value then fail(message) end end
local function assert_equal(actual, expected, message)
    if actual ~= expected then fail((message or "values differ") .. ": expected " .. tostring(expected)
        .. ", got " .. tostring(actual)) end
end

RPBox_ItemGuardDB = nil
TRP3_Security = { sender = {} }
TRP3_API = { globals = { player_id = "Self-SmokeRealm" } }
TRP3_DB = { my = { self = true } }

local ns = {}
assert(loadfile("addon/RPBox_Addon/ItemGuardBlacklist.lua"))("RPBox_Addon", ns)
assert(loadfile("addon/RPBox_Addon/ItemGuardPublisherWhitelist.lua"))("RPBox_Addon", ns)
local whitelist = ns.ItemGuardPublisherWhitelist
assert_true(whitelist.Initialize(), "whitelist did not initialize")

assert_equal(#whitelist.GetEntries(), 0, "publisher whitelist should not contain system defaults")
assert_equal(whitelist.MatchRoot("not-default", { MD = { CB = "伊迪-金色平原" } }), nil,
    "publisher was trusted without an explicit user entry")

local ownRoot = { MD = { CB = "Self-SmokeRealm" } }
TRP3_DB.my.self = ownRoot
local selfMatch = whitelist.MatchRoot("self", ownRoot)
assert_equal(selfMatch and selfMatch.source, "self", "current player was not trusted")
assert_equal(whitelist.MatchRoot("spoofed-self", { MD = { CB = "Self-SmokeRealm" } }), nil,
    "self identity metadata was trusted without TRP3_DB.my ownership")

local changed = 0
whitelist.RegisterOnChanged(function() changed = changed + 1 end)
local ok, message = whitelist.AddUser("Trusted-SmokeRealm", "reviewed publisher")
assert_true(ok, message)
assert_equal(changed, 1, "publisher add did not notify")
assert_true(RPBox_ItemGuardDB.publisherWhitelist["trusted-smokerealm"] ~= nil,
    "publisher trust was not persisted")

local userMatch = whitelist.MatchRoot("user", { MD = { CB = "Trusted-SmokeRealm" } })
assert_equal(userMatch and userMatch.source, "user", "explicit user publisher trust was ignored")
assert_equal(whitelist.MatchRoot("self", { MD = { CB = "Self-SmokeRealm" } }), nil,
    "a replacement sharing an owned ID inherited trust")

TRP3_Security.sender.transport = "Unknown-SmokeRealm"
local spoofed = whitelist.MatchRoot("transport", { MD = { CB = "Trusted-SmokeRealm" } })
assert_equal(spoofed, nil, "trusted creator overrode an untrusted transport sender")

TRP3_Security.sender.transport = "Trusted-SmokeRealm"
local transported = whitelist.MatchRoot("transport", { MD = { CB = "Unknown-SmokeRealm" } })
assert_equal(transported and transported.field, "TRP3_Security.sender",
    "trusted transport sender was not authoritative")

ok, message = whitelist.RemoveUser("Trusted-SmokeRealm")
assert_true(ok, message)
assert_equal(changed, 2, "publisher remove did not notify")
assert_equal(whitelist.MatchRoot("user", { MD = { CB = "Trusted-SmokeRealm" } }), nil,
    "removed publisher remained trusted")

TRP3_API.security = { resolveEffectSecurity = function(id, effect)
    return id == "native-trusted" and effect == "script", 3
end }
local native = whitelist.MatchRoot("native-trusted", { MD = {} })
assert_equal(native and native.source, "trp3", "TRP3 authorization was not respected")
print("item_guard_publisher_whitelist_smoke: PASS")
