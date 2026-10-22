import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import UserContributionSection from './UserContributionSection.vue'

const mocks = vi.hoisted(() => ({ list: vi.fn() }))
vi.mock('@/api/userContribution', () => ({ listUserContributions: mocks.list }))
vi.mock('@/api/item', () => ({ getImageUrl: () => '/cover.jpg', resolveApiUrl: (url: string) => url }))
const RouterLinkStub = defineComponent({ props: ['to'], template: '<a :href="to"><slot /></a>' })
const entry = (id: number, kind = 'post') => ({ id, kind, title: `${kind}-${id}`, has_cover: false, cover_url: '', created_at: '2026-10-20T12:00:00Z', updated_at: '', like_count: 4, view_count: 8 })
function mountSection() {
  return mount(UserContributionSection, { props: { userId: 8, isOwnProfile: true }, global: { stubs: { RouterLink: RouterLinkStub } } })
}
afterEach(() => vi.clearAllMocks())

describe('UserContributionSection', () => {
  it('links mixed content to its real details and requests the next global page', async () => {
    mocks.list.mockResolvedValueOnce({ entries: [entry(1), entry(1, 'rpdb'), entry(1, 'item')], total: 9 })
      .mockResolvedValueOnce({ entries: [entry(7)], total: 9 })
    const wrapper = mountSection()
    await flushPromises()
    expect(wrapper.find('a[href="/community/post/1"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/rpdb/1"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/market/1"]').exists()).toBe(true)
    await wrapper.get('[aria-label="下一页帖子"]').trigger('click')
    await flushPromises()
    expect(mocks.list).toHaveBeenLastCalledWith(8, { page: 2, page_size: 6, kind: 'all', sort: 'created_at' })
    expect(wrapper.text()).toContain('第 2 / 2 页')
    await wrapper.get('[aria-label="帖子排序"] button:last-child').trigger('click')
    await flushPromises()
    expect(mocks.list).toHaveBeenLastCalledWith(8, { page: 1, page_size: 6, kind: 'all', sort: 'popular' })
    wrapper.unmount()
  })

  it('ignores a stale user response after navigating to another profile', async () => {
    let finishOld!: (value: unknown) => void
    mocks.list.mockReturnValueOnce(new Promise(resolve => { finishOld = resolve }))
      .mockResolvedValueOnce({ entries: [entry(20, 'rpdb')], total: 1 })
    const wrapper = mountSection()
    await wrapper.setProps({ userId: 20, isOwnProfile: false })
    await flushPromises()
    finishOld({ entries: [entry(8)], total: 1 })
    await flushPromises()
    expect(wrapper.find('a[href="/rpdb/20"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/community/post/8"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('我的帖子')
    wrapper.unmount()
  })

  it('shows request errors with a working retry instead of an empty success state', async () => {
    mocks.list.mockRejectedValueOnce(new Error('连接失败')).mockResolvedValueOnce({ entries: [], total: 0 })
    const wrapper = mountSection()
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('连接失败')
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('还没有发布的内容')
    wrapper.unmount()
  })
})
