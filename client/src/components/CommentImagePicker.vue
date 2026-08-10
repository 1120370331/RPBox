<script setup lang="ts">
import { computed, ref } from 'vue'
import { uploadCommentImage } from '@/api/commentImage'
import { resolveApiUrl } from '@/api/item'
import { useToast } from '@/composables/useToast'

const props = withDefaults(defineProps<{
  modelValue?: string
  disabled?: boolean
  compact?: boolean
}>(), {
  modelValue: '',
  disabled: false,
  compact: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'update:uploading', value: boolean): void
  (e: 'preview', src: string): void
}>()

const toast = useToast()
const inputRef = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const previewURL = computed(() => resolveApiUrl(props.modelValue || ''))
const isGIF = computed(() => /\.gif(?:$|[?#])/i.test(props.modelValue || ''))
const allowedTypes = new Set(['image/jpeg', 'image/png', 'image/gif', 'image/webp'])
const maxBytes = 20 * 1024 * 1024

function chooseFile() {
  if (!props.disabled && !uploading.value) inputRef.value?.click()
}

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
  if (file.size > maxBytes) {
    toast.error('评论配图不能超过 20MB')
    return
  }

  setUploading(true)
  try {
    const result = await uploadCommentImage(file)
    const url = result.url
    if (!url) throw new Error('图片上传失败')
    emit('update:modelValue', url)
  } catch (error) {
    toast.error((error as Error).message || '图片上传失败')
  } finally {
    setUploading(false)
  }
}

function removeImage() {
  if (props.disabled || uploading.value) return
  emit('update:modelValue', '')
}
</script>

<template>
  <div class="comment-image-picker" :class="{ compact }">
    <input
      ref="inputRef"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      hidden
      @change="handleFileChange"
    >
    <div v-if="modelValue" class="comment-image-preview">
      <button type="button" class="preview-button" :disabled="disabled" @click="emit('preview', previewURL)">
        <img :src="previewURL" alt="待发布评论配图">
        <span v-if="isGIF" class="gif-badge">GIF</span>
      </button>
      <button type="button" class="remove-button" :disabled="disabled || uploading" aria-label="移除配图" @click="removeImage">
        <i class="ri-close-line" />
      </button>
    </div>
    <button v-else type="button" class="pick-button" :disabled="disabled || uploading" @click="chooseFile">
      <i :class="uploading ? 'ri-loader-4-line spin' : 'ri-image-add-line'" />
      <span>{{ uploading ? '上传中' : '添加图片 / GIF' }}</span>
    </button>
    <p v-if="!compact" class="review-hint">带配图的评论将在版主审核通过后展示</p>
  </div>
</template>

<style scoped>
.comment-image-picker{display:flex;align-items:center;gap:10px;flex-wrap:wrap}.pick-button{display:inline-flex;min-height:34px;align-items:center;gap:6px;padding:0 10px;border:1px solid var(--color-border);border-radius:6px;background:transparent;color:var(--color-text-muted);font-size:12px;cursor:pointer}.pick-button:hover:not(:disabled){border-color:var(--color-accent);color:var(--color-accent)}.pick-button:disabled,.preview-button:disabled,.remove-button:disabled{opacity:.55;cursor:not-allowed}.review-hint{margin:0;color:var(--color-text-muted);font-size:11px}.comment-image-preview{position:relative;width:112px;height:82px}.preview-button{position:relative;width:100%;height:100%;overflow:hidden;padding:0;border:1px solid var(--color-border);border-radius:7px;background:var(--color-card-bg);cursor:zoom-in}.preview-button img{width:100%;height:100%;object-fit:cover}.gif-badge{position:absolute;right:5px;bottom:5px;padding:2px 5px;border-radius:4px;background:rgba(20,16,12,.75);color:#fff;font-size:9px;font-weight:800;letter-spacing:.06em}.remove-button{position:absolute;top:-7px;right:-7px;display:grid;width:22px;height:22px;padding:0;place-items:center;border:1px solid var(--color-border);border-radius:50%;background:var(--color-panel-bg);color:var(--color-text-main);cursor:pointer}.compact .comment-image-preview{width:88px;height:64px}.compact .pick-button{min-height:28px;padding:0 8px;font-size:11px}.spin{animation:comment-image-spin .8s linear infinite}@keyframes comment-image-spin{to{transform:rotate(360deg)}}
</style>
