-- Compare rewritten benign programs with native Lua semantics, including
-- precedence, right associativity, nested functions and string/comment tokens.
local ns = {}
assert(loadfile("addon/RPBox_Addon/ItemGuardLuaRules.lua"))("RPBox_Addon", ns)
assert(loadfile("addon/RPBox_Addon/ItemGuardLuaExpressions.lua"))("RPBox_Addon", ns)
local compile = loadstring or load
local examples = {
    [[return "a" .. "b" .. "c"]],
    [[return 1 + 2 .. 3 * 4]],
    [[local t = { ["}"] = "a", value = "b" .. "c" }; return t["}"] .. t.value]],
    [[local function f(...) return table.concat({...}, "-") end; return f("a" .. "b", "c")]],
    [[local n=0; repeat n=n+1 until n==3; return "n" .. n]],
    [[local t={}; function t:run(v) return "a" .. v end; return t:run("b")]],
    [[local v=""; for _, x in (function() return ipairs({"a","b"}) end)() do v=v..x end; return v]],
    "return 'a' .. -- end function while\n 'b' -- trailing comment",
    [[local a="end"; if a then a=a.."until" elseif false then a="" else a="x" end; return a]],
    [[return (function() return "function" .. "repeat" end)() .. "tail"]],
}
for index, code in ipairs(examples) do
    local tokens = assert(ns.ItemGuardLuaRules.Tokenize(code))
    local transformed = ns.ItemGuardLuaExpressions.Rewrite(code, tokens)
    local expected = assert(compile(code))()
    local actual = assert(compile("local __rpboxConcat=function(a,b) return a..b end;\n" .. transformed))()
    assert(actual == expected, "concatenation semantics changed: " .. index)
end
local rules = ns.ItemGuardLuaRules
assert(not rules.AnalyzeCode("local f=string.format; return f('%s','ok')").blocked)
assert(not rules.AnalyzeCode("for i=1,3 do local n=i end; effect('item_add',args,'safe',1)").blocked)
assert(rules.AnalyzeCode("local function done() return 1 end; while true do end").blocked)
assert(rules.AnalyzeCode("while (true) do end").blocked)
assert(rules.AnalyzeCode("for i=10,1,-0.000001 do local x=i end").blocked)
assert(rules.AnalyzeCode("return string.rep('abcd',1048576)").blocked)
assert(rules.AnalyzeCode('args["scripts"]["use"]={}').blocked)
print("PASS Lua expressions: native semantics, allocation rewrite, lexical false positives and invariants")
