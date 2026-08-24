import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { CharacterCard } from '@/api/characterCard'
import {
  buildCharacterCardJumpPlaceholder,
  handleJumpLinkClick,
  sanitizeJumpLinks,
} from './jumpLink'

const getCharacterCardMock = vi.hoisted(() => vi.fn())

vi.mock('@/api/characterCard', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/api/characterCard')>(),
  getCharacterCard: getCharacterCardMock,
}))

function makeCharacterCard(overrides: Partial<CharacterCard> = {}): CharacterCard {
  return {
    id: 27,
    user_id: 8,
    first_name: '伊莉娅',
    last_name: '星语',
    display_name: '伊莉娅·星语',
    title: '月之女祭司',
    full_title: '',
    race: '暗夜精灵',
    class: '牧师',
    eye_color: '银色',
    eye_color_hex: '#c9d5e7',
    age: '',
    height: '',
    weight: '',
    birthplace: '',
    residence: '',
    relationship_status: '',
    class_color: '',
    name_color: '',
    summary: '只在重新验证后显示的摘要',
    background_story: '',
    first_impression: '',
    impressions: [],
    other_content: '',
    portrait_image_url: '/api/v1/images/character-card-portrait/27?v=old',
    portrait_image_updated_at: '2026-08-24T08:00:00Z',
    portraits: [],
    status: 'published',
    visibility: 'public',
    review_status: 'approved',
    updated_at: '2026-08-24T08:00:00Z',
    ...overrides,
  }
}

function flushAsyncHydration() {
  return new Promise(resolve => setTimeout(resolve, 0))
}

beforeEach(() => {
  getCharacterCardMock.mockReset()
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('secure mobile character-card jump hydration', () => {
  it('redacts stale persisted identity before an approved public response hydrates it', async () => {
    let resolveCard!: (card: CharacterCard) => void
    getCharacterCardMock.mockReturnValue(new Promise<CharacterCard>((resolve) => {
      resolveCard = resolve
    }))
    document.body.innerHTML = `
      <div id="content">
        <span
          class="jump-card jump-card--character-card"
          data-jump-type="character_card"
          data-jump-id="27"
          data-jump-href="/character-cards/27"
          data-jump-title="不应泄漏的名字"
          data-jump-author="旧作者"
          data-jump-avatar="/private-avatar.jpg"
          data-jump-image="/private-portrait.jpg"
          data-jump-summary="不应泄漏的摘要"
          data-jump-status="published"
          data-jump-visibility="public"
          aria-label="查看人物卡：不应泄漏的名字"
        >
          <span class="jump-card__character-portrait"><img src="/private-portrait.jpg"></span>
          <span class="jump-card__title">不应泄漏的名字</span>
          <span class="jump-card__character-summary">不应泄漏的摘要</span>
        </span>
      </div>
    `
    const container = document.getElementById('content')!

    sanitizeJumpLinks(container)

    const card = container.querySelector<HTMLElement>('.jump-card')!
    expect(card.getAttribute('data-jump-pending')).toBe('true')
    expect(card.getAttribute('aria-disabled')).toBe('true')
    expect(card.textContent).not.toContain('不应泄漏')
    expect(card.querySelector('img')).toBeNull()
    expect(card.hasAttribute('data-jump-author')).toBe(false)
    expect(card.hasAttribute('data-jump-avatar')).toBe(false)
    expect(card.hasAttribute('data-jump-image')).toBe(false)
    expect(card.hasAttribute('data-jump-summary')).toBe(false)
    expect(card.hasAttribute('data-jump-status')).toBe(false)
    expect(card.hasAttribute('data-jump-visibility')).toBe(false)
    expect(card.hasAttribute('aria-label')).toBe(false)
    expect(getCharacterCardMock).toHaveBeenCalledOnce()

    resolveCard(makeCharacterCard())
    await flushAsyncHydration()

    expect(card.getAttribute('data-jump-verified')).toBe('true')
    expect(card.getAttribute('data-jump-pending')).toBeNull()
    expect(card.getAttribute('data-jump-href')).toBe('/character-cards/27')
    expect(card.textContent).toContain('伊莉娅·星语')
    expect(card.textContent).toContain('只在重新验证后显示的摘要')
    const portrait = card.querySelector<HTMLImageElement>('img')!
    expect(portrait.src).toContain('/api/v1/images/character-card-portrait/27')
    expect(new URL(portrait.src).searchParams.get('v')).toBe('2026-08-24T08:00:00Z')
    expect(card.hasAttribute('data-jump-summary')).toBe(false)
    expect(card.hasAttribute('data-jump-image')).toBe(false)
  })

  it('keeps an owner-readable private response neutral and non-clickable', async () => {
    getCharacterCardMock.mockResolvedValue(makeCharacterCard({
      id: 28,
      display_name: '已经转私密的角色',
      summary: '旧私密摘要',
      visibility: 'private',
    }))
    document.body.innerHTML = `
      <div id="content">
        <span class="jump-card" data-jump-type="character_card" data-jump-id="28" data-jump-href="/character-cards/28">
          <img src="/private-portrait.jpg"><span>已经转私密的角色</span>
        </span>
      </div>
    `
    const container = document.getElementById('content')!

    sanitizeJumpLinks(container)
    await flushAsyncHydration()

    const card = container.querySelector<HTMLElement>('.jump-card')!
    expect(card.getAttribute('data-jump-unavailable')).toBe('true')
    expect(card.getAttribute('aria-disabled')).toBe('true')
    expect(card.textContent).not.toContain('已经转私密的角色')
    expect(card.textContent).not.toContain('旧私密摘要')
    expect(card.querySelector('img')).toBeNull()

    const router = { push: vi.fn() }
    card.addEventListener('click', event => handleJumpLinkClick(event, router as never))
    const event = new MouseEvent('click', { bubbles: true, cancelable: true })
    card.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    expect(router.push).not.toHaveBeenCalled()
  })

  it('deduplicates concurrent verification requests for the same card id', async () => {
    let resolveCard!: (card: CharacterCard) => void
    getCharacterCardMock.mockReturnValue(new Promise<CharacterCard>((resolve) => {
      resolveCard = resolve
    }))
    document.body.innerHTML = `
      <div id="content">
        <span class="jump-card" data-jump-type="character_card" data-jump-id="31" data-jump-href="/character-cards/31"></span>
        <span class="jump-card" data-jump-type="character_card" data-jump-id="31" data-jump-href="/character-cards/31"></span>
      </div>
    `

    sanitizeJumpLinks(document.getElementById('content'))

    expect(getCharacterCardMock).toHaveBeenCalledTimes(1)
    resolveCard(makeCharacterCard({ id: 31 }))
    await flushAsyncHydration()
    expect(document.querySelectorAll('[data-jump-verified="true"]')).toHaveLength(2)
  })

  it('does not hydrate a DOMParser document until its safe HTML is mounted in the main document', async () => {
    getCharacterCardMock.mockResolvedValue(makeCharacterCard({ id: 35 }))
    const parsed = new DOMParser().parseFromString(`
      <span class="jump-card" data-jump-type="character_card" data-jump-id="35" data-jump-href="/character-cards/35">
        <img src="/stale-private.jpg"><span>旧人物名称</span>
      </span>
    `, 'text/html')

    expect(parsed.body.isConnected).toBe(true)
    sanitizeJumpLinks(parsed.body)

    expect(parsed.body.textContent).not.toContain('旧人物名称')
    expect(parsed.body.querySelector('img')).toBeNull()
    expect(getCharacterCardMock).not.toHaveBeenCalled()

    const mounted = document.createElement('div')
    mounted.innerHTML = parsed.body.innerHTML
    document.body.appendChild(mounted)
    await flushAsyncHydration()

    expect(getCharacterCardMock).toHaveBeenCalledTimes(1)
    expect(getCharacterCardMock).toHaveBeenCalledWith(35)
    expect(mounted.querySelector('[data-jump-verified="true"]')).not.toBeNull()
  })

  it('builds an insertion placeholder without mutable character display data', () => {
    const html = buildCharacterCardJumpPlaceholder(41)
    const container = document.createElement('div')
    container.innerHTML = html
    const card = container.firstElementChild as HTMLElement

    expect(card.getAttribute('data-jump-href')).toBe('/character-cards/41')
    expect(card.getAttribute('data-jump-type')).toBe('character_card')
    expect(card.getAttribute('data-jump-id')).toBe('41')
    expect(card.getAttribute('data-jump-safe-placeholder')).toBe('true')
    expect(card.hasAttribute('data-jump-author')).toBe(false)
    expect(card.hasAttribute('data-jump-avatar')).toBe(false)
    expect(card.hasAttribute('data-jump-image')).toBe(false)
    expect(card.hasAttribute('data-jump-summary')).toBe(false)
    expect(card.hasAttribute('data-jump-status')).toBe(false)
    expect(card.hasAttribute('data-jump-visibility')).toBe(false)
    expect(card.textContent).toBe('')
  })

  it('preserves navigation for ordinary internal links', () => {
    document.body.innerHTML = '<div id="content"><a href="/community/post/12">Open post</a></div>'
    const container = document.getElementById('content')!
    sanitizeJumpLinks(container)
    const link = container.querySelector<HTMLElement>('a')!
    const router = { push: vi.fn() }
    link.addEventListener('click', event => handleJumpLinkClick(event, router as never))

    link.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))

    expect(router.push).toHaveBeenCalledWith('/posts/12')
  })
})
