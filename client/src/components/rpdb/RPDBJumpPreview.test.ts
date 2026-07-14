import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import RPDBJumpPreview from './RPDBJumpPreview.vue'

const getPreviewMock = vi.hoisted(() => vi.fn())

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
      global: { plugins: [router] },
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
    await flushPromises()
    expect(document.body.querySelector('[data-testid="rpdb-jump-preview"]')).toBeNull()
    wrapper.unmount()
  })
})
