-- Smoke test for RPBox_Addon/ItemGuardAuraRules.lua.
-- Run from repository root:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_aura_rules_smoke.lua

local function fail(message)
    error("[item-guard-aura-rules-smoke] " .. tostring(message), 2)
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

local function workflow(effects)
    return { ST = { ["1"] = { t = "list", e = effects or {} } } }
end

local function aura(name, cancelWorkflow, scripts)
    return {
        TY = "AU",
        BA = { NA = name, CC = true },
        LI = cancelWorkflow and { OC = cancelWorkflow } or {},
        SC = scripts or {},
    }
end

local namespace = {}
local chunk = assert(loadfile("addon/RPBox_Addon/ItemGuardAuraRules.lua"))
chunk("RPBox_Addon", namespace)
local Rules = namespace.ItemGuardAuraRules
assert_true(Rules and Rules.Analyze, "aura rules were not exported")

local safe = aura("Safe", "cancel", {
    cancel = workflow({ effect("text", { "goodbye" }) }),
})
local safeResult = Rules.Analyze("safe", safe, {})
assert_false(safeResult.blocked, "ordinary cancellable aura was blocked")
assert_equal(safeResult.metrics.selfReapplications, 0, "safe aura reported a self reapplication")

local direct = aura("Direct", "cancel", {
    cancel = workflow({ effect("aura_apply", { "direct", "=" }) }),
})
local directResult = Rules.Analyze("direct", direct, {})
assert_true(directResult.blocked, "direct cancellation self-reapply was not blocked")
assert_equal(directResult.metrics.selfReapplications, 1, "direct self-reapply count is wrong")
assert_true(directResult.findings[1].hard, "self-reapply finding is not a hard block")

local indirect = aura("Indirect", "cancel", {
    cancel = workflow({ effect("run_workflow", { "o", "restore" }) }),
    restore = workflow({ effect("aura_apply", { "indirect", "=" }) }),
})
local indirectResult = Rules.Analyze("indirect", indirect, {})
assert_true(indirectResult.blocked, "indirect cancellation self-reapply was not blocked")
assert_equal(indirectResult.metrics.cancellationWorkflows, 2, "cancel call graph was not followed")

local nestedRoot = {
    TY = "IT",
    BA = { NA = "Aura carrier" },
    IN = {
        curse = aura("Nested curse", "cancel", {
            cancel = workflow({ effect("aura_apply", { "carrier curse", "=" }) }),
        }),
    },
}
local nestedResult = Rules.Analyze("carrier", nestedRoot, {})
assert_true(nestedResult.blocked, "nested aura cancellation self-reapply was not blocked")
assert_equal(nestedResult.findings[1].classID, "carrier curse", "nested aura ID was resolved incorrectly")

local delegated = aura("Delegated", "cancel", {
    cancel = workflow({ effect("aura_run_workflow", { "delegated", "restore" }) }),
    restore = workflow({ effect("aura_apply", { "delegated", "=" }) }),
})
assert_true(Rules.Analyze("delegated", delegated, {}).blocked,
    "same-aura delegated cancellation workflow was not followed")

local dynamic = aura("Dynamic", "cancel", {
    cancel = workflow({ effect("aura_apply", { "${target}", "=" }) }),
})
local dynamicResult = Rules.Analyze("dynamic", dynamic, {})
assert_false(dynamicResult.blocked, "unresolved dynamic aura target was treated as proven self-reapply")
assert_equal(dynamicResult.metrics.dynamicApplications, 1, "dynamic application evidence was lost")

local stableA = Rules.Analyze("direct", direct, {})
local directReordered = aura("Direct", "cancel", {})
directReordered.SC.cancel = workflow({ effect("aura_apply", { "direct", "=" }) })
local stableB = Rules.Analyze("direct", directReordered, {})
assert_equal(stableA.fingerprint, stableB.fingerprint, "fingerprint depends on table insertion order")

print("item_guard_aura_rules_smoke: PASS")
