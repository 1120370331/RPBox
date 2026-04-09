import { request } from '@shared/api/request'
import { normalizeApiOrigin } from './image'

export interface Item {
  id: number
  author_id: number
  author_username?: string
  author_avatar?: string
  author_name_color?: string
  author_name_bold?: boolean
  author_forum_level?: number
  author_forum_level_name?: string
  author_forum_level_color?: string
  author_forum_level_bold?: boolean
  name: string
  type: 'item' | 'campaign' | 'artwork'
  icon: string
  preview_image?: string
  preview_image_url?: string
  preview_image_updated_at?: string
  description: string
  detail_content?: string
  import_code?: string
  raw_data?: string
  requires_permission?: boolean
  enable_watermark?: boolean
  downloads: number
  rating: number
  rating_count: number
  like_count: number
  favorite_count?: number
  status: 'draft' | 'pending' | 'published' | 'removed'
  review_status?: 'pending' | 'approved' | 'rejected' | ''
  review_comment?: string
  is_public?: boolean
  created_at: string
  updated_at: string
}

export interface ListItemsParams {
  type?: string
  status?: string
  search?: string
  tag_id?: number
  author_id?: number
  author_name?: string
  sort?: 'created_at' | 'downloads' | 'rating'
  order?: 'asc' | 'desc'
  page?: number
  page_size?: number
}

export interface CreateItemRequest {
  name: string
  type: 'item' | 'campaign' | 'artwork'
  icon?: string
  preview_image?: string
  description?: string
  detail_content?: string
  import_code?: string
  raw_data?: string
  requires_permission?: boolean
  enable_watermark?: boolean
  tag_ids?: number[]
  status?: 'draft' | 'published'
  is_public?: boolean
}

export interface UpdateItemRequest {
  name?: string
  description?: string
  detail_content?: string
  icon?: string
  preview_image?: string
  import_code?: string
  raw_data?: string
  requires_permission?: boolean
  enable_watermark?: boolean
  tag_ids?: number[]
  status?: 'draft' | 'published'
  is_public?: boolean
}

export interface ItemAuthor {
  id: number
  username: string
  name_color?: string
  name_bold?: boolean
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
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
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
}

export interface ItemDetailResponse {
  item: Item
  author: ItemAuthor
  liked: boolean
  favorited: boolean
  tags?: Array<{ id: number; name: string; color?: string }>
  images?: Array<{ id: number; image_url: string; sort_order: number }>
  pending_edit?: Record<string, unknown>
}

export interface ItemImage {
  id: number
  image_url: string
  sort_order: number
}

export function listItems(params?: ListItemsParams) {
  return request.get<{ items: Item[]; total: number }>('/items', { params })
}

export function getItem(id: number) {
  return request.get<ItemDetailResponse>(`/items/${id}`)
}

export function createItem(data: CreateItemRequest) {
  return request.post<Item>('/items', data)
}

export function updateItem(id: number, data: UpdateItemRequest) {
  return request.put<Item>(`/items/${id}`, data)
}

export function deleteItem(id: number) {
  return request.delete<void>(`/items/${id}`)
}

export function likeItem(id: number) { return request.post(`/items/${id}/like`) }
export function unlikeItem(id: number) { return request.delete(`/items/${id}/like`) }
export function favoriteItem(id: number) { return request.post(`/items/${id}/favorite`) }
export function unfavoriteItem(id: number) { return request.delete(`/items/${id}/favorite`) }
export function listItemComments(id: number) {
  return request.get<ItemComment[] | { comments: ItemComment[] }>(`/items/${id}/comments`)
}
export function createItemComment(id: number, content: string, rating = 0) {
  return request.post(`/items/${id}/comments`, { content, rating })
}

export function uploadImage(file: File) {
  const formData = new FormData()
  formData.append('image', file)
  return request.post<{ url: string }>('/upload/image', formData)
}

export function uploadAttachment(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return request.post<{ url: string; name?: string }>('/upload/attachment', formData)
}

export function uploadItemImages(itemId: number, files: File[]) {
  const formData = new FormData()
  files.forEach((file) => formData.append('images', file))
  return request.post<{ images: ItemImage[] }>(`/items/${itemId}/images`, formData)
}

export function getItemImages(itemId: number) {
  return request.get<ItemImage[]>(`/items/${itemId}/images`)
}

export function deleteItemImage(itemId: number, imageId: number) {
  return request.delete<void>(`/items/${itemId}/images/${imageId}`)
}

const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'
const API_ORIGIN = normalizeApiOrigin(API_BASE)

export function resolveItemImageUrl(url: string): string {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('data:')) {
    return url
  }
  if (url.startsWith('/')) {
    return API_ORIGIN ? `${API_ORIGIN}${url}` : url
  }
  return API_ORIGIN ? `${API_ORIGIN}/${url}` : url
}
