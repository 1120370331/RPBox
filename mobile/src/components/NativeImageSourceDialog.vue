<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { NativeImageSource } from '@/utils/nativeImagePicker'

defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  select: [source: NativeImageSource]
}>()

const { t } = useI18n()

function closeDialog() {
  emit('update:modelValue', false)
}

function chooseSource(source: NativeImageSource) {
  emit('select', source)
  closeDialog()
}
</script>

<template>
  <div v-if="modelValue" class="dialog-mask" @click.self="closeDialog">
    <div class="dialog-sheet" role="dialog" aria-modal="true">
      <h3>{{ t('common.imagePicker.title') }}</h3>
      <p>{{ t('common.imagePicker.message') }}</p>
      <div class="dialog-actions">
        <button type="button" class="action-btn primary" @click="chooseSource('camera')">
          {{ t('common.imagePicker.camera') }}
        </button>
        <button type="button" class="action-btn" @click="chooseSource('photos')">
          {{ t('common.imagePicker.photos') }}
        </button>
        <button type="button" class="action-btn ghost" @click="closeDialog">
          {{ t('common.imagePicker.cancel') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.42);
}

.dialog-sheet {
  width: min(100%, 420px);
  border-radius: 18px;
  background: var(--color-panel-bg);
  box-shadow: 0 18px 36px rgba(44, 24, 16, 0.18);
  padding: 18px 16px 14px;
}

.dialog-sheet h3 {
  font-size: 17px;
  color: var(--text-dark);
}

.dialog-sheet p {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-text-secondary);
}

.dialog-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
}

.action-btn {
  width: 100%;
  min-height: 44px;
  border: 1px solid var(--input-border);
  border-radius: 12px;
  background: var(--input-bg);
  color: var(--text-dark);
  font-size: 14px;
  font-weight: 600;
}

.action-btn.primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}

.action-btn.ghost {
  background: transparent;
}
</style>
