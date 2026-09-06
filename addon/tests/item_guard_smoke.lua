-- Smoke test for RPBox_Addon/ItemGuard.lua.
-- Run from repository root:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_smoke.lua

local function fail(message)
    error("[item-guard-smoke] " .. tostring(message), 2)
end

local function assert_true(value, message)
    if not value then fail(message or "expected truthy value") end
end

local function assert_equal(actual, expected, message)
    if actual ~= expected then
        fail((message or "values differ") .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual))
    end
end

local Timer = { now = 0, queue = {} }
C_Timer = {}
function C_Timer.After(delay, callback)
    Timer.queue[#Timer.queue + 1] = { due = Timer.now + (tonumber(delay) or 0), callback = callback }
end
local function drain_timers(limit)
    limit = limit or 1000
    local count = 0
    while #Timer.queue > 0 do
        count = count + 1
        if count > limit then fail("timer queue did not settle") end
        local task = table.remove(Timer.queue, 1)
        Timer.now = math.max(Timer.now, task.due)
        task.callback()
    end
end
GetTime = function() return Timer.now end
time = function() return 1700000000 end

local printed = {}
local hostPrint = print
print = function(message) printed[#printed + 1] = tostring(message) end
CANCEL = "取消"

StaticPopupDialogs = {}
local LastPopup
local function popup_button()
    return {
        alpha = 1,
        enabled = true,
        SetAlpha = function(self, value) self.alpha = value end,
        Enable = function(self) self.enabled = true end,
        Disable = function(self) self.enabled = false end,
        IsEnabled = function(self) return self.enabled end,
    }
end
function StaticPopup_Show(name, arg1, arg2, data)
    local definition = StaticPopupDialogs[name]
    local dialog = {
        data = data,
        button1 = popup_button(),
        button2 = popup_button(),
        button3 = popup_button(),
    }
    LastPopup = { name = name, arg1 = arg1, arg2 = arg2, data = data, dialog = dialog, definition = definition }
    if definition and definition.OnShow then definition.OnShow(dialog, data) end
    return dialog
end

local callbacks = {}
local function register_callback(_, event, callback)
    callbacks[event] = callbacks[event] or {}
    callbacks[event][#callbacks[event] + 1] = callback
end

TRP3_Extended = {
    Events = {
        ON_SLOT_USE = "ON_SLOT_USE",
        ON_OBJECT_UPDATED = "ON_OBJECT_UPDATED",
        REFRESH_BAG = "REFRESH_BAG",
    },
}
function TRP3_Extended:TriggerEvent(event, ...)
    for _, callback in ipairs(callbacks[event] or {}) do
        callback(event, ...)
    end
end

local function effect(id, args)
    return { id = id, args = args or {} }
end

local function workflow(steps)
    return { ST = steps }
end

local function item(name, workflows)
    return {
        TY = "IT",
        MD = { V = 1, CB = "Tester" },
        BA = { NA = name, IC = "original_" .. name, US = true },
        US = { SC = "onUse" },
        SC = workflows or {},
    }
end

local function aura(name, cancelWorkflow, workflows)
    return {
        TY = "AU",
        MD = { V = 1, CB = "Tester" },
        BA = { NA = name, IC = "original_" .. name, CC = true },
        LI = cancelWorkflow and { OC = cancelWorkflow } or {},
        SC = workflows or {},
    }
end

local function document(name, pages)
    local data = { TY = "DO", MD = { V = 1, CB = "Tester" }, BA = { NA = name }, PA = {} }
    for index, text in ipairs(pages or {}) do data.PA[index] = { TX = text } end
    return data
end

local safe = item("Safe", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("text", { "ok" }) } },
        -- Disconnected editor leftovers are not compiled and must not be scanned
        -- as executable cycles or item stuffing.
        ["99"] = { t = "list", e = { effect("item_add", { "safe", "999" }) }, n = "99" },
    }),
})
local defaultBag = item("Default bag", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "bag" }) } } }),
})

local luaItem = item("LuaItem", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("script", {
            "local total=0; for i=1,10 do total=total+i end; setVar(args,'w','total',total)",
        }) } },
    }),
})

local untrustedLuaPolicy = item("UntrustedLuaPolicy", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("script", {
        "for i=1,20 do effect('sound_id_self',args,'SFX',42,false) end",
    }) } } }),
})
untrustedLuaPolicy.MD.CB = "Policy-SmokeRealm"
untrustedLuaPolicy.MD.SB = "Policy-SmokeRealm"
local trustedLuaPolicy = item("TrustedLuaPolicy", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("script", {
        "for i=1,20 do effect('sound_id_self',args,'SFX',42,false) end",
    }) } } }),
})
trustedLuaPolicy.MD.CB = "TrustedPublisher-SmokeRealm"
trustedLuaPolicy.MD.SB = "TrustedPublisher-SmokeRealm"

local hardLuaItem = item("HardLua", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("script", {
        "while true do local value=1 end",
    }) } } }),
})
hardLuaItem.MD.CB = "TrustedPublisher-SmokeRealm"
hardLuaItem.MD.SB = "TrustedPublisher-SmokeRealm"

local advisoryLuaItem = item("AdvisoryLua", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("script", {
        "args.object.vars.note='review'",
    }) } } }),
})

local macroEscapeItem = item("MacroEscape", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("secure_macro", {
        "/run TRP3_API.script.runLuaScriptEffect=function(c,a,s) a._G=_G end",
    }) } } }),
})
macroEscapeItem.MD.CB = "TrustedPublisher-SmokeRealm"
macroEscapeItem.MD.SB = "TrustedPublisher-SmokeRealm"

local cycle = item("Cycle", {
    onUse = workflow({
        ["1"] = { t = "list", e = {}, n = "1" },
    }),
})

local cycleAdd = item("CycleAdd", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("item_add", { "safe", "1" }) }, n = "1" },
    }),
})
local cycleAddChild = item("CycleAddChild", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "child" }) } } }),
})
local cycleAddGrandchild = item("CycleAddGrandchild", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "grandchild" }) } } }),
})
cycleAddChild.IN = { grandchild = cycleAddGrandchild }
cycleAdd.IN = { child = cycleAddChild }

local hugeAdd = item("HugeAdd", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("item_add", { "safe", "1001" }) } },
    }),
})

local blacklisted = item("Blacklisted", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
blacklisted.MD.CB = "工作人员二号-金色平原"

local userBlacklisted = item("UserBlacklisted", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
userBlacklisted.MD.SB = "ManualBad-SmokeRealm"

local recursive = item("Recursive", {
    A = workflow({
        ["1"] = { t = "list", e = { effect("run_workflow", { "o", "B" }) } },
    }),
    B = workflow({
        ["1"] = { t = "list", e = { effect("run_workflow", { "o", "A" }) } },
    }),
})

local recursiveAdd = item("RecursiveAdd", {
    A = workflow({
        ["1"] = {
            t = "list",
            e = {
                effect("item_add", { "safe", "1" }),
                effect("run_workflow", { "o", "B" }),
            },
        },
    }),
    B = workflow({
        ["1"] = { t = "list", e = { effect("run_workflow", { "o", "A" }) } },
    }),
})

local runtime = item("Runtime", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("text", { "runtime" }) } },
    }),
})

local runtimeLoot = item("RuntimeLoot", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})

local runtimeSlow = item("RuntimeSlow", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})

local runtimeSound = item("RuntimeSound", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})

local runtimeSoundReset = item("RuntimeSoundReset", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})

local runtimeVariable = item("RuntimeVariable", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})

local runtimeTemporaryVariable = item("RuntimeTemporaryVariable", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})

local runtimeLargeVariable = item("RuntimeLargeVariable", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})
local runtimeExistingVariable = item("RuntimeExistingVariable", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})
local runtimeLuaSafe = item("RuntimeLuaSafe", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})
local runtimeLuaLibrary = item("RuntimeLuaLibrary", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})
local runtimeLuaHook = item("RuntimeLuaHook", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})
local runtimeLuaVariable = item("RuntimeLuaVariable", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})
local runtimeLuaDepth = item("RuntimeLuaDepth", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})
local runtimeLuaRate = item("RuntimeLuaRate", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "runtime" }) } } }),
})

local runtimeRespawn = item("RuntimeRespawn", {
    onDestroy = workflow({
        ["1"] = { t = "list", e = { effect("item_add", { "${runtimeTarget}", "1" }) } },
    }),
})
runtimeRespawn.LI = { OD = "onDestroy" }

local staticSound = item("StaticSound", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("sound_id_self", { "SFX", 42, false }) }, n = "1" },
    }),
})

local staticVariable = item("StaticVariable", {
    onUse = workflow({
        ["1"] = {
            t = "list",
            e = { effect("var_object", { "o", "=", "entry-${counter}", "x" }) },
            n = "1",
        },
    }),
})

local carrierRisk = item("CarrierRisk", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("item_add", { "safe", "1" }) }, n = "1" },
    }),
})
local carrierRiskChild = item("CarrierRiskChild", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "child" }) } } }),
})
carrierRisk.IN = { child = carrierRiskChild }

local safeAuraCarrier = item("SafeAuraCarrier", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
local safeAura = aura("SafeAura", "cancel", {
    cancel = workflow({ ["1"] = { t = "list", e = { effect("text", { "gone" }) } } }),
})
safeAuraCarrier.IN = { aura = safeAura }

local staticAuraCarrier = item("StaticAuraCarrier", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
local staticAura = aura("StaticAura", "cancel", {
    cancel = workflow({
        ["1"] = { t = "list", e = { effect("aura_apply", { "static_aura aura", "=" }) } },
    }),
})
staticAuraCarrier.IN = { aura = staticAura }

local runtimeAuraCarrier = item("RuntimeAuraCarrier", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
local runtimeAura = aura("RuntimeAura", "cancel", {
    cancel = workflow({ ["1"] = { t = "list", e = { effect("script", { "opaque" }) } } }),
})
runtimeAuraCarrier.IN = { aura = runtimeAura }

local delayedAuraCarrier = item("DelayedAuraCarrier", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
local delayedAura = aura("DelayedAura", "cancel", {
    cancel = workflow({ ["1"] = { t = "list", e = { effect("script", { "opaque timer" }) } } }),
})
delayedAuraCarrier.IN = { aura = delayedAura }

local eventAuraCarrier = item("EventAuraCarrier", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
local eventAura = aura("EventAura", nil, {
    onEvent = workflow({
        ["1"] = { t = "list", e = { effect("item_add", { "safe", "1001" }) } },
    }),
})
eventAura.HA = { { EV = "TRP3_SIGNAL", SC = "onEvent" } }
eventAuraCarrier.IN = { aura = eventAura }

local variableAuraCarrier = item("VariableAuraCarrier", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
local variableAura = aura("VariableAura", nil, {})
variableAuraCarrier.IN = { aura = variableAura }

local safeDocumentCarrier = item("SafeDocumentCarrier", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("document_show", { "safe_document doc" }) } },
    }),
})
local safeDocument = document("SafeDocument", { string.rep("d", 128 * 1024) })
safeDocumentCarrier.IN = { doc = safeDocument }

local hugeDocumentCarrier = item("HugeDocumentCarrier", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("document_show", { "huge_document doc" }) } },
    }),
})
local hugeDocument = document("HugeDocument", { string.rep("x", (512 * 1024) + 1) })
hugeDocumentCarrier.IN = { doc = hugeDocument }

local expansionDocumentCarrier = item("ExpansionDocumentCarrier", {
    onUse = workflow({
        ["1"] = { t = "list", e = { effect("document_show", { "expansion_document doc" }) } },
    }),
})
local expansionDocument = document("ExpansionDocument", { "${payload}${payload}${payload}" })
expansionDocumentCarrier.IN = { doc = expansionDocument }

TRP3_Tools_DB = {
    safe = safe,
    default_bag = defaultBag,
    lua = luaItem,
    untrusted_lua_policy = untrustedLuaPolicy,
    trusted_lua_policy = trustedLuaPolicy,
    hard_lua = hardLuaItem,
    advisory_lua = advisoryLuaItem,
    macro_escape = macroEscapeItem,
    runtime = runtime,
    runtime_loot = runtimeLoot,
    runtime_slow = runtimeSlow,
    runtime_sound = runtimeSound,
    runtime_sound_reset = runtimeSoundReset,
    runtime_variable = runtimeVariable,
    runtime_temporary_variable = runtimeTemporaryVariable,
    runtime_large_variable = runtimeLargeVariable,
    runtime_existing_variable = runtimeExistingVariable,
    runtime_lua_safe = runtimeLuaSafe,
    runtime_lua_library = runtimeLuaLibrary,
    runtime_lua_hook = runtimeLuaHook,
    runtime_lua_variable = runtimeLuaVariable,
    runtime_lua_depth = runtimeLuaDepth,
    runtime_lua_rate = runtimeLuaRate,
    runtime_respawn = runtimeRespawn,
    safe_aura = safeAuraCarrier,
    static_aura = staticAuraCarrier,
    runtime_aura = runtimeAuraCarrier,
    delayed_aura = delayedAuraCarrier,
    event_aura = eventAuraCarrier,
    variable_aura = variableAuraCarrier,
    safe_document = safeDocumentCarrier,
    huge_document = hugeDocumentCarrier,
    expansion_document = expansionDocumentCarrier,
}
TRP3_Exchange_DB = {
    cycle = cycle,
    cycle_add = cycleAdd,
    huge = hugeAdd,
    recursive = recursive,
    recursive_add = recursiveAdd,
    blacklisted = blacklisted,
    user_blacklisted = userBlacklisted,
    static_sound = staticSound,
    static_variable = staticVariable,
    carrier_risk = carrierRisk,
    safe_aura = safeAuraCarrier,
    static_aura = staticAuraCarrier,
    runtime_aura = runtimeAuraCarrier,
    delayed_aura = delayedAuraCarrier,
    event_aura = eventAuraCarrier,
    variable_aura = variableAuraCarrier,
    safe_document = safeDocumentCarrier,
    huge_document = hugeDocumentCarrier,
    expansion_document = expansionDocumentCarrier,
}
TRP3_DB = {
    inner = {},
    global = {
        safe = safe,
        default_bag = defaultBag,
        lua = luaItem,
        untrusted_lua_policy = untrustedLuaPolicy,
        trusted_lua_policy = trustedLuaPolicy,
        hard_lua = hardLuaItem,
        advisory_lua = advisoryLuaItem,
        macro_escape = macroEscapeItem,
        runtime = runtime,
        runtime_loot = runtimeLoot,
        runtime_slow = runtimeSlow,
        runtime_sound = runtimeSound,
        runtime_sound_reset = runtimeSoundReset,
        runtime_variable = runtimeVariable,
        runtime_temporary_variable = runtimeTemporaryVariable,
        runtime_large_variable = runtimeLargeVariable,
        runtime_existing_variable = runtimeExistingVariable,
        runtime_lua_safe = runtimeLuaSafe,
        runtime_lua_library = runtimeLuaLibrary,
        runtime_lua_hook = runtimeLuaHook,
        runtime_lua_variable = runtimeLuaVariable,
        runtime_lua_depth = runtimeLuaDepth,
        runtime_lua_rate = runtimeLuaRate,
        runtime_respawn = runtimeRespawn,
        cycle = cycle,
        cycle_add = cycleAdd,
        huge = hugeAdd,
        recursive = recursive,
        recursive_add = recursiveAdd,
        blacklisted = blacklisted,
        user_blacklisted = userBlacklisted,
        static_sound = staticSound,
        static_variable = staticVariable,
        carrier_risk = carrierRisk,
        ["carrier_risk child"] = carrierRiskChild,
        safe_aura = safeAuraCarrier,
        ["safe_aura aura"] = safeAura,
        static_aura = staticAuraCarrier,
        ["static_aura aura"] = staticAura,
        runtime_aura = runtimeAuraCarrier,
        ["runtime_aura aura"] = runtimeAura,
        delayed_aura = delayedAuraCarrier,
        ["delayed_aura aura"] = delayedAura,
        event_aura = eventAuraCarrier,
        ["event_aura aura"] = eventAura,
        variable_aura = variableAuraCarrier,
        ["variable_aura aura"] = variableAura,
        safe_document = safeDocumentCarrier,
        ["safe_document doc"] = safeDocument,
        huge_document = hugeDocumentCarrier,
        ["huge_document doc"] = hugeDocument,
        expansion_document = expansionDocumentCarrier,
        ["expansion_document doc"] = expansionDocument,
    },
    types = { ITEM = "IT", AURA = "AU", DOCUMENT = "DO" },
}

local originalCalls = {
    execute = 0,
    effect = 0,
    setVar = 0,
    runLua = 0,
    add = 0,
    clearAll = 0,
    clearRoot = 0,
    auraApply = 0,
    auraCancel = 0,
    auraRemove = 0,
    auraSetVariable = 0,
    documentShow = 0,
    documentShowClass = 0,
    registerObject = 0,
}
local removeCalls = {}
local removeObjectCalls = {}
local playerInventory = { id = "player-inventory", content = {} }
local originalExecute = function(scriptID, _, _, fullID)
    originalCalls.execute = originalCalls.execute + 1
    if scriptID == "onDestroy" and fullID == "runtime_respawn" then
        TRP3_API.inventory.addItem(nil, "runtime_respawn", { count = 1 }, true)
    elseif scriptID == "cancel" and fullID == "runtime_aura aura" then
        TRP3_API.extended.auras.apply("runtime_aura aura", "=")
    elseif scriptID == "cancel" and fullID == "delayed_aura aura" then
        C_Timer.After(0.5, function()
            TRP3_API.extended.auras.apply("delayed_aura aura", "=")
        end)
    end
    return 7
end
local originalAdd = function()
    originalCalls.add = originalCalls.add + 1
    return 0
end
local originalSetVar = function()
    originalCalls.setVar = originalCalls.setVar + 1
end
local originalPlayEffect
local originalRunLua

TRP3_API = {
    RegisterCallback = register_callback,
    globals = { player_id = "Self-SmokeRealm" },
    extended = {
        ID_SEPARATOR = " ",
        getRootClassID = function(classID) return classID and classID:match("^[^ ]+") end,
        getClass = function(classID) return TRP3_DB.global[classID] end,
        auras = {},
        document = {},
    },
    script = {},
    inventory = {},
}

local function originalRegisterObject(objectFullID, object, count, registerTo)
    originalCalls.registerObject = originalCalls.registerObject + 1
    local target = registerTo or TRP3_DB.global
    target[objectFullID] = object
    for childID, child in pairs(type(object.IN) == "table" and object.IN or {}) do
        target[objectFullID .. " " .. tostring(childID)] = child
    end
    return (tonumber(count) or 0) + 1
end
TRP3_API.extended.registerObject = originalRegisterObject

local activeAuras = {}
local function originalAuraApply(auraID)
    originalCalls.auraApply = originalCalls.auraApply + 1
    activeAuras[auraID] = true
end
local function originalAuraRemove(auraID)
    originalCalls.auraRemove = originalCalls.auraRemove + 1
    local existed = activeAuras[auraID] == true
    activeAuras[auraID] = nil
    return existed
end
local function originalAuraCancel(auraID)
    originalCalls.auraCancel = originalCalls.auraCancel + 1
    if not activeAuras[auraID] then return false end
    local class = TRP3_API.extended.getClass(auraID)
    if class and class.LI and class.LI.OC then
        TRP3_API.script.executeClassScript(
            class.LI.OC,
            class.SC or {},
            { object = { id = auraID } },
            auraID
        )
    end
    activeAuras[auraID] = nil
    return true
end
local function originalAuraSetVariable()
    originalCalls.auraSetVariable = originalCalls.auraSetVariable + 1
end
TRP3_API.extended.auras.apply = originalAuraApply
TRP3_API.extended.auras.remove = originalAuraRemove
TRP3_API.extended.auras.cancel = originalAuraCancel
TRP3_API.extended.auras.setVariable = originalAuraSetVariable

local function originalDocumentShow()
    originalCalls.documentShow = originalCalls.documentShow + 1
    return true
end
local function originalDocumentShowClass()
    originalCalls.documentShowClass = originalCalls.documentShowClass + 1
    return true
end
TRP3_API.extended.document.showDocument = originalDocumentShow
TRP3_API.extended.document.showDocumentClass = originalDocumentShowClass
TRP3_API.extended.removeObject = function(rootID)
    removeObjectCalls[#removeObjectCalls + 1] = rootID
    TRP3_Tools_DB[rootID] = nil
    TRP3_Exchange_DB[rootID] = nil
    TRP3_DB.global[rootID] = nil
    TRP3_DB.global[rootID .. " child"] = nil
end

local function mockEffectBody(effectID, _, effectArgs, ...)
    originalCalls.effect = originalCalls.effect + 1
    local parameters = { ... }
    if effectID == "item_add" then
        return TRP3_API.inventory.addItem(nil, parameters[1], { count = tonumber(parameters[2]) or 1 }, true)
    elseif effectID == "var_object" then
        return TRP3_API.script.setVar(
            effectArgs,
            parameters[1],
            parameters[2],
            parameters[3],
            parameters[4]
        )
    end
    return 0
end

local mockEffects = {}
TRP3_API.script.getEffect = function(effectID)
    if not mockEffects[effectID] then
        mockEffects[effectID] = { method = function(_, parameters, effectArgs)
            return mockEffectBody(effectID, false, effectArgs, (unpack or table.unpack)(parameters))
        end }
    end
    return mockEffects[effectID]
end
TRP3_API.script.runWorkflow = function() end
TRP3_API.script.delayed = function(delay, callback) C_Timer.After(delay, callback) end
originalPlayEffect = function(effectID, _, effectArgs, ...)
    local info = TRP3_API.script.getEffect(effectID)
    return info.method(info, { ... }, effectArgs)
end

originalRunLua = function(code, effectArgs, secured)
    originalCalls.runLua = originalCalls.runLua + 1
    if code:find("RUNTIME_LIBRARY_TAMPER", 1, true) then
        string.lower = function() return "tampered" end
    elseif code:find("RUNTIME_HOOK_TAMPER", 1, true) then
        TRP3_API.script.playEffect = function() return "tampered" end
    elseif code:find("RAW_SETVAR_CRASH", 1, true) then
        TRP3_API.script.setVar(
            effectArgs,
            "o",
            "=",
            "payload",
            string.rep("z", (512 * 1024) + 1)
        )
    elseif code:find("DYNAMIC_NEST", 1, true) then
        TRP3_API.script.runLuaScriptEffect(code, effectArgs, secured)
    end
    return 9
end

TRP3_API.script.executeClassScript = originalExecute
TRP3_API.script.playEffect = originalPlayEffect
TRP3_API.script.setVar = originalSetVar
TRP3_API.script.runLuaScriptEffect = originalRunLua
TRP3_API.script.clearAllCompilations = function() originalCalls.clearAll = originalCalls.clearAll + 1 end
TRP3_API.script.clearRootCompilation = function() originalCalls.clearRoot = originalCalls.clearRoot + 1 end
TRP3_API.inventory.addItem = originalAdd
TRP3_API.inventory.getInventory = function() return playerInventory end
TRP3_API.inventory.removeSlotContent = function(container, slotID, slotInfo, manuallyDestroyed)
    removeCalls[#removeCalls + 1] = {
        container = container,
        slotID = slotID,
        slotInfo = slotInfo,
        manuallyDestroyed = manuallyDestroyed,
    }
    if container.content[slotID] == slotInfo then container.content[slotID] = nil end
end

RPBox_Config = { itemGuardEnabled = true }
cycle.BA.US = nil
cycle.BA.IC = "ui-engineering-90-remote-close-icon"
RPBox_ItemGuardDB = {
    version = 1,
    quarantined = {
        cycle = {
            rootID = "cycle",
            itemName = "Cycle",
            hadIcon = true,
            originalIcon = "original_Cycle",
            originalUsable = true,
            fingerprint = "legacy-static-recursion",
            source = "scan",
            reasons = { "旧版将纯递归判定为风险" },
        },
    },
    ignored = { cycle_add = "legacy-popup-ignore" },
    publisherWhitelist = {
        ["trustedpublisher-smokerealm"] = {
            identity = "TrustedPublisher-SmokeRealm",
            reason = "test trusted publisher",
        },
    },
    findings = {
        cycle = {
            rootID = "cycle",
            itemName = "Cycle",
            fingerprint = "legacy-static-recursion",
            source = "scan",
            reasons = { "旧版将纯递归判定为风险" },
        },
    },
}

TRP3_Security = { sender = {
    trusted_lua_policy = "TrustedPublisher-SmokeRealm",
    hard_lua = "TrustedPublisher-SmokeRealm",
    macro_escape = "TrustedPublisher-SmokeRealm",
    untrusted_lua_policy = "Policy-SmokeRealm",
} }
local namespace = {}
for _, path in ipairs({
    "addon/RPBox_Addon/ItemGuardStructure.lua",
    "addon/RPBox_Addon/ItemGuardRules.lua",
    "addon/RPBox_Addon/ItemGuardBlacklist.lua",
    "addon/RPBox_Addon/ItemGuardPublisherWhitelist.lua",
    "addon/RPBox_Addon/ItemGuardSoundRules.lua",
    "addon/RPBox_Addon/ItemGuardLifecycleRules.lua",
    "addon/RPBox_Addon/ItemGuardVariableRules.lua",
    "addon/RPBox_Addon/ItemGuardAuraRules.lua",
    "addon/RPBox_Addon/ItemGuardContentRules.lua",
    "addon/RPBox_Addon/ItemGuardLuaRules.lua",
    "addon/RPBox_Addon/ItemGuardLuaSandbox.lua",
    "addon/RPBox_Addon/ItemGuardLuaExpressions.lua",
    "addon/RPBox_Addon/ItemGuard.lua",
}) do
    local chunk = assert(loadfile(path))
    chunk("RPBox_Addon", namespace)
end

local Guard = namespace.ItemGuard
assert_true(Guard, "ItemGuard was not exported")
Guard:Initialize()
drain_timers()

assert_true(Guard:IsEnabled(), "guard should default to enabled")
assert_true(Guard._state.installed, "runtime hooks were not installed")
assert_true(not Guard:IsQuarantined("safe"), "safe workflow was quarantined")
assert_true(not Guard:IsQuarantined("lua"), "bounded local Lua was quarantined")
assert_true(Guard:IsQuarantined("untrusted_lua_policy"),
    "untrusted looped high-impact Lua effect was not quarantined")
assert_true(not Guard:IsQuarantined("trusted_lua_policy"),
    "system trusted publisher did not bypass policy-only Lua risk")
assert_true(Guard._state.scanCache.trusted_lua_policy.publisherTrust.identity == "trustedpublisher-smokerealm",
    "trusted publisher evidence is missing")
assert_true(Guard:IsQuarantined("hard_lua"), "trusted publisher bypassed a hard infinite-loop rule")
assert_true(Guard:IsQuarantined("macro_escape"), "trusted publisher bypassed a guard escape macro")
assert_true(not Guard:IsQuarantined("advisory_lua"), "advisory-only Lua finding was quarantined")
local advisoryEntry
for _, entry in ipairs(Guard:GetRiskEntries()) do
    if entry.rootID == "advisory_lua" then advisoryEntry = entry end
end
assert_true(advisoryEntry and advisoryEntry.status == "observed",
    "Lua advisory was not preserved in the protection ledger")
assert_true(TRP3_API.script.runLuaScriptEffect ~= originalRunLua,
    "runLuaScriptEffect hook was not installed")

local safeLuaCalls = originalCalls.runLua
assert_equal(TRP3_API.script.runLuaScriptEffect(
    "local value=1",
    { classID = "runtime_lua_safe", object = { id = "runtime_lua_safe" } },
    false
), 9, "safe runtime Lua did not execute")
assert_equal(originalCalls.runLua, safeLuaCalls + 1, "safe runtime Lua missed the original executor")

local infiniteLuaCalls = originalCalls.runLua
TRP3_API.script.runLuaScriptEffect(
    "while true do local value=1 end",
    { classID = "runtime_lua_safe", object = { id = "runtime_lua_safe" } },
    false
)
assert_true(Guard:IsQuarantined("runtime_lua_safe"), "runtime infinite Lua was not quarantined")
assert_equal(originalCalls.runLua, infiniteLuaCalls, "runtime infinite Lua reached the original executor")

local originalLower = string.lower
TRP3_API.script.runLuaScriptEffect(
    "RUNTIME_LIBRARY_TAMPER",
    { classID = "runtime_lua_library", object = { id = "runtime_lua_library" } },
    false
)
assert_equal(string.lower, originalLower, "shared string library was not restored after tampering")
assert_true(Guard:IsQuarantined("runtime_lua_library"), "runtime shared-library tampering was not quarantined")

local installedPlayEffect = TRP3_API.script.playEffect
TRP3_API.script.runLuaScriptEffect(
    "RUNTIME_HOOK_TAMPER",
    { classID = "runtime_lua_hook", object = { id = "runtime_lua_hook" } },
    false
)
assert_equal(TRP3_API.script.playEffect, installedPlayEffect, "guard hook was not restored after tampering")
assert_true(Guard:IsQuarantined("runtime_lua_hook"), "runtime guard-hook tampering was not quarantined")

local rawLuaSetVarBefore = originalCalls.setVar
TRP3_API.script.runLuaScriptEffect(
    "RAW_SETVAR_CRASH",
    { classID = "runtime_lua_variable", object = { id = "runtime_lua_variable" } },
    false
)
assert_equal(originalCalls.setVar, rawLuaSetVarBefore, "raw Lua crash-sized setVar reached storage")
assert_true(Guard:IsQuarantined("runtime_lua_variable"), "raw Lua crash-sized setVar was not quarantined")

TRP3_API.script.runLuaScriptEffect(
    "DYNAMIC_NEST",
    { classID = "runtime_lua_depth", object = { id = "runtime_lua_depth" } },
    false
)
assert_true(Guard:IsQuarantined("runtime_lua_depth"), "runtime nested Script depth was not limited")

for _ = 1, 41 do
    TRP3_API.script.runLuaScriptEffect(
        "local value=1",
        { classID = "runtime_lua_rate", object = { id = "runtime_lua_rate" } },
        false
    )
end
assert_true(Guard:IsQuarantined("runtime_lua_rate"), "runtime Lua call-rate limit did not quarantine")
assert_true(Guard:IsQuarantined("cycle"), "compile cycle was released")
assert_equal(cycle.BA.US, nil, "compile cycle remained usable")
assert_true(RPBox_ItemGuardDB.findings.cycle ~= nil, "compile invariant is missing from ledger")
assert_true(Guard:IsQuarantined("huge"), "excessive item_add was not quarantined")
assert_true(Guard._state.scanCache.huge.score >= 120, "excessive item_add did not receive behavior score")
assert_true(not Guard:IsQuarantined("recursive"), "pure workflow recursion was treated as malicious")
assert_equal(Guard._state.scanCache.recursive.score, 20, "pure workflow recursion score changed")
assert_true(Guard:IsQuarantined("cycle_add"), "step recursion with item_add was not quarantined")
assert_true(Guard._state.scanCache.cycle_add.score >= 100, "step recursion and item_add scores were not combined")
assert_true(Guard:IsQuarantined("recursive_add"), "recursive workflows with item_add were not quarantined")
assert_true(Guard._state.scanCache.recursive_add.score >= 100, "workflow recursion and item_add scores were not combined")
assert_true(RPBox_ItemGuardDB.findings.cycle_add.reasons[1]:find("编译", 1, true) ~= nil,
    "compile-time invariant was not prioritized")
assert_true(RPBox_ItemGuardDB.findings.recursive_add.reasons[1]:find("重复执行物品添加", 1, true) ~= nil,
    "workflow recursion was shown before its malicious item-add behavior")
assert_true(Guard:IsQuarantined("static_sound"), "sound rule module was not merged into the guard scan")
assert_true(Guard._state.scanCache.static_sound.moduleMetrics.sound ~= nil, "sound module metrics are missing")
assert_true(Guard:IsQuarantined("static_variable"), "variable rule module was not merged into the guard scan")
assert_true(Guard._state.scanCache.static_variable.moduleMetrics.variable ~= nil, "variable module metrics are missing")
assert_true(not Guard:IsQuarantined("runtime_respawn"), "dynamic destruction target was blocked before runtime confirmation")
assert_true(Guard._state.scanCache.runtime_respawn.moduleMetrics.lifecycle ~= nil, "lifecycle module metrics are missing")
assert_true(not Guard:IsQuarantined("safe_aura"), "ordinary cancellable aura was quarantined")
assert_true(Guard:IsQuarantined("static_aura"), "cancel-time aura self-reapplication was not quarantined")
assert_true(Guard._state.scanCache.static_aura.moduleMetrics.aura ~= nil, "aura module metrics are missing")
assert_true(Guard._state.scanCache.static_aura.moduleMetrics.aura.selfReapplications == 1,
    "aura self-reapplication metric is missing")
assert_true(Guard:IsQuarantined("event_aura"), "suspicious aura event-handler behavior was not quarantined")
assert_true(Guard._state.scanCache.event_aura.metrics.explicitEntryClasses >= 1,
    "aura event handler was not added to the executable entrypoint graph")
assert_true(not Guard:IsQuarantined("safe_document"), "ordinary document content was quarantined")
assert_true(not Guard:IsQuarantined("expansion_document"),
    "unexpanded variable document was quarantined statically")
assert_true(Guard:IsQuarantined("huge_document"), "crash-sized document content was not quarantined")
assert_equal(Guard._state.scanCache.huge_document.findings[1].kind, "invalid_structure",
    "oversize document did not stop at bounded preflight")
assert_true(TRP3_API.extended.auras.apply ~= originalAuraApply, "aura application hook was not installed")
assert_true(TRP3_API.extended.auras.cancel ~= originalAuraCancel, "aura cancellation hook was not installed")
assert_true(TRP3_API.extended.auras.setVariable ~= originalAuraSetVariable,
    "aura variable hook was not installed")
assert_true(TRP3_API.extended.document.showDocument ~= originalDocumentShow,
    "document display hook was not installed")
assert_true(TRP3_API.extended.document.showDocumentClass ~= originalDocumentShowClass,
    "document class display hook was not installed")
assert_true(TRP3_API.extended.registerObject ~= originalRegisterObject, "object registration hook was not installed")
assert_true(Guard:IsQuarantined("blacklisted"), "built-in source blacklist did not isolate matching author")
assert_true(Guard._state.scanCache.blacklisted.policyScore == 120, "blacklist policy score is missing")
assert_true(not Guard:IsQuarantined("user_blacklisted"), "user blacklist fixture started isolated")
assert_true(namespace.ItemGuardBlacklist.AddUser("ManualBad-SmokeRealm"))
drain_timers()
assert_true(Guard:IsQuarantined("user_blacklisted"), "adding user source blacklist did not trigger isolation scan")
assert_true(namespace.ItemGuardBlacklist.RemoveUser("ManualBad-SmokeRealm"))
drain_timers()
assert_true(not Guard:IsQuarantined("user_blacklisted"), "removing user source blacklist did not clear policy isolation")
assert_equal(userBlacklisted.BA.US, true, "removing blacklist did not restore usability")
assert_equal(cycleAdd.BA.US, nil, "quarantine did not remove usability")
assert_equal(cycleAdd.BA.IC, "ui-engineering-90-remote-close-icon", "quarantine icon mismatch")
assert_equal(cycleAddChild.BA.US, nil, "quarantine did not remove nested item usability")
assert_equal(cycleAddChild.BA.IC, "ui-engineering-90-remote-close-icon", "nested item quarantine icon mismatch")
assert_equal(cycleAddGrandchild.BA.US, nil, "quarantine did not remove deep nested item usability")
assert_equal(cycleAddGrandchild.BA.IC, "ui-engineering-90-remote-close-icon",
    "deep nested item quarantine icon mismatch")
assert_true(RPBox_ItemGuardDB.quarantined.cycle_add.visualStates["cycle_add child"] ~= nil,
    "nested item visual state was not persisted")
assert_true(RPBox_ItemGuardDB.quarantined.cycle_add.visualStates["cycle_add child grandchild"] ~= nil,
    "deep nested item visual state was not persisted")

local playerContainerFrame = { info = playerInventory }
local policySlotInfo = { id = "untrusted_lua_policy", count = 1 }
playerInventory.content["0"] = policySlotInfo
TRP3_Extended:TriggerEvent(
    "ON_SLOT_USE",
    { info = policySlotInfo, slotID = "0" },
    playerContainerFrame
)
assert_equal(LastPopup.name, "RPBOX_ITEM_GUARD_QUARANTINED", "policy item did not show quarantine popup")
assert_equal(LastPopup.definition.button3, "信任作者", "quarantine popup lacks trust-author action")
assert_true(LastPopup.dialog.button3.enabled, "trust-author action was disabled for a known publisher")
assert_equal(LastPopup.dialog.button3.alpha, 0.72, "trust-author action emphasis is wrong")
LastPopup.definition.OnAlt(LastPopup.dialog, LastPopup.data)
drain_timers()
assert_true(RPBox_ItemGuardDB.publisherWhitelist["policy-smokerealm"] ~= nil,
    "popup did not add the selected item's publisher")
assert_true(not Guard:IsQuarantined("untrusted_lua_policy"),
    "publisher trust did not release policy-only quarantine")

local cycleSlotInfo = { id = "cycle_add", count = 1 }
playerInventory.content["1"] = cycleSlotInfo
local cycleSlotButton = { info = cycleSlotInfo, slotID = "1" }
TRP3_Extended:TriggerEvent("ON_SLOT_USE", cycleSlotButton, playerContainerFrame)
assert_true(LastPopup, "right click did not show quarantine popup")
assert_equal(LastPopup.name, "RPBOX_ITEM_GUARD_QUARANTINED", "wrong popup key")
assert_equal(LastPopup.arg1, "CycleAdd", "popup item name mismatch")
assert_true(LastPopup.arg2:find("添加物品", 1, true), "popup omitted malicious behavior reason")
assert_true(LastPopup.arg2:find("1. ", 1, true) == 1, "popup reasons were not numbered")
assert_true(LastPopup.arg2:find("\n", 1, true) ~= nil, "popup reasons were not split across lines")
assert_true(LastPopup.definition.text:find("已扫描到的风险：", 1, true) ~= nil,
    "popup omitted the risk heading")
assert_true(not LastPopup.definition.text:find("循环与递归仅作为放大指标", 1, true),
    "popup retained the redundant generic explanation")
assert_equal(LastPopup.definition.button1, "移除道具", "popup removal action is missing")
assert_equal(LastPopup.definition.button2, "保持隔离", "popup keep-isolated action is missing")
assert_true(LastPopup.dialog.button1.enabled, "slot-backed popup disabled removal")
assert_true(not LastPopup.dialog.button3.enabled, "trust-author action was enabled without a usable publisher")
assert_equal(LastPopup.dialog.button3.alpha, 0.35, "unavailable trust-author action was not de-emphasized")

assert_true(not Guard:ReleaseQuarantine("cycle_add"), "compile invariant was released")
assert_true(not Guard:SetIgnored("cycle_add", true), "compile invariant was ignored")
assert_equal(cycleAddChild.BA.US, nil, "nested unsafe object was restored")

local hugeSlotInfo = { id = "huge", count = 2 }
playerInventory.content["2"] = hugeSlotInfo
TRP3_Extended:TriggerEvent(
    "ON_SLOT_USE",
    { info = hugeSlotInfo, slotID = "2" },
    playerContainerFrame
)
assert_equal(LastPopup.name, "RPBOX_ITEM_GUARD_QUARANTINED", "removal setup did not show quarantine popup")
LastPopup.definition.OnAccept(LastPopup.dialog, LastPopup.data)
drain_timers()
assert_equal(LastPopup.name, "RPBOX_ITEM_GUARD_REMOVE", "removal action did not open confirmation")
assert_true(LastPopup.arg2:find("根载体“HugeAdd”", 1, true) ~= nil,
    "top-level removal confirmation omitted its root object")
assert_true(not LastPopup.arg2:find("Default bag", 1, true),
    "physical inventory container leaked into carrier detection")
assert_equal(LastPopup.definition.button1, "确认移除", "confirmation uses the wrong accept label")
assert_true(LastPopup.definition.button3 == nil, "confirmation still exposes a removal-scope choice")
LastPopup.definition.OnAccept(LastPopup.dialog, LastPopup.data)
assert_true(playerInventory.content["2"] == nil, "current quarantined slot was not removed")
assert_equal(removeObjectCalls[#removeObjectCalls], "huge", "top-level removal did not remove its root object")
assert_true(RPBox_ItemGuardDB.findings.huge == nil and RPBox_ItemGuardDB.quarantined.huge == nil,
    "top-level removed object remained in the guard ledger")
assert_equal(removeCalls[#removeCalls].manuallyDestroyed, false,
    "current removal triggered the destruction workflow")

local internalRiskInfo = { id = "carrier_risk child", count = 1 }
local defaultBagInfo = { id = "default_bag", count = 1, content = { ["1"] = internalRiskInfo } }
playerInventory.content["3"] = defaultBagInfo
TRP3_Extended:TriggerEvent(
    "ON_SLOT_USE",
    { info = internalRiskInfo, slotID = "1" },
    { info = defaultBagInfo }
)
LastPopup.definition.OnAccept(LastPopup.dialog, LastPopup.data)
drain_timers()
assert_equal(LastPopup.name, "RPBOX_ITEM_GUARD_REMOVE", "internal-object removal did not open confirmation")
assert_true(LastPopup.arg2:find("根载体“CarrierRisk”", 1, true) ~= nil,
    "internal-object removal did not identify its root carrier")
assert_true(not LastPopup.arg2:find("Default bag", 1, true),
    "physical inventory container was mistaken for the root carrier")
assert_equal(LastPopup.definition.button1, "确认移除", "internal confirmation uses the wrong accept label")
assert_true(LastPopup.definition.button3 == nil, "internal confirmation still exposes a removal-scope choice")
LastPopup.definition.OnAccept(LastPopup.dialog, LastPopup.data)
assert_true(defaultBagInfo.content["1"] == nil, "internal item instance was not removed with its carrier")
assert_true(playerInventory.content["3"] == defaultBagInfo,
    "physical inventory container was removed instead of the root carrier")
assert_equal(removeObjectCalls[#removeObjectCalls], "carrier_risk", "carrier removal targeted the wrong root")
assert_true(TRP3_Exchange_DB.carrier_risk == nil and TRP3_DB.global.carrier_risk == nil,
    "carrier root remained in TRP3 object storage")
assert_true(RPBox_ItemGuardDB.findings.carrier_risk == nil
    and RPBox_ItemGuardDB.quarantined.carrier_risk == nil,
    "removed carrier remained in the guard ledger")
assert_equal(removeCalls[#removeCalls].slotInfo, internalRiskInfo,
    "carrier removal targeted a physical parent container")
assert_equal(removeCalls[#removeCalls].manuallyDestroyed, false,
    "carrier removal triggered the destruction workflow")

local beforeExecute = originalCalls.execute
local ret = TRP3_API.script.executeClassScript("onUse", cycleAdd.SC, { object = { id = "cycle_add" } }, "cycle_add")
assert_equal(ret, 0, "compile invariant reached original executor")
assert_equal(originalCalls.execute, beforeExecute, "blocked workflow executed")

local auraApplyBeforeStatic = originalCalls.auraApply
TRP3_API.extended.auras.apply("static_aura aura", "=")
assert_equal(originalCalls.auraApply, auraApplyBeforeStatic,
    "quarantined self-reapplying aura reached the original apply API")
assert_true(not activeAuras["static_aura aura"], "quarantined aura became active")

TRP3_API.extended.auras.apply("runtime_aura aura", "=")
assert_true(activeAuras["runtime_aura aura"], "safe aura did not reach the original apply API")
assert_true(TRP3_API.extended.auras.cancel("runtime_aura aura"), "runtime aura cancellation failed")
assert_true(Guard:IsQuarantined("runtime_aura"),
    "cancel-time runtime aura self-reapplication was not quarantined")
assert_true(not activeAuras["runtime_aura aura"], "runtime self-reapplying aura survived cancellation")
assert_equal(originalCalls.auraApply, auraApplyBeforeStatic + 1,
    "cancel-time self-reapplication reached the original aura apply API")

TRP3_API.extended.auras.apply("delayed_aura aura", "=")
assert_true(TRP3_API.extended.auras.cancel("delayed_aura aura"), "delayed aura cancellation failed")
assert_true(not activeAuras["delayed_aura aura"], "delayed aura was not removed by cancellation")
drain_timers()
assert_true(Guard:IsQuarantined("delayed_aura"),
    "delayed cancel-time aura self-reapplication escaped the cancellation watch")
assert_true(not activeAuras["delayed_aura aura"], "delayed self-reapplying aura became active")

local documentShowBefore = originalCalls.documentShow
assert_true(TRP3_API.extended.document.showDocument(
    "safe_document doc",
    { object = { id = "safe_document", vars = { name = "safe" } } }
), "safe document was blocked by the display hook")
assert_equal(originalCalls.documentShow, documentShowBefore + 1, "safe document did not reach the renderer")
TRP3_API.extended.document.showDocument("huge_document doc", { object = { id = "huge_document" } })
assert_equal(originalCalls.documentShow, documentShowBefore + 1,
    "crash-sized document reached the renderer")

local documentClassBefore = originalCalls.documentShowClass
TRP3_API.extended.document.showDocumentClass(
    document("PreviewBomb", { string.rep("p", (512 * 1024) + 1) }),
    nil,
    nil
)
assert_equal(originalCalls.documentShowClass, documentClassBefore,
    "crash-sized document preview reached the renderer")

local crashSizedValue = string.rep("v", (512 * 1024) + 1)
TRP3_API.extended.document.showDocumentClass(
    document("VariablePreviewBomb", { "safe text" }),
    nil,
    { object = { vars = { payload = crashSizedValue } } }
)
assert_equal(originalCalls.documentShowClass, documentClassBefore,
    "document preview with crash-sized variables reached the renderer")

TRP3_API.extended.document.showDocumentClass(
    document("ExpansionPreviewBomb", { "${payload}${payload}${payload}" }),
    nil,
    { object = { vars = { payload = string.rep("e", 200 * 1024) } } }
)
assert_equal(originalCalls.documentShowClass, documentClassBefore,
    "variable-expanded crash-sized document preview reached the renderer")

local documentEffectBefore = originalCalls.effect
TRP3_API.script.playEffect(
    "document_show",
    false,
    {
        classID = "expansion_document",
        object = {
            id = "expansion_document",
            vars = { payload = string.rep("e", 200 * 1024) },
        },
    },
    "expansion_document doc"
)
assert_true(Guard:IsQuarantined("expansion_document"),
    "variable-expanded crash-sized document effect was not quarantined")
assert_equal(originalCalls.effect, documentEffectBefore,
    "variable-expanded crash-sized document effect reached the renderer")

local auraVariableBefore = originalCalls.auraSetVariable
TRP3_API.extended.auras.setVariable("variable_aura aura", "=", "payload", crashSizedValue)
assert_true(Guard:IsQuarantined("variable_aura"), "crash-sized aura variable was not quarantined")
assert_equal(originalCalls.auraSetVariable, auraVariableBefore,
    "crash-sized aura variable reached the original setter")

local existingVariableExecuteBefore = originalCalls.execute
TRP3_API.script.executeClassScript(
    "onUse",
    runtimeExistingVariable.SC,
    { object = { id = "runtime_existing_variable", vars = { payload = crashSizedValue } } },
    "runtime_existing_variable"
)
assert_true(Guard:IsQuarantined("runtime_existing_variable"),
    "existing crash-sized object variable was not quarantined before execution")
assert_equal(originalCalls.execute, existingVariableExecuteBefore,
    "existing crash-sized object variable reached the executor")

local runtimeArgs = { classID = "runtime", object = { id = "runtime" } }
for _ = 1, 21 do
    TRP3_API.script.playEffect("item_add", false, runtimeArgs, "safe", "1")
end
assert_true(Guard:IsQuarantined("runtime"), "runtime item_add burst was not quarantined")
assert_equal(originalCalls.add, 20, "runtime breaker allowed too many writes")

local runtimeLootArgs = { classID = "runtime_loot", object = { id = "runtime_loot" } }
local dropInfo = { "Drop", "inv_misc_bag_07", { [1] = { classID = "safe", count = 10 } }, true }
for _ = 1, 21 do
    TRP3_API.script.playEffect("item_loot", false, runtimeLootArgs, dropInfo)
end
assert_true(Guard:IsQuarantined("runtime_loot"), "runtime ground-loot burst was not quarantined")

local runtimeSlowArgs = { classID = "runtime_slow", object = { id = "runtime_slow" } }
for _ = 1, 11 do
    TRP3_API.script.playEffect("item_add", false, runtimeSlowArgs, "safe", "100")
    Timer.now = Timer.now + 6
end
assert_true(Guard:IsQuarantined("runtime_slow"), "60-second cumulative write quota did not catch slow stuffing")

local runtimeSoundArgs = { classID = "runtime_sound", object = { id = "runtime_sound" } }
for _ = 1, 9 do
    TRP3_API.script.playEffect("sound_id_self", false, runtimeSoundArgs, "SFX", 42, false)
end
assert_true(not Guard:IsQuarantined("runtime_sound"), "first sound-rate breach quarantined instead of suppressing only sound")
for _ = 1, 9 do
    TRP3_API.script.playEffect("sound_id_self", false, runtimeSoundArgs, "SFX", 42, false)
end
assert_true(Guard:IsQuarantined("runtime_sound"), "repeated sound-rate breach did not quarantine")

local runtimeSoundResetArgs = {
    classID = "runtime_sound_reset",
    object = { id = "runtime_sound_reset" },
}
for _ = 1, 8 do
    TRP3_API.script.playEffect("sound_id_self", false, runtimeSoundResetArgs, "SFX", 42, false)
end
TRP3_API.script.playEffect("sound_id_stop", false, runtimeSoundResetArgs, "SFX", 42)
for _ = 1, 8 do
    TRP3_API.script.playEffect("sound_id_self", false, runtimeSoundResetArgs, "SFX", 42, false)
end
assert_true(not Guard:IsQuarantined("runtime_sound_reset"), "matching sound stop did not reset the family quota")

local runtimeVariableArgs = { classID = "runtime_variable", object = { id = "runtime_variable" } }
for _ = 1, 101 do
    TRP3_API.script.playEffect("var_object", false, runtimeVariableArgs, "o", "=", "state", "x")
end
assert_true(Guard:IsQuarantined("runtime_variable"), "persistent variable runtime quota did not quarantine")

local runtimeTemporaryArgs = {
    classID = "runtime_temporary_variable",
    object = { id = "runtime_temporary_variable" },
}
for _ = 1, 150 do
    TRP3_API.script.playEffect("var_object", false, runtimeTemporaryArgs, "w", "=", "state", "x")
end
assert_true(not Guard:IsQuarantined("runtime_temporary_variable"), "workflow-local variables entered persistent quotas")

local largeVariableArgs = {
    classID = "runtime_large_variable",
    object = { id = "runtime_large_variable" },
}
local setVarBeforeLarge = originalCalls.setVar
TRP3_API.script.playEffect(
    "var_object",
    false,
    largeVariableArgs,
    "o",
    "=",
    "payload",
    crashSizedValue
)
assert_true(Guard:IsQuarantined("runtime_large_variable"),
    "single crash-sized runtime variable was not quarantined")
assert_equal(originalCalls.setVar, setVarBeforeLarge,
    "single crash-sized runtime variable reached the original setter")

local addBeforeVariablePayload = originalCalls.add
local addVariableResult = TRP3_API.inventory.addItem(
    nil,
    "runtime_temporary_variable",
    { count = 1, vars = { payload = crashSizedValue } },
    true
)
assert_equal(addVariableResult, 1, "crash-sized received item variables were not rejected")
assert_equal(originalCalls.add, addBeforeVariablePayload,
    "crash-sized received item variables reached the original inventory writer")
assert_true(Guard:IsQuarantined("runtime_temporary_variable"),
    "item carrying crash-sized variables was not quarantined")

local addCallsBeforeRespawn = originalCalls.add
TRP3_API.script.executeClassScript(
    "onDestroy",
    runtimeRespawn.SC,
    { object = { id = "runtime_respawn" } },
    "runtime_respawn"
)
assert_true(Guard:IsQuarantined("runtime_respawn"), "destruction-time self-respawn was not quarantined")
assert_equal(originalCalls.add, addCallsBeforeRespawn, "destruction-time self-respawn reached the original addItem")

local receivedSafe = item("ReceivedSafe", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "received" }) } } }),
})
TRP3_Exchange_DB.received_safe = receivedSafe
TRP3_API.extended.registerObject("received_safe", receivedSafe, 0)
assert_equal(Guard:GetScanStatus("received_safe"), "unscanned",
    "newly registered object did not enter the unscanned state")
assert_equal(receivedSafe.BA.US, nil, "unscanned received object remained visually executable")
local pendingEntry
for _, entry in ipairs(Guard:GetRiskEntries()) do
    if entry.rootID == "received_safe" then pendingEntry = entry end
end
assert_true(pendingEntry and pendingEntry.status == "unscanned" and pendingEntry.pending,
    "unscanned object was not exposed in the protection ledger")

local executeBeforePending = originalCalls.execute
local pendingResult = TRP3_API.script.executeClassScript(
    "onUse",
    receivedSafe.SC,
    { object = { id = "received_safe" } },
    "received_safe"
)
assert_equal(pendingResult, 0, "first execution attempt was not blocked while object was unscanned")
assert_equal(originalCalls.execute, executeBeforePending, "unscanned object reached the original executor")
assert_equal(Guard:GetScanStatus("received_safe"), "trusted",
    "clean received object was not trusted automatically after scanning")
assert_equal(receivedSafe.BA.US, true, "clean received object was not restored after scanning")
assert_equal(
    TRP3_API.script.executeClassScript(
        "onUse",
        receivedSafe.SC,
        { object = { id = "received_safe" } },
        "received_safe"
    ),
    7,
    "trusted received object did not execute after automatic release"
)

local rapidFirst = item("RapidFirst", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "first" }) } } }),
})
TRP3_Exchange_DB.rapid_received = rapidFirst
TRP3_API.extended.registerObject("rapid_received", rapidFirst, 0)
local rapidSecond = item("RapidSecond", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "second" }) } } }),
})
TRP3_Exchange_DB.rapid_received = rapidSecond
TRP3_API.extended.registerObject("rapid_received", rapidSecond, 0)
assert_equal(rapidSecond.BA.US, nil, "replacement received before scan was not protected")
TRP3_API.script.executeClassScript(
    "onUse",
    rapidSecond.SC,
    { object = { id = "rapid_received" } },
    "rapid_received"
)
assert_equal(rapidSecond.BA.NA, "RapidSecond", "pending scan restored metadata from the replaced object")
assert_equal(rapidSecond.BA.IC, "original_RapidSecond",
    "pending scan restored the replaced object's icon onto the current object")
assert_equal(rapidSecond.BA.US, true,
    "pending scan restored the replaced object's usability onto the current object")
drain_timers()

local replacedRiskFirst = item("ReplacedRiskFirst", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("item_add", { "safe", "1001" }) } } }),
})
TRP3_Exchange_DB.replaced_risk = replacedRiskFirst
TRP3_API.extended.registerObject("replaced_risk", replacedRiskFirst, 0)
drain_timers()
assert_true(Guard:IsQuarantined("replaced_risk"), "first risky replacement fixture was not quarantined")
local replacedRiskSecond = item("ReplacedRiskSecond", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("item_add", { "safe", "1002" }) } } }),
})
TRP3_Exchange_DB.replaced_risk = replacedRiskSecond
TRP3_API.extended.registerObject("replaced_risk", replacedRiskSecond, 0)
drain_timers()
assert_true(Guard:IsQuarantined("replaced_risk"), "updated risky object escaped quarantine")
assert_true(not Guard:ReleaseQuarantine("replaced_risk"), "updated hard-risk object was released")
assert_equal(RPBox_ItemGuardDB.quarantined.replaced_risk.originalIcon, "original_ReplacedRiskSecond",
    "replacement visual backup retained stale icon")

local replacedCleanRisk = item("ReplacedCleanRisk", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("item_add", { "safe", "1001" }) } } }),
})
TRP3_Exchange_DB.replaced_clean = replacedCleanRisk
TRP3_API.extended.registerObject("replaced_clean", replacedCleanRisk, 0)
drain_timers()
assert_true(Guard:IsQuarantined("replaced_clean"), "clean-replacement fixture did not start quarantined")
local replacedCleanSafe = item("ReplacedCleanSafe", {
    onUse = workflow({ ["1"] = { t = "list", e = { effect("text", { "safe" }) } } }),
})
TRP3_Exchange_DB.replaced_clean = replacedCleanSafe
TRP3_API.extended.registerObject("replaced_clean", replacedCleanSafe, 0)
drain_timers()
assert_true(not Guard:IsQuarantined("replaced_clean"), "clean replacement retained stale quarantine")
assert_equal(replacedCleanSafe.BA.IC, "original_ReplacedCleanSafe",
    "clean replacement inherited the quarantined object's icon")
assert_equal(replacedCleanSafe.BA.US, true,
    "clean replacement inherited the quarantined object's usability")

Guard:SetEnabled(false, true)
assert_true(not Guard:IsEnabled(), "guard did not disable")
assert_equal(TRP3_API.script.executeClassScript, originalExecute, "execute hook was not restored")
assert_equal(TRP3_API.script.playEffect, originalPlayEffect, "effect hook was not restored")
assert_equal(TRP3_API.script.setVar, originalSetVar, "setVar hook was not restored")
assert_equal(TRP3_API.script.runLuaScriptEffect, originalRunLua, "runLuaScriptEffect hook was not restored")
assert_equal(TRP3_API.inventory.addItem, originalAdd, "addItem hook was not restored")
assert_equal(TRP3_API.extended.registerObject, originalRegisterObject, "registerObject hook was not restored")
assert_equal(TRP3_API.extended.auras.apply, originalAuraApply, "aura apply hook was not restored")
assert_equal(TRP3_API.extended.auras.cancel, originalAuraCancel, "aura cancel hook was not restored")
assert_equal(TRP3_API.extended.auras.setVariable, originalAuraSetVariable,
    "aura variable hook was not restored")
assert_equal(TRP3_API.extended.document.showDocument, originalDocumentShow,
    "document display hook was not restored")
assert_equal(TRP3_API.extended.document.showDocumentClass, originalDocumentShowClass,
    "document class display hook was not restored")
assert_equal(runtime.BA.US, true, "disabling did not restore quarantined usability")
assert_equal(runtime.BA.IC, "original_Runtime", "disabling did not restore quarantined icon")

Guard:SetEnabled(true, true)
drain_timers()
assert_true(Guard._state.installed, "guard did not reinstall")
assert_true(Guard:IsQuarantined("runtime"), "runtime quarantine record was lost on re-enable")
assert_equal(runtime.BA.US, nil, "re-enable did not reapply visual quarantine")

hostPrint("PASS item guard: temporary release, re-scan quarantine, explicit ignore ledger, runtime breaker, toggle")
