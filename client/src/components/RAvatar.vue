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
      background: color || 'radial-gradient(circle at 30% 24%, rgba(255, 255, 255, 0.72), transparent 34%), linear-gradient(135deg, var(--gradient-start), var(--gradient-end))',
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
  color: var(--btn-primary-text, var(--color-primary));
  font-weight: 600;
  overflow: hidden;
  flex-shrink: 0;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.36);
}
.r-avatar--circle { border-radius: 50%; }
.r-avatar--square { border-radius: var(--radius-sm); }
.r-avatar__img { width: 100%; height: 100%; object-fit: cover; }
</style>
