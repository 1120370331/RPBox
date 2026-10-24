<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { dialog } from '../composables/useDialog'

const { t } = useI18n()
const props = withDefaults(defineProps<{
  modelValue?: boolean
  title?: string
  confirmText?: string
  cancelText?: string
  confirmDisabled?: boolean
  width?: string
}>(), { modelValue: undefined, confirmDisabled: false, width: '400px' })
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: []
  cancel: []
}>()
const controlled = computed(() => props.modelValue !== undefined)
const visible = computed(() => controlled.value ? props.modelValue : dialog.state.value.visible)
const panel = ref<HTMLElement | null>(null)
let previousFocus: HTMLElement | null = null
function focusableElements() {
  return [...(panel.value?.querySelectorAll<HTMLElement>('select:not(:disabled), input:not(:disabled):not([type="hidden"]), textarea:not(:disabled), button:not(:disabled), a[href]') || [])]
    .filter(element => element.getClientRects().length > 0)
}
watch(visible, async open => {
  if (!controlled.value) return
  if (open) {
    previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    await nextTick()
    if (visible.value) focusableElements()[0]?.focus()
  } else if (previousFocus?.isConnected) {
    previousFocus.focus()
    previousFocus = null
  }
}, { immediate: true })
function trapTab(event: KeyboardEvent) {
  if (!controlled.value) return
  const elements = focusableElements(), first = elements[0], last = elements.at(-1)
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus() }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() }
}
const options = computed(() => controlled.value ? {
  title: props.title, message: '', type: undefined,
  showCancel: true, confirmText: props.confirmText, cancelText: props.cancelText,
} : dialog.state.value.options)
function close(confirmed: boolean) {
  if (!controlled.value) return dialog.close(confirmed)
  if (confirmed) {
    if (!props.confirmDisabled) emit('confirm')
  } else {
    emit('update:modelValue', false)
    emit('cancel')
  }
}

const typeIcons = {
  info: 'ri-information-line',
  success: 'ri-checkbox-circle-line',
  warning: 'ri-alert-line',
  error: 'ri-close-circle-line',
}

const typeColors = {
  info: 'var(--link-color)',
  success: 'var(--color-success)',
  warning: 'var(--color-warning-dark)',
  error: 'var(--btn-danger-bg)',
}
</script>

<template>
  <Teleport to="body">
    <Transition name="r-dialog">
      <div v-if="visible" class="r-dialog__mask" @click.self="close(false)" @keydown.esc.stop="close(false)" @keydown.tab="trapTab">
        <div ref="panel" class="r-dialog" role="dialog" aria-modal="true" :aria-label="options.title" :style="{ width }">
          <div class="r-dialog__header">
            <i
              v-if="options.type"
              :class="typeIcons[options.type]"
              :style="{ color: typeColors[options.type] }"
            ></i>
            <span class="r-dialog__title">{{ options.title || t('common.status.loading').replace('...', '') }}</span>
          </div>
          <div class="r-dialog__body">
            <slot>{{ options.message }}</slot>
          </div>
          <div class="r-dialog__footer">
            <button
              v-if="options.showCancel"
              class="r-dialog__btn r-dialog__btn--cancel"
              @click="close(false)"
            >
              {{ options.cancelText || t('common.button.cancel') }}
            </button>
            <button
              class="r-dialog__btn r-dialog__btn--confirm"
              :class="{ 'r-dialog__btn--danger': options.type === 'error' }"
              :disabled="controlled && confirmDisabled"
              @click="close(true)"
            >
              {{ options.confirmText || t('common.button.confirm') }}
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
  background: rgba(var(--shadow-base), 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.r-dialog {
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  border-radius: 16px;
  width: 400px;
  max-width: 90vw;
  max-height: calc(100dvh - 32px);
  overflow-y: auto;
  box-shadow: var(--shadow-lg);
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

.r-dialog__btn:disabled { opacity: 0.5; cursor: not-allowed; }

.r-dialog__btn--cancel {
  background: var(--btn-secondary-bg);
  color: var(--color-text-main);
}
.r-dialog__btn--cancel:hover {
  background: var(--btn-secondary-hover);
}

.r-dialog__btn--confirm {
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
}
.r-dialog__btn--confirm:hover {
  background: var(--btn-primary-hover);
}

.r-dialog__btn--danger {
  background: var(--btn-danger-bg);
  color: var(--btn-danger-text);
}

.r-dialog__btn--danger:hover {
  background: var(--btn-danger-hover);
}

.r-dialog-enter-active, .r-dialog-leave-active { transition: all 0.2s; }
.r-dialog-enter-from, .r-dialog-leave-to { opacity: 0; }
.r-dialog-enter-from .r-dialog, .r-dialog-leave-to .r-dialog {
  transform: scale(0.9);
}
</style>
