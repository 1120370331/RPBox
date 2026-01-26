<script setup lang="ts">
import { computed, ref, watch } from 'vue'

interface Props {
  icon?: string
  size?: number | string
  fallback?: string
}

const props = withDefaults(defineProps<Props>(), {
  icon: '',
  size: 32,
  fallback: '?'
})

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

function normalizeIconName(value: string): string {
  const trimmed = value.trim()
  if (!trimmed) return ''
  const textureMatch = trimmed.match(/\|T([^:|]+)(?::\d+)?\|t/i)
  const source = textureMatch ? textureMatch[1] : trimmed
  let name = source.replace(/\\/g, '/')
  const lower = name.toLowerCase()
  if (lower.startsWith('interface/icons/')) {
    name = name.slice('interface/icons/'.length)
  }
  name = name.split('/').pop() || name
  name = name.replace(/\.(blp|tga|png|jpg|jpeg)$/i, '')
  name = name.toLowerCase().trim()
  if (!/^[a-z0-9_-]+$/.test(name)) return ''
  return name
}

const isImageUrl = computed(() => {
  if (!props.icon) return false
  return (
    props.icon.startsWith('http://') ||
    props.icon.startsWith('https://') ||
    props.icon.startsWith('data:') ||
    props.icon.startsWith('blob:') ||
    props.icon.startsWith('file:')
  )
})

const normalizedIconName = computed(() => {
  if (!props.icon || isImageUrl.value) return ''
  return normalizeIconName(props.icon)
})

const iconUrl = computed(() => {
  if (!props.icon) return ''
  if (isImageUrl.value) return props.icon
  if (!normalizedIconName.value) return ''
  return `${API_BASE}/icons/${normalizedIconName.value}`
})

const sizeStyle = computed(() => {
  const s = typeof props.size === 'number' ? `${props.size}px` : props.size
  return { width: s, height: s }
})

const loadFailed = ref(false)
const showFallback = computed(() => !iconUrl.value || loadFailed.value)

function handleError() {
  loadFailed.value = true
}

watch(
  () => props.icon,
  () => {
    loadFailed.value = false
  }
)
</script>

<template>
  <div class="wow-icon" :style="sizeStyle">
    <img
      v-if="!showFallback"
      :src="iconUrl"
      :alt="icon"
      @error="handleError"
    />
    <span v-if="showFallback" class="fallback">{{ fallback }}</span>
  </div>
</template>

<style scoped>
.wow-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  overflow: hidden;
  background: var(--color-bg-secondary, #f5f0e8);
}

.wow-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.fallback {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-secondary, #856a52);
}
</style>
