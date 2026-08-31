-- Static and runtime risk rules for TRP3 Extended persistent variables.
-- Lua Script Effect payloads are deliberately opaque to this module.

local ADDON_NAME, ns = ...

local Rules = {}
ns.ItemGuardVariableRules = Rules

Rules.RULE_VERSION = 2
Rules.LIMITS = {
    SHORT_WINDOW_SECONDS = 5,
    SHORT_WINDOW_WRITES = 100,
    SHORT_WINDOW_BYTES = 256 * 1024,
    LONG_WINDOW_SECONDS = 60,
    LONG_WINDOW_WRITES = 1000,
    LONG_WINDOW_BYTES = 1024 * 1024,
    STATIC_UNIQUE_KEYS = 64,
    RUNTIME_UNIQUE_KEYS = 256,
    RUNTIME_SINGLE_VALUE_BYTES = 512 * 1024,
    SINGLE_LITERAL_BYTES = 64 * 1024,
    TOTAL_LITERAL_BYTES = 256 * 1024,
}

local CHILD_GROUPS = { "IN", "QE", "ST" }
local BLOCK_SCORE = 100
local MAX_AMPLIFICATION_SCORE = 40

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
    return string.format("%04x%04x", math.floor(hash / 65536), hash % 65536)
end

local function ContainsInterpolation(value)
    return type(value) == "string" and value:find("%${.-}") ~= nil
end

local function NormalizeName(value)
    if value == nil then return "var" end
    if type(value) == "string" or type(value) == "number" then return tostring(value) end
    return nil
end

local function PersistentSource(source)
    return source == "o" or source == "c"
end

local function ReferencedVariable(capture)
    local name = capture
    local marker = name:find("::", 1, true)
    if marker then name = name:sub(1, marker - 1) end
    marker = name:find("#", 1, true)
    if marker then name = name:sub(1, marker - 1) end
    return name
end

local function GrowsStoredValue(operation, name, value)
    if operation ~= "=" or not name or type(value) ~= "string" then return false end
    local referencesSelf, referencesTotal = 0, 0
    for capture in value:gmatch("%${(.-)}") do
        referencesTotal = referencesTotal + 1
        if ReferencedVariable(capture) == name then
            referencesSelf = referencesSelf + 1
        end
    end
    if referencesSelf == 0 then return false end
    local literal = value:gsub("%${.-}", "")
    return referencesSelf > 1 or referencesTotal > 1 or #literal > 0
end

function Rules.ClassifyEffect(effectID, args)
    effectID = CanonicalID(effectID)
    if effectID ~= "var_object" and effectID ~= "var_operand" and effectID ~= "var_prompt" then
        return nil
    end
    args = type(args) == "table" and args or {}

    local source, rawName, operation, rawValue, kind
    local interactive, dynamicValue = false, false
    if effectID == "var_object" then
        source = CanonicalID(args[1]) or "w"
        operation = CanonicalID(args[2]) or "i"
        rawName = args[3]
        if rawName == nil then rawName = "var" end
        rawValue = args[4]
        if rawValue == nil then rawValue = "0" end
        kind = "literal"
        dynamicValue = ContainsInterpolation(rawValue)
    elseif effectID == "var_operand" then
        rawName = args[1]
        if rawName == nil then rawName = "var" end
        source = CanonicalID(args[2]) or "w"
        operation = "="
        rawValue = args[4]
        kind = "operand"
        dynamicValue = true
    else
        rawName = args[2]
        if rawName == nil then rawName = "var" end
        source = CanonicalID(args[3]) or "o"
        operation = "="
        kind = "prompt"
        interactive = true
        dynamicValue = true
    end

    local name = NormalizeName(rawName)
    local estimatedBytes
    if effectID == "var_object" then
        if type(rawValue) == "string" then
            estimatedBytes = #rawValue
        elseif type(rawValue) == "number" or type(rawValue) == "boolean" then
            estimatedBytes = #tostring(rawValue)
        end
    end

    return {
        effectID = effectID,
        kind = kind,
        source = source,
        name = name,
        rawName = rawName,
        operation = operation,
        dynamicName = name == nil or ContainsInterpolation(rawName),
        dynamicValue = dynamicValue,
        estimatedBytes = estimatedBytes,
        persistent = PersistentSource(source),
        interactive = interactive,
        growsValue = GrowsStoredValue(operation, name, rawValue),
    }
end

local function AddEdge(edges, source, target)
    source, target = CanonicalID(source), CanonicalID(target)
    if not source or not target then return end
    edges[source] = edges[source] or {}
    edges[source][target] = true
end

local function GetStep(steps, stepID)
    if type(steps) ~= "table" or not stepID then return nil end
    return steps[stepID] or steps[tonumber(stepID)]
end

local function MarkReachable(starts, nodes, edges)
    local reachable, queue = {}, {}
    for start in pairs(starts or {}) do
        if nodes[start] and not reachable[start] then
            reachable[start] = true
            queue[#queue + 1] = start
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
    local nextIndex = 0
    local indexes, lowLinks, onStack, stack = {}, {}, {}, {}
    local components = {}

    local function Visit(node)
        nextIndex = nextIndex + 1
        indexes[node], lowLinks[node] = nextIndex, nextIndex
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
                stack[#stack], onStack[member] = nil, nil
                component[#component + 1] = member
                if member == node then break end
            end
            table.sort(component)
            components[#components + 1] = component
        end
    end

    for _, node in ipairs(SortedKeys(nodes)) do
        if nodes[node] and not indexes[node] then Visit(node) end
    end
    return components
end

local function IsCyclic(component, edges)
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

local function BuildDescriptor(workflowID, workflow)
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
    descriptor.reachable = MarkReachable({ ["1"] = true }, descriptor.nodes, descriptor.edges)
    for stepID in pairs(descriptor.reachable) do
        local step = GetStep(steps, stepID)
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
                    if effectID == "var_object" or effectID == "var_operand" or effectID == "var_prompt" then
                        descriptor.effects[#descriptor.effects + 1] = {
                            stepID = stepID,
                            effectID = effectID,
                            -- Do not read args for unrelated effects, especially Script Effects.
                            args = effect.args,
                        }
                    elseif effectID == "run_workflow" then
                        local effectArgs = type(effect.args) == "table" and effect.args or {}
                        if (CanonicalID(effectArgs[1]) or "o") == "o" then
                            local target = CanonicalID(effectArgs[2])
                            if target then descriptor.calls[#descriptor.calls + 1] = target end
                        end
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
        local entry = CanonicalID(class.US.SC)
        if entry and descriptors[entry] then entries[entry] = true end
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
            for candidate, enabled in pairs(configured) do
                candidate = CanonicalID(candidate)
                if enabled and candidate and descriptors[candidate] then entries[candidate] = true end
            end
        end
    end
    if next(entries) then return entries, false end
    for workflowID, descriptor in pairs(descriptors) do
        if descriptor.nodes["1"] then entries[workflowID] = true end
    end
    return entries, true
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
        hasSideEffect = false,
        metrics = {
            classes = 0,
            workflowsAnalyzed = 0,
            disconnectedWorkflows = 0,
            stepsAnalyzed = 0,
            disconnectedSteps = 0,
            effectsAnalyzed = 0,
            temporaryWrites = 0,
            persistentWrites = 0,
            objectWrites = 0,
            campaignWrites = 0,
            promptWrites = 0,
            operandWrites = 0,
            repeatedPersistentWrites = 0,
            dynamicNames = 0,
            growingWrites = 0,
            uniquePersistentKeys = 0,
            literalBytes = 0,
            largestLiteralBytes = 0,
            fallbackClasses = 0,
        },
        _reasonSet = {},
        _hardBlocked = false,
        _behaviorBands = {},
        _amplificationBands = {},
        _keys = {},
    }
end

local function AddReason(result, reason)
    if reason and not result._reasonSet[reason] then
        result._reasonSet[reason] = true
        result.reasons[#result.reasons + 1] = reason
    end
end

local function AddFinding(result, finding)
    finding.score = tonumber(finding.score) or 0
    result.findings[#result.findings + 1] = finding
    AddReason(result, finding.reason)
    if finding.hard then result._hardBlocked = true end
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

local function AddAmplification(result, band, score)
    SetBand(result, "amplificationScore", "_amplificationBands", band, score)
    if result.amplificationScore > MAX_AMPLIFICATION_SCORE then
        result.amplificationScore = MAX_AMPLIFICATION_SCORE
    end
end

local function AddBehavior(result, band, score)
    SetBand(result, "behaviorScore", "_behaviorBands", band, score)
end

local function RecordPersistentWrite(result, classID, workflowID, stepID, info, stepRepeated, workflowRepeated)
    local repeated = stepRepeated or workflowRepeated
    result.hasSideEffect = true
    result.metrics.persistentWrites = result.metrics.persistentWrites + 1
    if info.source == "o" then
        result.metrics.objectWrites = result.metrics.objectWrites + 1
    elseif info.source == "c" then
        result.metrics.campaignWrites = result.metrics.campaignWrites + 1
    end
    if info.kind == "prompt" then result.metrics.promptWrites = result.metrics.promptWrites + 1 end
    if info.kind == "operand" then result.metrics.operandWrites = result.metrics.operandWrites + 1 end
    if repeated and not info.interactive then
        result.metrics.repeatedPersistentWrites = result.metrics.repeatedPersistentWrites + 1
        if stepRepeated then AddAmplification(result, "persistent_step_cycle", 15) end
        if workflowRepeated then AddAmplification(result, "persistent_workflow_recursion", 20) end
    end
    if info.dynamicName then result.metrics.dynamicNames = result.metrics.dynamicNames + 1 end
    if info.growsValue then result.metrics.growingWrites = result.metrics.growingWrites + 1 end

    if info.name and not info.dynamicName then
        result._keys[info.source .. ":" .. info.name] = true
    end
    if info.estimatedBytes then
        result.metrics.literalBytes = result.metrics.literalBytes + info.estimatedBytes
        result.metrics.largestLiteralBytes = math.max(result.metrics.largestLiteralBytes, info.estimatedBytes)
        if info.estimatedBytes > Rules.LIMITS.SINGLE_LITERAL_BYTES then
            AddBehavior(result, "single_literal_exhaustion", 120)
            AddFinding(result, {
                kind = "variable_single_literal_exhaustion",
                effectID = info.effectID,
                classID = classID,
                workflowID = workflowID,
                stepID = stepID,
                score = 120,
                hard = true,
                bytes = info.estimatedBytes,
                reason = "单次持久变量字面值超过 64 KiB",
            })
        end
    end

    if repeated and not info.interactive then
        if info.dynamicName then
            AddBehavior(result, "dynamic_key_growth", 100)
            AddFinding(result, {
                kind = "variable_dynamic_key_amplified",
                effectID = info.effectID,
                classID = classID,
                workflowID = workflowID,
                stepID = stepID,
                score = 100,
                reason = "循环或递归路径会使用动态名称写入持久变量",
            })
        elseif info.growsValue then
            AddBehavior(result, "stored_value_growth", 100)
            AddFinding(result, {
                kind = "variable_value_growth_amplified",
                effectID = info.effectID,
                classID = classID,
                workflowID = workflowID,
                stepID = stepID,
                score = 100,
                reason = "循环或递归路径会把持久变量旧值拼接回新值并持续增长",
            })
        end
    end
end

local function AnalyzeClass(result, classID, class, context)
    result.metrics.classes = result.metrics.classes + 1
    local descriptors, workflowNodes, workflowEdges = {}, {}, {}
    if type(class.SC) == "table" then
        for rawWorkflowID, workflow in pairs(class.SC) do
            local workflowID = CanonicalID(rawWorkflowID)
            if workflowID and type(workflow) == "table" then
                descriptors[workflowID] = BuildDescriptor(workflowID, workflow)
                workflowNodes[workflowID] = true
                workflowEdges[workflowID] = {}
            end
        end
    end
    for workflowID, descriptor in pairs(descriptors) do
        for _, target in ipairs(descriptor.calls) do AddEdge(workflowEdges, workflowID, target) end
    end

    local entries, fallback = CollectEntries(class, descriptors, classID, context)
    if fallback and next(descriptors) then result.metrics.fallbackClasses = result.metrics.fallbackClasses + 1 end
    local reachableWorkflows = MarkReachable(entries, workflowNodes, workflowEdges)
    for workflowID in pairs(workflowNodes) do
        if reachableWorkflows[workflowID] then
            result.metrics.workflowsAnalyzed = result.metrics.workflowsAnalyzed + 1
        else
            result.metrics.disconnectedWorkflows = result.metrics.disconnectedWorkflows + 1
        end
    end

    local recursiveWorkflows = {}
    for _, component in ipairs(StronglyConnected(reachableWorkflows, workflowEdges)) do
        if IsCyclic(component, workflowEdges) then
            for _, workflowID in ipairs(component) do recursiveWorkflows[workflowID] = true end
        end
    end

    for workflowID in pairs(reachableWorkflows) do
        local descriptor = descriptors[workflowID]
        local cyclicSteps = {}
        for _, component in ipairs(StronglyConnected(descriptor.reachable, descriptor.edges)) do
            if IsCyclic(component, descriptor.edges) then
                for _, stepID in ipairs(component) do cyclicSteps[stepID] = true end
            end
        end
        for stepID in pairs(descriptor.nodes) do
            if descriptor.reachable[stepID] then
                result.metrics.stepsAnalyzed = result.metrics.stepsAnalyzed + 1
            else
                result.metrics.disconnectedSteps = result.metrics.disconnectedSteps + 1
            end
        end
        for _, effectInfo in ipairs(descriptor.effects) do
            result.metrics.effectsAnalyzed = result.metrics.effectsAnalyzed + 1
            local info = Rules.ClassifyEffect(effectInfo.effectID, effectInfo.args)
            if info then
                if not info.persistent then
                    result.metrics.temporaryWrites = result.metrics.temporaryWrites + 1
                else
                    RecordPersistentWrite(
                        result,
                        classID,
                        workflowID,
                        effectInfo.stepID,
                        info,
                        cyclicSteps[effectInfo.stepID] or false,
                        recursiveWorkflows[workflowID] or false
                    )
                end
            end
        end
    end
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

local function ProjectEffect(effect)
    if type(effect) ~= "table" then return { invalid = type(effect) } end
    local effectID = CanonicalID(effect.id) or ""
    local projected = { id = effectID }
    if effectID == "var_object" or effectID == "var_operand" or effectID == "var_prompt"
        or effectID == "run_workflow" then
        local args = type(effect.args) == "table" and effect.args or {}
        projected.args = {}
        local count = effectID == "var_object" and 4
            or effectID == "var_operand" and 4
            or effectID == "var_prompt" and 5
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
        ruleVersion = Rules.RULE_VERSION,
        rootID = rootID,
        class = ProjectClass(root, rootID, {}),
    }
    local material = StableSerialize(projection)
    return "igv" .. tostring(Rules.RULE_VERSION) .. ":" .. HashMaterial(material) .. ":" .. tostring(#material)
end

function Rules.EvaluateRuntime(stats)
    stats = type(stats) == "table" and stats or {}
    local findings, reasons, score = {}, {}, 0
    local function Trip(kind, reason, actual, limit)
        score = 120
        findings[#findings + 1] = {
            kind = kind,
            score = 120,
            hard = true,
            reason = reason,
            actual = actual,
            limit = limit,
        }
        reasons[#reasons + 1] = reason
    end
    if (tonumber(stats.singleBytes) or 0) > Rules.LIMITS.RUNTIME_SINGLE_VALUE_BYTES then
        Trip("variable_runtime_single_crash_size", "单个运行时变量载荷超过崩溃防护上限 512 KiB",
            stats.singleBytes, Rules.LIMITS.RUNTIME_SINGLE_VALUE_BYTES)
    end
    if (tonumber(stats.shortWrites) or 0) > Rules.LIMITS.SHORT_WINDOW_WRITES then
        Trip("variable_runtime_short_writes", "5 秒内持久变量写入次数超过限制",
            stats.shortWrites, Rules.LIMITS.SHORT_WINDOW_WRITES)
    end
    if (tonumber(stats.shortBytes) or 0) > Rules.LIMITS.SHORT_WINDOW_BYTES then
        Trip("variable_runtime_short_bytes", "5 秒内持久变量写入字节超过限制",
            stats.shortBytes, Rules.LIMITS.SHORT_WINDOW_BYTES)
    end
    if (tonumber(stats.longWrites) or 0) > Rules.LIMITS.LONG_WINDOW_WRITES then
        Trip("variable_runtime_long_writes", "60 秒内持久变量写入次数超过限制",
            stats.longWrites, Rules.LIMITS.LONG_WINDOW_WRITES)
    end
    if (tonumber(stats.longBytes) or 0) > Rules.LIMITS.LONG_WINDOW_BYTES then
        Trip("variable_runtime_long_bytes", "60 秒内持久变量写入字节超过限制",
            stats.longBytes, Rules.LIMITS.LONG_WINDOW_BYTES)
    end
    if (tonumber(stats.uniqueKeys) or 0) > Rules.LIMITS.RUNTIME_UNIQUE_KEYS then
        Trip("variable_runtime_unique_keys", "运行时持久变量唯一键数量超过限制",
            stats.uniqueKeys, Rules.LIMITS.RUNTIME_UNIQUE_KEYS)
    end
    return {
        blocked = score >= 120,
        observationScore = score,
        findings = findings,
        reasons = reasons,
    }
end

function Rules.Analyze(rootID, root, context)
    rootID = CanonicalID(rootID) or "unknown"
    local result = NewResult(rootID)
    if type(root) == "table" then
        WalkClasses(result, rootID, root, context, {})
    else
        AddFinding(result, {
            kind = "invalid_root",
            score = 0,
            reason = "无法分析持久变量：根对象不是有效表",
        })
    end

    for _ in pairs(result._keys) do
        result.metrics.uniquePersistentKeys = result.metrics.uniquePersistentKeys + 1
    end
    if result.metrics.uniquePersistentKeys > Rules.LIMITS.STATIC_UNIQUE_KEYS then
        AddBehavior(result, "unique_key_exhaustion", 120)
        AddFinding(result, {
            kind = "variable_unique_key_exhaustion",
            score = 120,
            hard = true,
            count = result.metrics.uniquePersistentKeys,
            reason = "可达工作流包含超过 64 个独立持久变量键",
        })
    end
    if result.metrics.literalBytes > Rules.LIMITS.TOTAL_LITERAL_BYTES then
        AddBehavior(result, "total_literal_exhaustion", 120)
        AddFinding(result, {
            kind = "variable_total_literal_exhaustion",
            score = 120,
            hard = true,
            bytes = result.metrics.literalBytes,
            reason = "可达工作流持久变量字面值累计超过 256 KiB",
        })
    end

    if type(context) == "table" and type(context.runtime) == "table" then
        local runtime = Rules.EvaluateRuntime(context.runtime)
        result.observationScore = runtime.observationScore
        for _, finding in ipairs(runtime.findings) do AddFinding(result, finding) end
    end
    result.score = result.behaviorScore + result.amplificationScore + result.observationScore
    result.blocked = result._hardBlocked
        or result.observationScore >= 120
        or (result.hasSideEffect and result.score >= BLOCK_SCORE)
    result.fingerprint = BuildFingerprint(rootID, type(root) == "table" and root or {})

    result._reasonSet = nil
    result._hardBlocked = nil
    result._behaviorBands = nil
    result._amplificationBands = nil
    result._keys = nil
    return result
end
