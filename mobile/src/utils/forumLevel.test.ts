import { describe, expect, it } from 'vitest'
import {
  buildForumLevelGuide,
  computeLevelProgressPercent,
  levelThresholdExperience,
} from './forumLevel'

describe('mobile forumLevel utilities', () => {
  it('matches threshold boundaries used by the profile guide', () => {
    expect(levelThresholdExperience(6)).toBe(1890)
    expect(levelThresholdExperience(7)).toBe(3401)
    expect(levelThresholdExperience(8)).toBe(6122)
  })

  it('marks the final level as open-ended in the guide', () => {
    const guide = buildForumLevelGuide([
      { level: 8, name: '传承' },
      { level: 9, name: '神话' },
      { level: 10, name: '顶级' },
    ] as const)

    expect(guide[0]).toEqual({ level: 8, name: '传承', currentBase: 6122, nextBase: 11020 })
    expect(guide[1]).toEqual({ level: 9, name: '神话', currentBase: 11020, nextBase: 19836 })
    expect(guide[2]).toEqual({ level: 10, name: '顶级', currentBase: 19836, nextBase: null })
  })

  it('resets progress after a level-up and clamps overflow', () => {
    expect(computeLevelProgressPercent(0, 259)).toBe(0)
    expect(computeLevelProgressPercent(130, 259)).toBe(50)
    expect(computeLevelProgressPercent(400, 259)).toBe(100)
  })
})
