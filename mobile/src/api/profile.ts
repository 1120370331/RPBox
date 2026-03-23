import { request } from '@shared/api/request'

export interface CloudProfile {
  id: string
  user_id: number
  account_id: string
  profile_name: string
  raw_lua?: string
  checksum: string
  version: number
  created_at: string
  updated_at: string
}

export interface CloudProfileListResponse {
  profiles: CloudProfile[]
  total: number
  page: number
  page_size: number
}

export function listCloudProfiles(params?: { page?: number; page_size?: number }) {
  return request.get<CloudProfileListResponse>('/profiles', { params })
}

export function getCloudProfile(id: string) {
  return request.get<CloudProfile>(`/profiles/${encodeURIComponent(id)}`)
}
