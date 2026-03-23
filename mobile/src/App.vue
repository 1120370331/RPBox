<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
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

function handleOffline() {
  if (!userStore.token) return
  toast.error(t('common.status.networkOffline'))
  userStore.logout()
  router.replace({ name: 'login' })
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
})
</script>

<template>
  <router-view />
  <RToast />
</template>
