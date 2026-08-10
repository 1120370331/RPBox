import { describe, expect, it } from 'vitest'
import type { CharacterCardImpression } from '@/api/characterCard'
import {
  createEmptyCharacterCardImpressions,
  normalizeCharacterCardImpressions,
} from './characterCardDraft'

function impression(slot: number, title: string): CharacterCardImpression {
  return {
    slot,
    active: true,
    title,
    text: `${title}的描述`,
    trp3_icon: '',
    icon_image_url: '',
    icon_image_updated_at: null,
    image_url: '',
    image_updated_at: null,
  }
}

describe('character card impression drafts', () => {
  it('creates five independent, inactive TRP3-compatible slots', () => {
    const slots = createEmptyCharacterCardImpressions()
    const legacySlots = normalizeCharacterCardImpressions(undefined)

    expect(slots).toHaveLength(5)
    expect(legacySlots).toHaveLength(5)
    expect(slots.map((item) => item.slot)).toEqual([1, 2, 3, 4, 5])
    expect(slots.every((item) => item.active === false)).toBe(true)
    slots[0].title = '只改第一槽'
    expect(slots[1].title).toBe('')
  })

  it('sorts, sanitizes, and fills sparse server data without exceeding five slots', () => {
    const normalized = normalizeCharacterCardImpressions([
      impression(4, '第四槽'),
      impression(1, '第一槽'),
      impression(8, '越界槽'),
      impression(1, '重复槽'),
    ])

    expect(normalized).toHaveLength(5)
    expect(normalized.map((item) => item.slot)).toEqual([1, 2, 3, 4, 5])
    expect(normalized[0].title).toBe('第一槽')
    expect(normalized[1]).toMatchObject({ active: false, title: '' })
    expect(normalized[3].title).toBe('第四槽')
    expect(normalized.some((item) => item.title === '越界槽')).toBe(false)
    expect(normalized.some((item) => item.title === '重复槽')).toBe(false)
  })
})
