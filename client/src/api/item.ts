import request from './request'
import { getImageCacheVersion } from '@/utils/imageCache'

// API 基础地址（用于拼接图片等静态资源的完整 URL）
const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'
// 获取不带 /api/v1 的基础地址
const API_HOST = API_BASE.replace(/\/api\/v1\/?$/, '')

export interface Item {
  id: number
  author_id: number
  author_username?: string  // 作者用户名
  author_avatar?: string    // 作者头像
  author_name_color?: string
  author_name_bold?: boolean
  name: string
  type: 'item' | 'campaign' | 'artwork'  // item=道具, campaign=剧本, artwork=画作
  icon: string
  preview_image: string    // 预览图（详情页使用）
  preview_image_url?: string  // 预览图缩略图 URL（列表页使用）
  preview_image_updated_at?: string
  description: string
  detail_content: string   // 富文本详情
  import_code: string
  raw_data: string
  requires_permission: boolean  // 是否需要TRP3权限授权
  enable_watermark: boolean     // 画作是否启用水印
  downloads: number
  rating: number
  rating_count: number
  like_count: number
  favorite_count: number
  status: 'draft' | 'pending' | 'published' | 'removed'
  review_status: 'pending' | 'approved' | 'rejected' | ''
  review_comment?: string
  created_at: string
  updated_at: string
}

// 画作图片
export interface ItemImage {
  id: number
  image_url: string
  sort_order: number
}

export interface ItemComment {
  id: number
  item_id: number
  user_id: number
  rating: number
  content: string
  created_at: string
  updated_at: string
  username?: string
  name_color?: string
  name_bold?: boolean
}

export interface CreateItemRequest {
  name: string
  type: 'item' | 'campaign' | 'artwork'
  icon?: string
  preview_image?: string
  description?: string
  detail_content?: string
  import_code?: string  // 画作类型可选
  raw_data?: string
  requires_permission?: boolean
  enable_watermark?: boolean  // 画作水印开关
  tag_ids?: number[]
  status?: 'draft' | 'published'
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
  enable_watermark?: boolean  // 画作水印开关
  tag_ids?: number[]
  status?: 'draft' | 'published'
}

export interface ListItemsParams {
  type?: 'item' | 'campaign' | 'artwork'
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

// 获取作品列表
export function listItems(params?: ListItemsParams) {
  return request.get('/items', { params })
}

// 创建道具
export function createItem(data: CreateItemRequest) {
  return request.post('/items', data)
}

// 获取道具详情
export function getItem(id: number) {
  return request.get(`/items/${id}`)
}

// 更新道具
export function updateItem(id: number, data: UpdateItemRequest) {
  return request.put(`/items/${id}`, data)
}

// 删除道具
export function deleteItem(id: number) {
  return request.delete(`/items/${id}`)
}

// 下载道具（获取导入代码）
export function downloadItem(id: number) {
  return request.post(`/items/${id}/download`)
}

// 评分
export function rateItem(id: number, rating: number) {
  return request.post(`/items/${id}/rate`, { rating })
}

// 获取评论列表
export function getItemComments(id: number) {
  return request.get(`/items/${id}/comments`)
}

// 添加评论（带评分）
export function addItemComment(id: number, rating: number, content: string) {
  return request.post(`/items/${id}/comments`, { rating, content })
}

// 获取道具标签
export function getItemTags(id: number) {
  return request.get(`/items/${id}/tags`)
}

// 添加道具标签
export function addItemTag(id: number, tagId: number) {
  return request.post(`/items/${id}/tags`, { tag_id: tagId })
}

// 移除道具标签
export function removeItemTag(id: number, tagId: number) {
  return request.delete(`/items/${id}/tags/${tagId}`)
}

// 点赞道具
export function likeItem(id: number) {
  return request.post(`/items/${id}/like`)
}

// 取消点赞
export function unlikeItem(id: number) {
  return request.delete(`/items/${id}/like`)
}

// 收藏道具
export function favoriteItem(id: number) {
  return request.post(`/items/${id}/favorite`)
}

// 取消收藏
export function unfavoriteItem(id: number) {
  return request.delete(`/items/${id}/favorite`)
}

// 获取我收藏的道具
export function listMyItemFavorites() {
  return request.get('/items/favorites')
}

// 获取我点赞的道具
export function listMyItemLikes() {
  return request.get('/items/likes')
}

// 获取我浏览的道具
export function listMyItemViews() {
  return request.get('/items/views')
}

// 上传图片
export function uploadImage(file: File) {
  const formData = new FormData()
  formData.append('image', file)
  return request.post('/upload/image', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export function uploadAttachment(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return request.post('/upload/attachment', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// ========== 画作图片相关 API ==========

// 上传画作图片（支持多张）
export function uploadItemImages(itemId: number, files: File[]) {
  const formData = new FormData()
  files.forEach(file => {
    formData.append('images', file)
  })
  return request.post(`/items/${itemId}/images`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// 获取画作图片列表
export function getItemImages(itemId: number) {
  return request.get<{ code: number; data: ItemImage[] }>(`/items/${itemId}/images`)
}

// 删除画作图片
export function deleteItemImage(itemId: number, imageId: number) {
  return request.delete(`/items/${itemId}/images/${imageId}`)
}

// 获取图片显示URL（完整路径）
export function getItemImageUrl(itemId: number, imageId: number) {
  return `${API_HOST}/api/v1/items/${itemId}/images/${imageId}`
}

// 获取图片下载URL（带水印选项，完整路径）
export function getItemImageDownloadUrl(itemId: number, imageId: number, withWatermark = true) {
  return `${API_HOST}/api/v1/items/${itemId}/images/${imageId}/download?watermark=${withWatermark}`
}

// 获取通用图片URL（支持缩略图，完整路径）
export function getImageUrl(type: string, id: number, options?: { w?: number; q?: number; v?: string | number }) {
  const params = new URLSearchParams()
  if (options?.w) params.set('w', String(options.w))
  if (options?.q) params.set('q', String(options.q))
  if (options?.v !== undefined && options?.v !== null && options?.v !== '') {
    params.set('v', String(options.v))
  }
  const cacheVersion = getImageCacheVersion()
  if (cacheVersion) params.set('cv', cacheVersion)
  const queryString = params.toString()
  return `${API_HOST}/api/v1/images/${type}/${id}${queryString ? '?' + queryString : ''}`
}
