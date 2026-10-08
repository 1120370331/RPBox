// Focused worker verification with simulated responses, not database evidence.
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { createGuild, updateGuild } from '@/api/guild'
import GuildCreate from '../GuildCreate.vue'
import GuildDetail from '../GuildDetail.vue'
import GuildLayout1 from '../layouts/GuildLayout1.vue'
import GuildLayout2 from '../layouts/GuildLayout2.vue'
import GuildLayout3 from '../layouts/GuildLayout3.vue'
import GuildLayout4 from '../layouts/GuildLayout4.vue'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), alert: vi.fn() }))
vi.mock('@/api/request', () => ({ default: { get: mocks.get, post: mocks.post, put: mocks.put } }))
vi.mock('@/composables/useDialog', () => ({ useDialog: () => ({ alert: mocks.alert, confirm: vi.fn() }) }))

function fixture(server?: string) {
  return { id: 7, name: '暮色守望', description: '', banner: '', slogan: '', lore: '', color: 'B87333',
    faction: 'neutral' as const, layout: 1 as const, owner_id: 1, member_count: 0, story_count: 0,
    is_public: true, invite_code: 'invite', icon: '', visitor_can_view_stories: false,
    visitor_can_view_posts: false, member_can_view_stories: true, member_can_view_posts: true,
    auto_approve: false, created_at: '', updated_at: '', ...(server === undefined ? {} : { server }) }
}

let responseGuild = fixture()
let role = 'owner'
async function page(component: typeof GuildCreate | typeof GuildDetail, path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/guild/create', component: GuildCreate }, { path: '/guild/:id', component: GuildDetail },
  ] })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(component, { global: {
    plugins: [router, i18n, createPinia()], stubs: {
      ImageCropperDialog: true, TiptapEditor: true,
      LazyBgImage: { template: '<div><slot /></div>' },
      RModal: { props: ['modelValue'], template: '<section v-if="modelValue" class="test-modal"><slot /><slot name="footer" /></section>' },
    },
  } })
  await flushPromises()
  return { wrapper, router }
}

async function settings() {
  const mounted = await page(GuildDetail, '/guild/7')
  await mounted.wrapper.findAll('button').find(button => button.text() === '设置')!.trigger('click')
  return mounted
}
const serverInput = 'input[placeholder="如：暗影之月"]'

beforeEach(() => {
  vi.clearAllMocks()
  responseGuild = fixture()
  role = 'owner'
  i18n.global.locale.value = 'zh-CN'
  mocks.alert.mockResolvedValue(true)
  mocks.get.mockImplementation(async (path: string) => {
    if (path === '/guilds/7') return { guild: { ...responseGuild }, my_role: role }
    if (path.includes('/members')) return { members: [] }
    if (path.includes('/applications')) return { applications: [] }
    throw new Error(`Unexpected GET ${path}`)
  })
  mocks.post.mockResolvedValue(fixture())
  mocks.put.mockImplementation(async (_path, data) => {
    responseGuild = { ...responseGuild, ...data }
    return responseGuild
  })
})

describe('server request contract (simulated transport)', () => {
  it('trims create and update without mutating callers', async () => {
    const data = { name: '公会', server: '  暗影之月  ' }
    await createGuild(data)
    await updateGuild(7, data)
    expect(mocks.post).toHaveBeenCalledWith('/guilds', { name: '公会', server: '暗影之月' })
    expect(mocks.put).toHaveBeenCalledWith('/guilds/7', { name: '公会', server: '暗影之月' })
    expect(data.server).toBe('  暗影之月  ')
  })
  it('preserves omission and explicitly sends empty string', async () => {
    await updateGuild(7, { name: '公会' })
    expect(mocks.put.mock.calls[0][1]).not.toHaveProperty('server')
    await updateGuild(7, { server: '  ' })
    expect(mocks.put.mock.calls[1][1]).toEqual({ server: '' })
  })
})

describe('create server form', () => {
  it('submits normalized server and navigates', async () => {
    const { wrapper, router } = await page(GuildCreate, '/guild/create')
    await wrapper.get('input[placeholder="输入公会名称"]').setValue('暮色守望')
    await wrapper.get(serverInput).setValue('  暗影之月  ')
    await wrapper.findAll('button').find(button => button.text() === '创建')!.trigger('click')
    await flushPromises()
    expect(mocks.post.mock.calls[0][1].server).toBe('暗影之月')
    expect(router.currentRoute.value.path).toBe('/guild/7')
    wrapper.unmount()
  })
  it('rejects over 128 code points, accepts 128 emoji, and shows request errors', async () => {
    const { wrapper } = await page(GuildCreate, '/guild/create')
    await wrapper.get('input[placeholder="输入公会名称"]').setValue('暮色守望')
    await wrapper.get(serverInput).setValue('😀'.repeat(129))
    const submit = wrapper.findAll('button').find(button => button.text() === '创建')!
    await submit.trigger('click')
    expect(mocks.post).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('最多允许 128 个字符')
    await wrapper.get(serverInput).setValue('😀'.repeat(128))
    mocks.post.mockRejectedValueOnce(new Error('服务器保存失败'))
    await submit.trigger('click')
    await flushPromises()
    expect(mocks.post).toHaveBeenCalledOnce()
    expect(mocks.alert).toHaveBeenCalledWith(expect.objectContaining({ message: '服务器保存失败', type: 'error' }))
    expect(wrapper.get(serverInput).element).toHaveProperty('value', '😀'.repeat(128))
    wrapper.unmount()
  })
})

describe('settings and displayed server', () => {
  it('loads old responses as empty', async () => {
    const { wrapper } = await settings()
    expect(wrapper.get(serverInput).element).toHaveProperty('value', '')
    expect(wrapper.get('.guild-server').text()).toBe('')
    wrapper.unmount()
  })
  it('loads, modifies, rereads and clears through actual form events', async () => {
    responseGuild = fixture('暗影之月')
    const { wrapper } = await settings()
    expect(wrapper.get(serverInput).element).toHaveProperty('value', '暗影之月')
    await wrapper.get(serverInput).setValue('  月亮守卫  ')
    await wrapper.findAll('button').find(button => button.text() === '保存')!.trigger('click')
    await flushPromises()
    expect(mocks.put.mock.calls[0][1].server).toBe('月亮守卫')
    expect(mocks.get.mock.calls.filter(call => call[0] === '/guilds/7')).toHaveLength(2)
    expect(wrapper.get('.guild-server').text()).toBe('月亮守卫')
    await wrapper.findAll('button').find(button => button.text() === '设置')!.trigger('click')
    await wrapper.get(serverInput).setValue('   ')
    await wrapper.findAll('button').find(button => button.text() === '保存')!.trigger('click')
    await flushPromises()
    expect(mocks.put.mock.calls[1][1].server).toBe('')
    expect(wrapper.get('.guild-server').text()).toBe('')
    wrapper.unmount()
  })
  it('blocks invalid length and retains edited data on failed save', async () => {
    const { wrapper } = await settings()
    await wrapper.get(serverInput).setValue('服'.repeat(129))
    const save = wrapper.findAll('button').find(button => button.text() === '保存')!
    await save.trigger('click')
    expect(mocks.put).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('最多允许 128 个字符')
    await wrapper.get(serverInput).setValue('暗影之月')
    mocks.put.mockRejectedValueOnce(new Error('保存服务器失败'))
    await save.trigger('click')
    await flushPromises()
    expect(mocks.alert).toHaveBeenCalledWith(expect.objectContaining({ message: '保存服务器失败', type: 'error' }))
    expect(wrapper.get(serverInput).element).toHaveProperty('value', '暗影之月')
    wrapper.unmount()
  })
  it.each(['owner', 'admin', 'member', ''])('retains settings permission for role %s', async (value) => {
    role = value
    const { wrapper } = await page(GuildDetail, '/guild/7')
    expect(wrapper.findAll('button').some(button => button.text() === '设置')).toBe(['owner', 'admin'].includes(value))
    wrapper.unmount()
  })
})

describe('unrouted layouts verified separately', () => {
  it.each([GuildLayout1, GuildLayout2, GuildLayout3, GuildLayout4])('shows server and preserves absence', async component => {
    const wrapper = mount(component, { props: { guild: fixture('暗影之月'), members: [], myRole: 'owner' }, global: { plugins: [createPinia()] } })
    expect(wrapper.text()).toContain('暗影之月')
    await wrapper.setProps({ guild: fixture() })
    expect(wrapper.text()).not.toContain('暗影之月')
    expect(wrapper.text()).not.toContain('UNKNOWN')
    wrapper.unmount()
  })
})
