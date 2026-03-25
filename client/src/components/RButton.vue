<script setup lang="ts">
interface Props {
  type?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger'
  size?: 'sm' | 'small' | 'md' | 'lg'
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
  border: 1px solid;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

/* 尺寸 */
.r-button--sm, .r-button--small { padding: 6px 12px; font-size: 12px; }
.r-button--md { padding: 10px 20px; font-size: 13px; }
.r-button--lg { padding: 14px 28px; font-size: 15px; }

/* 类型 */
.r-button--primary {
  background: var(--btn-primary-bg);
  border-color: var(--btn-primary-bg);
  color: var(--btn-primary-text);
  box-shadow: 0 1px 3px rgba(var(--shadow-base), 0.2);
}
.r-button--primary:hover {
  background: var(--btn-primary-hover);
  box-shadow: 0 4px 6px rgba(var(--shadow-base), 0.24);
}

.r-button--secondary {
  background: var(--btn-secondary-bg);
  border-color: var(--btn-outline-border);
  color: var(--btn-secondary-text);
  box-shadow: 0 1px 2px rgba(var(--shadow-base), 0.12);
}
.r-button--secondary:hover {
  background: var(--btn-secondary-hover);
  box-shadow: 0 2px 4px rgba(var(--shadow-base), 0.16);
}

.r-button--outline {
  background: transparent;
  border-color: var(--btn-outline-border);
  color: var(--btn-outline-text);
}
.r-button--outline:hover {
  background: var(--btn-outline-hover);
}

.r-button--ghost {
  background: transparent;
  border-color: transparent;
  color: var(--color-text-main);
}
.r-button--ghost:hover {
  background: var(--btn-outline-hover);
}

.r-button--danger {
  background: var(--btn-danger-bg);
  border-color: var(--btn-danger-bg);
  color: var(--color-text-light);
  box-shadow: 0 1px 3px rgba(var(--shadow-base), 0.2);
}
.r-button--danger:hover {
  background: var(--btn-danger-hover);
  box-shadow: 0 4px 6px rgba(var(--shadow-base), 0.24);
}

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
