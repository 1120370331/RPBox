<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Capacitor, type PluginListenerHandle } from '@capacitor/core'
import { App as CapacitorApp } from '@capacitor/app'
import { useThemeStore } from '@shared/stores/theme'
import { useUserStore } from '@shared/stores/user'
import { useToastStore } from '@shared/stores/toast'
import { useMobileUpdater } from '@/composables/useMobileUpdater'
import { resolveInAppPathFromUrl } from '@/utils/appLink'
import RToast from './components/RToast.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const themeStore = useThemeStore()
const userStore = useUserStore()
const router = useRouter()
const toast = useToastStore()
const mobileUpdater = useMobileUpdater()
let backButtonHandle: PluginListenerHandle | null = null
let appUrlOpenHandle: PluginListenerHandle | null = null
let lastBackPressAt = 0
let viewportHandler: (() => void) | null = null

function handleOffline() {
  if (!userStore.token) return
  toast.error(t('common.status.networkOffline'))
  userStore.logout()
  router.replace({ name: 'login' })
}

function isHomeRoute(path: string) {
  return ['/community', '/stories', '/market', '/guild', '/profile'].includes(path)
}

function updateViewportVariables() {
  const viewport = window.visualViewport
  const baseWidth = viewport?.width && viewport.width > 200 ? viewport.width : window.innerWidth
  const baseHeight = viewport?.height && viewport.height > 300 ? viewport.height : window.innerHeight
  const width = Math.max(240, Math.round(baseWidth))
  const height = Math.max(320, Math.round(baseHeight))
  const root = document.documentElement
  root.style.setProperty('--app-width', `${width}px`)
  root.style.setProperty('--app-height', `${height}px`)
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

async function openSharedRoute(rawUrl: string) {
  const path = resolveInAppPathFromUrl(rawUrl)
  if (!path) return
  if (router.currentRoute.value.fullPath === path) return

  try {
    await router.replace(path)
  } catch (error) {
    console.error('Failed to open shared route', error)
  }
}

async function bindAppUrlOpen() {
  if (!Capacitor.isNativePlatform()) return

  appUrlOpenHandle = await CapacitorApp.addListener('appUrlOpen', ({ url }) => {
    void openSharedRoute(url)
  })

  const launchUrl = await CapacitorApp.getLaunchUrl()
  if (launchUrl?.url) {
    await openSharedRoute(launchUrl.url)
  }
}

onMounted(() => {
  themeStore.initTheme()
  updateViewportVariables()
  viewportHandler = () => updateViewportVariables()
  window.addEventListener('resize', viewportHandler)
  window.addEventListener('orientationchange', viewportHandler)
  window.visualViewport?.addEventListener('resize', viewportHandler)

  if (!navigator.onLine && userStore.token) {
    handleOffline()
  }
  window.addEventListener('offline', handleOffline)

  mobileUpdater.autoCheckForUpdate().then((update) => {
    if (!update) return
    toast.info(t('profile.about.update.availableToast', { v: update.version }))
  })

  void bindNativeBackButton()
  void bindAppUrlOpen()
})

onBeforeUnmount(() => {
  window.removeEventListener('offline', handleOffline)
  if (viewportHandler) {
    window.removeEventListener('resize', viewportHandler)
    window.removeEventListener('orientationchange', viewportHandler)
    window.visualViewport?.removeEventListener('resize', viewportHandler)
    viewportHandler = null
  }
  backButtonHandle?.remove()
  backButtonHandle = null
  appUrlOpenHandle?.remove()
  appUrlOpenHandle = null
})
</script>

<template>
  <router-view />
  <RToast />
</template>
