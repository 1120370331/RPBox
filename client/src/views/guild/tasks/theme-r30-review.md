# 公会页面主题修复交接（revision 30）

指示：07258fac-3de1-4271-9107-22b7e8ca8fb7。仅修改 `client/src/views/guild/`。工作区中的公共组件、其他页面和既有 beta 发布材料由其他所有者负责，本 Worker 未改写这些文件。

页面级实现完成，**尚不能标记集成验收通过**：公共姓名染色有明确低对比度缺陷，需 W1 处理；四套布局的真实入口也需要主线程确认。

## 问题与修复

| 页面 | 原问题 | 修复 |
| --- | --- | --- |
| 公会列表 | 加载文字固定棕色；任意品牌色图标固定白字；浅色高光干扰字母对比度 | 主题辅助文字；保持原品牌背景，按相对亮度选择黑/白前景；默认图标使用成对的主题渐变与文字 |
| 创建 | 固定深色标题/标签、白色下拉框和固定边框 | 主题文字、输入表面、焦点和占位符；原品牌色上传预览保留，提示加遮罩 |
| 详情/设置 | 头图徽章和按钮与全局 `.badge`/`.btn-outline` 冲突；成员头像渐变与文字不成对；设置下拉框没有明确前景色 | 使用页面专用类；照片区域保留白字及遮罩；普通表面用主题文字；设置控件使用输入变量和局部原生控件 color-scheme |
| 管理 | 头像被通用高光样式覆盖；成员/申请头像和会长徽章在经典主题中采用暗色文字配暗色渐变 | 页面头像专用类；头像使用主题渐变文字对；会长徽章使用 badge/primary-text；审批状态文字保留语义边框并提高可读性 |
| 帖子 | 搜索表面固定半透明白色；粉蓝主题中辅助文字/悬停文字直接采用浅色装饰色 | 搜索表面跟随输入主题；文本使用可读主色；精选标签、作者头像采用成对渐变与文字 |
| 剧情 | 自定义标签直接用品牌色作正文，黑色标签在黑金表面不可读；选中态前景借用系统按钮色，不对应标签背景 | 未选中/剧情标签采用主题文字和原品牌圆点；选中时保留原品牌实色并计算黑/白前景；日期与下拉控件局部适配原生深色样式 |
| 布局 1 | 固定浅背景、固定深标题；全局 `.panel` 覆盖排行榜特殊背景 | 主题表面/文字，排行榜保留独立渐变；使用公会专用 panel 类 |
| 布局 2 | 编辑式布局的表面、标题、统计、成员、分隔线均为固定咖啡色 | 全部普通表面和文字采用主题变量；保留版式、字体、图片灰度效果和照片遮罩 |
| 布局 3 | 固定浅色卡片；`.content-card` 被全局面板规则覆盖，照片白字失去原本深色背景 | 普通卡片主题化；照片内容卡专用类并保留深遮罩；保留圆角与照片布局 |
| 布局 4 | 浅色拟物表面/白色阴影固定；日期徽章受全局 badge 背景干扰；照片右上等级无局部遮罩 | 主题相对渐变及凸起/内凹阴影，保留布局特色；照片徽章和等级独立遮罩 |

固定颜色保留在照片遮罩、阵营品牌和装饰图标上。联盟和中立徽章的局部蓝/灰色深度调整用于提高其白字对比度；未修改公会、标签、成员保存的颜色值。

新增 `guildColorStyle.ts` 仅供三个公会页面使用，是页面局部展示逻辑，不修改公共工具或共享接口。`guildColorStyle` 保留有效的六位品牌色背景；`guildTagStyle` 只产生展示 CSS，不接触保存数据。

## 针对性检查

从 `client/` 执行：

| 命令 | 结果 |
| --- | --- |
| `node src/views/guild/tasks/theme-r30-check.mjs` | PASS：10 个 Vue 文件脚本/模板/样式编译；深色原生控件规则保持页面 scope；与 HEAD 比对保存 API/脚本及 buildNameStyle 调用保持原样；8 组保存品牌色前景对比度 4.69–21.00；4 种主题的主文案/按钮/渐变端点配对对比度通过 |
| `npx --no-install tsc --noEmit --target es2020 --moduleResolution bundler --module esnext --strict --skipLibCheck src/views/guild/guildColorStyle.ts` | PASS：新增页面局部模块类型检查 |
| `npm run test:run -- src/views/guild/GuildManage.test.ts` | PASS：最终公会样式候选的 2 个已有管理测试；15:19:18 开始，3.18 秒完成；普通权限重跑以绕过沙箱 esbuild 目录读取限制 |
| `npx --no-install vue-tsc --noEmit` | FAIL：全项目路径别名未配置导致大量 `@/` 模块解析错误，另有未使用变量、unknown 和现有 API 类型问题；未修改全局 tsconfig，不宣称全项目类型检查通过 |
| `git diff --check -- client/src/views/guild` | PASS |

最初定向测试在沙箱中因 esbuild 无法读取上层目录而启动失败；尝试 native/runner configLoader 又因现有配置使用 `__dirname` 启动失败。之后原测试命令在普通权限下通过，没有改写 Vite 配置。

检查脚本为本次交接的静态诊断材料，不是浏览器验收，也不证明后台实际持久化。

## 交 W1 的公共缺陷

`client/src/utils/userNameStyle.ts:1` 中 `buildNameStyle` 仍直接返回保存的 `color`。当前 `buildNameStyle('#000000', true)` 返回黑字，黑金面板 `#181818` 上对比度仅 **1.18**。默认文案已在本任务修复，显式染色仍依赖公共适配。冻结的函数签名和调用均未变更。

直接消费者：

- `GuildDetail.vue:545`：成员名；背景为主题面板。
- `GuildManage.vue:408`、`:480`：成员名和申请人；背景为主题卡片，悬停有卡片覆盖。
- `GuildPosts.vue:256`：作者名；背景为主题面板。
- `GuildStories.vue:432`：上传人名；背景为主题面板。
- `layouts/GuildLayout1.vue:109`、`:122`：成员名与排行榜；**排行榜为独立主题渐变，需要结合其实际背景处理**。
- `layouts/GuildLayout2.vue:107`、`:165`：会长及成员；主题面板。
- `layouts/GuildLayout3.vue:130`：成员名；主题面板及悬停覆盖。
- `layouts/GuildLayout4.vue:124`、`:163`：贡献成员与动态姓名；主题卡片/渐变表面。

以上应交 W1/主线程修复后再次进行染色检查，不能按当前静态 PASS 宣称全部姓名可读。

## 集成发现与未验证项

- `rg -n 'GuildLayout[1234]|guild.layout' client/src --glob '*.vue'` 没有命中；`GuildDetail.vue` 使用一套独立模板，未引用四个布局组件，也未消费 `guild.layout`。本 Worker 不将主题修复扩展为布局路由/产品功能重建。主线程需确认四套布局的真实验收入口，或明确采用单独组件验证。
- 没有提供图片：七份附件为源码、差异、发布说明、日志、报告、安装包和签名元数据，`references` 为空；没有可检查的图片引用。
- 未跑真实公会登录/API、表单保存、富文本染色内容、照片背景矩阵、浏览器/桌面深浅切换或安装回归。
- 未跑生产构建、未占用新的 beta 版本、未发布、未安装；既有 beta 1.0.5-11 不是本次新增修复的验收证据。
- 其他 Worker 对公共样式与控件的并行修改需要主线程整合验证；未覆盖或撤销其改动。

主线程建议实测路径：公会列表 → 创建/已有关联公会 → 详情 → 设置（含阵营、品牌色和隐私开关）→ 成员管理/申请 → 帖子 → 剧情标签未选中/选中/日期过滤。对可达四套布局分别切换经典与黑金，再抽查粉蓝/青绿的浅色装饰文字。用黑/白标签及深/浅姓名染色验证实际背景，保存前后比较颜色字段。

## 源码候选

基线 HEAD：`8e73130faa429af6de100eaa540768f77b4ee9d5`。`theme-r30.patch` 包含本 Worker 的 10 个 Vue 文件和新增页面局部模块，排除了其他 Worker 的修改和此交接文档。下面的 SHA256 清单标识本 Worker 的源码候选。


| 文件 | SHA256 |
| --- | --- |
| `guildColorStyle.ts` | `4a55e659d1fa18881d975720dc174ea2b8d086818c682d13b4e0de0f3ae083d5` |
| `GuildCreate.vue` | `85784acb38d0eaf9a8090a4b26eec0d7ddb75b97278dbac1d892831f1a0fa81a` |
| `GuildDetail.vue` | `792a7204a9adc49478590c1be6b4e92add5f43d5d5b95a92478f8715ce06f9c7` |
| `GuildList.vue` | `246aaf32222c640fec521da942b9c040447d682a1965ac0ea563af99ca1d6863` |
| `GuildManage.vue` | `238b0706f5fb5301b378abcaab69d07d01968da7dbba7a5218c7b4df271a43fc` |
| `GuildPosts.vue` | `d8f12dd473cbacb08ef2ac056d4736ff82d5532fcad45782715029ed7e8a927f` |
| `GuildStories.vue` | `637f7fa64c1001f76a3c597397542a9d113c42ff5cf192caad9b96d3fd9e6169` |
| `layouts/GuildLayout1.vue` | `4f216f30a55bdf5f10747f4207264d9121ded70a0739d666b3bff3a9cc8a58d7` |
| `layouts/GuildLayout2.vue` | `df28646161fae2ef08bce9b3d694026e71614a19016f5ace93fcdedb69534afa` |
| `layouts/GuildLayout3.vue` | `b360c45ad4796c2728fe13f17d0f69f6023160d2b031ddf1fcbe1d46e586a91f` |
| `layouts/GuildLayout4.vue` | `96f4859db9f1bd6c9e7b60b9033444f6d1e904339ff4573c97ecd7df138cf2fb` |

差异文件 SHA256：`22467608a89fdfca88c1df1958beb3bc456d1975cfbd53357f66b28bd551d3b5`。
