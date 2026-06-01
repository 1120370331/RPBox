<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Capacitor, type PluginListenerHandle } from '@capacitor/core'
import { App as CapacitorApp } from '@capacitor/app'
import { Keyboard, KeyboardResize } from '@capacitor/keyboard'
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
let keyboardWillShowHandle: PluginListenerHandle | null = null
let keyboardDidShowHandle: PluginListenerHandle | null = null
let keyboardWillHideHandle: PluginListenerHandle | null = null
let keyboardDidHideHandle: PluginListenerHandle | null = null
let keyboardVisible = false
let stableViewportWidth = 0
let stableViewportHeight = 0
let viewportRestoreTimers: Array<ReturnType<typeof setTimeout>> = []
let focusInHandler: ((event: FocusEvent) => void) | null = null
let focusOutHandler: ((event: FocusEvent) => void) | null = null

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
  const visualWidth = viewport?.width && viewport.width > 200 ? viewport.width : window.innerWidth
  const visualHeight = viewport?.height && viewport.height > 300 ? viewport.height : window.innerHeight
  const layoutWidth = window.innerWidth || visualWidth
  const layoutHeight = window.innerHeight || visualHeight
  const baseWidth = Math.max(visualWidth, layoutWidth)
  const baseHeight = keyboardVisible
    ? Math.max(visualHeight, 320)
    : Math.max(visualHeight, layoutHeight, stableViewportHeight)
  const width = Math.max(240, Math.round(baseWidth))
  const height = Math.max(320, Math.round(baseHeight))
  if (!keyboardVisible) {
    stableViewportWidth = Math.max(stableViewportWidth, width)
    stableViewportHeight = Math.max(stableViewportHeight, height)
  }
  const root = document.documentElement
  root.style.setProperty('--app-width', `${width}px`)
  root.style.setProperty('--app-height', `${height}px`)
}

function scheduleViewportRestore() {
  viewportRestoreTimers.forEach((timer) => clearTimeout(timer))
  viewportRestoreTimers = []

  const restore = () => {
    keyboardVisible = false
    document.body.style.height = ''
    document.documentElement.style.height = ''
    updateViewportVariables()
  }

  restore()
  viewportRestoreTimers = [60, 180, 360].map((delay) => {
    return setTimeout(restore, delay)
  })
}

function isEditableTarget(target: EventTarget | null) {
  const element = target instanceof HTMLElement ? target : null
  if (!element) return false
  return Boolean(element.closest('input, textarea, [contenteditable="true"]'))
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

async function bindKeyboardHandlers() {
  if (!Capacitor.isNativePlatform()) return

  try {
    await Keyboard.setResizeMode({ mode: KeyboardResize.Native })
  } catch {
    // Android ignores this API; config still applies resizeOnFullScreen there.
  }

  keyboardWillShowHandle = await Keyboard.addListener('keyboardWillShow', () => {
    keyboardVisible = true
    updateViewportVariables()
  })
  keyboardDidShowHandle = await Keyboard.addListener('keyboardDidShow', () => {
    keyboardVisible = true
    updateViewportVariables()
  })
  keyboardWillHideHandle = await Keyboard.addListener('keyboardWillHide', scheduleViewportRestore)
  keyboardDidHideHandle = await Keyboard.addListener('keyboardDidHide', scheduleViewportRestore)
}

onMounted(() => {
  themeStore.initTheme()
  updateViewportVariables()
  viewportHandler = () => updateViewportVariables()
  window.addEventListener('resize', viewportHandler)
  window.addEventListener('orientationchange', viewportHandler)
  window.visualViewport?.addEventListener('resize', viewportHandler)
  window.visualViewport?.addEventListener('scroll', viewportHandler)
  focusInHandler = (event: FocusEvent) => {
    if (!isEditableTarget(event.target)) return
    keyboardVisible = true
    updateViewportVariables()
  }
  focusOutHandler = (event: FocusEvent) => {
    if (!isEditableTarget(event.target)) return
    scheduleViewportRestore()
  }
  document.addEventListener('focusin', focusInHandler)
  document.addEventListener('focusout', focusOutHandler)

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
  void bindKeyboardHandlers()
})

onBeforeUnmount(() => {
  window.removeEventListener('offline', handleOffline)
  if (viewportHandler) {
    window.removeEventListener('resize', viewportHandler)
    window.removeEventListener('orientationchange', viewportHandler)
    window.visualViewport?.removeEventListener('resize', viewportHandler)
    window.visualViewport?.removeEventListener('scroll', viewportHandler)
    viewportHandler = null
  }
  viewportRestoreTimers.forEach((timer) => clearTimeout(timer))
  viewportRestoreTimers = []
  if (focusInHandler) {
    document.removeEventListener('focusin', focusInHandler)
    focusInHandler = null
  }
  if (focusOutHandler) {
    document.removeEventListener('focusout', focusOutHandler)
    focusOutHandler = null
  }
  backButtonHandle?.remove()
  backButtonHandle = null
  appUrlOpenHandle?.remove()
  appUrlOpenHandle = null
  keyboardWillShowHandle?.remove()
  keyboardWillShowHandle = null
  keyboardDidShowHandle?.remove()
  keyboardDidShowHandle = null
  keyboardWillHideHandle?.remove()
  keyboardWillHideHandle = null
  keyboardDidHideHandle?.remove()
  keyboardDidHideHandle = null
})
</script>

<template>
  <router-view />
  <RToast />
</template>
