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
          :class="[`toast-${item.type}`, item.rarity ? `toast-rarity-${item.rarity}` : '']"
          @click="toastStore.remove(item.id)"
        >
          <span class="toast-icon">
            <i v-if="item.type === 'achievement' && item.icon" :class="item.icon" />
            <i v-else :class="iconClass(item.type)" />
          </span>
          <span v-if="item.type === 'achievement'" class="toast-achievement-copy">
            <strong>{{ item.title || '成就解锁' }}</strong>
            <span>{{ item.message }}</span>
          </span>
          <span v-else>{{ item.message }}</span>
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
    achievement: 'ri-medal-line',
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

.toast-icon {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 22px;
}

.toast-success { background: var(--toast-success-bg); color: var(--toast-success-text); }
.toast-error { background: var(--toast-error-bg); color: var(--toast-error-text); }
.toast-warning { background: var(--toast-warning-bg); color: var(--toast-warning-text); }
.toast-info { background: var(--toast-info-bg); color: var(--toast-info-text); }
.toast-achievement {
  align-items: flex-start;
  border: 1px solid rgba(255, 201, 106, 0.34);
  background:
    radial-gradient(circle at 10% 20%, rgba(255, 204, 112, 0.22), transparent 34%),
    linear-gradient(135deg, #2D2118, #5B3519);
  color: #FFF4D8;
  box-shadow:
    0 18px 40px rgba(42, 25, 12, 0.34),
    inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}
.toast-achievement .toast-icon {
  background: linear-gradient(145deg, #FFD779, #B87333);
  color: #3E220F;
  box-shadow: 0 0 18px rgba(255, 196, 97, 0.42);
}
.toast-achievement-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.toast-achievement-copy strong {
  color: #FFF4D8;
  font-size: 13px;
}
.toast-achievement-copy span {
  color: rgba(255, 247, 224, 0.78);
  font-size: 12px;
  line-height: 1.35;
}
.toast-rarity-legendary {
  border-color: rgba(255, 178, 62, 0.62);
  box-shadow:
    0 18px 46px rgba(255, 178, 62, 0.24),
    inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}
.toast-rarity-epic {
  border-color: rgba(178, 108, 255, 0.52);
  box-shadow:
    0 18px 46px rgba(178, 108, 255, 0.2),
    inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.toast-enter-active { transition: all 0.3s ease; }
.toast-leave-active { transition: all 0.2s ease; }
.toast-enter-from { opacity: 0; transform: translateY(-20px); }
.toast-leave-to { opacity: 0; transform: translateX(100%); }
</style>
