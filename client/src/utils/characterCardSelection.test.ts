import { describe, expect, it } from 'vitest'
import type { CharacterCardSummary } from '@/api/characterCard'
import { buildCharacterChoices, cardOrigin, matchesCharacterKeywords } from './characterCardSelection'

describe('character selection search and sources', () => {
  it('matches multiple fields, case, colored names and joined Chinese names', () => {
    expect(matchesCharacterKeywords('|cffaabbcc艾琳 晨星|r 远行者 高等精灵 Mage ACCOUNT-1', '艾琳晨星 mage')).toBe(true)
    expect(matchesCharacterKeywords('艾琳·晨星 游侠', '艾琳晨星')).toBe(true)
    expect(matchesCharacterKeywords('艾琳 晨星 法师', '艾琳 战士')).toBe(false)
    expect(matchesCharacterKeywords('任何名字', '  ')).toBe(true)
  })
  it('uses origin metadata and prefers the linked archive instead of importing a duplicate', () => {
    const cards = [
      { id:1, display_name:'自定义', source_backup_id:null },
      { id:2, display_name:'云端已关联', source_backup_id:10, source_profile_id:'same', source_account_id:'A' },
      { id:3, display_name:'本地已关联', source_account_id:'A', source_profile_id:'local' },
    ] as CharacterCardSummary[]
    expect(cards.map(cardOrigin)).toEqual(['rpbox','cloud','local'])
    const choices = buildCharacterChoices(cards,
      [{ backup_id:10, account_id:'A', profile_id:'same' }, { backup_id:11, account_id:'B', profile_id:'same', display_name:'另一个账号' }],
      [{ account_id:'A', profile_id:'local', profile_name:'已导入', profile:{} }, { account_id:'B', profile_id:'local', profile_name:'新来源', profile:{ player:{ characteristics:{ FN:'本地', LN:'旅人', CL:'法师' } } } }],
    )
    expect(choices).toHaveLength(5)
    expect(choices.filter(choice => choice.kind === 'archive')).toHaveLength(3)
    expect(choices.filter(choice => choice.kind === 'local')[0]?.name).toBe('本地 旅人')
    expect(new Set(choices.map(choice => choice.key)).size).toBe(5)
  })
})
