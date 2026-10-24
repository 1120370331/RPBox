import { beforeEach, describe, expect, it } from 'vitest'
import { configuredGameFlavor, getGameInstallations, registerGameInstallation, removeGameInstallation, sameGamePath, selectGameInstallation } from './gameInstallations'

const retail = { path: 'D:\\World of Warcraft\\_retail_', flavor: '_retail_' }
const classic = { path: 'E:\\World of Warcraft\\_classic_era_', flavor: '_classic_era_' }
beforeEach(() => localStorage.clear())
describe('registered game directories', () => {
  it('deduplicates client/WTF/Account paths and makes the confirmed edition authoritative', () => {
    selectGameInstallation({ ...retail, path: retail.path + '\\WTF' })
    registerGameInstallation({ ...retail, flavor: '_classic_era_' })
    expect(getGameInstallations()).toEqual([{ ...retail, flavor: '_classic_era_' }])
    expect(configuredGameFlavor(retail.path + '/WTF/Account/')).toBe('_classic_era_')
    expect(sameGamePath(retail.path, retail.path.toLowerCase() + '\\WTF')).toBe(true)
    expect(localStorage.getItem('selected_flavor')).toBe('_classic_era_')
  })
  it('registers and edits another directory without changing the active directory', () => {
    registerGameInstallation(retail)
    registerGameInstallation(classic)
    registerGameInstallation({ ...classic, flavor: '_classic_anniversary_' })
    expect(localStorage.getItem('wow_path')).toBe(retail.path)
    expect(localStorage.getItem('selected_flavor')).toBe('_retail_')
    expect(getGameInstallations()).toHaveLength(2)
  })
  it('switches when the active entry is removed and clears the final entry without resurrecting legacy settings', () => {
    registerGameInstallation(retail)
    registerGameInstallation(classic)
    removeGameInstallation(retail.path + '/WTF')
    expect(localStorage.getItem('wow_path')).toBe(classic.path)
    removeGameInstallation(classic.path)
    expect(getGameInstallations()).toEqual([])
    expect(localStorage.getItem('wow_path')).toBeNull()
    expect(localStorage.getItem('selected_flavor')).toBeNull()
  })
  it('rejects incomplete confirmations without writing registry entries', () => {
    expect(() => registerGameInstallation({ path: retail.path, flavor: '' })).toThrow()
    expect(() => registerGameInstallation({ path: '', flavor: '_retail_' })).toThrow()
    expect(getGameInstallations()).toEqual([])
  })
})
