import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  clearJumpReturn,
  getJumpReturn,
  handleJumpLinkClick,
  hydrateJumpCardImages,
  sanitizeJumpLinks,
} from '../utils/jumpLink'

const { getGuildMock } = vi.hoisted(() => ({
  getGuildMock: vi.fn(),
}))

vi.mock('@/api/guild', () => ({
  getGuild: getGuildMock,
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
        <a id="external-link" href="https://example.com/docs">外部</a>
      </div>
    `
    const container = document.getElementById('content') as HTMLElement

    sanitizeJumpLinks(container)

    const postLink = document.getElementById('post-link') as HTMLAnchorElement
    const guildLink = document.getElementById('guild-link') as HTMLAnchorElement
    const externalLink = document.getElementById('external-link') as HTMLAnchorElement

    expect(postLink.getAttribute('data-jump-href')).toBe('/community/post/12')
    expect(postLink.hasAttribute('href')).toBe(false)
    expect(postLink.classList.contains('jump-link')).toBe(true)

    expect(guildLink.getAttribute('data-jump-href')).toBe('/guild/5?tab=posts')
    expect(guildLink.getAttribute('data-jump-guild-id')).toBe('5')
    expect(guildLink.hasAttribute('href')).toBe(false)

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
})
