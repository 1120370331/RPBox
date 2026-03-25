<script setup lang="ts">
import { useToastStore } from '@/stores/toast'

const toastStore = useToastStore()

const icons = {
  success: '✓',
  error: '✕',
  warning: '!',
  info: 'i',
}
</script>

<template>
  <Teleport to="body">
    <div class="r-toast-container">
      <TransitionGroup name="r-toast">
        <div v-for="t in toastStore.toasts" :key="t.id" class="r-toast" :class="`r-toast--${t.type}`">
          <span class="r-toast__icon">{{ icons[t.type] }}</span>
          <span class="r-toast__message">{{ t.message }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.r-toast-container {
  position: fixed;
  top: 24px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2000;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.r-toast {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  border-radius: var(--radius-sm);
  box-shadow: 0 4px 16px rgba(var(--shadow-base), 0.25);
  font-size: 14px;
}

.r-toast__icon {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
}

.r-toast--success { background: var(--color-success-light); color: var(--color-success); }
.r-toast--success .r-toast__icon { background: var(--color-success); color: var(--color-text-light); }

.r-toast--error { background: var(--btn-secondary-bg); color: var(--btn-danger-bg); }
.r-toast--error .r-toast__icon { background: var(--btn-danger-bg); color: var(--color-text-light); }

.r-toast--warning { background: var(--color-warning-light); color: var(--color-warning-dark); }
.r-toast--warning .r-toast__icon { background: var(--color-warning-dark); color: var(--color-text-light); }

.r-toast--info { background: var(--btn-secondary-bg); color: var(--link-color); }
.r-toast--info .r-toast__icon { background: var(--link-color); color: var(--color-text-light); }

.r-toast-enter-active, .r-toast-leave-active { transition: all 0.3s; }
.r-toast-enter-from { opacity: 0; transform: translateY(-20px); }
.r-toast-leave-to { opacity: 0; transform: translateX(100px); }
</style>
