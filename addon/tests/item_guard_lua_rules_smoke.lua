-- Smoke test for RPBox_Addon/ItemGuardLuaRules.lua.

local function fail(message) error("[item-guard-lua-rules-smoke] " .. tostring(message), 2) end
local function assert_true(value, message) if not value then fail(message) end end
local function assert_false(value, message) if value then fail(message) end end
local function assert_equal(actual, expected, message)
    if actual ~= expected then fail((message or "values differ") .. ": expected " .. tostring(expected)
        .. ", got " .. tostring(actual)) end
end
local function has_kind(result, kind)
    for _, finding in ipairs(result.findings or {}) do if finding.kind == kind then return true end end
    return false
end

TRP3_DB = { inner = {}, types = { ITEM = "IT" } }
local ns = {}
assert(loadfile("addon/RPBox_Addon/ItemGuardLuaRules.lua"))("RPBox_Addon", ns)
local Rules = ns.ItemGuardLuaRules

local safe = Rules.AnalyzeCode([[
-- while true do end; args.scripts.bad = true
local text = "effect('item_add') while true do end"
local total = 0
for i = 1, 20 do total = total + i end
setVar(args, "w", "total", total)
]], { rootID = "safe" })
assert_false(safe.blocked, "comments, strings, and bounded local work were blocked")

local infinite = Rules.AnalyzeCode("while true do local x = 1 end", { rootID = "infinite" })
assert_true(infinite.blocked, "unbounded loop was not blocked")
assert_true(has_kind(infinite, "lua_unbounded_execution"), "unbounded-loop finding is missing")

local hugeLoop = Rules.AnalyzeCode("for i=1,1000001 do local x=i end", { rootID = "huge-loop" })
assert_true(hugeLoop.blocked, "million-iteration literal loop was not blocked")

local recursive = Rules.AnalyzeCode("local function visit() visit() end; visit()", { rootID = "recursive" })
assert_true(recursive.blocked and has_kind(recursive, "lua_recursive_function"),
    "untrusted direct recursion did not require publisher trust")
local trustedRecursive = Rules.AnalyzeCode("local function visit() visit() end; visit()", {
    rootID = "trusted-recursive",
    trustedPublisher = true,
})
assert_false(trustedRecursive.blocked, "trusted publisher could not use policy-level recursion")

local sharedWrite = Rules.AnalyzeCode("string.gsub = nil", { rootID = "shared" })
assert_true(sharedWrite.blocked and has_kind(sharedWrite, "lua_shared_library_write"),
    "shared library mutation was not blocked")
local aliasWrite = Rules.AnalyzeCode("local s = string; s.find = nil", { rootID = "alias" })
assert_true(aliasWrite.blocked, "aliased shared library mutation was not blocked")

local scriptWrite = Rules.AnalyzeCode("args.scripts.onUse = {}", { rootID = "scripts" })
assert_true(scriptWrite.blocked and has_kind(scriptWrite, "lua_scripts_direct_write"),
    "direct script definition mutation was not blocked")
local contextMutator = Rules.AnalyzeCode(
    "table.insert(args.container.content, {id='bad'})",
    { rootID = "context-mutator" }
)
assert_true(contextMutator.blocked and has_kind(contextMutator, "lua_context_table_mutator"),
    "table-based inventory context mutation was not blocked")

local directVar = Rules.AnalyzeCode("args.object.vars.value = 1", { rootID = "var" })
assert_false(directVar.blocked, "one direct scalar variable write should be advisory")
assert_true(directVar.advisory and has_kind(directVar, "lua_variable_direct_write"),
    "direct variable advisory is missing")
local trustedDirectVar = Rules.AnalyzeCode("args.object.vars.value = 1", {
    rootID = "trusted-var",
    trustedPublisher = true,
})
assert_false(trustedDirectVar.blocked or trustedDirectVar.advisory,
    "trusted publisher did not suppress advisory-only Lua findings")
local loopingVar = Rules.AnalyzeCode(
    "for i=1,20 do args.object.vars[i] = string.rep('x', i) end",
    { rootID = "loop-var" }
)
assert_true(loopingVar.blocked and has_kind(loopingVar, "lua_loop_direct_persistence"),
    "looping direct variable growth was not blocked")

local safeOperand = Rules.AnalyzeCode('local x = op("check_event_var", args, 1)', { rootID = "safe-op" })
assert_false(safeOperand.blocked, "strict numeric operand was blocked")
local unsafeOperand = Rules.AnalyzeCode(
    'local x = op("check_event_var", args, "1) or _G")',
    { rootID = "unsafe-op", trustedPublisher = true }
)
assert_true(unsafeOperand.blocked and has_kind(unsafeOperand, "lua_operand_code_injection"),
    "unsafe operand code position was bypassed by publisher trust")
local dynamicOperand = Rules.AnalyzeCode("local x = op(which, args, 1)", { rootID = "dynamic-op" })
assert_true(dynamicOperand.blocked, "dynamic operand ID was not blocked")
local trustedNumericExpression = Rules.AnalyzeCode(
    'local x = op("check_event_var", args, 1+0)', {
    rootID = "trusted-numeric-op",
    trustedPublisher = true,
})
assert_false(trustedNumericExpression.blocked,
    "constant numeric operand expression was not proven safe")
local trustedDynamicOperand = Rules.AnalyzeCode("local x = op(which, args, 1)", {
    rootID = "trusted-dynamic-op",
    trustedPublisher = true,
})
assert_true(trustedDynamicOperand.blocked,
    "trusted publisher bypassed a dynamic operand escape surface")

local dynamicEffect = Rules.AnalyzeCode("effect(which, args)", { rootID = "dynamic-effect" })
assert_true(dynamicEffect.blocked, "untrusted dynamic effect dispatch was not blocked")
local trustedDynamicEffect = Rules.AnalyzeCode(
    "effect(which, args)",
    { rootID = "trusted-effect", trustedPublisher = true }
)
assert_false(trustedDynamicEffect.blocked, "trusted publisher did not bypass policy-only dynamic effect")

local effectLoop = Rules.AnalyzeCode(
    'for i=1,20 do effect("sound_id_self", args, "SFX", 42, false) end',
    { rootID = "effect-loop" }
)
assert_true(effectLoop.blocked and has_kind(effectLoop, "lua_loop_high_impact_effect"),
    "untrusted looped high-impact effect was not blocked")
local trustedEffectLoop = Rules.AnalyzeCode(
    'for i=1,20 do effect("sound_id_self", args, "SFX", 42, false) end',
    { rootID = "trusted-effect-loop", trustedPublisher = true }
)
assert_false(trustedEffectLoop.blocked, "trusted publisher did not bypass combination policy")

local unrolledParts = {}
for index = 1, 101 do
    unrolledParts[index] = 'effect("sound_id_self",args,"SFX",42,false)'
end
local unrolledFlood = Rules.AnalyzeCode(table.concat(unrolledParts, ";"), {
    rootID = "unrolled-flood",
    trustedPublisher = true,
})
assert_true(unrolledFlood.blocked and has_kind(unrolledFlood, "lua_unrolled_effect_flood"),
    "trusted publisher bypassed an unrolled high-impact effect flood")

local nestedLoop = Rules.AnalyzeCode(
    'for i=1,2 do effect("script", args, "return 0") end',
    { rootID = "nested-loop", trustedPublisher = true }
)
assert_true(nestedLoop.blocked, "looped nested Script Effect was bypassable")

local globalRequest = Rules.AnalyzeCode("local G = args._G", {
    rootID = "global-request",
    trustedPublisher = true,
})
assert_false(globalRequest.blocked, "trusted UI compatibility was blocked solely for requesting _G")
assert_true(Rules.AnalyzeCode("local G = args._G").blocked, "untrusted _G request was automatically authorized")
assert_false(Rules.AnalyzeCode("local n=math.ceil(1.2); n=3; local x=math.pi; x=2").blocked,
    "local scalar assignment was mistaken for a shared-library write")

local function effect(id, args) return { id = id, args = args or {} } end
local function workflow(effects) return { ST = { ["1"] = { t = "list", e = effects } } } end
local macroRoot = {
    TY = "IT",
    BA = { NA = "Bootstrap" },
    US = { SC = "onUse" },
    SC = {
        onUse = workflow({ effect("secure_macro", {
            "/run TRP3_API.script.runLuaScriptEffect=function(c,a,s) a._G=_G end",
        }) }),
    },
}
local macroResult = Rules.Analyze("macro", macroRoot, { trustedPublisher = true })
assert_true(macroResult.blocked and has_kind(macroResult, "secure_macro_guard_escape"),
    "trusted publisher bypassed the known global bootstrap macro")

local disconnectedRoot = {
    TY = "IT",
    BA = { NA = "Disconnected" },
    US = { SC = "safe" },
    SC = {
        safe = workflow({ effect("text", { "safe" }) }),
        unused = workflow({ effect("script", { "while true do end" }) }),
    },
}
assert_false(Rules.Analyze("disconnected", disconnectedRoot, {}).blocked,
    "disconnected editor Lua remnant was analyzed")

TRP3_DB.inner.reserved = { TY = "IT" }
local collisionRoot = {
    TY = "IT",
    BA = { NA = "Collision" },
    US = { SC = "onUse" },
    SC = { onUse = workflow({ effect("text", { "safe" }) }) },
}
local collision = Rules.Analyze("reserved", collisionRoot, { trustedPublisher = true })
assert_true(collision.blocked and has_kind(collision, "reserved_inner_id_collision"),
    "trusted publisher bypassed reserved TRP3 inner ID collision")

local first = Rules.AnalyzeCode("local value = 1", { rootID = "fingerprint" })
local second = Rules.AnalyzeCode("local value = 2", { rootID = "fingerprint" })
assert_true(first.fingerprint ~= second.fingerprint, "Lua source content was omitted from fingerprint")

print("item_guard_lua_rules_smoke: PASS")
