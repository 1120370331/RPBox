import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createPinia } from 'pinia'
import { useThemeStore } from '@/stores/theme'
import { displayContrast, parseDisplayColor } from '@/utils/displayColor'
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
      name_color: '#000000',
      name_bold: true,
      avatar: '',
      bio: '在艾泽拉斯记录旅途与角色故事。',
      location: '月光林地',
      post_count: 8,
      item_count: 3,
      guild_count: 1,
      forum_level: 6,
      forum_level_name: '资深旅人',
      forum_level_color: '#7E6896',
      forum_level_bold: true,
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
      global: { plugins: [createPinia(), router, i18n] },
    })

    await wrapper.get('.user-avatar-popover__trigger').trigger('mouseenter')
    await flushPromises()

    const popover = document.querySelector<HTMLElement>('.user-avatar-popover')
    expect(popover).not.toBeNull()
    expect(popover!.textContent).toContain('月桂旅人')
    expect(popover!.textContent).toContain('在艾泽拉斯记录旅途与角色故事。')
    expect(popover!.textContent).toContain('莱拉')
    expect(popover!.textContent).toContain('Lv6 资深旅人')
    expect(popover!.textContent).not.toContain('月光林地')
    expect(popover!.querySelector('.user-level-badge')).not.toBeNull()
    expect(popover!.querySelector('.user-avatar-popover__location')).toBeNull()
    expect(mocks.getUserProfile).toHaveBeenCalledWith(5)
    expect(mocks.listUserCharacterCards).toHaveBeenCalledWith(5)

    await wrapper.get('.user-avatar-popover__trigger').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/user/5')
    wrapper.unmount()
  })

  it('keeps list avatars compact and clickable without opening a hover preview', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/user/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const wrapper = mount(UserAvatarPopover, {
      attachTo: document.body,
      props: {
        userId: 5,
        username: '月桂旅人',
        avatarUrl: '/avatar.png',
        size: 20,
        showPopover: false,
      },
      global: { plugins: [createPinia(), router, i18n] },
    })

    const trigger = wrapper.get('.user-avatar-popover__trigger')
    expect(trigger.attributes('style')).toContain('width: 20px')
    expect(trigger.attributes('style')).toContain('height: 20px')
    expect(trigger.attributes('aria-haspopup')).toBeUndefined()
    await trigger.trigger('mouseenter')
    await flushPromises()

    expect(document.querySelector('.user-avatar-popover')).toBeNull()
    expect(mocks.getUserProfile).not.toHaveBeenCalled()
    await trigger.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/user/5')
    wrapper.unmount()
  })

  it('updates an open teleported dyed identity and TRP3 card name on theme changes without modifying API data', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/user/:id', component: { template: '<div />' } },
        { path: '/character-cards/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    const pinia = createPinia()
    const theme = useThemeStore(pinia)
    theme.setTheme('classic')
    const apiCard = Object.freeze({ id: 21, display_name: '黑色姓名', name_color: '80000000', class_color: '80000000' })
    mocks.listUserCharacterCards.mockResolvedValue({ character_cards: [apiCard] })
    const wrapper = mount(UserAvatarPopover, {
      attachTo: document.body,
      props: { userId: 5, username: '月桂旅人' },
      global: { plugins: [pinia, router, i18n] },
    })
    await wrapper.get('.user-avatar-popover__trigger').trigger('mouseenter')
    await flushPromises()
    for (const id of ['classic', 'black-gold', 'dreamy-pink-blue', 'classic']) {
      theme.setTheme(id)
      await flushPromises()
      const name = document.querySelector<HTMLElement>('.user-avatar-popover__identity strong')!
      const cardName = document.querySelector<HTMLElement>('.user-avatar-popover__card-copy strong')!
      for (const background of [theme.currentTheme.colors.gradientStart, theme.currentTheme.colors.gradientEnd]) {
        expect(displayContrast(parseDisplayColor(name.style.color)!, parseDisplayColor(background)!)).toBeGreaterThanOrEqual(4.5)
      }
      expect(name.style.fontWeight).toBe('700')
      expect(displayContrast(parseDisplayColor(cardName.style.color)!, parseDisplayColor(theme.currentTheme.colors.panelBg)!)).toBeGreaterThanOrEqual(4.5)
    }
    expect(apiCard.name_color).toBe('80000000')
    expect(mocks.getUserProfile).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})
