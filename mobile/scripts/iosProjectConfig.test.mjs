import assert from 'node:assert/strict'
import { execFileSync, spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptsDir = path.dirname(fileURLToPath(import.meta.url))
const prepareScript = path.join(scriptsDir, 'prepareNativeShare.mjs')
const verifyScript = path.join(scriptsDir, 'verifyIosProject.mjs')

const infoPlistFixture = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
</dict>
</plist>
`

const capacitorConfigFixture = `${JSON.stringify({
  appId: 'app.rpbox.mobile',
  packageClassList: ['AppPlugin', 'CAPCameraPlugin'],
}, null, 2)}\n`

const pbxprojFixture = `// !$*UTF8*$!
{
  objects = {

/* Begin PBXBuildFile section */
    50379B232058CBB4000EE86E /* capacitor.config.json in Resources */ = {isa = PBXBuildFile; fileRef = 50379B222058CBB4000EE86E /* capacitor.config.json */; };
/* End PBXBuildFile section */

/* Begin PBXFileReference section */
    50379B222058CBB4000EE86E /* capacitor.config.json */ = {isa = PBXFileReference; lastKnownFileType = text.json; path = capacitor.config.json; sourceTree = "<group>"; };
    504EC3131FED79650016851F /* Info.plist */ = {isa = PBXFileReference; lastKnownFileType = text.plist.xml; path = Info.plist; sourceTree = "<group>"; };
/* End PBXFileReference section */

/* Begin PBXGroup section */
    504EC3061FED79650016851F /* App */ = {
      isa = PBXGroup;
      children = (
        50379B222058CBB4000EE86E /* capacitor.config.json */,
        504EC3131FED79650016851F /* Info.plist */,
      );
      path = App;
      sourceTree = "<group>";
    };
/* End PBXGroup section */

/* Begin PBXResourcesBuildPhase section */
    504EC3021FED79650016851F /* Resources */ = {
      isa = PBXResourcesBuildPhase;
      buildActionMask = 2147483647;
      files = (
        50379B232058CBB4000EE86E /* capacitor.config.json in Resources */,
      );
      runOnlyForDeploymentPostprocessing = 0;
    };
/* End PBXResourcesBuildPhase section */

/* Begin XCBuildConfiguration section */
    504EC3171FED79650016851F /* Debug */ = {
      isa = XCBuildConfiguration;
      buildSettings = {
        CODE_SIGN_STYLE = Manual;
        CURRENT_PROJECT_VERSION = 1000041;
        INFOPLIST_FILE = App/Info.plist;
        MARKETING_VERSION = 1.0.41;
        PRODUCT_BUNDLE_IDENTIFIER = app.rpbox.mobile;
        DEVELOPMENT_TEAM = TEAM123456;
        PROVISIONING_PROFILE_SPECIFIER = "RPBox Distribution";
      };
      name = Debug;
    };
    504EC3181FED79650016851F /* Release */ = {
      isa = XCBuildConfiguration;
      buildSettings = {
        CODE_SIGN_STYLE = Manual;
        CURRENT_PROJECT_VERSION = 1000041;
        INFOPLIST_FILE = App/Info.plist;
        MARKETING_VERSION = 1.0.41;
        PRODUCT_BUNDLE_IDENTIFIER = app.rpbox.mobile;
        DEVELOPMENT_TEAM = TEAM123456;
        PROVISIONING_PROFILE_SPECIFIER = "RPBox Distribution";
      };
      name = Release;
    };
/* End XCBuildConfiguration section */
  };
}
`

const verifyEnv = {
  IOS_EXPECTED_VERSION: '1.0.41',
  IOS_EXPECTED_BUILD_NUMBER: '1000041',
  IOS_TEAM_ID: 'TEAM123456',
  IOS_BUNDLE_ID: 'app.rpbox.mobile',
  IOS_PROVISION_PROFILE_NAME: 'RPBox Distribution',
}

function writeFile(filePath, contents) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true })
  fs.writeFileSync(filePath, contents, 'utf8')
}

function createFixture(t) {
  const fixtureRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'rpbox-ios-project-'))
  const mobileRoot = path.join(fixtureRoot, 'mobile')
  const infoPlistPath = path.join(mobileRoot, 'ios', 'App', 'App', 'Info.plist')
  const capacitorConfigPath = path.join(mobileRoot, 'ios', 'App', 'App', 'capacitor.config.json')
  const pbxprojPath = path.join(mobileRoot, 'ios', 'App', 'App.xcodeproj', 'project.pbxproj')

  writeFile(infoPlistPath, infoPlistFixture)
  writeFile(capacitorConfigPath, capacitorConfigFixture)
  writeFile(pbxprojPath, pbxprojFixture)
  t.after(() => fs.rmSync(fixtureRoot, { recursive: true, force: true }))

  return {
    fixtureRoot,
    mobileRoot,
    infoPlistPath,
    entitlementsPath: path.join(mobileRoot, 'ios', 'App', 'App', 'App.entitlements'),
    privacyManifestPath: path.join(mobileRoot, 'ios', 'App', 'App', 'PrivacyInfo.xcprivacy'),
    pbxprojPath,
  }
}

function runPrepare(fixtureRoot) {
  execFileSync(process.execPath, [prepareScript, 'ios'], {
    cwd: fixtureRoot,
    env: process.env,
    stdio: 'pipe',
  })
}

function runPrepareResult(fixtureRoot) {
  return spawnSync(process.execPath, [prepareScript, 'ios'], {
    cwd: fixtureRoot,
    env: process.env,
    encoding: 'utf8',
  })
}

function runVerify(fixtureRoot) {
  return spawnSync(process.execPath, [verifyScript], {
    cwd: fixtureRoot,
    env: { ...process.env, ...verifyEnv },
    encoding: 'utf8',
  })
}

test('prepares the generated iOS project idempotently', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)

  const firstInfoPlist = fs.readFileSync(fixture.infoPlistPath, 'utf8')
  const firstEntitlements = fs.readFileSync(fixture.entitlementsPath, 'utf8')
  const firstPrivacyManifest = fs.readFileSync(fixture.privacyManifestPath, 'utf8')
  const firstPbxproj = fs.readFileSync(fixture.pbxprojPath, 'utf8')

  assert.match(firstInfoPlist, /<key>NSCameraUsageDescription<\/key>/)
  assert.match(firstInfoPlist, /<key>NSPhotoLibraryUsageDescription<\/key>/)
  assert.match(firstInfoPlist, /<key>NSPhotoLibraryAddUsageDescription<\/key>/)
  assert.match(firstInfoPlist, /<string>app\.rpbox\.mobile<\/string>/)
  assert.equal((firstInfoPlist.match(/<key>CFBundleURLTypes<\/key>/g) || []).length, 1)
  assert.match(firstEntitlements, /applinks:totalrpbox\.com/)
  assert.match(firstEntitlements, /applinks:www\.totalrpbox\.com/)
  assert.match(firstPbxproj, /CODE_SIGN_ENTITLEMENTS = App\/App\.entitlements;/)
  assert.match(firstPbxproj, /PrivacyInfo\.xcprivacy in Resources/)
  assert.match(firstPbxproj, /path = PrivacyInfo\.xcprivacy;/)

  runPrepare(fixture.fixtureRoot)

  assert.equal(fs.readFileSync(fixture.infoPlistPath, 'utf8'), firstInfoPlist)
  assert.equal(fs.readFileSync(fixture.entitlementsPath, 'utf8'), firstEntitlements)
  assert.equal(fs.readFileSync(fixture.privacyManifestPath, 'utf8'), firstPrivacyManifest)
  assert.equal(fs.readFileSync(fixture.pbxprojPath, 'utf8'), firstPbxproj)
})

test('preserves unrelated URL schemes while adding the RPBox scheme', (t) => {
  const fixture = createFixture(t)
  const plistWithExistingScheme = infoPlistFixture.replace(
    '</dict>',
    `\t<key>CFBundleURLTypes</key>
\t<array>
\t\t<dict>
\t\t\t<key>CFBundleURLName</key>
\t\t\t<string>oauth.example</string>
\t\t\t<key>CFBundleURLSchemes</key>
\t\t\t<array>
\t\t\t\t<string>oauth-example</string>
\t\t\t</array>
\t\t</dict>
\t</array>
</dict>`,
  )
  fs.writeFileSync(fixture.infoPlistPath, plistWithExistingScheme, 'utf8')

  runPrepare(fixture.fixtureRoot)

  const plist = fs.readFileSync(fixture.infoPlistPath, 'utf8')
  assert.match(plist, /<string>oauth-example<\/string>/)
  assert.match(plist, /<string>app\.rpbox\.mobile<\/string>/)
})

test('verifies a complete generated iOS project', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 0, `${result.stdout}\n${result.stderr}`)
  assert.match(result.stdout, /Generated iOS project is valid/)
})

test('reports a missing camera usage description', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const plist = fs.readFileSync(fixture.infoPlistPath, 'utf8').replace(
    /\s*<key>NSCameraUsageDescription<\/key>\s*<string>[\s\S]*?<\/string>/,
    '',
  )
  fs.writeFileSync(fixture.infoPlistPath, plist, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /NSCameraUsageDescription/)
})

test('reports an incomplete privacy manifest', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const privacyManifest = fs.readFileSync(fixture.privacyManifestPath, 'utf8').replace('C617.1', 'INVALID')
  fs.writeFileSync(fixture.privacyManifestPath, privacyManifest, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /privacy manifest reason/)
})

test('reports a privacy manifest missing from the resources build phase', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const pbxproj = fs.readFileSync(fixture.pbxprojPath, 'utf8').replace(
    /^\s*52B0F1012B00000000000001 \/\* PrivacyInfo\.xcprivacy in Resources \*\/,\s*$/m,
    '',
  )
  fs.writeFileSync(fixture.pbxprojPath, pbxproj, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /Resources build phase/)
})

test('reports a privacy manifest missing from the App group', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const pbxproj = fs.readFileSync(fixture.pbxprojPath, 'utf8').replace(
    /^\s*52B0F1002B00000000000001 \/\* PrivacyInfo\.xcprivacy \*\/,\s*$/m,
    '',
  )
  fs.writeFileSync(fixture.pbxprojPath, pbxproj, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /App group/)
})

test('reports a broken privacy manifest build-file relationship', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const pbxproj = fs.readFileSync(fixture.pbxprojPath, 'utf8').replace(
    'fileRef = 52B0F1002B00000000000001 /* PrivacyInfo.xcprivacy */;',
    'fileRef = 000000000000000000000000 /* PrivacyInfo.xcprivacy */;',
  )
  fs.writeFileSync(fixture.pbxprojPath, pbxproj, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /build file relationship/)
})

test('reports an empty camera permission description', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const plist = fs.readFileSync(fixture.infoPlistPath, 'utf8').replace(
    /(<key>NSCameraUsageDescription<\/key>\s*<string>)[\s\S]*?(<\/string>)/,
    '$1   $2',
  )
  fs.writeFileSync(fixture.infoPlistPath, plist, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /non-empty NSCameraUsageDescription/)
})

test('reports a missing RPBox URL scheme array', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const plist = fs.readFileSync(fixture.infoPlistPath, 'utf8').replace(
    /\s*<key>CFBundleURLSchemes<\/key>\s*<array>[\s\S]*?<\/array>/,
    '',
  )
  fs.writeFileSync(fixture.infoPlistPath, plist, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /custom URL scheme/)
})

test('reports associated domains outside the expected entitlement key', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  const entitlements = fs.readFileSync(fixture.entitlementsPath, 'utf8').replace(
    'com.apple.developer.associated-domains',
    'rpbox.invalid-associated-domains',
  )
  fs.writeFileSync(fixture.entitlementsPath, entitlements, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /com\.apple\.developer\.associated-domains/)
})

test('validates every target build configuration independently', (t) => {
  const fixture = createFixture(t)
  runPrepare(fixture.fixtureRoot)
  let pbxproj = fs.readFileSync(fixture.pbxprojPath, 'utf8')
  pbxproj = pbxproj.replace(
    /(504EC3171FED79650016851F \/\* Debug \*\/[\s\S]*?PRODUCT_BUNDLE_IDENTIFIER = app\.rpbox\.mobile;)/,
    '$1\n        PRODUCT_BUNDLE_IDENTIFIER = app.rpbox.mobile;',
  )
  pbxproj = pbxproj.replace(
    /(504EC3181FED79650016851F \/\* Release \*\/[\s\S]*?)PRODUCT_BUNDLE_IDENTIFIER = app\.rpbox\.mobile;/,
    '$1PRODUCT_BUNDLE_IDENTIFIER = app.rpbox.invalid;',
  )
  fs.writeFileSync(fixture.pbxprojPath, pbxproj, 'utf8')

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /Release.*PRODUCT_BUNDLE_IDENTIFIER/)
})

test('fails preparation when the resources build phase is missing', (t) => {
  const fixture = createFixture(t)
  const pbxproj = fs.readFileSync(fixture.pbxprojPath, 'utf8').replace(
    /\/\* Begin PBXResourcesBuildPhase section \*\/[\s\S]*?\/\* End PBXResourcesBuildPhase section \*\//,
    '',
  )
  fs.writeFileSync(fixture.pbxprojPath, pbxproj, 'utf8')

  const result = runPrepareResult(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /Missing PBXResourcesBuildPhase/)
})

test('reports a missing Xcode project file', (t) => {
  const fixture = createFixture(t)
  fs.rmSync(fixture.pbxprojPath)

  const result = runVerify(fixture.fixtureRoot)

  assert.equal(result.status, 1)
  assert.match(result.stderr, /ios\/App\/App\.xcodeproj\/project\.pbxproj: missing file/)
})
