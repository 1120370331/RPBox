import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import SyncMain from './SyncMain.vue'

const mocks = vi.hoisted(() => ({
  invoke: vi.fn(), list: vi.fn(), get: vi.fn(), upsert: vi.fn(), remove: vi.fn(),
  confirm: vi.fn(), alert: vi.fn(),
}))
vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))
vi.mock('@/api/accountBackup', () => ({
  listAccountBackups: mocks.list, getAccountBackup: mocks.get,
  upsertAccountBackup: mocks.upsert, deleteAccountBackup: mocks.remove,
}))
vi.mock('@/composables/useDialog', () => ({
  dialog: { confirm: mocks.confirm, alert: mocks.alert },
  useDialog: () => ({ confirm: mocks.confirm, alert: mocks.alert }),
}))

const retail = 'D:\\Game Center\\Blizzard\\World of Warcraft\\_retail_\\WTF'
const retailOther = 'E:\\World of Warcraft\\_retail_\\WTF'
const classic = 'D:\\Game Center\\Blizzard\\World of Warcraft\\_classic_era_\\WTF'
const backup = (id: number, flavor: string, name: string) => ({
  id, account_id: 'WOW1', game_flavor: flavor, profiles_count: 1, version: 7,
  checksum: `cloud-${id}`, profiles_data: JSON.stringify({ profile: { profileName: name } }),
})
const historyStub = { name: 'AccountBackupVersionHistory', props: ['accountId', 'gameFlavor'], template: '<div class="history-stub" />' }
let wrapper: VueWrapper | undefined

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  localStorage.setItem('token', 'fixture')
  localStorage.setItem('wow_path', retail)
  localStorage.setItem('rpbox_game_installations', JSON.stringify([
    { path: retail, flavor: '_retail_' }, { path: retailOther, flavor: '_retail_' },
    { path: classic, flavor: '_classic_era_' },
  ]))
  Object.defineProperty(window, '__TAURI_INTERNALS__', { configurable: true, value: {} })
  i18n.global.locale.value = 'zh-CN'
  const backups = [backup(1, '_retail_', '正式服人物卡'), backup(2, '_classic_era_', '怀旧服人物卡')]
  mocks.list.mockResolvedValue(backups)
  mocks.get.mockImplementation(async (_account: string, flavor: string) => backups.find(item => item.game_flavor === flavor))
  mocks.confirm.mockResolvedValue(true)
  mocks.alert.mockResolvedValue(undefined)
  mocks.upsert.mockResolvedValue(backups[0])
  mocks.invoke.mockImplementation(async (command: string) => command === 'scan_profiles' ? {
    accounts: [{ account_id: 'WOW1', profiles: [{ id: 'local', name: '本地人物卡', checksum: 'local', raw_lua: '{"profileName":"本地人物卡"}', account_id: 'WOW1', saved_variables_path: 'SavedVariables' }] }],
  } : undefined)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  Reflect.deleteProperty(window, '__TAURI_INTERNALS__')
  localStorage.clear()
})

async function open(query: Record<string, string> = { mode: 'cloud', flavor: '' }) {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/sync', component: SyncMain },
    { path: '/tools/directories', component: { template: '<div />' } },
  ] })
  await router.push({ path: '/sync', query })
  await router.isReady()
  wrapper = mount(SyncMain, { global: {
    plugins: [router, i18n],
    stubs: { AccountBackupVersionHistory: historyStub, LocalTRP3VersionHistory: true },
  } })
  await flushPromises()
  return wrapper
}

describe('bound game version and path', () => {
  it('uses one edition for the path, local scan and cloud data, including an empty old link flavor', async () => {
    const view = await open()
    expect(view.get<HTMLSelectElement>('select[aria-label="游戏版本"]').element.value).toBe('_retail_')
    expect(view.findAll('#sync-game-path option').map(option => option.attributes('value'))).toEqual([retail, retailOther])
    expect(view.text()).not.toContain('标记旧存档')
    expect(view.text()).not.toContain('云端游戏版本')
    expect(view.text()).not.toContain('本机写回目录')
    expect(mocks.get).toHaveBeenLastCalledWith('WOW1', '_retail_')

    await view.get('select[aria-label="游戏版本"]').setValue('_classic_era_')
    await flushPromises()
    expect(view.get<HTMLSelectElement>('#sync-game-path').element.value).toBe(classic)
    expect(view.findAll('#sync-game-path option').map(option => option.attributes('value'))).toEqual([classic])
    expect(mocks.invoke).toHaveBeenLastCalledWith('scan_profiles', { wowPath: classic })
    expect(mocks.get).toHaveBeenLastCalledWith('WOW1', '_classic_era_')
    expect(localStorage.getItem('wow_path')).toBe(classic)
    expect(localStorage.getItem('selected_flavor')).toBe('_classic_era_')
    expect(view.text()).toContain('怀旧服人物卡')
    expect(view.text()).not.toContain('正式服人物卡')
  })

  it('switches only between paths belonging to the selected edition', async () => {
    const view = await open()
    await view.get('#sync-game-path').setValue(retailOther)
    await flushPromises()
    expect(view.get<HTMLSelectElement>('select[aria-label="游戏版本"]').element.value).toBe('_retail_')
    expect(mocks.invoke).toHaveBeenLastCalledWith('scan_profiles', { wowPath: retailOther })
    expect(mocks.get).toHaveBeenLastCalledWith('WOW1', '_retail_')
  })

  it('never reuses another edition path when the requested edition has no installation', async () => {
    const view = await open()
    mocks.invoke.mockClear()
    await view.get('select[aria-label="游戏版本"]').setValue('_beta_')
    await flushPromises()
    expect(view.get<HTMLSelectElement>('#sync-game-path').element.value).toBe('')
    expect(view.get('#sync-game-path').attributes('disabled')).toBeDefined()
    expect(view.get('.toolbar-actions > .btn-primary').attributes('disabled')).toBeDefined()
    expect(mocks.invoke).not.toHaveBeenCalled()
  })

  it('keeps older backups readable and writable through their existing key without a labeling operation', async () => {
    const previous = backup(3, '', '已有云端人物卡')
    mocks.list.mockResolvedValue([previous])
    mocks.get.mockResolvedValue(previous)
    const view = await open()
    expect(mocks.get).toHaveBeenLastCalledWith('WOW1', '')
    expect(view.text()).toContain('已有云端人物卡')
    expect(view.findComponent(historyStub).props('gameFlavor')).toBe('')
    await view.get('.toolbar-actions > .btn-primary').trigger('click')
    await flushPromises()
    expect(mocks.invoke).toHaveBeenCalledWith('apply_account_backup', expect.objectContaining({ wowPath: retail, profilesJson: previous.profiles_data }))

    // A later local edit must still update the same archive and preserve its history.
    mocks.invoke.mockResolvedValueOnce({ accounts: [{ account_id: 'WOW1', profiles: [
      { id: 'local', name: '本地人物卡', checksum: 'edited', raw_lua: '{"profileName":"编辑后的内容"}', account_id: 'WOW1', saved_variables_path: 'SavedVariables' },
    ] }] })
    await view.get('.refresh-info button').trigger('click')
    await flushPromises()
    await view.findAll('.mode-tabs button')[0]!.trigger('click')
    await view.get('.toolbar-actions > .btn-primary').trigger('click')
    await flushPromises()
    await view.get('.modal-actions .btn-primary').trigger('click')
    await flushPromises()
    expect(mocks.upsert).toHaveBeenCalledWith(expect.objectContaining({ account_id: 'WOW1', game_flavor: '' }))
  })

  it('prefers an exact edition backup over an older unversioned backup for the same account', async () => {
    const exact = backup(1, '_retail_', '正式服人物卡')
    mocks.list.mockResolvedValue([exact, backup(3, '', '旧内容')])
    mocks.get.mockResolvedValue(exact)
    const view = await open()
    expect(mocks.get).toHaveBeenLastCalledWith('WOW1', '_retail_')
    expect(view.findComponent(historyStub).props('gameFlavor')).toBe('_retail_')
  })
})
