import { describe, expect, it } from 'vitest'
import {
  buildForumLevelGuide,
  computeLevelProgressPercent,
  levelStepExperience,
  levelThresholdExperience,
} from '../utils/forumLevel'

describe('forumLevel utilities', () => {
  it('uses independent thresholds instead of cumulative sums', () => {
    expect(levelThresholdExperience(1)).toBe(0)
    expect(levelThresholdExperience(2)).toBe(180)
    expect(levelThresholdExperience(3)).toBe(324)
    expect(levelThresholdExperience(4)).toBe(583)
  })

  it('builds guide ranges from level thresholds', () => {
    const guide = buildForumLevelGuide([
      { level: 1, name: '新人' },
      { level: 2, name: '启源' },
      { level: 3, name: '常态' },
    ] as const)

    expect(guide).toEqual([
      { level: 1, name: '新人', currentBase: 0, nextBase: 180, stepExperience: 100 },
      { level: 2, name: '启源', currentBase: 180, nextBase: 324, stepExperience: 180 },
      { level: 3, name: '常态', currentBase: 324, nextBase: null, stepExperience: 324 },
    ])
  })

  it('uses the next listed level when building partial guides', () => {
    const guide = buildForumLevelGuide([
      { level: 8, name: '传承' },
      { level: 9, name: '神话' },
      { level: 10, name: '顶级' },
    ] as const)

    expect(guide[0].nextBase).toBe(11020)
    expect(guide[1].nextBase).toBe(19836)
    expect(guide[2].nextBase).toBeNull()
  })

  it('computes progress percent within the current level span', () => {
    expect(computeLevelProgressPercent(0, 144)).toBe(0)
    expect(computeLevelProgressPercent(72, 144)).toBe(50)
    expect(computeLevelProgressPercent(144, 144)).toBe(100)
  })

  it('clamps invalid progress values safely', () => {
    expect(computeLevelProgressPercent(-12, 144)).toBe(0)
    expect(computeLevelProgressPercent(160, 144)).toBe(100)
    expect(computeLevelProgressPercent(10, 0)).toBe(0)
  })

  it('keeps the published growth factor at 1.8', () => {
    expect(levelStepExperience(5)).toBe(1050)
    expect(levelStepExperience(10)).toBe(19836)
  })
})
