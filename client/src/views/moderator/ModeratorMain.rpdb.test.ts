import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import ModeratorMain from './ModeratorMain.vue'

const api = vi.hoisted(() => ({
  getModeratorStats: vi.fn(),
  getPendingPosts: vi.fn(),
  getPendingRPDBWorks: vi.fn(),
  getPendingRPDBMedia: vi.fn(),
  getPendingRPDBRevisions: vi.fn(),
  getModeratorReports: vi.fn(),
}))

vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ isModerator: true, isAdmin: true }),
}))

vi.mock('@/composables/useDialog', () => ({
  dialog: { confirm: vi.fn().mockResolvedValue(true) },
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ success: vi.fn(), error: vi.fn(), warning: vi.fn() }),
}))

vi.mock('@/api/moderator', async () => {
  const actual = await vi.importActual<typeof import('@/api/moderator')>('@/api/moderator')
  return {
    ...actual,
    getModeratorStats: api.getModeratorStats,
    getPendingPosts: api.getPendingPosts,
    getPendingRPDBWorks: api.getPendingRPDBWorks,
    getPendingRPDBMedia: api.getPendingRPDBMedia,
    getPendingRPDBRevisions: api.getPendingRPDBRevisions,
    getModeratorReports: api.getModeratorReports,
  }
})

describe('ModeratorMain RPDB review integration', () => {
  beforeEach(() => {
    api.getModeratorStats.mockResolvedValue({
      pending_posts: 0,
      pending_items: 0,
      pending_rpdb_works: 1,
      pending_rpdb_media: 1,
      pending_rpdb_revisions: 1,
      pending_guilds: 0,
      pending_reports: 0,
      total_pending_reviews: 3,
      total_posts: 0,
      total_items: 0,
      total_guilds: 0,
      total_users: 0,
      today_posts: 0,
      today_items: 0,
      today_users: 0,
    })
    api.getPendingPosts.mockResolvedValue({ posts: [], total: 0 })
    api.getPendingRPDBWorks.mockResolvedValue({ works: [], total: 0 })
    api.getPendingRPDBMedia.mockResolvedValue({ media: [], total: 0 })
    api.getPendingRPDBRevisions.mockResolvedValue({ revisions: [], total: 0 })
    api.getModeratorReports.mockResolvedValue({
      reports: [{
        id: 41,
        target_type: 'rpdb_work',
        target_id: 12,
        target_user_id: 2,
        target_title: '来源可疑的道具档案',
        target_author_name: 'rpdb-author',
        target_preview_text: '摘要说明与被举报的正文内容',
        target_preview_image: '/uploads/rpdb/report-cover.jpg',
        target_url: '/rpdb/12',
        status: 'pending',
        report_count: 1,
        latest_reported_at: '2026-07-13T00:00:00Z',
        reasons: [{ id: 8, reporter_id: 3, reporter_name: 'viewer', reason: 'fraud', detail: '来源不符', created_at: '2026-07-13T00:00:00Z' }],
      }],
      total: 1,
    })
  })

  it('opens the RPDB moderation queue from the review tabs', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/moderator', component: ModeratorMain },
        { path: '/', name: 'home', component: { template: '<div />' } },
        { path: '/rpdb/:id', name: 'rpdb-detail', component: { template: '<div />' } },
      ],
    })
    await router.push('/moderator')
    const wrapper = mount(ModeratorMain, {
      global: {
        plugins: [router],
        stubs: {
          ImageViewer: true,
          RModal: true,
        },
      },
    })
    await flushPromises()

    const tab = wrapper.find('[data-testid="moderator-tab-rpdb"]')
    expect(tab.exists()).toBe(true)
    await tab.trigger('click')
    await flushPromises()

    expect(api.getPendingRPDBWorks).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="rpdb-moderation-panel"]').exists()).toBe(true)

    await wrapper.find('[data-testid="rpdb-moderation-media"]').trigger('click')
    await flushPromises()
    expect(api.getPendingRPDBMedia).toHaveBeenCalledTimes(1)

    await wrapper.find('[data-testid="rpdb-moderation-revisions"]').trigger('click')
    await flushPromises()
    expect(api.getPendingRPDBRevisions).toHaveBeenCalledTimes(1)
  })

  it('shows RPDB work report evidence and opens its target page', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/moderator', component: ModeratorMain },
        { path: '/', name: 'home', component: { template: '<div />' } },
        { path: '/rpdb/:id', name: 'rpdb-detail', component: { template: '<div />' } },
      ],
    })
    await router.push('/moderator')
    const wrapper = mount(ModeratorMain, {
      global: { plugins: [router], stubs: { ImageViewer: true, RModal: true } },
    })
    await flushPromises()

    const reportTab = wrapper.findAll('.sub-tab-container button').find(button => button.text().includes('举报'))
    expect(reportTab).toBeTruthy()
    await reportTab!.trigger('click')
    await flushPromises()

    expect(api.getModeratorReports).toHaveBeenCalledWith(expect.objectContaining({ target_scope: 'content' }))
    const reportCard = wrapper.get('.report-card')
    expect(reportCard.text()).toContain('来源可疑的道具档案')
    expect(reportCard.text()).toContain('RP 数据库作品')
    expect(reportCard.text()).toContain('摘要说明与被举报的正文内容')
    expect(reportCard.text()).toContain('来源不符')

    await reportCard.get('.btn-preview').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/rpdb/12')
  })
})
