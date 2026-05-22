export type AchievementRarity = 'common' | 'rare' | 'fine' | 'epic' | 'legendary'

export type AchievementCategory =
  | 'account'
  | 'checkin'
  | 'community'
  | 'guild'
  | 'market'
  | 'social'
  | 'story'
  | 'profile'
  | 'sponsor'
  | 'level'

export type AchievementMetric =
  | 'registered'
  | 'totalSignIns'
  | 'signInStreak'
  | 'postCount'
  | 'guildCount'
  | 'itemCount'
  | 'maxPostViews'
  | 'maxItemDownloads'
  | 'totalLikes'
  | 'totalItemDownloads'
  | 'storyCount'
  | 'profileCount'
  | 'sponsorLevel'
  | 'forumLevel'

export interface AchievementDefinition {
  id: string
  title: string
  condition: string
  category: AchievementCategory
  rarity: AchievementRarity
  metric: AchievementMetric
  threshold: number
  icon: string
  centerMotif: string
  spriteCell: {
    row: number
    col: number
  }
}

export interface AchievementProgressContext {
  registered?: boolean
  totalSignIns?: number
  signInStreak?: number
  postCount?: number
  guildCount?: number
  itemCount?: number
  maxPostViews?: number
  maxItemDownloads?: number
  totalLikes?: number
  totalItemDownloads?: number
  storyCount?: number
  profileCount?: number
  sponsorLevel?: number
  forumLevel?: number
}

export interface AchievementProgress {
  current: number
  required: number
  percent: number
  earned: boolean
  label: string
}

export const ACHIEVEMENT_RARITY_META: Record<AchievementRarity, {
  label: string
  edge: string
  glow: string
  text: string
}> = {
  common: {
    label: '普通',
    edge: '#B9A996',
    glow: 'rgba(185, 169, 150, 0.24)',
    text: '#6F6257',
  },
  rare: {
    label: '稀有',
    edge: '#5AB7FF',
    glow: 'rgba(90, 183, 255, 0.32)',
    text: '#216EA8',
  },
  fine: {
    label: '精良',
    edge: '#55D66B',
    glow: 'rgba(85, 214, 107, 0.32)',
    text: '#207739',
  },
  epic: {
    label: '史诗',
    edge: '#B26CFF',
    glow: 'rgba(178, 108, 255, 0.34)',
    text: '#6F35B6',
  },
  legendary: {
    label: '传说',
    edge: '#FFB23E',
    glow: 'rgba(255, 178, 62, 0.44)',
    text: '#9A5A00',
  },
}

export const ACHIEVEMENT_CATEGORY_META: Record<AchievementCategory, {
  label: string
  shape: string
}> = {
  account: { label: '账号', shape: 'round' },
  checkin: { label: '签到', shape: 'calendar' },
  community: { label: '社区', shape: 'hex' },
  guild: { label: '公会', shape: 'shield' },
  market: { label: '创意市场', shape: 'diamond' },
  social: { label: '声望', shape: 'heart' },
  story: { label: '剧情归档', shape: 'scroll' },
  profile: { label: '人物卡', shape: 'oval' },
  sponsor: { label: '赞助', shape: 'crown' },
  level: { label: '等级', shape: 'star' },
}

export const ACHIEVEMENTS: AchievementDefinition[] = [
  {
    id: 'account.registered',
    title: '初入 RPBox',
    condition: '完成注册并登录',
    category: 'account',
    rarity: 'common',
    metric: 'registered',
    threshold: 1,
    icon: 'ri-user-add-line',
    centerMotif: 'open account seal',
    spriteCell: { row: 1, col: 1 },
  },
  {
    id: 'checkin.total.7',
    title: '一周问候',
    condition: '累计签到 7 天',
    category: 'checkin',
    rarity: 'common',
    metric: 'totalSignIns',
    threshold: 7,
    icon: 'ri-calendar-check-line',
    centerMotif: 'seven stamped calendar pages',
    spriteCell: { row: 1, col: 2 },
  },
  {
    id: 'checkin.total.31',
    title: '月度常驻',
    condition: '累计签到 31 天',
    category: 'checkin',
    rarity: 'rare',
    metric: 'totalSignIns',
    threshold: 31,
    icon: 'ri-calendar-event-line',
    centerMotif: 'lunar calendar with crescent mark',
    spriteCell: { row: 1, col: 3 },
  },
  {
    id: 'checkin.total.365',
    title: '四季巡礼',
    condition: '累计签到 365 天',
    category: 'checkin',
    rarity: 'epic',
    metric: 'totalSignIns',
    threshold: 365,
    icon: 'ri-calendar-2-line',
    centerMotif: 'four season wheel around a calendar',
    spriteCell: { row: 1, col: 4 },
  },
  {
    id: 'checkin.streak.30',
    title: '不熄灯塔',
    condition: '连续签到 30 天',
    category: 'checkin',
    rarity: 'fine',
    metric: 'signInStreak',
    threshold: 30,
    icon: 'ri-fire-line',
    centerMotif: 'steady lantern flame with thirty sparks',
    spriteCell: { row: 1, col: 5 },
  },
  {
    id: 'checkin.streak.365',
    title: '永续星轨',
    condition: '连续签到 365 天',
    category: 'checkin',
    rarity: 'legendary',
    metric: 'signInStreak',
    threshold: 365,
    icon: 'ri-sun-line',
    centerMotif: 'golden orbit ring with unbroken star trail',
    spriteCell: { row: 2, col: 1 },
  },
  {
    id: 'post.first',
    title: '第一张公告',
    condition: '第一次发布帖子',
    category: 'community',
    rarity: 'common',
    metric: 'postCount',
    threshold: 1,
    icon: 'ri-article-line',
    centerMotif: 'fresh parchment post pinned with wax',
    spriteCell: { row: 2, col: 2 },
  },
  {
    id: 'guild.first',
    title: '同盟之约',
    condition: '第一次加入公会',
    category: 'guild',
    rarity: 'common',
    metric: 'guildCount',
    threshold: 1,
    icon: 'ri-shield-star-line',
    centerMotif: 'joined guild shield with handshake rune',
    spriteCell: { row: 2, col: 3 },
  },
  {
    id: 'item.first',
    title: '造物上架',
    condition: '第一次发布道具',
    category: 'market',
    rarity: 'common',
    metric: 'itemCount',
    threshold: 1,
    icon: 'ri-box-3-line',
    centerMotif: 'crafted item crate with tiny magic spark',
    spriteCell: { row: 2, col: 4 },
  },
  {
    id: 'post.views.20',
    title: '小有围观',
    condition: '单个帖子浏览量达到 20',
    category: 'community',
    rarity: 'rare',
    metric: 'maxPostViews',
    threshold: 20,
    icon: 'ri-eye-line',
    centerMotif: 'watchful eye over twenty small dots',
    spriteCell: { row: 2, col: 5 },
  },
  {
    id: 'item.downloads.10',
    title: '被人带走',
    condition: '单个道具下载量达到 10',
    category: 'market',
    rarity: 'rare',
    metric: 'maxItemDownloads',
    threshold: 10,
    icon: 'ri-download-cloud-line',
    centerMotif: 'download arrow entering a glowing satchel',
    spriteCell: { row: 3, col: 1 },
  },
  {
    id: 'likes.total.100',
    title: '百赞回响',
    condition: '账号累计获得 100 个点赞',
    category: 'social',
    rarity: 'rare',
    metric: 'totalLikes',
    threshold: 100,
    icon: 'ri-heart-3-line',
    centerMotif: 'heart echo with one hundred tiny sparks',
    spriteCell: { row: 3, col: 2 },
  },
  {
    id: 'likes.total.1000',
    title: '千赞声望',
    condition: '账号累计获得 1000 个点赞',
    category: 'social',
    rarity: 'epic',
    metric: 'totalLikes',
    threshold: 1000,
    icon: 'ri-heart-3-fill',
    centerMotif: 'radiant heart crown with applause waves',
    spriteCell: { row: 3, col: 3 },
  },
  {
    id: 'downloads.total.100',
    title: '百次流通',
    condition: '账号累计道具下载量达到 100',
    category: 'market',
    rarity: 'fine',
    metric: 'totalItemDownloads',
    threshold: 100,
    icon: 'ri-download-2-line',
    centerMotif: 'stack of shared item scrolls with download mark',
    spriteCell: { row: 3, col: 4 },
  },
  {
    id: 'downloads.total.1000',
    title: '千人收藏',
    condition: '账号累计道具下载量达到 1000',
    category: 'market',
    rarity: 'epic',
    metric: 'totalItemDownloads',
    threshold: 1000,
    icon: 'ri-download-cloud-2-line',
    centerMotif: 'treasure vault sending item trails outward',
    spriteCell: { row: 3, col: 5 },
  },
  {
    id: 'story.archive.1',
    title: '第一卷归档',
    condition: '归档第一条剧情',
    category: 'story',
    rarity: 'common',
    metric: 'storyCount',
    threshold: 1,
    icon: 'ri-book-open-line',
    centerMotif: 'first open storybook with bookmark ribbon',
    spriteCell: { row: 4, col: 1 },
  },
  {
    id: 'story.archive.1000',
    title: '千卷书库',
    condition: '归档一千条剧情',
    category: 'story',
    rarity: 'fine',
    metric: 'storyCount',
    threshold: 1000,
    icon: 'ri-bookmark-3-line',
    centerMotif: 'library shelf of one thousand glowing volumes',
    spriteCell: { row: 4, col: 2 },
  },
  {
    id: 'story.archive.10000',
    title: '万卷编年',
    condition: '归档一万条剧情',
    category: 'story',
    rarity: 'epic',
    metric: 'storyCount',
    threshold: 10000,
    icon: 'ri-book-2-line',
    centerMotif: 'ancient archive tower filled with floating books',
    spriteCell: { row: 4, col: 3 },
  },
  {
    id: 'story.archive.100000',
    title: '十万史诗',
    condition: '归档十万条剧情',
    category: 'story',
    rarity: 'legendary',
    metric: 'storyCount',
    threshold: 100000,
    icon: 'ri-ancient-gate-line',
    centerMotif: 'mythic archive gate opening to endless manuscripts',
    spriteCell: { row: 4, col: 4 },
  },
  {
    id: 'profile.first',
    title: '亮出名牌',
    condition: '第一次上传人物卡',
    category: 'profile',
    rarity: 'common',
    metric: 'profileCount',
    threshold: 1,
    icon: 'ri-id-card-line',
    centerMotif: 'character card portrait frame with quill',
    spriteCell: { row: 4, col: 5 },
  },
  {
    id: 'sponsor.lv1',
    title: '炉火赞助 Lv1',
    condition: '成为 Lv1 赞助',
    category: 'sponsor',
    rarity: 'rare',
    metric: 'sponsorLevel',
    threshold: 1,
    icon: 'ri-vip-crown-line',
    centerMotif: 'small bronze patron crown',
    spriteCell: { row: 5, col: 1 },
  },
  {
    id: 'sponsor.lv2',
    title: '星辉赞助 Lv2',
    condition: '成为 Lv2 赞助',
    category: 'sponsor',
    rarity: 'fine',
    metric: 'sponsorLevel',
    threshold: 2,
    icon: 'ri-vip-crown-2-line',
    centerMotif: 'silver patron crown with twin gems',
    spriteCell: { row: 5, col: 2 },
  },
  {
    id: 'sponsor.lv3',
    title: '王冠赞助 Lv3',
    condition: '成为 Lv3 赞助',
    category: 'sponsor',
    rarity: 'epic',
    metric: 'sponsorLevel',
    threshold: 3,
    icon: 'ri-vip-diamond-line',
    centerMotif: 'ornate purple-gold patron crown',
    spriteCell: { row: 5, col: 3 },
  },
  {
    id: 'level.7',
    title: '传说账号',
    condition: '账号等级达到 7 级',
    category: 'level',
    rarity: 'fine',
    metric: 'forumLevel',
    threshold: 7,
    icon: 'ri-star-smile-line',
    centerMotif: 'level seven star crest with gold plume',
    spriteCell: { row: 5, col: 4 },
  },
  {
    id: 'level.10',
    title: '神话账号',
    condition: '账号等级达到 10 级',
    category: 'level',
    rarity: 'legendary',
    metric: 'forumLevel',
    threshold: 10,
    icon: 'ri-sparkling-2-line',
    centerMotif: 'mythic crimson star with haloed wings',
    spriteCell: { row: 5, col: 5 },
  },
]

export function getAchievementProgress(
  achievement: AchievementDefinition,
  context: AchievementProgressContext,
): AchievementProgress {
  const current = getMetricValue(achievement.metric, context)
  const required = achievement.threshold
  const earned = current >= required
  const percent = required > 0 ? Math.max(0, Math.min(100, Math.round((current / required) * 100))) : 0

  return {
    current,
    required,
    percent,
    earned,
    label: earned ? '已获得' : `${formatAchievementNumber(current)} / ${formatAchievementNumber(required)}`,
  }
}

export function formatAchievementNumber(value: number) {
  if (value >= 10000) {
    return `${Number((value / 10000).toFixed(value >= 100000 ? 0 : 1))}万`
  }
  return String(value)
}

function getMetricValue(metric: AchievementMetric, context: AchievementProgressContext) {
  if (metric === 'registered') return context.registered ? 1 : 0
  return Number(context[metric] || 0)
}
