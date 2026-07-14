import type { Tag } from '@/api/tag'

export const RPDB_STYLE_PRESETS: Array<Pick<Tag, 'id' | 'name' | 'color'>> = [
  { id: 0, name: '联盟风格', color: '2F66C8' },
  { id: 0, name: '部落风格', color: 'B83030' },
  { id: 0, name: '库尔提拉斯风格', color: '356A8A' },
  { id: 0, name: '洛丹伦风格', color: '6E6A85' },
  { id: 0, name: '暴风城风格', color: '356AB8' },
  { id: 0, name: '银月城风格', color: 'C08A2C' },
  { id: 0, name: '暗夜精灵风格', color: '6D5DB8' },
  { id: 0, name: '矮人风格', color: '8A6448' },
  { id: 0, name: '侏儒工程风格', color: 'C46B3A' },
  { id: 0, name: '地精工程风格', color: '5D8F3A' },
  { id: 0, name: '被遗忘者风格', color: '5E6E5A' },
  { id: 0, name: '熊猫人风格', color: '4F8C62' },
  { id: 0, name: '德鲁斯瓦风格', color: '5A5B68' },
  { id: 0, name: '达拉然风格', color: '8A6DCC' },
  { id: 0, name: '海盗风格', color: '9A5B38' },
  { id: 0, name: '泰坦遗迹风格', color: 'C2A15A' },
  { id: 0, name: '龙族风格', color: 'B35C42' },
  { id: 0, name: '荒野游侠风格', color: '557A45' },
  { id: 0, name: '圣光教会风格', color: 'C7A95B' },
  { id: 0, name: '暗影诅咒风格', color: '57456F' },
  { id: 0, name: '贵族沙龙风格', color: '8B6F96' },
  { id: 0, name: '海港酒馆风格', color: '4B7991' },
  { id: 0, name: '炼金工坊风格', color: '6F8F46' },
  { id: 0, name: '军旅哨站风格', color: '727A54' },
]

const RPDB_STYLE_RANK = new Map(RPDB_STYLE_PRESETS.map((tag, index) => [tag.name, index]))

export function isRPDBStyleTag(tag: Pick<Tag, 'name'>) {
  return tag.name.endsWith('风格')
}

export function getRPDBStyleRank(name: string) {
  return RPDB_STYLE_RANK.get(name) ?? RPDB_STYLE_PRESETS.length
}

export function sortRPDBStyleTags<T extends Pick<Tag, 'name'> & Partial<Pick<Tag, 'id' | 'usage_count'>>>(tags: T[]) {
  return [...tags].sort((a, b) => {
    const rankDiff = getRPDBStyleRank(a.name) - getRPDBStyleRank(b.name)
    if (rankDiff !== 0) return rankDiff
    const usageDiff = (b.usage_count || 0) - (a.usage_count || 0)
    if (usageDiff !== 0) return usageDiff
    return (a.id || 0) - (b.id || 0)
  })
}
