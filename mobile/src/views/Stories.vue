<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { listStories, type Story } from '@/api/story'

const { t } = useI18n()
const router = useRouter()
const stories = ref<Story[]>([])
const loading = ref(false)
const searchText = ref('')
const sortBy = ref('updated_at')

const sortOptions = computed(() => [
  { key: 'updated_at', label: t('stories.sort.recentUpdate') },
  { key: 'created_at', label: t('stories.sort.newest') },
  { key: 'view_count', label: t('stories.sort.mostViewed') },
])

async function loadStories() {
  loading.value = true
  try {
    const params: Record<string, string> = {
      sort: sortBy.value,
      order: 'desc',
      is_public: 'true',
    }
    if (searchText.value.trim()) params.search = searchText.value.trim()
    const res = await listStories(params)
    stories.value = res.stories || []
  } catch (e) {
    console.error('Failed to load stories', e)
  } finally {
    loading.value = false
  }
}

function changeSort(key: string) {
  sortBy.value = key
}

function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

let searchTimer: ReturnType<typeof setTimeout>
function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(loadStories, 400)
}

watch(sortBy, loadStories)
onMounted(loadStories)
</script>

<template>
  <div class="page stories-page">
    <header class="page-header">
      <h1>{{ $t('stories.title') }}</h1>
    </header>

    <div class="search-bar">
      <i class="ri-search-line" />
      <input
        v-model="searchText"
        :placeholder="$t('stories.searchPlaceholder')"
        @input="onSearchInput"
      />
    </div>

    <div class="sort-row">
      <button
        v-for="opt in sortOptions" :key="opt.key"
        :class="['sort-btn', { active: sortBy === opt.key }]"
        @click="changeSort(opt.key)"
      >{{ opt.label }}</button>
    </div>

    <div class="page-body">
      <div v-if="loading && stories.length === 0" class="loading-hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="stories.length === 0" class="empty-hint">{{ $t('stories.empty') }}</div>

      <div v-else class="story-list">
        <button
          v-for="story in stories"
          :key="story.id"
          class="story-card"
          @click="router.push({ name: 'story-detail', params: { id: story.id } })"
        >
          <div class="story-header">
            <h3 class="story-title">{{ story.title }}</h3>
            <span class="story-status" :class="story.status">{{ story.status === 'active' ? $t('stories.status.active') : $t('stories.status.ended') }}</span>
          </div>
          <p v-if="story.description" class="story-desc">{{ story.description }}</p>
          <div v-if="story.tag_list?.length" class="story-tags">
            <span
              v-for="tag in story.tag_list" :key="tag.name"
              class="tag"
              :style="{ background: tag.color ? tag.color + '20' : undefined, color: tag.color || undefined }"
            >{{ tag.name }}</span>
          </div>
          <div class="story-meta">
            <span v-if="story.entry_count"><i class="ri-file-text-line" /> {{ $t('stories.entryCount', { n: story.entry_count }) }}</span>
            <span><i class="ri-eye-line" /> {{ story.view_count }}</span>
            <span class="story-date">{{ formatDate(story.updated_at) }}</span>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding: calc(var(--safe-top, 0px) + 2px) var(--page-gutter) calc(26px + var(--safe-bottom, 0px));
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.page-header { padding: 6px 0 0; }
.page-header h1 { font-size: 24px; line-height: 1.1; letter-spacing: 0.01em; }

.search-bar {
  display: flex; align-items: center; gap: 8px;
  background: var(--input-bg);
  border: 1px solid rgba(75, 54, 33, 0.12);
  border-radius: 20px;
  padding: 9px 14px;
}
.search-bar i { color: var(--input-placeholder); font-size: 16px; }
.search-bar input {
  flex: 1; border: none; background: transparent; outline: none;
  font-size: 14px; color: var(--text-dark);
}

.sort-row { display: flex; gap: 8px; flex-wrap: wrap; }
.sort-btn {
  padding: 6px 12px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: rgba(128, 64, 48, 0.08);
  color: var(--text-dark);
  font-size: 12px;
  cursor: pointer;
}
.sort-btn.active { background: var(--color-primary); border-color: var(--color-primary); color: var(--text-light); }

.loading-hint, .empty-hint {
  text-align: center; padding: 60px 0; color: var(--color-accent); font-size: 14px;
}

.story-list { display: flex; flex-direction: column; gap: 14px; }

.story-card {
  width: 100%;
  border: none;
  text-align: left;
  cursor: pointer;
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  border: 1px solid rgba(75, 54, 33, 0.08);
  padding: 14px;
  box-shadow: var(--shadow-sm);
}

.story-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.story-title { font-size: 15px; font-weight: 600; flex: 1; }
.story-status {
  font-size: 11px; padding: 2px 8px; border-radius: 8px; flex-shrink: 0; margin-left: 8px;
  background: var(--color-primary-light); color: var(--color-text-secondary);
}
.story-status.active { background: var(--color-success-light); color: var(--color-success); }

.story-desc {
  font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; margin-bottom: 8px;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}

.story-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 8px; }
.tag {
  font-size: 11px; padding: 2px 8px; border-radius: 8px;
  background: var(--tag-bg); color: var(--tag-text);
}

.story-meta { display: flex; gap: 14px; font-size: 12px; color: var(--color-text-secondary); align-items: center; }
.story-meta i { margin-right: 2px; }
.story-date { margin-left: auto; }

@media (max-width: 360px) {
  .page-header h1 { font-size: 22px; }
  .story-title { font-size: 14px; }
}
</style>
