import { createApp, nextTick, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { CharacterCard, UpdateCharacterCardRequest } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  addPortrait: vi.fn(),
  deletePortrait: vi.fn(),
  getCharacterCard: vi.fn(),
  publishCharacterCard: vi.fn(),
  setCover: vi.fn(),
  updateCharacterCard: vi.fn(),
  uploadPortrait: vi.fn(),
  routerPush: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  toastWarning: vi.fn(),
  userStore: { user: { id: 7 } as { id: number } | null },
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '42' } }),
  useRouter: () => ({ push: mocks.routerPush }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
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

vi.mock('@/api/characterCard', () => ({
  addCharacterCardPortrait: mocks.addPortrait,
  deleteCharacterCardPortrait: mocks.deletePortrait,
  getCharacterCard: mocks.getCharacterCard,
  publishCharacterCard: mocks.publishCharacterCard,
  setCharacterCardPortraitCover: mocks.setCover,
  updateCharacterCard: mocks.updateCharacterCard,
  uploadCharacterCardPortrait: mocks.uploadPortrait,
}))

vi.mock('@/api/image', () => ({
  resolveApiUrl: (value?: string | null) => value || '',
}))

vi.mock('@/utils/nativeImagePicker', () => ({
  canUseNativeImagePicker: () => false,
  pickSingleNativeImageFile: vi.fn(),
}))

vi.mock('@/components/CachedImage.vue', () => ({
  default: {
    props: ['src', 'alt'],
    template: '<img :src="src" :alt="alt">',
  },
}))

vi.mock('@/components/MobileRichEditor.vue', () => ({
  default: {
    props: ['modelValue', 'placeholder'],
    emits: ['update:modelValue'],
    template: '<textarea :value="modelValue" :placeholder="placeholder" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}))

vi.mock('@/components/NativeImageSourceDialog.vue', () => ({
  default: { template: '<div />' },
}))

import CharacterCardEditor from './CharacterCardEditor.vue'

const baseCard: CharacterCard = {
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
  icon: '',
  class_color: '',
  name_color: '',
  additional_info: [],
  personality_traits: [],
  summary: 'A moonlit traveler.',
  background_story: '<p>Background</p>',
  first_impression: '',
  impressions: [],
  other_content: '',
  portrait_image_url: '',
  portraits: [],
  status: 'published',
  visibility: 'private',
  review_status: 'none',
  sort_order: 4,
  updated_at: '2026-08-25T00:00:00Z',
}

let app: App<Element> | null = null

async function flushUi() {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

async function mountEditor() {
  const host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(CharacterCardEditor)
  app.mount(host)
  await flushUi()
  return host
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.userStore.user = { id: 7 }
  mocks.getCharacterCard.mockResolvedValue({ ...baseCard })
  mocks.routerPush.mockResolvedValue(undefined)
  mocks.updateCharacterCard.mockImplementation((id: number, payload: UpdateCharacterCardRequest) => (
    Promise.resolve({ ...baseCard, ...payload, id, user_id: 7 })
  ))
  mocks.publishCharacterCard.mockResolvedValue({
    ...baseCard,
    status: 'published',
    visibility: 'public',
    review_status: 'pending',
  })
})

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
})

describe('mobile character-card owner editor', () => {
  it('shows the real owner state for a draft instead of calling it unsubmitted', async () => {
    mocks.getCharacterCard.mockResolvedValue({
      ...baseCard,
      status: 'draft',
      visibility: 'private',
      review_status: 'none',
    })
    const host = await mountEditor()

    expect(host.querySelector('.status-pill')?.textContent)
      .toContain('characterCards.common.status.draft')
  })

  it('fails closed when the signed-in user does not own the card', async () => {
    mocks.userStore.user = { id: 99 }
    const host = await mountEditor()

    expect(host.textContent).toContain('characterCards.editor.noPermission')
    expect(host.querySelector('[data-testid="save-character-card"]')).toBeNull()
    expect(mocks.updateCharacterCard).not.toHaveBeenCalled()
  })

  it('saves the complete owner draft without changing its status or visibility', async () => {
    const host = await mountEditor()
    expect(host.querySelector('.status-pill')?.textContent)
      .toContain('characterCards.common.status.private')
    const displayName = host.querySelector<HTMLInputElement>('[data-testid="display-name-input"]')
    expect(displayName).not.toBeNull()
    if (displayName) {
      displayName.value = 'Elia of the Moon'
      displayName.dispatchEvent(new Event('input', { bubbles: true }))
    }
    const classColor = host.querySelector<HTMLInputElement>('[data-testid="class-color-input"]')
    if (classColor) {
      classColor.value = '#123456'
      classColor.dispatchEvent(new Event('input', { bubbles: true }))
    }
    await nextTick()
    host.querySelector<HTMLButtonElement>('[data-testid="save-character-card"]')?.click()
    await flushUi()

    expect(mocks.updateCharacterCard).toHaveBeenCalledTimes(1)
    const [id, payload] = mocks.updateCharacterCard.mock.calls[0] as [number, UpdateCharacterCardRequest]
    expect(id).toBe(42)
    expect(payload).toMatchObject({
      display_name: 'Elia of the Moon',
      status: 'published',
      visibility: 'private',
      background_story: '<p>Background</p>',
      class_color: '#123456',
      name_color: '#123456',
      sort_order: 4,
    })
    expect(payload.impressions).toHaveLength(5)
    expect(payload.impressions[0]).not.toHaveProperty('image_updated_at')
    expect(payload).not.toHaveProperty('portrait_image_url')
  })

  it('always saves public state before calling the publish endpoint', async () => {
    const order: string[] = []
    mocks.updateCharacterCard.mockImplementation((id: number, payload: UpdateCharacterCardRequest) => {
      order.push('PUT')
      return Promise.resolve({ ...baseCard, ...payload, id, user_id: 7 })
    })
    mocks.publishCharacterCard.mockImplementation(() => {
      order.push('POST')
      return Promise.resolve({
        ...baseCard,
        status: 'published',
        visibility: 'public',
        review_status: 'pending',
      })
    })
    const host = await mountEditor()

    host.querySelector<HTMLButtonElement>('[data-testid="publish-character-card"]')?.click()
    await flushUi()

    expect(order).toEqual(['PUT', 'POST'])
    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(42, expect.objectContaining({
      status: 'published',
      visibility: 'public',
    }))
    expect(mocks.publishCharacterCard).toHaveBeenCalledWith(42)
    expect(host.textContent).toContain('characterCards.common.status.pending')
  })

  it('persists a pending portrait reference without marking the read URL as unsaved', async () => {
    const existingPortrait = {
      id: 1,
      image_url: '/api/v1/images/character-card-portrait-item/1',
      sort_order: 0,
      is_cover: true,
    }
    const addedPortrait = {
      id: 2,
      image_url: '/api/v1/images/character-card-portrait-item/2',
      sort_order: 1,
      is_cover: false,
    }
    mocks.getCharacterCard.mockResolvedValue({
      ...baseCard,
      portrait_image_url: '/api/v1/images/character-card-portrait/42?v=old',
      portraits: [existingPortrait],
    })
    mocks.uploadPortrait.mockResolvedValue('pending/portrait.webp')
    mocks.addPortrait.mockResolvedValue({
      ...baseCard,
      portrait_image_url: '/api/v1/images/character-card-portrait/42?v=old',
      portraits: [existingPortrait, addedPortrait],
    })
    mocks.setCover.mockResolvedValue({
      ...baseCard,
      portrait_image_url: '/api/v1/images/character-card-portrait/42?v=new',
      portraits: [
        { ...existingPortrait, is_cover: false },
        { ...addedPortrait, is_cover: true },
      ],
    })
    const host = await mountEditor()
    const file = new File(['portrait'], 'portrait.webp', { type: 'image/webp' })
    const input = host.querySelector<HTMLInputElement>('[data-testid="portrait-file-input"]')
    expect(input).not.toBeNull()
    if (input) {
      Object.defineProperty(input, 'files', { configurable: true, value: [file] })
      input.dispatchEvent(new Event('change', { bubbles: true }))
    }
    await flushUi()

    expect(mocks.uploadPortrait).toHaveBeenCalledWith(file)
    expect(mocks.addPortrait).toHaveBeenCalledWith(42, 'pending/portrait.webp')
    expect(mocks.setCover).toHaveBeenCalledWith(42, 2)
    expect(host.textContent).toContain('characterCards.editor.allSaved')
    expect(host.textContent).not.toContain('characterCards.editor.unsavedChanges')

    mocks.deletePortrait.mockResolvedValue({
      ...baseCard,
      portrait_image_url: '/api/v1/images/character-card-portrait/42?v=old',
      portraits: [existingPortrait],
    })
    host.querySelectorAll<HTMLButtonElement>('.portrait-delete')[1]?.click()
    await nextTick()
    host.querySelector<HTMLButtonElement>('[data-testid="confirm-portrait-delete"]')?.click()
    await flushUi()
    expect(mocks.deletePortrait).toHaveBeenCalledWith(42, 2)
    expect(host.textContent).toContain('characterCards.editor.allSaved')
  })
})
