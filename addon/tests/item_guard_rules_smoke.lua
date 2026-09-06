-- Smoke test for RPBox_Addon/ItemGuardRules.lua.
-- Run from repository root:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_rules_smoke.lua

local function fail(message)
    error("[item-guard-rules-smoke] " .. tostring(message), 2)
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

local function item_root(entry, scripts)
    return {
        TY = "IT",
        BA = { NA = "test" },
        US = entry and { SC = entry } or nil,
        SC = scripts,
    }
end

local function analyze(root, context)
    return ns.ItemGuardRules.Analyze("root", root, context)
end

local function has_kind(result, kind)
    for _, finding in ipairs(result.findings) do
        if finding.kind == kind then return true, finding end
    end
    return false
end

ns = {}
local chunk, loadError = loadfile("addon/RPBox_Addon/ItemGuardRules.lua")
assert_true(chunk, loadError)
chunk("RPBox_Addon", ns)

local Rules = ns.ItemGuardRules
assert_true(type(Rules) == "table", "rules namespace was not registered")
assert_true(type(Rules.RULE_VERSION) == "number", "rule version is missing")
assert_true(type(Rules.Analyze) == "function", "Analyze interface is missing")

-- A reachable step SCC cannot be compiled safely, even without side effects.
local pureStepLoop = analyze(item_root("main", {
    main = workflow({
        ["1"] = list({}, "2"),
        ["2"] = list({}, "1"),
    }),
}))
assert_true(pureStepLoop.blocked, "compile cycle must be quarantined")
assert_equal(pureStepLoop.behaviorScore, 0, "pure step loop gained behavior score")
assert_equal(pureStepLoop.amplificationScore, 15, "step SCC amplification score changed")
assert_true(has_kind(pureStepLoop, "step_cycle"), "step SCC finding is missing")

-- Same-class workflow recursion is also structural evidence only.
local pureWorkflowRecursion = analyze(item_root("a", {
    a = workflow({ ["1"] = list({ effect("run_workflow", { "o", "b" }) }) }),
    b = workflow({ ["1"] = list({ effect("run_workflow", { "o", "a" }) }) }),
}))
assert_false(pureWorkflowRecursion.blocked, "pure workflow recursion must not be quarantined")
assert_equal(pureWorkflowRecursion.behaviorScore, 0, "pure recursion gained behavior score")
assert_equal(pureWorkflowRecursion.amplificationScore, 20, "workflow SCC amplification score changed")
assert_true(has_kind(pureWorkflowRecursion, "workflow_recursion"), "workflow SCC finding is missing")

-- Repeated item creation couples an actual side effect with the SCC.
local loopWithAdd = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_add", { "safe", 1 }) }, "1") }),
}))
assert_true(loopWithAdd.blocked, "step loop plus item_add must be quarantined")
assert_equal(loopWithAdd.amplificationScore, 15, "loop plus add lost structural score")
assert_true(loopWithAdd.behaviorScore >= 85, "loop plus add lost behavior score")
assert_true(loopWithAdd.score >= 100, "combined score did not reach threshold")
assert_true(has_kind(loopWithAdd, "item_add_amplified"), "amplified item_add finding is missing")

local recursionWithAdd = analyze(item_root("a", {
    a = workflow({ ["1"] = list({ effect("run_workflow", { "o", "b" }) }) }),
    b = workflow({ ["1"] = list({
        effect("item_add", { "safe", 1 }),
        effect("run_workflow", { "o", "a" }),
    }) }),
}))
assert_true(recursionWithAdd.blocked, "workflow recursion plus item_add must be quarantined")
assert_equal(recursionWithAdd.amplificationScore, 20, "recursive item_add lost workflow SCC score")
assert_true(recursionWithAdd.behaviorScore >= 85, "recursive item_add lost behavior score")

local ordinaryAdd = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_add", { "safe", 1 }) }) }),
}))
assert_false(ordinaryAdd.blocked, "ordinary single item_add must not be quarantined")
assert_true(ordinaryAdd.behaviorScore > 0 and ordinaryAdd.behaviorScore <= 10,
    "ordinary item_add score must remain in the 0-10 band")

local largeAdd = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_add", { "safe", "500" }) }) }),
}))
assert_false(largeAdd.blocked, "a literal 101-1000 add is scored, not an automatic hard block")
assert_true(largeAdd.behaviorScore >= 60, "101-1000 item_add did not receive volume score")
assert_true(has_kind(largeAdd, "item_add_large_count"), "large item_add finding is missing")

local excessiveAdd = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_add", { "safe", 1001 }) }) }),
}))
local hasExcessiveAdd, excessiveAddFinding = has_kind(excessiveAdd, "item_add_resource_exhaustion")
assert_true(excessiveAdd.blocked, ">1000 item_add must be a hard block")
assert_true(hasExcessiveAdd and excessiveAddFinding.hard, ">1000 item_add hard finding is missing")

local dynamicAdd = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_add", { "safe", "${object.qty}" }) }) }),
}))
assert_false(dynamicAdd.blocked, "dynamic item_add count must be deferred to runtime")
assert_true(has_kind(dynamicAdd, "item_add_dynamic_count"), "dynamic item_add report is missing")

local function loot_args(isDrop, slots)
    return { { "Loot", "inv_misc_bag_07", slots or {}, isDrop } }
end

local ordinaryDrop = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_loot", loot_args(true, {
        [1] = { classID = "safe", count = 2 },
    })) }) }),
}))
assert_false(ordinaryDrop.blocked, "ordinary one-shot ground drop must not be quarantined")
assert_equal(ordinaryDrop.behaviorScore, 30, "ordinary isDrop score changed")
assert_true(has_kind(ordinaryDrop, "item_loot_drop"), "isDrop finding is missing")

local repeatedDrop = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_loot", loot_args(true, {
        [1] = { classID = "safe", count = 1 },
    })) }, "1") }),
}))
assert_true(repeatedDrop.blocked, "looping ground drop must be quarantined")
assert_true(repeatedDrop.behaviorScore >= 120, "looping isDrop lost behavior score")
assert_true(has_kind(repeatedDrop, "item_loot_drop_amplified"), "looping isDrop finding is missing")

local tooManySlots = {}
for index = 1, 33 do tooManySlots[index] = { classID = "safe", count = 1 } end
local slotExhaustion = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_loot", loot_args(false, tooManySlots)) }) }),
}))
local hasSlotExhaustion, slotFinding = has_kind(slotExhaustion, "item_loot_slot_exhaustion")
assert_true(slotExhaustion.blocked, ">32 loot slots must be a hard block")
assert_true(hasSlotExhaustion and slotFinding.hard, "loot slot hard finding is missing")

local lootCountExhaustion = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("item_loot", loot_args(false, {
        [1] = { classID = "safe", count = 1001 },
    })) }) }),
}))
assert_true(lootCountExhaustion.blocked, ">1000 total loot count must be a hard block")
assert_true(has_kind(lootCountExhaustion, "item_loot_count_exhaustion"),
    "loot total-count hard finding is missing")

-- A disconnected editor remnant cannot execute and must not count as behavior.
local disconnectedStep = analyze(item_root("main", {
    main = workflow({
        ["1"] = list({ effect("text", { "safe" }) }),
        ["99"] = list({ effect("item_add", { "bad", 5000 }) }, "99"),
    }),
}))
assert_false(disconnectedStep.blocked, "disconnected malicious step was scored")
assert_equal(disconnectedStep.metrics.itemAdds, 0, "disconnected item_add entered reachable metrics")
assert_equal(disconnectedStep.metrics.itemAddsAll, 1, "all-effect scale metric lost disconnected item_add")

-- Once a real entry is known, unrelated workflows are not fallback entrypoints.
local unreachableWorkflow = analyze(item_root("safe", {
    safe = workflow({ ["1"] = list({ effect("text", { "safe" }) }) }),
    evil = workflow({ ["1"] = list({ effect("item_add", { "bad", 5000 }) }, "1") }),
}))
assert_false(unreachableWorkflow.blocked, "workflow unreachable from the real entry was scored")
assert_equal(unreachableWorkflow.metrics.workflowsAnalyzed, 1, "real-entry reachability analyzed extra workflow")
assert_equal(unreachableWorkflow.metrics.disconnectedWorkflows, 1, "unreachable workflow metric is missing")
assert_equal(unreachableWorkflow.metrics.entryConfidence, "explicit", "real entry lost explicit confidence")

-- Without a recognizable entry, each workflow step 1 is analyzed with marked confidence.
local fallbackEntry = analyze(item_root(nil, {
    a = workflow({ ["1"] = list({ effect("text", {}) }) }),
    b = workflow({ ["1"] = list({ effect("text", {}) }) }),
}))
assert_equal(fallbackEntry.metrics.workflowsAnalyzed, 2, "fallback did not analyze every workflow step 1")
assert_equal(fallbackEntry.metrics.entryConfidence, "fallback", "fallback confidence is not exposed")
assert_true(has_kind(fallbackEntry, "entrypoint_fallback"), "fallback entry finding is missing")

-- Cross-object calls are evidence, but are never invented as recursion edges.
local crossObject = analyze(item_root("main", {
    main = workflow({ ["1"] = list({
        effect("run_item_workflow", { "ch", "onUse", "2" }),
        effect("item_use", { "3" }),
    }) }),
}))
assert_false(crossObject.blocked, "unresolved cross-object edges must not fabricate a recursion block")
assert_equal(crossObject.metrics.externalEdges, 2, "cross-object edge metrics changed")
assert_equal(crossObject.metrics.unresolvedTargets, 2, "unresolved cross-object targets were not reported")
assert_true(has_kind(crossObject, "cross_object_target_unresolved"), "cross-object finding is missing")

-- Fingerprints are stable across insertion order and serialize loot tables by value.
local stableA = item_root("main", {})
stableA.SC.main = workflow({ ["1"] = list({ effect("item_loot", loot_args(true, {
    [2] = { classID = "b", count = 2 },
    [1] = { classID = "a", count = 1 },
})) }) })
stableA.SC.unused = workflow({ ["1"] = list({ effect("text", { "x" }) }) })

local stableB = item_root("main", {})
stableB.SC.unused = workflow({ ["1"] = list({ effect("text", { "x" }) }) })
local reversedSlots = {}
reversedSlots[1] = { count = 1, classID = "a" }
reversedSlots[2] = { count = 2, classID = "b" }
stableB.SC.main = workflow({ ["1"] = list({ effect("item_loot", loot_args(true, reversedSlots)) }) })
assert_equal(analyze(stableA).fingerprint, analyze(stableB).fingerprint,
    "fingerprint depends on table identity or insertion order")

local changedSlots = {
    [1] = { classID = "a", count = 2 },
    [2] = { classID = "b", count = 2 },
}
local stableChanged = item_root("main", {
    main = workflow({
        ["1"] = list({ effect("item_loot", loot_args(true, changedSlots)) }),
    }),
    unused = workflow({ ["1"] = list({ effect("text", { "x" }) }) }),
})
assert_true(analyze(stableA).fingerprint ~= analyze(stableChanged).fingerprint,
    "fingerprint ignored a relevant loot structure change")

-- Script Effect arguments remain opaque, even when they contain attack-looking text.
local opaqueArgs = setmetatable({}, {
    __index = function() error("Script Effect args were inspected") end,
})
local scriptOnly = analyze(item_root("main", {
    main = workflow({ ["1"] = list({ effect("script", opaqueArgs) }) }),
}))
assert_false(scriptOnly.blocked, "Lua payload text must not participate in the standard-workflow rules")
assert_equal(scriptOnly.behaviorScore, 0, "Lua payload gained standard behavior score")
assert_equal(scriptOnly.metrics.scriptEffects, 1, "opaque Script Effect was not counted")

-- Representative GnomeMap shape: large Lua/UI program, one ordinary item_add.
local gnomeScripts = {}
local remainingEffects = 15
local remainingSteps = 18
for workflowIndex = 1, 7 do
    local workflowID = "w" .. workflowIndex
    local stepCount = workflowIndex <= 4 and 3 or 2
    local steps = {}
    for stepIndex = 1, stepCount do
        local effects = {}
        if workflowIndex == 1 and stepIndex == 1 then
            effects[1] = effect("item_add", { "map_document", 1 })
        elseif (workflowIndex == 1 and stepIndex == 2) or (workflowIndex == 2 and stepIndex == 1) then
            effects[1] = effect("script", { "large UI source is intentionally opaque" })
        elseif remainingEffects > 0 then
            effects[1] = effect("text", { "UI stage" })
        end
        if effects[1] then remainingEffects = remainingEffects - 1 end
        remainingSteps = remainingSteps - 1
        steps[tostring(stepIndex)] = list(effects, stepIndex < stepCount and tostring(stepIndex + 1) or nil)
    end
    gnomeScripts[workflowID] = workflow(steps)
end
assert_equal(remainingSteps, 0, "representative fixture step count is wrong")
assert_equal(remainingEffects, 0, "representative fixture effect count is wrong")
local gnomeMap = analyze(item_root(nil, gnomeScripts))
assert_false(gnomeMap.blocked, "representative GnomeMap shape was falsely quarantined")
assert_equal(gnomeMap.metrics.workflowsTotal, 7, "representative workflow count changed")
assert_equal(gnomeMap.metrics.reachableSteps, 18, "representative reachable step count changed")
assert_equal(gnomeMap.metrics.effectsAnalyzed, 15, "representative effect count changed")
assert_equal(gnomeMap.metrics.scriptEffects, 2, "representative Script Effect count changed")
assert_equal(gnomeMap.metrics.itemAdds, 1, "representative item_add count changed")

print("item_guard_rules_smoke: PASS")
