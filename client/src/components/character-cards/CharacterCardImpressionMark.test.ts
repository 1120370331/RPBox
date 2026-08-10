import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import CharacterCardImpressionMark from './CharacterCardImpressionMark.vue'

describe('CharacterCardImpressionMark', () => {
  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('falls back from a custom icon to the internal TRP3 icon proxy and then to a default mark', async () => {
    localStorage.setItem('token', 'impression-token')
    const iconBlob = new Blob(['custom-icon'], { type: 'image/png' })
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, blob: vi.fn().mockResolvedValue(iconBlob) })
    vi.stubGlobal('fetch', fetchMock)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:protected-custom-icon')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})

    const wrapper = mount(CharacterCardImpressionMark, {
      props: {
        iconImageUrl: '/api/v1/images/character-card-impression-icon/12-3?v=1',
        trp3Icon: 'Interface\\Icons\\INV_Misc_Herb_19.blp',
        fallbackLabel: '3',
      },
    })
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/images/character-card-impression-icon/12-3?v=1',
      expect.objectContaining({ headers: { Authorization: 'Bearer impression-token' } }),
    )
    const customIcon = wrapper.get<HTMLImageElement>('.impression-mark__custom img')
    expect(customIcon.attributes('src')).toBe('blob:protected-custom-icon')
    await customIcon.trigger('error')

    const trp3Icon = wrapper.get<HTMLImageElement>('.wow-icon img')
    expect(trp3Icon.attributes('src')).toBe('http://localhost:8080/api/v1/icons/inv_misc_herb_19')
    await trp3Icon.trigger('error')

    expect(wrapper.get('.fallback').text()).toBe('3')
    expect(wrapper.get('.impression-mark').attributes('aria-label'))
      .toBe('TRP3 图标：Interface\\Icons\\INV_Misc_Herb_19.blp')
    wrapper.unmount()
  })
})
