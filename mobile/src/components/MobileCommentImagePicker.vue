<script setup lang="ts">
import { computed, ref } from 'vue'
import { uploadCommentImage } from '@/api/commentImage'
import { resolveApiUrl } from '@/api/image'
import { useToastStore } from '@shared/stores/toast'

const props = withDefaults(defineProps<{
  modelValue?: string
  disabled?: boolean
}>(), {
  modelValue: '',
  disabled: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'update:uploading', value: boolean): void
  (e: 'preview', src: string): void
}>()

const toast = useToastStore()
const inputRef = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const previewURL = computed(() => resolveApiUrl(props.modelValue || ''))
const isGIF = computed(() => /\.gif(?:$|[?#])/i.test(props.modelValue || ''))
const allowedTypes = new Set(['image/jpeg', 'image/png', 'image/gif', 'image/webp'])

function setUploading(value: boolean) {
  uploading.value = value
  emit('update:uploading', value)
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!allowedTypes.has(file.type)) {
    toast.error('仅支持 JPG、PNG、GIF 和 WebP 图片')
    return
  }
  if (file.size > 20 * 1024 * 1024) {
    toast.error('评论配图不能超过 20MB')
    return
  }

  setUploading(true)
  try {
    const result = await uploadCommentImage(file)
    if (!result.url) throw new Error('图片上传失败')
    emit('update:modelValue', result.url)
  } catch (error) {
    toast.error((error as Error).message || '图片上传失败')
  } finally {
    setUploading(false)
  }
}
</script>

<template>
  <div class="mobile-comment-image-picker">
    <input
      ref="inputRef"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      hidden
      @change="handleFileChange"
    >
    <div v-if="modelValue" class="image-preview">
      <button type="button" class="preview-main" @click="emit('preview', previewURL)">
        <img :src="previewURL" alt="待发布评论配图">
        <span v-if="isGIF">GIF</span>
      </button>
      <button type="button" class="remove" :disabled="disabled || uploading" aria-label="移除配图" @click="emit('update:modelValue', '')">
        <i class="ri-close-line" />
      </button>
    </div>
    <button v-else type="button" class="pick" :disabled="disabled || uploading" @click="inputRef?.click()">
      <i :class="uploading ? 'ri-loader-4-line spin' : 'ri-image-add-line'" />
      {{ uploading ? '上传中' : '图片 / GIF' }}
    </button>
    <small>配图评论审核通过后展示</small>
  </div>
</template>

<style scoped>
.mobile-comment-image-picker{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-top:9px}.pick{display:inline-flex;min-height:34px;align-items:center;gap:5px;padding:0 10px;border:1px solid var(--color-border);border-radius:7px;background:var(--color-card-bg);color:var(--color-text-secondary);font-size:11px}.pick:disabled,.remove:disabled{opacity:.55}.mobile-comment-image-picker>small{color:var(--color-text-secondary);font-size:9px}.image-preview{position:relative;width:92px;height:70px}.preview-main{position:relative;width:100%;height:100%;overflow:hidden;padding:0;border:1px solid var(--color-border);border-radius:7px;background:var(--color-card-bg)}.preview-main img{width:100%;height:100%;object-fit:cover}.preview-main span{position:absolute;right:4px;bottom:4px;padding:2px 4px;border-radius:3px;background:rgba(20,16,12,.74);color:#fff;font-size:8px;font-weight:800}.remove{position:absolute;top:-6px;right:-6px;display:grid;width:21px;height:21px;padding:0;place-items:center;border:1px solid var(--color-border);border-radius:50%;background:var(--color-panel-bg);color:var(--color-text-main)}.spin{animation:mobile-comment-image-spin .8s linear infinite}@keyframes mobile-comment-image-spin{to{transform:rotate(360deg)}}
</style>
