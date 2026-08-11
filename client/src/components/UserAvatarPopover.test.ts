import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { clearUserPopoverDataCache } from '@/utils/userPopoverData'
import UserAvatarPopover from './UserAvatarPopover.vue'

const mocks = vi.hoisted(() => ({
  getUserProfile: vi.fn(),
  listUserCharacterCards: vi.fn(),
}))

vi.mock('@/api/user', async () => {
  const actual = await vi.importActual<typeof import('@/api/user')>('@/api/user')
  return { ...actual, getUserProfile: mocks.getUserProfile }
})

vi.mock('@/api/characterCard', async () => {
  const actual = await vi.importActual<typeof import('@/api/characterCard')>('@/api/characterCard')
  return { ...actual, listUserCharacterCards: mocks.listUserCharacterCards }
})

describe('UserAvatarPopover', () => {
  beforeEach(() => {
    clearUserPopoverDataCache()
    mocks.getUserProfile.mockReset()
    mocks.listUserCharacterCards.mockReset()
    mocks.getUserProfile.mockResolvedValue({
      id: 5,
      username: '月桂旅人',
      avatar: '',
      bio: '在艾泽拉斯记录旅途与角色故事。',
      location: '月光林地',
      post_count: 8,
      item_count: 3,
      guild_count: 1,
      forum_level_name: '资深旅人',
    })
    mocks.listUserCharacterCards.mockResolvedValue({
      character_cards: [{
        id: 21,
        user_id: 5,
        first_name: '莱拉',
        last_name: '',
        display_name: '莱拉',
        title: '月光哨兵',
        race: '暗夜精灵',
        class: '德鲁伊',
        class_color: 'FF7D0A',
        name_color: 'FF7D0A',
        portrait_image_url: '',
        status: 'published',
        visibility: 'public',
        review_status: 'approved',
        created_at: '2026-08-11T00:00:00Z',
        updated_at: '2026-08-11T00:00:00Z',
      }],
    })
    i18n.global.locale.value = 'zh-CN'
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('opens a themed public profile card and links the avatar to the user page', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/user/:id', component: { template: '<div />' } },
        { path: '/character-cards/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const wrapper = mount(UserAvatarPopover, {
      attachTo: document.body,
      props: {
        userId: 5,
        username: '月桂旅人',
        avatarUrl: '',
      },
      attrs: { class: 'author-avatar' },
      global: { plugins: [router, i18n] },
    })

    await wrapper.get('.user-avatar-popover__trigger').trigger('mouseenter')
    await flushPromises()

    const popover = document.querySelector<HTMLElement>('.user-avatar-popover')
    expect(popover).not.toBeNull()
    expect(popover!.textContent).toContain('月桂旅人')
    expect(popover!.textContent).toContain('在艾泽拉斯记录旅途与角色故事。')
    expect(popover!.textContent).toContain('莱拉')
    expect(mocks.getUserProfile).toHaveBeenCalledWith(5)
    expect(mocks.listUserCharacterCards).toHaveBeenCalledWith(5)

    await wrapper.get('.user-avatar-popover__trigger').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/user/5')
    wrapper.unmount()
  })
})
