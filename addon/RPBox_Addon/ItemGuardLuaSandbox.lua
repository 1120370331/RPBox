-- Execute through TRP3's supported Lua environment with metered source,
-- detached data and per-execution libraries. No WoW/TRP3 source patching.
local _, ns = ...
local Sandbox = {}
ns.ItemGuardLuaSandbox = Sandbox
Sandbox.LIMITS = { operations = 50000, allocation = 2 * 1024 * 1024, value = 512 * 1024 }
local unpackValues = unpack or table.unpack
local rawRep, rawConcat, rawFormat, rawGsub = string.rep, table.concat, string.format, string.gsub
local rawFind, rawMatch, rawGmatch = string.find, string.match, string.gmatch
local memoryUsage = collectgarbage
local executionSignal = {}
function Sandbox.IsBudgetError(value)
    return type(value) == "table" and value.signal == executionSignal and value.kind == "budget"
end
function Sandbox.IsCancelledError(value)
    return type(value) == "table" and value.signal == executionSignal and value.kind == "cancelled"
end

function Sandbox.CopyContext(args)
    local nodes, bytes, seen = 0, 0, {}
    local function copy(value, depth)
        nodes = nodes + 1
        if nodes > 12000 or depth > 32 then error("Lua 上下文结构超过上限", 0) end
        if type(value) == "string" then
            bytes = bytes + #value
            if bytes > Sandbox.LIMITS.allocation then error("Lua 上下文大小超过上限", 0) end
        end
        if type(value) ~= "table" then
            if type(value) == "function" or type(value) == "userdata" then return nil end
            return value
        end
        if seen[value] then return seen[value] end
        local result = {}
        seen[value] = result
        for key, child in pairs(value) do
            if (type(key) == "string" or type(key) == "number")
                and not (depth == 1 and (key == "scripts" or key == "_G" or key == "class"
                    or (type(key) == "string" and key:find("^__rpbox")))) then
                result[key] = copy(child, depth + 1)
            end
        end
        return result
    end
    return copy(args or {}, 1)
end

local function CheckOperand(id, ...)
    if type(id) ~= "string" then error("Lua operand ID 无效", 0) end
    local parameters = { ... }
    local numeric = id == "check_event_var" or id == "check_event_var_n"
    local function numberAt(index)
        local value = parameters[index]
        if value == nil then return end
        local number = tonumber(value)
        if not number or number ~= number or math.abs(number) == math.huge
            or (type(value) == "string" and not value:match("^[%s%d%+%-%.eE]+$")) then
            error("Lua operand 代码参数不是有限数值", 0)
        end
    end
    if numeric then numberAt(1) end
    if id == "unit_distance_point" then numberAt(2); numberAt(3) end
    local sourceIndex = id == "inv_item_count" and 2 or id == "inv_item_weight" and 1
    if sourceIndex and parameters[sourceIndex] ~= nil then
        local source = parameters[sourceIndex]
        if source ~= "parent" and source ~= "self" and source ~= "nil" then
            error("Lua operand 来源参数无效", 0)
        end
    end
    for _, value in pairs(parameters) do
        if type(value) ~= "string" and type(value) ~= "number" and type(value) ~= "boolean" then
            error("Lua operand 参数类型不支持", 0)
        end
    end
end
Sandbox.ValidateOperand = CheckOperand

-- Existing TRP3-authorized UI applications need their real object context and
-- WoW APIs. Meter Lua control flow without replacing shared host libraries or
-- imposing a detached-variable model on these explicitly trusted programs.
function Sandbox.RunTrusted(original, code, args, secured, options)
    local tokens, reason = ns.ItemGuardLuaRules.Tokenize(code)
    if not tokens then error(reason, 0) end
    local positions, problem = ns.ItemGuardLuaRules.ControlFlow(tokens)
    if not positions then error(problem, 0) end
    local task = options.budget
    local window = GetTime()
    local operations = 0
    local function tick()
        local now = GetTime()
        -- UI callback closures outlive their initial workflow. Renew only the
        -- time window; object-version revocation and quarantine remain binding.
        if now - window >= 5 then
            window, operations = now, 0
            task.expiresAt = now + 900
        end
        if not options.valid(task) then error({ signal = executionSignal, kind = "cancelled" }, 0) end
        operations = operations + 1
        if operations > Sandbox.LIMITS.operations then
            task.revoked = true
            error({ signal = executionSignal, kind = "budget", message = "Lua 执行预算耗尽" }, 0)
        end
    end
    local output, cursor = {}, 1
    for _, position in ipairs(positions) do
        output[#output + 1] = code:sub(cursor, position - 1)
        output[#output + 1] = " __rpboxTick(); "
        cursor = position
    end
    output[#output + 1] = code:sub(cursor)
    args.__rpboxTick = tick
    local result = { pcall(original,
        "local __rpboxTick=args.__rpboxTick; args.__rpboxTick=nil; __rpboxTick();\n" .. rawConcat(output),
        args, secured) }
    args.__rpboxTick = nil
    if not result[1] then error(result[2], 0) end
    return unpackValues(result, 2)
end

function Sandbox.Run(original, code, args, secured, options)
    local tokens, why = ns.ItemGuardLuaRules.Tokenize(code)
    if not tokens then error(why, 0) end
    if not ns.ItemGuardLuaExpressions then error("Lua 表达式防护模块缺失", 0) end
    code = ns.ItemGuardLuaExpressions.Rewrite(code, tokens)
    tokens = ns.ItemGuardLuaRules.Tokenize(code)
    local positions, problem = ns.ItemGuardLuaRules.ControlFlow(tokens, true)
    if not positions then error(problem, 0) end
    local budget = options.budget
    if type(memoryUsage) ~= "function" then error("当前 Lua 不支持内存计量", 0) end
    local initialMemory = memoryUsage("count")
    local function tick()
        budget.luaOperations = (budget.luaOperations or 0) + 1
        if budget.revoked or budget.luaOperations > Sandbox.LIMITS.operations then
            error("Lua 执行预算耗尽或任务已撤销", 0)
        end
        if memoryUsage("count") - initialMemory > 8 * 1024 then error("Lua 工作内存增长超过 8 MiB", 0) end
    end
    local function allocate(bytes)
        tick()
        if type(bytes) ~= "number" or bytes ~= bytes or bytes < 0 or bytes > Sandbox.LIMITS.value then
            error("Lua 单次分配超过 512 KiB", 0)
        end
        budget.luaAllocation = (budget.luaAllocation or 0) + bytes
        if budget.luaAllocation > Sandbox.LIMITS.allocation then error("Lua 累计分配超过 2 MiB", 0) end
    end
    local function patternBudget(text, pattern, plain)
        if type(text) ~= "string" or type(pattern) ~= "string" then error("Lua 字符串参数无效", 0) end
        if #text > Sandbox.LIMITS.value or #pattern > 512 then error("Lua 字符串匹配规模超过上限", 0) end
        if not plain then
            local escaped, variable = false, 0
            for index = 1, #pattern do
                local c = pattern:sub(index, index)
                if escaped then escaped = false
                elseif c == "%" then escaped = true
                elseif c == "*" or c == "+" or c == "-" then variable = variable + 1 end
            end
            if variable > 1 and (#text + 1) ^ variable > Sandbox.LIMITS.operations then
                error("Lua 模式回溯预算超过上限", 0)
            end
        end
        tick()
    end
    local libraries = { string = {}, table = {}, math = {} }
    for name, originalLibrary in pairs({ string = string, table = table, math = math }) do
        for key, value in pairs(originalLibrary) do libraries[name][key] = value end
    end
    libraries.string.dump = nil
    libraries.string.rep = function(text, count, separator)
        if type(text) ~= "string" or type(count) ~= "number" or count ~= math.floor(count) then
            error("Lua string.rep 参数无效", 0)
        end
        local size = #text * math.max(0, count)
            + (type(separator) == "string" and #separator or 0) * math.max(0, count - 1)
        allocate(size)
        return rawRep(text, count, separator)
    end
    libraries.string.format = function(format, ...)
        if type(format) ~= "string" then error("Lua format 参数无效", 0) end
        local estimate = #format
        for width in rawGmatch(format, "%d+") do
            if tonumber(width) > Sandbox.LIMITS.value then error("Lua format 宽度超过上限", 0) end
            estimate = estimate + tonumber(width)
        end
        for index = 1, select("#", ...) do estimate = estimate + #tostring(select(index, ...)) end
        allocate(estimate)
        local result = rawFormat(format, ...)
        if #result > Sandbox.LIMITS.value then error("Lua format 输出超过上限", 0) end
        return result
    end
    libraries.string.find = function(text, pattern, start, plain)
        patternBudget(text, pattern, plain)
        return rawFind(text, pattern, start, plain)
    end
    libraries.string.match = function(text, pattern, start)
        patternBudget(text, pattern)
        return rawMatch(text, pattern, start)
    end
    libraries.string.gmatch = function(text, pattern)
        patternBudget(text, pattern)
        local nextMatch = rawGmatch(text, pattern)
        return function() tick(); return nextMatch() end
    end
    libraries.string.gsub = function(text, pattern, replacement, limit)
        patternBudget(text, pattern)
        if type(replacement) == "string" then
            allocate(#text + (#text + 1) * #replacement)
        else
            local originalReplacement = replacement
            replacement = function(...)
                tick()
                local value
                if type(originalReplacement) == "function" then value = originalReplacement(...)
                elseif type(originalReplacement) == "table" then value = originalReplacement[select(1, ...)] end
                if value then allocate(#tostring(value)) end
                return value
            end
        end
        return rawGsub(text, pattern, replacement, limit)
    end
    libraries.table.concat = function(values, separator, first, last)
        first, last, separator = first or 1, last or #values, separator or ""
        if last - first > 50000 then error("Lua table.concat 条目超过上限", 0) end
        local size = 0
        for index = first, last do size = size + #tostring(values[index]) + #separator end
        allocate(size)
        return rawConcat(values, separator, first, last)
    end
    local output, cursor = {}, 1
    for _, position in ipairs(positions) do
        output[#output + 1] = code:sub(cursor, position - 1)
        output[#output + 1] = " __rpboxTick(); "
        cursor = position
    end
    output[#output + 1] = code:sub(cursor)
    local function concatenate(left, right)
        if (type(left) ~= "string" and type(left) ~= "number")
            or (type(right) ~= "string" and type(right) ~= "number") then error("Lua 拼接类型无效", 0) end
        allocate(#tostring(left) + #tostring(right))
        return left .. right
    end
    local function boundedUnpack(values, first, last)
        first, last = first or 1, last or #values
        if last - first > 4096 then error("Lua unpack 返回数量超过上限", 0) end
        tick()
        return unpackValues(values, first, last)
    end
    libraries.table.unpack = libraries.table.unpack and boundedUnpack or nil
    args.__rpboxTick, args.__rpboxLibraries, args.__rpboxOperand = tick, libraries, CheckOperand
    args.__rpboxConcat, args.__rpboxUnpack = concatenate, boundedUnpack
    local prefix = [[local __rpboxTick, __rpboxLibraries, __rpboxOperand = args.__rpboxTick, args.__rpboxLibraries, args.__rpboxOperand;
local __rpboxConcat, unpack = args.__rpboxConcat, args.__rpboxUnpack;
args.__rpboxTick, args.__rpboxLibraries, args.__rpboxOperand, args.__rpboxConcat, args.__rpboxUnpack = nil, nil, nil, nil, nil;
local string, table, math = __rpboxLibraries.string, __rpboxLibraries.table, __rpboxLibraries.math;
local __rpboxOp = op;
local op = function(id, context, ...) __rpboxTick(); __rpboxOperand(id, ...); return __rpboxOp(id, context, ...); end;
__rpboxTick();
]]
    -- String method syntax uses the shared string metatable, so guard those
    -- allocation/matching entry points for the synchronous execution as well.
    local saved = {}
    for _, key in ipairs({ "rep", "format", "find", "match", "gmatch", "gsub" }) do
        saved[key], string[key] = string[key], libraries.string[key]
    end
    local result = { pcall(original, prefix .. rawConcat(output), args, secured) }
    for key, value in pairs(saved) do string[key] = value end
    args.__rpboxTick, args.__rpboxLibraries, args.__rpboxOperand = nil, nil, nil
    args.__rpboxConcat, args.__rpboxUnpack = nil, nil
    if not result[1] then error(result[2], 0) end
    return unpackValues(result, 2)
end
