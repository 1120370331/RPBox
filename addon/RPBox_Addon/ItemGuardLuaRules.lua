-- Static policy for TRP3 Extended raw Lua Script Effects and secure macros.
--
-- The scanner is token-based so comments and string contents do not fabricate
-- loop/global-write findings. Publisher trust may suppress advisory and
-- combination policy, but never the non-bypassable invariants marked hard.

local ADDON_NAME, ns = ...

local Rules = {}
ns.ItemGuardLuaRules = Rules

Rules.RULE_VERSION = 1
Rules.LIMITS = {
    SCRIPT_SOURCE_BYTES = 512 * 1024,
    OBJECT_SOURCE_BYTES = 2 * 1024 * 1024,
    LITERAL_LOOP_ITERATIONS = 1000000,
    AMPLIFIED_LOOP_ITERATIONS = 10000,
    STRING_REP_BYTES = 2 * 1024 * 1024,
    RUNTIME_DEPTH = 12,
    RUNTIME_CALLS = 40,
    RUNTIME_BYTES = 2 * 1024 * 1024,
}

local CHILD_GROUPS = { "IN", "QE", "ST" }
local SHARED_LIBRARIES = { string = true, table = true, math = true }
local HIGH_IMPACT_EFFECTS = {
    item_add = true,
    item_loot = true,
    item_use = true,
    aura_apply = true,
    aura_duration = true,
    aura_remove = true,
    aura_run_workflow = true,
    document_show = true,
    secure_macro = true,
    signal_send = true,
    speech_env = true,
    speech_npc = true,
    speech_player = true,
    do_emote = true,
    sound_id_self = true,
    sound_id_local = true,
    sound_music_self = true,
    sound_music_local = true,
    var_prompt = true,
}
local ESCAPE_IDENTIFIERS = {
    _G = true,
    getfenv = true,
    setfenv = true,
    load = true,
    loadstring = true,
    dofile = true,
    require = true,
    debug = true,
    getglobal = true,
    setglobal = true,
    hooksecurefunc = true,
    CreateFrame = true,
    C_Timer = true,
    RunScript = true,
}

local function CanonicalID(value)
    if type(value) == "string" or type(value) == "number" then
        local id = tostring(value)
        if id ~= "" then return id end
    end
    return nil
end

local function StableHash(value)
    value = tostring(value or "")
    local hash = 5381
    for index = 1, #value do
        hash = (hash * 33 + string.byte(value, index)) % 4294967296
    end
    return string.format("%04x%04x", math.floor(hash / 65536), hash % 65536)
end

local function NewResult(rootID, trustedPublisher)
    return {
        rootID = rootID,
        blocked = false,
        advisory = false,
        trustedPublisher = trustedPublisher,
        score = 0,
        behaviorScore = 0,
        amplificationScore = 0,
        observationScore = 0,
        hasSideEffect = false,
        reasons = {},
        findings = {},
        metrics = {
            scripts = 0,
            secureMacros = 0,
            sourceBytes = 0,
            loops = 0,
            literalLoopMax = 0,
            effectCalls = 0,
            highImpactEffectCalls = 0,
            dynamicEffectCalls = 0,
            operandCalls = 0,
            unsafeOperandCalls = 0,
            sharedLibraryWrites = 0,
            contextWrites = 0,
            sandboxEscapeReferences = 0,
        },
        _reasonSet = {},
        _signatures = {},
    }
end

local function AddFinding(result, finding, level)
    if result.trustedPublisher and (level == "policy" or level == "advisory") then return end
    if not result._reasonSet[finding.reason] then
        result._reasonSet[finding.reason] = true
        result.reasons[#result.reasons + 1] = finding.reason
    end
    finding.score = level == "hard" and 120 or level == "policy" and 100 or 20
    finding.hard = level == "hard" or level == "policy"
    finding.bypassable = level ~= "hard"
    result.findings[#result.findings + 1] = finding
    if level == "hard" or level == "policy" then
        result.blocked = true
        result.hasSideEffect = true
        result.behaviorScore = math.max(result.behaviorScore, finding.score)
    else
        result.advisory = true
        result.observationScore = math.max(result.observationScore, finding.score)
    end
    result.score = result.behaviorScore + result.amplificationScore + result.observationScore
end

local function LongBracketAt(code, position)
    local equals = code:sub(position):match("^%[(=*)%[")
    if equals == nil then return nil end
    return equals, 2 + #equals
end

local function DecodeQuoted(raw)
    return raw:gsub("\\([\\\"'])", "%1"):gsub("\\n", "\n"):gsub("\\r", "\r"):gsub("\\t", "\t")
end

local function Lex(code)
    local tokens = {}
    local position, length = 1, #code
    while position <= length do
        local character = code:sub(position, position)
        local nextCharacter = code:sub(position + 1, position + 1)
        if character:match("%s") then
            position = position + 1
        elseif character == "-" and nextCharacter == "-" then
            local equals, openerLength = LongBracketAt(code, position + 2)
            if equals then
                local closeToken = "]" .. equals .. "]"
                local closeStart, closeEnd = code:find(closeToken, position + 2 + openerLength, true)
                if not closeStart then return nil, "unterminated long comment" end
                position = closeEnd + 1
            else
                position = (code:find("\n", position + 2, true) or length) + 1
            end
        elseif character == "\"" or character == "'" then
            local quote, finish, parts = character, position + 1, {}
            while finish <= length do
                local current = code:sub(finish, finish)
                if current == "\\" and finish < length then
                    parts[#parts + 1] = code:sub(finish, finish + 1)
                    finish = finish + 2
                elseif current == quote then
                    break
                else
                    parts[#parts + 1] = current
                    finish = finish + 1
                end
            end
            if finish > length then return nil, "unterminated quoted string" end
            tokens[#tokens + 1] = { kind = "string", value = DecodeQuoted(table.concat(parts)), position = position }
            position = finish + 1
        elseif character == "[" then
            local equals, openerLength = LongBracketAt(code, position)
            if equals then
                local closeToken = "]" .. equals .. "]"
                local bodyStart = position + openerLength
                local closeStart, closeEnd = code:find(closeToken, bodyStart, true)
                if not closeStart then return nil, "unterminated long string" end
                tokens[#tokens + 1] = {
                    kind = "string",
                    value = code:sub(bodyStart, closeStart - 1),
                    position = position,
                }
                position = closeEnd + 1
            else
                tokens[#tokens + 1] = { kind = "symbol", value = character, position = position }
                position = position + 1
            end
        elseif character:match("[%a_]") then
            local finish = position + 1
            while finish <= length and code:sub(finish, finish):match("[%w_]") do finish = finish + 1 end
            tokens[#tokens + 1] = {
                kind = "identifier",
                value = code:sub(position, finish - 1),
                position = position,
            }
            position = finish
        elseif character:match("%d") or (character == "." and nextCharacter:match("%d")) then
            local finish = position + 1
            while finish <= length and code:sub(finish, finish):match("[%w%.]") do finish = finish + 1 end
            tokens[#tokens + 1] = {
                kind = "number",
                value = code:sub(position, finish - 1),
                position = position,
            }
            position = finish
        else
            local pair = code:sub(position, position + 1)
            if pair == "==" or pair == "~=" or pair == "<=" or pair == ">=" or pair == ".." then
                tokens[#tokens + 1] = { kind = "symbol", value = pair, position = position }
                position = position + 2
            else
                tokens[#tokens + 1] = { kind = "symbol", value = character, position = position }
                position = position + 1
            end
        end
    end
    return tokens
end

local function TokenValue(tokens, index, expected)
    local token = tokens[index]
    if not token or (expected and token.kind ~= expected) then return nil end
    return token.value
end

local function ParseCallArguments(tokens, openIndex)
    if TokenValue(tokens, openIndex) ~= "(" then return nil end
    local arguments, current, depth = {}, {}, 0
    local index = openIndex + 1
    while tokens[index] do
        local value = tokens[index].value
        if value == "(" or value == "{" or value == "[" then
            depth = depth + 1
            current[#current + 1] = tokens[index]
        elseif value == ")" then
            if depth == 0 then
                arguments[#arguments + 1] = current
                return arguments, index
            end
            depth = depth - 1
            current[#current + 1] = tokens[index]
        elseif value == "}" or value == "]" then
            depth = math.max(depth - 1, 0)
            current[#current + 1] = tokens[index]
        elseif value == "," and depth == 0 then
            arguments[#arguments + 1] = current
            current = {}
        else
            current[#current + 1] = tokens[index]
        end
        index = index + 1
    end
    return nil
end

local function SingleLiteral(argument, kind)
    if type(argument) ~= "table" or #argument ~= 1 then return nil end
    if kind and argument[1].kind ~= kind then return nil end
    return argument[1].value
end

local function NumericArgumentStatus(argument)
    local numberValue = SingleLiteral(argument, "number")
    if numberValue ~= nil then return tonumber(numberValue) and "safe" or "unsafe" end
    local stringValue = SingleLiteral(argument, "string")
    if stringValue ~= nil then return tonumber(stringValue) and "safe" or "unsafe" end
    if type(argument) == "table" and #argument == 1
        and argument[1].kind == "identifier" and argument[1].value == "nil" then
        return "safe"
    end
    if type(argument) == "table" and #argument > 0 then
        local hasNumber = false
        for _, token in ipairs(argument) do
            if token.kind == "number" then
                hasNumber = true
            elseif token.kind ~= "symbol" or (token.value ~= "+" and token.value ~= "-"
                and token.value ~= "*" and token.value ~= "/" and token.value ~= "%"
                and token.value ~= "^" and token.value ~= "(" and token.value ~= ")") then
                return "dynamic"
            end
        end
        if hasNumber then return "safe" end
    end
    return "dynamic"
end

local function SourceArgumentStatus(argument)
    if argument == nil or #argument == 0 then return "safe" end
    local source = SingleLiteral(argument, "string")
    if source ~= nil then
        return (source == "parent" or source == "self" or source == "nil") and "safe" or "unsafe"
    end
    if #argument == 1 and argument[1].kind == "identifier" and argument[1].value == "nil" then
        return "safe"
    end
    return "dynamic"
end

local function CombineStatuses(...)
    local status = "safe"
    for index = 1, select("#", ...) do
        local current = select(index, ...)
        if current == "unsafe" then return "unsafe" end
        if current == "dynamic" then status = "dynamic" end
    end
    return status
end

local RAW_OPERAND_VALIDATORS = {
    check_event_var = function(arguments) return NumericArgumentStatus(arguments[3]) end,
    check_event_var_n = function(arguments) return NumericArgumentStatus(arguments[3]) end,
    inv_item_count = function(arguments) return SourceArgumentStatus(arguments[4]) end,
    inv_item_weight = function(arguments) return SourceArgumentStatus(arguments[3]) end,
    unit_distance_point = function(arguments)
        return CombineStatuses(
            NumericArgumentStatus(arguments[4]),
            NumericArgumentStatus(arguments[5])
        )
    end,
}

local function PathAt(tokens, index)
    if not tokens[index] or tokens[index].kind ~= "identifier" then return nil end
    local parts = { tokens[index].value }
    local cursor = index + 1
    while TokenValue(tokens, cursor) == "." and tokens[cursor + 1]
        and tokens[cursor + 1].kind == "identifier" do
        parts[#parts + 1] = tokens[cursor + 1].value
        cursor = cursor + 2
    end
    return table.concat(parts, "."), cursor
end

local function HasAssignmentSoon(tokens, index, limit)
    for cursor = index, math.min(#tokens, index + (limit or 8)) do
        local value = tokens[cursor].value
        if value == "=" then return true end
        if value == ";" then return false end
    end
    return false
end

local function ArgumentPath(argument, aliases)
    if type(argument) ~= "table" or #argument == 0 then return nil end
    local path, pathEnd = PathAt(argument, 1)
    if not path or pathEnd <= #argument then return nil end
    local first = path:match("^[^%.]+")
    if aliases and aliases[first] then path = aliases[first] .. path:sub(#first + 1) end
    return path
end

function Rules.AnalyzeCode(code, context)
    context = context or {}
    local result = NewResult(context.rootID or "runtime-lua", context.trustedPublisher)
    if type(code) ~= "string" then
        AddFinding(result, { kind = "lua_non_string", reason = "Lua Script Effect 不是有效字符串" }, "hard")
        result.fingerprint = "igl" .. Rules.RULE_VERSION .. ":invalid"
        result._reasonSet, result._signatures = nil, nil
        return result
    end
    result.metrics.scripts = 1
    result.metrics.sourceBytes = #code
    result._signatures[1] = tostring(#code) .. ":" .. StableHash(code)
    if #code > Rules.LIMITS.SCRIPT_SOURCE_BYTES then
        AddFinding(result, {
            kind = "lua_source_crash_size",
            bytes = #code,
            reason = "单段 Lua 源码超过崩溃防护上限 512 KiB",
        }, "hard")
    end

    local tokens, lexError = Lex(code)
    if not tokens then
        AddFinding(result, {
            kind = "lua_lex_error",
            reason = "Lua 词法结构无法安全分析：" .. tostring(lexError),
        }, "policy")
        result.fingerprint = "igl" .. Rules.RULE_VERSION .. ":" .. result._signatures[1]
        result._reasonSet, result._signatures = nil, nil
        return result
    end

    local hasLoop, hasUnboundedLoop, hasExit = false, false, false
    local hasHighImpactEffect, hasNestedScript, hasDynamicEffect = false, false, false
    local hasRecursiveFunction = false
    local highImpactEffectCalls = 0
    local hasContextWrite, hasPersistentContextWrite = false, false
    local maxLiteralLoop, maxRep = 0, 0
    local aliases = {}

    for index, token in ipairs(tokens) do
        local value = token.value
        if value == "break" or value == "return" then hasExit = true end
        if value == "function" and tokens[index + 1] and tokens[index + 1].kind == "identifier" then
            local functionName = tokens[index + 1].value
            for cursor = index + 2, math.min(#tokens, index + 400) do
                if TokenValue(tokens, cursor) == "end" then break end
                if TokenValue(tokens, cursor) == functionName and TokenValue(tokens, cursor + 1) == "(" then
                    hasRecursiveFunction = true
                    break
                end
            end
        end
        if value == "while" and (TokenValue(tokens, index + 1) == "true"
            or TokenValue(tokens, index + 1) == "1") then
            hasLoop, hasUnboundedLoop = true, true
        elseif value == "repeat" then
            hasLoop = true
            for cursor = index + 1, math.min(#tokens, index + 80) do
                if TokenValue(tokens, cursor) == "until"
                    and TokenValue(tokens, cursor + 1) == "false" then
                    hasUnboundedLoop = true
                    break
                end
            end
        elseif value == "for" then
            hasLoop = true
            if tokens[index + 1] and tokens[index + 1].kind == "identifier"
                and TokenValue(tokens, index + 2) == "=" then
                local from = tonumber(TokenValue(tokens, index + 3))
                local to
                for cursor = index + 4, math.min(#tokens, index + 12) do
                    if TokenValue(tokens, cursor) == "," then
                        to = tonumber(TokenValue(tokens, cursor + 1))
                        break
                    end
                end
                if from and to then maxLiteralLoop = math.max(maxLiteralLoop, math.abs(to - from) + 1) end
            end
        end

        if value == "local" and tokens[index + 1] and tokens[index + 1].kind == "identifier"
            and TokenValue(tokens, index + 2) == "=" then
            local sourcePath = PathAt(tokens, index + 3)
            if sourcePath then aliases[tokens[index + 1].value] = sourcePath end
        end

        local path, pathEnd = PathAt(tokens, index)
        if path then
            local first = path:match("^[^%.]+")
            if aliases[first] then
                path = aliases[first] .. path:sub(#first + 1)
                first = path:match("^[^%.]+")
            end
            if SHARED_LIBRARIES[first] and HasAssignmentSoon(tokens, pathEnd, 3) then
                result.metrics.sharedLibraryWrites = result.metrics.sharedLibraryWrites + 1
                AddFinding(result, {
                    kind = "lua_shared_library_write",
                    path = path,
                    reason = "Lua 尝试修改共享库 " .. path,
                }, "hard")
            elseif path:find("^args%.scripts") and HasAssignmentSoon(tokens, pathEnd, 8) then
                hasContextWrite = true
                result.metrics.contextWrites = result.metrics.contextWrites + 1
                AddFinding(result, {
                    kind = "lua_scripts_direct_write",
                    path = path,
                    reason = "Lua 直接修改工作流脚本定义，绕过对象扫描",
                }, "hard")
            elseif (path:find("^args%.container%.content")
                or path:find("^args%.object%.content")
                or path == "args.object.id" or path == "args.object.count")
                and HasAssignmentSoon(tokens, pathEnd, 8) then
                hasContextWrite = true
                result.metrics.contextWrites = result.metrics.contextWrites + 1
                AddFinding(result, {
                    kind = "lua_inventory_context_write",
                    path = path,
                    reason = "Lua 直接修改背包或对象核心字段，绕过受控 API",
                }, "hard")
            elseif path:find("^args%.object%.vars") and HasAssignmentSoon(tokens, pathEnd, 8) then
                hasContextWrite, hasPersistentContextWrite = true, true
                result.metrics.contextWrites = result.metrics.contextWrites + 1
                AddFinding(result, {
                    kind = "lua_variable_direct_write",
                    path = path,
                    reason = "Lua 直接写对象变量，未经过 setVar 检查",
                }, "advisory")
            end
            if path == "args._G" or path:find("^args%._G%.") then
                result.metrics.sandboxEscapeReferences = result.metrics.sandboxEscapeReferences + 1
                AddFinding(result, {
                    kind = "lua_global_environment_request",
                    reason = "Lua 请求外部注入完整 _G 环境",
                }, "hard")
            end
        end

        if token.kind == "identifier" and ESCAPE_IDENTIFIERS[value]
            and TokenValue(tokens, index + 1) == "(" then
            result.metrics.sandboxEscapeReferences = result.metrics.sandboxEscapeReferences + 1
            AddFinding(result, {
                kind = "lua_disabled_global_call",
                identifier = value,
                reason = "Lua 尝试调用沙箱未授权全局能力：" .. value,
            }, "advisory")
        end

        if value == "table" and TokenValue(tokens, index + 1) == "."
            and (TokenValue(tokens, index + 2) == "insert"
                or TokenValue(tokens, index + 2) == "remove"
                or TokenValue(tokens, index + 2) == "sort")
            and TokenValue(tokens, index + 3) == "(" then
            local arguments = ParseCallArguments(tokens, index + 3)
            local targetPath = arguments and ArgumentPath(arguments[1], aliases) or nil
            if targetPath == "string" or targetPath == "table" or targetPath == "math" then
                AddFinding(result, {
                    kind = "lua_shared_library_mutator",
                    path = targetPath,
                    reason = "Lua 通过 table 修改共享库 " .. targetPath,
                }, "hard")
            elseif targetPath and (targetPath:find("^args%.scripts")
                or targetPath:find("^args%.container%.content")
                or targetPath:find("^args%.object%.content")) then
                AddFinding(result, {
                    kind = "lua_context_table_mutator",
                    path = targetPath,
                    reason = "Lua 通过 table 直接修改工作流或背包上下文",
                }, "hard")
            elseif targetPath and targetPath:find("^args%.object%.vars") then
                hasPersistentContextWrite = true
                AddFinding(result, {
                    kind = "lua_variable_table_mutator",
                    path = targetPath,
                    reason = "Lua 通过 table 直接修改对象变量",
                }, hasLoop and "hard" or "advisory")
            end
        end

        if value == "effect" and TokenValue(tokens, index + 1) == "(" then
            local arguments = ParseCallArguments(tokens, index + 1)
            result.metrics.effectCalls = result.metrics.effectCalls + 1
            local effectID = arguments and SingleLiteral(arguments[1], "string")
            if not effectID then
                hasDynamicEffect = true
                result.metrics.dynamicEffectCalls = result.metrics.dynamicEffectCalls + 1
            else
                if HIGH_IMPACT_EFFECTS[effectID] then
                    hasHighImpactEffect = true
                    highImpactEffectCalls = highImpactEffectCalls + 1
                end
                hasNestedScript = hasNestedScript or effectID == "script"
            end
        elseif value == "op" and TokenValue(tokens, index + 1) == "(" then
            local arguments = ParseCallArguments(tokens, index + 1)
            result.metrics.operandCalls = result.metrics.operandCalls + 1
            local operandID = arguments and SingleLiteral(arguments[1], "string")
            local validator = operandID and RAW_OPERAND_VALIDATORS[operandID] or nil
            local operandStatus = validator and validator(arguments) or "safe"
            if not operandID or operandStatus ~= "safe" then
                result.metrics.unsafeOperandCalls = result.metrics.unsafeOperandCalls + 1
                AddFinding(result, {
                    kind = "lua_operand_code_injection",
                    operandID = operandID,
                    reason = operandStatus == "unsafe"
                            and "Lua op() 向未加引号代码位置传入可注入字符串"
                        or "Lua op() 使用动态 operand ID 或无法静态证明安全的代码位置参数",
                }, "hard")
            end
        elseif value == "string" and TokenValue(tokens, index + 1) == "."
            and TokenValue(tokens, index + 2) == "rep" and TokenValue(tokens, index + 3) == "(" then
            local arguments = ParseCallArguments(tokens, index + 3)
            local count = arguments and tonumber(SingleLiteral(arguments[2], "number")) or nil
            if count then maxRep = math.max(maxRep, count) end
        end
    end

    result.metrics.loops = hasLoop and 1 or 0
    result.metrics.literalLoopMax = maxLiteralLoop
    result.metrics.highImpactEffectCalls = highImpactEffectCalls
    if hasUnboundedLoop and not hasExit then
        AddFinding(result, {
            kind = "lua_unbounded_execution",
            reason = "Lua 包含没有显式退出路径的无限循环",
        }, "hard")
    end
    if maxLiteralLoop > Rules.LIMITS.LITERAL_LOOP_ITERATIONS then
        AddFinding(result, {
            kind = "lua_literal_loop_exhaustion",
            iterations = maxLiteralLoop,
            reason = "Lua 常量循环次数超过 100 万次",
        }, "hard")
    end
    if maxRep > Rules.LIMITS.STRING_REP_BYTES then
        AddFinding(result, {
            kind = "lua_string_allocation_exhaustion",
            bytes = maxRep,
            reason = "Lua string.rep 常量分配超过 2 MiB",
        }, "hard")
    end
    if hasDynamicEffect then
        AddFinding(result, {
            kind = "lua_dynamic_effect_dispatch",
            reason = "Lua 使用动态 effect ID，无法证明实际副作用",
        }, "policy")
    end
    if hasRecursiveFunction then
        AddFinding(result, {
            kind = "lua_recursive_function",
            reason = "Lua 定义了直接递归函数，需要发布者信任或人工复核",
        }, "policy")
    end
    if highImpactEffectCalls > 100 then
        AddFinding(result, {
            kind = "lua_unrolled_effect_flood",
            calls = highImpactEffectCalls,
            reason = "单段 Lua 包含超过 100 次高影响 effect 调用",
        }, "hard")
    elseif highImpactEffectCalls > 20 then
        AddFinding(result, {
            kind = "lua_dense_effect_dispatch",
            calls = highImpactEffectCalls,
            reason = "单段 Lua 密集调用背包、通信、声音、弹窗或光环效果",
        }, "policy")
    end
    if hasLoop and hasHighImpactEffect then
        AddFinding(result, {
            kind = "lua_loop_high_impact_effect",
            reason = "Lua 循环结合背包、光环、文档、通信、声音或弹窗效果",
        }, (maxLiteralLoop > Rules.LIMITS.AMPLIFIED_LOOP_ITERATIONS or hasUnboundedLoop)
            and "hard" or "policy")
    end
    if hasLoop and hasNestedScript then
        AddFinding(result, {
            kind = "lua_loop_nested_script",
            reason = "Lua 循环中递归启动新的 Script Effect",
        }, "hard")
    elseif hasNestedScript then
        AddFinding(result, {
            kind = "lua_nested_script",
            reason = "Lua 会动态启动另一段 Script Effect",
        }, "policy")
    end
    if hasLoop and hasPersistentContextWrite then
        AddFinding(result, {
            kind = "lua_loop_direct_persistence",
            reason = "Lua 循环直接增长持久对象变量",
        }, "hard")
    end

    result.score = result.behaviorScore + result.amplificationScore + result.observationScore
    result.fingerprint = "igl" .. Rules.RULE_VERSION .. ":" .. table.concat(result._signatures, ";")
    result._reasonSet, result._signatures = nil, nil
    return result
end

local function CollectLinkValues(value, output, seen)
    if type(value) == "string" or type(value) == "number" then
        output[tostring(value)] = true
    elseif type(value) == "table" and not seen[value] then
        seen[value] = true
        for _, child in pairs(value) do CollectLinkValues(child, output, seen) end
    end
end

local function ReachableSteps(workflow)
    local steps = type(workflow) == "table" and workflow.ST or nil
    steps = type(steps) == "table" and steps or {}
    local reachable, queue = {}, {}
    if steps["1"] or steps[1] then reachable["1"], queue[1] = true, "1" end
    local index = 1
    while queue[index] do
        local stepID = queue[index]
        index = index + 1
        local step = steps[stepID] or steps[tonumber(stepID)]
        local targets = {}
        if type(step) == "table" then
            if step.n ~= nil then targets[tostring(step.n)] = true end
            if step.t == "branch" and type(step.b) == "table" then
                for _, branch in pairs(step.b) do
                    if type(branch) == "table" and branch.n ~= nil then targets[tostring(branch.n)] = true end
                end
            end
        end
        for target in pairs(targets) do
            if not reachable[target] and (steps[target] or steps[tonumber(target)]) then
                reachable[target] = true
                queue[#queue + 1] = target
            end
        end
    end
    return reachable, steps
end

local function AnalyzeClass(result, classID, class, context)
    if type(class.SC) ~= "table" then return end
    local descriptors, workflowEdges, entries = {}, {}, {}
    for rawWorkflowID, workflow in pairs(class.SC) do
        local workflowID = CanonicalID(rawWorkflowID)
        if workflowID and type(workflow) == "table" then
            local reachable, steps = ReachableSteps(workflow)
            descriptors[workflowID] = { reachable = reachable, steps = steps }
            workflowEdges[workflowID] = {}
            for stepID in pairs(reachable) do
                local step = steps[stepID] or steps[tonumber(stepID)]
                if type(step) == "table" and type(step.e) == "table" then
                    for _, effect in pairs(step.e) do
                        if type(effect) == "table" and (effect.id == "run_workflow"
                            or effect.id == "aura_run_workflow") then
                            local args = type(effect.args) == "table" and effect.args or {}
                            if (effect.id == "run_workflow" and (CanonicalID(args[1]) or "o") == "o")
                                or (effect.id == "aura_run_workflow" and CanonicalID(args[1]) == classID) then
                                local target = CanonicalID(args[2])
                                if target then workflowEdges[workflowID][target] = true end
                            end
                        end
                    end
                end
            end
        end
    end
    if type(class.US) == "table" and CanonicalID(class.US.SC) then entries[CanonicalID(class.US.SC)] = true end
    if type(class.LI) == "table" then CollectLinkValues(class.LI, entries, {}) end
    if type(class.HA) == "table" then
        for _, handler in pairs(class.HA) do
            if type(handler) == "table" and CanonicalID(handler.SC) then entries[CanonicalID(handler.SC)] = true end
        end
    end
    local configured = context and context.entrypoints and context.entrypoints[classID]
    if type(configured) == "table" then for workflowID in pairs(configured) do entries[workflowID] = true end end
    if next(entries) == nil then for workflowID in pairs(descriptors) do entries[workflowID] = true end end

    local reachableWorkflows, queue = {}, {}
    for workflowID in pairs(entries) do
        if descriptors[workflowID] then reachableWorkflows[workflowID] = true; queue[#queue + 1] = workflowID end
    end
    local queueIndex = 1
    while queue[queueIndex] do
        local workflowID = queue[queueIndex]
        queueIndex = queueIndex + 1
        for target in pairs(workflowEdges[workflowID] or {}) do
            if descriptors[target] and not reachableWorkflows[target] then
                reachableWorkflows[target] = true
                queue[#queue + 1] = target
            end
        end
    end

    for workflowID in pairs(reachableWorkflows) do
        local descriptor = descriptors[workflowID]
        for stepID in pairs(descriptor.reachable) do
            local step = descriptor.steps[stepID] or descriptor.steps[tonumber(stepID)]
            for _, effect in pairs(type(step) == "table" and type(step.e) == "table" and step.e or {}) do
                if type(effect) == "table" and effect.id == "script" then
                    local code = type(effect.args) == "table" and effect.args[1] or nil
                    local analyzed = Rules.AnalyzeCode(code, {
                        rootID = result.rootID,
                        trustedPublisher = result.trustedPublisher,
                    })
                    result.metrics.scripts = result.metrics.scripts + 1
                    result.metrics.sourceBytes = result.metrics.sourceBytes + (analyzed.metrics.sourceBytes or 0)
                    result.metrics.effectCalls = result.metrics.effectCalls + (analyzed.metrics.effectCalls or 0)
                    result.metrics.highImpactEffectCalls = result.metrics.highImpactEffectCalls
                        + (analyzed.metrics.highImpactEffectCalls or 0)
                    result._signatures[#result._signatures + 1] = table.concat({
                        classID, workflowID, stepID, analyzed.fingerprint,
                    }, "|")
                    for _, finding in ipairs(analyzed.findings or {}) do
                        finding.classID = classID
                        finding.workflowID = workflowID
                        finding.stepID = stepID
                        AddFinding(result, finding, finding.hard
                            and (finding.bypassable and "policy" or "hard") or "advisory")
                    end
                elseif type(effect) == "table" and effect.id == "secure_macro" then
                    result.metrics.secureMacros = result.metrics.secureMacros + 1
                    local macro = type(effect.args) == "table" and tostring(effect.args[1] or "") or ""
                    local lower = macro:lower()
                    result._signatures[#result._signatures + 1] = table.concat({
                        classID, workflowID, stepID, "macro", tostring(#macro), StableHash(macro),
                    }, "|")
                    if (lower:find("/run", 1, true) or lower:find("/script", 1, true))
                        and (lower:find("runluascripteffect", 1, true)
                            or lower:find("rpbox_itemguarddb", 1, true)
                            or lower:find("trp3_security", 1, true)
                            or lower:find("trp3_db", 1, true)) then
                        AddFinding(result, {
                            kind = "secure_macro_guard_escape",
                            classID = classID,
                            workflowID = workflowID,
                            stepID = stepID,
                            reason = "安全宏尝试覆写 Lua 执行器、防护或 TRP3 核心数据",
                        }, "hard")
                    elseif lower:find("/run", 1, true) or lower:find("/script", 1, true) then
                        AddFinding(result, {
                            kind = "secure_macro_lua",
                            classID = classID,
                            workflowID = workflowID,
                            stepID = stepID,
                            reason = "对象包含需要硬件点击授权的全局 Lua 宏",
                        }, "policy")
                    end
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
            for childID, child in pairs(group) do
                WalkClasses(result, classID .. " " .. tostring(childID), child, context, seen)
            end
        end
    end
    seen[class] = nil
end

function Rules.Analyze(rootID, root, context)
    rootID = CanonicalID(rootID) or "unknown"
    context = context or {}
    local result = NewResult(rootID, context.trustedPublisher)
    if TRP3_DB and type(TRP3_DB.inner) == "table" and TRP3_DB.inner[rootID]
        and TRP3_DB.inner[rootID] ~= root then
        AddFinding(result, {
            kind = "reserved_inner_id_collision",
            reason = "接收对象冒用 TRP3 内建对象 ID，可绕过原生安全决议",
        }, "hard")
    end
    if type(root) == "table" then WalkClasses(result, rootID, root, context, {}) end
    if result.metrics.sourceBytes > Rules.LIMITS.OBJECT_SOURCE_BYTES then
        AddFinding(result, {
            kind = "lua_object_source_crash_size",
            bytes = result.metrics.sourceBytes,
            reason = "对象内 Lua 源码累计超过崩溃防护上限 2 MiB",
        }, "hard")
    end
    if result.metrics.highImpactEffectCalls > 100 then
        AddFinding(result, {
            kind = "lua_object_effect_flood",
            calls = result.metrics.highImpactEffectCalls,
            reason = "对象内 Lua 累计包含超过 100 次高影响 effect 调用",
        }, "hard")
    elseif result.metrics.highImpactEffectCalls > 40 then
        AddFinding(result, {
            kind = "lua_object_dense_effects",
            calls = result.metrics.highImpactEffectCalls,
            reason = "对象内 Lua 密集调用高影响 effect",
        }, "policy")
    end
    table.sort(result._signatures)
    result.fingerprint = "igl" .. Rules.RULE_VERSION .. ":" .. table.concat(result._signatures, ";")
    result.score = result.behaviorScore + result.amplificationScore + result.observationScore
    result._reasonSet, result._signatures = nil, nil
    return result
end
