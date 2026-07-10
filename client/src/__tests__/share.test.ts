import { afterEach, describe, expect, it, vi } from 'vitest'

import { buildPostShareText, buildPublicPostUrl, shareRouteLink } from '@/utils/share'

describe('desktop route sharing', () => {
  afterEach(() => {
    Object.defineProperty(navigator, 'share', {
      configurable: true,
      value: undefined,
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: undefined,
    })
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: undefined,
    })
    document.body.innerHTML = ''
    vi.restoreAllMocks()
  })

  it('builds the public post URL', () => {
    expect(buildPublicPostUrl(42)).toBe('https://totalrpbox.com/posts/42')
  })

  it('builds a compact plain-text post summary', () => {
    expect(buildPostShareText('<p>Hello <strong>Azeroth</strong></p>', 12)).toBe('Hello Azerot…')
  })

  it('uses Web Share when available', async () => {
    const share = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'share', {
      configurable: true,
      value: share,
    })

    await expect(shareRouteLink({
      path: '/posts/42',
      title: 'Test post',
      text: 'Read this',
    })).resolves.toEqual({
      method: 'shared',
      url: 'https://totalrpbox.com/posts/42',
    })
    expect(share).toHaveBeenCalledWith({
      title: 'Test post',
      text: 'Read this',
      url: 'https://totalrpbox.com/posts/42',
    })
  })

  it('copies the public URL when Web Share is unavailable', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })

    await expect(shareRouteLink({ path: '/posts/42' })).resolves.toEqual({
      method: 'copied',
      url: 'https://totalrpbox.com/posts/42',
    })
    expect(writeText).toHaveBeenCalledWith('https://totalrpbox.com/posts/42')
  })

  it('falls back to a temporary textarea when Clipboard API is unavailable', async () => {
    const execCommand = vi.fn().mockReturnValue(true)
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: execCommand,
    })

    await expect(shareRouteLink({ path: '/posts/42' })).resolves.toEqual({
      method: 'copied',
      url: 'https://totalrpbox.com/posts/42',
    })
    expect(execCommand).toHaveBeenCalledWith('copy')
    expect(document.querySelector('textarea')).toBeNull()
  })

  it('propagates a rejected Web Share request without copying', async () => {
    const shareError = new Error('share cancelled')
    const share = vi.fn().mockRejectedValue(shareError)
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'share', {
      configurable: true,
      value: share,
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })

    await expect(shareRouteLink({ path: '/posts/42' })).rejects.toBe(shareError)
    expect(writeText).not.toHaveBeenCalled()
  })
})
