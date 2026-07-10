import { describe, expect, it } from 'vitest'
import {
  getMobileUpdateMode,
  installAndroidUpdate,
  isNewerVersion,
  normalizeVersion,
  resolveIOSUpdateUrl,
  resolveMobileTarget,
  resolveUpdateDownloadUrl,
} from './updater'

describe('updater api helpers', () => {
  it('normalizes version values', () => {
    expect(normalizeVersion('0.1.0')).toBe('0.1.0')
    expect(normalizeVersion('v0.1.0')).toBe('0.1.0')
    expect(normalizeVersion(' V1.2.3 ')).toBe('1.2.3')
  })

  it('handles empty values', () => {
    expect(normalizeVersion('')).toBe('')
    expect(normalizeVersion('   ')).toBe('')
  })

  it('compares semver versions correctly', () => {
    expect(isNewerVersion('1.0.5', '1.0')).toBe(true)
    expect(isNewerVersion('1.0.5', '1.0.4')).toBe(true)
    expect(isNewerVersion('1.0.5', '1.0.5')).toBe(false)
    expect(isNewerVersion('v1.0.5', '1.0.6')).toBe(false)
  })

  it('falls back to string compare when version is non-semver', () => {
    expect(isNewerVersion('mobile-v1.0.6', '1.0.5')).toBe(true)
    expect(isNewerVersion('1.0.5', '1.0.5-build')).toBe(false)
  })

  it('keeps absolute update download URLs unchanged', () => {
    expect(resolveUpdateDownloadUrl('https://example.com/releases/RPBox.apk')).toBe('https://example.com/releases/RPBox.apk')
    expect(resolveUpdateDownloadUrl('http://example.com/releases/RPBox.apk')).toBe('http://example.com/releases/RPBox.apk')
  })

  it('resolves root-relative update download URLs against the current origin when API base is relative', () => {
    expect(resolveUpdateDownloadUrl('/releases/mobile/RPBox.apk')).toBe(
      new URL('/releases/mobile/RPBox.apk', window.location.origin).toString(),
    )
  })

  it('resolves relative update download URLs against the current page when API base is relative', () => {
    expect(resolveUpdateDownloadUrl('releases/mobile/RPBox.apk')).toBe(
      new URL('releases/mobile/RPBox.apk', window.location.href).toString(),
    )
  })

  it('uses the App Store URL scheme for iOS App Store links', () => {
    expect(resolveIOSUpdateUrl('https://apps.apple.com/cn/app/rpbox/id123456789?mt=8')).toBe(
      'itms-apps://apps.apple.com/cn/app/rpbox/id123456789?mt=8',
    )
  })

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
})
