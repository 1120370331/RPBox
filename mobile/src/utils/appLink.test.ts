import { describe, expect, it } from 'vitest'
import {
  APP_URL_SCHEME,
  buildCustomSchemeUrl,
  buildOpenAppRedirectUrl,
  buildPublicSitePathUrl,
  normalizeInAppPath,
  resolveInAppPathFromUrl,
} from './appLink'

describe('appLink helpers', () => {
  it('normalizes supported app paths', () => {
    expect(normalizeInAppPath('/posts/12')).toBe('/posts/12')
    expect(normalizeInAppPath('items/7')).toBe('/items/7')
    expect(normalizeInAppPath('/guild/2/posts/')).toBe('/guild/2/posts')
  })

  it('rejects unsupported paths', () => {
    expect(normalizeInAppPath('/admin/users')).toBeNull()
    expect(normalizeInAppPath('/posts/abc')).toBeNull()
  })

  it('builds custom scheme url with query path', () => {
    expect(buildCustomSchemeUrl('/posts/12')).toBe(`${APP_URL_SCHEME}://open?path=%2Fposts%2F12`)
  })

  it('builds redirect page url', () => {
    expect(buildOpenAppRedirectUrl('/posts/12')).toBe('https://totalrpbox.com/open-app.html?path=%2Fposts%2F12')
  })

  it('builds direct public site route url', () => {
    expect(buildPublicSitePathUrl('/posts/12')).toBe('https://totalrpbox.com/posts/12')
  })

  it('resolves path from custom scheme, redirect and direct site urls', () => {
    expect(resolveInAppPathFromUrl(`${APP_URL_SCHEME}://open?path=%2Fposts%2F12`)).toBe('/posts/12')
    expect(resolveInAppPathFromUrl('https://totalrpbox.com/open-app.html?path=%2Fitems%2F8')).toBe('/items/8')
    expect(resolveInAppPathFromUrl('https://totalrpbox.com/items/8')).toBe('/items/8')
  })
})
