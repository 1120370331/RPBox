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
  background: #4B3621;
  border-color: #4B3621;
  color: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
}
.r-button--primary:hover {
  background: #3a2a18;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.15);
}

.r-button--secondary {
  background: #F5EFE7;
  border-color: #E5D4C1;
  color: #2C1810;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}
.r-button--secondary:hover {
  background: #fff;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.r-button--outline {
  background: transparent;
  border-color: #E5D4C1;
  color: #2C1810;
}
.r-button--outline:hover {
  background: #F5EFE7;
}

.r-button--ghost {
  background: transparent;
  border-color: transparent;
  color: #2C1810;
}
.r-button--ghost:hover {
  background: rgba(44, 24, 16, 0.05);
}

.r-button--danger {
  background: #c41e3a;
  border-color: #c41e3a;
  color: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
}
.r-button--danger:hover {
  background: #a01828;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.15);
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
