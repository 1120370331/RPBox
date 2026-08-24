import type { Router } from 'vue-router'
import { getCharacterCard, isApprovedPublicCharacterCard, type CharacterCard } from '@/api/characterCard'
import { getImageUrl, resolveApiUrl } from '@/api/image'
import i18n from '@/i18n'

const JUMP_LINK_SELECTOR = '.jump-link, a.jump-card, [data-jump-href], [data-jump-type]'
const EDITOR_SELECTOR = '.tiptap, [contenteditable="true"]'
const CHARACTER_CARD_SELECTOR = '[data-jump-type="character_card"]'
const INTERNAL_PREFIXES = ['/archives/story/', '/community/post/', '/posts/', '/stories/', '/guild/', '/market/', '/items/', '/rpdb/', '/character-cards/']
const CHARACTER_CARD_ID_REGEX = /\/character-cards\/(\d+)(?:$|[/?#])/i
const CHARACTER_CARD_PRIVATE_ATTRS = [
  'data-jump-author',
  'data-jump-avatar',
  'data-jump-image',
  'data-jump-summary',
  'data-jump-status',
  'data-jump-visibility',
]

type CharacterCardResult = CharacterCard | 'unavailable'

const characterCardRequests = new Map<number, Promise<CharacterCardResult>>()
const characterCardVerificationTokens = new WeakMap<HTMLElement, symbol>()
let characterCardObserver: MutationObserver | null = null

function quickJumpText(key: string) {
  return i18n.global.t(`community.editor.quickJumpSheet.${key}`)
}

function isInMainDocument(node: Node) {
  return typeof document !== 'undefined'
    && Boolean(document.documentElement)
    && document.documentElement.contains(node)
}

function parsePositiveId(value: string | null | undefined) {
  if (!value || !/^[1-9]\d*$/.test(value)) return null
  const id = Number(value)
  return Number.isSafeInteger(id) ? id : null
}

function resolveCharacterCardId(card: HTMLElement) {
  const directId = parsePositiveId(card.getAttribute('data-jump-id'))
  if (directId) return directId
  const href = card.getAttribute('data-jump-href') || card.getAttribute('href') || ''
  return parsePositiveId(href.match(CHARACTER_CARD_ID_REGEX)?.[1])
}

function fetchCharacterCard(id: number) {
  const pending = characterCardRequests.get(id)
  if (pending) return pending

  const request: Promise<CharacterCardResult> = getCharacterCard(id)
    .then((characterCard) => {
      if (!characterCard || characterCard.id !== id || !isApprovedPublicCharacterCard(characterCard)) {
        return 'unavailable'
      }
      return characterCard
    })
    .catch(() => 'unavailable' as const)
    .finally(() => {
      characterCardRequests.delete(id)
    })

  characterCardRequests.set(id, request)
  return request
}

function getCharacterCardName(characterCard: CharacterCard) {
  return characterCard.display_name?.trim()
    || [characterCard.first_name, characterCard.last_name]
      .map(part => part?.trim())
      .filter(Boolean)
      .join(' ')
    || quickJumpText('unnamedCharacterCard')
}

function getCharacterCardPortrait(characterCard: CharacterCard) {
  if (!characterCard.portrait_image_url) return ''
  const version = characterCard.portrait_image_updated_at || characterCard.updated_at
  if (!version) return resolveApiUrl(characterCard.portrait_image_url)
  return getImageUrl('character-card-portrait', characterCard.id, {
    w: 360,
    q: 84,
    v: version,
  })
}

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

function removeCharacterCardPrivateAttrs(card: HTMLElement) {
  CHARACTER_CARD_PRIVATE_ATTRS.forEach(attr => card.removeAttribute(attr))
}

function prepareCharacterCardVerification(card: HTMLElement) {
  card.classList.add('jump-card', 'jump-card--character-card')
  card.removeAttribute('data-jump-unavailable')
  card.removeAttribute('data-jump-verified')
  card.removeAttribute('aria-label')
  card.removeAttribute('title')
  card.setAttribute('data-jump-pending', 'true')
  card.setAttribute('data-jump-safe-placeholder', 'true')
  card.setAttribute('data-jump-variant', 'character-card')
  card.setAttribute('data-jump-label', quickJumpText('characterCardReference'))
  card.setAttribute('data-jump-title', quickJumpText('characterCardVerifying'))
  card.setAttribute('aria-disabled', 'true')
  card.setAttribute('role', 'group')
  card.setAttribute('tabindex', '-1')
  if (card.hasAttribute('href')) card.removeAttribute('href')
  removeCharacterCardPrivateAttrs(card)
  renderCharacterCardState(card, {
    kind: quickJumpText('characterCardReference'),
    title: quickJumpText('characterCardPlaceholder'),
    subtitle: quickJumpText('characterCardVerifying'),
    summary: quickJumpText('characterCardVerificationHint'),
    icon: 'ri-shield-keyhole-line',
    actionIcon: 'ri-loader-4-line',
  })
}

function refreshCharacterCardPortrait(card: HTMLElement, imageUrl: string, title: string) {
  const portrait = card.querySelector<HTMLElement>('.jump-card__character-portrait')
  if (!portrait) return
  portrait.textContent = ''
  if (imageUrl) {
    const image = document.createElement('img')
    image.src = imageUrl
    image.alt = `${title} · ${quickJumpText('characterCardPortrait')}`
    image.loading = 'lazy'
    image.decoding = 'async'
    portrait.appendChild(image)
    return
  }
  const fallback = document.createElement('i')
  fallback.className = 'ri-user-star-line'
  fallback.setAttribute('aria-hidden', 'true')
  portrait.appendChild(fallback)
}

function refreshCharacterCard(card: HTMLElement, characterCard: CharacterCard) {
  if (!isInMainDocument(card)) return

  const title = getCharacterCardName(characterCard)
  const subtitle = characterCard.title?.trim()
    || characterCard.full_title?.trim()
    || quickJumpText('characterCardDossier')
  const summary = characterCard.summary?.trim()
    || [characterCard.race, characterCard.class].filter(Boolean).join(' · ')
    || quickJumpText('characterCardSummaryMissing')

  card.removeAttribute('data-jump-unavailable')
  card.removeAttribute('data-jump-pending')
  card.removeAttribute('data-jump-safe-placeholder')
  card.removeAttribute('aria-disabled')
  card.setAttribute('data-jump-verified', 'true')
  card.setAttribute('data-jump-id', String(characterCard.id))
  card.setAttribute('data-jump-type', 'character_card')
  card.setAttribute('data-jump-href', `/character-cards/${characterCard.id}`)
  card.setAttribute('data-jump-label', quickJumpText('characterCards'))
  card.setAttribute('data-jump-title', quickJumpText('characterCardVerified'))
  card.setAttribute('role', 'link')
  card.setAttribute('tabindex', '0')
  card.setAttribute('aria-label', `${quickJumpText('characterCardOpen')}: ${title}`)
  removeCharacterCardPrivateAttrs(card)

  const kind = card.querySelector<HTMLElement>('.jump-card__character-kind')
  if (kind) {
    kind.textContent = ''
    const icon = document.createElement('i')
    icon.className = 'ri-id-card-line'
    icon.setAttribute('aria-hidden', 'true')
    kind.append(icon, quickJumpText('characterCards'))
  }
  const titleElement = card.querySelector<HTMLElement>('.jump-card__title')
  if (titleElement) titleElement.textContent = title
  const subtitleElement = card.querySelector<HTMLElement>('.jump-card__character-subtitle')
  if (subtitleElement) subtitleElement.textContent = subtitle
  const summaryElement = card.querySelector<HTMLElement>('.jump-card__character-summary')
  if (summaryElement) summaryElement.textContent = summary
  refreshCharacterCardPortrait(card, getCharacterCardPortrait(characterCard), title)
}

function refreshUnavailableCharacterCard(card: HTMLElement, allowDetached = false) {
  if (!allowDetached && !isInMainDocument(card)) return
  card.removeAttribute('data-jump-pending')
  card.removeAttribute('data-jump-verified')
  card.removeAttribute('aria-label')
  card.setAttribute('data-jump-unavailable', 'true')
  card.setAttribute('data-jump-safe-placeholder', 'true')
  card.setAttribute('data-jump-label', quickJumpText('characterCardReference'))
  card.setAttribute('data-jump-title', quickJumpText('characterCardUnavailable'))
  card.setAttribute('aria-disabled', 'true')
  card.setAttribute('role', 'group')
  card.setAttribute('tabindex', '-1')
  removeCharacterCardPrivateAttrs(card)
  renderCharacterCardState(card, {
    kind: quickJumpText('characterCardReference'),
    title: quickJumpText('characterCardUnavailable'),
    subtitle: quickJumpText('characterCardNoAccess'),
    summary: quickJumpText('characterCardUnavailableHint'),
    icon: 'ri-link-unlink-m',
    actionIcon: 'ri-lock-line',
  })
}

function hydrateCharacterCard(card: HTMLElement, id: number) {
  if (!isInMainDocument(card)) return
  const token = Symbol(`character-card-${id}`)
  characterCardVerificationTokens.set(card, token)
  void fetchCharacterCard(id).then((characterCard) => {
    if (characterCardVerificationTokens.get(card) !== token) return
    if (characterCard === 'unavailable') {
      refreshUnavailableCharacterCard(card)
      return
    }
    refreshCharacterCard(card, characterCard)
  })
}

function secureCharacterCard(card: HTMLElement, shouldHydrate: boolean) {
  const id = resolveCharacterCardId(card)
  prepareCharacterCardVerification(card)
  if (!id) {
    refreshUnavailableCharacterCard(card, true)
    return
  }
  card.setAttribute('data-jump-id', String(id))
  card.setAttribute('data-jump-href', `/character-cards/${id}`)
  if (shouldHydrate && isInMainDocument(card) && !card.closest(EDITOR_SELECTOR)) hydrateCharacterCard(card, id)
}

function ensureCharacterCardObserver() {
  if (characterCardObserver || typeof MutationObserver === 'undefined' || !document.body) return
  characterCardObserver = new MutationObserver((records) => {
    records.forEach((record) => {
      record.addedNodes.forEach((node) => {
        if (!(node instanceof Element)) return
        const cards: HTMLElement[] = []
        if (node.matches(CHARACTER_CARD_SELECTOR)) cards.push(node as HTMLElement)
        cards.push(...node.querySelectorAll<HTMLElement>(CHARACTER_CARD_SELECTOR))
        cards.forEach((card) => {
          if (!card.closest(EDITOR_SELECTOR)) secureCharacterCard(card, true)
        })
      })
    })
  })
  characterCardObserver.observe(document.body, { childList: true, subtree: true })
}

export function buildCharacterCardJumpPlaceholder(id: number) {
  if (!Number.isSafeInteger(id) || id <= 0) throw new Error('Invalid character card id')
  const path = `/character-cards/${id}`
  return `<span class="jump-card jump-card--character-card" role="group" tabindex="-1" aria-disabled="true" data-jump-href="${path}" data-jump-type="character_card" data-jump-id="${id}" data-jump-variant="character-card" data-jump-pending="true" data-jump-safe-placeholder="true"></span>`
}

function resolveInternalHref(href: string): string | null {
  const trimmed = href.trim()
  if (!trimmed || trimmed === '#') return null

  if (trimmed.startsWith('#/')) return normalizeLegacyPath(trimmed.slice(1))
  if (trimmed.startsWith('/#/')) return normalizeLegacyPath(trimmed.slice(2))
  if (trimmed.startsWith('/')) return normalizeLegacyPath(trimmed)

  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    try {
      const url = new URL(trimmed)
      if (url.origin !== window.location.origin) return null
      if (url.hash.startsWith('#/')) return normalizeLegacyPath(url.hash.slice(1))
      return normalizeLegacyPath(`${url.pathname}${url.search}`)
    } catch {
      return null
    }
  }

  return null
}

function normalizeLegacyPath(path: string) {
  if (!INTERNAL_PREFIXES.some((prefix) => path.startsWith(prefix))) return null
  if (path.startsWith('/archives/story/')) return `/stories/${path.replace('/archives/story/', '').split('/')[0]}`
  if (path.startsWith('/community/post/')) return `/posts/${path.replace('/community/post/', '').split('/')[0]}`
  if (path.startsWith('/market/item/')) return `/items/${path.replace('/market/item/', '').split('/')[0]}`
  return path
}

function getJumpTarget(link: HTMLElement) {
  const href = link.getAttribute('data-jump-href') || link.getAttribute('href') || ''
  return resolveInternalHref(href)
}

export function sanitizeJumpLinks(container: HTMLElement | null) {
  if (!container) return
  ensureCharacterCardObserver()
  const links = container.querySelectorAll<HTMLElement>(JUMP_LINK_SELECTOR)
  links.forEach((link) => {
    if (link.getAttribute('data-jump-type') === 'character_card') {
      secureCharacterCard(link, isInMainDocument(link))
      return
    }

    const href = getJumpTarget(link)
    if (!href) return
    link.setAttribute('data-jump-href', href)
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
    const href = resolveInternalHref(anchor.getAttribute('href') || '')
    if (!href) return
    anchor.setAttribute('data-jump-href', href)
    anchor.classList.add('jump-link')
    anchor.removeAttribute('href')
  })
}

export function handleJumpLinkClick(event: MouseEvent, router: Router) {
  const target = event.target
  const element = target instanceof Element ? target : (target instanceof Node ? target.parentElement : null)
  if (!element) return false

  const link =
    (element.closest(JUMP_LINK_SELECTOR) as HTMLElement | null) ||
    (element.closest('a[href]') as HTMLElement | null)
  if (!link) return false

  const isUnverifiedCharacterCard = link.getAttribute('data-jump-type') === 'character_card'
    && link.getAttribute('data-jump-verified') !== 'true'
  if (isUnverifiedCharacterCard
    || link.getAttribute('data-jump-unavailable') === 'true'
    || link.getAttribute('data-jump-pending') === 'true') {
    event.preventDefault()
    event.stopPropagation()
    return true
  }

  const href = getJumpTarget(link)
  if (!href) return false

  event.preventDefault()
  event.stopPropagation()
  router.push(href)
  return true
}
