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
  latestByFlavor?: Record<string, string>
  versions: AddonVersionInfo[]
}

export function addonLatestForFlavor(manifest: AddonManifest, flavor: string): string {
  if (flavor === '_forever_') return manifest.latestByFlavor?._forever_ || ''
  if (!flavor || flavor === '_retail_') return manifest.latest
  return ''
}

export interface AddonLatestResponse {
  version: string
  downloadUrl: string
}

export interface TRP3AddonLatestInfo {
  id: string
  name: string
  projectId: number
  repository: string
  latestVersion: string
  downloadUrl: string
  fileName: string
  fileDate?: string
  sourceUrl: string
  curseforgeUrl: string
  license: string
}

export interface TRP3LatestResponse {
  source: 'github' | 'fallback' | 'mixed' | string
  note: string
  cachedUntil?: string
  addons: TRP3AddonLatestInfo[]
}

export async function getAddonManifest(): Promise<AddonManifest> {
  return request.get<AddonManifest>('/addon/manifest')
}

export async function getAddonLatest(flavor = ''): Promise<AddonLatestResponse> {
  return request.get<AddonLatestResponse>(`/addon/latest?flavor=${encodeURIComponent(flavor)}`)
}

export async function getTRP3Latest(): Promise<TRP3LatestResponse> {
  return request.get<TRP3LatestResponse>('/addon/trp3/latest')
}

export function getAddonDownloadUrl(version: string): string {
  const base = import.meta.env.VITE_API_BASE || (import.meta.env.DEV ? '/api/v1' : 'http://localhost:8080/api/v1')
  return `${base}/addon/download/${version}`
}
