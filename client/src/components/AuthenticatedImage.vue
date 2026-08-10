<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

function canAttachAuthorization(rawSource: string) {
  if (!/^[a-z][a-z\d+.-]*:/i.test(rawSource) && !rawSource.startsWith('//')) return true
  try {
    const target = new URL(rawSource, window.location.href)
    const apiOrigin = new URL(API_BASE, window.location.href).origin
    return target.origin === window.location.origin || target.origin === apiOrigin
  } catch {
    return false
  }
}

const props = withDefaults(defineProps<{
  src?: string
  alt?: string
}>(), {
  src: '',
  alt: '',
})

const emit = defineEmits<{
  load: []
  error: [error: Error]
}>()

const objectUrl = ref('')
const loading = ref(false)
const failed = ref(false)
const normalizedSource = computed(() => props.src.trim())
const directSource = computed(() => /^(blob:|data:)/i.test(normalizedSource.value))
const displaySource = computed(() => directSource.value ? normalizedSource.value : objectUrl.value)
let requestVersion = 0
let abortController: AbortController | null = null

watch(normalizedSource, () => void loadSource(), { immediate: true })

function revokeObjectUrl() {
  if (!objectUrl.value) return
  URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
}

function resetRequest() {
  requestVersion += 1
  abortController?.abort()
  abortController = null
  revokeObjectUrl()
}

async function loadSource() {
  resetRequest()
  const version = requestVersion
  failed.value = false

  if (!normalizedSource.value || directSource.value) {
    loading.value = false
    return
  }

  loading.value = true
  const controller = new AbortController()
  abortController = controller
  try {
    const authToken = localStorage.getItem('token')
    const authorizationAllowed = canAttachAuthorization(normalizedSource.value)
    const response = await fetch(normalizedSource.value, {
      headers: authToken && authorizationAllowed ? { Authorization: `Bearer ${authToken}` } : undefined,
      signal: controller.signal,
    })
    if (!response.ok) throw new Error(`image request failed: ${response.status}`)
    const blob = await response.blob()
    if (controller.signal.aborted || version !== requestVersion) return
    objectUrl.value = URL.createObjectURL(blob)
  } catch (error: unknown) {
    if (controller.signal.aborted || version !== requestVersion) return
    const normalizedError = error instanceof Error ? error : new Error('图片加载失败')
    failed.value = true
    emit('error', normalizedError)
    console.error('加载受保护图片失败:', normalizedError)
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

function handleImageLoad() {
  emit('load')
}

function handleImageError() {
  const error = new Error('图片内容无法显示')
  failed.value = true
  if (!directSource.value) revokeObjectUrl()
  emit('error', error)
}

onBeforeUnmount(() => {
  resetRequest()
})
</script>

<template>
  <span
    class="authenticated-image"
    :aria-busy="loading || undefined"
    :data-loading="loading || undefined"
    :data-failed="failed || undefined"
  >
    <img
      v-if="displaySource && !failed"
      :src="displaySource"
      :alt="alt"
      @load="handleImageLoad"
      @error="handleImageError"
    />
    <span
      v-else
      class="authenticated-image__state"
      :role="alt ? 'img' : undefined"
      :aria-label="alt || undefined"
      :aria-hidden="alt ? undefined : true"
    >
      <slot v-if="loading" name="loading">
        <i class="ri-loader-4-line authenticated-image__spin" aria-hidden="true"></i>
        <small>图片加载中</small>
      </slot>
      <slot v-else-if="failed" name="error">
        <i class="ri-image-close-line" aria-hidden="true"></i>
        <small>图片暂不可用</small>
      </slot>
      <slot v-else name="empty">
        <i class="ri-image-line" aria-hidden="true"></i>
        <small>图片待提供</small>
      </slot>
    </span>
  </span>
</template>

<style scoped>
.authenticated-image {
  display: inline-block;
  width: 100%;
  height: 100%;
  overflow: hidden;
  object-fit: inherit;
}

.authenticated-image > img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: inherit;
}

.authenticated-image__state {
  display: grid;
  width: 100%;
  height: 100%;
  place-content: center;
  justify-items: center;
  gap: 6px;
  color: inherit;
  text-align: center;
}

.authenticated-image__state i { font-size: 1.7em; }
.authenticated-image__state small { font-size: 9px; }
.authenticated-image__spin { animation: authenticated-image-spin 900ms linear infinite; }

@keyframes authenticated-image-spin { to { transform: rotate(360deg); } }

@media (prefers-reduced-motion: reduce) {
  .authenticated-image__spin { animation: none; }
}
</style>
