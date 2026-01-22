import request from './request'

// 类型定义
export interface ModeratorStats {
  pending_posts: number
  pending_items: number
  pending_guilds: number
  total_posts: number
  total_items: number
  total_guilds: number
  total_users: number
  today_posts: number
  today_items: number
  today_users: number
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

export function pinPost(id: number) {
  return request.post<{ message: string; is_pinned: boolean }>(`/moderator/manage/posts/${id}/pin`)
}

export function featurePost(id: number) {
  return request.post<{ message: string; is_featured: boolean }>(`/moderator/manage/posts/${id}/feature`)
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

export function changeGuildOwner(id: number, params: { new_owner_id?: number; new_owner_name?: string }) {
  return request.put<{ message: string; guild: any; new_owner: { id: number; username: string } }>(`/moderator/manage/guilds/${id}/owner`, params)
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
  is_sponsor?: boolean
  sponsor_level?: number
}

export interface SafeUser {
  id: number
  username: string
  email: string
  avatar: string
  role: string
  is_sponsor?: boolean
  sponsor_level?: number
  name_color?: string
  name_bold?: boolean
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

export function setUserSponsorLevel(id: number, sponsorLevel: number) {
  return request.put<{ message: string; user: any }>(`/admin/users/${id}/sponsor`, { sponsor_level: sponsorLevel })
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

// ========== 操作日志 ==========

export interface AdminActionLog {
  id: number
  operator_id: number
  operator_name: string
  operator_role: string
  operator_name_color?: string
  operator_name_bold?: boolean
  action_type: string
  target_type: string
  target_id: number
  target_name: string
  target_name_color?: string
  target_name_bold?: boolean
  details: string
  ip_address: string
  created_at: string
}

export interface ActionLogQueryParams {
  page?: number
  page_size?: number
  operator_id?: number
  action_type?: string
  target_type?: string
  start_date?: string
  end_date?: string
}

export function getActionLogs(params?: ActionLogQueryParams) {
  return request.get<{ logs: AdminActionLog[]; total: number }>('/moderator/action-logs', { params })
}

// ========== 数据统计 ==========

export interface DailyMetrics {
  date: string
  total_users: number
  total_posts: number
  total_items: number
  total_guilds: number
  new_users: number
  new_posts: number
  new_items: number
  new_guilds: number
}

export interface PeriodStats {
  users: number
  posts: number
  items: number
  guilds: number
}

export interface MetricsSummary {
  today: PeriodStats
  yesterday: PeriodStats
  week: PeriodStats
  month: PeriodStats
}

export interface BasicMetrics {
  story_archives: number
  story_entries: number
  profile_backups: number
}

export function getMetricsHistory(days: number = 30) {
  return request.get<{ metrics: DailyMetrics[]; days: number }>('/moderator/metrics/history', { params: { days } })
}

export function getMetricsSummary() {
  return request.get<MetricsSummary>('/moderator/metrics/summary')
}

export function getMetricsBasic() {
  return request.get<BasicMetrics>('/moderator/metrics/basic')
}
