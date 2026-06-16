<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import DesktopTiptapEditor from '@client/components/TiptapEditor.vue'
import NativeImageSourceDialog from '@/components/NativeImageSourceDialog.vue'
import {
  canUseNativeImagePicker,
  pickSingleNativeImageFile,
  type NativeImageSource,
} from '@/utils/nativeImagePicker'

defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const useNativeImagePicker = canUseNativeImagePicker()
const showImageSourceDialog = ref(false)
const editorRef = ref<InstanceType<typeof DesktopTiptapEditor> | null>(null)
let pendingSourceResolver: ((source: NativeImageSource | null) => void) | null = null

async function handlePickImages() {
  const source = await requestNativeImageSource()
  if (!source) return []

  const file = await pickSingleNativeImageFile(source)
  return file ? [file] : []
}

function requestNativeImageSource() {
  showImageSourceDialog.value = true
  return new Promise<NativeImageSource | null>((resolve) => {
    pendingSourceResolver = resolve
  })
}

function handleSourceSelect(source: NativeImageSource) {
  resolvePendingSource(source)
}

function handleDialogToggle(next: boolean) {
  showImageSourceDialog.value = next
  if (!next) {
    resolvePendingSource(null)
  }
}

function resolvePendingSource(source: NativeImageSource | null) {
  const resolve = pendingSourceResolver
  pendingSourceResolver = null
  showImageSourceDialog.value = false
  resolve?.(source)
}

function insertContent(html: string) {
  editorRef.value?.insertContent(html)
}

defineExpose({
  insertContent,
})

onBeforeUnmount(() => {
  resolvePendingSource(null)
})
</script>

<template>
  <div class="mobile-rich-editor">
    <DesktopTiptapEditor
      ref="editorRef"
      :model-value="modelValue"
      :placeholder="placeholder"
      :pick-images="useNativeImagePicker ? handlePickImages : undefined"
      @update:modelValue="emit('update:modelValue', $event)"
    >
      <template v-if="$slots.toolbar" #toolbar>
        <slot name="toolbar" />
      </template>
    </DesktopTiptapEditor>
    <NativeImageSourceDialog
      :model-value="showImageSourceDialog"
      @update:modelValue="handleDialogToggle"
      @select="handleSourceSelect"
    />
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
  flex-wrap: wrap;
  align-items: center;
  overflow: visible;
  gap: 6px;
  padding: 10px;
  border-bottom-width: 1px;
}

.mobile-rich-editor :deep(.toolbar button),
.mobile-rich-editor :deep(.toolbar .toolbar-slot:not(.toolbar-slot--featured)) {
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
}

.mobile-rich-editor :deep(.toolbar .toolbar-slot--featured) {
  width: auto;
  min-width: 108px;
  height: 34px;
  flex: 0 0 auto;
  padding: 0 12px;
  gap: 6px;
  white-space: nowrap;
}

.mobile-rich-editor :deep(.toolbar button i),
.mobile-rich-editor :deep(.toolbar .toolbar-slot i) {
  font-size: 17px;
}

.mobile-rich-editor :deep(.divider) {
  height: 20px;
  margin: 7px 2px;
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
