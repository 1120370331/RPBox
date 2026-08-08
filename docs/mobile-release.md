# 手机端发版与自动更新说明

本文档说明 RPBox 手机端（Capacitor）发布流水线、iOS TestFlight 流程、服务端 updater 元数据，以及客户端自动检测更新流程。

## 1. 流水线总览

- Android 触发方式：推送 `mobile-v*` tag（例如 `mobile-v2.0.2`）
- 工作流文件：`.github/workflows/release-mobile.yml`
- 主要动作：
  1. 构建 Android Release APK
  2. 使用 keystore 签名 APK
  3. 生成 `latest-android.json` 元数据
  4. 上传 APK 与元数据到服务器 `releases/mobile`
- iOS 触发方式：推送 `mobile-ios-v*` tag（例如 `mobile-ios-v2.0.2`）
- iOS 工作流文件：`.github/workflows/release-ios-testflight.yml`
- iOS 主要动作：
  1. 生成并校验 Capacitor iOS 工程
  2. 使用 Distribution 证书和 App Store provisioning profile 创建 archive
  3. 导出 IPA 并上传到 TestFlight
  4. 生成并上传 `latest-ios.json`（指向 App Store）

## 2. 发布命令

```bash
git tag mobile-v2.0.2
git push origin mobile-v2.0.2

# iOS TestFlight
git tag mobile-ios-v2.0.2
git push origin mobile-ios-v2.0.2
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
- `ANDROID_SIGNING_KEYSTORE_BASE64`
- `ANDROID_SIGNING_STORE_PASSWORD`
- `ANDROID_SIGNING_KEY_ALIAS`
- `ANDROID_SIGNING_KEY_PASSWORD`
- `IOS_APP_STORE_URL`（用于生成 iOS updater 元数据）
- `IOS_TEAM_ID`
- `IOS_BUNDLE_ID`
- `IOS_PROVISION_PROFILE_NAME`
- `IOS_P12_BASE64`
- `IOS_P12_PASSWORD`
- `IOS_PROVISION_PROFILE_BASE64`
- `IOS_KEYCHAIN_PASSWORD`（可选）
- `ASC_ISSUER_ID`
- `ASC_KEY_ID`
- `ASC_API_KEY_BASE64`
- `IOS_TESTFLIGHT_GROUPS`（可选）

说明：
- Android workflow 构建并上传签名的 Release APK。
- iOS workflow 会在 macOS runner 上完成 Xcode archive、IPA 导出和 TestFlight 上传；Windows 本机不能替代这一步。
- iOS workflow 在归档前会校验 Bundle ID、版本号、build number、签名方式、provisioning profile、相机权限、深链和隐私清单。

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
