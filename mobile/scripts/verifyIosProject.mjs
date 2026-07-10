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
  if (!contents.includes(expectedText)) {
    errors.push(`Missing ${label}: ${expectedText}`)
  }
}

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function requireSetting(pbxproj, key, value, minimumCount = 2) {
  const matches = pbxproj.match(new RegExp(`${key} = "?${escapeRegex(value)}"?;`, 'g')) || []
  if (matches.length < minimumCount) {
    errors.push(`Expected ${minimumCount} ${key} settings with value ${value}, found ${matches.length}`)
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

const infoPlist = readRequired(path.join('ios', 'App', 'App', 'Info.plist'))
const entitlements = readRequired(path.join('ios', 'App', 'App', 'App.entitlements'))
const capacitorConfig = readRequired(path.join('ios', 'App', 'App', 'capacitor.config.json'))
const pbxproj = readRequired(path.join('ios', 'App', 'App.xcodeproj', 'project.pbxproj'))
const privacyManifest = readRequired(path.join('ios', 'App', 'App', 'PrivacyInfo.xcprivacy'))

for (const key of [
  'NSCameraUsageDescription',
  'NSPhotoLibraryUsageDescription',
  'NSPhotoLibraryAddUsageDescription',
]) {
  requireText(infoPlist, `<key>${key}</key>`, key)
}
requireText(infoPlist, '<string>app.rpbox.mobile</string>', 'custom URL scheme')
requireText(entitlements, 'applinks:totalrpbox.com', 'associated domain')
requireText(entitlements, 'applinks:www.totalrpbox.com', 'associated domain')
requireText(pbxproj, 'CODE_SIGN_ENTITLEMENTS = App/App.entitlements;', 'entitlements build setting')
requireText(pbxproj, 'PrivacyInfo.xcprivacy in Resources', 'privacy manifest resource')
requireText(pbxproj, 'path = PrivacyInfo.xcprivacy;', 'privacy manifest file reference')
requireText(privacyManifest, 'NSPrivacyAccessedAPICategoryFileTimestamp', 'privacy manifest API category')
requireText(privacyManifest, 'C617.1', 'privacy manifest reason')

if (capacitorConfig) {
  try {
    const config = JSON.parse(capacitorConfig)
    if (!Array.isArray(config.packageClassList) || !config.packageClassList.includes('CAPCameraPlugin')) {
      errors.push('Missing Capacitor Camera plugin registration: CAPCameraPlugin')
    }
  } catch (error) {
    errors.push(`Invalid capacitor.config.json: ${error instanceof Error ? error.message : String(error)}`)
  }
}

if (pbxproj) {
  requireSetting(pbxproj, 'MARKETING_VERSION', expected.version)
  requireSetting(pbxproj, 'CURRENT_PROJECT_VERSION', expected.buildNumber)
  requireSetting(pbxproj, 'DEVELOPMENT_TEAM', expected.teamId)
  requireSetting(pbxproj, 'PRODUCT_BUNDLE_IDENTIFIER', expected.bundleId)
  requireSetting(pbxproj, 'CODE_SIGN_STYLE', 'Manual')
  requireSetting(pbxproj, 'PROVISIONING_PROFILE_SPECIFIER', expected.profileName)
  requireSetting(pbxproj, 'CODE_SIGN_ENTITLEMENTS', 'App/App.entitlements')
}

if (errors.length > 0) {
  for (const error of errors) console.error(`[iOS Verify] ${error}`)
  process.exit(1)
}

console.log('[iOS Verify] Generated iOS project is valid')
