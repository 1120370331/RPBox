import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AuthenticatedImage from './AuthenticatedImage.vue'

describe('AuthenticatedImage', () => {
  const fetchMock = vi.fn()
  const createObjectURL = vi.fn()
  const revokeObjectURL = vi.fn()

  beforeEach(() => {
    localStorage.clear()
    fetchMock.mockReset()
    createObjectURL.mockReset()
    revokeObjectURL.mockReset()
    vi.stubGlobal('fetch', fetchMock)
    vi.spyOn(URL, 'createObjectURL').mockImplementation(createObjectURL)
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(revokeObjectURL)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('fetches protected sources with Bearer and revokes owned object URLs across changes and unmount', async () => {
    localStorage.setItem('token', 'private-image-token')
    const firstBlob = new Blob(['first'], { type: 'image/png' })
    const secondBlob = new Blob(['second'], { type: 'image/webp' })
    fetchMock
      .mockResolvedValueOnce({ ok: true, blob: vi.fn().mockResolvedValue(firstBlob) })
      .mockResolvedValueOnce({ ok: true, blob: vi.fn().mockResolvedValue(secondBlob) })
    createObjectURL
      .mockReturnValueOnce('blob:authenticated-1')
      .mockReturnValueOnce('blob:authenticated-2')

    const wrapper = mount(AuthenticatedImage, {
      props: { src: '/api/v1/images/private/1', alt: '私有图片' },
    })
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/images/private/1', expect.objectContaining({
      headers: { Authorization: 'Bearer private-image-token' },
      signal: expect.any(AbortSignal),
    }))
    expect(wrapper.get('img').attributes('src')).toBe('blob:authenticated-1')

    await wrapper.setProps({ src: '/api/v1/images/private/2' })
    await flushPromises()

    expect(revokeObjectURL).toHaveBeenCalledWith('blob:authenticated-1')
    expect(wrapper.get('img').attributes('src')).toBe('blob:authenticated-2')

    wrapper.unmount()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:authenticated-2')
  })

  it('attaches Bearer only to relative, same-origin, and configured API-origin sources', async () => {
    localStorage.setItem('token', 'origin-scoped-token')
    const imageBlob = new Blob(['protected'], { type: 'image/png' })
    fetchMock.mockResolvedValue({ ok: true, blob: vi.fn().mockResolvedValue(imageBlob) })
    createObjectURL.mockReturnValue('blob:origin-scoped')

    const wrapper = mount(AuthenticatedImage, {
      props: { src: '/api/v1/images/private/relative', alt: '受保护图片' },
    })
    await flushPromises()

    const sameOriginSource = `${window.location.origin}/api/v1/images/private/same-origin`
    await wrapper.setProps({ src: sameOriginSource })
    await flushPromises()

    const apiOriginSource = 'http://localhost:8080/api/v1/images/private/api-origin'
    await wrapper.setProps({ src: apiOriginSource })
    await flushPromises()

    for (const source of [
      '/api/v1/images/private/relative',
      sameOriginSource,
      apiOriginSource,
    ]) {
      expect(fetchMock).toHaveBeenCalledWith(source, expect.objectContaining({
        headers: { Authorization: 'Bearer origin-scoped-token' },
      }))
    }

    wrapper.unmount()
  })

  it('aborts an in-flight request when the source changes', async () => {
    let resolveFirst!: (value: unknown) => void
    const firstRequest = new Promise((resolve) => {
      resolveFirst = resolve
    })
    const secondBlob = new Blob(['second'], { type: 'image/png' })
    fetchMock
      .mockReturnValueOnce(firstRequest)
      .mockResolvedValueOnce({ ok: true, blob: vi.fn().mockResolvedValue(secondBlob) })
    createObjectURL.mockReturnValue('blob:latest')

    const wrapper = mount(AuthenticatedImage, {
      props: { src: '/api/v1/images/private/slow', alt: '切换中的图片' },
    })
    await wrapper.vm.$nextTick()
    const firstSignal = fetchMock.mock.calls[0][1].signal as AbortSignal
    expect(wrapper.get('[data-loading="true"]').attributes('aria-busy')).toBe('true')

    await wrapper.setProps({ src: '/api/v1/images/private/latest' })
    await flushPromises()

    expect(firstSignal.aborted).toBe(true)
    expect(wrapper.get('img').attributes('src')).toBe('blob:latest')

    resolveFirst({ ok: true, blob: vi.fn().mockResolvedValue(new Blob(['stale'])) })
    await flushPromises()
    expect(createObjectURL).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('does not leak Bearer to external images and displays local blob previews without fetching them', async () => {
    localStorage.setItem('token', 'must-not-leak')
    const publicBlob = new Blob(['public'], { type: 'image/png' })
    fetchMock.mockResolvedValue({ ok: true, blob: vi.fn().mockResolvedValue(publicBlob) })
    createObjectURL.mockReturnValue('blob:public-response')

    const wrapper = mount(AuthenticatedImage, {
      props: { src: 'https://api.example.test/public-image', alt: '公开图片' },
    })
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('https://api.example.test/public-image', expect.objectContaining({
      headers: undefined,
    }))

    await wrapper.setProps({ src: 'blob:local-preview', alt: '本地预览' })
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:public-response')
    expect(wrapper.get('img').attributes('src')).toBe('blob:local-preview')
    wrapper.unmount()
    expect(revokeObjectURL).not.toHaveBeenCalledWith('blob:local-preview')
  })

  it('exposes an accessible error state when fetching fails', async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 403, blob: vi.fn() })
    vi.spyOn(console, 'error').mockImplementation(() => {})

    const wrapper = mount(AuthenticatedImage, {
      props: { src: '/api/v1/images/private/forbidden', alt: '不可用印象图' },
    })
    await flushPromises()

    const state = wrapper.get('[data-failed="true"] .authenticated-image__state')
    expect(state.attributes('role')).toBe('img')
    expect(state.attributes('aria-label')).toBe('不可用印象图')
    expect(state.text()).toContain('图片暂不可用')
    expect(wrapper.emitted('error')).toHaveLength(1)
  })
})
