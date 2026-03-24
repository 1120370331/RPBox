const IMAGE_CACHE_NAME = 'rpbox-mobile-image-cache-v1'

function canUseCacheApi() {
  return typeof window !== 'undefined' && 'caches' in window
}

function isCacheableUrl(raw: string) {
  if (!raw || raw.startsWith('data:')) return false
  try {
    const url = new URL(raw, window.location.href)
    if (!(url.protocol === 'http:' || url.protocol === 'https:')) return false
    if (url.origin !== window.location.origin) return false
    return url.pathname.includes('/api/v1/images') || url.pathname.includes('/uploads/')
  } catch {
    return false
  }
}

function buildImageRequest(url: string) {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  return new Request(url, { method: 'GET', headers, credentials: 'include', mode: 'cors' })
}

async function responseToObjectUrl(response: Response) {
  const blob = await response.blob()
  return URL.createObjectURL(blob)
}

export async function getCachedImageObjectUrl(rawUrl: string) {
  if (!canUseCacheApi() || !isCacheableUrl(rawUrl)) return ''
  try {
    const cache = await caches.open(IMAGE_CACHE_NAME)
    const request = buildImageRequest(rawUrl)

    const cached = await cache.match(request)
    if (cached) {
      return responseToObjectUrl(cached)
    }
    return ''
  } catch {
    return ''
  }
}

export async function warmImageCache(rawUrl: string) {
  if (!canUseCacheApi() || !isCacheableUrl(rawUrl)) return false
  try {
    const cache = await caches.open(IMAGE_CACHE_NAME)
    const request = buildImageRequest(rawUrl)
    const cached = await cache.match(request)
    if (cached) return true

    const network = await fetch(request)
    if (!network.ok) return false
    await cache.put(request, network.clone())
    return true
  } catch {
    return false
  }
}

export async function fetchImageObjectUrlWithAuth(rawUrl: string) {
  if (!canUseCacheApi() || !isCacheableUrl(rawUrl)) return ''
  try {
    const cache = await caches.open(IMAGE_CACHE_NAME)
    const request = buildImageRequest(rawUrl)
    const cached = await cache.match(request)
    if (cached) {
      return responseToObjectUrl(cached)
    }

    const network = await fetch(request)
    if (!network.ok) return ''
    await cache.put(request, network.clone())
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
