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

    // 表情选择器的大小（vue3-emoji-picker 默认大小）
    const pickerWidth = 352
    const pickerHeight = 435

    // 视口大小
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    // 默认位置：按钮右下方
    let top = rect.bottom + 8
    let left = rect.left

    // 检测是否超出右侧边界
    if (left + pickerWidth > viewportWidth) {
      // 改为显示在按钮右侧对齐
      left = viewportWidth - pickerWidth - 16
    }

    // 检测是否超出底部边界
    if (top + pickerHeight > viewportHeight) {
      // 改为显示在按钮上方
      top = rect.top - pickerHeight - 8

      // 如果上方也放不下，则显示在视口顶部
      if (top < 16) {
        top = 16
      }
    }

    pickerStyle.value = {
      position: 'fixed',
      top: `${top}px`,
      left: `${left}px`,
      zIndex: 1000,
      maxHeight: `${viewportHeight - top - 16}px`
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
  overflow: auto;
}
</style>
