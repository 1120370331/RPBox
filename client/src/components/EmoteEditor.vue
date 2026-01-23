<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { renderEmoteContent } from '@/utils/emote'
import { resolveEmoteUrl } from '@/api/emote'
import { useEmoteStore } from '@/stores/emote'
import { searchUsers, type UserMentionItem } from '@/api/user'
import { buildNameStyle } from '@/utils/userNameStyle'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const editorRef = ref<HTMLDivElement | null>(null)
const internalValue = ref(props.modelValue || '')
const isComposing = ref(false)
const lastSelection = ref<Range | null>(null)

const emoteStore = useEmoteStore()

const EMOTE_TOKEN_RE = /\[\[emote:([a-z0-9_-]+):([a-z0-9_-]+)\]\]/i
const MENTION_QUERY_RE = /(^|\s)@([\p{L}\p{N}_-]{0,30})$/u

const mentionItems = ref<UserMentionItem[]>([])
const mentionOpen = ref(false)
const mentionIndex = ref(0)
const mentionRange = ref<Range | null>(null)
let mentionTimer: ReturnType<typeof setTimeout> | null = null

function buildMentionToken(id: number | string, name: string) {
  const encoded = encodeURIComponent(name || '')
  return `[[mention:${id}:${encoded}]]`
}

function setContent(value: string) {
  internalValue.value = value || ''
  const editor = editorRef.value
  if (!editor) return
  if (!value) {
    editor.innerHTML = ''
    return
  }
  editor.innerHTML = renderEmoteContent(value, emoteStore.emoteMap)
}

function normalizeText(text: string) {
  return text.replace(/\u00a0/g, ' ')
}

function serializeNode(node: Node): string {
  let output = ''
  node.childNodes.forEach((child) => {
    if (child.nodeType === Node.TEXT_NODE) {
      output += normalizeText(child.textContent || '')
      return
    }
    if (child.nodeType !== Node.ELEMENT_NODE) {
      return
    }
    const el = child as HTMLElement
    const tag = el.tagName
    if (tag === 'BR') {
      output += '\n'
      return
    }
    if (tag === 'IMG') {
      const token = el.getAttribute('data-emote-token')
      if (token) {
        output += token
        return
      }
      const emote = el.getAttribute('data-emote')
      if (emote) {
        output += `[[emote:${emote}]]`
      }
      return
    }
    if (tag === 'SPAN') {
      const mentionId = el.getAttribute('data-mention-id')
      if (mentionId) {
        const mentionName = el.getAttribute('data-mention-name') || (el.textContent || '').replace(/^@/, '')
        output += buildMentionToken(mentionId, mentionName)
        return
      }
    }
    const isBlock = tag === 'DIV' || tag === 'P' || tag === 'LI'
    output += serializeNode(el)
    if (isBlock) {
      output += '\n'
    }
  })
  return output
}

function emitValue() {
  const editor = editorRef.value
  if (!editor) return
  const value = serializeNode(editor).replace(/\n+$/, '')
  internalValue.value = value
  emit('update:modelValue', value)
}

function closeMention() {
  mentionOpen.value = false
  mentionItems.value = []
  mentionIndex.value = 0
  mentionRange.value = null
  if (mentionTimer) {
    clearTimeout(mentionTimer)
    mentionTimer = null
  }
}

async function loadMentionUsers(query: string) {
  try {
    const res = await searchUsers(query, 8)
    mentionItems.value = res.users || []
    mentionIndex.value = 0
    mentionOpen.value = true
  } catch (error) {
    console.error('加载提及用户失败:', error)
    mentionItems.value = []
    mentionOpen.value = true
  }
}

function scheduleMentionSearch(query: string) {
  if (mentionTimer) clearTimeout(mentionTimer)
  mentionTimer = setTimeout(() => {
    void loadMentionUsers(query)
  }, 200)
}

function getMentionContext() {
  const range = getSelectionRange()
  if (!range) return null
  const container = range.startContainer
  if (container.nodeType !== Node.TEXT_NODE) return null
  const text = container.textContent || ''
  const upto = text.slice(0, range.startOffset)
  const match = upto.match(MENTION_QUERY_RE)
  if (!match) return null
  const atIndex = upto.lastIndexOf('@')
  if (atIndex < 0) return null
  const mentionTextRange = document.createRange()
  mentionTextRange.setStart(container, atIndex)
  mentionTextRange.setEnd(container, range.startOffset)
  return {
    query: match[2] || '',
    range: mentionTextRange,
  }
}

function updateMentionState() {
  if (props.disabled) {
    closeMention()
    return
  }
  const context = getMentionContext()
  if (!context) {
    closeMention()
    return
  }
  mentionRange.value = context.range
  mentionOpen.value = true
  scheduleMentionSearch(context.query)
}

function selectMention(item: UserMentionItem) {
  const editor = editorRef.value
  const range = mentionRange.value
  if (!editor || !range) return

  editor.focus()
  range.deleteContents()

  const span = document.createElement('span')
  span.className = 'comment-mention'
  span.setAttribute('data-mention-id', String(item.id))
  span.setAttribute('data-mention-name', item.username)
  span.setAttribute('contenteditable', 'false')
  span.textContent = `@${item.username}`

  range.insertNode(span)
  const spacer = document.createTextNode(' ')
  span.after(spacer)

  const selection = window.getSelection()
  if (selection) {
    const nextRange = document.createRange()
    nextRange.setStartAfter(spacer)
    nextRange.collapse(true)
    selection.removeAllRanges()
    selection.addRange(nextRange)
  }

  emitValue()
  closeMention()
}

function handleInput() {
  if (isComposing.value) return
  emitValue()
  updateMentionState()
}

function handleCompositionStart() {
  isComposing.value = true
}

function handleCompositionEnd() {
  isComposing.value = false
  emitValue()
  updateMentionState()
}

function saveSelection() {
  const editor = editorRef.value
  const selection = window.getSelection()
  if (!editor || !selection || selection.rangeCount === 0) return
  const range = selection.getRangeAt(0)
  if (editor.contains(range.startContainer)) {
    lastSelection.value = range
  }
}

function getSelectionRange() {
  const editor = editorRef.value
  if (!editor) return null
  const selection = window.getSelection()
  if (selection && selection.rangeCount > 0) {
    const range = selection.getRangeAt(0)
    if (editor.contains(range.startContainer)) {
      return range
    }
  }
  if (lastSelection.value && editor.contains(lastSelection.value.startContainer)) {
    return lastSelection.value
  }
  return null
}

function insertText(text: string) {
  const editor = editorRef.value
  if (!editor) return
  editor.focus()
  let range = getSelectionRange()
  if (!range) {
    range = document.createRange()
    range.selectNodeContents(editor)
    range.collapse(false)
  }
  range.deleteContents()
  const node = document.createTextNode(text)
  range.insertNode(node)
  range.setStartAfter(node)
  range.collapse(true)
  const selection = window.getSelection()
  if (selection) {
    selection.removeAllRanges()
    selection.addRange(range)
  }
  emitValue()
}

function buildEmoteElement(token: string) {
  const match = token.match(EMOTE_TOKEN_RE)
  if (!match) return null
  const packId = match[1]
  const itemId = match[2]
  const key = `${packId}:${itemId}`
  const item = emoteStore.emoteMap.get(key)
  const img = document.createElement('img')
  const url = item?.url || resolveEmoteUrl(`/emotes/${packId}/${itemId}.png`)
  img.src = url
  img.alt = item?.name || item?.text || ''
  img.title = item?.text ? `${item?.name || ''} · ${item?.text}` : (item?.name || '')
  img.className = 'comment-emote'
  img.setAttribute('data-emote', `${packId}:${itemId}`)
  img.setAttribute('data-emote-token', `[[emote:${packId}:${itemId}]]`)
  img.width = 64
  img.height = 64
  return img
}

function insertToken(token: string) {
  if (props.disabled) return
  const editor = editorRef.value
  if (!editor) return
  const img = buildEmoteElement(token)
  if (!img) {
    insertText(token)
    return
  }
  editor.focus()
  let range = getSelectionRange()
  if (!range) {
    range = document.createRange()
    range.selectNodeContents(editor)
    range.collapse(false)
  }
  range.deleteContents()
  range.insertNode(img)
  const spacer = document.createTextNode(' ')
  img.after(spacer)
  range.setStartAfter(spacer)
  range.collapse(true)
  const selection = window.getSelection()
  if (selection) {
    selection.removeAllRanges()
    selection.addRange(range)
  }
  emitValue()
}

function handleKeydown(event: KeyboardEvent) {
  if (mentionOpen.value) {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      if (mentionItems.value.length > 0) {
        mentionIndex.value = (mentionIndex.value + 1) % mentionItems.value.length
      }
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      if (mentionItems.value.length > 0) {
        mentionIndex.value = (mentionIndex.value - 1 + mentionItems.value.length) % mentionItems.value.length
      }
      return
    }
    if (event.key === 'Enter' || event.key === 'Tab') {
      if (mentionItems.value.length > 0) {
        event.preventDefault()
        selectMention(mentionItems.value[mentionIndex.value])
        return
      }
    }
    if (event.key === 'Escape') {
      event.preventDefault()
      closeMention()
      return
    }
  }
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    document.execCommand('insertLineBreak')
    emitValue()
  }
}

function handleKeyup() {
  saveSelection()
  if (!isComposing.value) {
    updateMentionState()
  }
}

function handlePaste(event: ClipboardEvent) {
  event.preventDefault()
  const text = event.clipboardData?.getData('text/plain') || ''
  insertText(text)
  closeMention()
}

defineExpose({
  insertToken,
})

onMounted(async () => {
  await emoteStore.loadPacks()
  setContent(internalValue.value)
})

watch(() => props.modelValue, (next) => {
  const value = next || ''
  if (value === internalValue.value) return
  setContent(value)
})
</script>

<template>
  <div class="emote-editor">
    <div
      ref="editorRef"
      class="emote-editor-input"
      :contenteditable="!disabled"
      :data-placeholder="placeholder"
      :aria-disabled="disabled ? 'true' : 'false'"
      @input="handleInput"
      @keydown="handleKeydown"
      @paste="handlePaste"
      @compositionstart="handleCompositionStart"
      @compositionend="handleCompositionEnd"
      @mouseup="saveSelection"
      @keyup="handleKeyup"
      @focus="saveSelection"
      @blur="closeMention"
    ></div>
    <div v-if="mentionOpen" class="mention-dropdown">
      <div v-if="mentionItems.length === 0" class="mention-empty">未找到用户</div>
      <button
        v-for="(item, index) in mentionItems"
        :key="item.id"
        type="button"
        class="mention-item"
        :class="{ active: index === mentionIndex }"
        @mousedown.prevent
        @click="selectMention(item)"
      >
        <div class="mention-avatar">
          <img v-if="item.avatar" :src="item.avatar" alt="" />
          <span v-else>{{ item.username.charAt(0).toUpperCase() }}</span>
        </div>
        <span class="mention-name" :style="buildNameStyle(item.name_color, item.name_bold)">
          {{ item.username }}
        </span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.emote-editor {
  width: 100%;
  position: relative;
}

.emote-editor-input {
  min-height: 80px;
  outline: none;
  white-space: pre-wrap;
  word-break: break-word;
}

.emote-editor-input:empty::before {
  content: attr(data-placeholder);
  color: rgba(141, 123, 104, 0.6);
}

.emote-editor-input :deep(.comment-emote) {
  display: inline-block;
  vertical-align: middle;
  margin: 2px 4px 2px 0;
}

.emote-editor-input :deep(.comment-mention) {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  margin: 0 2px;
  border-radius: 999px;
  background: rgba(128, 64, 48, 0.12);
  color: #804030;
  font-weight: 600;
  font-size: 0.9em;
}

.mention-dropdown {
  position: absolute;
  left: 0;
  top: 100%;
  margin-top: 6px;
  z-index: 20;
  min-width: 220px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 10px;
  box-shadow: 0 10px 20px rgba(44, 24, 16, 0.12);
  padding: 8px;
}

.mention-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.mention-item:hover,
.mention-item.active {
  background: rgba(128, 64, 48, 0.1);
}

.mention-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 1px solid #E5D4C1;
  background: #F5EFE7;
  color: #804030;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  overflow: hidden;
}

.mention-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mention-name {
  font-size: 13px;
  color: #2C1810;
}

.mention-empty {
  padding: 8px;
  font-size: 12px;
  color: #8D7B68;
  text-align: center;
}
</style>
