-- TRP3 Extended item-source blacklist.
--
-- This module only answers whether an object's recorded identities exactly
-- match a built-in or user-maintained entry.  ItemGuard owns the isolation
-- policy and calls MatchRoot as one of its risk inputs.

local ADDON_NAME, ns = ...

local Blacklist = {}
ns.ItemGuardBlacklist = Blacklist

local BUILTIN_REASON = "RPBox 内置恶意道具来源名单"
local BUILTIN_IDENTITIES = {
    "蕾火演员死冯-金色平原",
    "工作人员二号-金色平原",
    "绿宝石兽-金色平原",
}

local callbacks = {}
local builtinByIdentity = {}
local normalizedDatabase

local function Trim(value)
    return (tostring(value or ""):gsub("^%s+", ""):gsub("%s+$", ""))
end

function Blacklist.NormalizeIdentity(value, colonValue)
    if value == Blacklist then value = colonValue end
    if type(value) ~= "string" then return nil end

    local identity = Trim(value)
    if identity == "" then return nil end

    -- Accept common pasted dash glyphs while storing one canonical form.
    identity = identity:gsub("–", "-"):gsub("—", "-"):gsub("−", "-")
    identity = identity:gsub("%s*%-%s*", "-")
    identity = string.lower(identity)

    local name, realm = identity:match("^([^%-]+)%-(.+)$")
    name, realm = Trim(name), Trim(realm)
    if name == "" or realm == "" then return nil end
    return name .. "-" .. realm
end

for _, identity in ipairs(BUILTIN_IDENTITIES) do
    local normalized = Blacklist.NormalizeIdentity(identity)
    builtinByIdentity[normalized] = {
        identity = normalized,
        source = "builtin",
        reason = BUILTIN_REASON,
    }
end

local function EnsureDatabase()
    RPBox_ItemGuardDB = RPBox_ItemGuardDB or {}
    local current = RPBox_ItemGuardDB.userBlacklist
    if current == normalizedDatabase and type(current) == "table" then return current end
    if type(current) ~= "table" then current = {} end

    local normalizedEntries = {}
    for key, value in pairs(current) do
        local suppliedIdentity = type(value) == "table" and value.identity or key
        local normalized = Blacklist.NormalizeIdentity(suppliedIdentity)
        if normalized and not builtinByIdentity[normalized] then
            local reason
            if type(value) == "table" then reason = Trim(value.reason) end
            if type(value) == "string" then reason = Trim(value) end
            normalizedEntries[normalized] = {
                identity = normalized,
                reason = reason ~= "" and reason or nil,
            }
        end
    end

    RPBox_ItemGuardDB.userBlacklist = normalizedEntries
    normalizedDatabase = normalizedEntries
    return normalizedEntries
end

local function NotifyChanged(identity)
    for _, callback in ipairs(callbacks) do
        pcall(callback, identity)
    end
end

function Blacklist.Initialize()
    EnsureDatabase()
    return true
end

function Blacklist.GetEntries()
    local entries = {}
    for _, entry in pairs(builtinByIdentity) do
        entries[#entries + 1] = {
            identity = entry.identity,
            source = "builtin",
            reason = entry.reason,
        }
    end
    for identity, entry in pairs(EnsureDatabase()) do
        entries[#entries + 1] = {
            identity = identity,
            source = "user",
            reason = entry.reason,
        }
    end
    table.sort(entries, function(left, right)
        if left.source ~= right.source then return left.source < right.source end
        return left.identity < right.identity
    end)
    return entries
end

function Blacklist.AddUser(identity, reason, colonReason)
    if identity == Blacklist then identity, reason = reason, colonReason end
    local normalized = Blacklist.NormalizeIdentity(identity)
    if not normalized then
        return false, "请输入完整的 名字-服务器"
    end
    if builtinByIdentity[normalized] then
        return false, "该身份已在系统内置黑名单中"
    end

    local entries = EnsureDatabase()
    if entries[normalized] then
        return false, "该身份已在用户黑名单中"
    end

    reason = Trim(reason)
    entries[normalized] = {
        identity = normalized,
        reason = reason ~= "" and reason or nil,
    }
    NotifyChanged(normalized)
    return true, "已加入来源黑名单"
end

function Blacklist.RemoveUser(identity, colonIdentity)
    if identity == Blacklist then identity = colonIdentity end
    local normalized = Blacklist.NormalizeIdentity(identity)
    if not normalized then return false, "无效的身份" end
    if builtinByIdentity[normalized] then
        return false, "系统内置黑名单不可删除"
    end

    local entries = EnsureDatabase()
    if not entries[normalized] then
        return false, "用户黑名单中不存在该身份"
    end
    entries[normalized] = nil
    NotifyChanged(normalized)
    return true, "已移出来源黑名单"
end

local function FindEntry(identity)
    local normalized = Blacklist.NormalizeIdentity(identity)
    if not normalized then return nil end
    if builtinByIdentity[normalized] then return builtinByIdentity[normalized], normalized end
    local user = EnsureDatabase()[normalized]
    if user then
        return {
            identity = normalized,
            source = "user",
            reason = user.reason,
        }, normalized
    end
    return nil
end

function Blacklist.MatchRoot(rootID, root, colonRoot)
    if rootID == Blacklist then rootID, root = root, colonRoot end
    if type(root) ~= "table" then return nil end

    -- The transport sender is the strongest source signal.  Creator and last
    -- editor are still checked because locally imported objects may not retain
    -- a sender record.
    local candidates = {
        { field = "TRP3_Security.sender", value = TRP3_Security and TRP3_Security.sender and TRP3_Security.sender[rootID] },
        { field = "MD.CB", value = root.MD and root.MD.CB },
        { field = "MD.SB", value = root.MD and root.MD.SB },
    }
    for _, candidate in ipairs(candidates) do
        local entry, normalized = FindEntry(candidate.value)
        if entry then
            return {
                identity = normalized,
                source = entry.source,
                reason = entry.reason,
                field = candidate.field,
            }
        end
    end
    return nil
end

function Blacklist.RegisterOnChanged(callback, colonCallback)
    if callback == Blacklist then callback = colonCallback end
    if type(callback) ~= "function" then return false end
    callbacks[#callbacks + 1] = callback
    return true
end
