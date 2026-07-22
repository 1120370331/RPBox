# Addon playback smoke test

`main_frame_playback_smoke.lua` loads the real `addon/RPBox_Addon/MainFrame.lua` in a small mocked WoW UI runtime. It drives `ns.OpenMainFrame`, the navigation/copy/settings button scripts, and the real `C_Timer.After` scan/render queue.

Run it from the repository root:

```powershell
npx --yes --package=fengari-node-cli fengari addon/tests/main_frame_playback_smoke.lua
```

The fixture contains more than 3,000 chronological records, a legacy embedded identity, and a v2 profile-switch event. The assertions cover bounded scanning and rendering, the 40–120 page-size clamp, complete page navigation, current-page-only copy, profile/name filtering, and throttled live updates both on historical and latest pages.
