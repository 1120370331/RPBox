<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import 'emoji-picker-element'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  'select': [emoji: string]
  'close': []
}>()

const pickerRef = ref<HTMLElement | null>(null)

// 监听 show 变化，在显示时绑定事件
watch(() => props.show, async (newShow) => {
  if (newShow) {
    await nextTick()
    if (pickerRef.value) {
      const picker = pickerRef.value.querySelector('emoji-picker') as any
      if (picker) {
        // 移除旧的事件监听器（如果存在）
        picker.removeEventListener('emoji-click', handleEmojiClick)
        // 添加新的事件监听器
        picker.addEventListener('emoji-click', handleEmojiClick)
      }
    }
  }
})

function handleEmojiClick(event: any) {
  emit('select', event.detail.unicode)
}

function handleClose() {
  emit('close')
}
</script>

<template>
  <div v-if="show" class="emoji-picker-overlay" @click.self="handleClose">
    <div class="emoji-picker-container" ref="pickerRef">
      <emoji-picker></emoji-picker>
    </div>
  </div>
</template>

<style scoped>
.emoji-picker-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.emoji-picker-container {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}

emoji-picker {
  --border-radius: 12px;
  --background: #fff;
  --border-color: #E5D4C1;
}
</style>
