-- Bounded preflight shared by receipt, scanning and compilation.
local _, ns = ...
local Structure = {}
ns.ItemGuardStructure = Structure
Structure.RULE_VERSION = 2
Structure.LIMITS = {
    nodes = 50000, depth = 32, bytes = 2 * 1024 * 1024,
    stringBytes = 512 * 1024, workflows = 512, steps = 4096,
    effects = 8192, expanded = 4096, compileDepth = 128,
}
local groups = { "IN", "QE", "ST" }

-- Walk before projecting/hashing or passing data to TRP3's recursive registrar.
function Structure.Validate(root)
    local limits, nodes, bytes, active = Structure.LIMITS, 0, 0, {}
    local function visit(value, depth)
        nodes = nodes + 1
        if nodes > limits.nodes then return false, "对象结构节点超过上限" end
        local kind = type(value)
        if kind == "string" then
            bytes = bytes + #value
            if #value > limits.stringBytes then return false, "对象单字符串超过 512 KiB" end
            if bytes > limits.bytes then return false, "对象总字符串超过 2 MiB" end
        elseif kind == "table" then
            if depth > limits.depth then return false, "对象嵌套深度超过上限" end
            if active[value] then return false, "对象包含循环表引用" end
            if getmetatable(value) then return false, "对象包含不支持的元表" end
            active[value] = true
            for key, child in pairs(value) do
                if type(key) ~= "number" and type(key) ~= "string" then
                    return false, "对象包含无效字段键"
                end
                local ok, reason = visit(key, depth + 1)
                if not ok then return false, reason end
                ok, reason = visit(child, depth + 1)
                if not ok then return false, reason end
            end
            active[value] = nil
        elseif kind == "number" then
            if value ~= value or value == math.huge or value == -math.huge then
                return false, "对象包含非有限数值"
            end
        elseif kind ~= "boolean" and kind ~= "nil" then
            return false, "对象包含不可序列化的值"
        end
        return true
    end
    if type(root) ~= "table" then return false, "对象不是有效表" end
    local ok, reason = visit(root, 1)
    if not ok then return false, reason end
    local workflows, steps, effects = 0, 0, 0
    local function classes(class)
        for _, name in ipairs({ "BA", "MD", "SC", "US", "LI", "HA" }) do
            if class[name] ~= nil and type(class[name]) ~= "table" then
                return false, "对象字段类型无效：" .. name
            end
        end
        for _, workflow in pairs(class.SC or {}) do
            workflows = workflows + 1
            if workflows > limits.workflows then return false, "工作流数量超过上限" end
            if type(workflow) ~= "table" or type(workflow.ST) ~= "table" then
                return false, "工作流步骤结构无效"
            end
            for _, step in pairs(workflow.ST) do
                steps = steps + 1
                if steps > limits.steps then return false, "工作流步骤数量超过上限" end
                if type(step) ~= "table" then return false, "工作流步骤无效" end
                for _, field in ipairs({ "e", "b", "cond" }) do
                    if step[field] ~= nil and type(step[field]) ~= "table" then
                        return false, "工作流步骤字段无效：" .. field
                    end
                end
                for _, effect in pairs(step.e or {}) do
                    effects = effects + 1
                    if effects > limits.effects then return false, "工作流效果数量超过上限" end
                    if type(effect) ~= "table" or type(effect.id) ~= "string"
                        or (effect.args ~= nil and type(effect.args) ~= "table") then
                        return false, "工作流效果结构无效"
                    end
                end
            end
        end
        for _, group in ipairs(groups) do
            if class[group] ~= nil and type(class[group]) ~= "table" then
                return false, "子对象结构无效"
            end
            for _, child in pairs(class[group] or {}) do
                if type(child) ~= "table" then return false, "子对象无效" end
                local valid, why = classes(child)
                if not valid then return false, why end
            end
        end
        return true
    end
    return classes(root)
end

-- Mirrors compiler expansion, including delayed blocks (which are compiled
-- synchronously). Repeated DAG branches consume their full expansion cost.
function Structure.ValidateWorkflow(workflow)
    local steps = type(workflow) == "table" and workflow.ST
    if type(steps) ~= "table" then return false, "工作流步骤缺失" end
    local active, memo = {}, {}
    local function count(id, depth)
        id = tostring(id)
        if active[id] then return nil, "工作流步骤连接成环，无法安全编译" end
        if depth > Structure.LIMITS.compileDepth then return nil, "工作流编译深度超过上限" end
        if memo[id] then return memo[id] end
        local step = steps[id] or steps[tonumber(id)]
        if not step then return 0 end
        if type(step) ~= "table" then return nil, "工作流步骤无效" end
        active[id] = true
        local total, targets = 1, {}
        if step.t == "branch" then
            for _, branch in pairs(step.b or {}) do
                if type(branch) ~= "table" then return nil, "工作流分支无效" end
                if branch.n then targets[#targets + 1] = branch.n end
            end
        elseif step.t == "list" or step.t == "delay" then
            if step.n then targets[1] = step.n end
        else
            return nil, "工作流步骤类型不支持"
        end
        for _, target in ipairs(targets) do
            local size, reason = count(target, depth + 1)
            if not size then return nil, reason end
            total = total + size
            if total > Structure.LIMITS.expanded then return nil, "工作流编译展开超过上限" end
        end
        active[id] = nil
        memo[id] = total
        return total
    end
    local size, reason = count("1", 1)
    return size ~= nil, reason
end

-- A collision-free bounded serialization is used only as an in-memory cache
-- key. Persistent rule fingerprints retain their existing compact format.
function Structure.Revision(root)
    local output = {}
    local function write(value, visual)
        local kind = type(value)
        if kind == "table" then
            output[#output + 1] = "{"
            local keys = {}
            for key in pairs(value) do
                if not (visual and (key == "IC" or key == "US")) then keys[#keys + 1] = key end
            end
            table.sort(keys, function(a, b)
                if type(a) ~= type(b) then return type(a) < type(b) end
                return a < b
            end)
            for _, key in ipairs(keys) do write(key); write(value[key], key == "BA") end
            output[#output + 1] = "}"
        elseif kind == "string" then
            output[#output + 1] = "s" .. #value .. ":" .. value
        else
            output[#output + 1] = kind .. ":" .. tostring(value) .. ";"
        end
    end
    write(root)
    return table.concat(output)
end
