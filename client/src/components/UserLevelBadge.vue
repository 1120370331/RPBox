<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  level?: number | null
  name?: string
  color?: string
  bold?: boolean
  size?: 'xs' | 'sm' | 'md'
}>(), {
  level: null,
  name: '',
  color: '#8C7B70',
  bold: false,
  size: 'sm',
})

function normalizeHex(hex: string) {
  const normalized = hex.trim().replace('#', '')
  const full = normalized.length === 3
    ? normalized.split('').map((char) => char + char).join('')
    : normalized

  if (!/^[0-9a-fA-F]{6}$/.test(full)) {
    return '8C7B70'
  }

  return full
}

function hexToRgb(hex: string) {
  const normalized = normalizeHex(hex)
  const r = Number.parseInt(normalized.slice(0, 2), 16)
  const g = Number.parseInt(normalized.slice(2, 4), 16)
  const b = Number.parseInt(normalized.slice(4, 6), 16)
  return { r, g, b }
}

function rgbToHex(r: number, g: number, b: number) {
  const toHex = (value: number) => Math.max(0, Math.min(255, Math.round(value))).toString(16).padStart(2, '0')
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}

function hexToRgba(hex: string, alpha: number) {
  const { r, g, b } = hexToRgb(hex)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

function mixHexColors(from: string, to: string, ratio: number) {
  const start = hexToRgb(from)
  const end = hexToRgb(to)
  return rgbToHex(
    start.r + (end.r - start.r) * ratio,
    start.g + (end.g - start.g) * ratio,
    start.b + (end.b - start.b) * ratio,
  )
}

function getContrastTextColor(hex: string) {
  const { r, g, b } = hexToRgb(hex)
  const luminance = (r * 299 + g * 587 + b * 114) / 1000
  return luminance >= 160 ? '#1F2328' : '#FFF8EF'
}

const badgeGradientPresets: Record<number, { start: string, end: string }> = {
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

function resolveBadgeGradient(level: number | null, color: string) {
  const preset = level ? badgeGradientPresets[level] : undefined
  if (preset) {
    return preset
  }

  const base = normalizeHex(color)
  return {
    start: mixHexColors(`#${base}`, '#FFFFFF', 0.14),
    end: mixHexColors(`#${base}`, '#1A1A1A', 0.08),
  }
}

function getGradientTextColor(start: string, end: string) {
  return getContrastTextColor(mixHexColors(start, end, 0.5))
}

const badgeStyle = computed(() => {
  const gradient = resolveBadgeGradient(props.level, props.color)
  return {
    '--badge-bg': `linear-gradient(145deg, ${gradient.start} 0%, ${gradient.end} 100%)`,
    '--badge-color': getGradientTextColor(gradient.start, gradient.end),
    '--badge-border': hexToRgba(gradient.end, 0.3),
    '--badge-shadow': hexToRgba(gradient.end, 0.18),
  }
})

const badgeLabel = computed(() => {
  if (!props.level) return ''
  return props.name ? `Lv${props.level} ${props.name}` : `Lv${props.level}`
})
</script>

<template>
  <span
    v-if="level"
    class="user-level-badge"
    :class="[size, { bold }]"
    :style="badgeStyle"
  >
    <span class="badge-dot"></span>
    <span class="badge-text">{{ badgeLabel }}</span>
  </span>
</template>

<style scoped>
.user-level-badge {
  --badge-color: #fff8ef;
  --badge-bg: #8c7b70;
  --badge-border: rgba(140, 123, 112, 0.24);
  --badge-shadow: rgba(140, 123, 112, 0.28);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border-radius: 999px;
  background: var(--badge-bg);
  color: var(--badge-color);
  white-space: nowrap;
  box-shadow: 0 6px 16px -12px var(--badge-shadow);
}

.user-level-badge.bold {
  font-weight: 700;
}

.user-level-badge.xs {
  padding: 2px 8px;
  font-size: 10px;
  line-height: 1.2;
}

.user-level-badge.sm {
  padding: 4px 10px;
  font-size: 11px;
  line-height: 1.2;
}

.user-level-badge.md {
  padding: 6px 12px;
  font-size: 12px;
  line-height: 1.2;
}

.badge-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 6px var(--badge-border);
  flex-shrink: 0;
}

.badge-text {
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
