import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RPDBDetail from './RPDBDetail.vue'
import i18n from '@/i18n'

const getRPDBWork = vi.hoisted(() => vi.fn())
const listRPDBComments = vi.hoisted(() => vi.fn())
const listRPDBWorkRecommendations = vi.hoisted(() => vi.fn())
const clipboardWriteText = vi.hoisted(() => vi.fn())
const addRPDBWorkToList = vi.hoisted(() => vi.fn())
const createRPDBList = vi.hoisted(() => vi.fn())
const favoriteRPDBWork = vi.hoisted(() => vi.fn())
const likeRPDBWork = vi.hoisted(() => vi.fn())
const listRPDBLists = vi.hoisted(() => vi.fn())
const toastSuccess = vi.hoisted(() => vi.fn())
const toastError = vi.hoisted(() => vi.fn())
const toastWarning = vi.hoisted(() => vi.fn())
const createContentReport = vi.hoisted(() => vi.fn())
const createUserBlock = vi.hoisted(() => vi.fn())
const deleteRPDBComment = vi.hoisted(() => vi.fn())
const confirmDialog = vi.hoisted(() => vi.fn())

vi.mock('@/api/emote', () => ({
  listEmotePacks: vi.fn().mockResolvedValue({ packs: [] }),
  resolveEmoteUrl: (url: string) => url,
}))

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return {
    ...actual,
    getRPDBWork,
    listRPDBComments,
    listRPDBWorkRecommendations,
    addRPDBWorkToList,
    createRPDBList,
    createRPDBComment: vi.fn(),
    deleteRPDBComment,
    favoriteRPDBWork,
    likeRPDBWork,
    listRPDBLists,
    unfavoriteRPDBWork: vi.fn(),
    unlikeRPDBWork: vi.fn(),
    verifyRPDBWork: vi.fn(),
  }
})

vi.mock('@/stores/toast', () => ({
  useToastStore: () => ({
    success: toastSuccess,
    error: toastError,
    warning: toastWarning,
  }),
}))

vi.mock('@/api/safety', async () => {
  const actual = await vi.importActual<typeof import('@/api/safety')>('@/api/safety')
  return { ...actual, createContentReport, createUserBlock }
})

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ confirm: confirmDialog }),
}))

describe('RPDBDetail', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
    clipboardWriteText.mockReset()
    addRPDBWorkToList.mockReset()
    createRPDBList.mockReset()
    favoriteRPDBWork.mockReset()
    likeRPDBWork.mockReset()
    listRPDBLists.mockReset()
    listRPDBWorkRecommendations.mockReset()
    toastSuccess.mockReset()
    toastError.mockReset()
    toastWarning.mockReset()
    createContentReport.mockReset()
    createUserBlock.mockReset()
    deleteRPDBComment.mockReset()
    confirmDialog.mockReset()
    createContentReport.mockResolvedValue({ message: '举报已提交', report_id: 9, submitted_report: true })
    createUserBlock.mockResolvedValue({ message: '已屏蔽', submitted_report: false })
    deleteRPDBComment.mockResolvedValue(undefined)
    confirmDialog.mockResolvedValue(true)
    localStorage.setItem('token', 'viewer-token')
    localStorage.setItem('user', JSON.stringify({ id: 2, username: 'viewer', role: 'user' }))
    addRPDBWorkToList.mockResolvedValue({})
    createRPDBList.mockResolvedValue({ list: { id: 7, user_id: 1, name: '新建测试清单', description: '', is_default: false, is_public: false, item_count: 0, entries: [] } })
    favoriteRPDBWork.mockResolvedValue({})
    likeRPDBWork.mockResolvedValue({})
    listRPDBLists.mockResolvedValue({
      lists: [
        { id: 5, user_id: 1, name: '剧情道具清单', description: '道具收集', is_default: true, is_public: false, item_count: 2, entries: [] },
        { id: 6, user_id: 1, name: '幻化待刷', description: '幻化计划', is_default: false, is_public: false, item_count: 4, entries: [] },
      ],
    })
    listRPDBWorkRecommendations.mockResolvedValue({ recommendations: [] })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: clipboardWriteText },
    })
    getRPDBWork.mockResolvedValue({
      work: {
        id: 1,
        author_id: 1,
        author_name: 'rpdb_demo',
        type: 'item_showcase',
        title: '月光灯笼',
        slug: 'moon-lantern',
        summary: '适合巡夜剧情的暖色光源。',
        content: '<p>正文说明</p>',
        content_type: 'html',
        cover_image: '/uploads/rpdb/demo/item-01.jpg',
        rp_use_cases: '巡夜、调查和墓地剧情',
        effect_description: '稳定暖色光源',
        restrictions: '',
        extra: '',
        game_version: '11.2.7',
        expansion: '至暗之夜',
        availability_status: 'available',
        bind_type: 'yes',
        faction: 'neutral',
        armor_type: '',
        verification_status: 'verified',
        verified_count: 18,
        outdated_count: 1,
        status: 'published',
        is_public: true,
        review_status: 'approved',
        version: 3,
        view_count: 100,
        like_count: 86,
        favorite_count: 34,
        comment_count: 0,
        list_count: 12,
        media_count: 1,
        is_liked: false,
        is_favorited: false,
        in_collection_list: false,
        references: [{
          id: 9,
          external_type: 'toy',
          external_id: '12345',
          name: '旅店留言簿',
          icon: '/icons/inn-book.jpg',
          description: '供来访者记录故事与留言。',
          acquisition_method: '旅店老板赠送',
          quality: 'rare',
        }],
        media: [{
          id: 2,
          work_id: 1,
          type: 'image',
          url: '/uploads/rpdb/demo/item-02.jpg',
          caption: '效果图',
          sort_order: 2,
        }],
        transmog_slots: [],
        guide_steps: [{
          id: 1,
          sort_order: 1,
          title: '前往守夜营地',
          body: '找到任务 NPC。',
          zone: '暮色森林',
          map_id: '47',
          x: 42.6,
          y: 71.3,
        }],
        tags: [
          { id: 101, name: '联盟风格', color: '2F66C8' },
          { id: 103, name: '库尔提拉斯风格', color: '356A8A' },
        ],
        created_at: '2026-07-10T00:00:00Z',
        updated_at: '2026-07-10T00:00:00Z',
      },
    })
    listRPDBComments.mockResolvedValue({ comments: [] })
  })

  it('uses an upper media hero and lower editorial guide workspace', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
      attachTo: document.body,
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    await wrapper.get('[data-testid="floating-toc-collapse"]').trigger('click')

    expect(wrapper.find('[data-testid="detail-hero"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="detail-lower"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="guide-reading"]').exists()).toBe(true)
    expect(wrapper.find('.minimal-detail-shell').exists()).toBe(true)
    expect(wrapper.find('[data-testid="floating-toc"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-edit-button"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('/ttpaste')
    expect(wrapper.text()).toContain('联盟风格')
    expect(wrapper.text()).toContain('库尔提拉斯风格')
    expect(wrapper.text()).not.toContain('资料片')
    expect(wrapper.text()).not.toContain('版本未标注')
    expect(wrapper.text()).not.toContain('至暗之夜')
    expect(wrapper.text()).not.toContain('11.2.7')
    expect(wrapper.text()).toContain('实际效果与 RP 用途')
    expect(wrapper.text()).toContain('路线与坐标步骤')
    const reference = wrapper.find('[data-testid="rpdb-reference-object"]')
    expect(reference.exists()).toBe(true)
    expect(reference.find('[data-testid="rpdb-reference-icon"]').classes()).toContain('quality-rare')
    expect(reference.find('[data-testid="rpdb-reference-icon"] img').attributes('src')).toBe('/icons/inn-book.jpg')
    expect(reference.find('[data-testid="rpdb-reference-name"]').text()).toBe('旅店留言簿')
    expect(reference.element.tagName).toBe('DIV')
    expect(reference.text()).toContain('玩具 · 旅店老板赠送')
    expect(reference.text()).toContain('供来访者记录故事与留言。')
    const metadata = wrapper.find('.hero-metadata').text()
    expect(metadata).toContain('获取状态可获取')
    expect(metadata).toContain('是否绑定是')
    expect(metadata).toContain('阵营不限')
    expect(metadata).not.toContain('available')
    expect(metadata).not.toContain('neutral')
    expect(wrapper.text()).not.toContain('人确认有效')
    expect(wrapper.text()).not.toContain('社区验证')
    const metrics = wrapper.get('[data-testid="rpdb-detail-metrics"]')
    expect(metrics.text()).toContain('浏览100')
    expect(metrics.text()).toContain('点赞86')
    expect(metrics.text()).toContain('收藏34')
    expect(metrics.text()).toContain('清单12')
  })

  it('shows weighted related recommendations directly above the discussion section', async () => {
    listRPDBWorkRecommendations.mockResolvedValueOnce({
      recommendations: [
        {
          id: 2,
          author_id: 3,
          author_name: '巡夜人',
          type: 'transmog',
          title: '暮色巡林幻化',
          slug: 'dusk-ranger',
          summary: '浏览本物品的玩家也收藏了这套幻化。',
          content: '',
          content_type: 'html',
          cover_image: '/uploads/rpdb/demo/transmog.jpg',
          rp_use_cases: '',
          effect_description: '',
          restrictions: '',
          extra: '',
          game_version: '',
          expansion: '',
          availability_status: 'available',
          bind_type: '',
          faction: 'neutral',
          armor_type: 'leather',
          verification_status: 'verified',
          verified_count: 2,
          outdated_count: 0,
          status: 'published',
          is_public: true,
          visibility: 'public',
          review_status: 'approved',
          version: 1,
          view_count: 18,
          like_count: 7,
          favorite_count: 5,
          comment_count: 0,
          list_count: 3,
          media_count: 1,
          is_liked: false,
          is_favorited: false,
          in_collection_list: false,
          recommendation_score: 32,
          recommendation_reasons: ['2 位相关玩家收藏', '1 位相关玩家加入清单'],
          recommendation_signals: {
            likes: 0,
            favorites: 2,
            views: 0,
            lists: 1,
            creators: 0,
            same_author: false,
          },
          created_at: '2026-07-10T00:00:00Z',
          updated_at: '2026-07-10T00:00:00Z',
        },
      ],
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.find('[data-testid="rpdb-recommendations"]').exists()).toBe(true))
    const related = wrapper.get('[data-testid="rpdb-recommendations"]')
    const discussion = wrapper.get('#rpdb-section-discussion')
    expect(related.element.compareDocumentPosition(discussion.element)).toBe(Node.DOCUMENT_POSITION_FOLLOWING)
    expect(related.text()).toContain('暮色巡林幻化')
    expect(related.text()).toContain('2 位相关玩家收藏')
    expect(related.text()).toContain('32')
    expect(listRPDBWorkRecommendations).toHaveBeenCalledWith(1)
  })

  it('keeps the floating helper collapsed when the viewport cannot fit it beside the article', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    const panel = wrapper.find('[data-testid="floating-toc"]')
    const toggle = wrapper.find('[data-testid="floating-toc-collapse"]')

    expect(toggle.exists()).toBe(true)
    expect(toggle.attributes('aria-label')).toBe('展开悬浮目录')
    expect(panel.classes()).toContain('is-collapsed')

    await toggle.trigger('click')

    expect(panel.classes()).not.toContain('is-collapsed')
    expect(wrapper.find('[data-testid="floating-toc-content"]').exists()).toBe(true)
    expect(toggle.attributes('aria-label')).toBe('收起悬浮目录')

    window.dispatchEvent(new Event('resize'))
    await wrapper.vm.$nextTick()

    expect(panel.classes()).toContain('is-collapsed')
    expect(wrapper.find('[data-testid="floating-toc-content"]').exists()).toBe(false)
  })

  it('returns to collection lists when opened from a collection checklist', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb/lists', component: { template: '<div />' } },
        { path: '/rpdb/:id', component: RPDBDetail },
      ],
    })
    await router.push('/rpdb/1?from=collection')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    await wrapper.find('[data-testid="detail-back-button"]').trigger('click')

    await vi.waitFor(() => expect(router.currentRoute.value.fullPath).toBe('/rpdb/lists'))
  })

  it('returns directly to discovery instead of the previous editor page', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/rpdb/create', component: { template: '<div />' } },
        { path: '/rpdb/:id', component: RPDBDetail },
      ],
    })
    await router.push('/rpdb/create')
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    await wrapper.find('[data-testid="detail-back-button"]').trigger('click')

    await vi.waitFor(() => expect(router.currentRoute.value.fullPath).toBe('/rpdb'))
  })

  it.each([
    ['the author', { id: 1, username: 'rpdb_demo', role: 'user' }],
    ['an administrator', { id: 99, username: 'admin', role: 'admin' }],
  ])('shows the bottom edit action to %s', async (_label, user) => {
    localStorage.setItem('user', JSON.stringify(user))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb/:id', component: RPDBDetail },
        { path: '/rpdb/:id/edit', component: { template: '<div />' } },
      ],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    const editButton = wrapper.get('[data-testid="rpdb-edit-button"]')
    expect(editButton.text()).toContain('编辑帖子')
    await editButton.trigger('click')

    await vi.waitFor(() => expect(router.currentRoute.value.fullPath).toBe('/rpdb/1/edit'))
  })

  it('does not show the edit action to a moderator who is not the author', async () => {
    localStorage.setItem('user', JSON.stringify({ id: 99, username: 'moderator', role: 'moderator' }))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    expect(wrapper.find('[data-testid="rpdb-edit-button"]').exists()).toBe(false)
  })

  it('provides floating like, favorite and collection list actions with feedback', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    await wrapper.get('[data-testid="floating-toc-collapse"]').trigger('click')
    const floatingContent = wrapper.get('[data-testid="floating-toc-content"]')
    const collectionButtons = floatingContent.findAll('button').filter(button => (
      button.text().includes('加入清单')
      || button.text().includes('加入收集清单')
      || button.text().includes('已在清单')
    ))
    expect(collectionButtons).toHaveLength(1)

    await wrapper.find('[data-testid="floating-like-button"]').trigger('click')
    expect(likeRPDBWork).toHaveBeenCalledWith(1)
    expect(toastSuccess).toHaveBeenCalledWith('已点赞')

    await wrapper.find('[data-testid="floating-favorite-button"]').trigger('click')
    expect(favoriteRPDBWork).toHaveBeenCalledWith(1)
    expect(toastSuccess).toHaveBeenCalledWith('已收藏')

    await wrapper.find('[data-testid="floating-list-button"]').trigger('click')
    expect(listRPDBLists).toHaveBeenCalled()
    await vi.waitFor(() => expect(document.body.querySelector('[data-testid="rpdb-list-picker"]')).toBeTruthy())
    expect(document.body.textContent).toContain('剧情道具清单')
    expect(document.body.textContent).toContain('幻化待刷')

    ;(document.body.querySelectorAll('[data-testid="rpdb-list-picker-option"]')[0] as HTMLButtonElement).click()

    expect(addRPDBWorkToList).toHaveBeenCalledWith(1, 'wanted', 5)
    await vi.waitFor(() => expect(toastSuccess).toHaveBeenCalledWith('已加入「剧情道具清单」'))
    await vi.waitFor(() => expect(document.body.querySelector('[data-testid="rpdb-list-picker"]')).toBeFalsy())
    wrapper.unmount()
  })

  it('reuses the community plaza discussion layout for RPDB comments', async () => {
    listRPDBComments.mockResolvedValueOnce({
      comments: [
        {
          id: 11,
          work_id: 1,
          author_id: 2,
          author_name: '旅店老板',
          author_avatar: '/avatar.png',
          author_name_color: '#8b5cf6',
          author_name_bold: true,
          author_forum_level: 3,
          author_forum_level_name: '常驻写手',
          author_forum_level_color: '#7B9BC7',
          author_forum_level_bold: true,
          content: '这个灯笼适合夜巡。',
          like_count: 2,
          liked: true,
          created_at: new Date().toISOString(),
        },
        {
          id: 12,
          work_id: 1,
          author_id: 3,
          author_name: '巡夜人',
          parent_id: 11,
          content: '我也试过，效果很好。',
          like_count: 0,
          created_at: new Date().toISOString(),
        },
      ],
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('旅店老板'))

    expect(wrapper.find('.comments-section').exists()).toBe(true)
    expect(wrapper.find('.comments-title').exists()).toBe(true)
    expect(wrapper.find('.comment-badge').text()).toBe('2')
    expect(wrapper.find('.comment-item').exists()).toBe(true)
    expect(wrapper.find('.comment-avatar img').attributes('src')).toBe('/avatar.png')
    expect(wrapper.find('.comment-meta .like-btn-inline').classes()).toContain('active')
    expect(wrapper.find('.replies-list .reply-item').exists()).toBe(true)
    expect(wrapper.text()).toContain('回复 @旅店老板')
    expect(wrapper.find('.comment-input-box').exists()).toBe(true)
    expect(wrapper.find('.input-footer .emoji-btn').exists()).toBe(true)
    expect(wrapper.find('[data-testid="delete-rpdb-comment-11"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="report-rpdb-comment-11"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="delete-rpdb-comment-12"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="report-rpdb-comment-12"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="block-rpdb-comment-author-12"]').exists()).toBe(true)

    await wrapper.find('.comment-actions .reply-btn').trigger('click')
    expect(wrapper.findComponent({ name: 'CommentReplyBox' }).exists()).toBe(true)
  })

  it('lets the work author delete root comments and replies', async () => {
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'rpdb_demo', role: 'user' }))
    listRPDBComments.mockResolvedValueOnce({
      comments: [
        {
          id: 21,
          work_id: 1,
          author_id: 2,
          author_name: '旅店老板',
          content: '作者可以管理这条评论。',
          like_count: 0,
          created_at: new Date().toISOString(),
        },
        {
          id: 22,
          work_id: 1,
          author_id: 3,
          author_name: '巡夜人',
          parent_id: 21,
          content: '作者也可以管理回复。',
          like_count: 0,
          created_at: new Date().toISOString(),
        },
      ],
    })
    listRPDBComments.mockResolvedValueOnce({ comments: [] })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('作者可以管理这条评论'))
    expect(wrapper.find('[data-testid="delete-rpdb-comment-21"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="delete-rpdb-comment-22"]').exists()).toBe(true)

    await wrapper.get('[data-testid="delete-rpdb-comment-22"]').trigger('click')
    await vi.waitFor(() => expect(deleteRPDBComment).toHaveBeenCalledWith(22))
    expect(confirmDialog).toHaveBeenCalled()
    expect(toastSuccess).toHaveBeenCalledWith('评论已删除')
  })

  it('reports and blocks RPDB comment authors for root comments and replies', async () => {
    listRPDBComments.mockResolvedValueOnce({
      comments: [
        {
          id: 31,
          work_id: 1,
          author_id: 3,
          author_name: '可疑旅人',
          content: '这是一条需要举报的评论。',
          like_count: 0,
          created_at: new Date().toISOString(),
        },
        {
          id: 32,
          work_id: 1,
          author_id: 4,
          author_name: '陌生访客',
          parent_id: 31,
          content: '这是一条需要屏蔽作者的回复。',
          like_count: 0,
          created_at: new Date().toISOString(),
        },
      ],
    })
    listRPDBComments.mockResolvedValue({ comments: [] })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('可疑旅人'))
    await wrapper.get('[data-testid="report-rpdb-comment-31"]').trigger('click')
    const reportDialog = wrapper.getComponent({ name: 'SafetyReportDialog' })
    expect(reportDialog.props('targetType')).toBe('rpdb_comment')
    reportDialog.vm.$emit('submit', {
      reason: 'abuse',
      detail: '评论存在人身攻击',
      hideTarget: false,
      blockAuthor: false,
      submitReport: true,
    })
    await vi.waitFor(() => expect(createContentReport).toHaveBeenCalledWith({
      target_type: 'rpdb_comment',
      target_id: 31,
      reason: 'abuse',
      detail: '评论存在人身攻击',
      hide_target: false,
      block_author: false,
      submit_report: true,
    }))

    await wrapper.get('[data-testid="block-rpdb-comment-author-32"]').trigger('click')
    await vi.waitFor(() => expect(createUserBlock).toHaveBeenCalledWith(
      4,
      expect.stringContaining('陌生访客'),
    ))
    expect(toastSuccess).toHaveBeenCalledWith('已屏蔽该作者，相关评论已隐藏')
  })

  it('can create a collection checklist directly from the list picker and add the work', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    await wrapper.get('[data-testid="floating-toc-collapse"]').trigger('click')
    await wrapper.find('[data-testid="floating-list-button"]').trigger('click')
    await vi.waitFor(() => expect(document.body.querySelector('[data-testid="rpdb-list-picker-create"]')).toBeTruthy())

    const input = document.body.querySelector('[data-testid="rpdb-list-picker-create-input"]') as HTMLInputElement
    input.value = '新建测试清单'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await wrapper.vm.$nextTick()
    ;(document.body.querySelector('[data-testid="rpdb-list-picker-create-button"]') as HTMLButtonElement).click()

    await vi.waitFor(() => expect(createRPDBList).toHaveBeenCalledWith('新建测试清单'))
    await vi.waitFor(() => expect(addRPDBWorkToList).toHaveBeenCalledWith(1, 'wanted', 7))
    await vi.waitFor(() => expect(toastSuccess).toHaveBeenCalledWith('已加入「新建测试清单」'))
    wrapper.unmount()
  })

  it('keeps the article width independent from the floating helper panel', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/rpdb/RPDBDetail.vue'), 'utf8')

    expect(source).toContain('.floating-toc-panel{position:fixed')
    expect(source).not.toContain('.detail-page{max-width:1380px;margin:auto;padding-right:')
    expect(source).not.toContain('.detail-page{padding-right:')
  })

  it('renders a prominent home share code copy action for housing posts', async () => {
    getRPDBWork.mockResolvedValueOnce({
      work: {
        id: 2,
        author_id: 1,
        author_name: 'rpdb_home',
        type: 'home_showcase',
        title: '炉火旅店',
        slug: 'hearth-inn',
        summary: '可导入的旅店家宅。',
        content: '<p>旅店正文</p>',
        content_type: 'html',
        cover_image: '/uploads/rpdb/demo/home-01.jpg',
        rp_use_cases: '',
        effect_description: '',
        restrictions: '',
        extra: JSON.stringify({
          server: '白银之手',
          region: '艾尔文森林',
          home_style: '旅店',
          share_code: 'RPBOX-HOME-INN-001',
          visit_notes: '加好友后参观',
        }),
        game_version: '11.2.7',
        expansion: '至暗之夜',
        availability_status: '加好友后参观',
        bind_type: '',
        faction: 'neutral',
        armor_type: '',
        verification_status: 'verified',
        verified_count: 0,
        outdated_count: 0,
        status: 'published',
        is_public: true,
        review_status: 'approved',
        version: 1,
        view_count: 10,
        like_count: 2,
        favorite_count: 1,
        comment_count: 0,
        list_count: 0,
        media_count: 1,
        is_liked: false,
        is_favorited: false,
        in_collection_list: false,
        references: [],
        media: [],
        transmog_slots: [],
        guide_steps: [],
        tags: [
          { id: 201, name: '库尔提拉斯风格', color: '356A8A' },
        ],
        created_at: '2026-07-10T00:00:00Z',
        updated_at: '2026-07-10T00:00:00Z',
      },
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/2')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('炉火旅店'))
    await wrapper.get('[data-testid="floating-toc-collapse"]').trigger('click')
    const copyButton = wrapper.find('[data-testid="copy-home-share-code"]')

    expect(copyButton.exists()).toBe(true)
    expect(wrapper.find('[data-testid="floating-like-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="floating-favorite-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="floating-list-button"]').exists()).toBe(true)
    expect(copyButton.text()).toContain('复制住宅分享代码')
    expect(wrapper.text()).not.toContain('白银之手')
    expect(wrapper.text()).not.toContain('艾尔文森林')
    expect(wrapper.text()).not.toContain('布置风格')
    expect(wrapper.text()).not.toContain('RPBOX-HOME-INN-001')
    expect(wrapper.text()).not.toContain('库尔提拉斯风格')
    await copyButton.trigger('click')
    expect(clipboardWriteText).toHaveBeenCalledWith('RPBOX-HOME-INN-001')
  })

  it('orders transmog slots from head to feet instead of backend order', async () => {
    getRPDBWork.mockResolvedValueOnce({
      work: {
        id: 3,
        author_id: 1,
        author_name: 'rpdb_transmog',
        type: 'transmog',
        title: '海潮卫士',
        slug: 'tide-guard',
        summary: '库尔提拉斯卫兵幻化。',
        content: '<p>幻化正文</p>',
        content_type: 'html',
        cover_image: '/uploads/rpdb/demo/transmog-01.jpg',
        rp_use_cases: '',
        effect_description: '',
        restrictions: '',
        extra: JSON.stringify({ share_code: 'TRANSMOG:TIDE-GUARD-001' }),
        game_version: '11.2.7',
        expansion: '至暗之夜',
        availability_status: 'available',
        bind_type: '',
        faction: 'alliance',
        armor_type: 'plate',
        verification_status: 'verified',
        verified_count: 0,
        outdated_count: 0,
        status: 'published',
        is_public: true,
        review_status: 'approved',
        version: 1,
        view_count: 10,
        like_count: 2,
        favorite_count: 1,
        comment_count: 0,
        list_count: 0,
        media_count: 1,
        is_liked: false,
        is_favorited: false,
        in_collection_list: false,
        references: [],
        media: [],
        transmog_slots: [
          { id: 1, slot: 'feet', role: 'required', name: '潮汐长靴', description: '厚重海军制式长靴', source: '围攻伯拉勒斯掉落', sort_order: 1 },
          { id: 2, slot: 'head', role: 'required', name: '海潮卫士头盔', description: '遮面式卫兵头盔', source: '军需官兑换', wowhead_url: 'https://www.wowhead.com/item=190001', variant: '海潮守备头盔', sort_order: 2 },
          { id: 3, slot: 'chest', role: 'required', name: '海潮胸甲', description: '蓝白配色板甲', source: '世界掉落', sort_order: 3 },
          { id: 4, slot: 'hands', role: 'unused', note: '不显示', sort_order: 4 },
          { id: 5, slot: 'shoulder', role: 'optional', name: '水手肩甲', description: '可替换肩部', source: '任务奖励', sort_order: 5 },
        ],
        guide_steps: [],
        tags: [],
        created_at: '2026-07-10T00:00:00Z',
        updated_at: '2026-07-10T00:00:00Z',
      },
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/rpdb/:id', component: RPDBDetail }],
    })
    await router.push('/rpdb/3')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('海潮卫士'))
    await wrapper.get('[data-testid="floating-toc-collapse"]').trigger('click')
    const copyButton = wrapper.get('[data-testid="copy-transmog-share-code"]')
    const inlineCopyButton = wrapper.get('[data-testid="copy-transmog-share-code-inline"]')

    expect(wrapper.findAll('[data-testid="transmog-slot-label"]').map(item => item.text())).toEqual([
      '头部',
      '肩部',
      '胸甲',
      '脚部',
    ])
    expect(wrapper.text()).toContain('名称海潮卫士头盔')
    expect(wrapper.text()).toContain('介绍遮面式卫兵头盔')
    expect(wrapper.text()).toContain('来源军需官兑换')
    expect(wrapper.text()).toContain('Wowheadhttps://www.wowhead.com/item=190001')
    expect(wrapper.text()).toContain('替代海潮守备头盔')
    expect(wrapper.find('a[href="https://www.wowhead.com/item=190001"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('不显示')
    expect(copyButton.text()).toContain('复制幻化分享代码')
    expect(inlineCopyButton.text()).toContain('复制代码')
    expect(wrapper.get('[data-testid="inline-transmog-share-code"]').text()).toContain('复制后可在游戏内导入这套幻化方案')
    expect(wrapper.text()).not.toContain('TRANSMOG:TIDE-GUARD-001')
    await inlineCopyButton.trigger('click')
    expect(clipboardWriteText).toHaveBeenCalledWith('TRANSMOG:TIDE-GUARD-001')
    expect(toastSuccess).toHaveBeenCalledWith('幻化分享代码已复制')
    expect(copyButton.text()).toContain('已复制幻化分享代码')
    expect(inlineCopyButton.text()).toContain('已复制')
  })

  it('lets a signed-in viewer report an RPDB work', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/rpdb/:id', component: RPDBDetail },
      ],
    })
    await router.push('/rpdb/1')
    const wrapper = mount(RPDBDetail, {
      global: { plugins: [createPinia(), router] },
      attachTo: document.body,
    })

    await vi.waitFor(() => expect(wrapper.text()).toContain('月光灯笼'))
    await wrapper.get('[data-testid="rpdb-report-button"]').trigger('click')
    const reportDialog = wrapper.getComponent({ name: 'SafetyReportDialog' })
    expect(reportDialog.props('visible')).toBe(true)

    reportDialog.vm.$emit('submit', {
      reason: 'fraud',
      detail: '来源与实际内容不符',
      hideTarget: false,
      blockAuthor: false,
      submitReport: true,
    })
    await vi.waitFor(() => expect(createContentReport).toHaveBeenCalledWith({
      target_type: 'rpdb_work',
      target_id: 1,
      reason: 'fraud',
      detail: '来源与实际内容不符',
      hide_target: false,
      block_author: false,
      submit_report: true,
    }))
    expect(toastSuccess).toHaveBeenCalledWith('举报已提交')
    wrapper.unmount()
  })
})
