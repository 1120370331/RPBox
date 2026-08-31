import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useToastStore } from './toast'

describe('achievement celebration queue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows one completion at a time and advances after dismissal', () => {
    const store = useToastStore()
    store.achievement('第一项', '条件一', { icon: 'ri-star-line', completedAt: '2026-08-31T09:00:00Z' })
    store.achievement('第二项', '条件二')

    expect(store.activeAchievement?.title).toBe('第一项')
    expect(store.activeAchievement?.completedAt).toBe('2026-08-31T09:00:00.000Z')

    store.dismissAchievement()
    expect(store.activeAchievement).toBeNull()
    vi.advanceTimersByTime(180)
    expect(store.activeAchievement?.title).toBe('第二项')
  })
})
