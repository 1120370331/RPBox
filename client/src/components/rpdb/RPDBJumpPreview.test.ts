import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import RPDBJumpPreview from './RPDBJumpPreview.vue'

const getPreviewMock = vi.hoisted(() => vi.fn())

beforeEach(() => {
  i18n.global.locale.value = 'zh-CN'
})

vi.mock('@/api/rpdb', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/api/rpdb')>()
  return {
    ...original,
    getRPDBWorkPreview: getPreviewMock,
    resolveRPDBMediaURL: (value?: string) => value || '',
  }
})

vi.mock('@/api/item', () => ({
  resolveApiUrl: (value?: string) => value || '',
}))

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
  vi.useRealTimers()
  Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true })
})

describe('RPDBJumpPreview', () => {
  it('shows a pointer-side preview and closes when the pointer leaves', async () => {
    vi.useFakeTimers()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()
    getPreviewMock.mockResolvedValue({
      work: {
        id: 10,
        type: 'home_showcase',
        title: '港湾小屋',
        summary: '开放参观的临海家宅',
        author_name: '潮汐旅人',
        cover_image: '/uploads/home.jpg',
        view_count: 88,
        like_count: 9,
        favorite_count: 6,
        list_count: 3,
      },
    })
    const wrapper = mount(RPDBJumpPreview, {
      attachTo: document.body,
      global: { plugins: [router, i18n] },
    })
    const card = document.createElement('span')
    card.setAttribute('data-jump-type', 'rpdb_work')
    card.setAttribute('data-jump-href', '/rpdb/10')
    card.setAttribute('data-jump-rpdb-type', 'home_showcase')
    card.setAttribute('data-jump-title', '港湾小屋')
    document.body.appendChild(card)

    card.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, clientX: 240, clientY: 160 }))
    await vi.advanceTimersByTimeAsync(180)
    await flushPromises()

    const preview = document.body.querySelector('[data-testid="rpdb-jump-preview"]')
    expect(getPreviewMock).toHaveBeenCalledWith(10)
    expect(preview?.classList.contains('rpdb-jump-preview--home')).toBe(true)
    expect(preview?.textContent).toContain('开放参观的临海家宅')
    expect(preview?.textContent).toContain('88')

    card.dispatchEvent(new MouseEvent('mouseout', { bubbles: true, relatedTarget: document.body }))
    await vi.advanceTimersByTimeAsync(130)
    expect(document.body.querySelector('[data-testid="rpdb-jump-preview"]')).toBeNull()
    wrapper.unmount()
  })

  it('shows and copies a transmog share code from the interactive preview', async () => {
    vi.useFakeTimers()
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()
    getPreviewMock.mockResolvedValue({
      work: {
        id: 12,
        type: 'transmog',
        title: '银月秘法使',
        summary: '金红配色的法师幻化',
        author_name: '秘法裁缝',
        extra: JSON.stringify({ share_code: 'TRANSMOG:HEAD=34339;CHEST=34202' }),
        view_count: 12,
        like_count: 4,
        favorite_count: 3,
        list_count: 2,
      },
    })
    const wrapper = mount(RPDBJumpPreview, {
      attachTo: document.body,
      global: { plugins: [router, i18n] },
    })
    const card = document.createElement('span')
    card.setAttribute('data-jump-type', 'rpdb_work')
    card.setAttribute('data-jump-href', '/rpdb/12')
    card.setAttribute('data-jump-rpdb-type', 'transmog')
    document.body.appendChild(card)

    card.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, clientX: 240, clientY: 160 }))
    await vi.advanceTimersByTimeAsync(180)
    await flushPromises()

    const codeArea = document.body.querySelector('[data-testid="rpdb-jump-preview-code"]')
    expect(codeArea?.textContent).toContain('TRANSMOG:HEAD=34339;CHEST=34202')
    const copyButton = document.body.querySelector<HTMLButtonElement>('[data-testid="copy-transmog-share-code"]')
    copyButton?.click()
    await flushPromises()

    expect(writeText).toHaveBeenCalledWith('TRANSMOG:HEAD=34339;CHEST=34202')
    expect(copyButton?.textContent).toContain('已复制')
    wrapper.unmount()
  })
})
