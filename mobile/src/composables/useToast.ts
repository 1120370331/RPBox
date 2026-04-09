import { useToastStore } from '@shared/stores/toast'

export function useToast() {
  const toast = useToastStore()
  return {
    success: toast.success,
    error: toast.error,
    warning: toast.warning,
    info: toast.info,
  }
}
