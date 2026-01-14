<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface Props {
  modelValue: string
  presets?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  preValue: '',
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

// 内部颜色值（带#）
const internalColor = computed({
  get: () => props.modelValue ? `#${props.modelValue}` : '#FFFFFF',
  set: (val) => {
    const hex = val.replace('#', '').toUpperCase()
    emit('update:modelValue', hex)
  }
})

// 显示的颜色值（不带#）
const displayValue = computed({
  get: () => props.modelValue || '',
  set: (val) => {
    const hex = val.replace('#', '').toUpperCase().slice(0, 6)
    emit('update:modelValue', hex)
  }
})

function selectPreset(color: string) {
  emit('update:modelValue', color)
  showPicker.value = false
}

function handleClickOutside(e: MouseEvent) {
  if (pickerRef.value && !pickerRef.value.contains(e.target as Node)) {
    showPicker.value = false
  }
}

watch(showPicker, (val) => {
  if (val) {
    setTimeout(() => document.addEventListener('click', handleClickOutside), 10)
  } else {
    document.removeEventListener('click', handleClickOutside)
  }
})
</script>

<template>
  <div class="r-color-picker" ref="pickerRef">
    <div class="color-input-wrapper" @click="showPicker = !showPicker">
      <span
        class="color-preview"
        :style="{ background: internalColor }"
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
      <div v-if="showPicker" class="picker-dropdown">
        <div class="native-picker">
          <input type="color" v-model="internalColor" />
          <span>选择颜色</span>
        </div>
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
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.15);
  padding: 12px;
  z-index: 100;
  min-width: 200px;
}

.native-picker {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border);
}

.native-picker input[type="color"] {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  padding: 0;
}

.native-picker span {
  font-size: 13px;
  color: var(--color-secondary);
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
