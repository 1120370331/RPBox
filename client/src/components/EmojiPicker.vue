<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'

const props = defineProps<{
  show: boolean
  triggerElement?: HTMLElement | null
}>()

const emit = defineEmits<{
  'select': [emoji: string]
  'close': []
}>()

const pickerStyle = ref<any>({})

// 监听 show 变化，计算位置
watch(() => props.show, async (newShow) => {
  if (newShow && props.triggerElement) {
    await nextTick()
    const rect = props.triggerElement.getBoundingClientRect()

    // 计算选择器位置（按钮右下方）
    pickerStyle.value = {
      position: 'fixed',
      top: `${rect.bottom + 8}px`,
      left: `${rect.left}px`,
      zIndex: 1000
    }
  }
})

function handleSelect(emoji: any) {
  emit('select', emoji.i)
  emit('close')
}

function handleClose() {
  emit('close')
}
</script>

<template>
  <div v-if="show" class="emoji-picker-overlay" @click.self="handleClose">
    <div class="emoji-picker-container" :style="pickerStyle">
      <EmojiPicker
        :native="true"
        locale="zh"
        @select="handleSelect"
      />
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
  z-index: 999;
}

.emoji-picker-container {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}
</style>
