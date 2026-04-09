<script setup lang="ts">
import DesktopTiptapEditor from '@client/components/TiptapEditor.vue'

defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <div class="mobile-rich-editor">
    <DesktopTiptapEditor
      :model-value="modelValue"
      :placeholder="placeholder"
      @update:modelValue="emit('update:modelValue', $event)"
    >
      <template v-if="$slots.toolbar" #toolbar>
        <slot name="toolbar" />
      </template>
    </DesktopTiptapEditor>
  </div>
</template>

<style scoped>
.mobile-rich-editor {
  width: 100%;
}

.mobile-rich-editor :deep(.rich-editor) {
  border-width: 1px;
  border-color: var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
}

.mobile-rich-editor :deep(.toolbar) {
  flex-wrap: nowrap;
  overflow-x: auto;
  gap: 6px;
  padding: 8px;
  border-bottom-width: 1px;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}

.mobile-rich-editor :deep(.toolbar::-webkit-scrollbar) {
  display: none;
}

.mobile-rich-editor :deep(.toolbar button),
.mobile-rich-editor :deep(.toolbar .toolbar-slot) {
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
}

.mobile-rich-editor :deep(.toolbar button i),
.mobile-rich-editor :deep(.toolbar .toolbar-slot i) {
  font-size: 17px;
}

.mobile-rich-editor :deep(.divider) {
  margin: 5px 2px;
  flex: 0 0 auto;
}

.mobile-rich-editor :deep(.editor-content) {
  min-height: 220px;
  padding: 14px 12px 16px;
}

.mobile-rich-editor :deep(.editor-content .tiptap) {
  min-height: 190px;
  font-family: inherit;
  font-size: 15px;
  line-height: 1.7;
}

.mobile-rich-editor :deep(.editor-content h1) {
  font-size: 22px;
}

.mobile-rich-editor :deep(.editor-content h2) {
  font-size: 19px;
}

.mobile-rich-editor :deep(.editor-content h3) {
  font-size: 17px;
}

.mobile-rich-editor :deep(.editor-content blockquote) {
  padding-left: 14px;
  margin: 1em 0;
}

.mobile-rich-editor :deep(.editor-content pre) {
  padding: 12px;
  border-radius: 8px;
}

.mobile-rich-editor :deep(.editor-content img) {
  margin: 0.4em 0.25em;
  border-radius: 8px;
}
</style>
