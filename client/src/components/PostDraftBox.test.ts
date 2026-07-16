import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { listPostDrafts } from '@/api/post'
import PostDraftBox from './PostDraftBox.vue'

vi.mock('@/api/post', () => ({
  POST_CATEGORIES: [
    { value: 'novel', label: '小说', icon: 'ri-book-open-line' },
  ],
  listPostDrafts: vi.fn(),
}))

describe('PostDraftBox', () => {
  beforeEach(() => {
    vi.mocked(listPostDrafts).mockReset()
  })

  it('loads the authenticated users drafts from the dedicated endpoint', async () => {
    vi.mocked(listPostDrafts).mockResolvedValue({
      posts: [{
        id: 17,
        author_id: 3,
        author_name: '写作者',
        title: '未发布的调查报告',
        content: '',
        content_type: 'html',
        category: 'novel',
        status: 'draft',
        is_public: true,
        is_pinned: false,
        is_featured: false,
        view_count: 0,
        like_count: 0,
        comment_count: 0,
        favorite_count: 0,
        created_at: '2026-07-16T04:00:00Z',
        updated_at: '2026-07-16T05:00:00Z',
      }],
      total: 1,
    })

    const wrapper = mount(PostDraftBox, {
      global: { plugins: [i18n] },
    })
    await wrapper.get('.draft-trigger').trigger('click')

    await vi.waitFor(() => expect(listPostDrafts).toHaveBeenCalledTimes(1))
    expect(wrapper.get('.draft-list').text()).toContain('未发布的调查报告')
  })
})
