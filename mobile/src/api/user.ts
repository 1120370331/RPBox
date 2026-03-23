import { request } from '@shared/api/request'

export interface UserInfo {
  id: number
  username: string
  email: string
  avatar: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  name_color?: string
  name_bold?: boolean
}

export function getUserInfo() {
  return request.get<UserInfo>('/user/info')
}

export function uploadAvatar(file: File) {
  const formData = new FormData()
  formData.append('avatar', file)
  return request.post<{ avatar: string }>('/user/avatar', formData)
}
