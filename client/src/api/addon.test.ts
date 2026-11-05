import { describe, expect, it } from 'vitest'
import { addonLatestForFlavor, type AddonManifest } from './addon'

describe('RPBox Addon game version channels', () => {
  const manifest: AddonManifest = {
    name: 'RPBox_Addon',
    latest: '1.0.14',
    latestByFlavor: { _forever_: '1.0.15-beta.1' },
    versions: [],
  }

  it('keeps the Forever beta separate from the retail stable package', () => {
    expect(addonLatestForFlavor(manifest, '_retail_')).toBe('1.0.14')
    expect(addonLatestForFlavor(manifest, '_forever_')).toBe('1.0.15-beta.1')
    expect(addonLatestForFlavor(manifest, '_classic_era_')).toBe('')
  })

  it('does not fall back to a retail addon when the Forever package is missing', () => {
    expect(addonLatestForFlavor({ ...manifest, latestByFlavor: undefined }, '_forever_')).toBe('')
  })
})
