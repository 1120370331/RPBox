import { describe, expect, it } from 'vitest'
import { htmlToTRP3Markup, normalizeTRP3About, normalizeTRP3Icon, renderTRP3Markup } from './trp3Markup'

describe('TRP3 native formatting', () => {
  it('renders original header alignment, colors, icons and links', () => {
    const html = renderTRP3Markup('{h1:c}米拉{/h1}\n{p:r}{col:11aa33}文字{/col}{icon:INV_Misc_Book_09:25}{/p}{link*https://example.com*旅途}')
    const element = document.createElement('div'); element.innerHTML = html
    expect(element.querySelector('h1')?.style.textAlign).toBe('center')
    expect(element.querySelector('p')?.style.textAlign).toBe('right')
    expect(element.querySelector('span')?.getAttribute('style')).toContain('#11aa33')
    expect(element.querySelector('img')?.getAttribute('src')).toContain('/icons/inv_misc_book_09')
    expect(element.querySelector('a')?.textContent).toBe('旅途')
    expect(normalizeTRP3Icon('Interface\\Icons\\INV_Misc_Book_09.blp')).toBe('inv_misc_book_09')
  })
  it('keeps hostile HTML and script URLs inert without losing source text', () => {
    const html = renderTRP3Markup('<script>alert(1)</script>{link*javascript:alert(1)*点击}{img:javascript:alert(1):20:20}{col:red;position:fixed}x{/col}')
    const element = document.createElement('div'); element.innerHTML = html
    expect(element.querySelector('script,a,img')).toBeNull()
    expect(element.innerHTML).not.toContain('style="position')
    expect(element.textContent).toContain('<script>alert(1)</script>')
  })
  it('retains inactive templates, background IDs, music and unknown upstream fields', () => {
    const original = { TE: 2, BK: 7, MU: 123, T1: { TX: '{h1}保留{/h1}' }, T2: [{ TX: '段落', IC: 'inv_misc_book_09', BK: 19 }], T3: { PH: { TX: '外貌' }, PS: { TX: '性格' }, HI: { TX: '历史' } }, custom: { nested: true } }
    const result = normalizeTRP3About(original)
    result.T2[0]!.TX = '修改'
    expect(original.T2[0]!.TX).toBe('段落')
    expect(result.T1.TX).toBe('{h1}保留{/h1}')
    expect(result.T3.HI.TX).toBe('历史')
    expect(result).toMatchObject({ BK: 7, MU: 123, custom: { nested: true } })
    expect(result.T2[0]!.BK).toBe(19)
  })
  it('converts existing native HTML without interpreting arbitrary markup as executable content', () => {
    const raw = htmlToTRP3Markup('<h2 style="text-align:center">旧档案</h2><p>第一行<br>第二行</p>')
    expect(raw).toContain('{h2:c}旧档案{/h2}')
    expect(raw).toContain('第一行\n第二行')
  })
})
