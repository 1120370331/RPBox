import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import ModeratorMain from './ModeratorMain.vue'
import i18n from '@/i18n'

const mocks = vi.hoisted(() => ({
  confirm: vi.fn(),
  getModeratorStats: vi.fn(),
  getPendingCharacterCards: vi.fn(),
  getPendingPosts: vi.fn(),
  reviewCharacterCard: vi.fn(),
}))

vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ isModerator: true, isAdmin: false }),
}))

vi.mock('@/composables/useDialog', () => ({
  dialog: { confirm: mocks.confirm },
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ success: vi.fn(), error: vi.fn(), warning: vi.fn() }),
}))

vi.mock('@/api/moderator', async () => {
  const actual = await vi.importActual<typeof import('@/api/moderator')>('@/api/moderator')
  return {
    ...actual,
    getModeratorStats: mocks.getModeratorStats,
    getPendingCharacterCards: mocks.getPendingCharacterCards,
    getPendingPosts: mocks.getPendingPosts,
    reviewCharacterCard: mocks.reviewCharacterCard,
  }
})

const pendingCard = {
  id: 27,
  user_id: 8,
  first_name: '萨维娅',
  last_name: '暮羽',
  display_name: '萨维娅·暮羽',
  title: '夜航者',
  full_title: '',
  race: '夜之子',
  class: '法师',
  eye_color: '琥珀色',
  eye_color_hex: '#D59A43',
  age: '',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  relationship_status: '',
  icon: '',
  class_color: '#8B6CFF',
  name_color: '#8B6CFF',
  summary: '在苏拉玛与远洋航路之间往返的星象师。',
  background_story: '',
  first_impression: '',
  impressions: [],
  other_content: '',
  portrait_image_url: '',
  status: 'published',
  visibility: 'public',
  review_status: 'pending',
  author_name: '星海旅人',
  created_at: '2026-08-10T08:00:00Z',
  updated_at: '2026-08-11T08:00:00Z',
}

async function mountModerator() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/moderator', component: ModeratorMain },
      { path: '/', name: 'home', component: { template: '<div />' } },
      { path: '/character-cards/:id', component: { template: '<div />' } },
    ],
  })
  await router.push('/moderator')
  await router.isReady()
  const wrapper = mount(ModeratorMain, {
    global: {
      plugins: [router, i18n],
      stubs: {
        CharacterCardPortrait: true,
        ImageViewer: true,
        RModal: true,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('ModeratorMain character-card review queue', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
    vi.clearAllMocks()
    mocks.confirm.mockResolvedValue(true)
    mocks.getModeratorStats.mockResolvedValue({
      pending_posts: 0,
      pending_items: 0,
      pending_character_cards: 1,
      pending_guilds: 0,
      pending_reports: 0,
      total_pending_reviews: 1,
      total_posts: 0,
      total_items: 0,
      total_guilds: 0,
      total_users: 0,
      today_posts: 0,
      today_items: 0,
      today_users: 0,
    })
    mocks.getPendingPosts.mockResolvedValue({ posts: [], total: 0 })
    mocks.getPendingCharacterCards.mockResolvedValue({ character_cards: [pendingCard], total: 1 })
    mocks.reviewCharacterCard.mockResolvedValue({ message: 'ok' })
  })

  it.each([
    ['approve' as const, '.btn-approve'],
    ['reject' as const, '.btn-reject'],
  ])('renders the queue and submits a %s decision with the moderator comment', async (action, selector) => {
    const wrapper = await mountModerator()

    await wrapper.get('[data-testid="moderator-tab-character-cards"]').trigger('click')
    await flushPromises()

    expect(mocks.getPendingCharacterCards).toHaveBeenCalledWith({ status: 'pending', page: 1, page_size: 20 })
    const queue = wrapper.get('[data-testid="character-card-review-queue"]')
    expect(queue.text()).toContain('萨维娅·暮羽')
    expect(queue.text()).toContain('星海旅人')
    expect(queue.text()).toContain('在苏拉玛与远洋航路之间往返的星象师。')
    expect(queue.get('.btn-preview').attributes('href')).toBe('/character-cards/27?review=submission')

    await queue.get('textarea').setValue('请保留这条版主审核说明。')
    await queue.get(selector).trigger('click')
    await flushPromises()

    expect(mocks.reviewCharacterCard).toHaveBeenCalledWith(27, {
      action,
      comment: '请保留这条版主审核说明。',
    })
    expect(mocks.confirm).toHaveBeenCalledWith(expect.objectContaining({
      confirmText: action === 'approve' ? '通过人物卡' : '拒绝人物卡',
    }))
    wrapper.unmount()
  })
})
