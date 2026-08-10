import type { Router } from 'vue-router'
import { getImageUrl, resolveApiUrl } from '@/api/item'
import { getGuild, type Guild } from '@/api/guild'
import { getPostEmbedPreview, type Post } from '@/api/post'
import { getCharacterCard, getCharacterCardPortraitUrl, type CharacterCard } from '@/api/characterCard'

const JUMP_LINK_SELECTOR = '.jump-link, a.jump-card, [data-jump-href], [data-jump-type]'
const EDITOR_SELECTOR = '.tiptap, [contenteditable="true"]'
const INTERNAL_PREFIXES = ['/archives/story/', '/community/post/', '/guild/', '/market/', '/rpdb/', '/character-cards/']
const JUMP_RETURN_KEY = 'jump_return_post'
const GUILD_ID_REGEX = /\/guild\/(\d+)/i
const GUILD_IMAGE_REGEX = /\/images\/(?:guild-avatar|guild-banner)\/(\d+)/i
const POST_ID_REGEX = /\/(?:community\/post|posts)\/(\d+)(?:$|[/?#])/i
const CHARACTER_CARD_ID_REGEX = /\/character-cards\/(\d+)(?:$|[/?#])/i

const guildCache = new Map<number, Promise<Guild | null>>()
const postRequests = new Map<number, Promise<EmbeddedPostResult>>()
const characterCardRequests = new Map<number, Promise<CharacterCardResult>>()

interface EmbeddedPost {
  post: Post
  author_name: string
  author_avatar?: string
}

type EmbeddedPostResult = EmbeddedPost | 'unavailable' | null
type CharacterCardResult = CharacterCard | 'unavailable'

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

function resolvePostIdFromHref(href: string | null | undefined): number | null {
  if (!href) return null
  const match = href.match(POST_ID_REGEX)
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

function resolvePostIdForCard(card: HTMLElement): number | null {
  const href = card.getAttribute('data-jump-href') || card.getAttribute('href')
  return resolvePostIdFromHref(href)
}

function resolveCharacterCardIdForCard(card: HTMLElement): number | null {
  const direct = parseGuildId(card.getAttribute('data-jump-id'))
  if (direct) return direct
  const href = card.getAttribute('data-jump-href') || card.getAttribute('href') || ''
  const match = href.match(CHARACTER_CARD_ID_REGEX)
  return match ? parseGuildId(match[1]) : null
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

function fetchEmbeddedPost(id: number): Promise<EmbeddedPostResult> {
  const pending = postRequests.get(id)
  if (pending) return pending

  const request: Promise<EmbeddedPostResult> = getPostEmbedPreview(id)
    .then((res) => res)
    .catch((error) => {
      const status = typeof error === 'object' && error ? (error as { status?: number }).status : undefined
      if (status === 403 || status === 404) return 'unavailable'
      console.error('获取嵌入帖子信息失败:', error)
      return null
    })
    .finally(() => {
      postRequests.delete(id)
    })
  postRequests.set(id, request)
  return request
}

function fetchCharacterCard(id: number): Promise<CharacterCardResult> {
  const pending = characterCardRequests.get(id)
  if (pending) return pending

  const request: Promise<CharacterCardResult> = getCharacterCard(id)
    .then((characterCard) => (
      characterCard.status === 'published' && characterCard.visibility === 'public'
        ? characterCard
        : 'unavailable'
    ))
    .catch((error) => {
      const status = typeof error === 'object' && error ? (error as { status?: number }).status : undefined
      if (status !== 403 && status !== 404) console.error('获取人物卡嵌入信息失败:', error)
      return 'unavailable'
    })
    .finally(() => {
      characterCardRequests.delete(id)
    })
  characterCardRequests.set(id, request)
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

function getPostCardLabel(post: Post): string {
  if (post.category !== 'event') return '公开帖子'
  if (post.event_type === 'server') return '服务器'
  if (post.event_type === 'guild') return '公会'
  return '活动'
}

function buildPostCoverUrl(post: Post): string {
  if (!post.cover_image) return ''
  return getImageUrl('post-cover', post.id, {
    w: 800,
    q: 80,
    v: post.cover_image_updated_at || post.updated_at,
  })
}

function refreshPostCardLogo(card: HTMLElement, imageUrl: string, title: string) {
  const logo = card.querySelector<HTMLElement>('.jump-card__logo')
  if (!logo) return

  const existingImage = logo.querySelector<HTMLImageElement>('.jump-card__logo-image')
  const fallback = logo.querySelector<HTMLElement>('.jump-card__logo-fallback')
  if (imageUrl) {
    if (existingImage) {
      existingImage.src = imageUrl
      return
    }
    const image = document.createElement('img')
    image.className = 'jump-card__logo-image'
    image.src = imageUrl
    image.alt = ''
    if (fallback) {
      fallback.replaceWith(image)
    } else {
      logo.appendChild(image)
    }
    return
  }

  existingImage?.remove()
  if (fallback) {
    fallback.textContent = title.slice(0, 1)
    return
  }
  const nextFallback = document.createElement('span')
  nextFallback.className = 'jump-card__logo-fallback'
  nextFallback.textContent = title.slice(0, 1)
  logo.appendChild(nextFallback)
}

function refreshPostCard(card: HTMLElement, embeddedPost: EmbeddedPost) {
  if (!card.isConnected) return

  const post = embeddedPost.post
  const title = post.title?.trim() || '未命名帖子'
  const author = embeddedPost.author_name?.trim() || '未知作者'
  const label = getPostCardLabel(post)
  const avatarUrl = resolveApiUrl(embeddedPost.author_avatar)
  const imageUrl = buildPostCoverUrl(post)

  card.removeAttribute('data-jump-unavailable')
  card.removeAttribute('aria-disabled')
  card.setAttribute('role', 'link')
  card.setAttribute('tabindex', '0')
  card.setAttribute('data-jump-label', label)
  card.setAttribute('data-jump-title', title)
  card.setAttribute('data-jump-author', author)
  if (avatarUrl) {
    card.setAttribute('data-jump-avatar', avatarUrl)
  } else {
    card.removeAttribute('data-jump-avatar')
  }
  if (imageUrl) {
    card.setAttribute('data-jump-image', imageUrl)
  } else {
    card.removeAttribute('data-jump-image')
  }

  const labelElement = card.querySelector<HTMLElement>('.jump-card__label')
  if (labelElement) labelElement.textContent = label
  const titleElement = card.querySelector<HTMLElement>('.jump-card__title')
  if (titleElement) titleElement.textContent = title
  const authorElement = card.querySelector<HTMLElement>('.jump-card__stat-value')
  if (authorElement) authorElement.textContent = author
  refreshPostCardLogo(card, imageUrl, title)
}

function refreshUnavailablePostCard(card: HTMLElement) {
  if (!card.isConnected) return

  const label = '帖子引用'
  const title = '引用的帖子当前不可用'
  card.setAttribute('data-jump-unavailable', 'true')
  card.setAttribute('aria-disabled', 'true')
  card.setAttribute('tabindex', '-1')
  card.setAttribute('data-jump-label', label)
  card.setAttribute('data-jump-title', title)
  card.removeAttribute('data-jump-author')
  card.removeAttribute('data-jump-avatar')
  card.removeAttribute('data-jump-image')

  const labelElement = card.querySelector<HTMLElement>('.jump-card__label')
  if (labelElement) labelElement.textContent = label
  const titleElement = card.querySelector<HTMLElement>('.jump-card__title')
  if (titleElement) titleElement.textContent = title
  const authorElement = card.querySelector<HTMLElement>('.jump-card__stat-value')
  if (authorElement) authorElement.textContent = '无法访问'

  const logo = card.querySelector<HTMLElement>('.jump-card__logo')
  if (!logo) return
  logo.textContent = ''
  const fallback = document.createElement('span')
  fallback.className = 'jump-card__logo-fallback'
  fallback.textContent = '!'
  logo.appendChild(fallback)
}

function getCharacterCardName(characterCard: CharacterCard) {
  const explicit = characterCard.display_name?.trim()
  if (explicit) return explicit
  return [characterCard.first_name, characterCard.last_name]
    .map((part) => part?.trim())
    .filter(Boolean)
    .join(' ') || '未命名人物'
}

function refreshCharacterPortrait(card: HTMLElement, imageUrl: string, title: string) {
  const portrait = card.querySelector<HTMLElement>('.jump-card__character-portrait')
  if (!portrait) return
  portrait.textContent = ''
  if (imageUrl) {
    const image = document.createElement('img')
    image.src = imageUrl
    image.alt = `${title}的角色肖像`
    portrait.appendChild(image)
    return
  }
  const fallback = document.createElement('i')
  fallback.className = 'ri-user-star-line'
  fallback.setAttribute('aria-hidden', 'true')
  portrait.appendChild(fallback)
}

const CHARACTER_CARD_PRIVATE_ATTRS = [
  'data-jump-author',
  'data-jump-avatar',
  'data-jump-image',
  'data-jump-summary',
  'data-jump-status',
  'data-jump-visibility',
]

function renderCharacterCardState(
  card: HTMLElement,
  state: { kind: string; title: string; subtitle: string; summary: string; icon: string; actionIcon: string },
) {
  card.textContent = ''

  const rail = document.createElement('span')
  rail.className = 'jump-card__character-rail'

  const portrait = document.createElement('span')
  portrait.className = 'jump-card__character-portrait'
  const portraitIcon = document.createElement('i')
  portraitIcon.className = state.icon
  portraitIcon.setAttribute('aria-hidden', 'true')
  portrait.appendChild(portraitIcon)

  const body = document.createElement('span')
  body.className = 'jump-card__character-body'
  const kind = document.createElement('span')
  kind.className = 'jump-card__character-kind'
  const kindIcon = document.createElement('i')
  kindIcon.className = state.icon
  kindIcon.setAttribute('aria-hidden', 'true')
  kind.append(kindIcon, state.kind)
  const title = document.createElement('span')
  title.className = 'jump-card__title'
  title.textContent = state.title
  const subtitle = document.createElement('span')
  subtitle.className = 'jump-card__character-subtitle'
  subtitle.textContent = state.subtitle
  const summary = document.createElement('span')
  summary.className = 'jump-card__character-summary'
  summary.textContent = state.summary
  body.append(kind, title, subtitle, summary)

  const action = document.createElement('span')
  action.className = 'jump-card__character-open'
  const actionIcon = document.createElement('i')
  actionIcon.className = state.actionIcon
  actionIcon.setAttribute('aria-hidden', 'true')
  action.appendChild(actionIcon)

  card.append(rail, portrait, body, action)
}

function prepareCharacterCardVerification(card: HTMLElement) {
  card.removeAttribute('data-jump-unavailable')
  card.removeAttribute('data-jump-verified')
  card.setAttribute('data-jump-pending', 'true')
  card.setAttribute('data-jump-safe-placeholder', 'true')
  card.setAttribute('aria-disabled', 'true')
  card.setAttribute('role', 'group')
  card.setAttribute('tabindex', '-1')
  card.setAttribute('data-jump-label', '人物卡引用')
  card.setAttribute('data-jump-title', '正在验证人物卡访问权限')
  card.removeAttribute('aria-label')
  card.removeAttribute('title')
  CHARACTER_CARD_PRIVATE_ATTRS.forEach((attr) => card.removeAttribute(attr))
  renderCharacterCardState(card, {
    kind: '人物卡引用',
    title: '人物卡',
    subtitle: '正在验证访问权限',
    summary: '名称、摘要和肖像将在确认可访问后显示。',
    icon: 'ri-shield-keyhole-line',
    actionIcon: 'ri-loader-4-line',
  })
}

function refreshCharacterCard(card: HTMLElement, characterCard: CharacterCard) {
  if (!card.isConnected) return

  const title = getCharacterCardName(characterCard)
  const subtitle = characterCard.title?.trim() || characterCard.full_title?.trim() || '人物档案'
  const summary = characterCard.summary?.trim()
    || [characterCard.race, characterCard.class].filter(Boolean).join(' · ')
    || '角色摘要尚未填写。'
  const imageUrl = getCharacterCardPortraitUrl(characterCard, { w: 360, q: 84 })

  card.removeAttribute('data-jump-unavailable')
  card.removeAttribute('aria-disabled')
  card.removeAttribute('data-jump-pending')
  card.removeAttribute('data-jump-safe-placeholder')
  card.setAttribute('data-jump-verified', 'true')
  card.setAttribute('role', 'link')
  card.setAttribute('tabindex', '0')
  card.setAttribute('data-jump-id', String(characterCard.id))
  card.setAttribute('data-jump-type', 'character_card')
  card.setAttribute('data-jump-href', `/character-cards/${characterCard.id}`)
  card.setAttribute('data-jump-label', '人物卡')
  card.setAttribute('data-jump-title', '已验证的人物卡')
  card.setAttribute('aria-label', `查看人物卡：${title}`)
  CHARACTER_CARD_PRIVATE_ATTRS.forEach((attr) => card.removeAttribute(attr))

  const kind = card.querySelector<HTMLElement>('.jump-card__character-kind')
  if (kind) {
    kind.textContent = ''
    const icon = document.createElement('i')
    icon.className = 'ri-id-card-line'
    icon.setAttribute('aria-hidden', 'true')
    kind.append(icon, '人物卡')
  }
  const titleElement = card.querySelector<HTMLElement>('.jump-card__title')
  if (titleElement) titleElement.textContent = title
  const subtitleElement = card.querySelector<HTMLElement>('.jump-card__character-subtitle')
  if (subtitleElement) subtitleElement.textContent = subtitle
  const summaryElement = card.querySelector<HTMLElement>('.jump-card__character-summary')
  if (summaryElement) summaryElement.textContent = summary
  refreshCharacterPortrait(card, imageUrl, title)
}

function refreshUnavailableCharacterCard(card: HTMLElement) {
  if (!card.isConnected) return

  card.removeAttribute('data-jump-pending')
  card.removeAttribute('data-jump-verified')
  card.setAttribute('data-jump-unavailable', 'true')
  card.setAttribute('data-jump-safe-placeholder', 'true')
  card.setAttribute('aria-disabled', 'true')
  card.setAttribute('role', 'group')
  card.setAttribute('tabindex', '-1')
  card.setAttribute('data-jump-label', '人物卡引用')
  card.setAttribute('data-jump-title', '人物卡暂不可用')
  card.removeAttribute('aria-label')
  CHARACTER_CARD_PRIVATE_ATTRS.forEach((attr) => card.removeAttribute(attr))
  renderCharacterCardState(card, {
    kind: '人物卡引用',
    title: '人物卡暂不可用',
    subtitle: '无法访问',
    summary: '这份档案可能已删除、设为私密、尚未发布，或暂时无法验证。',
    icon: 'ri-link-unlink-m',
    actionIcon: 'ri-lock-line',
  })
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
  event: MouseEvent | KeyboardEvent,
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

  if (link.getAttribute('data-jump-unavailable') === 'true' || link.getAttribute('data-jump-pending') === 'true') {
    event.preventDefault()
    event.stopPropagation()
    return
  }

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

export function handleJumpLinkKeydown(
  event: KeyboardEvent,
  router: Router,
  options?: { ignoreEditor?: boolean; returnTo?: JumpReturnPayload },
) {
  if (event.key !== 'Enter' && event.key !== ' ') return
  const target = event.target
  const element = target instanceof Element ? target : null
  const link = element?.closest<HTMLElement>(JUMP_LINK_SELECTOR)
  if (!link) return
  if (link.getAttribute('role') !== 'link' && !link.classList.contains('jump-link') && !link.classList.contains('jump-card')) {
    return
  }
  handleJumpLinkClick(event, router, options)
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
    const isCharacterCard = link.getAttribute('data-jump-type') === 'character_card'
    if (isCharacterCard) prepareCharacterCardVerification(link)
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

export function hydrateJumpCards(container: HTMLElement | null) {
  if (!container) return
  const cards = Array.from(container.querySelectorAll<HTMLElement>('.jump-card'))
  if (!cards.length) return

  cards.forEach((card) => {
    const characterCardId = resolveCharacterCardIdForCard(card)
    if (characterCardId && card.getAttribute('data-jump-type') === 'character_card') {
      prepareCharacterCardVerification(card)
      void fetchCharacterCard(characterCardId).then((characterCard) => {
        if (characterCard === 'unavailable') {
          refreshUnavailableCharacterCard(card)
          return
        }
        if (characterCard) refreshCharacterCard(card, characterCard)
      })
      return
    }

    const postId = resolvePostIdForCard(card)
    if (postId) {
      void fetchEmbeddedPost(postId).then((post) => {
        if (post === 'unavailable') {
          refreshUnavailablePostCard(card)
          return
        }
        if (post) refreshPostCard(card, post)
      })
    }

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

export function hydrateJumpCardImages(container: HTMLElement | null) {
  hydrateJumpCards(container)
}
