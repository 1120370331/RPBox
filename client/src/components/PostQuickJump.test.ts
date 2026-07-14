import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import PostQuickJump from './PostQuickJump.vue'
import type { RPDBWork } from '@/api/rpdb'

const mocks = vi.hoisted(() => ({
  listGuilds: vi.fn(),
  listGuildStories: vi.fn(),
  listPosts: vi.fn(),
  listRPDBWorks: vi.fn(),
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

let wrapper: VueWrapper | null = null

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
