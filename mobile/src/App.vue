<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Capacitor } from '@capacitor/core'
import { App as CapacitorApp, type PluginListenerHandle } from '@capacitor/app'
import { useThemeStore } from '@shared/stores/theme'
import { useUserStore } from '@shared/stores/user'
import { useToastStore } from '@shared/stores/toast'
import { useMobileUpdater } from '@/composables/useMobileUpdater'
import RToast from './components/RToast.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const themeStore = useThemeStore()
const userStore = useUserStore()
const router = useRouter()
const toast = useToastStore()
const mobileUpdater = useMobileUpdater()
let backButtonHandle: PluginListenerHandle | null = null
let lastBackPressAt = 0

function handleOffline() {
  if (!userStore.token) return
  toast.error(t('common.status.networkOffline'))
  userStore.logout()
  router.replace({ name: 'login' })
}

function isHomeRoute(path: string) {
  return ['/community', '/stories', '/market', '/guild', '/profile'].includes(path)
}

async function bindNativeBackButton() {
  if (!Capacitor.isNativePlatform()) return
  backButtonHandle = await CapacitorApp.addListener('backButton', ({ canGoBack }) => {
    const path = router.currentRoute.value.path

    if (isHomeRoute(path)) {
      const now = Date.now()
      if (now - lastBackPressAt < 1200) {
        void CapacitorApp.exitApp()
        return
      }
      lastBackPressAt = now
      toast.info(t('common.status.pressAgainExit'))
      return
    }

    if (canGoBack) {
      window.history.back()
      return
    }
    router.replace('/community')
  })
}

onMounted(() => {
  themeStore.initTheme()
  if (!navigator.onLine && userStore.token) {
    handleOffline()
  }
  window.addEventListener('offline', handleOffline)

  mobileUpdater.autoCheckForUpdate().then((update) => {
    if (!update) return
    toast.info(t('profile.about.update.availableToast', { v: update.version }))
  })

  void bindNativeBackButton()
})

onBeforeUnmount(() => {
  window.removeEventListener('offline', handleOffline)
  backButtonHandle?.remove()
  backButtonHandle = null
})
</script>

<template>
  <router-view />
  <RToast />
</template>
