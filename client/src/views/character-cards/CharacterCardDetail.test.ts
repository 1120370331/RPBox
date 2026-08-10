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
  other_content: '<p>其他</p>',
  portrait_image_url: '',
  status: 'published',
  visibility: 'public',
  created_at: '2026-08-10T08:00:00Z',
  updated_at: '2026-08-10T08:00:00Z',
}

afterEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  document.body.innerHTML = ''
})

describe('CharacterCardDetail tabs', () => {
  it('keeps tab focus, selection, and panel relationships synchronized', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
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
})
