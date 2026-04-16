import { request } from '@shared/api/request'

export type ReportTargetType = 'post' | 'item' | 'user' | 'comment' | 'item_comment'

export interface CreateContentReportRequest {
  target_type: ReportTargetType
  target_id: number
  reason: string
  detail?: string
  hide_target?: boolean
  block_author?: boolean
}

export interface UserBlockItem {
  id: number
  blocked_user_id: number
  username: string
  avatar?: string
  reason?: string
  created_at: string
}

export function listUserBlocks() {
  return request.get<{ blocks: UserBlockItem[] }>('/user/blocks')
}

export function createUserBlock(blockedUserId: number, reason?: string) {
  return request.post<{ message: string }>('/user/blocks', {
    blocked_user_id: blockedUserId,
    reason,
  })
}

export function deleteUserBlock(blockedUserId: number) {
  return request.delete<{ message: string }>(`/user/blocks/${blockedUserId}`)
}

export function createContentReport(data: CreateContentReportRequest) {
  return request.post<{ message: string; report_id: number }>('/reports', data)
}
