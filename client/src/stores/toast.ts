import { ref } from 'vue'
import { defineStore } from 'pinia'

export interface ToastItem {
  id: number
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
}

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<ToastItem[]>([])
  let id = 0

  function show(type: ToastItem['type'], message: string, duration = 3000) {
    const toast = { id: ++id, type, message }
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

  return {
    toasts,
    show,
    remove,
    success,
    error,
    warning,
    info,
  }
})
