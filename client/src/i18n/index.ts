import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

// 检测系统语言
function getSystemLocale(): string {
  const lang = navigator.language || (navigator as any).userLanguage || 'zh-CN'
  // 如果是英文系统，返回 en-US，否则默认中文
  if (lang.startsWith('en')) {
    return 'en-US'
  }
  return 'zh-CN'
}

// 获取存储的语言偏好，如果没有则使用系统语言
function getStoredLocale(): string {
  const stored = localStorage.getItem('locale')
  if (stored && ['zh-CN', 'en-US'].includes(stored)) {
    return stored
  }
  return getSystemLocale()
}

const i18n = createI18n({
  legacy: false, // 使用 Composition API 模式
  locale: getStoredLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

export default i18n
