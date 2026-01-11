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

.r-tag--default { background: rgba(75,54,33,0.1); color: var(--color-primary); }
.r-tag--primary { background: rgba(184,115,51,0.15); color: var(--color-accent); }
.r-tag--success { background: rgba(46,125,50,0.15); color: #2e7d32; }
.r-tag--warning { background: rgba(230,81,0,0.15); color: #e65100; }
.r-tag--danger { background: rgba(196,30,58,0.15); color: #c41e3a; }

.r-tag__close {
  cursor: pointer;
  font-size: 14px;
  opacity: 0.6;
}
.r-tag__close:hover { opacity: 1; }
</style>
