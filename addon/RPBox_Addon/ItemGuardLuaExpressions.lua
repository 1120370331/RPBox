-- Lua 5.1 expression/statement parser used only to meter concatenation.
-- It preserves original source (including comments) and rejects unsupported
-- syntax instead of executing a partially transformed program.
local _, ns = ...
local Expressions = {}
ns.ItemGuardLuaExpressions = Expressions
local priorities = { ["or"] = 1, ["and"] = 2, ["<"] = 3, [">"] = 3, ["<="] = 3,
    [">="] = 3, ["~="] = 3, ["=="] = 3, [".."] = 4, ["+"] = 5, ["-"] = 5,
    ["*"] = 6, ["/"] = 6, ["%"] = 6, ["^"] = 8 }
local keywords = {}
for word in ("and break do else elseif end false for function if in local nil not or repeat return then true until while"):gmatch("%S+") do
    keywords[word] = true
end

function Expressions.Rewrite(source, tokens)
    local index, depth, edits, operations = 1, 0, {}, 0
    local expression, block, statement, functionBody, tableBody
    local function value(offset)
        local token = tokens[index + (offset or 0)]
        if not token then return nil end
        if token.kind == "string" or token.kind == "number" then return "<" .. token.kind .. ">" end
        return token.value
    end
    local function expect(word)
        if value() ~= word then error("Lua 语法不支持或不完整：期待 " .. word, 0) end
        index = index + 1
    end
    local function identifier()
        local token = tokens[index]
        if not token or token.kind ~= "identifier" or keywords[token.value] then error("Lua 标识符无效", 0) end
        index = index + 1
    end
    local function finish(tokenIndex)
        return tokens[tokenIndex + 1] and tokens[tokenIndex + 1].position - 1 or #source
    end
    local function render(first, last)
        local output, cursor = {}, first
        table.sort(edits, function(a, b) return a.first < b.first end)
        for _, edit in ipairs(edits) do
            if edit.first >= first and edit.last <= last then
                output[#output + 1] = source:sub(cursor, edit.first - 1)
                output[#output + 1] = edit.text
                cursor = edit.last + 1
            end
        end
        output[#output + 1] = source:sub(cursor, last)
        return table.concat(output)
    end
    local function list()
        expression(0)
        while value() == "," do index = index + 1; expression(0) end
    end
    local function arguments()
        if value() == "(" then
            index = index + 1
            if value() ~= ")" then list() end
            expect(")")
        elseif value() == "{" then tableBody()
        elseif tokens[index] and tokens[index].kind == "string" then index = index + 1
        else error("Lua 函数参数无效", 0) end
    end
    tableBody = function()
        expect("{")
        while value() and value() ~= "}" do
            if value() == "[" then index = index + 1; expression(0); expect("]"); expect("="); expression(0)
            elseif tokens[index].kind == "identifier" and value(1) == "=" then
                identifier(); expect("="); expression(0)
            else expression(0) end
            if value() == "," or value() == ";" then index = index + 1 else break end
        end
        expect("}")
    end
    functionBody = function()
        expect("(")
        if value() ~= ")" then
            if value() == "..." then index = index + 1
            else
                identifier()
                while value() == "," do
                    index = index + 1
                    if value() == "..." then index = index + 1; break else identifier() end
                end
            end
        end
        expect(")"); block(); expect("end")
    end
    expression = function(minimum)
        depth, operations = depth + 1, operations + 1
        if depth > 64 or operations > 50000 then error("Lua 表达式分析预算超过上限", 0) end
        local first, token = index, tokens[index]
        if not token then error("Lua 表达式缺失", 0) end
        local word = token.value
        if token.kind ~= "string" and (word == "-" or word == "not" or word == "#") then
            index = index + 1; expression(7)
        elseif token.kind == "string" or token.kind == "number"
            or word == "nil" or word == "true" or word == "false" or word == "..." then index = index + 1
        elseif word == "function" then index = index + 1; functionBody()
        elseif word == "{" then tableBody()
        elseif word == "(" then index = index + 1; expression(0); expect(")")
        else identifier() end
        while tokens[index] do
            word = value()
            if word == "." then index = index + 1; identifier()
            elseif word == "[" then index = index + 1; expression(0); expect("]")
            elseif word == ":" then index = index + 1; identifier(); arguments()
            elseif word == "(" or word == "{" or tokens[index].kind == "string" then arguments()
            else break end
        end
        while tokens[index] and tokens[index].kind ~= "string" do
            local operator = value()
            local priority = priorities[operator]
            if not priority or priority <= minimum then break end
            local leftLast = index - 1
            index = index + 1
            local rightFirst = index
            expression((operator == ".." or operator == "^") and priority - 1 or priority)
            if operator == ".." then
                local startPosition, endPosition = tokens[first].position, finish(index - 1)
                local replacement = "__rpboxConcat(\n" .. render(startPosition, finish(leftLast))
                    .. "\n,\n" .. render(tokens[rightFirst].position, endPosition) .. "\n)"
                for cursor = #edits, 1, -1 do
                    if edits[cursor].first >= startPosition and edits[cursor].last <= endPosition then
                        table.remove(edits, cursor)
                    end
                end
                edits[#edits + 1] = { first = startPosition, last = endPosition, text = replacement }
                if #edits > 2048 then error("Lua 拼接表达式数量超过上限", 0) end
            end
        end
        depth = depth - 1
    end
    statement = function()
        local word = value()
        if word == ";" then index = index + 1
        elseif word == "if" then
            index = index + 1; expression(0); expect("then"); block()
            while value() == "elseif" do index = index + 1; expression(0); expect("then"); block() end
            if value() == "else" then index = index + 1; block() end
            expect("end")
        elseif word == "while" then index = index + 1; expression(0); expect("do"); block(); expect("end")
        elseif word == "repeat" then index = index + 1; block(); expect("until"); expression(0)
        elseif word == "do" then index = index + 1; block(); expect("end")
        elseif word == "for" then
            index = index + 1; identifier()
            if value() == "=" then index = index + 1; list()
            else
                while value() == "," do index = index + 1; identifier() end
                expect("in"); list()
            end
            expect("do"); block(); expect("end")
        elseif word == "function" then
            index = index + 1; identifier()
            while value() == "." do index = index + 1; identifier() end
            if value() == ":" then index = index + 1; identifier() end
            functionBody()
        elseif word == "local" then
            index = index + 1
            if value() == "function" then index = index + 1; identifier(); functionBody()
            else
                identifier()
                while value() == "," do index = index + 1; identifier() end
                if value() == "=" then index = index + 1; list() end
            end
        elseif word == "return" then
            index = index + 1
            if value() and value() ~= ";" and value() ~= "end" and value() ~= "else"
                and value() ~= "elseif" and value() ~= "until" then list() end
        elseif word == "break" then index = index + 1
        else
            list()
            if value() == "=" then index = index + 1; list() end
        end
    end
    block = function()
        depth = depth + 1
        if depth > 64 then error("Lua 块嵌套超过上限", 0) end
        while value() and value() ~= "end" and value() ~= "else" and value() ~= "elseif" and value() ~= "until" do
            local previous = index
            statement()
            if index == previous then error("Lua 语法分析未推进", 0) end
        end
        depth = depth - 1
    end
    block()
    if tokens[index] then error("Lua 存在无法分析的尾部语法", 0) end
    return render(1, #source)
end
