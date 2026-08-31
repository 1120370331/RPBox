import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface UserData {
  id: number
  username: string
  avatar?: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  sponsor_color?: string
  sponsor_bold?: boolean
  name_color?: string
  name_bold?: boolean
  activity_points?: number
  activity_experience?: number
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
  current_level_exp?: number
  next_level_exp?: number
  signed_in_today?: boolean
  total_sign_in_days?: number
  consecutive_sign_in_days?: number
  post_count?: number
  guild_count?: number
  item_count?: number
  story_count?: number
  story_entry_count?: number
  profile_count?: number
  character_card_count?: number
  max_post_views?: number
  max_item_downloads?: number
  total_likes?: number
  total_item_downloads?: number
}

const ACCOUNT_HISTORY_KEY = 'account_history'
const ACCOUNT_SESSIONS_KEY = 'account_sessions'
const MAX_ACCOUNT_HISTORY = 5
const ACCOUNT_SWITCH_DAYS = 60

export interface AccountHistoryItem {
  id: number
  username: string
  avatar?: string
  name_color?: string
  name_bold?: boolean
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
  last_login_at: string
  session_expires_at?: string
}

export interface AccountSwitchSession {
  token: string
  expires_at: string
}

function loadAccountHistory(): AccountHistoryItem[] {
  const saved = localStorage.getItem(ACCOUNT_HISTORY_KEY)
  if (!saved) return []

  try {
    const parsed = JSON.parse(saved)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((item): item is AccountHistoryItem => (
      typeof item?.id === 'number'
      && typeof item?.username === 'string'
      && typeof item?.last_login_at === 'string'
    ))
  } catch (e) {
    console.error('解析账号历史失败:', e)
    localStorage.removeItem(ACCOUNT_HISTORY_KEY)
    return []
  }
}

function isFutureDate(value?: string) {
  if (!value) return false
  const time = new Date(value).getTime()
  return Number.isFinite(time) && time > Date.now()
}

function getDefaultSwitchExpiry() {
  return new Date(Date.now() + ACCOUNT_SWITCH_DAYS * 24 * 60 * 60 * 1000).toISOString()
}

function loadAccountSessions(): Record<string, AccountSwitchSession> {
  const saved = localStorage.getItem(ACCOUNT_SESSIONS_KEY)
  if (!saved) return {}

  try {
    const parsed = JSON.parse(saved)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}

    const sessions: Record<string, AccountSwitchSession> = {}
    for (const [id, session] of Object.entries(parsed as Record<string, AccountSwitchSession>)) {
      if (
        /^\d+$/.test(id)
        && typeof session?.token === 'string'
        && typeof session?.expires_at === 'string'
        && isFutureDate(session.expires_at)
      ) {
        sessions[id] = session
      }
    }
    return sessions
  } catch (e) {
    console.error('解析账号切换会话失败:', e)
    localStorage.removeItem(ACCOUNT_SESSIONS_KEY)
    return {}
  }
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<UserData | null>(null)
  const accountHistory = ref<AccountHistoryItem[]>(loadAccountHistory())
  const accountSessions = ref<Record<string, AccountSwitchSession>>(loadAccountSessions())

  function persistUserState() {
    if (user.value) {
      localStorage.setItem('user', JSON.stringify(user.value))
    } else {
      localStorage.removeItem('user')
    }
  }

  function persistAccountHistory() {
    localStorage.setItem(ACCOUNT_HISTORY_KEY, JSON.stringify(accountHistory.value))
  }

  function persistAccountSessions() {
    localStorage.setItem(ACCOUNT_SESSIONS_KEY, JSON.stringify(accountSessions.value))
  }

  function getAccountSwitchSession(id: number) {
    const key = String(id)
    const session = accountSessions.value[key]
    if (!session) return null
    if (!isFutureDate(session.expires_at)) {
      delete accountSessions.value[key]
      persistAccountSessions()
      return null
    }
    return session
  }

  function getAccountSwitchToken(id: number) {
    return getAccountSwitchSession(id)?.token || ''
  }

  function hasValidAccountSwitchSession(id: number) {
    return Boolean(getAccountSwitchSession(id))
  }

  function rememberAccount(u: UserData, session?: AccountSwitchSession) {
    const existingSession = session || getAccountSwitchSession(u.id) || undefined
    const item: AccountHistoryItem = {
      id: u.id,
      username: u.username,
      avatar: u.avatar,
      name_color: u.name_color,
      name_bold: u.name_bold,
      forum_level: u.forum_level,
      forum_level_name: u.forum_level_name,
      forum_level_color: u.forum_level_color,
      forum_level_bold: u.forum_level_bold,
      last_login_at: new Date().toISOString(),
      session_expires_at: existingSession?.expires_at,
    }

    accountHistory.value = [
      item,
      ...accountHistory.value.filter(account => account.id !== item.id),
    ].slice(0, MAX_ACCOUNT_HISTORY)
    persistAccountHistory()
  }

  const savedUser = localStorage.getItem('user')
  if (savedUser) {
    try {
      user.value = JSON.parse(savedUser)
    } catch (e) {
      console.error('解析用户信息失败:', e)
    }
  }

  const isModerator = computed(() => {
    return user.value?.role === 'moderator' || user.value?.role === 'admin'
  })

  const isAdmin = computed(() => {
    return user.value?.role === 'admin'
  })

  function setAuth(t: string, u: UserData, options?: { switchToken?: string; switchTokenExpiresAt?: string }) {
    token.value = t
    user.value = u
    localStorage.setItem('token', t)
    persistUserState()

    let session: AccountSwitchSession | undefined
    if (options?.switchToken) {
      session = {
        token: options.switchToken,
        expires_at: isFutureDate(options.switchTokenExpiresAt) ? options.switchTokenExpiresAt! : getDefaultSwitchExpiry(),
      }
      accountSessions.value[String(u.id)] = session
      persistAccountSessions()
    }
    rememberAccount(u, session)
  }

  function updateAvatar(avatar: string) {
    if (user.value) {
      user.value.avatar = avatar
      persistUserState()
    }
  }

  function updateRole(role: string) {
    if (user.value) {
      user.value.role = role
      persistUserState()
    }
  }

  function mergeUser(patch: Partial<UserData>) {
    if (user.value) {
      user.value = { ...user.value, ...patch }
      persistUserState()
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  function removeAccountHistoryItem(id: number) {
    accountHistory.value = accountHistory.value.filter(account => account.id !== id)
    delete accountSessions.value[String(id)]
    persistAccountHistory()
    persistAccountSessions()
  }

  function clearAccountSwitchSession(id: number) {
    delete accountSessions.value[String(id)]
    accountHistory.value = accountHistory.value.map(account => (
      account.id === id ? { ...account, session_expires_at: undefined } : account
    ))
    persistAccountSessions()
    persistAccountHistory()
  }

  return {
    token,
    user,
    accountHistory,
    isModerator,
    isAdmin,
    setAuth,
    updateAvatar,
    updateRole,
    mergeUser,
    logout,
    removeAccountHistoryItem,
    clearAccountSwitchSession,
    getAccountSwitchToken,
    hasValidAccountSwitchSession,
  }
})
