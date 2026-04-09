export interface ForumLevelDefinition {
  level: number
}

export interface ForumLevelGuideEntry<T extends ForumLevelDefinition> extends T {
  currentBase: number
  nextBase: number | null
}

export function levelStepExperience(level: number) {
  if (level < 1) return 0
  return Math.round(100 * Math.pow(1.8, level - 1))
}

export function levelThresholdExperience(level: number) {
  if (level <= 1) return 0
  return levelStepExperience(level)
}

export function buildForumLevelGuide<T extends ForumLevelDefinition>(definitions: readonly T[]): Array<ForumLevelGuideEntry<T>> {
  return definitions.map((definition, index) => {
    const nextDefinition = definitions[index + 1]

    return {
      ...definition,
      currentBase: levelThresholdExperience(definition.level),
      nextBase: nextDefinition ? levelThresholdExperience(nextDefinition.level) : null,
    }
  })
}

export function computeLevelProgressPercent(currentLevelExp?: number, nextLevelExp?: number) {
  const current = Number(currentLevelExp || 0)
  const next = Number(nextLevelExp || 0)
  if (!next || next <= 0) return 0
  return Math.max(0, Math.min(100, Math.round((current / next) * 100)))
}
