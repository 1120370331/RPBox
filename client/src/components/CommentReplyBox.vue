<script setup lang="ts">
import { nextTick, ref } from 'vue'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  disabled?: boolean
  cancelLabel: string
  submitLabel: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'openEmoji', event: MouseEvent): void
  (e: 'cancel'): void
  (e: 'submit'): void
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)

function updateValue(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}

function focus() {
  void nextTick(() => textareaRef.value?.focus())
}

function insertToken(token: string) {
  if (props.disabled) return
  const textarea = textareaRef.value
  const value = props.modelValue || ''
  const start = textarea?.selectionStart ?? value.length
  const end = textarea?.selectionEnd ?? start
  const before = value.slice(0, start)
  const after = value.slice(end)
  const prefix = before && !/\s$/.test(before) ? ' ' : ''
  const suffix = after && !/^\s/.test(after) ? ' ' : ''
  const spacer = !after ? ' ' : suffix
  const nextValue = `${before}${prefix}${token}${spacer}${after}`
  const cursor = before.length + prefix.length + token.length + spacer.length
  emit('update:modelValue', nextValue)
  void nextTick(() => {
    textareaRef.value?.focus()
    textareaRef.value?.setSelectionRange(cursor, cursor)
  })
}

defineExpose({
  focus,
  insertToken,
})
</script>

<template>
  <div class="reply-input-box">
    <textarea
      ref="textareaRef"
      class="reply-textarea"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      rows="3"
      @input="updateValue"
    ></textarea>
    <div class="reply-actions">
      <button class="emoji-btn-small" type="button" :disabled="disabled" @click="emit('openEmoji', $event)">
        <i class="ri-emotion-line"></i>
      </button>
      <div class="reply-actions-right">
        <button class="cancel-btn" type="button" :disabled="disabled" @click="emit('cancel')">{{ cancelLabel }}</button>
        <button class="submit-btn" type="button" :disabled="disabled || !modelValue.trim()" @click="emit('submit')">{{ submitLabel }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.reply-input-box {
  margin-top: 12px;
  padding: 12px;
  background: var(--color-card-bg);
  border-radius: 6px;
}

.reply-textarea {
  width: 100%;
  min-height: 72px;
  padding: 8px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-panel-bg, #fff);
  color: var(--color-primary);
  font: inherit;
  font-size: 13px;
  line-height: 1.5;
  resize: vertical;
  outline: none;
}

.reply-textarea:focus {
  border-color: var(--color-accent);
}

.reply-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}

.reply-actions-right {
  display: flex;
  gap: 8px;
}

.emoji-btn-small {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s;
}

.emoji-btn-small:hover:not(:disabled) {
  background: var(--color-card-bg);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.emoji-btn-small:disabled,
.cancel-btn:disabled,
.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.emoji-btn-small i {
  font-size: 14px;
}

.cancel-btn {
  padding: 6px 12px;
  background: none;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
}

.submit-btn {
  padding: 6px 12px;
  background: var(--color-secondary);
  border: none;
  border-radius: 4px;
  color: var(--btn-primary-text, var(--color-text-light));
  font-size: 12px;
  cursor: pointer;
}
</style>
