import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useUserStore, type UserData } from '@shared/stores/user'

function makeUser(id: number, username: string): UserData {
  return {
    id,
    username,
  }
}

describe('mobile account switching history', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('remembers recent accounts without storing tokens or passwords', () => {
    const store = useUserStore()

    store.setAuth('token-a', makeUser(1, 'alice'))
    store.logout()
    store.setAuth('token-b', makeUser(2, 'bob'))

    const savedHistory = localStorage.getItem('account_history') || ''

    expect(store.accountHistory.map(account => account.username)).toEqual(['bob', 'alice'])
    expect(savedHistory).toContain('alice')
    expect(savedHistory).not.toContain('token-a')
    expect(savedHistory).not.toContain('password')
  })

  it('moves the selected account to the front and supports removing it', () => {
    const store = useUserStore()

    store.setAuth('token-a', makeUser(1, 'alice'))
    store.setAuth('token-b', makeUser(2, 'bob'))
    store.setAuth('token-a2', makeUser(1, 'alice'))

    expect(store.accountHistory.map(account => account.username)).toEqual(['alice', 'bob'])

    store.removeAccountHistoryItem(1)

    expect(store.accountHistory.map(account => account.username)).toEqual(['bob'])
  })

  it('stores account switch sessions separately from account history', () => {
    const store = useUserStore()
    const expiresAt = new Date(Date.now() + 60 * 24 * 60 * 60 * 1000).toISOString()

    store.setAuth('token-a', makeUser(1, 'alice'), {
      switchToken: 'switch-token-a',
      switchTokenExpiresAt: expiresAt,
    })

    const savedHistory = localStorage.getItem('account_history') || ''
    expect(savedHistory).not.toContain('switch-token-a')
    expect(store.hasValidAccountSwitchSession(1)).toBe(true)
    expect(store.getAccountSwitchToken(1)).toBe('switch-token-a')
  })
})
