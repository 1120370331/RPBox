import { createApp, nextTick, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { CharacterCard } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  createContentReport: vi.fn(),
  getCharacterCard: vi.fn(),
  getCharacterCardShare: vi.fn(),
  routerBack: vi.fn(),
  routerPush: vi.fn(),
  routerReplace: vi.fn(),
  shareRouteLink: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  userStore: { user: { id: 99 } as { id: number } | null },
}))

function translate(key: string, params?: Record<string, unknown>) {
  if (key === 'characterCards.detail.safety.targetLabel') {
    return `${key}:${String(params?.id)}:${String(params?.name)}`
  }
  if (key === 'characterCards.detail.personalityValueShort') {
    return `${String(params?.value)} / 20`
  }
  if (key === 'characterCards.detail.personalityAria') {
    return `${String(params?.left)} to ${String(params?.right)}`
  }
  if (key === 'characterCards.detail.personalityValue') {
    return `${String(params?.value)} / 20 · ${String(params?.left)} to ${String(params?.right)}`
  }
  return key
}

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '42' } }),
  useRouter: () => ({
    back: mocks.routerBack,
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

vi.mock('@/api/characterCard', () => ({
  getCharacterCard: mocks.getCharacterCard,
  getCharacterCardShare: mocks.getCharacterCardShare,
}))

vi.mock('@/api/safety', () => ({
  createContentReport: mocks.createContentReport,
}))

vi.mock('@/api/image', () => ({
  resolveApiUrl: (url?: string | null) => url || '',
}))

vi.mock('@/utils/jumpLink', () => ({
  handleJumpLinkClick: () => false,
  sanitizeJumpLinks: vi.fn(),
}))

vi.mock('@/utils/mobileShare', () => ({
  shareRouteLink: mocks.shareRouteLink,
}))

import CharacterCardDetail from './CharacterCardDetail.vue'

const publicCard: CharacterCard = {
  id: 42,
  user_id: 7,
  first_name: 'Elia',
  last_name: 'Moonwhisper',
  display_name: 'Elia Moonwhisper',
  title: 'Moon Priestess',
  full_title: '',
  race: 'Night Elf',
  class: 'Priest',
  eye_color: 'Silver',
  eye_color_hex: '#c9d5e7',
  age: '',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  relationship_status: '',
  class_color: '',
  name_color: '',
  summary: 'A public role-playing character.',
  background_story: '',
  first_impression: '',
  impressions: [],
  other_content: '',
  portrait_image_url: '',
  portraits: [],
  status: 'published',
  visibility: 'public',
  review_status: 'approved',
  updated_at: '2026-08-24T08:00:00Z',
}

let app: App<Element> | null = null

async function flushUi() {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

function unmountCurrent() {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
}

async function mountCard(viewerId: number | null, overrides: Partial<CharacterCard> = {}) {
  unmountCurrent()
  mocks.userStore.user = viewerId ? { id: viewerId } : null
  mocks.getCharacterCard.mockResolvedValue({ ...publicCard, ...overrides })
  const host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(CharacterCardDetail)
  app.mount(host)
  await flushUi()
  return host
}

function setReportDetail(value: string) {
  const input = document.body.querySelector<HTMLTextAreaElement>('.sheet-mask textarea')
  expect(input).not.toBeNull()
  if (!input) return
  input.value = value
  input.dispatchEvent(new Event('input', { bubbles: true }))
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.userStore.user = { id: 99 }
  mocks.routerPush.mockResolvedValue(undefined)
  mocks.routerReplace.mockResolvedValue(undefined)
})

afterEach(() => {
  unmountCurrent()
})

describe('mobile public character-card safety actions', () => {
  it('shows the action only to a signed-in non-owner', async () => {
    let host = await mountCard(99)
    expect(host.querySelector('[data-testid="character-card-safety-open"]')).not.toBeNull()
    expect(host.querySelector('.additional-info-section')).toBeNull()
    expect(host.querySelector('.personality-section')).toBeNull()

    host = await mountCard(7)
    expect(host.querySelector('[data-testid="character-card-safety-open"]')).toBeNull()

    host = await mountCard(null)
    expect(host.querySelector('[data-testid="character-card-safety-open"]')).toBeNull()
    expect(mocks.createContentReport).not.toHaveBeenCalled()
  })

  it('renders safe additional fields and clamps personality markers to the continuum', async () => {
    const host = await mountCard(99, {
      additional_info: [
        { id: 7, name: 'Pronouns', value: 'she / her', icon: 'Interface\\Icons\\INV_Misc_Note_01.blp' },
        { id: 1, name: 'Unsafe icon', value: 'Still readable', icon: 'javascript:alert(1)' },
        { id: 3, name: '', value: '', icon: 'INV_Misc_QuestionMark' },
      ],
      personality_traits: [
        {
          preset_id: null,
          left_text: 'Reserved',
          right_text: 'Expressive',
          left_icon: 'Ability_Stealth',
          right_icon: 'javascript:alert(1)',
          left_color: { r: 0.2, g: 0.3, b: 0.4 },
          right_color: { r: 0.8, g: 0.7, b: 0.6 },
          value: 7,
        },
        {
          preset_id: null,
          left_text: 'Grounded',
          right_text: 'Unbound',
          left_icon: '',
          right_icon: '',
          left_color: { r: -3, g: 0.5, b: 4 },
          right_color: null,
          value: 99,
        },
        {
          preset_id: null,
          left_text: '',
          right_text: '',
          left_icon: 'INV_Misc_QuestionMark',
          right_icon: '',
          left_color: null,
          right_color: null,
          value: 10,
        },
      ],
    })

    const additionalRows = host.querySelectorAll('.additional-info-row')
    expect(additionalRows).toHaveLength(2)
    expect(host.textContent).toContain('Pronouns')
    expect(host.textContent).toContain('she / her')
    expect(host.textContent).toContain('Still readable')
    const additionalIcon = additionalRows[0].querySelector<HTMLImageElement>('img')
    expect(additionalIcon?.getAttribute('src')).toBe('/api/v1/icons/inv_misc_note_01')
    expect(additionalRows[1].querySelector('img')).toBeNull()
    expect(host.innerHTML).not.toContain('javascript:')

    const traits = host.querySelectorAll('.personality-trait')
    expect(traits).toHaveLength(2)
    expect(host.textContent).toContain('Reserved')
    expect(host.textContent).toContain('Expressive')
    expect(host.textContent).toContain('7 / 20')
    const clampedTrack = traits[1].querySelector<HTMLElement>('.personality-track')
    const clampedMarker = traits[1].querySelector<HTMLElement>('[data-testid="personality-trait-marker"]')
    expect(clampedTrack?.getAttribute('aria-valuenow')).toBe('20')
    expect(clampedMarker?.dataset.traitValue).toBe('20')
    expect(clampedMarker?.style.left).toBe('100%')
    expect(clampedMarker?.style.transform).toBe('translateX(-100%)')
  })

  it('submits one exact structured character-card report and blocks close while pending', async () => {
    let resolveReport!: (value: { message: string }) => void
    mocks.createContentReport.mockImplementation(() => new Promise<{ message: string }>((resolve) => {
      resolveReport = resolve
    }))
    const host = await mountCard(99)

    host.querySelector<HTMLButtonElement>('[data-testid="character-card-safety-open"]')?.click()
    await flushUi()

    const sheet = document.body.querySelector('.sheet-mask')
    expect(sheet?.querySelector('.sheet-header h3')?.textContent).toBe('characterCards.detail.safety.sheetTitle')
    expect(sheet?.querySelector('.sheet-header p')?.textContent)
      .toBe('characterCards.detail.safety.targetLabel:42:Elia Moonwhisper')
    expect(sheet?.querySelector('.sheet-local-actions span')?.textContent)
      .toBe('common.safetyReport.hide.character_card')
    setReportDetail('The character card contains abusive UGC.')
    await nextTick()

    const submit = sheet?.querySelector<HTMLButtonElement>('.sheet-btn.primary')
    submit?.click()
    submit?.dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(mocks.createContentReport).toHaveBeenCalledTimes(1)
    expect(mocks.createContentReport).toHaveBeenCalledWith({
      target_type: 'character_card',
      target_id: 42,
      reason: 'spam',
      detail: 'The character card contains abusive UGC.',
      hide_target: false,
      block_author: false,
      submit_report: true,
    })

    document.body.querySelector<HTMLButtonElement>('.sheet-close')?.click()
    await nextTick()
    expect(document.body.querySelector('.sheet-mask')).not.toBeNull()

    resolveReport({ message: 'submitted' })
    await flushUi()
    await new Promise(resolve => window.setTimeout(resolve, 250))

    expect(mocks.toastSuccess).toHaveBeenCalledWith('characterCards.detail.safety.reportSuccess')
    expect(mocks.routerReplace).not.toHaveBeenCalled()
    expect(document.body.querySelector('.sheet-mask')).toBeNull()
  })

  it('preserves hide independently and removes hidden content before leaving', async () => {
    mocks.createContentReport.mockResolvedValue({ message: 'hidden' })
    const host = await mountCard(99)
    host.querySelector<HTMLButtonElement>('[data-testid="character-card-safety-open"]')?.click()
    await flushUi()
    setReportDetail('Hide this character card and submit a report.')

    document.body.querySelector<HTMLInputElement>('.sheet-local-actions input')?.click()
    await nextTick()
    document.body.querySelector<HTMLButtonElement>('.sheet-btn.primary')?.click()
    await flushUi()

    expect(mocks.createContentReport).toHaveBeenCalledWith({
      target_type: 'character_card',
      target_id: 42,
      reason: 'spam',
      detail: 'Hide this character card and submit a report.',
      hide_target: true,
      block_author: false,
      submit_report: true,
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('characterCards.detail.safety.hiddenSuccess')
    expect(mocks.routerReplace).toHaveBeenCalledWith({ name: 'community' })
    expect(host.querySelector('.portrait-dossier')).toBeNull()
  })

  it('preserves block independently and removes the blocked owner content before leaving', async () => {
    mocks.createContentReport.mockResolvedValue({ message: 'blocked' })
    const host = await mountCard(99)
    host.querySelector<HTMLButtonElement>('[data-testid="character-card-safety-open"]')?.click()
    await flushUi()
    setReportDetail('Block this character-card owner.')

    const localActions = document.body.querySelectorAll<HTMLInputElement>('.sheet-local-actions input')
    localActions[1]?.click()
    await nextTick()
    document.body.querySelector<HTMLButtonElement>('.sheet-btn.primary')?.click()
    await flushUi()

    expect(mocks.createContentReport).toHaveBeenCalledWith({
      target_type: 'character_card',
      target_id: 42,
      reason: 'spam',
      detail: 'Block this character-card owner.',
      hide_target: false,
      block_author: true,
      submit_report: true,
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('characterCards.detail.safety.blockedSuccess')
    expect(mocks.routerReplace).toHaveBeenCalledWith({ name: 'community' })
    expect(host.querySelector('.portrait-dossier')).toBeNull()
  })

  it('keeps the sheet usable and surfaces the server error after failure', async () => {
    mocks.createContentReport.mockRejectedValue(new Error('Report service unavailable'))
    const host = await mountCard(99)
    host.querySelector<HTMLButtonElement>('[data-testid="character-card-safety-open"]')?.click()
    await flushUi()
    setReportDetail('Unsafe character-card content.')
    await nextTick()
    document.body.querySelector<HTMLButtonElement>('.sheet-btn.primary')?.click()
    await flushUi()

    expect(mocks.toastError).toHaveBeenCalledWith('Report service unavailable')
    expect(mocks.routerReplace).not.toHaveBeenCalled()
    expect(document.body.querySelector('.sheet-mask')).not.toBeNull()
    expect(document.body.querySelector<HTMLButtonElement>('.sheet-btn.primary')?.disabled).toBe(false)
  })
})
