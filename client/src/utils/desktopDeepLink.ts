import {
  type NavigationFailure,
  NavigationFailureType,
  type Router,
  isNavigationFailure,
} from 'vue-router'

const DEFAULT_PUBLIC_SITE_ORIGIN = 'https://totalrpbox.com'

const SUPPORTED_PUBLIC_PATH_PATTERNS = [
  /^\/posts\/\d+$/,
  /^\/items\/\d+$/,
  /^\/stories\/\d+$/,
  /^\/profiles\/\d+$/,
  /^\/guild\/\d+$/,
  /^\/guild\/\d+\/posts$/,
  /^\/guild\/\d+\/stories$/,
]

function trimTrailingSlash(input: string) {
  return input.replace(/\/+$/, '')
}

function getPublicSiteOrigin() {
  const raw = String(import.meta.env.VITE_PUBLIC_SITE_ORIGIN || DEFAULT_PUBLIC_SITE_ORIGIN).trim()
  return trimTrailingSlash(raw || DEFAULT_PUBLIC_SITE_ORIGIN)
}

function isPublicSiteHost(hostname: string) {
  if (!hostname) return false

  try {
    const canonicalHost = new URL(getPublicSiteOrigin()).hostname
    return hostname === canonicalHost || hostname === 'totalrpbox.com' || hostname === 'www.totalrpbox.com'
  } catch {
    return false
  }
}

export function normalizeSharedPath(input: string) {
  if (!input) return null

  let value = String(input).trim()
  if (!value) return null

  try {
    value = decodeURIComponent(value)
  } catch {
    return null
  }

  if (!value.startsWith('/')) {
    value = `/${value.replace(/^\/+/, '')}`
  }

  const normalized = value.replace(/\/{2,}/g, '/').replace(/\/+$/, '')
  if (!normalized) return null

  return SUPPORTED_PUBLIC_PATH_PATTERNS.some((pattern) => pattern.test(normalized)) ? normalized : null
}

export function resolveSharedPathFromUrl(rawUrl: string) {
  if (!rawUrl) return null

  try {
    const url = new URL(rawUrl)
    const queryPath = url.searchParams.get('path')
    if (queryPath) {
      return normalizeSharedPath(queryPath)
    }

    if ((url.protocol === 'https:' || url.protocol === 'http:') && isPublicSiteHost(url.hostname)) {
      return normalizeSharedPath(url.pathname)
    }

    const directPath = `${url.hostname ? `/${url.hostname}` : ''}${url.pathname || ''}`
    return normalizeSharedPath(directPath)
  } catch {
    return null
  }
}

export function mapSharedPathToDesktopRoute(path: string) {
  const normalized = normalizeSharedPath(path)
  if (!normalized) return null

  const postMatch = normalized.match(/^\/posts\/(\d+)$/)
  if (postMatch) return `/community/post/${postMatch[1]}`

  const itemMatch = normalized.match(/^\/items\/(\d+)$/)
  if (itemMatch) return `/market/${itemMatch[1]}`

  const storyMatch = normalized.match(/^\/stories\/(\d+)$/)
  if (storyMatch) return `/archives/story/${storyMatch[1]}`

  const profileMatch = normalized.match(/^\/profiles\/(\d+)$/)
  if (profileMatch) return `/user/${profileMatch[1]}`

  return normalized
}

export function resolveDesktopRouteFromUrl(rawUrl: string) {
  const sharedPath = resolveSharedPathFromUrl(rawUrl)
  if (!sharedPath) return null
  return mapSharedPathToDesktopRoute(sharedPath)
}

function isIgnoredNavigationFailure(error: unknown) {
  return isNavigationFailure(error as NavigationFailure, NavigationFailureType.duplicated)
}

export async function handleDesktopDeepLinkUrls(urls: string[], router: Router) {
  for (const rawUrl of urls) {
    const targetRoute = resolveDesktopRouteFromUrl(rawUrl)
    if (!targetRoute) continue

    if (router.currentRoute.value.fullPath === targetRoute) {
      return targetRoute
    }

    try {
      await router.push(targetRoute)
    } catch (error) {
      if (!isIgnoredNavigationFailure(error)) {
        throw error
      }
    }

    return targetRoute
  }

  return null
}
