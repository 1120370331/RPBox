import request from './request'

// 类型定义
export interface ModeratorStats {
  pending_posts: number
  pending_items: number
  pending_guilds: number
  total_posts: number
  total_items: number
  total_guilds: number
  today_posts: number
  today_items: number
}

export interface ReviewRequest {
  action: 'approve' | 'reject'
  comment?: string
}

// 获取版主统计数据
export function getModeratorStats() {
  return request.get<ModeratorStats>('/moderator/stats')
}

// ========== 审核中心 - 帖子 ==========

export function getPendingPosts(params?: { page?: number; page_size?: number; category?: string }) {
  return request.get<{ posts: any[]; total: number }>('/moderator/review/posts', { params })
}

export function reviewPost(id: number, data: ReviewRequest) {
  return request.post<{ message: string; post: any }>(`/moderator/review/posts/${id}`, data)
}

// ========== 审核中心 - 道具 ==========

export function getPendingItems(params?: { page?: number; page_size?: number; type?: string }) {
  return request.get<{ items: any[]; total: number }>('/moderator/review/items', { params })
}

export function reviewItem(id: number, data: ReviewRequest) {
  return request.post<{ message: string; item: any }>(`/moderator/review/items/${id}`, data)
}

// ========== 管理中心 - 帖子 ==========

export interface PostQueryParams {
  page?: number
  page_size?: number
  status?: string
  review_status?: string
  category?: string
  keyword?: string
}

export function getAllPosts(params?: PostQueryParams) {
  return request.get<{ posts: any[]; total: number }>('/moderator/manage/posts', { params })
}

export function deletePostByMod(id: number) {
  return request.delete<{ message: string }>(`/moderator/manage/posts/${id}`)
}

// ========== 管理中心 - 道具 ==========

export interface ItemQueryParams {
  page?: number
  page_size?: number
  status?: string
  review_status?: string
  type?: string
  keyword?: string
}

export function getAllItems(params?: ItemQueryParams) {
  return request.get<{ items: any[]; total: number }>('/moderator/manage/items', { params })
}

export function deleteItemByMod(id: number) {
  return request.delete<{ message: string }>(`/moderator/manage/items/${id}`)
}

// ========== 公会管理 ==========

export interface GuildQueryParams {
  page?: number
  page_size?: number
  status?: string
  keyword?: string
}

export function getPendingGuilds(params?: { page?: number; page_size?: number }) {
  return request.get<{ guilds: any[]; total: number }>('/moderator/review/guilds', { params })
}

export function reviewGuild(id: number, data: ReviewRequest) {
  return request.post<{ message: string; guild: any }>(`/moderator/review/guilds/${id}`, data)
}

export function getAllGuilds(params?: GuildQueryParams) {
  return request.get<{ guilds: any[]; total: number }>('/moderator/manage/guilds', { params })
}

export function changeGuildOwner(id: number, newOwnerId: number) {
  return request.put<{ message: string; guild: any }>(`/moderator/manage/guilds/${id}/owner`, { new_owner_id: newOwnerId })
}

export function deleteGuildByMod(id: number) {
  return request.delete<{ message: string }>(`/moderator/manage/guilds/${id}`)
}

// ========== 用户管理（仅管理员） ==========

export interface UserQueryParams {
  page?: number
  page_size?: number
  role?: string
  keyword?: string
}

export interface SafeUser {
  id: number
  username: string
  email: string
  avatar: string
  role: string
  created_at: string
}

export function getUsers(params?: UserQueryParams) {
  return request.get<{ users: SafeUser[]; total: number }>('/admin/users', { params })
}

export function setUserRole(id: number, role: 'user' | 'moderator') {
  return request.put<{ message: string; user: any }>(`/admin/users/${id}/role`, { role })
}
