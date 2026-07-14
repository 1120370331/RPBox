import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it } from 'vitest'
import i18n from '@/i18n'
import AppLayout from './AppLayout.vue'

describe('AppLayout', () => {
  beforeEach(() => {
    localStorage.clear()
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
})
