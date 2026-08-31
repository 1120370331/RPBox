-- Static lifecycle rules for TRP3 Extended item workflows.
--
-- TRP3 runs class.LI.OD immediately before a manually destroyed stack is
-- removed.  Re-adding the same class from that entry point can therefore make
-- an item appear impossible to destroy.  This module treats the lifecycle
-- entry as amplification evidence, but only blocks when a reachable standard
-- effect proves the item (or a related class in the same root) is recreated.
-- Lua Script Effect payloads are deliberately opaque.

local ADDON_NAME, ns = ...

local Rules = {}
ns.ItemGuardLifecycleRules = Rules

Rules.RULE_VERSION = 1

local BLOCK_SCORE = 100
local MAX_AMPLIFICATION_SCORE = 40
local CHILD_GROUPS = { "IN", "QE", "ST" }

local function CanonicalID(value)
    if type(value) == "string" or type(value) == "number" then
        local id = tostring(value)
        if id ~= "" then return id end
    end
    return nil
end

local function RootID(value)
    local id = CanonicalID(value)
    if not id then return nil end
    return id:match("^[^ ]+") or id
end

local function IsDynamic(value)
    if value == nil then return true end
    if type(value) ~= "string" then return false end
    return value == "" or value:find("${", 1, true) ~= nil
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

function Rules.GetDestructionWorkflowIDs(class)
    local workflows = {}
    if type(class) ~= "table" or type(class.LI) ~= "table" then return workflows end
    local workflowID = CanonicalID(class.LI.OD)
    if workflowID then workflows[workflowID] = true end
    return workflows
end

function Rules.IsDestructionExecution(rootID, class, workflowID)
    -- rootID is intentionally accepted so the runtime hook can use one stable
    -- signature for root and nested classes.  LI.OD itself is class-local.
    if not CanonicalID(rootID) then return false end
    local id = CanonicalID(workflowID)
    return id ~= nil and Rules.GetDestructionWorkflowIDs(class)[id] == true
end

function Rules.ClassifyRespawnEffect(rootID, class, effectID, args)
    effectID = CanonicalID(effectID)
    if not effectID or effectID == "script" then return nil end
    args = type(args) == "table" and args or {}

    if effectID == "item_add" then
        local targetID = CanonicalID(args[1])
        local countValue = args[2]
        if countValue == nil then countValue = 1 end
        local count = tonumber(countValue)
        local dynamicTarget = IsDynamic(args[1])
        local dynamicCount = count == nil
        local selfRespawn = false
        if targetID and not dynamicTarget and RootID(targetID) == RootID(rootID) then
            selfRespawn = count == nil or count > 0
        end
        return {
            kind = "item_add",
            targetID = targetID,
            count = count or countValue,
            selfRespawn = selfRespawn,
            dynamic = dynamicTarget or dynamicCount,
            dynamicTarget = dynamicTarget,
            dynamicCount = dynamicCount,
        }
    elseif effectID == "run_workflow" then
        return {
            kind = "workflow_call",
            source = CanonicalID(args[1]) or "o",
            targetID = CanonicalID(args[2]),
            count = 0,
            selfRespawn = false,
            dynamic = IsDynamic(args[2]),
        }
    elseif effectID == "run_item_workflow" then
        return {
            kind = "item_workflow_call",
            source = CanonicalID(args[1]) or "p",
            targetID = CanonicalID(args[2]),
            slotID = CanonicalID(args[3]),
            count = 0,
            selfRespawn = false,
            dynamic = IsDynamic(args[2]) or (args[3] ~= nil and IsDynamic(args[3])),
        }
    elseif effectID == "item_use" then
        return {
            kind = "item_use",
            source = "ch",
            targetID = nil,
            slotID = CanonicalID(args[1]),
            count = 0,
            selfRespawn = false,
            dynamic = IsDynamic(args[1]),
        }
    end
    return nil
end

local function ProjectEffect(effect)
    if type(effect) ~= "table" then return { invalid = type(effect) } end
    local effectID = CanonicalID(effect.id) or ""
    local projected = { id = effectID }
    -- Never index Script Effect args.  Only standard effects used by this rule
    -- are projected into the fingerprint.
    if effectID == "item_add" then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = { args[1], args[2], args[4] }
    elseif effectID == "run_workflow" then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = { args[1], args[2] }
    elseif effectID == "run_item_workflow" then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = { args[1], args[2], args[3] }
    elseif effectID == "item_use" then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = { args[1] }
    end
    return projected
end

local function ProjectClass(class, classID, seen)
    if type(class) ~= "table" then return nil end
    seen = seen or {}
    if seen[class] then return { id = classID, cycle = true } end
    seen[class] = true
    local projected = {
        id = classID,
        LI = type(class.LI) == "table" and { OD = class.LI.OD } or nil,
        SC = {},
        children = {},
    }
    if type(class.SC) == "table" then
        for _, rawWorkflowID in ipairs(SortedKeys(class.SC)) do
            local workflowID = CanonicalID(rawWorkflowID)
            local workflow = class.SC[rawWorkflowID]
            if workflowID and type(workflow) == "table" then
                local projectedWorkflow = { ST = {} }
                local steps = type(workflow.ST) == "table" and workflow.ST or {}
                for _, rawStepID in ipairs(SortedKeys(steps)) do
                    local stepID = CanonicalID(rawStepID)
                    local step = steps[rawStepID]
                    if stepID and type(step) == "table" then
                        local projectedStep = { t = step.t, n = step.n, b = {}, e = {} }
                        if type(step.b) == "table" then
                            for _, branchKey in ipairs(SortedKeys(step.b)) do
                                local branch = step.b[branchKey]
                                if type(branch) == "table" then
                                    projectedStep.b[branchKey] = {
                                        n = branch.n,
                                        failWorkflow = branch.failWorkflow,
                                    }
                                end
                            end
                        end
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
                local childID = classID .. " " .. tostring(childKey)
                projected.children[groupName][childKey] = ProjectClass(group[childKey], childID, seen)
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
    return "igl" .. Rules.RULE_VERSION .. ":" .. HashMaterial(material) .. ":" .. #material
end

local function NewResult(rootID)
    return {
        rootID = rootID,
        blocked = false,
        score = 0,
        behaviorScore = 0,
        amplificationScore = 0,
        reasons = {},
        findings = {},
        fingerprint = "",
        metrics = {
            classes = 0,
            destructionEntries = 0,
            workflowsAnalyzed = 0,
            reachableSteps = 0,
            disconnectedSteps = 0,
            effectsAnalyzed = 0,
            scriptEffects = 0,
            itemAdds = 0,
            rewardAdds = 0,
            selfRespawns = 0,
            dynamicRespawns = 0,
            workflowCalls = 0,
            itemUses = 0,
            recursiveWorkflows = 0,
            unresolvedCalls = 0,
        },
        hasSideEffect = false,
        _reasonSet = {},
        _behaviorBands = {},
        _amplificationBands = {},
    }
end

local function AddReason(result, reason)
    if reason and not result._reasonSet[reason] then
        result._reasonSet[reason] = true
        result.reasons[#result.reasons + 1] = reason
    end
end

local function AddFinding(result, finding, includeReason)
    finding.score = tonumber(finding.score) or 0
    result.findings[#result.findings + 1] = finding
    if includeReason ~= false then AddReason(result, finding.reason) end
end

local function SetBand(result, field, bandsField, band, score)
    score = tonumber(score) or 0
    local bands = result[bandsField]
    local previous = bands[band] or 0
    if score > previous then
        bands[band] = score
        result[field] = result[field] + score - previous
    end
end

local function SetBehaviorBand(result, band, score)
    SetBand(result, "behaviorScore", "_behaviorBands", band, score)
end

local function SetAmplificationBand(result, band, score)
    SetBand(result, "amplificationScore", "_amplificationBands", band, score)
    result.amplificationScore = math.min(MAX_AMPLIFICATION_SCORE, result.amplificationScore)
end

local function GetStep(steps, stepID)
    return steps[stepID] or steps[tonumber(stepID)]
end

local function AddEdge(edges, source, target)
    source, target = CanonicalID(source), CanonicalID(target)
    if not source or not target then return end
    edges[source] = edges[source] or {}
    edges[source][target] = true
end

local function MarkReachable(entries, nodes, edges)
    local reachable, queue, head = {}, {}, 1
    for _, entry in ipairs(SortedKeys(entries)) do
        if entries[entry] and nodes[entry] and not reachable[entry] then
            reachable[entry] = true
            queue[#queue + 1] = entry
        end
    end
    while queue[head] do
        local source = queue[head]
        head = head + 1
        for _, target in ipairs(SortedKeys(edges[source])) do
            if nodes[target] and not reachable[target] then
                reachable[target] = true
                queue[#queue + 1] = target
            end
        end
    end
    return reachable
end

local function CollectClasses(rootID, root)
    local classes, parents = {}, {}
    local seen = {}
    local function walk(classID, class, parentID)
        if type(class) ~= "table" or seen[class] then return end
        seen[class] = true
        classes[classID] = class
        parents[classID] = parentID
        for _, groupName in ipairs(CHILD_GROUPS) do
            local group = class[groupName]
            if type(group) == "table" then
                for _, childKey in ipairs(SortedKeys(group)) do
                    walk(classID .. " " .. tostring(childKey), group[childKey], classID)
                end
            end
        end
    end
    walk(rootID, root, nil)
    return classes, parents
end

local function BuildDescriptor(classID, workflowID, workflow)
    local steps = type(workflow) == "table" and workflow.ST or nil
    steps = type(steps) == "table" and steps or {}
    local descriptor = {
        classID = classID,
        workflowID = workflowID,
        steps = steps,
        stepNodes = {},
        stepEdges = {},
        reachableSteps = {},
        effects = {},
    }
    for _, rawStepID in ipairs(SortedKeys(steps)) do
        local stepID = CanonicalID(rawStepID)
        local step = steps[rawStepID]
        if stepID and type(step) == "table" then
            descriptor.stepNodes[stepID] = true
            descriptor.stepEdges[stepID] = descriptor.stepEdges[stepID] or {}
            AddEdge(descriptor.stepEdges, stepID, step.n)
            if step.t == "branch" and type(step.b) == "table" then
                for _, branch in pairs(step.b) do
                    if type(branch) == "table" then AddEdge(descriptor.stepEdges, stepID, branch.n) end
                end
            end
        end
    end
    descriptor.reachableSteps = MarkReachable({ ["1"] = true }, descriptor.stepNodes, descriptor.stepEdges)
    for _, stepID in ipairs(SortedKeys(descriptor.reachableSteps)) do
        local step = GetStep(steps, stepID)
        if step and type(step.e) == "table" then
            for _, effectKey in ipairs(SortedKeys(step.e)) do
                local effect = step.e[effectKey]
                if type(effect) == "table" then
                    descriptor.effects[#descriptor.effects + 1] = {
                        stepID = stepID,
                        effect = effect,
                    }
                end
            end
        end
        if step and step.t == "branch" and type(step.b) == "table" then
            for _, branchKey in ipairs(SortedKeys(step.b)) do
                local branch = step.b[branchKey]
                if type(branch) == "table" and CanonicalID(branch.failWorkflow) then
                    descriptor.effects[#descriptor.effects + 1] = {
                        stepID = stepID,
                        effect = { id = "run_workflow", args = { "o", branch.failWorkflow } },
                        synthetic = "failWorkflow",
                    }
                end
            end
        end
    end
    return descriptor
end

local function NodeID(classID, workflowID)
    return classID .. "\001" .. workflowID
end

local function NormalizeResolvedTarget(target, classes)
    if type(target) ~= "table" then return nil end
    local classID = CanonicalID(target.classID or target[1])
    local workflowID = CanonicalID(target.workflowID or target[2])
    local class = target.class or (classID and classes[classID])
    if classID and workflowID and type(class) == "table" then
        return classID, workflowID, class
    end
    return nil
end

local function ResolveCallTargets(rootID, root, classes, parents, classID, class, classification, context)
    local targets = {}
    local function add(targetClassID, workflowID, targetClass)
        if targetClassID and workflowID and type(targetClass) == "table" then
            targets[#targets + 1] = {
                classID = targetClassID,
                workflowID = workflowID,
                class = targetClass,
            }
        end
    end

    if classification.kind == "workflow_call" then
        if classification.source == "o" then
            add(classID, classification.targetID, class)
        end
    elseif classification.kind == "item_workflow_call" then
        if classification.source == "o" then
            add(classID, classification.targetID, class)
        elseif classification.source == "p" then
            local parentID = parents[classID]
            if parentID then add(parentID, classification.targetID, classes[parentID]) end
        end
    end

    -- Runtime/container information can prove ch/si/item_use targets that are
    -- intentionally not guessed from class definitions.  The callback returns
    -- an array of {classID=..., workflowID=..., class=...} descriptors.
    if type(context) == "table" and type(context.resolveEffectTargets) == "function" then
        local ok, resolved = pcall(
            context.resolveEffectTargets,
            classID,
            class,
            classification,
            rootID,
            root
        )
        if ok and type(resolved) == "table" then
            for _, target in ipairs(resolved) do
                local targetClassID, workflowID, targetClass = NormalizeResolvedTarget(target, classes)
                add(targetClassID, workflowID, targetClass)
            end
        end
    end
    return targets
end

local function HasPath(graph, source, target, seen)
    if source == target then return true end
    seen = seen or {}
    if seen[source] then return false end
    seen[source] = true
    for nextNode in pairs(graph[source] or {}) do
        if HasPath(graph, nextNode, target, seen) then return true end
    end
    return false
end

function Rules.Analyze(rootID, root, context)
    rootID = CanonicalID(rootID) or "unknown"
    local result = NewResult(rootID)
    if type(root) ~= "table" then
        AddFinding(result, {
            kind = "lifecycle_invalid_root",
            score = 0,
            reason = "无法分析道具的销毁生命周期：根对象不是有效表",
        })
        result.fingerprint = BuildFingerprint(rootID, {})
        result._reasonSet, result._behaviorBands, result._amplificationBands = nil, nil, nil
        return result
    end

    local classes, parents = CollectClasses(rootID, root)
    local descriptors, nodes, graph = {}, {}, {}
    local entries = {}
    for _, classID in ipairs(SortedKeys(classes)) do
        local class = classes[classID]
        result.metrics.classes = result.metrics.classes + 1
        local scripts = type(class.SC) == "table" and class.SC or {}
        for _, rawWorkflowID in ipairs(SortedKeys(scripts)) do
            local workflowID = CanonicalID(rawWorkflowID)
            local workflow = scripts[rawWorkflowID]
            if workflowID and type(workflow) == "table" then
                local node = NodeID(classID, workflowID)
                descriptors[node] = BuildDescriptor(classID, workflowID, workflow)
                nodes[node], graph[node] = true, {}
            end
        end
        for workflowID in pairs(Rules.GetDestructionWorkflowIDs(class)) do
            local node = NodeID(classID, workflowID)
            if descriptors[node] then
                entries[node] = true
                result.metrics.destructionEntries = result.metrics.destructionEntries + 1
                AddFinding(result, {
                    kind = "destruction_lifecycle_entry",
                    classID = classID,
                    workflowID = workflowID,
                    score = 20,
                    reason = "道具配置了摧毁堆叠时自动执行的工作流",
                }, false)
            end
        end
    end

    -- First build the call graph using only effects reachable from step 1.
    for _, node in ipairs(SortedKeys(descriptors)) do
        local descriptor = descriptors[node]
        local class = classes[descriptor.classID]
        for _, effectInfo in ipairs(descriptor.effects) do
            local effect = effectInfo.effect
            local classification = Rules.ClassifyRespawnEffect(
                descriptor.classID,
                class,
                effect.id,
                effect.args
            )
            if classification and classification.kind ~= "item_add" then
                local targets = ResolveCallTargets(
                    rootID,
                    root,
                    classes,
                    parents,
                    descriptor.classID,
                    class,
                    classification,
                    context
                )
                for _, target in ipairs(targets) do
                    local targetNode = NodeID(target.classID, target.workflowID)
                    if nodes[targetNode] then graph[node][targetNode] = true end
                end
            end
        end
    end

    local reachableNodes = MarkReachable(entries, nodes, graph)
    if next(reachableNodes) ~= nil then
        SetAmplificationBand(result, "destruction_lifecycle", 20)
    end

    local recursiveNodes = {}
    for _, source in ipairs(SortedKeys(reachableNodes)) do
        for target in pairs(graph[source] or {}) do
            if reachableNodes[target] and HasPath(graph, target, source, {}) then
                recursiveNodes[source], recursiveNodes[target] = true, true
            end
        end
    end
    if next(recursiveNodes) ~= nil then
        SetAmplificationBand(result, "lifecycle_recursion", 20)
        for _ in pairs(recursiveNodes) do
            result.metrics.recursiveWorkflows = result.metrics.recursiveWorkflows + 1
        end
        AddFinding(result, {
            kind = "destruction_workflow_recursion",
            score = 20,
            reason = "摧毁生命周期可达路径存在递归工作流调用",
        })
    end

    for _, node in ipairs(SortedKeys(reachableNodes)) do
        local descriptor = descriptors[node]
        local class = classes[descriptor.classID]
        result.metrics.workflowsAnalyzed = result.metrics.workflowsAnalyzed + 1
        for stepID in pairs(descriptor.stepNodes) do
            if descriptor.reachableSteps[stepID] then
                result.metrics.reachableSteps = result.metrics.reachableSteps + 1
            else
                result.metrics.disconnectedSteps = result.metrics.disconnectedSteps + 1
            end
        end
        for _, effectInfo in ipairs(descriptor.effects) do
            local effect = effectInfo.effect
            local effectID = CanonicalID(effect.id) or ""
            result.metrics.effectsAnalyzed = result.metrics.effectsAnalyzed + 1
            if effectID == "script" then
                result.metrics.scriptEffects = result.metrics.scriptEffects + 1
            else
                local classification = Rules.ClassifyRespawnEffect(
                    descriptor.classID,
                    class,
                    effectID,
                    effect.args
                )
                if classification and classification.kind == "item_add" then
                    result.metrics.itemAdds = result.metrics.itemAdds + 1
                    if classification.selfRespawn and not classification.dynamic then
                        result.metrics.selfRespawns = result.metrics.selfRespawns + 1
                        result.hasSideEffect = true
                        SetBehaviorBand(result, "lifecycle_self_respawn", 85)
                        AddFinding(result, {
                            kind = "destruction_self_respawn",
                            classID = descriptor.classID,
                            workflowID = descriptor.workflowID,
                            stepID = effectInfo.stepID,
                            effectID = effectID,
                            targetID = classification.targetID,
                            count = classification.count,
                            score = 85,
                            reason = "摧毁生命周期会重新给予当前道具或同一根对象中的相关道具",
                        })
                    elseif classification.dynamicTarget
                        or (classification.selfRespawn and classification.dynamicCount) then
                        result.metrics.dynamicRespawns = result.metrics.dynamicRespawns + 1
                        AddFinding(result, {
                            kind = "destruction_respawn_dynamic",
                            classID = descriptor.classID,
                            workflowID = descriptor.workflowID,
                            stepID = effectInfo.stepID,
                            effectID = effectID,
                            targetID = classification.targetID,
                            count = classification.count,
                            score = 0,
                            reason = "摧毁生命周期包含运行时才能确认目标或数量的物品添加",
                            runtimeConfirmation = true,
                        })
                    else
                        local count = tonumber(classification.count)
                        if count == nil or count > 0 then
                            result.metrics.rewardAdds = result.metrics.rewardAdds + 1
                            AddFinding(result, {
                                kind = "destruction_reward_add",
                                classID = descriptor.classID,
                                workflowID = descriptor.workflowID,
                                stepID = effectInfo.stepID,
                                effectID = effectID,
                                targetID = classification.targetID,
                                count = classification.count,
                                score = 0,
                                reason = "摧毁生命周期会给予其他道具；未发现自我再生",
                            }, false)
                        end
                    end
                elseif classification then
                    if classification.kind == "item_use" then
                        result.metrics.itemUses = result.metrics.itemUses + 1
                    else
                        result.metrics.workflowCalls = result.metrics.workflowCalls + 1
                    end
                    local targets = ResolveCallTargets(
                        rootID,
                        root,
                        classes,
                        parents,
                        descriptor.classID,
                        class,
                        classification,
                        context
                    )
                    local hasKnownTarget = false
                    for _, target in ipairs(targets) do
                        if nodes[NodeID(target.classID, target.workflowID)] then
                            hasKnownTarget = true
                            break
                        end
                    end
                    if not hasKnownTarget then
                        result.metrics.unresolvedCalls = result.metrics.unresolvedCalls + 1
                        AddFinding(result, {
                            kind = "destruction_call_unresolved",
                            classID = descriptor.classID,
                            workflowID = descriptor.workflowID,
                            stepID = effectInfo.stepID,
                            effectID = effectID,
                            targetID = classification.targetID,
                            score = 0,
                            reason = "摧毁生命周期包含需要在运行时确认目标的工作流调用",
                            runtimeConfirmation = true,
                        })
                    end
                end
            end
        end
    end

    result.score = result.behaviorScore + result.amplificationScore
    result.blocked = result.hasSideEffect and result.score >= BLOCK_SCORE
    result.fingerprint = BuildFingerprint(rootID, root)
    result._reasonSet, result._behaviorBands, result._amplificationBands = nil, nil, nil
    return result
end
