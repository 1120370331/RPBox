<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { dialog } from '../composables/useDialog'

const { t } = useI18n()

const typeIcons = {
  info: 'ri-information-line',
  success: 'ri-checkbox-circle-line',
  warning: 'ri-alert-line',
  error: 'ri-close-circle-line',
}

const typeColors = {
  info: '#1565c0',
  success: '#2e7d32',
  warning: '#e65100',
  error: '#c62828',
}
</script>

<template>
  <Teleport to="body">
    <Transition name="r-dialog">
      <div v-if="dialog.state.value.visible" class="r-dialog__mask" @click.self="dialog.close(false)">
        <div class="r-dialog">
          <div class="r-dialog__header">
            <i
              v-if="dialog.state.value.options.type"
              :class="typeIcons[dialog.state.value.options.type]"
              :style="{ color: typeColors[dialog.state.value.options.type] }"
            ></i>
            <span class="r-dialog__title">{{ dialog.state.value.options.title || t('common.status.loading').replace('...', '') }}</span>
          </div>
          <div class="r-dialog__body">
            {{ dialog.state.value.options.message }}
          </div>
          <div class="r-dialog__footer">
            <button
              v-if="dialog.state.value.options.showCancel"
              class="r-dialog__btn r-dialog__btn--cancel"
              @click="dialog.close(false)"
            >
              {{ dialog.state.value.options.cancelText || t('common.button.cancel') }}
            </button>
            <button
              class="r-dialog__btn r-dialog__btn--confirm"
              @click="dialog.close(true)"
            >
              {{ dialog.state.value.options.confirmText || t('common.button.confirm') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.r-dialog__mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.r-dialog {
  background: #fff;
  border-radius: 16px;
  width: 400px;
  max-width: 90vw;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
}

.r-dialog__header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 24px 0;
}

.r-dialog__header i {
  font-size: 24px;
}

.r-dialog__title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-main);
}

.r-dialog__body {
  padding: 16px 24px 24px;
  color: var(--color-text-secondary);
  line-height: 1.6;
  white-space: pre-wrap;
}

.r-dialog__footer {
  padding: 0 24px 20px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.r-dialog__btn {
  padding: 10px 20px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.r-dialog__btn--cancel {
  background: rgba(128, 64, 48, 0.08);
  color: var(--color-text-main);
}
.r-dialog__btn--cancel:hover {
  background: rgba(128, 64, 48, 0.15);
}

.r-dialog__btn--confirm {
  background: var(--color-primary);
  color: #fff;
}
.r-dialog__btn--confirm:hover {
  opacity: 0.9;
}

.r-dialog-enter-active, .r-dialog-leave-active { transition: all 0.2s; }
.r-dialog-enter-from, .r-dialog-leave-to { opacity: 0; }
.r-dialog-enter-from .r-dialog, .r-dialog-leave-to .r-dialog {
  transform: scale(0.9);
}
</style>
