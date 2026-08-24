import { createApp, nextTick, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  applyGuild: vi.fn(),
  cancelApplication: vi.fn(),
  createContentReport: vi.fn(),
  deleteGuild: vi.fn(),
  getGuild: vi.fn(),
  leaveGuild: vi.fn(),
  listGuildMembers: vi.fn(),
  listMyApplications: vi.fn(),
  routerPush: vi.fn(),
  routerReplace: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  userStore: { user: { id: 99 } as { id: number } | null },
}))

function translate(key: string, params?: Record<string, unknown>) {
  return params?.name ? `${key}:${String(params.name)}` : key
}

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '42' } }),
  useRouter: () => ({
    push: mocks.routerPush,
    replace: mocks.routerReplace,
  }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: translate }),
}))

vi.mock('@shared/stores/toast', () => ({
  useToastStore: () => ({
    error: mocks.toastError,
    success: mocks.toastSuccess,
  }),
}))

vi.mock('@shared/stores/user', () => ({
  useUserStore: () => mocks.userStore,
}))

vi.mock('@/api/safety', () => ({
  createContentReport: mocks.createContentReport,
}))

vi.mock('@/api/guild', () => ({
  applyGuild: mocks.applyGuild,
  cancelApplication: mocks.cancelApplication,
  deleteGuild: mocks.deleteGuild,
  getGuild: mocks.getGuild,
  leaveGuild: mocks.leaveGuild,
  listGuildMembers: mocks.listGuildMembers,
  listMyApplications: mocks.listMyApplications,
}))

vi.mock('@/api/image', () => ({
  resolveApiUrl: (url: string) => url,
}))

import GuildDetail from './GuildDetail.vue'

let app: App<Element> | null = null

async function flushUi() {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

async function mountGuild(role: '' | 'owner' | 'admin' | 'member') {
  mocks.getGuild.mockResolvedValue({
    guild: {
      id: 42,
      name: 'Moon Guard',
      description: 'Role-playing guild',
      icon: '',
      color: 'B87333',
      slogan: '',
      lore: '',
      faction: 'alliance',
      owner_id: 7,
      member_count: 1,
      story_count: 0,
      is_public: true,
      invite_code: 'RPBOX',
      created_at: '2026-08-01T00:00:00Z',
      updated_at: '2026-08-01T00:00:00Z',
    },
    my_role: role,
  })
  mocks.listGuildMembers.mockResolvedValue({ members: [] })
  mocks.listMyApplications.mockResolvedValue({ applications: [] })

  const host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(GuildDetail)
  app.config.globalProperties.$t = translate as typeof app.config.globalProperties.$t
  app.mount(host)
  await flushUi()
  return host
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.userStore.user = { id: 99 }
  mocks.routerPush.mockResolvedValue(undefined)
  mocks.routerReplace.mockResolvedValue(undefined)
})

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
})

describe('mobile guild detail owner and safety actions', () => {
  it('only exposes the dangerous action to the guild owner', async () => {
    const host = await mountGuild('admin')

    expect(host.querySelector('[data-testid="guild-disband-open"]')).toBeNull()
  })

  it('allows cancellation, then submits one delete and replaces the stale detail route', async () => {
    let resolveDelete: (() => void) | undefined
    mocks.deleteGuild.mockImplementation(() => new Promise<void>((resolve) => {
      resolveDelete = resolve
    }))
    const host = await mountGuild('owner')
    const openButton = host.querySelector<HTMLButtonElement>('[data-testid="guild-disband-open"]')

    expect(openButton).not.toBeNull()
    openButton?.click()
    await nextTick()
    expect(host.querySelector('[role="dialog"]')).not.toBeNull()

    host.querySelector<HTMLButtonElement>('[data-testid="guild-disband-cancel"]')?.click()
    await nextTick()
    expect(host.querySelector('[role="dialog"]')).toBeNull()
    expect(mocks.deleteGuild).not.toHaveBeenCalled()

    openButton?.click()
    await nextTick()
    const confirmButton = host.querySelector<HTMLButtonElement>('[data-testid="guild-disband-confirm"]')
    confirmButton?.click()
    confirmButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(mocks.deleteGuild).toHaveBeenCalledTimes(1)
    expect(mocks.deleteGuild).toHaveBeenCalledWith(42)

    resolveDelete?.()
    await flushUi()

    expect(mocks.toastSuccess).toHaveBeenCalledWith('guild.disband.success')
    expect(mocks.routerReplace).toHaveBeenCalledWith({ name: 'guild' })
    expect(host.querySelector('[role="dialog"]')).toBeNull()
  })

  it('keeps the confirmation usable and surfaces the server error after failure', async () => {
    mocks.deleteGuild.mockRejectedValue(new Error('Owner permission required'))
    const host = await mountGuild('owner')

    host.querySelector<HTMLButtonElement>('[data-testid="guild-disband-open"]')?.click()
    await nextTick()
    host.querySelector<HTMLButtonElement>('[data-testid="guild-disband-confirm"]')?.click()
    await flushUi()

    expect(mocks.toastError).toHaveBeenCalledWith('Owner permission required')
    expect(mocks.routerReplace).not.toHaveBeenCalled()
    expect(host.querySelector('[role="dialog"]')).not.toBeNull()
    expect(host.querySelector<HTMLButtonElement>('[data-testid="guild-disband-confirm"]')?.disabled).toBe(false)
  })

  it('never exposes the guild safety action to the owner', async () => {
    mocks.userStore.user = { id: 7 }
    const host = await mountGuild('owner')

    expect(host.querySelector('[data-testid="guild-safety-open"]')).toBeNull()
    expect(mocks.createContentReport).not.toHaveBeenCalled()
  })

  it('submits a structured guild report while preserving hide and block independently', async () => {
    let resolveReport: ((value: { message: string }) => void) | undefined
    mocks.createContentReport.mockImplementation(() => new Promise<{ message: string }>((resolve) => {
      resolveReport = resolve
    }))
    const host = await mountGuild('member')

    host.querySelector<HTMLButtonElement>('[data-testid="guild-safety-open"]')?.click()
    await flushUi()

    const sheet = document.body.querySelector('.sheet-mask')
    const detailInput = sheet?.querySelector<HTMLTextAreaElement>('textarea')
    expect(sheet).not.toBeNull()
    expect(sheet?.textContent).toContain('common.safetyReport.hide.guild')
    expect(detailInput).not.toBeNull()
    if (detailInput) {
      detailInput.value = 'The guild profile promotes a scam.'
      detailInput.dispatchEvent(new Event('input', { bubbles: true }))
    }

    const hideGuild = sheet?.querySelector<HTMLInputElement>('.sheet-local-actions input')
    hideGuild?.click()
    await nextTick()

    const submitButton = sheet?.querySelector<HTMLButtonElement>('.sheet-btn.primary')
    submitButton?.click()
    submitButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(mocks.createContentReport).toHaveBeenCalledTimes(1)
    expect(mocks.createContentReport).toHaveBeenCalledWith({
      target_type: 'guild',
      target_id: 42,
      reason: 'spam',
      detail: 'The guild profile promotes a scam.',
      hide_target: true,
      block_author: false,
      submit_report: true,
    })

    resolveReport?.({ message: 'server-localized-message-is-not-used' })
    await flushUi()
    await new Promise((resolve) => window.setTimeout(resolve, 250))

    expect(mocks.toastSuccess).toHaveBeenCalledWith('guild.safety.hiddenSuccess')
    expect(mocks.routerReplace).toHaveBeenCalledWith({ name: 'guild' })
    expect(document.body.querySelector('.sheet-mask')).toBeNull()
  })

  it('closes a report-only submission without leaving the guild detail', async () => {
    mocks.createContentReport.mockResolvedValue({ message: 'submitted' })
    const host = await mountGuild('member')

    host.querySelector<HTMLButtonElement>('[data-testid="guild-safety-open"]')?.click()
    await flushUi()
    const detailInput = document.body.querySelector<HTMLTextAreaElement>('.sheet-mask textarea')
    if (detailInput) {
      detailInput.value = 'The guild profile contains unsafe content.'
      detailInput.dispatchEvent(new Event('input', { bubbles: true }))
    }
    await nextTick()
    document.body.querySelector<HTMLButtonElement>('.sheet-mask .sheet-btn.primary')?.click()
    await flushUi()
    await new Promise((resolve) => window.setTimeout(resolve, 250))

    expect(mocks.createContentReport).toHaveBeenCalledWith({
      target_type: 'guild',
      target_id: 42,
      reason: 'spam',
      detail: 'The guild profile contains unsafe content.',
      hide_target: false,
      block_author: false,
      submit_report: true,
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('guild.safety.reportSuccess')
    expect(mocks.routerReplace).not.toHaveBeenCalled()
    expect(document.body.querySelector('.sheet-mask')).toBeNull()
  })

  it('preserves a block-only action without hiding the guild', async () => {
    mocks.createContentReport.mockResolvedValue({ message: 'blocked' })
    const host = await mountGuild('admin')

    host.querySelector<HTMLButtonElement>('[data-testid="guild-safety-open"]')?.click()
    await flushUi()
    const sheet = document.body.querySelector('.sheet-mask')
    const detailInput = sheet?.querySelector<HTMLTextAreaElement>('textarea')
    if (detailInput) {
      detailInput.value = 'The owner is harassing members.'
      detailInput.dispatchEvent(new Event('input', { bubbles: true }))
    }
    const localActions = sheet?.querySelectorAll<HTMLInputElement>('.sheet-local-actions input')
    localActions?.item(1).click()
    await nextTick()
    sheet?.querySelector<HTMLButtonElement>('.sheet-btn.primary')?.click()
    await flushUi()

    expect(mocks.createContentReport).toHaveBeenCalledWith({
      target_type: 'guild',
      target_id: 42,
      reason: 'spam',
      detail: 'The owner is harassing members.',
      hide_target: false,
      block_author: true,
      submit_report: true,
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('guild.safety.blockedSuccess')
    expect(mocks.routerReplace).toHaveBeenCalledWith({ name: 'guild' })
  })

  it('keeps the safety sheet usable and shows the server error after failure', async () => {
    mocks.createContentReport.mockRejectedValue(new Error('Report service unavailable'))
    const host = await mountGuild('admin')

    host.querySelector<HTMLButtonElement>('[data-testid="guild-safety-open"]')?.click()
    await flushUi()
    const detailInput = document.body.querySelector<HTMLTextAreaElement>('.sheet-mask textarea')
    if (detailInput) {
      detailInput.value = 'Unsafe guild profile.'
      detailInput.dispatchEvent(new Event('input', { bubbles: true }))
    }
    await nextTick()
    document.body.querySelector<HTMLButtonElement>('.sheet-mask .sheet-btn.primary')?.click()
    await flushUi()

    expect(mocks.toastError).toHaveBeenCalledWith('Report service unavailable')
    expect(mocks.routerReplace).not.toHaveBeenCalled()
    expect(document.body.querySelector('.sheet-mask')).not.toBeNull()
    expect(document.body.querySelector<HTMLButtonElement>('.sheet-mask .sheet-btn.primary')?.disabled).toBe(false)
  })
})
