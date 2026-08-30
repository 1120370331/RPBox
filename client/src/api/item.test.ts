import { describe, expect, it } from 'vitest'
import { resolveApiUrl } from './item'

describe('resolveApiUrl', () => {
  it('keeps absolute and inline image URLs unchanged', () => {
    expect(resolveApiUrl('https://cdn.example.com/comment.png')).toBe('https://cdn.example.com/comment.png')
    expect(resolveApiUrl('http://cdn.example.com/comment.gif')).toBe('http://cdn.example.com/comment.gif')
    expect(resolveApiUrl('data:image/png;base64,abc')).toBe('data:image/png;base64,abc')
  })

  it('resolves uploaded comment images against the API host', () => {
    expect(resolveApiUrl('/uploads/comment-images/7/example.png'))
      .toBe('http://localhost:8080/uploads/comment-images/7/example.png')
    expect(resolveApiUrl('uploads/comment-images/7/example.png'))
      .toBe('http://localhost:8080/uploads/comment-images/7/example.png')
  })

  it('continues to resolve API image routes against the API host', () => {
    expect(resolveApiUrl('/api/v1/images/item-preview/12'))
      .toBe('http://localhost:8080/api/v1/images/item-preview/12')
  })

  it('returns an empty string for empty input', () => {
    expect(resolveApiUrl('')).toBe('')
    expect(resolveApiUrl(undefined)).toBe('')
    expect(resolveApiUrl(null)).toBe('')
  })
})
