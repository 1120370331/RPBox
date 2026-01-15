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

export function hidePostByMod(id: number) {
  return request.post<{ message: string }>(`/moderator/manage/posts/${id}/hide`)
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

export function hideItemByMod(id: number) {
  return request.post<{ message: string }>(`/moderator/manage/items/${id}/hide`)
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
  is_muted: boolean
  muted_until: string | null
  mute_reason: string
  is_banned: boolean
  banned_until: string | null
  ban_reason: string
  post_count: number
  created_at: string
}

export function getUsers(params?: UserQueryParams) {
  return request.get<{ users: SafeUser[]; total: number }>('/admin/users', { params })
}

export function setUserRole(id: number, role: 'user' | 'moderator') {
  return request.put<{ message: string; user: any }>(`/admin/users/${id}/role`, { role })
}

// ========== 用户管理（版主可用） ==========

export function getModeratorUsers(params?: UserQueryParams) {
  return request.get<{ users: SafeUser[]; total: number }>('/moderator/users', { params })
}

export interface MuteRequest {
  duration: number  // 禁言时长（小时），0=永久
  reason: string
}

export interface BanRequest {
  duration: number  // 封禁时长（小时），0=永久
  reason: string
}

// 禁言用户
export function muteUser(id: number, data: MuteRequest) {
  return request.post<{ message: string; user: any }>(`/moderator/users/${id}/mute`, data)
}

// 解除禁言
export function unmuteUser(id: number) {
  return request.delete<{ message: string; user: any }>(`/moderator/users/${id}/mute`)
}

// 禁止登录
export function banUser(id: number, data: BanRequest) {
  return request.post<{ message: string; user: any }>(`/moderator/users/${id}/ban`, data)
}

// 解除登录禁止
export function unbanUser(id: number) {
  return request.delete<{ message: string; user: any }>(`/moderator/users/${id}/ban`)
}

// 禁用用户所有帖子
export function disableUserPosts(id: number) {
  return request.post<{ message: string; affected_count: number }>(`/moderator/users/${id}/posts/disable`)
}

// 删除用户所有帖子
export function deleteUserPosts(id: number) {
  return request.delete<{ message: string; affected_count: number }>(`/moderator/users/${id}/posts`)
}
