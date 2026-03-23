import { request } from '@shared/api/request'

export interface Item {
  id: number
  author_id: number
  author_username?: string
  author_avatar?: string
  name: string
  type: 'item' | 'campaign' | 'artwork'
  icon: string
  preview_image_url?: string
  description: string
  downloads: number
  rating: number
  rating_count: number
  like_count: number
  status: string
  created_at: string
  updated_at: string
}

export interface ListItemsParams {
  type?: string
  search?: string
  sort?: 'created_at' | 'downloads' | 'rating'
  order?: 'asc' | 'desc'
  page?: number
  page_size?: number
}

export interface ItemAuthor {
  id: number
  username: string
  name_color?: string
  name_bold?: boolean
}

export interface ItemComment {
  id: number
  item_id: number
  user_id: number
  rating: number
  content: string
  created_at: string
  updated_at: string
  username: string
  avatar?: string
  name_color?: string
  name_bold?: boolean
}

export interface ItemDetailResponse {
  item: Item
  author: ItemAuthor
  liked: boolean
  favorited: boolean
  tags?: Array<{ id: number; name: string; color?: string }>
  images?: Array<{ id: number; image_url: string; sort_order: number }>
}

export function listItems(params?: ListItemsParams) {
  return request.get<{ items: Item[]; total: number }>('/items', { params })
}

export function getItem(id: number) {
  return request.get<ItemDetailResponse>(`/items/${id}`)
}

export function likeItem(id: number) { return request.post(`/items/${id}/like`) }
export function unlikeItem(id: number) { return request.delete(`/items/${id}/like`) }
export function favoriteItem(id: number) { return request.post(`/items/${id}/favorite`) }
export function unfavoriteItem(id: number) { return request.delete(`/items/${id}/favorite`) }
export function listItemComments(id: number) { return request.get<ItemComment[]>(`/items/${id}/comments`) }
export function createItemComment(id: number, content: string, rating = 0) {
  return request.post(`/items/${id}/comments`, { content, rating })
}
