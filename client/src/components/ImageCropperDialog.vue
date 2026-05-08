<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import RButton from '@/components/RButton.vue'
import RModal from '@/components/RModal.vue'

interface Props {
  modelValue: boolean
  file: File | null
  title?: string
  aspectRatio?: number
  outputWidth?: number
  outputHeight?: number
  maxSizeKB?: number
  mimeType?: 'image/jpeg' | 'image/png' | 'image/webp'
  quality?: number
  roundPreview?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  title: '调整图片',
  aspectRatio: 1,
  outputWidth: 512,
  outputHeight: 512,
  maxSizeKB: 1024,
  mimeType: 'image/jpeg',
  quality: 0.9,
  roundPreview: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  cropped: [file: File]
  error: [error: Error]
}>()

const modalOpen = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const stageRef = ref<HTMLElement | null>(null)
const frameRef = ref<HTMLElement | null>(null)
const imageRef = ref<HTMLImageElement | null>(null)
const imageUrl = ref('')
const imageNatural = ref({ width: 0, height: 0 })
const zoom = ref(1)
const position = ref({ x: 0, y: 0 })
const layout = ref({
  frameWidth: 0,
  frameHeight: 0,
  displayWidth: 0,
  displayHeight: 0,
})
const dragging = ref(false)
const dragStart = ref({ x: 0, y: 0 })
const startPosition = ref({ x: 0, y: 0 })
const processing = ref(false)

const zoomPercent = computed(() => Math.round(zoom.value * 100))

const imageStyle = computed(() => ({
  width: `${layout.value.displayWidth}px`,
  height: `${layout.value.displayHeight}px`,
  transform: `translate(-50%, -50%) translate(${position.value.x}px, ${position.value.y}px)`,
}))

function revokeImageUrl() {
  if (imageUrl.value) {
    URL.revokeObjectURL(imageUrl.value)
    imageUrl.value = ''
  }
}

function resetView() {
  zoom.value = 1
  position.value = { x: 0, y: 0 }
  refreshLayout()
}

function loadFile(file: File | null) {
  revokeImageUrl()
  imageNatural.value = { width: 0, height: 0 }
  if (!file) return
  imageUrl.value = URL.createObjectURL(file)
}

function getFrameSize() {
  const stage = stageRef.value
  if (!stage) return null

  const stageRect = stage.getBoundingClientRect()
  const padding = 28
  const maxWidth = Math.max(120, stageRect.width - padding)
  const maxHeight = Math.max(120, stageRect.height - padding)
  let frameWidth = maxWidth
  let frameHeight = frameWidth / props.aspectRatio

  if (frameHeight > maxHeight) {
    frameHeight = maxHeight
    frameWidth = frameHeight * props.aspectRatio
  }

  return { frameWidth, frameHeight }
}

function clampPosition(next = position.value) {
  const { displayWidth, displayHeight, frameWidth, frameHeight } = layout.value
  const maxX = Math.max(0, (displayWidth - frameWidth) / 2)
  const maxY = Math.max(0, (displayHeight - frameHeight) / 2)

  position.value = {
    x: Math.min(maxX, Math.max(-maxX, next.x)),
    y: Math.min(maxY, Math.max(-maxY, next.y)),
  }
}

function refreshLayout() {
  const frameSize = getFrameSize()
  const { width, height } = imageNatural.value
  if (!frameSize || !width || !height) return

  const baseScale = Math.max(frameSize.frameWidth / width, frameSize.frameHeight / height)
  layout.value = {
    frameWidth: frameSize.frameWidth,
    frameHeight: frameSize.frameHeight,
    displayWidth: width * baseScale * zoom.value,
    displayHeight: height * baseScale * zoom.value,
  }
  clampPosition()
}

function onImageLoad(event: Event) {
  const img = event.target as HTMLImageElement
  imageNatural.value = {
    width: img.naturalWidth,
    height: img.naturalHeight,
  }
  void nextTick(resetView)
}

function onPointerDown(event: PointerEvent) {
  if (!imageNatural.value.width || processing.value) return
  dragging.value = true
  dragStart.value = { x: event.clientX, y: event.clientY }
  startPosition.value = { ...position.value }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function onPointerMove(event: PointerEvent) {
  if (!dragging.value) return
  const dx = event.clientX - dragStart.value.x
  const dy = event.clientY - dragStart.value.y
  clampPosition({
    x: startPosition.value.x + dx,
    y: startPosition.value.y + dy,
  })
}

function onPointerUp(event: PointerEvent) {
  if (!dragging.value) return
  dragging.value = false
  ;(event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId)
}

function onWheel(event: WheelEvent) {
  if (!imageNatural.value.width || processing.value) return
  const nextZoom = zoom.value + (event.deltaY > 0 ? -0.08 : 0.08)
  zoom.value = Math.min(4, Math.max(1, Number(nextZoom.toFixed(2))))
  refreshLayout()
}

function setZoom(value: number) {
  zoom.value = Number(value)
  refreshLayout()
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number) {
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error('图片处理失败'))
        return
      }
      resolve(blob)
    }, type, quality)
  })
}

async function renderCroppedBlob() {
  const image = imageRef.value
  const stage = stageRef.value
  const frame = frameRef.value
  const { width, height } = imageNatural.value
  const { displayWidth, displayHeight } = layout.value
  if (!image || !stage || !frame || !width || !height || !displayWidth || !displayHeight) {
    throw new Error('图片还未加载完成')
  }

  const stageRect = stage.getBoundingClientRect()
  const frameRect = frame.getBoundingClientRect()
  const imageLeft = stageRect.left + stageRect.width / 2 - displayWidth / 2 + position.value.x
  const imageTop = stageRect.top + stageRect.height / 2 - displayHeight / 2 + position.value.y
  const sourceX = Math.max(0, ((frameRect.left - imageLeft) / displayWidth) * width)
  const sourceY = Math.max(0, ((frameRect.top - imageTop) / displayHeight) * height)
  const sourceWidth = Math.min(width - sourceX, (frameRect.width / displayWidth) * width)
  const sourceHeight = Math.min(height - sourceY, (frameRect.height / displayHeight) * height)

  const canvas = document.createElement('canvas')
  canvas.width = props.outputWidth
  canvas.height = props.outputHeight
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('当前环境不支持图片处理')

  if (props.mimeType === 'image/jpeg') {
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
  }

  ctx.imageSmoothingEnabled = true
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(
    image,
    sourceX,
    sourceY,
    sourceWidth,
    sourceHeight,
    0,
    0,
    props.outputWidth,
    props.outputHeight,
  )

  let quality = props.quality
  let blob = await canvasToBlob(canvas, props.mimeType, quality)
  while (blob.size > props.maxSizeKB * 1024 && quality > 0.45 && props.mimeType !== 'image/png') {
    quality = Number((quality - 0.08).toFixed(2))
    blob = await canvasToBlob(canvas, props.mimeType, quality)
  }

  return blob
}

function outputFileName() {
  const original = props.file?.name || 'image'
  const baseName = original.replace(/\.[^.]+$/, '') || 'image'
  const extension = props.mimeType === 'image/png'
    ? 'png'
    : props.mimeType === 'image/webp'
      ? 'webp'
      : 'jpg'
  return `${baseName}-cropped.${extension}`
}

async function confirmCrop() {
  if (processing.value) return
  processing.value = true
  try {
    const blob = await renderCroppedBlob()
    emit('cropped', new File([blob], outputFileName(), { type: props.mimeType }))
    modalOpen.value = false
  } catch (error) {
    emit('error', error instanceof Error ? error : new Error('图片处理失败'))
  } finally {
    processing.value = false
  }
}

function cancel() {
  modalOpen.value = false
}

watch(() => props.file, (file) => {
  if (props.modelValue) loadFile(file)
})

watch(() => props.modelValue, async (open) => {
  if (open) {
    loadFile(props.file)
    await nextTick()
    refreshLayout()
  }
})

watch(() => props.aspectRatio, () => {
  void nextTick(refreshLayout)
})

onMounted(() => {
  window.addEventListener('resize', refreshLayout)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', refreshLayout)
  revokeImageUrl()
})
</script>

<template>
  <RModal
    v-model="modalOpen"
    :title="title"
    width="720px"
    :mask-closable="!processing"
    :closable="!processing"
  >
    <div class="cropper">
      <div
        ref="stageRef"
        class="crop-stage"
        :class="{ dragging }"
        @pointerdown="onPointerDown"
        @pointermove="onPointerMove"
        @pointerup="onPointerUp"
        @pointercancel="onPointerUp"
        @wheel.prevent="onWheel"
      >
        <img
          v-if="imageUrl"
          ref="imageRef"
          class="crop-image"
          :src="imageUrl"
          :style="imageStyle"
          alt=""
          draggable="false"
          @load="onImageLoad"
        />
        <div class="crop-shade"></div>
        <div
          ref="frameRef"
          class="crop-frame"
          :class="{ round: roundPreview }"
          :style="{
            width: `${layout.frameWidth}px`,
            height: `${layout.frameHeight}px`,
          }"
        ></div>
      </div>

      <div class="crop-controls">
        <div class="zoom-row">
          <i class="ri-zoom-out-line"></i>
          <input
            type="range"
            min="1"
            max="4"
            step="0.01"
            :value="zoom"
            :disabled="processing"
            aria-label="缩放"
            @input="setZoom(Number(($event.target as HTMLInputElement).value))"
          />
          <i class="ri-zoom-in-line"></i>
          <span class="zoom-value">{{ zoomPercent }}%</span>
        </div>
        <p class="crop-hint">拖动图片调整取景，滚轮或滑杆缩放。</p>
      </div>
    </div>

    <template #footer>
      <RButton type="secondary" :disabled="processing" @click="cancel">取消</RButton>
      <RButton type="primary" :loading="processing" @click="confirmCrop">应用裁剪</RButton>
    </template>
  </RModal>
</template>

<style scoped>
.cropper {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.crop-stage {
  position: relative;
  height: min(54vh, 420px);
  min-height: 280px;
  overflow: hidden;
  border-radius: 12px;
  background:
    linear-gradient(45deg, rgba(255, 255, 255, 0.06) 25%, transparent 25%),
    linear-gradient(-45deg, rgba(255, 255, 255, 0.06) 25%, transparent 25%),
    #15110f;
  background-size: 18px 18px;
  cursor: grab;
  touch-action: none;
  user-select: none;
}

.crop-stage.dragging {
  cursor: grabbing;
}

.crop-image {
  position: absolute;
  left: 50%;
  top: 50%;
  max-width: none;
  max-height: none;
  user-select: none;
  pointer-events: none;
  transform-origin: center;
}

.crop-shade {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.42);
  pointer-events: none;
}

.crop-frame {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  border: 2px solid rgba(255, 255, 255, 0.92);
  box-shadow: 0 0 0 999px rgba(0, 0, 0, 0.44);
  pointer-events: none;
}

.crop-frame::before,
.crop-frame::after {
  content: '';
  position: absolute;
  inset: 33.333% 0;
  border-top: 1px solid rgba(255, 255, 255, 0.34);
  border-bottom: 1px solid rgba(255, 255, 255, 0.34);
}

.crop-frame::after {
  inset: 0 33.333%;
  border: 0;
  border-left: 1px solid rgba(255, 255, 255, 0.34);
  border-right: 1px solid rgba(255, 255, 255, 0.34);
}

.crop-frame.round {
  border-radius: 50%;
}

.crop-controls {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.zoom-row {
  display: grid;
  grid-template-columns: 24px 1fr 24px 48px;
  align-items: center;
  gap: 10px;
  color: var(--color-secondary, #804030);
}

.zoom-row i {
  font-size: 18px;
  text-align: center;
}

.zoom-row input[type='range'] {
  width: 100%;
  accent-color: var(--color-accent, #B87333);
}

.zoom-value {
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-secondary, #8C7B70);
  text-align: right;
}

.crop-hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-muted, rgba(75, 54, 33, 0.5));
}

@media (max-width: 640px) {
  .crop-stage {
    height: 340px;
    min-height: 260px;
  }

  .zoom-row {
    grid-template-columns: 20px 1fr 20px 44px;
  }
}
</style>
