import { sanitizeRichHtml } from './sanitizeHtml'
import type { TRP3About, TRP3AboutSection } from '@/api/characterCard'

const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.DEV ? '/api/v1' : 'http://localhost:8080/api/v1')
const escapeHTML = (value: string) => value.replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]!))

export function normalizeTRP3Icon(value: string) {
  return value.trim().replace(/\\/g, '/').replace(/^interface\/icons\//i, '').replace(/\.(blp|tga|png|jpe?g)$/i, '').toLowerCase()
}

export function trp3IconURL(value: string) {
  const name = normalizeTRP3Icon(value)
  return /^[a-z0-9_-]+$/.test(name) ? `${API_BASE}/icons/${name}` : ''
}

function safeURL(value: string) {
  try {
    const url = new URL(value)
    return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password ? url.href : ''
  } catch { return '' }
}

/** Browser rendering of TRP3's documented tags; raw input is always escaped. */
export function renderTRP3Markup(value: string) {
  const aliases: Record<string, string> = { r: 'ff0000', g: '00ff00', b: '0000ff', w: 'ffffff', y: 'ffff00', o: 'ff8000', p: 'ff00ff' }
  const tokens = /\{[^{}]*\}|\|c[0-9a-fA-F]{8}|\|r|\|n|\|T[^|]+\|t/g
  let output = '', position = 0
  for (const match of value.matchAll(tokens)) {
    output += escapeHTML(value.slice(position, match.index)).replace(/\r?\n/g, '<br>')
    position = match.index! + match[0].length
    const raw = match[0]
    const tag = raw.startsWith('{') ? raw.slice(1, -1) : raw
    const container = tag.match(/^(h[123]|p)(?::([lcr]))?$/)
    const closing = tag.match(/^\/(h[123]|p)$/)
    const color = tag.match(/^col:([0-9a-fA-F]{6}|[rgbwyop])$/)
    const icon = tag.match(/^icon:([^:]+):(\d+)$/)
    const texture = tag.match(/^\|T([^:|]+)(?::(\d+))?(?::\d+)*\|t$/)
    const picture = tag.match(/^img:(.+):(\d+):(\d+)(?::([lcr]))?$/)
    const link = tag.match(/^link\*([^*]+)\*(.*)$/s)
    if (container) {
      const align = ({ l: 'left', c: 'center', r: 'right' } as Record<string, string>)[container[2] || 'l']
      output += `<${container[1]} style="text-align:${align}">`
    } else if (closing) output += `</${closing[1]}>`
    else if (color) output += `<span style="color:#${aliases[color[1]] || color[1]}">`
    else if (/^\|c[0-9a-fA-F]{8}$/.test(tag)) output += `<span style="color:#${tag.slice(4)}">`
    else if (tag === '/col' || tag === '|r') output += '</span>'
    else if (tag === '|n') output += '<br>'
    else if (icon || texture) {
      const source = (icon || texture)!
      const url = trp3IconURL(source[1]!)
      const size = Math.max(1, Math.min(512, Number(source[2]) || 25))
      output += url ? `<img class="trp3-inline-icon" src="${escapeHTML(url)}" alt="${escapeHTML(source[1]!)}" width="${size}" height="${size}">` : escapeHTML(raw)
    } else if (picture) {
      const url = safeURL(picture[1]!) || (/^interface[\\/]icons[\\/]/i.test(picture[1]!) ? trp3IconURL(picture[1]!) : '')
      const align = ({ l: 'left', c: 'center', r: 'right' } as Record<string, string>)[picture[4] || 'c']
      output += url ? `<div style="text-align:${align}"><img src="${escapeHTML(url)}" alt="" width="${Math.min(512, Number(picture[2]))}" height="${Math.min(512, Number(picture[3]))}"></div>` : escapeHTML(raw)
    } else if (link) {
      const url = safeURL(link[1]!)
      output += url ? `<a href="${escapeHTML(url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(link[2]!)}</a>` : escapeHTML(raw)
    } else output += escapeHTML(raw)
  }
  output += escapeHTML(value.slice(position)).replace(/\r?\n/g, '<br>')
  // Only fixed tags/attributes are emitted above. Text and URLs are escaped,
  // protocols are allowlisted, and style values come from bounded enums/hex.
  // Do not pass through source HTML or use raw tag/attribute interpolation here.
  return output
}

/** Convert a native RPBox HTML biography only when no TRP3 source is available. */
export function htmlToTRP3Markup(html: string): string {
  const doc = new DOMParser().parseFromString(sanitizeRichHtml(html), 'text/html')
  function walk(node: Node): string {
    if (node.nodeType === Node.TEXT_NODE) return node.textContent || ''
    if (!(node instanceof Element)) return ''
    const content = Array.from(node.childNodes).map(walk).join('')
    const tag = node.tagName.toLowerCase()
    if (tag === 'br') return '\n'
    if (/^(h[123]|p)$/.test(tag)) {
      const align = (node as HTMLElement).style.textAlign
      return `{${tag}${align === 'center' ? ':c' : align === 'right' ? ':r' : ''}}${content}{/${tag}}\n`
    }
    if (tag === 'a' && safeURL(node.getAttribute('href') || '')) return `{link*${node.getAttribute('href')}*${content}}`
    // WoW cannot load remote image URLs as textures; retain a usable link.
    if (tag === 'img' && safeURL(node.getAttribute('src') || '')) return `{link*${node.getAttribute('src')}*${node.getAttribute('alt') || '图片'}}`
    return content
  }
  return Array.from(doc.body.childNodes).map(walk).join('').trim()
}

export function normalizeTRP3About(raw?: unknown, fallback = ''): TRP3About {
  const source = raw && typeof raw === 'object' && !Array.isArray(raw) ? JSON.parse(JSON.stringify(raw)) as Record<string, unknown> : {}
  function section(value: unknown, icon?: string): TRP3AboutSection {
    const data = value && typeof value === 'object' ? value as Record<string, unknown> : {}
    return { ...data, TX: typeof data.TX === 'string' ? data.TX : '', IC: typeof data.IC === 'string' ? data.IC : icon }
  }
  const t3 = source.T3 && typeof source.T3 === 'object' ? source.T3 as Record<string, unknown> : {}
  const t1 = section(source.T1)
  if (!raw) t1.TX = htmlToTRP3Markup(fallback)
  const t2 = Array.isArray(source.T2) ? source.T2 : source.T2 && typeof source.T2 === 'object' ? Object.values(source.T2) : []
  return { ...source, TE: source.TE === 2 || source.TE === 3 ? source.TE : 1, T1: t1, T2: t2.map(item => section(item)), T3: { PH: section(t3.PH, 'ability_warrior_strengthofarms'), PS: section(t3.PS, 'spell_arcane_mindmastery'), HI: section(t3.HI, 'inv_misc_book_12') } } as TRP3About
}
