import request from './request'

export interface DesktopLatestRelease {
  latest_version?: string
  version?: string
  notes?: string
  pub_date?: string
}

export interface NormalizedDesktopLatestRelease {
  latest_version: string
  version: string
  notes: string
  pub_date: string
}

export function normalizeUpdaterVersion(version: string): string {
  const trimmed = version.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('v') || trimmed.startsWith('V')) {
    return trimmed.slice(1).trim()
  }
  return trimmed
}

export function normalizeDesktopLatestRelease(input: DesktopLatestRelease): NormalizedDesktopLatestRelease {
  const resolvedVersion = normalizeUpdaterVersion(input.latest_version || input.version || '')

  return {
    latest_version: resolvedVersion,
    version: resolvedVersion,
    notes: input.notes || '',
    pub_date: input.pub_date || '',
  }
}

export async function getDesktopLatestRelease(): Promise<NormalizedDesktopLatestRelease> {
  const latest = await request.get<DesktopLatestRelease>('/updater/latest')
  return normalizeDesktopLatestRelease(latest)
}
