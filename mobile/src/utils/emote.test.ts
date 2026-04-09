import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/api/emote', () => ({
  listEmotePacks: vi.fn(),
}))

vi.mock('@/api/image', () => ({
  resolveApiUrl: (url: string) => `https://api.test${url}`,
}))

describe('mobile emote utils', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders fallback emotes, mentions, and escaped text', async () => {
    const { buildEmoteToken, renderTextWithEmotes } = await import('./emote')

    expect(buildEmoteToken('pack1', 'wave')).toBe('[[emote:pack1:wave]]')

    const html = renderTextWithEmotes('Hi <there>\n[[mention:12:Foo%20Bar]] [[emote:pack1:wave]]')
    expect(html).toContain('Hi &lt;there&gt;<br>')
    expect(html).toContain('class="inline-mention"')
    expect(html).toContain('@Foo Bar')
    expect(html).toContain('https://api.test/emotes/pack1/wave.png')
  })

  it('loads emote urls from the pack api and reuses the cache', async () => {
    const emoteApi = await import('@/api/emote')
    vi.mocked(emoteApi.listEmotePacks).mockResolvedValue([
      {
        id: 'pack9',
        items: [
          { id: 'smile', url: 'https://img.test/smile.png' },
        ],
      },
    ] as any)

    const { ensureEmoteMapLoaded, renderTextWithEmotes } = await import('./emote')

    await ensureEmoteMapLoaded()
    await ensureEmoteMapLoaded()

    const html = renderTextWithEmotes('[[emote:pack9:smile]]')
    expect(emoteApi.listEmotePacks).toHaveBeenCalledTimes(1)
    expect(html).toContain('https://img.test/smile.png')
  })
})
