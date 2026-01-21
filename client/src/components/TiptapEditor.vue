<script setup lang="ts">
import { ref } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import { uploadImage } from '@/api/item'
import { useToast } from '@/composables/useToast'

const props = defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const imageInputRef = ref<HTMLInputElement | null>(null)
const toast = useToast()
const uploadCacheKey = 'tiptap_image_upload_cache'
const uploadCache = new Map<string, string>()

function loadUploadCache() {
  try {
    const cached = sessionStorage.getItem(uploadCacheKey)
    if (!cached) return
    const parsed = JSON.parse(cached) as Record<string, string>
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === 'string') {
        uploadCache.set(key, value)
      }
    }
  } catch (error) {
    console.warn('Failed to load upload cache:', error)
  }
}

function persistUploadCache() {
  try {
    const data: Record<string, string> = {}
    uploadCache.forEach((value, key) => {
      data[key] = value
    })
    sessionStorage.setItem(uploadCacheKey, JSON.stringify(data))
  } catch (error) {
    console.warn('Failed to persist upload cache:', error)
  }
}

function getFileCacheKey(file: File) {
  return `${file.name}-${file.size}-${file.lastModified}`
}

loadUploadCache()

const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit,
    Placeholder.configure({
      placeholder: props.placeholder || '开始写作...',
    }),
    Image.configure({
      inline: true,
      allowBase64: true,
    }),
  ],
  onUpdate: ({ editor }) => {
    emit('update:modelValue', editor.getHTML())
  },
})

// 监听外部值变化
import { watch } from 'vue'
watch(() => props.modelValue, (value) => {
  if (editor.value && editor.value.getHTML() !== value) {
    editor.value.commands.setContent(value, false)
  }
})

// 图片上传
function triggerImageUpload() {
  imageInputRef.value?.click()
}

async function handleImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files ? Array.from(input.files) : []
  if (files.length === 0) return

  for (const file of files) {
    if (file.size > 20 * 1024 * 1024) {
      toast.info(`图片 ${file.name} 不能超过20MB`)
      continue
    }

    const cacheKey = getFileCacheKey(file)
    const cachedUrl = uploadCache.get(cacheKey)
    if (cachedUrl) {
      editor.value?.chain().focus().setImage({ src: cachedUrl }).run()
      continue
    }

    try {
      const res: any = await uploadImage(file)
      const url = res?.data?.url || res?.url
      if (!url) {
        throw new Error('未获取到图片地址')
      }
      editor.value?.chain().focus().setImage({ src: url }).run()
      uploadCache.set(cacheKey, url)
      persistUploadCache()
    } catch (error: any) {
      console.error('图片上传失败:', error)
      toast.error(error.message || '图片上传失败')
    }
  }

  input.value = ''
}

// 通过URL插入图片
function insertImageByUrl() {
  const url = prompt('请输入图片URL:')
  if (url) {
    editor.value?.chain().focus().setImage({ src: url }).run()
  }
}
</script>

<template>
  <div class="rich-editor">
    <div class="toolbar">
      <button
        type="button"
        :class="{ active: editor?.isActive('bold') }"
        @click="editor?.chain().focus().toggleBold().run()"
        title="粗体"
      >
        <i class="ri-bold"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('italic') }"
        @click="editor?.chain().focus().toggleItalic().run()"
        title="斜体"
      >
        <i class="ri-italic"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('strike') }"
        @click="editor?.chain().focus().toggleStrike().run()"
        title="删除线"
      >
        <i class="ri-strikethrough"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        :class="{ active: editor?.isActive('heading', { level: 1 }) }"
        @click="editor?.chain().focus().toggleHeading({ level: 1 }).run()"
        title="标题1"
      >
        <i class="ri-h-1"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('heading', { level: 2 }) }"
        @click="editor?.chain().focus().toggleHeading({ level: 2 }).run()"
        title="标题2"
      >
        <i class="ri-h-2"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('heading', { level: 3 }) }"
        @click="editor?.chain().focus().toggleHeading({ level: 3 }).run()"
        title="标题3"
      >
        <i class="ri-h-3"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        :class="{ active: editor?.isActive('bulletList') }"
        @click="editor?.chain().focus().toggleBulletList().run()"
        title="无序列表"
      >
        <i class="ri-list-unordered"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('orderedList') }"
        @click="editor?.chain().focus().toggleOrderedList().run()"
        title="有序列表"
      >
        <i class="ri-list-ordered"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        :class="{ active: editor?.isActive('blockquote') }"
        @click="editor?.chain().focus().toggleBlockquote().run()"
        title="引用"
      >
        <i class="ri-double-quotes-l"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('codeBlock') }"
        @click="editor?.chain().focus().toggleCodeBlock().run()"
        title="代码块"
      >
        <i class="ri-code-box-line"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        @click="editor?.chain().focus().undo().run()"
        :disabled="!editor?.can().undo()"
        title="撤销"
      >
        <i class="ri-arrow-go-back-line"></i>
      </button>
      <button
        type="button"
        @click="editor?.chain().focus().redo().run()"
        :disabled="!editor?.can().redo()"
        title="重做"
      >
        <i class="ri-arrow-go-forward-line"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        @click="triggerImageUpload"
        title="上传图片"
      >
        <i class="ri-image-add-line"></i>
      </button>
      <button
        type="button"
        @click="insertImageByUrl"
        title="图片链接"
      >
        <i class="ri-link"></i>
      </button>
    </div>
    <input
      ref="imageInputRef"
      type="file"
      accept="image/*"
      multiple
      style="display: none"
      @change="handleImageUpload"
    />
    <EditorContent :editor="editor" class="editor-content" />
  </div>
</template>

<style scoped>
.rich-editor {
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  overflow: hidden;
  background: #fff;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 12px;
  background: #F5EFE7;
  border-bottom: 2px solid #E5D4C1;
}

.toolbar button {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: #4B3621;
  cursor: pointer;
  transition: all 0.2s;
}

.toolbar button:hover {
  background: #E5D4C1;
}

.toolbar button.active {
  background: #804030;
  color: #fff;
}

.toolbar button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.toolbar button i {
  font-size: 18px;
}

.divider {
  width: 1px;
  height: 24px;
  background: #E5D4C1;
  margin: 6px 8px;
}

.editor-content {
  min-height: 300px;
  padding: 24px 32px;
}

.editor-content :deep(.tiptap) {
  outline: none;
  min-height: 280px;
  font-family: 'Merriweather', serif;
  font-size: 16px;
  line-height: 1.9;
  color: #4B3621;
}

.editor-content :deep(.tiptap p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  color: #8D7B68;
  pointer-events: none;
  height: 0;
}

.editor-content :deep(h1) {
  font-size: 28px;
  font-weight: 700;
  margin: 16px 0 8px;
  color: #2C1810;
}

.editor-content :deep(h2) {
  font-size: 24px;
  font-weight: 700;
  color: #2C1810;
  margin: 1.5em 0 0.8em;
}

.editor-content :deep(h3) {
  font-size: 20px;
  font-weight: 600;
  color: #2C1810;
  margin: 1.2em 0 0.6em;
}

.editor-content :deep(p) {
  margin-bottom: 1.5em;
}

.editor-content :deep(ul),
.editor-content :deep(ol) {
  padding-left: 24px;
  margin: 8px 0;
}

.editor-content :deep(li) {
  margin: 4px 0;
}

.editor-content :deep(blockquote) {
  border-left: 4px solid #B87333;
  padding-left: 20px;
  margin: 1.5em 0;
  color: #6B5344;
  font-style: italic;
}

.editor-content :deep(pre) {
  background: #2C1810;
  color: #EED9C4;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 12px 0;
}

.editor-content :deep(code) {
  font-family: 'Fira Code', monospace;
  font-size: 14px;
}

.editor-content :deep(img) {
  max-width: 100%;
  height: auto;
  display: inline-block;
  border-radius: 4px;
  margin: 0.6em 0.6em;
  vertical-align: middle;
  cursor: pointer;
  transition: transform 0.2s;
}

.editor-content :deep(img:hover) {
  transform: scale(1.02);
}

.editor-content :deep(img.ProseMirror-selectednode) {
  outline: 3px solid #804030;
  outline-offset: 2px;
}

.editor-content :deep(strong) {
  color: #804030;
}
</style>
