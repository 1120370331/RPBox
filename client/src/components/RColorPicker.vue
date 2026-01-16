<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'

interface Props {
  modelValue: string
  presets?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  presets: () => [
    'FF6B6B', 'FF8E53', 'FFC93C', '6BCB77', '4D96FF',
    '9B59B6', 'E91E63', '00BCD4', '8D6E63', '607D8B',
    'FFFFFF', 'CCCCCC', '999999', '666666', '333333'
  ]
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const showPicker = ref(false)
const pickerRef = ref<HTMLElement>()
const saturationRef = ref<HTMLElement>()

// HSV 值
const hue = ref(0)
const saturation = ref(100)
const brightness = ref(100)

// 是否正在拖拽
const isDraggingSaturation = ref(false)
const isDraggingHue = ref(false)

// 显示的颜色值（不带#）
const displayValue = computed({
  get: () => props.modelValue || '',
  set: (val) => {
    const hex = val.replace('#', '').toUpperCase().slice(0, 6)
    emit('update:modelValue', hex)
  }
})

// 当前颜色的 hex 值
const currentColor = computed(() => {
  return hsvToHex(hue.value, saturation.value, brightness.value)
})

// 色相对应的纯色
const hueColor = computed(() => {
  return hsvToHex(hue.value, 100, 100)
})

// HSV 转 Hex
function hsvToHex(h: number, s: number, v: number): string {
  s = s / 100
  v = v / 100
  const c = v * s
  const x = c * (1 - Math.abs((h / 60) % 2 - 1))
  const m = v - c
  let r = 0, g = 0, b = 0

  if (h < 60) { r = c; g = x; b = 0 }
  else if (h < 120) { r = x; g = c; b = 0 }
  else if (h < 180) { r = 0; g = c; b = x }
  else if (h < 240) { r = 0; g = x; b = c }
  else if (h < 300) { r = x; g = 0; b = c }
  else { r = c; g = 0; b = x }

  const toHex = (n: number) => Math.round((n + m) * 255).toString(16).padStart(2, '0')
  return (toHex(r) + toHex(g) + toHex(b)).toUpperCase()
}

// Hex 转 HSV
function hexToHsv(hex: string): { h: number; s: number; v: number } {
  const r = parseInt(hex.slice(0, 2), 16) / 255
  const g = parseInt(hex.slice(2, 4), 16) / 255
  const b = parseInt(hex.slice(4, 6), 16) / 255

  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const d = max - min

  let h = 0
  if (d !== 0) {
    if (max === r) h = ((g - b) / d) % 6
    else if (max === g) h = (b - r) / d + 2
    else h = (r - g) / d + 4
    h = Math.round(h * 60)
    if (h < 0) h += 360
  }

  const s = max === 0 ? 0 : Math.round((d / max) * 100)
  const v = Math.round(max * 100)

  return { h, s, v }
}

// 从 modelValue 初始化 HSV
function initFromModelValue() {
  if (props.modelValue && props.modelValue.length === 6) {
    const hsv = hexToHsv(props.modelValue)
    hue.value = hsv.h
    saturation.value = hsv.s
    brightness.value = hsv.v
  }
}

// 饱和度/亮度面板点击
function handleSaturationMouseDown(e: MouseEvent) {
  isDraggingSaturation.value = true
  updateSaturationFromEvent(e)
}

function updateSaturationFromEvent(e: MouseEvent) {
  if (!saturationRef.value) return
  const rect = saturationRef.value.getBoundingClientRect()
  const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width))
  const y = Math.max(0, Math.min(e.clientY - rect.top, rect.height))
  saturation.value = Math.round((x / rect.width) * 100)
  brightness.value = Math.round((1 - y / rect.height) * 100)
  emit('update:modelValue', currentColor.value)
}

// 色相滑块
function handleHueMouseDown(e: MouseEvent) {
  isDraggingHue.value = true
  updateHueFromEvent(e)
}

function updateHueFromEvent(e: MouseEvent) {
  const target = e.currentTarget as HTMLElement || document.querySelector('.hue-slider')
  if (!target) return
  const rect = target.getBoundingClientRect()
  const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width))
  hue.value = Math.round((x / rect.width) * 360)
  emit('update:modelValue', currentColor.value)
}

function handleMouseMove(e: MouseEvent) {
  if (isDraggingSaturation.value) {
    updateSaturationFromEvent(e)
  } else if (isDraggingHue.value) {
    const slider = document.querySelector('.hue-slider') as HTMLElement
    if (slider) {
      const rect = slider.getBoundingClientRect()
      const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width))
      hue.value = Math.round((x / rect.width) * 360)
      emit('update:modelValue', currentColor.value)
    }
  }
}

function handleMouseUp() {
  isDraggingSaturation.value = false
  isDraggingHue.value = false
}

function selectPreset(color: string) {
  emit('update:modelValue', color)
  const hsv = hexToHsv(color)
  hue.value = hsv.h
  saturation.value = hsv.s
  brightness.value = hsv.v
}

function handleClickOutside(e: MouseEvent) {
  if (pickerRef.value && !pickerRef.value.contains(e.target as Node)) {
    showPicker.value = false
  }
}

watch(showPicker, (val) => {
  if (val) {
    initFromModelValue()
    setTimeout(() => {
      document.addEventListener('click', handleClickOutside)
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
    }, 10)
  } else {
    document.removeEventListener('click', handleClickOutside)
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('mouseup', handleMouseUp)
  }
})

watch(() => props.modelValue, (val) => {
  if (val && val.length === 6 && val !== currentColor.value) {
    initFromModelValue()
  }
})

onMounted(() => {
  initFromModelValue()
})
</script>

<template>
  <div class="r-color-picker" ref="pickerRef">
    <div class="color-input-wrapper" @click="showPicker = !showPicker">
      <span
        class="color-preview"
        :style="{ background: '#' + (modelValue || 'FFFFFF') }"
      ></span>
      <input
        type="text"
        v-model="displayValue"
        placeholder="FFFFFF"
        maxlength="6"
        @click.stop
      />
    </div>

    <Transition name="picker-fade">
      <div v-if="showPicker" class="picker-dropdown" @click.stop>
        <!-- 饱和度/亮度面板 -->
        <div
          ref="saturationRef"
          class="saturation-panel"
          :style="{ background: '#' + hueColor }"
          @mousedown="handleSaturationMouseDown"
        >
          <div class="saturation-white"></div>
          <div class="saturation-black"></div>
          <div
            class="saturation-pointer"
            :style="{
              left: saturation + '%',
              top: (100 - brightness) + '%',
              background: '#' + currentColor
            }"
          ></div>
        </div>

        <!-- 色相滑块 -->
        <div class="hue-slider" @mousedown="handleHueMouseDown">
          <div
            class="hue-pointer"
            :style="{ left: (hue / 360 * 100) + '%' }"
          ></div>
        </div>

        <!-- 当前颜色预览 -->
        <div class="color-result">
          <div
            class="result-preview"
            :style="{ background: '#' + currentColor }"
          ></div>
          <span class="result-hex">#{{ currentColor }}</span>
        </div>

        <!-- 预设颜色 -->
        <div class="preset-colors">
          <div
            v-for="color in presets"
            :key="color"
            class="preset-item"
            :class="{ active: modelValue === color }"
            :style="{ background: '#' + color }"
            @click="selectPreset(color)"
          ></div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.r-color-picker {
  position: relative;
}

.color-input-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  cursor: pointer;
  background: #fff;
}

.color-input-wrapper:hover {
  border-color: var(--color-accent);
}

.color-preview {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  border: 1px solid rgba(0,0,0,0.1);
  flex-shrink: 0;
}

.color-input-wrapper input {
  border: none;
  outline: none;
  font-size: 14px;
  font-family: monospace;
  width: 70px;
  color: var(--color-primary);
  background: transparent;
}

.picker-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.2);
  padding: 12px;
  z-index: 100;
  width: 240px;
}

/* 饱和度/亮度面板 */
.saturation-panel {
  position: relative;
  width: 100%;
  height: 150px;
  border-radius: 8px;
  cursor: crosshair;
  margin-bottom: 12px;
}

.saturation-white {
  position: absolute;
  inset: 0;
  background: linear-gradient(to right, #fff, transparent);
  border-radius: 8px;
}

.saturation-black {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, #000, transparent);
  border-radius: 8px;
}

.saturation-pointer {
  position: absolute;
  width: 14px;
  height: 14px;
  border: 2px solid #fff;
  border-radius: 50%;
  box-shadow: 0 1px 4px rgba(0,0,0,0.3);
  transform: translate(-50%, -50%);
  pointer-events: none;
}

/* 色相滑块 */
.hue-slider {
  position: relative;
  width: 100%;
  height: 14px;
  border-radius: 7px;
  background: linear-gradient(to right,
    #ff0000 0%,
    #ffff00 17%,
    #00ff00 33%,
    #00ffff 50%,
    #0000ff 67%,
    #ff00ff 83%,
    #ff0000 100%
  );
  cursor: pointer;
  margin-bottom: 12px;
}

.hue-pointer {
  position: absolute;
  top: 50%;
  width: 6px;
  height: 18px;
  background: #fff;
  border-radius: 3px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.3);
  transform: translate(-50%, -50%);
  pointer-events: none;
}

/* 颜色结果 */
.color-result {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border);
}

.result-preview {
  width: 36px;
  height: 36px;
  border-radius: 6px;
  border: 1px solid rgba(0,0,0,0.1);
}

.result-hex {
  font-family: monospace;
  font-size: 14px;
  color: var(--color-primary);
  font-weight: 500;
}

.preset-colors {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 6px;
}

.preset-item {
  width: 28px;
  height: 28px;
  border-radius: 4px;
  cursor: pointer;
  border: 2px solid transparent;
  transition: all 0.15s;
}

.preset-item:hover {
  transform: scale(1.1);
}

.preset-item.active {
  border-color: var(--color-accent);
}

.picker-fade-enter-active,
.picker-fade-leave-active {
  transition: all 0.2s ease;
}

.picker-fade-enter-from,
.picker-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
