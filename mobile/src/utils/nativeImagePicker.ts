import { Camera, CameraResultType, CameraSource } from '@capacitor/camera'
import { Capacitor } from '@capacitor/core'

const cameraPromptLabels = {
  promptLabelHeader: '选择图片',
  promptLabelPhoto: '从相册选择',
  promptLabelPicture: '拍照',
}

export function canUseNativeImagePicker() {
  return Capacitor.isNativePlatform()
}

export async function pickSingleNativeImageFile() {
  if (!canUseNativeImagePicker()) return null

  try {
    const photo = await Camera.getPhoto({
      source: CameraSource.Prompt,
      resultType: CameraResultType.Uri,
      quality: 90,
      saveToGallery: false,
      ...cameraPromptLabels,
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
  const file = await pickSingleNativeImageFile()
  return file ? [file] : []
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
