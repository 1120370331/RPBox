# PRD2: 人物卡跨设备备份同步

## 1. 概述

### 1.1 背景
TotalRP3 插件的人物卡数据存储在魔兽世界客户端的 `WTF/Account/{账号}/SavedVariables/` 目录下，主要涉及以下文件：

| 文件名 | 说明 | 包含数据 |
|--------|------|----------|
| `totalRP3.lua` | 人物卡配置 | TRP3_Profiles（所有人物卡） |
| `totalRP3_Data.lua` | 角色绑定数据 | TRP3_Register（角色-Profile映射、伙伴、黑名单） |
| `totalRP3_Extended.lua` | Extended运行时数据 | 背包、任务日志、光环、变量等 |
| `totalRP3_Extended_Tools.lua` | 道具数据库 | TRP3_Tools_DB（用户创建的道具/战役/任务） |
| `totalRP3_Extended_ImpExport.lua` | 导入导出缓存 | TRP3_Extended_ImpExport（临时导入导出数据） |

现有的魔兽插件管理软件（如 CurseForge、WowUp）主要聚焦于 `Interface/AddOns` 目录的管理，无法有效备份 WTF 目录下的角色数据。

### 1.2 目录结构

**参考数据源位置：**
```
C:\Program Files (x86)\World of Warcraft\_retail_\WTF\Account\
```

**目录层级：**
```
WTF/Account/
├── {账号ID1}/                         # 如：563986541#1
│   ├── SavedVariables/                # 账号级别数据（TRP3数据在此）
│   │   ├── totalRP3.lua
│   │   ├── totalRP3_Data.lua
│   │   └── ...
│   └── {服务器名}/                     # 如：金色平原
│       ├── {角色名1}/
│       └── {角色名2}/
├── {账号ID2}/                         # 如：331#1
│   ├── SavedVariables/
│   └── {服务器名}/
├── {账号ID3}/                         # 可能有更多账号
└── SavedVariables/                    # 全局SavedVariables（非TRP3）
```

**重要说明**：
- 一个WoW安装目录可能包含**多个战网账号**
- TRP3数据存储在**账号级别**的SavedVariables目录
- 需要扫描所有账号目录以获取完整数据

### 1.3 目标
提供一套完整的人物卡云端备份、同步、管理方案，让玩家可以：
- 一键备份本地人物卡到云端
- 跨设备同步人物卡数据
- 在云端查看和管理人物卡
- 支持版本历史和回滚

## 2. 用户故事

| 编号 | 角色 | 故事 | 优先级 |
|------|------|------|--------|
| US-01 | RP玩家 | 我想一键备份我的人物卡，以防数据丢失 | P0 |
| US-02 | RP玩家 | 我想在新电脑上快速恢复我的人物卡 | P0 |
| US-03 | RP玩家 | 我想在网页上查看我的人物卡内容 | P1 |
| US-04 | RP玩家 | 我想回滚到之前的人物卡版本 | P2 |

## 3. 功能需求

### 3.1 本地数据扫描 (P0)
- 自动检测魔兽世界安装目录
- 扫描 WTF 目录下所有账号和角色
- 解析 TRP3 Lua 数据文件
- 提取人物卡核心信息（名称、种族、职业、描述、头像等）

### 3.2 云端同步 (P0)
- 上传人物卡数据到云端
- 支持增量同步（仅同步变更部分）
- 冲突检测和解决策略
- 同步状态实时显示

### 3.3 数据恢复 (P0)
- 从云端下载人物卡数据
- 写入本地 WTF 目录
- 支持选择性恢复（单个角色/全部）

### 3.4 版本管理 (P1)
- 保留历史版本（最近 10 个版本）
- 版本对比功能
- 一键回滚到指定版本

### 3.5 云端预览 (P1)
- 网页端查看人物卡详情
- 人物卡卡片式展示
- 支持搜索和筛选

### 3.6 人物卡编辑 (P1)

**编辑能力：**
- 在RPBox内直接编辑人物卡所有字段
- 支持富文本编辑
- 实时预览效果
- 完全兼容TRP3模板结构

**TRP3模板结构：**

| 模板 | 结构 | 说明 |
|------|------|------|
| Template 1 | T1.TX | 单一文本框 |
| Template 2 | T2[].TX/IC/BK | 多个可自定义框架 |
| Template 3 | T3.PH/PS/HI | 外貌+性格+历史 |

**字段编辑分组：**

```
┌─ 基本信息 (characteristics)
│  ├─ 名字(FN)、姓氏(LN)、头衔(TI)
│  ├─ 种族(RA)、职业(CL)
│  ├─ 年龄(AG)、身高(HE)、体重(WE)
│  ├─ 眼睛颜色(EC)、出生地(BP)、居住地(RE)
│  └─ 头像图标(IC)
│
├─ 关于页面 (about)
│  ├─ 模板选择(TE): 1/2/3
│  ├─ [模板1] 自由文本
│  ├─ [模板2] 多框架编辑
│  └─ [模板3] 外貌/性格/历史
│
└─ RPBox扩展
   └─ 自定义图片（立绘、参考图）
```

### 3.7 写回本地 (P1)

- 将编辑后的人物卡写回WoW本地文件
- 需游戏关闭状态下操作
- 自动备份原文件

### 3.8 社区发布 (P2)

- 发布人物卡到RPBox社区
- 支持上传自定义图片（突破游戏限制）
- 设置可见性（公开/仅好友/私密）

### 3.9 图片扩展 (P2)

**突破游戏限制：**
- TRP3仅支持游戏内图标，无法上传自定义图片
- RPBox支持上传人物立绘、参考图等
- 图片仅在RPBox社区可见，不影响游戏内显示

## 4. 非功能性需求

### 4.1 性能要求

| 指标 | 要求 | 说明 |
|------|------|------|
| 应用启动时间 | < 3秒 | 冷启动到可交互 |
| 本地扫描速度 | < 5秒/100个Profile | 首次全量扫描 |
| 增量扫描速度 | < 1秒 | 检测文件变更 |
| 单Profile同步 | < 2秒 | 上传或下载 |
| 批量同步 | < 30秒/100个 | 并发控制 |
| Lua解析速度 | < 100ms/MB | 解析性能 |
| 内存占用 | < 200MB | 正常使用状态 |
| 磁盘占用 | < 100MB | 应用本身 |

### 4.2 安全性要求

| 类别 | 要求 |
|------|------|
| 数据传输 | 全程HTTPS加密 |
| 身份认证 | JWT Token，有效期7天 |
| 密码存储 | bcrypt加密，不可逆 |
| 敏感数据 | 本地不存储明文密码 |
| API安全 | 请求频率限制，防DDoS |
| 数据隔离 | 用户间数据严格隔离 |
| 日志脱敏 | 不记录敏感信息 |

### 4.3 可用性要求

| 类别 | 要求 |
|------|------|
| 界面语言 | 支持中文/英文 |
| 操作反馈 | 所有操作有明确反馈（成功/失败/进度） |
| 错误提示 | 友好的错误信息，提供解决建议 |
| 快捷操作 | 支持键盘快捷键 |
| 无障碍 | 支持屏幕阅读器基本功能 |
| 学习成本 | 首次使用有引导流程 |

### 4.4 可靠性要求

| 类别 | 要求 |
|------|------|
| 数据完整性 | 同步前后数据校验（MD5） |
| 自动备份 | 写回本地前自动备份原文件 |
| 断点续传 | 大文件支持断点续传 |
| 离线可用 | 无网络时可查看本地缓存 |
| 崩溃恢复 | 异常退出后数据不丢失 |
| 服务可用性 | 云端服务 99.9% 可用 |

### 4.5 兼容性要求

| 类别 | 要求 |
|------|------|
| 操作系统 | Windows 10/11 (64位) |
| WoW版本 | 正式服(_retail_)、怀旧服(_classic_) |
| TRP3版本 | 2.x 及以上 |
| Extended版本 | 1.x 及以上 |
| 屏幕分辨率 | 最低 1280x720 |
| 网络环境 | 支持代理配置 |

## 5. 技术方案

### 5.1 TRP3 字段对照表

**characteristics（角色特征）字段：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| v | version | 数据版本号 |
| FN | FirstName | 名字 |
| LN | LastName | 姓氏 |
| TI | Title | 头衔（如"经销商"） |
| FT | FullTitle | 全名/英文名 |
| RA | Race | 种族 |
| CL | Class | 职业（RP职业，非游戏职业） |
| AG | Age | 年龄 |
| EC | EyeColor | 眼睛颜色 |
| EH | EyeColorHex | 眼睛颜色（十六进制） |
| HE | Height | 身高 |
| WE | Weight | 体重 |
| BP | Birthplace | 出生地 |
| RE | Residence | 居住地 |
| RC | ResidenceCoordinates | 居住地地图坐标 [mapId, x, y, zoneName] |
| RS | RelationshipStatus | 感情状态（枚举值） |
| IC | Icon | 头像图标 |
| CH | ClassColorHex | 职业颜色（十六进制） |
| bkg | Background | 背景图ID |
| MI | MiscInfo | 其他信息（数组，见下方详细结构） |
| PS | PersonalityStats | 性格特征（数组，见下方详细结构） |

**RelationshipStatus（感情状态）枚举：**

| 值 | 常量 | 说明 |
|----|------|------|
| 0 | UNKNOWN | 不显示 |
| 1 | SINGLE | 单身 |
| 2 | TAKEN | 恋爱中 |
| 3 | MARRIED | 已婚 |
| 4 | DIVORCED | 离异 |
| 5 | WIDOWED | 丧偶 |

**about（关于）字段：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| v | version | 数据版本号 |
| TE | Template | 模板类型（1=模板1, 2=模板2, 3=模板3） |
| BK | Background | 背景图ID |
| MU | Music | 角色主题音乐（游戏音乐文件ID） |
| T1 | Template1 | 模板1内容 { TX } |
| T2 | Template2 | 模板2内容（数组）[{ TX, IC, BK }, ...] |
| T3 | Template3 | 模板3内容 { PH, PS, HI } |

**T3（模板3）子结构：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| PH | Physical | 外貌描述 { TX, IC, BK } |
| PS | Personality | 性格描述 { TX, IC, BK } |
| HI | History | 历史背景 { TX, IC, BK } |

**character（角色状态）字段：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| v | version | 数据版本号 |
| RP | RoleplayStatus | RP状态（1=IC, 2=OOC） |
| WU | WalkUp | 是否接受搭话（1=否, 2=是） |
| CU | Currently | 当前状态文本（IC状态描述） |
| CO | CurrentlyOOC | OOC状态文本 |

**MI（其他信息）数组元素结构：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| ID | PresetType | 预设类型ID（可选，见下表） |
| NA | Name | 字段名称 |
| VA | Value | 字段值 |
| IC | Icon | 图标 |

**MiscInfoType（其他信息预设类型）：**

| ID | 英文名 | 中文名 | 显示在Tooltip |
|----|--------|--------|---------------|
| 1 | Custom | 自定义 | 否 |
| 2 | House | 家族/家名 | 否 |
| 3 | Nickname | 昵称 | 否 |
| 4 | Motto | 座右铭 | 否 |
| 5 | FacialFeatures | 面部特征 | 否 |
| 6 | Piercings | 穿孔 | 否 |
| 7 | Pronouns | 代词 | 是 |
| 8 | GuildName | RP公会名 | 是 |
| 9 | GuildRank | RP公会头衔 | 是 |
| 10 | Tattoos | 纹身 | 否 |
| 11 | VoiceReference | 声音参考 | 是 |

**PS（性格特征）数组元素结构：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| ID | PresetID | 预设ID（使用预设时） |
| LT | LeftTrait | 左侧特征名（自定义时） |
| RT | RightTrait | 右侧特征名（自定义时） |
| LI | LeftIcon | 左侧图标 |
| RI | RightIcon | 右侧图标 |
| LC | LeftColor | 左侧颜色 { r, g, b } |
| RC | RightColor | 右侧颜色 { r, g, b } |
| V2 | Value | 数值（0-20，10为中间值） |

**性格特征预设（Psycho Presets）：**

| ID | 左侧(LT) | 右侧(RT) | 中文 |
|----|----------|----------|------|
| 1 | Chaotic | Lawful | 混乱 - 守序 |
| 2 | Chaste | Lustful | 贞洁 - 好色 |
| 3 | Forgiving | Vindictive | 宽容 - 记仇 |
| 4 | Altruistic | Selfish | 利他 - 自私 |
| 5 | Truthful | Deceitful | 诚实 - 欺骗 |
| 6 | Gentle | Brutal | 温和 - 残暴 |
| 7 | Superstitious | Rational | 迷信 - 理性 |
| 8 | Renegade | Paragon | 叛逆 - 典范 |
| 9 | Cautious | Impulsive | 谨慎 - 冲动 |
| 10 | Ascetic | BonVivant | 禁欲 - 享乐 |
| 11 | Valorous | Spineless | 勇敢 - 懦弱 |

### 5.1.2 TRP3 Extended 字段对照表

**Extended对象类型（TY）：**

| 值 | 常量 | 说明 |
|----|------|------|
| IT | ITEM | 道具 |
| CA | CAMPAIGN | 战役 |
| QU | QUEST | 任务 |
| DI | DIALOG | 对话 |
| CU | CUTSCENE | 过场动画 |
| DO | DOCUMENT | 文档 |
| AU | AURA | 光环 |

**道具基础属性（BA）：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| NA | Name | 道具名称 |
| DE | Description | 描述 |
| IC | Icon | 图标 |
| QA | Quality | 品质（0-5，对应灰/白/绿/蓝/紫/橙） |
| LE | LeftText | 左侧文本（如"单手剑"） |
| RI | RightText | 右侧文本（如"12mm"） |
| VA | Value | 价值（铜币） |
| WE | Weight | 重量（克） |
| SB | Soulbound | 是否灵魂绑定 |
| UN | Unique | 唯一数量限制 |
| ST | Stack | 堆叠数量上限 |
| US | Usable | 是否可使用 |
| WA | Wearable | 是否可穿戴 |
| CT | Container | 是否容器 |
| CO | Component | 是否组件 |
| CR | Craftable | 是否可制作 |
| QE | Quest | 是否任务物品 |
| PA | PreventAdd | 是否禁止添加到背包 |
| PS | PickSound | 拾取音效ID |
| DS | DropSound | 丢弃音效ID |

**容器属性（CO）：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| SI | Size | 尺寸（如"5x4"） |
| SR | Rows | 行数 |
| SC | Columns | 列数 |
| DU | Durability | 耐久度 |
| MW | MaxWeight | 最大承重 |
| OI | OnlyInner | 仅限内部物品 |

**元数据（MD）：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| MO | Mode | 模式（QUICK/NORMAL/EXPERT） |
| V | Version | 版本号 |
| CD | CreateDate | 创建日期 |
| CB | CreateBy | 创建者 |
| SD | SaveDate | 保存日期 |
| SB | SaveBy | 保存者 |
| tV | ToolVersion | 工具版本 |

**背包槽位结构：**

| 缩写 | 全称 | 说明 |
|------|------|------|
| id | ClassID | 道具类ID |
| count | Count | 数量 |
| madeBy | MadeBy | 制作者 |
| vars | Variables | 变量数据 |
| content | Content | 容器内容（如果是容器） |
| cooldown | Cooldown | 冷却时间 |

### 5.2 数据结构

**TRP3原始数据结构（Lua）：**
```lua
-- totalRP3.lua
TRP3_Profiles = {
  ["profileId"] = {
    ["profileName"] = "配置名称",
    ["player"] = {
      ["characteristics"] = {
        ["v"] = 1,                    -- 版本号
        ["FN"] = "名字",
        ["LN"] = "姓氏",
        ["TI"] = "头衔",
        ["FT"] = "全名",
        ["RA"] = "种族",
        ["CL"] = "职业",
        ["AG"] = "年龄",
        ["EC"] = "眼睛颜色",
        ["EH"] = "ff0000",            -- 眼睛颜色Hex
        ["HE"] = "身高",
        ["WE"] = "体重",
        ["BP"] = "出生地",
        ["RE"] = "居住地",
        ["RC"] = { mapId, x, y, "区域名" },  -- 居住地坐标
        ["RS"] = 1,                   -- 感情状态枚举
        ["IC"] = "图标名",
        ["CH"] = "ffffff",            -- 职业颜色Hex
        ["bkg"] = 1,                  -- 背景图ID
        ["MI"] = {                    -- 其他信息数组
          { ["ID"] = 3, ["NA"] = "昵称", ["VA"] = "小芙", ["IC"] = "图标" },
        },
        ["PS"] = {                    -- 性格特征数组
          { ["ID"] = 1, ["V2"] = 15 },  -- 预设：混乱-守序，偏守序
          { ["LT"] = "自定义左", ["RT"] = "自定义右", ["LI"] = "图标", ["RI"] = "图标", ["V2"] = 10 },
        },
      },
      ["about"] = {
        ["v"] = 1,
        ["TE"] = 3,                   -- 使用模板3
        ["BK"] = 1,                   -- 背景图ID
        ["MU"] = 123456,              -- 角色主题音乐ID
        ["T1"] = { ["TX"] = "模板1文本" },
        ["T2"] = {                    -- 模板2：多框架
          { ["TX"] = "文本", ["IC"] = "图标", ["BK"] = 1 },
        },
        ["T3"] = {                    -- 模板3：外貌/性格/历史
          ["PH"] = { ["TX"] = "外貌描述", ["IC"] = "图标", ["BK"] = 1 },
          ["PS"] = { ["TX"] = "性格描述", ["IC"] = "图标", ["BK"] = 1 },
          ["HI"] = { ["TX"] = "历史背景", ["IC"] = "图标", ["BK"] = 1 },
        },
      },
      ["character"] = {
        ["v"] = 1,
        ["RP"] = 1,                   -- 1=IC, 2=OOC
        ["WU"] = 2,                   -- 1=不接受搭话, 2=接受搭话
        ["CU"] = "当前状态",
        ["CO"] = "OOC状态",
      },
    },
  }
}

-- totalRP3_Data.lua
TRP3_Register = {
  ["character"] = {
    ["角色名-服务器"] = { profileID = "xxx", class = "WARRIOR", race = "Human" }
  },
  ["profiles"] = {
    ["xxx"] = { ... }                 -- 缓存的其他玩家人物卡
  },
  ["companion"] = { ... },
  ["blockList"] = { ... }
}

-- totalRP3_Extended.lua (角色级别运行时数据)
TRP3_Extended = {
  ["角色名-服务器"] = {
    ["inventory"] = {                 -- 背包数据
      ["id"] = "main",
      ["content"] = {
        ["1"] = { id = "itemClassID", count = 1 },
        ["17"] = { id = "bagClassID", content = { ... } },  -- 快捷背包槽
      },
      ["totalWeight"] = 12500,
    },
    ["questlog"] = {                  -- 任务日志
      ["currentCampaign"] = "campaignID",
      ["campaigns"] = {
        ["campaignID"] = {
          ["currentQuest"] = "questID",
          ["quests"] = { ... },
          ["variables"] = { ... },
        }
      }
    },
    ["auras"] = {                     -- 光环
      ["auraInstanceID"] = {
        ["id"] = "auraClassID",
        ["expiry"] = 1704931200,      -- 过期时间戳
        ["vars"] = { ... },
      }
    },
  }
}

-- totalRP3_Extended_Tools.lua (道具数据库)
TRP3_Tools_DB = {
  ["objectFullID"] = {                -- 如 "playerID_itemID"
    ["TY"] = "IT",                    -- 类型：IT/CA/QU/DI/CU/DO/AU
    ["MD"] = {                        -- 元数据
      ["MO"] = "NORMAL",              -- 模式
      ["V"] = 1,                      -- 版本
      ["CD"] = "28/04/16 17:36:38",   -- 创建日期
      ["CB"] = "玩家名-服务器",        -- 创建者
    },
    ["BA"] = {                        -- 基础属性
      ["NA"] = "道具名称",
      ["DE"] = "道具描述",
      ["IC"] = "INV_Misc_Bag_01",
      ["QA"] = 1,                     -- 品质
      ["WE"] = 500,                   -- 重量
      ["VA"] = 100,                   -- 价值
      ["CT"] = true,                  -- 是否容器
    },
    ["CO"] = {                        -- 容器属性（如果是容器）
      ["SI"] = "5x4",
      ["SR"] = "5",
      ["SC"] = "4",
    },
    ["SC"] = { ... },                 -- 脚本/工作流
    ["IN"] = { ... },                 -- 内部对象
    ["NT"] = "备注",
  }
}
```

**RPBox数据库模型（TypeScript）：**
```typescript
// WoW账号（本地）
interface WowAccount {
  accountId: string;             // 如：563986541#1
  wowPath: string;               // WoW安装路径
  profileCount: number;          // 人物卡数量
  lastScanned: Date;
}

// 性格特征
interface PersonalityTrait {
  presetId?: number;             // 预设ID（1-11），自定义时为空
  leftTrait?: string;            // LT - 左侧特征名（自定义时）
  rightTrait?: string;           // RT - 右侧特征名（自定义时）
  leftIcon?: string;             // LI - 左侧图标
  rightIcon?: string;            // RI - 右侧图标
  leftColor?: { r: number; g: number; b: number };   // LC
  rightColor?: { r: number; g: number; b: number };  // RC
  value: number;                 // V2 - 数值（0-20）
}

// 其他信息
interface MiscInfo {
  presetType?: number;           // ID - 预设类型（1-11）
  name: string;                  // NA - 字段名称
  value: string;                 // VA - 字段值
  icon: string;                  // IC - 图标
}

// 人物卡配置
interface Profile {
  id: string;                    // TRP3 profileId
  userId: number;                // RPBox用户ID
  wowAccountId: string;          // WoW账号ID（如：563986541#1）
  profileName: string;           // 配置名称

  characteristics: {
    version: number;             // v
    firstName?: string;          // FN
    lastName?: string;           // LN
    title?: string;              // TI
    fullTitle?: string;          // FT
    race?: string;               // RA
    class?: string;              // CL
    classColor?: string;         // CH - 职业颜色Hex
    age?: string;                // AG
    eyeColor?: string;           // EC
    eyeColorHex?: string;        // EH - 眼睛颜色Hex
    height?: string;             // HE
    weight?: string;             // WE
    birthplace?: string;         // BP
    residence?: string;          // RE
    residenceCoords?: {          // RC - 居住地坐标
      mapId: number;
      x: number;
      y: number;
      zoneName: string;
    };
    relationshipStatus?: number; // RS - 感情状态（0-5）
    icon?: string;               // IC
    background?: number;         // bkg - 背景图ID
    miscInfo: MiscInfo[];        // MI - 其他信息数组
    personalityTraits: PersonalityTrait[];  // PS - 性格特征数组
  };

  about: {
    version: number;             // v
    template: number;            // TE - 模板类型（1/2/3）
    background?: number;         // BK - 背景图ID
    music?: number;              // MU - 角色主题音乐ID
    template1?: { text: string };
    template2?: Array<{ text: string; icon: string; background?: number }>;
    template3?: {
      physical?: { text: string; icon?: string; background?: number };
      personality?: { text: string; icon?: string; background?: number };
      history?: { text: string; icon?: string; background?: number };
    };
  };

  character: {
    version: number;             // v
    rpStatus: number;            // RP - 1=IC, 2=OOC
    walkUp: number;              // WU - 1=否, 2=是
    currently?: string;          // CU - 当前状态
    currentlyOOC?: string;       // CO - OOC状态
  };

  rawLua: string;                // 原始Lua数据（用于完整恢复）
  checksum: string;              // 数据校验和
  version: number;               // 版本号
  syncedAt: Date;
  createdAt: Date;
  updatedAt: Date;
}

// ============ Extended 数据模型 ============

// Extended道具类型
type ExtendedObjectType = 'IT' | 'CA' | 'QU' | 'DI' | 'CU' | 'DO' | 'AU';

// 道具元数据
interface ItemMetadata {
  mode: 'QUICK' | 'NORMAL' | 'EXPERT';  // MO
  version: number;               // V
  createDate: string;            // CD
  createBy: string;              // CB
  saveDate: string;              // SD
  saveBy: string;                // SB
  toolVersion?: number;          // tV
}

// 道具基础属性
interface ItemBaseAttributes {
  name: string;                  // NA
  description?: string;          // DE
  icon: string;                  // IC
  quality?: number;              // QA (0-5)
  leftText?: string;             // LE
  rightText?: string;            // RI
  value?: number;                // VA (铜币)
  weight?: number;               // WE (克)
  soulbound?: boolean;           // SB
  unique?: number;               // UN
  stack?: number;                // ST
  usable?: boolean;              // US
  wearable?: boolean;            // WA
  container?: boolean;           // CT
  component?: boolean;           // CO
  craftable?: boolean;           // CR
  quest?: boolean;               // QE
  preventAdd?: boolean;          // PA
  pickSound?: number;            // PS
  dropSound?: number;            // DS
}

// 容器属性
interface ContainerAttributes {
  size: string;                  // SI (如"5x4")
  rows: number;                  // SR
  columns: number;               // SC
  durability?: number;           // DU
  maxWeight?: number;            // MW
  onlyInner?: boolean;           // OI
}

// Extended道具
interface ExtendedItem {
  id: string;                    // 道具完整ID
  userId: number;
  wowAccountId: string;
  type: ExtendedObjectType;      // TY
  metadata: ItemMetadata;        // MD
  baseAttributes: ItemBaseAttributes;  // BA
  containerAttributes?: ContainerAttributes;  // CO
  scripts?: Record<string, any>; // SC
  innerObjects?: Record<string, any>;  // IN
  notes?: string;                // NT
  rawLua: string;
  checksum: string;
  syncedAt: Date;
}

// 背包槽位
interface InventorySlot {
  classId: string;               // id
  count: number;                 // count
  madeBy?: string;               // madeBy
  variables?: Record<string, any>;  // vars
  content?: Record<string, InventorySlot>;  // content (容器内容)
  cooldown?: number;             // cooldown
}

// 角色背包数据
interface CharacterInventory {
  characterId: string;           // 角色名-服务器
  userId: number;
  mainInventory: {
    id: string;
    content: Record<string, InventorySlot>;
    totalWeight: number;
  };
  rawLua: string;
  syncedAt: Date;
}

// 任务日志
interface CharacterQuestLog {
  characterId: string;
  userId: number;
  currentCampaign?: string;
  campaigns: Record<string, {
    currentQuest?: string;
    quests: Record<string, any>;
    variables: Record<string, any>;
  }>;
  rawLua: string;
  syncedAt: Date;
}

// 光环数据
interface CharacterAuras {
  characterId: string;
  userId: number;
  auras: Record<string, {
    classId: string;
    expiry: number;
    variables?: Record<string, any>;
  }>;
  rawLua: string;
  syncedAt: Date;
}

// 角色绑定
interface CharacterBinding {
  id: number;
  userId: number;
  characterName: string;         // 角色名
  realmName: string;             // 服务器名
  profileId: string;             // 绑定的Profile ID
  gameClass: string;             // 游戏职业
  gameRace: string;              // 游戏种族
  faction: string;               // 阵营
}

// RPBox扩展图片（突破游戏限制）
interface ProfileImage {
  id: string;
  profileId: string;
  type: 'portrait' | 'fullbody' | 'reference';  // 头像/全身/参考图
  url: string;                   // 图片URL
  caption?: string;              // 图片说明
  order: number;                 // 排序
  createdAt: Date;
}

// 社区发布
interface PublishedProfile {
  id: string;
  profileId: string;
  userId: number;
  visibility: 'public' | 'friends' | 'private';
  images: ProfileImage[];        // 扩展图片
  likes: number;
  views: number;
  publishedAt: Date;
}
```

### 5.3 Lua 解析器

**技术选型：**
- 客户端（Tauri/Rust）：使用 `mlua` 或 `rlua` crate 解析Lua
- 备选方案：使用纯文本解析器（正则 + 状态机）

**解析流程：**
```
1. 读取 .lua 文件
2. 提取全局变量赋值（TRP3_Profiles = {...}）
3. 解析 Lua table 为 JSON
4. 映射字段缩写为完整名称
5. 验证数据完整性
```

**注意事项：**
- SavedVariables 文件使用 UTF-8 编码
- 需处理中文角色名和服务器名
- 保留原始 rawLua 用于完整恢复

### 5.4 文件监控
使用文件系统监控（Tauri fs watch）检测 WTF 目录变更，提示用户同步。

### 5.5 WoW路径自动检测

**检测优先级：**
1. 读取用户上次保存的路径
2. 检测常见安装位置
3. 读取注册表/启动器配置
4. 手动选择

**Windows常见路径：**
```
C:\Program Files (x86)\World of Warcraft\
C:\Program Files\World of Warcraft\
D:\World of Warcraft\
D:\Games\World of Warcraft\
E:\World of Warcraft\
```

**检测逻辑：**
```
遍历候选路径
    │
    ▼
检查 _retail_\WTF\Account 是否存在
    │
    ├─ 存在 → 加入有效路径列表
    │
    └─ 不存在 → 跳过

返回所有有效路径供用户选择
```

**注册表检测（备选）：**
```
HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft
```

### 5.6 同步触发时机

| 触发方式 | 说明 | 优先级 |
|----------|------|--------|
| 手动触发 | 用户点击"同步"按钮 | P0 |
| 启动检查 | 应用启动时自动扫描本地变更 | P0 |
| 文件监控 | 检测到 totalRP3.lua 变更时提示 | P1 |
| 定时检查 | 可配置的定时扫描（默认关闭） | P2 |

**文件监控细节：**
- 监控路径：`WTF/Account/*/SavedVariables/totalRP3*.lua`
- 触发条件：文件修改时间变化
- 防抖处理：文件变更后等待 5 秒再触发（避免游戏写入过程中触发）
- 游戏运行检测：检测到 WoW 进程运行时，暂停监控（游戏退出后再扫描）

### 5.7 版本标识机制

**核心标识字段：**

| 字段 | 来源 | 说明 |
|------|------|------|
| localModifiedAt | 本地文件mtime | 本地Lua文件的最后修改时间 |
| cloudModifiedAt | 云端数据库 | 云端记录的最后修改时间 |
| lastSyncedAt | 本地缓存 | 上次成功同步的时间 |
| checksum | MD5计算 | 文件内容校验和 |
| version | 递增整数 | 云端版本号（每次保存+1） |

**本地修改时间获取：**
```
本地文件: WTF/.../totalRP3.lua
    │
    ▼
读取文件系统 mtime (最后修改时间)
    │
    ▼
localModifiedAt = file.stat().mtime
```

**云端版本记录：**
```typescript
interface ProfileVersion {
  version: number;           // 递增版本号 v1, v2, v3...
  cloudModifiedAt: Date;     // 云端修改时间
  checksum: string;          // 内容校验和
  changeLog?: string;        // 变更摘要（自动生成）
}
```

**"最新"判断逻辑：**
```
┌─────────────────────────────────────────────────────────┐
│  判断哪个版本更新                                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 首先比较 checksum                                   │
│     │                                                   │
│     ├─ 相同 → 无变更，跳过同步                          │
│     │                                                   │
│     └─ 不同 → 继续判断                                  │
│                                                         │
│  2. 比较修改时间                                        │
│     │                                                   │
│     ├─ localModifiedAt > lastSyncedAt                  │
│     │   → 本地有新修改                                  │
│     │                                                   │
│     ├─ cloudModifiedAt > lastSyncedAt                  │
│     │   → 云端有新修改                                  │
│     │                                                   │
│     └─ 两者都有新修改 → 冲突                            │
│                                                         │
│  3. 冲突时比较时间戳决定"较新"                          │
│     │                                                   │
│     └─ localModifiedAt vs cloudModifiedAt              │
│        较大者为"较新版本"                               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**本地缓存的同步元数据：**
```typescript
// 存储在本地 SQLite 或 JSON 文件中
interface SyncMetadata {
  profileId: string;
  lastSyncedAt: Date;        // 上次同步时间
  lastSyncedChecksum: string; // 上次同步时的checksum
  cloudVersion: number;       // 上次同步时的云端版本号
}
```

**时间戳显示格式：**
- 界面显示：`2024-01-11 20:00` (本地时区)
- 存储格式：ISO 8601 UTC (`2024-01-11T12:00:00Z`)
- 精度：秒级

### 5.8 同步策略

**变更检测：**
```
本地文件 checksum (MD5) ←→ 云端存储 checksum
    │                           │
    └─────── 不一致 ─────────────┘
                │
                ▼
          [需要同步]
```

**同步方向判断：**
| 本地状态 | 云端状态 | 动作 |
|----------|----------|------|
| 有变更 | 无变更 | 上传本地 → 云端 |
| 无变更 | 有变更 | 下载云端 → 本地（需确认） |
| 有变更 | 有变更 | 冲突，需用户选择 |
| 无变更 | 无变更 | 跳过 |

### 5.9 冲突处理

**冲突场景：** 本地和云端都有修改（如在两台电脑上分别编辑）

**解决策略：**
1. **时间优先**（默认）：保留最后修改的版本
2. **本地优先**：始终以本地为准
3. **云端优先**：始终以云端为准
4. **手动选择**：弹窗让用户对比选择

**冲突提示界面：**
```
┌─────────────────────────────────────────┐
│  ⚠️ 检测到冲突：芙拉莉雅                  │
├─────────────────────────────────────────┤
│  本地版本: 2024-01-11 20:00             │
│  云端版本: 2024-01-11 18:30             │
├─────────────────────────────────────────┤
│  [使用本地]  [使用云端]  [查看差异]      │
└─────────────────────────────────────────┘
```

### 5.10 API 接口设计

**人物卡相关接口：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/profiles | 获取用户所有人物卡列表 |
| GET | /api/v1/profiles/:id | 获取单个人物卡详情 |
| POST | /api/v1/profiles | 上传/创建人物卡 |
| PUT | /api/v1/profiles/:id | 更新人物卡 |
| DELETE | /api/v1/profiles/:id | 删除人物卡 |
| GET | /api/v1/profiles/:id/versions | 获取版本历史 |
| POST | /api/v1/profiles/:id/rollback | 回滚到指定版本 |

### 5.11 同步流程

#### 5.11.1 写回本地机制

**核心原则：** 保留原始Lua格式，确保游戏能正确读取

**写入时机：**
```
检测WoW进程是否运行
    │
    ├─ 运行中 → 提示用户关闭游戏后再操作
    │
    └─ 未运行 → 允许写入
```

**写入流程：**
```
1. 备份原文件 → totalRP3.lua.rpbox_backup
2. 读取原文件完整内容
3. 定位目标Profile在Lua中的位置
4. 替换/插入Profile数据
5. 写入新文件
6. 验证文件语法正确性
7. 删除备份（或保留供回滚）
```

**Lua格式保持：**
```lua
-- 必须保持TRP3原始格式
TRP3_Profiles = {
    ["profileId"] = {
        ["profileName"] = "芙拉莉雅",
        ["player"] = {
            -- 保持原始缩写字段名（FN, LN, RA等）
            ["characteristics"] = {
                ["FN"] = "芙拉莉雅",
                ["RA"] = "人类",
                ...
            },
        },
    },
}
```

#### 5.11.2 云端迁移生效

**迁移场景：** 新电脑/重装系统后恢复人物卡

**账号ID说明：**
- 文件夹名格式：`<Battle.net账号ID>#<WoW许可证索引>`
- 例如 `563986541#1` 中，`563986541` 是Battle.net账号ID
- **此ID绑定账号而非设备，换电脑不会改变**
- 新电脑首次登录WoW会自动创建同名文件夹

**账号友好标识：**

从目录结构可提取用户友好信息：
```
WTF/Account/{账号ID}/
├── {服务器名}/           ← 可读取服务器名
│   ├── {角色名1}/       ← 可读取角色名列表
│   ├── {角色名2}/
│   └── ...
```

**显示效果：**
```
┌─────────────────────────────────────────┐
│  选择要恢复的账号：                      │
├─────────────────────────────────────────┤
│  ○ 563986541#1                          │
│    └─ 金色平原 (25个角色)                │
│       芙拉莉雅, 卡洛斯丶海顿, 塔莉恩...  │
│                                         │
│  ○ 331#1                                │
│    └─ 暴风城 (3个角色)                   │
│       角色A, 角色B...                    │
└─────────────────────────────────────────┘
```

**恢复前置条件检测：**

| 场景 | WTF/Account目录 | SavedVariables目录 | totalRP3.lua | 处理方式 |
|------|----------------|-------------------|--------------|----------|
| A | 不存在 | - | - | 提示：请先登录一次WoW |
| B | 存在 | 不存在 | - | 提示：请先进入一次游戏角色 |
| C | 存在 | 存在 | 不存在 | 提示：请先安装TRP3并进入游戏一次 |
| D | 存在 | 存在 | 存在(空/默认) | 可直接写入恢复数据 |
| E | 存在 | 存在 | 存在(有数据) | 提示是否覆盖/合并 |

**迁移流程：**
```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ 登录RPBox│───▶│ 拉取云端│───▶│ 选择要  │───▶│ 写入本地│
│ 账号    │    │ 人物卡  │    │ 恢复的卡│    │ WTF目录 │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
```

**关键保障：**
1. **rawLua字段**：云端存储原始Lua文本，恢复时直接写回
2. **角色绑定**：同时恢复TRP3_Register中的角色-Profile映射
3. **目录创建**：如果目标账号目录不存在，自动创建

**上传流程：**
```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ 扫描本地 │───▶│ 解析Lua │───▶│ 计算校验│───▶│ 对比云端│
│ WTF目录 │    │ 数据文件│    │ 和(MD5) │    │ 版本   │
└─────────┘    └─────────┘    └─────────┘    └────┬────┘
                                                  │
                              ┌───────────────────┴───────────────────┐
                              ▼                                       ▼
                        [有变更]                                [无变更]
                              │                                       │
                              ▼                                       ▼
                        ┌─────────┐                             ┌─────────┐
                        │ 上传到  │                             │ 跳过    │
                        │ 云端    │                             │         │
                        └─────────┘                             └─────────┘
```

**下载/恢复流程：**
```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ 选择要  │───▶│ 下载云端│───▶│ 备份本地│───▶│ 写入Lua│
│ 恢复的卡│    │ 数据    │    │ 原文件  │    │ 文件   │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
```

### 5.12 错误处理

**常见错误类型：**

| 错误类型 | 原因 | 处理方式 |
|----------|------|----------|
| 文件读取失败 | 权限不足/文件被占用 | 提示关闭游戏后重试 |
| Lua解析失败 | 文件损坏/格式异常 | 跳过并记录日志 |
| 网络超时 | 网络不稳定 | 自动重试3次 |
| 云端冲突 | 并发修改 | 进入冲突处理流程 |
| 磁盘空间不足 | 本地空间不够 | 提示清理空间 |

### 5.13 重试机制

```
请求失败
    │
    ▼
[重试次数 < 3?]──否──▶ 标记失败，提示用户
    │
   是
    │
    ▼
等待 (2^n) 秒  ← 指数退避
    │
    ▼
  重新请求
```

### 5.14 性能考虑

**大量人物卡处理：**
- 分页加载：每页显示 20 个人物卡
- 虚拟滚动：超过 50 个时启用
- 懒加载：详情数据按需加载

**同步优化：**
- 增量同步：仅同步变更的人物卡
- 并发控制：最多同时上传 3 个
- 压缩传输：gzip 压缩请求体

### 5.15 日志记录

**同步日志：**
- 记录每次同步的时间、结果、变更数量
- 本地存储最近 100 条记录
- 支持导出日志文件

**日志格式：**
```
[2024-01-11 20:00:15] [INFO] 开始同步，检测到 3 个变更
[2024-01-11 20:00:16] [INFO] 上传: 芙拉莉雅 (v3)
[2024-01-11 20:00:17] [INFO] 上传: 塔莉恩 (v2)
[2024-01-11 20:00:18] [ERROR] 上传失败: 克拉丽丝 - 网络超时
[2024-01-11 20:00:20] [INFO] 同步完成，成功 2，失败 1
```

### 5.16 数据安全

**传输安全：**
- 所有API请求使用 HTTPS
- JWT Token 认证，有效期 7 天
- 敏感操作需二次确认

**存储安全：**
- 云端数据按用户隔离
- 数据库字段级加密（可选）
- 定期备份（每日）

### 5.17 批量操作

**支持的批量操作：**
| 操作 | 说明 |
|------|------|
| 批量同步 | 选中多个人物卡一键同步 |
| 批量删除 | 删除云端多个人物卡 |
| 全选/反选 | 快速选择操作 |

**主界面批量模式：**
```
┌─────────────────────────────────────────────┐
│  人物卡管理          [批量模式] [同步全部]   │
├─────────────────────────────────────────────┤
│  ☑ 全选                                     │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │☑ 芙拉莉雅│ │☑ 塔莉恩  │ │☐ 克拉丽丝│       │
│  └─────────┘ └─────────┘ └─────────┘       │
├─────────────────────────────────────────────┤
│  已选择 2 个    [同步选中] [删除选中]        │
└─────────────────────────────────────────────┘
```

### 5.18 AI辅助功能 (P2)

**功能列表：**
| 功能 | 说明 | 优先级 |
|------|------|--------|
| 描述润色 | AI优化人物卡描述文本 | P2 |
| 内容翻译 | 中英文互译 | P2 |
| 智能补全 | 根据已有信息补全空白字段 | P3 |

**调用方式：**
- 在人物卡详情页提供"AI助手"按钮
- 用户选择需要处理的字段
- 调用后端AI服务处理

### 5.19 与剧情记录模块关联

**关联需求：** PRD3剧情记录需要能关联查看参与者的人物卡

#### 5.19.1 版本快照问题

**问题：** 人物卡有多个历史版本，剧情应关联哪个版本？

**方案：** 剧情记录时保存人物卡版本快照

```typescript
interface StoryParticipant {
  profileId: string;
  profileVersion: number;      // 剧情发生时的版本号
  profileSnapshot?: string;    // 可选：当时的人物卡快照(JSON)
  joinedAt: Date;              // 加入剧情的时间
}
```

**查看逻辑：**
- 默认显示当前最新版本
- 提供"查看剧情时版本"选项
- 版本对比功能

#### 5.19.2 剧情可索引标识

**问题：** 如何辨别人物卡是否可被剧情索引？

**方案：** Profile增加可见性设置

```typescript
interface Profile {
  // ... 其他字段
  visibility: {
    isIndexable: boolean;      // 是否可被剧情索引
    isPublic: boolean;         // 是否公开（社区可见）
  };
}
```

**用户控制：**
- 默认：可索引、不公开
- 用户可在人物卡设置中调整

#### 5.19.3 非本应用用户的人物卡

**问题：** 剧情中有其他玩家，他们没有使用RPBox，如何处理？

**数据来源：** TRP3会缓存遇到的其他玩家人物卡

```lua
-- totalRP3_Data.lua 中的 TRP3_Register
TRP3_Register = {
  ["character"] = {
    ["其他玩家-服务器"] = {
      profileID = "xxx",
      -- 基础信息（职业、种族等）
    }
  },
  ["profiles"] = {
    ["xxx"] = {
      -- 缓存的人物卡数据（可能不完整）
    }
  }
}
```

**处理方案：**

| 类型 | 数据来源 | 完整度 | 处理方式 |
|------|----------|--------|----------|
| 本用户人物卡 | TRP3_Profiles | 完整 | 正常关联 |
| 缓存他人人物卡 | TRP3_Register | 部分 | 标记为"缓存数据" |
| 无数据 | - | 无 | 仅显示游戏角色名 |

**界面显示：**
```
剧情参与者：
├─ 芙拉莉雅 [查看人物卡]      ← 本用户，完整
├─ 艾琳 [查看缓存]            ← 他人，有缓存
└─ 路人甲                     ← 无数据
```

**关联标识符：**

| 标识符 | 来源 | 说明 |
|--------|------|------|
| 游戏角色名 | 聊天记录原始数据 | 如"芙拉莉雅" |
| RP名字 | TRP3 Profile.FN | 如"芙拉莉雅"（可能与游戏名不同） |
| 服务器名 | 聊天记录/目录结构 | 如"金色平原" |
| profileId | TRP3_Register映射 | 如"0619121750YI1Nr" |

**关联链路：**
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ 剧情记录     │     │ TRP3_Register│     │ TRP3_Profiles│
│ speaker      │────▶│ character    │────▶│ Profile      │
│ (游戏角色名) │     │ 角色名-服务器│     │ 人物卡详情   │
└──────────────┘     │ → profileId  │     └──────────────┘
                     └──────────────┘
```

**数据结构扩展：**
```typescript
// StoryEntry 扩展（PRD3）
interface StoryEntry {
  speaker: string;           // 显示名（RP名字）
  speakerGameName?: string;  // 游戏角色名
  speakerRealm?: string;     // 服务器名
  profileId?: string;        // 关联的人物卡ID（可选）
}
```

**关联时机：**
1. 导入聊天记录时自动匹配
2. 用户手动关联（匹配失败时）

## 6. 界面原型

### 6.1 首次使用引导

**步骤1：选择WoW安装目录**
```
┌─────────────────────────────────────────┐
│  欢迎使用 RPBox 人物卡同步              │
├─────────────────────────────────────────┤
│  🔍 正在自动检测魔兽世界安装目录...      │
│                                         │
│  ✓ 已找到以下安装位置：                  │
│  ○ C:\Program Files (x86)\World of...  │
│  ○ D:\Games\World of Warcraft\         │
│                                         │
│  [手动选择其他目录...]                   │
├─────────────────────────────────────────┤
│                            [下一步 →]   │
└─────────────────────────────────────────┘
```

**步骤2：扫描人物卡**
```
┌─────────────────────────────────────────┐
│  正在扫描人物卡...                       │
├─────────────────────────────────────────┤
│  账号 563986541#1:                       │
│    ✓ 芙拉莉雅、塔莉恩、克拉丽丝...       │
│    共发现 12 个人物卡                    │
│                                         │
│  账号 331#1:                            │
│    ✓ 共发现 3 个人物卡                   │
├─────────────────────────────────────────┤
│  总计: 15 个人物卡待备份                 │
│                       [开始备份 →]       │
└─────────────────────────────────────────┘
```

**步骤3：登录/注册（未登录时）**
```
┌─────────────────────────────────────────┐
│  登录 RPBox 账号                         │
├─────────────────────────────────────────┤
│                                         │
│  邮箱: [____________________]           │
│  密码: [____________________]           │
│                                         │
│  [登录]                                 │
│                                         │
│  ─────────── 或 ───────────             │
│                                         │
│  [使用 Battle.net 登录]                  │
│  [使用 Discord 登录]                     │
│                                         │
│  还没有账号？ [立即注册]                  │
├─────────────────────────────────────────┤
│  [← 返回]              [跳过，仅本地使用] │
└─────────────────────────────────────────┘
```

**交互说明 - 首次使用引导：**

| 元素 | 交互行为 | 说明 |
|------|----------|------|
| WoW路径单选 | 点击选中 | 高亮显示选中项 |
| 手动选择目录 | 点击打开文件选择器 | 系统原生目录选择对话框 |
| 下一步按钮 | 未选择路径时禁用 | 选择后启用，点击进入扫描 |
| 扫描进度 | 实时更新 | 显示当前扫描的账号和进度 |
| 开始备份 | 点击开始同步 | 未登录则跳转登录页 |
| 跳过登录 | 仅使用本地功能 | 后续可在设置中登录 |

**异常状态 - 未找到WoW：**
```
┌─────────────────────────────────────────┐
│  欢迎使用 RPBox 人物卡同步              │
├─────────────────────────────────────────┤
│  ⚠️ 未能自动检测到魔兽世界安装目录        │
│                                         │
│  请手动选择WoW安装目录：                  │
│  [选择目录...]                          │
│                                         │
│  提示：通常位于以下位置                   │
│  • C:\Program Files (x86)\World of...   │
│  • 战网客户端 → 魔兽世界 → 选项 → 显示   │
├─────────────────────────────────────────┤
│                            [下一步 →]   │
└─────────────────────────────────────────┘
```

**异常状态 - 未找到TRP3数据：**
```
┌─────────────────────────────────────────┐
│  扫描结果                                │
├─────────────────────────────────────────┤
│  ⚠️ 未找到 TotalRP3 人物卡数据            │
│                                         │
│  可能的原因：                            │
│  1. 尚未安装 TotalRP3 插件               │
│  2. 安装后未进入过游戏                    │
│  3. 选择的WoW目录不正确                   │
│                                         │
│  [重新选择目录]  [查看帮助]               │
├─────────────────────────────────────────┤
│  [← 返回]                               │
└─────────────────────────────────────────┘
```

### 6.2 主界面
```
┌─────────────────────────────────────────────┐
│  人物卡管理                      [同步全部]  │
├─────────────────────────────────────────────┤
│  WoW路径: C:\Program Files (x86)\World...   │
│  账号选择: [563986541#1 ▼]  (共4个账号)      │
├─────────────────────────────────────────────┤
│  ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │ 芙拉莉雅 │ │ 塔莉恩   │ │ 克拉丽丝 │       │
│  │ ✓ 已同步 │ │ ⟳ 待同步 │ │ ✓ 已同步 │       │
│  └─────────┘ └─────────┘ └─────────┘       │
├─────────────────────────────────────────────┤
│  最后同步: 2024-01-11 20:00                 │
└─────────────────────────────────────────────┘
```

**人物卡卡片状态：**
```
┌─────────────────────────────────────────────────────────────┐
│  状态图示                                                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │ [头像]  │  │ [头像]  │  │ [头像]  │  │ [头像]  │        │
│  │ 角色名  │  │ 角色名  │  │ 角色名  │  │ 角色名  │        │
│  │ ✓ 已同步│  │ ⟳ 待同步│  │ ↑ 上传中│  │ ⚠ 冲突 │        │
│  │ (绿色) │  │ (橙色) │  │ (蓝色) │  │ (红色) │        │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘        │
└─────────────────────────────────────────────────────────────┘
```

**交互说明 - 主界面：**

| 元素 | 交互行为 | 说明 |
|------|----------|------|
| 人物卡卡片 | 单击 | 进入人物卡详情页 |
| 人物卡卡片 | 右键 | 显示上下文菜单（同步/删除/查看历史） |
| 人物卡卡片 | 悬停 | 显示简要信息Tooltip |
| 账号下拉框 | 点击展开 | 切换不同WoW账号 |
| 同步全部按钮 | 点击 | 同步当前账号所有人物卡 |
| WoW路径 | 点击 | 打开路径设置 |

**右键菜单：**
```
┌─────────────────┐
│ 📤 立即同步     │
│ 📋 查看详情     │
│ 📜 版本历史     │
│ ✏️ 编辑         │
│ ───────────── │
│ 📥 写回本地     │
│ 🗑️ 从云端删除   │
└─────────────────┘
```

### 6.3 人物卡详情页
```
┌─────────────────────────────────────────────┐
│  ← 返回                    [同步] [历史版本] │
├─────────────────────────────────────────────┤
│  ┌──────┐  芙拉莉雅                         │
│  │ 头像 │  人类 · 潜行者                     │
│  │      │  "影子中的低语者"                  │
│  └──────┘                                   │
├─────────────────────────────────────────────┤
│  基本信息                                    │
│  ├─ 名字: 芙拉莉雅                          │
│  ├─ 头衔: 影子中的低语者                     │
│  ├─ 种族: 人类                              │
│  ├─ 职业: 潜行者                            │
│  ├─ 年龄: 26                                │
│  └─ 居住地: 暴风城                          │
├─────────────────────────────────────────────┤
│  外貌描述                                    │
│  一头乌黑的长发...                           │
├─────────────────────────────────────────────┤
│  性格特点                                    │
│  沉默寡言，但内心善良...                      │
├─────────────────────────────────────────────┤
│  历史背景                                    │
│  出生于西部荒野的一个小村庄...                │
├─────────────────────────────────────────────┤
│  同步状态: ✓ 已同步  |  版本: v3            │
│  最后修改: 2024-01-11 20:00                 │
└─────────────────────────────────────────────┘
```

**详情页标签页切换：**
```
┌─────────────────────────────────────────────┐
│  [基本信息] [关于] [性格特征] [其他信息]      │
├─────────────────────────────────────────────┤
│  （根据选中标签显示对应内容）                 │
└─────────────────────────────────────────────┘
```

**交互说明 - 详情页：**

| 元素 | 交互行为 | 说明 |
|------|----------|------|
| 返回按钮 | 点击返回主界面 | 保留滚动位置 |
| 同步按钮 | 点击同步当前人物卡 | 显示同步进度 |
| 历史版本 | 点击进入版本历史页 | - |
| 头像 | 点击放大查看 | 支持游戏图标预览 |
| 标签页 | 点击切换 | 平滑过渡动画 |
| 编辑按钮 | 点击进入编辑模式 | 右上角悬浮 |

### 6.4 版本历史页
```
┌─────────────────────────────────────────────┐
│  ← 返回详情          芙拉莉雅 - 版本历史     │
├─────────────────────────────────────────────┤
│  ┌─────────────────────────────────────┐   │
│  │ v3 (当前)     2024-01-11 20:00      │   │
│  │ 修改了外貌描述                       │   │
│  │                      [查看] [恢复]  │   │
│  └─────────────────────────────────────┘   │
│  ┌─────────────────────────────────────┐   │
│  │ v2            2024-01-10 15:30      │   │
│  │ 更新了历史背景                       │   │
│  │                      [查看] [恢复]  │   │
│  └─────────────────────────────────────┘   │
│  ┌─────────────────────────────────────┐   │
│  │ v1            2024-01-08 10:00      │   │
│  │ 首次备份                            │   │
│  │                      [查看] [恢复]  │   │
│  └─────────────────────────────────────┘   │
├─────────────────────────────────────────────┤
│  共 3 个版本  |  保留最近 10 个版本         │
└─────────────────────────────────────────────┘
```

**版本对比视图：**
```
┌─────────────────────────────────────────────┐
│  版本对比: v2 → v3                          │
├─────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐        │
│  │ v2 (旧版本)  │  │ v3 (新版本)  │        │
│  ├──────────────┤  ├──────────────┤        │
│  │ 外貌描述:    │  │ 外貌描述:    │        │
│  │ 一头乌黑的   │  │ 一头乌黑的   │        │
│  │ 长发...      │  │ 长发垂至腰间 │ ← 变更 │
│  └──────────────┘  └──────────────┘        │
├─────────────────────────────────────────────┤
│  [关闭]                    [恢复到v2]       │
└─────────────────────────────────────────────┘
```

**交互说明 - 版本历史：**

| 元素 | 交互行为 | 说明 |
|------|----------|------|
| 查看按钮 | 点击查看该版本详情 | 只读模式 |
| 恢复按钮 | 点击弹出确认对话框 | 需二次确认 |
| 版本卡片 | 悬停高亮 | 显示详细变更摘要 |
| 对比视图 | 左右滑动 | 支持长文本滚动 |

### 6.5 人物卡编辑页
```
┌─────────────────────────────────────────────┐
│  ← 返回              [保存] [写回本地] [发布] │
├─────────────────────────────────────────────┤
│  基本信息                                    │
│  名字: [芙拉莉雅________]                    │
│  头衔: [影子中的低语者___]                   │
│  种族: [人类___] 职业: [潜行者___]           │
│  年龄: [26__] 身高: [165cm]                  │
├─────────────────────────────────────────────┤
│  头像: [游戏图标] [+ 上传立绘]               │
│  ┌──────┐  ┌──────┐                        │
│  │ 图标 │  │ 立绘 │  ← RPBox扩展图片        │
│  └──────┘  └──────┘                        │
├─────────────────────────────────────────────┤
│  外貌描述                          [富文本]  │
│  ┌─────────────────────────────────────┐   │
│  │ 一头乌黑的长发垂至腰间...            │   │
│  │                                     │   │
│  └─────────────────────────────────────┘   │
├─────────────────────────────────────────────┤
│  历史背景                          [富文本]  │
│  ┌─────────────────────────────────────┐   │
│  │ 出生于西部荒野的一个小村庄...        │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

**交互说明 - 编辑页：**

| 元素 | 交互行为 | 说明 |
|------|----------|------|
| 保存按钮 | 保存到云端 | 创建新版本 |
| 写回本地 | 写入WoW目录 | 需游戏关闭 |
| 发布按钮 | 发布到社区 | 设置可见性 |
| 文本输入框 | 实时保存草稿 | 本地缓存 |
| 富文本切换 | 切换编辑模式 | 支持TRP3颜色代码 |
| 上传立绘 | 打开文件选择 | 支持jpg/png，最大5MB |
| 游戏图标 | 打开图标选择器 | 搜索WoW图标 |

**未保存提示：**
```
┌─────────────────────────────────────────────┐
│  ⚠️ 有未保存的更改                           │
├─────────────────────────────────────────────┤
│  是否保存当前编辑内容？                       │
│                                             │
│  [不保存]        [取消]        [保存]        │
└─────────────────────────────────────────────┘
```

### 6.6 同步进度弹窗

**正常同步进度：**
```
┌─────────────────────────────────────────────┐
│  正在同步...                                 │
├─────────────────────────────────────────────┤
│  ████████████░░░░░░░░  60%                  │
│                                             │
│  ✓ 芙拉莉雅                                 │
│  ✓ 塔莉恩                                   │
│  ↑ 克拉丽丝 (上传中...)                      │
│  ○ 艾琳                                     │
│  ○ 卡洛斯                                   │
├─────────────────────────────────────────────┤
│  已完成 2/5                    [取消]        │
└─────────────────────────────────────────────┘
```

**同步完成：**
```
┌─────────────────────────────────────────────┐
│  同步完成                                    │
├─────────────────────────────────────────────┤
│  ✓ 成功同步 5 个人物卡                       │
│                                             │
│  耗时: 3.2 秒                               │
├─────────────────────────────────────────────┤
│                              [确定]          │
└─────────────────────────────────────────────┘
```

**同步失败：**
```
┌─────────────────────────────────────────────┐
│  同步部分失败                                │
├─────────────────────────────────────────────┤
│  ✓ 成功: 3 个                               │
│  ✗ 失败: 2 个                               │
│                                             │
│  失败详情:                                   │
│  • 克拉丽丝: 网络超时                        │
│  • 艾琳: 服务器错误                          │
├─────────────────────────────────────────────┤
│  [查看日志]              [重试失败] [关闭]   │
└─────────────────────────────────────────────┘
```

### 6.7 冲突处理弹窗

```
┌─────────────────────────────────────────────┐
│  ⚠️ 检测到数据冲突                           │
├─────────────────────────────────────────────┤
│  人物卡: 芙拉莉雅                            │
│                                             │
│  ┌─────────────┐    ┌─────────────┐        │
│  │ 本地版本    │    │ 云端版本    │        │
│  │ 2024-01-11  │    │ 2024-01-11  │        │
│  │ 20:00       │    │ 18:30       │        │
│  │ (较新)      │    │             │        │
│  └─────────────┘    └─────────────┘        │
│                                             │
│  本地修改了: 外貌描述                        │
│  云端修改了: 历史背景                        │
├─────────────────────────────────────────────┤
│  [使用本地]  [使用云端]  [查看详细对比]       │
└─────────────────────────────────────────────┘
```

**交互说明 - 冲突处理：**

| 元素 | 交互行为 | 说明 |
|------|----------|------|
| 使用本地 | 覆盖云端数据 | 本地版本上传 |
| 使用云端 | 覆盖本地数据 | 下载云端版本 |
| 查看详细对比 | 展开对比视图 | 逐字段对比 |

### 6.8 写回本地确认

```
┌─────────────────────────────────────────────┐
│  写回本地确认                                │
├─────────────────────────────────────────────┤
│  即将把人物卡数据写入本地WoW目录：            │
│                                             │
│  目标路径:                                   │
│  C:\...\WTF\Account\563986541#1\            │
│  SavedVariables\totalRP3.lua                │
│                                             │
│  ⚠️ 注意事项:                                │
│  • 请确保魔兽世界已完全关闭                   │
│  • 原文件将自动备份                          │
│  • 下次进入游戏后生效                        │
├─────────────────────────────────────────────┤
│  [取消]                          [确认写入]  │
└─────────────────────────────────────────────┘
```

**WoW运行中提示：**
```
┌─────────────────────────────────────────────┐
│  ⚠️ 检测到魔兽世界正在运行                    │
├─────────────────────────────────────────────┤
│  无法在游戏运行时写入数据文件。               │
│                                             │
│  请先关闭魔兽世界，然后重试。                 │
├─────────────────────────────────────────────┤
│                              [我知道了]      │
└─────────────────────────────────────────────┘
```

### 6.9 设置页面

```
┌─────────────────────────────────────────────┐
│  设置                                        │
├─────────────────────────────────────────────┤
│  账号信息                                    │
│  ├─ 用户名: example@email.com               │
│  ├─ 登录状态: 已登录                         │
│  └─ [退出登录]                              │
├─────────────────────────────────────────────┤
│  WoW路径设置                                 │
│  ├─ 当前路径: C:\Program Files (x86)\...    │
│  └─ [更改路径]                              │
├─────────────────────────────────────────────┤
│  同步设置                                    │
│  ├─ 启动时自动扫描: [✓]                      │
│  ├─ 文件变更提醒:   [✓]                      │
│  └─ 冲突处理策略:   [时间优先 ▼]             │
├─────────────────────────────────────────────┤
│  数据管理                                    │
│  ├─ 本地缓存: 12.5 MB  [清除缓存]            │
│  └─ [导出所有数据]  [导入数据]               │
├─────────────────────────────────────────────┤
│  关于                                        │
│  ├─ 版本: 1.0.0                             │
│  └─ [检查更新]  [查看日志]                   │
└─────────────────────────────────────────────┘
```

### 6.10 全局交互规范

**加载状态：**
```
┌─────────────────────────────────────────────┐
│                                             │
│              ◐ 加载中...                     │
│                                             │
└─────────────────────────────────────────────┘
```

**Toast 提示：**
```
┌─────────────────────────────────────────────┐
│  ✓ 同步成功                          [×]    │  ← 成功(绿色)
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│  ⚠️ 网络连接不稳定                    [×]    │  ← 警告(橙色)
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│  ✗ 同步失败: 服务器错误               [×]    │  ← 错误(红色)
└─────────────────────────────────────────────┘
```

**键盘快捷键：**

| 快捷键 | 功能 | 作用域 |
|--------|------|--------|
| Ctrl+S | 保存 | 编辑页 |
| Ctrl+R | 刷新/重新扫描 | 全局 |
| Ctrl+, | 打开设置 | 全局 |
| Esc | 关闭弹窗/返回 | 全局 |
| F5 | 同步全部 | 主界面 |

**页面导航流程：**
```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  首次使用引导 ──────────────────────────────────────────┐   │
│       │                                                │   │
│       ▼                                                │   │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐            │   │
│  │ 主界面  │◄──►│ 详情页  │◄──►│ 编辑页  │            │   │
│  └────┬────┘    └────┬────┘    └─────────┘            │   │
│       │              │                                 │   │
│       │              ▼                                 │   │
│       │         ┌─────────┐                           │   │
│       │         │版本历史 │                           │   │
│       │         └─────────┘                           │   │
│       │                                                │   │
│       ▼                                                │   │
│  ┌─────────┐                                          │   │
│  │ 设置页  │◄─────────────────────────────────────────┘   │
│  └─────────┘                                              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 7. 里程碑

| 阶段 | 功能 | 状态 |
|------|------|------|
| M1 | Lua解析器 + 本地扫描 | 待开发 |
| M2 | 云端上传/下载 | 待开发 |
| M3 | 版本管理 | 待开发 |
| M4 | 云端预览 | 待开发 |

## 8. 测试方法与用例

### 8.1 测试策略

**测试层级：**

| 层级 | 说明 | 工具 |
|------|------|------|
| 单元测试 | 核心函数、解析器、数据转换 | Vitest (前端) / Go testing (后端) |
| 集成测试 | API接口、数据库操作、文件系统 | Vitest + MSW / Go testing |
| E2E测试 | 完整用户流程 | Playwright / Tauri Driver |
| 手动测试 | 边界情况、真实WoW数据 | 测试清单 |

**测试数据准备：**
- 准备多套真实TRP3数据样本（不同版本、不同语言）
- 构造边界测试数据（空数据、超大数据、特殊字符）
- Mock云端API响应

### 8.2 单元测试用例

#### 8.2.1 Lua解析器测试

| 用例ID | 用例名称 | 输入 | 预期输出 | 优先级 |
|--------|----------|------|----------|--------|
| LUA-001 | 解析空文件 | 空的.lua文件 | 返回空对象，无错误 | P0 |
| LUA-002 | 解析标准Profile | 完整的TRP3_Profiles数据 | 正确解析所有字段 | P0 |
| LUA-003 | 解析中文内容 | 包含中文名字、描述的Profile | 正确解码UTF-8 | P0 |
| LUA-004 | 解析特殊字符 | 包含换行、引号、转义符的文本 | 正确处理转义 | P0 |
| LUA-005 | 解析嵌套表 | 多层嵌套的Lua table | 正确解析层级结构 | P0 |
| LUA-006 | 解析损坏文件 | 语法错误的.lua文件 | 返回解析错误，不崩溃 | P0 |
| LUA-007 | 解析超大文件 | >10MB的Profile数据 | 正常解析，性能可接受(<5s) | P1 |
| LUA-008 | 解析旧版本数据 | TRP3旧版本格式 | 兼容解析或提示升级 | P1 |

#### 8.2.2 字段映射测试

| 用例ID | 用例名称 | 输入 | 预期输出 | 优先级 |
|--------|----------|------|----------|--------|
| MAP-001 | 映射characteristics | Lua缩写字段 | 正确映射为完整字段名 | P0 |
| MAP-002 | 映射about模板1 | T1结构 | 正确提取文本内容 | P0 |
| MAP-003 | 映射about模板2 | T2数组结构 | 正确提取多框架内容 | P0 |
| MAP-004 | 映射about模板3 | T3结构(PH/PS/HI) | 正确提取三段内容 | P0 |
| MAP-005 | 映射性格特征 | PS数组（预设+自定义） | 正确区分预设ID和自定义 | P0 |
| MAP-006 | 映射其他信息 | MI数组 | 正确映射预设类型 | P0 |
| MAP-007 | 映射缺失字段 | 部分字段为nil | 使用默认值，不报错 | P0 |
| MAP-008 | 映射音乐字段 | MU字段（数字ID） | 正确保留音乐ID | P1 |

#### 8.2.3 Extended数据解析测试

| 用例ID | 用例名称 | 输入 | 预期输出 | 优先级 |
|--------|----------|------|----------|--------|
| EXT-001 | 解析道具数据库 | TRP3_Tools_DB | 正确解析所有道具定义 | P0 |
| EXT-002 | 解析背包数据 | inventory结构 | 正确解析槽位和嵌套容器 | P0 |
| EXT-003 | 解析任务日志 | questlog结构 | 正确解析战役和任务状态 | P1 |
| EXT-004 | 解析光环数据 | auras结构 | 正确解析光环和过期时间 | P1 |
| EXT-005 | 解析道具脚本 | SC字段（工作流） | 保留原始脚本结构 | P1 |
| EXT-006 | 解析内部对象 | IN字段（嵌套道具） | 正确解析递归结构 | P1 |

#### 8.2.4 本地扫描测试

| 用例ID | 用例名称 | 输入 | 预期输出 | 优先级 |
|--------|----------|------|----------|--------|
| SCAN-001 | 扫描单账号 | 单个WoW账号目录 | 正确识别所有Profile | P0 |
| SCAN-002 | 扫描多账号 | 多个WoW账号目录 | 正确区分不同账号 | P0 |
| SCAN-003 | 检测WoW路径 | 常见安装位置 | 自动发现有效路径 | P0 |
| SCAN-004 | 处理无效路径 | 不存在的目录 | 返回错误提示 | P0 |
| SCAN-005 | 处理权限问题 | 无读取权限的目录 | 返回权限错误 | P1 |
| SCAN-006 | 增量扫描 | 已扫描+新增文件 | 仅处理变更部分 | P1 |

### 8.3 集成测试用例

#### 8.3.1 API接口测试

| 用例ID | 用例名称 | 接口 | 测试内容 | 优先级 |
|--------|----------|------|----------|--------|
| API-001 | 上传人物卡 | POST /profiles | 正常上传、字段验证 | P0 |
| API-002 | 获取人物卡列表 | GET /profiles | 分页、筛选、排序 | P0 |
| API-003 | 获取人物卡详情 | GET /profiles/:id | 完整数据返回 | P0 |
| API-004 | 更新人物卡 | PUT /profiles/:id | 部分更新、版本冲突 | P0 |
| API-005 | 删除人物卡 | DELETE /profiles/:id | 软删除、权限验证 | P0 |
| API-006 | 版本历史 | GET /profiles/:id/versions | 历史记录查询 | P1 |
| API-007 | 版本回滚 | POST /profiles/:id/rollback | 回滚到指定版本 | P1 |
| API-008 | 未授权访问 | 无Token请求 | 返回401错误 | P0 |
| API-009 | 跨用户访问 | 访问他人数据 | 返回403错误 | P0 |

#### 8.3.2 同步流程测试

| 用例ID | 用例名称 | 场景 | 预期结果 | 优先级 |
|--------|----------|------|----------|--------|
| SYNC-001 | 首次同步 | 本地有数据，云端无数据 | 全量上传成功 | P0 |
| SYNC-002 | 增量上传 | 本地有变更 | 仅上传变更部分 | P0 |
| SYNC-003 | 云端恢复 | 云端有数据，本地无数据 | 下载并写入本地 | P0 |
| SYNC-004 | 无变更同步 | 本地云端一致 | 跳过同步，显示已同步 | P0 |
| SYNC-005 | 冲突检测 | 本地云端都有变更 | 提示冲突，让用户选择 | P0 |
| SYNC-006 | 网络中断 | 同步过程中断网 | 自动重试，失败后提示 | P1 |
| SYNC-007 | 大量数据同步 | 100+人物卡 | 分批处理，显示进度 | P1 |

### 8.4 E2E测试用例

| 用例ID | 用例名称 | 步骤 | 预期结果 | 优先级 |
|--------|----------|------|----------|--------|
| E2E-001 | 首次使用流程 | 安装→选择WoW路径→扫描→登录→同步 | 完整流程无阻塞 | P0 |
| E2E-002 | 查看人物卡详情 | 点击人物卡→查看各标签页 | 所有字段正确显示 | P0 |
| E2E-003 | 手动同步 | 点击同步按钮→等待完成 | 显示同步结果 | P0 |
| E2E-004 | 版本回滚 | 查看历史→选择版本→确认回滚 | 数据恢复到指定版本 | P1 |
| E2E-005 | 冲突解决 | 触发冲突→选择版本→确认 | 按选择结果同步 | P1 |
| E2E-006 | 批量操作 | 选择多个→批量同步/删除 | 批量操作成功 | P1 |

### 8.5 写回本地测试

| 用例ID | 用例名称 | 前置条件 | 预期结果 | 优先级 |
|--------|----------|----------|----------|--------|
| WB-001 | 正常写回 | WoW未运行 | 成功写入，游戏可读取 | P0 |
| WB-002 | WoW运行中写回 | WoW正在运行 | 阻止写入，提示关闭游戏 | P0 |
| WB-003 | 自动备份 | 写回前 | 创建.rpbox_backup文件 | P0 |
| WB-004 | 格式保持 | 写回后 | Lua格式正确，TRP3可读 | P0 |
| WB-005 | 中文编码 | 包含中文的数据 | UTF-8编码正确 | P0 |
| WB-006 | 写回失败回滚 | 写入过程出错 | 恢复备份文件 | P1 |

### 8.6 测试数据样例

#### 8.6.1 最小有效Profile

```lua
TRP3_Profiles = {
  ["test001"] = {
    ["profileName"] = "测试角色",
    ["player"] = {
      ["characteristics"] = {
        ["v"] = 1,
        ["FN"] = "测试",
        ["IC"] = "Achievement_Character_Human_Female",
      },
    },
  },
}
```

#### 8.6.2 完整Profile样例

```lua
TRP3_Profiles = {
  ["full001"] = {
    ["profileName"] = "完整测试",
    ["player"] = {
      ["characteristics"] = {
        ["v"] = 3,
        ["FN"] = "芙拉莉雅",
        ["LN"] = "海顿",
        ["TI"] = "影子中的低语者",
        ["FT"] = "Fluralia Hayden",
        ["RA"] = "人类",
        ["CL"] = "潜行者",
        ["AG"] = "26",
        ["EC"] = "翠绿色",
        ["EH"] = "00ff00",
        ["HE"] = "165cm",
        ["WE"] = "纤细",
        ["BP"] = "西部荒野",
        ["RE"] = "暴风城",
        ["RS"] = 1,
        ["IC"] = "Achievement_Character_Human_Female",
        ["CH"] = "ffffff",
        ["MI"] = {
          { ["ID"] = 3, ["NA"] = "昵称", ["VA"] = "小芙", ["IC"] = "Ability_Rogue_Disguise" },
          { ["ID"] = 4, ["NA"] = "座右铭", ["VA"] = "影中行，心向光", ["IC"] = "INV_Misc_Book_09" },
        },
        ["PS"] = {
          { ["ID"] = 1, ["V2"] = 15 },
          { ["ID"] = 6, ["V2"] = 8 },
        },
      },
      ["about"] = {
        ["v"] = 2,
        ["TE"] = 3,
        ["MU"] = 53183,
        ["T3"] = {
          ["PH"] = { ["TX"] = "一头乌黑的长发..." },
          ["PS"] = { ["TX"] = "沉默寡言但内心善良..." },
          ["HI"] = { ["TX"] = "出生于西部荒野..." },
        },
      },
      ["character"] = {
        ["v"] = 1,
        ["RP"] = 1,
        ["WU"] = 2,
        ["CU"] = "正在酒馆角落独饮",
      },
    },
  },
}
```

#### 8.6.3 Extended道具样例

```lua
TRP3_Tools_DB = {
  ["玩家名-服务器_item001"] = {
    ["TY"] = "IT",
    ["MD"] = {
      ["MO"] = "NORMAL",
      ["V"] = 1,
      ["CB"] = "玩家名-服务器",
    },
    ["BA"] = {
      ["NA"] = "测试背包",
      ["DE"] = "一个用于测试的背包",
      ["IC"] = "INV_Misc_Bag_01",
      ["QA"] = 1,
      ["WE"] = 500,
      ["CT"] = true,
    },
    ["CO"] = {
      ["SI"] = "5x4",
      ["SR"] = "5",
      ["SC"] = "4",
    },
  },
}
```

#### 8.6.4 边界测试数据

| 数据类型 | 描述 | 用途 |
|----------|------|------|
| 空Profile | 仅有profileId，无player数据 | 测试空数据处理 |
| 超长文本 | 描述字段>10000字符 | 测试大文本处理 |
| 特殊字符 | 包含`\n\r\t\"\'\\`等 | 测试转义处理 |
| Unicode | 包含emoji、日文、韩文 | 测试多语言支持 |
| 深层嵌套 | 10层以上嵌套容器 | 测试递归解析 |
| 大量数据 | 500+人物卡 | 测试性能边界 |

### 8.7 测试覆盖率要求

| 模块 | 最低覆盖率 | 说明 |
|------|------------|------|
| Lua解析器 | 90% | 核心功能，必须高覆盖 |
| 字段映射 | 85% | 数据转换逻辑 |
| API接口 | 80% | 后端服务 |
| 同步逻辑 | 80% | 核心业务流程 |
| UI组件 | 60% | 前端展示 |

### 8.8 验收标准

**功能验收：**
- [ ] 能正确解析真实TRP3数据文件
- [ ] 能正确解析Extended道具数据
- [ ] 同步功能正常工作（上传/下载）
- [ ] 版本历史和回滚功能正常
- [ ] 写回本地后游戏能正常读取

**性能验收：**
- [ ] 扫描100个Profile < 5秒
- [ ] 单个Profile同步 < 2秒
- [ ] 应用启动 < 3秒

**兼容性验收：**
- [ ] 支持TRP3 2.x版本数据
- [ ] 支持Extended数据
- [ ] 支持中文/英文客户端