import { request } from '@shared/api/request'
import type { UserData } from '@shared/stores/user'

export interface UserInfo extends UserData {
  email: string
  avatar: string
}

export interface SignInDailyResponse {
  activity_points?: number
  activity_experience?: number
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
  current_level_exp?: number
  next_level_exp?: number
  signed_in_today?: boolean
  message: string
  granted: boolean
  points_delta: number
  experience_delta: number
}

export interface SponsorUser {
  id: number
  username: string
  avatar?: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  name_color?: string
  name_bold?: boolean
}

export interface UserMentionItem {
  id: number
  username: string
  avatar?: string
  name_color?: string
  name_bold?: boolean
}

export function getUserInfo() {
  return request.get<UserInfo>('/user/info')
}

export function signInDaily() {
  return request.post<SignInDailyResponse>('/user/sign-in')
}

export function deleteAccount(password: string) {
  return request.delete<{ message: string }>('/user/account', { password })
}

export function listSponsors() {
  return request.get<{ users: SponsorUser[] }>('/sponsors')
}

export function uploadAvatar(file: File) {
  const formData = new FormData()
  formData.append('avatar', file)
  return request.post<{ avatar: string }>('/user/avatar', formData)
}

export function searchUsers(keyword: string, limit: number = 10) {
  return request.get<{ users: UserMentionItem[] }>('/users/search', {
    params: { q: keyword, limit },
  })
}
