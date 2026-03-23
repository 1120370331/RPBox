import { describe, expect, it } from 'vitest'
import { normalizeVersion } from './updater'

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
})

