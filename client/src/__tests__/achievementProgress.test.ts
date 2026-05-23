import { describe, expect, it } from 'vitest'
import { buildAchievementProgressContext } from '@/utils/achievementProgress'

describe('achievement progress context', () => {
  it('uses cumulative sign-in stats from the profile payload', () => {
    const context = buildAchievementProgressContext({
      profile: {
        id: 1,
        total_sign_in_days: 31,
        consecutive_sign_in_days: 7,
      },
    })

    expect(context.totalSignIns).toBe(31)
    expect(context.signInStreak).toBe(7)
  })

  it('counts story archive achievement progress by archived story lines', () => {
    const context = buildAchievementProgressContext({
      profile: {
        id: 1,
        story_count: 2,
        story_entry_count: 1200,
      },
    })

    expect(context.storyCount).toBe(1200)
  })

  it('treats story entry count as authoritative when it is zero', () => {
    const context = buildAchievementProgressContext({
      profile: {
        id: 1,
        story_count: 2,
        story_entry_count: 0,
      },
    })

    expect(context.storyCount).toBe(0)
  })
})
