# 手机端发版与自动更新说明

本文档说明 RPBox 手机端（Capacitor）发布流水线、服务端 updater 元数据，以及客户端自动检测更新流程。

## 1. 流水线总览

- 触发方式：推送 `mobile-v*` tag（例如 `mobile-v0.1.0`）
- 工作流文件：`.github/workflows/release-mobile.yml`
- 主要动作：
  1. 构建 Android Release APK
  2. 使用 keystore 签名 APK
  3. 生成 `latest-android.json` 元数据
  4. 上传 APK 与元数据到服务器 `releases/mobile`
  5. 生成并上传 `latest-ios.json`（指向 App Store）

## 2. 发布命令

```bash
git tag mobile-v0.1.0
git push origin mobile-v0.1.0
```

可选：在发版前新增更新说明文件 `mobile/release-notes/<version>.txt`，例如：

```text
mobile/release-notes/0.1.0.txt
```

## 3. 服务器目录结构

移动端发布目录（默认）：

```text
server/releases/mobile/
├── latest-android.json
├── latest-ios.json
└── 0.1.0/
    ├── RPBox_0.1.0_android.apk
    ├── latest-android.json
    └── latest-ios.json
```

## 4. Metadata 格式

Android (`latest-android.json`)：

```json
{
  "latest_version": "0.1.0",
  "notes": "更新说明",
  "pub_date": "2026-03-22T12:00:00Z",
  "url": "https://api.rpbox.app/releases/mobile/0.1.0/RPBox_0.1.0_android.apk",
  "mandatory": false
}
```

iOS (`latest-ios.json`)：

```json
{
  "latest_version": "0.1.0",
  "notes": "更新说明",
  "pub_date": "2026-03-22T12:00:00Z",
  "url": "https://apps.apple.com/app/rpbox/id1234567890",
  "mandatory": false
}
```

## 5. GitHub Secrets

基础部署（与桌面端共用）：

- `SSH_PRIVATE_KEY`
- `SERVER_HOST`
- `SERVER_USER`
- `RELEASE_PATH`

移动端新增：

- `MOBILE_RELEASE_PATH`（可选，不配则用 `${RELEASE_PATH}/mobile`）
- `MOBILE_RELEASE_BASE_URL`（可选，不配则默认 `https://api.rpbox.app/releases/mobile`）
- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`
- `IOS_APP_STORE_URL`（用于生成 iOS updater 元数据）

说明：
- 如果 Android 签名 Secrets 缺失，workflow 会自动回退构建 `debug APK` 并上传，保证下载链路可用（仅用于内测分发）。

## 6. 服务端 updater 行为

统一入口仍为：

```text
GET /api/v1/updater/:target/:arch/:current_version
```

新增支持：

- `target=android`
- `target=ios`

查询优先级：

1. `releases/mobile/latest-<target>.json`
2. `config.yaml` 的 `updater.mobile.<target>`

无更新时返回 `204 No Content`，有更新时返回 `200` + JSON。

## 7. 客户端自动更新管线

手机端代码位置：

- `mobile/src/api/updater.ts`
- `mobile/src/composables/useMobileUpdater.ts`
- `mobile/src/App.vue`
- `mobile/src/views/profile/About.vue`

流程：

1. App 启动时静默检查（6 小时节流）
2. 命中更新后 toast 提示用户前往「关于 RPBox」
3. 关于页可手动「检查更新」
4. 有新版本时点击「立即更新」
   - Android：打开 APK 下载链接
   - iOS：跳转 App Store 链接
