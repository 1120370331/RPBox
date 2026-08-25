import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  addCharacterCardPortrait,
  createCharacterCard,
  deleteCharacterCard,
  deleteCharacterCardPortrait,
  getCharacterCard,
  getCharacterCardSources,
  getCharacterCardShare,
  isApprovedPublicCharacterCard,
  listMyCharacterCards,
  publishCharacterCard,
  setCharacterCardPortraitCover,
  type CharacterCardAdditionalInfo,
  type CharacterCardPersonalityTrait,
  updateCharacterCard,
  uploadCharacterCardPortrait,
} from './characterCard'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
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

describe('mobile character-card management API', () => {
  it('normalizes data envelopes for sources and card mutations', async () => {
    requestMock.get.mockResolvedValueOnce({
      data: { sources: [{ backup_id: 8, account_id: 'MAIN', profile_id: 'elia' }] },
    })
    requestMock.post.mockResolvedValueOnce({
      data: { character_card: { id: 61, user_id: 7 } },
    })
    requestMock.put.mockResolvedValueOnce({
      character_card: { id: 61, user_id: 7, display_name: 'Elia' },
    })
    requestMock.post.mockResolvedValueOnce({
      character_card: { id: 61, review_status: 'pending' },
    })

    await expect(getCharacterCardSources()).resolves.toEqual({
      sources: [{ backup_id: 8, account_id: 'MAIN', profile_id: 'elia' }],
    })
    await expect(createCharacterCard({
      source_type: 'backup',
      source_backup_id: 8,
      source_profile_id: 'elia',
    })).resolves.toMatchObject({ id: 61, user_id: 7 })
    await expect(updateCharacterCard(61, { display_name: 'Elia' })).resolves.toMatchObject({
      id: 61,
      display_name: 'Elia',
    })
    await expect(publishCharacterCard(61)).resolves.toMatchObject({ id: 61, review_status: 'pending' })

    expect(requestMock.get).toHaveBeenCalledWith('/character-card-sources')
    expect(requestMock.post).toHaveBeenNthCalledWith(1, '/character-cards', {
      source_type: 'backup',
      source_backup_id: 8,
      source_profile_id: 'elia',
    })
    expect(requestMock.put).toHaveBeenCalledWith('/character-cards/61', { display_name: 'Elia' })
    expect(requestMock.post).toHaveBeenNthCalledWith(2, '/character-cards/61/publish')
  })

  it('uses the pending portrait reference and exact gallery paths', async () => {
    requestMock.post.mockResolvedValueOnce({ data: { portrait_image_ref: ' pending/portrait.webp ' } })
    requestMock.post.mockResolvedValueOnce({ character_card: { id: 61, portraits: [{ id: 91 }] } })
    requestMock.put.mockResolvedValueOnce({ character_card: { id: 61, portraits: [{ id: 91, is_cover: true }] } })
    requestMock.delete.mockResolvedValueOnce({ character_card: { id: 61, portraits: [] } })
    const file = new File(['portrait'], 'portrait.webp', { type: 'image/webp' })

    await expect(uploadCharacterCardPortrait(file)).resolves.toBe('pending/portrait.webp')
    const uploadCall = requestMock.post.mock.calls[0]
    expect(uploadCall[0]).toBe('/upload/character-card-portrait')
    expect(uploadCall[1]).toBeInstanceOf(FormData)
    expect((uploadCall[1] as FormData).get('image')).toBe(file)

    await addCharacterCardPortrait(61, 'pending/portrait.webp')
    await setCharacterCardPortraitCover(61, 91)
    await deleteCharacterCardPortrait(61, 91)
    await deleteCharacterCard(61)

    expect(requestMock.post).toHaveBeenNthCalledWith(2, '/character-cards/61/portraits', {
      image_ref: 'pending/portrait.webp',
    })
    expect(requestMock.put).toHaveBeenCalledWith('/character-cards/61/portraits/91/cover')
    expect(requestMock.delete).toHaveBeenNthCalledWith(1, '/character-cards/61/portraits/91')
    expect(requestMock.delete).toHaveBeenCalledWith('/character-cards/61')
  })
})
