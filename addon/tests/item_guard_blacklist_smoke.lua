-- Dynamic smoke test for RPBox_Addon/ItemGuardBlacklist.lua.
-- Run from the repository root with:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_blacklist_smoke.lua

local function fail(message)
    error("[item-guard-blacklist-smoke] " .. tostring(message), 2)
end

local function assert_true(value, message)
    if not value then fail(message or "expected truthy value") end
end

local function assert_equal(actual, expected, message)
    if actual ~= expected then
        fail((message or "values differ") .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual))
    end
end

RPBox_ItemGuardDB = nil
TRP3_Security = { sender = {} }

local ns = {}
local chunk, loadError = loadfile("addon/RPBox_Addon/ItemGuardBlacklist.lua")
if not chunk then fail(loadError) end
chunk("RPBox_Addon", ns)

local blacklist = ns.ItemGuardBlacklist
assert_true(blacklist ~= nil, "module was not exported")
assert_true(blacklist.Initialize(), "initialize failed")
assert_true(type(RPBox_ItemGuardDB.userBlacklist) == "table", "user blacklist was not initialized")

assert_equal(
    blacklist.NormalizeIdentity("  TestName — SmokeRealm  "),
    "testname-smokerealm",
    "identity normalization failed"
)
assert_equal(blacklist:NormalizeIdentity("ColonName-ColonRealm"), "colonname-colonrealm", "colon API normalization failed")
assert_equal(blacklist.NormalizeIdentity("missingrealm"), nil, "unqualified identity should be rejected")

local builtinExpected = {
    ["蕾火演员死冯-金色平原"] = true,
    ["工作人员二号-金色平原"] = true,
    ["绿宝石兽-金色平原"] = true,
}
local builtinCount = 0
for _, entry in ipairs(blacklist.GetEntries()) do
    if entry.source == "builtin" then
        builtinCount = builtinCount + 1
        assert_true(builtinExpected[entry.identity], "unexpected built-in identity: " .. tostring(entry.identity))
        assert_equal(entry.reason, "RPBox 内置恶意道具来源名单", "built-in reason changed")
    end
end
assert_equal(builtinCount, 3, "built-in list count is incorrect")
for identity in pairs(builtinExpected) do
    local builtinMatch = blacklist.MatchRoot("builtin-" .. identity, { MD = { CB = identity } })
    assert_equal(builtinMatch and builtinMatch.identity, identity, "built-in identity did not match: " .. identity)
    assert_equal(builtinMatch and builtinMatch.source, "builtin", "built-in source was not reported: " .. identity)
end

local changed = 0
blacklist.RegisterOnChanged(function() changed = changed + 1 end)
local ok, message = blacklist.AddUser("BadActor-SmokeRealm", "manual report")
assert_true(ok, message)
assert_equal(changed, 1, "add did not notify listeners")
assert_true(RPBox_ItemGuardDB.userBlacklist["badactor-smokerealm"] ~= nil, "user entry was not persisted")

-- Initialize is deliberately idempotent and preserves user-owned entries.
blacklist.Initialize()
assert_true(RPBox_ItemGuardDB.userBlacklist["badactor-smokerealm"] ~= nil, "initialize discarded persistence")

local rootID = "root-smoke"
local match = blacklist.MatchRoot(rootID, { MD = { CB = "BadActor-SmokeRealm" } })
assert_equal(match and match.identity, "badactor-smokerealm", "creator match failed")
assert_equal(match and match.field, "MD.CB", "creator field was not reported")
assert_equal(match and match.source, "user", "creator source was not reported")

match = blacklist.MatchRoot(rootID, { MD = { SB = "BadActor-SmokeRealm" } })
assert_equal(match and match.field, "MD.SB", "last-editor match failed")

TRP3_Security.sender[rootID] = "BadActor-SmokeRealm"
match = blacklist.MatchRoot(rootID, { MD = {} })
assert_equal(match and match.field, "TRP3_Security.sender", "transport sender match failed")
TRP3_Security.sender[rootID] = nil

assert_equal(
    blacklist.MatchRoot("similar-root", { MD = { CB = "绿宝石兽二号-金色平原" } }),
    nil,
    "similar identity must not fuzzy-match"
)

ok, message = blacklist.RemoveUser("蕾火演员死冯-金色平原")
assert_true(not ok, "built-in entry was removable")
assert_true(tostring(message):find("不可删除", 1, true) ~= nil, "built-in removal did not explain rejection")

ok, message = blacklist.RemoveUser("BadActor-SmokeRealm")
assert_true(ok, message)
assert_equal(changed, 2, "remove did not notify listeners")
assert_equal(RPBox_ItemGuardDB.userBlacklist["badactor-smokerealm"], nil, "user entry was not removed")
assert_equal(blacklist.MatchRoot(rootID, { MD = { CB = "BadActor-SmokeRealm" } }), nil, "removed user still matched")

print("item_guard_blacklist_smoke: PASS")
