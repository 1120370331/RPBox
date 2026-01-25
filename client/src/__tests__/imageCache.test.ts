import { describe, it, expect, vi, afterEach } from 'vitest'

describe('imageCache', () => {
  afterEach(() => {
    vi.useRealTimers()
    localStorage.clear()
  })

  it('persists cache version in localStorage', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-01-01T00:00:00Z'))
    localStorage.clear()
    vi.resetModules()

    const { getImageCacheVersion } = await import('../utils/imageCache')
    const first = getImageCacheVersion()
    const second = getImageCacheVersion()

    expect(first).toBe(second)
    expect(localStorage.getItem('rpbox_image_cache_v1')).toBe(first)
  })

  it('bumps cache version', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-01-01T00:00:00Z'))
    localStorage.clear()
    vi.resetModules()

    const { getImageCacheVersion, bumpImageCacheVersion } = await import('../utils/imageCache')
    const first = getImageCacheVersion()

    vi.setSystemTime(new Date('2024-01-01T00:00:02Z'))
    const next = bumpImageCacheVersion()

    expect(next).not.toBe(first)
    expect(localStorage.getItem('rpbox_image_cache_v1')).toBe(next)
  })
})
