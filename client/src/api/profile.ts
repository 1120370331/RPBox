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
  version: number
  created_at: string
  updated_at: string
}

export async function listProfiles(): Promise<CloudProfile[]> {
  const res = await request.get('/profiles')
  return res.data.profiles || []
}

export async function getProfile(id: string): Promise<CloudProfile> {
  const res = await request.get(`/profiles/${id}`)
  return res.data
}

export async function createProfile(data: ProfileData): Promise<CloudProfile> {
  const res = await request.post('/profiles', data)
  return res.data
}

export async function updateProfile(id: string, data: Partial<ProfileData>): Promise<CloudProfile> {
  const res = await request.put(`/profiles/${id}`, data)
  return res.data
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
  const res = await request.get(`/profiles/${id}/versions`)
  return res.data.versions || []
}

export async function rollback(id: string, version: number): Promise<CloudProfile> {
  const res = await request.post(`/profiles/${id}/rollback`, { version })
  return res.data
}
