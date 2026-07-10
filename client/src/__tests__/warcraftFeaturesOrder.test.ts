import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import WarcraftFeatures from '@/views/WarcraftFeatures.vue'

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  invoke: vi.fn(),
  listen: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mocks.push,
  }),
}))

vi.mock('@tauri-apps/api/core', () => ({
  invoke: mocks.invoke,
}))

vi.mock('@tauri-apps/api/event', () => ({
  listen: mocks.listen,
}))

vi.mock('@tauri-apps/plugin-shell', () => ({
  open: vi.fn(),
}))

vi.mock('@/api/addon', () => ({
  getAddonDownloadUrl: vi.fn(),
  getAddonLatest: vi.fn(),
  getTRP3Latest: vi.fn(),
}))

vi.mock('@/composables/useDialog', () => ({
  dialog: {
    confirm: vi.fn().mockResolvedValue(false),
  },
}))

vi.mock('@/stores/toast', () => ({
  useToastStore: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
  }),
}))

describe('Warcraft feature section order', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mocks.listen.mockResolvedValue(vi.fn())
    mocks.invoke.mockResolvedValue([])
  })

  it('shows shortcuts before plugin installation', async () => {
    const wrapper = shallowMount(WarcraftFeatures)
    await flushPromises()

    expect(wrapper.findAll('h2').map(node => node.text())).toEqual([
      '选择游戏目录',
      '功能快捷入口',
      '插件安装',
    ])
  })
})
