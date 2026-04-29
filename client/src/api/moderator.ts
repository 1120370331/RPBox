import request from './request'

// 类型定义
export interface ModeratorStats {
  pending_posts: number
  pending_items: number
  pending_guilds: number
  pending_reports: number
  pending_post_edits?: number
  pending_item_edits?: number
  pending_post_comment_images?: number
  pending_item_comment_images?: number
  pending_user_avatars?: number
  total_pending_reviews?: number
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

// ========== 审核中心 - 编辑申请 ==========

export interface PostEditReviewItem {
  id: number
  post_id: number
  author_id: number
  title: string
  content: string
  content_type?: string
  category?: string
  region?: string
  address?: string
  status: 'pending' | 'rejected'
  created_at: string
  updated_at: string
  author_name: string
  author_name_color?: string
  author_name_bold?: boolean
  original_title?: string
}

export interface ItemEditReviewItem {
  id: number
  item_id: number
  author_id: number
  name: string
  icon?: string
  description?: string
  import_code?: string
  review_status: 'pending' | 'approved' | 'rejected'
  created_at: string
  updated_at: string
  author_name: string
  author_name_color?: string
  author_name_bold?: boolean
  original_name?: string
}

export function getPendingPostEdits(params?: { page?: number; page_size?: number }) {
  return request.get<{ edits: PostEditReviewItem[]; total: number }>('/moderator/review/post-edits', { params })
}

export function reviewPostEdit(id: number, data: ReviewRequest) {
  return request.post<{ message: string; edit?: PostEditReviewItem; post?: any }>(`/moderator/review/post-edits/${id}`, data)
}

export function getPendingItemEdits(params?: { page?: number; page_size?: number }) {
  return request.get<{ edits: ItemEditReviewItem[]; total: number }>('/moderator/review/item-edits', { params })
}

export function reviewItemEdit(id: number, data: ReviewRequest) {
  return request.post<{ message: string; edit?: ItemEditReviewItem; item?: any }>(`/moderator/review/item-edits/${id}`, data)
}

// ========== 举报审查 ========== 

export interface ReportReviewQueryParams {
  page?: number
  page_size?: number
  status?: 'pending' | 'resolved' | 'rejected' | 'all'
  target_scope?: 'user' | 'content' | 'comment'
  target_type?: 'post' | 'item' | 'user' | 'comment' | 'item_comment'
  sort?: 'report_count' | 'latest_reported_at'
  order?: 'asc' | 'desc'
}

export interface ReportReasonItem {
  id: number
  reporter_id: number
  reporter_name?: string
  reason: string
  detail?: string
  created_at: string
}

export interface ReportReviewItem {
  id: number
  target_type: 'post' | 'item' | 'user' | 'comment' | 'item_comment'
  target_id: number
  target_user_id: number
  target_title: string
  target_author_name?: string
  parent_target_id?: number
  parent_target_title?: string
  target_preview_text?: string
  target_preview_image?: string
  status: 'pending' | 'resolved' | 'rejected'
  report_count: number
  latest_reported_at: string
  reasons: ReportReasonItem[]
  review_comment?: string
  reviewed_at?: string | null
}

export interface ReportReviewRequest {
  action: 'delete_content' | 'delete_and_mute_user' | 'delete_and_ban_user' | 'mute_user' | 'ban_user' | 'reject'
  duration?: number
  comment?: string
}

export function getModeratorReports(params?: ReportReviewQueryParams) {
  return request.get<{ reports: ReportReviewItem[]; total: number }>('/moderator/reports', { params })
}

export function reviewModeratorReport(id: number, data: ReportReviewRequest) {
  return request.post<{ message: string; affected_count: number; status: string }>(`/moderator/reports/${id}/review`, data)
}

// ========== 审核中心 - 图片审核 ==========

export type ImageReviewStatus = 'pending' | 'approved' | 'rejected'

export interface ImageReviewQueryParams {
  page?: number
  page_size?: number
  status?: ImageReviewStatus
}

export interface PostCommentImageReviewItem {
  id: number
  post_id: number
  author_id: number
  parent_id?: number | null
  content: string
  image_url: string
  image_review_status: ImageReviewStatus
  image_review_comment?: string
  image_reviewed_at?: string | null
  created_at: string
  author_name: string
  post_title: string
}

export interface ItemCommentImageReviewItem {
  id: number
  item_id: number
  user_id: number
  parent_id?: number | null
  content: string
  image_url: string
  image_review_status: ImageReviewStatus
  image_review_comment?: string
  image_reviewed_at?: string | null
  created_at: string
  author_name: string
  item_name: string
}

export interface UserAvatarReviewItem {
  id: number
  username: string
  role: string
  avatar_url: string
  avatar_review_status: ImageReviewStatus
  avatar_review_comment?: string
  avatar_reviewed_at?: string | null
  created_at: string
  updated_at: string
}

export function getPendingPostCommentImages(params?: ImageReviewQueryParams) {
  return request.get<{ comments: PostCommentImageReviewItem[]; total: number }>('/moderator/review/post-comment-images', { params })
}

export function reviewPostCommentImage(id: number, data: ReviewRequest) {
  return request.post<{ message: string; comment: PostCommentImageReviewItem }>(`/moderator/review/post-comment-images/${id}`, data)
}

export function getPendingItemCommentImages(params?: ImageReviewQueryParams) {
  return request.get<{ comments: ItemCommentImageReviewItem[]; total: number }>('/moderator/review/item-comment-images', { params })
}

export function reviewItemCommentImage(id: number, data: ReviewRequest) {
  return request.post<{ message: string; comment: ItemCommentImageReviewItem }>(`/moderator/review/item-comment-images/${id}`, data)
}

export function getPendingUserAvatars(params?: ImageReviewQueryParams) {
  return request.get<{ users: UserAvatarReviewItem[]; total: number }>('/moderator/review/user-avatars', { params })
}

export function reviewUserAvatar(id: number, data: ReviewRequest) {
  return request.post<{
    message: string
    user: {
      id: number
      username: string
      avatar_review_status: ImageReviewStatus
      avatar_reviewed_at?: string | null
      avatar_review_comment?: string
    }
  }>(`/moderator/review/user-avatars/${id}`, data)
}

// ========== 管理中心 - 帖子 ==========

export interface PostQueryParams {
  page?: number
  page_size?: number
  status?: string
  review_status?: string
  category?: string
  keyword?: string
  is_pinned?: boolean
  is_featured?: boolean
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
  sponsor_expires_at?: string | null
  name_color?: string
  name_bold?: boolean
  is_muted: boolean
  muted_until: string | null
  mute_reason: string
  is_banned: boolean
  banned_until: string | null
  ban_reason: string
  post_count: number
  activity_points: number
  activity_experience: number
  forum_level: number
  forum_level_name: string
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

export function setUserExperience(id: number, activityExperience: number) {
  return request.put<{ message: string; user: any }>(`/admin/users/${id}/experience`, { activity_experience: activityExperience })
}

export function setUserPoints(id: number, activityPoints: number) {
  return request.put<{ message: string; user: any }>(`/admin/users/${id}/points`, { activity_points: activityPoints })
}

export function adjustUserPoints(id: number, pointsDelta: number) {
  return request.put<{ message: string; user: any }>(`/admin/users/${id}/points`, { points_delta: pointsDelta })
}

export function broadcastSystemMessage(content: string) {
  return request.post<{ message: string; count: number }>('/admin/notifications/broadcast', { content })
}

export interface SponsorRedeemCode {
  id: number
  code: string
  sponsor_level: number
  duration_months: number
  expires_at: string | null
  used_by?: number | null
  used_at?: string | null
  created_by: number
  created_at: string
  updated_at: string
}

export interface CreateSponsorRedeemCodesRequest {
  count: number
  sponsor_level: number
  duration_months: number
  expires_months: number
}

export function createSponsorRedeemCodes(data: CreateSponsorRedeemCodesRequest) {
  return request.post<{ message: string; codes: SponsorRedeemCode[] }>('/admin/sponsor-codes', data)
}

export function getSponsorRedeemCodes(params?: { page?: number; page_size?: number; status?: 'all' | 'active' | 'used' | 'expired' }) {
  return request.get<{ codes: SponsorRedeemCode[]; total: number }>('/admin/sponsor-codes', { params })
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
  new_sign_ins: number
}

export interface PeriodStats {
  users: number
  posts: number
  items: number
  guilds: number
  sign_ins: number
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

export interface BasicDailyMetrics {
  date: string
  new_story_archives: number
  new_story_entries: number
  new_profile_backups: number
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

export function getMetricsBasicHistory(days: number = 30) {
  return request.get<{ metrics: BasicDailyMetrics[]; days: number }>('/moderator/metrics/basic/history', { params: { days } })
}
