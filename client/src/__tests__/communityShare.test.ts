import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import CommunityMain from '@/views/community/CommunityMain.vue'
import PostDetail from '@/views/community/PostDetail.vue'

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  back: vi.fn(),
  shareRouteLink: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
  getPost: vi.fn(),
  listComments: vi.fn(),
  listPosts: vi.fn(),
  listEvents: vi.fn(),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: () => ({
      push: mocks.push,
      back: mocks.back,
    }),
    useRoute: () => ({
      params: { id: '42' },
      query: {},
    }),
  }
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/utils/share', () => ({
  buildPostShareText: (html: string) => html.replace(/<[^>]+>/g, '').trim(),
  shareRouteLink: mocks.shareRouteLink,
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
  }),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    confirm: vi.fn().mockResolvedValue(false),
  }),
}))

vi.mock('@/stores/emote', () => ({
  useEmoteStore: () => ({
    loadPacks: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/api/post', () => ({
  POST_CATEGORIES: [{ value: 'other', label: 'Other' }],
  getPost: mocks.getPost,
  listComments: mocks.listComments,
  listPosts: mocks.listPosts,
  listEvents: mocks.listEvents,
  likePost: vi.fn(),
  unlikePost: vi.fn(),
  favoritePost: vi.fn(),
  unfavoritePost: vi.fn(),
  deletePost: vi.fn(),
  createComment: vi.fn(),
  deleteComment: vi.fn(),
  likeComment: vi.fn(),
  unlikeComment: vi.fn(),
}))

vi.mock('@/api/guild', () => ({
  getGuild: vi.fn(),
}))

vi.mock('@/api/item', () => ({
  getImageUrl: vi.fn().mockReturnValue(''),
  resolveApiUrl: vi.fn((url?: string) => url || ''),
}))

vi.mock('@/api/safety', () => ({
  createContentReport: vi.fn(),
  createUserBlock: vi.fn(),
}))

vi.mock('@/utils/imagePreview', () => ({
  attachImagePreview: vi.fn(),
}))

vi.mock('@/utils/userNameStyle', () => ({
  buildNameStyle: vi.fn().mockReturnValue({}),
}))

vi.mock('@/utils/emote', () => ({
  renderEmoteContent: vi.fn((content: string) => content),
}))

vi.mock('@/utils/jumpLink', () => ({
  handleJumpLinkClick: vi.fn().mockReturnValue(false),
  sanitizeJumpLinks: vi.fn(),
  hydrateJumpCardImages: vi.fn(),
}))

vi.mock('@/utils/download', () => ({
  handleAttachmentClick: vi.fn(),
}))

const detailPost = {
  id: 42,
  author_id: 9,
  title: 'Detail post',
  content: '<p>Detail body</p>',
  category: 'other',
  created_at: '2026-07-10T00:00:00Z',
  region: '',
  address: '',
  like_count: 2,
  favorite_count: 3,
  view_count: 4,
  comment_count: 0,
}

const listPost = {
  id: 7,
  author_id: 3,
  author_name: 'List author',
  author_avatar: '',
  author_name_color: '',
  author_name_bold: false,
  author_forum_level: 1,
  author_forum_level_name: '',
  author_forum_level_color: '',
  author_forum_level_bold: false,
  title: 'List post',
  content: '<p>List body</p>',
  category: 'other',
  created_at: '2026-07-10T00:00:00Z',
  updated_at: '2026-07-10T00:00:00Z',
  region: '',
  address: '',
  comment_count: 0,
  cover_image: '',
  cover_image_url: '',
  cover_image_updated_at: '',
  is_featured: false,
}

describe('community desktop sharing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mocks.getPost.mockResolvedValue({
      post: { ...detailPost },
      author_name: 'Detail author',
      author_avatar: '',
      author_name_color: '',
      author_name_bold: false,
      author_forum_level: 1,
      author_forum_level_name: '',
      author_forum_level_color: '',
      author_forum_level_bold: false,
      liked: false,
      favorited: false,
    })
    mocks.listComments.mockResolvedValue({ comments: [] })
    mocks.listEvents.mockResolvedValue({ events: [] })
    mocks.listPosts.mockImplementation(async (params: { is_pinned?: boolean }) => ({
      posts: params.is_pinned ? [] : [{ ...listPost }],
      total: params.is_pinned ? 0 : 1,
    }))
    mocks.shareRouteLink.mockResolvedValue({
      method: 'copied',
      url: 'https://totalrpbox.com/posts/42',
    })
  })

  it('shares the current detail post without navigating', async () => {
    const wrapper = shallowMount(PostDetail)
    await flushPromises()

    await wrapper.get('button.share-btn').trigger('click')
    await flushPromises()

    expect(mocks.shareRouteLink).toHaveBeenCalledWith({
      path: '/posts/42',
      title: 'Detail post',
      text: 'Detail body',
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('community.share.copied')
    expect(mocks.push).not.toHaveBeenCalled()
  })

  it('stops propagation when sharing a post card', async () => {
    const wrapper = shallowMount(CommunityMain)
    await flushPromises()
    const stopPropagation = vi.spyOn(Event.prototype, 'stopPropagation')

    await wrapper.get('button.card-share-btn').trigger('click')
    await flushPromises()

    expect(stopPropagation).toHaveBeenCalled()
    expect(mocks.shareRouteLink).toHaveBeenCalledWith({
      path: '/posts/7',
      title: 'List post',
      text: 'List body',
    })
    expect(mocks.push).not.toHaveBeenCalled()
  })

  it('shows an error toast when sharing fails', async () => {
    mocks.shareRouteLink.mockRejectedValue(new Error('share failed'))
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const wrapper = shallowMount(PostDetail)
    await flushPromises()

    await wrapper.get('button.share-btn').trigger('click')
    await flushPromises()

    expect(mocks.toastError).toHaveBeenCalledWith('community.share.failed')
    expect(consoleError).toHaveBeenCalledWith('分享帖子失败:', expect.any(Error))
    expect(mocks.push).not.toHaveBeenCalled()
  })
})
