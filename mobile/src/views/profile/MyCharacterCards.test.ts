import { createApp, nextTick, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { CharacterCardSummary } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  deleteCharacterCard: vi.fn(),
  getUserInfo: vi.fn(),
  listMyCharacterCards: vi.fn(),
  routerBack: vi.fn(),
  routerPush: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  userStore: {
    user: { id: 99, username: 'Rog' } as Record<string, unknown> | null,
    mergeUser: vi.fn(),
    logout: vi.fn(),
  },
}))

function translate(key: string, params?: Record<string, unknown>) {
  if (params?.name) return `${key}:${String(params.name)}`
  if (params?.time) return `${key}:${String(params.time)}`
  return key
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    back: mocks.routerBack,
    push: mocks.routerPush,
    replace: vi.fn(),
  }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: { value: 'en-US' }, t: translate }),
}))

vi.mock('@shared/stores/toast', () => ({
  useToastStore: () => ({
    achievement: vi.fn(),
    error: mocks.toastError,
    info: vi.fn(),
    success: mocks.toastSuccess,
    warning: vi.fn(),
  }),
}))

vi.mock('@shared/stores/user', () => ({
  useUserStore: () => mocks.userStore,
}))

vi.mock('@/api/characterCard', () => ({
  deleteCharacterCard: mocks.deleteCharacterCard,
  listMyCharacterCards: mocks.listMyCharacterCards,
}))

vi.mock('@/api/image', () => ({
  resolveApiUrl: (url?: string | null) => url || '',
}))

vi.mock('@/components/CachedImage.vue', () => ({
  default: {
    props: {
      alt: String,
      authFetch: Boolean,
      src: String,
    },
    template: '<div class="mock-cached-image" :data-auth-fetch="String(authFetch)" :data-src="src" />',
  },
}))

vi.mock('@/api/user', () => ({
  deleteAccount: vi.fn(),
  getUserInfo: mocks.getUserInfo,
  signInDaily: vi.fn(),
}))

vi.mock('@/utils/forumLevel', () => ({
  buildForumLevelGuide: () => [],
  computeLevelProgressPercent: () => 0,
}))

vi.mock('@/utils/achievementProgress', () => ({
  buildAchievementEntries: () => [],
  buildAchievementProgressContext: () => ({}),
  buildAchievementWallEntries: () => [],
  pickFeaturedAchievement: () => undefined,
}))

vi.mock('@/components/AchievementMedal.vue', () => ({
  default: { template: '<span />' },
}))

vi.mock('@/components/UserLevelBadge.vue', () => ({
  default: { template: '<span />' },
}))

import MyCharacterCards from './MyCharacterCards.vue'
import Profile from '../Profile.vue'

let app: App<Element> | null = null

function card(id: number, overrides: Partial<CharacterCardSummary> = {}) {
  return {
    id,
    user_id: 99,
    first_name: `First${id}`,
    last_name: 'Last',
    display_name: `Character ${id}`,
    title: `Title ${id}`,
    full_title: '',
    race: 'Night Elf',
    class: 'Druid',
    eye_color: '',
    eye_color_hex: '',
    age: '',
    height: '',
    weight: '',
    birthplace: '',
    residence: '',
    relationship_status: '',
    icon: '',
    class_color: '',
    name_color: '',
    summary: '',
    portrait_image_url: '',
    portraits: [],
    status: 'draft',
    visibility: 'private',
    review_status: null,
    created_at: '2026-08-20T08:00:00Z',
    updated_at: '2026-08-24T08:00:00Z',
    ...overrides,
  } as CharacterCardSummary
}

const statusCards = [
  card(1, {
    status: 'published',
    visibility: 'public',
    review_status: 'approved',
    portrait_image_url: '/api/v1/images/character-card/1',
  }),
  card(2, { status: 'published', visibility: 'public', review_status: 'pending' }),
  card(3, { status: 'draft', visibility: 'private', review_status: null }),
  card(4, { status: 'published', visibility: 'private', review_status: 'approved' }),
  card(5, { status: 'published', visibility: 'public', review_status: 'rejected' }),
  card(6, { status: 'published', visibility: 'public', review_status: null }),
]

async function flushUi() {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

async function mountComponent(component: object) {
  const host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(component)
  app.config.globalProperties.$t = translate as never
  app.mount(host)
  await flushUi()
  return host
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.userStore.user = { id: 99, username: 'Rog' }
  mocks.listMyCharacterCards.mockResolvedValue({ character_cards: statusCards })
  mocks.deleteCharacterCard.mockResolvedValue(undefined)
  mocks.getUserInfo.mockResolvedValue({ id: 99, username: 'Rog', role: 'user' })
  mocks.routerPush.mockResolvedValue(undefined)
})

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
})

describe('mobile RPBox character-card management', () => {
  it('keeps cloud profiles separate and opens the RPBox character-card list from Profile', async () => {
    const host = await mountComponent(Profile)
    const cloudProfiles = [...host.querySelectorAll<HTMLButtonElement>('.menu-item')]
      .find(button => button.textContent?.includes('profile.menu.cloudProfiles'))
    const rpboxCards = host.querySelector<HTMLButtonElement>('[data-testid="profile-rpbox-character-cards"]')

    expect(cloudProfiles).not.toBeNull()
    expect(rpboxCards?.textContent).toContain('profile.menu.rpboxCharacterCards')
    rpboxCards?.click()

    expect(mocks.routerPush).toHaveBeenCalledWith({ name: 'my-character-cards' })
  })

  it('classifies every owner status and routes New, View, and Edit actions', async () => {
    const host = await mountComponent(MyCharacterCards)

    for (const state of ['published', 'pending', 'draft', 'private', 'rejected', 'unsubmitted']) {
      expect(host.querySelector(`[data-card-state="${state}"]`)).not.toBeNull()
      expect(host.querySelector(`[data-status-count="${state}"]`)?.textContent).toContain('1')
    }
    expect(host.querySelector('.mock-cached-image')?.getAttribute('data-auth-fetch')).toBe('true')
    expect(host.querySelector('.mock-cached-image')?.getAttribute('data-src'))
      .toBe('/api/v1/images/character-card/1')

    host.querySelector<HTMLButtonElement>('[data-testid="character-card-new"]')?.click()
    host.querySelector<HTMLButtonElement>('[data-testid="character-card-view"]')?.click()
    host.querySelector<HTMLButtonElement>('[data-testid="character-card-edit"]')?.click()

    expect(mocks.routerPush).toHaveBeenCalledWith({ name: 'character-card-new' })
    expect(mocks.routerPush).toHaveBeenCalledWith({ name: 'character-card-detail', params: { id: 1 } })
    expect(mocks.routerPush).toHaveBeenCalledWith({ name: 'character-card-edit', params: { id: 1 } })
  })

  it('requires an in-app confirmation before deleting and removes only the confirmed card', async () => {
    const host = await mountComponent(MyCharacterCards)
    host.querySelector<HTMLButtonElement>('[data-testid="character-card-delete"]')?.click()
    await nextTick()

    expect(mocks.deleteCharacterCard).not.toHaveBeenCalled()
    expect(host.querySelector('[data-testid="character-card-delete-dialog"]')).not.toBeNull()

    host.querySelector<HTMLButtonElement>('[data-testid="character-card-delete-confirm"]')?.click()
    await flushUi()

    expect(mocks.deleteCharacterCard).toHaveBeenCalledTimes(1)
    expect(mocks.deleteCharacterCard).toHaveBeenCalledWith(1)
    expect(host.querySelector('[data-testid="character-card-delete-dialog"]')).toBeNull()
    expect(host.querySelectorAll('.dossier-card')).toHaveLength(5)
    expect(mocks.toastSuccess).toHaveBeenCalledWith('characterCards.management.deleteSuccess')
  })
})
