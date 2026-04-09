<script setup lang="ts">
import { computed } from 'vue'

interface RgbColor {
  r: number
  g: number
  b: number
}

const props = defineProps<{
  level?: number | null
  name?: string | null
  color?: string | null
  bold?: boolean
  compact?: boolean
}>()

const GRADIENT_PRESETS: Record<number, { start: string; end: string }> = {
  1: { start: '#7A6656', end: '#4B3E35' },
  2: { start: '#D7D9DE', end: '#9A9DA4' },
  3: { start: '#FBF8F1', end: '#E7E1D8' },
  4: { start: '#B7CFB1', end: '#6F8F71' },
  5: { start: '#A9BFD8', end: '#5B7598' },
  6: { start: '#C7B4D9', end: '#7E6896' },
  7: { start: '#E7BB86', end: '#C68E5B' },
  8: { start: '#A6C9CC', end: '#628C93' },
  9: { start: '#E7D9BE', end: '#C7B394' },
  10: { start: '#B61C3A', end: '#4B0817' },
}

function normalizeHex(color?: string | null) {
  if (!color) return ''
  const value = color.trim().replace(/^#/, '')
  if (value.length === 3) {
    return `#${value.split('').map((char) => `${char}${char}`).join('').toUpperCase()}`
  }
  if (value.length === 6) {
    return `#${value.toUpperCase()}`
  }
  return ''
}

function hexToRgb(color?: string | null): RgbColor | null {
  const normalized = normalizeHex(color)
  if (!normalized) return null
  return {
    r: Number.parseInt(normalized.slice(1, 3), 16),
    g: Number.parseInt(normalized.slice(3, 5), 16),
    b: Number.parseInt(normalized.slice(5, 7), 16),
  }
}

function rgbToHex(rgb: RgbColor) {
  const toHex = (value: number) => Math.max(0, Math.min(255, Math.round(value))).toString(16).padStart(2, '0')
  return `#${toHex(rgb.r)}${toHex(rgb.g)}${toHex(rgb.b)}`.toUpperCase()
}

function hexToRgba(color?: string | null, alpha: number = 1) {
  const rgb = hexToRgb(color)
  if (!rgb) return `rgba(75, 54, 33, ${alpha})`
  return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})`
}

function mixHexColors(colorA?: string | null, colorB?: string | null, weight: number = 0.5) {
  const rgbA = hexToRgb(colorA) || { r: 75, g: 54, b: 33 }
  const rgbB = hexToRgb(colorB) || rgbA
  const ratio = Math.max(0, Math.min(1, weight))
  return rgbToHex({
    r: rgbA.r * (1 - ratio) + rgbB.r * ratio,
    g: rgbA.g * (1 - ratio) + rgbB.g * ratio,
    b: rgbA.b * (1 - ratio) + rgbB.b * ratio,
  })
}

function getContrastTextColor(color?: string | null) {
  const rgb = hexToRgb(color)
  if (!rgb) return '#FFF8EF'
  const luminance = (0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b) / 255
  return luminance > 0.68 ? '#1F2328' : '#FFF8EF'
}

function resolveBadgeGradient(level?: number | null, color?: string | null) {
  if (level && GRADIENT_PRESETS[level]) return GRADIENT_PRESETS[level]
  const base = normalizeHex(color) || '#403B33'
  return {
    start: mixHexColors(base, '#FFFFFF', 0.14),
    end: mixHexColors(base, '#1A1A1A', 0.08),
  }
}

const safeLevel = computed(() => {
  const level = Number(props.level || 1)
  return Number.isFinite(level) && level > 0 ? Math.floor(level) : 1
})

const displayName = computed(() => props.name?.trim() || '新人')

const badgeStyle = computed(() => {
  const gradient = resolveBadgeGradient(safeLevel.value, props.color)
  const textColor = getContrastTextColor(gradient.start)
  const shadowColor = mixHexColors(gradient.end, '#2C1810', 0.24)
  return {
    '--badge-bg': `linear-gradient(145deg, ${gradient.start} 0%, ${gradient.end} 100%)`,
    '--badge-color': textColor,
    '--badge-shadow': `0 6px 16px -12px ${hexToRgba(shadowColor, 0.45)}`,
    '--badge-dot': hexToRgba(textColor, textColor === '#1F2328' ? 0.34 : 0.22),
  }
})
</script>

<template>
  <span class="user-level-badge" :class="{ bold, compact }" :style="badgeStyle">
    <span class="level-prefix">Lv{{ safeLevel }}</span>
    <span class="level-name">{{ displayName }}</span>
  </span>
</template>

<style scoped>
.user-level-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--badge-bg);
  color: var(--badge-color);
  box-shadow: var(--badge-shadow);
  font-size: 11px;
  line-height: 1;
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.user-level-badge.compact {
  padding: 3px 8px;
  gap: 4px;
  font-size: 10px;
}

.level-prefix,
.level-name {
  display: inline-flex;
  align-items: center;
}

.level-prefix {
  font-weight: 700;
  opacity: 0.88;
}

.level-name {
  font-weight: 600;
}

.user-level-badge.bold .level-prefix,
.user-level-badge.bold .level-name {
  font-weight: 700;
}

.user-level-badge::before {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--badge-dot);
  flex-shrink: 0;
}
</style>
