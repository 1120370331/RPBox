import {
  ACHIEVEMENTS,
  getAchievementProgress,
  type AchievementDefinition,
  type AchievementProgressContext,
} from '../data/achievements'

const STORAGE_PREFIX = 'rpbox:achievement-completion:v1'

export function detectNewAchievements(
  userId: number,
  context: AchievementProgressContext,
  storage: Pick<Storage, 'getItem' | 'setItem'> = localStorage,
): AchievementDefinition[] {
  if (!Number.isFinite(userId) || userId <= 0) return []

  const current = ACHIEVEMENTS.filter((achievement) => getAchievementProgress(achievement, context).earned)
  const key = `${STORAGE_PREFIX}:${userId}`
  const stored = readStoredAchievementIds(storage.getItem(key))

  if (stored === null) {
    storage.setItem(key, JSON.stringify(current.map((achievement) => achievement.id)))
    return []
  }

  const newlyEarned = current.filter((achievement) => !stored.has(achievement.id))
  for (const achievement of current) stored.add(achievement.id)
  storage.setItem(key, JSON.stringify([...stored]))
  return newlyEarned
}

function readStoredAchievementIds(value: string | null): Set<string> | null {
  if (value === null) return null
  try {
    const parsed = JSON.parse(value)
    if (!Array.isArray(parsed) || parsed.some((item) => typeof item !== 'string')) return null
    return new Set(parsed)
  } catch {
    return null
  }
}
