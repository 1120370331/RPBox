# Addon playback smoke test

`main_frame_playback_smoke.lua` loads the real `Theme.lua` and `MainFrame.lua` files in a small mocked WoW UI runtime. It drives `ns.OpenMainFrame`, theme switching, navigation/copy/settings button scripts, and the real `C_Timer.After` scan/render queue.

Run it from the repository root:

```powershell
npx --yes --package=fengari-node-cli fengari addon/tests/main_frame_playback_smoke.lua
```

The fixture contains more than 3,000 chronological records, a legacy embedded identity, and a v2 profile-switch event. The assertions cover modern/classic theme persistence and immediate application, removed-theme fallback, compact sidebar bounds, the date-time picker, searchable/avatar-backed/paginated speaker multi-selection, system messages hidden by default and restored by the setting, bounded scanning and rendering, the 40–120 page-size clamp, complete page navigation, current-page-only copy, profile/name filtering, and throttled live updates both on historical and latest pages.

`chat_logger_filter_smoke.lua` verifies the optional non-RP player filter: it is off when unset or explicitly disabled, can be enabled to require a detected RP profile, and preserves white/black list precedence.

```powershell
npx --yes --package=fengari-node-cli fengari addon/tests/chat_logger_filter_smoke.lua
```

`item_guard_smoke.lua` loads the real guard and all rule modules against a mocked TRP3 Extended runtime. It verifies default-on startup, default-deny unscanned receipt with automatic release after a clean scan, safe rapid replacement before scan completion, Lua hard/policy/advisory separation, user publisher trust with non-bypassable invariants, quarantine-popup author trust and policy release, `runLuaScriptEffect` source/depth/rate enforcement, shared-library and hook restoration, pure-recursion scoring without isolation, recursion-plus-`item_add` isolation, excessive and runtime `item_add` protection, crash-sized document rendering and object/container/aura/runtime variable blocking, suspicious aura event handlers, static/immediate/delayed cancel-time aura self-reapplication blocking, repeated sound suppression/quarantine, sound-stop quota reset, persistent-variable quotas, destruction-time self-respawn blocking, recursive internal-object isolation, numbered popup reasons, combined current-slot and root-carrier removal through TRP3's object API without destruction callbacks, temporary release followed by re-scan quarantine, explicit ignore-ledger behavior, exact restoration, and clean hook disable/re-enable behavior. The main-frame smoke additionally covers the protection GUI's responsive action rail, search, status filters, 20-row bounded pagination, risk scores, and delegated isolation/ignore actions.

```powershell
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_smoke.lua
```

`item_guard_rules_smoke.lua` verifies the standalone behavior/amplification scoring engine, real-entry reachability, `item_add`, `item_loot(isDrop)`, hard resource limits, stable fingerprints, opaque handling by the core module, and GnomeMap-shaped normal workloads. Dedicated Lua rules own tokenized Script Effect inspection and publisher-aware policy. `item_guard_blacklist_smoke.lua` verifies exact creator/editor/sender matching and persistent user-list behavior. The publisher whitelist, sound, lifecycle, variable, aura, content, and Lua rule smokes cover their independent classifiers and fingerprints.

```powershell
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_rules_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_blacklist_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_sound_rules_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_lifecycle_rules_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_variable_rules_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_aura_rules_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_content_rules_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_lua_rules_smoke.lua
npx --yes --package=fengari-node-cli fengari addon/tests/item_guard_publisher_whitelist_smoke.lua
```
