import { describe, expect, it } from 'vitest'
import type { AchievementProgressContext } from '../data/achievements'
import { detectNewAchievements } from './achievementCompletion'

function memoryStorage() {
  const values = new Map<string, string>()
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
  }
}

describe('achievement completion detection', () => {
  it('establishes a quiet baseline and reports each later completion once', () => {
    const storage = memoryStorage()
    const baseline: AchievementProgressContext = { registered: true, profileCount: 0 }
    const completed: AchievementProgressContext = { registered: true, profileCount: 1 }

    expect(detectNewAchievements(7, baseline, storage)).toEqual([])
    expect(detectNewAchievements(7, completed, storage).map(item => item.id)).toEqual(['profile.first'])
    expect(detectNewAchievements(7, completed, storage)).toEqual([])
  })

  it('keeps completion history when a live metric later drops', () => {
    const storage = memoryStorage()
    const baseline: AchievementProgressContext = { registered: true, profileCount: 0 }
    const completed: AchievementProgressContext = { registered: true, profileCount: 1 }

    detectNewAchievements(9, baseline, storage)
    detectNewAchievements(9, completed, storage)
    detectNewAchievements(9, baseline, storage)
    expect(detectNewAchievements(9, completed, storage)).toEqual([])
  })
})
