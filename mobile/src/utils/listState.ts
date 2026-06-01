import { nextTick, onBeforeUnmount } from 'vue'

type Source<T> = () => T

interface ListStateRegistration<T extends object> {
  key: string
  getState: Source<T>
  getScrollElement?: Source<Element | null | undefined>
}

const stateCache = new Map<string, Record<string, unknown>>()

function getDefaultScrollElement() {
  return document.querySelector('.mobile-content') || document.scrollingElement
}

export function getCachedListState<T extends object>(key: string): Partial<T> | null {
  return (stateCache.get(key) as Partial<T> | undefined) || null
}

export function setCachedListState<T extends object>(key: string, value: T) {
  stateCache.set(key, { ...(value as Record<string, unknown>) })
}

export function captureScrollTop(getScrollElement?: Source<Element | null | undefined>) {
  const element = getScrollElement?.() || getDefaultScrollElement()
  if (!element) return window.scrollY || 0
  return 'scrollTop' in element ? Number((element as HTMLElement).scrollTop) || 0 : 0
}

export function restoreScrollTop(scrollTop: number, getScrollElement?: Source<Element | null | undefined>) {
  const top = Math.max(0, Math.round(scrollTop || 0))
  void nextTick(() => {
    const restore = () => {
      const element = getScrollElement?.() || getDefaultScrollElement()
      if (element && 'scrollTo' in element) {
        element.scrollTo({ top, behavior: 'auto' })
        return
      }
      window.scrollTo({ top, behavior: 'auto' })
    }
    requestAnimationFrame(restore)
    ;[80, 220].forEach((delay) => setTimeout(restore, delay))
  })
}

export function useListStateCache<T extends object>({
  key,
  getState,
  getScrollElement,
}: ListStateRegistration<T>) {
  const save = () => {
    setCachedListState(key, {
      ...getState(),
      scrollTop: captureScrollTop(getScrollElement),
    })
  }

  onBeforeUnmount(save)

  return { save }
}
