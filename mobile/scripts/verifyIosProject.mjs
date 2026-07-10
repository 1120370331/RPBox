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
const iosPrivacyFileReferenceId = '52B0F1002B00000000000001'
const iosPrivacyBuildFileId = '52B0F1012B00000000000001'
const projectPaths = {
  infoPlist: path.join('ios', 'App', 'App', 'Info.plist'),
  entitlements: path.join('ios', 'App', 'App', 'App.entitlements'),
  capacitorConfig: path.join('ios', 'App', 'App', 'capacitor.config.json'),
  pbxproj: path.join('ios', 'App', 'App.xcodeproj', 'project.pbxproj'),
  privacyManifest: path.join('ios', 'App', 'App', 'PrivacyInfo.xcprivacy'),
}

function displayPath(relativePath) {
  return relativePath.split(path.sep).join('/')
}

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function requireEnv(name, value) {
  if (!value) errors.push(`Missing environment value: ${name}`)
}

function readRequired(relativePath) {
  const filePath = path.join(mobileRoot, relativePath)
  if (!fs.existsSync(filePath)) {
    errors.push(`${displayPath(relativePath)}: missing file`)
    return ''
  }
  return fs.readFileSync(filePath, 'utf8')
}

function requireText(contents, expectedText, label, relativePath) {
  if (!contents.includes(expectedText)) {
    errors.push(`${displayPath(relativePath)}: missing ${label}: ${expectedText}`)
  }
}

function requirePattern(contents, pattern, label, relativePath) {
  if (!pattern.test(contents)) {
    errors.push(`${displayPath(relativePath)}: missing ${label}`)
  }
}

function requirePlistString(contents, key, relativePath) {
  const pattern = new RegExp(`<key>${escapeRegex(key)}<\\/key>\\s*<string>([\\s\\S]*?)<\\/string>`)
  const value = contents.match(pattern)?.[1]?.trim() || ''
  if (!value) {
    errors.push(`${displayPath(relativePath)}: missing non-empty ${key} string value`)
  }
}

function requirePlistArrayString(contents, key, value, relativePath) {
  const pattern = new RegExp(`<key>${escapeRegex(key)}<\\/key>\\s*<array>([\\s\\S]*?)<\\/array>`)
  const arrayContents = contents.match(pattern)?.[1] || ''
  const valuePattern = new RegExp(`<string>\\s*${escapeRegex(value)}\\s*<\\/string>`)
  if (!valuePattern.test(arrayContents)) {
    errors.push(`${displayPath(relativePath)}: ${key} is missing ${value}`)
  }
}

function requireRpboxUrlScheme(infoPlist) {
  const appId = 'app.rpbox.mobile'
  const escapedAppId = escapeRegex(appId)
  const pattern = new RegExp(
    `<dict>\\s*<key>CFBundleURLName<\\/key>\\s*<string>\\s*${escapedAppId}\\s*<\\/string>\\s*` +
    `<key>CFBundleURLSchemes<\\/key>\\s*<array>[\\s\\S]*?<string>\\s*${escapedAppId}\\s*<\\/string>[\\s\\S]*?<\\/array>\\s*<\\/dict>`,
  )
  if (!pattern.test(infoPlist)) {
    errors.push(`${displayPath(projectPaths.infoPlist)}: missing RPBox custom URL scheme dictionary`)
  }
}

function findAppTargetBuildConfigurations(pbxproj) {
  const section = pbxproj.match(/\/\* Begin XCBuildConfiguration section \*\/([\s\S]*?)\/\* End XCBuildConfiguration section \*\//)?.[1] || ''
  const entryPattern = /[A-F0-9]{24} \/\* ([^*]+) \*\/ = \{\s*isa = XCBuildConfiguration;[\s\S]*?buildSettings = \{([\s\S]*?)\n\s*\};\s*name = ([^;]+);/g
  const configurations = []
  let match
  while ((match = entryPattern.exec(section)) !== null) {
    const settings = match[2]
    if (!settings.includes('INFOPLIST_FILE = App/Info.plist;')) continue
    configurations.push({
      name: match[3].trim().replace(/^"|"$/g, ''),
      settings,
    })
  }
  return configurations
}

function getBuildSettingValues(settings, key) {
  const pattern = new RegExp(`^\\s*${escapeRegex(key)} = ([^;]+);\\s*$`, 'gm')
  return Array.from(settings.matchAll(pattern), (match) => match[1].trim().replace(/^"|"$/g, ''))
}

function requireConfigurationSetting(configuration, key, value) {
  const values = getBuildSettingValues(configuration.settings, key)
  if (values.length !== 1 || values[0] !== value) {
    errors.push(
      `${displayPath(projectPaths.pbxproj)}: ${configuration.name} configuration expected ${key}=${value}, found ${values.length > 0 ? values.join(', ') : 'missing'}`,
    )
  }
}

for (const [name, value] of [
  ['IOS_EXPECTED_VERSION', expected.version],
  ['IOS_EXPECTED_BUILD_NUMBER', expected.buildNumber],
  ['IOS_TEAM_ID', expected.teamId],
  ['IOS_BUNDLE_ID', expected.bundleId],
  ['IOS_PROVISION_PROFILE_NAME', expected.profileName],
]) {
  requireEnv(name, value)
}

const infoPlist = readRequired(projectPaths.infoPlist)
const entitlements = readRequired(projectPaths.entitlements)
const capacitorConfig = readRequired(projectPaths.capacitorConfig)
const pbxproj = readRequired(projectPaths.pbxproj)
const privacyManifest = readRequired(projectPaths.privacyManifest)

for (const key of [
  'NSCameraUsageDescription',
  'NSPhotoLibraryUsageDescription',
  'NSPhotoLibraryAddUsageDescription',
]) {
  requirePlistString(infoPlist, key, projectPaths.infoPlist)
}
requireRpboxUrlScheme(infoPlist)
requirePlistArrayString(
  entitlements,
  'com.apple.developer.associated-domains',
  'applinks:totalrpbox.com',
  projectPaths.entitlements,
)
requirePlistArrayString(
  entitlements,
  'com.apple.developer.associated-domains',
  'applinks:www.totalrpbox.com',
  projectPaths.entitlements,
)

requirePattern(
  pbxproj,
  new RegExp(`^\\s*${iosPrivacyFileReferenceId} \\/\\* PrivacyInfo\\.xcprivacy \\*\\/ = \\{isa = PBXFileReference;[^\\n]*path = PrivacyInfo\\.xcprivacy;[^\\n]*\\};\\s*$`, 'm'),
  'privacy manifest file reference definition',
  projectPaths.pbxproj,
)
requirePattern(
  pbxproj,
  new RegExp(`^\\s*${iosPrivacyBuildFileId} \\/\\* PrivacyInfo\\.xcprivacy in Resources \\*\\/ = \\{isa = PBXBuildFile; fileRef = ${iosPrivacyFileReferenceId} \\/\\* PrivacyInfo\\.xcprivacy \\*\\/; \\};\\s*$`, 'm'),
  'privacy manifest build file relationship',
  projectPaths.pbxproj,
)

const appGroupContents = pbxproj.match(/[A-F0-9]{24} \/\* App \*\/ = \{\s*isa = PBXGroup;\s*children = \(([\s\S]*?)\);\s*path = App;/)?.[1] || ''
requirePattern(
  appGroupContents,
  new RegExp(`^\\s*${iosPrivacyFileReferenceId} \\/\\* PrivacyInfo\\.xcprivacy \\*\\/,\\s*$`, 'm'),
  'privacy manifest App group entry',
  projectPaths.pbxproj,
)

const resourcesPhaseContents = Array.from(
  pbxproj.matchAll(/isa = PBXResourcesBuildPhase;[\s\S]*?files = \(([\s\S]*?)\);/g),
  (match) => match[1],
).find((contents) => contents.includes('capacitor.config.json in Resources')) || ''
requirePattern(
  resourcesPhaseContents,
  new RegExp(`^\\s*${iosPrivacyBuildFileId} \\/\\* PrivacyInfo\\.xcprivacy in Resources \\*\\/,\\s*$`, 'm'),
  'privacy manifest Resources build phase entry',
  projectPaths.pbxproj,
)

requireText(privacyManifest, 'NSPrivacyAccessedAPICategoryFileTimestamp', 'privacy manifest API category', projectPaths.privacyManifest)
requireText(privacyManifest, 'C617.1', 'privacy manifest reason', projectPaths.privacyManifest)

if (capacitorConfig) {
  try {
    const config = JSON.parse(capacitorConfig)
    if (!Array.isArray(config.packageClassList) || !config.packageClassList.includes('CAPCameraPlugin')) {
      errors.push(`${displayPath(projectPaths.capacitorConfig)}: missing Capacitor Camera plugin registration CAPCameraPlugin`)
    }
  } catch (error) {
    errors.push(`${displayPath(projectPaths.capacitorConfig)}: invalid JSON: ${error instanceof Error ? error.message : String(error)}`)
  }
}

if (pbxproj) {
  const configurations = findAppTargetBuildConfigurations(pbxproj)
  if (configurations.length === 0) {
    errors.push(`${displayPath(projectPaths.pbxproj)}: no application target build configurations found`)
  }
  for (const requiredName of ['Debug', 'Release']) {
    if (!configurations.some((configuration) => configuration.name === requiredName)) {
      errors.push(`${displayPath(projectPaths.pbxproj)}: missing ${requiredName} application target configuration`)
    }
  }
  for (const configuration of configurations) {
    requireConfigurationSetting(configuration, 'MARKETING_VERSION', expected.version)
    requireConfigurationSetting(configuration, 'CURRENT_PROJECT_VERSION', expected.buildNumber)
    requireConfigurationSetting(configuration, 'DEVELOPMENT_TEAM', expected.teamId)
    requireConfigurationSetting(configuration, 'PRODUCT_BUNDLE_IDENTIFIER', expected.bundleId)
    requireConfigurationSetting(configuration, 'CODE_SIGN_STYLE', 'Manual')
    requireConfigurationSetting(configuration, 'PROVISIONING_PROFILE_SPECIFIER', expected.profileName)
    requireConfigurationSetting(configuration, 'CODE_SIGN_ENTITLEMENTS', 'App/App.entitlements')
  }
}

if (errors.length > 0) {
  for (const error of errors) console.error(`[iOS Verify] ${error}`)
  process.exit(1)
}

console.log('[iOS Verify] Generated iOS project is valid')
