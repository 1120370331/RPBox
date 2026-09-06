-- Real TRP3 compiler/Lua runner; only WoW host effects/storage are substituted.
-- Optional first argument: installed TRP3 ScriptGeneration.lua for compatibility.
local hostPrint = print
local nativeTrusted = {}
local runtimeErrors = {}
local function check(value, message) if not value then error(message, 2) end; return value end
if not loadstring then loadstring = load end
if not unpack then unpack = table.unpack end
if not setfenv then
    function setfenv(fn, environment)
        local index = 1
        while true do
            local name = debug.getupvalue(fn, index)
            if not name then break end
            if name == "_ENV" then
                debug.upvaluejoin(fn, index, function() return environment end, 1)
                break
            end
            index = index + 1
        end
        return fn
    end
end
function wipe(value) for key in pairs(value) do value[key] = nil end end
function strsplit(_, value) return value:match("^[^ ]+") end
local now, timers = 0, {}
GetTime = function() return now end
time = function() return 1700000000 + now end
date = os.date
C_Timer = { After = function(delay, callback) timers[#timers + 1] = { now + delay, callback } end }
local function drain()
    local steps = 0
    while #timers > 0 do
        steps = steps + 1; check(steps < 1000, "timer queue did not settle")
        table.sort(timers, function(a, b) return a[1] < b[1] end)
        local timer = table.remove(timers, 1)
        now = timer[1]; timer[2]()
    end
end
StaticPopupDialogs = {}
StaticPopup_Show = function() end
CANCEL = "Cancel"
print = function() end
TRP3_DB = { global = {}, my = {}, inner = {}, types = { ITEM = "IT", AURA = "AU", DOCUMENT = "DO" } }
TRP3_Tools_DB, TRP3_Exchange_DB = {}, {}
TRP3_Security = { sender = {} }
TRP3_Extended = { Events = {}, TriggerEvent = function() end }
TRP3_API = {
    globals = { empty = {}, player_id = "Self-Realm" }, register = {}, loc = { SEC_SCRIPT_ERROR = "%s", SEC_MISSING_SCRIPT = "%s" },
    Log = function() end, RegisterCallback = function() end,
    utils = { table = { copy = function(to, from) for k, v in pairs(from) do to[k] = v end end },
        str = {}, message = { displayMessage = function(message) runtimeErrors[#runtimeErrors + 1] = message end } },
    security = { SECURITY_LEVEL = { HIGH = 1 }, resolveEffectSecurity = function(id) return nativeTrusted[id] == true end },
    extended = { ID_SEPARATOR = " ", getRootClassID = function(id) return id:match("^[^ ]+") end,
        getClass = function(id) return TRP3_DB.global[id] end,
        registerObject = function(id, object) TRP3_DB.global[id] = object; return 1 end,
        auras = { apply = function() end, cancel = function() return true end, remove = function() end },
        document = { showDocument = function() end, showDocumentClass = function() end },
    }, inventory = {}, quest = { getActiveCampaignLog = function() return nil end },
}
local reference = arg and arg[1] or "refs/Total-RP-3-Extended/totalRP3_Extended/Script/ScriptGeneration.lua"
check(loadfile(reference))()
local added, sounded, macros = 0, 0, 0
TRP3_API.inventory.addItem = function() added = added + 1; return 0 end
local effects = {
    secure_macro = { method = function() macros = macros + 1 end },
    script = { method = function(_, parameters, args)
        TRP3_API.script.runLuaScriptEffect(parameters[1], args, false)
    end },
    item_add = { method = function(_, parameters)
        return TRP3_API.inventory.addItem(nil, parameters[1], { count = tonumber(parameters[2]) or 1 })
    end },
    sound_id_self = { method = function() sounded = sounded + 1 end },
    run_workflow = { method = function(_, parameters, args)
        TRP3_API.script.runWorkflow(args, parameters[1], parameters[2])
    end },
}
TRP3_API.script.getEffect = function(id) return effects[id] end
local numericOperand = setmetatable({ env = { eventValue = "TRP3_API.script.eventVarCheckN" } }, {
    __index = { numeric = true, CodeReplacement = function(_, parameters)
        return "eventValue(args, " .. tostring(parameters[1] or 1) .. ")"
    end },
})
TRP3_API.script.getOperand = function(id)
    if id == "check_event_var_n" then return numericOperand end
end
local function root(id, code)
    local value = { TY = "IT", BA = { NA = id, IC = "test", US = true }, MD = {}, US = { SC = "use" },
        SC = { use = { ST = { ["1"] = { t = "list", e = code and { { id = "script", args = { code } } } or {} } } } } }
    TRP3_DB.global[id] = value
    return value
end
local ns = {}
for _, name in ipairs({ "Structure", "Rules", "Blacklist", "PublisherWhitelist", "SoundRules",
    "LifecycleRules", "VariableRules", "AuraRules", "ContentRules", "LuaRules", "LuaExpressions", "LuaSandbox", "" }) do
    check(loadfile("addon/RPBox_Addon/ItemGuard" .. name .. ".lua"))("RPBox_Addon", ns)
end
local guard = ns.ItemGuard
RPBox_Config = { itemGuardEnabled = true }
guard:Initialize(); drain()
local function scan(id, value)
    guard:MarkUnscanned(id, "test"); guard:ScanAndApply(id); return value
end
local function execute(id, object)
    object = object or { id = id, vars = {} }
    TRP3_API.script.executeClassScript("use", TRP3_DB.global[id].SC, { object = object }, id)
    return object
end

local safe = scan("safe", root("safe", "local f=string.format; setVar(args,'o','value',f('%s','ok'))"))
check(not guard:IsQuarantined("safe"), "normal local library alias was blocked")
local object = execute("safe")
check(object.vars.value == "ok", "real sandbox setVar did not update authoritative storage")

local ui = root("trusted_ui", "local G=args._G; args.object.vars.ui=G.RPBOX_NATIVE_TEST; local n=math.ceil(1.2); n=3")
ui.SC.use.ST["1"].e = {
    { id = "secure_macro", args = { "/run if hTAsr==nil then hTAsr=TRP3_API.script.runLuaScriptEffect;TRP3_API.script.runLuaScriptEffect=function(c,a,s) a._G=_G;return hTAsr(c,a,s);end;end" } },
    ui.SC.use.ST["1"].e[1],
}
nativeTrusted.trusted_ui = true
scan("trusted_ui", ui)
check(not guard:IsQuarantined("trusted_ui"), "TRP3-trusted UI application was quarantined")
RPBOX_NATIVE_TEST = 17
local executor = TRP3_API.script.runLuaScriptEffect
local uiObject = execute("trusted_ui")
check(uiObject.vars.ui == 17, "trusted UI lost global API access or its real variable context")
check(macros == 0 and TRP3_API.script.runLuaScriptEffect == executor, "legacy bootstrap replaced the global executor")
local large = root("large_library")
for i=1,330 do large.SC['library_'..i] = { ST = { ['1'] = { t='list', e={} } } } end
scan("large_library",large)
check(not guard:IsQuarantined("large_library"), "large normal workflow library was blocked by count alone")
TRP3_API.script.executeClassScript("optional_missing", large.SC, { object = { id = "large_library" } }, "large_library")
check(not guard:IsQuarantined("large_library"), "missing optional workflow quarantined a normal library")
nativeTrusted.authoring_error = true
scan("authoring_error", root("authoring_error", "local optional=nil; optional()"))
execute("authoring_error")
check(not guard:IsQuarantined("authoring_error"), "ordinary trusted Lua error was labeled malicious")
nativeTrusted.trusted_loop = true
scan("trusted_loop", root("trusted_loop", "local active=true; while active do end"))
execute("trusted_loop")
check(guard:IsQuarantined("trusted_loop"), "TRP3 trust bypassed the synchronous loop budget")

local condition = root("condition")
condition.SC.use.ST["1"].e = { { id = "item_add", args = { "safe", 1 },
    cond = { { { i = "check_event_var_n", a = { 1 } }, ">", { v = 0 } } } } }
scan("condition", condition)
local conditionBefore = added
TRP3_API.script.executeClassScript("use", condition.SC, { object = { id = "condition" }, event = { 2 } }, "condition")
check(added == conditionBefore + 1, "numeric operand inheritance or condition compilation changed")
condition.SC.use.ST["1"].e[1].cond[1][1].a[1] = "1) or _G"
scan("condition", condition)
TRP3_API.script.executeClassScript("use", condition.SC, { object = { id = "condition" } }, "condition")
check(guard:IsQuarantined("condition") and added == conditionBefore + 1, "condition operand injection escaped validation")

local loop = scan("loop", root("loop", "local yes=true; while yes do end"))
check(not guard:IsQuarantined("loop"), "dynamic loop should reach execution budget")
execute("loop")
check(guard:IsQuarantined("loop"), "dynamic infinite loop escaped execution budget")
check(not guard:ReleaseQuarantine("loop"), "runtime invariant was released")
check(not guard:SetIgnored("loop", true), "runtime invariant was ignored")

scan("strings", root("strings", "local n=1048576; return ('abcd'):rep(n)"))
execute("strings")
check(guard:IsQuarantined("strings"), "string method syntax escaped allocation budget")

local cycle = root("cycle")
cycle.SC.use.ST["1"].n = "1"
scan("cycle", cycle)
check(guard:IsQuarantined("cycle"), "compile cycle was trusted")
execute("cycle")

local sound = root("sound", "local play=effect; for i=1,30 do play('sound_id_self',args,'SFX',42,false) end")
scan("sound", sound); execute("sound")
check(sounded < 30, "Lua local effect dispatcher escaped sound quota")

local recursive = root("recursive")
recursive.SC.use.ST["1"].e = { { id = "run_workflow", args = { "o", "use" } } }
scan("recursive", recursive); execute("recursive")
check(guard:IsQuarantined("recursive"), "real local workflow recursion escaped budget: " .. table.concat(runtimeErrors, ";"))

local failed = root("failed", "local value=1")
local analyze = ns.ItemGuardLuaRules.Analyze
ns.ItemGuardLuaRules.Analyze = function() error("fixture failure") end
scan("failed", failed)
check(guard:GetScanStatus("failed") == "scan_failed", "analysis failure was trusted")
ns.ItemGuardLuaRules.Analyze = analyze
guard:MarkUnscanned("failed", "retry"); guard:ScanAndApply("failed")
check(guard:GetScanStatus("failed") == "trusted", "successful retry retained failed-scan isolation")

local exception = scan("exception", root("exception", "effect(effectName,args)"))
check(guard:IsQuarantined("exception"), "policy fixture was not quarantined")
check(guard:ReleaseQuarantine("exception"), "policy-only temporary release was refused")
guard:ScanAndApply("exception")
check(guard:IsQuarantined("exception"), "temporary release survived a new scan")
check(guard:SetIgnored("exception", true), "policy-only exception was refused")
local exceptionAdds = added
for i = 1, 30 do TRP3_API.script.playEffect("item_add", false, { classID = "exception" }, "safe", 1) end
check(added - exceptionAdds == 20, "ignored object bypassed runtime quota")
check(guard:IsQuarantined("exception"), "runtime breaker did not override a policy exception")
for _, entry in ipairs(guard:GetRiskEntries()) do
    if entry.rootID == "exception" then check(entry.status == "quarantined", "runtime quarantine displayed as ignored") end
end

scan("concat", root("concat", "local text='abcd'; for i=1,30 do text=text..text end; return text"))
execute("concat")
check(guard:IsQuarantined("concat"), "exponential concatenation escaped allocation budget")

local before = added
TRP3_API.script.executeClassScript("use", recursive.SC, { classID = "safe" }, "recursive")
check(added == before and guard:IsQuarantined("recursive"), "target identity bypassed quarantine")

local pending = root("pending")
pending.SC.use.ST = { ["1"] = { t = "delay", d = 4, n = "2" },
    ["2"] = { t = "list", e = { { id = "item_add", args = { "safe", "1" } } } } }
scan("pending", pending); execute("pending")
guard:SetIsolation("pending", true)
before = added; drain()
check(added == before, "quarantine failed to revoke delayed work")

local malformed = root("not_registered")
TRP3_DB.global.not_registered = nil
malformed.IN = { self = malformed }
local registered = pcall(TRP3_API.extended.registerObject, "not_registered", malformed, 0)
check(not registered and not TRP3_DB.global.not_registered, "invalid data reached registrar")
for index=1,300 do
    local unrelated = root("unrelated_"..index)
    unrelated.MD.CB = "Other-Realm"
end
local affected = root("policy_affected")
affected.MD.CB = "Policy-Realm"
ns.ItemGuardBlacklist.AddUser("Policy-Realm"); drain()
check(guard:IsQuarantined("policy_affected"), "policy fixture did not enter quarantine")
local scans, refreshes = {}, 0
local originalScan = guard.ScanRoot
guard.ScanRoot = function(self,id,...)
    scans[id] = (scans[id] or 0)+1
    return originalScan(self,id,...)
end
guard:RegisterOnChanged(function() refreshes=refreshes+1; guard:GetRiskEntries() end)
ns.ItemGuardBlacklist.RemoveUser("Policy-Realm"); drain()
check(not guard:IsQuarantined("policy_affected"), "removing a source policy did not release its clean object")
check(scans.policy_affected == 1, "affected object was not scanned exactly once")
for id in pairs(scans) do check(id == "policy_affected", "blacklist removal rescanned unrelated objects") end
check(refreshes == 1, "one policy edit caused repeated synchronous GUI refreshes")
guard.ScanRoot = originalScan
for index=1,35 do
    local id = "cached_"..index
    local cached = root(id); cached.metadataNote = string.rep("x",300000)
    guard:ScanRoot(id)
end
local retained=0
for _,result in pairs(guard._state.scanCache) do retained=retained+#(result.contentRevision or "") end
check(retained <= 8*1024*1024, "evicted revisions remained retained through scan results")
guard:SetEnabled(false, true)
hostPrint("PASS real TRP3 guard: compiler, Lua budget, local dispatch, writes, failure, identity, delayed revocation, receipt")
