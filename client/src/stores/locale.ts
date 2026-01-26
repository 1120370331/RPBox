import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

export type LocaleType = 'zh-CN' | 'en-US'

export const useLocaleStore = defineStore('locale', () => {
  const currentLocale = ref<LocaleType>(
    (localStorage.getItem('locale') as LocaleType) || 'zh-CN'
  )

  function setLocale(locale: LocaleType) {
    currentLocale.value = locale
    localStorage.setItem('locale', locale)
  }

  return {
    currentLocale,
    setLocale,
  }
})
