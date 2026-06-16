import { describe, expect, it } from 'vitest'
import {
  isNewerVersion,
  normalizeVersion,
  resolveIOSUpdateUrl,
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
})
