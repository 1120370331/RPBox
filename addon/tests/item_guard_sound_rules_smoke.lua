-- Smoke test for RPBox_Addon/ItemGuardSoundRules.lua.
-- Run from repository root:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_sound_rules_smoke.lua

local function fail(message)
    error("[item-guard-sound-rules-smoke] " .. tostring(message), 2)
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

local function root(workflows)
    return { TY = "IT", BA = { NA = "sound test" }, SC = workflows }
end

local function analyze(value)
    return ns.ItemGuardSoundRules.Analyze("root", value)
end

local function has_kind(result, kind)
    for _, finding in ipairs(result.findings) do
        if finding.kind == kind then return true, finding end
    end
    return false
end

ns = {}
local chunk, loadError = loadfile("addon/RPBox_Addon/ItemGuardSoundRules.lua")
assert_true(chunk, loadError)
chunk("RPBox_Addon", ns)

local Rules = ns.ItemGuardSoundRules
assert_true(type(Rules) == "table", "sound rules namespace was not registered")
assert_true(type(Rules.Analyze) == "function", "Analyze interface is missing")
assert_true(type(Rules.ClassifyEffect) == "function", "ClassifyEffect interface is missing")
assert_true(type(Rules.LIMITS) == "table", "runtime limit recommendations are missing")

local ordinary = analyze(root({
    main = workflow({ ["1"] = list({ effect("sound_id_self", { "SFX", 43569, false }) }) }),
}))
assert_false(ordinary.blocked, "one ordinary sound must not be blocked")
assert_true(ordinary.behaviorScore >= 0 and ordinary.behaviorScore <= 5,
    "one ordinary sound must stay in the 0-5 behavior band")
assert_equal(ordinary.metrics.starts, 1, "ordinary sound start was not counted")

local pureRecursion = analyze(root({
    a = workflow({ ["1"] = list({ effect("run_workflow", { "o", "b" }) }) }),
    b = workflow({ ["1"] = list({ effect("run_workflow", { "o", "a" }) }) }),
}))
assert_false(pureRecursion.blocked, "pure workflow recursion must not be blocked")
assert_equal(pureRecursion.behaviorScore, 0, "pure recursion gained sound behavior score")
assert_equal(pureRecursion.amplificationScore, 0, "pure recursion gained sound amplification score")

local loopWithoutStop = analyze(root({
    main = workflow({
        ["1"] = list({ effect("sound_id_self", { "SFX", 43569, false }) }, "1"),
    }),
}))
assert_true(loopWithoutStop.blocked, "looping sound without stop must be blocked")
assert_true(loopWithoutStop.behaviorScore >= 100, "uncontrolled sound loop lost behavior score")
assert_true(has_kind(loopWithoutStop, "sound_unstoppable_repeat"), "uncontrolled sound finding is missing")

local recursiveSound = analyze(root({
    a = workflow({ ["1"] = list({
        effect("sound_music_self", { "228575" }),
        effect("run_workflow", { "o", "b" }),
    }) }),
    b = workflow({ ["1"] = list({ effect("run_workflow", { "o", "a" }) }) }),
}))
assert_true(recursiveSound.blocked, "recursive music without stop must be blocked")
assert_true(has_kind(recursiveSound, "sound_continuous_without_handle_or_stop"),
    "continuous music handle/stop limitation was not explained")

local recursiveSoundWithStop = analyze(root({
    a = workflow({ ["1"] = list({
        effect("sound_music_self", { "228575" }),
        effect("run_workflow", { "o", "b" }),
    }) }),
    b = workflow({ ["1"] = list({
        effect("sound_music_stop", {}),
        effect("run_workflow", { "o", "a" }),
    }) }),
}))
assert_false(recursiveSoundWithStop.blocked, "recursive music with matching stop must not be blocked")
assert_equal(recursiveSoundWithStop.metrics.controlledRepeatedStarts, 1,
    "recursive music stop path was not recognized")

local loopWithStop = analyze(root({
    main = workflow({
        ["1"] = list({ effect("sound_id_self", { "SFX", 43569, false }) }, "2"),
        ["2"] = list({ effect("sound_id_stop", { "SFX", 43569, 0 }) }, "1"),
    }),
}))
assert_false(loopWithStop.blocked, "looping sound with matching stop must not be blocked")
assert_equal(loopWithStop.metrics.controlledRepeatedStarts, 1, "controlled sound loop was not recognized")
assert_true(has_kind(loopWithStop, "sound_repeat_with_stop"), "controlled repeat finding is missing")

local bypassableStop = analyze(root({
    main = workflow({
        ["1"] = { t = "branch", b = { { n = "2" }, { n = "3" } } },
        ["2"] = list({ effect("sound_id_stop", { "SFX", 43569, 0 }) }, "3"),
        ["3"] = list({ effect("sound_id_self", { "SFX", 43569, false }) }, "1"),
    }),
}))
assert_true(bypassableStop.blocked,
    "a stop on only one optional loop branch must not hide a stop-free replay cycle")

local ineffectiveStopAll = analyze(root({
    main = workflow({
        ["1"] = list({ effect("sound_id_self", { "SFX", 43569, false }) }, "2"),
        ["2"] = list({ effect("sound_id_stop", { "SFX", nil, 0 }) }, "1"),
    }),
}))
assert_true(ineffectiveStopAll.blocked, "TRP3 numeric-0 stop-all bug must not suppress loop detection")
assert_equal(ineffectiveStopAll.metrics.ineffectiveStops, 1, "ineffective stop-all was not reported")

local mismatchedStop = analyze(root({
    main = workflow({
        ["1"] = list({ effect("sound_id_self", { "SFX", 43569, false }) }, "2"),
        ["2"] = list({ effect("sound_id_stop", { "SFX", 99999, 0 }) }, "1"),
    }),
}))
assert_true(mismatchedStop.blocked, "a stop for another sound ID must not control the loop")

local selfMusic = Rules.ClassifyEffect("sound_music_self", { "228575" })
assert_equal(selfMusic.kind, "start", "self music classification kind changed")
assert_equal(selfMusic.family, "sound_music_self", "self music family changed")
assert_true(selfMusic.continuous, "music must be classified as continuous")
local localMusic = Rules.ClassifyEffect("sound_music_local", { "228575", 20 })
assert_equal(localMusic.family, "sound_music_local", "local music family changed")
assert_equal(localMusic.securedFamily, "sound_music_self", "local music secured fallback is missing")
local localSound = Rules.ClassifyEffect("sound_id_local", { "Ambience", 123, 20, true })
assert_equal(localSound.family, "sound_id_local:ambience", "local sound family/channel changed")
assert_equal(localSound.securedFamily, "sound_id_self:ambience", "local sound secured fallback is missing")
local localStop = Rules.ClassifyEffect("sound_id_local_stop", { "Ambience", 123, 0 })
assert_equal(localStop.kind, "stop", "local stop classification changed")
assert_equal(localStop.identifier, "123", "local stop identifier changed")

local disconnected = analyze(root({
    main = workflow({
        ["1"] = list({ effect("text", { "safe" }) }),
        ["99"] = list({ effect("sound_music_self", { "bad" }) }, "99"),
    }),
}))
assert_false(disconnected.blocked, "disconnected sound loop must not be analyzed")
assert_equal(disconnected.metrics.starts, 0, "disconnected sound start entered metrics")
assert_equal(disconnected.metrics.disconnectedSteps, 1, "disconnected step metric is missing")

local opaqueArgs = setmetatable({}, {
    __index = function() error("Lua Script Effect args were inspected") end,
    __pairs = function() error("Lua Script Effect args were iterated") end,
})
local scriptOnly = analyze(root({
    main = workflow({ ["1"] = list({ effect("script", opaqueArgs) }) }),
}))
assert_false(scriptOnly.blocked, "Lua payload must remain outside sound rules")
assert_equal(scriptOnly.metrics.scriptEffects, 1, "opaque Lua effect was not counted")

local stableA = root({})
stableA.SC.main = workflow({
    ["1"] = list({ effect("sound_id_self", { "SFX", 7, false }) }, "2"),
    ["2"] = list({ effect("sound_id_stop", { "SFX", 7, 0 }) }, "1"),
})
local stableB = root({})
stableB.SC.main = workflow({})
stableB.SC.main.ST["2"] = list({ effect("sound_id_stop", { "SFX", 7, 0 }) }, "1")
stableB.SC.main.ST["1"] = list({ effect("sound_id_self", { "SFX", 7, false }) }, "2")
assert_equal(analyze(stableA).fingerprint, analyze(stableB).fingerprint,
    "fingerprint depends on table identity or insertion order")

local changed = root({
    main = workflow({
        ["1"] = list({ effect("sound_id_self", { "SFX", 8, false }) }, "2"),
        ["2"] = list({ effect("sound_id_stop", { "SFX", 8, 0 }) }, "1"),
    }),
})
assert_true(analyze(stableA).fingerprint ~= analyze(changed).fingerprint,
    "fingerprint ignored a relevant sound identifier change")

print("item_guard_sound_rules_smoke: PASS")
