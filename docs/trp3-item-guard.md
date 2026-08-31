# RPBox TRP3 对象与光环防护

## RPBox_Addon 集成方案

当前正式入口已经集成到 [`addon/RPBox_Addon/ItemGuard.lua`](../addon/RPBox_Addon/ItemGuard.lua)，随 RPBox 自动加载，不需要 `/run` 或检测器道具。

当前集成范围：

- 新接收或更新的对象先进入 `unscanned` 状态，默认不信任并阻止执行；扫描通过后自动恢复图标、可使用状态并转为 `trusted`；
- 包装 TRP3 对象注册入口，覆盖聊天链接导入、交换、共享仓库等接收路径；无对象 ID 的更新事件还会通过对象引用变化补扫；
- 可达步骤环、递归工作流和异常展开只记结构分，不会单独隔离；
- 循环/递归路径中实际出现 `item_add` 时叠加恶意行为分并隔离；
- `item_loot(isDrop=true)` 纳入地面掉落行为分，循环掉落和超量槽位/数量会隔离；
- 单次 `item_add` 101–1000 只增加行为分，超过 1000 才硬阻断；对象内效果总数只报告，不再一刀切；
- 运行时限制 5 秒与 60 秒窗口内的 `item_add`、地面掉落和实际背包写入；递归调用本身只记录，不触发隔离；
- 循环或递归中的高频声音若缺少有效停止路径会隔离；运行时第一次超频只阻止声音，重复超频才隔离道具；
- 只在真实摧毁回调 `LI.OD` 中检测自我再生，普通摧毁奖励和纯递归不会单独隔离；
- 区分工作流临时变量与对象、战役持久变量，对动态键、值增长和运行时写入体积实施配额；
- 光环的应用、事件、Tick、到期和取消工作流进入同一套行为扫描，事件处理器会被加入可达入口；
- 光环取消工作流直接或间接重新施加自身时硬隔离；运行时还会在取消调用及后续 2 秒观察窗内拦截 Lua/短延时回调重新施加同一光环；
- 已隔离光环的脚本不再执行，活动光环会通过无回调的移除入口清理，避免取消回调再次触发恶意行为；
- 文档渲染前检查单页、总正文、页数及变量展开后的实际大小，超过 512 KiB/页、2 MiB/对象或 256 页时按崩溃级风险硬隔离；
- 对象、容器、光环及新写入变量检查单值、累计大小、条目数、嵌套深度与结构节点；单变量超过 512 KiB、累计超过 2 MiB、超过 2048 条或结构超过 4096 节点时阻断；
- 对可达 Lua Script Effect 做词法扫描并把源码 hash 纳入指纹；排除注释/字符串误报，硬阻断无限循环、共享库/执行上下文写入、`op()` 代码位置注入、核心宏覆写和超量分配；
- 包装 `runLuaScriptEffect`，限制每对象 5 秒调用次数、编译字节和递归深度；执行后恢复被修改的 `string/table/math` 共享库与 RPBox/TRP3 防护 hook；
- 普通有界循环、局部 UI 计算和单次安全 effect 只进入风险提示，不因“存在 Lua”本身隔离；
- 发布者白名单只包含用户显式添加的条目；TRP3 `my` 数据库确认的本人作品按本地所有权处理。可信发布者可跳过 Lua 提示/组合策略，但不能跳过无限循环、崩溃载荷、沙箱逃逸、核心数据篡改和来源黑名单；
- 默认开启，可在 RPBox 设置中手动关闭；
- 隔离后移除“可使用”并显示红叉图标；
- 右键隔离道具显示提示，可直接把当前对象的传输发送者/最后编辑者/创建者加入发布者白名单并重新扫描；硬风险不会因信任作者而解除；
- 隔离提示按风险优先级编号分行；确认移除时统一删除当前槽位、所属根载体及其内部对象，载体按根 ID 定位，不会误删背包容器，也不会触发 `LI.OD` 摧毁工作流；
- RPBox“对象防护”页持续保留风险台账和待扫描状态，可切换当前隔离状态；
- 风险台账拆分显示行为分、放大分、运行时实证分和策略分；步骤环 +15、同对象工作流递归 +20，结构分封顶 40；
- 只有在风险台账中显式“加入忽略”才会跨扫描放行，风险记录仍保留；
- 来源黑名单独立于行为评分：命中系统或用户黑名单时策略分 +120 并隔离；
- 内置黑名单精确包含“蕾火演员死冯-金色平原”“工作人员二号-金色平原”“绿宝石兽-金色平原”；
- 隔离、备份和允许记录保存在 `RPBox_ItemGuardDB`。

状态相互独立：

| 状态 | 含义 |
|---|---|
| `unscanned` | 新接收或已更新，尚未完成安全扫描；默认阻止执行 |
| `trusted` | 当前指纹已扫描通过，可正常执行；不进入风险台账 |
| `finding` | 风险事实与原因，临时放行不会删除 |
| `observed` | 非阻断风险提示；对象已扫描并可执行 |
| `quarantined` | 当前移除“可使用”并显示红叉 |
| `released` | 用户临时放行，可继续使用；下次扫描恢复隔离 |
| `ignored` | 用户在 GUI 中二次确认加入忽略清单，后续扫描不自动隔离 |

从数据库版本 2 起，旧版由弹窗“无视风险”产生的持久忽略记录会自动清除，避免一次点击永久绕过后续扫描。

集成文件：

- [`addon/RPBox_Addon/ItemGuard.lua`](../addon/RPBox_Addon/ItemGuard.lua)
- [`addon/RPBox_Addon/ItemGuardRules.lua`](../addon/RPBox_Addon/ItemGuardRules.lua)
- [`addon/RPBox_Addon/ItemGuardBlacklist.lua`](../addon/RPBox_Addon/ItemGuardBlacklist.lua)
- [`addon/RPBox_Addon/ItemGuardPublisherWhitelist.lua`](../addon/RPBox_Addon/ItemGuardPublisherWhitelist.lua)
- [`addon/RPBox_Addon/ItemGuardSoundRules.lua`](../addon/RPBox_Addon/ItemGuardSoundRules.lua)
- [`addon/RPBox_Addon/ItemGuardLifecycleRules.lua`](../addon/RPBox_Addon/ItemGuardLifecycleRules.lua)
- [`addon/RPBox_Addon/ItemGuardVariableRules.lua`](../addon/RPBox_Addon/ItemGuardVariableRules.lua)
- [`addon/RPBox_Addon/ItemGuardAuraRules.lua`](../addon/RPBox_Addon/ItemGuardAuraRules.lua)
- [`addon/RPBox_Addon/ItemGuardContentRules.lua`](../addon/RPBox_Addon/ItemGuardContentRules.lua)
- [`addon/RPBox_Addon/ItemGuardLuaRules.lua`](../addon/RPBox_Addon/ItemGuardLuaRules.lua)
- [`addon/RPBox_Addon/Core.lua`](../addon/RPBox_Addon/Core.lua)
- [`addon/RPBox_Addon/MainFrame.lua`](../addon/RPBox_Addon/MainFrame.lua)
- [`addon/tests/item_guard_smoke.lua`](../addon/tests/item_guard_smoke.lua)

发布者白名单命令：

```text
/rpbox guard trust 名字-服务器
/rpbox guard untrust 名字-服务器
/rpbox guard trustlist
```

发布者白名单没有任何系统预置条目，全部由用户自行添加，保存在 `RPBox_ItemGuardDB.publisherWhitelist`。隔离弹窗的“信任作者”会优先选择传输发送者，其次选择最后编辑者/创建者。来源黑名单优先级始终高于发布者信任。
TRP3 发布者字段不是密码学签名：在线接收优先使用 `TRP3_Security.sender`，本机作品优先使用 `TRP3_DB.my`；缺少这两项时才回退到最后编辑者/创建者元数据。因此发布者白名单只能降低策略误报，不能替代硬风险检查。

## 独立注入方案

早期独立脚本仍保存在 [`refs/TRP3ItemGuard.lua`](../refs/TRP3ItemGuard.lua)，用于规则研究，不是当前 RPBox 插件的运行入口。以下内容记录该方案。

## 独立方案结论

可以利用一次性 `_G` 引导实现 TRP3 Extended 道具的执行前扫描和会话级隔离，不需要制作或安装独立插件。但它不能像操作系统杀毒软件一样提供绝对保证。

最重要的边界是：如果恶意 Lua 已经进入不终止的同步循环，WoW 主线程会被占满，任何后装的检测器都无法获得执行机会。因此检测器必须在 `/reload` 后、再次使用或信任可疑道具之前安装。

该独立方案遵循以下原则：

- 默认隔离，不自动删除道具、剧本或 SavedVariables；
- 隔离时移除道具的“可使用”标记，并把图标替换成 `ui-engineering-90-remote-close-icon`；
- 在道具根对象中保存原始图标和可使用状态，以便跨 `/reload` 后恢复；
- 执行前扫描整个根对象，而不是等异常发生后处理；
- Lua、多步骤、`_G` 使用和体积本身只进入报告，不再因为作者来源直接隔离；
- 对工作流循环、异常数据规模和超量 `item_add` 做硬阻断；
- 运行中限制物品添加、变量写入和工作流调用速率；
- 通过强恶意规则检查且未被隔离的对象可以继续获得 `_G`；
- 函数拦截和允许列表只在当前 UI 会话有效；已经写入根对象的可见隔离标记会保留，直到显式解除。

## 为什么 TRP3 原信任机制不够

TRP3 会把 Lua `script` 效果标为低安全级别并要求信任，但工作流中的部分效果被视为高安全级别。例如 `item_add` 可以直接执行，不会因为脚本处于 secured 模式就自动失效。

同时，TRP3 的工作流编译器会递归展开步骤关系。如果步骤的 `.n`、分支或工作流调用形成环，可能在编译或运行阶段产生无限递归、持续计时器回调或反复调用。

此外，旧 `_G` 注入命令会给每一次 `runLuaScriptEffect` 都加入完整 `_G`，其中包括外来道具的脚本。一旦用户被诱导信任，外来脚本与检测器拥有同等级的插件权限。

## 安装顺序

如果当前已经发生卡顿、持续塞物品或数据快速增长：

1. 不要继续点击该道具。
2. 正常退出游戏；如主线程已经完全卡死，只能结束客户端。
3. 备份账号对应的 `WTF/Account/.../SavedVariables/`。
4. 重新进入游戏并执行 `/reload`，不要先使用任何可疑道具。
5. 执行下方的一次性 `_G` 引导命令。
6. 紧接着从自己创建的本地 TRP3 道具运行 `TRP3ItemGuard.lua`，中间不要运行其他 Lua 道具。
7. 查看自动扫描报告。

一次性引导命令也保存在 [`refs/TRP3ItemGuardBootstrap.txt`](../refs/TRP3ItemGuardBootstrap.txt)：

```lua
/run RPBG0=TRP3_API.script.runLuaScriptEffect;TRP3_API.script.runLuaScriptEffect=function(c,a,s)TRP3_API.script.runLuaScriptEffect=RPBG0;a=a or{};a._G=_G;return RPBG0(c,a,s)end
```

这段命令只对“紧接着执行的一个 Lua Script Effect”注入 `_G`，并且会在该效果真正开始前恢复 TRP3 原始执行器。检测器取得 `_G` 后，自行安装扫描、隔离和受控注入包装。之后不再按“本机/外来”一刀切；只要对象通过强恶意规则检查且未被隔离，就能继续获得 `_G`。

如果执行引导后误点了其他 Lua 道具，一次性机会已经被该道具消耗。应立即 `/reload`；确认没有异常后，重新执行引导并立刻运行可信检测器。

检测器的函数拦截在 `/reload` 后失效，需要重新安装；已经隔离的道具仍保持禁用和隔离图标。

## 命令

```text
/rpboxguard status
/rpboxguard scan
/rpboxguard list
/rpboxguard inspect ROOT_ID
/rpboxguard release 道具名
/rpboxguard allow ROOT_ID
/rpboxguard block ROOT_ID
/rpboxguard restore
```

含义：

| 命令 | 行为 |
|---|---|
| `status` | 显示安装状态、扫描次数和隔离数量 |
| `scan` | 重新扫描已注册、本地和交换数据库中的根对象 |
| `list` | 列出当前会话的隔离对象 |
| `inspect ID` | 显示对象得分、工作流规模和命中原因 |
| `release 道具名` | 按完整名称解除匹配隔离，恢复图标和可使用状态，并加入当前会话允许列表 |
| `allow ID` | 按根 ID 强制解除隔离、恢复外观并加入当前会话允许列表 |
| `block ID` | 手动隔离该对象 |
| `restore` | 卸载钩子并恢复原函数，防护随即关闭 |

`release` 和 `allow` 都是高风险操作。只应在人工检查完整对象代码后使用；它们不会关闭全局运行时熔断上限。

例如，道具名为“可疑的礼盒”时：

```text
/rpboxguard release 可疑的礼盒
```

名称采用去除颜色代码后的完整匹配。如果多个被隔离道具同名，会一起解除，并在聊天框逐个报告根 ID。

## 可见隔离

对判定需要阻断的根道具，检测器会先保存原始名称、原始图标及“可使用”状态，然后执行：

```lua
root.BA.US = nil
root.BA.IC = "ui-engineering-90-remote-close-icon"
```

`BA.US` 正是 TRP3 判断道具能否右键使用的字段。工作流拦截仍然保留，因此即使其他路径尝试直接执行剧本，也会被第二层防护阻止。

备份记录保存在道具根对象的 `RPBOX_GUARD_QUARANTINE` 字段中。即使 `/reload` 令函数钩子消失，道具仍保持不可使用和隔离图标；重新启动检测器后，可以用 `release 道具名` 恢复。

隔离不会修改道具名称。解除后会恢复原始 `BA.IC` 和 `BA.US`，删除备份字段，并将根 ID 加入当前会话允许列表，避免下一次使用时立即再次隔离。

## 静态检测内容

检测器遍历：

- `TRP3_Tools_DB`；
- `TRP3_Exchange_DB`；
- `TRP3_DB.global`；
- 根对象的 `IN`、`QE`、`ST` 子对象；
- 每个对象的 `SC` 工作流、步骤、分支和效果。

### 硬阻断

- 工作流步骤图存在环；
- 工作流之间存在递归调用环；
- 编译展开规模超过预算；
- 表节点、嵌套深度或字符串总量超过预算；
- 单个 `item_add` 请求数量超过上限；
- 明确的空死循环，如 `while true do end`；
- 无退出条件的无条件循环；
- 循环结合直接创建背包物品；
- 定时器或 `OnUpdate` 结合直接数据库写入或物品创建；
- 动态代码加载结合直接数据库写入或物品创建。

### 风险提示

- Lua 中存在循环；
- 请求 `args._G` 或 `_G`；
- 直接访问 TRP3 SavedVariables 或 `inventory.addItem`；
- 使用 `C_Timer.After`、`OnUpdate` 或递归工作流调用；
- 使用 `loadstring`、`setfenv`、`getfenv`、`string.char` 等动态构造手段；
- `item_add`、变量写入或工作流调用数量异常；
- 对不存在步骤的引用。
- Lua 来自其他作者。

扫描器使用访问表和节点预算，避免恶意循环表反过来卡死扫描器。

## 作者判定

仅仅存在于 `TRP3_Tools_DB` 并不必然代表对象可信，因为外来对象可能被导入本地工具数据库。

检测器同时检查：

- `TRP3_Tools_DB[rootID]`；
- `root.MD.CB` 创建者；
- `TRP3_Security.sender[rootID]` 发送者；
- `TRP3_API.globals.player_id` 当前玩家身份。

如果本地对象仍标记为其他创建者或发送者，它会在报告中标记为外来来源，但“外来”本身不再触发隔离。

该判断仍可能被伪造，因此不是密码学身份验证，只是降低“导入后自动变成可信对象”的风险。

## 大型正常道具校准样本

针对一份实际提供的 `!` 开头 TRP3 导出码，检测器按 LibDeflate 和 AceSerializer 格式离线解码，仅检查结构，没有执行其中 Lua。样本特征为：

| 指标 | 数值 |
|---|---:|
| 解压后的序列化数据 | 约 187 KB |
| 类对象 | 2 |
| 工作流 | 7 |
| 步骤 | 18 |
| 效果 | 15 |
| Lua Script Effect | 2 |
| Lua 总量 | 约 164 KB |
| `item_add` | 1 |
| 步骤环 | 0 |
| 工作流递归环 | 0 |

主体是大型 GnomeMap 界面程序，包含普通 `for`/`while`、`C_Timer.After`、`OnUpdate`、变量保存和大量 `_G` UI API 调用。这些特征说明程序复杂，但不能单独证明恶意。

从版本 4 开始，该类样本只会产生检查报告，不会因为 Lua 数量、作者来源、`_G`、定时器、界面循环或代码体积而自动隔离。只有出现结构环、明确无出口循环、超量添加或“循环/调度＋直接持久化膨胀”组合时才进入可见隔离。样本中的 83 个真实循环位置经过自动预算改写后仍可通过 Lua 5.1 编译。

## 执行拦截

安装后会包装以下入口：

```text
TRP3_API.script.executeClassScript
TRP3_API.script.playEffect
TRP3_API.script.runLuaScriptEffect
TRP3_API.inventory.addItem
```

### `executeClassScript`

每个根对象第一次执行前重新扫描。命中隔离规则时直接返回，不进入 TRP3 编译器。

### `runLuaScriptEffect`

在 Lua 代码交给 `loadstring` 前扫描文本。只有命中强恶意组合或已经处于隔离状态的 Lua 才会被拒绝；正常的大型外来 Lua 通过扫描后可以获得 `_G` 并继续执行。

对于通过静态检查的代码，检测器还会在字符串、长字符串和注释之外识别 `for`、`while`、`repeat`，为循环体注入共享迭代预算。默认整个 Lua 效果最多执行 50,000 次循环迭代；超过后会：

1. 调用运行时熔断并隔离当前根道具；
2. 移除“可使用”并替换隔离图标；
3. 立即 `return` 退出当前 Lua Script Effect。

这一步用于解决同步循环开始后普通 API 包装无法重新获得控制权的问题。循环变量是运行时生成的局部变量，不依赖 TRP3 的 `effect` 调度。

### `playEffect`

按根对象统计短时间内的效果、工作流调用、变量写入和 `item_add` 次数。超过阈值时隔离根对象。

### `inventory.addItem`

作为最后一道熔断，限制单次添加量以及短时间内的直接添加次数和总数量。

## 当前默认阈值

| 项目 | 默认值 |
|---|---:|
| 表节点 | 50,000 |
| 表嵌套深度 | 64 |
| 对象字符串总量 | 2 MiB |
| 单字符串大小 | 512 KiB |
| 工作流数 | 128 |
| 步骤数 | 768 |
| 效果数 | 3,072 |
| 编译展开步骤 | 4,096 |
| 单次添加物品 | 100 |
| 运行窗口 | 5 秒 |
| 5 秒效果数 | 500 |
| 5 秒工作流调用 | 120 |
| 5 秒对象变量写入 | 180 |
| 5 秒 `item_add` 调用 | 25 |
| 5 秒 `item_add` 数量 | 250 |
| 5 秒直接添加调用 | 35 |
| 5 秒直接添加数量 | 350 |
| 单次 Lua 效果循环迭代预算 | 50,000 |

这些值偏向“阻止数据膨胀，同时减少普通剧情道具误报”。如果社区出现新的样本，应根据真实恶意对象调整，而不是无限降低阈值。

## 检测器做不到的事

### 无法抢救安装前已经运行的死循环

检测器安装前已经开始的 Lua 同步死循环仍会占满主线程，无法事后注入看门狗。安装后通过 `runLuaScriptEffect` 的普通 `for`、`while`、`repeat` 会被自动加入循环预算；动态生成后绕过执行入口的代码或纯递归仍可能逃逸。

### 无法证明任意 Lua 安全

字符串扫描会被复杂混淆绕过，也可能误报合法脚本。因此当前策略不会再因“存在 Lua”或“作者是别人”直接隔离，而是把这些信息用于报告，只对结构性死循环、明确无出口循环、直接膨胀行为及其高风险组合执行隔离。

### 不能对抗同权限的可信恶意代码

通过检查或被手动 `allow` 的脚本获得 `_G` 后，理论上可以覆盖检测器、恢复原函数或修改其状态。因此扫描和运行时熔断只能降低风险，不能构成权限隔离。

### 不自动修复膨胀的 SavedVariables

检测器不会删除背包物品、交换缓存或数据库记录。已经膨胀的数据需要先备份，再离线审计和清理。

### 不替代来源验证

最佳防护仍是：不对陌生发送者建立全局信任，不运行来源不明的 Lua，不把完整 `_G` 注入所有 TRP3 脚本。

## 独立方案的后续方向

独立脚本本身仍是会话内 Lua 检测器。当前 RPBox_Addon 已经承担自动启动和跨会话隔离职责；后续可继续增加：

1. 在 TRP3 初始化后自动安装、在任何对象执行前生效；
2. 独立 SavedVariables 保存隔离名单、规则版本和审计记录；
3. 导入阶段扫描，而不是等首次使用；
4. 可视化报告、对象来源、命中代码片段和一键导出样本；
5. 只读隔离区和经过确认的清理流程；
6. 社区恶意样本哈希/规则更新；
7. 对背包和 SavedVariables 增长做跨会话监控；
8. 针对 TRP3 版本变化的兼容测试。

正式版本仍不应承诺“绝对安全”，应表述为执行前防护、异常熔断和恶意样本辅助识别。
