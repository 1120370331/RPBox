import { describe, expect, it } from 'vitest'
import { sanitizeRichHtml } from './sanitizeHtml'

describe('sanitizeRichHtml', () => {
  it('removes executable HTML, URL payloads, and active embeds', () => {
    const clean = sanitizeRichHtml(`
      <img src="/uploads/safe.png" onerror="alert(1)">
      <svg onload="alert(2)"><circle /></svg>
      <a href="javascript:alert(3)">bad link</a>
      <iframe srcdoc="<script>alert(4)</script>"></iframe>
      <form><input autofocus onfocus="alert(5)"><button>submit</button></form>
    `)

    expect(clean).toContain('<img src="/uploads/safe.png">')
    expect(clean).toContain('bad link')
    expect(clean).not.toMatch(/onerror|onload|javascript:|srcdoc|iframe|script|svg|form|input|button/i)
  })

  it('keeps only safe text alignment styles and blocks CSS URL/expression payloads', () => {
    const clean = sanitizeRichHtml(`
      <p style="text-align: center; background-image: url(javascript:alert(1))">center</p>
      <span style="width: expression(alert(2)); color: red">unsafe</span>
    `)

    expect(clean).toContain('style="text-align: center"')
    expect(clean).not.toMatch(/url\s*\(|expression\s*\(|color\s*:|width\s*:/i)
  })

  it('preserves Tiptap formatting, safe images and hardened external links', () => {
    const clean = sanitizeRichHtml(`
      <h2>Chapter</h2>
      <p style="text-align: right"><strong>Bold</strong> <em>note</em></p>
      <ul><li>First</li><li>Second</li></ul>
      <img src="https://cdn.rpbox.app/image.png" alt="preview" loading="lazy">
      <a href="https://example.com/reference" target="_blank">Reference</a>
      <img src="data:image/png;base64,AAAA" alt="inline">
    `)

    expect(clean).toContain('<h2>Chapter</h2>')
    expect(clean).toContain('<ul><li>First</li><li>Second</li></ul>')
    expect(clean).toContain('src="https://cdn.rpbox.app/image.png"')
    expect(clean).toContain('href="https://example.com/reference"')
    expect(clean).toContain('rel="noopener noreferrer"')
    expect(clean).not.toContain('data:image')
  })

  it('preserves RPBox jump cards, mentions and emotes while rejecting unsafe jump URLs', () => {
    const clean = sanitizeRichHtml(`
      <a class="jump-card jump-card--rpdb-item" role="link" tabindex="0"
        data-jump-href="/rpdb/42" data-jump-type="rpdb_work" data-jump-id="42"
        data-jump-title="Moonwell" data-jump-variant="rpdb-item">
        <span class="jump-card__title">Moonwell</span>
      </a>
      <span class="comment-mention" data-mention-id="7" contenteditable="false">@Ari</span>
      <img class="comment-emote" src="/emotes/pack/wave.png" data-emote="pack:wave">
      <span class="jump-link" data-jump-href="javascript:alert(1)">unsafe</span>
    `)

    expect(clean).toContain('class="jump-card jump-card--rpdb-item"')
    expect(clean).toContain('data-jump-href="/rpdb/42"')
    expect(clean).toContain('data-jump-type="rpdb_work"')
    expect(clean).toContain('data-mention-id="7"')
    expect(clean).toContain('class="comment-emote"')
    expect(clean).not.toContain('data-jump-href="javascript:')
  })
})
