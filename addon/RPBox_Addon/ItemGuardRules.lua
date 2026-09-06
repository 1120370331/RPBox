-- Static behavior rules for TRP3 Extended standard workflows.
-- Lua Script Effect payloads are deliberately opaque to this module.

local ADDON_NAME, ns = ...

local Rules = {}
ns.ItemGuardRules = Rules

Rules.RULE_VERSION = 3

local MAX_AMPLIFICATION_SCORE = 40
local BLOCK_SCORE = 100

local RELEVANT_EFFECT_ARGS = {
    item_add = 4,
    item_loot = 1,
    item_use = 1,
    run_item_workflow = 3,
    run_workflow = 2,
}

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
        return tostring(leftType) < tostring(rightType)
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
    local high = math.floor(hash / 65536)
    local low = hash % 65536
    return string.format("%04x%04x", high, low)
end

local function ProjectEffect(effect)
    if type(effect) ~= "table" then return { invalid = type(effect) } end
    local effectID = CanonicalID(effect.id) or ""
    local projected = { id = effectID }
    local argCount = RELEVANT_EFFECT_ARGS[effectID]
    -- In particular, do not even index effect.args for Script Effects.
    if argCount then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = {}
        for index = 1, argCount do projected.args[index] = args[index] end
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
        US = type(class.US) == "table" and { SC = class.US.SC } or nil,
        LI = class.LI,
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
    local projection = {
        ruleVersion = Rules.RULE_VERSION,
        rootID = rootID,
        class = ProjectClass(root, rootID, {}),
    }
    local material = StableSerialize(projection)
    return "ig" .. tostring(Rules.RULE_VERSION) .. ":" .. HashMaterial(material) .. ":" .. tostring(#material)
end

local function NewResult(rootID)
    return {
        rootID = rootID,
        blocked = false,
        score = 0,
        behaviorScore = 0,
        amplificationScore = 0,
        observationScore = 0,
        reasons = {},
        findings = {},
        fingerprint = "",
        metrics = {
            classes = 0,
            workflowsTotal = 0,
            workflowsAnalyzed = 0,
            disconnectedWorkflows = 0,
            stepsTotal = 0,
            reachableSteps = 0,
            disconnectedSteps = 0,
            effectsAnalyzed = 0,
            standardEffects = 0,
            scriptEffects = 0,
            itemAdds = 0,
            itemAddsAll = 0,
            itemLoots = 0,
            itemUses = 0,
            workflowCalls = 0,
            externalEdges = 0,
            unresolvedTargets = 0,
            fallbackClasses = 0,
            explicitEntryClasses = 0,
            entryConfidence = "unknown",
        },
        _reasonSet = {},
        _behaviorBands = {},
        _hardBlocked = false,
        _hasSideEffect = false,
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
    if finding.hard then result._hardBlocked = true end
end

local function SetBehaviorBand(result, band, score)
    score = tonumber(score) or 0
    local previous = result._behaviorBands[band] or 0
    if score > previous then
        result._behaviorBands[band] = score
        result.behaviorScore = result.behaviorScore + score - previous
    end
end

local function AddAmplification(result, score)
    result.amplificationScore = math.min(
        MAX_AMPLIFICATION_SCORE,
        result.amplificationScore + (tonumber(score) or 0)
    )
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

local function MarkReachable(startNodes, nodes, edges)
    local reachable, queue = {}, {}
    for startNode in pairs(startNodes or {}) do
        if nodes[startNode] and not reachable[startNode] then
            reachable[startNode] = true
            queue[#queue + 1] = startNode
        end
    end
    local cursor = 1
    while cursor <= #queue do
        local source = queue[cursor]
        cursor = cursor + 1
        for target in pairs(edges[source] or {}) do
            if nodes[target] and not reachable[target] then
                reachable[target] = true
                queue[#queue + 1] = target
            end
        end
    end
    return reachable
end

local function StronglyConnected(nodes, edges)
    local index = 0
    local indexes, lowLinks, onStack, stack = {}, {}, {}, {}
    local components, componentOf = {}, {}

    local function Visit(node)
        index = index + 1
        indexes[node], lowLinks[node] = index, index
        stack[#stack + 1] = node
        onStack[node] = true

        for target in pairs(edges[node] or {}) do
            if nodes[target] then
                if not indexes[target] then
                    Visit(target)
                    lowLinks[node] = math.min(lowLinks[node], lowLinks[target])
                elseif onStack[target] then
                    lowLinks[node] = math.min(lowLinks[node], indexes[target])
                end
            end
        end

        if lowLinks[node] == indexes[node] then
            local component = {}
            while true do
                local member = stack[#stack]
                stack[#stack] = nil
                onStack[member] = nil
                component[#component + 1] = member
                componentOf[member] = #components + 1
                if member == node then break end
            end
            table.sort(component)
            components[#components + 1] = component
        end
    end

    for _, node in ipairs(SortedKeys(nodes)) do
        if nodes[node] and not indexes[node] then Visit(node) end
    end
    return components, componentOf
end

local function IsCyclicComponent(component, edges)
    if #component > 1 then return true end
    local node = component[1]
    return node and edges[node] and edges[node][node] or false
end

local function CollectLinkValues(value, output, seen)
    local valueType = type(value)
    if valueType == "string" or valueType == "number" then
        local link = CanonicalID(value)
        if link then output[link] = true end
    elseif valueType == "table" and not seen[value] then
        seen[value] = true
        for _, child in pairs(value) do CollectLinkValues(child, output, seen) end
    end
end

local function ResolveExternal(context, request)
    if type(context) ~= "table" then return nil end
    local resolver = context.resolver
    if type(resolver) == "table" then resolver = resolver.resolve end
    if type(resolver) ~= "function" then return nil end
    local ok, resolved = pcall(resolver, request)
    if ok then return resolved end
    return nil
end

local function DescribeExternalEdge(result, edge, context)
    result.metrics.externalEdges = result.metrics.externalEdges + 1
    local request = {
        rootID = result.rootID,
        classID = edge.classID,
        workflowID = edge.workflowID,
        stepID = edge.stepID,
        effectID = edge.effectID,
        source = edge.source,
        target = edge.target,
        slotID = edge.slotID,
    }
    local resolved = ResolveExternal(context, request)
    local finding
    if resolved ~= nil then
        finding = {
            kind = "cross_object_target_resolved",
            effectID = edge.effectID,
            workflowID = edge.workflowID,
            classID = edge.classID,
            score = 0,
            reason = "跨对象调用目标已解析；未将其臆造成当前对象的递归边",
            target = edge.target,
            source = edge.source,
            resolved = true,
        }
    else
        result.metrics.unresolvedTargets = result.metrics.unresolvedTargets + 1
        finding = {
            kind = "cross_object_target_unresolved",
            effectID = edge.effectID,
            workflowID = edge.workflowID,
            classID = edge.classID,
            score = 0,
            reason = "存在无法静态解析的跨对象工作流目标",
            target = edge.target,
            source = edge.source,
            slotID = edge.slotID,
        }
    end
    AddFinding(result, finding)
end

local function ParseNumericCount(value, defaultValue)
    if value == nil then return defaultValue, "literal" end
    if type(value) == "number" then
        if value ~= value or value == math.huge or value == -math.huge then return nil, "nonfinite" end
        return value, "literal"
    end
    if type(value) == "string" then
        local parsed = tonumber(value)
        if parsed == nil then return nil, "dynamic" end
        if parsed ~= parsed or parsed == math.huge or parsed == -math.huge then return nil, "nonfinite" end
        return parsed, "literal"
    end
    return nil, "malformed"
end

local function AnalyzeItemAdd(result, classID, workflowID, stepID, effect, repeated)
    result.metrics.itemAdds = result.metrics.itemAdds + 1
    result.metrics.standardEffects = result.metrics.standardEffects + 1
    result._hasSideEffect = true
    local args = type(effect.args) == "table" and effect.args or {}
    local count, countKind = ParseNumericCount(args[2], 1)
    local common = {
        effectID = "item_add",
        workflowID = workflowID,
        classID = classID,
        stepID = stepID,
    }

    if countKind == "dynamic" then
        common.kind = "item_add_dynamic_count"
        common.score = 0
        common.reason = "物品添加数量为动态表达式；静态扫描不据此隔离，将由运行时配额解析"
        AddFinding(result, common)
        return
    elseif countKind == "nonfinite" or countKind == "malformed" then
        common.kind = "item_add_invalid_count"
        common.score = 120
        common.reason = "物品添加数量不是有限的有效数值"
        common.hard = true
        AddFinding(result, common)
        SetBehaviorBand(result, "item_add_hard", 120)
        return
    elseif count <= 0 then
        common.kind = "item_add_nonpositive_count"
        common.score = 0
        common.reason = "物品添加数量不是正数；记录但不按数据膨胀隔离"
        AddFinding(result, common)
        return
    end

    local baseScore = count <= 20 and 5 or 10
    SetBehaviorBand(result, "item_add_base", baseScore)
    AddFinding(result, {
        kind = "item_add",
        effectID = "item_add",
        workflowID = workflowID,
        classID = classID,
        stepID = stepID,
        score = baseScore,
        reason = "工作流会添加物品（数量 " .. tostring(count) .. "）",
        count = count,
    })

    if count > 1000 then
        SetBehaviorBand(result, "item_add_hard", 120)
        AddFinding(result, {
            kind = "item_add_resource_exhaustion",
            effectID = "item_add",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 120,
            reason = "单次物品添加数量超过 1000：" .. tostring(count),
            count = count,
            hard = true,
        })
    elseif count > 100 then
        SetBehaviorBand(result, "item_add_volume", 60)
        AddFinding(result, {
            kind = "item_add_large_count",
            effectID = "item_add",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 60,
            reason = "单次物品添加数量处于高风险区间：" .. tostring(count),
            count = count,
        })
    end

    if repeated then
        SetBehaviorBand(result, "item_add_repeated", 80)
        AddFinding(result, {
            kind = "item_add_amplified",
            effectID = "item_add",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 80,
            reason = "循环或递归路径会重复执行物品添加",
            count = count,
        })
    end
end

local function AnalyzeItemLoot(result, classID, workflowID, stepID, effect, repeated)
    result.metrics.itemLoots = result.metrics.itemLoots + 1
    result.metrics.standardEffects = result.metrics.standardEffects + 1
    local args = type(effect.args) == "table" and effect.args or {}
    local lootInfo = args[1]
    if type(lootInfo) ~= "table" then
        AddFinding(result, {
            kind = "item_loot_malformed",
            effectID = "item_loot",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 0,
            reason = "战利品效果参数结构无法静态解析",
        })
        return
    end

    local isDrop = not not lootInfo[4]
    local slots = type(lootInfo[3]) == "table" and lootInfo[3] or {}
    local slotCount, maxSlot, totalCount = 0, 0, 0
    local dynamicCounts = 0
    for slotKey, slot in pairs(slots) do
        if type(slot) == "table" then
            slotCount = slotCount + 1
            local numericSlot = tonumber(slotKey)
            if numericSlot and numericSlot > maxSlot then maxSlot = numericSlot end
            local count, countKind = ParseNumericCount(slot.count, 1)
            if countKind == "literal" then
                if count > 0 then totalCount = totalCount + count end
            elseif countKind == "dynamic" then
                dynamicCounts = dynamicCounts + 1
            else
                result._hasSideEffect = true
                SetBehaviorBand(result, "item_loot_hard", 120)
                AddFinding(result, {
                    kind = "item_loot_invalid_count",
                    effectID = "item_loot",
                    workflowID = workflowID,
                    classID = classID,
                    stepID = stepID,
                    score = 120,
                    reason = "战利品槽位数量不是有限的有效数值",
                    hard = true,
                })
            end
        end
    end

    if dynamicCounts > 0 then
        AddFinding(result, {
            kind = "item_loot_dynamic_count",
            effectID = "item_loot",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 0,
            reason = "战利品包含动态数量；静态扫描不据此隔离",
            dynamicCounts = dynamicCounts,
        })
    end

    if isDrop then
        result._hasSideEffect = true
        SetBehaviorBand(result, "item_loot_drop", 30)
        AddFinding(result, {
            kind = "item_loot_drop",
            effectID = "item_loot",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 30,
            reason = "工作流会直接向地面写入战利品",
            slotCount = slotCount,
            totalCount = totalCount,
        })
        if repeated then
            SetBehaviorBand(result, "item_loot_drop_repeated", 90)
            AddFinding(result, {
                kind = "item_loot_drop_amplified",
                effectID = "item_loot",
                workflowID = workflowID,
                classID = classID,
                stepID = stepID,
                score = 90,
                reason = "循环或递归路径会重复写入地面战利品",
                slotCount = slotCount,
                totalCount = totalCount,
            })
        end
    end

    if slotCount > 32 or maxSlot > 32 then
        result._hasSideEffect = true
        SetBehaviorBand(result, "item_loot_hard", 120)
        AddFinding(result, {
            kind = "item_loot_slot_exhaustion",
            effectID = "item_loot",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 120,
            reason = "战利品槽位超过 32",
            slotCount = slotCount,
            maxSlot = maxSlot,
            hard = true,
        })
    end
    if totalCount > 1000 then
        result._hasSideEffect = true
        SetBehaviorBand(result, "item_loot_hard", 120)
        AddFinding(result, {
            kind = "item_loot_count_exhaustion",
            effectID = "item_loot",
            workflowID = workflowID,
            classID = classID,
            stepID = stepID,
            score = 120,
            reason = "战利品总数量超过 1000：" .. tostring(totalCount),
            totalCount = totalCount,
            hard = true,
        })
    end
end

local function BuildWorkflowDescriptor(result, classID, workflowID, workflow)
    local steps = type(workflow) == "table" and workflow.ST or nil
    steps = type(steps) == "table" and steps or {}
    local descriptor = {
        id = workflowID,
        workflow = workflow,
        steps = steps,
        stepNodes = {},
        stepEdges = {},
        reachableSteps = {},
        effects = {},
        calls = {},
        externalEdges = {},
    }

    for rawStepID, step in pairs(steps) do
        local stepID = CanonicalID(rawStepID)
        if stepID and type(step) == "table" then
            descriptor.stepNodes[stepID] = true
            descriptor.stepEdges[stepID] = descriptor.stepEdges[stepID] or {}
            result.metrics.stepsTotal = result.metrics.stepsTotal + 1
            AddEdge(descriptor.stepEdges, stepID, step.n)
            if step.t == "branch" and type(step.b) == "table" then
                for _, branch in pairs(step.b) do
                    if type(branch) == "table" then
                        AddEdge(descriptor.stepEdges, stepID, branch.n)
                    end
                end
            end
        end
    end
    descriptor.reachableSteps = MarkReachable({ ["1"] = true }, descriptor.stepNodes, descriptor.stepEdges)

    for stepID in pairs(descriptor.reachableSteps) do
        local step = GetStep(steps, stepID)
        if step and step.t == "branch" and type(step.b) == "table" then
            for _, branch in pairs(step.b) do
                local target = type(branch) == "table" and CanonicalID(branch.failWorkflow) or nil
                if target then
                    descriptor.calls[#descriptor.calls + 1] = {
                        target = target,
                        source = "o",
                        effectID = "failWorkflow",
                        stepID = stepID,
                    }
                end
            end
        end

        if step and type(step.e) == "table" then
            for _, effect in pairs(step.e) do
                if type(effect) == "table" then
                    local effectID = CanonicalID(effect.id) or ""
                    descriptor.effects[#descriptor.effects + 1] = {
                        id = effectID,
                        effect = effect,
                        stepID = stepID,
                    }
                    if effectID == "run_workflow" or effectID == "run_item_workflow" then
                        local args = type(effect.args) == "table" and effect.args or {}
                        local defaultSource = effectID == "run_item_workflow" and "p" or "o"
                        descriptor.calls[#descriptor.calls + 1] = {
                            target = CanonicalID(args[2]),
                            source = CanonicalID(args[1]) or defaultSource,
                            slotID = CanonicalID(args[3]),
                            effectID = effectID,
                            stepID = stepID,
                        }
                    elseif effectID == "item_use" then
                        local args = type(effect.args) == "table" and effect.args or {}
                        descriptor.externalEdges[#descriptor.externalEdges + 1] = {
                            source = "ch",
                            slotID = CanonicalID(args[1]),
                            effectID = effectID,
                            stepID = stepID,
                        }
                    end
                end
            end
        end
    end
    return descriptor
end

local function CollectEntries(class, descriptors, classID, context)
    local entries = {}
    if type(class.US) == "table" then
        local useWorkflow = CanonicalID(class.US.SC)
        if useWorkflow and descriptors[useWorkflow] then entries[useWorkflow] = true end
    end
    if type(class.LI) == "table" then
        local candidates = {}
        CollectLinkValues(class.LI, candidates, {})
        for candidate in pairs(candidates) do
            if descriptors[candidate] then entries[candidate] = true end
        end
    end
    if type(context) == "table" and type(context.entrypoints) == "table" then
        local configured = context.entrypoints[classID] or context.entrypoints
        if type(configured) == "table" then
            for key, value in pairs(configured) do
                local candidate = CanonicalID(type(key) == "number" and value or key)
                if candidate and descriptors[candidate] then entries[candidate] = true end
            end
        end
    end
    return entries
end

local function AnalyzeClass(result, classID, class, context)
    local scripts = type(class.SC) == "table" and class.SC or nil
    if not scripts then return end
    result.metrics.classes = result.metrics.classes + 1

    local descriptors, workflowNodes, workflowEdges = {}, {}, {}
    for rawWorkflowID, workflow in pairs(scripts) do
        local workflowID = CanonicalID(rawWorkflowID)
        if workflowID and type(workflow) == "table" then
            descriptors[workflowID] = BuildWorkflowDescriptor(result, classID, workflowID, workflow)
            workflowNodes[workflowID] = true
            workflowEdges[workflowID] = {}
            result.metrics.workflowsTotal = result.metrics.workflowsTotal + 1
        end
    end
    if next(workflowNodes) == nil then return end

    for workflowID, descriptor in pairs(descriptors) do
        for _, call in ipairs(descriptor.calls) do
            if call.source == "o" and call.target and descriptors[call.target] then
                workflowEdges[workflowID][call.target] = true
            end
        end
    end

    local entries = CollectEntries(class, descriptors, classID, context)
    local hasExplicitEntry = next(entries) ~= nil
    if not hasExplicitEntry then
        for workflowID in pairs(workflowNodes) do entries[workflowID] = true end
        result.metrics.fallbackClasses = result.metrics.fallbackClasses + 1
        AddFinding(result, {
            kind = "entrypoint_fallback",
            classID = classID,
            score = 0,
            reason = "对象没有可识别的工作流入口，已以低置信度分析各工作流步骤 1",
            confidence = "fallback",
        }, false)
    else
        result.metrics.explicitEntryClasses = result.metrics.explicitEntryClasses + 1
    end

    local reachableWorkflows = MarkReachable(entries, workflowNodes, workflowEdges)
    for workflowID in pairs(workflowNodes) do
        if reachableWorkflows[workflowID] then
            result.metrics.workflowsAnalyzed = result.metrics.workflowsAnalyzed + 1
        else
            result.metrics.disconnectedWorkflows = result.metrics.disconnectedWorkflows + 1
        end
    end

    local recursiveWorkflows = {}
    local workflowComponents = StronglyConnected(reachableWorkflows, workflowEdges)
    for _, component in ipairs(workflowComponents) do
        if IsCyclicComponent(component, workflowEdges) then
            AddAmplification(result, 20)
            for _, workflowID in ipairs(component) do recursiveWorkflows[workflowID] = true end
            AddFinding(result, {
                kind = "workflow_recursion",
                workflowID = component[1],
                classID = classID,
                score = 20,
                reason = "存在同一对象内的递归工作流调用",
                workflows = component,
            })
        end
    end

    for workflowID in pairs(reachableWorkflows) do
        local descriptor = descriptors[workflowID]
        if ns.ItemGuardStructure then
            local valid, reason = ns.ItemGuardStructure.ValidateWorkflow(descriptor.workflow)
            if not valid then
                AddFinding(result, { kind = "unsafe_compilation", classID = classID,
                    workflowID = workflowID, score = 120, hard = true, bypassable = false,
                    reason = reason })
            end
        end
        local cyclicSteps = {}
        local stepComponents = StronglyConnected(descriptor.reachableSteps, descriptor.stepEdges)
        for _, component in ipairs(stepComponents) do
            if IsCyclicComponent(component, descriptor.stepEdges) then
                AddAmplification(result, 15)
                for _, stepID in ipairs(component) do cyclicSteps[stepID] = true end
                AddFinding(result, {
                    kind = "step_cycle",
                    workflowID = workflowID,
                    classID = classID,
                    score = 120,
                    hard = true,
                    bypassable = false,
                    reason = "工作流“" .. workflowID .. "”的步骤连接成环，无法安全编译",
                    steps = component,
                })
            end
        end

        for stepID in pairs(descriptor.stepNodes) do
            if descriptor.reachableSteps[stepID] then
                result.metrics.reachableSteps = result.metrics.reachableSteps + 1
            else
                result.metrics.disconnectedSteps = result.metrics.disconnectedSteps + 1
            end
        end

        for _, call in ipairs(descriptor.calls) do
            result.metrics.workflowCalls = result.metrics.workflowCalls + 1
            if call.source == "o" then
                if not call.target or not descriptors[call.target] then
                    result.metrics.unresolvedTargets = result.metrics.unresolvedTargets + 1
                    AddFinding(result, {
                        kind = "workflow_target_unresolved",
                        effectID = call.effectID,
                        workflowID = workflowID,
                        classID = classID,
                        score = 0,
                        reason = "同对象工作流调用目标不存在或无法解析",
                        target = call.target,
                    })
                end
            else
                DescribeExternalEdge(result, {
                    classID = classID,
                    workflowID = workflowID,
                    stepID = call.stepID,
                    effectID = call.effectID,
                    source = call.source,
                    target = call.target,
                    slotID = call.slotID,
                }, context)
            end
        end
        for _, edge in ipairs(descriptor.externalEdges) do
            DescribeExternalEdge(result, {
                classID = classID,
                workflowID = workflowID,
                stepID = edge.stepID,
                effectID = edge.effectID,
                source = edge.source,
                target = edge.target,
                slotID = edge.slotID,
            }, context)
            result.metrics.itemUses = result.metrics.itemUses + 1
            result.metrics.standardEffects = result.metrics.standardEffects + 1
        end

        for _, effectInfo in ipairs(descriptor.effects) do
            local effectID = effectInfo.id
            local repeated = cyclicSteps[effectInfo.stepID] or recursiveWorkflows[workflowID] or false
            result.metrics.effectsAnalyzed = result.metrics.effectsAnalyzed + 1
            if effectID == "script" then
                result.metrics.scriptEffects = result.metrics.scriptEffects + 1
            elseif effectID == "item_add" then
                AnalyzeItemAdd(result, classID, workflowID, effectInfo.stepID, effectInfo.effect, repeated)
            elseif effectID == "item_loot" then
                AnalyzeItemLoot(result, classID, workflowID, effectInfo.stepID, effectInfo.effect, repeated)
            elseif effectID == "run_workflow" or effectID == "run_item_workflow" then
                result.metrics.standardEffects = result.metrics.standardEffects + 1
            end
        end
    end
end

local function CountAllItemAdds(class, seen)
    if type(class) ~= "table" or seen[class] then return 0 end
    seen[class] = true
    local count = 0
    if type(class.SC) == "table" then
        for _, workflow in pairs(class.SC) do
            local steps = type(workflow) == "table" and workflow.ST or nil
            if type(steps) == "table" then
                for _, step in pairs(steps) do
                    if type(step) == "table" and type(step.e) == "table" then
                        for _, effect in pairs(step.e) do
                            if type(effect) == "table" and effect.id == "item_add" then count = count + 1 end
                        end
                    end
                end
            end
        end
    end
    for _, groupName in ipairs(CHILD_GROUPS) do
        local group = class[groupName]
        if type(group) == "table" then
            for _, child in pairs(group) do count = count + CountAllItemAdds(child, seen) end
        end
    end
    return count
end

local function WalkClasses(result, classID, class, context, seen)
    if type(class) ~= "table" or seen[class] then return end
    seen[class] = true
    AnalyzeClass(result, classID, class, context)
    for _, groupName in ipairs(CHILD_GROUPS) do
        local group = class[groupName]
        if type(group) == "table" then
            for childKey, child in pairs(group) do
                WalkClasses(result, classID .. " " .. tostring(childKey), child, context, seen)
            end
        end
    end
end

function Rules.Analyze(rootID, root, context)
    rootID = CanonicalID(rootID) or "unknown"
    local result = NewResult(rootID)
    if type(root) ~= "table" then
        AddFinding(result, {
            kind = "invalid_root",
            classID = rootID,
            score = 0,
            reason = "无法分析：根对象不是有效表",
        })
        result.fingerprint = BuildFingerprint(rootID, {})
        result._reasonSet, result._behaviorBands = nil, nil
        result._hardBlocked, result._hasSideEffect = nil, nil
        return result
    end

    result.metrics.itemAddsAll = CountAllItemAdds(root, {})
    WalkClasses(result, rootID, root, context, {})
    if result.metrics.fallbackClasses > 0 and result.metrics.explicitEntryClasses > 0 then
        result.metrics.entryConfidence = "mixed"
    elseif result.metrics.fallbackClasses > 0 then
        result.metrics.entryConfidence = "fallback"
    else
        result.metrics.entryConfidence = "explicit"
    end
    if result.metrics.itemAddsAll > 24 then
        AddFinding(result, {
            kind = "item_add_scale",
            classID = rootID,
            score = 0,
            reason = "对象包含较多物品添加效果；仅作规模报告，不直接隔离",
            count = result.metrics.itemAddsAll,
        })
    end

    result.observationScore = 0
    result.score = result.behaviorScore + result.amplificationScore + result.observationScore
    result.blocked = result._hardBlocked
        or (result._hasSideEffect and result.score >= BLOCK_SCORE)
    result.fingerprint = BuildFingerprint(rootID, root)

    result._reasonSet, result._behaviorBands = nil, nil
    result._hardBlocked, result._hasSideEffect = nil, nil
    return result
end
