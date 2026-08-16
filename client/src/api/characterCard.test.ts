import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createCharacterCard,
  getCharacterCardSources,
  listMyCharacterCards,
  publishCharacterCard,
  uploadCharacterCardImpressionImage,
  uploadCharacterCardPortrait,
} from './characterCard'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('./request', () => ({
  default: requestMock,
  request: requestMock,
}))

afterEach(() => {
  vi.clearAllMocks()
})

describe('character card API contract', () => {
  it('keeps source summary names separate from backup creation request names', async () => {
    requestMock.get.mockResolvedValue({
      sources: [{
        backup_id: 18,
        account_id: 'ACCOUNT-A',
        profile_id: 'profile-7',
        display_name: '伊莉娅',
      }],
    })
    requestMock.post.mockResolvedValue({ id: 42, display_name: '伊莉娅' })

    const sourceResponse = await getCharacterCardSources()
    const source = sourceResponse.sources[0]
    const card = await createCharacterCard({
      source_type: 'backup',
      source_backup_id: source.backup_id,
      source_profile_id: source.profile_id,
    })

    expect(requestMock.get).toHaveBeenCalledWith('/character-card-sources')
    expect(source).toMatchObject({
      backup_id: 18,
      account_id: 'ACCOUNT-A',
      profile_id: 'profile-7',
    })
    expect(requestMock.post).toHaveBeenCalledWith('/character-cards', {
      source_type: 'backup',
      source_backup_id: 18,
      source_profile_id: 'profile-7',
    })
    expect(card.id).toBe(42)
  })

  it('accepts the existing data envelope without changing the list result shape', async () => {
    requestMock.get.mockResolvedValue({ data: { character_cards: [{ id: 9 }] } })

    const response = await listMyCharacterCards()

    expect(response.character_cards).toEqual([{ id: 9 }])
  })

  it('submits the explicitly published character-card snapshot through its own endpoint', async () => {
    requestMock.post.mockResolvedValue({ data: { character_card: { id: 42, review_status: 'pending' } } })

    const published = await publishCharacterCard(42)

    expect(requestMock.post).toHaveBeenCalledWith('/character-cards/42/publish')
    expect(published).toMatchObject({ id: 42, review_status: 'pending' })
  })

  it('uploads portraits through the protected character-card endpoint', async () => {
    requestMock.post.mockResolvedValue({ portrait_image_ref: 'character-card-pending://portrait-token' })
    const file = new File(['portrait'], 'portrait.webp', { type: 'image/webp' })

    const url = await uploadCharacterCardPortrait(file)

    expect(url).toBe('character-card-pending://portrait-token')
    expect(requestMock.post).toHaveBeenCalledWith(
      '/upload/character-card-portrait',
      expect.any(FormData),
      { headers: { 'Content-Type': 'multipart/form-data' } },
    )
    const formData = requestMock.post.mock.calls[0][1] as FormData
    expect(formData.get('image')).toBe(file)
  })

  it('uploads impression media with an explicit image kind', async () => {
    requestMock.post.mockResolvedValue({ data: { image_ref: 'character-card-impression-pending://icon-token' } })
    const file = new File(['icon'], 'glance-icon.png', { type: 'image/png' })

    const imageRef = await uploadCharacterCardImpressionImage(file, 'icon')

    expect(imageRef).toBe('character-card-impression-pending://icon-token')
    expect(requestMock.post).toHaveBeenCalledWith(
      '/upload/character-card-impression-image',
      expect.any(FormData),
      { headers: { 'Content-Type': 'multipart/form-data' } },
    )
    const formData = requestMock.post.mock.calls[0][1] as FormData
    expect(formData.get('image')).toBe(file)
    expect(formData.get('kind')).toBe('icon')
  })
})
