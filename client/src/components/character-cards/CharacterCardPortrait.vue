<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { getCharacterCardPortraitUrl, type CharacterCard } from '@/api/characterCard'

const props = withDefaults(defineProps<{
  card: Pick<CharacterCard, 'id' | 'portrait_image_url' | 'portrait_image_updated_at' | 'updated_at'>
  alt: string
  width?: number
  quality?: number
}>(), {
  width: 900,
  quality: 88,
})

const objectUrl = ref('')
const loading = ref(false)
const failed = ref(false)
const sourceUrl = computed(() => getCharacterCardPortraitUrl(props.card, {
  w: props.width,
  q: props.quality,
}))
let requestToken = 0
let abortController: AbortController | null = null

watch(sourceUrl, () => void loadPortrait(), { immediate: true })

function revokeObjectUrl() {
  if (!objectUrl.value) return
  URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
}

async function loadPortrait() {
  const token = ++requestToken
  abortController?.abort()
  abortController = null
  revokeObjectUrl()
  failed.value = false

  if (!sourceUrl.value) {
    loading.value = false
    return
  }

  loading.value = true
  const controller = new AbortController()
  abortController = controller
  try {
    const authToken = localStorage.getItem('token')
    const response = await fetch(sourceUrl.value, {
      headers: authToken ? { Authorization: `Bearer ${authToken}` } : undefined,
      signal: controller.signal,
    })
    if (!response.ok) throw new Error(`portrait request failed: ${response.status}`)
    const blob = await response.blob()
    if (token !== requestToken) return
    objectUrl.value = URL.createObjectURL(blob)
  } catch (error: unknown) {
    if (controller.signal.aborted || token !== requestToken) return
    failed.value = true
    console.error('加载人物卡肖像失败:', error)
  } finally {
    if (token === requestToken) loading.value = false
  }
}

onBeforeUnmount(() => {
  requestToken += 1
  abortController?.abort()
  abortController = null
  revokeObjectUrl()
})
</script>

<template>
  <img v-if="objectUrl" :src="objectUrl" :alt="alt" />
  <span v-else class="character-portrait-state" role="img" :aria-label="alt" :data-loading="loading || undefined" :data-failed="failed || undefined">
    <slot name="fallback" :loading="loading" :failed="failed">
      <i :class="loading ? 'ri-loader-4-line character-portrait-state__spin' : failed ? 'ri-image-close-line' : 'ri-user-star-line'" aria-hidden="true"></i>
      <small>{{ loading ? '肖像加载中' : failed ? '肖像暂不可用' : '肖像待归档' }}</small>
    </slot>
  </span>
</template>

<style scoped>
.character-portrait-state {
  display: grid;
  width: 100%;
  height: 100%;
  place-content: center;
  gap: 8px;
  color: inherit;
  text-align: center;
}

.character-portrait-state i { font-size: 1.8em; }
.character-portrait-state small { font-size: 10px; }
.character-portrait-state__spin { animation: portrait-spin 900ms linear infinite; }
@keyframes portrait-spin { to { transform: rotate(360deg); } }

@media (prefers-reduced-motion: reduce) {
  .character-portrait-state__spin { animation: none; }
}
</style>
