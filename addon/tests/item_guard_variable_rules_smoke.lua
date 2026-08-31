-- Smoke test for RPBox_Addon/ItemGuardVariableRules.lua.
-- Run from repository root:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_variable_rules_smoke.lua

local function fail(message)
    error("[item-guard-variable-rules-smoke] " .. tostring(message), 2)
end

local function assert_true(value, message)
    if not value then fail(message or "expected truthy value") end
end

local function assert_false(value, message)
    if value then fail(message or "expected falsy value") end
end

local function assert_equal(actual, expected, message)
    if actual ~= expected then
        fail((message or "values differ") .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual))
    end
end

local function effect(effectID, args)
    return { id = effectID, args = args or {} }
end

local function list(effects, nextStep)
    return { t = "list", e = effects or {}, n = nextStep }
end

local function workflow(steps)
    return { ST = steps }
end

local function root(entry, workflows)
    return { TY = "IT", BA = { NA = "test" }, US = entry and { SC = entry } or nil, SC = workflows }
end

local function analyze(value, context)
    return ns.ItemGuardVariableRules.Analyze("root", value, context)
end

local function has_kind(result, kind)
    for _, finding in ipairs(result.findings) do
        if finding.kind == kind then return true, finding end
    end
    return false
end

ns = {}
local chunk, loadError = loadfile("addon/RPBox_Addon/ItemGuardVariableRules.lua")
assert_true(chunk, loadError)
chunk("RPBox_Addon", ns)

local Rules = ns.ItemGuardVariableRules
assert_true(type(Rules) == "table", "namespace was not registered")
assert_true(type(Rules.Analyze) == "function", "Analyze is missing")
assert_true(type(Rules.ClassifyEffect) == "function", "ClassifyEffect is missing")
assert_true(type(Rules.LIMITS) == "table", "LIMITS are missing")

local temporary = analyze(root("main", {
    main = workflow({ ["1"] = list({ effect("var_object", { "w", "=", "temp", string.rep("x", 70000) }) }) }),
}))
assert_false(temporary.blocked, "workflow-local variables are not persistent")
assert_equal(temporary.behaviorScore, 0, "workflow-local value gained behavior score")
assert_equal(temporary.metrics.temporaryWrites, 1, "temporary write metric is missing")
assert_equal(temporary.metrics.persistentWrites, 0, "temporary write counted as persistent")

local ordinary = analyze(root("main", {
    main = workflow({ ["1"] = list({
        effect("var_object", { "o", "=", "objectFlag", "yes" }),
        effect("var_object", { "c", "+", "campaignCount", 1 }),
    }) }),
}))
assert_false(ordinary.blocked, "ordinary fixed persistent writes must not block")
assert_equal(ordinary.behaviorScore, 0, "ordinary persistent writes gained risk score")
assert_equal(ordinary.metrics.uniquePersistentKeys, 2, "persistent keys were not separated by source")

local fixedLoop = analyze(root("main", {
    main = workflow({ ["1"] = list({ effect("var_object", { "o", "=", "state", "ready" }) }, "1") }),
}))
assert_false(fixedLoop.blocked, "a loop overwriting one fixed short value must not block")
assert_equal(fixedLoop.behaviorScore, 0, "fixed overwrite loop gained behavior score")
assert_equal(fixedLoop.amplificationScore, 15, "fixed overwrite loop lost its structural score")
assert_equal(fixedLoop.metrics.repeatedPersistentWrites, 1, "repeated write metric is missing")

local dynamicLoop = analyze(root("main", {
    main = workflow({ ["1"] = list({ effect("var_object", { "o", "=", "entry-${counter}", "x" }) }, "1") }),
}))
assert_true(dynamicLoop.blocked, "a repeated dynamic persistent key must block")
assert_true(has_kind(dynamicLoop, "variable_dynamic_key_amplified"), "dynamic-key finding is missing")

local growingLoop = analyze(root("main", {
    main = workflow({ ["1"] = list({ effect("var_object", {
        "o", "=", "log", "${log}xxxxxxxxxxxxxxxx"
    }) }, "1") }),
}))
assert_true(growingLoop.blocked, "self-appending persistent value in a loop must block")
assert_true(has_kind(growingLoop, "variable_value_growth_amplified"), "growth finding is missing")

local doublingLoop = analyze(root("main", {
    main = workflow({ ["1"] = list({ effect("var_object", {
        "o", "=", "log", "${log}${log}"
    }) }, "1") }),
}))
assert_true(doublingLoop.blocked, "self-doubling persistent value in a loop must block")

local manyEffects = {}
for index = 1, Rules.LIMITS.STATIC_UNIQUE_KEYS + 1 do
    manyEffects[index] = effect("var_object", { "o", "=", "key" .. index, "x" })
end
local manyKeys = analyze(root("main", { main = workflow({ ["1"] = list(manyEffects) }) }))
assert_true(manyKeys.blocked, "too many independent persistent keys must block")
assert_true(has_kind(manyKeys, "variable_unique_key_exhaustion"), "unique-key finding is missing")

local hugeValue = analyze(root("main", {
    main = workflow({ ["1"] = list({ effect("var_object", {
        "o", "=", "blob", string.rep("x", Rules.LIMITS.SINGLE_LITERAL_BYTES + 1)
    }) }) }),
}))
assert_true(hugeValue.blocked, "a huge persistent literal must block")
assert_true(has_kind(hugeValue, "variable_single_literal_exhaustion"), "single-literal finding is missing")

local accumulatedEffects = {}
for index = 1, 8 do
    accumulatedEffects[index] = effect("var_object", { "o", "=", "blob" .. index, string.rep("x", 40000) })
end
local accumulated = analyze(root("main", { main = workflow({ ["1"] = list(accumulatedEffects) }) }))
assert_true(accumulated.blocked, "large accumulated persistent literals must block")
assert_true(has_kind(accumulated, "variable_total_literal_exhaustion"), "total-literal finding is missing")

local operand = Rules.ClassifyEffect("var_operand", { "roll", "c", "random", { 1, 100 } })
assert_true(operand and operand.persistent, "campaign operand write was not classified persistent")
assert_equal(operand.kind, "operand", "operand kind changed")
assert_equal(operand.name, "roll", "operand variable name changed")
assert_true(operand.dynamicValue, "operand output must remain runtime-unknown")
assert_equal(operand.estimatedBytes, nil, "operand output size must not be invented")

local promptLoop = analyze(root("prompt", {
    prompt = workflow({ ["1"] = list({ effect("var_prompt", {
        "Enter a value", "answer", "o", "prompt", "o"
    }) }) }),
}))
assert_false(promptLoop.blocked, "human-gated prompt callback must not be automatic recursion")
assert_equal(promptLoop.amplificationScore, 0, "prompt callback fabricated amplification")
assert_equal(promptLoop.metrics.promptWrites, 1, "prompt metric is missing")

local disconnected = analyze(root("main", {
    main = workflow({
        ["1"] = list({ effect("text", { "safe" }) }),
        ["99"] = list({ effect("var_object", {
            "o", "=", "blob", string.rep("x", Rules.LIMITS.SINGLE_LITERAL_BYTES + 1)
        }) }, "99"),
    }),
}))
assert_false(disconnected.blocked, "disconnected editor remnant must not be analyzed")
assert_equal(disconnected.metrics.persistentWrites, 0, "disconnected persistent write was counted")

local guardedArgs = setmetatable({}, {
    __index = function() error("Lua Script Effect args must remain opaque") end,
    __pairs = function() error("Lua Script Effect args must remain opaque") end,
})
local opaqueLua = analyze(root("main", {
    main = workflow({ ["1"] = list({ { id = "script", args = guardedArgs } }) }),
}))
assert_false(opaqueLua.blocked, "opaque Lua effect fabricated variable risk")
assert_equal(opaqueLua.metrics.effectsAnalyzed, 0, "unrelated Lua effect entered variable analysis")

local stableA = root("main", {})
stableA.SC.main = workflow({ ["1"] = list({
    effect("var_object", { "o", "=", "a", "1" }),
    effect("var_operand", { "b", "c", "random", { min = 1, max = 2 } }),
}) })
local stableB = root("main", {})
stableB.SC.main = workflow({ ["1"] = list({
    [2] = effect("var_operand", { "b", "c", "random", { max = 2, min = 1 } }),
    [1] = effect("var_object", { "o", "=", "a", "1" }),
}) })
assert_equal(analyze(stableA).fingerprint, analyze(stableB).fingerprint,
    "fingerprint changed with insertion order")

local runtime = Rules.EvaluateRuntime({ shortWrites = Rules.LIMITS.SHORT_WINDOW_WRITES + 1 })
assert_true(runtime.blocked, "runtime short-window quota did not produce evidence")
assert_true(has_kind(runtime, "variable_runtime_short_writes"), "runtime finding is missing")

local crashSizedRuntime = Rules.EvaluateRuntime({
    singleBytes = Rules.LIMITS.RUNTIME_SINGLE_VALUE_BYTES + 1,
})
assert_true(crashSizedRuntime.blocked, "crash-sized single runtime value was not blocked")
assert_true(has_kind(crashSizedRuntime, "variable_runtime_single_crash_size"),
    "crash-sized runtime finding is missing")

print("[item-guard-variable-rules-smoke] PASS")
