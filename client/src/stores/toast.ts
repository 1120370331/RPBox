import { ref } from 'vue'
import { defineStore } from 'pinia'

export interface ToastItem {
  id: number
  type: 'success' | 'error' | 'warning' | 'info' | 'achievement'
  message: string
  title?: string
  icon?: string
  rarity?: string
}

export interface AchievementCelebrationItem {
  id: number
  title: string
  message: string
  icon?: string
  rarity?: string
  completedAt: string
}

interface AchievementMeta extends Pick<ToastItem, 'icon' | 'rarity'> {
  completedAt?: string
}

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<ToastItem[]>([])
  const activeAchievement = ref<AchievementCelebrationItem | null>(null)
  const achievementQueue: AchievementCelebrationItem[] = []
  let id = 0
  let achievementId = 0
  let achievementTransitionTimer: ReturnType<typeof setTimeout> | null = null

  function show(
    type: ToastItem['type'],
    message: string,
    duration = 3000,
    meta: Pick<ToastItem, 'title' | 'icon' | 'rarity'> = {},
  ) {
    const toast = { id: ++id, type, message, ...meta }
    toasts.value.push(toast)
    setTimeout(() => remove(toast.id), duration)
  }

  function remove(toastId: number) {
    const index = toasts.value.findIndex(t => t.id === toastId)
    if (index > -1) toasts.value.splice(index, 1)
  }

  function success(message: string, duration?: number) {
    show('success', message, duration)
  }

  function error(message: string, duration?: number) {
    show('error', message, duration)
  }

  function warning(message: string, duration?: number) {
    show('warning', message, duration)
  }

  function info(message: string, duration?: number) {
    show('info', message, duration)
  }

  function achievement(title: string, message: string, meta: AchievementMeta = {}) {
    if (activeAchievement.value?.title === title || achievementQueue.some(item => item.title === title)) return
    const completedAt = meta.completedAt && !Number.isNaN(Date.parse(meta.completedAt))
      ? new Date(meta.completedAt).toISOString()
      : new Date().toISOString()
    const item: AchievementCelebrationItem = {
      id: ++achievementId,
      title,
      message,
      icon: meta.icon,
      rarity: meta.rarity,
      completedAt,
    }
    if (activeAchievement.value || achievementTransitionTimer) {
      achievementQueue.push(item)
      return
    }
    activeAchievement.value = item
  }

  function dismissAchievement() {
    if (!activeAchievement.value) return
    activeAchievement.value = null
    const next = achievementQueue.shift()
    if (!next) return
    achievementTransitionTimer = setTimeout(() => {
      achievementTransitionTimer = null
      activeAchievement.value = next
    }, 180)
  }

  return {
    toasts,
    activeAchievement,
    show,
    remove,
    success,
    error,
    warning,
    info,
    achievement,
    dismissAchievement,
  }
})
