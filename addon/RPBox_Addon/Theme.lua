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
    scrollFrames = {},
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

local function CaptureTemplateTextures(widget, excludedRegion)
    if not widget or widget._rpboxTemplateTextures then return end
    widget._rpboxTemplateTextures = {}

    local visited = {}
    local function CaptureTree(node)
        if not node or visited[node] then return end
        visited[node] = true

        if node.GetRegions then
            local regions = { node:GetRegions() }
            for _, region in ipairs(regions) do
                if IsTexture(region) and region ~= excludedRegion then
                    widget._rpboxTemplateTextures[#widget._rpboxTemplateTextures + 1] = {
                        region = region,
                        alpha = region.GetAlpha and region:GetAlpha() or 1,
                    }
                end
            end
        end

        if node.GetChildren then
            local children = { node:GetChildren() }
            for _, child in ipairs(children) do
                CaptureTree(child)
            end
        end
    end
    CaptureTree(widget)
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

local function GetNamedWidget(widget, key, suffix)
    if not widget then return nil end
    if widget[key] then return widget[key] end
    if widget.GetName then
        local name = widget:GetName()
        if name and _G then return _G[name .. suffix] end
    end
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
    local theme = UI.NormalizeTheme(config.uiTheme)
    if RPBox_Config and config.uiTheme ~= theme then config.uiTheme = theme end
    return theme
end

function UI.IsModern()
    return UI.GetTheme() == UI.THEME_MODERN
end

function UI.IsStyled()
    return UI.IsModern()
end

local function GetThemeColors()
    return UI.COLORS
end

local function ResolveThemeColor(color, fallbackRole)
    return color or UI.COLORS[fallbackRole]
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
        local colors = GetThemeColors()
        SetBackdrop(self, colors.dangerSoft, colors.danger)
        ApplyTextColor(closeLabel, colors.text)
    end)
    closeButton:HookScript("OnLeave", function(self)
        local colors = GetThemeColors()
        SetBackdrop(self, colors.raised, colors.border)
        ApplyTextColor(closeLabel, colors.text)
    end)

    frame._rpboxModernChrome = titleBar
    frame._rpboxModernRail = rail
    frame._rpboxModernTitle = title
    frame._rpboxModernContext = context
    frame._rpboxModernClose = closeButton
    frame._rpboxModernCloseLabel = closeLabel
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
    local theme = UI.GetTheme()
    local styled = UI.IsStyled()
    local colors = GetThemeColors()
    frame._rpboxTheme = theme

    if styled then
        SetNativeWindowChromeShown(frame, false)
        SetBackdrop(frame, colors.canvas, colors.border)
        SetBackdrop(frame._rpboxModernChrome, colors.title, colors.borderDim)
        SetBackdrop(frame._rpboxModernRail, colors.accent, colors.accent)
        SetBackdrop(frame._rpboxModernClose, colors.raised, colors.border)
        frame._rpboxModernTitle:SetText(options.title or "RPBox")
        frame._rpboxModernContext:SetText(options.context or "CHRONICLE")
        ApplyTextColor(frame._rpboxModernTitle, colors.text)
        ApplyTextColor(frame._rpboxModernContext, colors.accent)
        ApplyTextColor(frame._rpboxModernCloseLabel, colors.text)
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
    if UI.IsStyled() then
        local background = ResolveThemeColor(options.background, options.inset and "raised" or "surface")
        local border = ResolveThemeColor(options.border, options.inset and "border" or "borderDim")
        SetBackdrop(frame, background, border)
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
    if not UI.IsStyled() then
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
    local colors = GetThemeColors()
    local background = colors.raised
    local border = colors.border
    local textColor = colors.text

    if options.variant == "danger" then
        background = hovered and colors.dangerSoft or colors.raised
        border = hovered and colors.danger or colors.border
    elseif selected then
        background = colors.accentSoft
        border = colors.accent
    elseif hovered and enabled then
        background = colors.hover
        border = colors.accent
    elseif not enabled then
        background = colors.surface
        border = colors.borderDim
        textColor = colors.muted
    end

    SetBackdrop(button, background, border)
    ApplyTextColor(fontString, textColor)
    if button._rpboxActiveLine then
        SetBackdrop(button._rpboxActiveLine, colors.accent, colors.accent)
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
    if not UI.IsStyled() then
        SetTemplateTexturesShown(editBox, true)
        ClearBackdrop(editBox)
        if editBox.SetTextColor and editBox._rpboxOriginalTextColor then
            local color = editBox._rpboxOriginalTextColor
            editBox:SetTextColor(color[1], color[2], color[3], color[4] or 1)
        end
        return
    end

    SetTemplateTexturesShown(editBox, false)
    local colors = GetThemeColors()
    local border = editBox._rpboxFocused and colors.accent or colors.border
    SetBackdrop(editBox, colors.input, border)
    if editBox.SetTextColor then
        if not editBox._rpboxOriginalTextColor and editBox.GetTextColor then
            local r, g, b, a = editBox:GetTextColor()
            editBox._rpboxOriginalTextColor = { r, g, b, a }
        end
        editBox:SetTextColor(colors.text[1], colors.text[2], colors.text[3], 1)
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
    local dropdownText = GetNamedWidget(dropdown, "Text", "Text")
    if not UI.IsStyled() then
        SetTemplateTexturesShown(dropdown, true)
        ClearBackdrop(dropdown)
        RestoreTextColor(dropdownText)
        SetShown(dropdown._rpboxDropdownArrow, false)
        return
    end

    SetTemplateTexturesShown(dropdown, false)
    local colors = GetThemeColors()
    SetBackdrop(
        dropdown,
        dropdown._rpboxHovered and colors.hover or colors.input,
        dropdown._rpboxHovered and colors.accent or colors.border
    )
    ApplyTextColor(dropdownText, colors.text)
    ApplyTextColor(dropdown._rpboxDropdownArrow, colors.accent)
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
    if not dropdown._rpboxThemeHooks then
        local clickTarget = GetNamedWidget(dropdown, "Button", "Button") or dropdown
        clickTarget:HookScript("OnEnter", function()
            dropdown._rpboxHovered = true
            ApplyDropdown(dropdown)
        end)
        clickTarget:HookScript("OnLeave", function()
            dropdown._rpboxHovered = false
            ApplyDropdown(dropdown)
        end)
        dropdown._rpboxThemeHooks = true
    end
    registries.dropdowns[dropdown] = true
    ApplyDropdown(dropdown)
end

local function ApplyCheckButton(checkButton)
    if not UI.IsStyled() then
        SetTemplateTexturesShown(checkButton, true)
        SetShown(checkButton._rpboxCheckSurface, false)
        SetShown(checkButton._rpboxCheckMark, false)
        return
    end

    SetTemplateTexturesShown(checkButton, false)
    local colors = GetThemeColors()
    SetBackdrop(checkButton._rpboxCheckSurface, colors.input, colors.border)
    SetBackdrop(checkButton._rpboxCheckMark, colors.accent, colors.accent)
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

local function GetScrollBar(scrollFrame)
    return GetNamedWidget(scrollFrame, "ScrollBar", "ScrollBar")
end

local function GetScrollButton(scrollBar, key, suffix)
    return GetNamedWidget(scrollBar, key, suffix)
end

local function CaptureThumbState(thumb)
    if not thumb or thumb._rpboxOriginalThumbState then return end
    local state = {}
    if thumb.GetAtlas then state.atlas = thumb:GetAtlas() end
    if thumb.GetTexture then state.texture = thumb:GetTexture() end
    if thumb.GetVertexColor then
        state.r, state.g, state.b, state.a = thumb:GetVertexColor()
    end
    if thumb.GetAlpha then state.alpha = thumb:GetAlpha() end
    if thumb.GetWidth then state.width = thumb:GetWidth() end
    if thumb.GetHeight then state.height = thumb:GetHeight() end
    thumb._rpboxOriginalThumbState = state
end

local function RestoreThumb(thumb)
    local state = thumb and thumb._rpboxOriginalThumbState
    if not state then return end
    if state.atlas and thumb.SetAtlas then
        thumb:SetAtlas(state.atlas, false)
    elseif state.texture and thumb.SetTexture then
        thumb:SetTexture(state.texture)
    end
    if thumb.SetVertexColor and state.r then
        thumb:SetVertexColor(state.r, state.g, state.b, state.a or 1)
    end
    if thumb.SetAlpha and state.alpha then thumb:SetAlpha(state.alpha) end
    if thumb.SetSize and state.width and state.height then thumb:SetSize(state.width, state.height) end
end

local function ApplyScrollButton(button)
    if not button or not button._rpboxScrollSurface then return end
    local styled = UI.IsStyled()
    SetShown(button._rpboxScrollSurface, styled)
    SetShown(button._rpboxScrollGlyph, styled)
    if not styled then return end

    local enabled = not button.IsEnabled or button:IsEnabled()
    local hovered = button._rpboxHovered == true and enabled
    local colors = GetThemeColors()
    SetBackdrop(
        button._rpboxScrollSurface,
        hovered and colors.hover or colors.raised,
        hovered and colors.accent or colors.borderDim
    )
    ApplyTextColor(button._rpboxScrollGlyph, enabled and colors.accent or colors.muted)
end

local function EnsureScrollButton(button, glyph)
    if not button or button._rpboxScrollSurface then return end

    local surface = CreateFrame("Frame", nil, button, "BackdropTemplate")
    surface:SetAllPoints(button)
    surface:EnableMouse(false)
    local label = button:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    label:SetPoint("CENTER", 0, 0)
    label:SetText(glyph)
    button._rpboxScrollSurface = surface
    button._rpboxScrollGlyph = label

    button:HookScript("OnEnter", function(self)
        self._rpboxHovered = true
        ApplyScrollButton(self)
    end)
    button:HookScript("OnLeave", function(self)
        self._rpboxHovered = false
        ApplyScrollButton(self)
    end)
    button:HookScript("OnEnable", ApplyScrollButton)
    button:HookScript("OnDisable", ApplyScrollButton)
end

local function EnsureScrollFrameChrome(scrollFrame)
    local scrollBar = GetScrollBar(scrollFrame)
    if not scrollBar then return nil end
    if scrollFrame._rpboxScrollBar then return scrollBar end

    local thumb = scrollBar.GetThumbTexture and scrollBar:GetThumbTexture()
        or GetNamedWidget(scrollBar, "ThumbTexture", "ThumbTexture")
    CaptureThumbState(thumb)
    CaptureTemplateTextures(scrollBar, thumb)

    local track = scrollBar:CreateTexture(nil, "BACKGROUND")
    track:SetPoint("TOP", 0, -18)
    track:SetPoint("BOTTOM", 0, 18)
    track:SetWidth(6)

    local upButton = GetScrollButton(scrollBar, "ScrollUpButton", "ScrollUpButton")
    local downButton = GetScrollButton(scrollBar, "ScrollDownButton", "ScrollDownButton")
    EnsureScrollButton(upButton, "▲")
    EnsureScrollButton(downButton, "▼")

    scrollFrame._rpboxScrollBar = scrollBar
    scrollFrame._rpboxScrollThumb = thumb
    scrollFrame._rpboxScrollTrack = track
    scrollFrame._rpboxScrollUpButton = upButton
    scrollFrame._rpboxScrollDownButton = downButton
    return scrollBar
end

local function ApplyScrollFrame(scrollFrame)
    local scrollBar = EnsureScrollFrameChrome(scrollFrame)
    if not scrollBar then return end
    local styled = UI.IsStyled()
    SetTemplateTexturesShown(scrollBar, not styled)
    SetShown(scrollFrame._rpboxScrollTrack, styled)
    ApplyScrollButton(scrollFrame._rpboxScrollUpButton)
    ApplyScrollButton(scrollFrame._rpboxScrollDownButton)

    local thumb = scrollFrame._rpboxScrollThumb
    if styled then
        local colors = GetThemeColors()
        if scrollFrame._rpboxScrollTrack and scrollFrame._rpboxScrollTrack.SetColorTexture then
            local color = colors.input
            scrollFrame._rpboxScrollTrack:SetColorTexture(color[1], color[2], color[3], color[4] or 1)
        end
        if thumb then
            if thumb.SetTexture then thumb:SetTexture(WHITE_TEXTURE) end
            if thumb.SetVertexColor then
                local color = colors.accent
                thumb:SetVertexColor(color[1], color[2], color[3], color[4] or 1)
            end
            if thumb.SetAlpha then thumb:SetAlpha(1) end
            if thumb.SetWidth then thumb:SetWidth(8) end
        end
    else
        RestoreThumb(thumb)
    end
end

function UI.RegisterScrollFrame(scrollFrame)
    if not scrollFrame then return end
    registries.scrollFrames[scrollFrame] = true
    ApplyScrollFrame(scrollFrame)
end

local function ApplyStyledText(fontString)
    if UI.IsStyled() then
        local colors = GetThemeColors()
        local role = fontString._rpboxTextRole
        local roleColors = {
            heading = colors.accent,
            primary = colors.text,
            secondary = colors.text,
            muted = colors.muted,
            success = colors.text,
            danger = colors.danger,
        }
        ApplyTextColor(fontString, roleColors[role] or colors.text)
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
    for scrollFrame in pairs(registries.scrollFrames) do ApplyScrollFrame(scrollFrame) end
    for fontString in pairs(registries.text) do ApplyStyledText(fontString) end
end

function UI.SetTheme(theme)
    RPBox_Config = RPBox_Config or {}
    RPBox_Config.uiTheme = UI.NormalizeTheme(theme)
    UI.ApplyAll()
    return RPBox_Config.uiTheme
end
