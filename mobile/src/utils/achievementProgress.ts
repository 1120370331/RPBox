import {
  ACHIEVEMENTS,
  ACHIEVEMENT_RARITY_META,
  getAchievementProgress,
  type AchievementDefinition,
  type AchievementProgress,
  type AchievementProgressContext,
  type AchievementRarity,
} from '@/data/achievements'

export interface AchievementEntry {
  definition: AchievementDefinition
  progress: AchievementProgress
}

const rarityRank: Record<AchievementRarity, number> = {
  common: 1,
  rare: 2,
  fine: 3,
  epic: 4,
  legendary: 5,
}

export function buildAchievementProgressContext(profile?: Record<string, unknown> | null): AchievementProgressContext {
  const sponsorLevel = readUserNumber(profile, 'sponsor_level') || (profile?.is_sponsor ? 2 : 0)

  return {
    registered: Boolean(profile?.id),
    totalSignIns: readUserNumber(profile, 'total_sign_in_days', 'sign_in_days', 'sign_in_count'),
    signInStreak: readUserNumber(profile, 'consecutive_sign_in_days', 'sign_in_streak', 'continuous_sign_in_days'),
    postCount: readUserNumber(profile, 'post_count'),
    guildCount: readUserNumber(profile, 'guild_count', 'guilds_count'),
    itemCount: readUserNumber(profile, 'item_count', 'items_count'),
    maxPostViews: readUserNumber(profile, 'max_post_views', 'max_post_view_count'),
    maxItemDownloads: readUserNumber(profile, 'max_item_downloads', 'max_item_download_count'),
    totalLikes: readUserNumber(profile, 'total_likes', 'like_count', 'received_likes'),
    totalItemDownloads: readUserNumber(profile, 'total_item_downloads', 'item_download_count', 'download_count'),
    storyCount: readUserNumber(profile, 'story_entry_count', 'story_archive_line_count', 'story_count'),
    profileCount: readUserNumber(profile, 'profile_count'),
    sponsorLevel,
    forumLevel: readUserNumber(profile, 'forum_level'),
  }
}

export function buildAchievementEntries(context: AchievementProgressContext): AchievementEntry[] {
  return ACHIEVEMENTS.map((definition) => ({
    definition,
    progress: getAchievementProgress(definition, context),
  }))
}

export function summarizeAchievementRarities(entries: AchievementEntry[]) {
  return (Object.keys(ACHIEVEMENT_RARITY_META) as AchievementRarity[]).map((rarity) => {
    const rarityEntries = entries.filter((entry) => entry.definition.rarity === rarity)
    return {
      rarity,
      label: ACHIEVEMENT_RARITY_META[rarity].label,
      earned: rarityEntries.filter((entry) => entry.progress.earned).length,
      total: rarityEntries.length,
    }
  })
}

export function pickFeaturedAchievement(entries: AchievementEntry[]) {
  return [...entries]
    .filter((entry) => entry.progress.earned)
    .sort(compareAchievementPriority)[0] || entries[0]
}

export function buildAchievementWallEntries(entries: AchievementEntry[], limit = 5) {
  const earned = entries.filter((entry) => entry.progress.earned).sort(compareAchievementPriority)
  const inProgress = entries
    .filter((entry) => !entry.progress.earned)
    .sort((a, b) => b.progress.percent - a.progress.percent || compareAchievementPriority(a, b))

  return [...earned, ...inProgress].slice(0, limit)
}

function compareAchievementPriority(a: AchievementEntry, b: AchievementEntry) {
  return rarityRank[b.definition.rarity] - rarityRank[a.definition.rarity]
    || b.progress.percent - a.progress.percent
    || a.definition.threshold - b.definition.threshold
}

function readUserNumber(profile: Record<string, unknown> | null | undefined, ...keys: string[]) {
  for (const key of keys) {
    const value = profile?.[key]
    if (value === undefined || value === null || value === '') {
      continue
    }
    const numberValue = Number(value)
    if (Number.isFinite(numberValue) && numberValue >= 0) {
      return numberValue
    }
  }
  return 0
}
