import { resolveEmoteUrl, type EmoteItem } from '@/api/emote'

const TOKEN_RE = /\[\[(emote|mention):([^\]]+)\]\]/gi
const EMOTE_DISPLAY_SIZE = 64

export function buildEmoteToken(packId: string, itemId: string) {
  return `[[emote:${packId}:${itemId}]]`
}

export function renderEmoteContent(raw: string, emoteMap: Map<string, EmoteItem>): string {
  if (!raw) return ''

  let result = ''
  let lastIndex = 0
  for (const match of raw.matchAll(TOKEN_RE)) {
    const index = match.index ?? 0
    if (index > lastIndex) {
      result += formatText(raw.slice(lastIndex, index))
    }
    const tokenType = match[1]
    const tokenValue = match[2]
    const token = match[0]
    if (tokenType === 'emote') {
      const [packId, itemId] = tokenValue.split(':')
      const key = `${packId}:${itemId}`
      const item = emoteMap?.get(key)
      if (item) {
        const altText = escapeHtml(item.name || item.text || '')
        result += `<img class="comment-emote" src="${item.url}" alt="${altText}" title="${altText}" width="${EMOTE_DISPLAY_SIZE}" height="${EMOTE_DISPLAY_SIZE}" data-emote="${packId}:${itemId}" data-emote-token="${token}" />`
      } else {
        const fallbackUrl = resolveEmoteUrl(`/emotes/${packId}/${itemId}.png`)
        result += `<img class="comment-emote" src="${fallbackUrl}" alt="" width="${EMOTE_DISPLAY_SIZE}" height="${EMOTE_DISPLAY_SIZE}" data-emote="${packId}:${itemId}" data-emote-token="${token}" />`
      }
    } else if (tokenType === 'mention') {
      const splitIndex = tokenValue.indexOf(':')
      const idPart = splitIndex > -1 ? tokenValue.slice(0, splitIndex) : tokenValue
      const labelPart = splitIndex > -1 ? tokenValue.slice(splitIndex + 1) : ''
      const label = decodeMentionLabel(labelPart)
      const safeLabel = escapeHtml(label || '未知用户')
      result += `<span class="comment-mention" data-mention-id="${escapeHtml(idPart)}" data-mention-name="${safeLabel}" contenteditable="false">@${safeLabel}</span>`
    }
    lastIndex = index + match[0].length
  }

  if (lastIndex < raw.length) {
    result += formatText(raw.slice(lastIndex))
  }

  return result
}

function decodeMentionLabel(value: string) {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
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
