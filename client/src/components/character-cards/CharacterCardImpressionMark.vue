<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
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
  fallbackLabel: '',
  size: 68,
})
const { t } = useI18n()

const customIconFailed = ref(false)
const customIconUrl = computed(() => resolveApiUrl(props.previewUrl || props.iconImageUrl))
const resolvedFallbackLabel = computed(() => props.fallbackLabel || t('characterCards.impressionMark.fallback'))
const sizeStyle = computed(() => {
  const size = typeof props.size === 'number' ? `${props.size}px` : props.size
  return { width: size, height: size }
})
const accessibleLabel = computed(() => {
  if (customIconUrl.value && !customIconFailed.value) return t('characterCards.impressionMark.custom')
  if (props.trp3Icon.trim()) return t('characterCards.impressionMark.trp3', { icon: props.trp3Icon.trim() })
  return t('characterCards.impressionMark.default')
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
      <template #error><span>{{ resolvedFallbackLabel }}</span></template>
    </AuthenticatedImage>
    <WowIcon
      v-else
      class="impression-mark__wow"
      :icon="trp3Icon"
      size="100%"
      :fallback="resolvedFallbackLabel"
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
  border: 1px solid var(--color-border-hover);
  border-radius: 9px;
  background:
    linear-gradient(145deg, var(--gradient-surface), transparent 42%),
    var(--gradient-end);
  box-shadow:
    inset 0 0 0 3px color-mix(in srgb, var(--gradient-end) 84%, black),
    0 5px 12px color-mix(in srgb, var(--color-primary) 22%, transparent);
}

.impression-mark__custom {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.impression-mark__custom :deep(.authenticated-image__state) { color: var(--gradient-text); }
.impression-mark__custom :deep(.authenticated-image__state span) { font-family: Georgia, 'Noto Serif SC', serif; }

.impression-mark__wow {
  width: 100% !important;
  height: 100% !important;
  border-radius: 0;
}

.impression-mark :deep(.wow-icon) {
  border-radius: 0;
  background:
    radial-gradient(circle, color-mix(in srgb, var(--color-accent) 30%, transparent), transparent 58%),
    var(--gradient-end);
}

.impression-mark :deep(.fallback) {
  color: var(--gradient-text);
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
