import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RPDBMyUploads from './RPDBMyUploads.vue'

const api = vi.hoisted(() => ({
  listMyRPDBWorks: vi.fn(),
  updateRPDBWorkVisibility: vi.fn(),
  deleteRPDBWork: vi.fn(),
  listGuilds: vi.fn(),
  confirm: vi.fn(),
}))

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return {
    ...actual,
    listMyRPDBWorks: api.listMyRPDBWorks,
    updateRPDBWorkVisibility: api.updateRPDBWorkVisibility,
    deleteRPDBWork: api.deleteRPDBWork,
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
    api.updateRPDBWorkVisibility.mockReset()
    api.deleteRPDBWork.mockReset()
    api.listGuilds.mockReset()
    api.confirm.mockReset()
    api.listMyRPDBWorks.mockResolvedValue({ works: [{ ...work }] })
    api.listGuilds.mockResolvedValue({ guilds: [{ id: 5, name: '暮色守望' }] })
    api.updateRPDBWorkVisibility.mockImplementation(async (_id: number, visibility: string, guildId?: number) => ({
      work: { ...work, visibility, guild_id: guildId, is_public: visibility === 'public' },
    }))
    api.deleteRPDBWork.mockResolvedValue({ message: 'deleted' })
    api.confirm.mockResolvedValue(true)
  })

  async function mountPage() {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/rpdb/my-uploads', component: RPDBMyUploads },
        { path: '/rpdb/:id', component: { template: '<div />' } },
        { path: '/rpdb/:id/edit', component: { template: '<div />' } },
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
    expect(api.updateRPDBWorkVisibility).toHaveBeenCalledWith(11, 'guild', 5)
    expect(wrapper.findAll('.visibility-control select')).toHaveLength(2)

    await wrapper.findAll('.visibility-control select')[0].setValue('private')
    await flushPromises()
    expect(api.updateRPDBWorkVisibility).toHaveBeenLastCalledWith(11, 'private', 5)
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
