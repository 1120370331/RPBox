"""Build the self-contained hybrid OvoFrame menu viewer from two raster pages.

The generated Lua keeps ornaments and section headings as bitmap regions,
replaces body copy with native FontStrings, merges equal horizontal runs
vertically, and packs the remaining stream with base-32 varints.
"""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image, ImageFilter, ImageOps


ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
TARGET_WIDTH = 406
TARGET_HEIGHT = 640
# Keep the fitted aspect ratio unchanged while moving the crop window down.
# The source artwork has much more blank paper above its ornament than below;
# this split preserves the complete lower flourish instead of cutting through it.
TOP_CROP = 100
BOTTOM_CROP = 10
TOP_ART_BOTTOM = 94
BOTTOM_ART_TOP = 586

HEADING_REGIONS = {
    1: (
        (35, 85, 176, 115),
        (214, 85, 352, 115),
        (35, 246, 176, 274),
        (214, 190, 352, 215),
        (214, 315, 352, 339),
        (214, 392, 352, 417),
        (214, 475, 352, 501),
    ),
    2: (
        (35, 85, 176, 115),
        (214, 85, 352, 115),
        (35, 246, 176, 274),
        (214, 190, 352, 215),
        (214, 315, 352, 339),
        (214, 475, 352, 501),
    ),
}


def quantize_page(path: Path, page_index: int) -> list[list[int]]:
    with Image.open(path) as source:
        source = source.convert("L")
        source = source.crop((0, TOP_CROP, source.width, source.height - BOTTOM_CROP))
        grayscale = ImageOps.fit(
            source,
            (TARGET_WIDTH, TARGET_HEIGHT),
            method=Image.Resampling.LANCZOS,
            centering=(0.5, 0.5),
        ).filter(ImageFilter.GaussianBlur(0.2))

    pixels = grayscale.load()
    rows: list[list[int]] = []
    for y in range(TARGET_HEIGHT):
        row: list[int] = []
        for x in range(TARGET_WIDTH):
            value = pixels[x, y]
            in_heading = any(
                left <= x < right and top <= y < bottom
                for left, top, right, bottom in HEADING_REGIONS[page_index]
            )
            keep_pixel = (
                y < TOP_ART_BOTTOM
                or y >= BOTTOM_ART_TOP
                or x < 34
                or x >= TARGET_WIDTH - 34
                or in_heading
            )
            if not keep_pixel or value >= 218:
                level = 0
            elif value >= 155:
                level = 1
            else:
                level = 2
            row.append(level)
        rows.append(row)
    return rows


def merge_rectangles(rows: list[list[int]]) -> list[tuple[int, int, int, int, int]]:
    active: dict[tuple[int, int, int], list[int]] = {}
    rectangles: list[list[int]] = []

    for y, row in enumerate(rows):
        runs: list[tuple[int, int, int]] = []
        x = 0
        while x < TARGET_WIDTH:
            level = row[x]
            if level == 0:
                x += 1
                continue

            start = x
            while x < TARGET_WIDTH and row[x] == level:
                x += 1
            runs.append((start, x - start, level))

        next_active: dict[tuple[int, int, int], list[int]] = {}
        for run in runs:
            if run in active:
                rectangle = active[run]
                rectangle[3] += 1
            else:
                rectangle = [run[0], y, run[1], 1, run[2]]
            next_active[run] = rectangle

        for run, rectangle in active.items():
            if run not in next_active:
                rectangles.append(rectangle)
        active = next_active

    rectangles.extend(active.values())
    return [tuple(rectangle) for rectangle in sorted(rectangles, key=lambda item: (item[1], item[0], item[4]))]


def encode_uint(value: int) -> str:
    encoded: list[str] = []
    while value >= 32:
        encoded.append(ALPHABET[(value % 32) + 32])
        value //= 32
    encoded.append(ALPHABET[value])
    return "".join(encoded)


def encode_rectangles(rectangles: list[tuple[int, int, int, int, int]]) -> str:
    encoded: list[str] = []
    previous_y = 0
    previous_x = 0

    for x, y, width, height, level in rectangles:
        delta_y = y - previous_y
        encoded.append(encode_uint(delta_y))
        encoded.append(encode_uint(x - previous_x if delta_y == 0 else x))
        encoded.append(encode_uint(width))
        encoded.append(encode_uint(height))
        encoded.append(encode_uint(level))
        previous_y = y
        previous_x = x

    return "".join(encoded)


def lua_string_chunks(data: str, indent: str = "        ") -> str:
    chunks = [data[index : index + 120] for index in range(0, len(data), 120)]
    return "\n".join(f'{indent}"{chunk}",' for chunk in chunks)


def make_lua_ascii_safe(source: str) -> str:
    """Encode non-ASCII source characters as Lua decimal UTF-8 byte escapes."""
    output: list[str] = []
    for character in source:
        if ord(character) < 128:
            output.append(character)
        else:
            output.extend(f"\\{byte:03d}" for byte in character.encode("utf-8"))
    return "".join(output)


def build_lua(page_data: list[str], counts: list[int]) -> str:
    first_chunks = lua_string_chunks(page_data[0])
    second_chunks = lua_string_chunks(page_data[1])
    return f'''-- Self-contained hybrid OvoFrame / TRP3 Extended menu viewer.
-- Generated by scripts/build_ovo_embedded_menu.py.
-- Ornaments, title and section headings are packed as merged pixel rectangles;
-- menu items and prices are rendered with native WoW FontStrings.

if not args or not args._G then
    if effect then
        effect("text", args, "OvoMenu：未检测到 args._G，请先执行受信任的 /run 注入命令。", "4")
    end
    return
end

local G = args._G
local FRAME_NAME = "RPBoxOvoHybridMenuViewerV10"
local SOURCE_WIDTH = {TARGET_WIDTH}
local SOURCE_HEIGHT = {TARGET_HEIGHT}
local PAGE_ASPECT = SOURCE_WIDTH / SOURCE_HEIGHT
local ALPHABET = "{ALPHABET}"
local PAGE_COUNTS = {{ {counts[0]}, {counts[1]} }}
local PAGE_ORDER = {{ 2, 1 }}
local PAGE_DATA = {{
    table.concat({{
{first_chunks}
    }}),
    table.concat({{
{second_chunks}
    }}),
}}

local PAGE_SECTIONS = {{
    {{
        {{ x = 34, y = 116, priceX = 164, priceWidth = 43, title = "Non-alcoholic", step = 17, items = {{
            {{ "净水", "免费提供" }}, {{ "散养羊奶", "10银币" }}, {{ "绽放花蜜", "15银币" }},
            {{ "香浓酸奶", "10银币" }}, {{ "金色林地果汁", "15银币" }}, {{ "苏拉玛香料茶", "95银币" }},
        }} }},
        {{ x = 34, y = 238, priceX = 164, priceWidth = 43, title = "Main course", step = 18, items = {{
            {{ "蜜梅馅饼", "15银币" }}, {{ "绒羊里脊", "25银币" }}, {{ "小动物烤肉", "25银币" }},
            {{ "精致镰蛙腿", "30银币" }}, {{ "血骑士汉堡", "30银币" }}, {{ "法力鳗鱼卷", "45银币" }},
            {{ "奢华煎蛋卷", "45银币" }}, {{ "盐腌潮路鳕鱼", "25银币" }}, {{ "胶冻深海鳗鱼", "25银币" }},
            {{ "罗兰的著名腊肠", "10银币" }}, {{ "坦德士上校的香辣鸡", "10银币" }},
            {{ "闪金镇农场烟熏香肠", "15银币" }}, {{ "迪米·吉恩的旭日猪肉", "10银币" }},
            {{ "罗思科·福莱尔的多肉肠", "15银币" }}, {{ "奶酪拼盘", "45银币" }},
        }} }},
        {{ x = 220, y = 116, priceX = 350, priceWidth = 43, title = "Salads", step = 17, items = {{
            {{ "葡萄干土豆沙拉", "20银币" }}, {{ "凯尔甘蓝阳光沙拉", "25银币" }}, {{ "花岗岩沙拉", "20银币" }},
        }} }},
        {{ x = 220, y = 196, priceX = 350, priceWidth = 43, title = "Cheese", step = 17, items = {{
            {{ "奶油芝士", "5银币" }}, {{ "伯拉勒斯蓝酪", "5银币" }}, {{ "海瑟福特芝士", "5银币" }},
            {{ "阿苏纳湿奶酪", "5银币" }}, {{ "提拉加德尖奶酪", "5银币" }},
        }} }},
        {{ x = 220, y = 310, priceX = 350, priceWidth = 43, title = "Side", step = 17, items = {{
            {{ "山羊肉干", "5银币" }}, {{ "库尔提拉斯肉丸子", "10银币" }}, {{ "群鸦鱼子酱", "45银币" }},
        }} }},
        {{ x = 220, y = 390, priceX = 350, priceWidth = 43, title = "Set", step = 17, items = {{
            {{ "晴风盛宴", "75银币" }}, {{ "银月城晚宴拼盘", "95银币" }}, {{ "奎尔丹纳斯便餐", "30银币" }},
        }} }},
        {{ x = 220, y = 470, priceX = 350, priceWidth = 43, title = "Desert", step = 17, items = {{
            {{ "腌花芽", "20银币" }}, {{ "雪梅奶油冻", "10银币" }}, {{ "酥炸黄油曲奇", "10银币" }},
            {{ "糖霜幽影坚果蛋糕", "15银币" }},
        }} }},
    }},
    {{
        {{ x = 34, y = 116, priceX = 164, priceWidth = 43, title = "Coffee", step = 17, items = {{
            {{ "星勺特制混合咖啡", "10银币" }}, {{ "德鲁斯瓦烘焙咖啡", "10银币" }}, {{ "浓缩咔啡", "40银币" }},
            {{ "水手必备咖啡", "10银币" }}, {{ "传奇轻焙咖啡", "30银币" }}, {{ "少校的泡沫咖啡", "10铜币" }},
        }} }},
        {{ x = 34, y = 248, priceX = 164, priceWidth = 43, title = "Spirit", step = 18, items = {{
            {{ "落锚黑啤", "45银币" }}, {{ "雪林灰啤", "45银币" }}, {{ "旧铁炉堡", "50银币" }},
            {{ "汇帆啤酒", "75银币" }}, {{ "碧蓝魔酒", "10银币" }}, {{ "上古火酒", "10银币" }},
            {{ "卡多雷姜酒", "10银币" }}, {{ "锚角杜松子酒", "65银币" }}, {{ "锚角波特啤酒", "65银币" }},
            {{ "苦味暗根伏特加", "65银币" }}, {{ "驳船比尔森啤酒", "75银币" }}, {{ "米登霍尔德蜜酒", "65银币" }},
            {{ "鲑鱼溪金雀花酒", "20银币" }}, {{ "库尔提拉斯三料啤酒", "80银币" }},
            {{ "布伦纳丹苹果白兰地", "65银币" }},
        }} }},
        {{ x = 220, y = 116, priceX = 350, priceWidth = 43, title = "Wine", step = 17, items = {{
            {{ "达拉然白葡萄酒", "15银币" }}, {{ "达拉然红葡萄酒", "20银币" }}, {{ "陈年达拉然红葡萄酒", "95银币" }},
        }} }},
        {{ x = 220, y = 205, priceX = 318, priceWidth = 75, title = "Aging", step = 22, items = {{
            {{ "8年·辛特兰麦芽酒", "1金币70银币" }}, {{ "12年·克莱因陈酿", "2金币85银币" }},
            {{ "25年·符文林地", "3金币95银币" }}, {{ "阿拉索孤桶", "25金币80银币" }},
        }} }},
        {{ x = 220, y = 325, priceX = 350, priceWidth = 43, title = "Cocktail", step = 18, items = {{
            {{ "北极光", "5银币" }}, {{ "净化甜酒", "15银币" }}, {{ "恒春之水", "15银币" }},
            {{ "醉亦何妨", "15银币" }}, {{ "龙鹰特调", "15银币" }}, {{ "魔导师蜂蜜酒", "30银币" }},
        }} }},
        {{ x = 220, y = 465, priceX = 350, priceWidth = 43, title = "Sparckaling", step = 18, items = {{
            {{ "永歌气泡水", "15银币" }}, {{ "逐春者气泡酒", "15银币" }},
            {{ "法兰纳尔气泡酒", "20银币" }}, {{ "晴风法兰恰寇塔", "25银币" }},
        }} }},
    }},
}}

-- Each entry stores heading anchor, row pitch and heading-to-first-row offset.
-- Values are calibrated against WoW's actual Chinese-font metrics rather than
-- the source artwork alone. Short food groups use a tighter pitch so their
-- final row clears the following bitmap heading.
local SECTION_LAYOUT = {{
    ["Non-alcoholic"] = {{ 98, 20, 25 }},
    ["Salads"] = {{ 98, 20, 25 }},
    ["Main course"] = {{ 260, 20, 21 }},
    ["Cheese"] = {{ 203, 17, 21 }},
    ["Side"] = {{ 324, 16, 21 }},
    ["Set"] = {{ 402, 16, 21 }},
    ["Desert"] = {{ 484, 16, 21 }},
    ["Coffee"] = {{ 98, 20, 25 }},
    ["Wine"] = {{ 98, 20, 25 }},
    ["Spirit"] = {{ 260, 20, 21 }},
    ["Aging"] = {{ 203, 21, 21 }},
    ["Cocktail"] = {{ 324, 20, 21 }},
    ["Sparckaling"] = {{ 486, 20, 21 }},
}}

if G.RPBoxOvoMenuViewer then
    G.RPBoxOvoMenuViewer:Hide()
end
if G.RPBoxOvoEmbeddedMenuViewer then
    G.RPBoxOvoEmbeddedMenuViewer:Hide()
end
if G.RPBoxOvoHybridMenuViewerV2 then
    G.RPBoxOvoHybridMenuViewerV2:Hide()
end
if G.RPBoxOvoHybridMenuViewerV3 then
    G.RPBoxOvoHybridMenuViewerV3:Hide()
end
if G.RPBoxOvoHybridMenuViewerV4 then
    G.RPBoxOvoHybridMenuViewerV4:Hide()
end
if G.RPBoxOvoHybridMenuViewerV5 then
    G.RPBoxOvoHybridMenuViewerV5:Hide()
end
if G.RPBoxOvoHybridMenuViewerV6 then
    G.RPBoxOvoHybridMenuViewerV6:Hide()
end
if G.RPBoxOvoHybridMenuViewerV7 then
    G.RPBoxOvoHybridMenuViewerV7:Hide()
end
if G.RPBoxOvoHybridMenuViewerV8 then
    G.RPBoxOvoHybridMenuViewerV8:Hide()
end
if G.RPBoxOvoHybridMenuViewerV9 then
    G.RPBoxOvoHybridMenuViewerV9:Hide()
end

local existing = G[FRAME_NAME]
if existing then
    existing:Show()
    if existing.RestartRender then
        existing:RestartRender(existing.currentPage or 1)
    end
    return
end

local decode = {{}}
for index = 1, #ALPHABET do
    decode[string.sub(ALPHABET, index, index)] = index - 1
end

local viewer = G.CreateFrame("Frame", FRAME_NAME, G.UIParent, "BackdropTemplate")
G[FRAME_NAME] = viewer
viewer:Hide()
viewer:SetPoint("CENTER")
viewer:SetFrameStrata("FULLSCREEN_DIALOG")
viewer:SetFrameLevel(200)
viewer:SetClampedToScreen(true)
viewer:EnableMouse(true)
viewer:EnableMouseWheel(true)
viewer:SetBackdrop({{
    bgFile = "Interface\\\\Buttons\\\\WHITE8X8",
    edgeFile = "Interface\\\\DialogFrame\\\\UI-DialogBox-Border",
    tile = true,
    tileSize = 16,
    edgeSize = 20,
    insets = {{ left = 5, right = 5, top = 5, bottom = 5 }},
}})
viewer:SetBackdropColor(0.035, 0.028, 0.022, 0.98)
viewer:SetBackdropBorderColor(0.72, 0.59, 0.38, 1)

local shadow = G.CreateFrame("Frame", nil, viewer, "BackdropTemplate")
shadow:SetPoint("TOPLEFT", viewer, "TOPLEFT", -8, 8)
shadow:SetPoint("BOTTOMRIGHT", viewer, "BOTTOMRIGHT", 8, -8)
shadow:SetFrameLevel(viewer:GetFrameLevel() - 1)
shadow:SetBackdrop({{ bgFile = "Interface\\\\Buttons\\\\WHITE8X8" }})
shadow:SetBackdropColor(0, 0, 0, 0.55)

local title = viewer:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
title:SetPoint("TOP", viewer, "TOP", 0, -17)
title:SetText("海妖之颂菜单")
title:SetTextColor(0.92, 0.80, 0.57, 1)

local canvas = G.CreateFrame("Frame", nil, viewer)
canvas:SetPoint("TOP", viewer, "TOP", 0, -45)
canvas:SetFrameLevel(viewer:GetFrameLevel() + 1)

local paper = canvas:CreateTexture(nil, "BACKGROUND")
paper:SetAllPoints(canvas)
paper:SetColorTexture(0.945, 0.925, 0.875, 1)

local textLayers = {{}}
local vectorEntries = {{}}
local bodyProbe = canvas:CreateFontString(nil, "OVERLAY", "GameFontHighlight")
local bodyFont = bodyProbe:GetFont()
bodyProbe:Hide()
local headingProbe = canvas:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
local headingFont = headingProbe:GetFont()
headingProbe:Hide()
local locale = G.GetLocale and G.GetLocale() or ""
if locale == "zhCN" then
    bodyFont = "Fonts\\\\ARKai_C.ttf"
    headingFont = "Fonts\\\\ARKai_C.ttf"
elseif locale == "zhTW" then
    bodyFont = "Fonts\\\\bKAI00M.ttf"
    headingFont = "Fonts\\\\bKAI00M.ttf"
end
title:SetFont(headingFont, 20, "")

local function addVectorText(layer, text, x, y, width, align, sourceSize, isHeading)
    local label = layer:CreateFontString(nil, "OVERLAY", isHeading and "GameFontNormalLarge" or "GameFontHighlight")
    label:SetText(text)
    label:SetJustifyH(align or "LEFT")
    label:SetJustifyV("TOP")
    label:SetWordWrap(false)
    label:SetTextColor(
        isHeading and 0.16 or 0.22,
        isHeading and 0.145 or 0.205,
        isHeading and 0.125 or 0.18,
        1
    )
    table.insert(vectorEntries, {{
        label = label,
        layer = layer,
        x = x,
        y = y,
        width = width,
        sourceSize = sourceSize,
        font = isHeading and headingFont or bodyFont,
    }})
end

for pageIndex, sections in ipairs(PAGE_SECTIONS) do
    local layer = G.CreateFrame("Frame", nil, canvas)
    layer:SetAllPoints(canvas)
    layer:SetFrameLevel(canvas:GetFrameLevel() + 2)
    textLayers[pageIndex] = layer

    for _, section in ipairs(sections) do
        local layout = SECTION_LAYOUT[section.title]
        if layout then
            section.y = layout[1]
            section.step = layout[2]
            section.itemOffset = layout[3]
        end

        if section.x < 200 then
            section.x = 50
            section.priceX = 158
            section.priceWidth = 48
        elseif section.title == "Aging" then
            section.x = 232
            section.priceX = 316
            section.priceWidth = 64
        else
            section.x = 232
            section.priceX = 328
            section.priceWidth = 42
        end

        local itemY = section.y + (section.itemOffset or 21)
        for _, item in ipairs(section.items) do
            addVectorText(layer, item[1], section.x, itemY, section.priceX - section.x - 4, "LEFT", 9.2, false)
            addVectorText(layer, item[2], section.priceX, itemY, section.priceWidth, "RIGHT", 9.2, false)
            itemY = itemY + section.step
        end
    end
end

local function layoutVectorText()
    local scaleX = canvas:GetWidth() / SOURCE_WIDTH
    local scaleY = canvas:GetHeight() / SOURCE_HEIGHT
    for _, entry in ipairs(vectorEntries) do
        local label = entry.label
        label:ClearAllPoints()
        label:SetPoint("TOPLEFT", entry.layer, "TOPLEFT", entry.x * scaleX, -entry.y * scaleY)
        label:SetWidth(entry.width * scaleX)
        label:SetHeight((entry.sourceSize + 4) * scaleY)
        local maximumWidth = entry.width * scaleX
        local fontSize = math.max(7.5, entry.sourceSize * scaleY)
        label:SetFont(entry.font, fontSize, "")
        local actualWidth = label:GetStringWidth()
        if actualWidth and actualWidth > maximumWidth and actualWidth > 0 then
            label:SetFont(entry.font, math.max(7, fontSize * maximumWidth / actualWidth), "")
        end
    end
end

local function showVectorPage(pageIndex)
    for index, layer in ipairs(textLayers) do
        if index == pageIndex then
            layer:Show()
        else
            layer:Hide()
        end
    end
end

local function hideVectorPages()
    for _, layer in ipairs(textLayers) do
        layer:Hide()
    end
end

local pageBorder = G.CreateFrame("Frame", nil, viewer, "BackdropTemplate")
pageBorder:SetFrameLevel(viewer:GetFrameLevel() + 2)
pageBorder:SetBackdrop({{
    edgeFile = "Interface\\\\Buttons\\\\WHITE8X8",
    edgeSize = 1,
}})
pageBorder:SetBackdropBorderColor(0.75, 0.66, 0.48, 0.55)
pageBorder:SetPoint("TOPLEFT", canvas, "TOPLEFT", -1, 1)
pageBorder:SetPoint("BOTTOMRIGHT", canvas, "BOTTOMRIGHT", 1, -1)

local pageIndexText = viewer:CreateFontString(nil, "OVERLAY", "GameFontHighlight")
pageIndexText:SetPoint("BOTTOM", viewer, "BOTTOM", 0, 20)
pageIndexText:SetTextColor(0.88, 0.82, 0.70, 1)
pageIndexText:SetFont(bodyFont, 13, "")

local loadingText = canvas:CreateFontString(nil, "OVERLAY", "GameFontHighlightLarge")
loadingText:SetPoint("BOTTOM", canvas, "BOTTOM", 0, 9)
loadingText:SetTextColor(0.28, 0.23, 0.17, 1)
loadingText:SetFont(bodyFont, 12, "")

local previousButton = G.CreateFrame("Button", nil, viewer, "UIPanelButtonTemplate")
previousButton:SetSize(96, 28)
previousButton:SetPoint("BOTTOMRIGHT", viewer, "BOTTOM", -42, 14)
previousButton:SetText("上一页")
previousButton:GetFontString():SetFont(bodyFont, 13, "")

local nextButton = G.CreateFrame("Button", nil, viewer, "UIPanelButtonTemplate")
nextButton:SetSize(96, 28)
nextButton:SetPoint("BOTTOMLEFT", viewer, "BOTTOM", 42, 14)
nextButton:SetText("下一页")
nextButton:GetFontString():SetFont(bodyFont, 13, "")

local closeButton = G.CreateFrame("Button", nil, viewer, "UIPanelCloseButton")
closeButton:SetSize(30, 30)
closeButton:SetPoint("TOPRIGHT", viewer, "TOPRIGHT", -7, -7)
closeButton:SetScript("OnClick", function()
    viewer:Hide()
end)

local texturePool = {{}}
local renderState
local renderBatch
local restartRender
local updateLayout
viewer.currentPage = 1

local function readUInt(state)
    local value = 0
    local factor = 1

    while true do
        local character = string.sub(state.data, state.position, state.position)
        local code = decode[character]
        state.position = state.position + 1
        value = value + (code % 32) * factor
        if code < 32 then
            return value
        end
        factor = factor * 32
    end
end

local function acquireTexture(index)
    local texture = texturePool[index]
    if not texture then
        texture = canvas:CreateTexture(nil, "ARTWORK")
        texturePool[index] = texture
    end
    return texture
end

renderBatch = function()
    if not renderState then
        viewer:SetScript("OnUpdate", nil)
        return
    end

    local batchEnd = math.min(renderState.drawn + 240, renderState.total)
    while renderState.drawn < batchEnd and renderState.position <= #renderState.data do
        local deltaY = readUInt(renderState)
        local encodedX = readUInt(renderState)
        local width = readUInt(renderState)
        local height = readUInt(renderState)
        local level = readUInt(renderState)

        renderState.y = renderState.y + deltaY
        if deltaY == 0 then
            renderState.x = renderState.x + encodedX
        else
            renderState.x = encodedX
        end

        renderState.drawn = renderState.drawn + 1
        local texture = acquireTexture(renderState.drawn)
        texture:ClearAllPoints()
        texture:SetPoint(
            "TOPLEFT",
            canvas,
            "TOPLEFT",
            renderState.x * renderState.scaleX,
            -renderState.y * renderState.scaleY
        )
        texture:SetSize(width * renderState.scaleX, height * renderState.scaleY)
        if level == 1 then
            texture:SetColorTexture(0.20, 0.18, 0.15, 0.46)
        else
            texture:SetColorTexture(0.16, 0.145, 0.125, 0.84)
        end
        texture:Show()
    end

    if renderState.drawn >= renderState.total or renderState.position > #renderState.data then
        loadingText:Hide()
        showVectorPage(PAGE_ORDER[viewer.currentPage])
        viewer:SetScript("OnUpdate", nil)
    else
        local percentage = math.floor((renderState.drawn / renderState.total) * 100)
        loadingText:SetText("正在绘制第 " .. viewer.currentPage .. " 页 · " .. percentage .. "%")
    end
end

restartRender = function(pageIndex)
    if pageIndex < 1 then
        pageIndex = 1
    elseif pageIndex > #PAGE_DATA then
        pageIndex = #PAGE_DATA
    end

    viewer.currentPage = pageIndex
    local sourcePage = PAGE_ORDER[pageIndex]
    hideVectorPages()
    for _, texture in ipairs(texturePool) do
        texture:Hide()
    end

    renderState = {{
        data = PAGE_DATA[sourcePage],
        total = PAGE_COUNTS[sourcePage],
        position = 1,
        x = 0,
        y = 0,
        drawn = 0,
        scaleX = canvas:GetWidth() / SOURCE_WIDTH,
        scaleY = canvas:GetHeight() / SOURCE_HEIGHT,
    }}

    pageIndexText:SetText("第 " .. pageIndex .. " / " .. #PAGE_DATA .. " 页")
    previousButton:SetEnabled(pageIndex > 1)
    nextButton:SetEnabled(pageIndex < #PAGE_DATA)
    loadingText:SetText("正在绘制第 " .. pageIndex .. " 页 · 0%")
    loadingText:Show()
    viewer:SetScript("OnUpdate", renderBatch)
end

local function showPage(index)
    if index == viewer.currentPage or index < 1 or index > #PAGE_DATA then
        return
    end
    if G.PlaySound then
        G.PlaySound(856)
    end
    restartRender(index)
end

updateLayout = function()
    local parentWidth = G.UIParent:GetWidth()
    local parentHeight = G.UIParent:GetHeight()
    local frameHeight = math.min(900, parentHeight * 0.84)
    local imageHeight = frameHeight - 88
    local imageWidth = imageHeight * PAGE_ASPECT
    local maximumFrameWidth = parentWidth * 0.86

    if imageWidth + 28 > maximumFrameWidth then
        imageWidth = maximumFrameWidth - 28
        imageHeight = imageWidth / PAGE_ASPECT
        frameHeight = imageHeight + 88
    end

    canvas:SetSize(imageWidth, imageHeight)
    viewer:SetSize(imageWidth + 28, frameHeight)
    layoutVectorText()

    if renderState then
        restartRender(viewer.currentPage)
    end
end

-- Expose this as a real frame method. Re-running the script calls it with `:`,
-- so the first argument is the frame and the page number is the second one.
viewer.RestartRender = function(_, pageIndex)
    restartRender(pageIndex or viewer.currentPage or 1)
end
viewer.UpdateLayout = updateLayout

previousButton:SetScript("OnClick", function()
    showPage(viewer.currentPage - 1)
end)
nextButton:SetScript("OnClick", function()
    showPage(viewer.currentPage + 1)
end)
viewer:SetScript("OnMouseWheel", function(_, delta)
    if delta < 0 then
        showPage(viewer.currentPage + 1)
    elseif delta > 0 then
        showPage(viewer.currentPage - 1)
    end
end)
viewer:RegisterEvent("DISPLAY_SIZE_CHANGED")
viewer:RegisterEvent("UI_SCALE_CHANGED")
viewer:SetScript("OnEvent", updateLayout)
viewer:SetScript("OnHide", function()
    viewer:SetScript("OnUpdate", nil)
end)
viewer:SetScript("OnShow", function()
    if renderState and renderState.drawn < renderState.total then
        viewer:SetScript("OnUpdate", renderBatch)
    end
end)

if G.UISpecialFrames then
    local registered = false
    for _, frameName in ipairs(G.UISpecialFrames) do
        if frameName == FRAME_NAME then
            registered = true
            break
        end
    end
    if not registered then
        table.insert(G.UISpecialFrames, FRAME_NAME)
    end
end

updateLayout()
restartRender(1)
viewer:Show()
'''


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("page1", type=Path)
    parser.add_argument("page2", type=Path)
    args = parser.parse_args()

    rectangles = [
        merge_rectangles(quantize_page(args.page1, 1)),
        merge_rectangles(quantize_page(args.page2, 2)),
    ]
    encoded = [encode_rectangles(page) for page in rectangles]
    lua = make_lua_ascii_safe(build_lua(encoded, [len(page) for page in rectangles]))

    print("*** Begin Patch")
    print("*** Add File: refs/OvoMenuViewer.lua")
    for line in lua.splitlines():
        print("+" + line)
    print("*** End Patch")


if __name__ == "__main__":
    main()
