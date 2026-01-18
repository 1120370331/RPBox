<script setup lang="ts">
import { ref, watch, nextTick, defineAsyncComponent } from 'vue'

// 懒加载 emoji-picker 组件和样式
const EmojiPicker = defineAsyncComponent({
  loader: async () => {
    // 动态导入样式
    await import('vue3-emoji-picker/css')
    // 动态导入组件
    return import('vue3-emoji-picker')
  }
})

const props = defineProps<{
  show: boolean
  triggerElement?: HTMLElement | null
}>()

const emit = defineEmits<{
  'select': [emoji: string]
  'close': []
}>()

const pickerStyle = ref<any>({})
const isLoading = ref(false)

// 中文分组名称
const groupNames = {
  smileys_people: '笑脸和人物',
  animals_nature: '动物和自然',
  food_drink: '食物和饮料',
  activities: '活动',
  travel_places: '旅行和地点',
  objects: '物品',
  symbols: '符号',
  flags: '旗帜'
}

// 中文静态文本
const staticTexts = {
  placeholder: '搜索表情',
  skinTone: '肤色'
}

// 监听 show 变化，计算位置
watch(() => props.show, async (newShow) => {
  if (newShow && props.triggerElement) {
    isLoading.value = true
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

    // 延迟一帧后隐藏加载状态
    setTimeout(() => {
      isLoading.value = false
    }, 100)
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
      <div v-if="isLoading" class="loading-placeholder">
        <i class="ri-loader-4-line loading-icon"></i>
        <span>加载中...</span>
      </div>
      <EmojiPicker
        v-else
        :native="true"
        :group-names="groupNames"
        :static-texts="staticTexts"
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

.loading-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 40px;
  color: #8D7B68;
  font-size: 14px;
}

.loading-icon {
  font-size: 32px;
  margin-bottom: 12px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
