# RPBox 手机端开发计划

> 最低成本、最高复用的移动端方案

## 1. 技术选型：Capacitor + Vue 3

### 为什么选 Capacitor

项目现有代码有三个关键特征：

- **API 层零 Tauri 依赖** — 16 个 API 模块全部是纯 HTTP 请求，`request.ts` 基于标准 fetch，可直接复用
- **Pinia stores 零 Tauri 依赖** — 6 个 store 全部是纯逻辑 + localStorage，可直接复用
- **Tauri 专属功能集中且有限** — 只有 `useUpdater`、文件对话框、shell 命令（读取 WoW SavedVariables）三处

Capacitor 包裹现有 Vue 3 代码，复用率可达 **85%+**。

### 方案对比

| 方案 | 代码复用率 | 额外学习成本 | 上架能力 |
|------|-----------|-------------|---------|
| **Capacitor + Vue 3** | ~85% | 极低 | iOS / Android |
| PWA | ~90% | 无 | 无应用商店 |
| React Native | ~5%（仅类型） | 高 | iOS / Android |
| Flutter | 0% | 高 | iOS / Android |

PWA 复用更高但没有应用商店入口、没有推送通知、没有原生体验。Capacitor 是最佳平衡点。

---

## 2. 功能取舍

WoW 只在 PC 上运行，手机端定位是 **"随时随地看和社交"**，不是 **"同步数据"**。

| 功能模块 | 手机端 | 说明 |
|---------|--------|------|
| 社区帖子 | ✅ 完整保留 | 核心社交场景，手机天然适合 |
| 公会系统 | ✅ 完整保留 | 管理、浏览、互动 |
| 剧情归档 | ⚠️ 只读浏览 + 回放 | 导入需要 WoW 文件，手机做不了 |
| 道具市场 | ⚠️ 浏览 + 收藏 | 导入游戏需要 PC |
| 通知中心 | ✅ 完整保留 + 推送 | 手机推送是杀手级功能 |
| 用户资料 | ✅ 完整保留 | 编辑头像、个人信息 |
| 人物卡同步 | ❌ 不做 | 依赖本地 WoW SavedVariables |
| 插件管理 | ❌ 不做 | 依赖本地 WoW 安装目录 |
| 客户端更新 | ❌ 替换为应用商店 | Capacitor 走商店更新 |

---

## 3. 项目结构

```
RPBox/
├── client/                    # 现有桌面端（不动）
├── mobile/                    # 新增手机端
│   ├── src/
│   │   ├── api/              → 软链接或复制 client/src/api/
│   │   ├── stores/           → 复用 client/src/stores/
│   │   ├── composables/      → 复用 useToast, useDialog
│   │   ├── i18n/             → 复用 client/src/i18n/
│   │   ├── utils/            → 复用 client/src/utils/
│   │   ├── components/       # 移动端适配组件
│   │   │   ├── MobileLayout.vue    # 底部 Tab 导航
│   │   │   ├── R*.vue              # 复用 + 适配触控
│   │   │   └── PullRefresh.vue     # 下拉刷新
│   │   ├── views/            # 移动端页面
│   │   │   ├── community/    # 复用改造
│   │   │   ├── guild/        # 复用改造
│   │   │   ├── story/        # 只读版本
│   │   │   ├── market/       # 浏览版本
│   │   │   ├── notifications/
│   │   │   ├── user/
│   │   │   └── auth/
│   │   ├── router.ts         # 移动端路由（精简版）
│   │   ├── App.vue
│   │   └── main.ts
│   ├── ios/                   # Capacitor iOS 工程
│   ├── android/               # Capacitor Android 工程
│   ├── capacitor.config.ts
│   ├── package.json
│   └── vite.config.ts
└── shared/                    # 提取共享代码
    ├── api/                   # 从 client/src/api/ 提取
    ├── stores/                # 从 client/src/stores/ 提取
    ├── utils/                 # 从 client/src/utils/ 提取
    ├── i18n/                  # 从 client/src/i18n/ 提取
    └── types/                 # 共享类型定义
```

### 共享代码策略

推荐使用 monorepo（pnpm workspace）管理，将可复用代码提取到 `shared/` 包：

```yaml
# pnpm-workspace.yaml
packages:
  - 'client'
  - 'mobile'
  - 'shared'
```

`shared` 作为内部依赖被 `client` 和 `mobile` 同时引用，避免代码重复。

---

## 4. 分阶段实施

### 第一阶段：基础框架 + 认证

**目标**：跑通 Capacitor 工程，完成登录注册。

1. 初始化 Capacitor 项目，配置 Vite + Vue 3 + TypeScript
2. 将 `client/src/api/request.ts` 和 `client/src/api/auth.ts` 提取到 shared
3. 将 `client/src/stores/user.ts` 提取到 shared
4. 实现移动端布局：底部 Tab 导航（社区 / 公会 / 通知 / 我的）
5. 适配 Login.vue、Register.vue、ForgotPassword.vue（表单组件触控优化）
6. 配置 Capacitor HTTP 插件处理 CORS（原生层发请求无跨域问题）

**需要改造的组件**：

| 组件 | 改造内容 |
|------|---------|
| `RInput.vue` | 增大触控区域，min-height 44px |
| `RButton.vue` | 增大触控区域，min-height 44px |
| `RToast.vue` | 适配安全区域（刘海屏） |
| `RDialog.vue` | 全屏弹窗模式 |

**新增组件**：

| 组件 | 用途 |
|------|------|
| `MobileLayout.vue` | 底部 TabBar + 顶部 NavBar |
| `PullRefresh.vue` | 下拉刷新容器 |

### 第二阶段：社区 + 通知

**目标**：核心社交功能上线。

1. 复用 `api/post.ts`、`api/notification.ts`、`stores/notification.ts`
2. 改造 `CommunityMain.vue` → 单列卡片流（移除侧边栏布局）
3. 改造 `PostDetail.vue` → 全屏阅读模式
4. 改造 `PostCreate.vue` → 简化编辑器（降级为 textarea + markdown 或简化 Tiptap 工具栏）
5. 复用 `Notifications.vue` → 列表适配
6. 接入 Capacitor Push Notifications 插件（Firebase FCM + APNs）
7. WebSocket 通知保持复用 `services/websocket.ts`

**富文本编辑器策略**：

Tiptap 在移动端体验一般，建议分场景处理：

| 场景 | 方案 |
|------|------|
| 浏览/阅读 | 保留 Tiptap 渲染（只读模式体验 OK） |
| 发帖/编辑 | 降级为 Markdown 输入 + 实时预览，或精简 Tiptap 工具栏（仅保留加粗、图片、链接） |

### 第三阶段：公会 + 用户

**目标**：公会体系和个人中心上线。

1. 复用 `api/guild.ts`、`api/user.ts`、`api/collection.ts`
2. 改造公会列表 → 卡片流布局
3. 改造公会详情 → 顶部 Banner + Tab 切换（信息 / 成员 / 帖子 / 剧情）
4. 改造成员管理 → 列表 + 滑动操作（踢出、改角色）
5. 改造用户资料页、设置页
6. 头像上传接入 Capacitor Camera 插件（拍照 / 相册选择）
7. 收藏夹功能复用 `api/collection.ts`

### 第四阶段：剧情 + 市场（只读）

**目标**：内容消费功能补全。

1. 复用 `api/story.ts`、`api/item.ts`
2. 剧情列表 + 详情页（只读浏览）
3. `StoryPlayback.vue` 复用（本身是独立页面，适配性好）
4. 道具市场浏览 + 收藏
5. 移除所有"导入""同步""上传道具"入口

---

## 5. 核心适配要点

### 5.1 导航模式

桌面端使用侧边栏 `AppLayout.vue`，移动端改为底部 Tab + 顶部导航栏：

```
┌─────────────────────────┐
│  ← 标题            ···  │  ← NavBar（返回、标题、操作）
├─────────────────────────┤
│                         │
│       页面内容           │
│                         │
│                         │
├─────────────────────────┤
│  社区  │ 公会 │ 通知 │ 我的 │  ← TabBar
└─────────────────────────┘
```

- 一级页面（Tab 页）：显示 TabBar，隐藏返回按钮
- 二级页面（详情页）：隐藏 TabBar，显示返回按钮
- 使用 `safe-area-inset-bottom` 适配全面屏

### 5.2 列表布局

桌面端多列网格 → 移动端单列卡片流。复用 `RCard.vue` 但通过 CSS 媒体查询调整：

```css
/* 移动端覆盖 */
.card-grid {
  grid-template-columns: 1fr;  /* 桌面端可能是 repeat(3, 1fr) */
  gap: 12px;
  padding: 12px;
}
```

### 5.3 图片加载

复用 `utils/imageCache.ts` 的 `getImageUrl`，但传更小的 `w` 参数节省流量：

| 场景 | 桌面端 | 移动端 |
|------|--------|--------|
| 列表缩略图 | `w=400` | `w=200` |
| 详情大图 | `w=1200` | `w=750` |
| 头像 | `w=200` | `w=100` |

### 5.4 主题系统

`stores/theme.ts` 和 80+ CSS 自定义属性直接复用，零改造。移动端自动继承桌面端的主题切换能力。

### 5.5 触控适配

所有可交互元素遵循 Apple HIG 最小触控区域标准：

```css
/* 全局触控区域基准 */
.touchable {
  min-height: 44px;
  min-width: 44px;
}

/* 列表项增加间距 */
.list-item {
  padding: 12px 16px;
}

/* 按钮间距防误触 */
.action-group .r-button + .r-button {
  margin-left: 12px;
}
```

---

## 6. 移动端路由

精简版路由，砍掉 PC 专属页面：

```typescript
const routes = [
  // 底部 Tab 页
  { path: '/', component: MobileLayout, children: [
    { path: '', component: CommunityFeed },
    { path: 'guilds', component: GuildList },
    { path: 'notifications', component: Notifications },
    { path: 'me', component: UserCenter },
  ]},

  // 社区
  { path: '/post/:id', component: PostDetail },
  { path: '/post/create', component: PostCreate },
  { path: '/post/:id/edit', component: PostEdit },

  // 公会
  { path: '/guild/:id', component: GuildDetail },
  { path: '/guild/:id/manage', component: GuildManage },
  { path: '/guild/:id/posts', component: GuildPosts },
  { path: '/guild/:id/stories', component: GuildStories },
  { path: '/guild/create', component: GuildCreate },

  // 剧情（只读）
  { path: '/archives', component: ArchivesList },
  { path: '/story/:id', component: StoryDetail },
  { path: '/story/:code/playback', component: StoryPlayback },

  // 市场（只读）
  { path: '/market', component: MarketList },
  { path: '/market/:id', component: ItemDetail },

  // 用户
  { path: '/user/:id', component: UserProfile },
  { path: '/settings', component: Settings },
  { path: '/collection/:id', component: CollectionDetail },

  // 认证
  { path: '/login', component: Login },
  { path: '/register', component: Register },
  { path: '/forgot-password', component: ForgotPassword },
]
```

对比桌面端 30+ 路由，移动端精简到约 20 个，移除了：

- `/sync/*` — 人物卡同步（PC 专属）
- `/market/upload`、`/market/:id/edit` — 道具上传编辑（PC 专属）
- `/guide` — 桌面端引导
- `/moderator` — 版主后台（建议保留桌面端操作）

---

## 7. 服务端改动

现有 API 需要三块新增：推送通知、移动端自动更新、展示页适配。

### 7.1 新增接口

```
POST /api/v1/user/device-token    # 注册/更新移动端推送 token
DELETE /api/v1/user/device-token   # 注销推送 token（退出登录时）
```

请求体：

```json
{
  "token": "fcm_or_apns_token_string",
  "platform": "ios" | "android"
}
```

### 7.2 推送通知网关

在现有 WebSocket Hub 广播逻辑中增加分支：当目标用户无活跃 WebSocket 连接时，通过 FCM/APNs 发送推送。

```go
// 伪代码：在 Hub.broadcast 中增加
func (h *Hub) broadcast(userID uint, msg Message) {
    if client, ok := h.clients[userID]; ok {
        // 用户在线，走 WebSocket
        client.send <- msg
    } else {
        // 用户离线，走推送
        pushService.Send(userID, msg)
    }
}
```

### 7.3 数据模型

新增 `device_tokens` 表：

```sql
CREATE TABLE device_tokens (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    token       VARCHAR(512) NOT NULL,
    platform    VARCHAR(10) NOT NULL,  -- 'ios' / 'android'
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, platform)
);
```

### 7.4 移动端自动更新接口

现有桌面端 updater 端点 `/api/v1/updater/:target/:arch/:current_version` 只处理 `windows`/`darwin`/`linux`。需要扩展支持 `android` 和 `ios` 两个 target。

**扩展现有端点**：

```
GET /api/v1/updater/android/arm64/0.1.0
GET /api/v1/updater/ios/arm64/0.1.0
```

**Android 响应**（有新版本时）：

```json
{
  "version": "0.2.0",
  "notes": "更新说明",
  "pub_date": "2026-02-27T12:00:00Z",
  "url": "https://api.rpbox.app/releases/mobile/0.2.0/RPBox_0.2.0.apk",
  "mandatory": false
}
```

**iOS 响应**（有新版本时）：

```json
{
  "version": "0.2.0",
  "notes": "更新说明",
  "pub_date": "2026-02-27T12:00:00Z",
  "url": "https://apps.apple.com/app/rpbox/id123456789",
  "mandatory": false
}
```

**无新版本**：返回 `204 No Content`（与桌面端一致）。

**平台差异**：

| 平台 | 更新方式 | 说明 |
|------|---------|------|
| Android | 应用内下载 APK + 安装 | 调用系统安装器，用户确认后安装 |
| iOS | 跳转 App Store | iOS 不允许应用内安装，只能跳商店 |

**客户端更新流程**：

```
App 启动
  │
  ├─ 调用 GET /api/v1/updater/{platform}/{arch}/{current_version}
  │
  ├─ 204 → 已是最新，静默结束
  │
  └─ 200 → 有新版本
       │
       ├─ 弹出更新弹窗（版本号 + 更新说明）
       │
       ├─ Android：下载 APK → 调用系统安装器
       │
       └─ iOS：打开 App Store 页面
```

**服务端 `config.yaml` 扩展**：

```yaml
updater:
  # 桌面端（已有）
  latest_version: "0.2.10"
  base_url: "https://api.rpbox.app/releases"
  release_notes: "桌面端更新说明"
  pub_date: "2026-02-27T12:00:00Z"

  # 移动端（新增）
  mobile:
    android:
      latest_version: "0.1.0"
      url: "https://api.rpbox.app/releases/mobile/0.1.0/RPBox_0.1.0.apk"
      release_notes: "首个移动端版本"
      pub_date: "2026-03-01T12:00:00Z"
    ios:
      latest_version: "0.1.0"
      url: "https://apps.apple.com/app/rpbox/id123456789"
      release_notes: "首个移动端版本"
      pub_date: "2026-03-01T12:00:00Z"
```

### 7.5 修改静态展示页提供移动端下载

现有 `shared/download.html` 通过 UA 检测平台并动态拉取桌面端下载链接。需要扩展支持移动端。

**改造要点**：

1. **UA 检测扩展** — 识别 Android / iOS 移动浏览器
2. **移动端展示** — 手机访问时优先展示移动端下载按钮，桌面端链接折叠到"其他平台"
3. **下载源** — Android 提供 APK 直接下载 + 应用商店链接，iOS 跳转 App Store

**页面布局变化**：

```
┌─ 手机浏览器访问 ─────────────────┐
│                                  │
│         RPBox Logo               │
│    "随时随地，RP不停歇"            │
│                                  │
│  ┌──────────────────────────┐    │
│  │  📱 下载 Android 版本     │    │  ← 主按钮
│  └──────────────────────────┘    │
│                                  │
│  ┌──────────────────────────┐    │
│  │  🍎 App Store 下载        │    │  ← 主按钮
│  └──────────────────────────┘    │
│                                  │
│     也有桌面版 →                  │  ← 折叠链接
│                                  │
└──────────────────────────────────┘
```

**实现方式**：

在现有 `download.html` 的 JS 中扩展平台检测逻辑：

```javascript
// 现有：只检测桌面平台
// 扩展：增加移动端检测
function detectPlatform() {
  const ua = navigator.userAgent
  if (/Android/i.test(ua)) return 'android'
  if (/iPhone|iPad|iPod/i.test(ua)) return 'ios'
  if (/Win/i.test(ua)) return 'windows'
  if (/Mac/i.test(ua)) return 'darwin'
  if (/Linux/i.test(ua)) return 'linux'
  return 'unknown'
}
```

**数据源**：复用扩展后的 updater 接口，手机端访问时请求 `/api/v1/updater/android/arm64/0.0.0` 获取最新版本和下载链接。

**CI 联动**：移动端发版时（推送 `mobile-v*` tag），CI 自动更新 `download.html` 中的 `fallbackMobileVersion`，与现有桌面端 `fallbackVersion` 更新逻辑一致。

---

## 8. 依赖清单

### 移动端 `mobile/package.json`

```json
{
  "dependencies": {
    "@capacitor/core": "^6.0.0",
    "@capacitor/ios": "^6.0.0",
    "@capacitor/android": "^6.0.0",
    "@capacitor/push-notifications": "^6.0.0",
    "@capacitor/camera": "^6.0.0",
    "@capacitor/haptics": "^6.0.0",
    "@capacitor/status-bar": "^6.0.0",
    "@capacitor/keyboard": "^6.0.0",
    "vue": "^3.4.0",
    "vue-router": "^4.2.5",
    "pinia": "^2.1.7",
    "vue-i18n": "^9.14.0",
    "remixicon": "^4.8.0"
  },
  "devDependencies": {
    "@capacitor/cli": "^6.0.0",
    "vite": "^5.0.0",
    "@vitejs/plugin-vue": "^5.0.0",
    "typescript": "^5.3.0"
  }
}
```

**不需要的桌面端依赖**：

| 依赖 | 原因 |
|------|------|
| `@tauri-apps/*` | Tauri 专属 |
| `tiptap` 全套 | 编辑降级为 Markdown（阅读可用轻量渲染） |
| `echarts` | 移动端不做数据看板 |
| `vue3-emoji-picker` | 可选，后续按需加入 |

---

## 9. Capacitor 配置

```typescript
// mobile/capacitor.config.ts
import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.rpbox.mobile',
  appName: 'RPBox',
  webDir: 'dist',
  server: {
    // 生产环境不需要，开发时指向 Vite dev server
    // url: 'http://localhost:3102',
    androidScheme: 'https',
  },
  plugins: {
    PushNotifications: {
      presentationOptions: ['badge', 'sound', 'alert'],
    },
    Keyboard: {
      resize: 'body',
      resizeOnFullScreen: true,
    },
    StatusBar: {
      style: 'dark',
    },
  },
}

export default config
```

---

## 10. 复用统计

### 按层级统计

| 层级 | 总文件数 | 直接复用 | 需适配 | 需重写 | 不复用 |
|------|---------|---------|--------|--------|--------|
| API 模块 | 16 | 14 | 0 | 0 | 2 |
| Pinia Stores | 6 | 5 | 0 | 0 | 1 |
| Utils | 9 | 8 | 0 | 0 | 1 |
| Composables | 3 | 2 | 0 | 0 | 1 |
| i18n | 全部 | 全部 | 0 | 0 | 0 |
| 组件 (47) | 47 | ~20 | ~15 | ~5 | ~7 |
| 页面 (30+) | 30+ | ~5 | ~15 | ~5 | ~5 |

### 不复用的模块明细

| 模块 | 原因 |
|------|------|
| `api/addon.ts` | 插件管理，PC 专属 |
| `api/accountBackup.ts` | 账号备份，依赖本地文件 |
| `composables/useUpdater.ts` | Tauri 自动更新专属 |
| `services/syncService.ts` | WoW SavedVariables 文件操作 |
| `stores/emote.ts` | 表情包可后续按需加入 |
| `views/sync/*` (4 个页面) | 人物卡同步，PC 专属 |
| `components/AddonInstaller.vue` | 插件安装，PC 专属 |
| `components/AddonUpdateDialog.vue` | 插件更新，PC 专属 |
| `components/UpdateNotification.vue` | Tauri 更新提示，PC 专属 |
| `components/ConflictDialog.vue` | 同步冲突解决，PC 专属 |

---

## 11. 开发命令

```bash
# 初始化项目
cd mobile
npm install
npx cap init RPBox com.rpbox.mobile

# 开发调试
npm run dev                    # Vite dev server
npx cap open android           # 打开 Android Studio
npx cap open ios               # 打开 Xcode

# 同步 Web 资源到原生工程
npm run build && npx cap sync

# 实时调试（推荐）
npx cap run android --livereload --external
npx cap run ios --livereload --external
```

---

## 12. CI/CD 集成

在现有 GitHub Actions 基础上新增移动端构建流程：

### 触发条件

| 流程 | 触发条件 | 配置文件 |
|------|----------|----------|
| Android 构建 | 推送 `mobile-v*` tag | `.github/workflows/release-mobile-android.yml` |
| iOS 构建 | 推送 `mobile-v*` tag | `.github/workflows/release-mobile-ios.yml` |

### 新增 GitHub Secrets

| Secret | 用途 |
|--------|------|
| `ANDROID_KEYSTORE_BASE64` | Android 签名密钥（base64 编码） |
| `ANDROID_KEYSTORE_PASSWORD` | 密钥库密码 |
| `ANDROID_KEY_ALIAS` | 密钥别名 |
| `ANDROID_KEY_PASSWORD` | 密钥密码 |
| `IOS_CERTIFICATE_BASE64` | iOS 分发证书 |
| `IOS_CERTIFICATE_PASSWORD` | 证书密码 |
| `IOS_PROVISION_PROFILE_BASE64` | 描述文件 |

---

## 13. 风险与注意事项

### 技术风险

| 风险 | 影响 | 应对 |
|------|------|------|
| Tiptap 移动端性能差 | 编辑体验卡顿 | 编辑降级为 Markdown，阅读保留渲染 |
| iOS WebView 键盘遮挡输入框 | 表单体验差 | 使用 `@capacitor/keyboard` 插件 + `resize: body` |
| 推送通知需要 Apple 开发者账号 | iOS 推送无法测试 | 优先开发 Android 版本，iOS 后续跟进 |
| 图片上传在弱网下失败 | 用户体验差 | 压缩图片 + 断点续传（后续优化） |

### 开发顺序建议

优先 Android — 不需要开发者账号即可侧载测试，开发迭代快。iOS 在功能稳定后再接入，主要增量工作是证书配置和 App Store 审核。

---

## 14. 总结

| 维度 | 数据 |
|------|------|
| 技术方案 | Capacitor 6 + Vue 3 + TypeScript |
| 代码复用率 | ~85%（API / Store / Utils / i18n / 主题 全量复用） |
| 新增代码量 | 移动端布局 + 页面适配 + 推送集成 |
| 服务端改动 | 推送网关 + 移动端 updater 接口扩展 + 设备 token 注册 |
| 基础设施 | 展示页移动端适配 + 应用内自动更新 + CI/CD 移动端构建 |
| 功能覆盖 | 社区、公会、通知、剧情浏览、市场浏览、用户中心 |
| 砍掉功能 | 人物卡同步、插件管理（PC 专属，手机无意义） |

核心思路：**不重写，只适配**。把桌面端的侧边栏换成底部 Tab，把多列网格换成单列卡片，把 Tauri 原生能力换成 Capacitor 插件，其余全部复用。
