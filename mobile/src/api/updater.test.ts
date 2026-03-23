import { describe, expect, it } from 'vitest'
import { isNewerVersion, normalizeVersion } from './updater'

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
})
