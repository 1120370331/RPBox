import { describe, expect, it } from 'vitest'
import { buildAchievementEntries, buildAchievementProgressContext } from './achievementProgress'

describe('achievement progress context', () => {
  it('uses RPBox character cards for the nameplate achievement', () => {
    const currentContext = buildAchievementProgressContext({
      profile: {
        id: 1,
        profile_count: 0,
        character_card_count: 1,
      },
    })
    expect(currentContext.profileCount).toBe(1)
    expect(buildAchievementEntries(currentContext).find(entry => entry.definition.id === 'profile.first')?.progress.earned).toBe(true)

    expect(buildAchievementProgressContext({
      profile: {
        id: 1,
        profile_count: 3,
        character_card_count: 0,
      },
    }).profileCount).toBe(0)
  })
})
