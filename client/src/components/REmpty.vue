<script setup lang="ts">
interface Props {
  icon?: string
  title?: string
  description?: string
}

withDefaults(defineProps<Props>(), {
  icon: 'ri-inbox-2-line',
  title: '暂无数据',
})
</script>

<template>
  <div class="r-empty">
    <div class="r-empty__icon">
      <i v-if="icon?.startsWith('ri-')" :class="icon"></i>
      <span v-else>{{ icon }}</span>
    </div>
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

.r-empty__icon {
  width: 58px;
  height: 58px;
  margin-bottom: 16px;
  border-radius: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at 28% 22%, rgba(255, 255, 255, 0.9), transparent 34%),
    linear-gradient(
      135deg,
      color-mix(in srgb, var(--color-panel-bg, #fff) 64%, var(--color-secondary, #D4A373)),
      color-mix(in srgb, var(--color-panel-bg, #fff) 78%, var(--color-accent, #8C7B70))
    );
  color: var(--icon-color, var(--color-primary));
  border: 1px solid color-mix(in srgb, var(--color-secondary, #D4A373) 26%, transparent);
  box-shadow: 0 10px 24px rgba(var(--shadow-base, 75, 54, 33), 0.1);
}

.r-empty__icon i,
.r-empty__icon span {
  font-size: 30px;
  line-height: 1;
}

.r-empty__title { font-size: 15px; color: var(--color-primary); margin-bottom: 4px; }
.r-empty__desc { font-size: 13px; color: var(--link-color, var(--color-secondary)); }
.r-empty__action { margin-top: 16px; }
</style>
