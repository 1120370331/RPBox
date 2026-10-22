import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useIconPageImages } from './useIconPageImages'

afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks() })
describe('icon page request scheduling', () => {
  it('bounds concurrency, aborts obsolete pages and prioritizes the newly selected page', async () => {
    const calls: Array<{ url: string; signal: AbortSignal }> = []
    vi.stubGlobal('fetch', vi.fn((url: string, options: RequestInit) => new Promise((_resolve, reject) => {
      const signal = options.signal as AbortSignal
      calls.push({ url, signal })
      signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
    })))
    const current = ref(Array.from({ length: 80 }, (_, i) => `icon_a_${i}`)), next = ref(['icon_prefetch'])
    const wrapper = mount(defineComponent({ setup() { useIconPageImages(current, next); return () => null } }))
    expect(calls).toHaveLength(6)
    expect(calls.every(call => call.url.includes('icon_a_'))).toBe(true)
    current.value = ['icon_b_0', 'icon_b_1']; next.value = []
    await flushPromises()
    expect(calls.slice(0, 6).every(call => call.signal.aborted)).toBe(true)
    expect(calls.slice(6).map(call => call.url.split('/').pop()?.split('?')[0])).toEqual(['icon_b_0', 'icon_b_1'])
    wrapper.unmount()
    expect(calls.every(call => call.signal.aborted)).toBe(true)
  })
  it('preloads the next page and serves revisits from memory without new requests', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, blob: async () => new Blob(['icon'], { type: 'image/jpeg' }) })
    vi.stubGlobal('fetch', fetcher)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:icon')
    const revoke = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const current = ref(['icon_a']), next = ref(['icon_b'])
    let cache!: ReturnType<typeof useIconPageImages>
    const wrapper = mount(defineComponent({ setup() { cache = useIconPageImages(current, next); return () => null } }))
    await flushPromises()
    expect(fetcher).toHaveBeenCalledTimes(2)
    current.value = ['icon_b']; next.value = ['icon_a']
    await flushPromises()
    expect(cache.images.get('icon_b')).toBe('blob:icon')
    expect(fetcher).toHaveBeenCalledTimes(2)
    wrapper.unmount()
    expect(revoke).toHaveBeenCalledTimes(2)
  })
})
