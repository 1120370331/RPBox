<script setup lang="ts">
interface Props {
  icon?: string
  title?: string
  description?: string
}

withDefaults(defineProps<Props>(), {
  icon: '📭',
  title: '暂无数据',
})
</script>

<template>
  <div class="r-empty">
    <div class="r-empty__icon">{{ icon }}</div>
    <div class="r-empty__title">{{ title }}</div>
    <div v-if="description" class="r-empty__desc">{{ description }}</div>
    <div v-if="$slots.default" class="r-empty__action"><slot /></div>
  </div>
</template>

<style scoped>
.r-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48px 24px;
}

.r-empty__icon { font-size: 48px; margin-bottom: 16px; opacity: 0.6; }
.r-empty__title { font-size: 15px; color: var(--color-primary); margin-bottom: 4px; }
.r-empty__desc { font-size: 13px; color: var(--color-secondary); }
.r-empty__action { margin-top: 16px; }
</style>
