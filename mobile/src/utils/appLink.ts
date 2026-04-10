const APP_URL_SCHEME = 'app.rpbox.mobile'
const DEFAULT_PUBLIC_SITE_ORIGIN = 'https://www.totalrpbox.com'

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

export function buildOpenAppRedirectUrl(path: string) {
  const normalized = normalizeInAppPath(path)
  if (!normalized) {
    throw new Error(`Unsupported app path: ${path}`)
  }

  const url = new URL('/open-app.html', `${getPublicSiteOrigin()}/`)
  url.searchParams.set('path', normalized)
  return url.toString()
}

export function resolveInAppPathFromUrl(rawUrl: string) {
  if (!rawUrl) return null

  try {
    const url = new URL(rawUrl)
    const queryPath = url.searchParams.get('path')
    if (queryPath) {
      return normalizeInAppPath(queryPath)
    }

    const directPath = `${url.hostname ? `/${url.hostname}` : ''}${url.pathname || ''}`
    return normalizeInAppPath(directPath)
  } catch {
    return null
  }
}

export function buildPostShareRedirectUrl(postId: number) {
  return buildOpenAppRedirectUrl(`/posts/${postId}`)
}

export { APP_URL_SCHEME }
