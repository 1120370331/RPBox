<script setup lang="ts">
interface Props {
  value?: number | string
  max?: number
  dot?: boolean
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  hidden?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  max: 99,
  type: 'danger',
})

const displayValue = typeof props.value === 'number' && props.value > props.max
  ? `${props.max}+`
  : props.value
</script>

<template>
  <div class="r-badge">
    <slot />
    <span
      v-if="!hidden && (value || dot)"
      class="r-badge__content"
      :class="[`r-badge--${type}`, { 'r-badge--dot': dot }]"
    >
      {{ dot ? '' : displayValue }}
    </span>
  </div>
</template>

<style scoped>
.r-badge {
  position: relative;
  display: inline-flex;
}

.r-badge__content {
  position: absolute;
  top: 0;
  right: 0;
  transform: translate(50%, -50%);
  padding: 2px 6px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
  color: #fff;
}

.r-badge--dot {
  width: 8px;
  height: 8px;
  min-width: 8px;
  padding: 0;
  border-radius: 50%;
}

.r-badge--primary { background: var(--color-accent); }
.r-badge--success { background: #2e7d32; }
.r-badge--warning { background: #e65100; }
.r-badge--danger { background: #c41e3a; }
.r-badge--info { background: #1565c0; }
</style>
