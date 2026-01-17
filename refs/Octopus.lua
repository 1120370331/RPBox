---@v 1.1.5
Octopus = {
    logic = {
        axis_square = function (axis,x1,y1,x2,y2)

            
        end,
        ---@return table
        indexput = function (table)
            local j = 0
            local tb = {}
            for i,v in pairs(table) do
                tb[tostring(j)] = v
                j = j + 1
            end
            return tb
        end,
        conclude = function (table,obj) --是否有此值
    
            for i,v in pairs(table) do
    
                if v == obj then
                    return true
                end
    
            end
    
            return false
    
    
    
        end,
        haskey = function (table,key)--是否有此键
            for i,v in pairs(table) do
                if tostring(key) == tostring(i) then
                    return true
                end
    
            end
            return false
    
    
        end,
        allset = function(table,value)
    
            for i,v in pairs(table)do
                if type(i) == "table" then
                    Octopus.logic.allset(table,value)
                else
                    table[i] = value
                end
            end
    
            return table
    
        end ,
        isEmpty = function (tbl)
            if next(tbl) ~= nil then
                return false
            else
                return true
            end
        end,
        ---@param tb table
        tostring = function ( tb )
    
            --禁止使用以下字符：" [ ] $ ^
    
            local str = "{"
    
            for i,v in pairs(tb) do
    
                if type(v) == "string" then
    
                    str = str.."["..tostring(i).."] = ".."$"..v.."^"..","
                
                elseif type(v) == "table" then
    
                    str = str.."["..tostring(i).."] = "..Octopus.logic.tostring(v)..","
    
                else
    
                    str = str.."["..tostring(i).."] = ".."$"..tostring(v).."^"..","
                end
    
            end
    
            str = str.."}"
    
            return str
    
            --[[
                [key] = $value$
            ]]
    
    
    
        end,
    
        strtotable = function (tstring,runargs)
    
            local function wordstotable(str) --将字符串拆开，每个字符表内的一个元素。
                local i = 0
                local tbl = {}
                while i<= string.len(str) do
                    i = i+1
                    local tmp = string.sub(str,i,i)
                    tbl[i] = tmp
    
                end
                return tbl
            end
    
    
    
            tstring = string.sub(tstring,2,tstring.len(tstring)-1)
    
            local onkeytyping = false
            local onvaluetyping = false
            local onmorestrtyping = false
    
            local tbl = {}
    
            local count_key = 0
    
            local tempkey = ""
    
            local tempfield = ""
    
            local sontableamount = 0
    
            if type(runargs) == "nil" then
                runargs = {
    
                    tablelevel = 1,
                    tableamount = 0,
    
                }
            end
    
            local level = "【第 "..tostring(runargs["tablelevel"]).." 层】"
    
            --Logger.log(level.."开始处理，要处理的字符串："..tstring)
    
    
    
    
    
            for i,v in pairs(wordstotable(tstring)) do
    
                --Logger.log(level.."一次新循环，onkeytyping = "..tostring(onkeytyping).."| onvaluetyping = "..tostring(onvaluetyping).."| onmorestrtyping = "..tostring(onmorestrtyping).."| count_key = "..tostring(count_key).."| tempfield = "..tempfield.."| 已有发现子表数量 = " ..tostring(sontableamount))
    
                if tostring(v) == "{" then
    
                    if sontableamount == 0 and onvaluetyping == false then
    
    
    
                        onmorestrtyping = true
    
                        --Logger.log(level.."开始处理一个表中表")
    
                    end
    
                end
    
    
    
    
                if onkeytyping == false or onvaluetyping == false or onmorestrtyping  == false then
    
    
                    if tostring(v) == "[" and onmorestrtyping == false then
    
                        if count_key == 0 then
    
                            --Logger.log(level.."开始处理一个键")
    
                            onkeytyping = true
    
                        end
    
    
    
    
                    elseif tostring(v) == "$" and onmorestrtyping == false then
    
                        --Logger.log(level.."开始处理一个值")
    
                        onvaluetyping = true
    
                    end
    
                end
                if onkeytyping  == true and onvaluetyping == false and onmorestrtyping == false then
    
                    if tostring(v) == "]" then
    
                        tbl[tempfield] = "NOVALUE"
    
                        --Logger.log(level.."已完成储存一个键："..tempfield)
    
                        count_key = count_key + 1
    
                        --Logger.log(level.."现有的键数量："..tostring(count_key))
    
                        tempkey = tempfield
    
                        --Logger.log(level.."临时的键值："..tempkey)
    
                        tempfield = ""
    
                        --Logger.log(level.."（键）tempfield清零")
    
                        onkeytyping = false
    
                    else
                        if tostring(v) ~= "[" then
    
                        tempfield = tempfield..tostring(v)
                        --Logger.log(level.."（键）tempfield被写为"..tempfield)
                        end
                    end
                end
    
                if onvaluetyping == true and onkeytyping == false and onmorestrtyping == false then
    
                    if tostring(v) == "^" then
    
                        tbl[tempkey] = tempfield
    
                        --Logger.log(level.."已完成储存一个值："..tempfield)
    
                        count_key = count_key - 1
    
                        --Logger.log(level.."现有的键数量："..tostring(count_key))
    
                        tempkey = ""
    
                        tempfield = ""
    
                        --Logger.log(level.."（值）tempfield清零")
    
                        onvaluetyping = false
    
                    else
    
                        if tostring(v) ~= "$" then
    
    
    
                        tempfield = tempfield..tostring(v)
                        --Logger.log(level.."（值）tempfield被写为"..tempfield)
                        end
                    end
                end
    
    
    
    
                if onmorestrtyping == true then
    
    
    
    
    
                    if tostring(v) == "{" then
                        sontableamount = sontableamount + 1
    
                        tempfield = tempfield..tostring(v)
                    end
    
    
    
                    if tostring(v) == "}"  then
    
                        --Logger.log(level.."}处理中，目前有子表："..tostring(sontableamount))
    
    
                        if tonumber(sontableamount) == 1 then
    
    
    
                            tempfield = tempfield..tostring(v)
    
                            runargs["tablelevel"] = runargs["tablelevel"] + 1
    
    
    
    
                            local temptable = Octopus.logic.strtotable(tempfield,runargs)
    
                            runargs["tablelevel"] = runargs["tablelevel"] - 1
    
                            --Logger.log(level.."已完成处理一个嵌套表！")
    
                            tbl[tempkey] = temptable
    
                            count_key = count_key - 1
    
                            tempfield = ""
    
                            sontableamount = sontableamount - 1
    
    
    
                            --Logger.log(level.."（表）tempfield清零")
    
                            onmorestrtyping = false
    
                        else
    
                            tempfield = tempfield..tostring(v)
    
                            sontableamount = sontableamount - 1
    
                            --Logger.log(level.."已录入完一个子表")
    
    
    
    
                        end
                    end
    
    
    
                    if tostring(v) ~= "{" and tostring(v) ~= "}" then
    
    
    
    
    
    
    
                        --Logger.log(level.."（表）tempfield被写为"..tempfield)
    
                        tempfield = tempfield..tostring(v)
    
                        --Logger.log(level.."tempfield被写为"..tempfield)
    
    
                    end
    
    
    
    
                end
    
    
            end
    
            --Logger.log(level.."已返回最终结果。")
    
            return tbl
    
    
    
        end,
        
    
    
        complete = function ( tb ,targettb,temp)---@targettb为模板表，按照targettb将tb中不包含的键一一赋值
            if temp == nil then
                model = targettb
            else
                model = temp
            end
    
    
            for i , v in pairs(model) do
    
                if type(v) ~= "table" and type(tb[i]) == nil then
    
                    tb[i] = model[i]
                end
    
                if type(v) == "table" and type(tb[i]) == "nil" then
    
                    tb[i] = model[i]
                end
    
                    
    
                if type(v) == "table" and type(tb[i]) == "table" then
    
                    tb[i] = Octopus.logic.complete(tb[i],targettb,v)
                end
    
    
    
            end
    
            return tb
    
        end,
    
        length = function (table)
            
            local count = 0
            for i,v in pairs(table) do
                count = count + 1
            end
            return count
            
    
        end,
        ---@return table
        sortkeybynumber = function (table)
            local c = 0
            local temp = {}
            for i,v in pairs(table) do
                c = c + 1
                temp[tostring(c)] = v
            end
            return temp
        end,

        ---向一个字符索引表加入一个元素
        append = function (table,object)
            local len = Octopus.logic.length(table)
            table[tostring(len)] = object
        end,
        ---向一个字符索引表中删除一个元素，并重新编排索引
        remove = function (table,key)
            table[tostring(key)] = nil
            local j = 0
            local table2 = {}
            for i,v in pairs(table) do
                table2[tostring(j)] = v
                table[i] = nil
                j = j + 1
            end
            for i,v in pairs(table2) do
                table[i] =  v 
            end
            
        end,
        getKeyByValue = function (table, value)
            for k, v in pairs(table) do
                if v == value then
                    return k
                end
            end
            return nil -- 如果没有找到对应的键，返回 nil
        end
    
    },
    LOG = {
        writelog = function (text)
            if GLOBAL_LOG == nil then
                GLOBAL_LOG = ""
            end
            
            GLOBAL_LOG = GLOBAL_LOG..text.."\n"
        end,
        showlog = function ()
            Octopus.GUI.Menu.open(GLOBAL_LOG)
        end,
        clearlog = function ()
            GLOBAL_LOG = ""
            
        end,
        error = function (text)
            effect("text",args,'error:'..text,"4")
            effect("text",args,'error:'..text,"1")
        end
    },

    Assests = {
        PICTURES = {
            HEADLINE = "{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}"
        },
        SoundLibrary = {
    
            UI = {
        
        
                CANCEL = 44310,
                JIHESHI = 47615,
                DIG = 193991,
                BEAT = 116826,
                MACHINE = 138317,
                GEAR = 138318,
                MAGIC = 12988,
                FLAME = 145118,
                CLICK = 169567,
                ATTACK = 177165,
                SUCCESS = 165970
        
            }
        
        },
        
        Icons = {
    
            ['mage'] = '{icon:classicon_mage:16}',
            ['warrior'] = '{icon:classicon_warrior:16}',
            ['warlock'] = '{icon:classicon_warlock:16}',
            ['deathknight'] = '{icon:classicon_deathknight:16}',
            ['demonhunter'] = '{icon:classicon_demonhunter:16}',
            ['hunter'] = '{icon:classicon_hunter:16}',
            ['monk'] = '{icon:classicon_monk:16}',
            ['paladin'] = '{icon:classicon_paladin:16}',
            ['priest'] = '{icon:classicon_priest:16}',
            ['rogue'] = '{icon:classicon_rogue:16}',
            ['shaman'] = '{icon:classicon_shaman:16}',
            ['druid'] = '{icon:classicon_druid:16}',
            ['others'] = '{icon:ability_xaril_masterpoisoner_red:16}'
    
        },
        Colour = {
            paint = function (text,colour)
    
                text = "{col:"..colour.."}"..text.."{/col}"
    
                return text
                
            end,
    
            GREY = "878787",
            RED = "FF0500",
            DARK_RED = "680200",
            LIGHT_GREEN = "44FF56",
            DARK_BLUE = "020082",
    
    
        }
        
            
        },

    
    GUI = {

        Builder = {
            ---@param height integer|nil 菜单行数目，不填则无限
            ---@return table
            new_menu = function (height)
                local menu = {}
                menu["lines"] = {}
                height = height or 0
                for i=0,height-1 do
                    menu["lines"][tostring(i)] = Octopus.GUI.Builder.new_line()
                end
                menu["settings"] = {
                    ["CutIfOutSize"] = "FALSE",
                    ["MenuHeight"] = 768,
                    ["MenuWidth"] = 768

                }
                return menu
            end,
            ---@return table
            new_line = function ()
                return {
                    ["settings"] = {
                        ["BASE_COLOUR"] = "000000",
                        ["FONT"] = "h1",
                        ["LOCATION"] = "c",
                        ["IS_IMAGE"] = "FALSE"
                    },
                    ["objects"] = {
                        
                    }
                }
            end,
            ---@param type string text | link | image | icon
            ---@param args table text: {text,colour} 
            ---@param args table link: {text,colour,link} 
            ---@param args table image: {url,height,width} 
            ---@param args table icon: {url,size} 
            new_object = function (type,Aargs,line)
                if type == "text" then
                   
                    return {
                        ["type"] = "text",
                        ["text"] = Aargs["0"] or "",
                        ["colour"] = Aargs["1"] or "000000",
                    }
                elseif type == "link" then
                    return {
                        ["type"] = "link",
                        ["text"] = Aargs["0"] or "",
                        ["colour"] = Aargs["1"] or "000000",
                        ["link"] =Aargs["2"] or "NOLINK"
                                        }
                elseif type == "image" then
                    return {
                        ["type"] = "image",
                        ["url"] = Aargs["0"] or "Interface\\GLUES\\LOADINGSCREENS\\Expansion07\\Main\\Loadscreen_NzothRaid_Visions",
                        ["height"] = Aargs["1"] or 64,
                        ["width"] = Aargs["2"] or 64,
                    }
                elseif type == "icon" then
                    return {
                        ["type"] = "icon",
                        ["url"] = Aargs["0"] or "text",
                        ["size"] = Aargs["1"] or 64,
                    }
                else
                    error("Invalid menu object type.")
                end
            end,

            object_set = function (object,key,value)
                object[key] = value
            end,
            ---在某个位置之前插入，填loc=nil为末尾
            line_insert = function (line,object,loc)
                if loc then
                    if loc < Octopus.logic.length(line["objects"]) then
                        local newtb = {};
                        local j = 0
                        for i=0,Octopus.logic.length(line["objects"]) -1 do
                            if tostring(loc) == i then
                                newtb[tostring(j)] = object

                                
                                j = j + 1
                                newtb[tostring(j)] = line['objects'][tostring(i)]
                            else
                                newtb[tostring(j)] = line['objects'][tostring(i)]

                            end
                            j = j +1
                        end
                        for k in pairs(line['objects']) do
                            line['objects'][k] = nil
                        end
                        for k in pairs(newtb) do
                            line['objects'][k] = newtb[k]
                        end
                        
                    else

                    end
                else

                end
                
            end,
            line_replace = function (line,object,loc)
                line["objects"][tostring(loc)] = object
            end,
            line_remove = function (line,loc)
                line["objects"][tostring(loc)] = nil
            end,
            preset_menu = {
                ---@param ConfirmMessage string
                ---@return table
                confirmation = function (ConfirmMessage,ConfirmLore,ConfirmScript,ConfirmArgs,
                    CancelScript,CancelArgs)
                    local menu = Octopus.GUI.Builder.new_menu(3)
                    menu["lines"]["0"]["objects"]["0"] = 
                        Octopus.GUI.Builder.new_object("text",Octopus.logic.indexput({ConfirmMessage}))
                    
                    menu["lines"]["1"]["FONT"] = "h3"
                    menu["lines"]["1"]["objects"]["0"] = 
                        Octopus.GUI.Builder.new_object("text",Octopus.logic.indexput({ConfirmLore,"dc5a5a"}))
                    
                    menu["lines"]["2"]["objects"]["0"] = 
                        Octopus.GUI.Builder.new_object("link",Octopus.logic.indexput({"确定",
                        "444444",
                        "runner(scriptname="..ConfirmScript..",scriptargs="..ConfirmArgs.. ")"
                    }))
                    menu["lines"]["2"]["objects"]["1"] = 
                    Octopus.GUI.Builder.new_object("icon",Octopus.logic.indexput({"ability_paladin_judgementsofthejust","32"}))
                
                    menu["lines"]["2"]["objects"]["2"] = 
                    Octopus.GUI.Builder.new_object("link",Octopus.logic.indexput({"取消",
                    "444444",
                    "runner(scriptname="..CancelScript..",scriptargs="..CancelArgs.. ")"
                
                })
            )
                return menu
                    
                end
            },
            

            ---@return string
            build = function (menu)
                local menutext = ""
                for i=0,Octopus.logic.length(menu["lines"])-1 do
                    local line = menu["lines"][tostring(i)]
                        if line["settings"]["IS_IMAGE"] == "FALSE" then
                            menutext = menutext.."{"..line["settings"]["FONT"]..":" .. line["settings"]["LOCATION"] .. "}"
                            
                            for j=0,Octopus.logic.length(line["objects"])-1 do 
                                
                                local obj = line["objects"][tostring(j)]
                               
                                if obj["type"] == "text" then
                                    Octopus.LOG.writelog(tostring(obj["text"]))
                                    Octopus.LOG.showlog()
                                    menutext = menutext.. "{col:" .. obj["colour"] .. "}"
                                    ..obj["text"] 
                                    .."{/col}"
                                elseif obj["type"] == "link" then
                                    menutext = menutext.. "{col:" .. obj["colour"] .. "}"
                                    .. "{link*".. obj["link"] .."*"..obj["text"].."}" 
                                    
                                    .."{/col}"
                                elseif obj["type"] == "icon" then
                                    menutext = menutext .. "{icon"..":"..obj["url"]..":"..tostring(obj["size"]).."}"
                                end
                            end 
                            menutext = menutext.."{/"..line["settings"]["FONT"] .. "}"
                        else
                            local img = line['objects']["0"]
                            menutext = menutext .. "{img:"..img['url']..":"..tostring(img['width'])..":"..tostring(img["height"]).."}"

                        end

                        
                        
                end
                return menutext
            end,
        }

        ,
        Menu = {
    

            set = function (text)
        
                setVar(args,"c","MenuText",text)
                
            end,
            show = function (...)
                --Menu.close()
                Octopus.GUI.Menu.Config.MenuItem_UseAble("FALSE")
                effect("run_workflow",args,"c","open_menu")
            end,
        
            show_without_close = function ()
                effect("run_workflow",args,"c","open_menu")
            end,
        
            open_without_close = function (text)
        
                setVar(args,"c","MenuText",text)
                Octopus.Data.save("OctopusOptions","MENU_RECORD_LATEST","FALSE")
                effect("run_workflow",args,"c","open_menu")
                
            end,
        
            open = function (text)
        
                --Menu.close()
                setVar(args,"c","MenuText",text)
                Octopus.GUI.Menu.Config.MenuItem_UseAble("FALSE")
                Octopus.Data.save("OctopusOptions","MENU_RECORD_LATEST","TRUE")
                effect("run_workflow",args,"c","open_menu")
                
            end,
            disable_record = function ()
                Octopus.Data.save("OctopusOptions","MENU_RECORD_LATEST","FALSE")
            end,
        
            setlatest = function (text)
        
                setVar(args,"c","LatestMenu",text)
                
            end,
        
            openlatest = function ()
        
                Octopus.GUI.Menu.open(getVar(args,"c","LatestMenu"))
                
            end,
        
            close = function ()
                if Octopus.Data.get_value("OctopusOptions","MENU_RECORD_LATEST")=="TRUE" then
                    Octopus.GUI.Menu.setlatest(getVar(args,"c","MenuText"))
                end
                Octopus.GUI.Menu.Config.MenuItem_UseAble("TRUE")
                
        
                effect("run_workflow",args,"c","close_menu")
            end,
        
            linkinfotemplate = {
                
                scriptname = {
                ["1"] = 'DefaultLinkScript',
                ["2"] = 'DefaultLinkScript',
                ["3"] = 'DefaultLinkScript',
                ["4"] = 'DefaultLinkScript',
                ["5"] = 'DefaultLinkScript',
                ["6"] = 'DefaultLinkScript',},
        
                scriptargs = {        
                ["1"] = 'noargs',
                ["2"] = 'noargs',
                ["3"] = 'noargs',
                ["4"] = 'noargs',
                ["5"] = 'noargs',
                ["6"] = 'noargs',
                }
                
        
        
                
        
            },
        
            LinkSetter = function(linkinfo)--传入一个表，{"scriptname" = {["1"] = ...},"scriptargs" = {...}}
        
                linkinfo = Octopus.logic.complete(linkinfo,Octopus.GUI.Menu.linkinfotemplate)
                for i , v in pairs(linkinfo["scriptname"]) do
                    setVar(args,"c","MenuScript_"..i,v)
                end  
                for i , v in pairs(linkinfo["scriptargs"]) do
                    setVar(args,"c","MenuArgs_"..i,v)
                end
                
            end,
            -- setVar(args,"c","ScriptArgs",getVar(args,"c","MenuArgs_input"..getVar(args,"c","link_input_case")))
        
            LinkSetter_input = function (linkinfo) --传入一个表，{"scriptname" = {["1"] = ...},"scriptargs" = {...}}
        
                linkinfo = Octopus.logic.complete(linkinfo,Octopus.GUI.Menu.linkinfotemplate)
                for i , v in pairs(linkinfo["scriptname"]) do
                    
                    setVar(args,"c","MenuScript_input"..i,v)
                end  
                for i , v in pairs(linkinfo["scriptargs"]) do
                    
                    setVar(args,"c","MenuArgs_input"..i,v)
                end
                
            end,
        
            Config = {
                set = function (text,value)
                    setVar(args,"c",text,value)
                end,
        
                CloseAble = function (text)
                    
                    if text == nil then
                        return getVar(args,"c","Menu_CloseAble")
                    else
                        Octopus.GUI.Menu.Config.set("Menu_CloseAble",text)
                    end
                end,
        
        
                MenuItem_UseAble = function (text)
                    if text == nil then
                        return getVar(args,"c","MenuItem_UseAble")
                    else
                        Octopus.GUI.Menu.Config.set("MenuItem_UseAble",text)
                    end
                end,
        
                Confirming = function (text)
                    if text == nil then
                        return getVar(args,"c","Menu_Confirming")
                    else
                        Octopus.GUI.Menu.Config.set("Menu_Confirming",text)
                    end
                end,
        
            },
        
            --用来快速构造一些有功能的菜单
            Builder = {
                
        
                confirmation = function(confirmtype,descripe,warns,scriptname,...)--确认窗口
                --[[
                    会弹出一个确认菜单：
                    你确定要： confirmtype 吗？
                    descripe
                    warns
                    并锁定菜单使其无法被关闭
        
                ]]--
        
                    local menutext = "{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}\n"
                    if type(confirmtype) ~= nil then
                        menutext = menutext.."{h1:c}你确定要：{col:670001} "..confirmtype.." {/col}吗？{/h1}\n"
                    end 
                    if type(descripe) ~= nil then
                        menutext = menutext.."{h2:c}"..descripe.."{/h2}\n"
                    end
                    if type(warns) ~= nil then
                        menutext = menutext.."{h3:c}{col:ff2800}"..warns.."{/col}{/h3}\n"
                    end
                    menutext = menutext.."{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}{h1:c}{link*confirm*{icon:inv_misc_ticket_tarot_twistingnether_01:16}}{col:71ff50}{link*confirm*确定}".."\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t".."{/col}{col:BA2026}{link*cancel*取消}{/col}{link*cancel*{icon:70_inscription_deck_immortality:16}}{/h1}"
                    menutext = menutext.."\n"
                    setVar(args,"c","ScriptName",scriptname)
                    Octopus.GUI.Menu.Config.Confirming("TRUE")
        
                    return menutext
        
        
        
                end,
        
                choose = function (objects,lore,afterrun)--objtemplate = {["1"] = {text = "",scriptname = "",scriptargs = "",},}
                    local title = "{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}{h1:c}请选择："..lore.."{/h1}{h2:c}\n"
                    local text = title
        
                    local o = ""
                    local c = 0
                    local pc = 0
                    local p = 1
                    local lc = 0
                    local pages = {}
                    pages["1"] = {                        
                        linksetter = {
                        scriptname = {},
                        scriptargs = {}
                    }}
                    
                    while c < Octopus.logic.length(objects) do
                        
                        c = c + 1
                        o = objects[tostring(c)] ---@class objtemplate
                        if pc < 5 then
                            pc = pc + 1
                            text = text .. o["text"] .. "{link*" .. "link"..tostring(pc) .. "*选择}\n"
                            lc = lc + 1
                            pages[tostring(p)]["linksetter"]["scriptname"][tostring(lc)] = "choose_selecetedvalue"
                            pages[tostring(p)]["linksetter"]["scriptargs"][tostring(lc)] = o["scriptargs"]
        
        
                            
                            
                        end
                        if pc == 5 or c + 1 > Octopus.logic.length(objects) then
                            
                            pc = 0
                            lc = 0
                            text = text .. "\n{link*choose.last*上一页} 第 "..tostring(p) .." 页 {link*choose.next*下一页}\n\n"
                            text = text ..Octopus.Assests.Colour.paint("{link*choose.cancel*取消选择}\n",Octopus.Assests.Colour.DARK_RED)
                            text = text .. "{/h2}{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}"
                            
                            pages[tostring(p)]["text"] = text
                            text = title
                            p = p + 1
                            pages[tostring(p)] = {
                                linksetter = {
                                    scriptname = {},
                                    scriptargs = {}
                                }
                            }
                    end
                    setVar(args,"c","temp_choose_pages",Octopus.logic.tostring(pages))
                    Octopus.GUI.Menu.open(pages["1"]["text"])
                    Octopus.GUI.Menu.LinkSetter(pages["1"]["linksetter"])
                    setVar(args,"c","choose_afterrun",afterrun)
                    setVar(args,"c","choose_nowopenpage","1")
                    
        
                end
        
        
                end,
                ---@param objects table
                ---@param lore string
                ---@param afterrun string the name of the script that will execute after cancel 
                ---@param maxinpage integer max objects in per page
                choose2 = function (objects,lore,afterrun,maxinpage)--objtemplate = {["1"] = {text = "",scriptname = "",scriptargs = "",},}
                    local title = "{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}{h1:c}请选择："..lore.."{/h1}{h2:c}\n"
                    local text = title
        
                    local o = ""
                    local c = 0
                    local pc = 0
                    local p = 1
                    local pages = {}
                    maxinpage = maxinpage or 5
                    pages["1"] = {}
                    
                    while c < Octopus.logic.length(objects) do
                        c = c + 1
                        o = objects[tostring(c)] ---@class objtemplate
                        if pc < maxinpage then
                            pc = pc + 1
                            text = text .. o["text"] .. "{link*" .. "runner(scriptname="..o["scriptname"]
                            ..",scriptargs="..o["scriptargs"]..")"
                            .."*选择}\n"
                        end
                        if pc == maxinpage or c + 1 > Octopus.logic.length(objects) then
                            
                            pc = 0
                            text = text .. "\n{link*runner(scriptname=choose_page,scriptargs=last)*上一页} 第 "..tostring(p) .." 页 {link*runner(scriptname=choose_page,scriptargs=next)*下一页}\n\n"
                            text = text ..Octopus.Assests.Colour.paint("{link*runner(scriptname=choose_cancel)*取消选择}\n",Octopus.Assests.Colour.DARK_RED)
                            text = text .. "{/h2}{img:Interface\\QUESTFRAME\\UI-HorizontalBreak:312:64}"
                            
                            pages[tostring(p)]["text"] = text
                            --nextpage
                            text = title
                            p = p + 1
                            pages[tostring(p)] = {}
                    end
                    Octopus.Data.save("temp_choose_pages",nil,pages)
                    Octopus.GUI.Menu.open(pages["1"]["text"])
                    setVar(args,"c","choose_nowopenpage","1")
                    setVar(args,"c","choose_afterrun",afterrun)
                    
        
                end
        
        
                end
        
        
        
        
                
        
            },
            choose_getvalue = function ()
                local a = getVar(args,"c","temp_choose_result")
                
                setVar(args,"c","temp_choose_result","cancel")
        
                return a
                
            end,

        
    },}
    ,
    Data = {

        ---详见Octopus(Data Logic).png


        delete_folder = function (folder)
            setVar(args,"c","."..folder,nil)
        end,
        ---保存一个值到指定目录
        
        ---@param folder string
        ---@param name string|nil
        ---@param value any
        save = function (folder,name,value)
            --.foldername 储存一个表
            --使用get_value直接获取表中的一个值
            --如果name为nil，直接覆盖目录为value
            --如果name不为nil，则修改name的值
            local folderlist = Octopus.logic.strtotable(getVar(args,"c",".*folderlist"))
        
            local function hasregisteredfolder(foldername)
                local folderlist = Octopus.logic.strtotable(getVar(args,"c",".*folderlist"))        --带*号的是不开放式目录
                if folderlist ~= nil then
                    for i,v in pairs(folderlist) do
                        if v == foldername then
                            return true
                        end
                    end
                end
                return false
            end
        
            local function init()
                setVar(args,"c",".*folderlist","{}")
            end
        
            local function registerfolder(foldername)
                --{[1]=xxx}
                folderlist[#folderlist+1] = foldername
                setVar(args,"c",".*folderlist",Octopus.logic.tostring(folderlist))
                setVar(args,"c","."..foldername,"{}")
        
            end
        
            
            if tostring(getVar(args,"c",".*folderlist")) == "nil" then
                init()
            end
        
            --如果是value表就把它变成可储存字段
            if type(value) ~= "string" then
                if type(value) == 'number' then
                    value = tostring(value)
                elseif type(value) == 'function' then--函数是取返回值
                    value = value()
                end
            end
        
            
            if tostring(name) ~= 'nil' then
                local fd = Octopus.logic.strtotable(getVar(args,"c","."..folder))
---@diagnostic disable-next-line: need-check-nil
                fd[name] = value
                setVar(args,"c","."..folder,Octopus.logic.tostring(fd))
            else
                setVar(args,"c","."..folder,Octopus.logic.tostring(value))
            end
            
            if hasregisteredfolder(folder) then
                registerfolder(folder)
            end
        
            
        
        
            
        end,
        ---@param folder string
        ---@param name string
        get_value = function (folder,name)
            return Octopus.logic.strtotable(getVar(args,"c","."..folder))[name]
        end,
        ---@param folder string
        get_folder = function (folder)
        
            return Octopus.logic.strtotable(getVar(args,"c","."..folder))
        
            
    end,},
        cleartemp = function ()
            for i ,folder_name in pairs(Octopus.Data.get_folder("*folderlist")) do
                if string.find(folder_name,"temp_") ~= nil then
                    Octopus.Data.save(folder_name,nil,nil)
                end
            end
        end,
        
    
    Sound = {

        playSoundSelf = function (soundID)
    
            effect("sound_id_self",args,nil,soundID)
            
        end
    
    },
    Listener = {
        types = {
            "OnMessage_say","OnMessage_yell","OnMessage_emote"
            ,"OnMessage_raid","OnMessage_party","OnMessage_raidwarning",
            "OnMessage_guild","OnMessage_whisper","OnPlayerStartMove","Always"
        },
        __init = function ()

            local R_Listner = Octopus.Data.get_folder("RegisteredListener")
            if R_Listner == nil or R_Listner == {} or not R_Listner["OnMessage_say"] then
                R_Listner = {
                    OnMessage_say = {},
                    OnMessage_yell = {},
                    OnMessage_emote = {},
                    OnMessage_raid = {},
                    OnMessage_party = {},
                    OnMessage_raidwarning = {},
                    OnMessage_guild = {},
                    OnMessage_whisper = {},
                    OnPlayerStartMove = {},
                    Always = {}
                
                }
                Octopus.Data.save("RegisteredListener",nil,R_Listner)
            end
            
        end,
---@diagnostic disable-next-line: undefined-doc-name
        --使一个监听器开始工作
        --监听器会在监听到时，运行记录的脚本
        --
        open = function (Aargs)---@param Aargs listener listner:{type = ,scriptname = scriptname,id = id}
            Octopus.Listener.__init()
            local type = Aargs["type"]
            local id = Aargs["id"]
            setVar(args,"c",type,"TRUE")
            local R_Listner = Octopus.Data.get_folder("RegisteredListener")
            R_Listner[type][id] = {}
            R_Listner[type][id]["scriptname"] = Aargs["scriptname"] 
            R_Listner[type][id]["scriptargs"] = Aargs["scriptargs"]
            Octopus.Data.save("RegisteredListener",nil,R_Listner)
        end,
---@diagnostic disable-next-line: undefined-doc-name
        close = function (Aargs) ---@param Aargs class listner:{type = ...,scriptname = ...,id = ...}
            local R_Listner_type = Octopus.Data.get_value("RegisteredListener",Aargs["type"])
            R_Listner_type["id"] = nil
            if Octopus.logic.length(R_Listner_type) == 0 then
                setVar(args,"c",Aargs["type"],"FALSE")
            end
                Octopus.Data.save("RegisteredListener",Aargs["type"],R_Listner_type)
        end,

        ---@param type string the type of the listner (OnMessage_say/OnMessage_yell/OnMessage_emote/OnMessage_raid/OnMessage_party/OnPlayerStartMove/Always)
        ---@param scriptname string the script that the listner will execute
        ---@param id string the id of the lisner,must be unique
        new = function (type,scriptname,scriptargs,id) 
            if 
                Octopus.logic.conclude(Octopus.Listener.types,type)
            then
                return {type = type,scriptname = scriptname,scriptargs = scriptargs,id = id}
            else
                error("ERROR:Invaild Listener Type")
            end
        end
        
        
        
    },
    Operands = {
        refresh = function ()
            effect("run_workflow",args,"c","refresh_operands")

        end,
        PlayerName = function ()
            return  getVar(args,"c","Octopus_PlayerName")
        end,
        TargetName = function ()
            Octopus.Operands.refresh()
            return getVar(args,"c","Octopus_TargetName")
            
        end
    },
    
    RunScript = function (scriptname,scripts)  --运行指定脚本

        for i,v in pairs(scripts) do
    
            if type(v) == "table" then
                Octopus.RunScript(scriptname,v)
    
            else
                if i == scriptname then
                    scripts[scriptname]()
                end
            end
    
        end
    
    end,
    BasicScripts = {
        menuclose = function ()

            if Octopus.GUI.Menu.Config.Confirming() == "TRUE" then
                Octopus.Sound.playSoundSelf(Octopus.Assests.SoundLibrary.UI.CANCEL)
                
                effect("text",args,"已取消！","4")

                Octopus.GUI.Menu.Config.Confirming("FALSE")

                Octopus.GUI.Menu.openlatest()
            end

            

            if Octopus.GUI.Menu.Config.CloseAble() ~= "FALSE" then


                Octopus.GUI.Menu.setlatest(getVar(args,"c","MenuText"))

                Octopus.GUI.Menu.Config.MenuItem_UseAble("TRUE")
                
            
            else
                
                Octopus.GUI.Menu.show()

                Octopus.Sound.playSoundSelf(Octopus.Assests.SoundLibrary.UI.CANCEL)
                
                effect("text",args,"此页面不能被关闭！","4")

            end
        end,
        ListnerRelatives = {
            ListenerHeard = function ()
            
                local listners = Octopus.Data.get_folder("RegisteredListener")
                local atype = getVar(args,"c","ListenerScriptArgs")
                if listners[atype] then
                    for i,v in pairs(listners[atype]) do
                        ScriptArgs = v["scriptargs"]
                        Octopus.RunScript(v["scriptname"],SCRIPTS)
                    end
                end
            end
        },

        
        windup = function ()
            setVar(args,"c","ScriptName","nil")
            setVar(args,"c","ScriptArgs","nil")
            setVar(args,"c","ScriptFilter","nil")
            setVar(args,"c","ListenerScriptName",nil)
        end

    }

    
    
}