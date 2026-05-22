import request from './request'

export interface DesktopLatestRelease {
  latest_version?: string
  version?: string
  notes?: string
  pub_date?: string
  channel?: UpdateChannel
}

export type UpdateChannel = 'stable' | 'beta'

export interface NormalizedDesktopLatestRelease {
  latest_version: string
  version: string
  notes: string
  pub_date: string
  channel: UpdateChannel
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
  const channel = input.channel === 'beta' ? 'beta' : 'stable'

  return {
    latest_version: resolvedVersion,
    version: resolvedVersion,
    notes: input.notes || '',
    pub_date: input.pub_date || '',
    channel,
  }
}

export async function getDesktopLatestRelease(options?: { channel?: UpdateChannel }): Promise<NormalizedDesktopLatestRelease> {
  const latest = await request.get<DesktopLatestRelease>(
    '/updater/latest',
    options?.channel ? { params: { channel: options.channel } } : undefined,
  )
  return normalizeDesktopLatestRelease(latest)
}
