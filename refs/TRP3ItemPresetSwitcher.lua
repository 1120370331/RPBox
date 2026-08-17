-- TRP3 Extended item preset switcher experiment.
--
-- Usage:
--   1. Run the trusted _G injection command for the current UI session.
--   2. Add a Lua Script Effect to the item's "On use" workflow.
--   3. Paste this complete script into the effect.
--   4. Each use changes the current TRP3 item class to the next preset.
--
-- Scope:
--   * Only edits an item owned in TRP3_Tools_DB.
--   * Name/icon are class data, so every local instance with this class ID
--     changes together.
--   * The preset is inferred from the current name and icon; no extra state
--     is stored in the inventory slot.

local function notify(message)
    if effect then
        effect("text", args, message, "4")
    end
end

if not args or not args._G then
    notify("\230\156\170\230\163\128\230\181\139\229\136\176 _G \230\179\168\229\133\165")
    return
end

local G = args._G
local API = G.TRP3_API
local Extended = G.TRP3_Extended
local DB = G.TRP3_DB

if not API or not API.extended or not Extended or not DB then
    notify("TRP3 Extended API is not ready")
    return
end

-- Edit these entries to define the cycle. Icon values are WoW icon names
-- without the "Interface\\Icons\\" prefix.
local PRESETS = {
    {
        name = "\230\189\174\230\177\144\228\185\139\233\146\165", -- Tide Key
        icon = "inv_misc_kingsring1",
    },
    {
        name = "\230\181\183\229\166\150\228\185\139\231\156\188", -- Siren's Eye
        icon = "inv_jewelry_ring_36",
    },
    {
        name = "\230\178\137\232\136\185\232\136\170\230\181\183\229\155\190", -- Shipwreck Chart
        icon = "inv_misc_map_01",
    },
}

-- In an item workflow args.object is the inventory instance. Its id points
-- to the registered TRP3 Extended class that owns BA.NA and BA.IC.
local itemID = args.object and args.object.id
if type(itemID) ~= "string" or itemID == "" then
    notify("\229\183\165\228\189\156\230\181\129\230\178\161\230\156\137\230\143\144\228\190\155\229\189\147\229\137\141\233\129\147\229\133\183")
    return
end

local rootID = API.extended.getRootClassID(itemID)
local root = G.TRP3_Tools_DB and G.TRP3_Tools_DB[rootID]
if not root then
    notify("\229\143\170\232\131\189\228\191\174\230\148\185\232\135\170\229\183\177\230\139\165\230\156\137\231\154\132 TRP3 Extended \233\129\147\229\133\183")
    return
end

local class = API.extended.getClass(itemID)
if not class or class.missing then
    notify("\230\137\190\228\184\141\229\136\176\229\189\147\229\137\141\233\129\147\229\133\183\229\137\167\230\156\172")
    return
end

class.BA = class.BA or {}
local currentName = class.BA.NA or ""
local currentIcon = class.BA.IC or ""
local currentIndex = 0

for index, preset in ipairs(PRESETS) do
    if currentName == preset.name and currentIcon == preset.icon then
        currentIndex = index
        break
    end
end

local nextIndex = (currentIndex % #PRESETS) + 1
local nextPreset = PRESETS[nextIndex]

-- Mutate the saved root object through the registered class reference.
class.BA.NA = nextPreset.name
class.BA.IC = nextPreset.icon

-- Mirror the metadata updates performed by TRP3 Extended's own editor.
root.MD = root.MD or {}
root.MD.V = (tonumber(root.MD.V) or 0) + 1
root.MD.SD = G.date("%d/%m/%y %H:%M:%S")
root.MD.SB = API.globals.player_id

-- Rebuild security, registration and compiled workflow caches, then refresh
-- consumers that may already be showing the old name or icon.
API.security.computeSecurity(rootID, root)
API.extended.unregisterObject(rootID)
API.extended.registerObject(rootID, root, 0)
API.script.clearRootCompilation(rootID)

Extended:TriggerEvent(Extended.Events.REFRESH_BAG)
Extended:TriggerEvent(
    Extended.Events.ON_OBJECT_UPDATED,
    rootID,
    DB.types.ITEM
)

notify(
    "\229\189\147\229\137\141\233\129\147\229\133\183 ["
        .. itemID
        .. "] "
        .. (currentName ~= "" and currentName or "?")
        .. " -> \229\183\178\229\136\135\230\141\162\228\184\186 "
        .. nextPreset.name
)
