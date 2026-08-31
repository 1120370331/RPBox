-- Static protection rules for TRP3 Extended auras.
--
-- Aura cancellation runs LI.OC before TRP3 invalidates the active aura.  A
-- cancellation workflow that applies the same aura can therefore make the
-- aura appear impossible to remove.  This module follows the reachable
-- standard workflow graph and blocks that pattern without reading Lua Script
-- Effect payloads.

local ADDON_NAME, ns = ...

local Rules = {}
ns.ItemGuardAuraRules = Rules

Rules.RULE_VERSION = 1

local CHILD_GROUPS = { "IN", "QE", "ST" }

local function CanonicalID(value)
    if type(value) == "string" or type(value) == "number" then
        local id = tostring(value)
        if id ~= "" then return id end
    end
    return nil
end

local function SortedKeys(value)
    local keys = {}
    for key in pairs(value or {}) do keys[#keys + 1] = key end
    table.sort(keys, function(left, right)
        local leftType, rightType = type(left), type(right)
        if leftType ~= rightType then return leftType < rightType end
        if leftType == "number" or leftType == "string" then return left < right end
        return tostring(left) < tostring(right)
    end)
    return keys
end

local function StableNumber(value)
    if value ~= value then return "nan" end
    if value == math.huge then return "+inf" end
    if value == -math.huge then return "-inf" end
    if value == math.floor(value) then return tostring(value) end
    return string.format("%.17g", value)
end

local function StableSerialize(value, stack)
    local valueType = type(value)
    if valueType == "nil" then return "n" end
    if valueType == "boolean" then return value and "b1" or "b0" end
    if valueType == "number" then return "d" .. StableNumber(value) end
    if valueType == "string" then return "s" .. #value .. ":" .. value end
    if valueType ~= "table" then return "x" .. valueType end

    stack = stack or {}
    if stack[value] then return "cycle" end
    stack[value] = true
    local parts = { "{" }
    for _, key in ipairs(SortedKeys(value)) do
        parts[#parts + 1] = StableSerialize(key, stack)
        parts[#parts + 1] = "="
        parts[#parts + 1] = StableSerialize(value[key], stack)
        parts[#parts + 1] = ";"
    end
    parts[#parts + 1] = "}"
    stack[value] = nil
    return table.concat(parts)
end

local function HashMaterial(material)
    local hash = 5381
    for index = 1, #material do
        hash = (hash * 33 + string.byte(material, index)) % 4294967296
    end
    return string.format("%04x%04x", math.floor(hash / 65536), hash % 65536)
end

local function IsDynamic(value)
    return type(value) ~= "string" or value == "" or value:find("${", 1, true) ~= nil
end

local function GetStep(steps, stepID)
    if type(steps) ~= "table" or not stepID then return nil end
    return steps[stepID] or steps[tonumber(stepID)]
end

local function AddEdge(edges, source, target)
    source, target = CanonicalID(source), CanonicalID(target)
    if not source or not target then return end
    edges[source] = edges[source] or {}
    edges[source][target] = true
end

local function MarkReachableSteps(steps, edges)
    local reachable, queue = {}, {}
    if GetStep(steps, "1") then
        reachable["1"] = true
        queue[1] = "1"
    end
    local index = 1
    while queue[index] do
        local current = queue[index]
        index = index + 1
        for target in pairs(edges[current] or {}) do
            if not reachable[target] and GetStep(steps, target) then
                reachable[target] = true
                queue[#queue + 1] = target
            end
        end
    end
    return reachable
end

local function BuildWorkflow(workflowID, workflow, auraID)
    local steps = type(workflow) == "table" and workflow.ST or nil
    steps = type(steps) == "table" and steps or {}
    local descriptor = {
        id = workflowID,
        steps = steps,
        edges = {},
        calls = {},
        applies = {},
        effects = 0,
    }

    for rawStepID, step in pairs(steps) do
        local stepID = CanonicalID(rawStepID)
        if stepID and type(step) == "table" then
            descriptor.edges[stepID] = descriptor.edges[stepID] or {}
            AddEdge(descriptor.edges, stepID, step.n)
            if step.t == "branch" and type(step.b) == "table" then
                for _, branch in pairs(step.b) do
                    if type(branch) == "table" then AddEdge(descriptor.edges, stepID, branch.n) end
                end
            end
        end
    end

    local reachable = MarkReachableSteps(steps, descriptor.edges)
    for stepID in pairs(reachable) do
        local step = GetStep(steps, stepID)
        if step and step.t == "branch" and type(step.b) == "table" then
            for _, branch in pairs(step.b) do
                local target = type(branch) == "table" and CanonicalID(branch.failWorkflow) or nil
                if target then descriptor.calls[target] = true end
            end
        end
        if step and type(step.e) == "table" then
            for _, effect in pairs(step.e) do
                if type(effect) == "table" then
                    descriptor.effects = descriptor.effects + 1
                    local effectID = CanonicalID(effect.id) or ""
                    local args = type(effect.args) == "table" and effect.args or {}
                    if effectID == "run_workflow" then
                        if (CanonicalID(args[1]) or "o") == "o" then
                            local target = CanonicalID(args[2])
                            if target then descriptor.calls[target] = true end
                        end
                    elseif effectID == "aura_run_workflow" then
                        local targetAura = CanonicalID(args[1])
                        local targetWorkflow = CanonicalID(args[2])
                        if targetAura == auraID and targetWorkflow then
                            descriptor.calls[targetWorkflow] = true
                        end
                    elseif effectID == "aura_apply" then
                        descriptor.applies[#descriptor.applies + 1] = {
                            target = CanonicalID(args[1]),
                            dynamic = IsDynamic(args[1]),
                            stepID = stepID,
                        }
                    end
                end
            end
        end
    end
    return descriptor
end

local function ProjectEffect(effect)
    if type(effect) ~= "table" then return { invalid = type(effect) } end
    local effectID = CanonicalID(effect.id) or ""
    local projected = { id = effectID }
    if effectID == "aura_apply" or effectID == "aura_duration"
        or effectID == "aura_remove" or effectID == "aura_var_set"
        or effectID == "aura_run_workflow" or effectID == "run_workflow" then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = {}
        local count = effectID == "aura_var_set" and 4
            or effectID == "aura_duration" and 3
            or 2
        for index = 1, count do projected.args[index] = args[index] end
    end
    return projected
end

local function ProjectClass(class, classID, seen)
    if type(class) ~= "table" then return nil end
    if seen[class] then return { id = classID, cycle = true } end
    seen[class] = true
    local projected = {
        id = classID,
        TY = class.TY,
        BA = type(class.BA) == "table" and {
            CC = class.BA.CC,
            IV = class.BA.IV,
        } or nil,
        LI = class.LI,
        HA = class.HA,
        SC = {},
        children = {},
    }
    if type(class.SC) == "table" then
        for _, rawWorkflowID in ipairs(SortedKeys(class.SC)) do
            local workflowID = CanonicalID(rawWorkflowID)
            local workflow = class.SC[rawWorkflowID]
            if workflowID and type(workflow) == "table" then
                local projectedWorkflow = { ST = {} }
                for _, rawStepID in ipairs(SortedKeys(workflow.ST)) do
                    local stepID = CanonicalID(rawStepID)
                    local step = workflow.ST[rawStepID]
                    if stepID and type(step) == "table" then
                        local projectedStep = { t = step.t, n = step.n, b = step.b, e = {} }
                        if type(step.e) == "table" then
                            for _, effectKey in ipairs(SortedKeys(step.e)) do
                                projectedStep.e[effectKey] = ProjectEffect(step.e[effectKey])
                            end
                        end
                        projectedWorkflow.ST[stepID] = projectedStep
                    end
                end
                projected.SC[workflowID] = projectedWorkflow
            end
        end
    end
    for _, groupName in ipairs(CHILD_GROUPS) do
        local group = class[groupName]
        if type(group) == "table" then
            projected.children[groupName] = {}
            for _, childKey in ipairs(SortedKeys(group)) do
                projected.children[groupName][childKey] = ProjectClass(
                    group[childKey], classID .. " " .. tostring(childKey), seen
                )
            end
        end
    end
    seen[class] = nil
    return projected
end

local function BuildFingerprint(rootID, root)
    local material = StableSerialize({
        ruleVersion = Rules.RULE_VERSION,
        rootID = rootID,
        class = ProjectClass(root, rootID, {}),
    })
    return "iga" .. Rules.RULE_VERSION .. ":" .. HashMaterial(material) .. ":" .. #material
end

local function NewResult(rootID)
    return {
        rootID = rootID,
        blocked = false,
        score = 0,
        behaviorScore = 0,
        amplificationScore = 0,
        observationScore = 0,
        hasSideEffect = false,
        reasons = {},
        findings = {},
        metrics = {
            auraClasses = 0,
            cancellationWorkflows = 0,
            cancellationEffects = 0,
            selfReapplications = 0,
            dynamicApplications = 0,
        },
    }
end

local function AddSelfReapply(result, classID, workflowID, stepID)
    local reason = "光环取消工作流会重新施加自身，导致无法正常取消"
    result.blocked = true
    result.hasSideEffect = true
    result.behaviorScore = math.max(result.behaviorScore, 120)
    result.score = result.behaviorScore
    result.metrics.selfReapplications = result.metrics.selfReapplications + 1
    result.reasons[#result.reasons + 1] = reason
    result.findings[#result.findings + 1] = {
        kind = "aura_cancel_self_reapply",
        effectID = "aura_apply",
        classID = classID,
        workflowID = workflowID,
        stepID = stepID,
        score = 120,
        reason = reason,
        hard = true,
    }
end

local function AnalyzeAuraClass(result, classID, class)
    if class.TY ~= "AU" then return end
    result.metrics.auraClasses = result.metrics.auraClasses + 1
    local cancellationWorkflow = type(class.LI) == "table" and CanonicalID(class.LI.OC) or nil
    if not cancellationWorkflow or type(class.SC) ~= "table" then return end

    local descriptors = {}
    for rawWorkflowID, workflow in pairs(class.SC) do
        local workflowID = CanonicalID(rawWorkflowID)
        if workflowID and type(workflow) == "table" then
            descriptors[workflowID] = BuildWorkflow(workflowID, workflow, classID)
        end
    end
    if not descriptors[cancellationWorkflow] then return end

    local reachable = { [cancellationWorkflow] = true }
    local queue = { cancellationWorkflow }
    local index = 1
    while queue[index] do
        local workflowID = queue[index]
        index = index + 1
        local descriptor = descriptors[workflowID]
        if descriptor then
            result.metrics.cancellationWorkflows = result.metrics.cancellationWorkflows + 1
            result.metrics.cancellationEffects = result.metrics.cancellationEffects + descriptor.effects
            for target in pairs(descriptor.calls) do
                if descriptors[target] and not reachable[target] then
                    reachable[target] = true
                    queue[#queue + 1] = target
                end
            end
        end
    end

    for workflowID in pairs(reachable) do
        local descriptor = descriptors[workflowID]
        for _, apply in ipairs(descriptor and descriptor.applies or {}) do
            if apply.dynamic then
                result.metrics.dynamicApplications = result.metrics.dynamicApplications + 1
            elseif apply.target == classID then
                AddSelfReapply(result, classID, workflowID, apply.stepID)
            end
        end
    end
end

local function WalkClasses(result, classID, class, seen)
    if type(class) ~= "table" or seen[class] then return end
    seen[class] = true
    AnalyzeAuraClass(result, classID, class)
    for _, groupName in ipairs(CHILD_GROUPS) do
        local group = class[groupName]
        if type(group) == "table" then
            for childID, child in pairs(group) do
                WalkClasses(result, classID .. " " .. tostring(childID), child, seen)
            end
        end
    end
end

function Rules.Analyze(rootID, root, context)
    rootID = CanonicalID(rootID) or "unknown"
    local result = NewResult(rootID)
    if type(root) == "table" then WalkClasses(result, rootID, root, {}) end
    result.fingerprint = BuildFingerprint(rootID, type(root) == "table" and root or {})
    return result
end
