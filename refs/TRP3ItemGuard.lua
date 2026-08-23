-- RPBox TRP3 Extended Item Guard (experimental, session-scoped).
--
-- Install this from a trusted Lua Script Effect immediately after running the
-- one-shot bootstrap below. The bootstrap restores TRP3's original executor
-- before this code starts; this guard then owns the gated injection chain.
--
-- /run RPBG0=TRP3_API.script.runLuaScriptEffect;TRP3_API.script.runLuaScriptEffect=function(c,a,s)TRP3_API.script.runLuaScriptEffect=RPBG0;a=a or{};a._G=_G;return RPBG0(c,a,s)end
--
-- The guard does not delete objects or inventory.
-- It statically scans item/campaign workflows, quarantines high-confidence
-- threats, blocks quarantined workflow execution, withholds _G from foreign
-- Lua effects, and applies runtime circuit breakers to repeated writes/adds.
--
-- Slash commands:
--   /rpboxguard status
--   /rpboxguard scan
--   /rpboxguard list
--   /rpboxguard inspect ROOT_ID
--   /rpboxguard release ITEM_NAME
--   /rpboxguard allow ROOT_ID
--   /rpboxguard block ROOT_ID
--   /rpboxguard restore

if not args or not args._G then
    if effect then
        effect("text", args, "RPBox Guard: args._G is required", "4")
    end
    return
end

local G = args._G
local API = G.TRP3_API
local Extended = G.TRP3_Extended
local DB = G.TRP3_DB

if not API or not API.script or not API.extended or not API.inventory or not Extended or not DB then
    if effect then
        effect("text", args, "RPBox Guard: TRP3 Extended is not ready", "4")
    end
    return
end

local VERSION = 4
local PREFIX = "|cff35d07fRPBox Guard|r "
local ISOLATION_ICON = "ui-engineering-90-remote-close-icon"
local ISOLATION_FIELD = "RPBOX_GUARD_QUARANTINE"

local CONFIG = {
    maxTableNodes = 50000,
    maxTableDepth = 64,
    maxStringBytes = 2 * 1024 * 1024,
    maxSingleStringBytes = 512 * 1024,
    maxWorkflows = 128,
    maxSteps = 768,
    maxEffects = 3072,
    maxExpandedSteps = 4096,
    maxItemAddEffects = 24,
    maxSingleItemAdd = 100,
    runtimeWindow = 5,
    runtimeEffects = 500,
    runtimeWorkflowCalls = 120,
    runtimeVariableWrites = 180,
    runtimeItemAddCalls = 25,
    runtimeItemAddCount = 250,
    runtimeDirectAddCalls = 35,
    runtimeDirectAddCount = 350,
    luaLoopIterationBudget = 50000,
}

local function chat(message)
    if G.DEFAULT_CHAT_FRAME and G.DEFAULT_CHAT_FRAME.AddMessage then
        G.DEFAULT_CHAT_FRAME:AddMessage(PREFIX .. tostring(message))
    elseif G.print then
        G.print("RPBox Guard " .. tostring(message))
    end
end

local previous = G.RPBoxTRP3Guard
if previous and previous.version == VERSION and previous.installed then
    if previous.ScanAll then
        previous:ScanAll(true)
    end
    chat("already installed; scan refreshed")
    return
end

if previous and type(previous.Restore) == "function" then
    G.pcall(function()
        previous:Restore(true)
    end)
end

local guard = {
    version = VERSION,
    installed = false,
    results = {},
    quarantine = {},
    manualBlocks = {},
    allow = {},
    runtime = {},
    visualBackups = {},
    lastNotice = {},
    currentRoot = nil,
    scanCount = 0,
}
G.RPBoxTRP3Guard = guard

local function isRootID(id)
    return type(id) == "string" and id ~= "" and not string.find(id, " ", 1, true)
end

function guard:GetRootID(id)
    if type(id) ~= "string" or id == "" then
        return nil
    end
    if API.extended.getRootClassID then
        return API.extended.getRootClassID(id)
    end
    return string.match(id, "^[^ ]+")
end

function guard:GetRootObject(rootID)
    if not rootID then
        return nil
    end
    if G.TRP3_Tools_DB and G.TRP3_Tools_DB[rootID] then
        return G.TRP3_Tools_DB[rootID]
    end
    if G.TRP3_Exchange_DB and G.TRP3_Exchange_DB[rootID] then
        return G.TRP3_Exchange_DB[rootID]
    end
    return DB.global and DB.global[rootID]
end

function guard:IsOwned(rootID)
    local root = rootID and G.TRP3_Tools_DB and G.TRP3_Tools_DB[rootID]
    if not root then
        return false
    end
    local playerID = API.globals and API.globals.player_id
    local creator = type(root.MD) == "table" and root.MD.CB or nil
    local sender = G.TRP3_Security
        and type(G.TRP3_Security.sender) == "table"
        and G.TRP3_Security.sender[rootID]
        or nil
    if playerID and creator and creator ~= playerID then
        return false
    end
    if playerID and sender and sender ~= playerID then
        return false
    end
    return true
end

function guard:ResolveRootFromArgs(eArgs, fallbackID)
    local id = fallbackID
    if type(eArgs) == "table" then
        id = eArgs.classID or id
        if not id and type(eArgs.object) == "table" then
            id = eArgs.object.id
        end
        if not id and type(eArgs.container) == "table" then
            id = eArgs.container.id
        end
    end
    return self:GetRootID(id)
end

local function newResult(rootID, owned)
    return {
        rootID = rootID,
        owned = owned,
        block = false,
        hardBlock = false,
        score = 0,
        reasons = {},
        reasonSet = {},
        classes = 0,
        workflows = 0,
        steps = 0,
        effects = 0,
        scripts = 0,
        itemAdds = 0,
        workflowCalls = 0,
        variableWrites = 0,
        tableNodes = 0,
        stringBytes = 0,
    }
end

local function addReason(result, points, reason, hard)
    if not result.reasonSet[reason] then
        result.reasonSet[reason] = true
        table.insert(result.reasons, reason)
        result.score = result.score + (points or 0)
    end
    if hard then
        result.hardBlock = true
    end
end

local function measureValue(value, result, seen, depth)
    local valueType = type(value)
    if valueType == "string" then
        local length = #value
        result.stringBytes = result.stringBytes + length
        if length > CONFIG.maxSingleStringBytes then
            addReason(result, 80, "single embedded string exceeds safety limit", true)
        end
        if result.stringBytes > CONFIG.maxStringBytes then
            addReason(result, 100, "object string data exceeds safety limit", true)
        end
        return
    end
    if valueType ~= "table" or seen[value] then
        return
    end
    seen[value] = true
    result.tableNodes = result.tableNodes + 1
    if result.tableNodes > CONFIG.maxTableNodes then
        addReason(result, 100, "object table exceeds node budget", true)
        return
    end
    if depth > CONFIG.maxTableDepth then
        addReason(result, 100, "object table exceeds depth budget", true)
        return
    end
    for key, child in pairs(value) do
        measureValue(key, result, seen, depth + 1)
        measureValue(child, result, seen, depth + 1)
        if result.hardBlock and result.tableNodes > CONFIG.maxTableNodes then
            return
        end
    end
end

local function scanLuaCode(code, result, context)
    if type(code) ~= "string" then
        addReason(result, 30, context .. " has a non-string Lua payload", false)
        return
    end

    result.scripts = result.scripts + 1
    local lower = string.lower(code)
    local hasLoop = false
    local hasUnconditionalLoop = false
    local hasDirectInventoryWrite = false
    local hasDirectDatabaseWrite = false
    local hasVariableWrite = false
    local hasScheduler = false
    local hasLoader = false

    if string.find(lower, "while%s+true%s+do")
        or string.find(lower, "while%s*%(%s*true%s*%)%s*do")
        or string.find(lower, "while%s+1%s+do")
        or string.find(lower, "repeat[%s%S]-until%s+false") then
        hasLoop = true
        hasUnconditionalLoop = true
        addReason(result, 100, context .. " contains an unconditional Lua loop", false)
    elseif string.find(lower, "%f[%a]while%f[%A]")
        or string.find(lower, "%f[%a]repeat%f[%A]")
        or string.find(lower, "%f[%a]for%f[%A]") then
        hasLoop = true
        addReason(result, 30, context .. " contains a Lua loop", false)
    end

    if string.find(lower, "while%s+true%s+do%s*end")
        or string.find(lower, "while%s*%(%s*true%s*%)%s*do%s*end")
        or string.find(lower, "repeat%s*until%s+false") then
        addReason(result, 200, context .. " contains a non-terminating empty loop", true)
    end

    if string.find(lower, "inventory%s*%.%s*additem")
        or string.find(lower, "effect%s*%(%s*[\"']item_add[\"']") then
        hasDirectInventoryWrite = true
        addReason(result, 70, context .. " directly adds inventory items", false)
    end

    if string.find(lower, "trp3_tools_db", 1, true)
        or string.find(lower, "trp3_exchange_db", 1, true)
        or string.find(lower, "trp3_stashes", 1, true)
        or string.find(lower, "trp3_drop", 1, true) then
        hasDirectDatabaseWrite = true
        addReason(result, 70, context .. " directly writes a TRP3 database", false)
    end

    if string.find(lower, "setvar%s*%(")
        or string.find(lower, "%.vars%s*%[") then
        hasVariableWrite = true
        addReason(result, 20, context .. " writes workflow variables", false)
    end

    if string.find(lower, "c_timer%s*%.%s*after")
        or string.find(lower, "setscript%s*%(%s*[\"']onupdate")
        or string.find(lower, "runworkflow%s*%(")
        or string.find(lower, "executeclassscript%s*%(") then
        hasScheduler = true
        addReason(result, 40, context .. " schedules or recursively invokes work", false)
    end

    if string.find(lower, "loadstring%s*%(")
        or string.find(lower, "setfenv%s*%(")
        or string.find(lower, "getfenv%s*%(")
        or string.find(lower, "string%s*%.%s*char%s*%(") then
        hasLoader = true
        addReason(result, 45, context .. " dynamically constructs or loads code", false)
    end

    if string.find(lower, "args%s*%.%s*_g")
        or string.find(lower, "%f[%w]_g%f[%W]") then
        addReason(result, 35, context .. " requests the WoW global environment", false)
    end

    local hasExplicitExit = string.find(lower, "%f[%a]break%f[%A]")
        or string.find(lower, "%f[%a]return%f[%A]")

    if hasUnconditionalLoop and not hasExplicitExit then
        addReason(result, 180, context .. " has an unconditional loop without an explicit exit", true)
    end
    if hasLoop and hasDirectInventoryWrite then
        addReason(result, 180, context .. " combines a loop with inventory creation", true)
    end
    if hasScheduler and (hasDirectInventoryWrite or hasDirectDatabaseWrite) then
        addReason(result, 180, context .. " schedules repeated persistent data changes", true)
    end
    if hasLoader and (hasDirectInventoryWrite or hasDirectDatabaseWrite) then
        addReason(result, 160, context .. " hides direct persistent writes behind dynamic code", true)
    end
    if hasScheduler and hasVariableWrite then
        addReason(result, 35, context .. " schedules ordinary workflow variable updates", false)
    end
end

local LOOP_COUNTER_NAME = "__RPBoxGuardLoopBudget"
    .. tostring(math.floor(((G.GetTime and G.GetTime()) or 0) * 1000))

local function loopBudgetStatement()
    return LOOP_COUNTER_NAME
        .. "=" .. LOOP_COUNTER_NAME .. "+1;if "
        .. LOOP_COUNTER_NAME .. ">" .. CONFIG.luaLoopIterationBudget .. " then "
        .. "if args and args._G and args._G.RPBoxTRP3Guard then "
        .. "args._G.RPBoxTRP3Guard:Trip(args.classID or (args.object and args.object.id),"
        .. "\"Lua loop iteration budget exceeded\") end return 0 end;"
end

local function longBracketAt(code, position)
    local equals = string.match(string.sub(code, position), "^%[(=*)%[")
    if equals == nil then
        return nil
    end
    return equals, 2 + #equals
end

function guard:InstrumentLua(code)
    if type(code) ~= "string" then
        return nil, "non-string Lua payload"
    end

    local output = {}
    local pendingLoopDepths = {}
    local loopCount = 0
    local depth = 0
    local position = 1
    local length = #code
    local budgetStatement = loopBudgetStatement()

    while position <= length do
        local character = string.sub(code, position, position)
        local nextCharacter = string.sub(code, position + 1, position + 1)

        if character == "\"" or character == "'" then
            local quote = character
            local finish = position + 1
            local closed = false
            while finish <= length do
                local current = string.sub(code, finish, finish)
                if current == "\\" then
                    finish = finish + 2
                elseif current == quote then
                    finish = finish + 1
                    closed = true
                    break
                else
                    finish = finish + 1
                end
            end
            if not closed then
                return nil, "unterminated quoted string"
            end
            table.insert(output, string.sub(code, position, finish - 1))
            position = finish
        elseif character == "-" and nextCharacter == "-" then
            local equals, openerLength = longBracketAt(code, position + 2)
            if equals then
                local closeToken = "]" .. equals .. "]"
                local closeStart, closeEnd = string.find(code, closeToken, position + 2 + openerLength, true)
                if not closeStart then
                    return nil, "unterminated long comment"
                end
                table.insert(output, string.sub(code, position, closeEnd))
                position = closeEnd + 1
            else
                local lineEnd = string.find(code, "\n", position + 2, true)
                if lineEnd then
                    table.insert(output, string.sub(code, position, lineEnd))
                    position = lineEnd + 1
                else
                    table.insert(output, string.sub(code, position))
                    position = length + 1
                end
            end
        elseif character == "[" then
            local equals, openerLength = longBracketAt(code, position)
            if equals then
                local closeToken = "]" .. equals .. "]"
                local closeStart, closeEnd = string.find(code, closeToken, position + openerLength, true)
                if not closeStart then
                    return nil, "unterminated long string"
                end
                table.insert(output, string.sub(code, position, closeEnd))
                position = closeEnd + 1
            else
                table.insert(output, character)
                depth = depth + 1
                position = position + 1
            end
        elseif string.find(character, "[%a_]") then
            local finish = position + 1
            while finish <= length and string.find(string.sub(code, finish, finish), "[%w_]") do
                finish = finish + 1
            end
            local token = string.sub(code, position, finish - 1)
            table.insert(output, token)
            if token == "while" or token == "for" then
                table.insert(pendingLoopDepths, depth)
                loopCount = loopCount + 1
            elseif token == "repeat" then
                table.insert(output, " " .. budgetStatement .. " ")
                loopCount = loopCount + 1
            elseif token == "do" then
                for index = #pendingLoopDepths, 1, -1 do
                    if pendingLoopDepths[index] == depth then
                        table.remove(pendingLoopDepths, index)
                        table.insert(output, " " .. budgetStatement .. " ")
                        break
                    end
                end
            end
            position = finish
        else
            table.insert(output, character)
            if character == "(" or character == "{" then
                depth = depth + 1
            elseif character == ")" or character == "}" or character == "]" then
                depth = math.max(0, depth - 1)
            end
            position = position + 1
        end
    end

    if #pendingLoopDepths > 0 then
        return nil, "could not identify every loop body"
    end
    if loopCount == 0 then
        return code, 0
    end
    return "local " .. LOOP_COUNTER_NAME .. "=0;\n" .. table.concat(output), loopCount
end

local function canonicalID(value)
    if type(value) == "string" or type(value) == "number" then
        return tostring(value)
    end
    return nil
end

local function getStep(steps, id)
    if not steps or not id then
        return nil
    end
    return steps[id] or steps[tonumber(id)]
end

local function addEdge(edges, value)
    local id = canonicalID(value)
    if id then
        edges[id] = true
    end
end

local function scanWorkflow(workflowID, workflow, result, workflowEdges, classLabel)
    result.workflows = result.workflows + 1
    if result.workflows > CONFIG.maxWorkflows then
        addReason(result, 100, "workflow count exceeds safety limit", true)
        return
    end

    local steps = type(workflow) == "table" and workflow.ST or nil
    if type(steps) ~= "table" then
        addReason(result, 10, classLabel .. "/" .. workflowID .. " has no step table", false)
        return
    end

    local stepEdges = {}
    local workflowTargets = workflowEdges[workflowID] or {}
    workflowEdges[workflowID] = workflowTargets

    for rawStepID, step in pairs(steps) do
        local stepID = canonicalID(rawStepID)
        if stepID and type(step) == "table" then
            result.steps = result.steps + 1
            if result.steps > CONFIG.maxSteps then
                addReason(result, 100, "workflow step count exceeds safety limit", true)
                return
            end

            local edges = {}
            stepEdges[stepID] = edges
            addEdge(edges, step.n)

            if step.t == "branch" and type(step.b) == "table" then
                for _, branch in pairs(step.b) do
                    if type(branch) == "table" then
                        addEdge(edges, branch.n)
                        if type(branch.failWorkflow) == "string" and branch.failWorkflow ~= "" then
                            workflowTargets[branch.failWorkflow] = true
                            result.workflowCalls = result.workflowCalls + 1
                        end
                    end
                end
            end

            if type(step.e) == "table" then
                for _, fx in pairs(step.e) do
                    if type(fx) == "table" then
                        result.effects = result.effects + 1
                        if result.effects > CONFIG.maxEffects then
                            addReason(result, 100, "workflow effect count exceeds safety limit", true)
                            return
                        end

                        local effectID = tostring(fx.id or "")
                        local effectArgs = type(fx.args) == "table" and fx.args or {}
                        if effectID == "script" then
                            scanLuaCode(effectArgs[1], result, classLabel .. "/" .. workflowID)
                        elseif effectID == "item_add" then
                            result.itemAdds = result.itemAdds + 1
                            local requested = tonumber(effectArgs[2])
                            if requested and requested > CONFIG.maxSingleItemAdd then
                                addReason(result, 100, "item_add requests an excessive count", true)
                            end
                        elseif effectID == "var_object" then
                            result.variableWrites = result.variableWrites + 1
                        elseif effectID == "run_workflow" or effectID == "run_item_workflow" then
                            result.workflowCalls = result.workflowCalls + 1
                            local source = tostring(effectArgs[1] or "o")
                            local target = tostring(effectArgs[2] or "")
                            if source == "o" and target ~= "" then
                                workflowTargets[target] = true
                            end
                        elseif effectID == "var_prompt" then
                            local target = tostring(effectArgs[4] or "")
                            local source = tostring(effectArgs[5] or "o")
                            if source == "o" and target ~= "" then
                                workflowTargets[target] = true
                            end
                        end
                    end
                end
            end
        end
    end

    if result.itemAdds > CONFIG.maxItemAddEffects then
        addReason(result, 80, "object contains an excessive number of item_add effects", false)
    end

    local visiting = {}
    local complete = {}
    local cycleFound = false

    local function visitStep(stepID)
        if cycleFound or complete[stepID] then
            return
        end
        if visiting[stepID] then
            cycleFound = true
            return
        end
        visiting[stepID] = true
        local edges = stepEdges[stepID]
        if edges then
            for target in pairs(edges) do
                if getStep(steps, target) then
                    visitStep(target)
                else
                    addReason(result, 8, classLabel .. "/" .. workflowID .. " references a missing step", false)
                end
            end
        end
        visiting[stepID] = nil
        complete[stepID] = true
    end

    for stepID in pairs(stepEdges) do
        visitStep(stepID)
    end

    if cycleFound then
        addReason(result, 150, classLabel .. "/" .. workflowID .. " contains a cyclic step graph", true)
    end

    local costCache = {}
    local calculating = {}
    local function expandedCost(stepID)
        if costCache[stepID] then
            return costCache[stepID]
        end
        if calculating[stepID] then
            return CONFIG.maxExpandedSteps + 1
        end
        calculating[stepID] = true
        local cost = 1
        local edges = stepEdges[stepID]
        if edges then
            for target in pairs(edges) do
                if getStep(steps, target) then
                    cost = cost + expandedCost(target)
                    if cost > CONFIG.maxExpandedSteps then
                        break
                    end
                end
            end
        end
        calculating[stepID] = nil
        costCache[stepID] = cost
        return cost
    end

    if getStep(steps, "1") and expandedCost("1") > CONFIG.maxExpandedSteps then
        addReason(result, 120, classLabel .. "/" .. workflowID .. " expands beyond compilation budget", true)
    end
end

local function detectWorkflowCycles(workflowEdges, result, classLabel)
    local visiting = {}
    local complete = {}
    local cycleFound = false

    local function visit(workflowID)
        if cycleFound or complete[workflowID] then
            return
        end
        if visiting[workflowID] then
            cycleFound = true
            return
        end
        visiting[workflowID] = true
        local edges = workflowEdges[workflowID]
        if edges then
            for target in pairs(edges) do
                if workflowEdges[target] then
                    visit(target)
                end
            end
        end
        visiting[workflowID] = nil
        complete[workflowID] = true
    end

    for workflowID in pairs(workflowEdges) do
        visit(workflowID)
    end

    if cycleFound then
        addReason(result, 150, classLabel .. " contains a recursive workflow cycle", true)
    end
end

local function scanClass(class, classLabel, result, seenClasses, depth)
    if type(class) ~= "table" or seenClasses[class] then
        return
    end
    seenClasses[class] = true
    result.classes = result.classes + 1

    if depth > CONFIG.maxTableDepth then
        addReason(result, 100, "nested class depth exceeds safety limit", true)
        return
    end

    if type(class.SC) == "table" then
        local workflowEdges = {}
        for rawWorkflowID, workflow in pairs(class.SC) do
            local workflowID = canonicalID(rawWorkflowID)
            if workflowID then
                scanWorkflow(workflowID, workflow, result, workflowEdges, classLabel)
            end
        end
        detectWorkflowCycles(workflowEdges, result, classLabel)
    end

    local childGroups = { "IN", "QE", "ST" }
    for _, groupName in ipairs(childGroups) do
        local group = class[groupName]
        if type(group) == "table" then
            for childID, child in pairs(group) do
                scanClass(child, classLabel .. " " .. tostring(childID), result, seenClasses, depth + 1)
            end
        end
    end
end

function guard:ScanRoot(rootID, root)
    rootID = self:GetRootID(rootID)
    root = root or self:GetRootObject(rootID)
    if not rootID or type(root) ~= "table" then
        return nil
    end

    local result = newResult(rootID, self:IsOwned(rootID))
    measureValue(root, result, {}, 0)
    if not result.hardBlock then
        scanClass(root, rootID, result, {}, 0)
    end

    if not result.owned and result.scripts > 0 then
        addReason(result, 25, "object contains Lua from another author; review recommended", false)
    end
    if result.hardBlock then
        result.block = true
    end
    if self.allow[rootID] then
        result.block = false
    end

    self.results[rootID] = result
    return result
end

function guard:RefreshRoot(rootID)
    if API.script.clearRootCompilation then
        API.script.clearRootCompilation(rootID)
    end
    if Extended.Events.ON_OBJECT_UPDATED then
        Extended:TriggerEvent(Extended.Events.ON_OBJECT_UPDATED, rootID, DB.types.ITEM)
    end
    if Extended.Events.REFRESH_BAG then
        Extended:TriggerEvent(Extended.Events.REFRESH_BAG)
    end
end

function guard:ApplyVisualIsolation(rootID)
    local root = self:GetRootObject(rootID)
    if type(root) ~= "table" or root.TY ~= DB.types.ITEM then
        return false
    end

    root.BA = root.BA or {}
    local backup = root[ISOLATION_FIELD]
    if type(backup) ~= "table" then
        backup = {
            version = VERSION,
            name = root.BA.NA or rootID,
            hadIcon = root.BA.IC ~= nil,
            icon = root.BA.IC,
            usable = root.BA.US and true or false,
        }
        root[ISOLATION_FIELD] = backup
    end
    self.visualBackups[rootID] = backup

    local changed = root.BA.US ~= nil or root.BA.IC ~= ISOLATION_ICON
    root.BA.US = nil
    root.BA.IC = ISOLATION_ICON
    if changed then
        self:RefreshRoot(rootID)
    end
    return true
end

function guard:RestoreVisualIsolation(rootID)
    local root = self:GetRootObject(rootID)
    if type(root) ~= "table" or root.TY ~= DB.types.ITEM then
        return false
    end
    local backup = self.visualBackups[rootID] or root[ISOLATION_FIELD]
    if type(backup) ~= "table" then
        return false
    end

    root.BA = root.BA or {}
    if backup.hadIcon then
        root.BA.IC = backup.icon
    else
        root.BA.IC = nil
    end
    if backup.usable then
        root.BA.US = true
    else
        root.BA.US = nil
    end
    root[ISOLATION_FIELD] = nil
    self.visualBackups[rootID] = nil
    self:RefreshRoot(rootID)
    return true
end

local function normalizedItemName(value)
    value = tostring(value or "")
    value = value:gsub("|c%x%x%x%x%x%x%x%x", ""):gsub("|r", "")
    value = value:gsub("^%s+", ""):gsub("%s+$", "")
    return string.lower(value)
end

function guard:ReleaseRoot(rootID)
    rootID = self:GetRootID(rootID)
    if not rootID then
        return false
    end
    self.allow[rootID] = true
    self.manualBlocks[rootID] = nil
    self.quarantine[rootID] = nil
    local restored = self:RestoreVisualIsolation(rootID)
    return restored or self:GetRootObject(rootID) ~= nil
end

function guard:ReleaseByName(itemName)
    local wanted = normalizedItemName(itemName)
    if wanted == "" then
        return 0
    end

    local candidates = {}
    for rootID in pairs(self.quarantine) do
        candidates[rootID] = true
    end
    local stores = { G.TRP3_Tools_DB, G.TRP3_Exchange_DB, DB.global }
    for _, store in ipairs(stores) do
        if type(store) == "table" then
            for rootID, root in pairs(store) do
                if isRootID(rootID) and type(root) == "table" and type(root[ISOLATION_FIELD]) == "table" then
                    candidates[rootID] = true
                end
            end
        end
    end

    local released = 0
    for rootID in pairs(candidates) do
        local root = self:GetRootObject(rootID)
        local backup = type(root) == "table" and root[ISOLATION_FIELD] or nil
        local currentName = type(root) == "table" and root.BA and root.BA.NA or nil
        local originalName = type(backup) == "table" and backup.name or nil
        if normalizedItemName(currentName) == wanted or normalizedItemName(originalName) == wanted then
            if self:ReleaseRoot(rootID) then
                released = released + 1
                chat("released quarantine: " .. rootID)
            end
        end
    end
    return released
end

function guard:Quarantine(rootID, result, reason)
    rootID = self:GetRootID(rootID)
    if not rootID or self.allow[rootID] then
        return false
    end
    result = result or self.results[rootID] or newResult(rootID, self:IsOwned(rootID))
    if reason then
        addReason(result, 100, reason, false)
    end
    result.block = true
    self.results[rootID] = result
    self.quarantine[rootID] = result
    self:ApplyVisualIsolation(rootID)
    return true
end

function guard:IsBlocked(rootID)
    rootID = self:GetRootID(rootID)
    return rootID and not self.allow[rootID]
        and (self.manualBlocks[rootID] or self.quarantine[rootID]) ~= nil
end

function guard:NotifyBlocked(rootID, reason)
    local now = G.GetTime and G.GetTime() or 0
    local last = self.lastNotice[rootID or "?"] or -100
    if now - last >= 5 then
        self.lastNotice[rootID or "?"] = now
        chat("blocked " .. tostring(rootID or "unknown") .. ": " .. tostring(reason or "quarantined"))
    end
end

function guard:ScanAll(verbose)
    local roots = {}
    local function collect(source)
        if type(source) ~= "table" then
            return
        end
        for id, object in pairs(source) do
            if isRootID(id) and type(object) == "table" then
                roots[id] = object
            end
        end
    end

    collect(G.TRP3_Tools_DB)
    collect(G.TRP3_Exchange_DB)
    collect(DB.global)

    local scanned = 0
    local suspicious = 0
    local blocked = 0
    for rootID, root in pairs(roots) do
        local result = self:ScanRoot(rootID, root)
        if result then
            scanned = scanned + 1
            if result.score > 0 then
                suspicious = suspicious + 1
            end
            local persistedIsolation = type(root[ISOLATION_FIELD]) == "table"
            if persistedIsolation and not self.allow[rootID] then
                addReason(result, 20, "item has a persisted RPBox Guard quarantine", false)
                result.block = true
                self:Quarantine(rootID, result)
                blocked = blocked + 1
            elseif result.block and not self.allow[rootID] then
                self:Quarantine(rootID, result)
                blocked = blocked + 1
            elseif not self.manualBlocks[rootID] then
                self.quarantine[rootID] = nil
            end
        end
    end
    self.scanCount = self.scanCount + 1
    if verbose ~= false then
        chat("scan complete: " .. scanned .. " roots, " .. suspicious .. " flagged, " .. blocked .. " quarantined")
    end
    return scanned, suspicious, blocked
end

function guard:PrintResult(rootID)
    rootID = self:GetRootID(rootID)
    local result = rootID and (self.results[rootID] or self:ScanRoot(rootID))
    if not result then
        chat("unknown root ID: " .. tostring(rootID))
        return
    end
    chat(rootID .. " score=" .. result.score
        .. " owned=" .. tostring(result.owned)
        .. " blocked=" .. tostring(self:IsBlocked(rootID))
        .. " workflows=" .. result.workflows
        .. " effects=" .. result.effects
        .. " scripts=" .. result.scripts
        .. " item_add=" .. result.itemAdds)
    local limit = math.min(#result.reasons, 10)
    for index = 1, limit do
        chat("  - " .. result.reasons[index])
    end
    if #result.reasons > limit then
        chat("  - ... and " .. (#result.reasons - limit) .. " more")
    end
end

function guard:PrintList()
    local ids = {}
    for rootID in pairs(self.quarantine) do
        table.insert(ids, rootID)
    end
    for rootID in pairs(self.manualBlocks) do
        if not self.quarantine[rootID] then
            table.insert(ids, rootID)
        end
    end
    table.sort(ids)
    chat("quarantine contains " .. #ids .. " root(s)")
    for index = 1, math.min(#ids, 30) do
        local rootID = ids[index]
        local result = self.results[rootID]
        chat("  " .. rootID .. (result and (" (score " .. result.score .. ")") or ""))
    end
    if #ids > 30 then
        chat("  ... and " .. (#ids - 30) .. " more")
    end
end

local function resetRuntimeState(state, now)
    state.started = now
    state.effects = 0
    state.workflowCalls = 0
    state.variableWrites = 0
    state.itemAddCalls = 0
    state.itemAddCount = 0
    state.directAddCalls = 0
    state.directAddCount = 0
end

function guard:GetRuntimeState(rootID)
    rootID = rootID or "<unknown>"
    local now = G.GetTime and G.GetTime() or 0
    local state = self.runtime[rootID]
    if not state then
        state = {}
        self.runtime[rootID] = state
        resetRuntimeState(state, now)
    elseif now - state.started > CONFIG.runtimeWindow then
        resetRuntimeState(state, now)
    end
    return state, now
end

function guard:Trip(rootID, reason)
    if rootID and rootID ~= "<unknown>" then
        self:Quarantine(rootID, nil, "runtime circuit breaker: " .. reason)
        self:NotifyBlocked(rootID, reason)
    else
        self.unknownCircuitUntil = (G.GetTime and G.GetTime() or 0) + 10
        self:NotifyBlocked(nil, "global circuit breaker: " .. reason)
    end
    return false
end

function guard:CheckEffect(effectID, eArgs, effectArgs)
    local rootID = self:ResolveRootFromArgs(eArgs) or self.currentRoot or "<unknown>"
    local state = self:GetRuntimeState(rootID)
    state.effects = state.effects + 1
    if state.effects > CONFIG.runtimeEffects then
        return self:Trip(rootID, "too many effects in " .. CONFIG.runtimeWindow .. " seconds")
    end

    if effectID == "item_add" then
        state.itemAddCalls = state.itemAddCalls + 1
        local count = tonumber(effectArgs[2]) or 1
        state.itemAddCount = state.itemAddCount + math.max(0, count)
        if count > CONFIG.maxSingleItemAdd
            or state.itemAddCalls > CONFIG.runtimeItemAddCalls
            or state.itemAddCount > CONFIG.runtimeItemAddCount then
            return self:Trip(rootID, "item_add rate/count exceeded")
        end
    elseif effectID == "var_object" then
        state.variableWrites = state.variableWrites + 1
        if state.variableWrites > CONFIG.runtimeVariableWrites then
            return self:Trip(rootID, "object variable write rate exceeded")
        end
    elseif effectID == "run_workflow" or effectID == "run_item_workflow" then
        state.workflowCalls = state.workflowCalls + 1
        if state.workflowCalls > CONFIG.runtimeWorkflowCalls then
            return self:Trip(rootID, "workflow invocation rate exceeded")
        end
    end
    return true
end

function guard:CheckDirectAdd(classID, itemData)
    local rootID = self.currentRoot or "<unknown>"
    local state, now = self:GetRuntimeState(rootID)
    if self.unknownCircuitUntil and now < self.unknownCircuitUntil then
        return false
    end
    local count = 1
    if type(itemData) == "table" then
        count = tonumber(itemData.count) or 1
    end
    state.directAddCalls = state.directAddCalls + 1
    state.directAddCount = state.directAddCount + math.max(0, count)
    if count > CONFIG.maxSingleItemAdd
        or state.directAddCalls > CONFIG.runtimeDirectAddCalls
        or state.directAddCount > CONFIG.runtimeDirectAddCount then
        return self:Trip(rootID, "direct inventory add rate/count exceeded")
    end
    return true
end

function guard:CheckBeforeExecute(fullID, eArgs)
    local rootID = self:ResolveRootFromArgs(eArgs, fullID)
    if not rootID then
        return true, nil
    end
    if self:IsBlocked(rootID) then
        self:NotifyBlocked(rootID, "object is quarantined")
        return false, rootID
    end
    local result = self:ScanRoot(rootID)
    if result and result.block and not self.allow[rootID] then
        self:Quarantine(rootID, result)
        self:NotifyBlocked(rootID, result.reasons[1] or "static scan failed")
        return false, rootID
    end
    return true, rootID
end

guard.original = {
    executeClassScript = API.script.executeClassScript,
    playEffect = API.script.playEffect,
    runLuaScriptEffect = G.hTAsr or G.RPBG0 or API.script.runLuaScriptEffect,
    addItem = API.inventory.addItem,
}

local function callWithRoot(rootID, func, ...)
    local previousRoot = guard.currentRoot
    guard.currentRoot = rootID or previousRoot
    local packed = { G.pcall(func, ...) }
    guard.currentRoot = previousRoot
    if not packed[1] then
        G.error(packed[2], 0)
    end
    return G.unpack(packed, 2)
end

guard.wrappedExecuteClassScript = function(scriptID, classScripts, eArgs, fullID)
    local allowed, rootID = guard:CheckBeforeExecute(fullID, eArgs)
    if not allowed then
        return 0
    end
    return callWithRoot(rootID, guard.original.executeClassScript, scriptID, classScripts, eArgs, fullID)
end

guard.wrappedPlayEffect = function(effectID, shouldBeSecured, eArgs, ...)
    local rootID = guard:ResolveRootFromArgs(eArgs) or guard.currentRoot
    if rootID and guard:IsBlocked(rootID) then
        guard:NotifyBlocked(rootID, "effect blocked for quarantined object")
        return 0
    end
    local effectArgs = { ... }
    if not guard:CheckEffect(effectID, eArgs, effectArgs) then
        return 0
    end
    return callWithRoot(rootID, guard.original.playEffect, effectID, shouldBeSecured, eArgs, ...)
end

guard.wrappedRunLuaScriptEffect = function(code, eArgs, secured)
    local rootID = guard:ResolveRootFromArgs(eArgs)
    if rootID and guard:IsBlocked(rootID) then
        guard:NotifyBlocked(rootID, "Lua effect blocked for quarantined object")
        return 0
    end

    local luaResult = newResult(rootID or "<unknown>", guard:IsOwned(rootID))
    scanLuaCode(code, luaResult, "runtime Lua effect")
    local shouldBlock = luaResult.hardBlock
    if shouldBlock and not guard.allow[rootID] then
        if rootID then
            guard:Quarantine(rootID, luaResult, "runtime Lua effect rejected")
        end
        guard:NotifyBlocked(rootID, luaResult.reasons[1] or "foreign Lua effect")
        return 0
    end

    if type(eArgs) == "table" then
        if rootID and not guard:IsBlocked(rootID) then
            eArgs._G = G
        else
            eArgs._G = nil
        end
    end

    local guardedCode, instrumentError = guard:InstrumentLua(code)
    if not guardedCode then
        if rootID then
            guard:Quarantine(rootID, luaResult, "Lua loop instrumentation failed: " .. tostring(instrumentError))
        end
        guard:NotifyBlocked(rootID, "Lua loop instrumentation failed")
        return 0
    end

    return callWithRoot(rootID, guard.original.runLuaScriptEffect, guardedCode, eArgs, secured)
end

guard.wrappedAddItem = function(givenContainer, classID, itemData, dropIfFull, toSlot)
    if not guard:CheckDirectAdd(classID, itemData) then
        return 1
    end
    return guard.original.addItem(givenContainer, classID, itemData, dropIfFull, toSlot)
end

function guard:Restore(silent)
    if API.script.executeClassScript == self.wrappedExecuteClassScript then
        API.script.executeClassScript = self.original.executeClassScript
    end
    if API.script.playEffect == self.wrappedPlayEffect then
        API.script.playEffect = self.original.playEffect
    end
    if API.script.runLuaScriptEffect == self.wrappedRunLuaScriptEffect then
        API.script.runLuaScriptEffect = self.original.runLuaScriptEffect
    end
    if API.inventory.addItem == self.wrappedAddItem then
        API.inventory.addItem = self.original.addItem
    end
    if API.script.clearAllCompilations then
        API.script.clearAllCompilations()
    end
    self.installed = false
    if not silent then
        chat("hooks restored; protection is OFF")
    end
end

function guard:Install()
    API.script.executeClassScript = self.wrappedExecuteClassScript
    API.script.playEffect = self.wrappedPlayEffect
    API.script.runLuaScriptEffect = self.wrappedRunLuaScriptEffect
    API.inventory.addItem = self.wrappedAddItem
    if API.script.clearAllCompilations then
        API.script.clearAllCompilations()
    end
    self.installed = true
end

local function trim(value)
    return (tostring(value or ""):gsub("^%s+", ""):gsub("%s+$", ""))
end

function guard:HandleCommand(message)
    local command, argument = string.match(trim(message), "^(%S+)%s*(.-)$")
    command = string.lower(command or "status")
    argument = trim(argument)

    if command == "scan" then
        self:ScanAll(true)
    elseif command == "list" then
        self:PrintList()
    elseif command == "inspect" then
        self:PrintResult(argument)
    elseif command == "release" then
        local released = self:ReleaseByName(argument)
        if released == 0 then
            chat("no quarantined item matched exact name: " .. argument)
        else
            chat("released " .. released .. " quarantined item(s); session allowlist applied")
        end
    elseif command == "allow" then
        local rootID = self:GetRootID(argument)
        if rootID then
            self:ReleaseRoot(rootID)
            chat("session allowlist added and visual quarantine removed: " .. rootID)
        else
            chat("usage: /rpboxguard allow ROOT_ID")
        end
    elseif command == "block" then
        local rootID = self:GetRootID(argument)
        if rootID then
            self.allow[rootID] = nil
            self.manualBlocks[rootID] = true
            self:Quarantine(rootID, self.results[rootID], "manually blocked")
            chat("manually quarantined: " .. rootID)
        else
            chat("usage: /rpboxguard block ROOT_ID")
        end
    elseif command == "restore" then
        self:Restore(false)
    elseif command == "status" or command == "" then
        local count = 0
        for _ in pairs(self.quarantine) do
            count = count + 1
        end
        chat("version=" .. self.version
            .. " installed=" .. tostring(self.installed)
            .. " scans=" .. self.scanCount
            .. " quarantined=" .. count)
        chat("commands: scan, list, inspect ID, release NAME, allow ID, block ID, restore")
    else
        chat("unknown command: " .. command)
        chat("commands: scan, list, inspect ID, release NAME, allow ID, block ID, restore")
    end
end

guard:Install()

G.SLASH_RPBOXTRP3GUARD1 = "/rpboxguard"
G.SlashCmdList = G.SlashCmdList or {}
G.SlashCmdList.RPBOXTRP3GUARD = function(message)
    guard:HandleCommand(message)
end

guard:ScanAll(true)
chat("installed for this UI session; no objects were deleted")
