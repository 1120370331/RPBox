import { ref, createApp, h } from 'vue'

interface DialogOptions {
  title?: string
  message: string
  type?: 'info' | 'success' | 'warning' | 'error'
  confirmText?: string
  cancelText?: string
  showCancel?: boolean
}

interface DialogState {
  visible: boolean
  options: DialogOptions
  resolve: ((value: boolean) => void) | null
}

const state = ref<DialogState>({
  visible: false,
  options: { message: '' },
  resolve: null,
})

export function useDialog() {
  function show(options: DialogOptions): Promise<boolean> {
    return new Promise((resolve) => {
      state.value = {
        visible: true,
        options,
        resolve,
      }
    })
  }

  function confirm(options: DialogOptions | string): Promise<boolean> {
    const opts = typeof options === 'string' ? { message: options } : options
    return show({ showCancel: true, confirmText: '确定', cancelText: '取消', ...opts })
  }

  function alert(options: DialogOptions | string): Promise<boolean> {
    const opts = typeof options === 'string' ? { message: options } : options
    return show({ showCancel: false, confirmText: '确定', ...opts })
  }

  function close(result: boolean) {
    if (state.value.resolve) {
      state.value.resolve(result)
    }
    state.value.visible = false
    state.value.resolve = null
  }

  return {
    state,
    confirm,
    alert,
    close,
  }
}

// 单例导出
export const dialog = useDialog()
