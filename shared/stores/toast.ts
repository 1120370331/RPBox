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

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<ToastItem[]>([])
  let id = 0

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

  function success(message: string, duration?: number) { show('success', message, duration) }
  function error(message: string, duration?: number) { show('error', message, duration) }
  function warning(message: string, duration?: number) { show('warning', message, duration) }
  function info(message: string, duration?: number) { show('info', message, duration) }
  function achievement(title: string, message: string, meta: Pick<ToastItem, 'icon' | 'rarity'> = {}, duration = 7000) {
    show('achievement', message, duration, { ...meta, title })
  }

  return { toasts, show, remove, success, error, warning, info, achievement }
})
