<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Tag } from '@/api/tag'
import type { Guild } from '@/api/guild'
import RButton from './RButton.vue'
import RInput from './RInput.vue'

interface FilterValues {
  tagIds: number[]
  guildId: number | null
  search: string
  startDate: string
  endDate: string
  sort: string
  order: 'asc' | 'desc'
}

const props = defineProps<{
  tags: Tag[]
  guilds: Guild[]
}>()

const emit = defineEmits<{
  filter: [values: FilterValues]
  reset: []
}>()

const selectedTags = ref<number[]>([])
const selectedGuild = ref<number | null>(null)
const searchKeyword = ref('')
const startDate = ref('')
const endDate = ref('')
const sortBy = ref('created_at')
const sortOrder = ref<'asc' | 'desc'>('desc')

const hasActiveFilters = computed(() => {
  return selectedTags.value.length > 0 ||
    selectedGuild.value !== null ||
    searchKeyword.value !== '' ||
    startDate.value !== '' ||
    endDate.value !== ''
})

function toggleTag(tagId: number) {
  const index = selectedTags.value.indexOf(tagId)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tagId)
  }
}

function applyFilter() {
  emit('filter', {
    tagIds: selectedTags.value,
    guildId: selectedGuild.value,
    search: searchKeyword.value,
    startDate: startDate.value,
    endDate: endDate.value,
    sort: sortBy.value,
    order: sortOrder.value
  })
}

function resetFilter() {
  selectedTags.value = []
  selectedGuild.value = null
  searchKeyword.value = ''
  startDate.value = ''
  endDate.value = ''
  sortBy.value = 'created_at'
  sortOrder.value = 'desc'
  emit('reset')
}
</script>

<template>
  <div class="story-filter">
    <div class="filter-section">
      <label class="filter-label">搜索</label>
      <RInput
        v-model="searchKeyword"
        placeholder="搜索标题或描述..."
        @keyup.enter="applyFilter"
      >
        <template #prefix>
          <i class="ri-search-line"></i>
        </template>
      </RInput>
    </div>

    <div class="filter-section">
      <label class="filter-label">标签筛选</label>
      <div class="tag-filters">
        <span
          v-for="tag in tags"
          :key="tag.id"
          class="filter-tag"
          :class="{ active: selectedTags.includes(tag.id) }"
          :style="selectedTags.includes(tag.id) ? { background: `#${tag.color}`, color: '#fff' } : { borderColor: `#${tag.color}`, color: `#${tag.color}` }"
          @click="toggleTag(tag.id)"
        >
          {{ tag.name }}
        </span>
      </div>
    </div>

    <div class="filter-section">
      <label class="filter-label">公会筛选</label>
      <select v-model="selectedGuild" class="filter-select">
        <option :value="null">全部公会</option>
        <option v-for="guild in guilds" :key="guild.id" :value="guild.id">
          {{ guild.name }}
        </option>
      </select>
    </div>

    <div class="filter-section">
      <label class="filter-label">日期范围</label>
      <div class="date-range">
        <input v-model="startDate" type="date" class="date-input" />
        <span class="date-separator">至</span>
        <input v-model="endDate" type="date" class="date-input" />
      </div>
    </div>

    <div class="filter-section">
      <label class="filter-label">排序</label>
      <div class="sort-controls">
        <select v-model="sortBy" class="filter-select">
          <option value="created_at">创建时间</option>
          <option value="updated_at">更新时间</option>
          <option value="start_time">剧情时间</option>
        </select>
        <button
          class="sort-order-btn"
          @click="sortOrder = sortOrder === 'desc' ? 'asc' : 'desc'"
        >
          <i :class="sortOrder === 'desc' ? 'ri-sort-desc' : 'ri-sort-asc'"></i>
        </button>
      </div>
    </div>

    <div class="filter-actions">
      <RButton v-if="hasActiveFilters" @click="resetFilter">重置</RButton>
      <RButton type="primary" @click="applyFilter">应用筛选</RButton>
    </div>
  </div>
</template>

<style scoped>
.story-filter {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px;
  background: linear-gradient(135deg, #f5f0eb 0%, #ebe4dc 100%);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.filter-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.filter-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
}

.tag-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.filter-tag {
  padding: 6px 12px;
  border: 1.5px solid;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.filter-tag:hover {
  opacity: 0.8;
}

.filter-tag.active {
  font-weight: 600;
}

.filter-select {
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  font-size: 14px;
  background: #fff;
  color: var(--color-primary);
  cursor: pointer;
}

.filter-select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.date-range {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-input {
  flex: 1;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  font-size: 14px;
  color: var(--color-primary);
}

.date-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.date-separator {
  font-size: 13px;
  color: var(--color-secondary);
}

.sort-controls {
  display: flex;
  gap: 8px;
}

.sort-controls .filter-select {
  flex: 1;
}

.sort-order-btn {
  padding: 10px 16px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: #fff;
  color: var(--color-primary);
  cursor: pointer;
  transition: all 0.2s;
}

.sort-order-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.sort-order-btn i {
  font-size: 18px;
}

.filter-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding-top: 8px;
  border-top: 1px solid var(--color-border);
}
</style>
