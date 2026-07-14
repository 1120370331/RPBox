import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import Favorites from './Favorites.vue'

const api = vi.hoisted(() => ({
  listMyFavorites: vi.fn(),
  listMyItemFavorites: vi.fn(),
  listMyCollectionFavorites: vi.fn(),
  listMyRPDBFavorites: vi.fn(),
}))

vi.mock('@/api/post', async () => {
  const actual = await vi.importActual<typeof import('@/api/post')>('@/api/post')
  return { ...actual, listMyFavorites: api.listMyFavorites }
})

vi.mock('@/api/item', async () => {
  const actual = await vi.importActual<typeof import('@/api/item')>('@/api/item')
  return { ...actual, listMyItemFavorites: api.listMyItemFavorites }
})

vi.mock('@/api/collection', async () => {
  const actual = await vi.importActual<typeof import('@/api/collection')>('@/api/collection')
  return { ...actual, listMyCollectionFavorites: api.listMyCollectionFavorites }
})

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return { ...actual, listMyRPDBFavorites: api.listMyRPDBFavorites }
})

describe('Favorites RPDB tab', () => {
  beforeEach(() => {
    api.listMyFavorites.mockResolvedValue({ posts: [] })
    api.listMyItemFavorites.mockResolvedValue({ code: 0, data: { items: [] } })
    api.listMyCollectionFavorites.mockResolvedValue({ collections: [] })
    api.listMyRPDBFavorites.mockResolvedValue({
      works: [{
        id: 42,
        author_id: 2,
        author_name: '雾灯',
        type: 'home_showcase',
        title: '暮色森林炼金小屋',
        slug: 'twilight-home',
        summary: '一套适合隐居炼金术士的家宅布置',
        content: '',
        content_type: 'html',
        cover_image: '',
        rp_use_cases: '',
        effect_description: '',
        restrictions: '{}',
        extra: '{}',
        game_version: '',
        expansion: '',
        availability_status: 'available',
        bind_type: '',
        faction: '',
        armor_type: '',
        verification_status: 'verified',
        verified_count: 3,
        outdated_count: 0,
        status: 'published',
        is_public: true,
        review_status: 'approved',
        version: 1,
        view_count: 18,
        like_count: 4,
        favorite_count: 2,
        comment_count: 1,
        list_count: 0,
        media_count: 0,
        is_liked: false,
        is_favorited: true,
        in_collection_list: false,
        created_at: '2026-07-10T00:00:00Z',
        updated_at: '2026-07-10T00:00:00Z',
      }],
    })
  })

  it('loads RPDB favorites from the dedicated tab', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/favorites', component: Favorites },
        { path: '/rpdb/:id', name: 'rpdb-detail', component: { template: '<div />' } },
      ],
    })
    const i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      messages: {
        'zh-CN': {
          library: {
            favorites: {
              back: '返回',
              title: '我的收藏',
              subtitle: '集中查看已收藏内容',
              tabs: { posts: '帖子', items: '道具', collections: '合集', rpdb: 'RP 数据库' },
              searchPlaceholder: '搜索收藏',
              loading: '加载中',
              empty: {
                posts: '暂无帖子',
                postsSearch: '没有匹配帖子',
                items: '暂无道具',
                itemsSearch: '没有匹配道具',
                collections: '暂无合集',
                collectionsSearch: '没有匹配合集',
                rpdb: '暂无 RP 数据库作品',
                rpdbSearch: '没有匹配作品',
              },
              itemTypes: { item: '道具', campaign: '战役', artwork: '作品' },
              anonymous: '匿名',
              noDescription: '暂无描述',
              itemCount: '{count} 项',
            },
          },
        },
      },
    })
    await router.push('/favorites')
    const wrapper = mount(Favorites, { global: { plugins: [router, i18n] } })
    await flushPromises()

    const tab = wrapper.find('[data-testid="favorites-tab-rpdb"]')
    expect(tab.exists()).toBe(true)
    await tab.trigger('click')
    await flushPromises()

    expect(api.listMyRPDBFavorites).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('暮色森林炼金小屋')
  })
})
