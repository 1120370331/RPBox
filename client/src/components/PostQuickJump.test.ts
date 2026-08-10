import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PostQuickJump from './PostQuickJump.vue'
import type { RPDBWork } from '@/api/rpdb'
import type { CharacterCardSummary } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  listGuilds: vi.fn(),
  listGuildStories: vi.fn(),
  listPosts: vi.fn(),
  listRPDBWorks: vi.fn(),
  listCharacterCards: vi.fn(),
}))

vi.mock('@/api/guild', () => ({
  listGuilds: mocks.listGuilds,
  listGuildStories: mocks.listGuildStories,
}))

vi.mock('@/api/post', () => ({
  listPosts: mocks.listPosts,
  POST_CATEGORIES: [],
}))

vi.mock('@/api/item', () => ({
  getImageUrl: vi.fn(),
  resolveApiUrl: (value?: string) => value || '',
}))

vi.mock('@/api/rpdb', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/api/rpdb')>()
  return {
    ...original,
    listRPDBWorks: mocks.listRPDBWorks,
    resolveRPDBMediaURL: (value?: string) => value ? `resolved:${value}` : '',
  }
})

vi.mock('@/api/characterCard', () => ({
  listMyCharacterCards: mocks.listCharacterCards,
  getCharacterCardPortraitUrl: (card: CharacterCardSummary) => card.portrait_image_url
    ? `/api/v1/images/character-card-portrait/${card.id}`
    : '',
}))

const baseWork: RPDBWork = {
  id: 1,
  author_id: 8,
  author_name: '织雾者',
  type: 'item_showcase',
  title: '月光灯笼',
  slug: 'moon-lantern',
  summary: '适合夜间巡逻场景',
  content: '',
  content_type: 'html',
  cover_image: '/uploads/lantern.jpg',
  rp_use_cases: '',
  effect_description: '',
  restrictions: '',
  extra: '',
  game_version: '',
  expansion: '',
  availability_status: '',
  bind_type: '',
  faction: '',
  armor_type: '',
  verification_status: 'verified',
  verified_count: 0,
  outdated_count: 0,
  status: 'published',
  is_public: true,
  visibility: 'public',
  review_status: 'approved',
  version: 1,
  view_count: 120,
  like_count: 12,
  favorite_count: 7,
  comment_count: 0,
  list_count: 4,
  media_count: 1,
  is_liked: false,
  is_favorited: false,
  in_collection_list: false,
  created_at: '2026-07-13T00:00:00Z',
  updated_at: '2026-07-13T00:00:00Z',
}

const baseCharacterCard: CharacterCardSummary = {
  id: 27,
  user_id: 8,
  first_name: '伊莉娅',
  last_name: '星语',
  display_name: '伊莉娅·星语',
  title: '月之女祭司',
  full_title: '',
  race: '暗夜精灵',
  class: '牧师',
  eye_color: '银色',
  eye_color_hex: '#C9D5E7',
  age: '',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  relationship_status: '',
  icon: '',
  name_color: '',
  summary: '在月神殿保存远行者的旧信。',
  portrait_image_url: '/api/v1/images/character-card-portrait/27',
  portrait_image_updated_at: '2026-08-10T08:00:00Z',
  status: 'published',
  visibility: 'public',
  created_at: '2026-08-10T08:00:00Z',
  updated_at: '2026-08-10T08:00:00Z',
}

let wrapper: VueWrapper | null = null

beforeEach(() => {
  mocks.listCharacterCards.mockResolvedValue({ character_cards: [] })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

describe('PostQuickJump RPDB works', () => {
  it('lists all three categories and inserts the selected work as an RPDB card', async () => {
    const works: RPDBWork[] = [
      baseWork,
      { ...baseWork, id: 2, type: 'transmog', title: '银色守望者' },
      { ...baseWork, id: 3, type: 'home_showcase', title: '港湾小屋' },
    ]
    mocks.listGuilds.mockResolvedValue({ guilds: [] })
    mocks.listPosts.mockResolvedValue({ posts: [], total: 0 })
    mocks.listRPDBWorks.mockResolvedValue({ works, total: 3, page: 1, page_size: 12 })
    const onInsert = vi.fn()

    wrapper = mount(PostQuickJump, {
      attachTo: document.body,
      props: { modelValue: false, onInsert },
    })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const tab = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.quick-jump__tabs button'))
      .find((button) => button.textContent?.includes('RP数据库'))
    expect(tab).toBeTruthy()
    tab?.click()
    await flushPromises()

    expect(document.body.querySelector('.jump-item--rpdb-item')).toBeTruthy()
    expect(document.body.querySelector('.jump-item--rpdb-transmog')).toBeTruthy()
    expect(document.body.querySelector('.jump-item--rpdb-home')).toBeTruthy()

    document.body.querySelector<HTMLButtonElement>('.jump-item--rpdb-item button')?.click()
    expect(onInsert).toHaveBeenCalledOnce()
    const html = onInsert.mock.calls[0][0] as string
    expect(html).toContain('data-jump-href="/rpdb/1"')
    expect(html).toContain('data-jump-type="rpdb_work"')
    expect(html).toContain('data-jump-variant="rpdb-item"')
    expect(html).toContain('data-jump-summary="适合夜间巡逻场景"')
    expect(html).toContain('data-jump-views="120"')
  })
})

describe('PostQuickJump character cards', () => {
  it('inserts a published card with a stable id and character-card route', async () => {
    mocks.listGuilds.mockResolvedValue({ guilds: [] })
    mocks.listPosts.mockResolvedValue({ posts: [], total: 0 })
    mocks.listRPDBWorks.mockResolvedValue({ works: [], total: 0 })
    mocks.listCharacterCards.mockResolvedValue({ character_cards: [
      baseCharacterCard,
      { ...baseCharacterCard, id: 28, status: 'draft', display_name: '不应出现的草稿' },
    ] })
    const onInsert = vi.fn()

    wrapper = mount(PostQuickJump, {
      attachTo: document.body,
      props: { modelValue: false, onInsert },
    })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const tab = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.quick-jump__tabs button'))
      .find((button) => button.textContent?.trim() === '人物卡')
    tab?.click()
    await flushPromises()

    expect(document.body.textContent).toContain('伊莉娅·星语')
    expect(document.body.textContent).not.toContain('不应出现的草稿')
    document.body.querySelector<HTMLButtonElement>('.jump-item--character button')?.click()

    const html = onInsert.mock.calls[0][0] as string
    expect(html).toContain('data-jump-type="character_card"')
    expect(html).toContain('data-jump-id="27"')
    expect(html).toContain('data-jump-href="/character-cards/27"')
    expect(html).toContain('data-jump-variant="character-card"')
    expect(html).toContain('data-jump-safe-placeholder="true"')
    expect(html).not.toContain('伊莉娅·星语')
    expect(html).not.toContain('月之女祭司')
    expect(html).not.toContain('在月神殿保存远行者的旧信')
    expect(html).not.toContain('character-card-portrait/27')
  })
})
