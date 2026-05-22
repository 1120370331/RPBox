import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useUserStore, type UserData } from '../stores/user'

function makeUser(id: number, username: string): UserData {
  return {
    id,
    username,
  }
}

describe('user store account history', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('remembers recently logged in accounts without storing credentials', () => {
    const store = useUserStore()

    store.setAuth('token-a', makeUser(1, 'alice'))
    store.logout()
    store.setAuth('token-b', makeUser(2, 'bob'))

    expect(store.accountHistory.map(account => account.username)).toEqual(['bob', 'alice'])
    expect(localStorage.getItem('account_history')).toContain('alice')
    expect(localStorage.getItem('account_history')).not.toContain('token-a')
    expect(localStorage.getItem('account_history')).not.toContain('password')
  })

  it('keeps the latest login first and allows removing a remembered account', () => {
    const store = useUserStore()

    store.setAuth('token-a', makeUser(1, 'alice'))
    store.setAuth('token-b', makeUser(2, 'bob'))
    store.setAuth('token-a2', makeUser(1, 'alice'))

    expect(store.accountHistory.map(account => account.username)).toEqual(['alice', 'bob'])

    store.removeAccountHistoryItem(1)

    expect(store.accountHistory.map(account => account.username)).toEqual(['bob'])
  })

  it('stores 60-day account switch sessions separately from account history', () => {
    const store = useUserStore()
    const expiresAt = new Date(Date.now() + 60 * 24 * 60 * 60 * 1000).toISOString()

    store.setAuth('token-a', makeUser(1, 'alice'), {
      switchToken: 'switch-token-a',
      switchTokenExpiresAt: expiresAt,
    })

    expect(localStorage.getItem('account_history')).not.toContain('switch-token-a')
    expect(store.hasValidAccountSwitchSession(1)).toBe(true)
    expect(store.getAccountSwitchToken(1)).toBe('switch-token-a')
  })
})
