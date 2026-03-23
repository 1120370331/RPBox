import { describe, expect, it } from 'vitest'
import { normalizeApiOrigin, resolveApiUrl } from './image'

describe('resolveApiUrl', () => {
  it('returns absolute URLs as-is', () => {
    expect(resolveApiUrl('https://example.com/a.png')).toBe('https://example.com/a.png')
    expect(resolveApiUrl('http://example.com/a.png')).toBe('http://example.com/a.png')
    expect(resolveApiUrl('data:image/png;base64,xxx')).toBe('data:image/png;base64,xxx')
  })

  it('keeps relative API paths unchanged when API_BASE is relative', () => {
    expect(resolveApiUrl('/api/v1/images/item-preview/1')).toBe('/api/v1/images/item-preview/1')
  })

  it('returns empty string for empty input', () => {
    expect(resolveApiUrl('')).toBe('')
    expect(resolveApiUrl(undefined)).toBe('')
    expect(resolveApiUrl(null)).toBe('')
  })

  it('normalizes API origin from absolute VITE_API_BASE', () => {
    expect(normalizeApiOrigin('https://example.com/api/v1')).toBe('https://example.com')
    expect(normalizeApiOrigin('https://example.com/api/v1/')).toBe('https://example.com')
    expect(normalizeApiOrigin('/api/v1')).toBe('')
  })
})
