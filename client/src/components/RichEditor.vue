<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'

interface Props {
  modelValue: string
  placeholder?: string
  minHeight?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  placeholder: '输入内容...',
  minHeight: '150px'
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorRef = ref<HTMLDivElement | null>(null)

function updateContent() {
  if (editorRef.value) {
    emit('update:modelValue', editorRef.value.innerHTML)
  }
}

function execCommand(cmd: string, value?: string) {
  document.execCommand(cmd, false, value)
  editorRef.value?.focus()
  updateContent()
}

function insertImage() {
  const url = prompt('请输入图片URL:')
  if (url) {
    execCommand('insertImage', url)
  }
}

watch(() => props.modelValue, (newVal) => {
  if (editorRef.value && editorRef.value.innerHTML !== newVal) {
    editorRef.value.innerHTML = newVal
  }
})

onMounted(() => {
  if (editorRef.value) {
    editorRef.value.innerHTML = props.modelValue
  }
})
</script>

<template>
  <div class="rich-editor">
    <div class="toolbar">
      <button type="button" @click="execCommand('bold')" title="粗体">
        <i class="ri-bold"></i>
      </button>
      <button type="button" @click="execCommand('italic')" title="斜体">
        <i class="ri-italic"></i>
      </button>
      <button type="button" @click="execCommand('underline')" title="下划线">
        <i class="ri-underline"></i>
      </button>
      <span class="divider"></span>
      <button type="button" @click="execCommand('insertUnorderedList')" title="无序列表">
        <i class="ri-list-unordered"></i>
      </button>
      <button type="button" @click="execCommand('insertOrderedList')" title="有序列表">
        <i class="ri-list-ordered"></i>
      </button>
      <span class="divider"></span>
      <button type="button" @click="insertImage" title="插入图片">
        <i class="ri-image-line"></i>
      </button>
    </div>
    <div
      ref="editorRef"
      class="editor-content"
      contenteditable="true"
      :style="{ minHeight }"
      :data-placeholder="placeholder"
      @input="updateContent"
      @blur="updateContent"
    ></div>
  </div>
</template>

<style scoped>
.rich-editor {
  border: 1px solid var(--color-border, #d1bfa8);
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px;
  background: var(--color-bg-secondary, #f5f0e8);
  border-bottom: 1px solid var(--color-border, #d1bfa8);
}

.toolbar button {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-primary, #4B3621);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.toolbar button:hover {
  background: rgba(0, 0, 0, 0.05);
}

.divider {
  width: 1px;
  height: 20px;
  background: var(--color-border, #d1bfa8);
  margin: 0 4px;
}

.editor-content {
  padding: 12px;
  outline: none;
  line-height: 1.6;
  color: var(--color-text, #333);
}

.editor-content:empty::before {
  content: attr(data-placeholder);
  color: var(--color-secondary, #856a52);
  pointer-events: none;
}

.editor-content img {
  max-width: 100%;
  border-radius: 4px;
}
</style>
