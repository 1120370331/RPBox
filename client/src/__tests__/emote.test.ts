import { describe, it, expect, vi } from 'vitest'
import { buildEmoteToken, renderEmoteContent } from '../utils/emote'

vi.mock('@/api/emote', () => ({
  resolveEmoteUrl: (url: string) => `https://cdn.test${url}`,
}))

describe('emote utils', () => {
  it('builds emote token', () => {
    expect(buildEmoteToken('pack', 'item')).toBe('[[emote:pack:item]]')
  })

  it('renders emote tokens with map items', () => {
    const map = new Map<string, { id: string; name: string; url: string }>()
    map.set('pack1:item1', { id: 'item1', name: 'Smile', url: 'https://img.test/smile.png' })

    const html = renderEmoteContent('Hello [[emote:pack1:item1]]', map as any)
    expect(html).toContain('comment-emote')
    expect(html).toContain('data-emote="pack1:item1"')
    expect(html).toContain('https://img.test/smile.png')
  })

  it('renders mention tokens', () => {
    const html = renderEmoteContent('Hi [[mention:12:Foo%20Bar]]', new Map())
    expect(html).toContain('@Foo Bar')
    expect(html).toContain('data-mention-id="12"')
  })

  it('falls back to resolved emote url when missing', () => {
    const html = renderEmoteContent('[[emote:pack2:item2]]', new Map())
    expect(html).toContain('https://cdn.test/emotes/pack2/item2.png')
  })
})
