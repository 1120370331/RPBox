import { describe, expect, it } from 'vitest'
import { isFeatureEnabled, mobileFeatures } from './mobileFeatures'

describe('mobileFeatures', () => {
  it('keeps only mobile-available feature entries', () => {
    expect(mobileFeatures).toEqual([
      'community',
      'stories',
      'market',
      'profiles',
      'profile',
    ])
  })

  it('does not enable local-client-only feature flags', () => {
    expect(isFeatureEnabled('sync')).toBe(false)
    expect(isFeatureEnabled('chat-logger')).toBe(false)
    expect(isFeatureEnabled('wow-export')).toBe(false)
  })
})
