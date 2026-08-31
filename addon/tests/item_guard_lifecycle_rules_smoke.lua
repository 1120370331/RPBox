-- Smoke test for RPBox_Addon/ItemGuardLifecycleRules.lua.
-- Run from repository root:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_lifecycle_rules_smoke.lua

local function fail(message)
    error("[item-guard-lifecycle-rules-smoke] " .. tostring(message), 2)
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

local function item_root(destroyWorkflow, scripts)
    return {
        TY = "IT",
        BA = { NA = "test" },
        LI = destroyWorkflow and { OD = destroyWorkflow } or nil,
        SC = scripts or {},
    }
end

local function has_kind(result, kind)
    for _, finding in ipairs(result.findings) do
        if finding.kind == kind then return true, finding end
    end
    return false
end

ns = {}
local chunk, loadError = loadfile("addon/RPBox_Addon/ItemGuardLifecycleRules.lua")
assert_true(chunk, loadError)
chunk("RPBox_Addon", ns)

local Rules = ns.ItemGuardLifecycleRules
assert_true(type(Rules) == "table", "rules namespace was not registered")
assert_true(type(Rules.RULE_VERSION) == "number", "rule version is missing")
assert_true(type(Rules.Analyze) == "function", "Analyze interface is missing")
assert_true(type(Rules.GetDestructionWorkflowIDs) == "function", "destruction workflow interface is missing")
assert_true(type(Rules.IsDestructionExecution) == "function", "runtime lifecycle interface is missing")
assert_true(type(Rules.ClassifyRespawnEffect) == "function", "runtime effect interface is missing")

-- A destruction callback is lifecycle context, not proof of malicious behavior.
local ordinaryDestroy = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("text", { "goodbye" }) }) }),
})
local ordinaryResult = Rules.Analyze("root", ordinaryDestroy)
assert_false(ordinaryResult.blocked, "ordinary destruction callback must not be blocked")
assert_equal(ordinaryResult.behaviorScore, 0, "ordinary callback gained behavior score")
assert_equal(ordinaryResult.amplificationScore, 20, "lifecycle amplification changed")
assert_true(Rules.GetDestructionWorkflowIDs(ordinaryDestroy).destroy, "LI.OD was not exposed")
assert_true(Rules.IsDestructionExecution("root", ordinaryDestroy, "destroy"), "LI.OD runtime match failed")
assert_false(Rules.IsDestructionExecution("root", ordinaryDestroy, "other"), "unrelated workflow matched LI.OD")

-- Giving an unrelated reward on destruction remains a legitimate pattern.
local reward = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("item_add", { "reward", 1 }) }) }),
})
local rewardResult = Rules.Analyze("root", reward)
assert_false(rewardResult.blocked, "destruction reward must not be blocked")
assert_equal(rewardResult.metrics.rewardAdds, 1, "destruction reward was not classified")
assert_true(has_kind(rewardResult, "destruction_reward_add"), "reward report is missing")

-- Directly re-adding the current root makes stack destruction self-defeating.
local direct = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("item_add", { "root", 1 }) }) }),
})
local directResult = Rules.Analyze("root", direct)
assert_true(directResult.blocked, "direct self-respawn must be blocked")
assert_equal(directResult.behaviorScore, 85, "direct self-respawn behavior score changed")
assert_equal(directResult.amplificationScore, 20, "direct self-respawn lifecycle score changed")
assert_true(directResult.score >= 100, "direct self-respawn did not reach threshold")
assert_true(directResult.hasSideEffect, "direct self-respawn lost side-effect evidence")
assert_true(has_kind(directResult, "destruction_self_respawn"), "direct self-respawn finding is missing")

-- Related child IDs belong to the same root object and also recreate it.
local related = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("item_add", { "root child", 1 }) }) }),
})
assert_true(Rules.Analyze("root", related).blocked, "related-root respawn must be blocked")

-- Same-object calls are followed from the destruction entry to the eventual add.
local indirect = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("run_workflow", { "o", "again" }) }) }),
    again = workflow({ ["1"] = list({
        effect("run_workflow", { "o", "again" }),
        effect("item_add", { "root", 1 }),
    }) }),
})
local indirectResult = Rules.Analyze("root", indirect)
assert_true(indirectResult.blocked, "indirect recursive self-respawn must be blocked")
assert_equal(indirectResult.amplificationScore, 40, "recursive lifecycle amplification changed")
assert_true(indirectResult.metrics.recursiveWorkflows > 0, "recursive workflow metric is missing")

-- A pure recursive destruction workflow remains structural evidence only.
local pureRecursion = item_root("a", {
    a = workflow({ ["1"] = list({ effect("run_workflow", { "o", "b" }) }) }),
    b = workflow({ ["1"] = list({ effect("run_workflow", { "o", "a" }) }) }),
})
local pureRecursionResult = Rules.Analyze("root", pureRecursion)
assert_false(pureRecursionResult.blocked, "pure lifecycle recursion must not be blocked")
assert_equal(pureRecursionResult.behaviorScore, 0, "pure lifecycle recursion gained behavior score")
assert_equal(pureRecursionResult.amplificationScore, 40, "pure lifecycle recursion score changed")

-- Dynamic targets and counts are deferred to the runtime hook.
local dynamic = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("item_add", { "${target}", "${count}" }) }) }),
})
local dynamicResult = Rules.Analyze("root", dynamic)
assert_false(dynamicResult.blocked, "dynamic target must not be statically blocked")
assert_equal(dynamicResult.metrics.dynamicRespawns, 1, "dynamic target report is missing")
assert_true(has_kind(dynamicResult, "destruction_respawn_dynamic"), "dynamic finding is missing")

-- Runtime classification sees the resolved concrete target and can block it.
local runtimeClass = Rules.ClassifyRespawnEffect("root", direct, "item_add", { "root", 2 })
assert_true(runtimeClass and runtimeClass.selfRespawn, "runtime self-respawn was not classified")
assert_false(runtimeClass.dynamic, "resolved runtime add remained dynamic")
assert_equal(runtimeClass.count, 2, "runtime count changed")

-- Disconnected editor remnants do not execute and must not be scored.
local disconnected = item_root("destroy", {
    destroy = workflow({
        ["1"] = list({ effect("text", {}) }),
        ["99"] = list({ effect("item_add", { "root", 1 }) }, "99"),
    }),
})
local disconnectedResult = Rules.Analyze("root", disconnected)
assert_false(disconnectedResult.blocked, "disconnected self-add was scored")
assert_equal(disconnectedResult.metrics.itemAdds, 0, "disconnected self-add entered metrics")

-- A self-add outside LI.OD is a different risk class and is not labeled as
-- impossible-to-destroy behavior here.
local nonDestroy = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("text", {}) }) }),
    use = workflow({ ["1"] = list({ effect("item_add", { "root", 1 }) }) }),
})
local nonDestroyResult = Rules.Analyze("root", nonDestroy)
assert_false(nonDestroyResult.blocked, "non-destruction self-add entered lifecycle policy")
assert_equal(nonDestroyResult.metrics.itemAdds, 0, "unreachable non-destruction add was analyzed")

-- Script Effect arguments are opaque even if indexing them would throw.
local opaqueArgs = setmetatable({}, {
    __index = function() error("Script Effect args were inspected") end,
})
local scriptOnly = item_root("destroy", {
    destroy = workflow({ ["1"] = list({ effect("script", opaqueArgs) }) }),
})
local scriptResult = Rules.Analyze("root", scriptOnly)
assert_false(scriptResult.blocked, "opaque Lua payload influenced lifecycle policy")
assert_equal(scriptResult.metrics.scriptEffects, 1, "opaque Lua effect was not counted")

-- A parent workflow target can be proven for a nested related class.
local nested = item_root(nil, {
    restore = workflow({ ["1"] = list({ effect("item_add", { "root child", 1 }) }) }),
})
nested.IN = {
    child = {
        TY = "IT",
        LI = { OD = "destroy" },
        SC = {
            destroy = workflow({ ["1"] = list({ effect("run_item_workflow", { "p", "restore" }) }) }),
        },
    },
}
assert_true(Rules.Analyze("root", nested).blocked, "parent workflow self-respawn path was not followed")

-- Fingerprints are deterministic and change only with relevant lifecycle data.
local stableA = item_root("destroy", {})
stableA.SC.destroy = workflow({ ["1"] = list({ effect("item_add", { "reward", 1 }) }) })
stableA.SC.unused = workflow({ ["1"] = list({ effect("text", { "a" }) }) })
local stableB = item_root("destroy", {})
stableB.SC.unused = workflow({ ["1"] = list({ effect("text", { "b" }) }) })
stableB.SC.destroy = workflow({ ["1"] = list({ effect("item_add", { "reward", 1 }) }) })
assert_equal(Rules.Analyze("root", stableA).fingerprint, Rules.Analyze("root", stableB).fingerprint,
    "fingerprint depends on insertion order or irrelevant standard-effect args")
stableB.SC.destroy.ST["1"].e[1].args[1] = "root"
assert_true(Rules.Analyze("root", stableA).fingerprint ~= Rules.Analyze("root", stableB).fingerprint,
    "fingerprint ignored a relevant respawn target")

print("item_guard_lifecycle_rules_smoke: PASS")
