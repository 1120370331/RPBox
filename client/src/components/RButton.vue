<script setup lang="ts">
interface Props {
  type?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  disabled?: boolean
  block?: boolean
  icon?: string
}

const props = withDefaults(defineProps<Props>(), {
  type: 'primary',
  size: 'md',
  loading: false,
  disabled: false,
  block: false,
})

const emit = defineEmits<{
  click: [e: MouseEvent]
}>()

function handleClick(e: MouseEvent) {
  if (!props.loading && !props.disabled) {
    emit('click', e)
  }
}
</script>

<template>
  <button
    class="r-button"
    :class="[
      `r-button--${type}`,
      `r-button--${size}`,
      { 'r-button--loading': loading },
      { 'r-button--disabled': disabled },
      { 'r-button--block': block },
    ]"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <span v-if="loading" class="r-button__spinner"></span>
    <span v-if="icon && !loading" class="r-button__icon">{{ icon }}</span>
    <span class="r-button__text"><slot /></span>
  </button>
</template>

<style scoped>
.r-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-sm);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

/* 尺寸 */
.r-button--sm { padding: 8px 16px; font-size: 12px; }
.r-button--md { padding: 12px 24px; font-size: 14px; }
.r-button--lg { padding: 16px 32px; font-size: 16px; }

/* 类型 */
.r-button--primary {
  background: var(--color-accent);
  color: var(--color-primary);
}
.r-button--primary:hover { filter: brightness(1.1); }

.r-button--secondary {
  background: var(--color-primary);
  color: var(--text-light);
}
.r-button--secondary:hover { filter: brightness(1.2); }

.r-button--outline {
  background: transparent;
  border: 2px solid var(--color-accent);
  color: var(--color-accent);
}
.r-button--outline:hover { background: rgba(184, 115, 51, 0.1); }

.r-button--ghost {
  background: transparent;
  color: var(--color-primary);
}
.r-button--ghost:hover { background: rgba(75, 54, 33, 0.1); }

.r-button--danger {
  background: #c41e3a;
  color: #fff;
}
.r-button--danger:hover { filter: brightness(1.1); }

/* 状态 */
.r-button--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.r-button--block {
  width: 100%;
}

.r-button--loading {
  cursor: wait;
}

/* 加载动画 */
.r-button__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
