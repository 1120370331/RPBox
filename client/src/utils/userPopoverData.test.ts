import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearUserPopoverDataCache, loadUserPopoverData } from './userPopoverData'

const mocks = vi.hoisted(() => ({
  getUserProfile: vi.fn(),
  listUserCharacterCards: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  getUserProfile: mocks.getUserProfile,
}))

vi.mock('@/api/characterCard', () => ({
  listUserCharacterCards: mocks.listUserCharacterCards,
}))

describe('userPopoverData cache', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-11T00:00:00Z'))
    clearUserPopoverDataCache()
    mocks.getUserProfile.mockReset().mockResolvedValue({ id: 5, username: '月桂旅人' })
    mocks.listUserCharacterCards.mockReset().mockResolvedValue({ character_cards: [] })
  })

  afterEach(() => {
    clearUserPopoverDataCache()
    vi.useRealTimers()
  })

  it('deduplicates active requests and refreshes resolved data after the short TTL', async () => {
    const first = loadUserPopoverData(5)
    const duplicate = loadUserPopoverData(5)
    await Promise.all([first, duplicate])
    expect(mocks.getUserProfile).toHaveBeenCalledTimes(1)
    expect(mocks.listUserCharacterCards).toHaveBeenCalledTimes(1)

    await loadUserPopoverData(5)
    expect(mocks.getUserProfile).toHaveBeenCalledTimes(1)
    expect(mocks.listUserCharacterCards).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(60_001)
    await loadUserPopoverData(5)
    expect(mocks.getUserProfile).toHaveBeenCalledTimes(2)
    expect(mocks.listUserCharacterCards).toHaveBeenCalledTimes(2)
  })
})
