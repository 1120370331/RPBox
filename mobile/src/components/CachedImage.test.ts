import { createApp, defineComponent, h, nextTick, ref, type App } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

const imageCacheMocks = vi.hoisted(() => ({
  fetchImageObjectUrlWithAuth: vi.fn(),
  getCachedImageObjectUrl: vi.fn(),
  warmImageCache: vi.fn(),
}))

vi.mock('@/utils/imageCache', () => imageCacheMocks)

import CachedImage from './CachedImage.vue'

let app: App<Element> | null = null

async function flushUi() {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe('CachedImage authentication changes', () => {
  it('refetches the same source when it becomes owner-authenticated', async () => {
    imageCacheMocks.getCachedImageObjectUrl.mockResolvedValue('')
    imageCacheMocks.warmImageCache.mockResolvedValue(false)
    imageCacheMocks.fetchImageObjectUrlWithAuth.mockResolvedValue('')

    const authFetch = ref(false)
    const Root = defineComponent({
      setup: () => () => h(CachedImage, {
        src: '/api/v1/images/character-card-portrait/42',
        authFetch: authFetch.value,
      }),
    })
    const host = document.createElement('div')
    document.body.appendChild(host)
    app = createApp(Root)
    app.mount(host)
    await flushUi()

    expect(imageCacheMocks.fetchImageObjectUrlWithAuth).not.toHaveBeenCalled()
    authFetch.value = true
    await flushUi()

    expect(imageCacheMocks.fetchImageObjectUrlWithAuth).toHaveBeenCalledWith(
      '/api/v1/images/character-card-portrait/42',
    )
  })
})
