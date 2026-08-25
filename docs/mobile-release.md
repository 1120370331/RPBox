# 手机端发版与自动更新说明

本文档说明 RPBox 手机端（Capacitor）发布流水线、iOS TestFlight 流程、服务端 updater 元数据，以及客户端自动检测更新流程。

Apple App Store 提交不是流水线成功后的自动步骤。每次 iOS 提交前必须完成 [Apple App Store 审核清单](./app-store-review.md)，关闭其中全部 `BLOCKED` 项，并在 App Store Connect 安全填写审核账号；不要把真实审核凭据写入仓库。

## 1. 流水线总览

- Android 触发方式：推送 `mobile-v*` tag（例如 `mobile-v2.0.3`）
- 工作流文件：`.github/workflows/release-mobile.yml`
- 主要动作：
  1. 构建 Android Release APK 与 Google Play AAB
  2. 使用同一 upload keystore 签名并验证 APK/AAB
  3. 生成 `latest-android.json` 元数据
  4. 将 APK/AAB 作为 Actions artifact 保存，并只把 APK 与元数据部署到服务器 `releases/mobile`
- iOS 触发方式：新版本可推送 `ios-v*` 或兼容的 `mobile-ios-v*` tag；同一商店版本递增 build 时使用手动 workflow dispatch。版本必须与 `mobile/ios/release.json` 一致（当前为 `1.1 / 1000042`）
- iOS 工作流文件：`.github/workflows/release-ios-testflight.yml`
- iOS 主要动作：
  1. 生成并校验 Capacitor iOS 工程
  2. 使用 Distribution 证书和 App Store provisioning profile 创建 archive
  3. 导出 IPA、上传到 TestFlight，并等待 App Store Connect 完成 build processing
  4. TestFlight 阶段不更新生产 `latest-ios.json`；只有 Apple 公共页面已经显示同版本 Ready for Sale 时，才可显式执行受保护的公开元数据发布

## 2. 发布命令

```bash
git tag mobile-v2.0.3
git push origin mobile-v2.0.3

# iOS TestFlight：1.1 tag 已用于旧 build，不移动或复用旧 tag
gh workflow run release-ios-testflight.yml --ref main -f version=1.1
```

可选：在发版前新增更新说明文件 `mobile/release-notes/<version>.txt`，例如：

```text
mobile/release-notes/0.1.0.txt
```

iOS 版本和默认 build number 由 `mobile/ios/release.json` 冻结；当前待构建、上传和验证的候选为 `1.1 / 1000042`，`1.1 / 1000041` 只作为不含本次 RPBox 人物卡功能的旧候选保留。Android 当前为 `2.0.3 / 2000003`。两端共享功能源码，但不强制共享商店版本序列；不得在新候选通过 TestFlight 和人工门禁前提交审核或提升生产 iOS updater。

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
  "url": "https://ksxvodevhonx.sealosbja.site/releases/mobile/0.1.0/RPBox_0.1.0_android.apk",
  "mandatory": false
}
```

iOS (`latest-ios.json`)：

```json
{
  "latest_version": "0.1.0",
  "notes": "更新说明",
  "pub_date": "2026-03-22T12:00:00Z",
  "url": "https://apps.apple.com/cn/app/rpbox/id6761112311",
  "mandatory": false
}
```

公开更新 URL 采用 fail-closed 校验：只接受 `https://apps.apple.com` 的规范 App 路径和 9–12 位数字 App ID，拒绝 TestFlight、凭据、占位 ID、非 HTTPS、其他主机或非规范路径。workflow 按 `IOS_PUBLIC_UPDATE_URL`、兼容变量 `IOS_APP_STORE_URL`、内置正式 URL 的顺序取值，任一候选必须通过 `mobile/scripts/iosCompliance.mjs` 后才生成元数据；仍须在真机确认打开正确商店页面。

## 5. GitHub Secrets

基础部署（与桌面端共用）：

- `SSH_PRIVATE_KEY`
- `SERVER_HOST`
- `SERVER_USER`
- `RELEASE_PATH`

移动端新增：

- `MOBILE_RELEASE_PATH`（可选，不配则用 `${RELEASE_PATH}/mobile`）
- `MOBILE_RELEASE_BASE_URL`（可选，不配则默认当前生产地址 `https://ksxvodevhonx.sealosbja.site/releases/mobile`；迁移自有域名前必须先完成入口绑定、DNS 和 TLS）
- `ANDROID_SIGNING_KEYSTORE_BASE64`
- `ANDROID_SIGNING_STORE_PASSWORD`
- `ANDROID_SIGNING_KEY_ALIAS`
- `ANDROID_SIGNING_KEY_PASSWORD`
- `IOS_PUBLIC_UPDATE_URL`（可选，iOS updater 的首选公开 App Store URL；必须通过规范 URL 校验）
- `IOS_APP_STORE_URL`（可选，兼容旧配置；同样必须通过规范 URL 校验）
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
- Android workflow 构建并验证签名的 Release APK 与 AAB；APK 用于官网分发，AAB 作为 Google Play 待上传产物保留。
- iOS workflow 会在 macOS runner 上完成 Xcode archive、IPA 导出、TestFlight 上传和 processing 等待；Windows 本机不能替代这一步。
- iOS 相机/照片权限说明以 `mobile/scripts/iosCompliance.mjs` 为单一来源，由 `prepareNativeShare.mjs` 写入并由 `verifyIosProject.mjs` 逐项校验。
- iOS workflow 在归档前会校验 Bundle ID、版本号、build number、签名方式、provisioning profile、相机/照片权限、深链、隐私清单和公开更新 URL。
- iOS TestFlight 成功不等于 App Store 已发布，禁止在审核通过前提升生产 iOS updater 版本。

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
