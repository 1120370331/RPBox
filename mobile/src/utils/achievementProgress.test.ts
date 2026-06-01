import { describe, expect, it } from 'vitest'
import { buildAchievementProgressContext } from './achievementProgress'

describe('mobile achievement progress context', () => {
  it('uses cumulative sign-in stats from the profile payload', () => {
    const context = buildAchievementProgressContext({
      id: 1,
      total_sign_in_days: 31,
      consecutive_sign_in_days: 7,
    })

    expect(context.totalSignIns).toBe(31)
    expect(context.signInStreak).toBe(7)
  })

  it('counts story archive achievement progress by archived story lines', () => {
    const context = buildAchievementProgressContext({
      id: 1,
      story_count: 2,
      story_entry_count: 1200,
    })

    expect(context.storyCount).toBe(1200)
  })

  it('maps community and market stats from the profile payload', () => {
    const context = buildAchievementProgressContext({
      id: 1,
      post_count: 3,
      guild_count: 2,
      item_count: 4,
      max_post_views: 25,
      max_item_downloads: 10,
      total_likes: 100,
      total_item_downloads: 12,
    })

    expect(context.postCount).toBe(3)
    expect(context.guildCount).toBe(2)
    expect(context.itemCount).toBe(4)
    expect(context.maxPostViews).toBe(25)
    expect(context.maxItemDownloads).toBe(10)
    expect(context.totalLikes).toBe(100)
    expect(context.totalItemDownloads).toBe(12)
  })

  it('treats story entry count as authoritative when it is zero', () => {
    const context = buildAchievementProgressContext({
      id: 1,
      story_count: 2,
      story_entry_count: 0,
    })

    expect(context.storyCount).toBe(0)
  })
})
