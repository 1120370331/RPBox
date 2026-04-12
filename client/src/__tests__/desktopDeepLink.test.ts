import { describe, expect, it, vi } from 'vitest'
import {
  handleDesktopDeepLinkUrls,
  mapSharedPathToDesktopRoute,
  normalizeSharedPath,
  resolveDesktopRouteFromUrl,
  resolveSharedPathFromUrl,
} from '../utils/desktopDeepLink'

describe('desktop deep link utils', () => {
  it('normalizes supported shared paths', () => {
    expect(normalizeSharedPath('/posts/12')).toBe('/posts/12')
    expect(normalizeSharedPath('items/7')).toBe('/items/7')
    expect(normalizeSharedPath('/guild/2/posts/')).toBe('/guild/2/posts')
    expect(normalizeSharedPath('/posts/abc')).toBeNull()
  })

  it('resolves public site and custom scheme urls to shared paths', () => {
    expect(resolveSharedPathFromUrl('rpbox://open?path=%2Fposts%2F12')).toBe('/posts/12')
    expect(resolveSharedPathFromUrl('https://totalrpbox.com/items/8')).toBe('/items/8')
    expect(resolveSharedPathFromUrl('https://www.totalrpbox.com/stories/5')).toBe('/stories/5')
    expect(resolveSharedPathFromUrl('https://example.com/items/8')).toBeNull()
  })

  it('maps shared paths to desktop routes', () => {
    expect(mapSharedPathToDesktopRoute('/posts/12')).toBe('/community/post/12')
    expect(mapSharedPathToDesktopRoute('/items/8')).toBe('/market/8')
    expect(mapSharedPathToDesktopRoute('/stories/5')).toBe('/archives/story/5')
    expect(mapSharedPathToDesktopRoute('/profiles/9')).toBe('/user/9')
    expect(mapSharedPathToDesktopRoute('/guild/4/stories')).toBe('/guild/4/stories')
  })

  it('resolves desktop routes directly from external urls', () => {
    expect(resolveDesktopRouteFromUrl('rpbox://open?path=%2Fposts%2F12')).toBe('/community/post/12')
    expect(resolveDesktopRouteFromUrl('https://totalrpbox.com/items/8')).toBe('/market/8')
    expect(resolveDesktopRouteFromUrl('https://totalrpbox.com/profiles/9')).toBe('/user/9')
  })

  it('navigates the first supported deep link url', async () => {
    const router = {
      currentRoute: { value: { fullPath: '/' } },
      push: vi.fn().mockResolvedValue(undefined),
    }

    const result = await handleDesktopDeepLinkUrls(
      ['https://example.com/nope', 'rpbox://open?path=%2Fitems%2F17'],
      router as any
    )

    expect(result).toBe('/market/17')
    expect(router.push).toHaveBeenCalledWith('/market/17')
  })
})
