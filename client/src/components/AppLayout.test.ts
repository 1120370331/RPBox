import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { useNotificationStore } from '@/stores/notification'
import { useSidebarBadgesStore } from '@/stores/sidebarBadges'
import { useUserStore } from '@/stores/user'
import AppLayout from './AppLayout.vue'

const getUserInfo = vi.hoisted(() => vi.fn())
const sidebarApiMocks = vi.hoisted(() => ({
  listPosts: vi.fn(),
  listEvents: vi.fn(),
  listItems: vi.fn(),
  listRPDBWorks: vi.fn(),
  getAddonLatest: vi.fn(),
  getTRP3Latest: vi.fn(),
  invoke: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  getUserInfo,
}))

vi.mock('@/api/post', () => ({
  listPosts: sidebarApiMocks.listPosts,
  listEvents: sidebarApiMocks.listEvents,
}))
vi.mock('@/api/item', () => ({ listItems: sidebarApiMocks.listItems }))
vi.mock('@/api/rpdb', () => ({ listRPDBWorks: sidebarApiMocks.listRPDBWorks }))
vi.mock('@/api/addon', () => ({
  getAddonLatest: sidebarApiMocks.getAddonLatest,
  getTRP3Latest: sidebarApiMocks.getTRP3Latest,
}))
vi.mock('@tauri-apps/api/core', () => ({ invoke: sidebarApiMocks.invoke }))

describe('AppLayout', () => {
  beforeEach(() => {
    localStorage.clear()
    i18n.global.locale.value = 'zh-CN'
    getUserInfo.mockReset()
    getUserInfo.mockResolvedValue({ id: 1, username: 'tester', role: 'user' })
    sidebarApiMocks.listPosts.mockResolvedValue({ posts: [], total: 0 })
    sidebarApiMocks.listEvents.mockResolvedValue({ events: [] })
    sidebarApiMocks.listItems.mockResolvedValue({ data: { items: [], total: 0 } })
    sidebarApiMocks.listRPDBWorks.mockResolvedValue({ works: [], total: 0, page: 1, page_size: 1 })
  })

  it('keeps RPDB routes inside the standard application frame', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/login', component: { template: '<div />' } },
      ],
    })
    await router.push('/rpdb')
    await router.isReady()

    const wrapper = mount(AppLayout, {
      global: { plugins: [createPinia(), i18n, router] },
    })

    expect(wrapper.find('.app-layout.rpdb-mode').exists()).toBe(false)
    expect(wrapper.find('.sidebar').exists()).toBe(true)
    expect(wrapper.find('.main-content.rpdb-main-content').exists()).toBe(false)
    wrapper.unmount()
  })

  it('connects notifications for an authenticated session and disconnects on unmount', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/notifications', component: { template: '<div />' } },
        { path: '/user/:id', component: { template: '<div />' } },
        { path: '/login', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const pinia = createPinia()
    const userStore = useUserStore(pinia)
    userStore.setAuth('test-token', { id: 1, username: 'tester', role: 'user' })

    const notificationStore = useNotificationStore(pinia)
    const connectWebSocket = vi.spyOn(notificationStore, 'connectWebSocket').mockImplementation(() => {})
    const disconnectWebSocket = vi.spyOn(notificationStore, 'disconnectWebSocket').mockImplementation(() => {})
    const loadUnreadCount = vi.spyOn(notificationStore, 'loadUnreadCount').mockResolvedValue()

    const wrapper = mount(AppLayout, {
      global: { plugins: [pinia, i18n, router] },
    })

    await wrapper.vm.$nextTick()
    expect(connectWebSocket).toHaveBeenCalledTimes(1)
    expect(loadUnreadCount).toHaveBeenCalled()

    wrapper.unmount()
    expect(disconnectWebSocket).toHaveBeenCalledTimes(1)
  })

  it('renders distinct content, event, and add-on bubbles and marks a menu as read on click', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/warcraft', component: { template: '<div />' } },
        { path: '/market', component: { template: '<div />' } },
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/community', component: { template: '<div />' } },
        { path: '/guild', component: { template: '<div />' } },
        { path: '/settings', component: { template: '<div />' } },
        { path: '/login', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const pinia = createPinia()
    const sidebarBadgesStore = useSidebarBadgesStore(pinia)
    sidebarBadgesStore.$patch({
      unreadCounts: { community: 8, events: 2, market: 4, rpdb: 6 },
      addonUpdateCount: 2,
      systemUpdateAvailable: true,
      systemUpdateVersion: '1.2.0',
    })
    const markMenuRead = vi.spyOn(sidebarBadgesStore, 'markMenuRead')

    const wrapper = mount(AppLayout, {
      global: { plugins: [pinia, i18n, router] },
    })

    expect(wrapper.findAll('.sidebar-badge--count')).toHaveLength(3)
    expect(wrapper.get('.sidebar-badge--event').text()).toContain('2')
    expect(wrapper.get('.sidebar-badge--addon').text()).toContain('更新')
    expect(wrapper.get('.sidebar-badge--system').attributes('title')).toContain('1.2.0')

    const communityButton = wrapper.findAll('.menu-item')
      .find(button => button.text().includes('社区广场'))
    await communityButton!.trigger('click')
    expect(markMenuRead).toHaveBeenCalledWith('community')

    const settingsButton = wrapper.findAll('.menu-item')
      .find(button => button.text().includes('系统设置'))
    await settingsButton!.trigger('click')
    expect(markMenuRead).toHaveBeenCalledWith('settings')

    wrapper.unmount()
  })
})
