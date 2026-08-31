-- RPBox TRP3 Extended object protection.
-- Standard workflows, raw Lua Script Effects, runtime side effects, and
-- trusted-publisher policy are evaluated before execution.

local ADDON_NAME, ns = ...
local Unpack = unpack or table.unpack

local Guard = {}
ns.ItemGuard = Guard

local ISOLATION_ICON = "ui-engineering-90-remote-close-icon"
local VISUAL_CHILD_GROUPS = { "IN", "QE", "ST" }
local DB_VERSION = 2
local RULE_VERSION = table.concat({
    "core=" .. tostring(ns.ItemGuardRules and ns.ItemGuardRules.RULE_VERSION or 1),
    "sound=" .. tostring(ns.ItemGuardSoundRules and ns.ItemGuardSoundRules.RULE_VERSION or 0),
    "lifecycle=" .. tostring(ns.ItemGuardLifecycleRules and ns.ItemGuardLifecycleRules.RULE_VERSION or 0),
    "variable=" .. tostring(ns.ItemGuardVariableRules and ns.ItemGuardVariableRules.RULE_VERSION or 0),
    "aura=" .. tostring(ns.ItemGuardAuraRules and ns.ItemGuardAuraRules.RULE_VERSION or 0),
    "content=" .. tostring(ns.ItemGuardContentRules and ns.ItemGuardContentRules.RULE_VERSION or 0),
    "lua=" .. tostring(ns.ItemGuardLuaRules and ns.ItemGuardLuaRules.RULE_VERSION or 0),
}, ";")

local LIMITS = {
    maxWorkflows = 128,
    maxSteps = 768,
    maxEffects = 3072,
    maxExpandedSteps = 4096,
    maxItemAddEffects = 24,
    maxSingleItemAdd = 1000,
    runtimeWindow = 5,
    runtimeItemAddCalls = 20,
    runtimeItemAddCount = 200,
    runtimeDirectAddCalls = 20,
    runtimeDirectAddCount = 200,
    runtimeLongWindow = 60,
    runtimeLongWriteCalls = 100,
    runtimeLongWriteCount = 1000,
    auraCancelWatchSeconds = 2,
}

local state = {
    enabled = false,
    installed = false,
    callbacksRegistered = false,
    blacklistCallbackRegistered = false,
    publisherCallbackRegistered = false,
    mutating = false,
    scanGeneration = 0,
    retryGeneration = 0,
    currentRoot = nil,
    currentDestruction = nil,
    currentVariableEffect = nil,
    currentAuraCancellation = nil,
    auraCancellationWatch = {},
    runtime = {},
    luaDepth = {},
    luaLibraryBaseline = nil,
    scanCache = {},
    scanStatus = {},
    pendingProtection = {},
    pendingScans = {},
    knownRoots = {},
    auraRemovalQueued = {},
    temporaryAllow = {},
    original = {},
    changeCallbacks = {},
}
Guard._state = state

local function Print(message, color)
    color = color or "00ff00"
    print("|cff" .. color .. "[RPBox 防护]|r " .. tostring(message))
end

local function EnsureDatabase()
    RPBox_ItemGuardDB = RPBox_ItemGuardDB or {}
    local previousVersion = tonumber(RPBox_ItemGuardDB.version) or 0
    RPBox_ItemGuardDB.quarantined = RPBox_ItemGuardDB.quarantined or {}
    RPBox_ItemGuardDB.ignored = RPBox_ItemGuardDB.ignored or {}
    RPBox_ItemGuardDB.findings = RPBox_ItemGuardDB.findings or {}
    if previousVersion < 2 then
        -- Earlier data used the popup's one-time "ignore risk" action as a
        -- persistent allowlist. Current data requires an explicit GUI action.
        RPBox_ItemGuardDB.ignored = {}
    end
    RPBox_ItemGuardDB.version = DB_VERSION
    return RPBox_ItemGuardDB
end

local function NotifyChanged()
    for _, callback in ipairs(state.changeCallbacks) do
        local ok, err = pcall(callback)
        if not ok then Print("风险界面刷新失败：" .. tostring(err), "ff5555") end
    end
end

function Guard:RegisterOnChanged(callback)
    if type(callback) == "function" then
        state.changeCallbacks[#state.changeCallbacks + 1] = callback
    end
end

local function IsReady()
    return TRP3_API
        and TRP3_API.extended
        and TRP3_API.script
        and TRP3_API.inventory
        and TRP3_API.script.executeClassScript
        and TRP3_API.script.playEffect
        and TRP3_API.script.setVar
        and TRP3_API.script.runLuaScriptEffect
        and TRP3_API.inventory.addItem
        and TRP3_API.extended.registerObject
        and TRP3_API.extended.auras
        and TRP3_API.extended.auras.apply
        and TRP3_API.extended.auras.cancel
        and TRP3_API.extended.auras.remove
        and TRP3_API.extended.document
        and TRP3_API.extended.document.showDocument
        and TRP3_API.extended.document.showDocumentClass
        and TRP3_Extended
        and TRP3_Extended.Events
        and TRP3_DB
end

local function GetRootID(classID)
    if type(classID) ~= "string" or classID == "" then return nil end
    if TRP3_API and TRP3_API.extended and TRP3_API.extended.getRootClassID then
        return TRP3_API.extended.getRootClassID(classID)
    end
    return classID:match("^[^ ]+")
end

local function GetRootObject(rootID)
    if not rootID then return nil end
    -- TRP3_DB.global is the authoritative registered class used by execution.
    -- Prefer it so a newly received replacement cannot be shadowed by an
    -- older tools/exchange copy during the pending scan.
    if TRP3_DB and TRP3_DB.global and TRP3_DB.global[rootID] then
        return TRP3_DB.global[rootID]
    end
    if TRP3_Tools_DB and TRP3_Tools_DB[rootID] then
        return TRP3_Tools_DB[rootID]
    end
    if TRP3_Exchange_DB and TRP3_Exchange_DB[rootID] then
        return TRP3_Exchange_DB[rootID]
    end
    return nil
end

local function GetObjectType(root)
    if type(root) ~= "table" or not TRP3_DB or not TRP3_DB.types then return nil end
    if root.TY == TRP3_DB.types.ITEM then return "item" end
    local auraType = TRP3_DB.types.AURA
    local documentType = TRP3_DB.types.DOCUMENT
    if auraType and root.TY == auraType then return "aura" end
    if documentType and root.TY == documentType then return "document" end
    -- Auras and documents can be nested below campaign-style roots. The root
    -- remains the isolation owner and is therefore a protected object too.
    local seen = {}
    local function NestedProtectedType(class)
        if type(class) ~= "table" or seen[class] then return nil end
        seen[class] = true
        if auraType and class.TY == auraType then return "aura" end
        if documentType and class.TY == documentType then return "document" end
        if type(class.SC) == "table" then
            for _, workflow in pairs(class.SC) do
                for _, step in pairs(type(workflow) == "table" and type(workflow.ST) == "table"
                    and workflow.ST or {}) do
                    for _, effect in pairs(type(step) == "table" and type(step.e) == "table"
                        and step.e or {}) do
                        if type(effect) == "table" and (effect.id == "script" or effect.id == "secure_macro") then
                            return "lua"
                        end
                    end
                end
            end
        end
        for _, groupName in ipairs(VISUAL_CHILD_GROUPS) do
            for _, child in pairs(type(class[groupName]) == "table" and class[groupName] or {}) do
                local childType = NestedProtectedType(child)
                if childType then return childType end
            end
        end
        return nil
    end
    return NestedProtectedType(root)
end

local function IsProtectedRoot(root)
    return GetObjectType(root) ~= nil
end

local function IsVisuallyIsolated(root)
    return type(root) == "table" and type(root.BA) == "table"
        and root.BA.IC == ISOLATION_ICON and root.BA.US == nil
end

local function ResolveRootFromArgs(effectArgs, fallbackID)
    local classID = fallbackID
    if type(effectArgs) == "table" then
        classID = effectArgs.classID or classID
        if not classID and type(effectArgs.object) == "table" then
            classID = effectArgs.object.id
        end
        if not classID and type(effectArgs.container) == "table" then
            classID = effectArgs.container.id
        end
    end
    return GetRootID(classID)
end

local function CanonicalID(value)
    if type(value) == "string" or type(value) == "number" then
        return tostring(value)
    end
    return nil
end

local function GetStep(steps, stepID)
    if type(steps) ~= "table" or not stepID then return nil end
    return steps[stepID] or steps[tonumber(stepID)]
end

local function AddReason(result, reason)
    result.reasons = result.reasons or {}
    if not result.reasonSet then
        result.reasonSet = {}
        for _, existing in ipairs(result.reasons) do result.reasonSet[existing] = true end
    end
    if not result.reasonSet[reason] then
        result.reasonSet[reason] = true
        result.reasons[#result.reasons + 1] = reason
    end
end

local function AddScore(result, points, reason)
    result.score = (result.score or 0) + (tonumber(points) or 0)
    if reason then AddReason(result, reason) end
end

local function NewScanResult(rootID)
    return {
        rootID = rootID,
        blocked = false,
        score = 0,
        reasons = {},
        reasonSet = {},
        signatureParts = {},
        workflows = 0,
        steps = 0,
        effects = 0,
        itemAdds = 0,
        workflowCalls = 0,
    }
end

local function AddEdge(edges, target)
    target = CanonicalID(target)
    if target then edges[target] = true end
end

local function FindStepCycleNodes(steps, edges)
    local visiting, complete = {}, {}
    local stack, stackIndex, cycleNodes = {}, {}, {}

    local function Visit(stepID)
        if complete[stepID] then return end
        if visiting[stepID] then
            local first = stackIndex[stepID] or 1
            for index = first, #stack do cycleNodes[stack[index]] = true end
            return
        end
        visiting[stepID] = true
        stack[#stack + 1] = stepID
        stackIndex[stepID] = #stack
        for target in pairs(edges[stepID] or {}) do
            if GetStep(steps, target) then Visit(target) end
        end
        stackIndex[stepID] = nil
        stack[#stack] = nil
        visiting[stepID] = nil
        complete[stepID] = true
    end

    if GetStep(steps, "1") then Visit("1") end
    return cycleNodes
end

local function EstimateExpandedSteps(steps, edges)
    local cache, calculating = {}, {}

    local function Cost(stepID)
        if cache[stepID] then return cache[stepID] end
        if calculating[stepID] then return LIMITS.maxExpandedSteps + 1 end
        calculating[stepID] = true
        local total = 1
        for target in pairs(edges[stepID] or {}) do
            if GetStep(steps, target) then
                total = total + Cost(target)
                if total > LIMITS.maxExpandedSteps then break end
            end
        end
        calculating[stepID] = nil
        cache[stepID] = total
        return total
    end

    if not GetStep(steps, "1") then return 0 end
    return Cost("1")
end

local function ScanWorkflow(classLabel, workflowID, workflow, result, workflowEdges, workflowItemAdds)
    result.workflows = result.workflows + 1
    if result.workflows > LIMITS.maxWorkflows then
        AddScore(result, 30, "工作流数量超过常规规模")
        return
    end

    local steps = type(workflow) == "table" and workflow.ST or nil
    if type(steps) ~= "table" then return end

    local stepEdges = {}
    local itemAddSteps = {}
    local currentWorkflowItemAdds = 0
    local targets = workflowEdges[workflowID] or {}
    workflowEdges[workflowID] = targets

    -- First build the structural graph without inspecting effect payloads.
    for rawStepID, step in pairs(steps) do
        local stepID = CanonicalID(rawStepID)
        if stepID and type(step) == "table" then
            local edges = {}
            stepEdges[stepID] = edges
            AddEdge(edges, step.n)
            if step.t == "branch" and type(step.b) == "table" then
                for _, branch in pairs(step.b) do
                    if type(branch) == "table" then AddEdge(edges, branch.n) end
                end
            end
        end
    end

    -- Only reachable steps can be compiled or executed. Disconnected editor
    -- leftovers must not create false-positive cycles or item_add counts.
    local reachable = {}
    local function MarkReachable(stepID)
        if reachable[stepID] or not GetStep(steps, stepID) then return end
        reachable[stepID] = true
        for target in pairs(stepEdges[stepID] or {}) do MarkReachable(target) end
    end
    MarkReachable("1")

    for stepID in pairs(reachable) do
        local step = GetStep(steps, stepID)
        result.steps = result.steps + 1
        if result.steps > LIMITS.maxSteps then
            AddScore(result, 30, "工作流步骤数量超过常规规模")
            return
        end

        if step.t == "branch" and type(step.b) == "table" then
            for _, branch in pairs(step.b) do
                if type(branch) == "table"
                    and type(branch.failWorkflow) == "string"
                    and branch.failWorkflow ~= "" then
                    targets[branch.failWorkflow] = true
                    result.workflowCalls = result.workflowCalls + 1
                end
            end
        end

        if type(step.e) == "table" then
            for _, itemEffect in pairs(step.e) do
                if type(itemEffect) == "table" then
                    result.effects = result.effects + 1
                    if result.effects > LIMITS.maxEffects then
                        AddScore(result, 30, "工作流效果数量超过常规规模")
                        return
                    end

                    local effectID = tostring(itemEffect.id or "")
                    local effectArgs = type(itemEffect.args) == "table" and itemEffect.args or {}
                    local signatureArg1, signatureArg2 = "", ""
                    if effectID ~= "script" then
                        signatureArg1 = tostring(effectArgs[1] or "")
                        signatureArg2 = tostring(effectArgs[2] or "")
                    end
                    result.signatureParts[#result.signatureParts + 1] = table.concat({
                        classLabel,
                        tostring(workflowID),
                        stepID,
                        effectID,
                        signatureArg1,
                        signatureArg2,
                    }, "\30")

                    if effectID == "item_add" then
                        result.itemAdds = result.itemAdds + 1
                        currentWorkflowItemAdds = currentWorkflowItemAdds + 1
                        itemAddSteps[stepID] = true
                        local requestedCount = tonumber(effectArgs[2])
                        if requestedCount and requestedCount > LIMITS.maxSingleItemAdd then
                            result.blocked = true
                            AddScore(result, 120, "单次添加物品数量异常：" .. tostring(requestedCount))
                        end
                    elseif effectID == "run_workflow" or effectID == "run_item_workflow" then
                        result.workflowCalls = result.workflowCalls + 1
                        local source = tostring(effectArgs[1] or "o")
                        local target = tostring(effectArgs[2] or "")
                        if source == "o" and target ~= "" then targets[target] = true end
                    end
                end
            end
        end
    end

    workflowItemAdds[workflowID] = currentWorkflowItemAdds

    local cycleNodes = FindStepCycleNodes(steps, stepEdges)
    local hasCycle, cycleAddsItems = false, false
    for stepID in pairs(cycleNodes) do
        hasCycle = true
        if itemAddSteps[stepID] then cycleAddsItems = true end
    end
    if hasCycle then
        AddScore(result, 20, "工作流“" .. tostring(workflowID) .. "”存在步骤循环")
        if cycleAddsItems then
            result.blocked = true
            AddScore(result, 120, "步骤循环中包含重复添加物品行为")
        end
    elseif EstimateExpandedSteps(steps, stepEdges) > LIMITS.maxExpandedSteps then
        AddScore(result, 25, "工作流“" .. tostring(workflowID) .. "”展开规模异常")
        if currentWorkflowItemAdds > 0 then
            result.blocked = true
            AddScore(result, 100, "异常展开工作流中包含物品添加行为")
        end
    end
end

local function FindWorkflowCycleNodes(workflowEdges)
    local visiting, complete = {}, {}
    local stack, stackIndex, cycleNodes = {}, {}, {}

    local function Visit(workflowID)
        if complete[workflowID] then return end
        if visiting[workflowID] then
            local first = stackIndex[workflowID] or 1
            for index = first, #stack do cycleNodes[stack[index]] = true end
            return
        end
        visiting[workflowID] = true
        stack[#stack + 1] = workflowID
        stackIndex[workflowID] = #stack
        for target in pairs(workflowEdges[workflowID] or {}) do
            if workflowEdges[target] then Visit(target) end
        end
        stackIndex[workflowID] = nil
        stack[#stack] = nil
        visiting[workflowID] = nil
        complete[workflowID] = true
    end

    for workflowID in pairs(workflowEdges) do Visit(workflowID) end
    return cycleNodes
end

local function ScanClass(class, classLabel, result, seen)
    if type(class) ~= "table" or seen[class] then return end
    seen[class] = true

    if type(class.SC) == "table" then
        local workflowEdges = {}
        local workflowItemAdds = {}
        for rawWorkflowID, workflow in pairs(class.SC) do
            local workflowID = CanonicalID(rawWorkflowID)
            if workflowID then
                ScanWorkflow(classLabel, workflowID, workflow, result, workflowEdges, workflowItemAdds)
            end
        end
        local recursiveNodes = FindWorkflowCycleNodes(workflowEdges)
        local hasRecursion, recursiveAddsItems = false, false
        for workflowID in pairs(recursiveNodes) do
            hasRecursion = true
            if (workflowItemAdds[workflowID] or 0) > 0 then recursiveAddsItems = true end
        end
        if hasRecursion then
            AddScore(result, 20, "存在递归工作流调用")
            if recursiveAddsItems then
                result.blocked = true
                AddScore(result, 120, "递归工作流中包含重复添加物品行为")
            end
        end
    end

    for _, groupName in ipairs({ "IN", "QE", "ST" }) do
        local group = class[groupName]
        if type(group) == "table" then
            for childID, child in pairs(group) do
                ScanClass(child, classLabel .. " " .. tostring(childID), result, seen)
            end
        end
    end
end

local function BuildFingerprint(root, result)
    local metadata = type(root.MD) == "table" and root.MD or {}
    table.sort(result.signatureParts)
    return table.concat({
        tostring(RULE_VERSION),
        tostring(metadata.V or ""),
        tostring(metadata.SD or ""),
        tostring(metadata.CB or ""),
        tostring(result.workflows),
        tostring(result.steps),
        tostring(result.effects),
        tostring(result.itemAdds),
        table.concat(result.signatureParts, "\29"),
    }, "\31")
end

local function HasHardFinding(result)
    for _, finding in ipairs(result and result.findings or {}) do
        if finding.hard then return true end
    end
    return false
end

local function BuildRuleContext(rootID, root)
    local context = { entrypoints = {} }
    local separator = TRP3_API and TRP3_API.extended and TRP3_API.extended.ID_SEPARATOR or " "
    local auraType = TRP3_DB and TRP3_DB.types and TRP3_DB.types.AURA
    local seen = {}
    local function Visit(classID, class)
        if type(class) ~= "table" or seen[class] then return end
        seen[class] = true
        if auraType and class.TY == auraType then
            local entries = {}
            if type(class.HA) == "table" then
                for _, handler in pairs(class.HA) do
                    local workflowID = type(handler) == "table" and CanonicalID(handler.SC) or nil
                    if workflowID then entries[workflowID] = true end
                end
            end
            if next(entries) then context.entrypoints[classID] = entries end
        end
        for _, groupName in ipairs(VISUAL_CHILD_GROUPS) do
            local group = class[groupName]
            if type(group) == "table" then
                for childID, child in pairs(group) do
                    Visit(classID .. separator .. tostring(childID), child)
                end
            end
        end
    end
    Visit(rootID, root)
    if ns.ItemGuardPublisherWhitelist and ns.ItemGuardPublisherWhitelist.MatchRoot then
        context.publisherTrust = ns.ItemGuardPublisherWhitelist.MatchRoot(rootID, root)
        context.trustedPublisher = context.publisherTrust ~= nil
    end
    return context
end

local function MergeRuleResults(rootID, root, result, context)
    local modules = {
        { name = "sound", rules = ns.ItemGuardSoundRules },
        { name = "lifecycle", rules = ns.ItemGuardLifecycleRules },
        { name = "variable", rules = ns.ItemGuardVariableRules },
        { name = "aura", rules = ns.ItemGuardAuraRules },
        { name = "content", rules = ns.ItemGuardContentRules },
        { name = "lua", rules = ns.ItemGuardLuaRules },
    }
    local behaviorScore = tonumber(result.behaviorScore) or 0
    local amplificationScore = tonumber(result.amplificationScore) or 0
    local observationScore = tonumber(result.observationScore) or 0
    local hasSideEffect = behaviorScore > 0
    local hardBlocked = HasHardFinding(result)
    local fingerprints = { "core=" .. tostring(result.fingerprint or "") }
    result.moduleMetrics = result.moduleMetrics or {}

    for _, module in ipairs(modules) do
        local rules = module.rules
        if rules and rules.Analyze then
            local ok, moduleResult = pcall(rules.Analyze, rootID, root, context or {})
            if ok and type(moduleResult) == "table" then
                behaviorScore = behaviorScore + (tonumber(moduleResult.behaviorScore) or 0)
                -- Structural modules observe many of the same loops. Taking the
                -- strongest result avoids charging the same cycle more than once.
                amplificationScore = math.max(
                    amplificationScore,
                    tonumber(moduleResult.amplificationScore) or 0
                )
                observationScore = math.max(
                    observationScore,
                    tonumber(moduleResult.observationScore) or 0
                )
                hasSideEffect = hasSideEffect or moduleResult.hasSideEffect == true
                    or (tonumber(moduleResult.behaviorScore) or 0) > 0
                hardBlocked = hardBlocked or HasHardFinding(moduleResult)
                result.advisory = result.advisory or moduleResult.advisory == true
                result.moduleMetrics[module.name] = moduleResult.metrics
                for _, reason in ipairs(moduleResult.reasons or {}) do AddReason(result, reason) end
                result.findings = result.findings or {}
                for _, finding in ipairs(moduleResult.findings or {}) do
                    if finding.module == nil then finding.module = module.name end
                    result.findings[#result.findings + 1] = finding
                end
                fingerprints[#fingerprints + 1] = module.name .. "=" .. tostring(moduleResult.fingerprint or "")
            else
                result.moduleMetrics[module.name] = { analysisError = tostring(moduleResult) }
                fingerprints[#fingerprints + 1] = module.name .. "=error"
            end
        end
    end

    result.behaviorScore = behaviorScore
    result.amplificationScore = math.min(amplificationScore, 40)
    result.observationScore = observationScore
    result.hasSideEffect = hasSideEffect
    result.score = result.behaviorScore + result.amplificationScore + result.observationScore
    result.blocked = hardBlocked or (result.hasSideEffect and result.score >= 100)
    result.ruleVersion = RULE_VERSION
    result.publisherTrust = context and context.publisherTrust or nil
    result.fingerprint = table.concat(fingerprints, "|")
    return result
end

function Guard:ScanRoot(rootID, root)
    rootID = GetRootID(rootID)
    root = root or GetRootObject(rootID)
    if not rootID or type(root) ~= "table" then return nil end

    local context = BuildRuleContext(rootID, root)
    local result
    if ns.ItemGuardRules and ns.ItemGuardRules.Analyze then
        result = ns.ItemGuardRules.Analyze(rootID, root, context)
    else
        result = NewScanResult(rootID)
        ScanClass(root, rootID, result, {})
        if result.itemAdds > LIMITS.maxItemAddEffects then
            result.blocked = true
            AddScore(result, 120, "物品添加效果数量异常：" .. tostring(result.itemAdds))
        end
        result.fingerprint = BuildFingerprint(root, result)
    end
    result = MergeRuleResults(rootID, root, result, context)

    if ns.ItemGuardBlacklist and ns.ItemGuardBlacklist.MatchRoot then
        local blacklistMatch = ns.ItemGuardBlacklist.MatchRoot(rootID, root)
        if blacklistMatch then
            local reason = "对象来源命中"
                .. (blacklistMatch.source == "builtin" and "系统" or "用户")
                .. "黑名单：" .. tostring(blacklistMatch.identity)
            result.blocked = true
            result.policyScore = 120
            result.score = (tonumber(result.score) or 0) + result.policyScore
            result.reasons = result.reasons or {}
            result.reasons[#result.reasons + 1] = reason
            result.findings = result.findings or {}
            result.findings[#result.findings + 1] = {
                kind = "source_blacklist",
                score = 120,
                reason = reason,
                identity = blacklistMatch.identity,
                source = blacklistMatch.source,
                field = blacklistMatch.field,
                hard = true,
            }
            result.blacklistMatch = blacklistMatch
            result.fingerprint = tostring(result.fingerprint)
                .. ":blacklist:" .. tostring(blacklistMatch.source)
                .. ":" .. tostring(blacklistMatch.identity)
        end
    end

    state.scanCache[rootID] = result
    return result
end

local function GetIgnoredFingerprint(rootID)
    return EnsureDatabase().ignored[rootID]
end

function Guard:IsIgnored(rootID, fingerprint)
    local ignoredFingerprint = GetIgnoredFingerprint(rootID)
    return ignoredFingerprint ~= nil and ignoredFingerprint == fingerprint
end

function Guard:IsQuarantined(rootID)
    return EnsureDatabase().quarantined[GetRootID(rootID)] ~= nil
end

local function BuildPrioritizedReasons(result)
    local candidates = {}
    local seen = {}
    local function AddCandidate(reason, score, hard, kind, index)
        if type(reason) ~= "string" or reason == "" or seen[reason] then return end
        seen[reason] = true
        candidates[#candidates + 1] = {
            reason = reason,
            score = tonumber(score) or 0,
            hard = hard and true or false,
            structural = kind == "workflow_recursion"
                or kind == "step_cycle"
                or kind == "destruction_workflow_recursion",
            index = index or (#candidates + 1),
        }
    end

    for index, finding in ipairs(result and result.findings or {}) do
        AddCandidate(finding.reason, finding.score, finding.hard, finding.kind, index)
    end
    for index, reason in ipairs(result and result.reasons or {}) do
        AddCandidate(reason, 0, false, nil, 10000 + index)
    end
    table.sort(candidates, function(left, right)
        if left.hard ~= right.hard then return left.hard end
        if left.score ~= right.score then return left.score > right.score end
        if left.structural ~= right.structural then return not left.structural end
        return left.index < right.index
    end)

    local reasons = {}
    for _, candidate in ipairs(candidates) do reasons[#reasons + 1] = candidate.reason end
    return reasons
end

function Guard:UpdateFinding(rootID, result, source)
    rootID = GetRootID(rootID)
    local root = GetRootObject(rootID)
    if not rootID or type(root) ~= "table" or not result then return nil end
    local database = EnsureDatabase()
    local finding = database.findings[rootID]
    if not finding then
        finding = {
            rootID = rootID,
            firstSeenAt = time(),
        }
        database.findings[rootID] = finding
    end
    result.reasons = BuildPrioritizedReasons(result)
    finding.itemName = root.BA and root.BA.NA or rootID
    finding.objectType = GetObjectType(root) or "object"
    finding.reasons = result.reasons
    finding.score = result.score or 0
    finding.behaviorScore = result.behaviorScore or 0
    finding.amplificationScore = result.amplificationScore or 0
    finding.observationScore = result.observationScore or 0
    finding.policyScore = result.policyScore or 0
    finding.findings = result.findings
    finding.advisory = result.advisory == true
    finding.publisherTrust = result.publisherTrust
    finding.fingerprint = result.fingerprint
    finding.lastSeenAt = time()
    finding.source = source or finding.source or "scan"
    return finding
end

function Guard:RemoveFinding(rootID)
    rootID = GetRootID(rootID)
    if not rootID then return end
    local database = EnsureDatabase()
    database.findings[rootID] = nil
    database.ignored[rootID] = nil
end

local function RefreshTRP3(rootID)
    if not IsReady() then return end
    state.mutating = true
    if TRP3_API.script.clearRootCompilation then
        TRP3_API.script.clearRootCompilation(rootID)
    end
    if TRP3_Extended.Events.ON_OBJECT_UPDATED then
        TRP3_Extended:TriggerEvent(TRP3_Extended.Events.ON_OBJECT_UPDATED, rootID, TRP3_DB.types.ITEM)
    end
    if TRP3_Extended.Events.REFRESH_BAG then
        TRP3_Extended:TriggerEvent(TRP3_Extended.Events.REFRESH_BAG)
    end
    state.mutating = false
end

local function CollectVisualTargets(rootID, root)
    local targets = {}
    local seen = {}
    local separator = TRP3_API and TRP3_API.extended and TRP3_API.extended.ID_SEPARATOR or " "
    local function Visit(classID, class)
        if type(class) ~= "table" or seen[class] then return end
        seen[class] = true
        if type(class.BA) == "table" then
            targets[#targets + 1] = { classID = classID, class = class }
        end
        for _, groupName in ipairs(VISUAL_CHILD_GROUPS) do
            local group = class[groupName]
            if type(group) == "table" then
                for childID, child in pairs(group) do
                    Visit(classID .. separator .. tostring(childID), child)
                end
            end
        end
    end
    Visit(rootID, root)
    return targets
end

function Guard:ApplyVisualIsolation(rootID, record)
    rootID = GetRootID(rootID)
    local root = GetRootObject(rootID)
    if type(root) ~= "table" or type(record) ~= "table" then return false end
    record.visualStates = record.visualStates or {}
    local changed = false
    for _, target in ipairs(CollectVisualTargets(rootID, root)) do
        local classID, class = target.classID, target.class
        local visual = record.visualStates[classID]
        if not visual then
            if classID == rootID and record.hadIcon ~= nil then
                visual = {
                    hadIcon = record.hadIcon and true or false,
                    originalIcon = record.originalIcon,
                    hadUsable = record.originalUsable and true or false,
                    originalUsable = record.originalUsable and true or nil,
                }
            else
                visual = {
                    hadIcon = class.BA.IC ~= nil,
                    originalIcon = class.BA.IC,
                    hadUsable = class.BA.US ~= nil,
                    originalUsable = class.BA.US,
                }
            end
            record.visualStates[classID] = visual
        end
        if class.BA.US ~= nil or class.BA.IC ~= ISOLATION_ICON then changed = true end
        class.BA.US = nil
        class.BA.IC = ISOLATION_ICON
    end
    if changed then RefreshTRP3(rootID) end
    return true
end

function Guard:RestoreVisualIsolation(rootID, record)
    rootID = GetRootID(rootID)
    local root = GetRootObject(rootID)
    if type(root) ~= "table" or type(record) ~= "table" then return false end
    local changed = false
    for _, target in ipairs(CollectVisualTargets(rootID, root)) do
        local classID, class = target.classID, target.class
        local visual = record.visualStates and record.visualStates[classID]
        if not visual and classID == rootID and record.hadIcon ~= nil then
            visual = {
                hadIcon = record.hadIcon and true or false,
                originalIcon = record.originalIcon,
                hadUsable = record.originalUsable and true or false,
                originalUsable = record.originalUsable and true or nil,
            }
        end
        if visual then
            local restoredIcon = visual.hadIcon and visual.originalIcon or nil
            local restoredUsable = visual.hadUsable and visual.originalUsable or nil
            if class.BA.IC ~= restoredIcon or class.BA.US ~= restoredUsable then changed = true end
            class.BA.IC = restoredIcon
            class.BA.US = restoredUsable
        end
    end
    if changed then RefreshTRP3(rootID) end
    return true
end

function Guard:GetScanStatus(rootID)
    rootID = GetRootID(rootID)
    local status = rootID and state.scanStatus[rootID]
    return status and status.status or "unscanned", status and status.fingerprint or nil
end

function Guard:MarkUnscanned(rootID, source)
    rootID = GetRootID(rootID)
    local root = GetRootObject(rootID)
    if not rootID or not IsProtectedRoot(root) then return false end
    state.scanCache[rootID] = nil
    state.temporaryAllow[rootID] = nil
    state.scanStatus[rootID] = {
        status = "unscanned",
        source = source or "update",
        markedAt = time(),
        replaced = state.knownRoots[rootID] ~= nil and state.knownRoots[rootID] ~= root,
    }

    local database = EnsureDatabase()
    local pending = state.pendingProtection[rootID]
    if pending and pending.rootRef ~= root then
        -- The root was replaced again before its prior scan completed. Never
        -- restore the old object's visual state onto the new received data.
        state.pendingProtection[rootID] = nil
        pending = nil
    end
    local visualPending = source == "received" or source == "updated"
    if visualPending and root.TY == TRP3_DB.types.ITEM
        and not database.quarantined[rootID]
        and not pending then
        root.BA = root.BA or {}
        local record = {
            rootID = rootID,
            rootRef = root,
            itemName = root.BA.NA or rootID,
            hadIcon = root.BA.IC ~= nil,
            originalIcon = root.BA.IC,
            originalUsable = root.BA.US and true or false,
            source = "pending_scan",
        }
        state.pendingProtection[rootID] = record
        self:ApplyVisualIsolation(rootID, record)
    end
    NotifyChanged()
    return true
end

local function ReleasePendingProtection(rootID)
    local record = state.pendingProtection[rootID]
    if not record then return end
    if record.rootRef == GetRootObject(rootID) then
        Guard:RestoreVisualIsolation(rootID, record)
    end
    state.pendingProtection[rootID] = nil
end

function Guard:QueueScan(rootID, source)
    rootID = GetRootID(rootID)
    if not rootID or not state.enabled then return false end
    if not self:MarkUnscanned(rootID, source) then return false end
    local token = (state.pendingScans[rootID] or 0) + 1
    state.pendingScans[rootID] = token
    C_Timer.After(0, function()
        if not state.enabled or state.pendingScans[rootID] ~= token then return end
        state.pendingScans[rootID] = nil
        Guard:ScanAndApply(rootID)
    end)
    return true
end

local function QueueAuraRemoval(rootID, root)
    local auras = TRP3_API and TRP3_API.extended and TRP3_API.extended.auras
    if not auras or not auras.remove then return end
    local auraType = TRP3_DB and TRP3_DB.types and TRP3_DB.types.AURA
    if not auraType then return end
    local separator = TRP3_API.extended.ID_SEPARATOR or " "
    local seen = {}
    local function Visit(classID, class)
        if type(class) ~= "table" or seen[class] then return end
        seen[class] = true
        if class.TY == auraType and not state.auraRemovalQueued[classID] then
            state.auraRemovalQueued[classID] = true
            C_Timer.After(0, function()
                state.auraRemovalQueued[classID] = nil
                if state.enabled and Guard:IsQuarantined(rootID) then
                    auras.remove(classID)
                end
            end)
        end
        for _, groupName in ipairs(VISUAL_CHILD_GROUPS) do
            local group = class[groupName]
            if type(group) == "table" then
                for childID, child in pairs(group) do
                    Visit(classID .. separator .. tostring(childID), child)
                end
            end
        end
    end
    Visit(rootID, root)
end

function Guard:Quarantine(rootID, result, source)
    if not state.enabled then return false end
    rootID = GetRootID(rootID)
    local root = GetRootObject(rootID)
    if not rootID or not IsProtectedRoot(root) then return false end

    result = result or self:ScanRoot(rootID, root)
    if not result or self:IsIgnored(rootID, result.fingerprint) then return false end
    ReleasePendingProtection(rootID)

    local database = EnsureDatabase()
    local finding = self:UpdateFinding(rootID, result, source)
    local record = database.quarantined[rootID]
    local scan = state.scanStatus[rootID]
    if not record then
        root.BA = root.BA or {}
        record = {
            rootID = rootID,
            itemName = root.BA.NA or rootID,
            hadIcon = root.BA.IC ~= nil,
            originalIcon = root.BA.IC,
            originalUsable = root.BA.US and true or false,
            detectedAt = time(),
        }
        database.quarantined[rootID] = record
    elseif (scan and scan.replaced) or (record.fingerprint ~= result.fingerprint
        and root.BA and root.BA.IC ~= ISOLATION_ICON) then
        record.itemName = root.BA.NA or rootID
        record.hadIcon = root.BA.IC ~= nil
        record.originalIcon = root.BA.IC
        record.originalUsable = root.BA.US and true or false
        record.visualStates = nil
    end
    record.reasons = result.reasons
    record.fingerprint = result.fingerprint
    record.ruleVersion = RULE_VERSION
    record.source = source or "scan"
    if finding then finding.disposition = "quarantined" end

    if root.TY == TRP3_DB.types.ITEM then self:ApplyVisualIsolation(rootID, record) end
    state.scanStatus[rootID] = { status = "blocked", fingerprint = result.fingerprint }
    QueueAuraRemoval(rootID, root)
    NotifyChanged()
    return true
end

function Guard:ReleaseQuarantine(rootID)
    rootID = GetRootID(rootID)
    local database = EnsureDatabase()
    local record = rootID and database.quarantined[rootID]
    if not record then return false end

    local root = GetRootObject(rootID)
    local scan = rootID and state.scanStatus[rootID]
    if root and root.TY == TRP3_DB.types.ITEM and IsVisuallyIsolated(root)
        and not (scan and scan.replaced) then
        self:RestoreVisualIsolation(rootID, record)
    end
    database.quarantined[rootID] = nil
    state.temporaryAllow[rootID] = record.fingerprint
    state.scanStatus[rootID] = { status = "released", fingerprint = record.fingerprint }
    local finding = database.findings[rootID]
    if finding then finding.disposition = "released" end
    NotifyChanged()
    return true
end

function Guard:SetIgnored(rootID, ignored)
    rootID = GetRootID(rootID)
    local root = GetRootObject(rootID)
    if not rootID or type(root) ~= "table" then return false end
    local database = EnsureDatabase()

    if ignored then
        state.temporaryAllow[rootID] = nil
        local result = self:ScanRoot(rootID, root)
        local existingFinding = database.findings[rootID]
        local findingSource = existingFinding and existingFinding.source
            or (result and result.blocked and "scan")
            or "manual"
        local finding = result and self:UpdateFinding(rootID, result, findingSource) or existingFinding
        database.ignored[rootID] = result and result.fingerprint or (finding and finding.fingerprint)
        if database.quarantined[rootID] then self:ReleaseQuarantine(rootID) end
        state.temporaryAllow[rootID] = nil
        state.scanStatus[rootID] = {
            status = "ignored",
            fingerprint = result and result.fingerprint or (finding and finding.fingerprint),
        }
        if finding then finding.disposition = "ignored" end
        NotifyChanged()
        return true
    end

    database.ignored[rootID] = nil
    state.temporaryAllow[rootID] = nil
    local finding = database.findings[rootID]
    if finding then finding.disposition = "released" end
    state.scanStatus[rootID] = { status = "unscanned" }
    self:ScanAndApply(rootID)
    NotifyChanged()
    return true
end

function Guard:SetIsolation(rootID, isolated)
    rootID = GetRootID(rootID)
    if not rootID then return false end
    local database = EnsureDatabase()
    if isolated then
        state.temporaryAllow[rootID] = nil
        database.ignored[rootID] = nil
        local result = self:ScanRoot(rootID)
        if not result then return false end
        result.blocked = true
        AddScore(result, 120, "用户选择保持隔离")
        return self:Quarantine(rootID, result, "manual")
    end
    return self:ReleaseQuarantine(rootID)
end

function Guard:GetRiskEntries()
    local database = EnsureDatabase()
    local entries = {}
    for rootID, finding in pairs(database.findings) do
        finding.reasons = BuildPrioritizedReasons(finding)
        local root = GetRootObject(rootID)
        local quarantine = database.quarantined[rootID]
        local ignored = database.ignored[rootID] ~= nil
            and database.ignored[rootID] == finding.fingerprint
        local scan = state.scanStatus[rootID]
        local status
        if scan and scan.status == "unscanned" then
            status = "unscanned"
        elseif ignored then
            status = "ignored"
        elseif quarantine then
            status = "quarantined"
        elseif finding.advisory or finding.disposition == "observed" then
            status = "observed"
        else
            status = "released"
        end
        entries[#entries + 1] = {
            rootID = rootID,
            itemName = finding.itemName or (root and root.BA and root.BA.NA) or rootID,
            objectType = finding.objectType or GetObjectType(root) or "object",
            reasons = status == "unscanned"
                    and { "等待安全扫描，执行已暂时阻止" }
                or finding.reasons or {},
            score = finding.score or 0,
            behaviorScore = finding.behaviorScore or 0,
            amplificationScore = finding.amplificationScore or 0,
            observationScore = finding.observationScore or 0,
            policyScore = finding.policyScore or 0,
            status = status,
            quarantined = quarantine ~= nil,
            ignored = ignored,
            pending = status == "unscanned",
            advisory = finding.advisory == true,
            publisherTrust = finding.publisherTrust,
            icon = quarantine and quarantine.originalIcon
                or (root and root.BA and root.BA.IC)
                or "inv_misc_questionmark",
            firstSeenAt = finding.firstSeenAt,
            lastSeenAt = finding.lastSeenAt,
        }
    end
    for rootID, scan in pairs(state.scanStatus) do
        if scan.status == "unscanned" and not database.findings[rootID] then
            local root = GetRootObject(rootID)
            if IsProtectedRoot(root) then
                entries[#entries + 1] = {
                    rootID = rootID,
                    itemName = root.BA and root.BA.NA or rootID,
                    objectType = GetObjectType(root) or "object",
                    reasons = { "等待安全扫描，执行已暂时阻止" },
                    score = 0,
                    behaviorScore = 0,
                    amplificationScore = 0,
                    observationScore = 0,
                    policyScore = 0,
                    status = "unscanned",
                    quarantined = false,
                    ignored = false,
                    pending = true,
                    icon = state.pendingProtection[rootID]
                            and state.pendingProtection[rootID].originalIcon
                        or (root.BA and root.BA.IC)
                        or "inv_misc_questionmark",
                }
            end
        end
    end
    local statusOrder = { unscanned = 1, quarantined = 2, observed = 3, released = 4, ignored = 5 }
    table.sort(entries, function(left, right)
        local leftOrder = statusOrder[left.status] or 9
        local rightOrder = statusOrder[right.status] or 9
        if leftOrder ~= rightOrder then return leftOrder < rightOrder end
        return tostring(left.itemName) < tostring(right.itemName)
    end)
    return entries
end

local function JoinReasons(record)
    if not record or type(record.reasons) ~= "table" or #record.reasons == 0 then
        return "1. 检测到异常工作流行为"
    end
    local lines = {}
    local visibleCount = math.min(#record.reasons, 6)
    for index = 1, visibleCount do
        lines[#lines + 1] = tostring(index) .. ". " .. tostring(record.reasons[index])
    end
    if #record.reasons > visibleCount then
        lines[#lines + 1] = "其余 " .. tostring(#record.reasons - visibleCount)
            .. " 项可在对象防护页面查看"
    end
    return table.concat(lines, "\n")
end

local function GetItemName(classID)
    if TRP3_API and TRP3_API.extended and TRP3_API.extended.getClass then
        local ok, class = pcall(TRP3_API.extended.getClass, classID)
        if ok and type(class) == "table" and class.BA and class.BA.NA then return class.BA.NA end
    end
    return tostring(classID or "未知道具")
end

local function BuildRemovalContext(rootID, slotButton, containerFrame)
    local slotInfo = slotButton and slotButton.info
    local container = containerFrame and containerFrame.info
    local slotID = slotButton and slotButton.slotID
    if type(slotInfo) ~= "table" or type(container) ~= "table"
        or type(container.content) ~= "table" or slotID == nil then return nil end
    if container.content[slotID] ~= slotInfo then
        local stringSlotID = tostring(slotID)
        if container.content[stringSlotID] ~= slotInfo then return nil end
        slotID = stringSlotID
    end
    local classID = tostring(slotInfo.id or "")
    local carrier
    if rootID and classID ~= "" then
        local root = GetRootObject(rootID)
        if type(root) == "table" then
            carrier = {
                rootID = rootID,
                classID = classID,
                name = root.BA and root.BA.NA or rootID,
                type = root.TY,
            }
        end
    end
    return {
        current = {
            container = container,
            slotID = slotID,
            slotInfo = slotInfo,
            name = GetItemName(slotInfo.id),
        },
        carrier = carrier,
    }
end

local function RemoveInventorySlot(target, silent)
    if type(target) ~= "table" or type(target.container) ~= "table"
        or type(target.container.content) ~= "table"
        or target.container.content[target.slotID] ~= target.slotInfo then
        Print("道具位置已经改变，未执行移除。", "ffcc00")
        return false
    end
    if not TRP3_API or not TRP3_API.inventory or not TRP3_API.inventory.removeSlotContent then
        Print("当前 TRP3 Extended 不支持安全移除槽位。", "ff5555")
        return false
    end
    local ok, err = pcall(
        TRP3_API.inventory.removeSlotContent,
        target.container,
        target.slotID,
        target.slotInfo,
        false
    )
    if not ok then
        Print("移除道具失败：" .. tostring(err), "ff5555")
        return false
    end
    if target.container.content[target.slotID] == target.slotInfo then
        Print("TRP3 未能移除该道具。", "ff5555")
        return false
    end
    if not silent then
        Print("已直接移除“" .. tostring(target.name or "道具") .. "”；未运行摧毁工作流。", "ffcc66")
    end
    return true
end

function Guard:RemoveInventoryTarget(data)
    local removal = data and data.removal
    local target = removal and removal.current
    local carrier = removal and removal.carrier
    if carrier then
        if not TRP3_API or not TRP3_API.extended or not TRP3_API.extended.removeObject then
            Print("当前 TRP3 Extended 不支持移除所属载体。", "ff5555")
            return false
        end
        if not target or type(target.container) ~= "table" or type(target.container.content) ~= "table"
            or target.container.content[target.slotID] ~= target.slotInfo then
            Print("道具位置已经改变，未移除所属载体。", "ffcc00")
            return false
        end
        if not RemoveInventorySlot(target, true) then return false end
        local ok, err = pcall(TRP3_API.extended.removeObject, carrier.rootID)
        if not ok then
            Print("当前道具已移除，但所属载体移除失败：" .. tostring(err), "ff5555")
            return false
        end
        local database = EnsureDatabase()
        database.quarantined[carrier.rootID] = nil
        database.ignored[carrier.rootID] = nil
        database.findings[carrier.rootID] = nil
        state.scanCache[carrier.rootID] = nil
        state.runtime[carrier.rootID] = nil
        state.temporaryAllow[carrier.rootID] = nil
        NotifyChanged()
        Print("已移除所属根载体“" .. tostring(carrier.name) .. "”及其内部对象。", "ffcc66")
        return true
    end
    if not target then
        Print("没有可移除的当前道具。", "ffcc00")
        return false
    end
    return RemoveInventorySlot(target)
end

function Guard:ShowRemovalConfirmation(data)
    local removal = data and data.removal
    if not removal or not removal.current or not StaticPopup_Show then return false end
    local function ShowPopup()
        local removalText = removal.carrier
            and ("将同时移除当前道具、根载体“" .. tostring(removal.carrier.name) .. "”及其内部对象。")
            or "将移除当前道具。"
        StaticPopup_Show(
            "RPBOX_ITEM_GUARD_REMOVE",
            removal.current.name,
            removalText,
            data
        )
    end
    if C_Timer and C_Timer.After then C_Timer.After(0, ShowPopup) else ShowPopup() end
    return true
end

function Guard:TrustRootPublisher(rootID)
    rootID = GetRootID(rootID)
    local root = GetRootObject(rootID)
    local whitelist = ns.ItemGuardPublisherWhitelist
    if not rootID or type(root) ~= "table" or not whitelist
        or not whitelist.ResolveRootPublisher or not whitelist.AddUser then
        return false, "无法识别该对象的发布者"
    end
    if ns.ItemGuardBlacklist and ns.ItemGuardBlacklist.MatchRoot
        and ns.ItemGuardBlacklist.MatchRoot(rootID, root) then
        return false, "该对象来源命中黑名单，不能加入发布者白名单"
    end
    local publisher = whitelist.ResolveRootPublisher(rootID, root)
    if not publisher or not publisher.identity then return false, "对象没有可用的发布者身份" end

    local added, message
    if publisher.source == "self" then
        added, message = true, "该对象已由 TRP3 确认为当前玩家所有"
    else
        added, message = whitelist.AddUser(
            publisher.identity,
            "从隔离对象“" .. tostring(root.BA and root.BA.NA or rootID) .. "”信任"
        )
        if not added and whitelist.MatchRoot and whitelist.MatchRoot(rootID, root) then
            added, message = true, "该发布者已在白名单中"
        end
    end
    if not added then return false, message or "无法信任该发布者" end

    self:ScanAndApply(rootID)
    if self:IsQuarantined(rootID) then
        return true, "已信任发布者“" .. tostring(publisher.identity)
            .. "”，但对象仍命中不可豁免硬风险，继续保持隔离"
    end
    return true, "已信任发布者“" .. tostring(publisher.identity) .. "”并解除策略隔离"
end

function Guard:ShowQuarantinePopup(rootID, slotButton, containerFrame)
    if not state.enabled then return end
    rootID = GetRootID(rootID)
    local record = rootID and EnsureDatabase().quarantined[rootID]
    if not record then return end
    local publisher = ns.ItemGuardPublisherWhitelist
        and ns.ItemGuardPublisherWhitelist.ResolveRootPublisher
        and ns.ItemGuardPublisherWhitelist.ResolveRootPublisher(rootID, GetRootObject(rootID)) or nil
    if StaticPopup_Show then
        StaticPopup_Show(
            "RPBOX_ITEM_GUARD_QUARANTINED",
            record.itemName or rootID,
            JoinReasons(record),
            {
                rootID = rootID,
                removal = BuildRemovalContext(rootID, slotButton, containerFrame),
                publisher = publisher,
            }
        )
    end
end

StaticPopupDialogs["RPBOX_ITEM_GUARD_QUARANTINED"] = {
    text = "道具“%s”已被 RPBox 隔离。\n\n已扫描到的风险：\n%s",
    button1 = "移除道具",
    button2 = "保持隔离",
    button3 = "信任作者",
    OnAccept = function(self, data)
        Guard:ShowRemovalConfirmation(data or self.data)
    end,
    OnAlt = function(self, data)
        data = data or self.data
        local rootID = data and data.rootID
        if rootID then
            local ok, message = Guard:TrustRootPublisher(rootID)
            Print(message or "无法信任该对象发布者", ok and "66ddff" or "ff5555")
        end
    end,
    OnShow = function(self)
        local canRemove = self.data and self.data.removal ~= nil
        if self.button1 then
            if canRemove and self.button1.Enable then self.button1:Enable()
            elseif not canRemove and self.button1.Disable then self.button1:Disable() end
            if self.button1.SetAlpha then self.button1:SetAlpha(canRemove and 1 or 0.35) end
        end
        local canTrust = self.data and self.data.publisher and self.data.publisher.identity ~= nil
        if self.button3 then
            if canTrust and self.button3.Enable then self.button3:Enable()
            elseif not canTrust and self.button3.Disable then self.button3:Disable() end
            if self.button3.SetAlpha then self.button3:SetAlpha(canTrust and 0.72 or 0.35) end
        end
    end,
    OnHide = function(self)
        if self.button1 then
            if self.button1.Enable then self.button1:Enable() end
            if self.button1.SetAlpha then self.button1:SetAlpha(1) end
        end
        if self.button3 and self.button3.SetAlpha then
            if self.button3.Enable then self.button3:Enable() end
            self.button3:SetAlpha(1)
        end
    end,
    timeout = 0,
    whileDead = true,
    hideOnEscape = true,
    preferredIndex = 3,
}

StaticPopupDialogs["RPBOX_ITEM_GUARD_REMOVE"] = {
    text = "确定直接移除“%s”？\n\n%s\n\n该操作无法撤销，并且不会运行道具的摧毁工作流。",
    button1 = "确认移除",
    button2 = CANCEL,
    OnAccept = function(self, data)
        Guard:RemoveInventoryTarget(data or self.data)
    end,
    timeout = 0,
    whileDead = true,
    hideOnEscape = true,
    showAlert = true,
    preferredIndex = 3,
}

local function ResetShortRuntime(bucket, now)
    bucket.shortStarted = now
    bucket.itemAddCalls = 0
    bucket.itemAddCount = 0
    bucket.workflowCalls = 0
    bucket.directAddCalls = 0
    bucket.directAddCount = 0
    bucket.lootDropCalls = 0
    bucket.lootDropCount = 0
    bucket.variableShortWrites = 0
    bucket.variableShortBytes = 0
    bucket.luaCalls = 0
    bucket.luaBytes = 0
end

local function ResetLongRuntime(bucket, now)
    bucket.longStarted = now
    bucket.longWriteCalls = 0
    bucket.longWriteCount = 0
    bucket.variableLongWrites = 0
    bucket.variableLongBytes = 0
end

local function GetRuntimeBucket(rootID)
    local now = GetTime and GetTime() or 0
    local bucket = state.runtime[rootID]
    if not bucket then
        bucket = {
            soundFamilies = {},
            variableKeys = {},
        }
        state.runtime[rootID] = bucket
        ResetShortRuntime(bucket, now)
        ResetLongRuntime(bucket, now)
    else
        if now - bucket.shortStarted > LIMITS.runtimeWindow then
            ResetShortRuntime(bucket, now)
        end
        if now - bucket.longStarted > LIMITS.runtimeLongWindow then
            ResetLongRuntime(bucket, now)
        end
    end
    return bucket
end

local RuntimeTrip

local function SoundFamilyKeys(classification, shouldBeSecured)
    local primary = shouldBeSecured and classification.securedFamily or classification.family
    primary = primary or classification.family
    local keys = { primary }
    local family = classification.family or ""
    if family:find("sound_id_local:", 1, true) == 1 then
        keys[#keys + 1] = family:gsub("^sound_id_local:", "sound_id_self:")
    elseif family:find("sound_id_self:", 1, true) == 1 then
        keys[#keys + 1] = family:gsub("^sound_id_self:", "sound_id_local:")
    elseif family == "sound_music_local" then
        keys[#keys + 1] = "sound_music_self"
    elseif family == "sound_music_self" then
        keys[#keys + 1] = "sound_music_local"
    end
    return keys
end

local function CheckSoundRate(effectID, effectArgs, rootID, shouldBeSecured)
    local rules = ns.ItemGuardSoundRules
    if not rules or not rules.ClassifyEffect then return true, nil end
    local classification = rules.ClassifyEffect(effectID, effectArgs)
    if not classification then return true, nil end
    if not rootID then return true, classification end

    local bucket = GetRuntimeBucket(rootID)
    local families = SoundFamilyKeys(classification, shouldBeSecured)
    if classification.kind == "stop" then
        if classification.effective ~= false then
            for _, family in ipairs(families) do bucket.soundFamilies[family] = nil end
        end
        return true, classification
    end

    local now = GetTime and GetTime() or 0
    local limits = rules.LIMITS or {}
    local shortSeconds = tonumber(limits.shortWindowSeconds) or 5
    local longSeconds = tonumber(limits.longWindowSeconds) or 60
    local family = families[1]
    local sound = bucket.soundFamilies[family]
    if not sound then
        sound = {
            shortStarted = now,
            longStarted = now,
            shortStarts = 0,
            longStarts = 0,
            breaches = 0,
        }
        bucket.soundFamilies[family] = sound
    end
    if now - sound.shortStarted > shortSeconds then
        sound.shortStarted, sound.shortStarts = now, 0
    end
    if now - sound.longStarted > longSeconds then
        sound.longStarted, sound.longStarts = now, 0
    end
    sound.shortStarts = sound.shortStarts + 1
    sound.longStarts = sound.longStarts + 1
    local exceeded = sound.shortStarts > (tonumber(limits.shortStartLimit) or 8)
        or sound.longStarts > (tonumber(limits.longStartLimit) or 30)
    if not exceeded then return true, classification end

    sound.breaches = sound.breaches + 1
    sound.shortStarted, sound.shortStarts = now, 0
    sound.longStarted, sound.longStarts = now, 0
    if sound.breaches >= (tonumber(limits.quarantineAfterBreaches) or 2) then
        return RuntimeTrip(rootID, "重复播放声音超过安全频率"), classification
    end
    Print("已阻止“" .. tostring(rootID) .. "”的高频声音播放；再次超限将隔离道具。", "ffcc66")
    return false, classification
end

local function EstimateRuntimeValueBytes(value)
    local contentRules = ns.ItemGuardContentRules
    if contentRules and contentRules.MeasureValue then
        local measured = contentRules.MeasureValue(value)
        if measured and (measured.tooDeep or measured.tooManyNodes or measured.cyclic) then
            return math.huge
        end
        if measured then return tonumber(measured.bytes) or 0 end
    end
    local valueType = type(value)
    if valueType == "nil" then return 0 end
    if valueType == "string" then return #value end
    if valueType == "number" or valueType == "boolean" then return #tostring(value) end
    return 0
end

local function CheckVariableWrite(rootID, classification, source, varName, varValue)
    local rules = ns.ItemGuardVariableRules
    if not rootID or not rules or not rules.EvaluateRuntime or not classification then return true end

    local bytes = EstimateRuntimeValueBytes(varValue)
    local contentLimits = ns.ItemGuardContentRules and ns.ItemGuardContentRules.LIMITS or {}
    if bytes > (tonumber(contentLimits.VARIABLE_VALUE_BYTES) or (512 * 1024)) then
        return RuntimeTrip(
            rootID,
            "单个运行时变量载荷超过崩溃防护上限 512 KiB",
            "variable_runtime_single_crash_size"
        )
    end
    if classification.interactive or (source ~= "o" and source ~= "c") then return true end

    local bucket = GetRuntimeBucket(rootID)
    bucket.variableShortWrites = bucket.variableShortWrites + 1
    bucket.variableShortBytes = bucket.variableShortBytes + bytes
    bucket.variableLongWrites = bucket.variableLongWrites + 1
    bucket.variableLongBytes = bucket.variableLongBytes + bytes
    local key = tostring(source) .. ":" .. tostring(varName or classification.name or "dynamic")
    bucket.variableKeys[key] = true
    local uniqueKeys = 0
    for _ in pairs(bucket.variableKeys) do uniqueKeys = uniqueKeys + 1 end
    local runtime = rules.EvaluateRuntime({
        shortWrites = bucket.variableShortWrites,
        shortBytes = bucket.variableShortBytes,
        longWrites = bucket.variableLongWrites,
        longBytes = bucket.variableLongBytes,
        uniqueKeys = uniqueKeys,
        singleBytes = bytes,
    })
    if runtime and runtime.blocked then
        local reason = runtime.reasons and runtime.reasons[1] or "持久变量写入超过安全限制"
        return RuntimeTrip(rootID, reason)
    end
    return true
end

RuntimeTrip = function(rootID, reason, findingKind)
    local result = Guard:ScanRoot(rootID) or NewScanResult(rootID)
    result.blocked = true
    result.observationScore = (tonumber(result.observationScore) or 0) + 120
    result.score = (tonumber(result.behaviorScore) or 0)
        + (tonumber(result.amplificationScore) or 0)
        + result.observationScore
        + (tonumber(result.policyScore) or 0)
    AddReason(result, reason)
    result.findings = result.findings or {}
    result.findings[#result.findings + 1] = {
        kind = findingKind or "runtime_quota_exceeded",
        score = 120,
        reason = reason,
        hard = true,
    }
    if Guard:Quarantine(rootID, result, "runtime") then
        local root = GetRootObject(rootID)
        local itemName = root and root.BA and root.BA.NA or rootID
        Print("已隔离“" .. tostring(itemName) .. "”：" .. reason, "ff5555")
    end
    return false
end

local function GetLootDropStats(effectArgs)
    local lootInfo = type(effectArgs) == "table" and effectArgs[1] or nil
    if type(lootInfo) ~= "table" or not lootInfo[4] then return false, 0, 0 end
    local slots = type(lootInfo[3]) == "table" and lootInfo[3] or {}
    local slotCount, totalCount = 0, 0
    for _, slot in pairs(slots) do
        if type(slot) == "table" then
            slotCount = slotCount + 1
            local count = tonumber(slot.count) or 1
            if count > 0 then totalCount = totalCount + count end
        end
    end
    return true, slotCount, totalCount
end

local function CheckEffectRate(effectID, effectArgs, rootID, shouldBeSecured)
    if not rootID then return true end
    local soundAllowed = CheckSoundRate(effectID, effectArgs, rootID, shouldBeSecured)
    if not soundAllowed then return false end
    local bucket = GetRuntimeBucket(rootID)

    if effectID == "item_add" then
        local requestedCount = tonumber(effectArgs[2]) or 1
        bucket.itemAddCalls = bucket.itemAddCalls + 1
        bucket.itemAddCount = bucket.itemAddCount + math.max(0, requestedCount)
        if requestedCount > LIMITS.maxSingleItemAdd
            or bucket.itemAddCalls > LIMITS.runtimeItemAddCalls
            or bucket.itemAddCount > LIMITS.runtimeItemAddCount then
            return RuntimeTrip(rootID, "短时间内添加物品的次数或数量异常")
        end
    elseif effectID == "item_loot" then
        local isDrop, slotCount, totalCount = GetLootDropStats(effectArgs)
        if isDrop then
            bucket.lootDropCalls = bucket.lootDropCalls + 1
            bucket.lootDropCount = bucket.lootDropCount + totalCount
            bucket.longWriteCalls = bucket.longWriteCalls + 1
            bucket.longWriteCount = bucket.longWriteCount + totalCount
            if slotCount > 32
                or totalCount > LIMITS.maxSingleItemAdd
                or bucket.lootDropCalls > LIMITS.runtimeItemAddCalls
                or bucket.lootDropCount > LIMITS.runtimeItemAddCount
                or bucket.longWriteCalls > LIMITS.runtimeLongWriteCalls
                or bucket.longWriteCount > LIMITS.runtimeLongWriteCount then
                return RuntimeTrip(rootID, "地面战利品写入次数或数量异常")
            end
        end
    elseif effectID == "run_workflow" or effectID == "run_item_workflow" then
        bucket.workflowCalls = bucket.workflowCalls + 1
    end
    return true
end

local function CheckLifecycleRespawn(rootID, classID, itemData)
    local context = state.currentDestruction
    local rules = ns.ItemGuardLifecycleRules
    if not rootID or not context or context.rootID ~= rootID
        or not rules or not rules.ClassifyRespawnEffect then return true end
    local count = type(itemData) == "table" and itemData.count or 1
    local classification = rules.ClassifyRespawnEffect(
        context.rootID,
        context.class,
        "item_add",
        { classID, count }
    )
    if classification and classification.selfRespawn and (tonumber(classification.count) or 1) > 0 then
        return RuntimeTrip(rootID, "道具在被摧毁时尝试重新生成自身")
    end
    return true
end

local function CheckDirectAdd(rootID, itemData)
    if not rootID then return true end
    local bucket = GetRuntimeBucket(rootID)
    local count = type(itemData) == "table" and tonumber(itemData.count) or 1
    count = count or 1
    bucket.directAddCalls = bucket.directAddCalls + 1
    bucket.directAddCount = bucket.directAddCount + math.max(0, count)
    bucket.longWriteCalls = bucket.longWriteCalls + 1
    bucket.longWriteCount = bucket.longWriteCount + math.max(0, count)
    if count > LIMITS.maxSingleItemAdd
        or bucket.directAddCalls > LIMITS.runtimeDirectAddCalls
        or bucket.directAddCount > LIMITS.runtimeDirectAddCount
        or bucket.longWriteCalls > LIMITS.runtimeLongWriteCalls
        or bucket.longWriteCount > LIMITS.runtimeLongWriteCount then
        return RuntimeTrip(rootID, "背包写入次数或数量异常")
    end
    return true
end

local function CallWithContext(rootID, destruction, callback, ...)
    local previousRoot = state.currentRoot
    local previousDestruction = state.currentDestruction
    state.currentRoot = rootID or previousRoot
    state.currentDestruction = destruction or previousDestruction
    local results = { pcall(callback, ...) }
    state.currentRoot = previousRoot
    state.currentDestruction = previousDestruction
    if not results[1] then error(results[2], 0) end
    return Unpack(results, 2)
end


local function CallWithRoot(rootID, callback, ...)
    return CallWithContext(rootID, nil, callback, ...)
end

local function CallWithVariableEffect(rootID, classification, callback, ...)
    local previous = state.currentVariableEffect
    state.currentVariableEffect = classification or previous
    local results = { pcall(CallWithRoot, rootID, callback, ...) }
    state.currentVariableEffect = previous
    if not results[1] then error(results[2], 0) end
    return Unpack(results, 2)
end

local function GetDestructionContext(rootID, fullID, scriptID)
    local rules = ns.ItemGuardLifecycleRules
    if not rootID or not rules or not rules.IsDestructionExecution then return nil end
    local class
    if TRP3_API and TRP3_API.extended and TRP3_API.extended.getClass then
        local ok, resolved = pcall(TRP3_API.extended.getClass, fullID)
        if ok and type(resolved) == "table" then class = resolved end
    end
    class = class or GetRootObject(rootID)
    if type(class) ~= "table" or not rules.IsDestructionExecution(rootID, class, scriptID) then return nil end
    return {
        rootID = rootID,
        classID = fullID,
        class = class,
        workflowID = scriptID,
    }
end

local function CheckCrashVariableTable(rootID, vars)
    local rules = ns.ItemGuardContentRules
    if type(vars) ~= "table" or not rules or not rules.AnalyzeVariables then return true end
    local payload = rules.AnalyzeVariables(vars, { rootID = rootID })
    if not payload or not payload.blocked then return true end
    local finding = payload.findings and payload.findings[1]
    local reason = finding and finding.reason or "变量载荷超过崩溃防护上限"
    RuntimeTrip(rootID, reason, finding and finding.kind or "variable_payload_crash_size")
    return false
end

local function CheckCrashPayload(rootID, effectArgs)
    if type(effectArgs) ~= "table" then return true end
    local objectVars = type(effectArgs.object) == "table" and effectArgs.object.vars or nil
    if not CheckCrashVariableTable(rootID, objectVars) then return false end
    local containerVars = type(effectArgs.container) == "table" and effectArgs.container.vars or nil
    if containerVars ~= objectVars and not CheckCrashVariableTable(rootID, containerVars) then return false end
    return true
end

local function CheckDocumentRenderEffect(effectID, effectParameters, effectArgs)
    if effectID ~= "document_show" then return true end
    local rules = ns.ItemGuardContentRules
    if not rules or not TRP3_API.extended.getClass then return true end
    local documentID = CanonicalID(type(effectParameters) == "table" and effectParameters[1])
    local targetRootID = GetRootID(documentID)
    if not documentID or not targetRootID then return true end
    local ok, document = pcall(TRP3_API.extended.getClass, documentID)
    if not ok or type(document) ~= "table" then return true end

    local staticResult = rules.Analyze and rules.Analyze(targetRootID, document, {}) or nil
    local vars = type(effectArgs) == "table" and type(effectArgs.object) == "table"
        and effectArgs.object.vars or nil
    local rendered = rules.AnalyzeRenderedDocument
        and rules.AnalyzeRenderedDocument(document, vars, { rootID = targetRootID }) or nil
    local blocked = staticResult and staticResult.blocked and staticResult or rendered
    if not blocked or not blocked.blocked then return true end
    local reason = blocked.reasons and blocked.reasons[1] or "文档内容超过崩溃防护上限"
    RuntimeTrip(targetRootID, reason, "document_rendered_crash_size")
    return false
end

local function CheckBeforeExecute(fullID, effectArgs)
    local rootID = ResolveRootFromArgs(effectArgs, fullID)
    if not state.enabled or not rootID then return true, rootID end
    local root = GetRootObject(rootID)
    if not IsProtectedRoot(root) then return true, rootID end
    if not CheckCrashPayload(rootID, effectArgs) then return false, rootID end
    local result = Guard:ScanRoot(rootID)
    if result and state.temporaryAllow[rootID] == result.fingerprint then
        return true, rootID
    end
    if result and Guard:IsIgnored(rootID, result.fingerprint) then
        return true, rootID
    end
    local scan = state.scanStatus[rootID]
    if not result or not scan or scan.status == "unscanned"
        or scan.fingerprint ~= result.fingerprint then
        Guard:MarkUnscanned(rootID, "execute")
        if result then Guard:ScanAndApply(rootID, result) else Guard:QueueScan(rootID, "execute") end
        return false, rootID
    end
    if result and result.blocked then
        Guard:Quarantine(rootID, result, "execute")
    end
    if Guard:IsQuarantined(rootID) then
        if root.TY == TRP3_DB.types.ITEM then Guard:ShowQuarantinePopup(rootID) end
        return false, rootID
    end
    return true, rootID
end

local function HasCurrentScan(rootID)
    local scan = rootID and state.scanStatus[rootID]
    local result = rootID and state.scanCache[rootID]
    return scan ~= nil and scan.status ~= "unscanned" and result ~= nil
        and scan.fingerprint == result.fingerprint
end

local function GetPublisherTrust(rootID)
    local root = GetRootObject(rootID)
    if not root or not ns.ItemGuardPublisherWhitelist
        or not ns.ItemGuardPublisherWhitelist.MatchRoot then return nil end
    if ns.ItemGuardBlacklist and ns.ItemGuardBlacklist.MatchRoot
        and ns.ItemGuardBlacklist.MatchRoot(rootID, root) then return nil end
    return ns.ItemGuardPublisherWhitelist.MatchRoot(rootID, root)
end

local function SnapshotSharedLibraries()
    local snapshot = {}
    for name, library in pairs({ string = string, table = table, math = math }) do
        local values = {}
        for key, value in pairs(library) do values[key] = value end
        snapshot[name] = { library = library, values = values, metatable = getmetatable(library) }
    end
    return snapshot
end

local function RestoreSharedLibraries()
    local changed = false
    for _, entry in pairs(state.luaLibraryBaseline or {}) do
        local library, values = entry.library, entry.values
        for key in pairs(library) do
            if values[key] == nil then library[key] = nil; changed = true end
        end
        for key, value in pairs(values) do
            if library[key] ~= value then library[key] = value; changed = true end
        end
        if getmetatable(library) ~= entry.metatable then
            pcall(setmetatable, library, entry.metatable)
            changed = true
        end
    end
    return changed
end

local function RestoreGuardHookIntegrity()
    local changed = false
    if TRP3_API.script.executeClassScript ~= state.wrappedExecute then
        TRP3_API.script.executeClassScript = state.wrappedExecute
        changed = true
    end
    if TRP3_API.script.playEffect ~= state.wrappedPlayEffect then
        TRP3_API.script.playEffect = state.wrappedPlayEffect
        changed = true
    end
    if TRP3_API.script.setVar ~= state.wrappedSetVar then
        TRP3_API.script.setVar = state.wrappedSetVar
        changed = true
    end
    if TRP3_API.script.runLuaScriptEffect ~= state.wrappedRunLuaScriptEffect then
        TRP3_API.script.runLuaScriptEffect = state.wrappedRunLuaScriptEffect
        changed = true
    end
    if TRP3_API.inventory.addItem ~= state.wrappedAddItem then
        TRP3_API.inventory.addItem = state.wrappedAddItem
        changed = true
    end
    if TRP3_API.extended.registerObject ~= state.wrappedRegisterObject then
        TRP3_API.extended.registerObject = state.wrappedRegisterObject
        changed = true
    end
    local auras = TRP3_API.extended.auras
    if auras and state.wrappedAuraApply and auras.apply ~= state.wrappedAuraApply then
        auras.apply = state.wrappedAuraApply
        changed = true
    end
    if auras and state.wrappedAuraCancel and auras.cancel ~= state.wrappedAuraCancel then
        auras.cancel = state.wrappedAuraCancel
        changed = true
    end
    if auras and state.wrappedAuraSetVariable and auras.setVariable ~= state.wrappedAuraSetVariable then
        auras.setVariable = state.wrappedAuraSetVariable
        changed = true
    end
    local documents = TRP3_API.extended.document
    if documents and state.wrappedDocumentShow and documents.showDocument ~= state.wrappedDocumentShow then
        documents.showDocument = state.wrappedDocumentShow
        changed = true
    end
    if documents and state.wrappedDocumentShowClass
        and documents.showDocumentClass ~= state.wrappedDocumentShowClass then
        documents.showDocumentClass = state.wrappedDocumentShowClass
        changed = true
    end
    return changed
end

local function InstallHooks()
    if state.installed or not IsReady() then return false end

    state.original.executeClassScript = TRP3_API.script.executeClassScript
    state.original.playEffect = TRP3_API.script.playEffect
    state.original.setVar = TRP3_API.script.setVar
    state.original.runLuaScriptEffect = TRP3_API.script.runLuaScriptEffect
    state.original.addItem = TRP3_API.inventory.addItem
    state.original.registerObject = TRP3_API.extended.registerObject
    local auras = TRP3_API.extended.auras
    state.original.auraApply = auras and auras.apply or nil
    state.original.auraCancel = auras and auras.cancel or nil
    state.original.auraRemove = auras and auras.remove or nil
    state.original.auraSetVariable = auras and auras.setVariable or nil
    local documents = TRP3_API.extended.document
    state.original.documentShow = documents and documents.showDocument or nil
    state.original.documentShowClass = documents and documents.showDocumentClass or nil
    state.luaLibraryBaseline = SnapshotSharedLibraries()

    state.wrappedExecute = function(scriptID, classScripts, effectArgs, fullID)
        local allowed, rootID = CheckBeforeExecute(fullID, effectArgs)
        if not allowed then return 0 end
        local destruction = GetDestructionContext(rootID, fullID, scriptID)
        return CallWithContext(
            rootID,
            destruction,
            state.original.executeClassScript,
            scriptID,
            classScripts,
            effectArgs,
            fullID
        )
    end

    state.wrappedPlayEffect = function(effectID, shouldBeSecured, effectArgs, ...)
        if not state.enabled then
            return state.original.playEffect(effectID, shouldBeSecured, effectArgs, ...)
        end
        local rootID = ResolveRootFromArgs(effectArgs) or state.currentRoot
        local effectParameters = { ... }
        local soundClassification
        if ns.ItemGuardSoundRules and ns.ItemGuardSoundRules.ClassifyEffect then
            soundClassification = ns.ItemGuardSoundRules.ClassifyEffect(effectID, effectParameters)
        end
        local variableClassification
        if ns.ItemGuardVariableRules and ns.ItemGuardVariableRules.ClassifyEffect then
            variableClassification = ns.ItemGuardVariableRules.ClassifyEffect(effectID, effectParameters)
        end
        if not CheckDocumentRenderEffect(effectID, effectParameters, effectArgs) then return 0 end
        if rootID and not HasCurrentScan(rootID) and not (Guard:IsQuarantined(rootID)
            and soundClassification and soundClassification.kind == "stop") then
            local allowed = CheckBeforeExecute(rootID, effectArgs)
            if not allowed then return 0 end
        end
        local result = rootID and (state.scanCache[rootID] or Guard:ScanRoot(rootID)) or nil
        if result and Guard:IsIgnored(rootID, result.fingerprint) then
            return CallWithRoot(rootID, state.original.playEffect, effectID, shouldBeSecured, effectArgs, ...)
        end
        if rootID and Guard:IsQuarantined(rootID) then
            -- A stop effect must remain available so quarantine cannot leave an
            -- already playing malicious sound stuck on the client.
            if soundClassification and soundClassification.kind == "stop" then
                CheckSoundRate(effectID, effectParameters, rootID, shouldBeSecured)
                return CallWithRoot(rootID, state.original.playEffect, effectID, shouldBeSecured, effectArgs, ...)
            end
            return 0
        end
        if not CheckEffectRate(effectID, effectParameters, rootID, shouldBeSecured) then return 0 end
        return CallWithVariableEffect(
            rootID,
            variableClassification,
            state.original.playEffect,
            effectID,
            shouldBeSecured,
            effectArgs,
            ...
        )
    end

    state.wrappedSetVar = function(effectArgs, source, operationType, varName, varValue)
        local rootID = ResolveRootFromArgs(effectArgs) or state.currentRoot
        local classification = state.currentVariableEffect or {
            interactive = false,
            name = tostring(varName or "dynamic"),
            persistent = source == "o" or source == "c",
        }
        if state.enabled and rootID then
            if Guard:IsQuarantined(rootID)
                or not CheckVariableWrite(rootID, classification, source, varName, varValue) then
                return nil
            end
        end
        return state.original.setVar(effectArgs, source, operationType, varName, varValue)
    end

    state.wrappedAddItem = function(container, classID, itemData, dropIfFull, toSlot)
        local rootID = state.currentRoot
        local payloadRootID = rootID or GetRootID(classID)
        if state.enabled and type(itemData) == "table"
            and not CheckCrashVariableTable(payloadRootID, itemData.vars) then
            return 1
        end
        if state.enabled and rootID then
            local result = state.scanCache[rootID] or Guard:ScanRoot(rootID)
            if not (result and Guard:IsIgnored(rootID, result.fingerprint)) then
                if Guard:IsQuarantined(rootID)
                    or not CheckLifecycleRespawn(rootID, classID, itemData)
                    or not CheckDirectAdd(rootID, itemData) then
                    return 1
                end
            end
        end
        return state.original.addItem(container, classID, itemData, dropIfFull, toSlot)
    end

    state.wrappedRunLuaScriptEffect = function(code, effectArgs, secured)
        if not state.enabled then
            return state.original.runLuaScriptEffect(code, effectArgs, secured)
        end
        local rootID = ResolveRootFromArgs(effectArgs) or state.currentRoot
        local runtimeRootID = rootID or "<unknown>"
        if rootID and Guard:IsQuarantined(rootID) then return 0 end

        local rules = ns.ItemGuardLuaRules
        local publisherTrust = rootID and GetPublisherTrust(rootID) or nil
        local analysis = rules and rules.AnalyzeCode and rules.AnalyzeCode(code, {
            rootID = runtimeRootID,
            trustedPublisher = publisherTrust ~= nil,
        }) or nil
        if analysis and analysis.blocked then
            local finding = analysis.findings and analysis.findings[1]
            RuntimeTrip(
                rootID,
                analysis.reasons and analysis.reasons[1] or "Lua Script Effect 命中高风险规则",
                finding and finding.kind or "lua_runtime_policy"
            )
            return 0
        end

        local bucket = GetRuntimeBucket(runtimeRootID)
        local luaLimits = rules and rules.LIMITS or {}
        local sourceBytes = type(code) == "string" and #code or 0
        bucket.luaCalls = (bucket.luaCalls or 0) + 1
        bucket.luaBytes = (bucket.luaBytes or 0) + sourceBytes
        if bucket.luaCalls > (tonumber(luaLimits.RUNTIME_CALLS) or 40)
            or bucket.luaBytes > (tonumber(luaLimits.RUNTIME_BYTES) or (2 * 1024 * 1024)) then
            RuntimeTrip(rootID, "短时间内 Lua 编译次数或源码字节超过限制", "lua_runtime_rate")
            return 0
        end

        local depth = (state.luaDepth[runtimeRootID] or 0) + 1
        if depth > (tonumber(luaLimits.RUNTIME_DEPTH) or 12) then
            RuntimeTrip(rootID, "Lua Script Effect 递归深度超过限制", "lua_runtime_depth")
            return 0
        end
        state.luaDepth[runtimeRootID] = depth
        local previousFingerprint = rootID and state.scanStatus[rootID]
            and state.scanStatus[rootID].fingerprint or nil
        local results = { pcall(
            CallWithRoot,
            rootID,
            state.original.runLuaScriptEffect,
            code,
            effectArgs,
            secured
        ) }
        state.luaDepth[runtimeRootID] = depth > 1 and depth - 1 or nil

        local librariesChanged = RestoreSharedLibraries()
        local hooksChanged = RestoreGuardHookIntegrity()
        if librariesChanged or hooksChanged then
            RuntimeTrip(
                rootID,
                librariesChanged and "Lua 修改了共享 string/table/math 库，已恢复并隔离"
                    or "Lua 覆写了 RPBox/TRP3 防护入口，已恢复并隔离",
                librariesChanged and "lua_shared_library_runtime_tamper" or "lua_guard_hook_runtime_tamper"
            )
            return 0
        end
        if rootID and not CheckCrashPayload(rootID, effectArgs) then return 0 end

        if rootID and previousFingerprint then
            local rescanned = Guard:ScanRoot(rootID)
            if rescanned and rescanned.fingerprint ~= previousFingerprint then
                Guard:MarkUnscanned(rootID, "runtime-lua-mutation")
                Guard:ScanAndApply(rootID, rescanned)
                if Guard:IsQuarantined(rootID) then return 0 end
            end
        end
        if not results[1] then error(results[2], 0) end
        return Unpack(results, 2)
    end

    if state.original.registerObject then
        state.wrappedRegisterObject = function(objectFullID, object, count, registerTo)
            local result = state.original.registerObject(objectFullID, object, count, registerTo)
            if state.enabled and not state.mutating then
                Guard:QueueScan(objectFullID, "received")
            end
            return result
        end
    end

    if state.original.auraApply then
        state.wrappedAuraApply = function(auraID, mergeMode)
            local canonicalAuraID = CanonicalID(auraID)
            local rootID = GetRootID(canonicalAuraID)
            local cancellation = state.currentAuraCancellation
            local watched = canonicalAuraID and state.auraCancellationWatch[canonicalAuraID] or nil
            if watched and GetTime() > watched.expiresAt then
                state.auraCancellationWatch[canonicalAuraID] = nil
                watched = nil
            end
            if not cancellation or canonicalAuraID ~= cancellation.auraID then
                cancellation = watched
            end
            if state.enabled and cancellation and canonicalAuraID == cancellation.auraID then
                RuntimeTrip(
                    cancellation.rootID,
                    "光环在取消过程中尝试重新施加自身",
                    "aura_cancel_self_reapply_runtime"
                )
                return nil
            end
            local allowed = CheckBeforeExecute(canonicalAuraID, { classID = canonicalAuraID })
            if not allowed then return nil end
            return state.original.auraApply(auraID, mergeMode)
        end
    end

    if state.original.auraCancel then
        state.wrappedAuraCancel = function(auraID)
            local canonicalAuraID = CanonicalID(auraID)
            local rootID = GetRootID(canonicalAuraID)
            if state.enabled and rootID and Guard:IsQuarantined(rootID) and state.original.auraRemove then
                return state.original.auraRemove(auraID)
            end
            local previous = state.currentAuraCancellation
            local cancellation = { auraID = canonicalAuraID, rootID = rootID }
            local watchToken = {}
            local priorWatch = canonicalAuraID and state.auraCancellationWatch[canonicalAuraID] or nil
            if canonicalAuraID then
                cancellation.token = watchToken
                cancellation.expiresAt = GetTime() + LIMITS.auraCancelWatchSeconds
                state.auraCancellationWatch[canonicalAuraID] = cancellation
            end
            state.currentAuraCancellation = cancellation
            local results = { pcall(state.original.auraCancel, auraID) }
            state.currentAuraCancellation = previous
            if canonicalAuraID then
                if results[1] and results[2] == true then
                    C_Timer.After(LIMITS.auraCancelWatchSeconds, function()
                        local current = state.auraCancellationWatch[canonicalAuraID]
                        if current and current.token == watchToken then
                            state.auraCancellationWatch[canonicalAuraID] = nil
                        end
                    end)
                elseif state.auraCancellationWatch[canonicalAuraID]
                    and state.auraCancellationWatch[canonicalAuraID].token == watchToken then
                    state.auraCancellationWatch[canonicalAuraID] = priorWatch
                        and priorWatch.expiresAt >= GetTime() and priorWatch or nil
                end
            end
            if not results[1] then error(results[2], 0) end
            return Unpack(results, 2)
        end
    end

    if state.original.auraSetVariable then
        state.wrappedAuraSetVariable = function(auraID, operationType, varName, varValue)
            local rootID = GetRootID(CanonicalID(auraID))
            if state.enabled and not CheckCrashVariableTable(rootID, {
                [tostring(varName or "dynamic")] = varValue,
            }) then
                return nil
            end
            local allowed = CheckBeforeExecute(auraID, { classID = auraID })
            if not allowed then return nil end
            return state.original.auraSetVariable(auraID, operationType, varName, varValue)
        end
    end

    if state.original.documentShow then
        state.wrappedDocumentShow = function(documentID, parentArgs)
            local rules = ns.ItemGuardContentRules
            if state.enabled and rules and rules.AnalyzeRenderedDocument
                and TRP3_API.extended.getClass then
                local ok, document = pcall(TRP3_API.extended.getClass, documentID)
                local vars = type(parentArgs) == "table" and type(parentArgs.object) == "table"
                    and parentArgs.object.vars or nil
                local rendered = ok and rules.AnalyzeRenderedDocument(
                    document,
                    vars,
                    { rootID = GetRootID(CanonicalID(documentID)) }
                ) or nil
                if rendered and rendered.blocked then
                    local rootID = GetRootID(CanonicalID(documentID))
                    local reason = rendered.reasons and rendered.reasons[1]
                        or "变量展开后的文档超过崩溃防护上限"
                    RuntimeTrip(rootID, reason, "document_rendered_crash_size")
                    return nil
                end
            end
            local allowed = CheckBeforeExecute(documentID, {
                classID = documentID,
                object = type(parentArgs) == "table" and parentArgs.object or nil,
            })
            if not allowed then return nil end
            return state.original.documentShow(documentID, parentArgs)
        end
    end

    if state.original.documentShowClass then
        state.wrappedDocumentShowClass = function(document, documentID, parentArgs)
            local rules = ns.ItemGuardContentRules
            if state.enabled and rules and rules.Analyze then
                local rootID = GetRootID(CanonicalID(documentID)) or "document-preview"
                local parentVars = type(parentArgs) == "table"
                    and type(parentArgs.object) == "table" and parentArgs.object.vars or nil
                local variableResult = rules.AnalyzeVariables
                    and rules.AnalyzeVariables(parentVars, { rootID = rootID }) or nil
                if variableResult and variableResult.blocked then
                    local reason = variableResult.reasons and variableResult.reasons[1]
                        or "变量载荷超过崩溃防护上限"
                    if rootID ~= "document-preview" then
                        RuntimeTrip(rootID, reason, "variable_payload_crash_size")
                    else
                        Print("已阻止打开过量文档变量：" .. reason, "ff5555")
                    end
                    return nil
                end
                local rendered = rules.AnalyzeRenderedDocument
                    and rules.AnalyzeRenderedDocument(document, parentVars, { rootID = rootID }) or nil
                if rendered and rendered.blocked then
                    local reason = rendered.reasons and rendered.reasons[1]
                        or "变量展开后的文档超过崩溃防护上限"
                    if rootID ~= "document-preview" then
                        RuntimeTrip(rootID, reason, "document_rendered_crash_size")
                    else
                        Print("已阻止打开变量展开过量的文档：" .. reason, "ff5555")
                    end
                    return nil
                end
                local result = rules.Analyze(rootID, document, {})
                if result and result.blocked then
                    local reason = result.reasons and result.reasons[1] or "文档内容超过崩溃防护上限"
                    if rootID ~= "document-preview" then
                        RuntimeTrip(rootID, reason, "document_content_crash_size")
                    else
                        Print("已阻止打开过量文档：" .. reason, "ff5555")
                    end
                    return nil
                end
            end
            return state.original.documentShowClass(document, documentID, parentArgs)
        end
    end

    TRP3_API.script.executeClassScript = state.wrappedExecute
    TRP3_API.script.playEffect = state.wrappedPlayEffect
    TRP3_API.script.setVar = state.wrappedSetVar
    TRP3_API.script.runLuaScriptEffect = state.wrappedRunLuaScriptEffect
    TRP3_API.inventory.addItem = state.wrappedAddItem
    if state.wrappedRegisterObject then TRP3_API.extended.registerObject = state.wrappedRegisterObject end
    if auras and state.wrappedAuraApply then auras.apply = state.wrappedAuraApply end
    if auras and state.wrappedAuraCancel then auras.cancel = state.wrappedAuraCancel end
    if auras and state.wrappedAuraSetVariable then auras.setVariable = state.wrappedAuraSetVariable end
    if documents and state.wrappedDocumentShow then documents.showDocument = state.wrappedDocumentShow end
    if documents and state.wrappedDocumentShowClass then
        documents.showDocumentClass = state.wrappedDocumentShowClass
    end
    if TRP3_API.script.clearAllCompilations then
        TRP3_API.script.clearAllCompilations()
    end
    state.installed = true
    return true
end

local function RestoreHooks()
    if not state.installed then return end
    if TRP3_API and TRP3_API.script then
        if TRP3_API.script.executeClassScript == state.wrappedExecute then
            TRP3_API.script.executeClassScript = state.original.executeClassScript
        end
        if TRP3_API.script.playEffect == state.wrappedPlayEffect then
            TRP3_API.script.playEffect = state.original.playEffect
        end
        if TRP3_API.script.setVar == state.wrappedSetVar then
            TRP3_API.script.setVar = state.original.setVar
        end
        if TRP3_API.script.runLuaScriptEffect == state.wrappedRunLuaScriptEffect then
            TRP3_API.script.runLuaScriptEffect = state.original.runLuaScriptEffect
        end
    end
    if TRP3_API and TRP3_API.inventory and TRP3_API.inventory.addItem == state.wrappedAddItem then
        TRP3_API.inventory.addItem = state.original.addItem
    end
    if TRP3_API and TRP3_API.extended then
        if TRP3_API.extended.registerObject == state.wrappedRegisterObject then
            TRP3_API.extended.registerObject = state.original.registerObject
        end
        local auras = TRP3_API.extended.auras
        if auras and auras.apply == state.wrappedAuraApply then auras.apply = state.original.auraApply end
        if auras and auras.cancel == state.wrappedAuraCancel then auras.cancel = state.original.auraCancel end
        if auras and auras.setVariable == state.wrappedAuraSetVariable then
            auras.setVariable = state.original.auraSetVariable
        end
        local documents = TRP3_API.extended.document
        if documents and documents.showDocument == state.wrappedDocumentShow then
            documents.showDocument = state.original.documentShow
        end
        if documents and documents.showDocumentClass == state.wrappedDocumentShowClass then
            documents.showDocumentClass = state.original.documentShowClass
        end
    end
    if TRP3_API and TRP3_API.script and TRP3_API.script.clearAllCompilations then
        TRP3_API.script.clearAllCompilations()
    end
    state.currentRoot = nil
    state.currentDestruction = nil
    state.currentVariableEffect = nil
    state.currentAuraCancellation = nil
    state.auraCancellationWatch = {}
    state.luaDepth = {}
    state.luaLibraryBaseline = nil
    state.installed = false
end

local function RegisterCallbacks()
    if state.callbacksRegistered or not IsReady() or not TRP3_API.RegisterCallback then return end

    if TRP3_Extended.Events.ON_SLOT_USE then
        TRP3_API.RegisterCallback(TRP3_Extended, TRP3_Extended.Events.ON_SLOT_USE, function(_, slotButton, containerFrame)
            if not state.enabled or not slotButton or not slotButton.info then return end
            local rootID = GetRootID(slotButton.info.id)
            if Guard:IsQuarantined(rootID) then
                Guard:ShowQuarantinePopup(rootID, slotButton, containerFrame)
            end
        end)
    end

    if TRP3_Extended.Events.ON_OBJECT_UPDATED then
        TRP3_API.RegisterCallback(TRP3_Extended, TRP3_Extended.Events.ON_OBJECT_UPDATED, function(_, objectID)
            if state.mutating or not state.enabled then return end
            local rootID = GetRootID(objectID)
            if rootID then
                Guard:QueueScan(rootID, "updated")
            else
                -- Several TRP3 receive paths emit ON_OBJECT_UPDATED without an
                -- ID. registerObject is hooked above, while this fallback also
                -- catches replacements made by integrations that bypass it.
                local roots = {}
                local function Collect(store)
                    if type(store) ~= "table" then return end
                    for candidateID, root in pairs(store) do
                        if type(candidateID) == "string" and not candidateID:find(" ", 1, true)
                            and IsProtectedRoot(root) then
                            roots[candidateID] = root
                        end
                    end
                end
                Collect(TRP3_Tools_DB)
                Collect(TRP3_Exchange_DB)
                Collect(TRP3_DB and TRP3_DB.global)
                for candidateID, root in pairs(roots) do
                    if state.knownRoots[candidateID] ~= root then
                        Guard:QueueScan(candidateID, "received")
                    end
                end
            end
        end)
    end

    state.callbacksRegistered = true
end

local function RegisterBlacklistCallback()
    if state.blacklistCallbackRegistered or not ns.ItemGuardBlacklist then return end
    if ns.ItemGuardBlacklist.Initialize then ns.ItemGuardBlacklist.Initialize() end
    if ns.ItemGuardBlacklist.RegisterOnChanged then
        ns.ItemGuardBlacklist.RegisterOnChanged(function()
            state.scanCache = {}
            if state.enabled then Guard:ScanAll() end
        end)
    end
    state.blacklistCallbackRegistered = true
end

local function RegisterPublisherCallback()
    if state.publisherCallbackRegistered or not ns.ItemGuardPublisherWhitelist then return end
    if ns.ItemGuardPublisherWhitelist.Initialize then ns.ItemGuardPublisherWhitelist.Initialize() end
    if ns.ItemGuardPublisherWhitelist.RegisterOnChanged then
        ns.ItemGuardPublisherWhitelist.RegisterOnChanged(function()
            state.scanCache = {}
            if state.enabled then Guard:ScanAll() end
        end)
    end
    state.publisherCallbackRegistered = true
end

function Guard:ScanAndApply(rootID, precomputedResult)
    if not state.enabled then return nil end
    local canonicalRootID = GetRootID(rootID)
    if not canonicalRootID then return nil end
    rootID = canonicalRootID
    state.pendingScans[rootID] = nil
    state.temporaryAllow[rootID] = nil
    local root = GetRootObject(rootID)
    if not IsProtectedRoot(root) then
        ReleasePendingProtection(rootID)
        state.scanStatus[rootID] = nil
        state.knownRoots[rootID] = nil
        return nil
    end

    local result = precomputedResult or self:ScanRoot(rootID, root)
    if not result then return nil end
    state.scanCache[rootID] = result
    local scanWasReplacement = state.scanStatus[rootID] and state.scanStatus[rootID].replaced
    ReleasePendingProtection(rootID)
    state.knownRoots[rootID] = root
    local database = EnsureDatabase()
    local existingFinding = database.findings[rootID]
    local preservedRiskSource
    if not result.blocked
        and existingFinding
        and existingFinding.fingerprint == result.fingerprint
        and (existingFinding.source == "runtime" or existingFinding.source == "manual") then
        result.blocked = true
        preservedRiskSource = existingFinding.source
        for _, reason in ipairs(existingFinding.reasons or {}) do AddReason(result, reason) end
    end
    local finding
    if result.blocked or result.advisory then
        finding = self:UpdateFinding(rootID, result, preservedRiskSource or "scan")
    end

    if result.blocked and self:IsIgnored(rootID, result.fingerprint) then
        local record = database.quarantined[rootID]
        if record then
            if root.TY == TRP3_DB.types.ITEM and IsVisuallyIsolated(root) and not scanWasReplacement then
                self:RestoreVisualIsolation(rootID, record)
            end
            database.quarantined[rootID] = nil
        end
        state.scanStatus[rootID] = { status = "ignored", fingerprint = result.fingerprint }
        if finding then finding.disposition = "ignored" end
        NotifyChanged()
    elseif result.blocked then
        self:Quarantine(rootID, result, preservedRiskSource or "scan")
    elseif database.quarantined[rootID] then
        local record = database.quarantined[rootID]
        if record.source ~= "runtime" and record.source ~= "manual" then
            if root.TY == TRP3_DB.types.ITEM and IsVisuallyIsolated(root) and not scanWasReplacement then
                self:RestoreVisualIsolation(rootID, record)
            end
            database.quarantined[rootID] = nil
            if result.advisory then
                finding = finding or self:UpdateFinding(rootID, result, "scan")
                if finding then finding.disposition = "observed" end
            else
                self:RemoveFinding(rootID)
            end
            state.scanStatus[rootID] = { status = "trusted", fingerprint = result.fingerprint }
            NotifyChanged()
        else
            if root.TY == TRP3_DB.types.ITEM then self:ApplyVisualIsolation(rootID, record) end
            state.scanStatus[rootID] = { status = "blocked", fingerprint = record.fingerprint }
        end
    else
        if result.advisory then
            finding = finding or self:UpdateFinding(rootID, result, "scan")
            if finding then finding.disposition = "observed" end
        else
            self:RemoveFinding(rootID)
        end
        state.scanStatus[rootID] = { status = "trusted", fingerprint = result.fingerprint }
        NotifyChanged()
    end
    return result
end

local function CollectRootIDs()
    local roots = {}
    local function Collect(store)
        if type(store) ~= "table" then return end
        for rootID, root in pairs(store) do
            if type(rootID) == "string" and not rootID:find(" ", 1, true) and IsProtectedRoot(root) then
                roots[rootID] = true
            end
        end
    end
    Collect(TRP3_Tools_DB)
    Collect(TRP3_Exchange_DB)
    Collect(TRP3_DB and TRP3_DB.global)

    local list = {}
    for rootID in pairs(roots) do list[#list + 1] = rootID end
    local quarantined = EnsureDatabase().quarantined
    table.sort(list, function(left, right)
        local leftPriority = quarantined[left] and 0 or 1
        local rightPriority = quarantined[right] and 0 or 1
        if leftPriority ~= rightPriority then return leftPriority < rightPriority end
        return left < right
    end)
    return list
end

function Guard:ScanAll()
    if not state.enabled or not IsReady() then return end
    state.scanGeneration = state.scanGeneration + 1
    local generation = state.scanGeneration
    local rootIDs = CollectRootIDs()
    local index, blocked = 1, 0

    for _, rootID in ipairs(rootIDs) do self:MarkUnscanned(rootID, "scan") end

    local function ProcessBatch()
        if not state.enabled or generation ~= state.scanGeneration then return end
        local batchEnd = math.min(index + 9, #rootIDs)
        while index <= batchEnd do
            Guard:ScanAndApply(rootIDs[index])
            if Guard:IsQuarantined(rootIDs[index]) then blocked = blocked + 1 end
            index = index + 1
        end
        if index <= #rootIDs then
            C_Timer.After(0, ProcessBatch)
        else
            Print("扫描完成：" .. tostring(#rootIDs) .. " 个对象，隔离 " .. tostring(blocked) .. " 个。")
            NotifyChanged()
        end
    end

    C_Timer.After(0, ProcessBatch)
end

local function RestoreAllVisuals()
    local pendingIDs = {}
    for rootID in pairs(state.pendingProtection) do pendingIDs[#pendingIDs + 1] = rootID end
    for _, rootID in ipairs(pendingIDs) do ReleasePendingProtection(rootID) end
    for rootID, record in pairs(EnsureDatabase().quarantined) do
        local root = GetRootObject(rootID)
        local scan = state.scanStatus[rootID]
        if root and root.TY == TRP3_DB.types.ITEM and IsVisuallyIsolated(root)
            and not (scan and scan.replaced) then
            Guard:RestoreVisualIsolation(rootID, record)
        end
    end
end

local function TryStart(generation, attempt)
    if generation ~= state.retryGeneration then return end
    if not state.enabled then
        if IsReady() then RestoreAllVisuals() end
        return
    end
    if IsReady() then
        InstallHooks()
        RegisterCallbacks()
        RegisterBlacklistCallback()
        RegisterPublisherCallback()
        Guard:ScanAll()
        return
    end
    if attempt < 20 then
        C_Timer.After(1, function() TryStart(generation, attempt + 1) end)
    else
        Print("未检测到 Total RP 3 Extended，对象防护保持待命。", "ffcc00")
    end
end

function Guard:SetEnabled(enabled, silent)
    enabled = enabled and true or false
    EnsureDatabase()
    RPBox_Config = RPBox_Config or {}
    RPBox_Config.itemGuardEnabled = enabled
    state.enabled = enabled
    state.retryGeneration = state.retryGeneration + 1
    state.scanGeneration = state.scanGeneration + 1

    if enabled then
        local generation = state.retryGeneration
        TryStart(generation, 0)
        if not silent then Print("TRP3 对象与光环防护已开启。") end
    else
        RestoreHooks()
        if IsReady() then RestoreAllVisuals() end
        if not silent then Print("TRP3 对象与光环防护已关闭；隔离记录仍保留。", "ffcc00") end
    end
end

function Guard:IsEnabled()
    return state.enabled
end

function Guard:Initialize()
    EnsureDatabase()
    if ns.ItemGuardBlacklist and ns.ItemGuardBlacklist.Initialize then
        ns.ItemGuardBlacklist.Initialize()
    end
    if ns.ItemGuardPublisherWhitelist and ns.ItemGuardPublisherWhitelist.Initialize then
        ns.ItemGuardPublisherWhitelist.Initialize()
    end
    self:SetEnabled(not RPBox_Config or RPBox_Config.itemGuardEnabled ~= false, true)
end

function Guard:HandleSlash(subcommand, parameter)
    subcommand = tostring(subcommand or "status"):lower()
    parameter = tostring(parameter or "")
    if subcommand == "on" then
        self:SetEnabled(true)
    elseif subcommand == "off" then
        self:SetEnabled(false)
    elseif subcommand == "scan" then
        self:ScanAll()
    elseif subcommand == "list" then
        local entries = self:GetRiskEntries()
        for _, entry in ipairs(entries) do
            Print(tostring(entry.itemName) .. "  [" .. entry.rootID .. "]  " .. entry.status, "ffcc66")
        end
        Print("当前防护记录：" .. tostring(#entries) .. " 个。")
    elseif subcommand == "allow" and parameter ~= "" then
        if self:ReleaseQuarantine(parameter) then
            Print("已临时解除隔离；下次扫描仍会重新隔离：" .. parameter, "ffcc66")
        else
            Print("未找到隔离记录：" .. parameter, "ff5555")
        end
    elseif subcommand == "ignore" and parameter ~= "" then
        if self:SetIgnored(parameter, true) then
            Print("已加入忽略清单：" .. parameter, "ffcc66")
        end
    elseif subcommand == "unignore" and parameter ~= "" then
        if self:SetIgnored(parameter, false) then
            Print("已移出忽略清单并重新扫描：" .. parameter, "ffcc66")
        end
    elseif subcommand == "trust" and parameter ~= "" then
        local whitelist = ns.ItemGuardPublisherWhitelist
        local ok, message
        if whitelist and whitelist.AddUser then
            ok, message = whitelist.AddUser(parameter, "用户手动信任")
        else
            ok, message = false, "发布者白名单模块未就绪"
        end
        Print(message or (ok and "已信任发布者" or "无法信任发布者"), ok and "00ff00" or "ff5555")
    elseif subcommand == "untrust" and parameter ~= "" then
        local whitelist = ns.ItemGuardPublisherWhitelist
        local ok, message
        if whitelist and whitelist.RemoveUser then
            ok, message = whitelist.RemoveUser(parameter)
        else
            ok, message = false, "发布者白名单模块未就绪"
        end
        Print(message or (ok and "已取消信任发布者" or "无法取消信任"), ok and "ffcc66" or "ff5555")
    elseif subcommand == "trustlist" then
        local whitelist = ns.ItemGuardPublisherWhitelist
        local entries = whitelist and whitelist.GetEntries and whitelist.GetEntries() or {}
        for _, entry in ipairs(entries) do
            Print(tostring(entry.identity) .. "  [" .. tostring(entry.source) .. "]", "66ddff")
        end
        Print("当前发布者白名单：" .. tostring(#entries) .. " 个。")
    else
        Print("状态：" .. (state.enabled and "已开启" or "已关闭")
            .. "；运行时拦截：" .. (state.installed and "已安装" or "未安装"))
        Print("命令：/rpbox guard on/off/scan/list/allow/ignore/unignore <根ID>")
        Print("发布者：/rpbox guard trust/untrust 名字-服务器；trustlist")
    end
end
