import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CharacterSaveTargets from './CharacterSaveTargets.vue'
import type { CharacterCard } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({ accounts: vi.fn(), bindings: vi.fn() }))
vi.mock('@/api/accountBackup', () => ({ listAccountBackups: mocks.accounts, ensureAccountBackup: vi.fn() }))
vi.mock('@/api/characterCard', () => ({ getCharacterCardSources: async () => ({ sources: [] }), getCharacterCardTargets: mocks.bindings }))

beforeEach(() => {
  mocks.bindings.mockResolvedValue({ targets: [{ backup_id: 1, profile_id: 'existing-profile' }] })
  mocks.accounts.mockResolvedValue([
    { id: 1, account_id: 'WOW1', game_flavor: '_retail_', version: 3 },
    { id: 2, account_id: 'WOW1', game_flavor: '_classic_era_', version: 5 },
  ])
})

describe('cloud save destinations', () => {
  it('preselects bound accounts and keeps same-name accounts isolated by backup identity and version', async () => {
    mocks.accounts.mockResolvedValue([
      { id: 1, account_id: 'WOW1', game_flavor: '_retail_', version: 3 },
      { id: 2, account_id: 'WOW1', game_flavor: '_classic_era_', version: 5 },
    ])
    const wrapper = mount(CharacterSaveTargets, { props: { card: { id: 10, first_name: '晨星' } as CharacterCard, busy: false }, global: { stubs: { RouterLink: true } } })
    await flushPromises()
    const changes = () => wrapper.emitted('change')?.at(-1)?.[0]
    expect(changes()).toEqual([{ backup_id: 1, profile_id: 'existing-profile', expected_version: 3 }])
    expect(wrapper.get<HTMLInputElement>('input[aria-label="经典怀旧服 WOW1"]').element.checked).toBe(false)
    await wrapper.get('input[aria-label="经典怀旧服 WOW1"]').setValue(true)
    expect(changes()).toEqual([
      { backup_id: 1, profile_id: 'existing-profile', expected_version: 3 },
      { backup_id: 2, profile_id: 'rpbox-10', expected_version: 5 },
    ])
    await wrapper.get('input[aria-label="正式服 WOW1"]').setValue(false)
    await wrapper.get('header button').trigger('click')
    await flushPromises()
    expect(changes()).toEqual([{ backup_id: 2, profile_id: 'rpbox-10', expected_version: 5 }])
    mocks.accounts.mockRejectedValueOnce(new Error('读取失败'))
    await wrapper.get('header button').trigger('click')
    await flushPromises()
    expect(changes()).toEqual([])
    expect(wrapper.get('[role="alert"]').text()).toContain('读取失败')
    wrapper.unmount()
  })
  it('preselects cloud provenance, switches cards cleanly and leaves blank cards unselected', async () => {
    mocks.bindings.mockResolvedValue({ targets: [] })
    const wrapper = mount(CharacterSaveTargets, { props: { card: { id: 10, source_backup_id: 2, source_profile_id: 'source-profile' } as CharacterCard, busy: false }, global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(wrapper.emitted('change')?.at(-1)?.[0]).toEqual([{ backup_id: 2, profile_id: 'source-profile', expected_version: 5 }])
    await wrapper.setProps({ card: { id: 11 } as CharacterCard })
    await flushPromises()
    expect(wrapper.emitted('change')?.at(-1)?.[0]).toEqual([])
    wrapper.unmount()
  })

  it('matches local provenance by account and edition, never by account name alone', async () => {
    mocks.bindings.mockResolvedValue({ targets: [] })
    const wrapper = mount(CharacterSaveTargets, { props: { card: { id: 10, source_account_id: 'WOW1', source_game_flavor: '_classic_era_', source_profile_id: 'local-profile' } as CharacterCard, busy: false }, global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(wrapper.emitted('change')?.at(-1)?.[0]).toEqual([{ backup_id: 2, profile_id: 'local-profile', expected_version: 5 }])
    await wrapper.setProps({ card: { id: 11, source_account_id: 'WOW1', source_profile_id: 'ambiguous-profile' } as CharacterCard })
    await flushPromises()
    expect(wrapper.emitted('change')?.at(-1)?.[0]).toEqual([])
    wrapper.unmount()
  })

})
