<script setup lang="ts">
interface Props {
  src?: string
  alt?: string
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | number
  shape?: 'circle' | 'square'
  text?: string
  color?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  shape: 'circle',
})

const sizeMap = { xs: 24, sm: 32, md: 40, lg: 56, xl: 80 }
const computedSize = typeof props.size === 'number' ? props.size : sizeMap[props.size]
</script>

<template>
  <div
    class="r-avatar"
    :class="`r-avatar--${shape}`"
    :style="{
      width: `${computedSize}px`,
      height: `${computedSize}px`,
      fontSize: `${computedSize * 0.4}px`,
      background: color || 'var(--color-accent)',
    }"
  >
    <img v-if="src" :src="src" :alt="alt" class="r-avatar__img" />
    <span v-else-if="text" class="r-avatar__text">{{ text }}</span>
    <slot v-else />
  </div>
</template>

<style scoped>
.r-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  font-weight: 600;
  overflow: hidden;
  flex-shrink: 0;
}
.r-avatar--circle { border-radius: 50%; }
.r-avatar--square { border-radius: var(--radius-sm); }
.r-avatar__img { width: 100%; height: 100%; object-fit: cover; }
</style>
