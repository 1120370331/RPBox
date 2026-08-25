# RPBox Apple App Store 审核清单

更新日期：2026-08-25

本文是 RPBox iOS 提交前的仓库证据与人工门禁清单，不代表 App Store Connect 已配置完成，也不代表当前构建已经通过真机、TestFlight 或 Apple 审核。所有标为 `BLOCKED` 的项目在提交前必须由发布负责人关闭。

审核基准：[Apple App Review Guidelines](https://developer.apple.com/app-store/review/guidelines/)，规则于 2026-08-24 核对。Apple 可能随时更新要求，实际提交前应再次检查官方原文。

状态含义：

- `PASS`：当前仓库已有直接证据；若条目同时要求线上或真机状态，仍须完成写明的发布检查。
- `CONDITIONAL`：实现或声明基本具备，但必须针对最终构建或线上环境复核。
- `BLOCKED`：仓库无法证明，或必须在 App Store Connect、生产服务、真机或运营流程中完成。

## 1. 审核门禁矩阵

| 审核主题 | 状态 | 当前证据 | 提交前动作 |
|---|---|---|---|
| 应用内隐私政策 | PASS | 注册页和「关于 RPBox」可进入 `/legal/privacy`；政策已说明数据类别、目的、公开展示、服务提供方处理、保存/删除、权限撤销和联系渠道 | 在最终归档包中逐项打开并切换中英文检查 |
| 公开隐私政策 URL | PASS | GitHub Pages 部署 `32719231844` 和 `32720969002` 均成功；2026-08-24 匿名复核 `https://totalrpbox.com/privacy.html` 与 `https://www.totalrpbox.com/privacy.html` 均为 HTTPS 200、长度 12512，并包含 2026-08-24 生效日期、中英文账号删除说明和公开联系渠道，与候选源文件一致 | 最终提交前再从移动网络打开一次；后续修改政策时保持仓库源文件、线上页面与 App Store Connect URL 一致 |
| 支持与联系 | CONDITIONAL | 2026-08-24 匿名请求 `https://github.com/1120370331/RPBox/issues` 返回 HTTP 200；「关于 RPBox」、应用内政策和公开政策均链接该地址，官网为 `https://totalrpbox.com` | 确认 Issues 可创建反馈且有人持续响应；不要让用户在公开 Issue 中提交密码或令牌 |
| 账号删除（5.1.1） | CONDITIONAL | 聚焦服务端测试证明：帖子、道具、剧情、人物卡、私有人物资料、备份和非公开 RP 数据库数据会删除；已审核通过、已发布且公开的 RP 数据库知识作品可能保留，但账号壳会匿名化且保留媒体的归属信息会清除。删除流程同时清理已删除 RPDB、人物卡和公会目标的举报，只为实际存续的公开 RPDB 知识作品保留相应举报 | 使用一次性账号在生产候选后端做真机删除，分别检查普通内容及其举报被删除、符合条件的公开知识作品及其举报按规则存续、登录失效和媒体归属清除；保留脱敏证据 |
| UGC 举报和屏蔽（1.2） | CONDITIONAL | 人物卡和公会举报分别使用一等 `character_card` / `guild` 目标及对象 ID，不再合并到用户目标；同一作者的不同对象保持独立。版主队列可解析目标标题、预览、URL 和作者。人物卡/公会的持久隐藏与屏蔽在详情和公开列表均执行；共用 `SafetyReportSheet` 的标题、原因、提示和操作已完整提供中英文 | 在最终构建走通帖子、人物卡、公会三条代表性路径；对同一作者的两个对象确认举报不合并，版主字段可用，重启后隐藏/屏蔽仍在详情及公开列表生效，并切换 zh/en 检查举报表单 |
| UGC 过滤与及时响应（1.2） | BLOCKED | 帖子、道具、人物卡和 RP 数据库公开内容已有发布前审核，评论配图进入待审状态；举报队列支持删除、禁言、封禁、驳回和归档。但仓库不能证明线上值班人员与实际响应时效 | 发布负责人须确认版主账号、举报队列、升级规则和可执行的响应时限；无人处理的举报系统不满足提交条件 |
| 登录与 Sign in with Apple（4.8） | PASS | 当前主登录仅为 RPBox 自有用户名/邮箱/密码，未发现第三方或社交登录作为主登录方式 | 不为此单独增加 Sign in with Apple；若以后增加 Google、Facebook 等主登录，重新评估 4.8 |
| 数字内容支付（3.1.1） | PASS | 当前赞助者页面仅展示名单，不含购买、捐赠链接或数字权益解锁 | 提交前再次扫描最终构建；任何捐赠、会员、去广告、虚拟物品或功能解锁入口都必须先做 IAP 合规评估 |
| 相机和照片权限 | CONDITIONAL | JS 照片路径调用 `Camera.pickImages({ limit: 1 })` 且没有 JS Photos 权限请求；但解析到的 `@capacitor/camera` 6.1.3 原生 `showPhotos()` 仍会在 PHPicker 前调用 `PHPhotoLibrary.requestAuthorization`，因此不能推断一定没有广泛照片权限提示。拍照使用 `saveToGallery: false`，未发现保存功能；三个权限说明由 `mobile/scripts/iosCompliance.mjs` 单一来源生成并由 verifier 校验 | 必须在最终 IPA 的真实 iOS 版本上首次选择照片，记录实际提示和授权后的可见范围；确认共享权限文案与真机行为准确，并在审核备注说明应用不自动保存到照片库 |
| 推送通知 | PASS | 推送插件、Capacitor 配置、Pod 和 lock 项均已移除；对候选仓库的 `PushNotifications`、`push-notifications` 和 `aps-environment` 检索无命中；当前无推送能力且不收集令牌 | 若未来启用，先补 capability/entitlement、用户触发的权限请求、令牌生命周期、政策和 App Privacy 标签，再提交新版本 |
| 广告、跟踪与 ATT | PASS | 当前仓库未发现广告 SDK 或跨应用/网站跟踪代码；政策只对当前版本作此声明 | 对最终 lockfile、原生依赖和网络请求复扫；保持无跟踪时不要显示无意义的 ATT 提示；一旦引入跟踪须重做标签和同意流程 |
| iOS 签名、配置与更新 URL | CONDITIONAL | iOS 流水线 `32836194697` 已对 1.1 build 1000042 完成测试、生产 API 预检、archive、IPA 导出、签名/entitlement/PrivacyInfo 校验和 TestFlight 上传；ASC 审计 `32836936209` 确认 processing=`VALID`。公开更新 URL 强制绑定公开 App Store 页面，生产 updater 尚未提升 | 在支持的 iPhone/iPad 安装该 build，完成冷启动、登录、人物卡、图片、隐私/安全及深链路径。只有 Apple 公共页面显示 1.1 后才显式发布生产 iOS updater 元数据 |
| 稳定性、后端与演示账号（2.1） | BLOCKED | 2026-08-24 验证当前生产地址 `https://ksxvodevhonx.sealosbja.site`：DNS/TLS 正常，`/health` 返回 200，数据库驱动的 `/api/v1/rpdb/works` 返回 200，匿名访问受保护路径返回 401。ASC 1.1 的审核联系人和演示账号字段已填写，但值被审计流程脱敏，尚未验证真实登录；`rpbox.app` 本身未注册 | 保持现有生产主机在线并从第二个独立网络复核；用 TestFlight 验证 ASC 中的主演示账号和一次性删除账号仍有效，凭据只留在 App Store Connect |
| 截图、描述与版本元数据（2.3） | CONDITIONAL | prepare dry-run `32837110119` 与 apply `32837219047` 已将 1.1 精确绑定 build 1000042，并更新限定的中文 What’s New；提交 dry-run `32840566774` 对元数据、审核备注、关键词指纹、固定顺序的 iPhone/iPad 共 8 张截图和年龄分级做了精确复核。正式提交 `32840643823` 成功，最终只读审计 `32840730773` 确认 1.1 / 1000042 为 `WAITING_FOR_REVIEW`、提交记录存在且仅含 1 项 | 等待 Apple 审核；在 App Store Connect 持续确认没有元数据问题，并用 1000042 真机确认截图描述的功能没有调试、占位或误导内容。公开 1.0 的 App Privacy 标签仍需与最终实际数据流持续保持一致 |
| 深链与 Universal Links | CONDITIONAL | iOS 工程声明 `app.rpbox.mobile` scheme，并关联 `totalrpbox.com` 与 `www.totalrpbox.com` | 在已安装最终包上验证一个自定义 scheme 和一个 Universal Link；确认线上 AASA 文件、Team ID/Bundle ID、路径范围及 HTTPS 均正确；失败时修复或删除元数据声明 |
| 年龄分级、UGC 与知识产权 | BLOCKED | ASC 1.1 年龄分级声明已将 `userGeneratedContent` 和 `messagingAndChat` 设为 true，其他内容强度仍为 NONE；条款包含 UGC 禁止行为和 Blizzard/WoW 非隶属声明，但线上内容强度及素材权利仍需人工复核 | 按实际公开内容复核幻想暴力、粗俗语言、成人暗示等频率；核查截图、图标、名称、示例人物卡及用户内容的使用权；避免暗示 Blizzard 赞助或认可 |

## 2. App Privacy 标签映射草案

此表是填写 App Store Connect 前的起点，不是已提交状态。以最终生产构建的实际网络行为和服务端日志为准；基础设施提供方代为处理的数据仍可能属于开发者收集的数据。

| 实际数据 | 建议 App Privacy 类别 | 与身份关联 | 跟踪 | 主要目的/备注 |
|---|---|---:|---:|---|
| 邮箱 | Contact Info → Email Address | 是 | 否 | App Functionality：注册、登录、账号支持 |
| 用户名、内部账号 ID | Identifiers → User ID | 是 | 否 | App Functionality：身份、同步、社区署名、审核 |
| 帖子、剧情、道具描述、RP 数据库作品、人物卡、评论 | User Content → Other User Content | 是 | 否 | App Functionality；公开发布时会按设计展示给其他用户 |
| 头像和用户选择上传的图片 | User Content → Photos or Videos | 是 | 否 | App Functionality：资料与社区内容；只有用户选择后上传 |
| 认证/限流日志中的 IP、用户名或用户 ID、请求路径/时间/状态和安全事件 | 按最终字段分别评估 Identifiers → User ID、Usage Data → Product Interaction、Diagnostics → Other Diagnostic Data 或 Other Data | 登录后事件通常可关联；匿名限流事件需按实际确认 | 否 | App Functionality（认证、安全/防欺诈和故障排查）；服务端已有这些字段的日志证据，须确认生产保存期限和访问范围 |
| 登录状态、语言/界面偏好、本地图片缓存 | 通常不因仅保存在设备上而填写为“收集” | — | 否 | 当前作为设备本地数据处理；只有某功能明确传输时才重新分类，不能从“本地存在”推导为服务端收集 |
| 推送令牌和通知偏好 | 当前版本不收集 | — | 否 | 推送依赖、配置和原生项目项已移除；仅在未来重新启用并上传令牌后加入标签 |

提交前必须再次确认：

- 不要勾选 Developer's Advertising、Third-Party Advertising 或 Tracking，除非最终构建实际使用。
- 核对生产反向代理、认证/限流日志、对象存储和错误日志实际保留的 IP、用户标识、请求路径/时间/状态与安全事件；不要凭“可能存在设备数据”勾选仓库没有证据的字段。
- 用户名、头像和公开内容“向其他用户展示”属于功能设计，应与商店描述和政策保持一致。
- 相机/照片权限本身不等于收集整个照片库；标签应反映实际上传的所选媒体。
- 数据标签、`PrivacyInfo.xcprivacy`、权限文案、隐私政策和最终二进制必须一致。

## 3. UGC、举报和屏蔽验收路径

使用两个测试账号，A 发布内容，B 执行安全操作。不要使用开发者自己的主账号做删除测试。

1. B 登录后打开「社区 → 任意非本人帖子 → 举报内容」，选择原因、填写说明，并勾选提交给版主审核。
2. 确认成功提示；使用版主账号确认举报进入审核队列，并完成一次审核处理。
3. B 在同一详情页选择「屏蔽作者」，确认作者内容从相应列表/详情隐藏。
4. 进入「个人中心 → 屏蔽用户」，确认可看到并解除屏蔽。
5. 对同一持有人的两张公开人物卡分别举报，确认生成独立的 `character_card` + 对象 ID；仅隐藏其中一张并重启，确认该卡在详情和公开列表持续隐藏，再验证屏蔽持有人及名单管理。
6. 对非本人公会验证 `guild` + 公会 ID 举报、持久隐藏和屏蔽会长；重启后确认详情、公开列表和屏蔽名单状态。
7. 在版主队列确认人物卡和公会举报均提供正确标题、预览、目标 URL 和作者，且不同对象不会合并。
8. 将应用分别切换为中文和英文，确认举报表单的标题、原因、提示、字段、按钮和状态文字无残留硬编码。
9. 对道具或 RP 数据库作品再抽查一条举报路径；剧情详情抽查「举报」入口。
10. 验证公开支持链接可作为应用内举报不可用时的后备渠道，并确认实际有人接收。
11. A 打开「个人中心 → RPBox 人物卡」，分别从空白和 TRP3 云备份创建卡片；验证查看、五类资料编辑、角色大图管理、保存为私密及发布送审，并确认草稿/私密/待审状态不会误显示为公开已审核。

拒绝提交的情况：举报只显示成功但未进入队列、屏蔽不影响内容展示、版主无法处理、支持链接不可访问、线上没有明确负责人或响应流程。

## 4. 权限和数据最小化检查

- 相机：只在用户点击拍摄/上传时请求；文案应说明用于拍摄并上传用户选择的内容。
- 照片：JS 只在用户点击后调用 `Camera.pickImages({ limit: 1 })`，没有主动调用 JS Photos 权限 API；但是 `@capacitor/camera` 6.1.3 的 iOS 原生实现会在 PHPicker 前请求照片库授权。必须以最终 IPA 真机行为为准，不能仅根据 JS 调用宣称“不出现广泛权限提示”。
- 照片写入：代码使用 `saveToGallery: false` 且未发现保存功能。三个 Usage Description key 仍因当前插件构建要求保留；审核备注应说明应用不写入照片库，并在升级插件后重新评估能否安全移除 Add 文案。
- 通知：当前依赖、配置、Pod、lock 和 entitlement 均无推送项，无运行时能力。未来启用时应在用户理解用途后请求，而不是首次启动即请求。
- ATT：当前无广告或跨应用跟踪，不应仅为“保险”请求跟踪权限。
- 未使用的相机、照片、通知或其他 capability 应从最终工程移除，减少审核疑问和攻击面。

## 5. 年龄分级、素材与商标

- App Store Connect 年龄分级问卷必须如实选择用户生成内容、评论/社区交互及最终线上内容可能出现的强度；仓库不能替代人工查看线上内容。
- 提交前清理演示账号中的违法、色情、仇恨、骚扰、未授权音乐/图片和其他明显违规内容。
- 截图只使用开发者有权展示的素材和专门准备的演示内容，不使用无法证明授权的玩家作品。
- 商店描述和截图应称 RPBox 为独立社区工具；不要使用“官方”“合作”“授权”等会暗示 Blizzard Entertainment 认可的措辞。
- World of Warcraft、魔兽世界和相关标识归各自权利人所有；应用内条款已包含非隶属声明，但仍需人工检查图标、商标、截图和元数据。

## 6. App Store Connect 字段

| 字段 | 提交值/要求 |
|---|---|
| Privacy Policy URL | `https://totalrpbox.com/privacy.html`（必须先验证线上最终内容） |
| Support URL | `https://github.com/1120370331/RPBox/issues`（必须保持公开可访问和有人响应） |
| Marketing URL | `https://totalrpbox.com` |
| Sign-in required | 是；在 Review Information 安全填写演示账号，不写入仓库 |
| Notes for Review | 使用下一节模板，替换全部方括号占位符 |
| App Privacy | 按第 2 节和最终生产数据流填写，不能照抄未复核草案 |
| Age Rating | 按最终线上 UGC、评论和内容强度如实填写 |
| In-App Purchases | 当前无；未来任何数字权益/内容购买必须在提交前重新评估 |

截图与描述检查：

- 每张截图必须来自本次提交的最终 UI，不使用桌面端截图冒充手机端。
- iPhone/iPad 尺寸、语言和状态栏内容符合 App Store Connect 当前要求。
- 不出现测试服务器地址、调试面板、占位符、测试凭据、审核中功能或虚假的数据规模。
- 功能描述与真实可用的社区、剧情、人物卡、RP 数据库、道具和账号删除路径一致。

## 7. Copy-ready Notes for Review

以下文本可复制到 App Store Connect，但必须先替换每个 `[PLACEHOLDER]`。不要把真实密码提交到 Git 或截图中。

```text
RPBox is an independent community toolbox for World of Warcraft role-players. It is not affiliated with, sponsored by, or endorsed by Blizzard Entertainment.

REVIEW ACCESS
Username: [DEMO_USERNAME]
Password: [DEMO_PASSWORD]
Backend/environment: [LIVE_REVIEW_BACKEND_DESCRIPTION]
Health endpoint verified from independent networks: [VERIFIED_HEALTH_URL_AND_DATE]

The account is active, contains representative non-sensitive demo content, and will remain available throughout review. Please enter these credentials only in App Store Connect Review Information; they are intentionally not stored in the source repository.

CORE REVIEW PATHS
1. Sign in, open Profile → RPBox Character Cards, and create a card from either a blank card or a TRP3 cloud backup. Open the card, edit its profile sections and character image, then verify private saving and public submission states.
2. Open Community to view a seeded post named “[DEMO_POST_TITLE]”. On that post, use “Report Content” to select a reason, add an explanation, and submit it to moderators. “Block Author” is available on the same detail screen.
3. A non-owned character card and guild each provide their own object-specific Report/Hide action and owner blocking. Reports for different objects remain separate, moderator review includes the object's title, preview, URL, and author, and hidden/blocked objects remain excluded from detail and public lists. The safety sheet is localized in Chinese and English.
4. Manage blocked users at Profile → Blocked Users.
5. Privacy Policy and Terms are available before sign-in and at Profile → About RPBox. Public privacy URL: https://totalrpbox.com/privacy.html
6. Support URL: https://github.com/1120370331/RPBox/issues
7. Account deletion is at Profile → Delete Account and requires the current password. It removes posts, items, stories, character cards, private profiles, backups, and non-public RP Database data. Reports for deleted RP Database, character-card, and guild targets are removed. An RP Database knowledge work may remain only if it is approved, published, and public; the deleted user's account shell is anonymized, media attribution is cleared, and reports remain only when their public RP Database target survives. To test destructive deletion, use the separate disposable account below rather than the primary demo account.

DELETION TEST ACCOUNT
Username: [DISPOSABLE_DELETION_USERNAME]
Password: [DISPOSABLE_DELETION_PASSWORD]

PERMISSIONS
Camera and photo selection are invoked only after the reviewer chooses to take or select an image for upload, and only the selected media is uploaded. The iOS photo picker may still display a system photo-library authorization prompt. RPBox uses saveToGallery=false and does not provide a save-to-library feature; the Photo Library Add usage string remains because the installed camera plugin requires all three usage keys. This build has no push-notification capability and does not collect a push token.

LOGIN / PAYMENTS
RPBox uses its own username/email/password account system and does not offer third-party or social login, so Guideline 4.8 is not applicable to this build. The sponsor list is acknowledgement only; it does not sell or unlock digital content, features, or benefits. There are no in-app purchases in this build.

DEEP LINK SAMPLE
[FINAL_TESTED_DEEP_LINK_AND_EXPECTED_DESTINATION]

If anything in the review environment is unavailable, please contact us through the Support URL above. The GitHub Issues page is public, so credentials and other sensitive information should remain in App Store Connect communication.
```

## 8. 最终人工阻塞项

在以下项目全部完成前不要点击 Submit for Review：

- [x] 当前候选 1.1 build 1000042 的 IPA 已通过类型检查、构建、原生工程验证、原生归档、签名/隐私校验和 App Store Connect `VALID` 门禁；旧 build 1000041 的证据未被复用。
- [ ] 至少一台受支持 iPhone（以及声明支持 iPad 时的一台 iPad）通过 TestFlight 冷启动、登录、上传、举报、屏蔽、删除账号和外链检查。
- [ ] 从至少两个独立网络确认最终编入 IPA 的生产 API 的 DNS、TLS、`/health`、数据库读取和实际登录路径；生产审核后端、主演示账号和一次性删除账号在审核期间持续可用，凭据只放 App Store Connect。当前候选使用已验证的 Sealos 生产地址，不得回退到未注册的 `api.rpbox.app`。
- [ ] 线上隐私政策、支持 URL、官网、AASA 和深链均从非开发者网络验证。
- [ ] 版主值班、过滤、举报队列和升级流程已实际运行，不只是存在接口。
- [ ] App Privacy、年龄分级、出口合规、内容权利、截图、本地化描述和 Notes for Review 已由发布负责人复核。
- [ ] 所有 Notes for Review 占位符已替换，且没有测试密码出现在 Git、日志或截图中。
- [ ] `IOS_PUBLIC_UPDATE_URL`（或兼容变量 `IOS_APP_STORE_URL`）已通过规范数字 App ID 校验，且 Bundle ID、Team ID、版本号和 build number 对应最终 App Store Connect 记录。
