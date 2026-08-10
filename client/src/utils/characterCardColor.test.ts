import { describe, expect, it } from 'vitest'
import {
  normalizeCharacterCardHexForCSS,
  normalizeCharacterCardHexForTRP3,
} from './characterCardColor'

describe('character card TRP3 colors', () => {
  it('normalizes bare TRP3 RGB and ARGB values for CSS display', () => {
    expect(normalizeCharacterCardHexForCSS('6F8CA3')).toBe('#6F8CA3')
    expect(normalizeCharacterCardHexForCSS('806F8CA3')).toBe('#6F8CA380')
    expect(normalizeCharacterCardHexForCSS('#6F8CA380')).toBe('#6F8CA380')
  })

  it('preserves TRP3 semantics when converting CSS colors for write-back', () => {
    expect(normalizeCharacterCardHexForTRP3('#6F8CA3')).toBe('6F8CA3')
    expect(normalizeCharacterCardHexForTRP3('#6F8CA380')).toBe('806F8CA3')
    expect(normalizeCharacterCardHexForTRP3('806F8CA3')).toBe('806F8CA3')
  })
})
