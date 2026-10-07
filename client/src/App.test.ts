import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { defineComponent, h, nextTick, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { dialog, useDialog } from '@/composables/useDialog'
import App from './App.vue'
import AppLayout from '@/components/AppLayout.vue'

// Trigger the public API from a routed page inside the real application layout.
const ConfirmationPage = defineComponent({
  setup() {
    const result = ref<boolean | null>(null)
    const { confirm } = useDialog()
    return () => h('div', [
      h('button', {
        'data-testid': 'open-confirmation',
        onClick: async () => {
          result.value = await confirm('确认操作？')
        },
      }, '打开确认弹窗'),
      h('output', { 'data-testid': 'confirmation-result' }, String(result.value)),
    ])
  },
})

describe('App global confirmation dialog', () => {
  let wrapper: VueWrapper | undefined

  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()
    sessionStorage.clear()
    i18n.global.locale.value = 'zh-CN'
    dialog.close(false)
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    dialog.close(false)
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
    document.body.innerHTML = ''
  })

  it.each([
    { action: 'confirm', expected: true },
    { action: 'cancel', expected: false },
  ])('renders one dialog and returns $expected on $action', async ({ action, expected }) => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: AppLayout, children: [{ path: '', component: ConfirmationPage }] },
        { path: '/login', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    wrapper = mount(App, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), i18n, router],
        stubs: {
          UpdateNotification: true,
          ChangelogDialog: true,
          AchievementCelebration: true,
          RToast: true,
          RPDBJumpPreview: true,
          CharacterCardJumpPreview: true,
        },
      },
    })
    expect(wrapper.findComponent(AppLayout).exists()).toBe(true)
    expect(document.body.querySelectorAll('.r-dialog__mask')).toHaveLength(0)

    await wrapper.get('[data-testid="open-confirmation"]').trigger('click')
    expect(document.body.querySelectorAll('.r-dialog__mask')).toHaveLength(1)
    expect(document.body.querySelectorAll('.r-dialog__footer')).toHaveLength(1)
    expect(document.body.querySelectorAll('.r-dialog__btn')).toHaveLength(2)
    expect(document.body.querySelectorAll('.r-dialog__btn--confirm')).toHaveLength(1)
    expect(document.body.querySelectorAll('.r-dialog__btn--cancel')).toHaveLength(1)

    const button = document.body.querySelector<HTMLButtonElement>(`.r-dialog__btn--${action}`)
    expect(button).not.toBeNull()
    button!.click()
    await nextTick()
    expect(wrapper.get('[data-testid="confirmation-result"]').text()).toBe(String(expected))
    expect(dialog.state.value.visible).toBe(false)
    expect(dialog.state.value.resolve).toBeNull()
    expect(document.body.querySelectorAll('.r-dialog__mask')).toHaveLength(0)
  })
})
