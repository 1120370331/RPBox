import { request } from '@shared/api/request'

export type ReportTargetType = 'post' | 'item' | 'user' | 'comment' | 'item_comment' | 'rpdb_comment' | 'story' | 'rpdb_work' | 'character_card' | 'guild'

export interface CreateContentReportRequest {
  target_type: ReportTargetType
  target_id: number
  reason: string
  detail?: string
  hide_target?: boolean
  block_author?: boolean
  submit_report?: boolean
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
  return request.post<{ message: string; submitted_report: boolean }>('/user/blocks', {
    blocked_user_id: blockedUserId,
    reason,
    submit_report: false,
  })
}

export function deleteUserBlock(blockedUserId: number) {
  return request.delete<{ message: string }>(`/user/blocks/${blockedUserId}`)
}

export function createContentReport(data: CreateContentReportRequest) {
  return request.post<{ message: string; report_id: number; submitted_report: boolean }>('/reports', {
    ...data,
    submit_report: data.submit_report ?? false,
  })
}
