import { createApp, nextTick, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  createBookmark: vi.fn(),
  createContentReport: vi.fn(),
  deleteBookmark: vi.fn(),
  deleteStoryEntry: vi.fn(),
  ensureEmoteMapLoaded: vi.fn(),
  getCharacter: vi.fn(),
  getCharacterCard: vi.fn(),
  getStory: vi.fn(),
  listBookmarks: vi.fn(),
  routerBack: vi.fn(),
  routerPush: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  toastWarning: vi.fn(),
  updateBookmark: vi.fn(),
  updateEntriesBackgroundColor: vi.fn(),
  updateLastViewBookmark: vi.fn(),
  updateStoryEntry: vi.fn(),
  userStore: {
    token: 'signed-in',
    user: { id: 7 },
    isAdmin: false,
    isModerator: false,
  },
}))

function translate(key: string, params?: Record<string, unknown>) {
  return params?.name ? `${key}:${String(params.name)}` : key
}

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '9' }, fullPath: '/stories/9' }),
  useRouter: () => ({
    back: mocks.routerBack,
    push: mocks.routerPush,
  }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: translate }),
}))

vi.mock('@shared/stores/toast', () => ({
  useToastStore: () => ({
    error: mocks.toastError,
    success: mocks.toastSuccess,
    warning: mocks.toastWarning,
  }),
}))

vi.mock('@shared/stores/user', () => ({
  useUserStore: () => mocks.userStore,
}))

vi.mock('@/api/story', () => ({
  createBookmark: mocks.createBookmark,
  deleteBookmark: mocks.deleteBookmark,
  deleteStoryEntry: mocks.deleteStoryEntry,
  getStory: mocks.getStory,
  listBookmarks: mocks.listBookmarks,
  updateBookmark: mocks.updateBookmark,
  updateEntriesBackgroundColor: mocks.updateEntriesBackgroundColor,
  updateLastViewBookmark: mocks.updateLastViewBookmark,
  updateStoryEntry: mocks.updateStoryEntry,
}))

vi.mock('@/api/character', () => ({
  getCharacter: mocks.getCharacter,
}))

// Story detail must consume the safe card map returned with the story instead
// of fetching a character card for every bound entry.
vi.mock('@/api/characterCard', () => ({
  getCharacterCard: mocks.getCharacterCard,
}))

vi.mock('@/api/safety', () => ({
  createContentReport: mocks.createContentReport,
}))

vi.mock('@/utils/emote', () => ({
  ensureEmoteMapLoaded: mocks.ensureEmoteMapLoaded,
  renderTextWithEmotes: (content: string) => content,
}))

import StoryDetail from './StoryDetail.vue'

let app: App<Element> | null = null

const storyResponse = {
  story: {
    id: 9,
    user_id: 7,
    title: 'Moonlit Archive',
    description: 'An archived scene.',
    region: '',
    address: '',
    participants: '',
    start_time: '2026-08-24T08:00:00Z',
    end_time: '2026-08-24T09:00:00Z',
    status: 'ended',
    is_public: true,
    view_count: 3,
    created_at: '2026-08-24T08:00:00Z',
    updated_at: '2026-08-24T09:00:00Z',
  },
  entries: [
    {
      id: 101,
      story_id: 9,
      source_id: 'bound-entry',
      type: 'dialogue' as const,
      character_card_id: 42,
      speaker: 'Archived Speaker',
      content: 'The archived line stays immutable.',
      channel: 'SAY',
      timestamp: '2026-08-24T08:10:00Z',
      sort_order: 1,
    },
    {
      id: 102,
      story_id: 9,
      source_id: 'legacy-entry',
      type: 'dialogue' as const,
      character_id: 777,
      speaker: 'Legacy Archived Speaker',
      content: 'The legacy line still renders.',
      channel: 'SAY',
      timestamp: '2026-08-24T08:11:00Z',
      sort_order: 2,
    },
  ],
  character_cards: {
    42: {
      id: 42,
      first_name: 'Mutable',
      last_name: 'Card',
      display_name: 'Mutable Card Name',
      icon: 'INV_Misc_QuestionMark',
      class_color: '6F8CA3',
      name_color: '6F8CA3',
      portrait_image_url: '/api/v1/images/character-card-portrait/42',
      portrait_image_updated_at: 'portrait-v7',
      updated_at: '2026-08-24T09:00:00Z',
    },
  },
}

async function flushUi() {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

async function mountStory() {
  const host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(StoryDetail)
  app.config.globalProperties.$t = translate as typeof app.config.globalProperties.$t
  app.mount(host)
  await flushUi()
  return host
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.getStory.mockResolvedValue(storyResponse)
  mocks.listBookmarks.mockResolvedValue({ bookmarks: [] })
  mocks.getCharacter.mockRejectedValue(new Error('Legacy character is unavailable'))
  mocks.ensureEmoteMapLoaded.mockResolvedValue(undefined)
  mocks.routerPush.mockResolvedValue(undefined)
  mocks.updateLastViewBookmark.mockResolvedValue(undefined)
})

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
})

describe('mobile story RPBox character-card bindings', () => {
  it('renders the immutable speaker and versioned card identity without per-card fetching', async () => {
    const host = await mountStory()
    const entries = host.querySelectorAll<HTMLElement>('.entry-item')
    const boundEntry = entries[0]
    const legacyEntry = entries[1]

    expect(entries).toHaveLength(2)
    expect(boundEntry.textContent).toContain('Archived Speaker')
    expect(boundEntry.textContent).not.toContain('Mutable Card Name')
    expect(legacyEntry.textContent).toContain('Legacy Archived Speaker')
    expect(legacyEntry.textContent).toContain('The legacy line still renders.')

    const portrait = boundEntry.querySelector<HTMLImageElement>('.character-card-link img')
    expect(portrait?.getAttribute('src')).toBe(
      '/api/v1/images/character-card-portrait/42?w=96&q=82&v=portrait-v7',
    )
    const linkedName = boundEntry.querySelector<HTMLElement>('.character-card-name-link strong')
    expect(linkedName?.style.color).toBe('rgb(111, 140, 163)')

    expect(mocks.getCharacter).toHaveBeenCalledTimes(1)
    expect(mocks.getCharacter).toHaveBeenCalledWith(777)
    expect(mocks.getCharacterCard).not.toHaveBeenCalled()

    boundEntry.querySelector<HTMLButtonElement>('.character-card-link')?.click()
    expect(mocks.routerPush).toHaveBeenCalledTimes(1)
    expect(mocks.routerPush).toHaveBeenLastCalledWith({
      name: 'character-card-detail',
      params: { id: 42 },
    })

    mocks.routerPush.mockClear()
    boundEntry.querySelector<HTMLButtonElement>('.character-card-name-link')?.click()
    expect(mocks.routerPush).toHaveBeenCalledTimes(1)
    expect(mocks.routerPush).toHaveBeenLastCalledWith({
      name: 'character-card-detail',
      params: { id: 42 },
    })
  })

  it('selects the bound entry in manage mode without navigating', async () => {
    const host = await mountStory()

    host.querySelector<HTMLButtonElement>('.manage-btn')?.click()
    await nextTick()

    const boundEntry = host.querySelector<HTMLElement>('.entry-item')
    expect(boundEntry?.querySelector('.character-card-link')).toBeNull()
    expect(boundEntry?.querySelector('.character-card-name-link')).toBeNull()

    boundEntry?.click()
    await nextTick()

    expect(boundEntry?.classList.contains('selected')).toBe(true)
    expect(mocks.routerPush).not.toHaveBeenCalled()
  })
})
