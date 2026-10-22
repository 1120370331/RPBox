<script setup lang="ts">
import { nextTick, ref } from 'vue'
import RModal from '@/components/RModal.vue'
import TRP3IconPicker from './TRP3IconPicker.vue'
const props = withDefaults(defineProps<{ modelValue: string; label?: string; rows?: number }>(), { label: '关于内容', rows: 16 })
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const area = ref<HTMLTextAreaElement | null>(null)
const align = ref(''), color = ref('#ffffff'), iconOpen = ref(false)
const mediaOpen = ref(false), mediaMode = ref<'image' | 'link'>('image')
const url = ref(''), linkText = ref(''), width = ref(256), height = ref(256), mediaError = ref('')
function insert(open: string, close = '', fallback = '') {
  const start = area.value?.selectionStart ?? props.modelValue.length
  const end = area.value?.selectionEnd ?? start
  const selected = props.modelValue.slice(start, end) || fallback
  emit('update:modelValue', props.modelValue.slice(0, start) + open + selected + close + props.modelValue.slice(end))
  void nextTick(() => { area.value?.focus(); area.value?.setSelectionRange(start + open.length, start + open.length + selected.length) })
}
function container(tag: string) { insert(`{${tag}${align.value ? ':' + align.value : ''}}`, `{/${tag}}`, '你的文字') }
function openMedia(mode: 'image' | 'link') { mediaMode.value = mode; url.value = ''; linkText.value = ''; mediaError.value = ''; mediaOpen.value = true }
function insertMedia() {
  const value = url.value.trim()
  const safe = mediaMode.value === 'image' ? /^Interface[\\/][^:]+$/i.test(value) : /^https?:\/\//i.test(value)
  if (!safe || /[{}*\n\r]/.test(value)) { mediaError.value = mediaMode.value === 'image' ? 'TRP3 图片需要 Interface\\ 开头的游戏材质路径；网络图片请使用链接。' : '请输入 HTTP(S) 链接地址。'; return }
  if (mediaMode.value === 'link' && /[{}*]/.test(linkText.value)) { mediaError.value = '链接文字不能含有格式分隔符。'; return }
  insert(mediaMode.value === 'link' ? `{link*${value}*${linkText.value || value}}` : `{img:${value}:${Math.max(1, Math.min(512, width.value))}:${Math.max(1, Math.min(512, height.value))}${align.value ? ':' + align.value : ''}}`)
  mediaOpen.value = false
}
</script>
<template>
  <div class="trp3-text-editor">
    <fieldset class="trp3-toolbar"><legend>格式工具</legend><div>
      <button v-for="tag in ['h1','h2','h3','p']" :key="tag" type="button" :aria-label="`插入 ${tag.toUpperCase()} 标签`" @mousedown.prevent @click="container(tag)">{{ tag.toUpperCase() }}</button>
      <select v-model="align" aria-label="文字对齐"><option value="">左对齐</option><option value="c">居中</option><option value="r">右对齐</option></select>
      <button type="button" @click="openMedia('image')">图片</button><button type="button" @click="iconOpen = true">图标</button>
      <label class="toolbar-color" title="文字颜色"><input v-model="color" type="color" aria-label="格式文字颜色" /></label><button type="button" @mousedown.prevent @click="insert(`{col:${color.slice(1)}}`, '{/col}', '你的文字')">颜色</button><button type="button" @click="openMedia('link')">链接</button>
    </div></fieldset>
    <textarea ref="area" :value="modelValue" :rows="rows" :aria-label="label" spellcheck="false" placeholder="在这里书写。使用上方工具插入 TRP3 格式标签。" @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"></textarea>
    <TRP3IconPicker v-model="iconOpen" @select="insert(`{icon:${$event}:25}`)" />
    <RModal v-model="mediaOpen" :title="mediaMode === 'image' ? '插入图片' : '插入链接'" width="520px">
      <form class="media-form" @submit.prevent="insertMedia"><label>{{ mediaMode === 'image' ? '游戏材质路径（Interface\\…）' : '链接地址' }}<input v-model="url" required /></label><label v-if="mediaMode === 'link'">链接文字<input v-model="linkText" /></label><div v-else class="media-dimensions"><label>宽度<input v-model.number="width" type="number" min="1" max="512" /></label><label>高度<input v-model.number="height" type="number" min="1" max="512" /></label></div><small v-if="mediaMode === 'image'">游戏专有材质会保留原始路径与尺寸，网页未收录的图片需在游戏中预览。</small><p v-if="mediaError" role="alert">{{ mediaError }}</p><button type="submit">插入</button></form>
    </RModal>
  </div>
</template>
<style scoped>
.trp3-toolbar{border:1px solid #75674b;padding:8px 10px 10px;margin:0 0 10px;border-radius:3px}.trp3-toolbar legend{margin:auto;padding:0 8px;color:#ffd46b;font-size:12px}.trp3-toolbar>div{display:flex;align-items:center;justify-content:center;gap:5px;flex-wrap:wrap}.trp3-toolbar button,.trp3-toolbar select{min-height:26px;padding:3px 11px;border:1px solid #8c7860;border-radius:3px;background:linear-gradient(#84362d,#4b1714);box-shadow:inset 0 0 0 1px #2a1710;color:#ffd470;font-size:12px;cursor:pointer}.trp3-toolbar select{background:#26221d}.trp3-toolbar button:hover{filter:brightness(1.25)}textarea{box-sizing:border-box;width:100%;resize:vertical;padding:14px;border:1px solid #75674b;border-radius:3px;background:#121211;color:#eee6d1;font-family:Consolas,'Noto Sans SC',monospace;font-size:13px;line-height:1.8;tab-size:2;min-height:120px}textarea:focus{outline:1px solid #d3b269}.toolbar-color input{width:26px;height:26px;padding:1px;border:1px solid #8c7860;background:#222;cursor:pointer}.media-form{display:grid;gap:14px;color:var(--color-text-main)}.media-form label{display:grid;gap:7px;font-size:13px}.media-form input{height:36px;padding:0 10px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);color:var(--input-text)}.media-dimensions{display:grid;grid-template-columns:1fr 1fr;gap:12px}.media-form button{padding:10px;border:0;border-radius:6px;background:var(--btn-primary-bg);color:var(--btn-primary-text);cursor:pointer}.media-form p{color:var(--color-danger);font-size:12px}.trp3-toolbar :is(button,select):focus-visible{outline:2px solid #ffd470;outline-offset:2px}
</style>
