-- Static risk rules for TRP3 Extended standard sound effects.
-- Lua Script Effect payloads are deliberately opaque to this module.

local ADDON_NAME, ns = ...

local SoundRules = {}
ns.ItemGuardSoundRules = SoundRules

SoundRules.RULE_VERSION = 1

-- These are integration recommendations, not timers owned by this module.
-- A first breach should suppress only the offending sound effect. Repeated
-- breaches can then be promoted to item quarantine by the ItemGuard policy.
SoundRules.LIMITS = {
    shortWindowSeconds = 5,
    shortStartLimit = 8,
    longWindowSeconds = 60,
    longStartLimit = 30,
    firstBreachAction = "block_sound_effect",
    quarantineAfterBreaches = 2,
    stopResetsFamily = true,
}

local BLOCK_SCORE = 100
local MAX_AMPLIFICATION_SCORE = 40
local CHILD_GROUPS = { "IN", "QE", "ST" }

local SOUND_EFFECT_IDS = {
    sound_id_self = true,
    sound_id_stop = true,
    sound_music_self = true,
    sound_music_stop = true,
    sound_id_local = true,
    sound_id_local_stop = true,
    sound_music_local = true,
    sound_music_local_stop = true,
}

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
        return leftType < rightType
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

local function NormalizeChannel(value)
    local channel = CanonicalID(value) or "SFX"
    return string.lower(channel)
end

local function NormalizeSoundIdentifier(value)
    local numeric = tonumber(value or 0)
    if numeric then return tostring(numeric) end
    return CanonicalID(value) or "0"
end

-- Mirrors the argument layout and security fallback in ScriptEffects.lua.
-- Extra fields are intentionally exposed so the runtime integrator can
-- distinguish the normal local-broadcast path from the secured self fallback.
function SoundRules.ClassifyEffect(effectID, args)
    if not SOUND_EFFECT_IDS[effectID] then return nil end
    args = type(args) == "table" and args or {}

    if effectID == "sound_id_self" then
        local channel = NormalizeChannel(args[1])
        return {
            kind = "start",
            family = "sound_id_self:" .. channel,
            identifier = NormalizeSoundIdentifier(args[2]),
            continuous = false,
            channel = channel,
            isSoundFileID = not not args[3],
            secured = "HIGH",
            handleAvailableToWorkflow = false,
        }
    elseif effectID == "sound_id_stop" then
        local channel = NormalizeChannel(args[1])
        local identifier = NormalizeSoundIdentifier(args[2])
        return {
            kind = "stop",
            family = "sound_id_self:" .. channel,
            identifier = identifier,
            continuous = false,
            channel = channel,
            secured = "HIGH",
            -- TRP3 converts an omitted ID to numeric 0, while stopSoundID only
            -- treats nil or string "0" as stop-all. Numeric 0 is ineffective.
            effective = identifier ~= "0",
            intendedStopAll = identifier == "0",
        }
    elseif effectID == "sound_music_self" then
        return {
            kind = "start",
            family = "sound_music_self",
            identifier = CanonicalID(args[1]) or "",
            continuous = true,
            channel = "music",
            secured = "HIGH",
            handleAvailableToWorkflow = false,
        }
    elseif effectID == "sound_music_stop" then
        return {
            kind = "stop",
            family = "sound_music_self",
            continuous = true,
            channel = "music",
            secured = "HIGH",
            effective = true,
        }
    elseif effectID == "sound_id_local" then
        local channel = NormalizeChannel(args[1])
        return {
            kind = "start",
            family = "sound_id_local:" .. channel,
            securedFamily = "sound_id_self:" .. channel,
            identifier = NormalizeSoundIdentifier(args[2]),
            continuous = false,
            channel = channel,
            distance = tonumber(args[3]) or 0,
            isSoundFileID = not not args[4],
            secured = "MEDIUM",
            securedFallback = "self",
            handleAvailableToWorkflow = false,
        }
    elseif effectID == "sound_id_local_stop" then
        local channel = NormalizeChannel(args[1])
        local identifier = NormalizeSoundIdentifier(args[2])
        return {
            kind = "stop",
            family = "sound_id_local:" .. channel,
            identifier = identifier,
            continuous = false,
            channel = channel,
            secured = "HIGH",
            effective = identifier ~= "0",
            intendedStopAll = identifier == "0",
        }
    elseif effectID == "sound_music_local" then
        return {
            kind = "start",
            family = "sound_music_local",
            securedFamily = "sound_music_self",
            identifier = CanonicalID(args[1]) or "",
            continuous = true,
            channel = "music",
            distance = tonumber(args[2]) or 0,
            secured = "MEDIUM",
            securedFallback = "self",
            handleAvailableToWorkflow = false,
        }
    end

    return {
        kind = "stop",
        family = "sound_music_local",
        continuous = true,
        channel = "music",
        secured = "HIGH",
        effective = true,
    }
end

local function ProjectEffect(effect)
    if type(effect) ~= "table" then return { invalid = type(effect) } end
    local effectID = CanonicalID(effect.id) or ""
    local projected = { id = effectID }
    -- Never index Script Effect arguments. Only sound and same-object workflow
    -- call arguments affect this rule module's result.
    if SOUND_EFFECT_IDS[effectID] then
        local classified = SoundRules.ClassifyEffect(effectID, effect.args)
        projected.sound = classified
    elseif effectID == "run_workflow" then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = { args[1], args[2] }
    end
    return projected
end

local function ProjectClass(class, classID, seen)
    if type(class) ~= "table" then return nil end
    seen = seen or {}
    if seen[class] then return { id = classID, cycle = true } end
    seen[class] = true

    local projected = { id = classID, SC = {}, children = {} }
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
    local projection = {
        ruleVersion = SoundRules.RULE_VERSION,
        rootID = rootID,
        class = ProjectClass(root, rootID, {}),
    }
    local material = StableSerialize(projection)
    return "igs" .. SoundRules.RULE_VERSION .. ":" .. HashMaterial(material) .. ":" .. #material
end

local function AddEdge(edges, source, target)
    source, target = CanonicalID(source), CanonicalID(target)
    if not source or not target then return end
    edges[source] = edges[source] or {}
    edges[source][target] = true
end

local function MarkReachable(nodes, edges)
    local reachable, queue = {}, {}
    if nodes["1"] then
        reachable["1"] = true
        queue[1] = "1"
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
        stack[#stack + 1], onStack[node] = node, true
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
        hasSideEffect = false,
        metrics = {
            classes = 0,
            workflows = 0,
            reachableSteps = 0,
            disconnectedSteps = 0,
            effectsAnalyzed = 0,
            scriptEffects = 0,
            starts = 0,
            stops = 0,
            musicStarts = 0,
            repeatedStarts = 0,
            controlledRepeatedStarts = 0,
            uncontrolledRepeatedStarts = 0,
            ineffectiveStops = 0,
        },
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

local function SetBehaviorBand(result, band, score)
    local previous = result._behaviorBands[band] or 0
    if score > previous then
        result._behaviorBands[band] = score
        result.behaviorScore = result.behaviorScore + score - previous
    end
end

local function SetAmplificationBand(result, band, score)
    local previous = result._amplificationBands[band] or 0
    if score > previous then
        result._amplificationBands[band] = score
        local total = 0
        for _, value in pairs(result._amplificationBands) do total = total + value end
        result.amplificationScore = math.min(MAX_AMPLIFICATION_SCORE, total)
    end
end

local function CompatibleStop(startInfo, stopInfo)
    if not startInfo or not stopInfo or stopInfo.kind ~= "stop" or not stopInfo.effective then
        return false
    end
    if startInfo.family ~= stopInfo.family then return false end
    if string.sub(startInfo.family, 1, 8) == "sound_id" then
        return stopInfo.identifier == startInfo.identifier
    end
    return true
end

local function BuildWorkflow(workflowID, workflow)
    local steps = type(workflow) == "table" and workflow.ST or nil
    steps = type(steps) == "table" and steps or {}
    local descriptor = {
        id = workflowID,
        steps = steps,
        nodes = {},
        edges = {},
        reachable = {},
        effects = {},
        calls = {},
        cyclicComponentOf = {},
        cyclicComponents = {},
    }
    for rawStepID, step in pairs(steps) do
        local stepID = CanonicalID(rawStepID)
        if stepID and type(step) == "table" then
            descriptor.nodes[stepID] = true
            descriptor.edges[stepID] = descriptor.edges[stepID] or {}
            AddEdge(descriptor.edges, stepID, step.n)
            if step.t == "branch" and type(step.b) == "table" then
                for _, branch in pairs(step.b) do
                    if type(branch) == "table" then AddEdge(descriptor.edges, stepID, branch.n) end
                end
            end
        end
    end
    descriptor.reachable = MarkReachable(descriptor.nodes, descriptor.edges)

    for stepID in pairs(descriptor.reachable) do
        local step = steps[stepID] or steps[tonumber(stepID)]
        if step and step.t == "branch" and type(step.b) == "table" then
            for _, branch in pairs(step.b) do
                local target = type(branch) == "table" and CanonicalID(branch.failWorkflow) or nil
                if target then descriptor.calls[#descriptor.calls + 1] = target end
            end
        end
        if step and type(step.e) == "table" then
            for _, effect in pairs(step.e) do
                if type(effect) == "table" then
                    local effectID = CanonicalID(effect.id) or ""
                    local info = SoundRules.ClassifyEffect(effectID, effect.args)
                    descriptor.effects[#descriptor.effects + 1] = {
                        effectID = effectID,
                        stepID = stepID,
                        sound = info,
                    }
                    if effectID == "run_workflow" then
                        local args = type(effect.args) == "table" and effect.args or {}
                        if (CanonicalID(args[1]) or "o") == "o" then
                            local target = CanonicalID(args[2])
                            if target then descriptor.calls[#descriptor.calls + 1] = target end
                        end
                    end
                end
            end
        end
    end

    local components = StronglyConnected(descriptor.reachable, descriptor.edges)
    for _, component in ipairs(components) do
        if IsCyclicComponent(component, descriptor.edges) then
            descriptor.cyclicComponents[#descriptor.cyclicComponents + 1] = component
            for _, stepID in ipairs(component) do
                descriptor.cyclicComponentOf[stepID] = component
            end
        end
    end
    return descriptor
end

local function FindStopsInEffects(startInfo, effects, allowedSteps)
    local matching, stopNodes = nil, {}
    local securedFallbackMismatch = false
    for _, candidate in ipairs(effects) do
        if (not allowedSteps or allowedSteps[candidate.stepID]) and candidate.sound then
            if CompatibleStop(startInfo, candidate.sound) then
                matching = matching or candidate
                stopNodes[candidate.stepID] = true
                if startInfo.securedFamily and startInfo.securedFamily ~= candidate.sound.family then
                    securedFallbackMismatch = true
                end
            end
        end
    end
    return matching, stopNodes, securedFallbackMismatch
end

local function IsNodeCyclicWithoutStops(startNode, nodes, edges, stopNodes)
    if stopNodes[startNode] then return false end
    local filtered = {}
    for node in pairs(nodes) do
        if not stopNodes[node] then filtered[node] = true end
    end
    local components = StronglyConnected(filtered, edges)
    for _, component in ipairs(components) do
        if IsCyclicComponent(component, edges) then
            for _, node in ipairs(component) do
                if node == startNode then return true end
            end
        end
    end
    return false
end

local function AnalyzeRepeatedStart(result, classID, workflowID, descriptor, effectInfo,
                                    workflowComponent, descriptors, workflowEdges)
    local startInfo = effectInfo.sound
    local component = descriptor.cyclicComponentOf[effectInfo.stepID]
    local stop, securedFallbackMismatch
    local repeatedKind, amplification

    if component then
        local allowedSteps = {}
        for _, stepID in ipairs(component) do allowedSteps[stepID] = true end
        local stopNodes
        stop, stopNodes, securedFallbackMismatch = FindStopsInEffects(
            startInfo, descriptor.effects, allowedSteps
        )
        if stop and IsNodeCyclicWithoutStops(effectInfo.stepID, allowedSteps,
                descriptor.edges, stopNodes) then
            stop = nil
        end
        repeatedKind, amplification = "step_cycle", 20
    elseif workflowComponent then
        local effects = {}
        local workflowNodes = {}
        local stopWorkflows = {}
        for _, memberWorkflowID in ipairs(workflowComponent) do
            local member = descriptors[memberWorkflowID]
            workflowNodes[memberWorkflowID] = true
            for _, candidate in ipairs(member.effects) do
                effects[#effects + 1] = candidate
                if candidate.sound and CompatibleStop(startInfo, candidate.sound) then
                    stopWorkflows[memberWorkflowID] = true
                end
            end
        end
        local stopNodes
        stop, stopNodes, securedFallbackMismatch = FindStopsInEffects(startInfo, effects)
        if stop and IsNodeCyclicWithoutStops(workflowID, workflowNodes,
                workflowEdges, stopWorkflows) then
            stop = nil
        end
        repeatedKind, amplification = "workflow_recursion", 25
    else
        return
    end

    result.metrics.repeatedStarts = result.metrics.repeatedStarts + 1
    if stop then
        result.metrics.controlledRepeatedStarts = result.metrics.controlledRepeatedStarts + 1
        SetBehaviorBand(result, "controlled_sound_repeat", 15)
        SetAmplificationBand(result, repeatedKind, math.floor(amplification / 2))
        AddFinding(result, {
            kind = "sound_repeat_with_stop",
            classID = classID,
            workflowID = workflowID,
            stepID = effectInfo.stepID,
            effectID = effectInfo.effectID,
            family = startInfo.family,
            identifier = startInfo.identifier,
            score = 15,
            reason = "循环声音存在可达且匹配的停止效果；保留风险记录但不自动隔离",
            repeatedBy = repeatedKind,
            stopEffectID = stop.effectID,
            securedFallbackMismatch = securedFallbackMismatch,
        })
        if securedFallbackMismatch then
            AddFinding(result, {
                kind = "sound_local_secured_stop_mismatch",
                classID = classID,
                workflowID = workflowID,
                stepID = effectInfo.stepID,
                effectID = effectInfo.effectID,
                family = startInfo.family,
                score = 0,
                reason = "局部声音在未授权时会降级为自身播放，而局部停止效果不能保证停止该降级声音",
            })
        end
    else
        result.metrics.uncontrolledRepeatedStarts = result.metrics.uncontrolledRepeatedStarts + 1
        SetBehaviorBand(result, "uncontrolled_sound_repeat", 100)
        SetAmplificationBand(result, repeatedKind, amplification)
        AddFinding(result, {
            kind = "sound_unstoppable_repeat",
            classID = classID,
            workflowID = workflowID,
            stepID = effectInfo.stepID,
            effectID = effectInfo.effectID,
            family = startInfo.family,
            identifier = startInfo.identifier,
            continuous = startInfo.continuous,
            score = 100,
            reason = startInfo.continuous
                and "循环或递归会持续重启音乐，重复路径内没有有效的对应停止效果"
                or "循环或递归会重复启动声音，重复路径内没有有效的对应停止效果",
            repeatedBy = repeatedKind,
            handleAvailableToWorkflow = startInfo.handleAvailableToWorkflow,
        })
        if startInfo.continuous and not startInfo.handleAvailableToWorkflow then
            AddFinding(result, {
                kind = "sound_continuous_without_handle_or_stop",
                classID = classID,
                workflowID = workflowID,
                stepID = effectInfo.stepID,
                effectID = effectInfo.effectID,
                family = startInfo.family,
                score = 0,
                reason = "持续音乐不会向工作流提供播放句柄，且重复路径内没有可用的停止效果",
            })
        end
    end
end

local function AnalyzeClass(result, classID, class)
    local scripts = type(class.SC) == "table" and class.SC or nil
    if not scripts then return end
    result.metrics.classes = result.metrics.classes + 1

    local descriptors, workflowNodes, workflowEdges = {}, {}, {}
    for rawWorkflowID, workflow in pairs(scripts) do
        local workflowID = CanonicalID(rawWorkflowID)
        if workflowID and type(workflow) == "table" then
            descriptors[workflowID] = BuildWorkflow(workflowID, workflow)
            workflowNodes[workflowID] = true
            workflowEdges[workflowID] = {}
            result.metrics.workflows = result.metrics.workflows + 1
        end
    end
    for workflowID, descriptor in pairs(descriptors) do
        for _, target in ipairs(descriptor.calls) do
            if descriptors[target] then workflowEdges[workflowID][target] = true end
        end
    end

    local recursiveComponentOf = {}
    local workflowComponents = StronglyConnected(workflowNodes, workflowEdges)
    for _, component in ipairs(workflowComponents) do
        if IsCyclicComponent(component, workflowEdges) then
            for _, workflowID in ipairs(component) do recursiveComponentOf[workflowID] = component end
        end
    end

    for workflowID, descriptor in pairs(descriptors) do
        for stepID in pairs(descriptor.nodes) do
            if descriptor.reachable[stepID] then
                result.metrics.reachableSteps = result.metrics.reachableSteps + 1
            else
                result.metrics.disconnectedSteps = result.metrics.disconnectedSteps + 1
            end
        end
        for _, effectInfo in ipairs(descriptor.effects) do
            result.metrics.effectsAnalyzed = result.metrics.effectsAnalyzed + 1
            if effectInfo.effectID == "script" then
                result.metrics.scriptEffects = result.metrics.scriptEffects + 1
            elseif effectInfo.sound then
                if effectInfo.sound.kind == "start" then
                    result.metrics.starts = result.metrics.starts + 1
                    if effectInfo.sound.continuous then
                        result.metrics.musicStarts = result.metrics.musicStarts + 1
                    end
                    result.hasSideEffect = true
                    SetBehaviorBand(result, "ordinary_sound_start", 5)
                    AnalyzeRepeatedStart(result, classID, workflowID, descriptor, effectInfo,
                        recursiveComponentOf[workflowID], descriptors, workflowEdges)
                else
                    result.metrics.stops = result.metrics.stops + 1
                    if not effectInfo.sound.effective then
                        result.metrics.ineffectiveStops = result.metrics.ineffectiveStops + 1
                        AddFinding(result, {
                            kind = "sound_stop_all_ineffective",
                            classID = classID,
                            workflowID = workflowID,
                            stepID = effectInfo.stepID,
                            effectID = effectInfo.effectID,
                            score = 0,
                            reason = "停止声音未指定 ID；TRP3 当前会把它转换为无效的数值 0，不能可靠停止全部声音",
                        })
                    end
                end
            end
        end
    end
end

local function WalkClasses(result, classID, class, seen)
    if type(class) ~= "table" or seen[class] then return end
    seen[class] = true
    AnalyzeClass(result, classID, class)
    for _, groupName in ipairs(CHILD_GROUPS) do
        local group = class[groupName]
        if type(group) == "table" then
            for childKey, child in pairs(group) do
                WalkClasses(result, classID .. " " .. tostring(childKey), child, seen)
            end
        end
    end
end

function SoundRules.Analyze(rootID, root, context)
    rootID = CanonicalID(rootID) or "unknown"
    local result = NewResult(rootID)
    if type(root) == "table" then
        WalkClasses(result, rootID, root, {})
    else
        AddFinding(result, {
            kind = "invalid_root",
            classID = rootID,
            score = 0,
            reason = "无法分析声音风险：根对象不是有效表",
        })
    end

    result.score = result.behaviorScore + result.amplificationScore
    result.blocked = result.hasSideEffect and result.score >= BLOCK_SCORE
    result.fingerprint = BuildFingerprint(rootID, type(root) == "table" and root or {})
    result._reasonSet = nil
    result._behaviorBands = nil
    result._amplificationBands = nil
    return result
end
