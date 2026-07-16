import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RPDBMyUploads from './RPDBMyUploads.vue'

const api = vi.hoisted(() => ({
  listMyRPDBWorks: vi.fn(),
  listRPDBDrafts: vi.fn(),
  updateRPDBWorkVisibility: vi.fn(),
  deleteRPDBWork: vi.fn(),
  deleteRPDBDraft: vi.fn(),
  listGuilds: vi.fn(),
  confirm: vi.fn(),
}))

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return {
    ...actual,
    listMyRPDBWorks: api.listMyRPDBWorks,
    listRPDBDrafts: api.listRPDBDrafts,
    updateRPDBWorkVisibility: api.updateRPDBWorkVisibility,
    deleteRPDBWork: api.deleteRPDBWork,
    deleteRPDBDraft: api.deleteRPDBDraft,
    resolveRPDBMediaURL: (value?: string) => value || '',
  }
})

vi.mock('@/api/guild', () => ({ listGuilds: api.listGuilds }))
vi.mock('@/composables/useDialog', () => ({ dialog: { confirm: api.confirm } }))
vi.mock('@/stores/toast', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

const work = {
  id: 11,
  author_id: 1,
  type: 'item_showcase',
  title: '守夜人的提灯',
  slug: 'watch-lantern',
  summary: '用于巡夜与调查剧情。',
  content: '',
  content_type: 'html',
  cover_image: '/uploads/rpdb/lantern.jpg',
  status: 'published',
  review_status: 'approved',
  visibility: 'public',
  is_public: true,
  version: 1,
  view_count: 10,
  like_count: 3,
  favorite_count: 3,
  list_count: 4,
  comment_count: 0,
  media_count: 1,
  verified_count: 0,
  outdated_count: 0,
  is_liked: false,
  is_favorited: false,
  in_collection_list: false,
  created_at: '2026-07-10T00:00:00Z',
  updated_at: '2026-07-12T00:00:00Z',
}

describe('RPDBMyUploads', () => {
  beforeEach(() => {
    api.listMyRPDBWorks.mockReset()
    api.listRPDBDrafts.mockReset()
    api.updateRPDBWorkVisibility.mockReset()
    api.deleteRPDBWork.mockReset()
    api.deleteRPDBDraft.mockReset()
    api.listGuilds.mockReset()
    api.confirm.mockReset()
    api.listMyRPDBWorks.mockResolvedValue({ works: [{ ...work }] })
    api.listRPDBDrafts.mockResolvedValue({ drafts: [] })
    api.listGuilds.mockResolvedValue({ guilds: [{ id: 5, name: '暮色守望' }, { id: 6, name: '夜色议会' }] })
    api.updateRPDBWorkVisibility.mockImplementation(async (_id: number, visibility: string, guildIds: number[] = []) => ({
      work: { ...work, visibility, guild_id: guildIds[0], guild_ids: guildIds, is_public: visibility === 'public' },
    }))
    api.deleteRPDBWork.mockResolvedValue({ message: 'deleted' })
    api.deleteRPDBDraft.mockResolvedValue({ message: 'deleted' })
    api.confirm.mockResolvedValue(true)
  })

  async function mountPage() {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb/my-uploads', component: RPDBMyUploads },
        { path: '/rpdb/:id', component: { template: '<div />' } },
        { path: '/rpdb/:id/edit', component: { template: '<div />' } },
        { path: '/rpdb/drafts/:draftId/edit', component: { template: '<div />' } },
        { path: '/rpdb', component: { template: '<div />' } },
      ],
    })
    await router.push('/rpdb/my-uploads')
    const wrapper = mount(RPDBMyUploads, { global: { plugins: [createPinia(), router] } })
    await flushPromises()
    return { wrapper, router }
  }

  it('lists upload metrics and updates public, guild, and private visibility', async () => {
    const { wrapper } = await mountPage()

    expect(wrapper.text()).toContain('守夜人的提灯')
    expect(wrapper.get('.upload-metrics').text()).toBe('10334')

    const visibilitySelect = wrapper.get('.visibility-control select')
    await visibilitySelect.setValue('guild')
    await flushPromises()
    expect(api.updateRPDBWorkVisibility).toHaveBeenCalledWith(11, 'guild', [5])
    expect(wrapper.findAll('.visibility-guilds input[type="checkbox"]')).toHaveLength(2)

    await wrapper.findAll('.visibility-guilds input[type="checkbox"]')[1].setValue(true)
    await flushPromises()
    expect(api.updateRPDBWorkVisibility).toHaveBeenLastCalledWith(11, 'guild', [5, 6])

    await wrapper.find('.visibility-control select').setValue('private')
    await flushPromises()
    expect(api.updateRPDBWorkVisibility).toHaveBeenLastCalledWith(11, 'private', [5, 6])
  })

  it('lists separate drafts and opens them without treating them as formal works', async () => {
    api.listRPDBDrafts.mockResolvedValue({
      drafts: [{
        id: 21,
        author_id: 1,
        work_id: 11,
        type: 'item_showcase',
        title: '提灯修改草稿',
        payload: { title: '提灯修改草稿' },
        base_version: 1,
        status: 'active',
        created_at: '2026-07-13T00:00:00Z',
        updated_at: '2026-07-14T00:00:00Z',
      }],
    })
    const { wrapper, router } = await mountPage()

    const publishedList = wrapper.get('[data-testid="rpdb-my-uploads-list"]').element
    const draftList = wrapper.get('[data-testid="rpdb-draft-list"]').element
    expect(publishedList.compareDocumentPosition(draftList) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(wrapper.get('[data-testid="rpdb-draft-list"]').text()).toContain('提灯修改草稿')
    expect(wrapper.get('[data-testid="rpdb-draft-list"]').text()).toContain('发布前不会改动线上内容')
    await wrapper.get('[data-testid="rpdb-draft-row"] .row-actions button').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/rpdb/drafts/21/edit')
  })

  it('navigates to view and edit, then removes a confirmed upload', async () => {
    const { wrapper, router } = await mountPage()
    const actions = wrapper.findAll('.row-actions button')

    await actions[0].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/rpdb/11')
    await router.push('/rpdb/my-uploads')
    await actions[1].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/rpdb/11/edit')

    await actions[2].trigger('click')
    await flushPromises()
    expect(api.confirm).toHaveBeenCalled()
    expect(api.deleteRPDBWork).toHaveBeenCalledWith(11)
    expect(wrapper.findAll('[data-testid="rpdb-upload-row"]')).toHaveLength(0)
  })
})
