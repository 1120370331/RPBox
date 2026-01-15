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
  box-shadow: 0 4px 16px rgba(0,0,0,0.15);
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

.r-toast--success { background: #e8f5e9; color: #2e7d32; }
.r-toast--success .r-toast__icon { background: #2e7d32; color: #fff; }

.r-toast--error { background: #ffebee; color: #c62828; }
.r-toast--error .r-toast__icon { background: #c62828; color: #fff; }

.r-toast--warning { background: #fff3e0; color: #e65100; }
.r-toast--warning .r-toast__icon { background: #e65100; color: #fff; }

.r-toast--info { background: #e3f2fd; color: #1565c0; }
.r-toast--info .r-toast__icon { background: #1565c0; color: #fff; }

.r-toast-enter-active, .r-toast-leave-active { transition: all 0.3s; }
.r-toast-enter-from { opacity: 0; transform: translateY(-20px); }
.r-toast-leave-to { opacity: 0; transform: translateX(100px); }
</style>
