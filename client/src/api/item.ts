import request from './request'

export interface Item {
  id: number
  author_id: number
  name: string
  type: 'item' | 'script'  // item=道具, script=剧本
  icon: string
  description: string
  import_code: string
  raw_data: string
  downloads: number
  rating: number
  rating_count: number
  like_count: number
  favorite_count: number
  status: 'draft' | 'published' | 'removed'
  created_at: string
  updated_at: string
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
}

export interface CreateItemRequest {
  name: string
  type: 'item' | 'script'
  icon?: string
  description?: string
  import_code: string
  raw_data?: string
  tag_ids?: number[]
  status?: 'draft' | 'published'
}

export interface UpdateItemRequest {
  name?: string
  description?: string
  icon?: string
  status?: 'draft' | 'published'
}

export interface ListItemsParams {
  type?: 'item' | 'script'
  status?: string
  search?: string
  tag_id?: number
  sort?: 'created_at' | 'downloads' | 'rating'
  order?: 'asc' | 'desc'
  page?: number
  page_size?: number
}

// 获取道具列表
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
