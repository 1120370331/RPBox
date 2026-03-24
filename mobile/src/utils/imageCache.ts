const IMAGE_CACHE_NAME = 'rpbox-mobile-image-cache-v1'
const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'

function normalizeApiOrigin(apiBase: string): string {
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    return apiBase.replace(/\/api\/v1\/?$/, '')
  }
  return ''
}

const API_ORIGIN = normalizeApiOrigin(API_BASE)

function canUseCacheApi() {
  return typeof window !== 'undefined' && 'caches' in window
}

function resolveUrl(raw: string) {
  if (!raw) return ''
  try {
    const base = API_ORIGIN || window.location.origin
    return new URL(raw, base).toString()
  } catch {
    return raw
  }
}

function isHttpUrl(raw: string) {
  if (!raw || raw.startsWith('data:')) return false
  try {
    const url = new URL(raw)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function isCacheableUrl(raw: string) {
  if (!isHttpUrl(raw)) return false
  try {
    const url = new URL(raw)
    return (
      url.pathname.includes('/api/v1/images') ||
      url.pathname.includes('/uploads/') ||
      url.pathname.includes('/emotes/')
    )
  } catch {
    return false
  }
}

function buildAuthHeaders() {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  return headers
}

function buildCacheKeyRequest(url: string) {
  return new Request(url, { method: 'GET', credentials: 'include' })
}

async function fetchImage(url: string, withAuth: boolean) {
  return fetch(url, {
    method: 'GET',
    headers: withAuth ? buildAuthHeaders() : undefined,
    credentials: 'include',
    mode: 'cors',
  })
}

async function responseToObjectUrl(response: Response) {
  const blob = await response.blob()
  return URL.createObjectURL(blob)
}

export async function getCachedImageObjectUrl(rawUrl: string) {
  const source = resolveUrl(rawUrl)
  if (!canUseCacheApi() || !isCacheableUrl(source)) return ''
  try {
    const cache = await caches.open(IMAGE_CACHE_NAME)
    const cached = await cache.match(buildCacheKeyRequest(source))
    if (cached) {
      return responseToObjectUrl(cached)
    }
    return ''
  } catch {
    return ''
  }
}

export async function warmImageCache(rawUrl: string) {
  const source = resolveUrl(rawUrl)
  if (!canUseCacheApi() || !isCacheableUrl(source)) return false
  try {
    const cache = await caches.open(IMAGE_CACHE_NAME)
    const cacheKey = buildCacheKeyRequest(source)
    const cached = await cache.match(cacheKey)
    if (cached) return true

    const network = await fetchImage(source, false)
    if (!network.ok) return false
    await cache.put(cacheKey, network.clone())
    return true
  } catch {
    return false
  }
}

export async function fetchImageObjectUrlWithAuth(rawUrl: string) {
  const source = resolveUrl(rawUrl)
  if (!isHttpUrl(source)) return ''
  try {
    if (!canUseCacheApi() || !isCacheableUrl(source)) {
      const network = await fetchImage(source, true)
      if (!network.ok) return ''
      return responseToObjectUrl(network)
    }

    const cache = await caches.open(IMAGE_CACHE_NAME)
    const cacheKey = buildCacheKeyRequest(source)
    const cached = await cache.match(cacheKey)
    if (cached) {
      return responseToObjectUrl(cached)
    }

    const network = await fetchImage(source, true)
    if (!network.ok) return ''
    await cache.put(cacheKey, network.clone())
    return responseToObjectUrl(network)
  } catch {
    return ''
  }
}

export async function clearImageCache() {
  if (!canUseCacheApi()) return false
  try {
    return await caches.delete(IMAGE_CACHE_NAME)
  } catch {
    return false
  }
}
