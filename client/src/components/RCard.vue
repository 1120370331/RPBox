<script setup lang="ts">
interface Props {
  title?: string
  subtitle?: string
  hoverable?: boolean
  bordered?: boolean
  shadow?: 'none' | 'sm' | 'md' | 'lg'
  padding?: 'none' | 'sm' | 'md' | 'lg'
}

withDefaults(defineProps<Props>(), {
  hoverable: false,
  bordered: true,
  shadow: 'none',
  padding: 'md',
})
</script>

<template>
  <div
    class="r-card"
    :class="[
      `r-card--shadow-${shadow}`,
      `r-card--padding-${padding}`,
      { 'r-card--hoverable': hoverable },
      { 'r-card--bordered': bordered },
    ]"
  >
    <div v-if="title || $slots.header" class="r-card__header">
      <slot name="header">
        <div class="r-card__title">{{ title }}</div>
        <div v-if="subtitle" class="r-card__subtitle">{{ subtitle }}</div>
      </slot>
    </div>
    <div class="r-card__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="r-card__footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.r-card {
  background: #fff;
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: all 0.2s;
}

.r-card--bordered { border: 1px solid rgba(75, 54, 33, 0.1); }
.r-card--hoverable:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(75, 54, 33, 0.12);
}

.r-card--shadow-sm { box-shadow: 0 2px 8px rgba(75, 54, 33, 0.08); }
.r-card--shadow-md { box-shadow: 0 4px 16px rgba(75, 54, 33, 0.1); }
.r-card--shadow-lg { box-shadow: 0 8px 32px rgba(75, 54, 33, 0.12); }

.r-card--padding-none .r-card__body { padding: 0; }
.r-card--padding-sm .r-card__body { padding: 12px; }
.r-card--padding-md .r-card__body { padding: 20px; }
.r-card--padding-lg .r-card__body { padding: 28px; }

.r-card__header {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(75, 54, 33, 0.08);
}

.r-card__title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-primary);
}

.r-card__subtitle {
  font-size: 13px;
  color: var(--color-secondary);
  margin-top: 4px;
}

.r-card__footer {
  padding: 12px 20px;
  border-top: 1px solid rgba(75, 54, 33, 0.08);
  background: rgba(75, 54, 33, 0.02);
}
</style>
