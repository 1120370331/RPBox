import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  getCharacterCard,
  getCharacterCardShare,
  isApprovedPublicCharacterCard,
  listMyCharacterCards,
  type CharacterCardAdditionalInfo,
  type CharacterCardPersonalityTrait,
} from './characterCard'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@shared/api/request', () => ({
  request: requestMock,
}))

afterEach(() => {
  vi.clearAllMocks()
})

describe('mobile character card sharing API', () => {
  it('loads the server-owned public share metadata', async () => {
    requestMock.get.mockResolvedValue({
      path: '/character-cards/42',
      title: '伊莉娅·星语',
      summary: '月光下的远行者',
      updated_at: '2026-08-10T08:00:00Z',
    })

    await expect(getCharacterCardShare(42)).resolves.toMatchObject({
      path: '/character-cards/42',
      title: '伊莉娅·星语',
      summary: '月光下的远行者',
    })
    expect(requestMock.get).toHaveBeenCalledWith('/character-cards/42/share')
  })

  it('rejects malformed metadata without a public path', async () => {
    requestMock.get.mockResolvedValue({
      path: '',
      title: '伊莉娅·星语',
      summary: '',
      updated_at: '2026-08-10T08:00:00Z',
    })

    await expect(getCharacterCardShare(42)).rejects.toThrow('Missing character card share path')
  })
})

describe('mobile character card quick-jump API', () => {
  it('loads the current user character-card list', async () => {
    requestMock.get.mockResolvedValue({ character_cards: [{ id: 27 }] })

    await expect(listMyCharacterCards()).resolves.toEqual({ character_cards: [{ id: 27 }] })
    expect(requestMock.get).toHaveBeenCalledWith('/character-cards')
  })

  it('accepts only approved, published, public cards for embeds', () => {
    const publicState = {
      status: 'published' as const,
      visibility: 'public' as const,
      review_status: 'approved' as const,
    }

    expect(isApprovedPublicCharacterCard(publicState)).toBe(true)
    expect(isApprovedPublicCharacterCard({ ...publicState, visibility: 'private' })).toBe(false)
    expect(isApprovedPublicCharacterCard({ ...publicState, status: 'draft' })).toBe(false)
    expect(isApprovedPublicCharacterCard({ ...publicState, review_status: 'pending' })).toBe(false)
  })
})

describe('mobile character card approved DTO fields', () => {
  it('preserves exact additional-info and personality-trait payloads', async () => {
    const additionalInfo: CharacterCardAdditionalInfo = {
      id: 7,
      name: 'Pronouns',
      value: 'she / her',
      icon: 'INV_Misc_Note_01',
    }
    const personalityTrait: CharacterCardPersonalityTrait = {
      preset_id: null,
      left_text: 'Reserved',
      right_text: 'Expressive',
      left_icon: 'Ability_Stealth',
      right_icon: 'Spell_Holy_WordFortitude',
      left_color: { r: 0.2, g: 0.3, b: 0.4 },
      right_color: { r: 0.8, g: 0.7, b: 0.6 },
      value: 14,
    }
    requestMock.get.mockResolvedValue({
      character_card: {
        id: 42,
        additional_info: [additionalInfo],
        personality_traits: [personalityTrait],
      },
    })

    const card = await getCharacterCard(42)

    expect(requestMock.get).toHaveBeenCalledWith('/character-cards/42')
    expect(card.additional_info).toEqual([additionalInfo])
    expect(card.personality_traits).toEqual([personalityTrait])
  })
})
