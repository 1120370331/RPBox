<script setup lang="ts">
interface Props {
  rows?: number
  avatar?: boolean
  animated?: boolean
}

withDefaults(defineProps<Props>(), {
  rows: 3,
  animated: true,
})
</script>

<template>
  <div class="r-skeleton" :class="{ 'r-skeleton--animated': animated }">
    <div v-if="avatar" class="r-skeleton__avatar"></div>
    <div class="r-skeleton__content">
      <div v-for="i in rows" :key="i" class="r-skeleton__row" :style="{ width: i === rows ? '60%' : '100%' }"></div>
    </div>
  </div>
</template>

<style scoped>
.r-skeleton { display: flex; gap: 16px; }
.r-skeleton__avatar { width: 48px; height: 48px; border-radius: 50%; background: var(--btn-secondary-bg); flex-shrink: 0; }
.r-skeleton__content { flex: 1; display: flex; flex-direction: column; gap: 12px; }
.r-skeleton__row { height: 16px; background: var(--btn-secondary-bg); border-radius: 4px; }

.r-skeleton--animated .r-skeleton__avatar,
.r-skeleton--animated .r-skeleton__row {
  background: linear-gradient(90deg, rgba(var(--shadow-base), 0.14) 25%, rgba(var(--shadow-base), 0.08) 50%, rgba(var(--shadow-base), 0.14) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer { to { background-position: -200% 0; } }
</style>
