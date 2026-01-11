<script setup lang="ts">
interface Props {
  size?: 'sm' | 'md' | 'lg'
  text?: string
}

withDefaults(defineProps<Props>(), {
  size: 'md',
})
</script>

<template>
  <div class="r-loading" :class="`r-loading--${size}`">
    <div class="r-loading__spinner"></div>
    <span v-if="text" class="r-loading__text">{{ text }}</span>
  </div>
</template>

<style scoped>
.r-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.r-loading__spinner {
  border: 3px solid rgba(75, 54, 33, 0.1);
  border-top-color: var(--color-accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.r-loading--sm .r-loading__spinner { width: 20px; height: 20px; }
.r-loading--md .r-loading__spinner { width: 32px; height: 32px; }
.r-loading--lg .r-loading__spinner { width: 48px; height: 48px; }

.r-loading__text {
  font-size: 14px;
  color: var(--color-secondary);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
