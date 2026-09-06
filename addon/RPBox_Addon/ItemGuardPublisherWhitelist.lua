-- Trusted publisher policy for TRP3 Extended objects.
--
-- Publisher trust suppresses Lua advisory/combination findings only. It never
-- bypasses crash-size limits, proven infinite execution, direct context or
-- shared-library corruption, blacklist matches, or runtime circuit breakers.

local ADDON_NAME, ns = ...

local Whitelist = {}
ns.ItemGuardPublisherWhitelist = Whitelist

local callbacks = {}
local normalizedDatabase

local function Trim(value)
    return (tostring(value or ""):gsub("^%s+", ""):gsub("%s+$", ""))
end

function Whitelist.NormalizeIdentity(value, colonValue)
    if value == Whitelist then value = colonValue end
    if ns.ItemGuardBlacklist and ns.ItemGuardBlacklist.NormalizeIdentity then
        return ns.ItemGuardBlacklist.NormalizeIdentity(value)
    end
    if type(value) ~= "string" then return nil end
    local identity = Trim(value)
    identity = identity:gsub("–", "-"):gsub("—", "-"):gsub("−", "-")
    identity = identity:gsub("%s*%-%s*", "-")
    identity = string.lower(identity)
    local name, realm = identity:match("^([^%-]+)%-(.+)$")
    name, realm = Trim(name), Trim(realm)
    if name == "" or realm == "" then return nil end
    return name .. "-" .. realm
end

local function EnsureDatabase()
    RPBox_ItemGuardDB = RPBox_ItemGuardDB or {}
    local current = RPBox_ItemGuardDB.publisherWhitelist
    if current == normalizedDatabase and type(current) == "table" then return current end
    if type(current) ~= "table" then current = {} end
    local normalizedEntries = {}
    for key, value in pairs(current) do
        local suppliedIdentity = type(value) == "table" and value.identity or key
        local normalized = Whitelist.NormalizeIdentity(suppliedIdentity)
        if normalized then
            local reason = type(value) == "table" and Trim(value.reason)
                or type(value) == "string" and Trim(value)
                or ""
            normalizedEntries[normalized] = {
                identity = normalized,
                reason = reason ~= "" and reason or nil,
            }
        end
    end
    RPBox_ItemGuardDB.publisherWhitelist = normalizedEntries
    normalizedDatabase = normalizedEntries
    return normalizedEntries
end

local function NotifyChanged(identity)
    for _, callback in ipairs(callbacks) do pcall(callback, identity) end
end

local function FindEntry(identity)
    local normalized = Whitelist.NormalizeIdentity(identity)
    if not normalized then return nil end
    local user = EnsureDatabase()[normalized]
    if user then
        return {
            identity = normalized,
            source = "user",
            reason = user.reason,
        }
    end
    return nil
end

function Whitelist.Initialize()
    EnsureDatabase()
    return true
end

function Whitelist.GetEntries()
    local entries = {}
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

function Whitelist.AddUser(identity, reason, colonReason)
    if identity == Whitelist then identity, reason = reason, colonReason end
    local normalized = Whitelist.NormalizeIdentity(identity)
    if not normalized then return false, "请输入完整的 名字-服务器" end
    local entries = EnsureDatabase()
    if entries[normalized] then return false, "该身份已在发布者白名单中" end
    reason = Trim(reason)
    entries[normalized] = {
        identity = normalized,
        reason = reason ~= "" and reason or nil,
    }
    NotifyChanged(normalized)
    return true, "已信任该内容发布者"
end

function Whitelist.RemoveUser(identity, colonIdentity)
    if identity == Whitelist then identity = colonIdentity end
    local normalized = Whitelist.NormalizeIdentity(identity)
    if not normalized then return false, "无效的身份" end
    local entries = EnsureDatabase()
    if not entries[normalized] then return false, "发布者白名单中不存在该身份" end
    entries[normalized] = nil
    NotifyChanged(normalized)
    return true, "已取消信任该内容发布者"
end

function Whitelist.ResolveRootPublisher(rootID, root, colonRoot)
    if rootID == Whitelist then rootID, root = root, colonRoot end
    if type(root) ~= "table" then return nil end

    if TRP3_DB and type(TRP3_DB.my) == "table" and TRP3_DB.my[rootID] == root then
        return {
            identity = Whitelist.NormalizeIdentity(
                TRP3_API and TRP3_API.globals and TRP3_API.globals.player_id
            ) or "self",
            source = "self",
            reason = "TRP3 本地作者数据库确认属于当前玩家",
            field = "TRP3_DB.my",
        }
    end

    local sender = TRP3_Security and TRP3_Security.sender and TRP3_Security.sender[rootID]
    if Whitelist.NormalizeIdentity(sender) then
        return {
            identity = Whitelist.NormalizeIdentity(sender),
            source = "transport",
            field = "TRP3_Security.sender",
        }
    end

    -- Locally imported data may have no sender. The last editor is stronger
    -- than the original creator and prevents a trusted CB value from hiding an
    -- untrusted later modification.
    local editor = root.MD and root.MD.SB
    local creator = root.MD and root.MD.CB
    local candidate, field = editor or creator, editor and "MD.SB" or "MD.CB"
    local normalized = Whitelist.NormalizeIdentity(candidate)
    if not normalized then return nil end
    return {
        identity = normalized,
        source = "metadata",
        field = field,
    }
end

function Whitelist.MatchRoot(rootID, root, colonRoot)
    if rootID == Whitelist then rootID, root = root, colonRoot end
    -- RPBox extends TRP3's policy rather than silently discarding the user's
    -- existing per-object, sender or effect-group permissions.
    local security = TRP3_API and TRP3_API.security
    if security and type(security.resolveEffectSecurity) == "function" then
        local ok, trusted, reason = pcall(security.resolveEffectSecurity, rootID, "script")
        if ok and trusted == true then
            return { identity = "TRP3", source = "trp3", field = "TRP3.security",
                reason = "TRP3 已允许此对象执行 Lua（来源 " .. tostring(reason or "user") .. "）" }
        end
    end
    local publisher = Whitelist.ResolveRootPublisher(rootID, root)
    if not publisher then return nil end
    if publisher.source == "self" then return publisher end
    -- Metadata alone grants nothing. An explicit user-created publisher entry
    -- can suppress compatibility policy; runtime and hard behavior checks stay.
    local entry = FindEntry(publisher.identity)
    if not entry then return nil end
    return {
        identity = entry.identity,
        source = entry.source,
        reason = entry.reason,
        field = publisher.field,
    }
end

function Whitelist.RegisterOnChanged(callback, colonCallback)
    if callback == Whitelist then callback = colonCallback end
    if type(callback) ~= "function" then return false end
    callbacks[#callbacks + 1] = callback
    return true
end
