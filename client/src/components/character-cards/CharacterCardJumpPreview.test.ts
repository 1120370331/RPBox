import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import CharacterCardJumpPreview from './CharacterCardJumpPreview.vue'
import type { CharacterCard } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  getCharacterCard: vi.fn(),
}))

vi.mock('@/api/characterCard', () => ({
  getCharacterCard: mocks.getCharacterCard,
}))

const card: CharacterCard = {
  id: 31,
  user_id: 4,
  first_name: '莱恩',
  last_name: '晨歌',
  display_name: '莱恩·晨歌',
  title: '灰港巡林者',
  full_title: '',
  race: '人类',
  class: '猎人',
  eye_color: '',
  eye_color_hex: '',
  age: '',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  relationship_status: '',
  icon: '',
  name_color: '',
  summary: '常年巡守灰港以北的林地。',
  background_story: '',
  first_impression: '',
  impressions: [],
  other_content: '',
  portrait_image_url: '',
  status: 'published',
  visibility: 'public',
  created_at: '2026-08-10T08:00:00Z',
  updated_at: '2026-08-10T08:00:00Z',
}

async function mountPreview() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }],
  })
  await router.push('/')
  await router.isReady()
  return mount(CharacterCardJumpPreview, {
    attachTo: document.body,
    global: {
      plugins: [router, i18n],
      stubs: { CharacterCardPortrait: true },
    },
  })
}

function appendJumpCard(id: number) {
  const link = document.createElement('a')
  link.href = `/character-cards/${id}`
  link.setAttribute('data-jump-type', 'character_card')
  link.setAttribute('data-jump-id', String(id))
  link.textContent = '查看人物卡'
  document.body.appendChild(link)
  return link
}

describe('CharacterCardJumpPreview', () => {
  let wrapper: VueWrapper | null = null

  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
    vi.useFakeTimers()
    mocks.getCharacterCard.mockReset()
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    vi.useRealTimers()
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('loads and presents the latest public card when its link receives keyboard focus', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    wrapper = await mountPreview()
    const link = appendJumpCard(card.id)

    link.dispatchEvent(new FocusEvent('focusin', { bubbles: true }))
    await vi.advanceTimersByTimeAsync(150)
    await flushPromises()

    expect(mocks.getCharacterCard).toHaveBeenCalledWith(card.id)
    const preview = document.body.querySelector<HTMLElement>('[data-testid="character-card-jump-preview"]')
    expect(preview).not.toBeNull()
    expect(preview?.textContent).toContain('莱恩·晨歌')
    expect(preview?.textContent).toContain('灰港巡林者')
    expect(preview?.textContent).toContain('人类 · 猎人')
  })

  it('redacts stale metadata and shows a privacy-safe fallback after an unavailable hover target', async () => {
    mocks.getCharacterCard.mockRejectedValue(Object.assign(new Error('not found'), { status: 404 }))
    wrapper = await mountPreview()
    const link = appendJumpCard(404)
    link.setAttribute('data-jump-image', 'private-portrait')
    link.setAttribute('data-jump-summary', 'private-summary')

    link.dispatchEvent(new MouseEvent('mouseover', {
      bubbles: true,
      clientX: 80,
      clientY: 90,
    }))
    await vi.advanceTimersByTimeAsync(150)
    await flushPromises()

    expect(mocks.getCharacterCard).toHaveBeenCalledWith(404)
    expect(link.getAttribute('data-jump-unavailable')).toBe('true')
    expect(link.getAttribute('aria-disabled')).toBe('true')
    expect(link.hasAttribute('data-jump-image')).toBe(false)
    expect(link.hasAttribute('data-jump-summary')).toBe(false)
    expect(document.body.textContent).toContain('人物卡暂不可用')
    expect(document.body.textContent).not.toContain('private-summary')
  })

  it('does not preview a private card even when the current owner can still fetch it', async () => {
    mocks.getCharacterCard.mockResolvedValue({
      ...card,
      id: 32,
      display_name: '不应出现在帖子预览中的私密人物',
      visibility: 'private',
    })
    wrapper = await mountPreview()
    const link = appendJumpCard(32)

    link.dispatchEvent(new FocusEvent('focusin', { bubbles: true }))
    await vi.advanceTimersByTimeAsync(150)
    await flushPromises()

    expect(link.getAttribute('data-jump-unavailable')).toBe('true')
    expect(document.body.textContent).toContain('人物卡暂不可用')
    expect(document.body.textContent).not.toContain('不应出现在帖子预览中的私密人物')
  })

  it('revalidates a previously public card instead of caching its visibility for the session', async () => {
    mocks.getCharacterCard
      .mockResolvedValueOnce(card)
      .mockResolvedValueOnce({ ...card, visibility: 'private' })
    wrapper = await mountPreview()
    const link = appendJumpCard(card.id)

    link.dispatchEvent(new FocusEvent('focusin', { bubbles: true }))
    await vi.advanceTimersByTimeAsync(150)
    await flushPromises()
    expect(document.body.textContent).toContain('莱恩·晨歌')

    link.dispatchEvent(new FocusEvent('focusout', { bubbles: true }))
    await vi.advanceTimersByTimeAsync(130)
    await flushPromises()
    link.dispatchEvent(new FocusEvent('focusin', { bubbles: true }))
    await vi.advanceTimersByTimeAsync(150)
    await flushPromises()

    expect(mocks.getCharacterCard).toHaveBeenCalledTimes(2)
    expect(link.getAttribute('data-jump-unavailable')).toBe('true')
    expect(document.body.textContent).toContain('人物卡暂不可用')
    expect(document.body.textContent).not.toContain('莱恩·晨歌')
  })
})
