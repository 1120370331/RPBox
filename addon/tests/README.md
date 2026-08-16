# Addon playback smoke test

`main_frame_playback_smoke.lua` loads the real `Theme.lua` and `MainFrame.lua` files in a small mocked WoW UI runtime. It drives `ns.OpenMainFrame`, theme switching, navigation/copy/settings button scripts, and the real `C_Timer.After` scan/render queue.

Run it from the repository root:

```powershell
npx --yes --package=fengari-node-cli fengari addon/tests/main_frame_playback_smoke.lua
```

The fixture contains more than 3,000 chronological records, a legacy embedded identity, and a v2 profile-switch event. The assertions cover modern/classic theme persistence and immediate application, compact sidebar bounds, the date-time picker, searchable/avatar-backed/paginated speaker multi-selection, system messages hidden by default and restored by the setting, bounded scanning and rendering, the 40–120 page-size clamp, complete page navigation, current-page-only copy, profile/name filtering, and throttled live updates both on historical and latest pages.

`chat_logger_filter_smoke.lua` verifies the optional non-RP player filter: it is off when unset or explicitly disabled, can be enabled to require a detected RP profile, and preserves white/black list precedence.

```powershell
npx --yes --package=fengari-node-cli fengari addon/tests/chat_logger_filter_smoke.lua
```
