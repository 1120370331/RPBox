# RPBox iOS Mobile Parity Design

**Date:** 2026-07-10

**Scope:** Synchronize the current mobile 1.0.41 behavior and native configuration with the iOS/TestFlight build path without publishing a release.

## Goals

1. Ensure all shared mobile features currently shipped on Android are included in the generated iOS application.
2. Preserve the platform-specific update behavior: Android installs downloaded APKs, while iOS opens the App Store/TestFlight destination.
3. Make iOS native project generation deterministic and safe to repeat from a clean GitHub Actions runner.
4. Fail the iOS workflow early when required permissions, deep-link settings, entitlements, privacy declarations, version values, or bundle settings are missing.
5. Add automated coverage for the iOS-specific branches that can run on Windows and Linux before Xcode compilation.

## Current State

The Android and iOS applications already share the Vue 3 source under `mobile/src`. The current 1.0.41 editor toolbar fix, update checks, account flows, content pages, safety features, and other mobile behavior therefore do not require a second iOS implementation.

Native projects are intentionally generated during CI:

- `.github/workflows/release-mobile.yml` creates and patches an Android project;
- `.github/workflows/release-ios-testflight.yml` creates and patches an iOS project;
- `mobile/scripts/prepareNativeShare.mjs` applies RPBox-specific native configuration after Capacitor generation.

The main parity risk is not duplicated application code. It is incomplete or silently ineffective iOS project patching. The current workflow verifies camera configuration, but it does not comprehensively verify URL schemes, associated domains, entitlements linkage, privacy manifest placement, bundle identity, or native version synchronization.

## Architecture

### Shared Mobile Application

`mobile/src` remains the single implementation for both platforms. No Android page or component is copied into an iOS-specific source tree.

Platform branching remains limited to capabilities that genuinely differ:

- Android native builds may use the `RPBoxUpdater` plugin to download and install an APK.
- iOS builds use the existing updater metadata and open the App Store URL through the `itms-apps` scheme.
- Web or unsupported environments continue to use external update links.

The current `MobileUpdateMode` contract remains:

```ts
type MobileUpdateMode = 'android-in-app' | 'ios-store' | 'external'
```

The iOS build must never register or call the Android updater plugin.

### Native Project Preparation

Keep `mobile/scripts/prepareNativeShare.mjs` as the entry point used by both release workflows, but separate its platform-independent values and platform-specific mutations clearly enough to test them.

The iOS preparation path must deterministically ensure:

- `CFBundleURLTypes` contains the `app.rpbox.mobile` custom scheme;
- camera, photo-library read, and photo-library write usage descriptions exist;
- `App.entitlements` contains `applinks:totalrpbox.com` and `applinks:www.totalrpbox.com`;
- every build configuration links `CODE_SIGN_ENTITLEMENTS` to `App/App.entitlements`;
- the privacy manifest exists at `mobile/ios/App/App/PrivacyInfo.xcprivacy` and is referenced by the Xcode project and application resources build phase;
- repeated execution produces the same resulting project instead of duplicate plist entries or build settings.

Android-only permissions, file providers, Java sources, and APK installation code remain inside the Android preparation path.

### iOS Release Workflow

The TestFlight workflow continues to generate `mobile/ios` on `macos-latest`. It does not rely on a committed Xcode project.

The workflow will use the following order:

1. Resolve and validate semantic version and numeric build number.
2. Install dependencies and build the shared mobile web assets.
3. Generate the Capacitor iOS project and assets.
4. Run Capacitor sync.
5. Apply RPBox iOS native configuration.
6. Synchronize marketing version, build number, bundle ID, development team, signing style, and provisioning profile.
7. Run explicit native project validation.
8. Resolve the scheme, archive, export, and upload when the workflow is used for an actual release.

This task does not push an iOS tag, dispatch the workflow, upload an IPA, or alter production updater metadata.

### Native Validation

Add `mobile/scripts/verifyIosProject.mjs` so CI can validate generated iOS output without relying on a sequence of loose `grep` commands.

Validation must report the missing setting and exit non-zero for:

- absent or malformed camera/photo permissions;
- missing custom URL scheme;
- missing associated domains;
- missing entitlements linkage;
- missing or unreferenced privacy manifest;
- a generated bundle ID that does not match the workflow-provided `IOS_BUNDLE_ID`;
- missing marketing version or build number;
- missing Capacitor Camera plugin registration.

The GitHub Actions workflow should call this validator immediately after project preparation and version/signing synchronization.

## Data Flow

The runtime update flow remains platform-specific after the shared update check:

1. The application detects `android` or `ios` through Capacitor.
2. It obtains the native application version through `App.getInfo()`.
3. It requests `/mobile/<target>/latest`, falling back to the legacy updater endpoint.
4. It compares normalized semantic versions.
5. Android native builds invoke the APK installer path.
6. iOS builds transform an Apple App Store HTTPS URL into `itms-apps://` and navigate to it.

No iOS binary download or in-app installation mechanism is introduced.

## Error Handling

- Native preparation must fail instead of silently returning when an explicitly requested iOS project file is missing.
- Mutation helpers must verify that the expected plist or Xcode build setting exists after writing.
- Validation errors must identify the affected file and setting.
- Missing signing secrets remain a release-workflow failure and are not replaced by fallback credentials.
- Shared update checks keep their current graceful fallback from the stable metadata endpoint to the legacy updater endpoint.
- Store-link failures remain visible through the existing updater state and UI rather than falling back to Android behavior.

## Testing

### Cross-Platform Unit Tests

Extend updater tests to cover:

- iOS platform detection;
- `ios-store` update mode;
- conversion of App Store HTTPS links to `itms-apps://`;
- preservation of non-App-Store HTTPS links;
- rejection of Android installer behavior outside Android native mode;
- version comparison for the current 1.0.41 release.

### Native Preparation Tests

Test iOS project preparation against temporary fixture files:

- a minimal `Info.plist`;
- a minimal `App.entitlements`;
- representative `project.pbxproj` build configurations;
- privacy manifest presence and project linkage;
- first-run mutation;
- repeated-run idempotency;
- clear failures for missing required files or unsuccessful injection.

The tests must not require Xcode and must run through the existing Node/Vitest toolchain.

### Build Verification

Run locally:

- mobile unit tests;
- mobile TypeScript type checking;
- mobile production build;
- native preparation fixture tests;
- JSON parsing for modified package files and `actionlint` validation for the modified GitHub Actions workflow.

The final iOS archive remains a macOS/Xcode responsibility. The workflow must contain all required deterministic preparation and validation steps so a later TestFlight run does not depend on local native files.

## Acceptance Criteria

- Current Android and shared mobile features are present in the iOS web bundle without duplicated implementation.
- Android APK installation code is not included or invoked in the iOS native path.
- iOS update actions open the configured App Store destination.
- A clean CI runner can generate and patch the iOS project from repository files.
- iOS permissions, URL scheme, associated domains, entitlements, privacy manifest, bundle ID, version, and build number are explicitly validated.
- Native preparation is idempotent.
- Mobile tests, type checking, and production build pass.
- No release tag is created and no TestFlight upload is triggered.

## Non-Goals

- No UI redesign or feature expansion.
- No change to Android APK release behavior.
- No committed `mobile/ios` generated project.
- No App Store Connect submission, TestFlight group distribution, or production metadata deployment.
- No signing-certificate or provisioning-profile rotation.
- No unrelated changes to the current dirty worktree.
