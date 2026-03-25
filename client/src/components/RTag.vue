<script setup lang="ts">
interface Props {
  type?: 'default' | 'primary' | 'success' | 'warning' | 'danger'
  size?: 'sm' | 'md'
  closable?: boolean
}

withDefaults(defineProps<Props>(), {
  type: 'default',
  size: 'md',
})

const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <span class="r-tag" :class="[`r-tag--${type}`, `r-tag--${size}`]">
    <slot />
    <span v-if="closable" class="r-tag__close" @click="emit('close')">×</span>
  </span>
</template>

<style scoped>
.r-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border-radius: 4px;
  font-weight: 500;
}

.r-tag--sm { padding: 2px 8px; font-size: 11px; }
.r-tag--md { padding: 4px 10px; font-size: 12px; }

.r-tag--default { background: var(--btn-secondary-bg); color: var(--color-text-main); }
.r-tag--primary { background: var(--tag-bg); color: var(--tag-text); }
.r-tag--success { background: var(--color-success-light); color: var(--color-success); }
.r-tag--warning { background: var(--color-warning-light); color: var(--color-warning-dark); }
.r-tag--danger { background: var(--btn-secondary-bg); color: var(--btn-danger-bg); }

.r-tag__close {
  cursor: pointer;
  font-size: 14px;
  opacity: 0.6;
}
.r-tag__close:hover { opacity: 1; }
</style>
