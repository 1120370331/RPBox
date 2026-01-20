const IMAGE_CACHE_KEY = 'rpbox_image_cache_v1'

let cacheVersion = ''

function loadCacheVersion(): string {
  if (cacheVersion) return cacheVersion
  if (typeof window !== 'undefined') {
    try {
      const stored = localStorage.getItem(IMAGE_CACHE_KEY)
      if (stored) {
        cacheVersion = stored
        return cacheVersion
      }
    } catch {
    }
  }
  const generated = Date.now().toString(36)
  cacheVersion = generated
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem(IMAGE_CACHE_KEY, generated)
    } catch {
    }
  }
  return cacheVersion
}

export function getImageCacheVersion(): string {
  return loadCacheVersion()
}

export function bumpImageCacheVersion(): string {
  const next = Date.now().toString(36)
  cacheVersion = next
  try {
    localStorage.setItem(IMAGE_CACHE_KEY, next)
  } catch {
  }
  return next
}
