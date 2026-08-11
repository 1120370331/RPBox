import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import CharacterCardWall from './CharacterCardWall.vue'
import type { CharacterCardSummary } from '@/api/characterCard'

i18n.global.locale.value = 'zh-CN'

const mocks = vi.hoisted(() => ({
  listMyCharacterCards: vi.fn(),
  listUserCharacterCards: vi.fn(),
}))

vi.mock('@/api/characterCard', () => ({
  listMyCharacterCards: mocks.listMyCharacterCards,
  listUserCharacterCards: mocks.listUserCharacterCards,
}))

const RouterLinkStub = defineComponent({
  props: { to: { type: String, required: true } },
  template: '<a :href="to"><slot /></a>',
})

const publicCard: CharacterCardSummary = {
  id: 51,
  user_id: 8,
  first_name: '米拉',
  last_name: '铜枝',
  display_name: '米拉·铜枝',
  title: '远行制图师',
  full_title: '',
  race: '矮人',
  class: '猎人',
  eye_color: '',
  eye_color_hex: '',
  age: '',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  relationship_status: '',
  icon: '',
  name_color: '',
  summary: '记录荒野道路与失落营地。',
  portrait_image_url: '',
  status: 'published',
  visibility: 'public',
  created_at: '2026-08-10T00:00:00Z',
  updated_at: '2026-08-10T00:00:00Z',
}

function mountWall(isOwnProfile: boolean) {
  return mount(CharacterCardWall, {
    props: { userId: 8, isOwnProfile },
    global: {
      plugins: [i18n],
      stubs: {
        RouterLink: RouterLinkStub,
        CharacterCardPortrait: true,
      },
    },
  })
}

afterEach(() => {
  vi.clearAllMocks()
})

describe('CharacterCardWall', () => {
  it('keeps the management empty state visible on the owner profile', async () => {
    mocks.listMyCharacterCards.mockResolvedValue({ character_cards: [] })

    const wrapper = mountWall(true)
    await flushPromises()

    expect(mocks.listMyCharacterCards).toHaveBeenCalledOnce()
    expect(wrapper.find('.character-wall').exists()).toBe(true)
    expect(wrapper.text()).toContain('为你的第一个角色留下正式档案')
    expect(wrapper.get('a[href="/character-cards/new"]').exists()).toBe(true)
  })

  it('does not expose an empty management section on another user profile', async () => {
    mocks.listUserCharacterCards.mockResolvedValue({ character_cards: [] })

    const wrapper = mountWall(false)
    await flushPromises()

    expect(mocks.listUserCharacterCards).toHaveBeenCalledWith(8)
    expect(wrapper.find('.character-wall').exists()).toBe(false)
  })

  it('renders the public portrait wall for a profile visitor without owner controls', async () => {
    mocks.listUserCharacterCards.mockResolvedValue({ character_cards: [publicCard] })

    const wrapper = mountWall(false)
    await flushPromises()

    expect(wrapper.text()).toContain('米拉·铜枝')
    expect(wrapper.text()).toContain('远行制图师')
    expect(wrapper.get('a[href="/character-cards/51"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/character-cards/new"]').exists()).toBe(false)
  })
})
