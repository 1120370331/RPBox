<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  listRPDBHotWorks,
  listRPDBWorks,
  resolveRPDBMediaUrl,
  type ListRPDBWorksParams,
  type RPDBWork,
  type RPDBWorkType,
} from '@/api/rpdb'
import CachedImage from '@/components/CachedImage.vue'
import MobilePagination from '@/components/MobilePagination.vue'
import MobileRPDBWorkCard from '@/components/MobileRPDBWorkCard.vue'
import { getPresetTags, type Tag } from '@/api/tag'
import { getCachedListState, restoreScrollTop, useListStateCache } from '@/utils/listState'
import { getRPDBSummary, getRPDBTypeIcon, getRPDBTypeLabel } from '@/utils/rpdb'

const router = useRouter()
const LIST_STATE_KEY = 'rpdb'
const pageSize = 12

interface RPDBListState {
  page: number
  type: RPDBWorkType | ''
  search: string
  sort: NonNullable<ListRPDBWorksParams['sort']>
  availability: string
  faction: string
  armorType: string
  bindType: string
  tagId?: number
  scrollTop: number
}

const cachedState = getCachedListState<RPDBListState>(LIST_STATE_KEY)
const works = ref<RPDBWork[]>([])
const hotWorks = ref<RPDBWork[]>([])
const total = ref(0)
const loading = ref(false)
const hotLoading = ref(false)
const error = ref('')
const filterSheetOpen = ref(false)
const styleTags = ref<Tag[]>([])
const requestSerial = ref(0)
const filters = reactive<Required<Pick<RPDBListState, 'page' | 'type' | 'search' | 'sort'>>>({
  page: cachedState?.page || 1,
  type: cachedState?.type || '',
  search: cachedState?.search || '',
  sort: cachedState?.sort || 'updated_at',
})
const advancedFilters = reactive({
  availability: cachedState?.availability || '',
  faction: cachedState?.faction || '',
  armorType: cachedState?.armorType || '',
  bindType: cachedState?.bindType || '',
  tagId: cachedState?.tagId,
})
let searchTimer: ReturnType<typeof setTimeout> | null = null
let shouldRestoreInitialScroll = Boolean(cachedState?.scrollTop)

const channels: Array<{ id: RPDBWorkType | ''; label: string; icon: string }> = [
  { id: '', label: '全部', icon: 'ri-compass-3-line' },
  { id: 'item_showcase', label: '物品', icon: 'ri-magic-line' },
  { id: 'transmog', label: '幻化', icon: 'ri-shirt-line' },
  { id: 'home_showcase', label: '家宅', icon: 'ri-home-heart-line' },
  { id: 'musician_midi', label: 'MIDI', icon: 'ri-music-2-line' },
]

const sortOptions: Array<{ value: NonNullable<ListRPDBWorksParams['sort']>; label: string }> = [
  { value: 'updated_at', label: '最近更新' },
  { value: 'popular', label: '最多浏览' },
  { value: 'favorite', label: '最多收藏' },
  { value: 'verified', label: '最多验证' },
]

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const resultTitle = computed(() => filters.type ? getRPDBTypeLabel(filters.type) : '最新作品')
const activeAdvancedFilterCount = computed(() => [
  advancedFilters.availability,
  advancedFilters.faction,
  advancedFilters.armorType,
  advancedFilters.bindType,
  advancedFilters.tagId,
].filter(Boolean).length)

async function loadWorks() {
  const serial = ++requestSerial.value
  loading.value = true
  error.value = ''
  try {
    const result = await listRPDBWorks({
      search: filters.search.trim(),
      type: filters.type,
      sort: filters.sort,
      availability_status: advancedFilters.availability,
      faction: advancedFilters.faction,
      armor_type: advancedFilters.armorType,
      bind_type: advancedFilters.bindType,
      tag_id: advancedFilters.tagId,
      page: filters.page,
      page_size: pageSize,
    })
    if (serial !== requestSerial.value) return
    works.value = result.works || []
    total.value = result.total || 0
    const maxPage = Math.max(1, Math.ceil(total.value / pageSize))
    if (filters.page > maxPage) {
      filters.page = maxPage
    }
  } catch (cause) {
    if (serial !== requestSerial.value) return
    error.value = (cause as Error).message || '作品加载失败'
  } finally {
    if (serial === requestSerial.value) loading.value = false
  }
}

async function loadHotWorks() {
  hotLoading.value = true
  try {
    const result = await listRPDBHotWorks({ type: filters.type, limit: 3 })
    hotWorks.value = result.works || []
  } catch {
    hotWorks.value = []
  } finally {
    hotLoading.value = false
  }
}

function selectType(type: RPDBWorkType | '') {
  if (filters.type === type) return
  filters.type = type
  if (type !== 'item_showcase') advancedFilters.bindType = ''
  filters.page = 1
  void loadHotWorks()
}

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    if (filters.page === 1) {
      void loadWorks()
    } else {
      filters.page = 1
    }
  }, 350)
}

function clearSearch() {
  filters.search = ''
  if (filters.page === 1) {
    void loadWorks()
  } else {
    filters.page = 1
  }
}

function applyAdvancedFilters() {
  filterSheetOpen.value = false
  if (filters.page === 1) void loadWorks()
  else filters.page = 1
}

function resetAdvancedFilters() {
  advancedFilters.availability = ''
  advancedFilters.faction = ''
  advancedFilters.armorType = ''
  advancedFilters.bindType = ''
  advancedFilters.tagId = undefined
}

function openWork(id: number) {
  saveListState()
  void router.push({ name: 'rpdb-detail', params: { id } })
}

function onPageChange(page: number) {
  if (page === filters.page) return
  filters.page = page
  document.querySelector('.mobile-content')?.scrollTo({ top: 0, behavior: 'smooth' })
}

const { save: saveListState } = useListStateCache<RPDBListState>({
  key: LIST_STATE_KEY,
  getState: () => ({
    page: filters.page,
    type: filters.type,
    search: filters.search,
    sort: filters.sort,
    availability: advancedFilters.availability,
    faction: advancedFilters.faction,
    armorType: advancedFilters.armorType,
    bindType: advancedFilters.bindType,
    tagId: advancedFilters.tagId,
    scrollTop: 0,
  }),
})

watch(() => [filters.type, filters.sort, filters.page], loadWorks)

onMounted(async () => {
  await Promise.all([
    loadWorks(),
    loadHotWorks(),
    getPresetTags('rpdb').then(result => {
      styleTags.value = (result.tags || []).filter(tag => tag.name.endsWith('风格'))
    }).catch(() => {
      styleTags.value = []
    }),
  ])
  if (shouldRestoreInitialScroll) {
    shouldRestoreInitialScroll = false
    restoreScrollTop(cachedState?.scrollTop || 0)
  }
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<template>
  <div class="rpdb-page">
    <header class="rpdb-header">
      <div>
        <span class="archive-label">PLAYER ARCHIVE</span>
        <h1>RP 数据库</h1>
        <p>物品特效、幻化方案与玩家家宅档案</p>
      </div>
      <button class="archive-seal" type="button" aria-label="发布 RP 数据库内容" @click="router.push({ name: 'rpdb-create' })">
        <i class="ri-archive-stack-line" />
        <span><i class="ri-add-line" /></span>
      </button>
    </header>

    <nav class="workspace-actions" aria-label="RP 数据库个人工作区">
      <button type="button" @click="router.push({ name: 'rpdb-create' })">
        <i class="ri-draft-line" />
        <span><b>发布内容</b><small>创建作品档案</small></span>
      </button>
      <button type="button" @click="router.push({ name: 'rpdb-my-uploads' })">
        <i class="ri-upload-cloud-2-line" />
        <span><b>我的上传</b><small>草稿与审核状态</small></span>
      </button>
      <button type="button" @click="router.push({ name: 'rpdb-lists' })">
        <i class="ri-list-check-3" />
        <span><b>收集清单</b><small>进度与路线导出</small></span>
      </button>
    </nav>

    <div class="search-row">
      <div class="search-box">
        <i class="ri-search-line" />
        <input v-model="filters.search" type="search" placeholder="搜索名称、效果或获取方式" @input="onSearchInput">
        <button v-if="filters.search" type="button" aria-label="清空搜索" @click="clearSearch">
          <i class="ri-close-circle-fill" />
        </button>
      </div>
      <button class="filter-button" type="button" aria-label="打开高级筛选" :class="{ active: activeAdvancedFilterCount }" @click="filterSheetOpen = true">
        <i class="ri-equalizer-2-line" />
        <span v-if="activeAdvancedFilterCount">{{ activeAdvancedFilterCount }}</span>
      </button>
    </div>

    <nav class="channel-strip" aria-label="内容分类">
      <button
        v-for="channel in channels"
        :key="channel.id || 'all'"
        type="button"
        :class="{ active: filters.type === channel.id }"
        @click="selectType(channel.id)"
      >
        <i :class="channel.icon" />
        {{ channel.label }}
      </button>
    </nav>

    <section v-if="hotLoading || hotWorks.length" class="hot-section">
      <header class="section-title">
        <div>
          <span>近 7 日</span>
          <h2>玩家正在查看</h2>
        </div>
        <i v-if="hotLoading" class="ri-loader-4-line spin" />
      </header>

      <div class="hot-rail">
        <button v-for="(work, index) in hotWorks" :key="work.id" class="hot-card" type="button" @click="openWork(work.id)">
          <CachedImage
            v-if="work.cover_image"
            :src="resolveRPDBMediaUrl(work.cover_image)"
            :alt="work.title"
            loading="eager"
          />
          <div v-else class="hot-placeholder"><i :class="getRPDBTypeIcon(work.type)" /></div>
          <div class="hot-shade" />
          <span class="hot-index">{{ String(index + 1).padStart(2, '0') }}</span>
          <div class="hot-copy">
            <small>{{ getRPDBTypeLabel(work.type) }}</small>
            <h3>{{ work.title }}</h3>
            <p>{{ getRPDBSummary(work) }}</p>
          </div>
        </button>
      </div>
    </section>

    <section class="feed-section">
      <header class="feed-head">
        <div>
          <span>{{ total }} 份玩家档案</span>
          <h2>{{ resultTitle }}</h2>
        </div>
        <label class="sort-select">
          <i class="ri-sort-desc" />
          <select v-model="filters.sort" aria-label="排序方式">
            <option v-for="option in sortOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
      </header>

      <div v-if="loading && !works.length" class="work-grid">
        <div v-for="index in 6" :key="index" class="work-skeleton">
          <div />
          <span />
          <span />
        </div>
      </div>

      <div v-else-if="error" class="page-state">
        <i class="ri-error-warning-line" />
        <b>没有加载出作品</b>
        <p>{{ error }}</p>
        <button type="button" @click="loadWorks">重新加载</button>
      </div>

      <div v-else-if="!works.length" class="page-state">
        <i class="ri-archive-drawer-line" />
        <b>这个分类还没有作品</b>
        <p>换个关键词，或者浏览其他内容分类。</p>
      </div>

      <div v-else class="work-grid">
        <MobileRPDBWorkCard v-for="work in works" :key="work.id" :work="work" @open="openWork(work.id)" />
      </div>

      <MobilePagination
        v-if="totalPages > 1"
        :model-value="filters.page"
        :total-pages="totalPages"
        :disabled="loading"
        @change="onPageChange"
      />
    </section>

    <Teleport to="body">
      <div v-if="filterSheetOpen" class="filter-mask" @click.self="filterSheetOpen = false">
        <section class="filter-sheet" role="dialog" aria-modal="true" aria-label="高级筛选">
          <header>
            <div><small>精确查找</small><h2>高级筛选</h2></div>
            <button type="button" aria-label="关闭" @click="filterSheetOpen = false"><i class="ri-close-line" /></button>
          </header>
          <div class="filter-grid">
            <label v-if="filters.type !== 'musician_midi'">
              <span>获取状态</span>
              <select v-model="advancedFilters.availability">
                <option value="">全部状态</option>
                <option value="available">可获取</option>
                <option value="limited">限时获取</option>
                <option value="removed">已绝版</option>
                <option value="unknown">未知</option>
              </select>
            </label>
            <label v-if="filters.type !== 'musician_midi'">
              <span>阵营</span>
              <select v-model="advancedFilters.faction">
                <option value="">全部阵营</option>
                <option value="neutral">中立</option>
                <option value="alliance">联盟</option>
                <option value="horde">部落</option>
              </select>
            </label>
            <label v-if="filters.type === 'transmog' || !filters.type">
              <span>护甲类型</span>
              <select v-model="advancedFilters.armorType">
                <option value="">全部护甲</option>
                <option value="cloth">布甲</option>
                <option value="leather">皮甲</option>
                <option value="mail">锁甲</option>
                <option value="plate">板甲</option>
                <option value="cosmetic">装饰品</option>
              </select>
            </label>
            <label v-if="filters.type === 'item_showcase'">
              <span>绑定状态</span>
              <select v-model="advancedFilters.bindType">
                <option value="">全部绑定状态</option>
                <option value="yes">绑定</option>
                <option value="no">不绑定</option>
              </select>
            </label>
            <label v-if="styleTags.length">
              <span>扮演风格</span>
              <select v-model.number="advancedFilters.tagId">
                <option :value="undefined">全部风格</option>
                <option v-for="tag in styleTags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
              </select>
            </label>
          </div>
          <footer>
            <button type="button" class="secondary" @click="resetAdvancedFilters">重置</button>
            <button type="button" class="primary" @click="applyAdvancedFilters">查看结果</button>
          </footer>
        </section>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.rpdb-page {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  gap: 16px;
  padding:
    calc(var(--safe-top, 0px) + 4px)
    var(--page-gutter)
    calc(28px + var(--safe-bottom, 0px));
}

.rpdb-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 8px 2px 2px;
}

.archive-label,
.section-title span,
.feed-head span {
  display: block;
  color: var(--color-accent);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.rpdb-header h1 {
  margin: 4px 0 3px;
  font-family: Georgia, 'Microsoft YaHei', serif;
  font-size: 27px;
  line-height: 1.1;
}

.rpdb-header p {
  color: var(--color-text-secondary);
  font-size: 11px;
}

.archive-seal {
  display: grid;
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  place-items: center;
  border: 1px solid rgba(128, 64, 48, 0.25);
  border-radius: 8px;
  background: var(--color-primary);
  color: #e8b979;
  box-shadow: inset 0 0 0 4px rgba(255, 255, 255, 0.07);
  position: relative;
}

.archive-seal i {
  font-size: 23px;
}

.archive-seal span {
  position: absolute;
  right: -5px;
  bottom: -5px;
  display: grid;
  width: 20px;
  height: 20px;
  place-items: center;
  border: 2px solid var(--color-background);
  border-radius: 50%;
  background: var(--color-accent);
  color: #fff;
}

.archive-seal span i {
  font-size: 12px;
}

.workspace-actions {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid rgba(75, 54, 33, 0.12);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.58);
  box-shadow: var(--shadow-sm);
}

.workspace-actions button {
  display: flex;
  min-width: 0;
  min-height: 58px;
  align-items: center;
  gap: 7px;
  padding: 8px;
  border: 0;
  border-right: 1px solid rgba(75, 54, 33, 0.1);
  background: transparent;
  color: var(--color-text-main);
  text-align: left;
}

.workspace-actions button:last-child {
  border-right: 0;
}

.workspace-actions > button > i {
  flex: 0 0 auto;
  color: var(--color-secondary);
  font-size: 19px;
}

.workspace-actions span {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.workspace-actions b,
.workspace-actions small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-actions b {
  font-size: 11px;
}

.workspace-actions small {
  margin-top: 3px;
  color: var(--color-text-secondary);
  font-size: 8px;
}

.search-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 44px;
  gap: 8px;
}

.search-box {
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 9px;
  padding: 0 12px;
  border: 1px solid rgba(75, 54, 33, 0.14);
  border-radius: 8px;
  background: var(--color-panel-bg);
  box-shadow: var(--shadow-sm);
}

.filter-button {
  position: relative;
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border: 1px solid rgba(75, 54, 33, 0.14);
  border-radius: 8px;
  background: var(--color-panel-bg);
  color: var(--color-secondary);
  font-size: 18px;
  box-shadow: var(--shadow-sm);
}

.filter-button.active {
  background: var(--color-primary);
  color: #fff;
}

.filter-button span {
  position: absolute;
  top: -5px;
  right: -5px;
  display: grid;
  min-width: 18px;
  height: 18px;
  place-items: center;
  border: 2px solid var(--color-background);
  border-radius: 9px;
  background: var(--color-accent);
  color: #fff;
  font-size: 8px;
  font-weight: 800;
}

.search-box > i {
  color: var(--color-secondary);
  font-size: 17px;
}

.search-box input {
  min-width: 0;
  flex: 1;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text-main);
  font-size: 13px;
}

.search-box button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 0;
  background: transparent;
  color: var(--color-text-muted);
  font-size: 17px;
}

.channel-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.channel-strip button {
  display: flex;
  min-width: 0;
  min-height: 42px;
  align-items: center;
  justify-content: center;
  gap: 5px;
  border: 1px solid var(--color-border);
  border-radius: 7px;
  background: rgba(255, 255, 255, 0.46);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.channel-strip button.active {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: #fff;
}

.hot-section,
.feed-section {
  min-width: 0;
}

.section-title,
.feed-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  padding: 0 2px;
}

.section-title h2,
.feed-head h2 {
  margin-top: 3px;
  font-family: Georgia, 'Microsoft YaHei', serif;
  font-size: 19px;
  line-height: 1.2;
}

.hot-rail {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  scrollbar-width: none;
}

.hot-rail::-webkit-scrollbar {
  display: none;
}

.hot-card {
  position: relative;
  width: min(82vw, 318px);
  min-width: min(82vw, 318px);
  aspect-ratio: 16 / 10;
  overflow: hidden;
  border: 0;
  border-radius: 8px;
  background: var(--color-primary);
  color: #fff;
  scroll-snap-align: start;
  text-align: left;
  box-shadow: var(--shadow-md);
}

.hot-card :deep(.cached-image),
.hot-placeholder,
.hot-shade {
  position: absolute;
  inset: 0;
}

.hot-placeholder {
  display: grid;
  place-items: center;
  background: linear-gradient(145deg, #4b3621, #804030);
  color: #d6a66c;
}

.hot-placeholder i {
  font-size: 44px;
}

.hot-shade {
  background: linear-gradient(180deg, rgba(44, 24, 16, 0.05) 22%, rgba(44, 24, 16, 0.9) 100%);
}

.hot-index {
  position: absolute;
  top: 10px;
  left: 11px;
  color: rgba(255, 255, 255, 0.88);
  font: 800 22px/1 Georgia, serif;
}

.hot-copy {
  position: absolute;
  right: 14px;
  bottom: 13px;
  left: 14px;
}

.hot-copy small {
  color: #f0bb7c;
  font-size: 10px;
  font-weight: 800;
}

.hot-copy h3 {
  overflow: hidden;
  margin: 4px 0;
  font-family: Georgia, 'Microsoft YaHei', serif;
  font-size: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hot-copy p {
  overflow: hidden;
  color: rgba(255, 255, 255, 0.76);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sort-select {
  display: flex;
  min-height: 34px;
  align-items: center;
  gap: 3px;
  padding: 0 8px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-panel-bg);
  color: var(--color-secondary);
}

.sort-select select {
  max-width: 84px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text-main);
  font-size: 11px;
}

.work-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.work-skeleton {
  overflow: hidden;
  border-radius: 8px;
  background: var(--color-card-bg);
}

.work-skeleton div,
.work-skeleton span {
  display: block;
  background: linear-gradient(90deg, #eadfd4, #fff, #eadfd4);
  background-size: 220% 100%;
  animation: shimmer 1.1s linear infinite;
}

.work-skeleton div {
  aspect-ratio: 4 / 3;
}

.work-skeleton span {
  width: 72%;
  height: 10px;
  margin: 10px;
  border-radius: 4px;
}

.work-skeleton span:last-child {
  width: 46%;
}

.page-state {
  display: grid;
  min-height: 220px;
  place-items: center;
  align-content: center;
  gap: 7px;
  padding: 24px;
  color: var(--color-text-secondary);
  text-align: center;
}

.page-state > i {
  color: var(--color-secondary);
  font-size: 34px;
}

.page-state b {
  color: var(--color-text-main);
  font-size: 15px;
}

.page-state p {
  font-size: 12px;
  line-height: 1.5;
}

.page-state button {
  min-height: 38px;
  margin-top: 4px;
  padding: 0 14px;
  border: 0;
  border-radius: 6px;
  background: var(--color-primary);
  color: #fff;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes shimmer {
  from { background-position: 200% 0; }
  to { background-position: -20% 0; }
}

@media (max-width: 350px) {
  .rpdb-header h1 {
    font-size: 24px;
  }

  .channel-strip button {
    gap: 3px;
    font-size: 11px;
  }

  .work-grid {
    grid-template-columns: 1fr;
  }

  .workspace-actions small {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .spin,
  .work-skeleton div,
  .work-skeleton span {
    animation: none;
  }
}

.filter-mask {
  position: fixed;
  inset: 0;
  z-index: 2200;
  display: flex;
  align-items: flex-end;
  background: rgba(44, 24, 16, 0.48);
  backdrop-filter: blur(3px);
}

.filter-sheet {
  width: 100%;
  padding: 16px var(--page-gutter) calc(18px + var(--safe-bottom, 0px));
  border-radius: 14px 14px 0 0;
  background: var(--color-panel-bg);
}

.filter-sheet header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.filter-sheet header small {
  color: var(--color-accent);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.filter-sheet h2 {
  margin-top: 3px;
  font-size: 21px;
}

.filter-sheet header button {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border: 1px solid var(--color-border);
  border-radius: 7px;
  background: var(--color-card-bg);
  font-size: 18px;
}

.filter-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 11px;
  margin-top: 16px;
}

.filter-grid label {
  display: grid;
  gap: 6px;
}

.filter-grid span {
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.filter-grid select {
  width: 100%;
  min-width: 0;
  height: 42px;
  padding: 0 9px;
  border: 1px solid var(--input-border);
  border-radius: 7px;
  background: var(--input-bg);
  color: var(--color-text-main);
}

.filter-sheet footer {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 16px;
}

.filter-sheet footer button {
  min-height: 42px;
  border-radius: 7px;
  font-weight: 800;
}

.filter-sheet footer .secondary {
  border: 1px solid var(--color-border);
  background: var(--color-card-bg);
  color: var(--color-text-main);
}

.filter-sheet footer .primary {
  border: 0;
  background: var(--color-primary);
  color: #fff;
}
</style>
