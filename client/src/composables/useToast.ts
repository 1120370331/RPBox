import { createApp, h } from 'vue'
import Toast from '@/components/Toast.vue'

type ToastType = 'success' | 'error' | 'info'

interface ToastOptions {
  message: string
  type?: ToastType
  duration?: number
}

export function useToast() {
  function showToast(options: ToastOptions | string) {
    const opts = typeof options === 'string' ? { message: options } : options

    const container = document.createElement('div')
    document.body.appendChild(container)

    const app = createApp({
      render() {
        return h(Toast, {
          message: opts.message,
          type: opts.type || 'success',
          duration: opts.duration || 2000,
          onClose: () => {
            app.unmount()
            container.remove()
          }
        })
      }
    })

    app.mount(container)
  }

  return {
    success: (msg: string) => showToast({ message: msg, type: 'success' }),
    error: (msg: string) => showToast({ message: msg, type: 'error' }),
    info: (msg: string) => showToast({ message: msg, type: 'info' }),
    show: showToast
  }
}
