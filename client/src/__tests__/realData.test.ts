import { describe, it, expect } from 'vitest'
import { mapCharacteristics, mapProfile } from '../utils/fieldMapper'

// 真实TRP3人物卡数据
const realProfile1 = {
  profileName: 'Kulyth',
  player: {
    characteristics: {
      CL: '圣骑士',
      RA: '人类',
      RS: 0,
      PS: {},
      MI: {},
      FN: '克拉丽丝',
      IC: 'achievement_character_human_male',
      v: 3,
    },
    character: {
      RP: 2,
      v: 1,
      WU: 1,
    },
    about: {
      T2: {},
      T3: { PH: {}, PS: {}, HI: {} },
      TE: 1,
      T1: {},
      v: 1,
    },
  },
}

const realProfile2 = {
  profileName: 'Fulaliya',
  player: {
    characteristics: {
      CL: '潜行者',
      RA: '人类',
      PS: {},
      MI: {},
      FN: '芙拉莉雅',
      IC: 'achievement_character_human_female',
      v: 1,
    },
  },
}

describe('Real TRP3 Data Tests', () => {
  describe('mapCharacteristics with real data', () => {
    it('should parse 克拉丽丝 profile', () => {
      const result = mapCharacteristics(realProfile1.player.characteristics)

      expect(result.firstName).toBe('克拉丽丝')
      expect(result.race).toBe('人类')
      expect(result.class).toBe('圣骑士')
      expect(result.icon).toBe('achievement_character_human_male')
      expect(result.relationshipStatus).toBe(0)
    })

    it('should parse 芙拉莉雅 profile', () => {
      const result = mapCharacteristics(realProfile2.player.characteristics)

      expect(result.firstName).toBe('芙拉莉雅')
      expect(result.race).toBe('人类')
      expect(result.class).toBe('潜行者')
    })
  })

  describe('mapProfile with real data', () => {
    it('should map full profile structure', () => {
      const result = mapProfile(realProfile1, '0110234908i1hCr')

      expect(result.id).toBe('0110234908i1hCr')
      expect(result.profileName).toBe('Kulyth')
      expect(result.characteristics.firstName).toBe('克拉丽丝')
      expect(result.character.rpStatus).toBe(2)
      expect(result.character.walkUp).toBe(1)
    })
  })
})
