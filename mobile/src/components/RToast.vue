<script setup lang="ts">
import { useToastStore } from '@shared/stores/toast'

const toastStore = useToastStore()
</script>

<template>
  <Teleport to="body">
    <div class="toast-container">
      <TransitionGroup name="toast">
        <div
          v-for="item in toastStore.toasts"
          :key="item.id"
          class="toast-item"
          :class="'toast-' + item.type"
          @click="toastStore.remove(item.id)"
        >
          <i :class="iconClass(item.type)" />
          <span>{{ item.message }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script lang="ts">
function iconClass(type: string) {
  const map: Record<string, string> = {
    success: 'ri-check-line',
    error: 'ri-close-circle-line',
    warning: 'ri-alert-line',
    info: 'ri-information-line',
  }
  return map[type] || map.info
}
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: calc(var(--safe-top, 0px) + 12px);
  left: 16px;
  right: 16px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
}

.toast-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: var(--radius-sm);
  font-size: 14px;
  pointer-events: auto;
  box-shadow: var(--shadow-md);
}

.toast-success { background: var(--toast-success-bg); color: var(--toast-success-text); }
.toast-error { background: var(--toast-error-bg); color: var(--toast-error-text); }
.toast-warning { background: var(--toast-warning-bg); color: var(--toast-warning-text); }
.toast-info { background: var(--toast-info-bg); color: var(--toast-info-text); }

.toast-enter-active { transition: all 0.3s ease; }
.toast-leave-active { transition: all 0.2s ease; }
.toast-enter-from { opacity: 0; transform: translateY(-20px); }
.toast-leave-to { opacity: 0; transform: translateX(100%); }
</style>
