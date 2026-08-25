import { describe, expect, it } from 'vitest'
import type { CharacterCard, CharacterCardImpression } from '@/api/characterCard'
import {
  createCharacterCardDraft,
  createEmptyCharacterCardImpressions,
  getCharacterCardDisplayName,
  normalizeCharacterCardImpressions,
} from './characterCardDraft'

function impression(slot: number, title: string): CharacterCardImpression {
  return {
    slot,
    active: true,
    title,
    text: `${title} text`,
    trp3_icon: '',
    icon_image_url: '',
    icon_image_updated_at: null,
    image_url: '',
    image_updated_at: null,
  }
}

describe('mobile character-card drafts', () => {
  it('always builds five independent, ordered impression slots', () => {
    const empty = createEmptyCharacterCardImpressions()
    const normalized = normalizeCharacterCardImpressions([
      impression(4, 'Fourth'),
      impression(1, 'First'),
      impression(8, 'Out of range'),
      impression(1, 'Duplicate'),
    ])

    expect(empty.map((item) => item.slot)).toEqual([1, 2, 3, 4, 5])
    empty[0].title = 'Only first'
    expect(empty[1].title).toBe('')
    expect(normalized.map((item) => item.slot)).toEqual([1, 2, 3, 4, 5])
    expect(normalized[0].title).toBe('First')
    expect(normalized[3].title).toBe('Fourth')
    expect(normalized.some((item) => item.title === 'Out of range')).toBe(false)
  })

  it('normalizes imported trait rows and preserves the complete cloud draft', () => {
    const card = {
      id: 42,
      user_id: 7,
      first_name: 'Elia',
      last_name: 'Moonwhisper',
      display_name: '',
      title: 'Priestess',
      full_title: 'Priestess of Elune',
      race: 'Night Elf',
      class: 'Priest',
      eye_color: 'Silver',
      eye_color_hex: '#c9d5e7',
      age: '320',
      height: 'Tall',
      weight: 'Lean',
      birthplace: 'Suramar',
      residence: 'Darnassus',
      relationship_status: '1',
      icon: 'INV_Misc_Note_01',
      class_color: '#ffffff',
      name_color: '#eeeeee',
      additional_info: [{ id: 99, name: 'Pronouns', value: 'she/her', icon: '' }],
      personality_traits: [{
        preset_id: 99,
        left_text: 'Reserved',
        right_text: 'Expressive',
        left_icon: '',
        right_icon: '',
        left_color: { r: -1, g: 0.5, b: 4 },
        right_color: null,
        value: 31,
      }],
      summary: 'A moonlit traveler.',
      background_story: '<p>Background</p>',
      first_impression: '<p>Notes</p>',
      impressions: [impression(2, 'Silver eyes')],
      other_content: '<p>Other</p>',
      portrait_image_url: '/portrait',
      portraits: [],
      status: 'published',
      visibility: 'public',
      review_status: 'pending',
      sort_order: 3,
      updated_at: '2026-08-25T00:00:00Z',
    } satisfies CharacterCard

    const draft = createCharacterCardDraft(card)

    expect(draft).toMatchObject({
      first_name: 'Elia',
      background_story: '<p>Background</p>',
      other_content: '<p>Other</p>',
      portrait_image_url: '/portrait',
      status: 'published',
      visibility: 'public',
      sort_order: 3,
    })
    expect(draft.additional_info[0]).toMatchObject({ id: 1, name: 'Pronouns' })
    expect(draft.personality_traits[0]).toMatchObject({ preset_id: null, value: 20 })
    expect(draft.personality_traits[0].left_color).toEqual({ r: 0, g: 0.5, b: 1 })
    expect(draft.impressions).toHaveLength(5)
    expect(getCharacterCardDisplayName(draft)).toBe('Elia Moonwhisper')
    expect(getCharacterCardDisplayName({
      display_name: '',
      first_name: '',
      last_name: '',
    }, 'Unnamed character')).toBe('Unnamed character')
  })
})
