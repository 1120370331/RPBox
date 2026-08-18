import { describe, expect, it } from 'vitest'
import { isAddonVersionNewer } from './addonVersion'

describe('isAddonVersionNewer', () => {
  it('does not offer a downgrade as an update', () => {
    expect(isAddonVersionNewer('3.3.6', '3.4.1')).toBe(false)
  })

  it('recognizes a newer release, including a v prefix', () => {
    expect(isAddonVersionNewer('v3.4.1', '3.3.6')).toBe(true)
  })

  it('orders prereleases below the matching stable release', () => {
    expect(isAddonVersionNewer('1.0.0', '1.0.0-beta.2')).toBe(true)
    expect(isAddonVersionNewer('1.0.0-beta.2', '1.0.0')).toBe(false)
  })

  it('does not guess when the remote version is not comparable', () => {
    expect(isAddonVersionNewer('latest', '3.4.1')).toBe(false)
  })
})
