<script setup lang="ts">
import { ref, computed } from 'vue'
import { createTag, type Tag } from '@/api/tag'
import { contrastingTextColor } from '@/utils/displayColor'

const props = defineProps<{
  selectedTags: Tag[]
  allTags: Tag[]
}>()

const emit = defineEmits<{
  add: [tagId: number]
  remove: [tagId: number]
  create: [tag: Tag]
}>()

const showCreate = ref(false)
const newTagName = ref('')
const newTagColor = ref('#B87333')

// 可选的标签（排除已选中的）
const availableTags = computed(() => {
  const selectedIds = props.selectedTags.map(t => t.id)
  return props.allTags.filter(t => !selectedIds.includes(t.id))
})

// Keep the saved dye intact; only choose the foreground for a filled tag.
function tagStyle(tag: Tag) {
  const color = tag.color || 'B87333'
  return {
    '--tag-color': '#' + color,
    '--tag-contrast': contrastingTextColor('#' + color),
  }
}

function handleAdd(tagId: number) {
  emit('add', tagId)
}

function handleRemove(tagId: number) {
  emit('remove', tagId)
}

async function handleCreate() {
  if (!newTagName.value.trim()) return
  try {
    const color = newTagColor.value.replace('#', '')
    const tag = await createTag({ name: newTagName.value, color })
    emit('create', tag)
    emit('add', tag.id)
    newTagName.value = ''
    newTagColor.value = '#B87333'
    showCreate.value = false
  } catch (e) {
    console.error('创建标签失败:', e)
  }
}
</script>

<template>
  <div class="tag-selector">
    <!-- 已选中的标签 -->
    <div v-if="selectedTags.length > 0" class="section">
      <div class="section-title">已添加的标签</div>
      <div class="tag-list">
        <span
          v-for="tag in selectedTags"
          :key="tag.id"
          class="tag-item selected"
          :style="tagStyle(tag)"
        >
          {{ tag.name }}
          <i class="ri-close-line" @click="handleRemove(tag.id)"></i>
        </span>
      </div>
    </div>

    <!-- 可选的标签 -->
    <div class="section">
      <div class="section-title">可用标签</div>
      <div class="tag-list">
        <span
          v-for="tag in availableTags"
          :key="tag.id"
          class="tag-item clickable"
          :style="tagStyle(tag)"
          @click="handleAdd(tag.id)"
        >
          <i class="ri-add-line"></i> {{ tag.name }}
        </span>
        <span v-if="availableTags.length === 0 && allTags.length === 0" class="empty-hint">
          暂无可用标签
        </span>
      </div>
    </div>

    <!-- 创建新标签 -->
    <div class="section">
      <div class="section-title">创建新标签</div>
      <div class="create-form">
        <input
          v-model="newTagName"
          placeholder="输入标签名称"
          @keyup.enter="handleCreate"
        />
        <input
          v-model="newTagColor"
          type="color"
          class="color-picker"
        />
        <button class="btn-create" @click="handleCreate">创建</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tag-selector {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  padding: 6px 12px;
  border-radius: 16px;
  font-size: 13px;
  background: color-mix(in srgb, var(--tag-color) 10%, var(--color-panel-bg));
  color: var(--color-text-main);
  border: 1.5px solid var(--tag-color);
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tag-item.clickable {
  cursor: pointer;
}

.tag-item.clickable:hover {
  background: var(--tag-color);
  color: var(--tag-contrast);
  border-color: var(--color-border-hover);
}

.tag-item.selected {
  background: var(--tag-color);
  color: var(--tag-contrast);
  border-color: var(--color-border-hover);
}

.tag-item.selected i {
  cursor: pointer;
  opacity: 0.8;
}

.tag-item.selected i:hover {
  opacity: 1;
}

.empty-hint {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.create-form {
  display: flex;
  gap: 8px;
  align-items: center;
}

.create-form input {
  padding: 8px 12px;
  border: 1px solid var(--input-border);
  background: var(--input-bg);
  color: var(--color-text-main);
  border-radius: 6px;
  font-size: 13px;
  flex: 1;
}

.create-form input:focus {
  outline: none;
  border-color: var(--input-focus);
}

.create-form .color-picker {
  width: 36px;
  height: 36px;
  padding: 2px;
  border: 1px solid var(--input-border);
  border-radius: 6px;
  cursor: pointer;
  flex: none;
}

.btn-create {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
  font-weight: 500;
}

.btn-create:hover {
  background: var(--btn-primary-hover);
}
</style>
