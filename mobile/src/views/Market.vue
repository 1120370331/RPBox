<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { listItems, type Item, type ListItemsParams } from '@/api/item'
import { resolveApiUrl } from '@/api/image'

const { t } = useI18n()
const router = useRouter()

const items = ref<Item[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 12

const activeType = ref('')
const searchText = ref('')
const sortBy = ref<'created_at' | 'downloads' | 'rating'>('created_at')

const typeOptions = computed(() => [
  { key: '', label: t('market.types.all') },
  { key: 'item', label: t('market.types.item') },
  { key: 'campaign', label: t('market.types.campaign') },
  { key: 'artwork', label: t('market.types.artwork') },
])

const sortOptions = computed(() => [
  { key: 'created_at', label: t('market.sort.latest') },
  { key: 'downloads', label: t('market.sort.popular') },
  { key: 'rating', label: t('market.sort.rating') },
])

async function loadItems() {
  loading.value = true
  try {
    const params: ListItemsParams = {
      page: currentPage.value,
      page_size: pageSize,
      sort: sortBy.value,
      order: 'desc',
    }
    if (activeType.value) params.type = activeType.value
    if (searchText.value.trim()) params.search = searchText.value.trim()
    const res = await listItems(params)
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e) {
    console.error('Failed to load items', e)
  } finally {
    loading.value = false
  }
}

function selectType(key: string) {
  activeType.value = key
  currentPage.value = 1
}

function changeSort(key: string) {
  sortBy.value = key as typeof sortBy.value
  currentPage.value = 1
}

let searchTimer: ReturnType<typeof setTimeout>
function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadItems()
  }, 400)
}

function prevPage() {
  if (currentPage.value > 1) currentPage.value--
}
function nextPage() {
  if (currentPage.value * pageSize < total.value) currentPage.value++
}
const totalPages = () => Math.max(1, Math.ceil(total.value / pageSize))

function renderStars(rating: number) {
  const full = Math.floor(rating)
  const half = rating - full >= 0.5 ? 1 : 0
  const empty = 5 - full - half
  return '★'.repeat(full) + (half ? '½' : '') + '☆'.repeat(empty)
}

watch([activeType, sortBy, currentPage], loadItems)
onMounted(loadItems)
</script>

<template>
  <div class="page market-page">
    <header class="page-header">
      <h1>{{ $t('market.title') }}</h1>
    </header>

    <div class="search-bar">
      <i class="ri-search-line" />
      <input v-model="searchText" :placeholder="$t('market.searchPlaceholder')" @input="onSearchInput" />
    </div>

    <div class="type-bar">
      <button
        v-for="tp in typeOptions" :key="tp.key"
        :class="['type-btn', { active: activeType === tp.key }]"
        @click="selectType(tp.key)"
      >{{ tp.label }}</button>
    </div>

    <div class="sort-row">
      <button
        v-for="opt in sortOptions" :key="opt.key"
        :class="['sort-btn', { active: sortBy === opt.key }]"
        @click="changeSort(opt.key)"
      >{{ opt.label }}</button>
    </div>

    <div class="page-body">
      <div v-if="loading && items.length === 0" class="loading-hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="items.length === 0" class="empty-hint">{{ $t('market.empty') }}</div>

      <div v-else class="item-grid">
        <button
          v-for="item in items"
          :key="item.id"
          class="item-card"
          @click="router.push({ name: 'item-detail', params: { id: item.id } })"
        >
          <div class="item-preview">
            <img
              v-if="item.preview_image_url"
              :src="resolveApiUrl(item.preview_image_url)"
              alt="" loading="lazy"
            />
            <div v-else class="preview-placeholder">
              <i class="ri-box-3-line" />
            </div>
            <span class="type-badge">{{ $t('market.typeBadge.' + item.type) }}</span>
          </div>
          <div class="item-info">
            <h3 class="item-name">{{ item.name }}</h3>
            <div class="item-author">
              <img
                v-if="item.author_avatar"
                :src="resolveApiUrl(item.author_avatar)"
                class="author-avatar" alt=""
              />
              <i v-else class="ri-user-3-fill avatar-icon" />
              <span>{{ item.author_username }}</span>
            </div>
            <p class="item-desc">{{ item.description }}</p>
            <div class="item-stats">
              <span class="rating">{{ renderStars(item.rating) }} <em>{{ item.rating_count }}</em></span>
              <span><i class="ri-download-line" /> {{ item.downloads }}</span>
            </div>
          </div>
        </button>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <button :disabled="currentPage <= 1" @click="prevPage">{{ $t('common.pagination.prev') }}</button>
        <span>{{ $t('common.pagination.pageInfo', { current: currentPage, total: totalPages() }) }}</span>
        <button :disabled="currentPage >= totalPages()" @click="nextPage">{{ $t('common.pagination.next') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 0 16px 16px; }
.page-header { padding: 12px 0 8px; }
.page-header h1 { font-size: 22px; }

.search-bar {
  display: flex; align-items: center; gap: 8px;
  background: var(--input-bg); border-radius: 20px; padding: 8px 14px; margin-bottom: 10px;
}
.search-bar i { color: var(--input-placeholder); font-size: 16px; }
.search-bar input {
  flex: 1; border: none; background: transparent; outline: none;
  font-size: 14px; color: var(--text-dark);
}

.type-bar { display: flex; gap: 8px; margin-bottom: 8px; }
.type-btn {
  flex: 1; padding: 6px 0; border: 1px solid var(--color-border);
  border-radius: 16px; background: transparent; color: var(--text-dark);
  font-size: 13px; cursor: pointer; text-align: center;
}
.type-btn.active { background: var(--color-primary); color: var(--text-light); border-color: var(--color-primary); }

.sort-row { display: flex; gap: 6px; margin-bottom: 12px; }
.sort-btn {
  padding: 4px 10px; border: none; border-radius: 12px;
  background: transparent; color: var(--text-dark); font-size: 12px; cursor: pointer;
}
.sort-btn.active { background: var(--color-primary); color: var(--text-light); }

.loading-hint, .empty-hint {
  text-align: center; padding: 60px 0; color: var(--color-accent); font-size: 14px;
}

.item-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }

.item-card {
  width: 100%;
  border: none;
  text-align: left;
  cursor: pointer;
  background: var(--color-card-bg); border-radius: var(--radius-md); overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.item-preview { position: relative; width: 100%; height: 120px; overflow: hidden; background: linear-gradient(135deg, var(--color-border), var(--color-border-light)); }
.item-preview img { width: 100%; height: 100%; object-fit: cover; }
.preview-placeholder { display: flex; align-items: center; justify-content: center; height: 100%; }
.preview-placeholder i { font-size: 32px; color: var(--color-text-muted); }
.type-badge {
  position: absolute; top: 6px; left: 6px; font-size: 10px; padding: 2px 6px;
  border-radius: 6px; background: var(--overlay-bg); color: var(--overlay-text);
}

.item-info { padding: 10px; }
.item-name { font-size: 13px; font-weight: 600; margin-bottom: 4px; line-height: 1.3;
  display: -webkit-box; -webkit-line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden;
}

.item-author { display: flex; align-items: center; gap: 4px; margin-bottom: 4px; font-size: 11px; color: var(--color-text-secondary); }
.author-avatar { width: 16px; height: 16px; border-radius: 50%; object-fit: cover; }
.avatar-icon { font-size: 14px; color: var(--color-secondary); }

.item-desc {
  font-size: 11px; color: var(--color-text-secondary); line-height: 1.4; margin-bottom: 6px;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}

.item-stats { display: flex; justify-content: space-between; font-size: 11px; color: var(--color-text-secondary); }
.rating { color: var(--color-accent); }
.rating em { font-style: normal; color: var(--color-text-secondary); }

.pagination {
  display: flex; align-items: center; justify-content: center; gap: 16px; padding: 16px 0;
}
.pagination button {
  padding: 8px 16px; border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  background: transparent; color: var(--text-dark); font-size: 13px; cursor: pointer;
}
.pagination button:disabled { opacity: 0.4; cursor: default; }
.pagination span { font-size: 13px; color: var(--color-text-secondary); }
</style>
