import { request } from './request'

export interface Notification {
  id: number
  user_id: number
  type: string
  actor_id?: number
  target_type: string
  target_id: number
  content: string
  is_read: boolean
  created_at: string
  actor_name?: string
  actor_avatar?: string
}

export interface NotificationsResponse {
  notifications: Notification[]
  total: number
  page: number
  page_size: number
}

export interface UnreadCountResponse {
  count: number
}

// 获取通知列表
export function getNotifications(type: string = 'all', page: number = 1, pageSize: number = 20) {
  return request<NotificationsResponse>('/notifications', {
    method: 'GET',
    params: { type, page, page_size: pageSize },
  })
}

// 标记通知为已读
export function markNotificationAsRead(id: number) {
  return request(`/notifications/${id}/read`, {
    method: 'PUT',
  })
}

// 标记所有通知为已读
export function markAllNotificationsAsRead() {
  return request('/notifications/read-all', {
    method: 'PUT',
  })
}

// 获取未读通知数量
export function getUnreadCount() {
  return request<UnreadCountResponse>('/notifications/unread-count', {
    method: 'GET',
  })
}
