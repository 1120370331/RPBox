<script setup lang="ts">
import { ref, computed } from 'vue'
import WowIcon from './WowIcon.vue'
import RInput from './RInput.vue'
import RButton from './RButton.vue'

interface Props {
  modelValue: string
  fallbackIcon?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// 模式: 'wow' | 'upload'
const mode = ref<'wow' | 'upload'>('wow')

// WoW图标名称
const wowIconName = ref('')

// 上传的图片URL
const uploadedUrl = ref('')

// 初始化
if (props.modelValue) {
  if (props.modelValue.startsWith('http') || props.modelValue.startsWith('data:')) {
    mode.value = 'upload'
    uploadedUrl.value = props.modelValue
  } else {
    mode.value = 'wow'
    wowIconName.value = props.modelValue
  }
}

// 当前显示的值
const displayValue = computed(() => {
  if (mode.value === 'wow') {
    return wowIconName.value
  }
  return uploadedUrl.value
})

// 是否是图片URL
const isImageUrl = computed(() => {
  return displayValue.value.startsWith('http') || displayValue.value.startsWith('data:')
})

function switchMode(newMode: 'wow' | 'upload') {
  mode.value = newMode
  updateValue()
}

function updateValue() {
  if (mode.value === 'wow') {
    emit('update:modelValue', wowIconName.value)
  } else {
    emit('update:modelValue', uploadedUrl.value)
  }
}

function onWowIconChange(val: string) {
  wowIconName.value = val
  updateValue()
}

// 处理文件上传
const fileInput = ref<HTMLInputElement>()

function triggerUpload() {
  fileInput.value?.click()
}

function handleFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  // 检查文件大小 (最大 500KB)
  if (file.size > 500 * 1024) {
    alert('图片大小不能超过 500KB')
    return
  }

  // 转为 base64
  const reader = new FileReader()
  reader.onload = () => {
    uploadedUrl.value = reader.result as string
    updateValue()
  }
  reader.readAsDataURL(file)
}

function clearAvatar() {
  wowIconName.value = ''
  uploadedUrl.value = ''
  emit('update:modelValue', '')
}
</script>

<template>
  <div class="avatar-picker">
    <!-- 预览区 -->
    <div class="preview-section">
      <div class="avatar-preview">
        <img v-if="isImageUrl" :src="displayValue" alt="avatar" />
        <WowIcon v-else-if="displayValue" :icon="displayValue" :size="80" />
        <span v-else class="placeholder">{{ fallbackIcon || '?' }}</span>
      </div>
      <button v-if="displayValue" class="btn-clear" @click="clearAvatar">
        <i class="ri-close-line"></i>
      </button>
    </div>

    <!-- 模式切换 -->
    <div class="mode-tabs">
      <button
        :class="{ active: mode === 'wow' }"
        @click="switchMode('wow')"
      >
        <i class="ri-gamepad-line"></i> WoW图标
      </button>
      <button
        :class="{ active: mode === 'upload' }"
        @click="switchMode('upload')"
      >
        <i class="ri-upload-2-line"></i> 上传图片
      </button>
    </div>

    <!-- WoW图标输入 -->
    <div v-if="mode === 'wow'" class="input-section">
      <RInput
        :model-value="wowIconName"
        @update:model-value="onWowIconChange"
        placeholder="输入WoW图标名称，如 inv_misc_questionmark"
      />
      <span class="hint">可在 wowhead.com 查找图标名称</span>
    </div>

    <!-- 上传图片 -->
    <div v-else class="input-section">
      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        style="display: none"
        @change="handleFileChange"
      />
      <RButton @click="triggerUpload">
        <i class="ri-upload-2-line"></i> 选择图片
      </RButton>
      <span class="hint">支持 JPG/PNG，最大 500KB</span>
    </div>
  </div>
</template>

<style scoped>
.avatar-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.preview-section {
  position: relative;
  width: fit-content;
}

.avatar-preview {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-bg-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px dashed var(--color-border);
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-preview .placeholder {
  font-size: 32px;
  color: var(--color-secondary);
}

.btn-clear {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: none;
  background: #e74c3c;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}

.mode-tabs {
  display: flex;
  gap: 8px;
}

.mode-tabs button {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  background: #fff;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: var(--color-secondary);
  transition: all 0.2s;
}

.mode-tabs button:hover {
  border-color: var(--color-accent);
}

.mode-tabs button.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}

.input-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.hint {
  font-size: 12px;
  color: var(--color-secondary);
}
</style>
