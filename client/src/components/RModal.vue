<script setup lang="ts">
interface Props {
  modelValue: boolean
  title?: string
  width?: string
  closable?: boolean
  maskClosable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  width: '480px',
  closable: true,
  maskClosable: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

function close() {
  emit('update:modelValue', false)
  emit('close')
}

function onMaskClick() {
  if (props.maskClosable) close()
}
</script>

<template>
  <Teleport to="body">
    <Transition name="r-modal">
      <div v-if="modelValue" class="r-modal__mask" @click.self="onMaskClick">
        <div class="r-modal" :style="{ '--r-modal-width': width }">
          <div v-if="title || closable" class="r-modal__header">
            <span class="r-modal__title">{{ title }}</span>
            <button v-if="closable" class="r-modal__close" @click="close">×</button>
          </div>
          <div class="r-modal__body"><slot /></div>
          <div v-if="$slots.footer" class="r-modal__footer"><slot name="footer" /></div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.r-modal__mask {
  position: fixed;
  inset: 0;
  background: rgba(var(--shadow-base), 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.r-modal {
  width: var(--r-modal-width, 480px);
  max-width: calc(100vw - 24px);
  background: var(--color-panel-bg);
  border-radius: var(--radius-lg);
  max-height: calc(100vh - 24px);
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg);
}

.r-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--color-border-light);
}

.r-modal__title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-primary);
}

.r-modal__close {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 24px;
  cursor: pointer;
  color: var(--color-secondary);
  border-radius: 50%;
  transition: all 0.2s;
}
.r-modal__close:hover { background: var(--btn-outline-hover); }

.r-modal__body {
  padding: 24px;
  overflow-y: auto;
}

.r-modal__footer {
  padding: 16px 24px;
  border-top: 1px solid var(--color-border-light);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 动画 */
.r-modal-enter-active, .r-modal-leave-active { transition: all 0.25s; }
.r-modal-enter-from, .r-modal-leave-to { opacity: 0; }
.r-modal-enter-from .r-modal, .r-modal-leave-to .r-modal {
  transform: scale(0.9);
}

@media (max-width: 640px) {
  .r-modal__mask {
    padding: 12px;
    align-items: flex-end;
  }

  .r-modal {
    width: 100%;
    max-width: 100%;
    max-height: calc(100vh - 24px);
    border-radius: 18px;
  }

  .r-modal__header {
    padding: 16px 18px;
  }

  .r-modal__title {
    font-size: 16px;
  }

  .r-modal__body {
    padding: 18px;
  }

  .r-modal__footer {
    padding: 14px 18px;
  }
}
</style>
