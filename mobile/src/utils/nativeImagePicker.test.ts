import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const {
  checkPermissionsMock,
  requestPermissionsMock,
  getPhotoMock,
  pickImagesMock,
  isNativePlatformMock,
  isPluginAvailableMock,
} = vi.hoisted(() => ({
  checkPermissionsMock: vi.fn(),
  requestPermissionsMock: vi.fn(),
  getPhotoMock: vi.fn(),
  pickImagesMock: vi.fn(),
  isNativePlatformMock: vi.fn(),
  isPluginAvailableMock: vi.fn(),
}))

vi.mock('@capacitor/camera', () => ({
  Camera: {
    checkPermissions: checkPermissionsMock,
    requestPermissions: requestPermissionsMock,
    getPhoto: getPhotoMock,
    pickImages: pickImagesMock,
  },
  CameraResultType: {
    Uri: 'uri',
  },
  CameraSource: {
    Camera: 'CAMERA',
    Photos: 'PHOTOS',
  },
}))

vi.mock('@capacitor/core', () => ({
  Capacitor: {
    isNativePlatform: isNativePlatformMock,
    isPluginAvailable: isPluginAvailableMock,
  },
}))

import { CameraResultType, CameraSource } from '@capacitor/camera'
import { canUseNativeImagePicker, pickSingleNativeImageFile } from './nativeImagePicker'

describe('nativeImagePicker', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      blob: async () => new Blob(['rpbox'], { type: 'image/jpeg' }),
    }))

    isNativePlatformMock.mockReturnValue(true)
    isPluginAvailableMock.mockReturnValue(true)
    checkPermissionsMock.mockResolvedValue({
      camera: 'granted',
      photos: 'granted',
    })
    requestPermissionsMock.mockResolvedValue({
      camera: 'granted',
      photos: 'granted',
    })
    getPhotoMock.mockResolvedValue({
      webPath: 'https://example.com/rpbox.jpg',
      format: 'jpeg',
    })
    pickImagesMock.mockResolvedValue({
      photos: [{
        webPath: 'https://example.com/rpbox.jpg',
        format: 'jpeg',
      }],
    })

    Object.defineProperty(window.navigator, 'userAgent', {
      configurable: true,
      value: 'Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X)',
    })
    Object.defineProperty(window.navigator, 'maxTouchPoints', {
      configurable: true,
      value: 5,
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('only enables the native picker when both native platform and camera plugin exist', () => {
    isNativePlatformMock.mockReturnValue(false)
    expect(canUseNativeImagePicker()).toBe(false)

    isNativePlatformMock.mockReturnValue(true)
    isPluginAvailableMock.mockReturnValue(false)
    expect(canUseNativeImagePicker()).toBe(false)

    isPluginAvailableMock.mockReturnValue(true)
    expect(canUseNativeImagePicker()).toBe(true)
  })

  it('requests camera permission before opening the native camera on iPad', async () => {
    checkPermissionsMock.mockResolvedValue({
      camera: 'prompt',
      photos: 'granted',
    })

    const file = await pickSingleNativeImageFile('camera')

    expect(requestPermissionsMock).toHaveBeenCalledWith({
      permissions: ['camera'],
    })
    expect(getPhotoMock).toHaveBeenCalledWith(expect.objectContaining({
      source: CameraSource.Camera,
      resultType: CameraResultType.Uri,
      presentationStyle: 'popover',
    }))
    expect(pickImagesMock).not.toHaveBeenCalled()
    expect(file).toBeInstanceOf(File)
  })

  it('uses the single-selection system picker without checking broad photo permission', async () => {
    checkPermissionsMock.mockResolvedValue({
      camera: 'granted',
      photos: 'prompt',
    })

    const file = await pickSingleNativeImageFile('photos')

    expect(checkPermissionsMock).not.toHaveBeenCalled()
    expect(requestPermissionsMock).not.toHaveBeenCalled()
    expect(getPhotoMock).not.toHaveBeenCalled()
    expect(pickImagesMock).toHaveBeenCalledWith({
      quality: 90,
      limit: 1,
      presentationStyle: 'popover',
    })
    expect(file).toBeInstanceOf(File)
  })

  it('does not open the camera after camera permission is denied', async () => {
    checkPermissionsMock.mockResolvedValue({
      camera: 'prompt',
      photos: 'granted',
    })
    requestPermissionsMock.mockResolvedValue({
      camera: 'denied',
      photos: 'granted',
    })

    await expect(pickSingleNativeImageFile('camera')).rejects.toThrow('需要相机权限才能拍照')

    expect(getPhotoMock).not.toHaveBeenCalled()
    expect(pickImagesMock).not.toHaveBeenCalled()
  })

  it('returns null when the user cancels the native picker', async () => {
    pickImagesMock.mockRejectedValue(new Error('User cancelled photos app'))

    await expect(pickSingleNativeImageFile('photos')).resolves.toBeNull()
  })
})
