<script setup lang="ts">
import { computed, getCurrentInstance, onBeforeUnmount, ref, watch } from 'vue'

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

const instance = getCurrentInstance()
const hasDownloadListener = computed(() => {
  return !!instance?.vnode?.props?.onDownload
})

const minScale = 0.2
const maxScale = 5
const scaleStep = 0.2
const controlsFadeDelay = 2200

const currentIndex = ref(0)
const scale = ref(1)
const rotation = ref(0)
const translateX = ref(0)
const translateY = ref(0)
const isDragging = ref(false)
const controlsVisible = ref(true)
const viewerContentRef = ref<HTMLDivElement | null>(null)
const panStartX = ref(0)
const panStartY = ref(0)
let controlsFadeTimer: ReturnType<typeof setTimeout> | null = null

const currentImage = computed(() => props.images[currentIndex.value] || '')
const transformStyle = computed(() => ({
  transform: `translate3d(${translateX.value}px, ${translateY.value}px, 0) scale(${scale.value}) rotate(${rotation.value}deg)`,
}))

watch(() => props.modelValue, (visible) => {
  if (visible) {
    currentIndex.value = Math.min(props.startIndex || 0, Math.max(props.images.length - 1, 0))
    resetTransform()
    window.addEventListener('keydown', handleKeydown)
    showControlsTemporarily()
  } else {
    window.removeEventListener('keydown', handleKeydown)
    clearControlsFadeTimer()
    controlsVisible.value = true
    isDragging.value = false
  }
})

watch(() => props.startIndex, (value) => {
  if (props.modelValue) {
    currentIndex.value = Math.min(value || 0, Math.max(props.images.length - 1, 0))
    resetTransform()
  }
})

watch(currentIndex, () => {
  resetTransform()
  showControlsTemporarily()
})

watch(scale, (value) => {
  if (value <= 1) {
    resetPan()
    isDragging.value = false
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  clearControlsFadeTimer()
})

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

function resetTransform() {
  scale.value = 1
  rotation.value = 0
  resetPan()
}

function resetPan() {
  translateX.value = 0
  translateY.value = 0
}

function clampScale(value: number) {
  return Math.min(maxScale, Math.max(minScale, value))
}

function zoomIn() {
  scale.value = clampScale(scale.value + scaleStep)
  showControlsTemporarily()
}

function zoomOut() {
  scale.value = clampScale(scale.value - scaleStep)
  showControlsTemporarily()
}

function rotateLeft() {
  rotation.value -= 90
  showControlsTemporarily()
}

function rotateRight() {
  rotation.value += 90
  showControlsTemporarily()
}

function handleWheel(event: WheelEvent) {
  event.preventDefault()
  showControlsTemporarily()
  if (event.deltaY < 0) {
    zoomIn()
  } else {
    zoomOut()
  }
}

function canPan() {
  return scale.value > 1
}

function handlePointerDown(event: PointerEvent) {
  if (!canPan()) return
  event.preventDefault()
  showControlsTemporarily()
  isDragging.value = true
  panStartX.value = event.clientX - translateX.value
  panStartY.value = event.clientY - translateY.value
  viewerContentRef.value?.setPointerCapture(event.pointerId)
}

function handlePointerMove(event: PointerEvent) {
  if (!isDragging.value) return
  event.preventDefault()
  translateX.value = event.clientX - panStartX.value
  translateY.value = event.clientY - panStartY.value
}

function stopDragging(pointerId?: number) {
  if (!isDragging.value) return
  isDragging.value = false
  if (pointerId !== undefined) {
    try {
      viewerContentRef.value?.releasePointerCapture(pointerId)
    } catch {
      // ignore pointer capture release errors
    }
  }
  showControlsTemporarily()
}

function handlePointerUp(event: PointerEvent) {
  stopDragging(event.pointerId)
}

function handleUserActivity() {
  if (!props.modelValue) return
  showControlsTemporarily()
}

function clearControlsFadeTimer() {
  if (controlsFadeTimer) {
    clearTimeout(controlsFadeTimer)
    controlsFadeTimer = null
  }
}

function showControlsTemporarily() {
  controlsVisible.value = true
  clearControlsFadeTimer()

  if (!props.modelValue) return
  controlsFadeTimer = setTimeout(() => {
    if (isDragging.value) {
      showControlsTemporarily()
      return
    }
    controlsVisible.value = false
  }, controlsFadeDelay)
}

function getDefaultFilename(url: string) {
  const cleaned = url.split('?')[0]
  const match = cleaned.match(/\/([^/]+)$/)
  return match?.[1] || `image-${Date.now()}.jpg`
}

function downloadCurrentImage() {
  const url = currentImage.value
  if (!url) return
  const link = document.createElement('a')
  link.href = url
  link.download = getDefaultFilename(url)
  link.target = '_blank'
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

function handleDownload() {
  emit('download', currentIndex.value)
  if (!hasDownloadListener.value) {
    downloadCurrentImage()
  }
  showControlsTemporarily()
}

function handleKeydown(event: KeyboardEvent) {
  if (!props.modelValue) return
  showControlsTemporarily()
  if (event.key === 'Escape') {
    closeViewer()
    return
  }
  if (event.key === 'ArrowLeft') {
    prevImage()
    return
  }
  if (event.key === 'ArrowRight') {
    nextImage()
    return
  }
  if (event.key === '+' || event.key === '=') {
    zoomIn()
    return
  }
  if (event.key === '-') {
    zoomOut()
    return
  }
  if (event.key === '0') {
    resetTransform()
    return
  }
  if (event.key.toLowerCase() === 'r') {
    rotateRight()
  }
}
</script>

<template>
  <div
    v-if="modelValue"
    class="image-viewer-modal"
    @click.self="closeViewer"
    @mousemove="handleUserActivity"
    @touchstart="handleUserActivity"
  >
    <button class="viewer-close viewer-control" :class="{ 'controls-hidden': !controlsVisible }" @click="closeViewer">
      <i class="ri-close-line"></i>
    </button>

    <button class="viewer-nav prev viewer-control" :class="{ 'controls-hidden': !controlsVisible }" @click="prevImage" v-if="images.length > 1">
      <i class="ri-arrow-left-s-line"></i>
    </button>

    <div
      ref="viewerContentRef"
      class="viewer-content"
      :class="{ 'is-pannable': scale > 1, 'is-dragging': isDragging }"
      @wheel="handleWheel"
      @pointerdown="handlePointerDown"
      @pointermove="handlePointerMove"
      @pointerup="handlePointerUp"
      @pointercancel="handlePointerUp"
      @pointerleave="handlePointerUp"
    >
      <img v-if="currentImage" :src="currentImage" alt="" :style="transformStyle" draggable="false" @dragstart.prevent />
    </div>

    <button class="viewer-nav next viewer-control" :class="{ 'controls-hidden': !controlsVisible }" @click="nextImage" v-if="images.length > 1">
      <i class="ri-arrow-right-s-line"></i>
    </button>

    <div class="viewer-toolbar viewer-control" :class="{ 'controls-hidden': !controlsVisible }" v-if="images.length > 0">
      <span class="viewer-counter">{{ currentIndex + 1 }} / {{ images.length }}</span>
      <button class="viewer-tool-btn" @click="zoomOut" title="缩小 (-)">
        <i class="ri-zoom-out-line"></i>
      </button>
      <span class="viewer-scale">{{ Math.round(scale * 100) }}%</span>
      <button class="viewer-tool-btn" @click="zoomIn" title="放大 (+)">
        <i class="ri-zoom-in-line"></i>
      </button>
      <button class="viewer-tool-btn" @click="rotateLeft" title="左转 90°">
        <i class="ri-anticlockwise-2-line"></i>
      </button>
      <button class="viewer-tool-btn" @click="rotateRight" title="右转 90° (R)">
        <i class="ri-clockwise-2-line"></i>
      </button>
      <button class="viewer-tool-btn" @click="resetTransform" title="重置 (0)">
        <i class="ri-refresh-line"></i>
      </button>
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
  background: rgba(0, 0, 0, 0.9);
  z-index: 4000;
  display: flex;
  align-items: stretch;
  justify-content: stretch;
}

.viewer-control {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.viewer-control.controls-hidden {
  opacity: 0;
  pointer-events: none;
}

.viewer-close {
  position: absolute;
  top: 24px;
  right: 24px;
  background: rgba(255, 255, 255, 0.1);
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
  background: rgba(255, 255, 255, 0.25);
}

.viewer-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(255, 255, 255, 0.12);
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
  background: rgba(255, 255, 255, 0.3);
}

.viewer-nav.prev {
  left: 24px;
}

.viewer-nav.next {
  right: 24px;
}

.viewer-content {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: default;
  touch-action: none;
  user-select: none;
}

.viewer-content.is-pannable {
  cursor: grab;
}

.viewer-content.is-dragging {
  cursor: grabbing;
}

.viewer-content img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  display: block;
  transform-origin: center center;
  transition: transform 0.15s ease-out;
  will-change: transform;
  user-select: none;
  -webkit-user-drag: none;
}

.viewer-toolbar {
  position: absolute;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(16, 16, 16, 0.72);
  color: #fff;
  font-size: 13px;
}

.viewer-counter {
  padding: 4px 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.1);
  font-weight: 600;
}

.viewer-scale {
  min-width: 54px;
  text-align: center;
  font-variant-numeric: tabular-nums;
}

.viewer-tool-btn,
.viewer-download {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 10px;
  border: 1px solid rgba(255, 255, 255, 0.28);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 13px;
}

.viewer-tool-btn {
  min-width: 32px;
  padding: 0;
}

.viewer-tool-btn:hover,
.viewer-download:hover {
  background: rgba(255, 255, 255, 0.2);
}

@media (max-width: 768px) {
  .viewer-close {
    top: 16px;
    right: 16px;
  }

  .viewer-nav.prev {
    left: 12px;
  }

  .viewer-nav.next {
    right: 12px;
  }

  .viewer-toolbar {
    bottom: 16px;
    max-width: calc(100vw - 24px);
    overflow-x: auto;
    justify-content: flex-start;
  }
}
</style>
