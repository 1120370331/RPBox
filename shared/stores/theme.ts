import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface ThemeColors {
  primary: string
  primaryHover: string
  primaryLight: string
  secondary: string
  secondaryHover: string
  accent: string
  accentHover: string
  background: string
  highlight: string
  sidebarBg: string
  sidebarText: string
  sidebarTextMuted: string
  sidebarHover: string
  panelBg: string
  cardBg: string
  cardBgHover: string
  textMain: string
  textLight: string
  textSecondary: string
  textMuted: string
  border: string
  borderLight: string
  borderHover: string
  btnPrimaryBg: string
  btnPrimaryText: string
  btnPrimaryHover: string
  btnSecondaryBg: string
  btnSecondaryText: string
  btnSecondaryHover: string
  btnOutlineBorder: string
  btnOutlineText: string
  btnOutlineHover: string
  btnDangerBg: string
  btnDangerHover: string
  inputBg: string
  inputBorder: string
  inputFocus: string
  inputPlaceholder: string
  switchActive: string
  switchInactive: string
  checkboxActive: string
  tagBg: string
  tagText: string
  badgeBg: string
  shadowBase: string
  scrollbarThumb: string
  scrollbarThumbHover: string
  iconBg: string
  iconColor: string
  linkColor: string
  linkHover: string
  gradientStart: string
  gradientEnd: string
  success: string
  successLight: string
  warning: string
  warningLight: string
  warningBorder: string
  warningDark: string
}

export interface Theme {
  id: string
  name: string
  colors: ThemeColors
}

const classicTheme: Theme = {
  id: 'classic',
  name: '经典咖啡',
  colors: {
    primary: '#4B3621', primaryHover: '#3D2C1B', primaryLight: 'rgba(75, 54, 33, 0.1)',
    secondary: '#804030', secondaryHover: '#6B3528',
    accent: '#B87333', accentHover: '#A66329',
    background: '#EED9C4', highlight: '#D2691E',
    sidebarBg: '#4B3621', sidebarText: '#FBF5EF',
    sidebarTextMuted: 'rgba(251, 245, 239, 0.7)', sidebarHover: 'rgba(238, 217, 196, 0.1)',
    panelBg: '#FFFFFF', cardBg: '#FDFBF9', cardBgHover: 'rgba(128, 64, 48, 0.05)',
    textMain: '#2C1810', textLight: '#FBF5EF', textSecondary: '#8C7B70', textMuted: 'rgba(75, 54, 33, 0.5)',
    border: '#E8DCCF', borderLight: '#F0E6DC', borderHover: '#D4A373',
    btnPrimaryBg: '#804030', btnPrimaryText: '#FFFFFF', btnPrimaryHover: '#6B3528',
    btnSecondaryBg: 'rgba(128, 64, 48, 0.1)', btnSecondaryText: '#804030', btnSecondaryHover: 'rgba(128, 64, 48, 0.15)',
    btnOutlineBorder: '#E8DCCF', btnOutlineText: '#2C1810', btnOutlineHover: 'rgba(128, 64, 48, 0.05)',
    btnDangerBg: '#DC3545', btnDangerHover: '#C82333',
    inputBg: '#FDFBF9', inputBorder: '#E8DCCF', inputFocus: '#804030', inputPlaceholder: '#8C7B70',
    switchActive: '#804030', switchInactive: '#D4D4D4', checkboxActive: '#804030',
    tagBg: 'rgba(128, 64, 48, 0.1)', tagText: '#804030', badgeBg: '#804030',
    shadowBase: '75, 54, 33',
    scrollbarThumb: '#DCC8B8', scrollbarThumbHover: '#BFA08A',
    iconBg: 'rgba(128, 64, 48, 0.1)', iconColor: '#804030',
    linkColor: '#B87333', linkHover: '#D2691E',
    gradientStart: '#804030', gradientEnd: '#4B3621',
    success: '#5B8C5A', successLight: 'rgba(91, 140, 90, 0.15)',
    warning: '#E6A23C', warningLight: 'rgba(230, 162, 60, 0.15)',
    warningBorder: 'rgba(217, 119, 6, 0.35)', warningDark: '#D97706',
  },
}

const tealTheme: Theme = {
  id: 'teal',
  name: '翡翠青绿',
  colors: {
    primary: '#126260', primaryHover: '#0e4f4d', primaryLight: 'rgba(18, 98, 96, 0.1)',
    secondary: '#49b5a4', secondaryHover: '#3a9a8b',
    accent: '#49b5a4', accentHover: '#3a9a8b',
    background: '#d8ebe7', highlight: '#49b5a4',
    sidebarBg: '#092a30', sidebarText: '#e8f5f3',
    sidebarTextMuted: 'rgba(232, 245, 243, 0.7)', sidebarHover: 'rgba(106, 233, 213, 0.15)',
    panelBg: '#f0f9f7', cardBg: '#e8f5f3', cardBgHover: 'rgba(73, 181, 164, 0.12)',
    textMain: '#0a3538', textLight: '#e8f5f3', textSecondary: '#3d6663', textMuted: 'rgba(10, 53, 56, 0.55)',
    border: '#a8d8d0', borderLight: '#c5e8e2', borderHover: '#49b5a4',
    btnPrimaryBg: '#126260', btnPrimaryText: '#FFFFFF', btnPrimaryHover: '#0e4f4d',
    btnSecondaryBg: 'rgba(73, 181, 164, 0.15)', btnSecondaryText: '#126260', btnSecondaryHover: 'rgba(73, 181, 164, 0.25)',
    btnOutlineBorder: '#c5ebe4', btnOutlineText: '#092a30', btnOutlineHover: 'rgba(73, 181, 164, 0.1)',
    btnDangerBg: '#DC3545', btnDangerHover: '#C82333',
    inputBg: '#f5fdfb', inputBorder: '#c5ebe4', inputFocus: '#49b5a4', inputPlaceholder: '#4a7a75',
    switchActive: '#49b5a4', switchInactive: '#D4D4D4', checkboxActive: '#49b5a4',
    tagBg: 'rgba(73, 181, 164, 0.15)', tagText: '#126260', badgeBg: '#126260',
    shadowBase: '9, 42, 48',
    scrollbarThumb: '#a8ddd4', scrollbarThumbHover: '#6ae9d5',
    iconBg: 'rgba(73, 181, 164, 0.15)', iconColor: '#126260',
    linkColor: '#49b5a4', linkHover: '#6ae9d5',
    gradientStart: '#126260', gradientEnd: '#092a30',
    success: '#3d9970', successLight: 'rgba(61, 153, 112, 0.15)',
    warning: '#f0ad4e', warningLight: 'rgba(240, 173, 78, 0.15)',
    warningBorder: 'rgba(240, 173, 78, 0.35)', warningDark: '#ec971f',
  },
}

export const themes: Theme[] = [classicTheme, tealTheme]

export function getThemeById(id: string): Theme {
  return themes.find((t) => t.id === id) || classicTheme
}

export const useThemeStore = defineStore('theme', () => {
  const currentThemeId = ref(localStorage.getItem('theme') || 'classic')
  const currentTheme = computed(() => getThemeById(currentThemeId.value))

  function setTheme(themeId: string) {
    const theme = getThemeById(themeId)
    currentThemeId.value = theme.id
    localStorage.setItem('theme', theme.id)
    applyTheme(theme)
  }

  function applyTheme(theme: Theme) {
    const root = document.documentElement
    const c = theme.colors
    const map: Record<string, string> = {
      '--color-primary': c.primary, '--color-primary-hover': c.primaryHover,
      '--color-primary-light': c.primaryLight, '--color-secondary': c.secondary,
      '--color-secondary-hover': c.secondaryHover, '--color-accent': c.accent,
      '--color-accent-hover': c.accentHover, '--color-background': c.background,
      '--color-highlight': c.highlight,
      '--color-sidebar-bg': c.sidebarBg, '--color-sidebar-text': c.sidebarText,
      '--color-sidebar-text-muted': c.sidebarTextMuted, '--color-sidebar-hover': c.sidebarHover,
      '--color-panel-bg': c.panelBg, '--color-main-bg': c.background,
      '--color-card-bg': c.cardBg, '--color-card-bg-hover': c.cardBgHover,
      '--color-text-main': c.textMain, '--color-text-light': c.textLight,
      '--color-text-secondary': c.textSecondary, '--color-text-muted': c.textMuted,
      '--text-light': c.textLight, '--text-dark': c.textMain,
      '--color-border': c.border, '--color-border-light': c.borderLight,
      '--color-border-hover': c.borderHover,
      '--btn-primary-bg': c.btnPrimaryBg, '--btn-primary-text': c.btnPrimaryText,
      '--btn-primary-hover': c.btnPrimaryHover,
      '--btn-secondary-bg': c.btnSecondaryBg, '--btn-secondary-text': c.btnSecondaryText,
      '--btn-secondary-hover': c.btnSecondaryHover,
      '--btn-outline-border': c.btnOutlineBorder, '--btn-outline-text': c.btnOutlineText,
      '--btn-outline-hover': c.btnOutlineHover,
      '--btn-danger-bg': c.btnDangerBg, '--btn-danger-hover': c.btnDangerHover,
      '--input-bg': c.inputBg, '--input-border': c.inputBorder,
      '--input-focus': c.inputFocus, '--input-placeholder': c.inputPlaceholder,
      '--switch-active': c.switchActive, '--switch-inactive': c.switchInactive,
      '--checkbox-active': c.checkboxActive,
      '--tag-bg': c.tagBg, '--tag-text': c.tagText, '--badge-bg': c.badgeBg,
      '--shadow-base': c.shadowBase,
      '--shadow-sm': `0 2px 8px rgba(${c.shadowBase}, 0.05)`,
      '--shadow-md': `0 4px 20px rgba(${c.shadowBase}, 0.08)`,
      '--shadow-lg': `0 8px 32px rgba(${c.shadowBase}, 0.12)`,
      '--scrollbar-thumb': c.scrollbarThumb, '--scrollbar-thumb-hover': c.scrollbarThumbHover,
      '--icon-bg': c.iconBg, '--icon-color': c.iconColor,
      '--link-color': c.linkColor, '--link-hover': c.linkHover,
      '--gradient-start': c.gradientStart, '--gradient-end': c.gradientEnd,
      '--color-success': c.success, '--color-success-light': c.successLight,
      '--color-warning': c.warning, '--color-warning-light': c.warningLight,
      '--color-warning-border': c.warningBorder, '--color-warning-dark': c.warningDark,
    }
    for (const [prop, val] of Object.entries(map)) {
      root.style.setProperty(prop, val)
    }
  }

  function initTheme() {
    applyTheme(currentTheme.value)
  }

  return { currentThemeId, currentTheme, setTheme, initTheme }
})
