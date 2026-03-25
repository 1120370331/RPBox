import { resolveApiUrl } from '@/api/image'
import { request } from '@shared/api/request'

export interface EmoteItem {
  id: string
  name: string
  text?: string
  url: string
  width?: number
  height?: number
}

export interface EmotePack {
  id: string
  name: string
  icon_url: string
  items: EmoteItem[]
}

export async function listEmotePacks() {
  const res = await request.get<{ packs?: EmotePack[] }>('/emotes')
  const packs = Array.isArray(res?.packs) ? res.packs : []
  return packs.map((pack) => ({
    ...pack,
    icon_url: resolveEmoteUrl(pack.icon_url),
    items: (pack.items || []).map(item => ({
      ...item,
      url: resolveEmoteUrl(item.url),
      width: item.width || 128,
      height: item.height || 128,
    })),
  }))
}

export function resolveEmoteUrl(url: string) {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  return resolveApiUrl(url.startsWith('/') ? url : `/${url}`)
}
