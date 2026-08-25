import { createApp, nextTick, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  createCharacterCard: vi.fn(),
  getCharacterCardSources: vi.fn(),
  replace: vi.fn(),
  push: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  toastWarning: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: mocks.replace, push: mocks.push }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: 'zh-CN' },
  }),
}))

vi.mock('@shared/stores/toast', () => ({
  useToastStore: () => ({
    error: mocks.toastError,
    success: mocks.toastSuccess,
    warning: mocks.toastWarning,
  }),
}))

vi.mock('@/api/characterCard', () => ({
  createCharacterCard: mocks.createCharacterCard,
  getCharacterCardSources: mocks.getCharacterCardSources,
}))

import CharacterCardNew from './CharacterCardNew.vue'

let app: App<Element> | null = null

async function flushUi() {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

async function mountView() {
  const host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(CharacterCardNew)
  app.mount(host)
  await flushUi()
  return host
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.replace.mockResolvedValue(undefined)
  mocks.push.mockResolvedValue(undefined)
  mocks.getCharacterCardSources.mockResolvedValue({
    sources: [{
      backup_id: 8,
      account_id: 'MAIN',
      profile_id: 'elia',
      display_name: 'Elia Moonwhisper',
      race: 'Night Elf',
      class: 'Priest',
      backup_updated_at: '2026-08-25T00:00:00Z',
    }],
  })
})

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
})

describe('mobile character-card source creation', () => {
  it('creates from the selected cloud backup and replaces into the editor', async () => {
    mocks.createCharacterCard.mockResolvedValue({ id: 61 })
    const host = await mountView()

    host.querySelector<HTMLButtonElement>('[data-testid="character-card-source"]')?.click()
    await nextTick()
    host.querySelector<HTMLButtonElement>('[data-testid="create-from-backup"]')?.click()
    await flushUi()

    expect(mocks.createCharacterCard).toHaveBeenCalledWith({
      source_type: 'backup',
      source_backup_id: 8,
      source_profile_id: 'elia',
    })
    expect(mocks.replace).toHaveBeenCalledWith({
      name: 'character-card-edit',
      params: { id: 61 },
    })
  })

  it('creates an independent private draft without a TRP3 source', async () => {
    mocks.createCharacterCard.mockResolvedValue({ id: 62 })
    const host = await mountView()

    host.querySelector<HTMLButtonElement>('[data-testid="create-blank-card"]')?.click()
    await flushUi()

    expect(mocks.createCharacterCard).toHaveBeenCalledWith({ source_type: 'blank' })
    expect(mocks.replace).toHaveBeenCalledWith({
      name: 'character-card-edit',
      params: { id: 62 },
    })
  })
})
