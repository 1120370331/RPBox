import type { EmoteItem } from '@/api/emote'

const EMOTE_TOKEN_RE = /\[\[emote:([a-z0-9_-]+):([a-z0-9_-]+)\]\]/gi

export function buildEmoteToken(packId: string, itemId: string) {
  return `[[emote:${packId}:${itemId}]]`
}

export function renderEmoteContent(raw: string, emoteMap: Map<string, EmoteItem>): string {
  if (!raw) return ''
  if (!emoteMap || emoteMap.size === 0) {
    return escapeHtml(raw).replace(/\n/g, '<br>')
  }

  let result = ''
  let lastIndex = 0
  for (const match of raw.matchAll(EMOTE_TOKEN_RE)) {
    const index = match.index ?? 0
    if (index > lastIndex) {
      result += formatText(raw.slice(lastIndex, index))
    }
    const packId = match[1]
    const itemId = match[2]
    const key = `${packId}:${itemId}`
    const item = emoteMap.get(key)
    if (item) {
      const altText = escapeHtml(item.name || item.text || '')
      result += `<img class="comment-emote" src="${item.url}" alt="${altText}" title="${altText}" width="${item.width || 128}" height="${item.height || 128}" />`
    } else {
      result += formatText(match[0])
    }
    lastIndex = index + match[0].length
  }

  if (lastIndex < raw.length) {
    result += formatText(raw.slice(lastIndex))
  }

  return result
}

function formatText(value: string) {
  return escapeHtml(value).replace(/\n/g, '<br>')
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
