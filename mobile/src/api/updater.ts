import { App } from '@capacitor/app'
import { Capacitor } from '@capacitor/core'
import { request } from '@shared/api/request'

export type MobileTarget = 'android' | 'ios'

export interface MobileUpdateInfo {
  version: string
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
  const path = `/updater/${target}/${arch}/${currentVersion}`

  return request.get<MobileUpdateInfo | null>(path)
}

export function openUpdateUrl(url: string) {
  if (!url) return
  const popup = window.open(url, '_blank', 'noopener,noreferrer')
  if (!popup) {
    window.location.href = url
  }
}

