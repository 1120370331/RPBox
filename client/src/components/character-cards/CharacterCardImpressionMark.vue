<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { resolveApiUrl } from '@/api/item'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import WowIcon from '@/components/WowIcon.vue'

interface Props {
  iconImageUrl?: string
  trp3Icon?: string
  previewUrl?: string
  fallbackLabel?: string
  size?: number | string
}

const props = withDefaults(defineProps<Props>(), {
  iconImageUrl: '',
  trp3Icon: '',
  previewUrl: '',
  fallbackLabel: '印',
  size: 68,
})

const customIconFailed = ref(false)
const customIconUrl = computed(() => resolveApiUrl(props.previewUrl || props.iconImageUrl))
const sizeStyle = computed(() => {
  const size = typeof props.size === 'number' ? `${props.size}px` : props.size
  return { width: size, height: size }
})
const accessibleLabel = computed(() => {
  if (customIconUrl.value && !customIconFailed.value) return '自定义印象图标'
  if (props.trp3Icon.trim()) return `TRP3 图标：${props.trp3Icon.trim()}`
  return '默认观察印记'
})

watch(customIconUrl, () => {
  customIconFailed.value = false
})
</script>

<template>
  <span class="impression-mark" :style="sizeStyle" role="img" :aria-label="accessibleLabel">
    <AuthenticatedImage
      v-if="customIconUrl && !customIconFailed"
      class="impression-mark__custom"
      :src="customIconUrl"
      alt=""
      @error="customIconFailed = true"
    >
      <template #loading><i class="ri-loader-4-line impression-mark__spin" aria-hidden="true"></i></template>
      <template #error><span>{{ fallbackLabel }}</span></template>
    </AuthenticatedImage>
    <WowIcon
      v-else
      class="impression-mark__wow"
      :icon="trp3Icon"
      size="100%"
      :fallback="fallbackLabel"
      aria-hidden="true"
    />
  </span>
</template>

<style scoped>
.impression-mark {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  box-sizing: border-box;
  border: 1px solid #9c673d;
  border-radius: 9px;
  background:
    linear-gradient(145deg, rgba(255, 250, 242, 0.12), transparent 42%),
    #38241a;
  box-shadow:
    inset 0 0 0 3px #241710,
    0 5px 12px rgba(44, 24, 16, 0.2);
}

.impression-mark__custom {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.impression-mark__custom :deep(.authenticated-image__state) { color: #e4c19d; }
.impression-mark__custom :deep(.authenticated-image__state span) { font-family: Georgia, 'Noto Serif SC', serif; }

.impression-mark__wow {
  width: 100% !important;
  height: 100% !important;
  border-radius: 0;
}

.impression-mark :deep(.wow-icon) {
  border-radius: 0;
  background:
    radial-gradient(circle, rgba(184, 115, 51, 0.28), transparent 58%),
    #2b1b14;
}

.impression-mark :deep(.fallback) {
  color: #e4c19d;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: clamp(14px, 35%, 25px);
  font-weight: 600;
}

.impression-mark__spin { animation: impression-mark-spin 900ms linear infinite; }
@keyframes impression-mark-spin { to { transform: rotate(360deg); } }

@media (prefers-reduced-motion: reduce) {
  .impression-mark__spin { animation: none; }
}
</style>
