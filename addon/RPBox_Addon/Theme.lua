-- RPBox Addon UI themes
-- Flat, reusable styling primitives for the addon-owned frames.

local ADDON_NAME, ns = ...

local UI = {}
ns.UI = UI

UI.THEME_MODERN = "modern"
UI.THEME_CLASSIC = "classic"
UI.DEFAULT_THEME = UI.THEME_MODERN

UI.COLORS = {
    canvas = { 0.035, 0.043, 0.052, 0.98 },
    title = { 0.055, 0.066, 0.078, 1 },
    surface = { 0.065, 0.078, 0.092, 0.96 },
    raised = { 0.085, 0.098, 0.115, 1 },
    hover = { 0.12, 0.135, 0.155, 1 },
    input = { 0.025, 0.031, 0.038, 0.96 },
    border = { 0.20, 0.225, 0.255, 1 },
    borderDim = { 0.125, 0.145, 0.17, 1 },
    accent = { 0.76, 0.43, 0.20, 1 },
    accentSoft = { 0.30, 0.17, 0.085, 1 },
    text = { 0.90, 0.91, 0.92, 1 },
    muted = { 0.54, 0.58, 0.63, 1 },
    danger = { 0.68, 0.20, 0.19, 1 },
    dangerSoft = { 0.24, 0.075, 0.07, 1 },
}

local WHITE_TEXTURE = "Interface\\Buttons\\WHITE8X8"
local SOLID_BACKDROP = {
    bgFile = WHITE_TEXTURE,
    edgeFile = WHITE_TEXTURE,
    edgeSize = 1,
}

local registries = {
    windows = {},
    panels = {},
    buttons = {},
    editBoxes = {},
    dropdowns = {},
    checkButtons = {},
    text = {},
}

local function SetShown(region, shown)
    if not region then return end
    if shown then
        if region.Show then region:Show() end
    elseif region.Hide then
        region:Hide()
    end
end

local function SetBackdrop(frame, background, border)
    if not frame or not frame.SetBackdrop then return end
    frame:SetBackdrop(SOLID_BACKDROP)
    frame:SetBackdropColor(background[1], background[2], background[3], background[4] or 1)
    border = border or UI.COLORS.border
    frame:SetBackdropBorderColor(border[1], border[2], border[3], border[4] or 1)
end

local function ClearBackdrop(frame)
    if frame and frame.SetBackdrop then frame:SetBackdrop(nil) end
end

local function IsTexture(region)
    if not region then return false end
    if region.IsObjectType then return region:IsObjectType("Texture") end
    if region.GetObjectType then return region:GetObjectType() == "Texture" end
    return false
end

local function CaptureTemplateTextures(widget)
    if not widget or widget._rpboxTemplateTextures then return end
    widget._rpboxTemplateTextures = {}
    if not widget.GetRegions then return end

    local regions = { widget:GetRegions() }
    for _, region in ipairs(regions) do
        if IsTexture(region) then
            widget._rpboxTemplateTextures[#widget._rpboxTemplateTextures + 1] = {
                region = region,
                alpha = region.GetAlpha and region:GetAlpha() or 1,
            }
        end
    end
end

local function SetTemplateTexturesShown(widget, shown)
    CaptureTemplateTextures(widget)
    for _, entry in ipairs(widget._rpboxTemplateTextures or {}) do
        if entry.region.SetAlpha then
            entry.region:SetAlpha(shown and entry.alpha or 0)
        else
            SetShown(entry.region, shown)
        end
    end
end

local function GetFontString(widget)
    if widget and widget.GetFontString then return widget:GetFontString() end
    return nil
end

local function RememberTextColor(fontString)
    if not fontString or fontString._rpboxOriginalTextColor or not fontString.GetTextColor then return end
    local r, g, b, a = fontString:GetTextColor()
    fontString._rpboxOriginalTextColor = { r, g, b, a }
end

local function ApplyTextColor(fontString, color)
    if not fontString or not fontString.SetTextColor then return end
    RememberTextColor(fontString)
    fontString:SetTextColor(color[1], color[2], color[3], color[4] or 1)
end

local function RestoreTextColor(fontString)
    if not fontString or not fontString.SetTextColor then return end
    local color = fontString._rpboxOriginalTextColor
    if color then fontString:SetTextColor(color[1], color[2], color[3], color[4] or 1) end
end

function UI.NormalizeTheme(theme)
    return theme == UI.THEME_CLASSIC and UI.THEME_CLASSIC or UI.THEME_MODERN
end

function UI.GetTheme()
    local config = RPBox_Config or {}
    return UI.NormalizeTheme(config.uiTheme)
end

function UI.IsModern()
    return UI.GetTheme() == UI.THEME_MODERN
end

local function EnsureWindowChrome(frame)
    if frame._rpboxModernChrome then return end

    local titleBar = CreateFrame("Frame", nil, frame, "BackdropTemplate")
    titleBar:SetPoint("TOPLEFT", 1, -1)
    titleBar:SetPoint("TOPRIGHT", -1, -1)
    titleBar:SetHeight(28)
    titleBar:EnableMouse(false)

    local rail = CreateFrame("Frame", nil, frame, "BackdropTemplate")
    rail:SetPoint("TOPLEFT", 1, -1)
    rail:SetPoint("BOTTOMLEFT", 1, 1)
    rail:SetWidth(3)
    rail:EnableMouse(false)

    local title = titleBar:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    title:SetPoint("LEFT", 12, 0)
    title:SetJustifyH("LEFT")

    local context = titleBar:CreateFontString(nil, "OVERLAY", "GameFontDisableSmall")
    context:SetPoint("LEFT", title, "RIGHT", 8, -1)
    context:SetText("CHRONICLE")

    local closeButton = CreateFrame("Button", nil, titleBar, "BackdropTemplate")
    closeButton:SetSize(24, 20)
    closeButton:SetPoint("RIGHT", -4, 0)
    local closeLabel = closeButton:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    closeLabel:SetPoint("CENTER", 0, 0)
    closeLabel:SetText("X")
    closeButton:SetScript("OnClick", function() frame:Hide() end)
    closeButton:HookScript("OnEnter", function(self)
        SetBackdrop(self, UI.COLORS.dangerSoft, UI.COLORS.danger)
    end)
    closeButton:HookScript("OnLeave", function(self)
        SetBackdrop(self, UI.COLORS.raised, UI.COLORS.border)
    end)

    frame._rpboxModernChrome = titleBar
    frame._rpboxModernRail = rail
    frame._rpboxModernTitle = title
    frame._rpboxModernContext = context
    frame._rpboxModernClose = closeButton
end

local WINDOW_CHROME_KEYS = {
    "Bg",
    "TitleBg",
    "Inset",
    "NineSlice",
    "Portrait",
    "PortraitContainer",
}

local function SetNativeWindowChromeShown(frame, shown)
    for _, key in ipairs(WINDOW_CHROME_KEYS) do
        SetShown(frame[key], shown)
    end
    SetShown(frame.TitleText, shown)
    SetShown(frame.CloseButton, shown)
end

local function ApplyWindow(frame)
    EnsureWindowChrome(frame)
    local options = frame._rpboxThemeOptions or {}
    local modern = UI.IsModern()
    frame._rpboxTheme = modern and UI.THEME_MODERN or UI.THEME_CLASSIC

    if modern then
        SetNativeWindowChromeShown(frame, false)
        SetBackdrop(frame, UI.COLORS.canvas, UI.COLORS.border)
        SetBackdrop(frame._rpboxModernChrome, UI.COLORS.title, UI.COLORS.borderDim)
        SetBackdrop(frame._rpboxModernRail, UI.COLORS.accent, UI.COLORS.accent)
        SetBackdrop(frame._rpboxModernClose, UI.COLORS.raised, UI.COLORS.border)
        frame._rpboxModernTitle:SetText(options.title or "RPBox")
        frame._rpboxModernContext:SetText(options.context or "CHRONICLE")
        ApplyTextColor(frame._rpboxModernTitle, UI.COLORS.text)
        ApplyTextColor(frame._rpboxModernContext, UI.COLORS.accent)
        SetShown(frame._rpboxModernChrome, true)
        SetShown(frame._rpboxModernRail, true)
    else
        ClearBackdrop(frame)
        SetShown(frame._rpboxModernChrome, false)
        SetShown(frame._rpboxModernRail, false)
        SetNativeWindowChromeShown(frame, true)
    end
end

function UI.RegisterWindow(frame, options)
    if not frame then return end
    frame._rpboxThemeOptions = options or frame._rpboxThemeOptions or {}
    registries.windows[frame] = true
    ApplyWindow(frame)
end

function UI.SetWindowTitle(frame, title, context)
    if not frame then return end
    frame._rpboxThemeOptions = frame._rpboxThemeOptions or {}
    frame._rpboxThemeOptions.title = title
    frame._rpboxThemeOptions.context = context or frame._rpboxThemeOptions.context
    if frame.TitleText then frame.TitleText:SetText(title) end
    if frame._rpboxModernTitle then frame._rpboxModernTitle:SetText(title) end
    if frame._rpboxModernContext and context then frame._rpboxModernContext:SetText(context) end
end

local function ApplyPanel(frame)
    local options = frame._rpboxThemeOptions or {}
    if UI.IsModern() then
        SetBackdrop(frame, options.background or UI.COLORS.surface, options.border or UI.COLORS.borderDim)
        if options.alwaysVisible then frame:Show() end
    elseif options.modernOnly then
        frame:Hide()
    elseif options.classicBackground then
        SetBackdrop(frame, options.classicBackground, options.classicBorder or UI.COLORS.border)
    else
        ClearBackdrop(frame)
    end
end

function UI.RegisterPanel(frame, options)
    if not frame then return end
    frame._rpboxThemeOptions = options or frame._rpboxThemeOptions or {}
    registries.panels[frame] = true
    ApplyPanel(frame)
end

local function ApplyButton(button)
    local options = button._rpboxThemeOptions or {}
    local fontString = GetFontString(button)
    if not UI.IsModern() then
        SetTemplateTexturesShown(button, true)
        ClearBackdrop(button)
        RestoreTextColor(fontString)
        SetShown(button._rpboxActiveLine, false)
        return
    end

    SetTemplateTexturesShown(button, false)
    local selected = button._rpboxSelected == true
    local enabled = not button.IsEnabled or button:IsEnabled()
    local hovered = button._rpboxHovered == true
    local background = UI.COLORS.raised
    local border = UI.COLORS.border
    local textColor = UI.COLORS.text

    if options.variant == "danger" then
        background = hovered and UI.COLORS.dangerSoft or UI.COLORS.raised
        border = hovered and UI.COLORS.danger or UI.COLORS.border
    elseif selected then
        background = UI.COLORS.accentSoft
        border = UI.COLORS.accent
    elseif hovered and enabled then
        background = UI.COLORS.hover
        border = UI.COLORS.accent
    elseif not enabled then
        background = UI.COLORS.surface
        border = UI.COLORS.borderDim
        textColor = UI.COLORS.muted
    end

    SetBackdrop(button, background, border)
    ApplyTextColor(fontString, textColor)
    if button._rpboxActiveLine then
        SetBackdrop(button._rpboxActiveLine, UI.COLORS.accent, UI.COLORS.accent)
        SetShown(button._rpboxActiveLine, selected)
    end
end

function UI.RegisterButton(button, options)
    if not button then return end
    button._rpboxThemeOptions = options or button._rpboxThemeOptions or {}
    CaptureTemplateTextures(button)
    if not button._rpboxActiveLine then
        local line = CreateFrame("Frame", nil, button, "BackdropTemplate")
        line:SetPoint("BOTTOMLEFT", 1, 1)
        line:SetPoint("BOTTOMRIGHT", -1, 1)
        line:SetHeight(2)
        line:EnableMouse(false)
        button._rpboxActiveLine = line
    end
    if not button._rpboxThemeHooks then
        button:HookScript("OnEnter", function(self)
            self._rpboxHovered = true
            ApplyButton(self)
        end)
        button:HookScript("OnLeave", function(self)
            self._rpboxHovered = false
            ApplyButton(self)
        end)
        button:HookScript("OnEnable", ApplyButton)
        button:HookScript("OnDisable", ApplyButton)
        button._rpboxThemeHooks = true
    end
    registries.buttons[button] = true
    ApplyButton(button)
end

function UI.SetButtonSelected(button, selected)
    if not button then return end
    button._rpboxSelected = selected == true
    ApplyButton(button)
end

function UI.RefreshButton(button)
    if button then ApplyButton(button) end
end

local function ApplyEditBox(editBox)
    if not UI.IsModern() then
        SetTemplateTexturesShown(editBox, true)
        ClearBackdrop(editBox)
        if editBox.SetTextColor and editBox._rpboxOriginalTextColor then
            local color = editBox._rpboxOriginalTextColor
            editBox:SetTextColor(color[1], color[2], color[3], color[4] or 1)
        end
        return
    end

    SetTemplateTexturesShown(editBox, false)
    local border = editBox._rpboxFocused and UI.COLORS.accent or UI.COLORS.border
    SetBackdrop(editBox, UI.COLORS.input, border)
    if editBox.SetTextColor then
        if not editBox._rpboxOriginalTextColor and editBox.GetTextColor then
            local r, g, b, a = editBox:GetTextColor()
            editBox._rpboxOriginalTextColor = { r, g, b, a }
        end
        editBox:SetTextColor(UI.COLORS.text[1], UI.COLORS.text[2], UI.COLORS.text[3], 1)
    end
end

function UI.RegisterEditBox(editBox)
    if not editBox then return end
    CaptureTemplateTextures(editBox)
    if not editBox._rpboxThemeHooks then
        editBox:HookScript("OnEditFocusGained", function(self)
            self._rpboxFocused = true
            ApplyEditBox(self)
        end)
        editBox:HookScript("OnEditFocusLost", function(self)
            self._rpboxFocused = false
            ApplyEditBox(self)
        end)
        editBox._rpboxThemeHooks = true
    end
    registries.editBoxes[editBox] = true
    ApplyEditBox(editBox)
end

local function ApplyDropdown(dropdown)
    if not UI.IsModern() then
        SetTemplateTexturesShown(dropdown, true)
        ClearBackdrop(dropdown)
        SetShown(dropdown._rpboxDropdownArrow, false)
        return
    end

    SetTemplateTexturesShown(dropdown, false)
    SetBackdrop(dropdown, UI.COLORS.input, UI.COLORS.border)
    ApplyTextColor(dropdown._rpboxDropdownArrow, UI.COLORS.accent)
    SetShown(dropdown._rpboxDropdownArrow, true)
end

function UI.RegisterDropdown(dropdown)
    if not dropdown then return end
    CaptureTemplateTextures(dropdown)
    if not dropdown._rpboxDropdownArrow then
        local arrow = dropdown:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        arrow:SetPoint("RIGHT", -10, 1)
        arrow:SetText("▼")
        dropdown._rpboxDropdownArrow = arrow
    end
    registries.dropdowns[dropdown] = true
    ApplyDropdown(dropdown)
end

local function ApplyCheckButton(checkButton)
    if not UI.IsModern() then
        SetTemplateTexturesShown(checkButton, true)
        SetShown(checkButton._rpboxCheckSurface, false)
        SetShown(checkButton._rpboxCheckMark, false)
        return
    end

    SetTemplateTexturesShown(checkButton, false)
    SetBackdrop(checkButton._rpboxCheckSurface, UI.COLORS.input, UI.COLORS.border)
    SetBackdrop(checkButton._rpboxCheckMark, UI.COLORS.accent, UI.COLORS.accent)
    SetShown(checkButton._rpboxCheckSurface, true)
    SetShown(checkButton._rpboxCheckMark, checkButton:GetChecked() == true)
end

function UI.RegisterCheckButton(checkButton)
    if not checkButton then return end
    CaptureTemplateTextures(checkButton)
    if not checkButton._rpboxCheckSurface then
        local surface = CreateFrame("Frame", nil, checkButton, "BackdropTemplate")
        surface:SetSize(16, 16)
        surface:SetPoint("CENTER", 0, 0)
        surface:EnableMouse(false)
        local mark = CreateFrame("Frame", nil, surface, "BackdropTemplate")
        mark:SetSize(8, 8)
        mark:SetPoint("CENTER", 0, 0)
        mark:EnableMouse(false)
        checkButton._rpboxCheckSurface = surface
        checkButton._rpboxCheckMark = mark
    end
    if not checkButton._rpboxThemeHooks then
        checkButton:HookScript("OnClick", ApplyCheckButton)
        checkButton._rpboxThemeHooks = true
    end
    registries.checkButtons[checkButton] = true
    ApplyCheckButton(checkButton)
end

function UI.RefreshCheckButton(checkButton)
    if checkButton then ApplyCheckButton(checkButton) end
end

local TEXT_ROLE_COLORS = {
    heading = UI.COLORS.accent,
    primary = UI.COLORS.text,
    muted = UI.COLORS.muted,
}

local function ApplyStyledText(fontString)
    if UI.IsModern() then
        ApplyTextColor(fontString, TEXT_ROLE_COLORS[fontString._rpboxTextRole] or UI.COLORS.text)
    else
        RestoreTextColor(fontString)
    end
end

function UI.RegisterText(fontString, role)
    if not fontString then return end
    fontString._rpboxTextRole = role or "primary"
    registries.text[fontString] = true
    ApplyStyledText(fontString)
end

function UI.ApplyAll()
    for frame in pairs(registries.windows) do ApplyWindow(frame) end
    for frame in pairs(registries.panels) do ApplyPanel(frame) end
    for button in pairs(registries.buttons) do ApplyButton(button) end
    for editBox in pairs(registries.editBoxes) do ApplyEditBox(editBox) end
    for dropdown in pairs(registries.dropdowns) do ApplyDropdown(dropdown) end
    for checkButton in pairs(registries.checkButtons) do ApplyCheckButton(checkButton) end
    for fontString in pairs(registries.text) do ApplyStyledText(fontString) end
end

function UI.SetTheme(theme)
    RPBox_Config = RPBox_Config or {}
    RPBox_Config.uiTheme = UI.NormalizeTheme(theme)
    UI.ApplyAll()
    return RPBox_Config.uiTheme
end
