import { request } from '@shared/api/request'

export interface Tag {
  id: number
  name: string
  color?: string
  type?: string
  category?: 'story' | 'item' | 'post' | 'rpdb'
  usage_count?: number
}

export function getPresetTags(category?: 'story' | 'item' | 'post' | 'rpdb') {
  const params = category ? { category } : undefined
  return request.get<{ tags: Tag[] }>('/tags/preset', { params })
}
