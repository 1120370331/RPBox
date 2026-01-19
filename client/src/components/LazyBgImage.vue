<template>
  <div ref="container" class="lazy-bg" :style="computedStyle">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  src?: string
  fallbackGradient?: string
  fallbackColor?: string
}>()

const container = ref<HTMLElement>()
const shouldLoad = ref(false)

const computedStyle = computed(() => {
  // 如果应该加载且有图片 src，使用背景图
  if (shouldLoad.value && props.src) {
    return { backgroundImage: `url(${props.src})` }
  }
  // 否则使用 fallback 渐变或颜色
  if (props.fallbackGradient) {
    return { background: props.fallbackGradient }
  }
  if (props.fallbackColor) {
    return { background: props.fallbackColor }
  }
  return {}
})

let observer: IntersectionObserver | null = null

onMounted(() => {
  // 如果没有图片 src，不需要懒加载
  if (!props.src) {
    shouldLoad.value = true
    return
  }

  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting) {
        shouldLoad.value = true
        observer?.disconnect()
      }
    },
    {
      rootMargin: '200px',  // 提前 200px 开始加载
      threshold: 0
    }
  )

  if (container.value) {
    observer.observe(container.value)
  }
})

onUnmounted(() => {
  observer?.disconnect()
})
</script>

<style scoped>
.lazy-bg {
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}
</style>
