import { resolveApiUrl } from '@/api/image'
import { listEmotePacks } from '@/api/emote'

const TOKEN_RE = /\[\[(emote|mention):([^\]]+)\]\]/gi
const EMOTE_SIZE = 34
let emoteUrlMap = new Map<string, string>()
let loadingPromise: Promise<void> | null = null

export function buildEmoteToken(packId: string, itemId: string) {
  return `[[emote:${packId}:${itemId}]]`
}

export async function ensureEmoteMapLoaded() {
  if (emoteUrlMap.size > 0) return
  if (loadingPromise) return loadingPromise
  loadingPromise = (async () => {
    try {
      const packs = await listEmotePacks()
      const map = new Map<string, string>()
      for (const pack of packs) {
        const packId = String(pack?.id || '')
        for (const item of pack?.items || []) {
          const itemId = String(item?.id || '')
          const url = String(item?.url || '')
          if (packId && itemId && url) {
            map.set(`${packId}:${itemId}`, url)
          }
        }
      }
      emoteUrlMap = map
    } catch {
      // Ignore: rendering will fall back to static path guess.
    } finally {
      loadingPromise = null
    }
  })()
  return loadingPromise
}

export function renderTextWithEmotes(raw: string) {
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
    if (tokenType === 'emote') {
      const [packId, itemId] = tokenValue.split(':')
      if (packId && itemId) {
        const key = `${packId}:${itemId}`
        const src = emoteUrlMap.get(key) || resolveApiUrl(`/emotes/${encodeURIComponent(packId)}/${encodeURIComponent(itemId)}.png`)
        result += `<img class="inline-emote" src="${src}" alt="" loading="lazy" width="${EMOTE_SIZE}" height="${EMOTE_SIZE}" />`
      } else {
        result += escapeHtml(match[0])
      }
    } else if (tokenType === 'mention') {
      const splitIndex = tokenValue.indexOf(':')
      const labelPart = splitIndex > -1 ? tokenValue.slice(splitIndex + 1) : tokenValue
      const label = decodeMentionLabel(labelPart)
      result += `<span class="inline-mention">@${escapeHtml(label || 'user')}</span>`
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
