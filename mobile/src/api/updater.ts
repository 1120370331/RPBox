import { App } from '@capacitor/app'
import { Capacitor, registerPlugin, type PluginListenerHandle } from '@capacitor/core'
import { request } from '@shared/api/request'

export type MobileTarget = 'android' | 'ios'
export type MobileUpdateMode = 'android-in-app' | 'ios-store' | 'external'

export interface MobileUpdateInfo {
  version: string
  latest_version?: string
  notes?: string
  pub_date?: string
  url: string
  mandatory?: boolean
}

export interface UpdateDownloadProgress {
  downloadedBytes: number
  totalBytes: number
  percent: number
  finished: boolean
}

export interface AndroidInstallPermissionStatus {
  required: boolean
  granted: boolean
}

interface CheckMobileUpdateOptions {
  target?: MobileTarget
  arch?: string
  currentVersion?: string
}

interface RPBoxUpdaterPlugin {
  getInstallPermissionStatus(): Promise<AndroidInstallPermissionStatus>
  requestInstallPermission(): Promise<{ opened: boolean; granted: boolean }>
  downloadAndInstall(options: { url: string; version: string }): Promise<{ path: string }>
  addListener(
    eventName: 'downloadProgress',
    listenerFunc: (progress: UpdateDownloadProgress) => void,
  ): Promise<PluginListenerHandle>
}

const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'
const RPBoxUpdater = registerPlugin<RPBoxUpdaterPlugin>('RPBoxUpdater')

function normalizeApiOrigin(apiBase: string): string {
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    return apiBase.replace(/\/api\/v1\/?$/, '')
  }
  return ''
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

export function resolveMobileTarget(platform: string, userAgent: string): MobileTarget | null {
  if (platform === 'android' || platform === 'ios') {
    return platform
  }

  if (/Android/i.test(userAgent)) return 'android'
  if (/iPhone|iPad|iPod/i.test(userAgent)) return 'ios'
  return null
}

export function detectMobileTarget(): MobileTarget | null {
  return resolveMobileTarget(Capacitor.getPlatform(), navigator.userAgent || '')
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

export function getMobileUpdateMode(target: MobileTarget | null = detectMobileTarget()): MobileUpdateMode {
  if (target === 'android' && Capacitor.isNativePlatform()) {
    return 'android-in-app'
  }
  if (target === 'ios') {
    return 'ios-store'
  }
  return 'external'
}

export function resolveUpdateDownloadUrl(url: string): string {
  const trimmed = url.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return trimmed
  }

  const apiOrigin = normalizeApiOrigin(API_BASE)
  if (trimmed.startsWith('/')) {
    if (apiOrigin) return `${apiOrigin}${trimmed}`
    if (window.location.protocol === 'http:' || window.location.protocol === 'https:') {
      return new URL(trimmed, window.location.origin).toString()
    }
    return trimmed
  }

  if (apiOrigin) return `${apiOrigin}/${trimmed}`
  return new URL(trimmed, window.location.href).toString()
}

export function resolveIOSUpdateUrl(url: string): string {
  const resolved = resolveUpdateDownloadUrl(url)
  if (!resolved.startsWith('https://apps.apple.com/')) {
    return resolved
  }

  const parsed = new URL(resolved)
  return `itms-apps://${parsed.host}${parsed.pathname}${parsed.search}`
}

export async function getAndroidInstallPermissionStatus(): Promise<AndroidInstallPermissionStatus> {
  if (detectMobileTarget() !== 'android' || !Capacitor.isNativePlatform()) {
    return { required: false, granted: true }
  }
  try {
    return await RPBoxUpdater.getInstallPermissionStatus()
  } catch (error) {
    console.warn('[MobileUpdater] failed to read Android install permission status:', error)
    return { required: false, granted: true }
  }
}

export async function requestAndroidInstallPermission() {
  if (detectMobileTarget() !== 'android' || !Capacitor.isNativePlatform()) {
    return { opened: false, granted: true }
  }
  try {
    return await RPBoxUpdater.requestInstallPermission()
  } catch (error) {
    console.warn('[MobileUpdater] failed to request Android install permission:', error)
    return { opened: false, granted: false }
  }
}

export async function installAndroidUpdate(
  update: MobileUpdateInfo,
  onProgress?: (progress: UpdateDownloadProgress) => void,
) {
  if (detectMobileTarget() !== 'android' || !Capacitor.isNativePlatform()) {
    throw new Error('Android in-app updater is unavailable')
  }

  const url = resolveUpdateDownloadUrl(update.url)
  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    throw new Error('Update URL must be absolute for Android installation')
  }

  const listener = onProgress
    ? await RPBoxUpdater.addListener('downloadProgress', onProgress)
    : null
  try {
    return await RPBoxUpdater.downloadAndInstall({
      url,
      version: update.version || update.latest_version || 'latest',
    })
  } finally {
    await listener?.remove()
  }
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
    const resolvedVersion = normalizeVersion(latest.latest_version || latest.version || '')
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

export function openUpdateUrl(url: string, target: MobileTarget | null = detectMobileTarget()) {
  if (!url) return
  window.location.href = target === 'ios' ? resolveIOSUpdateUrl(url) : resolveUpdateDownloadUrl(url)
}
