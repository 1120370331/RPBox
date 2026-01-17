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
    
            local parts = {"{"}

            local escape_value = function(v)
                -- Escape in a specific order to avoid re-escaping
                return tostring(v):gsub("%%", "%%p"):gsub("$", "%%s"):gsub("^", "%%c")
            end

            for i,v in pairs(tb) do
                local key = tostring(i)
    
                if type(v) == "string" then
                    table.insert(parts, "[" .. key .. "] = $" .. escape_value(v) .. "^,")
                elseif type(v) == "table" then
                    table.insert(parts, "[" .. key .. "] = " .. Octopus.logic.tostring(v) .. ",")
                else
                    table.insert(parts, "[" .. key .. "] = $" .. escape_value(v) .. "^,")
                end
            end
    
            table.insert(parts, "}")
    
            return table.concat(parts)
    
        end,
    
        strtotable = function (tstring)
            
            local unescape_value = function(v)
                -- Unescape in the reverse order
                return v:gsub("%%c", "^"):gsub("%%s", "$"):gsub("%%p", "%%")
            end

            local function parse(s)
                if type(s) ~= "string" or not s:match("^{.+}$") then return {} end

                local tbl = {}
                -- Trim the outer braces and the final comma if it exists
                s = s:sub(2, -2):gsub(",$", "")

                local balance = 0
                local last_cut = 1

                for i = 1, #s do
                    local char = s:sub(i, i)
                    if char == '{' then
                        balance = balance + 1
                    elseif char == '}' then
                        balance = balance - 1
                    elseif char == ',' and balance == 0 then
                        -- We found a key-value pair
                        local pair = s:sub(last_cut, i - 1)
                        local key, value_str = pair:match("%[([^%[%]]+)%]%s*=%s*(.+)")
                        
                        if key and value_str then
                           if value_str:sub(1,1) == '{' then
                                tbl[key] = parse(value_str)
                           else
                                local val = value_str:match("%$([^%^]*)%^")
                                tbl[key] = unescape_value(val or "")
                           end
                        end
                        last_cut = i + 1
                    end
                end

                -- Process the last pair
                local pair = s:sub(last_cut)
                local key, value_str = pair:match("%[([^%[%]]+)%]%s*=%s*(.+)")
                if key and value_str then
                   if value_str:sub(1,1) == '{' then
                        tbl[key] = parse(value_str)
                   else
                        local val = value_str:match("%$([^%^]*)%^")
                        tbl[key] = unescape_value(val or "")
                   end
                end
                
                return tbl
            end

            return parse(tstring)
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
            "OnMessage_say","OnMessage_yell","OnMessage_emote","OnMessage_text_emote"
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
                    OnMessage_text_emote = {},
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
ScriptName = getVar(args,"c","ScriptName")
ScriptArgs = getVar(args,"c","ScriptArgs")

ListenerScriptName = getVar(args,"c","ListenerScriptName")


Octopus.RunScript(ScriptName,Octopus.BasicScripts)
RPRecorder = {
    Menu = {
        MainMenu = function ()
            local menu = Octopus.GUI.Builder.new_menu(10)
            menu["lines"]["0"]["settings"]["FONT"] = "h1"
            
            menu["lines"]["0"]["objects"] = {
                ["0"] = {
                    ["type"] = "text",
                    ["colour"] = "000000",
                    ["text"] = "欢迎使用RP记录工具\n\n"
                }
            }
            menu["lines"]["1"]["objects"] = {["0"] = {
                ["type"] = "link",
                ["text"] = "{icon:spell_nature_elementalshields:16}",
                ["colour"] = "146212",
                ["link"] = "link_input1"
            },["1"] = {
                ["type"] = "link",
                ["text"] = "创建一个记录",
                ["colour"] = "146212",
                ["link"] = "link_input1"
            },["2"] = {
                ["type"] = "link",
                ["text"] = "{icon:spell_nature_elementalshields:16}",
                ["colour"] = "146212",
                ["link"] = "link_input1"
            ,},["3"] = Octopus.GUI.Builder.new_object("text",{[0]="\n\n"})
            }
            menu["lines"]["2"]["objects"] = {["0"] = {
                ["type"] = "link",
                ["text"] = "{icon:inv_engineering_90_scope_blue:16}",
                ["colour"] = "ffffff",
                ["link"] = "runner(scriptname=OpenRecordList)"
            },["1"] = {
                ["type"] = "link",
                ["text"] = "查看已有记录",
                ["colour"] = "ffffff",
                ["link"] = "runner(scriptname=OpenRecordList)"
            },["2"] = {
                ["type"] = "link",
                ["text"] = "{icon:inv_engineering_90_scope_blue:16}",
                ["colour"] = "ffffff",
                ["link"] = "runner(scriptname=OpenRecordList)"
            }}
            menu["lines"]["3"]["objects"] = {["0"] = {
                ["type"] = "link",
                ["text"] = "{icon:inv_misc_gear_01:16}",
                ["colour"] = "000000",
                ["link"] = "runner(scriptname=OpenSettingMenu)"
            },
            ["1"] = {
                ["type"] = "link",
                ["text"] = "进入设置界面",
                ["colour"] = "000000",
                ["link"] = "runner(scriptname=OpenSettingMenu)"
            },
            ["2"] = {
                ["type"] = "link",
                ["text"] = "{icon:inv_misc_gear_01:16}",
                ["colour"] = "000000",
                ["link"] = "runner(scriptname=OpenSettingMenu)"
            }}
            menu["lines"]["4"]["objects"] = {["0"] = {
                ["type"] = "link",
                ["text"] = "{icon:ability_garrison_orangebird:16}",
                ["colour"] = "000000",
                ["link"] = "runner(scriptname=OpenGuideMenu)"
            },
            ["1"] = {
                ["type"] = "link",
                ["text"] = "打开使用指南",
                ["colour"] = "000000",
                ["link"] = "runner(scriptname=OpenGuideMenu)"
            },
            ["2"] = {
                ["type"] = "link",
                ["text"] = "{icon:ability_garrison_orangebird:16}",
                ["colour"] = "000000",
                ["link"] = "runner(scriptname=OpenGuideMenu)"
            }}

            Octopus.GUI.Menu.open(Octopus.GUI.Builder.build(menu))
        end,
        RecordList = function ()
            local menu = Octopus.GUI.Builder.new_menu()
            local list = RPRecorder.Records.Library.getlist()
            local linecount = 0
            local bd = Octopus.GUI.Builder
            
            menu["lines"]["0"] = Octopus.GUI.Builder.new_line()
            linecount = linecount + 2
            menu["lines"]["0"]["objects"]["0"] = bd.new_object("text",Octopus.logic.indexput({"记录列表"}))
            menu["lines"]["1"] = Octopus.GUI.Builder.new_line()
            if getVar(args,"c","Recording") == "TRUE" then
                menu["lines"]["1"]["objects"]["0"] = bd.new_object("text",Octopus.logic.indexput({"正在记录："..getVar(
                    args,"c","RecordingRecorder"
                )})
                )
                Octopus.logic.append(
                    menu["lines"]["1"]["objects"],
                    bd.new_object("link",Octopus.logic.indexput(
                    {"停止记录","7e250a","runner(scriptname=StopRecord,scriptargs="..getVar(
                        args,"c","RecordingRecorder")
                    ..")",})
                ))
            else
                menu["lines"]["1"]["objects"]["0"] = bd.new_object("text",Octopus.logic.indexput({"还未开始记录！","FFFFFF"})
                )
            end
            for i,v in pairs(list) do
                local lc = tostring(linecount)
                local index = tostring(linecount-1)

                menu["lines"][lc] = Octopus.GUI.Builder.new_line()
                local tgl = menu["lines"][lc]
                Octopus.logic.append(tgl["objects"],bd.new_object("text",Octopus.logic.indexput({index.."："..v})))
                Octopus.logic.append(tgl["objects"],bd.new_object("text",Octopus.logic.indexput({
                    "(长度："..tostring(
                        RPRecorder.Records.piece_num(v)
                    )..")"
                    ,"444444"
            })))
                Octopus.logic.append(
                    tgl["objects"],
                    bd.new_object("link",Octopus.logic.indexput(
                    {"开始记录","269859","runner(scriptname=StartRecord,scriptargs="..v..")",})
                ))
                
                Octopus.logic.append(
                    tgl["objects"],
                    bd.new_object("link",
                    Octopus.logic.indexput({"回放","3d3f3f","runner(scriptname=ReplayRecord,scriptargs="..v..")"})
                ))

                Octopus.logic.append(
                    tgl["objects"],
                    bd.new_object("link",
                    Octopus.logic.indexput({"删除","ac0e0e","runner(scriptname=DeleteRecordStart,scriptargs="..v..")"})
                ))
                linecount = linecount + 1
            

            end
            menu["lines"][tostring(Octopus.logic.length(menu["lines"]))] = bd.new_line()
        
            menu["lines"][tostring(Octopus.logic.length(menu["lines"])-1)]["objects"]["0"] = bd.new_object(
                "link",Octopus.logic.indexput(
                    {"返回主界面","3d3f3f","runner(scriptname=OpenMainMenu)"}
                )
            )
            Octopus.GUI.Menu.open(Octopus.GUI.Builder.build(menu))
        end,
        SettingMenu = function()
            RPRecorder.Config.refresh_config()
            local menu = Octopus.GUI.Builder.new_menu()
            menu["lines"]["0"] = Octopus.GUI.Builder.new_line()
            menu["lines"]["0"]["objects"]["0"] = Octopus.GUI.Builder.new_object(
                "text",Octopus.logic.indexput({"设置菜单\n\n"})
            )
            local lc = 1
            for i,v in pairs(RPRecorder.Config.configs) do 
                if v == "TRUE" then
                    v = "{col:14d40d}是{/col}"
                else
                    v = "{col:d40d0d}否{/col}"
                end
                Octopus.logic.append(menu["lines"],Octopus.GUI.Builder.new_line())
                local objs = menu["lines"][tostring(lc)]["objects"]
                Octopus.logic.append(objs,
                Octopus.GUI.Builder.new_object("text",Octopus.logic.indexput(
                    {RPRecorder.Config["config_lang"][i]}
                )))
                Octopus.logic.append(objs,
                Octopus.GUI.Builder.new_object("text",Octopus.logic.indexput(
                    {v}
                )))
                Octopus.logic.append(objs,
                Octopus.GUI.Builder.new_object("link",Octopus.logic.indexput(
                    {"切换","444444","runner(scriptname=SwitchConfig,scriptargs="
                ..i..")"}
                )))


                lc = lc + 1
            end
            menu["lines"][tostring(Octopus.logic.length(menu["lines"]))] = Octopus.GUI.Builder.new_line()
        
            menu["lines"][tostring(Octopus.logic.length(menu["lines"])-1)]["objects"]["0"] = Octopus.GUI.Builder.new_object(
                "link",Octopus.logic.indexput(
                    {"返回主界面","3d3f3f","runner(scriptname=OpenMainMenu)"}
                )
            )
            Octopus.GUI.Menu.open(Octopus.GUI.Builder.build(menu))
        end,
        Guide = function ()
            local menu = Octopus.GUI.Builder.new_menu(10)
            local lines = menu["lines"]
            lines["0"]["objects"]["0"] = Octopus.GUI.Builder.new_object("text",
        Octopus.logic.indexput({
            "RP记录员使用指南"
        })
        )
        lines["1"]["settings"]["FONT"] = "h3"
        lines["1"]["objects"]["0"] = Octopus.GUI.Builder.new_object("text",
        Octopus.logic.indexput({
            "\n\n开始记录：在主界面创建一个记录，然后打开记录列表，找到对应记录并开始记录，便可以自动记录信息并存储在该记录中了。\n"
            .."回放记录：在记录列表中可以随时回放记录。\n"
            .."设置：可以设置监听哪些频道的信息，不被开放的频道将不会被记录。\n"
            .."{col:c4191f}保存记录：强烈建议您不要在同一个人物卡中建立过多的记录（经测试，一万条以内不会出现太大的问题，但本物品限制最大数量为一千条以免出现安全问题），不同人物卡不共享记录数据库，您可以将记录数据分人物卡管理，或建立专门的记录仓库。您可以通过在激活本剧本时保存人物卡来起到保存记录的功能。若因为单人物卡记录过多导致坏卡，则可能会丢失所有人物卡信息！这是由TRP3本身限制造成的。请做好人物卡备份工作！{/col}"
            .."\n作者联系邮箱：1120370331@qq.com"
            .."\n如有BUG，欢迎反馈！"
            .."\n"
        })  
        )
        menu["lines"][tostring(Octopus.logic.length(menu["lines"]))] = Octopus.GUI.Builder.new_line()
        
        menu["lines"][tostring(Octopus.logic.length(menu["lines"])-1)]["objects"]["0"] = Octopus.GUI.Builder.new_object(
            "link",Octopus.logic.indexput(
                {"返回主界面","3d3f3f","runner(scriptname=OpenMainMenu)"}
            )
        )
        Octopus.GUI.Menu.open(Octopus.GUI.Builder.build(menu))
        end
    },
    Records = {
        Library = {
        
            ---@param name string
            append = function (recorder)
                local RecordList = Octopus.Data.get_folder("RecordList") or {}
                Octopus.logic.append(RecordList,recorder["name"])
                Octopus.Data.save("Record_"..recorder["name"],nil,recorder)
                Octopus.Data.save("RecordList",nil,RecordList)
            end,
            ---@return table
            get = function (name)
                local record = Octopus.Data.get_folder("Record_"..name)
                record["chunk_num"] = tonumber(record["chunk_num"])
                record["data_size"] = tonumber(record["data_size"])
                return record

            end,
            update = function (record)
                Octopus.Data.save("Record_"..record["name"],nil,record)
            end,
            getlist= function ()
                return Octopus.Data.get_folder("RecordList")
            end,
            delete = function (name)
                local RecordList = Octopus.Data.get_folder("RecordList")
                if getVar(args,"c","RecordingRecorder") == name then
                    SCRIPTS.StopRecord()
                    setVar(args,"c","Recording","FALSE")
                    setVar(args,"c","RecordingRecorder","nil")

                end
                local record = RPRecorder.Records.Library.get(name)
                local i=0
                while i < record["chunk_num"] do
                    i = i + 1
                    Octopus.Data.delete_folder("Record_"..name.."_".."chunk"..tostring(i))
                end
                Octopus.logic.remove(RecordList,Octopus.logic.getKeyByValue(RecordList,name))
                Octopus.Data.delete_folder("Record_"..name)
                
                Octopus.Data.save("RecordList",nil,RecordList)

                if getVar(args,"c","Player_playingrecord") == name then
                    Octopus.LOG.error("因为源记录被删除，回放被终止。")
                    effect("run_workflow",args,"c","stopplay")
                end
                
            end

        },
        --返回记录数目
        piece_num = function ( name )

            return (RPRecorder.Records.Library.get(name)["chunk_num"] -1 )*50 + 
            Octopus.logic.length(RPRecorder.Records.chunk_get_newest(name))
        end,
        new_record = function (name)
                return {
                    ["name"] = name,
                    ["data_size"] = 0,
                    ["chunk_num"] = 1,
                }
        end,
        new_data = function (case,speaker,text)
            local timestamp = math.floor(time())
            return {
                ["case"] = case,
                ["speaker"] = speaker,
                ["text"] = text,
                ["timestamp"] = tostring(timestamp)
            }
        end,
        ---@return string
        data_decode_text = function (data)
            
            local pattern = ""
            if data["case"] == "say" then
                pattern = "说："
            elseif data["case"] == "yell" then
                pattern = "喊："
            elseif data["case"] == "whisper" then
                pattern = "悄悄说："
            end
            return pattern..data["text"]
        end,
        start_record = function(name)
            setVar(args,"c","RecordingRecorder",name)
            setVar(args,"c","Recording","TRUE")

        end,
        stop_record = function()
            setVar(args,"c","RecordingRecorder",nil)
            setVar(args,"c","Recording","FALSE")

        end,
        --采用区块链储存数据
        --新建并保存区块
        chunk_new = function (name)
            local ri = RPRecorder.Records.Library.get(name)
            local new_chunk = {}
            ri["chunk_num"] = ri["chunk_num"] + 1
            Octopus.Data.save("Record_"..name.."_".."chunk"..tostring(ri["chunk_num"]),nil,new_chunk)




            
            RPRecorder.Records.Library.update(ri)
            
        end,
        --通过索引找寻应取区块
        ---@return table 
        chunk_get = function (name,index)
            local ri = RPRecorder.Records.Library.get(name)
            
            local chunk = Octopus.Data.get_folder(
                "Record_"..name.."_".."chunk"..tostring(index))
            
            return chunk
        end,
        chunk_get_newest = function (name)
            local ri = RPRecorder.Records.Library.get(name)
            
            local chunk = Octopus.Data.get_folder(
                "Record_"..name.."_".."chunk"..tostring(ri["chunk_num"]))
            return chunk
        end,
        chunk_save = function (name,chunk,index)
            index = index or RPRecorder.Records.Library.get(name)["chunk_num"]
            
            Octopus.Data.save("Record_"..name.."_".."chunk"..tostring(index),nil,chunk)
            
        end,
        chunk_data_auto_append = function (name,data)
            local ri = RPRecorder.Records.Library.get(name)
            local chunk = RPRecorder.Records.chunk_get_newest(name)
            local length = Octopus.logic.length(chunk)
            
            if length == 50 then
                RPRecorder.Records.chunk_new(name)
                chunk = RPRecorder.Records.chunk_get_newest(name)
            end
            Octopus.logic.append(chunk,data)
           
            RPRecorder.Records.chunk_save(name,chunk)
          

        end,
        chunk_data_get = function (name,index)
            local ri = RPRecorder.Records.Library.get(name)

        end,
        -- 生成消息签名用于去重
        generate_message_signature = function (speaker, text, timestamp)
            return speaker .. "|" .. text .. "|" .. timestamp
        end,
        -- 检查消息是否为重复
        is_duplicate_message = function (speaker, text, timestamp)
            local cache = Octopus.Data.get_folder("MessageDedupeCache") or {}
            local signature = RPRecorder.Records.generate_message_signature(speaker, text, timestamp)

            -- 清理超过5秒的旧缓存
            local current_time = math.floor(time())
            for sig, cached_time in pairs(cache) do
                if tonumber(cached_time) and (current_time - tonumber(cached_time)) > 5 then
                    cache[sig] = nil
                end
            end

            -- 检查是否存在
            if cache[signature] then
                return true
            end

            -- 添加到缓存
            cache[signature] = tostring(timestamp)
            Octopus.Data.save("MessageDedupeCache", nil, cache)
            return false
        end

    },
    SpeechPlayer = {
        --用内置播放器播放一个记录
        StartPlay = function (record)
            --第一步，填装演员、预备演员
            --第二步，打开播放器
            --第三步，演员下台、预备演员上台、下一个预备(NextPrepare)、order设置为2
            setVar(args,"c","Player_order","0")
            
            local datas = RPRecorder.Records.chunk_get(record["name"],1)

            local Player_order = 0
           
            
            setVar(args,"c","Player_playingtext",datas[tostring(Player_order)]["text"])
            setVar(args,"c","Player_playingspeaker",datas[tostring(Player_order)]["speaker"])
            
            local Player_order = Player_order + 1
            
        
            setVar(args,"c","Player_playingrecord",record["name"])
            setVar(args,"c","Player_order",tostring(Player_order))
            effect("run_workflow",args,"c","open_player")
        end,
        next = function ()
            local Player_order = tonumber(getVar(args,"c","Player_order"))
            local PlayingRecordName = getVar(args,"c","Player_playingrecord")
            local record = RPRecorder.Records.Library.get(getVar(args,"c","Player_playingrecord"))
            local chunk_index = math.modf(Player_order/50) +1
            local datas = RPRecorder.Records.chunk_get(PlayingRecordName,chunk_index)
            local this_chunk_index = math.fmod(Player_order,50) 
            if Player_order == RPRecorder.Records.piece_num(PlayingRecordName)  then
                effect("run_workflow",args,"c","stopplay")
            else
                Player_order = Player_order + 1
                setVar(args,"c","Player_playingtext",datas[tostring(this_chunk_index)]["text"])
                setVar(args,"c","Player_playingspeaker",datas[tostring(this_chunk_index)]["speaker"])
                setVar(args,"c","Player_order",tostring(Player_order))
                
            end
        end,
        PartA = function ()
            local Player_order = tonumber(getVar(args,"c","Player_order"))
            local record = RPRecorder.Records.Library.get(getVar(args,"c","Player_playingrecord"))
            local datas = record["RecordDatas"]
            if Player_order == Octopus.logic.length(datas) then
                effect("run_workflow",args,"c","stopplay")
            else
                Player_order = Player_order + 1
                setVar(args,"c","Player_preparingtext",datas[tostring(Player_order-1)]["text"])
                setVar(args,"c","Player_preparingspeaker",datas[tostring(Player_order-1)]["speaker"])
                
                setVar(args,"c","Player_order",tostring(Player_order))
                setVar(args,"c","Player_ReadyEnd","TRUE")
            end
            
        end,
        PartB = function ()
            local Player_order = tonumber(getVar(args,"c","Player_order"))
                if Player_order ~= 1 then
                    
                
                local record = RPRecorder.Records.Library.get(getVar(args,"c","Player_playingrecord"))
                local datas = record["RecordDatas"]
                if Player_order == Octopus.logic.length(datas) then
                    effect("run_workflow",args,"c","stopplay")
                else
                    Player_order = Player_order + 1
                    setVar(args,"c","Player_playingtext",datas[tostring(Player_order-1)]["text"])
                    setVar(args,"c","Player_playingspeaker",datas[tostring(Player_order-1)]["speaker"])
                    
                    setVar(args,"c","Player_order",tostring(Player_order))
                    setVar(args,"c","Player_ReadyEnd","TRUE")
                end
            end
            
        end 
    },
    Config  = {
        refresh_config = function ()
            for i,v in pairs(RPRecorder.Config.configs) do
                RPRecorder.Config.configs[i] = getVar(args,"c",i)

            end
            
        end,
        init_config = function ()
            for i,v in pairs(RPRecorder.Config.configs) do
                setVar(args,"c",i,v)

            end
        end,
        ["configs"] = {
            ["LISTEN_SAY"] = "TRUE",
            ["LISTEN_YELL"] = "TRUE",
            ["LISTEN_EMOTE"] = "TRUE",
            ["LISTEN_TEXT_EMOTE"] = "TRUE",
            ["LISTEN_RAIDWARNING"] = "TRUE",
            ["LISTEN_GUILD"] = "FALSE",
            ["LISTEN_PARTY"] = "FALSE",
            ["LISTEN_RAID"] = "FALSE",
            ["LISTEN_WHISPER"] = "TRUE",
        },
        ["config_lang"] = {
            ["LISTEN_SAY"] = "接收来自说话的信息",
            ["LISTEN_YELL"] = "接收来自叫喊的信息",
            ["LISTEN_EMOTE"] = "接收来自表情的信息",
            ["LISTEN_TEXT_EMOTE"] = "接收来自斜杠表情的信息",
            ["LISTEN_RAIDWARNING"] = "接收来自团队警告的信息",
            ["LISTEN_GUILD"] = "接收来自公会的信息",
            ["LISTEN_PARTY"] = "接收来自小队的信息",
            ["LISTEN_RAID"] = "接收来自团队的信息",
            ["LISTEN_WHISPER"] = "接收来自密语的信息",
        }
    }
}



SCRIPTS = {
    ["OpenMainMenu"] = function ()
        RPRecorder.Menu.MainMenu()
    end,
    ["RegisterSayListener"] = function ()
        Octopus.Listener.open(Octopus.Listener.new("OnMessage_say","ListenerHeardSay","111"))
    end,
    ["ListenerProcess"] = function ()
        local sa = ScriptArgs
        local case = ""
        if sa == "OnMessage_say" then
            case = "say"
        elseif sa == "OnMessage_yell" then
            -- body
            case = "yell"
        elseif sa == "OnMessage_emote" then
            case = "emote"
        elseif sa == "OnMessage_text_emote" then
            case = "text_emote"
        elseif sa == "OnMessage_raid" then
            case = "raid"
        elseif sa == "OnMessage_party" then
            case = "party"
        elseif sa == "OnMessage_raidwarning" then
            case = "raidwarning"
        elseif sa == "OnMessage_guild" then
            case = "guild"
        elseif sa == "OnMessage_whisper" then
            case = "whisper"
        end
        local key = string.upper("LISTEN_"..case)
        RPRecorder.Config.refresh_config()

        
        if getVar(args,"c","Recording") == "TRUE" then
            if RPRecorder.Config.configs[key] == "TRUE" then

                local speechname = getVar(args,"c","Listener_2")
                local text = getVar(args,"c","Listener_1")

                local data = RPRecorder.Records.new_data(case,speechname,text)

                -- 去重检查：基于秒级时间戳
                local timestamp = tonumber(data["timestamp"])
                if not RPRecorder.Records.is_duplicate_message(speechname, text, timestamp) then
                    text = RPRecorder.Records.data_decode_text(data)
                    data["text"] = RPRecorder.Records.data_decode_text(data)
                    RPRecorder.Records.chunk_data_auto_append(getVar(args,"c","RecordingRecorder"),data)
                end
            
                
                
                
                
                
            
                
               
                
                
                
            end

        end
    end,

    ["InputRecorderNameEnd"] = function ()
        local input = getVar(args,"c","temp_input")
        local list = RPRecorder.Records.Library.getlist()
        if not Octopus.logic.conclude(
            list,
            input
        ) then
            if Octopus.logic.length(list) < 1000 then

        
                local recorder = RPRecorder.Records.new_record(input)

                RPRecorder.Records.Library.append(recorder) 
            else 
                Octopus.LOG.error("您已达到最大记录数量限制，请查阅使用指南。")
            end
        else
            Octopus.LOG.error("已存在此名称记录，请先删除。")
        end 
        RPRecorder.Menu.RecordList()
    end,
    ["OpenRecordList"] = function()
        RPRecorder.Menu.RecordList()
    end,


    ["PlayerPlay"] = function ()
        RPRecorder.SpeechPlayer.next()
    end,

    ["open_latest_menu_bycancel"] = function () 
        RPRecorder.Menu.MainMenu()
    end,
    ["ReplayRecord"] = function ()
        local RecordName = ScriptArgs
        if RPRecorder.Records.piece_num(RecordName) > 1 then
            RPRecorder.SpeechPlayer.StartPlay(
                RPRecorder.Records.Library.get(RecordName)
            )
        else
            Octopus.LOG.error("记录的长度太短。（至少为2）")
        end
        end,
    ["DeleteRecordStart"] = function ()
        local menu = Octopus.GUI.Builder.preset_menu.confirmation(
            "你确定要删除此记录吗？","这将是不可逆的，请确保您已备份。",
            "DeleteRecordConfirm",ScriptArgs,"OpenRecordList","1")
        Octopus.GUI.Menu.open(Octopus.GUI.Builder.build(menu))

        

        
    end,
    ["StartRecord"] = function ()
        if getVar(args,"c","Recording") ~= "TRUE" then
            RPRecorder.Records.start_record(ScriptArgs)
            effect("text",args,"已为您开始记录！","2")
            effect("run_workflow",args,"c","startrecord")
            RPRecorder.Menu.RecordList()
        else
            Octopus.LOG.error("同时只能进行一个记录哦。")
        end
    end,
    ["StopRecord"] = function ()
        RPRecorder.Records.stop_record()
        effect("text",args,"记录已停止。","2")
        effect("run_workflow",args,"c","stoprecord")
        RPRecorder.Menu.RecordList()
    end,
    ["DeleteRecordConfirm"] = function ()
        RPRecorder.Records.Library.delete(ScriptArgs)
        RPRecorder.Menu.RecordList()
    end,
    ["Init"] = function ()
        for i,v in pairs (Octopus.Listener.types) do

            Octopus.Listener.open(Octopus.Listener.new(v,"ListenerProcess",v,tostring(i)))
        end
        RPRecorder.Config.init_config()

    end,
    ["OpenSettingMenu"] = function ()
        RPRecorder.Menu.SettingMenu()
    end,
    ["SwitchConfig"] = function ()
        RPRecorder.Config.refresh_config()

        local tg = RPRecorder.Config.configs[ScriptArgs]
        local ban = ""
        if  tg == "TRUE" then
            ban = "FALSE"
        else
            ban = "TRUE"
        end
        setVar(args,"c",ScriptArgs,ban)
        RPRecorder.Config.refresh_config()
        RPRecorder.Menu.SettingMenu()
    end,
    ["OpenGuideMenu"] = function ()
        RPRecorder.Menu.Guide()
    end,
    ["ClearTemp"] = function ()
        --遭遇疑难问题时的可能解决方案
        setVar()
        
    end,
}
Octopus.RunScript(ListenerScriptName,Octopus.BasicScripts)
Octopus.RunScript(ScriptName,SCRIPTS)
Octopus.BasicScripts.windup()