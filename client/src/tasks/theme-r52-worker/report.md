# W1 源码收口交付（revision 52）

指示 ID：d16e134f-30b3-44cd-b557-35d1cd0e49b8。

姓名主题修复候选已交付。**全量类型检查仍失败：10 条诊断均为公会 server 字段与既有 DTO 不一致，需要主代理作契约决策，不能发布为类型检查通过的候选。** 本 Worker 未发布、未安装新版 beta，也未执行真实公会／社区桌面路由验收。

## 所有权与候选

只修改 client/src/。未修改类型检查配置、依赖、版本、发布文件，未创建子代理，未提交。基准 HEAD 为 8e73130faa429af6de100eaa540768f77b4ee9d5；工作开始时已存在大量其他 Worker 的未提交源码修改。

worker-incremental.patch 相对于本 Worker 开始时的工作树生成，包含本 Worker 的源码增量，不包含之前交付的主题改造。manifest.json 列出 58 个源码文件的实际字节 SHA256，并记录读取的配置哈希；这些配置未由本 Worker 修改。审查材料位于 client/src/tasks/theme-r52-worker/，可从产品提交中排除。

## 具体缺陷及修复

| 缺陷 | 修复与消费者 |
| --- | --- |
| buildNameStyle 直接返回保存色，黑字在 #181818 面板上 1.18:1、白字在白底上 1:1 | userNameStyle.ts 读取当前主题，按普通 panel/card、实际 alpha hover 背景计算显示色，低对比度色向可读端点最小幅度混合至 4.5:1；可读原色保留。既有两参数调用继续有效，第三参数允许主题背景键、CSS RGB/RGBA 色或多个渐变端点。 |
| 侧栏姓名使用面板背景计算会反向变暗 | AppLayout.vue 的姓名明确传 sidebarBg；头像字母与 gradient-text 配对。 |
| 公会排行榜与人物卡铭牌是独立渐变 | GuildLayout1.vue 排行榜、CharacterCardWall.vue 名牌明确传 gradientStart/gradientEnd。其他 GuildDetail/Manage/Posts/Stories、社区作者评论、用户主页等现有两参数消费者通过公共函数获取主题显示色。 |
| 用户头像悬浮卡的昵称及公开人物卡染色未适配 | UserAvatarPopover.vue 身份标题使用实际渐变端点，公开人物卡姓名使用普通表面。头像字母使用成对渐变文字；移除身份标题背景上的额外径向亮斑，确保声明背景与实际文字背景一致。RouterLink 元素引用改为结构化 unknown 加 HTMLElement 检查，未新增 any。 |
| 公共剧情角色弹层显示原始 TRP3 黑/白色 | CharacterCard.vue 的标题使用实际 cardBg；标题背景改为主题卡片表面。CharacterCardJumpPreview.vue 接入公共姓名显示。所有 TRP3 RGB/ARGB 仍先使用既有 normalizeCharacterCardHexForCSS；CharacterCardColorField 的真实色值及 v-model 不变。 |
| 多处亮度/合成/对比度算法重复 | 新增 utils/displayColor.ts，复用于姓名、StoryDetail、archiveDisplayColors、guildColorStyle、TagSelector；剧情的 alpha 合成与既有 rgb 输出行为保留。没有回写角色、条目、标签、公会或用户保存的染色。 |
| 编辑器的用户建议列表直接染色且打开时换主题不更新 | TiptapEditor.vue 建议姓名统一使用 buildNameStyle，并监听主题切换更新当前打开的 DOM 建议列表；原始颜色留在仅供显示的 dataset，不影响 mention 写入内容。 |
| 公共覆盖移除后头像字母／RPDB 作者姓名仍有局部低对比度 | Settings.vue 使用 gradient-text；RPDBWorkCard.vue 作者色先作既有 hex 规范化再使用显示适配。 |
| 部分集合、帖子编辑页面调用不存在的 useToast.warning | 四个消费者改为已有 useToastStore/RToast；AppLayout 不重复挂载 RToast，保留 App.vue 的全局入口。 |

## 源码类型错误收口

- Tiptap setContent 适配已安装 v3 的 `{ emitUpdate: false }` 参数；删除未使用的私有插入函数。
- 公共更新弹窗使用 RButton 支持的 lg 尺寸；删除未读取的解构字段。useUpdater 仅移除 SDK 不提供的 downloadUrl 调试日志，不改更新流程。
- PostDetail 的回复姓名使用本地 DisplayReply 类型，移除两个 as any。PostEdit 与 ItemEdit 的本地表单将编辑器需要的字符串字段设为必有值；不更改 API DTO。PostEdit 将服务端 pending 审核状态映射为表单的发布意图，符合既有 UpdatePostRequest 的 draft/published 契约，不修改取回的 post 对象。
- GuildStories 的筛选参数由实际 Story 列表推导，避免将可选 uploader 误当必有字段。
- forumLevel 的泛型组合改用合法交叉类型；jumpLink 的 Promise 回调标注既有结果联合，不改变返回值。
- accountBackup 的 version 字段只有为对象时才解包，避免把完整版本记录的数字版本号返回给调用者；实际请求和返回契约不变。
- RPDB 编辑器使用浏览器 number 定时器类型与本地标签形状；回复组件 ref 以 unknown 做能力检查，无 any 掩盖。
- ItemUpload 的 form submit 显式传 published，避免 SubmitEvent 被误作业务状态；MyItems 使用现有 dialog.error 类型。
- 清除编译器确认未使用的私有声明／导入，修正测试夹具缺失的 class_color/visibility、错误的 RPDB type、DOM wrapper 用法和 mock 类型；不关闭严格检查，不改变共享业务接口。

## 针对性检查

在 client/ 下执行：

1. `npm run test:run -- src/__tests__/userNameStyle.test.ts src/components/UserAvatarPopover.test.ts src/components/CharacterCard.test.ts src/components/AppLayout.test.ts src/components/character-cards/CharacterCardJumpPreview.test.ts src/components/character-cards/CharacterCardWall.test.ts src/views/guild/GuildManage.test.ts src/utils/characterCardColor.test.ts src/__tests__/forumLevel.test.ts src/components/rpdb/RPDBWorkCard.test.ts src/components/rpdb/RPDBWorkContent.test.ts src/views/rpdb/RPDBEditor.test.ts src/views/rpdb/RPDBDetail.test.ts src/views/archives/StagingPool.test.ts`
   - PASS：14 文件、95 测试，18:11:19 开始，8.28 秒。
   - 姓名矩阵覆盖四主题、9 个染色、普通/卡片/hover/侧栏/渐变共 396 个配色断言，保留粗体和冻结的保存值。
   - 含已打开 Teleport 用户悬浮卡的真实 Vue DOM 主题切换，以及 TRP3 ARGB 字段不变断言；这是组件级测试，**不是安装版的真实路由验收**。
2. 后续改变名牌背景信息、hover 合成及测试路由后，仅重跑 `npm run test:run -- src/__tests__/userNameStyle.test.ts src/components/UserAvatarPopover.test.ts src/components/character-cards/CharacterCardWall.test.ts src/components/AppLayout.test.ts`
   - PASS：4 文件、13 测试，18:13:50 开始，2.71 秒。其他无行为变化的空行和重复类型字段清理由 SFC 检查覆盖。
3. `node src/tasks/theme-r52-worker/check.mjs`
   - PASS：33 个修改后的 Vue SFC 脚本／模板／样式编译，且当前源码 SHA256 与 manifest 相同。
   - PASS：真实 accountBackup 源码在 mocked request adapter 下的 8 个响应形状断言，数字版本号及冻结记录保留。不是后端 API 联调。
   - 原始输出见 checks.log。
4. `npx --no-install vue-tsc --noEmit --pretty false`
   - FAIL：剩余 10 个公会 server 字段诊断，原始输出见 typecheck.log；没有将失败宣称通过。
5. 仓库根目录 `git diff --check -- client/src`
   - PASS，退出 0。Git 仅提示 Windows 的 LF/CRLF 自动转换，不是差异空白错误。

最初四个测试文件在沙箱中因 esbuild 无法读取上层目录而启动失败；普通权限运行原命令后通过。没有为绕过限制修改配置。部分既有 RPDB 测试输出缺少测试路由的 Vue Router 警告，测试均通过。未执行全库测试、生产构建或 beta 发布。

## 未完成项与主代理交接

1. **类型门禁阻塞：Guild.server。** 前端 api/guild.ts 的 Guild/CreateGuildRequest 无 server；后端 model/model.go 的 Guild 也无 server。GuildCreate.vue 收集并提交该字段，GuildList.vue 和四套独立布局读取该字段。不能仅给 DTO 添加虚构字段来掩盖诊断。请主代理决定删除当前不受支持的 UI 功能，或另行实现有持久化契约的服务器字段；前后端契约扩展不在本次主题修复决策范围内。
2. 四套 GuildLayout 的真实接入仍交主代理核查，未自行重建公会路由。此前的 theme-r30-check.mjs 冻结旧函数调用文本及旧算法引用，不能作为本候选姓名适配的通过证据；新检查使用本次 manifest 与测试。
3. 未验证实际新版安装包、公会／剧情／社区／用户页面真实 API 及保存往返；主代理需完成新版深浅主题安装验收。额外应抽查本次源码类型修复涉及的集合消息、帖子编辑、道具提交及账号备份版本响应。
4. 附件中的浅色与黑金公共表单 PNG 已查看；它们是历史夹具截图，未冒充本候选运行截图。
5. 静态发现 App.vue 与 AppLayout.vue 都挂载 RDialog，本 Worker 保留该既有实现；请主代理在真实布局路径确认是否重复显示确认弹窗。

## 需审查源码文件

- `client/src/__tests__/conflict.test.ts`
- `client/src/__tests__/fieldMapper.test.ts`
- `client/src/__tests__/syncService.test.ts`
- `client/src/__tests__/userNameStyle.test.ts`
- `client/src/api/accountBackup.ts`
- `client/src/components/AppLayout.vue`
- `client/src/components/CharacterCard.test.ts`
- `client/src/components/CharacterCard.vue`
- `client/src/components/CollectionBanner.vue`
- `client/src/components/CollectionSelector.vue`
- `client/src/components/PostQuickJump.test.ts`
- `client/src/components/RInput.vue`
- `client/src/components/RTabs.vue`
- `client/src/components/StoryFilter.vue`
- `client/src/components/TagSelector.vue`
- `client/src/components/TiptapEditor.vue`
- `client/src/components/UpdateNotification.vue`
- `client/src/components/UserAvatarPopover.test.ts`
- `client/src/components/UserAvatarPopover.vue`
- `client/src/components/character-cards/CharacterCardJumpPreview.test.ts`
- `client/src/components/character-cards/CharacterCardJumpPreview.vue`
- `client/src/components/character-cards/CharacterCardWall.test.ts`
- `client/src/components/character-cards/CharacterCardWall.vue`
- `client/src/components/rpdb/RPDBWorkCard.test.ts`
- `client/src/components/rpdb/RPDBWorkCard.vue`
- `client/src/components/rpdb/RPDBWorkContent.test.ts`
- `client/src/components/sync/ConflictDialog.vue`
- `client/src/composables/useDialog.ts`
- `client/src/composables/useUpdater.ts`
- `client/src/stores/locale.ts`
- `client/src/utils/displayColor.ts`
- `client/src/utils/forumLevel.ts`
- `client/src/utils/imagePreview.ts`
- `client/src/utils/jumpLink.ts`
- `client/src/utils/userNameStyle.ts`
- `client/src/views/Settings.vue`
- `client/src/views/archives/StoryDetail.vue`
- `client/src/views/archives/StoryList.vue`
- `client/src/views/archives/archiveDisplayColors.ts`
- `client/src/views/character-cards/CharacterCardDetail.test.ts`
- `client/src/views/character-cards/CharacterCardEditor.test.ts`
- `client/src/views/collection/CollectionDetail.vue`
- `client/src/views/community/PostDetail.vue`
- `client/src/views/community/PostEdit.vue`
- `client/src/views/guild/GuildDetail.vue`
- `client/src/views/guild/GuildPosts.vue`
- `client/src/views/guild/GuildStories.vue`
- `client/src/views/guild/guildColorStyle.ts`
- `client/src/views/guild/layouts/GuildLayout1.vue`
- `client/src/views/market/ItemEdit.vue`
- `client/src/views/market/ItemUpload.vue`
- `client/src/views/market/MyItems.vue`
- `client/src/views/moderator/ModeratorMain.vue`
- `client/src/views/rpdb/RPDBDetail.vue`
- `client/src/views/rpdb/RPDBEditor.test.ts`
- `client/src/views/rpdb/RPDBEditor.vue`
- `client/src/views/rpdb/RPDBLists.vue`
- `client/src/views/sync/SyncMain.vue`
