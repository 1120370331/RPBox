import request from './request'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'
const API_HOST = API_BASE.replace(/\/api\/v1\/?$/, '')

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

export function listEmotePacks() {
  return request.get<{ packs: EmotePack[] }>('/emotes')
}

export function resolveEmoteUrl(url: string) {
  if (!url) return url
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  if (url.startsWith('/')) return `${API_HOST}${url}`
  return `${API_HOST}/${url}`
}
