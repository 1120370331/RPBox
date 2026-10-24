import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useUserStore } from '@/stores/user'
import i18n from '@/i18n'
import type { HomeContent, HomeFeedResponse } from '@/api/home'
import Home from './Home.vue'

const mocks = vi.hoisted(() => ({ featured: vi.fn(), recommendations: vi.fn(), signIn: vi.fn() }))
vi.mock('@/api/home', () => ({
  getHomeFeatured: mocks.featured,
  getHomeRecommendations: mocks.recommendations,
  homeContentKey: (entry: HomeContent) => `${entry.kind}:${entry.id}`,
}))
vi.mock('@/api/user', () => ({ signInDaily: mocks.signIn }))
vi.mock('@/api/item', () => ({
  getImageUrl: (type: string, id: number, options: { v: string }) => `/images/${type}/${id}?v=${options.v}`,
  resolveApiUrl: (url: string) => url,
}))

const entry = (id: number, kind: HomeContent['kind'] = 'post'): HomeContent => ({
  id, kind, title: `内容 ${kind}-${id}`, author_name: '创作者', has_cover: true,
  cover_url: '/uploads/cover.png', cover_updated_at: '2026-10-22', updated_at: '2026-10-22',
  weekly_score: 12, reason: 'interest',
  view_count: 128, like_count: 24, favorite_count: 9, comment_count: 6, list_count: 4, downloads: 37,
})
const feed = (entries: HomeContent[], hasMore = true): HomeFeedResponse => ({ entries, personalized: true, has_more: hasMore })
let wrapper: VueWrapper
let store: ReturnType<typeof useUserStore>

beforeEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
  i18n.global.locale.value = 'zh-CN'
  mocks.featured.mockResolvedValue(feed([entry(1), entry(2, 'rpdb'), entry(3, 'item')]))
  mocks.recommendations.mockResolvedValue(feed([entry(4), entry(5, 'rpdb'), entry(6, 'item')]))
})
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

async function openHome() {
  const pinia = createPinia()
  store = useUserStore(pinia)
  store.setAuth('test-token', { id: 1, username: 'reader' })
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/', component: Home },
    { path: '/community/post/:id', component: { template: '<div />' } },
    { path: '/rpdb/:id', component: { template: '<div />' } },
    { path: '/market/:id', component: { template: '<div />' } },
  ] })
  await router.push('/')
  wrapper = mount(Home, { attachTo: document.body, global: { plugins: [pinia, router, i18n] } })
  await flushPromises()
  return router
}

describe('home discovery feed', () => {
  it('replaces shortcuts with two rows of three cards and correct detail links', async () => {
    await openHome()
    expect(wrapper.find('.quick-grid').exists()).toBe(false)
    const rows = wrapper.findAll('.home-content-row')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('今日精选')
    expect(rows[1]!.text()).toContain('为您推荐')
    expect(rows[0]!.findAll('.feed-card')).toHaveLength(3)
    expect(rows[1]!.findAll('.feed-card')).toHaveLength(3)
    expect(rows[0]!.findAll('.feed-card').map(card => card.attributes('href'))).toEqual(['/community/post/1', '/rpdb/2', '/market/3'])
    expect(rows[0]!.find('img').attributes('src')).toContain('v=2026-10-22')
    expect(rows[1]!.find('.feed-refresh').exists()).toBe(true)
    const cards = rows[0]!.findAll('.feed-card')
    expect(cards[0]!.findAll('.work-card__metrics span').map(metric => metric.attributes('title'))).toEqual(['浏览', '点赞', '收藏', '评论'])
    expect(cards[0]!.findAll('.work-card__metrics b').map(metric => metric.text())).toEqual(['128', '24', '9', '6'])
    expect(cards[1]!.findAll('.work-card__metrics span').map(metric => metric.attributes('title'))).toEqual(['浏览', '点赞', '收藏', '加入清单'])
    expect(cards[1]!.findAll('.work-card__metrics b').map(metric => metric.text())).toEqual(['128', '24', '9', '4'])
    expect(cards[2]!.findAll('.work-card__metrics span').map(metric => metric.attributes('title'))).toEqual(['下载', '点赞', '收藏'])
    expect(cards[2]!.findAll('.work-card__metrics b').map(metric => metric.text())).toEqual(['37', '24', '9'])
    for (const row of rows) {
      for (const metrics of row.findAll('.work-card__metrics')) {
        expect(metrics.text()).not.toContain('七日人气')
        expect(metrics.text()).not.toContain('兴趣')
      }
    }
  })

  it('shows unavailable counters as unknown while preserving genuine zero values', async () => {
    mocks.featured.mockResolvedValueOnce(feed([{ ...entry(1), view_count: undefined, like_count: 0 }]))
    await openHome()
    const counters = wrapper.findAll('.home-content-row')[0]!.findAll('.work-card__metrics b')
    expect(counters[0]!.text()).toBe('—')
    expect(counters[1]!.text()).toBe('0')
  })

  it('excludes previous cards while refreshing and prevents overlapping refreshes', async () => {
    await openHome()
    let finish!: (response: HomeFeedResponse) => void
    mocks.recommendations.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const refresh = wrapper.get('.feed-refresh')
    await refresh.trigger('click')
    await refresh.trigger('click')
    expect(mocks.recommendations).toHaveBeenCalledTimes(2)
    expect(mocks.recommendations).toHaveBeenLastCalledWith(['post:4', 'rpdb:5', 'item:6'])
    expect(wrapper.findAll('.home-content-row')[1]!.text()).toContain('内容 post-4')
    finish(feed([entry(7), entry(8), entry(9)]))
    await flushPromises()
    const text = wrapper.findAll('.home-content-row')[1]!.text()
    expect(text).toContain('内容 post-7')
    expect(text).not.toContain('内容 post-4')
  })

  it('keeps the current batch on failure and retries without losing content', async () => {
    await openHome()
    mocks.recommendations.mockRejectedValueOnce(new Error('offline'))
    await wrapper.get('.feed-refresh').trigger('click')
    await flushPromises()
    const row = wrapper.findAll('.home-content-row')[1]!
    expect(row.get('[role="alert"]').text()).toContain('内容加载失败')
    expect(row.findAll('.feed-card')).toHaveLength(3)
    mocks.recommendations.mockResolvedValueOnce(feed([entry(10)], false))
    await row.get('.feed-error button').trigger('click')
    await flushPromises()
    expect(row.text()).toContain('内容 post-10')
    expect(row.get('.feed-refresh').attributes('disabled')).toBeUndefined()
  })

  it('cycles an exhausted candidate history while still excluding the visible batch', async () => {
    await openHome()
    mocks.recommendations.mockResolvedValueOnce(feed([])).mockResolvedValueOnce(feed([entry(8), entry(9), entry(10)]))
    await wrapper.get('.feed-refresh').trigger('click')
    await flushPromises()
    expect(mocks.recommendations).toHaveBeenCalledTimes(3)
    expect(mocks.recommendations).toHaveBeenLastCalledWith(['post:4', 'rpdb:5', 'item:6'])
    expect(wrapper.findAll('.home-content-row')[1]!.text()).toContain('内容 post-8')
  })

  it('ignores late responses from a previous account', async () => {
    let previous!: (response: HomeFeedResponse) => void
    mocks.recommendations.mockImplementationOnce(() => new Promise(resolve => { previous = resolve }))
    await openHome()
    mocks.recommendations.mockResolvedValueOnce(feed([entry(99)]))
    store.setAuth('other-token', { id: 2, username: 'other' })
    await flushPromises()
    previous(feed([entry(777)]))
    await flushPromises()
    expect(wrapper.text()).toContain('内容 post-99')
    expect(wrapper.text()).not.toContain('内容 post-777')
  })

  it('shows honest empty and failed states instead of inventing recommendations', async () => {
    mocks.featured.mockRejectedValueOnce(new Error('offline'))
    mocks.recommendations.mockResolvedValueOnce(feed([], false))
    await openHome()
    expect(wrapper.findAll('.feed-card')).toHaveLength(0)
    expect(wrapper.get('[role="alert"]').text()).toContain('内容加载失败')
    expect(wrapper.get('.feed-empty').text()).toContain('还没有可展示的公开内容')
  })
})
