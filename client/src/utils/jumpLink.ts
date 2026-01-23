import type { Router } from 'vue-router'
import { getImageUrl } from '@/api/item'
import { getGuild, type Guild } from '@/api/guild'

const JUMP_LINK_SELECTOR = '.jump-link, a.jump-card, [data-jump-href], [data-jump-type]'
const EDITOR_SELECTOR = '.tiptap, [contenteditable="true"]'
const INTERNAL_PREFIXES = ['/archives/story/', '/community/post/', '/guild/', '/market/']
const JUMP_RETURN_KEY = 'jump_return_post'
const GUILD_ID_REGEX = /\/guild\/(\d+)/i
const GUILD_IMAGE_REGEX = /\/images\/(?:guild-avatar|guild-banner)\/(\d+)/i

const guildCache = new Map<number, Promise<Guild | null>>()

function parseGuildId(value: string | null | undefined): number | null {
  if (!value) return null
  const id = Number(value)
  if (!Number.isFinite(id) || id <= 0) return null
  return id
}

function resolveGuildIdFromHref(href: string | null | undefined): number | null {
  if (!href) return null
  const match = href.match(GUILD_ID_REGEX)
  if (!match) return null
  return parseGuildId(match[1])
}

function resolveGuildIdFromImageSrc(src: string | null | undefined): number | null {
  if (!src) return null
  const match = src.match(GUILD_IMAGE_REGEX)
  if (!match) return null
  return parseGuildId(match[1])
}

function resolveGuildIdForCard(card: HTMLElement): number | null {
  const direct = parseGuildId(card.getAttribute('data-jump-guild-id'))
  if (direct) return direct

  const href = card.getAttribute('data-jump-href') || card.getAttribute('href')
  const fromHref = resolveGuildIdFromHref(href)
  if (fromHref) return fromHref

  const dataImage = card.getAttribute('data-jump-image')
  const dataAvatar = card.getAttribute('data-jump-avatar')
  const fromDataImage = resolveGuildIdFromImageSrc(dataImage || dataAvatar)
  if (fromDataImage) return fromDataImage

  const image = card.querySelector<HTMLImageElement>('img[src]')
  if (image) {
    const fromImage = resolveGuildIdFromImageSrc(image.getAttribute('src'))
    if (fromImage) return fromImage
  }

  return null
}

function fetchGuildInfo(id: number): Promise<Guild | null> {
  const cached = guildCache.get(id)
  if (cached) return cached
  const request = getGuild(id)
    .then((res) => res.guild)
    .catch((error) => {
      console.error('获取公会信息失败:', error)
      return null
    })
  guildCache.set(id, request)
  return request
}

function buildGuildAvatarUrl(guild: Guild): string {
  if (!guild.avatar_url && !guild.avatar) return ''
  return getImageUrl('guild-avatar', guild.id, {
    w: 200,
    q: 80,
    v: guild.avatar_updated_at || guild.updated_at,
  })
}

function refreshGuildStoryCard(card: HTMLElement, guild: Guild) {
  const avatarUrl = buildGuildAvatarUrl(guild)
  if (!avatarUrl) return

  card.setAttribute('data-jump-image', avatarUrl)

  const media = card.querySelector<HTMLElement>('.jump-card__media')
  if (!media) return

  const overlay = media.querySelector('.jump-card__media-overlay')
  const existingImage = media.querySelector<HTMLImageElement>('.jump-card__image')
  if (existingImage) {
    existingImage.src = avatarUrl
    return
  }

  const image = document.createElement('img')
  image.className = 'jump-card__image'
  image.src = avatarUrl
  image.alt = ''
  const fallback = media.querySelector('.jump-card__media-fallback')
  if (fallback) {
    fallback.replaceWith(image)
    return
  }
  if (overlay) {
    media.insertBefore(image, overlay)
  } else {
    media.appendChild(image)
  }
}

function refreshGuildHomeCard(card: HTMLElement, guild: Guild) {
  const avatarUrl = buildGuildAvatarUrl(guild)
  if (!avatarUrl) return

  card.setAttribute('data-jump-avatar', avatarUrl)

  const avatarWrap = card.querySelector<HTMLElement>('.jump-card__author-avatar')
  if (!avatarWrap) return

  const existingImage = avatarWrap.querySelector<HTMLImageElement>('img')
  if (existingImage) {
    existingImage.src = avatarUrl
    return
  }

  avatarWrap.textContent = ''
  const image = document.createElement('img')
  image.src = avatarUrl
  image.alt = ''
  avatarWrap.appendChild(image)
}

export type JumpReturnPayload = {
  type: 'post'
  path: string
  title?: string
}

export type JumpReturnInfo = JumpReturnPayload & {
  createdAt: number
}

function resolveInternalHref(href: string, allowAnyPath: boolean): string | null {
  const trimmed = href.trim()
  if (!trimmed || trimmed === '#') return null

  if (trimmed.startsWith('#/')) {
    const path = trimmed.slice(1)
    if (allowAnyPath) return path
    return INTERNAL_PREFIXES.some((prefix) => path.startsWith(prefix)) ? path : null
  }

  if (trimmed.startsWith('/#/')) {
    const path = trimmed.slice(2)
    if (allowAnyPath) return path
    return INTERNAL_PREFIXES.some((prefix) => path.startsWith(prefix)) ? path : null
  }

  if (trimmed.startsWith('/')) {
    if (allowAnyPath) return trimmed
    return INTERNAL_PREFIXES.some((prefix) => trimmed.startsWith(prefix)) ? trimmed : null
  }

  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    try {
      const url = new URL(trimmed)
      if (url.origin !== window.location.origin) return null
      if (url.hash.startsWith('#/')) {
        const hashPath = url.hash.slice(1)
        if (allowAnyPath) return hashPath
        return INTERNAL_PREFIXES.some((prefix) => hashPath.startsWith(prefix)) ? hashPath : null
      }
      if (allowAnyPath) return `${url.pathname}${url.search}`
      return INTERNAL_PREFIXES.some((prefix) => url.pathname.startsWith(prefix))
        ? `${url.pathname}${url.search}`
        : null
    } catch {
      return null
    }
  }

  return null
}

function getJumpTarget(link: HTMLElement): string | null {
  const dataHref = link.getAttribute('data-jump-href')
  const href = dataHref || link.getAttribute('href') || ''
  const hasJumpMeta = Boolean(
    dataHref ||
      link.classList.contains('jump-link') ||
      link.classList.contains('jump-card') ||
      link.getAttribute('data-jump-type')
  )
  return resolveInternalHref(href, hasJumpMeta)
}

export function handleJumpLinkClick(
  event: MouseEvent,
  router: Router,
  options?: { ignoreEditor?: boolean; returnTo?: JumpReturnPayload }
) {
  const target = event.target
  const element = target instanceof Element ? target : (target instanceof Node ? target.parentElement : null)
  if (!element) return
  if (options?.ignoreEditor && element.closest(EDITOR_SELECTOR)) return

  const link =
    (element.closest(JUMP_LINK_SELECTOR) as HTMLElement | null) ||
    (element.closest('a[href]') as HTMLElement | null)
  if (!link) return

  const href = getJumpTarget(link)
  if (!href) return

  if (options?.returnTo?.path) {
    setJumpReturn(options.returnTo)
  }

  event.preventDefault()
  event.stopPropagation()
  if (typeof event.stopImmediatePropagation === 'function') {
    event.stopImmediatePropagation()
  }

  void router.push(href)
}

export function setJumpReturn(payload: JumpReturnPayload) {
  if (!payload.path) return
  const data: JumpReturnInfo = {
    ...payload,
    createdAt: Date.now(),
  }
  try {
    sessionStorage.setItem(JUMP_RETURN_KEY, JSON.stringify(data))
  } catch {
    // ignore storage failures
  }
}

export function getJumpReturn(): JumpReturnInfo | null {
  try {
    const raw = sessionStorage.getItem(JUMP_RETURN_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as JumpReturnInfo
    if (!parsed?.path || parsed.type !== 'post') return null
    return parsed
  } catch {
    return null
  }
}

export function clearJumpReturn() {
  try {
    sessionStorage.removeItem(JUMP_RETURN_KEY)
  } catch {
    // ignore storage failures
  }
}

export function sanitizeJumpLinks(container: HTMLElement | null) {
  if (!container) return
  const links = container.querySelectorAll<HTMLElement>(JUMP_LINK_SELECTOR)
  links.forEach((link) => {
    const href = getJumpTarget(link)
    if (!href) return
    link.setAttribute('data-jump-href', href)
    const guildId = resolveGuildIdFromHref(href)
    if (guildId) {
      link.setAttribute('data-jump-guild-id', String(guildId))
    }
    if (link.hasAttribute('href')) {
      link.removeAttribute('href')
    }
    if (!link.classList.contains('jump-link') && !link.classList.contains('jump-card')) {
      link.classList.add('jump-link')
    }
  })

  const anchors = container.querySelectorAll<HTMLAnchorElement>('a[href]')
  anchors.forEach((anchor) => {
    if (anchor.closest(JUMP_LINK_SELECTOR)) return
    const href = resolveInternalHref(anchor.getAttribute('href') || '', false)
    if (!href) return
    anchor.setAttribute('data-jump-href', href)
    const guildId = resolveGuildIdFromHref(href)
    if (guildId) {
      anchor.setAttribute('data-jump-guild-id', String(guildId))
    }
    anchor.classList.add('jump-link')
    anchor.removeAttribute('href')
  })
}

export function hydrateJumpCardImages(container: HTMLElement | null) {
  if (!container) return
  const cards = Array.from(container.querySelectorAll<HTMLElement>('.jump-card'))
  if (!cards.length) return

  cards.forEach((card) => {
    const guildId = resolveGuildIdForCard(card)
    if (!guildId) return

    void fetchGuildInfo(guildId).then((guild) => {
      if (!guild) return
      const variant = card.getAttribute('data-jump-variant') || ''
      if (variant === 'story-guild' || card.querySelector('.jump-card__media')) {
        refreshGuildStoryCard(card, guild)
        return
      }
      if (variant === 'guild-home' || card.querySelector('.jump-card__author-avatar')) {
        refreshGuildHomeCard(card, guild)
      }
    })
  })
}
