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
  requireGeneratedWorkspace: process.env.IOS_REQUIRE_GENERATED_WORKSPACE === 'true',
}
const errors = []
const privacyFileReferenceId = '52B0F1002B00000000000001'
const privacyBuildFileId = '52B0F1012B00000000000001'
const projectPaths = {
  infoPlist: path.join('ios', 'App', 'App', 'Info.plist'),
  entitlements: path.join('ios', 'App', 'App', 'App.entitlements'),
  capacitorConfig: path.join('ios', 'App', 'App', 'capacitor.config.json'),
  pbxproj: path.join('ios', 'App', 'App.xcodeproj', 'project.pbxproj'),
  workspaceContents: path.join('ios', 'App', 'App.xcworkspace', 'contents.xcworkspacedata'),
  podfileLock: path.join('ios', 'App', 'Podfile.lock'),
  podsManifest: path.join('ios', 'App', 'Pods', 'Manifest.lock'),
  privacyManifest: path.join('ios', 'App', 'App', 'PrivacyInfo.xcprivacy'),
}

function displayPath(relativePath) {
  return relativePath.split(path.sep).join('/')
}

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
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
    errors.push(`${displayPath(relativePath)}: missing ${label}`)
  }
}

function requireSetting(settings, key, value, configurationName) {
  const pattern = new RegExp(`^\\s*${escapeRegex(key)} = ([^;]+);\\s*$`, 'm')
  const actual = settings.match(pattern)?.[1]?.trim().replace(/^"|"$/g, '')
  if (actual !== value) {
    errors.push(`iOS ${configurationName}: expected ${key}=${value}, found ${actual || 'missing'}`)
  }
}

for (const [name, value] of Object.entries(expected).filter(([, value]) => typeof value === 'string')) {
  if (!value) {
    errors.push(`Missing environment value: ${name === 'version' ? 'IOS_EXPECTED_VERSION' : name === 'buildNumber' ? 'IOS_EXPECTED_BUILD_NUMBER' : name === 'teamId' ? 'IOS_TEAM_ID' : name === 'bundleId' ? 'IOS_BUNDLE_ID' : 'IOS_PROVISION_PROFILE_NAME'}`)
  }
}

const infoPlist = readRequired(projectPaths.infoPlist)
const entitlements = readRequired(projectPaths.entitlements)
const capacitorConfig = readRequired(projectPaths.capacitorConfig)
const pbxproj = readRequired(projectPaths.pbxproj)
const privacyManifest = readRequired(projectPaths.privacyManifest)

if (expected.requireGeneratedWorkspace) {
  readRequired(projectPaths.workspaceContents)
  readRequired(projectPaths.podfileLock)
  readRequired(projectPaths.podsManifest)
}

for (const key of [
  'NSCameraUsageDescription',
  'NSPhotoLibraryUsageDescription',
  'NSPhotoLibraryAddUsageDescription',
]) {
  requireText(infoPlist, `<key>${key}</key>`, `${key} usage description`, projectPaths.infoPlist)
}
requireText(infoPlist, '<string>app.rpbox.mobile</string>', 'RPBox URL scheme', projectPaths.infoPlist)
requireText(entitlements, '<string>applinks:totalrpbox.com</string>', 'totalrpbox.com associated domain', projectPaths.entitlements)
requireText(entitlements, '<string>applinks:www.totalrpbox.com</string>', 'www.totalrpbox.com associated domain', projectPaths.entitlements)
requireText(privacyManifest, 'NSPrivacyAccessedAPICategoryFileTimestamp', 'privacy manifest API category', projectPaths.privacyManifest)
requireText(privacyManifest, 'C617.1', 'privacy manifest reason', projectPaths.privacyManifest)
requireText(pbxproj, `${privacyFileReferenceId} /* PrivacyInfo.xcprivacy */`, 'privacy manifest file reference', projectPaths.pbxproj)
requireText(pbxproj, `${privacyBuildFileId} /* PrivacyInfo.xcprivacy in Resources */`, 'privacy manifest resources reference', projectPaths.pbxproj)
requireText(pbxproj, 'CODE_SIGN_ENTITLEMENTS = App/App.entitlements;', 'code signing entitlements', projectPaths.pbxproj)

if (capacitorConfig) {
  try {
    const config = JSON.parse(capacitorConfig)
    if (!Array.isArray(config.packageClassList) || !config.packageClassList.includes('CAPCameraPlugin')) {
      errors.push(`${displayPath(projectPaths.capacitorConfig)}: missing Capacitor Camera plugin registration`)
    }
  } catch (error) {
    errors.push(`${displayPath(projectPaths.capacitorConfig)}: invalid JSON: ${error instanceof Error ? error.message : String(error)}`)
  }
}

if (pbxproj) {
  const configurationPattern = /XCBuildConfiguration;[\s\S]*?buildSettings = \{([\s\S]*?)\n\s*\};\s*name = ([^;]+);/g
  const configurations = []
  let match
  while ((match = configurationPattern.exec(pbxproj)) !== null) {
    if (match[1].includes('INFOPLIST_FILE = App/Info.plist;')) {
      configurations.push({ name: match[2].trim(), settings: match[1] })
    }
  }

  for (const requiredName of ['Debug', 'Release']) {
    const configuration = configurations.find((entry) => entry.name === requiredName)
    if (!configuration) {
      errors.push(`${displayPath(projectPaths.pbxproj)}: missing ${requiredName} application configuration`)
      continue
    }
    requireSetting(configuration.settings, 'MARKETING_VERSION', expected.version, requiredName)
    requireSetting(configuration.settings, 'CURRENT_PROJECT_VERSION', expected.buildNumber, requiredName)
    requireSetting(configuration.settings, 'DEVELOPMENT_TEAM', expected.teamId, requiredName)
    requireSetting(configuration.settings, 'PRODUCT_BUNDLE_IDENTIFIER', expected.bundleId, requiredName)
    requireSetting(configuration.settings, 'CODE_SIGN_STYLE', 'Manual', requiredName)
    requireSetting(configuration.settings, 'PROVISIONING_PROFILE_SPECIFIER', expected.profileName, requiredName)
    requireSetting(configuration.settings, 'CODE_SIGN_ENTITLEMENTS', 'App/App.entitlements', requiredName)
  }
}

if (errors.length > 0) {
  for (const error of errors) console.error(`[iOS Verify] ${error}`)
  process.exit(1)
}

console.log('[iOS Verify] Generated iOS project is valid')
