import { onBeforeUnmount, reactive, watch, type Ref } from 'vue'
import { trp3IconURL } from '@/utils/trp3Markup'

const MAX_CONCURRENT = 6
const MAX_CACHED = 320

/** Current page first; obsolete requests are cancelled rather than queued. */
export function useIconPageImages(current: Ref<string[]>, next: Ref<string[]>) {
  const images = reactive(new Map<string, string>())
  const failed = reactive(new Set<string>())
  const active = new Map<string, AbortController>()
  let desired: string[] = []
  let disposed = false

  function evict() {
    const keep = new Set(desired)
    for (const [name, url] of images) {
      if (images.size <= MAX_CACHED) break
      if (keep.has(name)) continue
      URL.revokeObjectURL(url)
      images.delete(name)
    }
  }
  function pump() {
    if (disposed) return
    for (const name of desired) {
      if (active.size >= MAX_CONCURRENT) break
      if (images.has(name) || active.has(name) || failed.has(name)) continue
      const controller = new AbortController()
      active.set(name, controller)
      // Separate CORS fetches from legacy <img> responses cached without Origin.
      // This fixed key is reused on every visit; it is not a per-request cache bust.
      void fetch(`${trp3IconURL(name)}?picker=1`, { signal: controller.signal, cache: 'force-cache' })
        .then(async response => {
          if (!response.ok) throw new Error(`Icon request failed: ${response.status}`)
          const blob = await response.blob()
          if (disposed || controller.signal.aborted) return
          images.set(name, URL.createObjectURL(blob))
          evict()
        })
        .catch(() => { if (!disposed && !controller.signal.aborted) failed.add(name) })
        .finally(() => { active.delete(name); pump() })
    }
  }
  watch([current, next], ([visible, upcoming]) => {
    desired = [...new Set([...visible, ...upcoming])]
    const visibleSet = new Set(visible)
    const needsCurrent = visible.some(name => !images.has(name) && !failed.has(name))
    for (const [name, controller] of active) {
      if (!desired.includes(name) || (needsCurrent && !visibleSet.has(name))) controller.abort()
    }
    // Refresh LRU order on visits, including revisits that require no fetch.
    for (const name of visible) {
      const url = images.get(name)
      if (url) { images.delete(name); images.set(name, url) }
    }
    evict()
    pump()
  }, { immediate: true })

  function retry() {
    for (const name of current.value) failed.delete(name)
    pump()
  }
  onBeforeUnmount(() => {
    disposed = true
    for (const controller of active.values()) controller.abort()
    for (const url of images.values()) URL.revokeObjectURL(url)
    images.clear()
  })
  return { images, failed, retry }
}
