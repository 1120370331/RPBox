<script setup lang="ts">
import { computed } from 'vue'

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

const iconUrl = computed(() => {
  if (!props.icon) return ''
  return `${API_BASE}/icons/${props.icon.toLowerCase()}`
})

const sizeStyle = computed(() => {
  const s = typeof props.size === 'number' ? `${props.size}px` : props.size
  return { width: s, height: s }
})

const showFallback = computed(() => !props.icon)
</script>

<template>
  <div class="wow-icon" :style="sizeStyle">
    <img
      v-if="!showFallback"
      :src="iconUrl"
      :alt="icon"
      @error="($event.target as HTMLImageElement).style.display = 'none'"
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
