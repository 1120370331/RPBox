import { request } from './request'

export interface ProfileData {
  id: string
  account_id: string
  profile_name: string
  raw_lua: string
  checksum: string
}

export interface CloudProfile {
  id: string
  user_id: number
  account_id: string
  profile_name: string
  checksum: string
  raw_lua?: string
  version: number
  created_at: string
  updated_at: string
}

export async function listProfiles(): Promise<CloudProfile[]> {
  const res = await request.get<{ profiles?: CloudProfile[] }>('/profiles')
  return res.profiles || []
}

export async function getProfile(id: string): Promise<CloudProfile> {
  return request.get<CloudProfile>(`/profiles/${id}`)
}

export async function createProfile(data: ProfileData): Promise<CloudProfile> {
  return request.post<CloudProfile>('/profiles', data)
}

export async function updateProfile(id: string, data: Partial<ProfileData>): Promise<CloudProfile> {
  return request.put<CloudProfile>(`/profiles/${id}`, data)
}

export async function deleteProfile(id: string): Promise<void> {
  await request.delete(`/profiles/${id}`)
}

export interface ProfileVersion {
  id: number
  profile_id: string
  version: number
  checksum: string
  change_log: string
  created_at: string
}

export async function getVersions(id: string): Promise<ProfileVersion[]> {
  const res = await request.get<{ versions?: ProfileVersion[] }>(`/profiles/${id}/versions`)
  return res.versions || []
}

export async function rollback(id: string, version: number): Promise<CloudProfile> {
  return request.post<CloudProfile>(`/profiles/${id}/rollback`, { version })
}
