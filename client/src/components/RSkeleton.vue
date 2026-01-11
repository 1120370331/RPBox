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
.r-skeleton__avatar { width: 48px; height: 48px; border-radius: 50%; background: rgba(75,54,33,0.1); flex-shrink: 0; }
.r-skeleton__content { flex: 1; display: flex; flex-direction: column; gap: 12px; }
.r-skeleton__row { height: 16px; background: rgba(75,54,33,0.1); border-radius: 4px; }

.r-skeleton--animated .r-skeleton__avatar,
.r-skeleton--animated .r-skeleton__row {
  background: linear-gradient(90deg, rgba(75,54,33,0.1) 25%, rgba(75,54,33,0.05) 50%, rgba(75,54,33,0.1) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer { to { background-position: -200% 0; } }
</style>
