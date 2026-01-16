<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'

interface Props {
  modelValue: string
  placeholder?: string
  minHeight?: string
  simple?: boolean  // 简单模式，只有内省按钮
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  placeholder: '输入内容...',
  minHeight: '150px',
  simple: false
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

// 插入内省标记（用斜体表示内心独白）
function insertIntrospection() {
  const selection = window.getSelection()
  if (selection && selection.rangeCount > 0) {
    const range = selection.getRangeAt(0)
    const selectedText = range.toString()
    if (selectedText) {
      // 包裹选中文本
      const em = document.createElement('em')
      em.className = 'introspection'
      em.textContent = selectedText
      range.deleteContents()
      range.insertNode(em)
    } else {
      // 插入占位符
      const em = document.createElement('em')
      em.className = 'introspection'
      em.textContent = '（内心独白）'
      range.insertNode(em)
    }
    updateContent()
  }
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
      <!-- 简单模式：只有内省按钮 -->
      <template v-if="simple">
        <button type="button" @click="insertIntrospection" title="内省/内心独白">
          <i class="ri-mind-map"></i>
          <span class="btn-text">内省</span>
        </button>
      </template>
      <!-- 完整模式 -->
      <template v-else>
        <button type="button" @click="insertIntrospection" title="内省/内心独白">
          <i class="ri-mind-map"></i>
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
      </template>
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
  min-width: 32px;
  height: 32px;
  padding: 0 8px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-primary, #4B3621);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.toolbar button .btn-text {
  font-size: 12px;
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

/* 内省/内心独白样式 */
.editor-content :deep(.introspection),
.editor-content :deep(em.introspection) {
  font-style: italic;
  color: #9b59b6;
  background: rgba(155, 89, 182, 0.1);
  padding: 0 4px;
  border-radius: 2px;
}
</style>
