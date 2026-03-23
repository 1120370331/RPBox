import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

function getSystemLocale(): string {
  const lang = navigator.language || (navigator as any).userLanguage || 'zh-CN'
  if (lang.startsWith('en')) return 'en-US'
  return 'zh-CN'
}

function getStoredLocale(): string {
  const stored = localStorage.getItem('locale')
  if (stored && ['zh-CN', 'en-US'].includes(stored)) return stored
  return getSystemLocale()
}

const i18n = createI18n({
  legacy: false,
  locale: getStoredLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

export default i18n
