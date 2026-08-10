import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  clearJumpReturn,
  getJumpReturn,
  handleJumpLinkClick,
  hydrateJumpCards,
  hydrateJumpCardImages,
  sanitizeJumpLinks,
} from '../utils/jumpLink'

const { getGuildMock, getPostEmbedPreviewMock, getCharacterCardMock } = vi.hoisted(() => ({
  getGuildMock: vi.fn(),
  getPostEmbedPreviewMock: vi.fn(),
  getCharacterCardMock: vi.fn(),
}))

vi.mock('@/api/guild', () => ({
  getGuild: getGuildMock,
}))

vi.mock('@/api/post', () => ({
  getPostEmbedPreview: getPostEmbedPreviewMock,
}))

vi.mock('@/api/characterCard', () => ({
  getCharacterCard: getCharacterCardMock,
  getCharacterCardPortraitUrl: (card: { id: number; portrait_image_url?: string }) => (
    card.portrait_image_url ? `/api/v1/images/character-card-portrait/${card.id}` : ''
  ),
}))

describe('jumpLink utils', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    sessionStorage.clear()
    vi.clearAllMocks()
    clearJumpReturn()
  })

  it('sanitizes internal links and preserves external ones', () => {
    document.body.innerHTML = `
      <div id="content">
        <a id="post-link" href="/community/post/12">帖子</a>
        <a id="guild-link" href="${window.location.origin}/guild/5?tab=posts">公会</a>
        <a id="rpdb-link" href="/rpdb/10">RP数据库</a>
        <a id="external-link" href="https://example.com/docs">外部</a>
      </div>
    `
    const container = document.getElementById('content') as HTMLElement

    sanitizeJumpLinks(container)

    const postLink = document.getElementById('post-link') as HTMLAnchorElement
    const guildLink = document.getElementById('guild-link') as HTMLAnchorElement
    const rpdbLink = document.getElementById('rpdb-link') as HTMLAnchorElement
    const externalLink = document.getElementById('external-link') as HTMLAnchorElement

    expect(postLink.getAttribute('data-jump-href')).toBe('/community/post/12')
    expect(postLink.hasAttribute('href')).toBe(false)
    expect(postLink.classList.contains('jump-link')).toBe(true)

    expect(guildLink.getAttribute('data-jump-href')).toBe('/guild/5?tab=posts')
    expect(guildLink.getAttribute('data-jump-guild-id')).toBe('5')
    expect(guildLink.hasAttribute('href')).toBe(false)

    expect(rpdbLink.getAttribute('data-jump-href')).toBe('/rpdb/10')
    expect(rpdbLink.hasAttribute('href')).toBe(false)
    expect(rpdbLink.classList.contains('jump-link')).toBe(true)

    expect(externalLink.getAttribute('href')).toBe('https://example.com/docs')
    expect(externalLink.hasAttribute('data-jump-href')).toBe(false)
  })

  it('handles click navigation and stores return target', async () => {
    document.body.innerHTML = `<a id="jump" class="jump-link" href="/community/post/36">查看帖子</a>`
    const link = document.getElementById('jump') as HTMLAnchorElement
    const router = { push: vi.fn().mockResolvedValue(undefined) }

    link.addEventListener('click', (event) => {
      handleJumpLinkClick(event as MouseEvent, router as any, {
        returnTo: { type: 'post', path: '/community/post/18', title: '来源帖子' },
      })
    })

    link.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await Promise.resolve()

    expect(router.push).toHaveBeenCalledWith('/community/post/36')
    expect(getJumpReturn()).toMatchObject({
      type: 'post',
      path: '/community/post/18',
      title: '来源帖子',
    })
  })

  it('hydrates guild jump cards with the fetched avatar image', async () => {
    getGuildMock.mockResolvedValue({
      guild: {
        id: 5,
        avatar: '/guild-avatar.png',
        updated_at: '2026-04-07T00:00:00Z',
        avatar_updated_at: '2026-04-07T08:00:00Z',
      },
    })

    document.body.innerHTML = `
      <div id="content">
        <div class="jump-card" data-jump-href="/guild/5" data-jump-variant="story-guild">
          <div class="jump-card__media">
            <div class="jump-card__media-fallback">G</div>
            <div class="jump-card__media-overlay"></div>
          </div>
        </div>
      </div>
    `
    const container = document.getElementById('content') as HTMLElement

    hydrateJumpCardImages(container)
    await new Promise((resolve) => setTimeout(resolve, 0))

    const image = container.querySelector('.jump-card__image') as HTMLImageElement | null
    const card = container.querySelector('.jump-card') as HTMLElement | null
    expect(getGuildMock).toHaveBeenCalledWith(5)
    expect(image).not.toBeNull()
    expect(card?.getAttribute('data-jump-image')).toContain('/api/v1/images/guild-avatar/5')
    expect(image?.getAttribute('src')).toContain('/api/v1/images/guild-avatar/5')
  })

  it('hydrates embedded post cards with current source data', async () => {
    getPostEmbedPreviewMock.mockResolvedValue({
      post: {
        id: 12,
        title: '更新后的战报',
        category: 'event',
        event_type: 'server',
        cover_image: '/uploads/post-cover.jpg',
        cover_image_updated_at: '2026-07-18T08:00:00Z',
        updated_at: '2026-07-18T08:00:00Z',
      },
      author_name: '新作者',
      author_avatar: '/uploads/avatar.jpg',
    })

    document.body.innerHTML = `
      <div id="content">
        <span class="jump-card jump-card--guild-home" data-jump-href="/community/post/12" data-jump-type="post">
          <span class="jump-card__logo"><span class="jump-card__logo-fallback">旧</span></span>
          <span class="jump-card__content">
            <span class="jump-card__label">公开帖子</span>
            <span class="jump-card__title">旧标题</span>
          </span>
          <span class="jump-card__stat"><span class="jump-card__stat-value">旧作者</span></span>
        </span>
      </div>
    `
    const container = document.getElementById('content') as HTMLElement

    hydrateJumpCards(container)
    await new Promise((resolve) => setTimeout(resolve, 0))

    const card = container.querySelector('.jump-card') as HTMLElement
    expect(getPostEmbedPreviewMock).toHaveBeenCalledWith(12)
    expect(card.getAttribute('data-jump-label')).toBe('服务器')
    expect(card.getAttribute('data-jump-title')).toBe('更新后的战报')
    expect(card.getAttribute('data-jump-author')).toBe('新作者')
    expect(card.getAttribute('data-jump-avatar')).toBe('/uploads/avatar.jpg')
    expect(card.getAttribute('data-jump-image')).toContain('/api/v1/images/post-cover/12')
    expect(card.querySelector('.jump-card__label')?.textContent).toBe('服务器')
    expect(card.querySelector('.jump-card__title')?.textContent).toBe('更新后的战报')
    expect(card.querySelector('.jump-card__stat-value')?.textContent).toBe('新作者')
    expect(card.querySelector('.jump-card__logo-image')?.getAttribute('src')).toContain('/api/v1/images/post-cover/12')
  })

  it('replaces inaccessible embedded posts with an unavailable card', async () => {
    getPostEmbedPreviewMock.mockRejectedValue(Object.assign(new Error('帖子不存在'), { status: 404 }))

    document.body.innerHTML = `
      <div id="content">
        <span class="jump-card jump-card--guild-home" data-jump-href="/community/post/20" data-jump-type="post">
          <span class="jump-card__logo"><img class="jump-card__logo-image" src="/old-cover.jpg"></span>
          <span class="jump-card__content">
            <span class="jump-card__label">公开帖子</span>
            <span class="jump-card__title">旧标题</span>
          </span>
          <span class="jump-card__stat"><span class="jump-card__stat-value">旧作者</span></span>
        </span>
      </div>
    `
    const container = document.getElementById('content') as HTMLElement

    hydrateJumpCards(container)
    await new Promise((resolve) => setTimeout(resolve, 0))

    const card = container.querySelector('.jump-card') as HTMLElement
    expect(card.getAttribute('data-jump-unavailable')).toBe('true')
    expect(card.getAttribute('aria-disabled')).toBe('true')
    expect(card.getAttribute('data-jump-title')).toBe('引用的帖子当前不可用')
    expect(card.hasAttribute('data-jump-author')).toBe(false)
    expect(card.hasAttribute('data-jump-image')).toBe(false)
    expect(card.querySelector('.jump-card__title')?.textContent).toBe('引用的帖子当前不可用')
    expect(card.querySelector('.jump-card__stat-value')?.textContent).toBe('无法访问')
    expect(card.querySelector('.jump-card__logo-fallback')?.textContent).toBe('!')

    const router = { push: vi.fn() }
    card.addEventListener('click', (event) => handleJumpLinkClick(event as MouseEvent, router as any))
    card.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    expect(router.push).not.toHaveBeenCalled()
  })

  it('redacts persisted character-card snapshots before validation and stays redacted on network failure', async () => {
    getCharacterCardMock.mockRejectedValue(new Error('network unavailable'))
    vi.spyOn(console, 'error').mockImplementation(() => {})
    document.body.innerHTML = `
      <div id="content">
        <span
          class="jump-card jump-card--character-card"
          data-jump-href="/character-cards/27"
          data-jump-id="27"
          data-jump-type="character_card"
          data-jump-title="不应泄漏的角色名"
          data-jump-summary="不应泄漏的摘要"
          data-jump-image="/private-portrait.jpg"
          aria-label="查看人物卡：不应泄漏的角色名"
        >
          <span class="jump-card__character-portrait"><img src="/private-portrait.jpg"></span>
          <span class="jump-card__character-body">
            <span class="jump-card__character-kind">人物卡</span>
            <span class="jump-card__title">不应泄漏的角色名</span>
            <span class="jump-card__character-subtitle">秘密称号</span>
            <span class="jump-card__character-summary">不应泄漏的摘要</span>
          </span>
        </span>
      </div>
    `
    const container = document.getElementById('content') as HTMLElement

    sanitizeJumpLinks(container)
    const card = container.querySelector('.jump-card') as HTMLElement
    expect(card.getAttribute('data-jump-pending')).toBe('true')
    expect(card.getAttribute('aria-disabled')).toBe('true')
    expect(card.textContent).toContain('正在验证访问权限')
    expect(card.textContent).not.toContain('不应泄漏')
    expect(card.textContent).not.toContain('秘密称号')
    expect(card.querySelector('img')).toBeNull()
    expect(card.hasAttribute('aria-label')).toBe(false)
    expect(getCharacterCardMock).not.toHaveBeenCalled()

    hydrateJumpCards(container)
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(getCharacterCardMock).toHaveBeenCalledWith(27)
    expect(card.getAttribute('data-jump-unavailable')).toBe('true')
    expect(card.getAttribute('aria-disabled')).toBe('true')
    expect(card.getAttribute('data-jump-title')).toBe('人物卡暂不可用')
    expect(card.hasAttribute('data-jump-summary')).toBe(false)
    expect(card.hasAttribute('data-jump-image')).toBe(false)
    expect(card.textContent).toContain('人物卡暂不可用')
    expect(card.textContent).not.toContain('不应泄漏')
    expect(card.querySelector('img')).toBeNull()

    const router = { push: vi.fn() }
    card.addEventListener('click', (event) => handleJumpLinkClick(event as MouseEvent, router as any))
    card.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    expect(router.push).not.toHaveBeenCalled()
  })

  it('adds character identity and portrait only after a fresh public-access check succeeds', async () => {
    getCharacterCardMock.mockResolvedValue({
      id: 29,
      user_id: 1,
      display_name: '刚验证的公开角色',
      first_name: '',
      last_name: '',
      title: '公开称号',
      full_title: '',
      race: '人类',
      class: '法师',
      summary: '鉴权成功后才显示的摘要',
      portrait_image_url: '/portrait.jpg',
      status: 'published',
      visibility: 'public',
      updated_at: '2026-08-10T00:00:00Z',
    })
    document.body.innerHTML = `
      <div id="content">
        <span class="jump-card jump-card--character-card" data-jump-href="/character-cards/29" data-jump-id="29" data-jump-type="character_card">
          <span class="jump-card__title">旧快照名称</span>
        </span>
      </div>
    `
    const container = document.getElementById('content') as HTMLElement

    sanitizeJumpLinks(container)
    const embedded = container.querySelector<HTMLElement>('.jump-card')!
    expect(embedded.textContent).not.toContain('旧快照名称')
    expect(embedded.textContent).not.toContain('刚验证的公开角色')

    hydrateJumpCards(container)
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(embedded.getAttribute('data-jump-verified')).toBe('true')
    expect(embedded.getAttribute('data-jump-pending')).toBeNull()
    expect(embedded.textContent).toContain('刚验证的公开角色')
    expect(embedded.textContent).toContain('鉴权成功后才显示的摘要')
    expect(embedded.querySelector('img')?.getAttribute('src')).toContain('/api/v1/images/character-card-portrait/29')
    expect(embedded.hasAttribute('data-jump-summary')).toBe(false)
    expect(embedded.hasAttribute('data-jump-image')).toBe(false)
  })

  it('redacts a character-card embed that is still owner-readable but no longer public', async () => {
    getCharacterCardMock.mockResolvedValue({
      id: 28,
      user_id: 1,
      display_name: '已经转私密的角色',
      first_name: '',
      last_name: '',
      title: '旧称号',
      full_title: '',
      race: '',
      class: '',
      summary: '旧摘要',
      portrait_image_url: '/private-portrait.jpg',
      status: 'published',
      visibility: 'private',
      updated_at: '2026-08-10T00:00:00Z',
    })
    document.body.innerHTML = `
      <div id="content">
        <span class="jump-card jump-card--character-card" data-jump-href="/character-cards/28" data-jump-id="28" data-jump-type="character_card">
          <span class="jump-card__character-portrait"><img src="/private-portrait.jpg"></span>
          <span class="jump-card__title">已经转私密的角色</span>
          <span class="jump-card__character-subtitle">旧称号</span>
          <span class="jump-card__character-summary">旧摘要</span>
        </span>
      </div>
    `

    hydrateJumpCards(document.getElementById('content'))
    await new Promise((resolve) => setTimeout(resolve, 0))

    const embedded = document.querySelector<HTMLElement>('.jump-card')!
    expect(embedded.getAttribute('data-jump-unavailable')).toBe('true')
    expect(embedded.textContent).toContain('人物卡暂不可用')
    expect(embedded.textContent).not.toContain('已经转私密的角色')
    expect(embedded.querySelector('img')).toBeNull()
  })
})
