import request from './request'

export interface AddonVersionInfo {
  version: string
  releaseDate: string
  minClientVersion: string
  changelog: string
  downloadUrl: string
}

export interface AddonManifest {
  name: string
  latest: string
  versions: AddonVersionInfo[]
}

export interface AddonLatestResponse {
  version: string
  downloadUrl: string
}

export async function getAddonManifest(): Promise<AddonManifest> {
  return request.get<AddonManifest>('/addon/manifest')
}

export async function getAddonLatest(): Promise<AddonLatestResponse> {
  return request.get<AddonLatestResponse>('/addon/latest')
}

export function getAddonDownloadUrl(version: string): string {
  const base = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'
  return `${base}/addon/download/${version}`
}
