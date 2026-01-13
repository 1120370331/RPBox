<script setup lang="ts">
import { inject, computed, onMounted } from 'vue'

interface Props {
  name: string
  label: string
}

const props = defineProps<Props>()
const tabs = inject<any>('tabs')

const isActive = computed(() => tabs?.activeTab.value === props.name)

// 在挂载时注册到父组件
onMounted(() => {
  if (tabs?.registerTab) {
    tabs.registerTab({ name: props.name, label: props.label })
  }
})
</script>

<template>
  <!-- 只渲染内容区，标签头由父组件 RTabs 统一渲染 -->
  <div v-show="isActive" class="r-tab-pane">
    <slot />
  </div>
</template>

<style scoped>
.r-tab-pane {
  /* 内容区不需要额外 padding，由父组件 r-tabs__content 控制 */
}
</style>
