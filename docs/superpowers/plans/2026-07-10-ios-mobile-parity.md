# RPBox iOS Mobile Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the generated iOS application deterministically include the current mobile 1.0.41 behavior and all required native/TestFlight configuration without publishing a release.

**Architecture:** Keep `mobile/src` as the shared Android/iOS application and add deterministic iOS platform resolution tests. Extend the existing native preparation script to patch the generated Xcode project, add a standalone validator for generated iOS artifacts, and wire both into the TestFlight workflow after explicit version and signing synchronization.

**Tech Stack:** Vue 3, TypeScript, Vitest, Node.js ESM, Node test runner, Capacitor 6, GitHub Actions, Xcode project text format.

## Global Constraints

- Keep `mobile/src` as the single Android/iOS feature implementation.
- Keep Android APK installation code Android-only.
- Keep iOS updates on the App Store/TestFlight URL path.
- Do not commit the generated `mobile/ios` directory.
- Do not create or push a release tag.
- Do not dispatch TestFlight or upload an IPA.
- Do not modify unrelated dirty worktree files.
- Keep JSON edits on ASCII quotes and validate JSON parsing.

## File Map

- Modify `mobile/src/api/updater.ts`: expose a pure mobile-target resolver while preserving the runtime wrapper.
- Modify `mobile/src/api/updater.test.ts`: cover deterministic iOS detection and iOS update behavior.
- Modify `mobile/scripts/prepareNativeShare.mjs`: make iOS plist and Xcode project patching idempotent and include the application privacy manifest.
- Create `mobile/scripts/verifyIosProject.mjs`: validate generated iOS permissions, plugins, links, entitlements, privacy resources, version, bundle, and signing settings.
- Create `mobile/scripts/iosProjectConfig.test.mjs`: exercise native preparation and verification against temporary generated-project fixtures without Xcode.
- Modify `mobile/package.json`: add native iOS test and verification commands.
- Modify `.github/workflows/release-ios-testflight.yml`: run tests, synchronize all native settings, and invoke the validator before archive.

---

### Task 1: Deterministic iOS Runtime Branch Coverage

**Files:**
- Modify: `mobile/src/api/updater.ts:55-76`
- Modify: `mobile/src/api/updater.test.ts:1-55`

**Interfaces:**
- Produces: `resolveMobileTarget(platform: string, userAgent: string): MobileTarget | null`
- Preserves: `detectMobileTarget(): MobileTarget | null`
- Preserves: `getMobileUpdateMode(target?: MobileTarget | null): MobileUpdateMode`

- [ ] **Step 1: Write failing platform-resolution tests**

Add imports and cases to `mobile/src/api/updater.test.ts`:

```ts
import {
  getMobileUpdateMode,
  installAndroidUpdate,
  isNewerVersion,
  normalizeVersion,
  resolveIOSUpdateUrl,
  resolveMobileTarget,
  resolveUpdateDownloadUrl,
} from './updater'

it('resolves native and browser mobile targets deterministically', () => {
  expect(resolveMobileTarget('ios', 'Mozilla/5.0')).toBe('ios')
  expect(resolveMobileTarget('android', 'Mozilla/5.0')).toBe('android')
  expect(resolveMobileTarget('web', 'Mozilla/5.0 (iPhone)')).toBe('ios')
  expect(resolveMobileTarget('web', 'Mozilla/5.0 (Linux; Android 15)')).toBe('android')
  expect(resolveMobileTarget('web', 'Mozilla/5.0 (Windows NT 10.0)')).toBeNull()
})

it('uses the iOS store update mode', () => {
  expect(getMobileUpdateMode('ios')).toBe('ios-store')
})

it('keeps non-App-Store iOS update links unchanged', () => {
  expect(resolveIOSUpdateUrl('https://example.com/ios-beta')).toBe('https://example.com/ios-beta')
})

it('compares the synchronized 1.0.41 release correctly', () => {
  expect(isNewerVersion('1.0.41', '1.0.40')).toBe(true)
  expect(isNewerVersion('1.0.41', '1.0.41')).toBe(false)
})

it('does not expose Android installation outside Android native mode', async () => {
  await expect(installAndroidUpdate({
    version: '1.0.41',
    url: 'https://example.com/RPBox_1.0.41_android.apk',
  })).rejects.toThrow('Android in-app updater is unavailable')
})
```

- [ ] **Step 2: Run the focused test and confirm the new helper is missing**

Run:

```powershell
pnpm --filter rpbox-mobile test -- src/api/updater.test.ts
```

Expected: FAIL because `resolveMobileTarget` is not exported.

- [ ] **Step 3: Implement the pure resolver and keep runtime detection thin**

Replace the current `detectMobileTarget` body in `mobile/src/api/updater.ts` with:

```ts
export function resolveMobileTarget(platform: string, userAgent: string): MobileTarget | null {
  if (platform === 'android' || platform === 'ios') {
    return platform
  }

  if (/Android/i.test(userAgent)) return 'android'
  if (/iPhone|iPad|iPod/i.test(userAgent)) return 'ios'
  return null
}

export function detectMobileTarget(): MobileTarget | null {
  return resolveMobileTarget(Capacitor.getPlatform(), navigator.userAgent || '')
}
```

- [ ] **Step 4: Run the focused updater tests**

Run:

```powershell
pnpm --filter rpbox-mobile test -- src/api/updater.test.ts
```

Expected: all updater tests PASS.

- [ ] **Step 5: Commit the runtime regression coverage**

```powershell
git add mobile/src/api/updater.ts mobile/src/api/updater.test.ts
git commit -m "test: cover ios mobile updater behavior"
```

---

### Task 2: Idempotent iOS Native Preparation and Validation

**Files:**
- Modify: `mobile/scripts/prepareNativeShare.mjs:430-571`
- Create: `mobile/scripts/verifyIosProject.mjs`
- Create: `mobile/scripts/iosProjectConfig.test.mjs`
- Modify: `mobile/package.json:6-16`

**Interfaces:**
- Preserves CLI: `node mobile/scripts/prepareNativeShare.mjs ios`
- Produces CLI: `node mobile/scripts/verifyIosProject.mjs`
- Consumes environment:
  - `IOS_EXPECTED_VERSION`
  - `IOS_EXPECTED_BUILD_NUMBER`
  - `IOS_TEAM_ID`
  - `IOS_BUNDLE_ID`
  - `IOS_PROVISION_PROFILE_NAME`

- [ ] **Step 1: Add failing native-project fixture tests**

Create `mobile/scripts/iosProjectConfig.test.mjs` using `node:test`, `node:assert/strict`, `node:child_process`, `node:fs`, `node:os`, and `node:path`.

The fixture must create:

```text
<temp>/mobile/ios/App/App/Info.plist
<temp>/mobile/ios/App/App/capacitor.config.json
<temp>/mobile/ios/App/App.xcodeproj/project.pbxproj
```

Use this minimal plist:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
</dict>
</plist>
```

Use this Capacitor config:

```json
{
  "appId": "app.rpbox.mobile",
  "packageClassList": ["AppPlugin", "CAPCameraPlugin"]
}
```

The PBX fixture must include:

- `PBXBuildFile`, `PBXFileReference`, and `PBXResourcesBuildPhase` sections;
- an `App` PBX group containing `Info.plist`;
- Debug and Release target build configurations;
- `CODE_SIGN_STYLE`, `CURRENT_PROJECT_VERSION`, `INFOPLIST_FILE`, `MARKETING_VERSION`, and `PRODUCT_BUNDLE_IDENTIFIER` in both target configurations.

Add tests that:

1. run `prepareNativeShare.mjs ios` from the temporary parent directory;
2. assert camera/photo keys, the custom scheme, associated domains, entitlements linkage, and `App/PrivacyInfo.xcprivacy`;
3. assert the PBX file reference, App group entry, and Resources build entry for `PrivacyInfo.xcprivacy`;
4. run preparation twice and assert `Info.plist`, `App.entitlements`, privacy manifest, and PBX contents are unchanged;
5. run `verifyIosProject.mjs` after replacing fixture build settings with:

```text
MARKETING_VERSION = 1.0.41;
CURRENT_PROJECT_VERSION = 1000041;
PRODUCT_BUNDLE_IDENTIFIER = app.rpbox.mobile;
DEVELOPMENT_TEAM = TEAM123456;
CODE_SIGN_STYLE = Manual;
PROVISIONING_PROFILE_SPECIFIER = "RPBox Distribution";
```

6. assert the validator exits non-zero and names `NSCameraUsageDescription` when that plist key is removed.

Invoke scripts with:

```js
execFileSync(process.execPath, [prepareScript, 'ios'], {
  cwd: fixtureRoot,
  env: process.env,
  stdio: 'pipe',
})
```

Run the validator with the five required environment values.

- [ ] **Step 2: Add the native test command and verify failure**

Add to `mobile/package.json`:

```json
"test:native-ios": "node --test ./scripts/iosProjectConfig.test.mjs",
"verify:ios-project": "node ./scripts/verifyIosProject.mjs",
```

Run:

```powershell
pnpm --filter rpbox-mobile test:native-ios
```

Expected: FAIL because the current preparation script writes the privacy manifest outside the app group and does not add it to the Xcode project; the validator script is also missing.

- [ ] **Step 3: Correct and harden iOS preparation**

In `mobile/scripts/prepareNativeShare.mjs`:

1. Replace the nested-array URL-types matcher with:

```js
/<key>CFBundleURLTypes<\/key>\s*<array>\s*<dict>[\s\S]*?<\/dict>\s*<\/array>/
```

2. Change the privacy manifest path to:

```js
const privacyManifestPath = path.join(mobileRoot, 'ios', 'App', 'App', 'PrivacyInfo.xcprivacy')
```

3. Add deterministic Xcode IDs:

```js
const iosPrivacyFileReferenceId = '52B0F1002B00000000000001'
const iosPrivacyBuildFileId = '52B0F1012B00000000000001'
```

4. Add an `ensureIosPrivacyManifestReferences(pbxproj)` helper that inserts exactly once:

```text
52B0F1012B00000000000001 /* PrivacyInfo.xcprivacy in Resources */ = {isa = PBXBuildFile; fileRef = 52B0F1002B00000000000001 /* PrivacyInfo.xcprivacy */; };
52B0F1002B00000000000001 /* PrivacyInfo.xcprivacy */ = {isa = PBXFileReference; lastKnownFileType = text.xml; path = PrivacyInfo.xcprivacy; sourceTree = "<group>"; };
```

It must also insert:

```text
52B0F1002B00000000000001 /* PrivacyInfo.xcprivacy */,
```

into the App PBX group before `Info.plist`, and:

```text
52B0F1012B00000000000001 /* PrivacyInfo.xcprivacy in Resources */,
```

into the application `PBXResourcesBuildPhase`.

Each insertion must check for its exact ID first. If the target PBX section or anchor cannot be found, throw an error naming the missing section.

5. Call `ensureIosPrivacyManifestReferences` before writing `project.pbxproj`.

6. Verify after writing that the PBX project contains:

```js
if (!pbxproj.includes('PrivacyInfo.xcprivacy in Resources')) {
  throw new Error(`Failed to add PrivacyInfo.xcprivacy to ${pbxprojPath}`)
}
```

- [ ] **Step 4: Implement the standalone iOS validator**

Create `mobile/scripts/verifyIosProject.mjs` with:

```js
import fs from 'node:fs'
import path from 'node:path'

const cwd = process.cwd()
const mobileRoot = path.basename(cwd) === 'mobile' ? cwd : path.join(cwd, 'mobile')
const expected = {
  version: process.env.IOS_EXPECTED_VERSION || '',
  buildNumber: process.env.IOS_EXPECTED_BUILD_NUMBER || '',
  teamId: process.env.IOS_TEAM_ID || '',
  bundleId: process.env.IOS_BUNDLE_ID || '',
  profileName: process.env.IOS_PROVISION_PROFILE_NAME || '',
}
const errors = []

function requireEnv(name, value) {
  if (!value) errors.push(`Missing environment value: ${name}`)
}

function readRequired(relativePath) {
  const filePath = path.join(mobileRoot, relativePath)
  if (!fs.existsSync(filePath)) {
    errors.push(`Missing file: ${filePath}`)
    return ''
  }
  return fs.readFileSync(filePath, 'utf8')
}

function requireText(contents, expectedText, label) {
  if (!contents.includes(expectedText)) errors.push(`Missing ${label}: ${expectedText}`)
}

function requireSetting(pbxproj, key, value, minimumCount = 2) {
  const escaped = value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const matches = pbxproj.match(new RegExp(`${key} = "?${escaped}"?;`, 'g')) || []
  if (matches.length < minimumCount) {
    errors.push(`Expected ${minimumCount} ${key} settings with value ${value}, found ${matches.length}`)
  }
}
```

Then:

- require all five environment values;
- read `ios/App/App/Info.plist`;
- read `ios/App/App/App.entitlements`;
- read `ios/App/App/capacitor.config.json`;
- read `ios/App/App.xcodeproj/project.pbxproj`;
- require `ios/App/App/PrivacyInfo.xcprivacy` to exist;
- require the three photo/camera keys;
- require `<string>app.rpbox.mobile</string>`;
- require both associated domains;
- parse `capacitor.config.json` and require `CAPCameraPlugin`;
- require PBX entitlements linkage and privacy resource strings;
- require the expected version, build number, team ID, bundle ID, manual signing style, and provisioning profile in both target configurations.

Finish with:

```js
if (errors.length > 0) {
  for (const error of errors) console.error(`[iOS Verify] ${error}`)
  process.exit(1)
}

console.log('[iOS Verify] Generated iOS project is valid')
```

Wrap JSON parsing in `try/catch` and add the parse error to `errors`.

- [ ] **Step 5: Run native preparation and validator tests**

Run:

```powershell
pnpm --filter rpbox-mobile test:native-ios
```

Expected: all native iOS fixture tests PASS.

- [ ] **Step 6: Validate package JSON**

Run:

```powershell
node -e "JSON.parse(require('fs').readFileSync('mobile/package.json','utf8')); console.log('mobile/package.json valid')"
```

Expected: `mobile/package.json valid`.

- [ ] **Step 7: Commit native preparation and validation**

```powershell
git add mobile/scripts/prepareNativeShare.mjs mobile/scripts/verifyIosProject.mjs mobile/scripts/iosProjectConfig.test.mjs mobile/package.json
git commit -m "fix: harden ios native project generation"
```

---

### Task 3: TestFlight Workflow Parity

**Files:**
- Modify: `.github/workflows/release-ios-testflight.yml:118-191`

**Interfaces:**
- Consumes: `pnpm --filter rpbox-mobile test:native-ios`
- Consumes: `pnpm --filter rpbox-mobile verify:ios-project`
- Passes validator environment values from the resolved workflow version and signing secrets.

- [ ] **Step 1: Add pre-build mobile validation**

After `Install dependencies`, add:

```yaml
      - name: Test mobile application
        run: |
          pnpm --filter rpbox-mobile test
          pnpm --filter rpbox-mobile test:native-ios
          pnpm --filter rpbox-mobile type-check
```

- [ ] **Step 2: Remove the partial camera-only verification**

Delete the existing `Verify iOS camera setup` step. Its checks are replaced by `verifyIosProject.mjs`, which also validates camera registration.

- [ ] **Step 3: Synchronize every required target build setting**

In `Sync iOS native version`, keep the existing semantic version and build number replacements, then ensure:

```bash
perl -0777 -i -pe "s/PRODUCT_BUNDLE_IDENTIFIER = [^;]+;/PRODUCT_BUNDLE_IDENTIFIER = ${IOS_BUNDLE_ID};/g" "$PBXPROJ"
perl -0777 -i -pe "s/CODE_SIGN_STYLE = [^;]+;/CODE_SIGN_STYLE = Manual;/g" "$PBXPROJ"
```

For the development team:

```bash
if grep -q "DEVELOPMENT_TEAM =" "$PBXPROJ"; then
  perl -0777 -i -pe "s/DEVELOPMENT_TEAM = [^;]+;/DEVELOPMENT_TEAM = ${IOS_TEAM_ID};/g" "$PBXPROJ"
else
  perl -0777 -i -pe "s/(CODE_SIGN_STYLE = Manual;)/\\1\\n\\t\\t\\t\\tDEVELOPMENT_TEAM = ${IOS_TEAM_ID};/g" "$PBXPROJ"
fi
```

For the provisioning profile:

```bash
if grep -q "PROVISIONING_PROFILE_SPECIFIER =" "$PBXPROJ"; then
  perl -0777 -i -pe "s/PROVISIONING_PROFILE_SPECIFIER = [^;]+;/PROVISIONING_PROFILE_SPECIFIER = \"${IOS_PROVISION_PROFILE_NAME}\";/g" "$PBXPROJ"
else
  perl -0777 -i -pe "s/(PRODUCT_BUNDLE_IDENTIFIER = ${IOS_BUNDLE_ID};)/\\1\\n\\t\\t\\t\\tPROVISIONING_PROFILE_SPECIFIER = \"${IOS_PROVISION_PROFILE_NAME}\";/g" "$PBXPROJ"
fi
```

- [ ] **Step 4: Add comprehensive generated-project validation**

Immediately after native version synchronization, add:

```yaml
      - name: Verify generated iOS project
        env:
          IOS_EXPECTED_VERSION: ${{ steps.version.outputs.VERSION }}
          IOS_EXPECTED_BUILD_NUMBER: ${{ steps.version.outputs.BUILD_NUMBER }}
          IOS_TEAM_ID: ${{ secrets.IOS_TEAM_ID }}
          IOS_BUNDLE_ID: ${{ secrets.IOS_BUNDLE_ID }}
          IOS_PROVISION_PROFILE_NAME: ${{ secrets.IOS_PROVISION_PROFILE_NAME }}
        run: pnpm --filter rpbox-mobile verify:ios-project
```

Keep scheme resolution after this validator so malformed projects fail before Xcode inspection.

- [ ] **Step 5: Validate workflow syntax**

Run:

```powershell
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/release-ios-testflight.yml
```

Expected: no output and exit code 0.

- [ ] **Step 6: Commit workflow parity**

```powershell
git add .github/workflows/release-ios-testflight.yml
git commit -m "ci: validate ios project before testflight build"
```

---

### Task 4: Full Verification and Clean Handoff

**Files:**
- Verify only; no new production files.

**Interfaces:**
- Confirms all acceptance criteria from `docs/superpowers/specs/2026-07-10-ios-mobile-parity-design.md`.

- [ ] **Step 1: Run all mobile unit tests**

Run:

```powershell
pnpm --filter rpbox-mobile test
```

Expected: all Vitest suites PASS.

- [ ] **Step 2: Run native iOS fixture tests**

Run:

```powershell
pnpm --filter rpbox-mobile test:native-ios
```

Expected: all Node native-project tests PASS.

- [ ] **Step 3: Run mobile type checking**

Run:

```powershell
pnpm --filter rpbox-mobile type-check
```

Expected: exit code 0 with no TypeScript errors.

- [ ] **Step 4: Build the production mobile bundle**

Run:

```powershell
pnpm --filter rpbox-mobile build
```

Expected: Vite production build completes successfully.

- [ ] **Step 5: Re-run workflow validation**

Run:

```powershell
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/release-ios-testflight.yml
```

Expected: no output and exit code 0.

- [ ] **Step 6: Inspect the scoped diff**

Run:

```powershell
git diff --check HEAD~3..HEAD
git show --stat --oneline HEAD~2..HEAD
git status --short
```

Expected:

- no whitespace errors;
- only the planned mobile iOS parity files in the new commits;
- unrelated pre-existing worktree changes remain untouched.

- [ ] **Step 7: Do not publish**

Confirm no `ios-v*` or `mobile-ios-v*` tag was created and no workflow was dispatched:

```powershell
git tag --points-at HEAD
```

Expected: no new iOS release tag.
