import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { useNotificationStore } from '@/stores/notification'
import { useUserStore } from '@/stores/user'
import AppLayout from './AppLayout.vue'

const getUserInfo = vi.hoisted(() => vi.fn())

vi.mock('@/api/user', () => ({
  getUserInfo,
}))

describe('AppLayout', () => {
  beforeEach(() => {
    localStorage.clear()
    getUserInfo.mockReset()
    getUserInfo.mockResolvedValue({ id: 1, username: 'tester', role: 'user' })
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
})
