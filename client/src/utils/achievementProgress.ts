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

export interface AchievementContextInput {
  profile?: Record<string, unknown> | null
  guilds?: Array<Record<string, unknown>>
  sponsorLevel?: number
}

const rarityRank: Record<AchievementRarity, number> = {
  common: 1,
  rare: 2,
  fine: 3,
  epic: 4,
  legendary: 5,
}

export function buildAchievementProgressContext({
  profile,
  guilds = [],
  sponsorLevel = 0,
}: AchievementContextInput): AchievementProgressContext {
  return {
    registered: Boolean(profile?.id),
    totalSignIns: readProfileNumber(profile, 'total_sign_in_days', 'sign_in_days', 'sign_in_count'),
    signInStreak: readProfileNumber(profile, 'consecutive_sign_in_days', 'sign_in_streak', 'continuous_sign_in_days'),
    postCount: readProfileNumber(profile, 'post_count'),
    guildCount: guilds.filter((guild) => guild.status !== 'pending').length || readProfileNumber(profile, 'guild_count', 'guilds_count'),
    itemCount: readProfileNumber(profile, 'item_count', 'items_count'),
    maxPostViews: readProfileNumber(profile, 'max_post_views', 'max_post_view_count'),
    maxItemDownloads: readProfileNumber(profile, 'max_item_downloads', 'max_item_download_count'),
    totalLikes: readProfileNumber(profile, 'total_likes', 'like_count', 'received_likes'),
    totalItemDownloads: readProfileNumber(profile, 'total_item_downloads', 'item_download_count', 'download_count'),
    storyCount: readProfileNumber(profile, 'story_entry_count', 'story_archive_line_count', 'story_count'),
    profileCount: readProfileNumber(profile, 'character_card_count', 'profile_count'),
    sponsorLevel,
    forumLevel: readProfileNumber(profile, 'forum_level'),
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

export function buildAchievementWallEntries(entries: AchievementEntry[], limit = 6) {
  const earned = entries
    .filter((entry) => entry.progress.earned)
    .sort(compareAchievementPriority)
  const inProgress = entries
    .filter((entry) => !entry.progress.earned)
    .sort((a, b) => b.progress.percent - a.progress.percent || compareAchievementPriority(a, b))

  return [...earned, ...inProgress].slice(0, limit)
}

export function compareAchievementPriority(a: AchievementEntry, b: AchievementEntry) {
  return rarityRank[b.definition.rarity] - rarityRank[a.definition.rarity]
    || b.progress.percent - a.progress.percent
    || a.definition.threshold - b.definition.threshold
}

function readProfileNumber(profile: Record<string, unknown> | null | undefined, ...keys: string[]) {
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
