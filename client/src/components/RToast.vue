<script setup lang="ts">
import { useToastStore } from '@/stores/toast'

const toastStore = useToastStore()

const icons = {
  success: '✓',
  error: '✕',
  warning: '!',
  info: 'i',
  achievement: '✦',
}
</script>

<template>
  <Teleport to="body">
    <div class="r-toast-container">
      <TransitionGroup name="r-toast">
        <div
          v-for="t in toastStore.toasts"
          :key="t.id"
          class="r-toast"
          :class="[`r-toast--${t.type}`, t.rarity ? `r-toast--rarity-${t.rarity}` : '']"
        >
          <span class="r-toast__icon">
            <i v-if="t.type === 'achievement' && t.icon" :class="t.icon"></i>
            <template v-else>{{ icons[t.type] }}</template>
          </span>
          <span v-if="t.type === 'achievement'" class="r-toast__achievement-copy">
            <strong>{{ t.title || '成就解锁' }}</strong>
            <span>{{ t.message }}</span>
          </span>
          <span v-else class="r-toast__message">{{ t.message }}</span>
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

.r-toast__icon i {
  font-size: 14px;
}

.r-toast__achievement-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.r-toast__achievement-copy strong {
  color: var(--achievement-toast-title, #FFF4D8);
  font-size: 13px;
}

.r-toast__achievement-copy span {
  color: rgba(255, 247, 224, 0.78);
  font-size: 12px;
}

.r-toast--success { background: var(--color-success-light); color: var(--color-success); }
.r-toast--success .r-toast__icon { background: var(--color-success); color: var(--color-text-light); }

.r-toast--error { background: var(--btn-secondary-bg); color: var(--btn-danger-bg); }
.r-toast--error .r-toast__icon { background: var(--btn-danger-bg); color: var(--color-text-light); }

.r-toast--warning { background: var(--color-warning-light); color: var(--color-warning-dark); }
.r-toast--warning .r-toast__icon { background: var(--color-warning-dark); color: var(--color-text-light); }

.r-toast--info { background: var(--btn-secondary-bg); color: var(--link-color); }
.r-toast--info .r-toast__icon { background: var(--link-color); color: var(--color-text-light); }

.r-toast--achievement {
  min-width: min(360px, calc(100vw - 32px));
  border: 1px solid rgba(255, 201, 106, 0.34);
  background:
    radial-gradient(circle at 10% 20%, rgba(255, 204, 112, 0.22), transparent 34%),
    linear-gradient(135deg, #2D2118, #5B3519);
  color: #FFF4D8;
  box-shadow:
    0 18px 40px rgba(42, 25, 12, 0.34),
    inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.r-toast--achievement .r-toast__icon {
  background: linear-gradient(145deg, #FFD779, #B87333);
  color: #3E220F;
  box-shadow: 0 0 18px rgba(255, 196, 97, 0.42);
}

.r-toast--rarity-legendary {
  border-color: rgba(255, 178, 62, 0.62);
  box-shadow:
    0 18px 46px rgba(255, 178, 62, 0.24),
    inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.r-toast--rarity-epic {
  border-color: rgba(178, 108, 255, 0.52);
  box-shadow:
    0 18px 46px rgba(178, 108, 255, 0.2),
    inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.r-toast-enter-active, .r-toast-leave-active { transition: all 0.3s; }
.r-toast-enter-from { opacity: 0; transform: translateY(-20px); }
.r-toast-leave-to { opacity: 0; transform: translateX(100px); }
</style>
