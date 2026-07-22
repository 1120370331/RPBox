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
function ObjectMethods:ClearAllPoints() self._point = nil end
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
function ObjectMethods:SetAutoFocus(value) self._autoFocus = value end
function ObjectMethods:SetNumeric(value) self._numeric = value end
function ObjectMethods:SetMultiLine(value) self._multiLine = value end
function ObjectMethods:SetFontObject(value) self._fontObject = value end
function ObjectMethods:SetJustifyH(value) self._justifyH = value end
function ObjectMethods:SetJustifyV(value) self._justifyV = value end
function ObjectMethods:SetWordWrap(value) self._wordWrap = value end
function ObjectMethods:SetNonSpaceWrap(value) self._nonSpaceWrap = value end
function ObjectMethods:SetText(value) self._text = tostring(value or "") end
function ObjectMethods:GetText() return self._text end
function ObjectMethods:GetStringHeight()
    local _, newlines = tostring(self._text or ""):gsub("\n", "")
    return math.max(16, (newlines + 1) * 16)
end
function ObjectMethods:SetChecked(value) self._checked = not not value end
function ObjectMethods:GetChecked() return self._checked end
function ObjectMethods:SetEnabled(value) self._enabled = not not value end
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
function ObjectMethods:CreateFontString(name, layer, template)
    return new_object("FontString", name, nil, template)
end
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
    if template == "BasicFrameTemplateWithInset" then
        frame.TitleText = frame:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        frame.CloseButton = new_object("Button", nil, frame, nil)
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
local ns = {
    L = {},
    GetProfileSnapshot = function(key) return snapshots[key] end,
    GetCachedProfile = function() return nil end,
    GetPlayerID = function() return "Tester-SmokeRealm" end,
    RemoveFromWhitelist = function() end,
    RemoveFromBlacklist = function() end,
    RegisterOnNewMessage = function(callback) newMessageCallbacks[#newMessageCallbacks + 1] = callback end,
    RegisterOnListChange = function(callback) listChangeCallbacks[#listChangeCallbacks + 1] = callback end,
}

local function trigger_new_message(times)
    for _ = 1, times or 1 do
        for _, callback in ipairs(newMessageCallbacks) do callback() end
    end
end

local TOTAL_INITIAL_RECORDS = 3205
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

local chunk, loadError = loadfile("addon/RPBox_Addon/MainFrame.lua")
if not chunk then fail("could not load production MainFrame.lua: " .. tostring(loadError)) end
local ok, loadRuntimeError = xpcall(function() chunk("RPBox_Addon", ns) end, debug.traceback)
if not ok then fail("production MainFrame.lua failed during load:\n" .. tostring(loadRuntimeError)) end

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

local function assert_settled(frame, message)
    assert_equal(#Timer.queue, 0, (message or "playback") .. " left pending timers")
    assert_true(frame.logScan == nil, (message or "playback") .. " left a scan active")
    assert_true(frame.logState == nil or frame.logState.rendering == false, (message or "playback") .. " left rendering active")
end

-- Load all history through the real asynchronous scan and bounded renderer.
ns.OpenMainFrame()
local frame = _G.RPBoxMainFrame
assert_true(frame ~= nil and frame:IsShown(), "OpenMainFrame did not create and show the production frame")
Timer.drain()
assert_settled(frame, "initial playback")
assert_equal(frame.logState.totalMatched, TOTAL_INITIAL_RECORDS, "the full long archive was not reachable")
assert_equal(frame.logState.pageSize, 80, "default playback page size")
assert_equal(frame.logState.page, 1, "first open should show the earliest page")
assert_equal(frame.logState.startIndex, 1, "first page should start at the oldest record")
assert_equal(frame.logState.endIndex, 80, "default page should contain 80 records")
assert_equal(frame.logShownRowCount, 80, "default page visible row count")
assert_contains(frame.logContent.rows[1].text:GetText(), "legacy-marker", "oldest record was not rendered first")

-- Page buttons must traverse the entire archive and clamp at both ends.
click(frame.nextPageBtn)
Timer.drain()
assert_equal(frame.logState.page, 2, "continue button should move to the next page")
click(frame.prevPageBtn)
Timer.drain()
assert_equal(frame.logState.page, 1, "earlier button should move to the previous page")
assert_equal(click(frame.prevPageBtn), false, "earlier button should be disabled on the first page")
assert_equal(frame.logState.page, 1, "first-page boundary should not underflow")
click(frame.latestPageBtn)
Timer.drain()
assert_equal(frame.logState.page, frame.logState.totalPages, "latest button should reach the newest page")
assert_equal(click(frame.nextPageBtn), false, "continue button should be disabled on the latest page")
assert_equal(frame.logState.endIndex, TOTAL_INITIAL_RECORDS, "latest page should reach the final record")

-- Drive the production settings UI: values are clamped to 40..120 and only
-- the bounded row pool grows, never the full archive.
click(find_tab(frame, "settings"))
local pageSizeBox = frame.settingsContent.viewWindowSizeBox
assert_true(pageSizeBox ~= nil, "playback page-size setting was not created")
press_enter(pageSizeBox, "1")
assert_equal(RPBox_Config.logViewWindowSize, 40, "page size lower clamp")
assert_equal(pageSizeBox:GetText(), "40", "lower clamp should be reflected in the edit box")
click(find_tab(frame, "log"))
Timer.drain()
assert_equal(frame.logState.pageSize, 40, "lower-clamped page size was not applied")
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
assert_contains(copiedText, "record-00121", "copy omitted the first record on the current page")
assert_true(not copiedText:find("legacy-marker", 1, true), "copy leaked a record from the previous page")
assert_true(not copiedText:find("record-00241", 1, true), "copy leaked a record from the next page")
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
assert_equal(frame.logState.page, frame.logState.totalPages, "latest-page playback should continue following the newest page")
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
