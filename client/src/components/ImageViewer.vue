<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: boolean
  images: string[]
  startIndex?: number
  showDownload?: boolean
  downloadLabel?: string
}>(), {
  startIndex: 0,
  showDownload: false,
  downloadLabel: 'Download',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'download', index: number): void
}>()

const currentIndex = ref(0)

watch(() => props.modelValue, (visible) => {
  if (visible) {
    currentIndex.value = Math.min(props.startIndex || 0, Math.max(props.images.length - 1, 0))
  }
})

watch(() => props.startIndex, (value) => {
  if (props.modelValue) {
    currentIndex.value = Math.min(value || 0, Math.max(props.images.length - 1, 0))
  }
})

const currentImage = computed(() => props.images[currentIndex.value] || '')

function closeViewer() {
  emit('update:modelValue', false)
}

function prevImage() {
  if (props.images.length <= 1) return
  if (currentIndex.value > 0) {
    currentIndex.value--
  } else {
    currentIndex.value = props.images.length - 1
  }
}

function nextImage() {
  if (props.images.length <= 1) return
  if (currentIndex.value < props.images.length - 1) {
    currentIndex.value++
  } else {
    currentIndex.value = 0
  }
}

function handleDownload() {
  emit('download', currentIndex.value)
}
</script>

<template>
  <div v-if="modelValue" class="image-viewer-modal" @click.self="closeViewer">
    <button class="viewer-close" @click="closeViewer">
      <i class="ri-close-line"></i>
    </button>
    <button class="viewer-nav prev" @click="prevImage" v-if="images.length > 1">
      <i class="ri-arrow-left-s-line"></i>
    </button>
    <div class="viewer-content">
      <img v-if="currentImage" :src="currentImage" alt="" />
    </div>
    <button class="viewer-nav next" @click="nextImage" v-if="images.length > 1">
      <i class="ri-arrow-right-s-line"></i>
    </button>
    <div class="viewer-footer" v-if="images.length > 0">
      <span class="viewer-counter">{{ currentIndex + 1 }} / {{ images.length }}</span>
      <button v-if="showDownload" class="viewer-download" @click="handleDownload">
        <i class="ri-download-line"></i> {{ downloadLabel }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.image-viewer-modal {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.9);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.viewer-close {
  position: absolute;
  top: 24px;
  right: 24px;
  background: rgba(255,255,255,0.1);
  border: none;
  color: #fff;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.viewer-close:hover {
  background: rgba(255,255,255,0.25);
}

.viewer-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(255,255,255,0.1);
  border: none;
  color: #fff;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.viewer-nav:hover {
  background: rgba(255,255,255,0.25);
}

.viewer-nav.prev {
  left: 24px;
}

.viewer-nav.next {
  right: 24px;
}

.viewer-content {
  max-width: 90vw;
  max-height: 82vh;
}

.viewer-content img {
  max-width: 100%;
  max-height: 82vh;
  object-fit: contain;
  display: block;
}

.viewer-footer {
  position: absolute;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 16px;
  color: #fff;
  font-size: 14px;
}

.viewer-counter {
  background: rgba(0,0,0,0.5);
  padding: 6px 12px;
  border-radius: 20px;
}

.viewer-download {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid rgba(255,255,255,0.3);
  background: rgba(255,255,255,0.1);
  color: #fff;
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.2s;
}

.viewer-download:hover {
  background: rgba(255,255,255,0.2);
}
</style>
