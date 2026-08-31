-- Smoke test for RPBox_Addon/ItemGuardContentRules.lua.
-- Run from repository root:
--   npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_content_rules_smoke.lua

local function fail(message)
    error("[item-guard-content-rules-smoke] " .. tostring(message), 2)
end

local function assert_true(value, message)
    if not value then fail(message or "expected truthy value") end
end

local function assert_false(value, message)
    if value then fail(message or "expected falsy value") end
end

local function assert_equal(actual, expected, message)
    if actual ~= expected then
        fail((message or "values differ") .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual))
    end
end

TRP3_DB = { types = { ITEM = "IT", DOCUMENT = "DO" } }

local namespace = {}
local chunk = assert(loadfile("addon/RPBox_Addon/ItemGuardContentRules.lua"))
chunk("RPBox_Addon", namespace)
local Rules = namespace.ItemGuardContentRules
assert_true(Rules and Rules.Analyze and Rules.AnalyzeVariables, "content rules were not exported")

local function document(texts)
    local pages = {}
    for index, text in ipairs(texts) do pages[index] = { TX = text } end
    return { TY = "DO", BA = { NA = "Document" }, PA = pages }
end

local safe = {
    TY = "IT",
    BA = { NA = "Safe document" },
    IN = { doc = document({ string.rep("a", 128 * 1024), string.rep("b", 128 * 1024) }) },
}
local safeResult = Rules.Analyze("safe", safe, {})
assert_false(safeResult.blocked, "ordinary document content was blocked")
assert_equal(safeResult.metrics.documentBytes, 256 * 1024, "safe document bytes were counted incorrectly")

local hugePage = {
    TY = "IT",
    BA = { NA = "Huge page" },
    IN = { doc = document({ string.rep("x", Rules.LIMITS.DOCUMENT_PAGE_BYTES + 1) }) },
}
local hugePageResult = Rules.Analyze("huge_page", hugePage, {})
assert_true(hugePageResult.blocked, "crash-sized document page was not blocked")
assert_equal(hugePageResult.findings[1].kind, "document_page_crash_size",
    "document page used the wrong finding kind")

local totalPages = {}
for index = 1, 5 do totalPages[index] = string.rep(tostring(index), 450 * 1024) end
local hugeTotalResult = Rules.Analyze("huge_total", {
    TY = "IT",
    BA = { NA = "Huge total" },
    IN = { doc = document(totalPages) },
}, {})
assert_true(hugeTotalResult.blocked, "cumulative crash-sized document was not blocked")
assert_true(hugeTotalResult.metrics.largestDocumentPageBytes < Rules.LIMITS.DOCUMENT_PAGE_BYTES,
    "total-size fixture accidentally exceeded the single-page limit")

local manyPages = {}
for index = 1, Rules.LIMITS.DOCUMENT_PAGES + 1 do manyPages[index] = { TX = "x" } end
local manyPagesResult = Rules.Analyze("many_pages", {
    TY = "DO",
    BA = { NA = "Many pages" },
    PA = manyPages,
}, {})
assert_true(manyPagesResult.blocked, "excessive document page count was not blocked")

local hugeVariable = Rules.AnalyzeVariables({
    payload = string.rep("v", Rules.LIMITS.VARIABLE_VALUE_BYTES + 1),
}, { rootID = "huge_variable" })
assert_true(hugeVariable.blocked, "crash-sized variable value was not blocked")
assert_equal(hugeVariable.findings[1].kind, "variable_value_crash_size",
    "variable value used the wrong finding kind")

local renderedExpansion = Rules.AnalyzeRenderedDocument(
    document({ "${payload}${payload}${payload}" }),
    { payload = string.rep("r", 200 * 1024) },
    { rootID = "rendered_expansion" }
)
assert_true(renderedExpansion.blocked, "variable-expanded crash-sized document was not blocked")
assert_equal(renderedExpansion.findings[1].kind, "document_rendered_page_crash_size",
    "rendered document used the wrong finding kind")

local structured = {}
for index = 1, 2200 do structured["key" .. index] = index end
local structuredResult = Rules.AnalyzeVariables({ payload = structured }, { rootID = "structured" })
assert_true(structuredResult.blocked, "excessive variable structure was not blocked")
assert_true(structuredResult.findings[1].kind == "variable_nodes_crash_size"
    or structuredResult.findings[1].kind == "variable_total_nodes_crash_size",
    "structured variable used the wrong finding kind")

local cyclic = {}
cyclic.self = cyclic
local cyclicResult = Rules.AnalyzeVariables({ payload = cyclic }, { rootID = "cyclic" })
assert_true(cyclicResult.blocked, "cyclic variable structure was not blocked")
assert_equal(cyclicResult.findings[1].kind, "variable_cycle_crash_size",
    "cyclic variable used the wrong finding kind")

local manyVariables = {}
for index = 1, Rules.LIMITS.VARIABLE_ENTRIES + 1 do manyVariables["v" .. index] = index end
local manyVariablesResult = Rules.AnalyzeVariables(manyVariables, { rootID = "many_variables" })
assert_true(manyVariablesResult.blocked, "excessive variable entry count was not blocked")
assert_equal(manyVariablesResult.findings[1].kind, "variable_entries_crash_size",
    "variable entry count used the wrong finding kind")

local safeVariables = Rules.AnalyzeVariables({ name = "value", nested = { count = 2 } })
assert_false(safeVariables.blocked, "ordinary variables were blocked")

print("item_guard_content_rules_smoke: PASS")
