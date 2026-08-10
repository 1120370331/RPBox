import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import CharacterCardDetail from './CharacterCardDetail.vue'
import type { CharacterCard } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  getCharacterCard: vi.fn(),
  updateCharacterCard: vi.fn(),
  deleteCharacterCard: vi.fn(),
}))

vi.mock('@/api/characterCard', () => ({
  getCharacterCard: mocks.getCharacterCard,
  updateCharacterCard: mocks.updateCharacterCard,
  deleteCharacterCard: mocks.deleteCharacterCard,
  getCharacterCardPortraitUrl: vi.fn(() => ''),
}))

vi.mock('@/utils/jumpLink', () => ({
  hydrateJumpCards: vi.fn(),
  sanitizeJumpLinks: vi.fn(),
}))

const card: CharacterCard = {
  id: 12,
  user_id: 3,
  first_name: '伊莉娅',
  last_name: '星语',
  display_name: '伊莉娅·星语',
  title: '月之女祭司',
  full_title: '',
  race: '暗夜精灵',
  class: '牧师',
  eye_color: '银色',
  eye_color_hex: 'C9D5E7',
  age: '',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  relationship_status: '',
  icon: '',
  name_color: '80C9D5E7',
  summary: '',
  background_story: '<p>背景</p>',
  first_impression: '<p>印象</p>',
  impressions: [
    {
      slot: 1,
      active: true,
      title: '草药香',
      text: '袖口留着晒干草叶的气息。',
      trp3_icon: 'INV_Misc_Herb_19',
      icon_image_url: '',
      icon_image_updated_at: null,
      image_url: '/api/v1/images/character-card-impression-image/12-1?v=1',
      image_updated_at: '2026-08-10T08:00:00Z',
    },
    {
      slot: 2,
      active: false,
      title: '不会公开的观察',
      text: '禁用槽不应显示。',
      trp3_icon: '',
      icon_image_url: '',
      icon_image_updated_at: null,
      image_url: '',
      image_updated_at: null,
    },
    {
      slot: 3,
      active: true,
      title: '银色眼眸',
      text: '目光总是先落在门窗与退路上。',
      trp3_icon: '',
      icon_image_url: '/api/v1/images/character-card-impression-icon/12-3?v=2',
      icon_image_updated_at: '2026-08-10T08:00:00Z',
      image_url: '',
      image_updated_at: null,
    },
  ],
  other_content: '<p>其他</p>',
  portrait_image_url: '',
  status: 'published',
  visibility: 'public',
  created_at: '2026-08-10T08:00:00Z',
  updated_at: '2026-08-10T08:00:00Z',
}

afterEach(() => {
  vi.clearAllMocks()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
  document.body.innerHTML = ''
})

describe('CharacterCardDetail tabs', () => {
  it('keeps tab focus, selection, and panel relationships synchronized', async () => {
    mocks.getCharacterCard.mockResolvedValue({
      ...card,
      impressions: card.impressions.map((impression) => ({
        ...impression,
        icon_image_url: '',
        image_url: '',
      })),
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id', component: CharacterCardDetail }],
    })
    await router.push('/character-cards/12')
    await router.isReady()

    const wrapper = mount(CharacterCardDetail, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router],
        stubs: { CharacterCardPortrait: true },
      },
    })
    await flushPromises()

    const tabs = wrapper.findAll<HTMLButtonElement>('.detail-tabs [role="tab"]')
    const basic = tabs[0]
    const background = tabs[1]
    const other = tabs[3]
    basic.element.focus()

    await basic.trigger('keydown', { key: 'ArrowRight' })
    await wrapper.vm.$nextTick()
    expect(background.attributes('aria-selected')).toBe('true')
    expect(background.attributes('tabindex')).toBe('0')
    expect(background.attributes('aria-controls')).toBe('character-detail-panel-background')
    expect(document.activeElement).toBe(background.element)
    expect(wrapper.get('#character-detail-panel-background').attributes('aria-labelledby'))
      .toBe('character-detail-tab-background')

    await background.trigger('keydown', { key: 'End' })
    await wrapper.vm.$nextTick()
    expect(other.attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(other.element)

    await other.trigger('keydown', { key: 'Home' })
    await wrapper.vm.$nextTick()
    expect(basic.attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(basic.element)
    wrapper.unmount()
  })

  it('shows only enabled observation slots and keeps legacy notes as other notes', async () => {
    localStorage.setItem('token', 'detail-private-token')
    const privateCard: CharacterCard = { ...card, status: 'draft', visibility: 'private' }
    mocks.getCharacterCard.mockResolvedValue(privateCard)
    const fetchMock = vi.fn().mockImplementation(async () => ({
      ok: true,
      blob: vi.fn().mockResolvedValue(new Blob(['protected-image'], { type: 'image/png' })),
    }))
    let objectUrlIndex = 0
    vi.stubGlobal('fetch', fetchMock)
    vi.spyOn(URL, 'createObjectURL').mockImplementation(() => `blob:detail-protected-${++objectUrlIndex}`)
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id', component: CharacterCardDetail }],
    })
    await router.push('/character-cards/12')
    await router.isReady()

    const wrapper = mount(CharacterCardDetail, {
      global: {
        plugins: [createPinia(), router],
        stubs: { CharacterCardPortrait: true },
      },
    })
    await flushPromises()

    const impressionTab = wrapper.findAll<HTMLButtonElement>('.detail-tabs button')
      .find((button) => button.text().includes('第一印象'))!
    await impressionTab.trigger('click')

    const entries = wrapper.findAll('.impression-entry')
    expect(entries).toHaveLength(2)
    expect(wrapper.text()).toContain('草药香')
    expect(wrapper.text()).toContain('银色眼眸')
    expect(wrapper.text()).not.toContain('不会公开的观察')
    expect(wrapper.text()).toContain('其他备注')
    expect(wrapper.get('.impression-supplement').html()).toContain('印象')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/images/character-card-impression-image/12-1?v=1',
      expect.objectContaining({ headers: { Authorization: 'Bearer detail-private-token' } }),
    )
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/images/character-card-impression-icon/12-3?v=2',
      expect.objectContaining({ headers: { Authorization: 'Bearer detail-private-token' } }),
    )
    expect(wrapper.get('.impression-entry__protected-image img').attributes('src'))
      .toMatch(/^blob:detail-protected-/)
    wrapper.unmount()
  })
})
