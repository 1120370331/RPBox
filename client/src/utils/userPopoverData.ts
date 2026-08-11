import { listUserCharacterCards, type CharacterCardSummary } from '@/api/characterCard'
import { getUserProfile, type PublicUserProfile } from '@/api/user'

export interface UserPopoverData {
  profile: PublicUserProfile
  characterCards: CharacterCardSummary[]
}

interface CachedUserPopoverData {
  data: UserPopoverData
  expiresAt: number
}

const cacheTTL = 60_000
const resolvedCache = new Map<number, CachedUserPopoverData>()
const pendingCache = new Map<number, Promise<UserPopoverData>>()

export async function loadUserPopoverData(userId: number): Promise<UserPopoverData> {
  const resolved = resolvedCache.get(userId)
  if (resolved && resolved.expiresAt > Date.now()) return resolved.data
  if (resolved) resolvedCache.delete(userId)

  const pending = pendingCache.get(userId)
  if (pending) return pending

  const request = Promise.all([
    getUserProfile(userId),
    listUserCharacterCards(userId),
  ]).then(([profile, cards]) => {
    const data = {
      profile,
      characterCards: cards.character_cards,
    }
    resolvedCache.set(userId, { data, expiresAt: Date.now() + cacheTTL })
    pendingCache.delete(userId)
    return data
  }).catch((error) => {
    pendingCache.delete(userId)
    throw error
  })

  pendingCache.set(userId, request)
  return request
}

export function clearUserPopoverDataCache() {
  resolvedCache.clear()
  pendingCache.clear()
}
