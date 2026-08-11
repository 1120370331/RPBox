import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import GuildManage from './GuildManage.vue'

const mocks = vi.hoisted(() => ({
  getGuild: vi.fn(),
  listGuildMembers: vi.fn(),
  listGuildApplications: vi.fn(),
  deleteGuild: vi.fn(),
  confirm: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
  toastWarning: vi.fn(),
}))

vi.mock('@/api/guild', async () => {
  const actual = await vi.importActual<typeof import('@/api/guild')>('@/api/guild')
  return {
    ...actual,
    getGuild: mocks.getGuild,
    listGuildMembers: mocks.listGuildMembers,
    listGuildApplications: mocks.listGuildApplications,
    deleteGuild: mocks.deleteGuild,
  }
})

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ confirm: mocks.confirm }),
}))

vi.mock('@/stores/toast', () => ({
  useToastStore: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
    warning: mocks.toastWarning,
  }),
}))

const guild = {
  id: 7,
  name: '暮色守望',
  description: '',
  icon: '',
  color: '',
  banner: '',
  slogan: '',
  lore: '',
  faction: 'neutral' as const,
  layout: 3 as const,
  owner_id: 1,
  member_count: 1,
  story_count: 0,
  is_public: true,
  invite_code: 'invite',
  visitor_can_view_stories: false,
  visitor_can_view_posts: false,
  member_can_view_stories: true,
  member_can_view_posts: true,
  auto_approve: false,
  created_at: '2026-08-11T00:00:00Z',
  updated_at: '2026-08-11T00:00:00Z',
}

async function mountPage(role: 'owner' | 'admin') {
  mocks.getGuild.mockResolvedValue({ guild: { ...guild }, my_role: role })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/guild/:id/manage', component: GuildManage },
      { path: '/guild/:id', component: { template: '<div />' } },
      { path: '/guild', component: { template: '<div />' } },
    ],
  })
  await router.push('/guild/7/manage')
  await router.isReady()

  const wrapper = mount(GuildManage, {
    global: {
      plugins: [router, i18n],
      stubs: { ImageCropperDialog: true },
    },
  })
  await flushPromises()
  return { wrapper, router }
}

describe('GuildManage guild disbanding', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    i18n.global.locale.value = 'zh-CN'
    mocks.listGuildMembers.mockResolvedValue({ members: [] })
    mocks.listGuildApplications.mockResolvedValue({ applications: [] })
    mocks.deleteGuild.mockResolvedValue(undefined)
    mocks.confirm.mockResolvedValue(true)
  })

  it('shows the owner-only danger zone and disbands after confirmation', async () => {
    const { wrapper, router } = await mountPage('owner')

    expect(wrapper.get('.danger-zone').text()).toContain('解散并注销公会')
    await wrapper.get('.danger-zone .r-button').trigger('click')
    await flushPromises()

    expect(mocks.confirm).toHaveBeenCalledWith(expect.objectContaining({
      type: 'error',
      confirmText: '确认解散',
      message: expect.stringContaining('暮色守望'),
    }))
    expect(mocks.deleteGuild).toHaveBeenCalledWith(7)
    expect(mocks.toastSuccess).toHaveBeenCalledWith('公会已解散')
    expect(router.currentRoute.value.fullPath).toBe('/guild')
    wrapper.unmount()
  })

  it('does not expose guild deletion to an administrator', async () => {
    const { wrapper } = await mountPage('admin')

    expect(wrapper.find('.danger-zone').exists()).toBe(false)
    wrapper.unmount()
  })
})
