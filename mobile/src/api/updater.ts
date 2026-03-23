import { App } from '@capacitor/app'
import { Capacitor } from '@capacitor/core'
import { request } from '@shared/api/request'

export type MobileTarget = 'android' | 'ios'

export interface MobileUpdateInfo {
  version: string
  latest_version?: string
  notes?: string
  pub_date?: string
  url: string
  mandatory?: boolean
}

interface CheckMobileUpdateOptions {
  target?: MobileTarget
  arch?: string
  currentVersion?: string
}

export function normalizeVersion(version: string): string {
  const trimmed = version.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('v') || trimmed.startsWith('V')) {
    return trimmed.slice(1).trim()
  }
  return trimmed
}

function parseVersionParts(version: string): number[] | null {
  const normalized = normalizeVersion(version)
  if (!normalized) return null

  const core = normalized.split(/[+-]/)[0]
  const parts = core.split('.')
  const parsed: number[] = []
  for (const part of parts) {
    if (!part) return null
    const num = Number.parseInt(part, 10)
    if (!Number.isFinite(num)) return null
    parsed.push(num)
  }
  return parsed
}

export function isNewerVersion(latestVersion: string, currentVersion: string): boolean {
  const latest = normalizeVersion(latestVersion)
  const current = normalizeVersion(currentVersion)
  if (!latest) return false
  if (!current) return true

  const latestParts = parseVersionParts(latest)
  const currentParts = parseVersionParts(current)
  if (!latestParts || !currentParts) {
    return latest !== current
  }

  const maxLen = Math.max(latestParts.length, currentParts.length)
  for (let i = 0; i < maxLen; i += 1) {
    const a = latestParts[i] ?? 0
    const b = currentParts[i] ?? 0
    if (a > b) return true
    if (a < b) return false
  }
  return false
}

export function detectMobileTarget(): MobileTarget | null {
  const platform = Capacitor.getPlatform()
  if (platform === 'android' || platform === 'ios') {
    return platform
  }

  const ua = navigator.userAgent || ''
  if (/Android/i.test(ua)) return 'android'
  if (/iPhone|iPad|iPod/i.test(ua)) return 'ios'
  return null
}

export function detectArch(): string {
  const userAgentData = (navigator as Navigator & {
    userAgentData?: { architecture?: string }
  }).userAgentData

  if (userAgentData?.architecture && /arm/i.test(userAgentData.architecture)) {
    return 'aarch64'
  }

  const ua = navigator.userAgent || ''
  if (/arm|aarch64/i.test(ua)) {
    return 'aarch64'
  }
  return 'x86_64'
}

export async function getCurrentAppVersion(): Promise<string> {
  try {
    const info = await App.getInfo()
    return normalizeVersion(info.version || '0.0.0')
  } catch {
    return '0.0.0'
  }
}

export function isMobileUpdaterSupported(): boolean {
  return detectMobileTarget() !== null
}

export async function checkMobileUpdate(options: CheckMobileUpdateOptions = {}): Promise<MobileUpdateInfo | null> {
  const target = options.target ?? detectMobileTarget()
  if (!target) return null

  const arch = options.arch ?? detectArch()
  const currentVersion = normalizeVersion(options.currentVersion ?? await getCurrentAppVersion()) || '0.0.0'
  const latestPath = `/mobile/${target}/latest`
  const updaterPath = `/updater/${target}/${arch}/${currentVersion}`

  // Prefer stable latest endpoint, then fallback to legacy updater endpoint.
  try {
    const latest = await request.get<MobileUpdateInfo>(latestPath)
    const resolvedVersion = normalizeVersion(latest.version || latest.latest_version || '')
    if (resolvedVersion && isNewerVersion(resolvedVersion, currentVersion)) {
      return {
        ...latest,
        version: resolvedVersion,
      }
    }
  } catch {
    // Ignore and fallback to legacy endpoint.
  }

  return request.get<MobileUpdateInfo | null>(updaterPath)
}

export function openUpdateUrl(url: string) {
  if (!url) return
  const popup = window.open(url, '_blank', 'noopener,noreferrer')
  if (!popup) {
    window.location.href = url
  }
}
