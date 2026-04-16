import { Camera, CameraResultType, CameraSource, type CameraPermissionState } from '@capacitor/camera'
import { Capacitor } from '@capacitor/core'

export type NativeImageSource = 'camera' | 'photos'

export function canUseNativeImagePicker() {
  return Capacitor.isNativePlatform() && Capacitor.isPluginAvailable('Camera')
}

export async function pickSingleNativeImageFile(source: NativeImageSource = 'photos') {
  if (!canUseNativeImagePicker()) return null

  try {
    await ensureNativeImagePermission(source)

    const photo = await Camera.getPhoto({
      source: resolveCameraSource(source),
      resultType: CameraResultType.Uri,
      quality: 90,
      saveToGallery: false,
      presentationStyle: shouldUsePopoverPresentation() ? 'popover' : 'fullscreen',
    })

    return await photoToFile(photo.webPath, photo.format)
  } catch (error) {
    if (isUserCancelledError(error)) {
      return null
    }
    throw error
  }
}

export async function pickNativeEditorImages() {
  const file = await pickSingleNativeImageFile('photos')
  return file ? [file] : []
}

function resolveCameraSource(source: NativeImageSource) {
  return source === 'camera' ? CameraSource.Camera : CameraSource.Photos
}

async function ensureNativeImagePermission(source: NativeImageSource) {
  const permissionKey = source === 'camera' ? 'camera' : 'photos'
  const permission = await getPermissionState(permissionKey)

  if (isPermissionGranted(permission)) {
    return
  }

  const requested = await Camera.requestPermissions({
    permissions: [permissionKey],
  })

  if (!isPermissionGranted(requested[permissionKey])) {
    throw new Error(source === 'camera' ? '需要相机权限才能拍照' : '需要照片权限才能选择图片')
  }
}

async function getPermissionState(permissionKey: 'camera' | 'photos') {
  const permissions = await Camera.checkPermissions()
  return permissions[permissionKey]
}

function isPermissionGranted(state: CameraPermissionState) {
  return state === 'granted' || state === 'limited'
}

function shouldUsePopoverPresentation() {
  if (typeof navigator === 'undefined') return false

  const ua = navigator.userAgent || ''
  const isTouchMac = ua.includes('Macintosh') && navigator.maxTouchPoints > 1
  return /iPad/i.test(ua) || isTouchMac
}

async function photoToFile(webPath?: string, format?: string) {
  if (!webPath) {
    throw new Error('无法读取图片文件')
  }

  const response = await fetch(webPath)
  const blob = await response.blob()
  const extension = normalizeExtension(format, blob.type)
  const mimeType = blob.type || mimeTypeFromExtension(extension)
  const filename = `rpbox-mobile-${Date.now()}.${extension}`

  return new File([blob], filename, {
    type: mimeType,
    lastModified: Date.now(),
  })
}

function normalizeExtension(format?: string, mimeType?: string) {
  const normalized = String(format || '').trim().toLowerCase()
  if (normalized === 'jpg') return 'jpeg'
  if (normalized) return normalized

  if (mimeType === 'image/png') return 'png'
  if (mimeType === 'image/webp') return 'webp'
  if (mimeType === 'image/heic') return 'heic'
  if (mimeType === 'image/heif') return 'heif'
  return 'jpeg'
}

function mimeTypeFromExtension(extension: string) {
  switch (extension) {
    case 'png':
      return 'image/png'
    case 'webp':
      return 'image/webp'
    case 'heic':
      return 'image/heic'
    case 'heif':
      return 'image/heif'
    default:
      return 'image/jpeg'
  }
}

function isUserCancelledError(error: unknown) {
  const message = String((error as { message?: string } | undefined)?.message || error || '').toLowerCase()
  return message.includes('cancel') || message.includes('canceled') || message.includes('cancelled') || message.includes('user denied')
}
