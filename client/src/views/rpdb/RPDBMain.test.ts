import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createPinia } from 'pinia'
import RPDBMain from './RPDBMain.vue'

const { listRPDBWorks } = vi.hoisted(() => ({ listRPDBWorks: vi.fn() }))
vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return { ...actual, listRPDBWorks }
})

describe('RPDBMain', () => {
  beforeEach(() => {
    localStorage.clear()
    listRPDBWorks.mockReset()
    listRPDBWorks.mockResolvedValue({ works: [], total: 0, page: 1, page_size: 12 })
  })

  it('renders discovery controls and empty contribution action', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: RPDBMain },
        { path: '/rpdb', component: RPDBMain },
        { path: '/rpdb/create', component: { template: '<div />' } },
        { path: '/rpdb/lists', component: { template: '<div />' } },
        { path: '/rpdb/my-uploads', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    const wrapper = mount(RPDBMain, { global: { plugins: [createPinia(), router] } })
    await flushPromises()
    expect(wrapper.find('.rpdb-topbar').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('玩家共创资料库')
    expect(wrapper.text()).toContain('魔兽物品')
    expect(wrapper.text()).toContain('家宅分享')
    expect(wrapper.text()).toContain('发布第一个作品')
    expect(wrapper.find('.rpdb-shell').exists()).toBe(true)
    expect(wrapper.find('.content-rail').exists()).toBe(false)
    expect(wrapper.find('.content-nav').exists()).toBe(true)
    expect(wrapper.find('a[href="/rpdb/my-uploads"]').text()).toContain('我的上传')
    expect(wrapper.find('.featured-grid').exists()).toBe(true)
    expect(wrapper.find('.discovery-toolbar').classes()).not.toContain('sticky')
    expect(getComputedStyle(wrapper.find('.discovery-toolbar').element).position).not.toBe('sticky')
    expect(wrapper.findAll('.featured-card')).toHaveLength(3)
    expect(listRPDBWorks).toHaveBeenCalledWith(expect.objectContaining({ page: 1, page_size: 12 }))
    const typeButtons = wrapper.findAll('.channel-tabs button')
    expect(typeButtons).toHaveLength(4)
    expect(typeButtons.map(button => button.text())).toEqual(
      expect.arrayContaining([
        expect.stringContaining('全部内容'),
        expect.stringContaining('魔兽物品'),
        expect.stringContaining('幻化方案'),
        expect.stringContaining('家宅分享'),
      ]),
    )
  })

  it('paginates search results with at most 12 works per page', async () => {
    const makeWork = (id: number) => ({
      id,
      author_id: 1,
      type: 'item_showcase',
      title: `作品 ${id}`,
      slug: `work-${id}`,
      summary: '分页测试',
      content: '',
      content_type: 'html',
      cover_image: '',
      rp_use_cases: '',
      effect_description: '',
      restrictions: '',
      extra: '',
      game_version: '',
      expansion: '',
      availability_status: 'available',
      bind_type: '',
      faction: 'neutral',
      armor_type: '',
      verification_status: 'verified',
      verified_count: 0,
      outdated_count: 0,
      status: 'published',
      is_public: true,
      review_status: 'approved',
      version: 1,
      view_count: 0,
      like_count: 0,
      favorite_count: 0,
      comment_count: 0,
      list_count: 0,
      media_count: 0,
      is_liked: false,
      is_favorited: false,
      in_collection_list: false,
      created_at: '',
      updated_at: '',
    })
    listRPDBWorks
      .mockResolvedValueOnce({ works: Array.from({ length: 12 }, (_, index) => makeWork(index + 1)), total: 25, page: 1, page_size: 12 })
      .mockResolvedValueOnce({ works: Array.from({ length: 12 }, (_, index) => makeWork(index + 13)), total: 25, page: 2, page_size: 12 })

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: RPDBMain },
        { path: '/rpdb/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    const wrapper = mount(RPDBMain, { global: { plugins: [createPinia(), router] } })
    await flushPromises()

    expect(wrapper.findAll('[data-testid="rpdb-work-card"]')).toHaveLength(12)
    expect(wrapper.findAll('[data-testid="rpdb-featured-metrics"]')).toHaveLength(3)
    expect(wrapper.find('[data-testid="rpdb-featured-metrics"] [title="浏览"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-featured-metrics"] [title="点赞"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-featured-metrics"] [title="收藏"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-featured-metrics"] [title="加入清单"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('每页最多 12 个')
    expect(wrapper.find('[data-testid="rpdb-pagination"]').text()).toContain('第 1 / 3 页')
    expect(wrapper.get('[data-testid="rpdb-card-view"]').attributes('aria-pressed')).toBe('true')

    await wrapper.get('[data-testid="rpdb-compact-view"]').trigger('click')
    expect(wrapper.get('[data-testid="rpdb-discovery-results"]').classes()).toContain('compact')
    expect(wrapper.findAll('[data-testid="rpdb-work-card"]').every(card => card.classes().includes('work-card--compact'))).toBe(true)
    expect(localStorage.getItem('rpdb-view-mode')).toBe('compact')

    await wrapper.findAll('[data-testid="rpdb-pagination"] button')[1].trigger('click')
    await flushPromises()

    expect(listRPDBWorks).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, page_size: 12 }))
    expect(wrapper.find('[data-testid="rpdb-pagination"]').text()).toContain('第 2 / 3 页')
  })
})
