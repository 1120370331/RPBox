-- Crash-level payload limits for TRP3 Extended documents and variables.
--
-- These limits intentionally sit above normal authoring sizes. They are a
-- last-resort guard against payloads large enough to overwhelm HTML parsing,
-- SavedVariables serialization, or variable interpolation.

local ADDON_NAME, ns = ...

local Rules = {}
ns.ItemGuardContentRules = Rules

Rules.RULE_VERSION = 1
Rules.LIMITS = {
    DOCUMENT_PAGE_BYTES = 512 * 1024,
    DOCUMENT_TOTAL_BYTES = 2 * 1024 * 1024,
    DOCUMENT_PAGES = 256,
    VARIABLE_VALUE_BYTES = 512 * 1024,
    VARIABLE_TOTAL_BYTES = 2 * 1024 * 1024,
    VARIABLE_ENTRIES = 2048,
    VARIABLE_NODES = 4096,
    VARIABLE_DEPTH = 32,
}

local CHILD_GROUPS = { "IN", "QE", "ST" }

local function CanonicalID(value)
    if type(value) == "string" or type(value) == "number" then
        local id = tostring(value)
        if id ~= "" then return id end
    end
    return nil
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
            documents = 0,
            documentPages = 0,
            documentBytes = 0,
            largestDocumentPageBytes = 0,
            variableTables = 0,
            variableEntries = 0,
            variableBytes = 0,
            largestVariableBytes = 0,
            variableNodes = 0,
        },
        _reasonSet = {},
        _signatures = {},
    }
end

local function AddHardFinding(result, finding)
    if not result._reasonSet[finding.reason] then
        result._reasonSet[finding.reason] = true
        result.reasons[#result.reasons + 1] = finding.reason
    end
    finding.score = 120
    finding.hard = true
    result.findings[#result.findings + 1] = finding
    result.blocked = true
    result.hasSideEffect = true
    result.behaviorScore = 120
    result.score = 120
end

local function ScalarBytes(value)
    local valueType = type(value)
    if valueType == "string" then return #value end
    if valueType == "number" or valueType == "boolean" then return #tostring(value) end
    return 0
end

local function ReferencedVariable(capture)
    local name = tostring(capture or "")
    local marker = name:find("::", 1, true)
    if marker then name = name:sub(1, marker - 1) end
    marker = name:find("#", 1, true)
    if marker then name = name:sub(1, marker - 1) end
    return name
end

local function InterpolatedBytes(text, vars, hardLimit)
    if type(text) ~= "string" then return 0 end
    if type(vars) ~= "table" then return #text end
    local total, cursor = 0, 1
    while cursor <= #text do
        local openAt = text:find("${", cursor, true)
        if not openAt then
            total = total + (#text - cursor + 1)
            break
        end
        total = total + (openAt - cursor)
        local closeAt = text:find("}", openAt + 2, true)
        if not closeAt then
            total = total + (#text - openAt + 1)
            break
        end
        local name = ReferencedVariable(text:sub(openAt + 2, closeAt - 1))
        local value = vars[name]
        total = total + (value ~= nil and ScalarBytes(value) or (closeAt - openAt + 1))
        cursor = closeAt + 1
        if hardLimit and total > hardLimit then break end
    end
    return total
end

-- MeasureValue estimates serialized payload size without constructing another
-- copy of the value. Cycles, excessive depth, and excessive table nodes are
-- reported as overflow instead of recursing indefinitely.
function Rules.MeasureValue(value, limits)
    limits = limits or Rules.LIMITS
    local seen = {}
    local metrics = { bytes = 0, nodes = 0, tooDeep = false, tooManyNodes = false, cyclic = false }
    local function Visit(current, depth)
        if metrics.tooDeep or metrics.tooManyNodes then return end
        metrics.nodes = metrics.nodes + 1
        if metrics.nodes > (tonumber(limits.VARIABLE_NODES) or 4096) then
            metrics.tooManyNodes = true
            return
        end
        local currentType = type(current)
        if currentType ~= "table" then
            metrics.bytes = metrics.bytes + ScalarBytes(current)
            return
        end
        if seen[current] then
            metrics.cyclic = true
            return
        end
        if depth > (tonumber(limits.VARIABLE_DEPTH) or 32) then
            metrics.tooDeep = true
            return
        end
        seen[current] = true
        for key, child in pairs(current) do
            Visit(key, depth + 1)
            Visit(child, depth + 1)
            if metrics.tooDeep or metrics.tooManyNodes then break end
        end
        seen[current] = nil
    end
    Visit(value, 1)
    return metrics
end

function Rules.AnalyzeVariables(vars, context)
    local result = NewResult(context and context.rootID or "variables")
    if type(vars) ~= "table" then
        result.fingerprint = "igc" .. Rules.RULE_VERSION .. ":variables:0:0:0"
        result._reasonSet, result._signatures = nil, nil
        return result
    end

    result.metrics.variableTables = 1
    for name, value in pairs(vars) do
        result.metrics.variableEntries = result.metrics.variableEntries + 1
        if result.metrics.variableEntries > Rules.LIMITS.VARIABLE_ENTRIES then
            AddHardFinding(result, {
                kind = "variable_entries_crash_size",
                entries = result.metrics.variableEntries,
                limit = Rules.LIMITS.VARIABLE_ENTRIES,
                reason = "变量条目数量超过崩溃防护上限 2048",
            })
            break
        end
        local measured = Rules.MeasureValue(value)
        local bytes = measured.bytes + ScalarBytes(name)
        result.metrics.variableBytes = result.metrics.variableBytes + bytes
        result.metrics.variableNodes = result.metrics.variableNodes + measured.nodes
        result.metrics.largestVariableBytes = math.max(result.metrics.largestVariableBytes, bytes)
        if bytes > Rules.LIMITS.VARIABLE_VALUE_BYTES then
            AddHardFinding(result, {
                kind = "variable_value_crash_size",
                variable = CanonicalID(name) or "dynamic",
                bytes = bytes,
                limit = Rules.LIMITS.VARIABLE_VALUE_BYTES,
                reason = "单个变量载荷超过崩溃防护上限 512 KiB",
            })
        end
        if measured.tooDeep or measured.tooManyNodes or measured.cyclic then
            AddHardFinding(result, {
                kind = measured.cyclic and "variable_cycle_crash_size"
                    or measured.tooDeep and "variable_depth_crash_size"
                    or "variable_nodes_crash_size",
                variable = CanonicalID(name) or "dynamic",
                nodes = measured.nodes,
                reason = measured.cyclic
                        and "变量结构包含循环引用，已按崩溃风险阻止"
                    or measured.tooDeep
                        and "变量结构嵌套超过崩溃防护上限"
                    or "变量结构节点超过崩溃防护上限 4096",
            })
        end
        if result.blocked
            or result.metrics.variableBytes > Rules.LIMITS.VARIABLE_TOTAL_BYTES
            or result.metrics.variableNodes > Rules.LIMITS.VARIABLE_NODES then
            break
        end
    end
    if result.metrics.variableBytes > Rules.LIMITS.VARIABLE_TOTAL_BYTES then
        AddHardFinding(result, {
            kind = "variable_total_crash_size",
            bytes = result.metrics.variableBytes,
            limit = Rules.LIMITS.VARIABLE_TOTAL_BYTES,
            reason = "变量载荷累计超过崩溃防护上限 2 MiB",
        })
    end
    if result.metrics.variableNodes > Rules.LIMITS.VARIABLE_NODES then
        AddHardFinding(result, {
            kind = "variable_total_nodes_crash_size",
            nodes = result.metrics.variableNodes,
            limit = Rules.LIMITS.VARIABLE_NODES,
            reason = "变量结构累计节点超过崩溃防护上限 4096",
        })
    end
    result.fingerprint = table.concat({
        "igc" .. Rules.RULE_VERSION,
        "variables",
        tostring(result.metrics.variableEntries),
        tostring(result.metrics.variableBytes),
        tostring(result.metrics.largestVariableBytes),
        tostring(result.metrics.variableNodes),
    }, ":")
    result._reasonSet, result._signatures = nil, nil
    return result
end

function Rules.AnalyzeRenderedDocument(document, vars, context)
    local result = NewResult(context and context.rootID or "document-render")
    if type(document) ~= "table" then
        result.fingerprint = "igc" .. Rules.RULE_VERSION .. ":render:0:0"
        result._reasonSet, result._signatures = nil, nil
        return result
    end
    local pageCount, totalBytes, largestBytes = 0, 0, 0
    for _, page in pairs(type(document.PA) == "table" and document.PA or {}) do
        if type(page) == "table" then
            pageCount = pageCount + 1
            if pageCount > Rules.LIMITS.DOCUMENT_PAGES then break end
            local bytes = InterpolatedBytes(
                page.TX,
                vars,
                Rules.LIMITS.DOCUMENT_TOTAL_BYTES - totalBytes
            )
            totalBytes = totalBytes + bytes
            largestBytes = math.max(largestBytes, bytes)
            if bytes > Rules.LIMITS.DOCUMENT_PAGE_BYTES then
                AddHardFinding(result, {
                    kind = "document_rendered_page_crash_size",
                    bytes = bytes,
                    limit = Rules.LIMITS.DOCUMENT_PAGE_BYTES,
                    reason = "变量展开后的文档单页超过崩溃防护上限 512 KiB",
                })
                break
            end
            if totalBytes > Rules.LIMITS.DOCUMENT_TOTAL_BYTES then break end
        end
    end
    if totalBytes > Rules.LIMITS.DOCUMENT_TOTAL_BYTES then
        AddHardFinding(result, {
            kind = "document_rendered_total_crash_size",
            bytes = totalBytes,
            limit = Rules.LIMITS.DOCUMENT_TOTAL_BYTES,
            reason = "变量展开后的文档内容累计超过崩溃防护上限 2 MiB",
        })
    end
    result.metrics.documentPages = pageCount
    result.metrics.documentBytes = totalBytes
    result.metrics.largestDocumentPageBytes = largestBytes
    result.fingerprint = table.concat({
        "igc" .. Rules.RULE_VERSION,
        "render",
        tostring(pageCount),
        tostring(totalBytes),
        tostring(largestBytes),
    }, ":")
    result._reasonSet, result._signatures = nil, nil
    return result
end

local function MergeVariableResult(result, classID, variableResult)
    local metrics = variableResult.metrics or {}
    result.metrics.variableTables = result.metrics.variableTables + (metrics.variableTables or 0)
    result.metrics.variableEntries = result.metrics.variableEntries + (metrics.variableEntries or 0)
    result.metrics.variableBytes = result.metrics.variableBytes + (metrics.variableBytes or 0)
    result.metrics.largestVariableBytes = math.max(
        result.metrics.largestVariableBytes,
        metrics.largestVariableBytes or 0
    )
    result.metrics.variableNodes = result.metrics.variableNodes + (metrics.variableNodes or 0)
    result._signatures[#result._signatures + 1] = table.concat({
        "v", classID, tostring(metrics.variableEntries or 0), tostring(metrics.variableBytes or 0),
        tostring(metrics.largestVariableBytes or 0), tostring(metrics.variableNodes or 0),
    }, "|")
    for _, finding in ipairs(variableResult.findings or {}) do
        finding.classID = finding.classID or classID
        AddHardFinding(result, finding)
    end
end

local function AnalyzeDocument(result, classID, class)
    result.metrics.documents = result.metrics.documents + 1
    local pageCount, totalBytes, largestBytes = 0, 0, 0
    for _, page in pairs(type(class.PA) == "table" and class.PA or {}) do
        if type(page) == "table" then
            pageCount = pageCount + 1
            if pageCount > Rules.LIMITS.DOCUMENT_PAGES then break end
            local bytes = type(page.TX) == "string" and #page.TX or 0
            totalBytes = totalBytes + bytes
            largestBytes = math.max(largestBytes, bytes)
            if bytes > Rules.LIMITS.DOCUMENT_PAGE_BYTES then
                AddHardFinding(result, {
                    kind = "document_page_crash_size",
                    classID = classID,
                    bytes = bytes,
                    limit = Rules.LIMITS.DOCUMENT_PAGE_BYTES,
                    reason = "文档单页内容超过崩溃防护上限 512 KiB",
                })
            end
        end
    end
    local legacyBytes = type(class.BA) == "table" and type(class.BA.TX) == "string" and #class.BA.TX or 0
    if legacyBytes > 0 then
        pageCount = math.max(pageCount, 1)
        totalBytes = totalBytes + legacyBytes
        largestBytes = math.max(largestBytes, legacyBytes)
        if legacyBytes > Rules.LIMITS.DOCUMENT_PAGE_BYTES then
            AddHardFinding(result, {
                kind = "document_legacy_page_crash_size",
                classID = classID,
                bytes = legacyBytes,
                limit = Rules.LIMITS.DOCUMENT_PAGE_BYTES,
                reason = "文档正文内容超过崩溃防护上限 512 KiB",
            })
        end
    end
    if pageCount > Rules.LIMITS.DOCUMENT_PAGES then
        AddHardFinding(result, {
            kind = "document_page_count_crash_size",
            classID = classID,
            pages = pageCount,
            limit = Rules.LIMITS.DOCUMENT_PAGES,
            reason = "文档页数超过崩溃防护上限 256 页",
        })
    end
    if totalBytes > Rules.LIMITS.DOCUMENT_TOTAL_BYTES then
        AddHardFinding(result, {
            kind = "document_total_crash_size",
            classID = classID,
            bytes = totalBytes,
            limit = Rules.LIMITS.DOCUMENT_TOTAL_BYTES,
            reason = "文档内容累计超过崩溃防护上限 2 MiB",
        })
    end
    result.metrics.documentPages = result.metrics.documentPages + pageCount
    result.metrics.documentBytes = result.metrics.documentBytes + totalBytes
    result.metrics.largestDocumentPageBytes = math.max(result.metrics.largestDocumentPageBytes, largestBytes)
    result._signatures[#result._signatures + 1] = table.concat({
        "d", classID, tostring(pageCount), tostring(totalBytes), tostring(largestBytes),
    }, "|")
end

local function WalkClasses(result, classID, class, documentType, seen)
    if type(class) ~= "table" or seen[class] then return end
    seen[class] = true
    if documentType and class.TY == documentType then AnalyzeDocument(result, classID, class) end
    if type(class.vars) == "table" then
        MergeVariableResult(result, classID, Rules.AnalyzeVariables(class.vars, { rootID = result.rootID }))
    end
    for _, groupName in ipairs(CHILD_GROUPS) do
        local group = class[groupName]
        if type(group) == "table" then
            for childID, child in pairs(group) do
                WalkClasses(result, classID .. " " .. tostring(childID), child, documentType, seen)
            end
        end
    end
    seen[class] = nil
end

function Rules.Analyze(rootID, root, context)
    rootID = CanonicalID(rootID) or "unknown"
    local result = NewResult(rootID)
    local documentType = TRP3_DB and TRP3_DB.types and TRP3_DB.types.DOCUMENT or "DO"
    if type(root) == "table" then WalkClasses(result, rootID, root, documentType, {}) end
    if result.metrics.documentBytes > Rules.LIMITS.DOCUMENT_TOTAL_BYTES then
        AddHardFinding(result, {
            kind = "document_root_total_crash_size",
            bytes = result.metrics.documentBytes,
            limit = Rules.LIMITS.DOCUMENT_TOTAL_BYTES,
            reason = "对象内文档内容累计超过崩溃防护上限 2 MiB",
        })
    end
    if result.metrics.variableBytes > Rules.LIMITS.VARIABLE_TOTAL_BYTES then
        AddHardFinding(result, {
            kind = "variable_root_total_crash_size",
            bytes = result.metrics.variableBytes,
            limit = Rules.LIMITS.VARIABLE_TOTAL_BYTES,
            reason = "对象内变量载荷累计超过崩溃防护上限 2 MiB",
        })
    end
    if result.metrics.variableNodes > Rules.LIMITS.VARIABLE_NODES then
        AddHardFinding(result, {
            kind = "variable_root_nodes_crash_size",
            nodes = result.metrics.variableNodes,
            limit = Rules.LIMITS.VARIABLE_NODES,
            reason = "对象内变量结构累计节点超过崩溃防护上限 4096",
        })
    end
    table.sort(result._signatures)
    result.fingerprint = "igc" .. Rules.RULE_VERSION .. ":" .. table.concat(result._signatures, ";")
    result._reasonSet, result._signatures = nil, nil
    return result
end
