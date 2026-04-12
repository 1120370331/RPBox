const APP_URL_SCHEME = 'app.rpbox.mobile'
const APP_PACKAGE_NAME = 'app.rpbox.mobile'
const DEFAULT_PUBLIC_SITE_ORIGIN = 'https://totalrpbox.com'

const SUPPORTED_PATH_PATTERNS = [
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

export function getPublicSiteOrigin() {
  const raw = String(import.meta.env.VITE_PUBLIC_SITE_ORIGIN || DEFAULT_PUBLIC_SITE_ORIGIN).trim()
  return trimTrailingSlash(raw || DEFAULT_PUBLIC_SITE_ORIGIN)
}

export function getDownloadPageUrl() {
  return `${getPublicSiteOrigin()}/download.html`
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

export function normalizeInAppPath(input: string) {
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

  const sanitized = value.replace(/\/{2,}/g, '/').replace(/\/+$/, '')
  if (!sanitized) return null

  return SUPPORTED_PATH_PATTERNS.some((pattern) => pattern.test(sanitized)) ? sanitized : null
}

export function buildCustomSchemeUrl(path: string) {
  const normalized = normalizeInAppPath(path)
  if (!normalized) {
    throw new Error(`Unsupported app path: ${path}`)
  }

  return `${APP_URL_SCHEME}://open?path=${encodeURIComponent(normalized)}`
}

export function buildAndroidIntentUrl(path: string, fallbackUrl = getDownloadPageUrl()) {
  const normalized = normalizeInAppPath(path)
  if (!normalized) {
    throw new Error(`Unsupported app path: ${path}`)
  }

  const encodedFallbackUrl = encodeURIComponent(fallbackUrl)
  return `intent://open?path=${encodeURIComponent(normalized)}#Intent;scheme=${APP_URL_SCHEME};package=${APP_PACKAGE_NAME};S.browser_fallback_url=${encodedFallbackUrl};end`
}

export function buildOpenAppRedirectUrl(path: string) {
  const normalized = normalizeInAppPath(path)
  if (!normalized) {
    throw new Error(`Unsupported app path: ${path}`)
  }

  const url = new URL('/open-app.html', `${getPublicSiteOrigin()}/`)
  url.searchParams.set('path', normalized)
  return url.toString()
}

export function buildPublicSitePathUrl(path: string) {
  const normalized = normalizeInAppPath(path)
  if (!normalized) {
    throw new Error(`Unsupported app path: ${path}`)
  }

  return new URL(normalized, `${getPublicSiteOrigin()}/`).toString()
}

export function resolveInAppPathFromUrl(rawUrl: string) {
  if (!rawUrl) return null

  if (rawUrl.startsWith('intent://')) {
    const match = rawUrl.match(/^intent:\/\/([^#]+)#Intent;.*(?:^|;)scheme=([^;]+);/i)
    if (!match) return null
    const [, route, scheme] = match
    return resolveInAppPathFromUrl(`${scheme}://${route}`)
  }

  try {
    const url = new URL(rawUrl)
    const queryPath = url.searchParams.get('path')
    if (queryPath) {
      return normalizeInAppPath(queryPath)
    }

    if ((url.protocol === 'https:' || url.protocol === 'http:') && isPublicSiteHost(url.hostname)) {
      return normalizeInAppPath(url.pathname)
    }

    const directPath = `${url.hostname ? `/${url.hostname}` : ''}${url.pathname || ''}`
    return normalizeInAppPath(directPath)
  } catch {
    return null
  }
}

export function buildPostShareRedirectUrl(postId: number) {
  return buildPublicSitePathUrl(`/posts/${postId}`)
}

export { APP_PACKAGE_NAME, APP_URL_SCHEME }
