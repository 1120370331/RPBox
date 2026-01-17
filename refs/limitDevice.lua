local function CleanupExistingFrames()
    if args._G.simpleControlPanel and args._G.simpleControlPanel:IsShown() then args._G.simpleControlPanel:Hide() end
    if args._G.ClosePetPhoneInterface then args._G.ClosePetPhoneInterface() end
    if inputPanel and inputPanel:IsShown() then inputPanel:Hide() end
    if args._G.numgamePanel and args._G.numgamePanel:IsShown() then args._G.numgamePanel:Hide() end
end

CleanupExistingFrames()

local UI_WIDTH, UI_HEIGHT, BG_COLOR, BORDER_COLOR, UI_POSITION_KEY = 350, 600, {0.1, 0.2, 0.5, 1}, {0, 0.5, 0.3, 1}, "GnomePhone_Position"
local BASE_REAL_YEAR, BASE_WOW_YEAR, CURRENT_WOW_YEAR = 2025, 43, tonumber(date("%Y")) - 2025 + 43
local musicItems, currentPlayingIndex, isAnimating, diceCount, diceFaces, difficulty, modifier, inputPanel = {}, nil, false, 1, 20, 10, 0, nil

local function GetBlackTempleTime()
    local dateTable = date("*t")
    local weekdayNames = {"日", "一", "二", "三", "四", "五", "六"}
    local weekday = "星期"..weekdayNames[dateTable.wday]
    local subzone, zone, location = args._G.GetSubZoneText(), args._G.GetZoneText(), ""
    if subzone ~= "" and subzone ~= zone then location = zone.."-"..subzone else location = zone end
    return {time = string.format("%02d:%02d", dateTable.hour, dateTable.min), date = string.format("黑门%d年%d月%d日%s", CURRENT_WOW_YEAR, dateTable.month, dateTable.day, weekday), location = location}
end

if args._G.phoneUI then args._G.phoneUI:Hide(); args._G.phoneUI = nil end
local phoneFrame = args._G.CreateFrame("Frame", "GnomePhone", args._G.UIParent, "BackdropTemplate")
phoneFrame:SetSize(UI_WIDTH, UI_HEIGHT); phoneFrame:SetFrameStrata("DIALOG"); phoneFrame:EnableMouse(true); phoneFrame:SetMovable(true); args._G.phoneUI = phoneFrame
local savedPosition = getVar(args, "o", UI_POSITION_KEY)
if savedPosition then
    local point, relativeTo, relativePoint, xOfs, yOfs = args._G.strsplit(",", savedPosition)
    if point and xOfs and yOfs then phoneFrame:SetPoint(point, relativeTo or args._G.UIParent, relativePoint or "BOTTOMLEFT", tonumber(xOfs), tonumber(yOfs)) else phoneFrame:SetPoint("CENTER") end
else phoneFrame:SetPoint("CENTER") end
phoneFrame:SetBackdrop({bgFile = "Interface\\AchievementFrame\\UI-Achievement-StatsBackground", edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border", tile = true, tileSize = 356, edgeSize = 16, insets = {left = 4, right = 4, top = 4, bottom = 4}})
phoneFrame:SetBackdropColor(unpack(BG_COLOR)); phoneFrame:SetBackdropBorderColor(unpack(BORDER_COLOR))

local statusBar = args._G.CreateFrame("Frame", nil, phoneFrame); statusBar:SetHeight(30); statusBar:SetPoint("TOPLEFT", 5, -5); statusBar:SetPoint("TOPRIGHT", -5, -5)
local statusBg = statusBar:CreateTexture(nil, "BACKGROUND"); statusBg:SetAllPoints(); statusBg:SetTexture("Interface\\Tooltips\\UI-Tooltip-Background"); statusBg:SetVertexColor(0, 0, 0, 1)
local carrierText = statusBar:CreateFontString(nil, "OVERLAY", "GameFontNormal"); carrierText:SetPoint("LEFT", 10, 0); carrierText:SetText("诺莫瑞根电信"); carrierText:SetTextColor(1, 1, 1); carrierText:SetFont(carrierText:GetFont(), 12, "OUTLINE")
local signalText = statusBar:CreateFontString(nil, "OVERLAY", "GameFontNormal"); signalText:SetPoint("RIGHT", -30, 0); signalText:SetText("天翼3G.il"); signalText:SetTextColor(0, 1, 0); signalText:SetFont(signalText:GetFont(), 12, "OUTLINE")
local closeBtn = args._G.CreateFrame("Button", nil, phoneFrame, "UIPanelCloseButton"); closeBtn:SetSize(30, 30); closeBtn:SetPoint("TOPRIGHT", -5, -5)
closeBtn:SetScript("OnClick", function()
    args._G.PlaySound(3642)
    if args._G.ClosePetPhoneInterface then args._G.ClosePetPhoneInterface() end
    if inputPanel and inputPanel:IsShown() then inputPanel:Hide(); inputPanel = nil end
    if args._G.simpleControlPanel and args._G.simpleControlPanel:IsShown() then args._G.simpleControlPanel:Hide() end
    if args._G.numgamePanel and args._G.numgamePanel:IsShown() then args._G.numgamePanel:Hide() end
    local point, relativeTo, relativePoint, xOfs, yOfs = phoneFrame:GetPoint(1)
    setVar(args, "o", UI_POSITION_KEY, string.format("%s,%s,%s,%d,%d", point, relativeTo and relativeTo:GetName() or "UIParent", relativePoint, math.floor(xOfs), math.floor(yOfs)))
    phoneFrame:Hide()
end)

local mainContentFrame = args._G.CreateFrame("Frame", nil, phoneFrame); mainContentFrame:SetAllPoints()
local homePage = args._G.CreateFrame("Frame", nil, mainContentFrame); homePage:SetAllPoints()
local timeDisplay = args._G.CreateFrame("Frame", nil, homePage); timeDisplay:SetPoint("TOP", statusBar, "BOTTOM", 0, -25); timeDisplay:SetWidth(UI_WIDTH - 30); timeDisplay:SetHeight(100)
local locationText = timeDisplay:CreateFontString(nil, "OVERLAY", "GameFontNormal"); locationText:SetPoint("TOP", 0, 0); locationText:SetTextColor(0.8, 0.8, 0.8); locationText:SetFont(locationText:GetFont(), 18, "OUTLINE")
local bigTimeText = timeDisplay:CreateFontString(nil, "OVERLAY", "GameFontNormal"); bigTimeText:SetPoint("TOP", locationText, "BOTTOM", 0, 0); bigTimeText:SetTextColor(1, 1, 1); bigTimeText:SetFont(bigTimeText:GetFont(), 46, "OUTLINE")
local smallDateText = timeDisplay:CreateFontStrinag(nil, "OVERLAY", "GameFontNormal"); smallDateText:SetPoint("TOP", bigTimeText, "BOTTOM", 0, 0); smallDateText:SetTextColor(0.8, 0.8, 0.8); smallDateText:SetFont(smallDateText:GetFont(), 18, "OUTLINE")

local function UpdateTime()
    local timeData = GetBlackTempleTime()
    locationText:SetText(timeData.location); bigTimeText:SetText(timeData.time); smallDateText:SetText(timeData.date)
    args._G.C_Timer.After(1, UpdateTime)
end
UpdateTime()

local contentFrame = args._G.CreateFrame("Frame", nil, homePage); contentFrame:SetPoint("TOP", timeDisplay, "BOTTOM", 0, -10); contentFrame:SetPoint("LEFT", 10, 0); contentFrame:SetPoint("RIGHT", -30, 0); contentFrame:SetPoint("BOTTOM", -70, 15)
local emotePage = args._G.CreateFrame("Frame", nil, mainContentFrame); emotePage:SetAllPoints(); emotePage:Hide()

local EMOTES = {
    {icon = "ui_embercourt-emoji-happy", text = "你好", command = "hello", workflowID = "emo001"},
    {icon = "ability_hunter_beastsoothe", text = "再见", command = "bye", workflowID = "emo002"},
    {icon = "spell_holy_painsupression", text = "不", command = "no", workflowID = "emo006"},
    {icon = "spell_shadow_soothingkiss", text = "亲吻", command = "kiss", workflowID = "emo007"},
    {icon = "inv_pet_snowman", text = "哭泣", command = "cry", workflowID = "emo008"},
    {icon = "inv_misc_gift_01", text = "感谢", command = "thank", workflowID = "emo009"},
    {icon = "ui_embercourt-emoji-elated", text = "大笑", command = "laugh", workflowID = "emo010"},
    {icon = "inv_gnometoy", text = "咯咯笑", command = "giggle", workflowID = "emo011"},
    {icon = "garrison_building_barracks", text = "敬礼", command = "salute", workflowID = "emo012"},
    {icon = "pandarenracial_innerpeace", text = "鞠躬", command = "bow", workflowID = "emo013"},
    {icon = "inv_misc_surgeonglove_01", text = "鼓掌", command = "applaud", workflowID = "emo014"},
    {icon = "ability_priest_ascendance", text = "跳舞", command = "dance", workflowID = "emo015"},
    {icon = "achievement_profession_fishing_oldmanbarlowned", text = "倚靠", command = "lean", workflowID = "emo016"},
    {icon = "ability_hibernation", text = "躺下", command = "lie", workflowID = "emo079"},
    {icon = "spell_nature_sleep", text = "睡觉", command = "sleep", workflowID = "emo017"},
    {icon = "ui_warbands", text = "坐下", command = "sit", workflowID = "emo018"},
    {icon = "achievement_cooking_pandarianmasterchef", text = "站立", command = "stand", workflowID = "emo019"},
    {icon = "ability_druid_empoweredtouch", text = "指点", command = "point", workflowID = "emo030"},
    {icon = "inv_1115_warrior_victorypose", text = "欢呼", command = "cheer", workflowID = "emo020"},
    {icon = "achievement_bg_tophealer_ab", text = "点头", command = "nod", workflowID = "emo021"},
    {icon = "ships_ability_boardingpartyalliance", text = "招手", command = "wave", workflowID = "emo022"},
    {icon = "inv_drink_08", text = "干杯", command = "cheers", workflowID = "emo147"},
    {icon = "spell_misc_emotionangry", text = "怒气", command = "anger", workflowID = "emo076"},
    {icon = "ability_rogue_quickrecovery", text = "下跪", command = "kneel", workflowID = "emo023"},
    {icon = "ability_seal", text = "乞求", command = "beg", workflowID = "emo024"},
    {icon = "inv_misc_food_11", text = "吃饭", command = "eat", workflowID = "emo025"},
    {icon = "ability_druid_challangingroar", text = "咆哮", command = "roar", workflowID = "emo026"},
    {icon = "inv_pet_raccoon", text = "害羞", command = "shy", workflowID = "emo027"},
    {icon = "spell_shadow_auraofdarkness", text = "退缩", command = "shrink", workflowID = "emo078"},
    {icon = "inv_pet_sleepywilly", text = "惊呆", command = "stun", workflowID = "emo113"},
    {icon = "achievement_boss_grandmagustelestra", text = "疑问", command = "question", workflowID = "emo116"},
    {icon = "ability_monk_blackoutkick", text = "迷惑", command = "confused", workflowID = "emo102"},
    {icon = "achievement_dungeon_heartsbanetriad", text = "祈祷", command = "pray", workflowID = "emo100"},
    {icon = "spell_magic_polymorphchicken", text = "小鸡", command = "chicken", workflowID = "emo028"},
    {icon = "inv_orcclanworg", text = "嗅", command = "sniff", workflowID = "emo060"},
    {icon = "achievement_bg_kill_carrier_opposing_flagroom", text = "笑翻", command = "laughroll", workflowID = "emo094"},
    {icon = "spell_misc_emotionafraid", text = "羞愧", command = "shame", workflowID = "emo103"},
    {icon = "spell_argus_psychic_scarring", text = "考虑", command = "think", workflowID = "emo104"},
    {icon = "inv_stbernarddogpet", text = "耸肩", command = "shrug", workflowID = "emo106"},
    {icon = "spell_nature_strength", text = "强壮", command = "flex", workflowID = "emo029"},
    {icon = "ability_warrior_strengthofarms", text = "粗野", command = "rude", workflowID = "emo031"},
    {icon = "vas_namechange", text = "谈话", command = "talk", workflowID = "emo032"},
    {icon = "pvpcurrency-honor-alliance", text = "为了联盟", command = "forthealliance", workflowID = "emo033"},
    {icon = "spell_deathknight_armyofthedead", text = "敌袭", command = "incoming", workflowID = "emo034"},
    {icon = "spell_misc_petheal", text = "治疗我", command = "healme", workflowID = "emo035"},
    {icon = "inv_pet_exitbattle", text = "投降", command = "surrender", workflowID = "emo083"},
    {icon = "inv_misc_toy_10", text = "火车", command = "train", workflowID = "emo036"},
    {icon = "inv_helm_misc_rose_a_01_orange", text = "示好", command = "flirt", workflowID = "emo037"},
    {icon = "inv_misc_missilelargecluster_red", text = "祝贺", command = "congratulate", workflowID = "emo038"},
    {icon = "inv_encrypted19", text = "笑话", command = "joke", workflowID = "emo039"},
    {icon = "ability_mage_incantersabsorbtion", text = "魔法耗尽", command = "oom", workflowID = "emo041"},
    {icon = "achievement_guildperk_workingovertime", text = "等等", command = "wait", workflowID = "emo040"},
    {icon = "achievement_boss_zuldazar_jaina", text = "低泣", command = "whimper", workflowID = "emo049"},
    {icon = "achievement_boss_maidenofgrief", text = "不置可否", command = "boggle", workflowID = "emo051"},
    {icon = "spell_shaman_blessingoftheeternals", text = "弹鼻子", command = "boop", workflowID = "emo052"},
    {icon = "ability_demonhunter_vengefulretreat2", text = "撤退", command = "retreat", workflowID = "emo054"},
    {icon = "ability_rogue_ghostpirate", text = "跟着我", command = "followme", workflowID = "emo056"},
    {icon = "ability_hunter_snipershot", text = "攻击目标", command = "attacktarget", workflowID = "emo057"},
    {icon = "ability_warrior_charge", text = "冲锋", command = "charge", workflowID = "emo058"},
    {icon = "inv_112_rogue_betweentheeyes", text = "开火", command = "openfire", workflowID = "emo059"},
    {icon = "achievement_bg_xkills_avgraveyard", text = "哀悼", command = "mourn", workflowID = "emo064"},
    {icon = "ability_druid_lunarguidance", text = "屈膝", command = "bendknee", workflowID = "emo072"},
    {icon = "achievement_dungeon_harlansweete", text = "幸灾乐祸", command = "gloat", workflowID = "emo074"},
    {icon = "inv_pet_undeadeagle", text = "呻吟", command = "moan", workflowID = "emo061"},
    {icon = "achievement_halloween_cat_01", text = "呼噜声", command = "snore", workflowID = "emo062"},
    {icon = "ability_druid_cower", text = "哈气", command = "pant", workflowID = "emo063"},
    {icon = "achievement_boss_grandwidowfaerlina", text = "哼", command = "humph", workflowID = "emo065"},
    {icon = "ability_creature_disease_03", text = "打喷嚏", command = "sneeze", workflowID = "emo066"},
    {icon = "spell_nature_polymorph_cow", text = "牛叫", command = "moo", workflowID = "emo067"},
    {icon = "ability_druid_cyclone", text = "安抚", command = "soothe", workflowID = "emo068"},
    {icon = "spell_frost_coldhearted", text = "好冷", command = "cold", workflowID = "emo069"},
    {icon = "inv_boastfulsquire_hd", text = "就绪", command = "ready", workflowID = "emo070"},
    {icon = "inv_g_fishingbobber_05", text = "拍屁股", command = "spank", workflowID = "emo071"},
    {icon = "inv_misc_volatileearth", text = "干渴", command = "thirsty", workflowID = "emo073"},
    {icon = "spell_shadow_charm", text = "微笑", command = "smile", workflowID = "emo075"},
    {icon = "spell_shadow_corpseexplode", text = "怜悯", command = "pity", workflowID = "emo077"},
    {icon = "spell_magic_polymorphrabbit", text = "惊讶", command = "surprise", workflowID = "emo080"},
    {icon = "ability_mage_studentofthemind", text = "戳", command = "poke", workflowID = "emo081"},
    {icon = "ability_mage_burnout", text = "打嗝", command = "burp", workflowID = "emo082"},
    {icon = "inv_pet_otter", text = "抱歉", command = "sorry", workflowID = "emo084"},
    {icon = "spell_mage_overpowered", text = "拥抱", command = "hug", workflowID = "emo085"},
    {icon = "inv_misc_foot_centaur", text = "按摩", command = "massage", workflowID = "emo086"},
    {icon = "inv_helm_misc_pignosemask_a_01", text = "挖鼻孔", command = "picknose", workflowID = "emo087"},
    {icon = "ability_butcher_heavyhanded", text = "敲脑袋", command = "knockhead", workflowID = "emo088"},
    {icon = "inv_misc_primitive_toy01", text = "无聊", command = "bored", workflowID = "emo089"},
    {icon = "inv_misc_food_legion_gooamberblue_drop", text = "流口水", command = "drool", workflowID = "emo090"},
    {icon = "ability_shawaterelemental_split", text = "流汗", command = "sweat", workflowID = "emo091"},
    {icon = "ability_warrior_bloodfrenzy", text = "流血", command = "bleed", workflowID = "emo092"},
    {icon = "inv_helm_flowercrown_a_01_red", text = "爱", command = "love", workflowID = "emo93"},
    {icon = "ability_druid_mangle", text = "挠痒", command = "tickle", workflowID = "emo095"},
    {icon = "ability_mage_potentspirit", text = "皱眉", command = "frown", workflowID = "emo096"},
    {icon = "inv_10_jewelcrafting2_glasslens_color1", text = "打量", command = "sizeup", workflowID = "emo097"},
    {icon = "ability_hunter_murderofcrows", text = "眨眼", command = "wink", workflowID = "emo098"},
    {icon = "ability_warrior_revenge", text = "瞪眼", command = "glare", workflowID = "emo099"},
    {icon = "spell_misc_emotionhappy", text = "窃笑", command = "snicker", workflowID = "emo101"},
    {icon = "ability_druid_swipe", text = "耳光", command = "slap", workflowID = "emo105"},
    {icon = "ui_darkshore_warfront_alliance_archer", text = "聆听", command = "listen", workflowID = "emo107"},
    {icon = "inv_misc_toy_07", text = "胳肢", command = "胳肢", workflowID = "emo108"},
    {icon = "inv_misc_food_meat_raw_08_color02", text = "舔", command = "lick", workflowID = "emo109"},
    {icon = "inv_cooking_90_phantasmalsoufflefries", text = "饥饿", command = "hungry", workflowID = "emo110"},
    {icon = "spell_nature_mentalquickness", text = "马上回来", command = "berightback", workflowID = "emo111"},
    {icon = "ability_druid_ferociousbite", text = "哈欠", command = "yawn", workflowID = "emo112"},
    {icon = "ability_druid_lacerate", text = "警告", command = "warn", workflowID = "emo114"},
    {icon = "spell_shadow_mindsteal", text = "疲惫", command = "tired", workflowID = "emo115"},
    {icon = "ability_hunter_beastwithin", text = "亲昵", command = "intimate", workflowID = "emo117"},
    {icon = "spell_holy_searinglightpriest", text = "掐", command = "pinch", workflowID = "emo118"},
    {icon = "ability_rogue_sinistercalling", text = "绷脸", command = "scowl", workflowID = "emo119"},
    {icon = "inv_misc_noose_01", text = "把脉", command = "checkpulse", workflowID = "emo120"},
    {icon = "spell_holy_fistofjustice", text = "拳击", command = "punch", workflowID = "emo121"},
    {icon = "spell_shadow_painandsuffering", text = "后悔", command = "regret", workflowID = "emo122"},
    {icon = "ability_cheapshot", text = "翻白眼", command = "roll eyes", workflowID = "emo123"},
    {icon = "inv_misc_comb_01", text = "摸头发", command = "touslehair", workflowID = "emo124"},
    {icon = "spell_misc_emotionsad", text = "沮丧", command = "frustrated", workflowID = "emo125"},
    {icon = "inv_professions_inscription_scribesmagnifyingglass_silver", text = "搜找", command = "search", workflowID = "emo126"},
    {icon = "achievement_bg_most_damage_killingblow_dieleast", text = "魅力", command = "charm", workflowID = "emo127"},
    {icon = "inv_summerfest_groundflower", text = "信号", command = "signal", workflowID = "emo128"},
    {icon = "spell_holy_silence", text = "嘘", command = "shush", workflowID = "emo129"},
    {icon = "ability_priest_heavanlyvoice", text = "女声哼唱", command = "humfemale", workflowID = "emo130"},
    {icon = "inv_gauntlets_16", text = "响指", command = "snapfingers", workflowID = "emo131"},
    {icon = "inv_tradeskillitem_lessersorcererswater", text = "吐口水", command = "spit", workflowID = "emo132"},
    {icon = "ability_rogue_masterofsubtlety", text = "凝视", command = "gaze", workflowID = "emo133"},
    {icon = "achievement_explore_argus", text = "出发", command = "go", workflowID = "emo134"},
    {icon = "inv_misc_head_tuskarr", text = "头疼", command = "headache", workflowID = "emo135"},
    {icon = "ability_druid_catformattack", text = "击掌", command = "highfive", workflowID = "emo136"},
    {icon = "achievement_raid_torghast_sylvanaswindrunner", text = "鄙视", command = "despise", workflowID = "emo137"},
    {icon = "spell_holy_layonhands", text = "握住", command = "hold", workflowID = "emo138"},
    {icon = "ability_bullrush", text = "催促", command = "hurry", workflowID = "emo139"},
    {icon = "inv_helm_armor_gnomish_c_01_green", text = "新点子", command = "newidea", workflowID = "emo140"},
    {icon = "spell_shadow_seduction", text = "嫉妒", command = "envy", workflowID = "emo141"},
    {icon = "inv_misc_luckymoneyenvelope", text = "祝好运", command = "goodluck", workflowID = "emo142"},
    {icon = "inv_pet_cats_orangetabbycat", text = "猫叫", command = "meow", workflowID = "emo143"},
    {icon = "ability_rogue_disguise", text = "捂脸", command = "coverface", workflowID = "emo145"},
    {icon = "ability_pvp_hardiness", text = "鼓舞", command = "inspire", workflowID = "emo146"},
    {icon = "ui_mission_itemupgrade", text = "升级", command = "levelup", workflowID = "emo148"},
    {icon = "inv_misc_ear_human_02", text = "捂耳朵", command = "coverears", workflowID = "emo149"},
    {icon = "spell_nature_sicklypolymorph", text = "咳嗽", command = "cough", workflowID = "emo150"},
    {icon = "inv_babypig", text = "学猪叫", command = "oink", workflowID = "emo151"},
    {icon = "inv_sloth", text = "迷人", command = "enchanting", workflowID = "emo152"},
    {icon = "ability_parry", text = "发起挑战", command = "challenge", workflowID = "emo153"},
    {icon = "inv_drink_22", text = "倒酒", command = "daojiu", workflowID = "emo003"},
    {icon = "achievement_guildperk_bartering", text = "付钱", command = "fuqian", workflowID = "emo004"},
    {icon = "inv_misc_azsharacoin", text = "抛硬币", command = "paoyingbi", workflowID = "emo005"},
    {icon = "inv_misc_dice_01", text = "Roll点", command = "roll", workflowID = "emo055"},
    {icon = "achievement_guildperk_mountup", text = "坐骑特技", command = "mounttrick", workflowID = "emo144"},
    {icon = "inv_pet_broom", text = "扫地", command = "sweep", workflowID = "emo154"},
    {icon = "inv_weapon_rifle_10", text = "发令枪", command = "startinggun", workflowID = "emo156"},
    {icon = "inv_misc_trinketpanda_01", text = "测运势", command = "fortunetest", workflowID = "emo157"},
    {icon = "inv_checkered_flag", text = "赛场助威", command = "matchcheer", workflowID = "emo158"},
    {icon = "inv_misc_bomb_05", text = "装炸弹", command = "setbomb", workflowID = "emo159"},
    {icon = "inv_engineering_90_remote", text = "玩终端", command = "playphone", workflowID = "emo160"},
    {icon = "spell_holy_borrowedtime", text = "看时间", command = "whattime", workflowID = "emo161"}
}

local FAVORITE_EMOTES, MAX_FAVORITES, EMOTE_COOLDOWNS, EMOTE_BUTTONS = {}, 12, {}, {}
local EMOTE_COOLDOWN_DURATIONS = {
    ["emo028"] = 30, ["emo036"] = 30, ["emo042"] = 30, ["emo004"] = 20, ["emo005"] = 20, ["emo043"] = 20, ["emo055"] = 20, ["emo062"] = 30, ["emo063"] = 30, ["emo065"] = 30, ["emo067"] = 30, ["emo071"] = 20,
    ["emo088"] = 20, ["emo093"] = 20, ["emo105"] = 30, ["emo121"] = 30, ["emo130"] = 30, ["emo132"] = 30, ["emo140"] = 20, ["emo143"] = 30, ["emo148"] = 30, ["emo150"] = 20, ["emo151"] = 30, ["emo154"] = 30,
    ["emo155"] = 30, ["emo156"] = 60, ["emo158"] = 60, ["emo159"] = 60, ["emo160"] = 20, ["emo161"] = 60,
}

if not getVar(args, "o", "favorite_emotes") then
    for i=1, math.min(6, #EMOTES) do FAVORITE_EMOTES[i] = i end
    setVar(args, "o", "favorite_emotes", table.concat(FAVORITE_EMOTES, ","))
else
    local saved = getVar(args, "o", "favorite_emotes")
    if saved then for i, id in ipairs({args._G.strsplit(",", saved)}) do FAVORITE_EMOTES[i] = tonumber(id) end end
end

local scrollFrame = args._G.CreateFrame("ScrollFrame", nil, emotePage, "UIPanelScrollFrameTemplate")
scrollFrame:SetPoint("TOP", statusBar, "BOTTOM", 0, -5); scrollFrame:SetPoint("BOTTOM", phoneFrame, "BOTTOM", 0, 80); scrollFrame:SetWidth(UI_WIDTH - 20); scrollFrame:SetFrameLevel(10)
local scrollBar = scrollFrame.ScrollBar; scrollBar:ClearAllPoints(); scrollBar:SetPoint("TOPLEFT", scrollFrame, "TOPRIGHT", -15, -16); scrollBar:SetPoint("BOTTOMLEFT", scrollFrame, "BOTTOMRIGHT", -12, 16)
local contentHeight = 120 + math.ceil(#EMOTES/6) * 60; local emoteContent = args._G.CreateFrame("Frame", nil, scrollFrame); emoteContent:SetSize(UI_WIDTH - 40, contentHeight); scrollFrame:SetScrollChild(emoteContent)
local favoriteLabel = emoteContent:CreateFontString(nil, "OVERLAY", "GameFontNormal"); favoriteLabel:SetPoint("TOP", 0, -5); favoriteLabel:SetText("偏好表情"); favoriteLabel:SetTextColor(1, 0.3, 0.5); favoriteLabel:SetFont(favoriteLabel:GetFont(), 16, "OUTLINE")
local hintText1 = emoteContent:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); hintText1:SetPoint("TOP", favoriteLabel, "BOTTOM", 0, 0); hintText1:SetText("(右键点击图标设置偏好)"); hintText1:SetTextColor(0.8, 0.8, 0.8); hintText1:SetFont(hintText1:GetFont(), 12, "OUTLINE")
local favoriteFrame = args._G.CreateFrame("Frame", nil, emoteContent); favoriteFrame:SetPoint("TOP", hintText1, "BOTTOM", 0, -5); favoriteFrame:SetWidth(UI_WIDTH - 40); favoriteFrame:SetHeight(120)
local allLabel = emoteContent:CreateFontString(nil, "OVERLAY", "GameFontNormal"); allLabel:SetPoint("TOP", favoriteFrame, "BOTTOM", 0, -30); allLabel:SetText("全部表情"); allLabel:SetTextColor(0, 1, 0.3); allLabel:SetFont(allLabel:GetFont(), 16, "OUTLINE")
local hintText = emoteContent:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); hintText:SetPoint("TOP", allLabel, "BOTTOM", 0, 0); hintText:SetText("(黄字为拥有人物动作的表情)"); hintText:SetTextColor(0.8, 0.8, 0.8); hintText:SetFont(hintText:GetFont(), 12, "OUTLINE")
local allEmoteFrame = args._G.CreateFrame("Frame", nil, emoteContent); allEmoteFrame:SetPoint("TOP", hintText, "BOTTOM", 0, -5); allEmoteFrame:SetWidth(UI_WIDTH - 40); allEmoteFrame:SetHeight(math.ceil(#EMOTES/6) * 60)

local YELLOW_TEXT_EMOTES = {"你好", "再见", "不", "亲吻", "哭泣", "感谢", "大笑", "咯咯笑", "敬礼", "鞠躬", "鼓掌", "跳舞", "倚靠", "躺下", "睡觉", "坐下", "站立", "指点", "欢呼", "点头", "招手", "干杯", "怒气", "下跪", "乞求", "吃饭", "咆哮", "害羞", "退缩", "惊呆", "疑问", "迷惑", "祈祷", "小鸡", "嗅", "笑翻", "羞愧", "考虑", "耸肩", "强壮", "粗野", "谈话", "为了联盟", "敌袭", "治疗我", "投降", "火车", "示好", "祝贺", "笑话", "魔法耗尽", "等等", "低泣", "不置可否", "弹鼻子", "撤退", "跟着我", "攻击目标", "冲锋", "开火", "哀悼", "屈膝", "幸灾乐祸"}
local YELLOW_TEXT_LOOKUP = {}; for _, text in ipairs(YELLOW_TEXT_EMOTES) do YELLOW_TEXT_LOOKUP[text] = true end

local function UpdateEmoteCooldowns(emoteCommand, workflowID)
    if not EMOTE_BUTTONS[emoteCommand] then return end
    local cooldownDuration = EMOTE_COOLDOWN_DURATIONS[workflowID] or 5
    for _, btn in pairs(EMOTE_BUTTONS[emoteCommand]) do
        for _, region in pairs({btn:GetRegions()}) do if region:GetObjectType() == "Cooldown" then region:Hide() end end
        if EMOTE_COOLDOWNS[emoteCommand] and EMOTE_COOLDOWNS[emoteCommand] > args._G.GetTime() then
            local cooldown = args._G.CreateFrame("Cooldown", nil, btn, "CooldownFrameTemplate"); cooldown:SetAllPoints(); cooldown:SetDrawEdge(false)
            args._G.CooldownFrame_Set(cooldown, EMOTE_COOLDOWNS[emoteCommand] - cooldownDuration, cooldownDuration, true)
        end
    end
end

local function UpdateEmoteDisplay()
    EMOTE_BUTTONS = {}
    for _, child in ipairs({favoriteFrame:GetChildren()}) do child:Hide() end
    for _, child in ipairs({allEmoteFrame:GetChildren()}) do child:Hide() end
    for i, emoteIndex in ipairs(FAVORITE_EMOTES) do
        if i > MAX_FAVORITES then break end
        local row, col = math.floor((i-1)/6), (i-1) % 6
        local emote = EMOTES[emoteIndex]
        if emote then
            local btn = args._G.CreateFrame("Button", nil, favoriteFrame); btn:SetSize(50, 60); btn:SetPoint("TOPLEFT", col * (50 + 2), -row * (60 + 5)); btn:RegisterForClicks("LeftButtonUp", "RightButtonUp")
            if not EMOTE_BUTTONS[emote.command] then EMOTE_BUTTONS[emote.command] = {} end; table.insert(EMOTE_BUTTONS[emote.command], btn)
            local highlight = btn:CreateTexture(nil, "HIGHLIGHT"); highlight:SetAllPoints(); highlight:SetTexture("Interface\\Buttons\\UI-Common-MouseHilight"); highlight:SetBlendMode("ADD"); highlight:SetAlpha(0.5); btn:SetHighlightTexture(highlight)
            local icon = btn:CreateTexture(nil, "ARTWORK"); icon:SetSize(40, 40); icon:SetPoint("TOP", 0, -5); icon:SetTexture("Interface\\Icons\\"..emote.icon)
            local text = btn:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); text:SetPoint("TOP", icon, "BOTTOM", 0, -2); text:SetText(emote.text)
            if YELLOW_TEXT_LOOKUP[emote.text] then text:SetTextColor(1, 1, 0) else text:SetTextColor(1, 1, 1) end; text:SetFont(text:GetFont(), 12)
            local star = btn:CreateTexture(nil, "OVERLAY"); star:SetSize(12, 12); star:SetPoint("TOPRIGHT", 2, 2); star:SetTexture("Interface\\Common\\ReputationStar"); star:SetTexCoord(0, 0.5, 0, 0.5)
            local cooldownDuration = EMOTE_COOLDOWN_DURATIONS[emote.workflowID] or 5
            if EMOTE_COOLDOWNS[emote.command] and EMOTE_COOLDOWNS[emote.command] > args._G.GetTime() then
                local cooldown = args._G.CreateFrame("Cooldown", nil, btn, "CooldownFrameTemplate"); cooldown:SetAllPoints(); cooldown:SetDrawEdge(false)
                args._G.CooldownFrame_Set(cooldown, EMOTE_COOLDOWNS[emote.command] - cooldownDuration, cooldownDuration, true)
            end
            btn:SetScript("OnClick", function(self, button)
    if isAnimating then return end
    if button == "LeftButton" then
        isAnimating = true; icon:SetAlpha(0.7); icon:SetSize(35, 35)
        local cooldownDuration = EMOTE_COOLDOWN_DURATIONS[emote.workflowID] or 5
        if EMOTE_COOLDOWNS[emote.command] and EMOTE_COOLDOWNS[emote.command] > args._G.GetTime() then
            args._G.PlaySound(847); args._G.C_Timer.After(0.1, function() icon:SetAlpha(1); icon:SetSize(40, 40); isAnimating = false end); return
        end
        EMOTE_COOLDOWNS[emote.command] = args._G.GetTime() + cooldownDuration; UpdateEmoteCooldowns(emote.command, emote.workflowID)
        
        if emote.command == "lean" then
            local target = args._G.UnitName("target")
            if target then 
                args._G.DoEmote("lean", target)
            else 
                args._G.DoEmote("lean")
            end
        elseif emote.workflowID then 
            effect("run_workflow", args, "o", emote.workflowID) 
        end
        
        args._G.PlaySound(624); args._G.C_Timer.After(0.1, function() icon:SetAlpha(1); icon:SetSize(40, 40); isAnimating = false end)
    elseif button == "RightButton" then
        args._G.PlaySound(63)
        for j, index in ipairs(FAVORITE_EMOTES) do if index == emoteIndex then table.remove(FAVORITE_EMOTES, j); break end end
        setVar(args, "o", "favorite_emotes", table.concat(FAVORITE_EMOTES, ",")); star:SetAlpha(0); args._G.UIFrameFadeOut(star, 0.3, 1, 0); args._G.C_Timer.After(0.3, UpdateEmoteDisplay)
    end
end)
            btn:SetScript("OnMouseDown", function(self, button) if button == "RightButton" then return true end end)
        end
    end
    for i, emote in ipairs(EMOTES) do
        local col, row = (i-1) % 6, math.floor((i-1)/6)
        local btn = args._G.CreateFrame("Button", nil, allEmoteFrame); btn:SetSize(50, 60); btn:SetPoint("TOPLEFT", col * (50 + 2), -row * (60 + 5)); btn:RegisterForClicks("LeftButtonUp", "RightButtonUp")
        if not EMOTE_BUTTONS[emote.command] then EMOTE_BUTTONS[emote.command] = {} end; table.insert(EMOTE_BUTTONS[emote.command], btn)
        local highlight = btn:CreateTexture(nil, "HIGHLIGHT"); highlight:SetAllPoints(); highlight:SetTexture("Interface\\Buttons\\UI-Common-MouseHilight"); highlight:SetBlendMode("ADD"); highlight:SetAlpha(0.5); btn:SetHighlightTexture(highlight)
        local icon = btn:CreateTexture(nil, "ARTWORK"); icon:SetSize(40, 40); icon:SetPoint("TOP", 0, -5); icon:SetTexture("Interface\\Icons\\"..emote.icon)
        local text = btn:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); text:SetPoint("TOP", icon, "BOTTOM", 0, -2); text:SetText(emote.text)
        if YELLOW_TEXT_LOOKUP[emote.text] then text:SetTextColor(1, 1, 0) else text:SetTextColor(1, 1, 1) end; text:SetFont(text:GetFont(), 12)
        local isFavorite = false; for _, favIndex in ipairs(FAVORITE_EMOTES) do if favIndex == i then isFavorite = true; break end end
        if isFavorite then
            local star = btn:CreateTexture(nil, "OVERLAY"); star:SetSize(12, 12); star:SetPoint("TOPRIGHT", 2, 2); star:SetTexture("Interface\\Common\\ReputationStar"); star:SetTexCoord(0, 0.5, 0, 0.5)
        end
        local cooldownDuration = EMOTE_COOLDOWN_DURATIONS[emote.workflowID] or 5
        if EMOTE_COOLDOWNS[emote.command] and EMOTE_COOLDOWNS[emote.command] > args._G.GetTime() then
            local cooldown = args._G.CreateFrame("Cooldown", nil, btn, "CooldownFrameTemplate"); cooldown:SetAllPoints(); cooldown:SetDrawEdge(false)
            args._G.CooldownFrame_Set(cooldown, EMOTE_COOLDOWNS[emote.command] - cooldownDuration, cooldownDuration, true)
        end
        btn:SetScript("OnClick", function(self, button)
    if isAnimating then return end
    if button == "LeftButton" then
        isAnimating = true; icon:SetAlpha(0.7); icon:SetSize(35, 35)
        local cooldownDuration = EMOTE_COOLDOWN_DURATIONS[emote.workflowID] or 5
        if EMOTE_COOLDOWNS[emote.command] and EMOTE_COOLDOWNS[emote.command] > args._G.GetTime() then
            args._G.PlaySound(847); args._G.C_Timer.After(0.1, function() icon:SetAlpha(1); icon:SetSize(40, 40); isAnimating = false end); return
        end
        EMOTE_COOLDOWNS[emote.command] = args._G.GetTime() + cooldownDuration; UpdateEmoteCooldowns(emote.command, emote.workflowID)
        
        if emote.command == "lean" then
            local target = args._G.UnitName("target")
            if target then 
                args._G.DoEmote("lean", target)
            else 
                args._G.DoEmote("lean")
            end
        elseif emote.workflowID then 
            effect("run_workflow", args, "o", emote.workflowID) 
        else 
            args._G.DoEmote(emote.command:gsub("/", "")) 
        end
        
        args._G.PlaySound(624); args._G.C_Timer.After(0.1, function() icon:SetAlpha(1); icon:SetSize(40, 40); isAnimating = false end)
    elseif button == "RightButton" then
                local isFavorite, favoriteIndex = false, nil
                for j, index in ipairs(FAVORITE_EMOTES) do if index == i then isFavorite = true; favoriteIndex = j; break end end
                if isFavorite then args._G.PlaySound(63); table.remove(FAVORITE_EMOTES, favoriteIndex)
                else
                    args._G.PlaySound(11788)
                    if #FAVORITE_EMOTES >= MAX_FAVORITES then
                        args._G.PlaySound(847); args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, "最多只能设置12个偏好表情", {r=1.0, g=0.1, b=0.1}); return
                    end
                    table.insert(FAVORITE_EMOTES, 1, i)
                end
                setVar(args, "o", "favorite_emotes", table.concat(FAVORITE_EMOTES, ",")); UpdateEmoteDisplay()
            end
        end)
        btn:SetScript("OnMouseDown", function(self, button) if button == "RightButton" then return true end end)
    end
end

UpdateEmoteDisplay()
scrollFrame:EnableMouseWheel(true)
scrollFrame:SetScript("OnMouseWheel", function(self, delta)
    local currentPos = scrollBar:GetValue(); local minPos, maxPos = scrollBar:GetMinMaxValues()
    if delta < 0 and currentPos < maxPos then scrollBar:SetValue(math.min(currentPos + 80, maxPos)) elseif delta > 0 and currentPos > minPos then scrollBar:SetValue(math.max(currentPos - 80, minPos)) end
end)

local habitPage = args._G.CreateFrame("Frame", nil, mainContentFrame); habitPage:SetAllPoints(); habitPage:Hide()
local middlePanel = args._G.CreateFrame("Frame", nil, habitPage); middlePanel:SetPoint("TOP", statusBar, "BOTTOM", 0, -10); middlePanel:SetPoint("BOTTOM", phoneFrame, "BOTTOM", 0, 90); middlePanel:SetWidth(UI_WIDTH - 20)
local descText = middlePanel:CreateFontString(nil, "OVERLAY", "GameFontHighlight"); descText:SetPoint("TOP", 0, -10); descText:SetText("当你使用本机的发言功能时，会自动带上已设置的口癖内容。\n\n范例A：[恶魔语]（前缀）你们这是自寻死路！（正文）*嚣张的语气*（后缀）\n\n范例B：啊咧咧？（前缀）这个真的是给我的吗？（正文）啾咪~（后缀）"); descText:SetTextColor(0.8, 0.8, 1); descText:SetWidth(UI_WIDTH - 40); descText:SetJustifyH("LEFT")
local prefixLabel = middlePanel:CreateFontString(nil, "OVERLAY", "GameFontNormal"); prefixLabel:SetPoint("TOP", descText, "BOTTOM", 0, -30); prefixLabel:SetText("前缀:"); prefixLabel:SetTextColor(1, 1, 1)
local prefixText = middlePanel:CreateFontString(nil, "OVERLAY", "GameFontHighlight"); prefixText:SetPoint("TOP", prefixLabel, "BOTTOM", 0, -5); prefixText:SetText("加载中..."); prefixText:SetTextColor(1, 1, 0.5); prefixText:SetWidth(UI_WIDTH - 40); prefixText:SetJustifyH("CENTER")
local prefixEditBtn = args._G.CreateFrame("Button", nil, middlePanel, "UIPanelButtonTemplate"); prefixEditBtn:SetSize(80, 25); prefixEditBtn:SetPoint("TOP", prefixText, "BOTTOM", 0, -5); prefixEditBtn:SetText("修改")
prefixEditBtn:SetScript("OnClick", function() args._G.PlaySound(624); effect("run_workflow", args, "o", "ask4") end)
local suffixLabel = middlePanel:CreateFontString(nil, "OVERLAY", "GameFontNormal"); suffixLabel:SetPoint("TOP", prefixEditBtn, "BOTTOM", 0, -20); suffixLabel:SetText("后缀:"); suffixLabel:SetTextColor(1, 1, 1)
local suffixText = middlePanel:CreateFontString(nil, "OVERLAY", "GameFontHighlight"); suffixText:SetPoint("TOP", suffixLabel, "BOTTOM", 0, -5); suffixText:SetText("加载中..."); suffixText:SetTextColor(1, 1, 0.5); suffixText:SetWidth(UI_WIDTH - 40); suffixText:SetJustifyH("CENTER")
local suffixEditBtn = args._G.CreateFrame("Button", nil, middlePanel, "UIPanelButtonTemplate"); suffixEditBtn:SetSize(80, 25); suffixEditBtn:SetPoint("TOP", suffixText, "BOTTOM", 0, -5); suffixEditBtn:SetText("修改")
suffixEditBtn:SetScript("OnClick", function() args._G.PlaySound(624); effect("run_workflow", args, "o", "ask5") end)
local clearBtn = args._G.CreateFrame("Button", nil, middlePanel, "UIPanelButtonTemplate"); clearBtn:SetSize(150, 30); clearBtn:SetPoint("TOP", suffixEditBtn, "BOTTOM", 0, -20); clearBtn:SetText("一键清除口癖"); clearBtn:SetNormalFontObject("GameFontNormalLarge"); clearBtn:SetHighlightFontObject("GameFontHighlightLarge"); clearBtn:GetFontString():SetTextColor(1, 1, 1)
clearBtn:SetScript("OnClick", function() args._G.PlaySound(624); effect("run_workflow", args, "o", "口癖清除") end)
local clearBtnBorder = clearBtn:CreateTexture(nil, "BACKGROUND"); clearBtnBorder:SetAllPoints(); clearBtnBorder:SetTexture("Interface\\DialogFrame\\UI-DialogBox-Border"); clearBtnBorder:SetVertexColor(0, 0, 0, 0)

local function UpdateHabitDisplay()
    local currentPrefix, currentSuffix = getVar(args, "o", "前缀") or "", getVar(args, "o", "后缀") or ""
    prefixText:SetText(currentPrefix ~= "" and currentPrefix or "无"); suffixText:SetText(currentSuffix ~= "" and currentSuffix or "无")
end

local habitUpdateTimer = args._G.C_Timer.NewTicker(0.1, UpdateHabitDisplay)

local musicPage = args._G.CreateFrame("Frame", nil, mainContentFrame); musicPage:SetAllPoints(); musicPage:Hide()

local MUSIC_LIST = {
    {title = "海妖哀歌", workflowID = "mus01"}, {title = "青之竹", workflowID = "mus02"}, {title = "分秒必争！诺莫瑞根", workflowID = "mus03"}, {title = "刘浪吟", workflowID = "mus04"},
    {title = "暗黑破坏神周年纪念（吉他）", workflowID = "mus05"}, {title = "狮王之傲旅店", workflowID = "mus06"}, {title = "多恩的风", workflowID = "mus07"}, {title = "古神（克熙尔·风暴之歌）", workflowID = "mus08"},
    {title = "伯拉勒斯小调", workflowID = "mus09"}, {title = "冬幕节快乐", workflowID = "mus10"}, {title = "夜歌", workflowID = "mus11"}, {title = "假日舞曲", workflowID = "mus12"},
    {title = "为了洛丹伦", workflowID = "mus13"}, {title = "高等精灵之墓", workflowID = "mus14"}, {title = "灰熊丘陵林间风笛（需准备20秒）", workflowID = "mus15"}, {title = "安度因的花", workflowID = "mus16"},
    {title = "干杯！地精酒馆", workflowID = "mus17"}, {title = "不朽寒霜", workflowID = "mus18"}, {title = "迪门修斯之战", workflowID = "mus19"}, {title = "生态穹顶", workflowID = "mus20"},
    {title = "上层精灵的挽歌", workflowID = "mus21"}, {title = "莫高雷大草原", workflowID = "mus22"}, {title = "海的女儿人声清唱（需准备20秒）", workflowID = "mus23"}, {title = "纳斯利亚圆舞曲", workflowID = "mus24"},
    {title = "部落的力量", workflowID = "mus25"}, {title = "旅居灰熊丘陵", workflowID = "mus26"}, {title = "王子…凯旋？", workflowID = "mus27"}, {title = "这里是老子的加乐宫！", workflowID = "mus28"},
    {title = "塔拉多漫步", workflowID = "mus29"}, {title = "塞壬之声", workflowID = "mus30"}, {title = "为了钢铁部落", workflowID = "mus31"}, {title = "阿什兰风暴之盾", workflowID = "mus32"},
    {title = "雄狮之眠（人声颂唱）", workflowID = "mus33"}, {title = "海盗往事", workflowID = "mus34"}, {title = "祈愿泰达希尔", workflowID = "mus35"}, {title = "祖达萨百商集市", workflowID = "mus36"},
    {title = "旧世年代", workflowID = "mus37"}, {title = "燃烧的远征", workflowID = "mus38"}, {title = "巫妖王之怒", workflowID = "mus39"}, {title = "大地的裂变", workflowID = "mus40"},
    {title = "熊猫人之谜", workflowID = "mus41"}, {title = "德拉诺之王", workflowID = "mus42"}, {title = "军团再临", workflowID = "mus43"}, {title = "争霸艾泽拉斯", workflowID = "mus44"},
    {title = "暗影国度", workflowID = "mus45"}, {title = "巨龙时代", workflowID = "mus46"}, {title = "无敌", workflowID = "mus47"}
}

local savedCustomMusic = getVar(args, "o", "custom_music") or ""
if savedCustomMusic and savedCustomMusic ~= "" then
    local customEntries = {args._G.strsplit("|", savedCustomMusic)}
    for i = #customEntries, 1, -1 do
        local entry = customEntries[i]
        if entry and entry ~= "" then
            local id, title = args._G.strsplit(":", entry, 2)
            if id and title and id ~= "" and title ~= "" then
                local displayTitle = title
                if not title:match(" %-%d+$") then
                    displayTitle = string.format("%s - %s", title, id)
                end
                table.insert(MUSIC_LIST, 1, {title = displayTitle, workflowID = id, isCustom = true, originalTitle = title})
            end
        end
    end
end

local musicScrollFrame = args._G.CreateFrame("ScrollFrame", nil, musicPage, "UIPanelScrollFrameTemplate")
musicScrollFrame:SetPoint("TOP", statusBar, "BOTTOM", 0, -5)
musicScrollFrame:SetPoint("BOTTOM", phoneFrame, "BOTTOM", 0, 80)
musicScrollFrame:SetWidth(UI_WIDTH - 20)
musicScrollFrame:SetFrameLevel(5)

local musicScrollBar = musicScrollFrame.ScrollBar
musicScrollBar:ClearAllPoints()
musicScrollBar:SetPoint("TOPLEFT", musicScrollFrame, "TOPRIGHT", -15, -16)
musicScrollBar:SetPoint("BOTTOMLEFT", musicScrollFrame, "BOTTOMRIGHT", -12, 16)

local musicItems = {}
local inputPanel
local currentPlayingIndex = nil

local function ShowAddMusicInput()
    if inputPanel and inputPanel:IsShown() then return end
    
    inputPanel = args._G.CreateFrame("Frame", nil, args._G.UIParent, "BackdropTemplate")
    inputPanel:SetSize(300, 220)
    inputPanel:SetPoint("CENTER", 0, 0)
    inputPanel:SetFrameStrata("DIALOG")
    inputPanel:SetFrameLevel(1000)
    inputPanel:EnableMouse(true)
    
    inputPanel:SetBackdrop({
        bgFile = "Interface\\Tooltips\\UI-Tooltip-Background",
        edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border",
        edgeSize = 8,
        insets = {left = 4, right = 4, top = 4, bottom = 4}
    })
    inputPanel:SetBackdropColor(0, 0, 0, 1)
    inputPanel:SetBackdropBorderColor(0.5, 0.5, 0.5, 1)
    
    local titleText = inputPanel:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    titleText:SetPoint("TOP", 0, -15)
    titleText:SetText("添加更多音乐")
    titleText:SetTextColor(1, 1, 0)
    local font, size = titleText:GetFont()
    titleText:SetFont(font, size + 1, "OUTLINE")
    
    local idLabel = inputPanel:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    idLabel:SetPoint("TOPLEFT", 20, -40)
    idLabel:SetText("音乐数字代码(例：2180808)")
    idLabel:SetTextColor(0.8, 0.8, 0.8)
    
    local idInput = args._G.CreateFrame("EditBox", nil, inputPanel, "InputBoxTemplate")
    idInput:SetSize(250, 30)
    idInput:SetPoint("TOPLEFT", 28, -60)
    idInput:SetFontObject("GameFontNormal")
    idInput:SetTextInsets(5, 5, 5, 5)
    idInput:EnableMouse(true)
    idInput:SetAutoFocus(false)
    idInput:SetText("")
    

    idInput:SetScript("OnTextChanged", function(self)
        local text = self:GetText()

        text = text:gsub("%D", "")
        self:SetText(text)
    end)
    
    local nameLabel = inputPanel:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    nameLabel:SetPoint("TOPLEFT", 20, -100)
    nameLabel:SetText("自定义名称(例：麦卡贡点唱机)")
    nameLabel:SetTextColor(0.8, 0.8, 0.8)
    
    local nameInput = args._G.CreateFrame("EditBox", nil, inputPanel, "InputBoxTemplate")
    nameInput:SetSize(250, 30)
    nameInput:SetPoint("TOPLEFT", 28, -120)
    nameInput:SetFontObject("GameFontNormal")
    nameInput:SetTextInsets(5, 5, 5, 5)
    nameInput:EnableMouse(true)
    nameInput:SetAutoFocus(false)
    nameInput:SetText("")
    
    local btnContainer = args._G.CreateFrame("Frame", nil, inputPanel)
    btnContainer:SetSize(200, 30)
    btnContainer:SetPoint("TOP", nameInput, "BOTTOM", 0, -10)
    
    local confirmBtn = args._G.CreateFrame("Button", nil, btnContainer, "UIPanelButtonTemplate")
    confirmBtn:SetSize(80, 25)
    confirmBtn:SetPoint("RIGHT", btnContainer, "CENTER", -10, 0)
    confirmBtn:SetText("确定")
    
    confirmBtn:SetScript("OnClick", function()
    local musicId = idInput:GetText()
    local musicName = nameInput:GetText()
    
    if musicId and musicId ~= "" and musicName and musicName ~= "" then
        local cleanName = musicName:gsub(" %-%d+$", "")
        local formattedName = string.format("%s - %s", cleanName, musicId)
        
        table.insert(MUSIC_LIST, 1, {
            title = formattedName,
            workflowID = musicId,
            isCustom = true,
            originalTitle = cleanName
        })
        
        local customMusicEntries = {}
        for _, music in ipairs(MUSIC_LIST) do
            if music.isCustom then
                table.insert(customMusicEntries, music.workflowID .. ":" .. music.originalTitle)
            end
        end
        setVar(args, "o", "custom_music", table.concat(customMusicEntries, "|"))
        
        RebuildMusicList()
        args._G.PlaySound(856)
    else
        args._G.PlaySound(847)
        args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, "请输入完整信息", {r=1, g=0, b=0})
    end
    
    inputPanel:Hide()
    inputPanel = nil
end)

    
    local cancelBtn = args._G.CreateFrame("Button", nil, btnContainer, "UIPanelButtonTemplate")
    cancelBtn:SetSize(80, 25)
    cancelBtn:SetPoint("LEFT", btnContainer, "CENTER", 5, 0)
    cancelBtn:SetText("取消")
    
    cancelBtn:SetScript("OnClick", function()
        inputPanel:Hide()
        inputPanel = nil
    end)
    
    idInput:SetScript("OnEnterPressed", function() nameInput:SetFocus() end)
    nameInput:SetScript("OnEnterPressed", function() confirmBtn:Click() end)
    inputPanel:SetScript("OnHide", function() inputPanel = nil end)
    
    args._G.PlaySound(624)
    idInput:SetFocus()
end

local function CreateAddMusicItem(parent, index)
    local item = args._G.CreateFrame("Button", nil, parent)
    item:SetSize(UI_WIDTH - 10, 40)
    item:SetPoint("TOPLEFT", 0, -(index-1)*50)
    
    local bg = item:CreateTexture(nil, "BACKGROUND")
    bg:SetAllPoints()
    bg:SetTexture("Interface\\Tooltips\\UI-Tooltip-Background")
    bg:SetVertexColor(0.15, 0.15, 0.15, 0.7)
    item.bg = bg
    
    local highlight = item:CreateTexture(nil, "HIGHLIGHT")
    highlight:SetAllPoints()
    highlight:SetColorTexture(1, 1, 1, 0)
    item:SetHighlightTexture(highlight)
    
    item.title = item:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    item.title:SetPoint("CENTER")
    item.title:SetText("添加更多音乐")
    item.title:SetTextColor(1, 1, 0)
    item.title:SetJustifyH("CENTER")
    item.title:SetWidth(UI_WIDTH - 80)
    
    item:SetScript("OnClick", function()
        ShowAddMusicInput()
    end)
    
    item:SetScript("OnEnter", function()
        item.bg:SetVertexColor(0.25, 0.25, 0.25, 0.7)
        item.title:SetTextColor(1, 1, 0.7)
    end)
    
    item:SetScript("OnLeave", function()
        item.bg:SetVertexColor(0.15, 0.15, 0.15, 0.7)
        item.title:SetTextColor(1, 1, 0)
    end)
    
    musicItems[index] = item
    return item
end

local function StopAllMusic()
    effect("run_workflow", args, "o", "mus00")
    
    if args._G.StopMusic then
        args._G.StopMusic()
    elseif args._G.StopSound then
        args._G.StopSound(SOUNDKIT.MUSIC_GENERAL)
    end
end

local function CreateMusicItem(parent, index, musicData)
    local item = args._G.CreateFrame("Button", nil, parent)
    item:SetSize(UI_WIDTH - 10, 40)
    item:SetPoint("TOPLEFT", 0, -(index-1)*50)
    
    local bg = item:CreateTexture(nil, "BACKGROUND")
    bg:SetAllPoints()
    bg:SetTexture("Interface\\Tooltips\\UI-Tooltip-Background")
    bg:SetVertexColor(0.15, 0.15, 0.15, 0.7)
    item.bg = bg
    
    if musicData.isCustom then
        local customMarker = item:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        customMarker:SetPoint("LEFT", 10, 0)
        customMarker:SetText("★")
        customMarker:SetTextColor(1, 1, 0)
    end
    
    local highlight = item:CreateTexture(nil, "HIGHLIGHT")
    highlight:SetAllPoints()
    highlight:SetColorTexture(1, 1, 1, 0)
    item:SetHighlightTexture(highlight)
    
    item.title = item:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    item.title:SetPoint("LEFT", musicData.isCustom and 30 or 10, 0)
    item.title:SetText(musicData.title)
    item.title:SetTextColor(1, 1, 1)
    item.title:SetJustifyH("LEFT")
    item.title:SetWidth(UI_WIDTH - 120)
    
    if musicData.isCustom then
        local deleteBtn = args._G.CreateFrame("Button", nil, item, "UIPanelButtonTemplate")
        deleteBtn:SetSize(50, 20)
        deleteBtn:SetPoint("RIGHT", -20, 0)
        deleteBtn:SetText("移除")
        
        deleteBtn:SetScript("OnClick", function()
    local actualIndex = index - 1
    local deletedTitle = MUSIC_LIST[actualIndex].originalTitle or MUSIC_LIST[actualIndex].title
    
    table.remove(MUSIC_LIST, actualIndex)
    
    local customMusicEntries = {}
    for _, music in ipairs(MUSIC_LIST) do
        if music.isCustom then
            local originalTitle = music.originalTitle or music.title:gsub(" %-%d+$", "")
            table.insert(customMusicEntries, music.workflowID .. ":" .. originalTitle)
        end
    end
    setVar(args, "o", "custom_music", table.concat(customMusicEntries, "|"))
    
    RebuildMusicList()
    
    if currentPlayingIndex and currentPlayingIndex == actualIndex - 1 then
        StopAllMusic()
        currentPlayingIndex = nil
    end
    
    args._G.PlaySound(822)
end)
    end
    
    item:SetScript("OnClick", function()
        args._G.PlaySound(856)

        local adjustedIndex = index - 1
        

        if currentPlayingIndex and musicItems[currentPlayingIndex + 1] then
            local prevItem = musicItems[currentPlayingIndex + 1]
            prevItem.bg:SetVertexColor(0.15, 0.15, 0.15, 0.7)
            prevItem.title:SetTextColor(1, 1, 1)
        end
        
        if currentPlayingIndex == adjustedIndex then
            StopAllMusic()
            currentPlayingIndex = nil
            item.bg:SetVertexColor(0.15, 0.15, 0.15, 0.7)
            item.title:SetTextColor(1, 1, 1)
        else
            StopAllMusic()
            
            currentPlayingIndex = adjustedIndex
            local workflowID = musicData.workflowID
            
            if musicData.isCustom then
                if args._G.PlayMusic then
                    args._G.PlayMusic(workflowID)
                elseif args._G.PlaySoundFile then
                    args._G.PlaySoundFile(workflowID, "Master")
                else
                    args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, "不支持的音乐播放方式", {r=1, g=0, b=0})
                    currentPlayingIndex = nil
                    return
                end
            else
                effect("run_workflow", args, "o", workflowID)
            end
            
            item.bg:SetVertexColor(0.5, 0.1, 0.5, 0.7)
            item.title:SetTextColor(1, 1, 1)
        end
    end)
    
    item:SetScript("OnEnter", function()
        if currentPlayingIndex ~= (index - 1) then
            item.bg:SetVertexColor(0.25, 0.25, 0.25, 0.7)
            item.title:SetTextColor(0.8, 0.8, 1)
        end
    end)
    
    item:SetScript("OnLeave", function()
        if currentPlayingIndex ~= (index - 1) then
            item.bg:SetVertexColor(0.15, 0.15, 0.15, 0.7)
            item.title:SetTextColor(1, 1, 1)
        else
            item.bg:SetVertexColor(0.5, 0.1, 0.5, 0.7)
            item.title:SetTextColor(1, 1, 1)
        end
    end)
    
    musicItems[index] = item
    return item
end

function RebuildMusicList()
    for _, item in ipairs(musicItems) do
        item:Hide()
        item:SetParent(nil)
    end
    musicItems = {}
    
    local musicScrollContent = args._G.CreateFrame("Frame", nil, musicScrollFrame)
    musicScrollContent:SetSize(UI_WIDTH - 40, (#MUSIC_LIST + 1) * 50)  -- +1 是为了添加按钮
    musicScrollFrame:SetScrollChild(musicScrollContent)
    
    CreateAddMusicItem(musicScrollContent, 1)
    
    for i, music in ipairs(MUSIC_LIST) do
        CreateMusicItem(musicScrollContent, i + 1, music)  -- i + 1 是因为第一个位置是添加按钮
    end
end

RebuildMusicList()

local muteBtn = args._G.CreateFrame("Button", nil, musicPage)
muteBtn:SetSize(80, 80)
muteBtn:SetPoint("BOTTOMRIGHT", phoneFrame, "BOTTOMRIGHT", -10, 170)
muteBtn:SetFrameLevel(15)

local muteHighlight = muteBtn:CreateTexture(nil, "HIGHLIGHT")
muteHighlight:SetSize(90, 90)
muteHighlight:SetPoint("CENTER", 0, -20)
muteHighlight:SetTexture("Interface\\Buttons\\UI-Common-MouseHilight")
muteHighlight:SetBlendMode("ADD")
muteHighlight:SetVertexColor(0, 0, 0)
muteHighlight:SetAlpha(0.1)
muteBtn:SetHighlightTexture(muteHighlight)

local muteIcon = muteBtn:CreateTexture(nil, "ARTWORK")
muteIcon:SetSize(50, 50)
muteIcon:SetPoint("TOP", 0, -30)
muteIcon:SetTexture("Interface\\Buttons\\UI-GroupLoot-Pass-Up")
muteIcon:SetVertexColor(1, 1, 1)

local muteText = muteBtn:CreateFontString(nil, "OVERLAY", "GameFontNormal")
muteText:SetPoint("BOTTOM", 0, -15)
muteText:SetText("停止音乐")
muteText:SetTextColor(1, 1, 1)

local mutePushed = muteBtn:CreateTexture(nil, "OVERLAY")
mutePushed:SetSize(50, 50)
mutePushed:SetPoint("TOP", 0, -10)
mutePushed:SetTexture("Interface\\Buttons\\UI-Quickslot-Depress")
mutePushed:SetAlpha(0)
muteBtn:SetPushedTexture(mutePushed)

local muteAnimating = false

muteBtn:SetScript("OnClick", function()
    if muteAnimating then return end
    muteAnimating = true
    muteIcon:SetAlpha(0.7)
    muteIcon:SetSize(45, 45)
    
    StopAllMusic()
    args._G.PlaySound(822)
    
    if currentPlayingIndex and musicItems[currentPlayingIndex + 1] then
        local prevItem = musicItems[currentPlayingIndex + 1]
        prevItem.bg:SetVertexColor(0.15, 0.15, 0.15, 0.7)
        prevItem.title:SetTextColor(1, 1, 1)
    end
    
    currentPlayingIndex = nil
    args._G.C_Timer.NewTicker(0.1, function()
        muteIcon:SetAlpha(1)
        muteIcon:SetSize(50, 50)
        muteAnimating = false
    end, 1)
end)

local muteHoverSound = false
muteBtn:SetScript("OnEnter", function(self)
    if not muteHoverSound then
        args._G.PlaySound(807)
        muteHoverSound = true
    end
    self:GetHighlightTexture():SetAlpha(0.9)
    muteIcon:ClearAllPoints()
    muteIcon:SetPoint("TOP", 0, -30)
    muteIcon:SetSize(55, 55)
    muteText:SetTextColor(1, 1, 0.3)
end)

muteBtn:SetScript("OnLeave", function(self)
    muteHoverSound = false
    self:GetHighlightTexture():SetAlpha(0.7)
    muteIcon:ClearAllPoints()
    muteIcon:SetPoint("TOP", 0, -30)
    muteIcon:SetSize(50, 50)
    muteText:SetTextColor(1, 1, 1)
end)

local dicePage = args._G.CreateFrame("Frame", nil, mainContentFrame); dicePage:SetAllPoints(); dicePage:Hide()
local diceContentFrame = args._G.CreateFrame("Frame", nil, dicePage); diceContentFrame:SetPoint("TOP", statusBar, "BOTTOM", 0, -15); diceContentFrame:SetPoint("LEFT", 20, 0); diceContentFrame:SetPoint("RIGHT", -20, 0); diceContentFrame:SetPoint("BOTTOM", phoneFrame, "BOTTOM", 0, 100)
local diceTitleText = diceContentFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge"); diceTitleText:SetPoint("TOP", 0, 0); diceTitleText:SetText("跑团骰子判定器"); diceTitleText:SetTextColor(0.8, 0.8, 1); diceTitleText:SetFont(diceTitleText:GetFont(), 18, "OUTLINE")
local countFrame = args._G.CreateFrame("Frame", nil, diceContentFrame); countFrame:SetSize(UI_WIDTH - 60, 50); countFrame:SetPoint("TOP", diceTitleText, "BOTTOM", 0, -5)
local countLabel = countFrame:CreateFontString(nil, "OVERLAY", "GameFontNormal"); countLabel:SetPoint("LEFT", 0, 0); countLabel:SetText("骰子数量:"); countLabel:SetTextColor(1, 1, 1)
local countValue = countFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge"); countValue:SetPoint("CENTER", 0, 0); countValue:SetText(diceCount); countValue:SetTextColor(1, 1, 0)
local countAddBtn = args._G.CreateFrame("Button", nil, countFrame, "UIPanelButtonTemplate"); countAddBtn:SetSize(30, 30); countAddBtn:SetPoint("RIGHT", -10, 0); countAddBtn:SetText("+")
countAddBtn.isDown, countAddBtn.lastUpdate, countAddBtn.delay, countAddBtn.interval = false, 0, 0.3, 0.1
countAddBtn:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then self.isDown = true; self.lastUpdate = args._G.GetTime(); diceCount = math.min((diceCount or 1) + 1, 10); countValue:SetText(diceCount); args._G.PlaySound(624) end end)
countAddBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); countAddBtn:SetScript("OnLeave", function(self) self.isDown = false end)
countAddBtn:SetScript("OnUpdate", function(self) if self.isDown then local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then diceCount = math.min((diceCount or 1) + 1, 10); countValue:SetText(diceCount); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval end end end)
local countMinusBtn = args._G.CreateFrame("Button", nil, countFrame, "UIPanelButtonTemplate"); countMinusBtn:SetSize(30, 30); countMinusBtn:SetPoint("RIGHT", countAddBtn, "LEFT", -5, 0); countMinusBtn:SetText("-")
countMinusBtn.isDown, countMinusBtn.lastUpdate, countMinusBtn.delay, countMinusBtn.interval = false, 0, 0.3, 0.1
countMinusBtn:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then self.isDown = true; self.lastUpdate = args._G.GetTime(); diceCount = math.max((diceCount or 1) - 1, 1); countValue:SetText(diceCount); args._G.PlaySound(624) end end)
countMinusBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); countMinusBtn:SetScript("OnLeave", function(self) self.isDown = false end)
countMinusBtn:SetScript("OnUpdate", function(self) if self.isDown then local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then diceCount = math.max((diceCount or 1) - 1, 1); countValue:SetText(diceCount); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval end end end)
local facesFrame = args._G.CreateFrame("Frame", nil, diceContentFrame); facesFrame:SetSize(UI_WIDTH - 60, 50); facesFrame:SetPoint("TOP", countFrame, "BOTTOM", 0, -5)
local facesLabel = facesFrame:CreateFontString(nil, "OVERLAY", "GameFontNormal"); facesLabel:SetPoint("LEFT", 0, 0); facesLabel:SetText("骰子面数:"); facesLabel:SetTextColor(1, 1, 1)
local facesValue = facesFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge"); facesValue:SetPoint("CENTER", 0, 0); facesValue:SetText(diceFaces); facesValue:SetTextColor(1, 1, 0)
local facesAddBtn = args._G.CreateFrame("Button", nil, facesFrame, "UIPanelButtonTemplate"); facesAddBtn:SetSize(30, 30); facesAddBtn:SetPoint("RIGHT", -10, 0); facesAddBtn:SetText("+")
facesAddBtn.isDown, facesAddBtn.lastUpdate, facesAddBtn.delay, facesAddBtn.interval = false, 0, 0.3, 0.1
facesAddBtn:SetScript("OnMouseDown", function(self, button)
    if button == "LeftButton" then
        self.isDown = true; self.lastUpdate = args._G.GetTime(); local facesList = {4,6,8,10,12,20,100}; local currentIndex
        for i, v in ipairs(facesList) do if v == diceFaces then currentIndex = i; break end end
        if currentIndex and currentIndex < #facesList then diceFaces = facesList[currentIndex + 1] else diceFaces = 4 end
        facesValue:SetText(diceFaces); args._G.PlaySound(624)
    end
end)
facesAddBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); facesAddBtn:SetScript("OnLeave", function(self) self.isDown = false end)
facesAddBtn:SetScript("OnUpdate", function(self)
    if self.isDown then
        local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then
            local facesList = {4,6,8,10,12,20,100}; local currentIndex
            for i, v in ipairs(facesList) do if v == diceFaces then currentIndex = i; break end end
            if currentIndex and currentIndex < #facesList then diceFaces = facesList[currentIndex + 1] else diceFaces = 4 end
            facesValue:SetText(diceFaces); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval
        end
    end
end)
local facesMinusBtn = args._G.CreateFrame("Button", nil, facesFrame, "UIPanelButtonTemplate"); facesMinusBtn:SetSize(30, 30); facesMinusBtn:SetPoint("RIGHT", facesAddBtn, "LEFT", -5, 0); facesMinusBtn:SetText("-")
facesMinusBtn.isDown, facesMinusBtn.lastUpdate, facesMinusBtn.delay, facesMinusBtn.interval = false, 0, 0.3, 0.1
facesMinusBtn:SetScript("OnMouseDown", function(self, button)
    if button == "LeftButton" then
        self.isDown = true; self.lastUpdate = args._G.GetTime(); local facesList = {4,6,8,10,12,20,100}; local currentIndex
        for i, v in ipairs(facesList) do if v == diceFaces then currentIndex = i; break end end
        if currentIndex and currentIndex > 1 then diceFaces = facesList[currentIndex - 1] else diceFaces = 100 end
        facesValue:SetText(diceFaces); args._G.PlaySound(624)
    end
end)
facesMinusBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); facesMinusBtn:SetScript("OnLeave", function(self) self.isDown = false end)
facesMinusBtn:SetScript("OnUpdate", function(self)
    if self.isDown then
        local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then
            local facesList = {4,6,8,10,12,20,100}; local currentIndex
            for i, v in ipairs(facesList) do if v == diceFaces then currentIndex = i; break end end
            if currentIndex and currentIndex > 1 then diceFaces = facesList[currentIndex - 1] else diceFaces = 100 end
            facesValue:SetText(diceFaces); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval
        end
    end
end)
local difficultyFrame = args._G.CreateFrame("Frame", nil, diceContentFrame); difficultyFrame:SetSize(UI_WIDTH - 60, 50); difficultyFrame:SetPoint("TOP", facesFrame, "BOTTOM", 0, -5)
local difficultyLabel = difficultyFrame:CreateFontString(nil, "OVERLAY", "GameFontNormal"); difficultyLabel:SetPoint("LEFT", 0, 0); difficultyLabel:SetText("检定难度:"); difficultyLabel:SetTextColor(1, 1, 1)
local difficultyValue = difficultyFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge"); difficultyValue:SetPoint("CENTER", 0, 0); difficultyValue:SetText(difficulty); difficultyValue:SetTextColor(1, 1, 0)
local difficultyAddBtn = args._G.CreateFrame("Button", nil, difficultyFrame, "UIPanelButtonTemplate"); difficultyAddBtn:SetSize(30, 30); difficultyAddBtn:SetPoint("RIGHT", -10, 0); difficultyAddBtn:SetText("+")
difficultyAddBtn.isDown, difficultyAddBtn.lastUpdate, difficultyAddBtn.delay, difficultyAddBtn.interval = false, 0, 0.3, 0.1
difficultyAddBtn:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then self.isDown = true; self.lastUpdate = args._G.GetTime(); difficulty = math.min(difficulty + 1, 30); difficultyValue:SetText(difficulty); args._G.PlaySound(624) end end)
difficultyAddBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); difficultyAddBtn:SetScript("OnLeave", function(self) self.isDown = false end)
difficultyAddBtn:SetScript("OnUpdate", function(self) if self.isDown then local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then difficulty = math.min(difficulty + 1, 30); difficultyValue:SetText(difficulty); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval end end end)
local difficultyMinusBtn = args._G.CreateFrame("Button", nil, difficultyFrame, "UIPanelButtonTemplate"); difficultyMinusBtn:SetSize(30, 30); difficultyMinusBtn:SetPoint("RIGHT", difficultyAddBtn, "LEFT", -5, 0); difficultyMinusBtn:SetText("-")
difficultyMinusBtn.isDown, difficultyMinusBtn.lastUpdate, difficultyMinusBtn.delay, difficultyMinusBtn.interval = false, 0, 0.3, 0.1
difficultyMinusBtn:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then self.isDown = true; self.lastUpdate = args._G.GetTime(); difficulty = math.max(difficulty - 1, 1); difficultyValue:SetText(difficulty); args._G.PlaySound(624) end end)
difficultyMinusBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); difficultyMinusBtn:SetScript("OnLeave", function(self) self.isDown = false end)
difficultyMinusBtn:SetScript("OnUpdate", function(self) if self.isDown then local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then difficulty = math.max(difficulty - 1, 1); difficultyValue:SetText(difficulty); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval end end end)
local modifierFrame = args._G.CreateFrame("Frame", nil, diceContentFrame); modifierFrame:SetSize(UI_WIDTH - 60, 50); modifierFrame:SetPoint("TOP", difficultyFrame, "BOTTOM", 0, -5)
local modifierLabel = modifierFrame:CreateFontString(nil, "OVERLAY", "GameFontNormal"); modifierLabel:SetPoint("LEFT", 0, 0); modifierLabel:SetText("修正值:"); modifierLabel:SetTextColor(1, 1, 1)
local modifierValue = modifierFrame:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge"); modifierValue:SetPoint("CENTER", 0, 0); modifierValue:SetText(modifier); modifierValue:SetTextColor(1, 1, 0)
local modifierAddBtn = args._G.CreateFrame("Button", nil, modifierFrame, "UIPanelButtonTemplate"); modifierAddBtn:SetSize(30, 30); modifierAddBtn:SetPoint("RIGHT", -10, 0); modifierAddBtn:SetText("+")
modifierAddBtn.isDown, modifierAddBtn.lastUpdate, modifierAddBtn.delay, modifierAddBtn.interval = false, 0, 0.3, 0.1
modifierAddBtn:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then self.isDown = true; self.lastUpdate = args._G.GetTime(); modifier = math.min(modifier + 1, 10); modifierValue:SetText(modifier); args._G.PlaySound(624) end end)
modifierAddBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); modifierAddBtn:SetScript("OnLeave", function(self) self.isDown = false end)
modifierAddBtn:SetScript("OnUpdate", function(self) if self.isDown then local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then modifier = math.min(modifier + 1, 10); modifierValue:SetText(modifier); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval end end end)
local modifierMinusBtn = args._G.CreateFrame("Button", nil, modifierFrame, "UIPanelButtonTemplate"); modifierMinusBtn:SetSize(30, 30); modifierMinusBtn:SetPoint("RIGHT", modifierAddBtn, "LEFT", -5, 0); modifierMinusBtn:SetText("-")
modifierMinusBtn.isDown, modifierMinusBtn.lastUpdate, modifierMinusBtn.delay, modifierMinusBtn.interval = false, 0, 0.3, 0.1
modifierMinusBtn:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then self.isDown = true; self.lastUpdate = args._G.GetTime(); modifier = math.max(modifier - 1, -10); modifierValue:SetText(modifier); args._G.PlaySound(624) end end)
modifierMinusBtn:SetScript("OnMouseUp", function(self) self.isDown = false end); modifierMinusBtn:SetScript("OnLeave", function(self) self.isDown = false end)
modifierMinusBtn:SetScript("OnUpdate", function(self) if self.isDown then local currentTime = args._G.GetTime(); if currentTime - self.lastUpdate > self.delay then modifier = math.max(modifier - 1, -10); modifierValue:SetText(modifier); args._G.PlaySound(624); self.lastUpdate = currentTime; self.delay = self.interval end end end)
local actionBtn = args._G.CreateFrame("Button", nil, diceContentFrame, "UIPanelButtonTemplate"); actionBtn:SetSize(UI_WIDTH - 60, 40); actionBtn:SetPoint("TOP", modifierFrame, "BOTTOM", 0, -15); actionBtn:SetText("输入行为描述"); actionBtn:SetNormalFontObject("GameFontNormal"); actionBtn:SetHighlightFontObject("GameFontHighlight")
local resultFrame = args._G.CreateFrame("Frame", nil, diceContentFrame, "BackdropTemplate"); resultFrame:SetSize(UI_WIDTH - 60, 90); resultFrame:SetPoint("TOP", actionBtn, "BOTTOM", 0, -15)
resultFrame:SetBackdrop({bgFile = "Interface\\Tooltips\\UI-Tooltip-Background", edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border", edgeSize = 8, insets = {left = 4, right = 4, top = 4, bottom = 4}})
resultFrame:SetBackdropColor(0, 0, 0, 0.7); resultFrame:SetBackdropBorderColor(0.5, 0.5, 0.5, 1)
local resultText = resultFrame:CreateFontString(nil, "OVERLAY", "GameFontNormal"); resultText:SetPoint("CENTER", 0, 0); resultText:SetText("等待检定中…"); resultText:SetTextColor(1, 1, 1); resultText:SetWidth(UI_WIDTH - 80); resultText:SetJustifyH("CENTER"); resultText:SetJustifyV("MIDDLE")

local function RollDice(actionDesc)
    local diceTotal = 0; for i = 1, diceCount do diceTotal = diceTotal + math.random(1, diceFaces) end; local finalTotal = diceTotal + modifier
    local result; local diceNotation = diceCount .. "d" .. diceFaces; if modifier ~= 0 then diceNotation = diceNotation .. (modifier > 0 and "+" or "") .. modifier end
    if diceCount == 1 then
        if diceTotal == 1 then result = "大失败！" elseif diceTotal == diceFaces then result = "大成功！" elseif finalTotal >= difficulty then result = "成功。" else result = "失败。" end
    else result = finalTotal >= difficulty and "成功。" or "失败。" end
    local resultStr = string.format("%s(%s=%d/难度%d)→%s", actionDesc, diceNotation, finalTotal, difficulty, result)
    resultText:SetText(resultStr); args._G.SendChatMessage(resultStr, "EMOTE"); args._G.PlaySoundFile("840226", "Master")
end

actionBtn:SetScript("OnClick", function()
    if inputPanel and inputPanel:IsShown() then return end; if not dicePage:IsShown() then return end
    inputPanel = args._G.CreateFrame("Frame", nil, args._G.UIParent, "BackdropTemplate"); inputPanel:SetSize(UI_WIDTH - 40, 100); inputPanel:SetPoint("CENTER", 0, 0); inputPanel:SetFrameStrata("DIALOG"); inputPanel:SetFrameLevel(1000); inputPanel:EnableMouse(true)
    inputPanel:SetBackdrop({bgFile = "Interface\\Tooltips\\UI-Tooltip-Background", edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border", edgeSize = 8, insets = {left = 4, right = 4, top = 4, bottom = 4}})
    inputPanel:SetBackdropColor(0, 0, 0, 1); inputPanel:SetBackdropBorderColor(0.5, 0.5, 0.5, 1)
    local promptText = inputPanel:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); promptText:SetPoint("TOP", 0, -10); promptText:SetText("请输入行为描述（例如：尝试撬开宝箱锁）"); promptText:SetTextColor(0.8, 0.8, 0.8)
    local inputBox = args._G.CreateFrame("EditBox", nil, inputPanel, "InputBoxTemplate"); inputBox:SetSize(UI_WIDTH - 80, 30); inputBox:SetPoint("TOP", promptText, "BOTTOM", 0, -5); inputBox:SetText(""); inputBox:Show(); inputBox:SetFocus()
    local btnContainer = args._G.CreateFrame("Frame", nil, inputPanel); btnContainer:SetSize(UI_WIDTH - 80, 30); btnContainer:SetPoint("TOP", inputBox, "BOTTOM", 0, -5)
    local confirmBtn = args._G.CreateFrame("Button", nil, btnContainer, "UIPanelButtonTemplate"); confirmBtn:SetSize(70, 22); confirmBtn:SetPoint("RIGHT", btnContainer, "CENTER", -5, 0); confirmBtn:SetText("确定")
    confirmBtn:SetScript("OnMouseDown", function() args._G.PlaySound(856) end)
    confirmBtn:SetScript("OnClick", function() local text = inputBox:GetText(); if text and text ~= "" then RollDice(text) end; inputPanel:Hide() end)
    local cancelBtn = args._G.CreateFrame("Button", nil, btnContainer, "UIPanelButtonTemplate"); cancelBtn:SetSize(70, 22); cancelBtn:SetPoint("LEFT", btnContainer, "CENTER", 5, 0); cancelBtn:SetText("取消")
    cancelBtn:SetScript("OnMouseDown", function() args._G.PlaySound(856) end); cancelBtn:SetScript("OnClick", function() inputPanel:Hide() end)
    inputBox:SetScript("OnEnterPressed", function(self) local text = self:GetText(); if text and text ~= "" then RollDice(text) end; inputPanel:Hide() end)
    inputBox:SetScript("OnEscapePressed", function(self) inputPanel:Hide() end)
    dicePage:HookScript("OnHide", function() if inputPanel and inputPanel:IsShown() then inputPanel:Hide(); inputPanel = nil end end)
    args._G.PlaySound(624)
end)

local modelPage = args._G.CreateFrame("Frame", nil, mainContentFrame); modelPage:SetAllPoints(); modelPage:Hide()
local inputFrame = args._G.CreateFrame("Frame", nil, modelPage, "BackdropTemplate"); inputFrame:SetWidth(UI_WIDTH - 40); inputFrame:SetPoint("TOP", statusBar, "BOTTOM", 0, -20); inputFrame:SetPoint("BOTTOM", phoneFrame, "BOTTOM", 0, 80)
local inputBg = inputFrame:CreateTexture(nil, "BACKGROUND"); inputBg:SetAllPoints(); inputBg:SetTexture("Interface\\Tooltips\\UI-Tooltip-Background"); inputBg:SetVertexColor(0, 0, 0, 0.8)
inputFrame:SetBackdrop({edgeFile = "Interface\\Buttons\\WHITE8X8", edgeSize = 3, insets = {left = 1, right = 1, top = 1, bottom = 1}}); inputFrame:SetBackdropBorderColor(0, 1, 0, 1); inputFrame:SetFrameLevel(inputFrame:GetFrameLevel() + 1)
local inputHint = inputFrame:CreateFontString(nil, "OVERLAY", "GameFontNormal"); inputHint:SetPoint("TOP", 0, -20); inputHint:SetText("请输入NPC模型数字ID\n以生成全息投影（例：64572）"); inputHint:SetTextColor(1, 1, 1)
local npcInputBox = args._G.CreateFrame("EditBox", nil, inputFrame, "InputBoxTemplate"); npcInputBox:SetSize(200, 30); npcInputBox:SetPoint("TOP", inputHint, "BOTTOM", 0, -20); npcInputBox:SetFontObject("GameFontNormal"); npcInputBox:SetTextInsets(5, 5, 5, 5); npcInputBox:SetNumeric(true); npcInputBox:EnableMouse(true); npcInputBox:SetAutoFocus(false)
local btnSize = 80
local scanBtn = args._G.CreateFrame("Button", nil, inputFrame, "UIPanelButtonTemplate"); scanBtn:SetSize(btnSize, 30); scanBtn:SetPoint("TOP", npcInputBox, "BOTTOM", -btnSize - 5, -15); scanBtn:SetText("扫描目标")
local projectBtn = args._G.CreateFrame("Button", nil, inputFrame, "UIPanelButtonTemplate"); projectBtn:SetSize(btnSize, 30); projectBtn:SetPoint("TOP", npcInputBox, "BOTTOM", 0, -15); projectBtn:SetText("生成投影")
local randomBtn = args._G.CreateFrame("Button", nil, inputFrame, "UIPanelButtonTemplate"); randomBtn:SetSize(btnSize, 30); randomBtn:SetPoint("TOP", npcInputBox, "BOTTOM", btnSize + 5, -15); randomBtn:SetText("随机生成")
local modelFrame = args._G.CreateFrame("PlayerModel", nil, inputFrame); modelFrame:SetSize(200, 250); modelFrame:SetPoint("TOP", projectBtn, "BOTTOM", 0, -30); modelFrame:SetRotation(math.pi/2); modelFrame:EnableMouse(true)
local isDragging, lastMouseX, currentRotation = false, 0, 0

local function OnDragUpdate(self)
    if not isDragging then return end; local currentX = select(1, args._G.GetCursorPosition()); local deltaX = currentX - lastMouseX
    currentRotation = currentRotation + deltaX / 200; currentRotation = currentRotation % (2 * math.pi); self:SetFacing(currentRotation); lastMouseX = currentX
end

modelFrame:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then isDragging = true; lastMouseX = select(1, args._G.GetCursorPosition()); self:SetScript("OnUpdate", OnDragUpdate); args._G.PlaySound(806) end end)
modelFrame:SetScript("OnMouseUp", function(self, button) if button == "LeftButton" then isDragging = false; self:SetScript("OnUpdate", nil) end end)

local function LoadNPCModel(npcId)
    if not npcId or npcId == "" then args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, "请输入有效的ID。", {r=1, g=0, b=0}); args._G.PlaySound(847); return end
    local id = tonumber(npcId); if not id then args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, "ID必须是数字。", {r=1, g=0, b=0}); args._G.PlaySound(847); return end
    local loadCount, maxTries = 0, 3
    local function actuallyLoadModel()
        modelFrame:ClearModel(); modelFrame:SetModelScale(0.8); modelFrame:SetPosition(0, 0, 0); modelFrame:SetRotation(0); modelFrame:Show()
        local success = args._G.pcall(function() modelFrame:SetDisplayInfo(0); modelFrame:SetCreature(id); modelFrame:SetPosition(0, 0, 0); modelFrame:SetFacing(0) end)
        loadCount = loadCount + 1; if loadCount < maxTries then args._G.C_Timer.After(0.1, actuallyLoadModel) end
    end
    actuallyLoadModel()
end

randomBtn:SetScript("OnClick", function() local randomId = math.random(1, 240000); npcInputBox:SetText(tostring(randomId)); LoadNPCModel(randomId); args._G.PlaySound(15252) end)
scanBtn:SetScript("OnClick", function()
    local targetGUID = args._G.UnitGUID("target"); if not targetGUID then args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, "请先选择一个目标。", {r=1, g=0, b=0}); args._G.PlaySound(847); return end
    local type, _, _, _, _, npcId = args._G.strsplit("-", targetGUID or ""); if type == "Creature" and npcId then npcInputBox:SetText(npcId); args._G.PlaySound(15828)
    else args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, "你必须选中一个非玩家目标进行扫描。", {r=1, g=0, b=0}); args._G.PlaySound(847) end
end)
projectBtn:SetScript("OnClick", function() LoadNPCModel(npcInputBox:GetText()); args._G.PlaySound(3642) end)
npcInputBox:SetScript("OnEnterPressed", function() LoadNPCModel(npcInputBox:GetText()); if npcInputBox:GetText() ~= "" and tonumber(npcInputBox:GetText()) then args._G.PlaySound(3642) end; npcInputBox:ClearFocus() end)

local bottomBar = args._G.CreateFrame("Frame", nil, phoneFrame); bottomBar:SetHeight(70); bottomBar:SetPoint("BOTTOMLEFT", 5, 5); bottomBar:SetPoint("BOTTOMRIGHT", -5, 5)
local bottomBg = bottomBar:CreateTexture(nil, "BACKGROUND"); bottomBg:SetAllPoints(); bottomBg:SetTexture("Interface\\Tooltips\\UI-Tooltip-Background"); bottomBg:SetVertexColor(0, 0, 0, 1)
local returnBtn = args._G.CreateFrame("Button", nil, phoneFrame); returnBtn:SetSize(80, 80); returnBtn:SetPoint("BOTTOMRIGHT", -10, 80); returnBtn:Hide(); returnBtn:SetFrameLevel(20)
local highlight = returnBtn:CreateTexture(nil, "HIGHLIGHT"); highlight:SetSize(90, 90); highlight:SetPoint("CENTER"); highlight:SetTexture("Interface\\Buttons\\UI-Common-MouseHilight"); highlight:SetBlendMode("ADD"); highlight:SetVertexColor(0, 0, 0); highlight:SetAlpha(0.1); returnBtn:SetHighlightTexture(highlight)
local icon = returnBtn:CreateTexture(nil, "ARTWORK"); icon:SetSize(50, 50); icon:SetPoint("TOP", 0, -10); icon:SetTexture("Interface\\Buttons\\UI-SpellbookIcon-PrevPage-Up"); icon:SetVertexColor(1, 1, 1)
local text = returnBtn:CreateFontString(nil, "OVERLAY", "GameFontNormal"); text:SetPoint("BOTTOM", 0, 10); text:SetText("返回"); text:SetTextColor(1, 1, 1)
local pushed = returnBtn:CreateTexture(nil, "OVERLAY"); pushed:SetSize(50, 50); pushed:SetPoint("TOP", 0, -10); pushed:SetTexture("Interface\\Buttons\\UI-Quickslot-Depress"); pushed:SetAlpha(0); returnBtn:SetPushedTexture(pushed)
local isAnimating = false

returnBtn:SetScript("OnClick", function()
    if isAnimating then return end; isAnimating = true; icon:SetAlpha(0.7); icon:SetSize(45, 45)
    homePage:Show(); emotePage:Hide(); habitPage:Hide(); musicPage:Hide(); dicePage:Hide(); inputFrame:Hide(); returnBtn:Hide(); bottomBar:Show()
    args._G.PlaySound(624); args._G.C_Timer.NewTicker(0.1, function() icon:SetAlpha(1); icon:SetSize(50, 50); isAnimating = false end, 1)
end)

local hasPlayedHoverSound = false
returnBtn:SetScript("OnEnter", function(self)
    if not hasPlayedHoverSound then args._G.PlaySound(80); hasPlayedHoverSound = true end
    self:GetHighlightTexture():SetAlpha(0.9); icon:ClearAllPoints(); icon:SetPoint("TOP", 0, -10); icon:SetSize(55, 55); text:SetTextColor(1, 1, 0.3)
end)
returnBtn:SetScript("OnLeave", function(self)
    hasPlayedHoverSound = false; self:GetHighlightTexture():SetAlpha(0.7); icon:ClearAllPoints(); icon:SetPoint("TOP", 0, -10); icon:SetSize(50, 50); text:SetTextColor(1, 1, 1)
end)

local BOTTOM_BUTTONS = {
    {icon = "ui_chat", text = "发言", workflowID = "ask1"}, {icon = "spell_holy_stoicism", text = "动作", workflowID = "ask2"},
    {icon = "ability_warrior_rallyingcry", text = "大喊", workflowID = "ask3"}, {icon = "achievement_general", text = "预设表情"}
}

local BOTTOM_BTN_WIDTH, BOTTOM_BTN_PADDING, bottomGridWidth, bottomStartX = 60, 18, (60 * #BOTTOM_BUTTONS) + (18 * (#BOTTOM_BUTTONS - 1)), (UI_WIDTH - (60 * #BOTTOM_BUTTONS) - (18 * (#BOTTOM_BUTTONS - 1)) - 20) / 2 + 5
local presetButton

for i, btnData in ipairs(BOTTOM_BUTTONS) do
    local btn = args._G.CreateFrame("Button", nil, bottomBar); btn:SetSize(BOTTOM_BTN_WIDTH, 70); btn:SetPoint("LEFT", bottomStartX + (i-1)*(BOTTOM_BTN_WIDTH + BOTTOM_BTN_PADDING), 0)
    local highlight = btn:CreateTexture(nil, "HIGHLIGHT"); highlight:SetSize(BOTTOM_BTN_WIDTH * 1.1, 80); highlight:SetPoint("CENTER"); highlight:SetTexture("Interface\\Buttons\\UI-Common-MouseHilight"); highlight:SetBlendMode("ADD"); highlight:SetAlpha(0.7); btn:SetHighlightTexture(highlight)
    local icon = btn:CreateTexture(nil, "ARTWORK"); icon:SetSize(30, 30); icon:SetPoint("TOP", 0, -15); icon:SetTexture("Interface\\Icons\\"..btnData.icon); icon:SetVertexColor(1, 1, 1)
    local text = btn:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); text:SetPoint("BOTTOM", 0, 8); text:SetText(btnData.text); text:SetTextColor(1, 1, 1)
    if btnData.text == "预设表情" then
        presetButton = btn; local glowBorder = btn:CreateTexture(nil, "OVERLAY"); glowBorder:SetSize(BOTTOM_BTN_WIDTH + 10, 80); glowBorder:SetPoint("CENTER"); glowBorder:SetTexture("Interface\\Buttons\\UI-ActionButton-Border"); glowBorder:SetBlendMode("ADD"); glowBorder:SetAlpha(0.8); glowBorder:SetVertexColor(0, 0.8, 1); glowBorder:SetDesaturated(true); glowBorder:Hide(); btn.glowBorder = glowBorder
        local glowBg = btn:CreateTexture(nil, "BACKGROUND"); glowBg:SetSize(BOTTOM_BTN_WIDTH + 6, 76); glowBg:SetPoint("CENTER"); glowBg:SetTexture("Interface\\Buttons\\UI-ActionButton-Border"); glowBg:SetAlpha(0.3); glowBg:SetVertexColor(0, 0.5, 1); glowBg:Hide(); btn.glowBg = glowBg
    end
    local pushed = btn:CreateTexture(nil, "OVERLAY"); pushed:SetSize(30, 30); pushed:SetPoint("TOP", 0, -15); pushed:SetTexture("Interface\\Buttons\\UI-Quickslot-Depress"); pushed:SetAlpha(0); btn:SetPushedTexture(pushed)
    local isBtnAnimating = false
    btn:SetScript("OnClick", function()
        if isBtnAnimating then return end; isBtnAnimating = true; icon:SetAlpha(0.7); icon:SetSize(25, 25)
        if btnData.workflowID then effect("run_workflow", args, "o", btnData.workflowID)
        elseif btnData.text == "预设表情" then
            if args._G.ClosePetPhoneInterface then args._G.ClosePetPhoneInterface() end
            if inputPanel and inputPanel:IsShown() then inputPanel:Hide() end
            if args._G.simpleControlPanel and args._G.simpleControlPanel:IsShown() then args._G.simpleControlPanel:Hide() end
            if args._G.numgamePanel and args._G.numgamePanel:IsShown() then args._G.numgamePanel:Hide() end
            if homePage:IsShown() then homePage:Hide(); emotePage:Show(); habitPage:Hide(); musicPage:Hide(); dicePage:Hide(); btn.glowBorder:Show(); btn.glowBg:Show(); returnBtn:Hide()
            else homePage:Show(); emotePage:Hide(); habitPage:Hide(); musicPage:Hide(); dicePage:Hide(); btn.glowBorder:Hide(); btn.glowBg:Hide(); returnBtn:Hide() end
        end
        args._G.PlaySound(624); args._G.C_Timer.NewTicker(0.1, function() icon:SetAlpha(1); icon:SetSize(30, 30); isBtnAnimating = false end, 1)
    end)
    local hasPlayedHoverSound = false
    btn:SetScript("OnEnter", function(self)
        if not hasPlayedHoverSound then if args._G.PlaySound and type(args._G.PlaySound) == "function" then args._G.PlaySound(807) end; hasPlayedHoverSound = true end
        self:GetHighlightTexture():SetAlpha(0.9); icon:SetScale(1.1); text:SetTextColor(0.8, 0.8, 1)
    end)
    btn:SetScript("OnLeave", function(self) hasPlayedHoverSound = false; self:GetHighlightTexture():SetAlpha(0.7); icon:SetScale(1.0); text:SetTextColor(1, 1, 1) end)
end

local APP_GRID = {
    {icon = "inv_misc_map02", text = "地图", func = function() if args._G.WorldMapFrame and args._G.ToggleFrame then args._G.ToggleFrame(args._G.WorldMapFrame) end end},
    {icon = "inv_misc_groupneedmore", text = "联系人", func = function() if args._G.ToggleFriendsFrame then args._G.ToggleFriendsFrame(1) end end},
    {icon = "inv_misc_note_04", text = "备忘录", workflowID = "notes"},
    {icon = "inv_misc_discoball_01", text = "音乐随身听", func = function() homePage:Hide(); emotePage:Hide(); habitPage:Hide(); musicPage:Show(); dicePage:Hide(); if presetButton then presetButton.glowBorder:Hide(); presetButton.glowBg:Hide() end; returnBtn:Show() end},
    {icon = "inv_misc_note_03", text = "便签纸", workflowID = "字条"}, {icon = "achievement_guildperk_ladyluck_rank2", text = "纸牌:剥沙蟹", workflowID = "剥沙蟹"}, {icon = "inv_misc_number_2", text = "2048", workflowID = "2048"},
    {icon = "achievement_halloween_smiley_01", text = "口癖设置", func = function() homePage:Hide(); emotePage:Hide(); habitPage:Show(); musicPage:Hide(); dicePage:Hide(); if presetButton then presetButton.glowBorder:Hide(); presetButton.glowBg:Hide() end; returnBtn:Show() end},
    {icon = "inv_misc_dice_01", text = "跑团小助手", func = function() homePage:Hide(); emotePage:Hide(); habitPage:Hide(); musicPage:Hide(); dicePage:Show(); if presetButton then presetButton.glowBorder:Hide(); presetButton.glowBg:Hide() end; returnBtn:Show() end},
    {icon = "inv_misc_ selfiecamera_sketch", text = "滤镜大师", workflowID = "滤镜"}, {icon = "achievement_guildperk_gmail", text = "短信", workflowID = "短信1"}, {icon = "inv_pet_babypengu", text = "电子企鹅", workflowID = "宠物"},
    {icon = "inv_engineering_90_remote", text = "终端克隆", workflowID = "ask6"}, {icon = "inv_gnometoy", text = "开发者信息", workflowID = "开发者"},
    {icon = "ability_siege_engineer_pattern_recognition", text = "扫描仪", func = function() homePage:Hide(); emotePage:Hide(); habitPage:Hide(); musicPage:Hide(); dicePage:Hide(); modelPage:Show(); inputFrame:Show(); returnBtn:Show() end}
}

local ICON_SIZE, ICON_PADDING, GRID_COLS, GRID_ROWS, gridWidth, startX = 60, 18, 4, 4, (60 * 4) + (18 * 3), (UI_WIDTH - (60 * 4) - (18 * 3) - 30) / 2+5

for i, app in ipairs(APP_GRID) do
    local col, row = (i-1) % GRID_COLS, math.floor((i-1)/GRID_COLS)
    local btn = args._G.CreateFrame("Button", nil, contentFrame); btn:SetSize(ICON_SIZE, ICON_SIZE); btn:SetPoint("TOPLEFT", startX + col * (ICON_SIZE + ICON_PADDING), -row * (ICON_SIZE + ICON_PADDING + 5))
    local highlight = btn:CreateTexture(nil, "HIGHLIGHT"); highlight:SetSize(ICON_SIZE+20, ICON_SIZE+20); highlight:SetPoint("CENTER"); highlight:SetTexture("Interface\\Buttons\\UI-Common-MouseHilight"); highlight:SetBlendMode("ADD"); highlight:SetAlpha(0.7); btn:SetHighlightTexture(highlight)
    local iconBg = btn:CreateTexture(nil, "BACKGROUND"); iconBg:SetAllPoints(); iconBg:SetTexture("Interface\\Buttons\\UI-Quickslot-Depress"); iconBg:SetAlpha(0)
    local icon = btn:CreateTexture(nil, "ARTWORK"); icon:SetSize(ICON_SIZE-10, ICON_SIZE-10); icon:SetPoint("CENTER"); icon:SetTexture("Interface\\Icons\\"..app.icon)
    local text = btn:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); text:SetPoint("BOTTOM", btn, "BOTTOM", 0, -13); text:SetText(app.text); text:SetTextColor(1, 1, 1); text:SetFont(text:GetFont(), 13, "OUTLINE")
    local pushed = btn:CreateTexture(nil, "OVERLAY"); pushed:SetAllPoints(icon); pushed:SetTexture("Interface\\Buttons\\UI-Quickslot-Depress"); pushed:SetAlpha(0); btn:SetPushedTexture(pushed)
    local isBtnAnimating = false
    if app.workflowID then
        btn:SetScript("OnClick", function()
            if isBtnAnimating then return end
            if app.workflowID == "助威" then if btn.cooldown and btn.cooldown > args._G.GetTime() then args._G.PlaySound(847); return end; btn.cooldown = args._G.GetTime() + 15
                local cooldown = args._G.CreateFrame("Cooldown", nil, btn, "CooldownFrameTemplate"); cooldown:SetAllPoints(); cooldown:SetDrawEdge(false); args._G.CooldownFrame_Set(cooldown, args._G.GetTime(), 15, true)
            end
            isBtnAnimating = true; icon:SetAlpha(0.7); icon:SetSize(ICON_SIZE-15, ICON_SIZE-15)
            effect("run_workflow", args, "o", app.workflowID); args._G.PlaySound(624)
            args._G.C_Timer.NewTicker(0.1, function() icon:SetAlpha(1); icon:SetSize(ICON_SIZE-10, ICON_SIZE-10); isBtnAnimating = false end, 1)
        end)
    elseif app.func then
        btn:SetScript("OnClick", function()
            if isBtnAnimating then return end; isBtnAnimating = true; icon:SetAlpha(0.7); icon:SetSize(ICON_SIZE-15, ICON_SIZE-15)
            app.func(); args._G.PlaySound(624)
            args._G.C_Timer.NewTicker(0.1, function() icon:SetAlpha(1); icon:SetSize(ICON_SIZE-10, ICON_SIZE-10); isBtnAnimating = false end, 1)
        end)
    else btn:Disable() end
    local hasPlayedHoverSound = false
    btn:SetScript("OnEnter", function(self) if not hasPlayedHoverSound then args._G.PlaySound(807); hasPlayedHoverSound = true end; self:GetHighlightTexture():SetAlpha(0.9); text:SetTextColor(0.8, 0.8, 1) end)
    btn:SetScript("OnLeave", function(self) hasPlayedHoverSound = false; self:GetHighlightTexture():SetAlpha(0.7); text:SetTextColor(1, 1, 1) end)
end

local function ShowVolumeMessage(volume) local icon = ""; args._G.RaidNotice_AddMessage(args._G.RaidWarningFrame, icon.."当前主音量: "..math.floor(volume*100).."%", {r=0.1, g=1.0, b=0.1}) end
local function AdjustVolume(up) local current = args._G.GetCVar("Sound_MasterVolume"); current = tonumber(current) or 0.5; local new = up and math.min(current + 0.1, 1.0) or math.max(current - 0.1, 0.0); args._G.SetCVar("Sound_MasterVolume", new); args._G.PlaySound(624); ShowVolumeMessage(new) end
local volumePanel = args._G.CreateFrame("Frame", nil, phoneFrame, "BackdropTemplate"); volumePanel:SetSize(25, 100); volumePanel:SetPoint("TOPRIGHT", 22, -50)
volumePanel:SetBackdrop({bgFile = "Interface\\Tooltips\\UI-Tooltip-Background", edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border", edgeSize = 8, insets = {left = 2, right = 2, top = 2, bottom = 2}})
volumePanel:SetBackdropColor(0.1, 0.1, 0.1, 0.7); volumePanel:SetBackdropBorderColor(0.5, 0.5, 0.5, 1)
local volumeUp = args._G.CreateFrame("Button", nil, volumePanel, "UIPanelButtonTemplate"); volumeUp:SetSize(20, 25); volumeUp:SetPoint("TOP", 0, -5); volumeUp:SetText("+"); volumeUp:SetScript("OnClick", function() AdjustVolume(true) end)
local volumeDown = args._G.CreateFrame("Button", nil, volumePanel, "UIPanelButtonTemplate"); volumeDown:SetSize(20, 25); volumeDown:SetPoint("TOP", volumeUp, "BOTTOM", 0, -5); volumeDown:SetText("-"); volumeDown:SetScript("OnClick", function() AdjustVolume(false) end)
local volumeLabel = volumePanel:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall"); volumeLabel:SetPoint("BOTTOM", 0, 7); volumeLabel:SetText("音\n量"); volumeLabel:SetTextColor(1, 1, 1)

phoneFrame:SetScript("OnMouseDown", function(self, button) if button == "LeftButton" then self:StartMoving() end end)
phoneFrame:SetScript("OnMouseUp", function(self)
    self:StopMovingOrSizing()
    local point, relativeTo, relativePoint, xOfs, yOfs = self:GetPoint(1)
    setVar(args, "o", UI_POSITION_KEY, string.format("%s,%s,%s,%d,%d", point, relativeTo and relativeTo:GetName() or "UIParent", relativePoint, math.floor(xOfs), math.floor(yOfs)))
end)

UpdateHabitDisplay(); phoneFrame:Show()