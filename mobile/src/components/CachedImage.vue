<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { fetchImageObjectUrlWithAuth, getCachedImageObjectUrl, warmImageCache } from '@/utils/imageCache'

const props = withDefaults(
  defineProps<{
    src?: string
    alt?: string
    fit?: 'cover' | 'contain'
    useCache?: boolean
    authFetch?: boolean
    loading?: 'lazy' | 'eager'
  }>(),
  {
    src: '',
    alt: '',
    fit: 'cover',
    useCache: true,
    authFetch: false,
    loading: 'lazy',
  },
)

const resolvedSrc = ref('')
const loaded = ref(false)
const failed = ref(false)
let objectUrl = ''
let resolveToken = 0

function revokeObjectUrl() {
  if (objectUrl) {
    URL.revokeObjectURL(objectUrl)
    objectUrl = ''
  }
}

async function resolveSource() {
  const token = ++resolveToken
  loaded.value = false
  failed.value = false
  revokeObjectUrl()
  const source = String(props.src || '')
  if (!source) {
    resolvedSrc.value = ''
    return
  }

  if (!props.useCache) {
    resolvedSrc.value = source
    return
  }

  if (props.authFetch) {
    const fetched = await fetchImageObjectUrlWithAuth(source)
    if (token !== resolveToken) return
    if (fetched) {
      objectUrl = fetched
      resolvedSrc.value = fetched
      return
    }
    resolvedSrc.value = source
    return
  }

  // First paint with original URL, then try cache in background.
  resolvedSrc.value = source

  const cached = await getCachedImageObjectUrl(source)
  if (token !== resolveToken) return
  if (cached && cached !== resolvedSrc.value) {
    objectUrl = cached
    resolvedSrc.value = cached
  }

  void warmImageCache(source)
}

function handleLoad() {
  loaded.value = true
}

function handleError() {
  failed.value = true
}

watch(
  () => [props.src, props.useCache],
  () => {
    resolveSource()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  revokeObjectUrl()
})
</script>

<template>
  <div class="cached-image" :class="{ loaded, failed }">
    <div v-if="!loaded && !failed" class="image-skeleton" />
    <img
      v-if="resolvedSrc"
      :src="resolvedSrc"
      :alt="alt"
      :loading="loading"
      :style="{ objectFit: fit }"
      @load="handleLoad"
      @error="handleError"
    />
    <div v-if="failed" class="image-fallback"><i class="ri-image-line" /></div>
  </div>
</template>

<style scoped>
.cached-image {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: linear-gradient(135deg, var(--color-border), var(--color-border-light));
}

.image-skeleton {
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.22) 0%, rgba(255, 255, 255, 0.5) 50%, rgba(255, 255, 255, 0.22) 100%);
  background-size: 220% 100%;
  animation: imageShimmer 1.1s linear infinite;
}

img {
  width: 100%;
  height: 100%;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.loaded img {
  opacity: 1;
}

.image-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 18px;
}

@keyframes imageShimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -20% 0;
  }
}
</style>
