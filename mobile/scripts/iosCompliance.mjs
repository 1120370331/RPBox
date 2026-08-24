import path from 'node:path'
import { pathToFileURL } from 'node:url'

export const IOS_APP_STORE_ID = '6761112311'
export const IOS_DEFAULT_PUBLIC_URL = `https://apps.apple.com/cn/app/rpbox/id${IOS_APP_STORE_ID}`
export const IOS_DEFAULT_API_BASE = 'https://ksxvodevhonx.sealosbja.site/api/v1'

export const IOS_USAGE_DESCRIPTIONS = Object.freeze({
  NSCameraUsageDescription: 'RPBox 仅在您拍照时访问相机，用于上传帖子、道具或评论图片。RPBox accesses the camera only when you take a photo to upload to a post, item, or comment.',
  NSPhotoLibraryUsageDescription: 'RPBox 仅访问您主动选择的照片，用于上传帖子、道具或评论图片。RPBox accesses only photos you select to upload to a post, item, or comment.',
  NSPhotoLibraryAddUsageDescription: 'RPBox 不会自动保存图片；仅在您主动保存时写入照片库。RPBox never saves automatically and writes to Photos only when you choose to save.',
})

const placeholderAppIds = new Set([
  '0123456789',
  '123456789',
  '1234567890',
])

function parseStrictHttpsUrl(value, label) {
  if (typeof value !== 'string' || !value || value !== value.trim() || /[\r\n]/.test(value)) {
    throw new Error(`${label} must be non-empty and contain no surrounding whitespace`)
  }

  let parsed
  try {
    parsed = new URL(value)
  } catch {
    throw new Error(`${label} must be an absolute URL`)
  }

  if (parsed.protocol !== 'https:') {
    throw new Error(`${label} must use HTTPS`)
  }
  if (parsed.username || parsed.password) {
    throw new Error(`${label} must not contain credentials`)
  }
  return parsed
}

export function validateIosApiBase(value) {
  const parsed = parseStrictHttpsUrl(value, 'iOS API base')
  const hostname = parsed.hostname.toLowerCase().replace(/^\[|\]$/g, '').replace(/\.+$/, '')
  const isLocalhost = hostname === 'localhost'
    || hostname.endsWith('.localhost')
    || hostname === 'localhost.localdomain'
  const isIpv4Loopback = /^127(?:\.|$)/.test(hostname)
  const isIpv6Loopback = hostname === '::1' || /^::ffff:7f[0-9a-f]{2}:/i.test(hostname)

  if (isLocalhost || isIpv4Loopback || isIpv6Loopback) {
    throw new Error('iOS API base must not target localhost or a loopback address')
  }
  if (parsed.search || parsed.hash) {
    throw new Error('iOS API base must not contain a query string or fragment')
  }

  return parsed.href
}

export function validateIosPublicUpdateUrl(value) {
  const parsed = parseStrictHttpsUrl(value, 'Public iOS update URL')
  if (parsed.hostname.toLowerCase().replace(/\.+$/, '') === 'testflight.apple.com') {
    throw new Error('TestFlight URLs are not valid public App Store update URLs')
  }
  if (parsed.host.toLowerCase() !== 'apps.apple.com') {
    throw new Error('Public iOS update URL host must be exactly apps.apple.com')
  }

  const segments = parsed.pathname.split('/').filter(Boolean)
  const appIndex = segments.indexOf('app')
  const hasCanonicalPrefix = appIndex === 0 || (appIndex === 1 && /^[a-z]{2}$/.test(segments[0]))
  const idIndex = segments.length - 1
  const idMatch = segments[idIndex]?.match(/^id(\d+)$/)
  const appId = idMatch?.[1] || ''

  if (!hasCanonicalPrefix || idIndex !== appIndex + 2 || !/^\d{9,12}$/.test(appId)) {
    throw new Error('Public iOS update URL must contain a canonical App Store app path with a numeric id')
  }
  if (placeholderAppIds.has(appId) || /^(\d)\1{8,}$/.test(appId)) {
    throw new Error('Public iOS update URL must not use a placeholder app id')
  }
  if (appId !== IOS_APP_STORE_ID) {
    throw new Error(`Public iOS update URL must target RPBox App Store app id ${IOS_APP_STORE_ID}`)
  }

  return parsed.href
}

function runCli() {
  try {
    if (process.argv[2] === 'validate-public-update-url') {
      process.stdout.write(validateIosPublicUpdateUrl(process.env.IOS_PUBLIC_UPDATE_URL || ''))
      return
    }
    if (process.argv[2] === 'validate-api-base') {
      process.stdout.write(validateIosApiBase(process.env.IOS_API_BASE || ''))
      return
    }
    throw new Error('Usage: node iosCompliance.mjs <validate-public-update-url|validate-api-base>')
  } catch (error) {
    console.error(`[iOS Compliance] ${error instanceof Error ? error.message : String(error)}`)
    process.exitCode = 1
  }
}

const invokedPath = process.argv[1] ? pathToFileURL(path.resolve(process.argv[1])).href : ''
if (invokedPath === import.meta.url) {
  runCli()
}
