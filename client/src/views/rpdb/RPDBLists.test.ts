import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import RPDBLists from './RPDBLists.vue'

const updateRPDBListEntry = vi.hoisted(() => vi.fn())
const removeRPDBListEntry = vi.hoisted(() => vi.fn())
const createRPDBList = vi.hoisted(() => vi.fn())
const listRPDBLists = vi.hoisted(() => vi.fn())
const getRPDBWork = vi.hoisted(() => vi.fn())
const exportRPDBList = vi.hoisted(() => vi.fn())
const clipboardWriteText = vi.hoisted(() => vi.fn())
const toastSuccess = vi.hoisted(() => vi.fn())
const toastWarning = vi.hoisted(() => vi.fn())

vi.mock('@/stores/toast', () => ({
  useToastStore: () => ({ success: toastSuccess, error: vi.fn(), warning: toastWarning }),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ confirm: vi.fn().mockResolvedValue(true) }),
}))

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return {
    ...actual,
    createRPDBList,
    exportRPDBList,
    getRPDBWork,
    listRPDBLists,
    removeRPDBListEntry,
    resolveRPDBMediaURL: vi.fn((value?: string) => value ? `resolved:${value}` : ''),
    updateRPDBListEntry,
  }
})

describe('RPDBLists', () => {
  const defaultList = {
    id: 1,
    user_id: 1,
    name: '默认收集清单',
    description: '轻量清单测试',
    is_default: true,
    is_public: false,
    item_count: 1,
    entries: [{
      id: 1,
      list_id: 1,
      work_id: 10,
      status: 'wanted',
      priority: 1,
      quantity: 1,
      note: '',
      work: {
        id: 10,
        title: '月光灯笼',
        summary: '适合巡夜剧情。',
        cover_image: '/uploads/rpdb/demo/item-01.jpg',
      },
    }],
  }

  const guideWork = {
    id: 10,
    title: '月光灯笼',
    type: 'item_showcase',
    guide_steps: [{
      id: 1,
      work_id: 10,
      sort_order: 1,
      title: '前往守夜营地',
      body: '找到废弃哨塔附近的任务 NPC。',
      zone: '暮色森林',
      map_id: '47',
      x: 42.6,
      y: 71.3,
      label: '守夜营地',
    }],
  }

  it('renders the collection checklist workspace', async () => {
    listRPDBLists.mockReset()
    getRPDBWork.mockReset()
    createRPDBList.mockReset()
    updateRPDBListEntry.mockResolvedValue(undefined)
    removeRPDBListEntry.mockResolvedValue(undefined)
    exportRPDBList.mockReset()
    exportRPDBList.mockResolvedValue({
      format: 'tomtom',
      content: '/way #47 42.60 71.30 月光灯笼',
      missing_coordinates: [],
    })
    clipboardWriteText.mockReset()
    toastSuccess.mockReset()
    toastWarning.mockReset()
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: clipboardWriteText },
    })
    getRPDBWork.mockResolvedValue({ work: guideWork })
    listRPDBLists.mockResolvedValue({ lists: [defaultList] })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/rpdb/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/rpdb/lists')
    const wrapper = mount(RPDBLists, {
      global: { plugins: [router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('默认收集清单'))

    expect(wrapper.find('.minimal-lists-shell').exists()).toBe(true)
    expect(wrapper.text()).toContain('RP 收集清单')
    expect(wrapper.text()).toContain('把内容加入清单后，在这里追踪是否已收集，并随时进入帖子查看攻略。')
    expect(wrapper.text()).not.toContain('我的 RP 清单')
    expect(wrapper.text()).not.toContain('轻量状态表')
    expect(wrapper.text()).not.toContain('收藏夹')
    expect(wrapper.text()).not.toContain('收藏的月光灯笼')
    expect(wrapper.text()).toContain('清单列表')
    expect(wrapper.find('.list-rail').exists()).toBe(false)
    expect(wrapper.find('.list-tabs').exists()).toBe(false)
    expect(wrapper.find('[data-testid="rpdb-list-select"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-list-select"]').element).toHaveProperty('value', '0')
    expect(wrapper.text()).toContain('未收集')
    expect(wrapper.text()).toContain('已收集')
    expect(wrapper.text()).toContain('查攻略')
    expect(wrapper.text()).toContain('TomTom')

    await wrapper.find('[data-testid="tomtom-list-export"]').trigger('click')
    expect(exportRPDBList).toHaveBeenCalledWith(1, 'tomtom')
    expect(clipboardWriteText).toHaveBeenCalledWith('/way #47 42.60 71.30 月光灯笼')
    expect(toastSuccess).toHaveBeenCalledWith('TomTom 路线已复制，可在游戏内使用 /ttpaste')

    await wrapper.find('[data-testid="collection-owned-toggle"]').trigger('click')
    expect(updateRPDBListEntry).toHaveBeenCalledWith(1, 10, expect.objectContaining({ status: 'owned' }))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.entries article').classes()).toContain('collected')

    const guideLink = wrapper.find('[data-testid="open-guide-link"]')
    expect(guideLink.text()).toContain('查攻略')
    await guideLink.trigger('click')
    await vi.waitFor(() => expect(getRPDBWork).toHaveBeenCalledWith(10))
    await vi.waitFor(() => expect(document.body.querySelector('[data-testid="collection-guide-modal"]')).toBeTruthy())
    expect(document.body.textContent).toContain('前往守夜营地')
    expect(document.body.textContent).toContain('/way #47 42.60 71.30 守夜营地')

    const detailLink = wrapper.find('[data-testid="open-work-link"]')
    expect(detailLink.attributes('href')).toBe('/rpdb/10?from=collection')
    expect(detailLink.text()).toContain('帖子')

    await wrapper.find('[data-testid="remove-collection-entry"]').trigger('click')
    expect(removeRPDBListEntry).toHaveBeenCalledWith(1, 10)
  })

  it('switches collection checklists from a single-select dropdown', async () => {
    listRPDBLists.mockReset()
    listRPDBLists.mockResolvedValue({
      lists: [
        defaultList,
        {
          ...defaultList,
          id: 2,
          name: '幻化待收集',
          description: '第二份清单',
          item_count: 0,
          entries: [],
        },
      ],
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/rpdb/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/rpdb/lists')
    const wrapper = mount(RPDBLists, {
      global: { plugins: [router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('默认收集清单'))
    await wrapper.find('[data-testid="rpdb-list-select"]').setValue('1')

    expect(wrapper.text()).toContain('幻化待收集')
    expect(wrapper.text()).toContain('第二份清单')
    expect(wrapper.text()).toContain('清单还是空的')
  })

  it('reports an empty TomTom export without writing an undefined clipboard value', async () => {
    listRPDBLists.mockReset()
    exportRPDBList.mockReset()
    clipboardWriteText.mockReset()
    toastWarning.mockReset()
    listRPDBLists.mockResolvedValue({ lists: [defaultList] })
    exportRPDBList.mockResolvedValue({
      format: 'tomtom',
      content: '',
      missing_coordinates: [{ work_id: 10, title: '月光灯笼' }],
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: clipboardWriteText },
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/rpdb/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/rpdb/lists')
    const wrapper = mount(RPDBLists, { global: { plugins: [router] } })

    await vi.waitFor(() => expect(wrapper.text()).toContain('默认收集清单'))
    await wrapper.find('[data-testid="tomtom-list-export"]').trigger('click')

    expect(clipboardWriteText).not.toHaveBeenCalled()
    expect(toastWarning).toHaveBeenCalledWith('当前清单没有可导出的 TomTom 坐标')
  })

  it('creates a new collection checklist from a modal', async () => {
    listRPDBLists.mockReset()
    createRPDBList.mockReset()
    createRPDBList.mockResolvedValue({ list: { id: 2, name: '测试新清单' } })
    listRPDBLists
      .mockResolvedValueOnce({ lists: [defaultList] })
      .mockResolvedValueOnce({ lists: [defaultList, { ...defaultList, id: 2, name: '测试新清单', item_count: 0, entries: [] }] })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/rpdb/lists', component: { template: '<div />' } },
        { path: '/rpdb/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/rpdb/lists')
    const wrapper = mount(RPDBLists, {
      global: { plugins: [router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('默认收集清单'))
    expect(wrapper.find('[data-testid="rpdb-list-create-name"]').exists()).toBe(false)

    await wrapper.find('[data-testid="rpdb-list-create-open"]').trigger('click')
    await vi.waitFor(() => expect(document.body.querySelector('[data-testid="rpdb-list-create-name"]')).toBeTruthy())
    ;(document.body.querySelector('[data-testid="rpdb-list-create-name"]') as HTMLInputElement).value = '测试新清单'
    ;(document.body.querySelector('[data-testid="rpdb-list-create-name"]') as HTMLInputElement).dispatchEvent(new Event('input', { bubbles: true }))
    ;(document.body.querySelector('[data-testid="rpdb-list-create-description"]') as HTMLTextAreaElement).value = '测试详情'
    ;(document.body.querySelector('[data-testid="rpdb-list-create-description"]') as HTMLTextAreaElement).dispatchEvent(new Event('input', { bubbles: true }))
    await wrapper.vm.$nextTick()
    ;(document.body.querySelector('[data-testid="rpdb-list-create-submit"]') as HTMLButtonElement).click()

    await vi.waitFor(() => expect(createRPDBList).toHaveBeenCalledWith('测试新清单', '测试详情'))
    await vi.waitFor(() => expect(wrapper.text()).toContain('测试新清单'))
    await vi.waitFor(() => expect(document.body.querySelector('[data-testid="rpdb-list-create-name"]')).toBeFalsy())
  })
})
