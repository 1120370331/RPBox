import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import i18n from './i18n'
import App from './App.vue'
import 'remixicon/fonts/remixicon.css'
import './styles/variables.css'
import './styles/animations.css'
import './styles/content.css'
import './styles/theme.css'
import { handleDesktopDeepLinkUrls } from './utils/desktopDeepLink'

function isTauriRuntime() {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
}

async function setupDesktopDeepLinks() {
  if (!isTauriRuntime()) return

  try {
    const { getCurrent, onOpenUrl } = await import('@tauri-apps/plugin-deep-link')

    const navigateFromUrls = async (urls: string[]) => {
      try {
        await handleDesktopDeepLinkUrls(urls, router)
      } catch (error) {
        console.error('Failed to handle desktop deep link', error)
      }
    }

    const currentUrls = await getCurrent()
    if (currentUrls?.length) {
      await navigateFromUrls(currentUrls)
    }

    await onOpenUrl((urls) => {
      void navigateFromUrls(urls)
    })
  } catch (error) {
    console.error('Failed to initialize desktop deep links', error)
  }
}

async function bootstrap() {
  const app = createApp(App)
  app.use(createPinia())
  app.use(i18n)
  app.use(router)
  app.mount('#app')

  await router.isReady()
  await setupDesktopDeepLinks()
}

void bootstrap()
