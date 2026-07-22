-- RPBox LogWindow compatibility entrypoint
-- The archive/ledger lives in MainFrame so rendering and filters have one source of truth.

local ADDON_NAME, ns = ...

function ns.OpenLogWindow(todayOnly)
    ns.OpenMainFrame({
        reset = true,
        datePreset = todayOnly and 0 or "all",
    })
end
