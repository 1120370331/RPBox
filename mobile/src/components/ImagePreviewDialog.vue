<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    open: boolean
    src?: string
    alt?: string
  }>(),
  {
    src: '',
    alt: '',
  },
)

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

const scale = ref(1)
const translateX = ref(0)
const translateY = ref(0)
const closeButtonRef = ref<HTMLButtonElement | null>(null)

let previouslyFocusedElement: HTMLElement | null = null
let isListeningForKeydown = false

const startScale = ref(1)
const startTranslateX = ref(0)
const startTranslateY = ref(0)
const pinchStartDistance = ref(0)
const panStartX = ref(0)
const panStartY = ref(0)
const panTouchId = ref<number | null>(null)

const imageStyle = computed(() => ({
  transform: `translate3d(${translateX.value}px, ${translateY.value}px, 0) scale(${scale.value})`,
}))
const dialogLabel = computed(() => {
  const imageDescription = props.alt.trim()
  const label = t('common.imagePreview.label')
  return imageDescription ? `${label}: ${imageDescription}` : label
})

function clampScale(value: number) {
  return Math.min(4, Math.max(1, value))
}

function resetTransform() {
  scale.value = 1
  translateX.value = 0
  translateY.value = 0
}

function getDistance(t1: Touch, t2: Touch) {
  const dx = t1.clientX - t2.clientX
  const dy = t1.clientY - t2.clientY
  return Math.sqrt(dx * dx + dy * dy)
}

function getCenter(t1: Touch, t2: Touch) {
  return {
    x: (t1.clientX + t2.clientX) / 2,
    y: (t1.clientY + t2.clientY) / 2,
  }
}

function onTouchStart(event: TouchEvent) {
  if (event.touches.length >= 2) {
    const [t1, t2] = [event.touches[0], event.touches[1]]
    pinchStartDistance.value = getDistance(t1, t2)
    startScale.value = scale.value
    startTranslateX.value = translateX.value
    startTranslateY.value = translateY.value
    const center = getCenter(t1, t2)
    panStartX.value = center.x
    panStartY.value = center.y
    panTouchId.value = null
    return
  }

  const t = event.touches[0]
  if (!t) return
  if (scale.value > 1) {
    panTouchId.value = t.identifier
    panStartX.value = t.clientX - translateX.value
    panStartY.value = t.clientY - translateY.value
  }
}

function onTouchMove(event: TouchEvent) {
  if (event.touches.length >= 2) {
    event.preventDefault()
    const [t1, t2] = [event.touches[0], event.touches[1]]
    const distance = getDistance(t1, t2)
    if (pinchStartDistance.value <= 0) return
    scale.value = clampScale(startScale.value * (distance / pinchStartDistance.value))
    const center = getCenter(t1, t2)
    translateX.value = startTranslateX.value + (center.x - panStartX.value)
    translateY.value = startTranslateY.value + (center.y - panStartY.value)
    return
  }

  if (scale.value <= 1 || panTouchId.value === null) return
  const current = Array.from(event.touches).find(t => t.identifier === panTouchId.value) || event.touches[0]
  if (!current) return
  event.preventDefault()
  translateX.value = current.clientX - panStartX.value
  translateY.value = current.clientY - panStartY.value
}

function onTouchEnd() {
  if (scale.value <= 1) {
    resetTransform()
  }
  pinchStartDistance.value = 0
  panTouchId.value = null
}

function onDoubleClick() {
  if (scale.value > 1) {
    resetTransform()
    return
  }
  scale.value = 2
}

function handleKeydown(event: KeyboardEvent) {
  if (!props.open) return

  if (event.key === 'Tab') {
    event.preventDefault()
    event.stopPropagation()
    closeButtonRef.value?.focus({ preventScroll: true })
    return
  }

  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    emit('close')
  }
}

function startListeningForKeydown() {
  if (isListeningForKeydown) return
  window.addEventListener('keydown', handleKeydown, true)
  isListeningForKeydown = true
}

function stopListeningForKeydown() {
  if (!isListeningForKeydown) return
  window.removeEventListener('keydown', handleKeydown, true)
  isListeningForKeydown = false
}

function restorePreviousFocus() {
  const focusTarget = previouslyFocusedElement
  previouslyFocusedElement = null
  if (focusTarget?.isConnected) {
    focusTarget.focus({ preventScroll: true })
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      resetTransform()
      previouslyFocusedElement = document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null
      startListeningForKeydown()
      void nextTick(() => {
        if (props.open) {
          closeButtonRef.value?.focus({ preventScroll: true })
        }
      })
      return
    }

    stopListeningForKeydown()
    restorePreviousFocus()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  stopListeningForKeydown()
  restorePreviousFocus()
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="preview-mask"
      role="dialog"
      aria-modal="true"
      :aria-label="dialogLabel"
      @click.self="emit('close')"
    >
      <button
        ref="closeButtonRef"
        type="button"
        class="preview-close"
        :aria-label="t('common.imagePreview.close')"
        :title="t('common.imagePreview.close')"
        @click.stop="emit('close')"
      >
        <i class="ri-close-line" aria-hidden="true" />
      </button>
      <div
        class="preview-viewport"
        @touchstart="onTouchStart"
        @touchmove="onTouchMove"
        @touchend="onTouchEnd"
        @touchcancel="onTouchEnd"
      >
        <img
          v-if="props.src"
          class="preview-image"
          :src="props.src"
          :alt="alt"
          :style="imageStyle"
          @click.stop
          @dblclick="onDoubleClick"
        />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.preview-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.86);
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 14px;
}

.preview-close {
  position: absolute;
  top: calc(var(--safe-top, 0px) + 12px);
  right: calc(var(--safe-right, 0px) + 12px);
  z-index: 2;
  width: 48px;
  height: 48px;
  border: 1px solid rgba(255, 255, 255, 0.35);
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.35);
  color: #fff;
  font-size: 24px;
  touch-action: manipulation;
}

.preview-viewport {
  width: 100%;
  height: 100%;
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  touch-action: none;
}

.preview-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
  user-select: none;
  transform-origin: center center;
  transition: transform 0.08s linear;
}

@media (prefers-reduced-motion: reduce) {
  .preview-image {
    transition: none;
  }
}
</style>

