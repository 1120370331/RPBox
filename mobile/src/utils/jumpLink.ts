import type { Router } from 'vue-router'

const JUMP_LINK_SELECTOR = '.jump-link, a.jump-card, [data-jump-href], [data-jump-type]'
const INTERNAL_PREFIXES = ['/archives/story/', '/community/post/', '/posts/', '/stories/', '/guild/', '/market/', '/items/']

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
  const links = container.querySelectorAll<HTMLElement>(JUMP_LINK_SELECTOR)
  links.forEach((link) => {
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

  const href = getJumpTarget(link)
  if (!href) return false

  event.preventDefault()
  event.stopPropagation()
  router.push(href)
  return true
}
