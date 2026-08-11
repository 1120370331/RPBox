import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import CharacterCardPortrait from './CharacterCardPortrait.vue'

i18n.global.locale.value = 'zh-CN'

const mocks = vi.hoisted(() => ({
  getCharacterCardPortraitUrl: vi.fn((card: { id: number }) => `/portrait/${card.id}`),
}))

vi.mock('@/api/characterCard', () => ({
  getCharacterCardPortraitUrl: mocks.getCharacterCardPortraitUrl,
}))

function portraitCard(id: number) {
  return {
    id,
    portrait_image_url: `/api/v1/images/character-card-portrait/${id}`,
    portrait_image_updated_at: `2026-08-10T0${id}:00:00Z`,
    updated_at: '2026-08-10T00:00:00Z',
  }
}

describe('CharacterCardPortrait', () => {
  const fetchMock = vi.fn()
  const createObjectURL = vi.fn()
  const revokeObjectURL = vi.fn()

  beforeEach(() => {
    localStorage.clear()
    fetchMock.mockReset()
    createObjectURL.mockReset()
    revokeObjectURL.mockReset()
    vi.stubGlobal('fetch', fetchMock)
    vi.spyOn(URL, 'createObjectURL').mockImplementation(createObjectURL)
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(revokeObjectURL)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  it('loads portraits with the bearer token and manages object URLs across source changes and unmount', async () => {
    localStorage.setItem('token', 'portrait-token')
    const firstBlob = new Blob(['first'], { type: 'image/png' })
    const secondBlob = new Blob(['second'], { type: 'image/webp' })
    fetchMock
      .mockResolvedValueOnce({ ok: true, blob: vi.fn().mockResolvedValue(firstBlob) })
      .mockResolvedValueOnce({ ok: true, blob: vi.fn().mockResolvedValue(secondBlob) })
    createObjectURL
      .mockReturnValueOnce('blob:portrait-1')
      .mockReturnValueOnce('blob:portrait-2')

    const wrapper = mount(CharacterCardPortrait, {
      props: { card: portraitCard(1), alt: '第一张肖像' },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/portrait/1', expect.objectContaining({
      headers: { Authorization: 'Bearer portrait-token' },
      signal: expect.any(AbortSignal),
    }))
    expect(createObjectURL).toHaveBeenCalledWith(firstBlob)
    expect(wrapper.get('img').attributes('src')).toBe('blob:portrait-1')

    await wrapper.setProps({ card: portraitCard(2) })
    await flushPromises()

    expect(revokeObjectURL).toHaveBeenCalledWith('blob:portrait-1')
    expect(wrapper.get('img').attributes('src')).toBe('blob:portrait-2')

    wrapper.unmount()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:portrait-2')
  })

  it('shows a safe placeholder when the authenticated image request fails', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 404,
      blob: vi.fn(),
    })
    vi.spyOn(console, 'error').mockImplementation(() => {})

    const wrapper = mount(CharacterCardPortrait, {
      props: { card: portraitCard(3), alt: '不可用肖像' },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const fallback = wrapper.get('[data-failed="true"]')
    expect(fallback.attributes('aria-label')).toBe('不可用肖像')
    expect(fallback.text()).toContain('肖像暂不可用')
    expect(wrapper.find('img').exists()).toBe(false)
  })
})
