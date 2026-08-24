import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

import {
  IOS_APP_STORE_ID,
  IOS_DEFAULT_API_BASE,
  IOS_DEFAULT_PUBLIC_URL,
  IOS_USAGE_DESCRIPTIONS,
  validateIosApiBase,
  validateIosPublicUpdateUrl,
} from './iosCompliance.mjs'

test('accepts the verified production API base and other safe HTTPS API bases', () => {
  assert.equal(validateIosApiBase(IOS_DEFAULT_API_BASE), IOS_DEFAULT_API_BASE)
  assert.equal(validateIosApiBase('https://api.example.com/api/v1/'), 'https://api.example.com/api/v1/')
})

test('rejects unsafe iOS API bases', () => {
  const invalidUrls = [
    'http://api.example.com/api/v1',
    'https://localhost/api/v1',
    'https://sub.localhost/api/v1',
    'https://127.0.0.1/api/v1',
    'https://127.1/api/v1',
    'https://[::1]/api/v1',
    'https://user:password@api.example.com/api/v1',
    'https://api.example.com/api/v1?environment=staging',
    'https://api.example.com/api/v1#staging',
    ' https://api.example.com/api/v1',
  ]

  for (const value of invalidUrls) {
    assert.throws(() => validateIosApiBase(value), { name: 'Error' }, value)
  }
})

test('accepts canonical localized and global App Store URLs', () => {
  assert.equal(validateIosPublicUpdateUrl(IOS_DEFAULT_PUBLIC_URL), IOS_DEFAULT_PUBLIC_URL)
  assert.equal(
    validateIosPublicUpdateUrl('https://apps.apple.com/app/rpbox/id6761112311?mt=8'),
    'https://apps.apple.com/app/rpbox/id6761112311?mt=8',
  )
})

test('rejects non-public, malformed, and placeholder update URLs', () => {
  const invalidUrls = [
    'https://testflight.apple.com/join/ABC123',
    'https://apps.apple.com/cn/app/rpbox/id1234567890',
    'https://apps.apple.com/cn/app/rpbox/id9876543210',
    'http://apps.apple.com/cn/app/rpbox/id6761112311',
    'https://localhost/cn/app/rpbox/id6761112311',
    'https://example.com/cn/app/rpbox/id6761112311',
    'https://apps.apple.com/?id=6761112311',
    'https://apps.apple.com/cn/app/rpbox/idnotnumeric',
    'https://apps.apple.com/not-a-country/app/rpbox/id6761112311',
    'https://apps.apple.com/cn/app/category/rpbox/id6761112311',
    'https://user:password@apps.apple.com/cn/app/rpbox/id6761112311',
    'https://apps.apple.com:8443/cn/app/rpbox/id6761112311',
    ' https://apps.apple.com/cn/app/rpbox/id6761112311',
  ]

  for (const value of invalidUrls) {
    assert.throws(() => validateIosPublicUpdateUrl(value), { name: 'Error' }, value)
  }
})

test('tracked Info.plist uses the shared bilingual purpose strings', () => {
  const infoPlist = fs.readFileSync(new URL('../ios/App/App/Info.plist', import.meta.url), 'utf8')

  for (const [key, description] of Object.entries(IOS_USAGE_DESCRIPTIONS)) {
    assert.match(description, /[\u3400-\u9fff]/, `${key} must contain Chinese guidance`)
    assert.match(description, /[A-Za-z]/, `${key} must contain English guidance`)
    assert.ok(
      infoPlist.includes(`<key>${key}</key>`) && infoPlist.includes(`<string>${description}</string>`),
      `${key} must match the shared compliance source`,
    )
  }
  assert.match(
    infoPlist,
    /<key>ITSAppUsesNonExemptEncryption<\/key>\s*<false\s*\/>/,
    'App Store export-compliance declaration must be false',
  )
})

test('native preparation, verification, and release workflow use the shared compliance source', () => {
  const prepareSource = fs.readFileSync(new URL('./prepareNativeShare.mjs', import.meta.url), 'utf8')
  const verifierSource = fs.readFileSync(new URL('./verifyIosProject.mjs', import.meta.url), 'utf8')
  const workflow = fs.readFileSync(new URL('../../.github/workflows/release-ios-testflight.yml', import.meta.url), 'utf8')

  assert.match(prepareSource, /import \{ IOS_USAGE_DESCRIPTIONS \} from '\.\/iosCompliance\.mjs'/)
  assert.match(verifierSource, /validateIosApiBase/)
  assert.ok(workflow.includes(`IOS_DEFAULT_API_BASE: '${IOS_DEFAULT_API_BASE}'`))
  assert.ok(workflow.includes(`IOS_APP_STORE_ID: '${IOS_APP_STORE_ID}'`))
  assert.ok(workflow.includes(`IOS_DEFAULT_PUBLIC_URL: '${IOS_DEFAULT_PUBLIC_URL}'`))
  assert.match(workflow, /iosCompliance\.mjs validate-api-base/)
  assert.match(workflow, /iosCompliance\.mjs validate-public-update-url/)
  assert.doesNotMatch(workflow, /PlistBuddy.*NS(?:Camera|PhotoLibrary)UsageDescription/)
})

test('release workflow fails closed and keeps TestFlight separate from public metadata', () => {
  const workflow = fs.readFileSync(new URL('../../.github/workflows/release-ios-testflight.yml', import.meta.url), 'utf8')

  assert.match(workflow, /Requested iOS version must exactly match mobile\/package\.json/)
  assert.match(workflow, /Release notes file is missing or empty/)
  assert.match(workflow, /name: Preflight production API/)
  assert.match(workflow, /IOS_API_HEALTH_URL/)
  assert.match(workflow, /IOS_API_DATABASE_URL/)
  assert.match(workflow, /--skip_waiting_for_build_processing false/)
  assert.match(workflow, /--changelog "\$RELEASE_NOTES"/)
  assert.doesNotMatch(workflow, /--skip_waiting_for_build_processing true/)
  assert.match(workflow, /deploy_public_metadata:[\s\S]*?type: boolean[\s\S]*?default: false/)
  assert.match(
    workflow,
    /build-and-upload:[\s\S]*?if: \$\{\{ github\.event_name != 'workflow_dispatch' \|\| !inputs\.deploy_public_metadata \}\}/,
  )
  assert.match(
    workflow,
    /deploy-ios-metadata:[\s\S]*?if: \$\{\{ github\.event_name == 'workflow_dispatch' && inputs\.deploy_public_metadata \}\}/,
  )
  const metadataJobHeader = workflow.match(/deploy-ios-metadata:[\s\S]*?steps:/)?.[0] || ''
  assert.doesNotMatch(metadataJobHeader, /\n\s+needs:/, 'metadata-only mode must not depend on a TestFlight upload')
  assert.match(workflow, /itunes\.apple\.com\/lookup\?id=\$\{IOS_APP_STORE_ID\}&country=cn/)
  assert.match(workflow, /SERVER_PORT: \$\{\{ secrets\.SERVER_PORT \}\}/)
  assert.match(workflow, /PORT="\$\{SERVER_PORT:-2233\}"/)
})

test('release workflow verifies the signed exported IPA before upload', () => {
  const workflow = fs.readFileSync(new URL('../../.github/workflows/release-ios-testflight.yml', import.meta.url), 'utf8')

  assert.match(workflow, /name: Verify exported IPA/)
  assert.match(workflow, /CFBundleIdentifier/)
  assert.match(workflow, /CFBundleShortVersionString/)
  assert.match(workflow, /CFBundleVersion/)
  assert.match(workflow, /PrivacyInfo\.xcprivacy/)
  assert.match(workflow, /codesign --verify --deep --strict/)
  assert.match(workflow, /codesign -d --entitlements :-/)
  assert.match(workflow, /ITSAppUsesNonExemptEncryption/)
})
