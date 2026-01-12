import { describe, it, expect } from 'vitest'
import { mapCharacteristics, mapAbout } from '../utils/fieldMapper'

describe('fieldMapper', () => {
  describe('mapCharacteristics', () => {
    it('should map Lua fields to profile fields', () => {
      const luaData = {
        FN: 'Arthas',
        LN: 'Menethil',
        RA: 'Human',
        CL: 'Death Knight',
        AG: 'Unknown',
        TI: 'The Lich King',
      }
      const result = mapCharacteristics(luaData)

      expect(result.firstName).toBe('Arthas')
      expect(result.lastName).toBe('Menethil')
      expect(result.race).toBe('Human')
      expect(result.class).toBe('Death Knight')
      expect(result.age).toBe('Unknown')
      expect(result.title).toBe('The Lich King')
    })

    it('should handle empty data', () => {
      const result = mapCharacteristics({})
      expect(result.version).toBe(1)
      expect(result.firstName).toBeUndefined()
    })
  })
})
