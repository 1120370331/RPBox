-- Dynamic smoke test for RPBox_Addon/MainFrame.lua.
--
-- Run from the repository root with:
--   npx --yes --package=fengari-node-cli fengari addon/tests/main_frame_playback_smoke.lua
--
-- This is intentionally a small WoW UI runtime, not a reimplementation of the
-- playback algorithms.  It loads the production MainFrame.lua chunk and drives
-- its public entry point, button scripts, edit-box scripts, and timer callbacks.

local function fail(message)
    error("[main-frame-playback-smoke] " .. tostring(message), 2)
end

local function assert_true(value, message)
    if not value then fail(message or "expected truthy value") end
end

local function assert_equal(actual, expected, message)
    if actual ~= expected then
        fail((message or "values differ") .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual))
    end
end

local function assert_contains(text, expected, message)
    if not tostring(text or ""):find(expected, 1, true) then
        fail((message or "text did not contain expected value") .. ": " .. tostring(expected))
    end
end

local function assert_not_contains(text, expected, message)
    if tostring(text or ""):find(expected, 1, true) then
        fail((message or "text contained unexpected value") .. ": " .. tostring(expected))
    end
end

local Timer = {
    now = 0,
    nextSequence = 0,
    queue = {},
    callbacksRun = 0,
    maxScanDelta = 0,
    maxRenderDelta = 0,
    maxRowsCreated = 0,
    completedScans = 0,
}

function Timer.after(delay, callback)
    Timer.nextSequence = Timer.nextSequence + 1
    Timer.queue[#Timer.queue + 1] = {
        due = Timer.now + math.max(tonumber(delay) or 0, 0),
        sequence = Timer.nextSequence,
        callback = callback,
    }
end

local function sort_timers()
    table.sort(Timer.queue, function(a, b)
        if a.due ~= b.due then return a.due < b.due end
        return a.sequence < b.sequence
    end)
end

function Timer.peek()
    sort_timers()
    return Timer.queue[1]
end

function Timer.run_one()
    sort_timers()
    local task = table.remove(Timer.queue, 1)
    if not task then return false end

    Timer.now = math.max(Timer.now, task.due)
    Timer.callbacksRun = Timer.callbacksRun + 1

    local frame = _G.RPBoxMainFrame
    local scan = frame and frame.logScan or nil
    local scanBefore = scan and scan.scanned or nil
    local state = frame and frame.logState or nil
    local loadedBefore = state and state.loadedCount or nil

    local ok, err = xpcall(task.callback, debug.traceback)
    if not ok then fail("timer callback failed:\n" .. tostring(err)) end

    if scan and scanBefore ~= nil then
        local delta = (scan.scanned or scanBefore) - scanBefore
        Timer.maxScanDelta = math.max(Timer.maxScanDelta, delta)
        assert_true(delta <= 250, "a scan timer processed more than 250 records: " .. tostring(delta))
        if frame and frame.logScan == nil then
            Timer.completedScans = Timer.completedScans + 1
        end
    end

    if state and loadedBefore ~= nil and frame and frame.logState == state then
        local delta = (state.loadedCount or loadedBefore) - loadedBefore
        Timer.maxRenderDelta = math.max(Timer.maxRenderDelta, delta)
        assert_true(delta <= 20, "a render timer created more than 20 visible rows: " .. tostring(delta))
    end

    if frame and frame.logContent and frame.logContent.rows then
        Timer.maxRowsCreated = math.max(Timer.maxRowsCreated, #frame.logContent.rows)
        assert_true(#frame.logContent.rows <= 120, "more than 120 log row frames were created")
    end
    return true
end

function Timer.drain(limit)
    limit = limit or 20000
    local ran = 0
    while #Timer.queue > 0 do
        ran = ran + 1
        if ran > limit then fail("timer queue did not settle after " .. tostring(limit) .. " callbacks") end
        Timer.run_one()
    end
    return ran
end

C_Timer = { After = Timer.after }
GetTime = function() return Timer.now end

local ObjectMethods = {}
local ObjectMeta = { __index = ObjectMethods }

local function new_object(kind, name, parent, template)
    local object = setmetatable({
        _kind = kind,
        _name = name,
        _parent = parent,
        _template = template,
        _shown = true,
        _enabled = true,
        _checked = false,
        _text = "",
        _width = 500,
        _height = 400,
        _verticalScroll = 0,
        _scripts = {},
        _hooks = {},
        _children = {},
        _regions = {},
        _textColor = { 1, 1, 1, 1 },
        _alpha = 1,
    }, ObjectMeta)

    if parent and parent._children then
        parent._children[#parent._children + 1] = object
    end
    if name then _G[name] = object end
    return object
end

function ObjectMethods:SetSize(width, height) self._width, self._height = width, height end
function ObjectMethods:SetWidth(width) self._width = width end
function ObjectMethods:SetHeight(height) self._height = height end
function ObjectMethods:GetWidth() return self._width end
function ObjectMethods:GetHeight() return self._height end
function ObjectMethods:SetPoint(...) self._point = {...} end
function ObjectMethods:SetAllPoints(target) self._allPoints = target end
function ObjectMethods:ClearAllPoints() self._point = nil end
function ObjectMethods:GetName() return self._name end
function ObjectMethods:SetClipsChildren(value) self._clipsChildren = value end
function ObjectMethods:SetMovable(value) self._movable = value end
function ObjectMethods:SetResizable(value) self._resizable = value end
function ObjectMethods:SetResizeBounds(...) self._resizeBounds = {...} end
function ObjectMethods:SetClampedToScreen(value) self._clamped = value end
function ObjectMethods:EnableMouse(value) self._mouseEnabled = value end
function ObjectMethods:RegisterForDrag(...) self._dragButtons = {...} end
function ObjectMethods:SetFrameStrata(value) self._frameStrata = value end
function ObjectMethods:SetNormalTexture(value) self._normalTexture = value end
function ObjectMethods:SetHighlightTexture(value) self._highlightTexture = value end
function ObjectMethods:SetPushedTexture(value) self._pushedTexture = value end
function ObjectMethods:SetBackdrop(value) self._backdrop = value end
function ObjectMethods:SetBackdropColor(...) self._backdropColor = {...} end
function ObjectMethods:SetBackdropBorderColor(...) self._backdropBorderColor = {...} end
function ObjectMethods:SetAlpha(value) self._alpha = value end
function ObjectMethods:GetAlpha() return self._alpha end
function ObjectMethods:SetTexture(value) self._texture = value; self._atlas = nil end
function ObjectMethods:GetTexture() return self._texture end
function ObjectMethods:SetAtlas(value) self._atlas = value; self._texture = nil end
function ObjectMethods:GetAtlas() return self._atlas end
function ObjectMethods:SetVertexColor(r, g, b, a) self._vertexColor = { r, g, b, a or 1 } end
function ObjectMethods:GetVertexColor() return table.unpack(self._vertexColor or { 1, 1, 1, 1 }) end
function ObjectMethods:SetColorTexture(r, g, b, a) self._colorTexture = { r, g, b, a or 1 } end
function ObjectMethods:SetTexCoord(...) self._texCoord = {...} end
function ObjectMethods:SetAutoFocus(value) self._autoFocus = value end
function ObjectMethods:SetNumeric(value) self._numeric = value end
function ObjectMethods:SetMultiLine(value) self._multiLine = value end
function ObjectMethods:SetFontObject(value) self._fontObject = value end
function ObjectMethods:SetJustifyH(value) self._justifyH = value end
function ObjectMethods:SetJustifyV(value) self._justifyV = value end
function ObjectMethods:SetWordWrap(value) self._wordWrap = value end
function ObjectMethods:SetNonSpaceWrap(value) self._nonSpaceWrap = value end
function ObjectMethods:SetText(value)
    self._text = tostring(value or "")
    if self._fontString then self._fontString._text = self._text end
end
function ObjectMethods:GetText() return self._text end
function ObjectMethods:SetTextColor(r, g, b, a) self._textColor = { r, g, b, a or 1 } end
function ObjectMethods:GetTextColor() return table.unpack(self._textColor) end
function ObjectMethods:GetStringHeight()
    local _, newlines = tostring(self._text or ""):gsub("\n", "")
    return math.max(16, (newlines + 1) * 16)
end
function ObjectMethods:SetChecked(value) self._checked = not not value end
function ObjectMethods:GetChecked() return self._checked end
function ObjectMethods:SetEnabled(value)
    local enabled = not not value
    if self._enabled == enabled then return end
    self._enabled = enabled
    local event = enabled and "OnEnable" or "OnDisable"
    if self._scripts[event] then self._scripts[event](self) end
    for _, callback in ipairs(self._hooks[event] or {}) do callback(self) end
end
function ObjectMethods:IsEnabled() return self._enabled end
function ObjectMethods:SetScript(event, callback) self._scripts[event] = callback end
function ObjectMethods:HookScript(event, callback)
    self._hooks[event] = self._hooks[event] or {}
    self._hooks[event][#self._hooks[event] + 1] = callback
end
function ObjectMethods:Show()
    self._shown = true
    if self._scripts.OnShow then self._scripts.OnShow(self) end
end
function ObjectMethods:Hide()
    local wasShown = self._shown
    self._shown = false
    if wasShown and self._scripts.OnHide then self._scripts.OnHide(self) end
    if wasShown then
        for _, callback in ipairs(self._hooks.OnHide or {}) do callback(self) end
    end
end
function ObjectMethods:IsShown() return self._shown end
function ObjectMethods:SetScrollChild(child) self._scrollChild = child end
function ObjectMethods:UpdateScrollChildRect() end
function ObjectMethods:GetVerticalScroll() return self._verticalScroll end
function ObjectMethods:SetVerticalScroll(value) self._verticalScroll = value end
function ObjectMethods:GetVerticalScrollRange()
    local childHeight = self._scrollChild and self._scrollChild._height or 0
    return math.max(childHeight - (self._height or 0), 0)
end
function ObjectMethods:GetChildren() return table.unpack(self._children) end
function ObjectMethods:GetRegions() return table.unpack(self._regions) end
function ObjectMethods:GetObjectType() return self._kind end
function ObjectMethods:IsObjectType(kind) return self._kind == kind end
function ObjectMethods:CreateTexture(name, layer)
    local texture = new_object("Texture", name, nil, nil)
    texture._layer = layer
    self._regions[#self._regions + 1] = texture
    return texture
end
function ObjectMethods:CreateFontString(name, layer, template)
    local fontString = new_object("FontString", name, nil, template)
    self._regions[#self._regions + 1] = fontString
    return fontString
end
function ObjectMethods:GetFontString() return self._fontString end
function ObjectMethods:GetThumbTexture() return self.ThumbTexture end
function ObjectMethods:ClearFocus() self._focused = false end
function ObjectMethods:SetFocus() self._focused = true end
function ObjectMethods:HighlightText() self._highlighted = true end
function ObjectMethods:StartMoving() self._moving = true end
function ObjectMethods:StopMovingOrSizing() self._moving = false end
function ObjectMethods:StartSizing() self._sizing = true end
function ObjectMethods:RegisterEvent(event) self._registeredEvents = self._registeredEvents or {}; self._registeredEvents[event] = true end

UIParent = new_object("Frame", "UIParent", nil, nil)
GameFontNormal = {}
GameFontNormalSmall = {}
GameFontNormalLarge = {}
GameFontDisableSmall = {}
GameFontHighlightSmall = {}

function CreateFrame(kind, name, parent, template)
    local frame = new_object(kind, name, parent, template)
    if template and template:find("BasicFrameTemplateWithInset", 1, true) then
        frame.TitleText = frame:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        frame.CloseButton = new_object("Button", nil, frame, nil)
        frame.Bg = new_object("Texture", nil, nil, nil)
        frame.TitleBg = new_object("Texture", nil, nil, nil)
        frame.NineSlice = new_object("Frame", nil, frame, nil)
        frame.Inset = new_object("Frame", nil, frame, nil)
        frame._regions[#frame._regions + 1] = frame.Bg
        frame._regions[#frame._regions + 1] = frame.TitleBg
    end
    if kind == "Button" and template and template:find("UIPanelButtonTemplate", 1, true) then
        frame._fontString = frame:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        local normalTexture = new_object("Texture", nil, nil, nil)
        frame._regions[#frame._regions + 1] = normalTexture
    elseif kind == "EditBox" and template and template:find("InputBoxTemplate", 1, true) then
        local inputTexture = new_object("Texture", nil, nil, nil)
        frame._regions[#frame._regions + 1] = inputTexture
    elseif template and template:find("UIDropDownMenuTemplate", 1, true) then
        local dropdownTexture = new_object("Texture", nil, nil, nil)
        frame._regions[#frame._regions + 1] = dropdownTexture
        frame.Button = new_object("Button", nil, frame, nil)
        frame.Button._nativeTexture = frame.Button:CreateTexture(nil, "ARTWORK")
        frame.Text = frame:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    elseif kind == "ScrollFrame" and template and template:find("UIPanelScrollFrameTemplate", 1, true) then
        frame.ScrollBar = new_object("Slider", nil, frame, "UIPanelScrollBarTemplate")
        frame.ScrollBar.ScrollUpButton = new_object("Button", nil, frame.ScrollBar, nil)
        frame.ScrollBar.ScrollDownButton = new_object("Button", nil, frame.ScrollBar, nil)
        frame.ScrollBar.ScrollUpButton._nativeTexture = frame.ScrollBar.ScrollUpButton:CreateTexture(nil, "ARTWORK")
        frame.ScrollBar.ScrollDownButton._nativeTexture = frame.ScrollBar.ScrollDownButton:CreateTexture(nil, "ARTWORK")
        frame.ScrollBar.ThumbTexture = frame.ScrollBar:CreateTexture(nil, "ARTWORK")
        frame.ScrollBar.ThumbTexture:SetTexture("native-thumb")
        frame.ScrollBar.ThumbTexture:SetSize(16, 24)
    elseif kind == "CheckButton" and template and template:find("UICheckButtonTemplate", 1, true) then
        local checkTexture = new_object("Texture", nil, nil, nil)
        frame._regions[#frame._regions + 1] = checkTexture
    end
    return frame
end

function UIDropDownMenu_Initialize(dropdown, initializer) dropdown._dropdownInitializer = initializer end
function UIDropDownMenu_CreateInfo() return {} end
function UIDropDownMenu_AddButton(info, level) end
function UIDropDownMenu_SetText(dropdown, text) dropdown:SetText(text) end
function UIDropDownMenu_SetWidth(dropdown, width) dropdown:SetWidth(width) end

StaticPopupDialogs = {}
function StaticPopup_Show(name) return StaticPopupDialogs[name] end

RAID_CLASS_COLORS = {}
CLASS_ICON_TCOORDS = {}
ChatTypeInfo = {
    SAY = { r = 1.00, g = 0.25, b = 0.50 },
    YELL = { r = 1.00, g = 0.10, b = 0.10 },
    EMOTE = { r = 1.00, g = 0.50, b = 0.25 },
    PARTY = { r = 0.67, g = 0.67, b = 1.00 },
    RAID = { r = 1.00, g = 0.50, b = 0.00 },
    WHISPER = { r = 1.00, g = 0.50, b = 1.00 },
    WHISPER_INFORM = { r = 1.00, g = 0.50, b = 1.00 },
    GUILD = { r = 0.25, g = 1.00, b = 0.25 },
}
TRP3_API = nil
ReloadUI = function() end
GetRealmName = function() return "SmokeRealm" end

max = math.max
min = math.min
floor = math.floor
abs = math.abs
format = string.format
date = os.date
time = os.time

function strtrim(value)
    return tostring(value or ""):match("^%s*(.-)%s*$")
end

function strsplit(delimiter, value)
    value = tostring(value or "")
    local index = value:find(delimiter, 1, true)
    if not index then return value end
    return value:sub(1, index - 1), value:sub(index + #delimiter)
end

function wipe(values)
    for key in pairs(values) do values[key] = nil end
    return values
end

RPBox_Config = {
    enabled = true,
    showIcon = false,
    whitelist = {},
    blacklist = {},
    channels = {},
}

local snapshots = {
    ["snap-old"] = { ref = "profile-old", gameID = "Hero-SmokeRealm", FN = "旧卡名", pn = "旧人物卡" },
    ["snap-new"] = { ref = "profile-new", gameID = "Hero-SmokeRealm", FN = "新卡名", pn = "新人物卡" },
}

local newMessageCallbacks = {}
local listChangeCallbacks = {}
local itemGuardToggles = {}
local itemGuardActions = {}
local itemGuardChangeCallback
local itemGuardBlacklistActions = {}
local itemGuardBlacklistChangeCallback
local itemGuardBlacklistEntries = {
    { identity = "蕾火演员死冯-金色平原", source = "builtin", reason = "RPBox 内置恶意道具来源名单" },
    { identity = "工作人员二号-金色平原", source = "builtin", reason = "RPBox 内置恶意道具来源名单" },
    { identity = "绿宝石兽-金色平原", source = "builtin", reason = "RPBox 内置恶意道具来源名单" },
    { identity = "userbad-smokerealm", source = "user", reason = "手动添加" },
}
local itemGuardEntries = {
    {
        rootID = "risk-item",
        itemName = "风险道具",
        reasons = { "步骤循环中包含重复添加物品行为" },
        score = 140,
        status = "quarantined",
        quarantined = true,
        ignored = false,
        icon = "inv_misc_questionmark",
    },
}
for index = 2, 47 do
    local status
    if index % 3 == 0 then
        status = "ignored"
    elseif index % 3 == 1 then
        status = "released"
    else
        status = "quarantined"
    end
    itemGuardEntries[#itemGuardEntries + 1] = {
        rootID = string.format("risk-%03d", index),
        itemName = string.format("批量风险道具 %02d", index),
        reasons = {
            index == 37 and "海妖特殊原因：异常添加物品" or "单次添加物品数量异常",
            "用于风险筛选回放",
        },
        score = 100 + index,
        status = status,
        quarantined = status == "quarantined",
        ignored = status == "ignored",
        icon = "inv_misc_questionmark",
    }
end
local function render_chat_link_labels(text)
    text = tostring(text or "")
    text = text:gsub("|c%x%x%x%x%x%x%x%x|H[^|]+|h(%b[])|h|r", "%1")
    text = text:gsub("|H[^|]+|h(%b[])|h", "%1")
    text = text:gsub("%[TRP3:([^%]]+)%]", function(content)
        local name = content:match("^(.*):%d+$") or content
        return "[" .. name .. "]"
    end)
    text = text:gsub("|T.-|t", "")
    text = text:gsub("|c%x%x%x%x%x%x%x%x", "")
    text = text:gsub("|r", "")
    return text
end

local ns = {
    L = {},
    GetProfileSnapshot = function(key) return snapshots[key] end,
    GetCachedProfile = function() return nil end,
    GetPlayerID = function() return "Tester-SmokeRealm" end,
    RenderChatLinkLabels = function(text) return render_chat_link_labels(text) end,
    RemoveFromWhitelist = function() end,
    RemoveFromBlacklist = function() end,
    RegisterOnNewMessage = function(callback) newMessageCallbacks[#newMessageCallbacks + 1] = callback end,
    RegisterOnListChange = function(callback) listChangeCallbacks[#listChangeCallbacks + 1] = callback end,
    ItemGuard = {
        SetEnabled = function(_, enabled) itemGuardToggles[#itemGuardToggles + 1] = enabled end,
        GetRiskEntries = function() return itemGuardEntries end,
        ScanAll = function() itemGuardActions[#itemGuardActions + 1] = { action = "scan" } end,
        SetIsolation = function(_, rootID, isolated)
            itemGuardActions[#itemGuardActions + 1] = { action = "isolate", rootID = rootID, value = isolated }
        end,
        SetIgnored = function(_, rootID, ignored)
            itemGuardActions[#itemGuardActions + 1] = { action = "ignore", rootID = rootID, value = ignored }
        end,
        RequestIgnore = function(_, rootID)
            itemGuardActions[#itemGuardActions + 1] = { action = "requestIgnore", rootID = rootID }
        end,
        RegisterOnChanged = function(_, callback) itemGuardChangeCallback = callback end,
    },
    ItemGuardBlacklist = {
        Initialize = function() return true end,
        GetEntries = function() return itemGuardBlacklistEntries end,
        AddUser = function(identity)
            local normalized = string.lower((identity or ""):gsub("%s+", ""))
            itemGuardBlacklistActions[#itemGuardBlacklistActions + 1] = { action = "add", identity = normalized }
            itemGuardBlacklistEntries[#itemGuardBlacklistEntries + 1] = {
                identity = normalized,
                source = "user",
            }
            if itemGuardBlacklistChangeCallback then itemGuardBlacklistChangeCallback() end
            return true, "已加入来源黑名单"
        end,
        RemoveUser = function(identity)
            itemGuardBlacklistActions[#itemGuardBlacklistActions + 1] = { action = "remove", identity = identity }
            for index, entry in ipairs(itemGuardBlacklistEntries) do
                if entry.identity == identity and entry.source == "user" then
                    table.remove(itemGuardBlacklistEntries, index)
                    break
                end
            end
            if itemGuardBlacklistChangeCallback then itemGuardBlacklistChangeCallback() end
            return true, "已移出来源黑名单"
        end,
        RegisterOnChanged = function(callback) itemGuardBlacklistChangeCallback = callback end,
    },
}

local function trigger_new_message(times)
    for _ = 1, times or 1 do
        for _, callback in ipairs(newMessageCallbacks) do callback() end
    end
end

local TOTAL_INITIAL_RECORDS = 3205
local DEFAULT_VISIBLE_RECORDS = TOTAL_INITIAL_RECORDS - 1 -- system timeline nodes are hidden by default
local BASE_TIMESTAMP = 1700000000
local buckets = {
    [1] = {},
    [2] = {},
    [3] = {},
    [4] = {},
}

local function regular_record(index, marker)
    local profileNumber = (index % 7) + 1
    return {
        sv = 2,
        id = "record-" .. tostring(index),
        sid = "smoke-session",
        seq = index,
        t = BASE_TIMESTAMP + index,
        c = "SAY",
        m = marker or string.format("record-%05d", index),
        s = "Speaker" .. tostring(profileNumber) .. "-SmokeRealm",
        ref = "profile-" .. tostring(profileNumber),
        snapshot = {
            ref = "profile-" .. tostring(profileNumber),
            gameID = "Speaker" .. tostring(profileNumber) .. "-SmokeRealm",
            FN = "Speaker " .. tostring(profileNumber),
            IC = "INV_Misc_Book_09",
        },
        listeners = {
            { gameID = "Tester-SmokeRealm", ref = "listener-profile" },
        },
    }
end

for index = 1, TOTAL_INITIAL_RECORDS do
    local record
    if index == 1 then
        record = {
            timestamp = BASE_TIMESTAMP + index,
            channel = "CHAT_MSG_SAY",
            content = "legacy-marker",
            sender = {
                gameID = "Legacy-SmokeRealm",
                trp3 = { FN = "历史别名", LN = "旧记录" },
            },
        }
    elseif index == 123 then
        record = {
            sv = 2,
            id = "switch-event",
            sid = "smoke-session",
            seq = index,
            t = BASE_TIMESTAMP + index,
            c = "SYSTEM",
            mk = "S",
            s = "Hero-SmokeRealm",
            ps = "snap-new",
            ev = {
                kind = "profile_switch",
                certainty = "exact",
                from = { ref = "profile-old", ps = "snap-old", n = "旧卡名", pn = "旧人物卡" },
                to = { ref = "profile-new", ps = "snap-new", n = "新卡名", pn = "新人物卡" },
            },
        }
    elseif index == TOTAL_INITIAL_RECORDS then
        record = regular_record(index, "|cffffffff|Hitem:6948::::::::80:64:::::::::|h[炉石]|h|r |cff71d5ff|Hspell:100|h[冲锋]|h|r [TRP3:海兽之血宝石:1]")
    else
        record = regular_record(index)
    end

    local bucketIndex
    if index <= 900 then bucketIndex = 1
    elseif index <= 1800 then bucketIndex = 2
    elseif index <= 2700 then bucketIndex = 3
    else bucketIndex = 4 end
    buckets[bucketIndex][#buckets[bucketIndex] + 1] = record
end

RPBox_ChatLog = {
    ["2023-11-14"] = {
        ["01"] = buckets[1],
        ["12"] = buckets[2],
    },
    ["2023-11-15"] = {
        ["03"] = buckets[3],
        ["23"] = buckets[4],
    },
}

local nextRecordIndex = TOTAL_INITIAL_RECORDS
local function append_live_records(count, prefix)
    for i = 1, count do
        nextRecordIndex = nextRecordIndex + 1
        local marker = string.format("%s-%03d", prefix, i)
        buckets[4][#buckets[4] + 1] = regular_record(nextRecordIndex, marker)
    end
end

local function load_addon_file(path)
    local chunk, loadError = loadfile(path)
    if not chunk then fail("could not load production file " .. path .. ": " .. tostring(loadError)) end
    local ok, loadRuntimeError = xpcall(function() chunk("RPBox_Addon", ns) end, debug.traceback)
    if not ok then fail("production file failed during load " .. path .. ":\n" .. tostring(loadRuntimeError)) end
end

load_addon_file("addon/RPBox_Addon/Theme.lua")
load_addon_file("addon/RPBox_Addon/MainFrame.lua")

local function click(button)
    assert_true(button ~= nil, "button was not created")
    if button._enabled == false then return false end
    local callback = button._scripts.OnClick
    assert_true(type(callback) == "function", "button has no OnClick script")
    callback(button)
    return true
end

local function press_enter(editBox, text)
    editBox:SetText(text)
    local callback = editBox._scripts.OnEnterPressed
    assert_true(type(callback) == "function", "edit box has no OnEnterPressed script")
    callback(editBox)
end

local function find_tab(frame, tabName)
    for _, button in ipairs(frame.tabButtons or {}) do
        if button.tabName == tabName then return button end
    end
    fail("tab not found: " .. tostring(tabName))
end

local function count_text_lines(text)
    if text == nil or text == "" then return 0 end
    local _, newlineCount = text:gsub("\n", "")
    return newlineCount + 1
end

local function count_selected(values)
    local count = 0
    for _, selected in pairs(values or {}) do
        if selected then count = count + 1 end
    end
    return count
end

local function assert_settled(frame, message)
    assert_equal(#Timer.queue, 0, (message or "playback") .. " left pending timers")
    assert_true(frame.logScan == nil, (message or "playback") .. " left a scan active")
    assert_true(frame.logState == nil or frame.logState.rendering == false, (message or "playback") .. " left rendering active")
end

-- Load all history through the real asynchronous scan and bounded renderer.
ns.OpenMainFrame()
local frame = _G.RPBoxMainFrame
assert_true(frame ~= nil and frame:IsShown(), "OpenMainFrame did not create and show the production frame")
assert_equal(frame._rpboxTheme, "modern", "new installs should use the modern addon theme")
assert_true(frame._rpboxModernChrome and frame._rpboxModernChrome:IsShown(), "modern title chrome was not shown")
assert_true(not frame.TitleText:IsShown(), "native title chrome should be hidden by the modern theme")
assert_true(frame._backdrop ~= nil, "modern frame backdrop was not applied")
assert_equal(frame.latestPageBtn:GetText(), "首页", "first-page button should use standard pagination wording")
assert_equal(frame.prevPageBtn:GetText(), "上一页", "previous-page button should use standard pagination wording")
assert_equal(frame.nextPageBtn:GetText(), "下一页", "next-page button should use standard pagination wording")
assert_equal(frame.dateDropdown.Button._nativeTexture:GetAlpha(), 0, "modern dropdown should hide native arrow artwork")
assert_true(frame.logScroll._rpboxScrollTrack:IsShown(), "modern scroll track was not shown")
assert_equal(frame.logScroll.ScrollBar.ScrollUpButton._nativeTexture:GetAlpha(), 0, "modern scrollbar should hide native artwork")
Timer.drain()
assert_settled(frame, "initial playback")
assert_equal(frame.logState.totalMatched, DEFAULT_VISIBLE_RECORDS, "system timeline nodes should be hidden by default")
assert_equal(frame.logState.pageSize, 80, "default playback page size")
assert_equal(frame.logState.page, 1, "first open should show the latest page")
assert_equal(frame.logState.startIndex, 1, "first page should start at the newest display position")
assert_equal(frame.logState.endIndex, 80, "default page should contain 80 records")
assert_equal(frame.logShownRowCount, 80, "default page visible row count")
local latestRowText = frame.logContent.rows[1].text:GetText()
assert_contains(latestRowText, "[炉石]", "native item link label was not rendered")
assert_contains(latestRowText, "[冲锋]", "native spell link label was not rendered")
assert_contains(latestRowText, "[海兽之血宝石]", "TRP3 text link label was not rendered")
assert_not_contains(latestRowText, "|Hitem", "native item link source leaked into display")
assert_not_contains(latestRowText, "|Hspell", "native spell link source leaked into display")
assert_not_contains(latestRowText, "[TRP3:海兽之血宝石:1]", "TRP3 link source leaked into display")

-- Compact sidebar controls must stay inside the filter rail and open purpose-built pickers.
assert_equal(frame.filterFrame:GetWidth(), 206, "filter rail width should contain all controls")
assert_true(frame.filterFrame._clipsChildren ~= true, "filter rail should not clip or dim its controls")
assert_equal(frame.dateDropdown:GetWidth(), 154, "date dropdown should fit inside the filter rail")
assert_equal(frame.speakerDropdown:GetWidth(), 182, "speaker picker trigger should fit inside the filter rail")
click(frame.exactStartPickerBtn)
assert_true(frame.dateTimePicker and frame.dateTimePicker:IsShown(), "start-time picker did not open")
assert_equal(#frame.dateTimePicker.dayButtons, 42, "date picker should render a complete calendar grid")
click(frame.dateTimePicker.cancelBtn)
assert_true(not frame.dateTimePicker:IsShown(), "date picker cancel should close the picker")
click(frame.exactEndPickerBtn)
assert_equal(frame.dateTimePicker.hourBox:GetText(), "23", "empty end-time picker should default to end of day")
assert_equal(frame.dateTimePicker.minuteBox:GetText(), "59", "empty end-time picker should default to end of day")
click(frame.dateTimePicker.cancelBtn)

click(frame.speakerDropdown)
local speakerPicker = frame.speakerPicker
assert_true(speakerPicker and speakerPicker:IsShown(), "speaker picker did not open")
assert_true(speakerPicker.totalPages >= 2, "speaker picker fixture should exercise pagination")
assert_true(speakerPicker.rows[1].avatar._texture ~= nil, "speaker picker row should render an avatar")
local sawProfileIcon = false
for _, row in ipairs(speakerPicker.rows) do
    if tostring(row.avatar._texture or ""):find("INV_Misc_Book_09", 1, true) then sawProfileIcon = true end
end
assert_true(sawProfileIcon, "speaker picker should use the participant profile icon when available")
click(speakerPicker.rows[1])
click(speakerPicker.rows[2])
assert_true(speakerPicker:IsShown(), "speaker picker should remain open during multi-selection")
assert_equal(count_selected(speakerPicker.draftSelected), 2, "speaker picker should retain multiple draft selections")
click(speakerPicker.nextPageBtn)
assert_equal(speakerPicker.page, 2, "speaker picker next-page control did not advance")
speakerPicker.searchBox:SetText("Speaker 1")
speakerPicker.searchBox._scripts.OnTextChanged(speakerPicker.searchBox, true)
assert_true(#speakerPicker.filteredOptions >= 1, "speaker picker search did not find the expected participant")
assert_equal(speakerPicker.page, 1, "speaker picker search should reset pagination")
click(speakerPicker.cancelBtn)
assert_true(not speakerPicker:IsShown(), "speaker picker cancel should close without applying")
assert_contains(frame.speakerDropdown:GetText(), "全部发言者", "cancelled speaker draft should not change the active filter")

click(frame.copyBtn)
local latestCopiedText = frame.copyDialog.editBox:GetText()
assert_contains(latestCopiedText, "[炉石]", "copy should include rendered item label")
assert_contains(latestCopiedText, "[冲锋]", "copy should include rendered spell label")
assert_contains(latestCopiedText, "[海兽之血宝石]", "copy should include rendered TRP3 label")
assert_not_contains(latestCopiedText, "|Hitem", "copy leaked native item source")
assert_not_contains(latestCopiedText, "|Hspell", "copy leaked native spell source")
assert_not_contains(latestCopiedText, "[TRP3:海兽之血宝石:1]", "copy leaked TRP3 source")
click(frame.copyBtn)

-- Exact time range filtering uses inclusive start/end timestamps while the
-- rendered page remains newest-first.
assert_true(frame.exactStartBox ~= nil and frame.exactEndBox ~= nil, "exact time filter boxes were not created")
press_enter(frame.exactStartBox, date("%Y-%m-%d %H:%M:%S", BASE_TIMESTAMP + 3190))
Timer.drain()
press_enter(frame.exactEndBox, date("%Y-%m-%d %H:%M:%S", BASE_TIMESTAMP + TOTAL_INITIAL_RECORDS))
Timer.drain()
assert_equal(frame.logState.totalMatched, 16, "exact time range should include only the bounded records")
assert_equal(frame.dateDropdown:GetText(), "精确区间", "date dropdown should show exact range mode")
assert_contains(frame.filterSummary:GetText(), "精确", "exact time filter summary missing")
assert_contains(frame.logContent.rows[1].text:GetText(), "[炉石]", "exact time range should keep newest record at top")
assert_contains(frame.logContent.rows[frame.logState.displayCount].text:GetText(), "record-03190", "exact time range should include the oldest bounded record")
click(frame.clearFilterBtn)
Timer.drain()
assert_equal(frame.logState.totalMatched, DEFAULT_VISIBLE_RECORDS, "clearing filters should preserve the default system-message visibility")
assert_equal(frame.exactStartBox:GetText(), "", "clearing filters should reset exact start time")
assert_equal(frame.exactEndBox:GetText(), "", "clearing filters should reset exact end time")

-- Page buttons must traverse the entire archive and clamp at both ends.
click(frame.nextPageBtn)
Timer.drain()
assert_equal(frame.logState.page, 2, "next-page button should move to the next page")
click(frame.prevPageBtn)
Timer.drain()
assert_equal(frame.logState.page, 1, "previous-page button should move to the previous page")
assert_equal(click(frame.prevPageBtn), false, "previous-page button should be disabled on the first page")
assert_equal(click(frame.latestPageBtn), false, "first-page button should be disabled on the first page")
assert_equal(frame.logState.page, 1, "first-page boundary should not underflow")
click(frame.nextPageBtn)
Timer.drain()
assert_equal(frame.logState.page, 2, "next-page navigation setup")
click(frame.latestPageBtn)
Timer.drain()
assert_equal(frame.logState.page, 1, "first-page button should return to page one")
while frame.logState.page < frame.logState.totalPages do
    click(frame.nextPageBtn)
    Timer.drain()
end
assert_equal(click(frame.nextPageBtn), false, "next-page button should be disabled on the last page")
assert_contains(frame.logContent.rows[frame.logState.displayCount].text:GetText(), "legacy-marker", "oldest record was not reachable")
assert_true(frame.statusText:IsShown(), "log status should be visible on the log tab")
assert_true(frame.refreshBtn:IsShown(), "log refresh should be visible on the log tab")
assert_true(frame.clearBtn:IsShown() and frame.exportBtn:IsShown() and frame.copyBtn:IsShown(), "log actions should be visible on the log tab")

-- Drive the production settings UI: values are clamped to 40..120 and only
-- the bounded row pool grows, never the full archive.
click(find_tab(frame, "settings"))
local settingsContent = frame.settingsContent
assert_true(not frame.statusText:IsShown(), "settings tab leaked the log status")
assert_true(not frame.refreshBtn:IsShown(), "settings tab leaked the log refresh action")
assert_true(not frame.clearBtn:IsShown() and not frame.exportBtn:IsShown() and not frame.copyBtn:IsShown(), "settings tab leaked log-only actions")
assert_true(settingsContent.themeClassicBtn ~= nil and settingsContent.themeModernBtn ~= nil,
    "theme controls were not created")
assert_true(settingsContent.ignoreNonRPCb ~= nil, "non-RP player filter setting was not created")
assert_true(settingsContent.itemGuardCb ~= nil, "item-guard setting was not created")
assert_true(settingsContent.itemGuardCb:IsShown(), "item-guard setting is hidden in the real settings layout")
assert_equal(settingsContent.itemGuardCb.text:GetText(), "开启 TRP3 对象与光环防护",
    "object-and-aura guard setting uses stale wording")
assert_true(settingsContent.itemGuardTopOffset <= 60, "item-guard setting is not in the visible top feature area")
assert_equal(frame.settingsScroll:GetVerticalScroll(), 0, "settings tab did not return to its visible top area")
assert_true(settingsContent.itemGuardCb:GetChecked() == true, "item guard should default to enabled")
settingsContent.itemGuardCb:SetChecked(false)
click(settingsContent.itemGuardCb)
assert_true(RPBox_Config.itemGuardEnabled == false, "item-guard setting was not persisted")
assert_equal(itemGuardToggles[#itemGuardToggles], false, "item-guard module was not disabled")
settingsContent.itemGuardCb:SetChecked(true)
click(settingsContent.itemGuardCb)
assert_true(RPBox_Config.itemGuardEnabled == true, "item guard could not be re-enabled")
assert_equal(itemGuardToggles[#itemGuardToggles], true, "item-guard module was not enabled")

assert_equal(#settingsContent.checkboxes, 8, "channel grid should contain all eight channel toggles")
assert_equal(settingsContent.checkboxes[1].gridColumn, 1, "channel grid first item column is wrong")
assert_equal(settingsContent.checkboxes[1].gridRow, 1, "channel grid first item row is wrong")
assert_equal(settingsContent.checkboxes[4].gridColumn, 4, "channel grid fourth item should end row one")
assert_equal(settingsContent.checkboxes[5].gridColumn, 1, "channel grid fifth item should start row two")
assert_equal(settingsContent.checkboxes[8].gridColumn, 4, "channel grid last item should end row two")
assert_contains(settingsContent.checkboxes[1].text:GetText(), "|cffFF4080说话|r", "SAY did not use its native ChatTypeInfo color")
assert_equal(settingsContent.checkboxes[6].nativeChatType, "WHISPER", "incoming whisper used the wrong native chat type")
assert_equal(settingsContent.checkboxes[7].nativeChatType, "WHISPER_INFORM", "outgoing whisper used the wrong native chat type")
assert_true(settingsContent.checkboxes[4]._point[2] < settingsContent:GetWidth(), "four-column channel grid overflows settings content")
frame:SetSize(680, 540)
frame._scripts.OnSizeChanged(frame, 680, 540)
assert_equal(settingsContent:GetWidth(), 616, "settings content did not adapt to minimum window width")
assert_true(
    settingsContent.checkboxes[4]._point[2] + settingsContent.channelColumnWidth <= settingsContent:GetWidth(),
    "four-column channel grid overflows at minimum window width"
)
frame:SetSize(780, 560)
frame._scripts.OnSizeChanged(frame, 780, 560)

click(find_tab(frame, "guard"))
assert_true(not frame.statusText:IsShown(), "guard tab leaked the log status")
assert_true(not frame.refreshBtn:IsShown(), "guard tab leaked the log refresh action")
assert_true(not frame.clearBtn:IsShown() and not frame.exportBtn:IsShown() and not frame.copyBtn:IsShown(), "guard tab leaked log-only actions")
assert_true(frame.guardContent ~= nil and frame.guardContent.rows ~= nil, "risk-management GUI was not created")
local guardContent = frame.guardContent
assert_equal(#guardContent.rows, 20, "risk GUI should create only one bounded page of rows")
assert_equal(guardContent.totalEntries, 47, "risk GUI did not read all fixture entries")
assert_equal(guardContent.totalMatches, 47, "default risk filter should show all entries")
assert_equal(guardContent.totalPages, 3, "risk GUI pagination count is incorrect")
assert_contains(guardContent.pageInfo:GetText(), "匹配 47 / 总计 47", "risk pagination summary is incomplete")
click(guardContent.scanBtn)
assert_equal(itemGuardActions[#itemGuardActions].action, "scan", "manual risk scan was not delegated")
local guardRow = frame.guardContent.rows[1]
assert_true(guardRow ~= nil and guardRow:IsShown(), "risk item row was not rendered")
assert_equal(guardRow.name:GetText(), "风险道具", "risk item name was not rendered")
assert_contains(guardRow.reason:GetText(), "步骤循环", "risk reason was not rendered")
assert_contains(guardRow.idText:GetText(), "风险分 140", "risk score was not rendered")
assert_equal(guardRow.status:GetText(), "已隔离", "risk status was not rendered")
assert_true(guardRow.quarantineCb:GetChecked(), "isolation checkbox did not reflect current state")
assert_equal(guardRow.ignoreBtn:GetText(), "加入忽略", "ignore-list action was not shown")
assert_equal(guardRow.ignoreBtn:GetAlpha(), 0.48, "ignore-list action was not de-emphasized")

click(find_tab(frame, "log"))
Timer.drain()
assert_true(frame.statusText:IsShown(), "returning to log did not restore its status")
assert_true(frame.refreshBtn:IsShown(), "returning to log did not restore refresh")
assert_true(frame.clearBtn:IsShown() and frame.exportBtn:IsShown() and frame.copyBtn:IsShown(), "returning to log did not restore log actions")
click(find_tab(frame, "guard"))

-- The minimum supported window keeps the text rail and the fixed action rail
-- disjoint, and the row itself inside the guard content width.
frame:SetSize(680, 540)
frame._scripts.OnSizeChanged(frame, 680, 540)
assert_equal(guardContent:GetWidth(), 616, "risk content did not adapt to minimum window width")
assert_equal(guardRow:GetWidth(), 600, "risk row did not stay inside minimum-width content")
assert_true(guardRow._rpboxTextRight < guardRow._rpboxActionLeft, "risk text overlaps the action rail")
assert_equal(guardRow.ignoreBtn:GetWidth(), 96, "ignore action width should remain stable")
assert_equal(guardRow.ignoreBtn._point[1], "BOTTOMRIGHT", "ignore action lost its right-edge anchor")
frame:SetSize(780, 560)
frame._scripts.OnSizeChanged(frame, 780, 560)

-- Search covers names, root IDs, and reasons; every filter change returns to
-- page one without growing the row pool.
click(guardContent.nextPageBtn)
frame.guardScroll:SetVerticalScroll(120)
guardContent.searchBox:SetText("risk-037")
guardContent.searchBox._scripts.OnTextChanged(guardContent.searchBox, true)
assert_equal(guardContent.page, 1, "risk ID search did not reset pagination")
assert_equal(frame.guardScroll:GetVerticalScroll(), 0, "risk search did not return the list to the top")
assert_equal(guardContent.totalMatches, 1, "risk ID search returned the wrong number of entries")
assert_equal(guardContent.rows[1].entry.rootID, "risk-037", "risk ID search rendered the wrong entry")
guardContent.searchBox:SetText("海妖特殊原因")
guardContent.searchBox._scripts.OnTextChanged(guardContent.searchBox, true)
assert_equal(guardContent.totalMatches, 1, "risk reason search did not match reason text")
assert_equal(guardContent.rows[1].entry.rootID, "risk-037", "risk reason search rendered the wrong entry")
guardContent.searchBox:SetText("批量风险道具 07")
guardContent.searchBox._scripts.OnTextChanged(guardContent.searchBox, true)
assert_equal(guardContent.totalMatches, 1, "risk name search did not match item name")
assert_equal(guardContent.rows[1].entry.rootID, "risk-007", "risk name search rendered the wrong entry")
guardContent.searchBox:SetText("")
guardContent.searchBox._scripts.OnTextChanged(guardContent.searchBox, true)

click(guardContent.nextPageBtn)
click(guardContent.statusButtons.ignored)
assert_equal(guardContent.statusFilter, "ignored", "ignored-state filter was not selected")
assert_equal(guardContent.page, 1, "status filter did not reset pagination")
assert_true(guardContent.totalMatches > 0, "ignored-state fixture should not be empty")
for _, row in ipairs(guardContent.rows) do
    if row:IsShown() then
        assert_equal(row.entry.status, "ignored", "status filter rendered a non-ignored entry")
    end
end
click(guardContent.statusButtons.all)
assert_equal(guardContent.totalMatches, 47, "all-state filter did not restore all entries")
click(guardContent.nextPageBtn)
assert_equal(guardContent.page, 2, "risk next-page button did not advance")
assert_equal(guardContent.rows[1].entry.rootID, "risk-021", "risk second page starts at the wrong entry")
assert_true(#guardContent.rows <= 20, "risk pagination grew the bounded row pool")
click(guardContent.prevPageBtn)
assert_equal(guardContent.page, 1, "risk previous-page button did not return to page one")

guardRow = guardContent.rows[1]
guardRow.quarantineCb:SetChecked(false)
click(guardRow.quarantineCb)
assert_equal(itemGuardActions[#itemGuardActions].action, "isolate", "isolation state was not delegated")
assert_equal(itemGuardActions[#itemGuardActions].value, false, "temporary release state was not delegated")
click(guardRow.ignoreBtn)
assert_equal(itemGuardActions[#itemGuardActions].action, "requestIgnore", "ignore action did not request confirmation")
assert_true(itemGuardChangeCallback ~= nil, "risk GUI did not subscribe to guard updates")

-- The source blacklist is a bounded, collapsible manager inside the guard
-- page. Built-in entries are visible and read-only; user entries delegate all
-- mutations to ItemGuardBlacklist without changing risk pagination.
assert_true(itemGuardBlacklistChangeCallback ~= nil, "blacklist GUI did not subscribe to source-list updates")
assert_contains(guardContent.blacklistToggleBtn:GetText(), "来源黑名单 (4)", "source blacklist count is missing")
assert_contains(guardContent.blacklistToggleBtn:GetText(), "展开", "source blacklist toggle should use readable text")
assert_true(not guardContent.blacklistToggleBtn:GetText():find("▾", 1, true),
    "source blacklist toggle retained an unsupported glyph")
click(guardContent.blacklistToggleBtn)
assert_true(guardContent.blacklistPanel:IsShown(), "source blacklist panel did not expand")
assert_contains(guardContent.blacklistToggleBtn:GetText(), "收起", "expanded source blacklist lacks readable state text")
assert_equal(#guardContent.blacklistRows, 5, "source blacklist row pool must stay bounded")
assert_equal(guardContent.blacklistRows[1].source:GetText(), "系统", "built-in source label is missing")
assert_true(not guardContent.blacklistRows[1].removeBtn:IsShown(), "built-in source exposed a delete action")

guardContent.blacklistInput:SetText("NewBad-SmokeRealm")
guardContent.blacklistInput._scripts.OnEnterPressed(guardContent.blacklistInput)
assert_equal(itemGuardBlacklistActions[#itemGuardBlacklistActions].action, "add", "blacklist add was not delegated")
assert_equal(itemGuardBlacklistActions[#itemGuardBlacklistActions].identity, "newbad-smokerealm", "blacklist add identity changed")
assert_contains(guardContent.blacklistToggleBtn:GetText(), "来源黑名单 (5)", "blacklist GUI did not refresh after add")
assert_true(guardContent.blacklistInput._focused == false, "blacklist Enter submission did not clear focus")

local removableBlacklistRow
for _, row in ipairs(guardContent.blacklistRows) do
    if row.entry and row.entry.identity == "newbad-smokerealm" then removableBlacklistRow = row end
end
assert_true(removableBlacklistRow ~= nil, "new user blacklist entry was not rendered")
assert_equal(removableBlacklistRow.source:GetText(), "用户", "user source label is missing")
assert_true(removableBlacklistRow.removeBtn:IsShown(), "user source did not expose delete")
click(removableBlacklistRow.removeBtn)
assert_equal(itemGuardBlacklistActions[#itemGuardBlacklistActions].action, "remove", "blacklist remove was not delegated")
assert_equal(itemGuardBlacklistActions[#itemGuardBlacklistActions].identity, "newbad-smokerealm", "blacklist remove targeted wrong identity")
assert_equal(guardContent.totalEntries, 47, "blacklist mutations changed risk result count")
assert_equal(guardContent.totalPages, 3, "blacklist mutations broke risk pagination")
click(guardContent.blacklistToggleBtn)
assert_true(not guardContent.blacklistPanel:IsShown(), "source blacklist panel did not collapse")

click(find_tab(frame, "settings"))
assert_true(settingsContent.ignoreNonRPCb:GetChecked() == false, "non-RP player filter should default to off")
settingsContent.ignoreNonRPCb:SetChecked(true)
click(settingsContent.ignoreNonRPCb)
assert_true(RPBox_Config.ignoreNonRPPlayers == true, "non-RP player filter setting was not persisted")
settingsContent.ignoreNonRPCb:SetChecked(false)
click(settingsContent.ignoreNonRPCb)
assert_true(RPBox_Config.ignoreNonRPPlayers == false, "non-RP player filter setting could not be disabled")
assert_true(settingsContent.showSystemMessagesCb ~= nil, "system-message visibility setting was not created")
assert_true(settingsContent.showSystemMessagesCb:GetChecked() == false, "system messages should default to hidden")
settingsContent.showSystemMessagesCb:SetChecked(true)
click(settingsContent.showSystemMessagesCb)
assert_true(RPBox_Config.showSystemMessages == true, "system-message visibility setting was not persisted")
assert_true(settingsContent.themeModernBtn:IsEnabled() == false, "active modern theme should be selected")
RPBox_Config.uiTheme = "removed-theme"
assert_equal(ns.UI.GetTheme(), "modern", "removed theme should fall back to modern")
assert_equal(RPBox_Config.uiTheme, "modern", "removed theme fallback was not persisted")
ns.UI.ApplyAll()
click(settingsContent.themeClassicBtn)
assert_equal(RPBox_Config.uiTheme, "classic", "classic theme selection was not persisted")
assert_equal(frame._rpboxTheme, "classic", "classic theme was not applied immediately")
assert_true(frame.TitleText:IsShown(), "classic theme should restore native title chrome")
assert_true(not frame._rpboxModernChrome:IsShown(), "classic theme should hide modern title chrome")
assert_true(frame.contentSurface:IsShown() == false, "classic theme should hide the modern content surface")
assert_equal(frame.dateDropdown.Button._nativeTexture:GetAlpha(), 1, "classic dropdown should restore native arrow artwork")
assert_true(not frame.logScroll._rpboxScrollTrack:IsShown(), "classic theme should hide the modern scroll track")
assert_equal(frame.logScroll.ScrollBar.ScrollUpButton._nativeTexture:GetAlpha(), 1, "classic scrollbar should restore native artwork")
click(settingsContent.themeModernBtn)
assert_equal(RPBox_Config.uiTheme, "modern", "modern theme selection was not persisted")
assert_equal(frame._rpboxTheme, "modern", "modern theme was not restored immediately")
assert_true(frame._rpboxModernChrome:IsShown(), "modern title chrome should be restored")
assert_true(settingsContent.themeModernBtn:IsEnabled() == false, "restored modern theme should be selected")
assert_equal(frame.dateDropdown.Button._nativeTexture:GetAlpha(), 0, "restored modern dropdown should hide native arrow artwork")
assert_true(frame.logScroll._rpboxScrollTrack:IsShown(), "restored modern theme should show the modern scroll track")
local pageSizeBox = frame.settingsContent.viewWindowSizeBox
assert_true(pageSizeBox ~= nil, "playback page-size setting was not created")
press_enter(pageSizeBox, "1")
assert_equal(RPBox_Config.logViewWindowSize, 40, "page size lower clamp")
assert_equal(pageSizeBox:GetText(), "40", "lower clamp should be reflected in the edit box")
click(find_tab(frame, "log"))
Timer.drain()
assert_equal(frame.logState.pageSize, 40, "lower-clamped page size was not applied")
assert_equal(frame.logState.totalMatched, TOTAL_INITIAL_RECORDS, "enabling system messages should restore timeline nodes")
assert_equal(frame.logShownRowCount, 40, "lower-clamped page should render 40 rows")

click(find_tab(frame, "settings"))
press_enter(frame.settingsContent.viewWindowSizeBox, "999")
assert_equal(RPBox_Config.logViewWindowSize, 120, "page size upper clamp")
assert_equal(frame.settingsContent.viewWindowSizeBox:GetText(), "120", "upper clamp should be reflected in the edit box")
click(find_tab(frame, "log"))
Timer.drain()
assert_equal(frame.logState.pageSize, 120, "upper-clamped page size was not applied")
assert_equal(frame.logShownRowCount, 120, "upper-clamped page should render 120 rows")
assert_true(#frame.logContent.rows <= 120, "the live row pool exceeded its hard limit")

-- Copy uses the current page slice, not the thousands of matched records.
click(frame.nextPageBtn)
Timer.drain()
local copiedStart = frame.logState.startIndex
local copiedEnd = frame.logState.endIndex
assert_equal(copiedStart, 121, "copy test should be on page two")
click(frame.copyBtn)
local copiedText = frame.copyDialog.editBox:GetText()
assert_equal(count_text_lines(copiedText), copiedEnd - copiedStart + 1, "copy should contain exactly the current page")
local copiedChronoStart = frame.logState.totalMatched - copiedEnd + 1
local copiedChronoEnd = frame.logState.totalMatched - copiedStart + 1
local copiedOldestMarker = string.format("record-%05d", copiedChronoStart)
local copiedNewestMarker = string.format("record-%05d", copiedChronoEnd)
assert_contains(copiedText, copiedOldestMarker, "copy omitted the oldest record on the current displayed page")
assert_contains(copiedText, copiedNewestMarker, "copy omitted the newest record on the current displayed page")
assert_true(copiedText:find(copiedOldestMarker, 1, true) < copiedText:find(copiedNewestMarker, 1, true), "copy should keep the newest displayed record at the bottom")
assert_not_contains(copiedText, string.format("record-%05d", copiedChronoStart - 1), "copy leaked an older record from the next page")
assert_not_contains(copiedText, string.format("record-%05d", copiedChronoEnd + 1), "copy leaked a newer record from the previous page")
click(frame.copyBtn)

-- Old embedded identities and v2 profile-switch nodes must both survive the
-- production filter/renderer path.
ns.OpenMainFrame({ reset = true, search = "历史别名" })
Timer.drain()
assert_equal(frame.logState.totalMatched, 1, "legacy embedded identity should be searchable")
assert_contains(frame.logContent.rows[1].text:GetText(), "历史别名", "legacy identity should render without a live TRP3 lookup")

ns.OpenMainFrame({ reset = true, speakers = { "p:profile-old" } })
Timer.drain()
assert_true(frame.logState.totalMatched >= 1, "profile-switch endpoint should be selectable as a speaker")
local foundSwitch = false
for _, record in ipairs(frame.logState.records) do
    if record.mk == "S" and record.ev and record.ev.kind == "profile_switch" then foundSwitch = true end
end
assert_true(foundSwitch, "profile-switch event disappeared after speaker filtering")
assert_contains(frame.logContent.rows[1].text:GetText(), "旧卡名", "profile-switch event did not render its historical before-name")
assert_contains(frame.logContent.rows[1].text:GetText(), "新卡名", "profile-switch event did not render its historical after-name")

-- A burst received while reading history is coalesced into one delayed refresh,
-- and does not pull the reader away from the historical page.
ns.OpenMainFrame({ reset = true })
Timer.drain()
click(frame.nextPageBtn)
Timer.drain()
assert_equal(frame.logState.page, 2, "historical-page live refresh setup")
append_live_records(25, "history-burst")
local completedBeforeHistoryBurst = Timer.completedScans
trigger_new_message(100)
assert_equal(#Timer.queue, 1, "100 rapid messages should schedule one refresh timer")
assert_true(Timer.peek().due - Timer.now >= 1, "live refresh should be throttled for one second")
Timer.drain()
assert_equal(Timer.completedScans - completedBeforeHistoryBurst, 1, "a coalesced burst should cause one archive rescan")
assert_equal(frame.logState.page, 2, "live messages should not jump a reader off a historical page")
assert_equal(frame.logState.totalMatched, nextRecordIndex, "coalesced history burst was not incorporated")

-- On the latest page, new records follow automatically.  Messages arriving in
-- the middle of a scan set a dirty bit; the first scan/render finishes, then a
-- single follow-up refresh consumes the pending data.
click(frame.latestPageBtn)
Timer.drain()
append_live_records(20, "latest-burst")
local completedBeforeLatestBurst = Timer.completedScans
trigger_new_message(80)
assert_equal(#Timer.queue, 1, "latest-page burst should schedule one refresh timer")
assert_true(Timer.peek().due - Timer.now >= 1, "successive live refreshes should be at least one second apart")
Timer.run_one() -- delayed live-refresh callback: starts the production scan
assert_true(frame.logScan ~= nil, "live refresh timer did not start an archive scan")
Timer.run_one() -- first 250-record scan batch
assert_true(frame.logScan ~= nil and frame.logScan.scanned <= 250, "first scan batch was not bounded")
local queuedDuringScan = #Timer.queue
append_live_records(7, "pending-during-scan")
trigger_new_message(50)
assert_true(frame.logLiveRefreshPending == true, "messages during a scan did not leave a pending dirty bit")
assert_equal(#Timer.queue, queuedDuringScan, "messages during a scan scheduled redundant refresh timers")
Timer.drain()
assert_equal(Timer.completedScans - completedBeforeLatestBurst, 2, "pending scan data should settle through one follow-up rescan")
assert_equal(frame.logState.totalMatched, nextRecordIndex, "messages received during scanning were not eventually included")
assert_equal(frame.logState.page, 1, "latest-page playback should continue following the newest page")
assert_equal(frame.logState.records[frame.logState.totalMatched].m, "pending-during-scan-007", "newest pending record was not reachable")
assert_settled(frame, "live playback")

assert_equal(Timer.maxScanDelta, 250, "smoke data should exercise the 250-record scan batch ceiling")
assert_equal(Timer.maxRenderDelta, 20, "smoke data should exercise the 20-row render batch ceiling")
assert_true(Timer.maxRowsCreated <= 120, "created row-frame count exceeded the hard ceiling")

print(string.format(
    "PASS main-frame playback smoke: %d records, max scan/tick=%d, max render/tick=%d, max row frames=%d, timer callbacks=%d",
    frame.logState.totalMatched,
    Timer.maxScanDelta,
    Timer.maxRenderDelta,
    Timer.maxRowsCreated,
    Timer.callbacksRun
))
