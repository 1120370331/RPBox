import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

// 主题类型定义
export interface ThemeColors {
  // 主色调
  primary: string
  primaryHover: string
  primaryLight: string
  secondary: string
  secondaryHover: string
  accent: string
  accentHover: string
  background: string
  highlight: string

  // 侧边栏
  sidebarBg: string
  sidebarText: string
  sidebarTextMuted: string
  sidebarHover: string

  // 面板/卡片
  panelBg: string
  cardBg: string
  cardBgHover: string

  // 文字
  textMain: string
  textLight: string
  textSecondary: string
  textMuted: string

  // 边框
  border: string
  borderLight: string
  borderHover: string

  // 按钮
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

  // 输入框
  inputBg: string
  inputBorder: string
  inputFocus: string
  inputPlaceholder: string

  // 开关/选中状态
  switchActive: string
  switchInactive: string
  checkboxActive: string

  // 标签/徽章
  tagBg: string
  tagText: string
  badgeBg: string

  // 阴影基色 (rgba 格式)
  shadowBase: string

  // 滚动条
  scrollbarThumb: string
  scrollbarThumbHover: string

  // 特殊元素
  iconBg: string
  iconColor: string
  linkColor: string
  linkHover: string

  // 渐变
  gradientStart: string
  gradientEnd: string

  // 状态色
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

// 默认主题 - 经典咖啡色
const classicTheme: Theme = {
  id: 'classic',
  name: '经典咖啡',
  colors: {
    // 主色调
    primary: '#4B3621',
    primaryHover: '#3D2C1B',
    primaryLight: 'rgba(75, 54, 33, 0.1)',
    secondary: '#804030',
    secondaryHover: '#6B3528',
    accent: '#B87333',
    accentHover: '#A66329',
    background: '#EED9C4',
    highlight: '#D2691E',

    // 侧边栏
    sidebarBg: '#4B3621',
    sidebarText: '#FBF5EF',
    sidebarTextMuted: 'rgba(251, 245, 239, 0.7)',
    sidebarHover: 'rgba(238, 217, 196, 0.1)',

    // 面板/卡片
    panelBg: '#FFFFFF',
    cardBg: '#FDFBF9',
    cardBgHover: 'rgba(128, 64, 48, 0.05)',

    // 文字
    textMain: '#2C1810',
    textLight: '#FBF5EF',
    textSecondary: '#8C7B70',
    textMuted: 'rgba(75, 54, 33, 0.5)',

    // 边框
    border: '#E8DCCF',
    borderLight: '#F0E6DC',
    borderHover: '#D4A373',

    // 按钮
    btnPrimaryBg: '#804030',
    btnPrimaryText: '#FFFFFF',
    btnPrimaryHover: '#6B3528',
    btnSecondaryBg: 'rgba(128, 64, 48, 0.1)',
    btnSecondaryText: '#804030',
    btnSecondaryHover: 'rgba(128, 64, 48, 0.15)',
    btnOutlineBorder: '#E8DCCF',
    btnOutlineText: '#2C1810',
    btnOutlineHover: 'rgba(128, 64, 48, 0.05)',
    btnDangerBg: '#DC3545',
    btnDangerHover: '#C82333',

    // 输入框
    inputBg: '#FDFBF9',
    inputBorder: '#E8DCCF',
    inputFocus: '#804030',
    inputPlaceholder: '#8C7B70',

    // 开关/选中状态
    switchActive: '#804030',
    switchInactive: '#D4D4D4',
    checkboxActive: '#804030',

    // 标签/徽章
    tagBg: 'rgba(128, 64, 48, 0.1)',
    tagText: '#804030',
    badgeBg: '#804030',

    // 阴影基色
    shadowBase: '75, 54, 33',

    // 滚动条
    scrollbarThumb: '#DCC8B8',
    scrollbarThumbHover: '#BFA08A',

    // 特殊元素
    iconBg: 'rgba(128, 64, 48, 0.1)',
    iconColor: '#804030',
    linkColor: '#B87333',
    linkHover: '#D2691E',

    // 渐变
    gradientStart: '#804030',
    gradientEnd: '#4B3621',

    // 状态色
    success: '#5B8C5A',
    successLight: 'rgba(91, 140, 90, 0.15)',
    warning: '#E6A23C',
    warningLight: 'rgba(230, 162, 60, 0.15)',
    warningBorder: 'rgba(217, 119, 6, 0.35)',
    warningDark: '#D97706',
  },
}

// 青绿色主题
// 配色: #fafeff(最浅) #6ae9d5(浅) #49b5a4(中) #126260(深) #092a30(最深)
const tealTheme: Theme = {
  id: 'teal',
  name: '翡翠青绿',
  colors: {
    // 主色调
    primary: '#126260',
    primaryHover: '#0e4f4d',
    primaryLight: 'rgba(18, 98, 96, 0.1)',
    secondary: '#49b5a4',
    secondaryHover: '#3a9a8b',
    accent: '#49b5a4',
    accentHover: '#3a9a8b',
    background: '#d8ebe7',
    highlight: '#49b5a4',

    // 侧边栏
    sidebarBg: '#092a30',
    sidebarText: '#e8f5f3',
    sidebarTextMuted: 'rgba(232, 245, 243, 0.7)',
    sidebarHover: 'rgba(106, 233, 213, 0.15)',

    // 面板/卡片
    panelBg: '#f0f9f7',
    cardBg: '#e8f5f3',
    cardBgHover: 'rgba(73, 181, 164, 0.12)',

    // 文字
    textMain: '#0a3538',
    textLight: '#e8f5f3',
    textSecondary: '#3d6663',
    textMuted: 'rgba(10, 53, 56, 0.55)',

    // 边框
    border: '#a8d8d0',
    borderLight: '#c5e8e2',
    borderHover: '#49b5a4',

    // 按钮
    btnPrimaryBg: '#126260',
    btnPrimaryText: '#FFFFFF',
    btnPrimaryHover: '#0e4f4d',
    btnSecondaryBg: 'rgba(73, 181, 164, 0.15)',
    btnSecondaryText: '#126260',
    btnSecondaryHover: 'rgba(73, 181, 164, 0.25)',
    btnOutlineBorder: '#c5ebe4',
    btnOutlineText: '#092a30',
    btnOutlineHover: 'rgba(73, 181, 164, 0.1)',
    btnDangerBg: '#DC3545',
    btnDangerHover: '#C82333',

    // 输入框
    inputBg: '#f5fdfb',
    inputBorder: '#c5ebe4',
    inputFocus: '#49b5a4',
    inputPlaceholder: '#4a7a75',

    // 开关/选中状态
    switchActive: '#49b5a4',
    switchInactive: '#D4D4D4',
    checkboxActive: '#49b5a4',

    // 标签/徽章
    tagBg: 'rgba(73, 181, 164, 0.15)',
    tagText: '#126260',
    badgeBg: '#126260',

    // 阴影基色
    shadowBase: '9, 42, 48',

    // 滚动条
    scrollbarThumb: '#a8ddd4',
    scrollbarThumbHover: '#6ae9d5',

    // 特殊元素
    iconBg: 'rgba(73, 181, 164, 0.15)',
    iconColor: '#126260',
    linkColor: '#49b5a4',
    linkHover: '#6ae9d5',

    // 渐变
    gradientStart: '#126260',
    gradientEnd: '#092a30',

    // 状态色
    success: '#3d9970',
    successLight: 'rgba(61, 153, 112, 0.15)',
    warning: '#f0ad4e',
    warningLight: 'rgba(240, 173, 78, 0.15)',
    warningBorder: 'rgba(240, 173, 78, 0.35)',
    warningDark: '#ec971f',
  },
}

// 所有可用主题
export const themes: Theme[] = [classicTheme, tealTheme]

// 获取主题 by ID
export function getThemeById(id: string): Theme {
  return themes.find((t) => t.id === id) || classicTheme
}

export const useThemeStore = defineStore('theme', () => {
  // 当前主题 ID
  const currentThemeId = ref(localStorage.getItem('theme') || 'classic')

  // 当前主题对象
  const currentTheme = computed(() => getThemeById(currentThemeId.value))

  // 切换主题
  function setTheme(themeId: string) {
    const theme = getThemeById(themeId)
    currentThemeId.value = theme.id
    localStorage.setItem('theme', theme.id)
    applyTheme(theme)
  }

  // 应用主题到 DOM
  function applyTheme(theme: Theme) {
    const root = document.documentElement
    const c = theme.colors

    // 主色调
    root.style.setProperty('--color-primary', c.primary)
    root.style.setProperty('--color-primary-hover', c.primaryHover)
    root.style.setProperty('--color-primary-light', c.primaryLight)
    root.style.setProperty('--color-secondary', c.secondary)
    root.style.setProperty('--color-secondary-hover', c.secondaryHover)
    root.style.setProperty('--color-accent', c.accent)
    root.style.setProperty('--color-accent-hover', c.accentHover)
    root.style.setProperty('--color-background', c.background)
    root.style.setProperty('--color-highlight', c.highlight)

    // 侧边栏
    root.style.setProperty('--color-sidebar-bg', c.sidebarBg)
    root.style.setProperty('--color-sidebar-text', c.sidebarText)
    root.style.setProperty('--color-sidebar-text-muted', c.sidebarTextMuted)
    root.style.setProperty('--color-sidebar-hover', c.sidebarHover)

    // 面板/卡片
    root.style.setProperty('--color-panel-bg', c.panelBg)
    root.style.setProperty('--color-main-bg', c.background)
    root.style.setProperty('--color-card-bg', c.cardBg)
    root.style.setProperty('--color-card-bg-hover', c.cardBgHover)

    // 文字
    root.style.setProperty('--color-text-main', c.textMain)
    root.style.setProperty('--color-text-light', c.textLight)
    root.style.setProperty('--color-text-secondary', c.textSecondary)
    root.style.setProperty('--color-text-muted', c.textMuted)
    root.style.setProperty('--text-light', c.textLight)
    root.style.setProperty('--text-dark', c.textMain)

    // 边框
    root.style.setProperty('--color-border', c.border)
    root.style.setProperty('--color-border-light', c.borderLight)
    root.style.setProperty('--color-border-hover', c.borderHover)

    // 按钮
    root.style.setProperty('--btn-primary-bg', c.btnPrimaryBg)
    root.style.setProperty('--btn-primary-text', c.btnPrimaryText)
    root.style.setProperty('--btn-primary-hover', c.btnPrimaryHover)
    root.style.setProperty('--btn-secondary-bg', c.btnSecondaryBg)
    root.style.setProperty('--btn-secondary-text', c.btnSecondaryText)
    root.style.setProperty('--btn-secondary-hover', c.btnSecondaryHover)
    root.style.setProperty('--btn-outline-border', c.btnOutlineBorder)
    root.style.setProperty('--btn-outline-text', c.btnOutlineText)
    root.style.setProperty('--btn-outline-hover', c.btnOutlineHover)
    root.style.setProperty('--btn-danger-bg', c.btnDangerBg)
    root.style.setProperty('--btn-danger-hover', c.btnDangerHover)

    // 输入框
    root.style.setProperty('--input-bg', c.inputBg)
    root.style.setProperty('--input-border', c.inputBorder)
    root.style.setProperty('--input-focus', c.inputFocus)
    root.style.setProperty('--input-placeholder', c.inputPlaceholder)

    // 开关/选中状态
    root.style.setProperty('--switch-active', c.switchActive)
    root.style.setProperty('--switch-inactive', c.switchInactive)
    root.style.setProperty('--checkbox-active', c.checkboxActive)

    // 标签/徽章
    root.style.setProperty('--tag-bg', c.tagBg)
    root.style.setProperty('--tag-text', c.tagText)
    root.style.setProperty('--badge-bg', c.badgeBg)

    // 阴影
    root.style.setProperty('--shadow-base', c.shadowBase)
    root.style.setProperty('--shadow-sm', `0 2px 8px rgba(${c.shadowBase}, 0.05)`)
    root.style.setProperty('--shadow-md', `0 4px 20px rgba(${c.shadowBase}, 0.08)`)
    root.style.setProperty('--shadow-lg', `0 8px 32px rgba(${c.shadowBase}, 0.12)`)

    // 滚动条
    root.style.setProperty('--scrollbar-thumb', c.scrollbarThumb)
    root.style.setProperty('--scrollbar-thumb-hover', c.scrollbarThumbHover)

    // 特殊元素
    root.style.setProperty('--icon-bg', c.iconBg)
    root.style.setProperty('--icon-color', c.iconColor)
    root.style.setProperty('--link-color', c.linkColor)
    root.style.setProperty('--link-hover', c.linkHover)

    // 渐变
    root.style.setProperty('--gradient-start', c.gradientStart)
    root.style.setProperty('--gradient-end', c.gradientEnd)

    // 状态色
    root.style.setProperty('--color-success', c.success)
    root.style.setProperty('--color-success-light', c.successLight)
    root.style.setProperty('--color-warning', c.warning)
    root.style.setProperty('--color-warning-light', c.warningLight)
    root.style.setProperty('--color-warning-border', c.warningBorder)
    root.style.setProperty('--color-warning-dark', c.warningDark)
  }

  // 初始化主题
  function initTheme() {
    applyTheme(currentTheme.value)
  }

  return {
    currentThemeId,
    currentTheme,
    setTheme,
    initTheme,
  }
})
