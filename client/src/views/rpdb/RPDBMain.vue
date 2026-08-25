<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listRPDBHotWorks, listRPDBWorks, resolveRPDBMediaURL, type ListRPDBWorksParams, type RPDBWork, type RPDBWorkType } from '@/api/rpdb'
import { getPresetTags, type Tag } from '@/api/tag'
import RPDBFilterBar from '@/components/rpdb/RPDBFilterBar.vue'
import RPDBWorkCard from '@/components/rpdb/RPDBWorkCard.vue'
import { isRPDBStyleTag, sortRPDBStyleTags } from '@/constants/rpdbStyles'

const router = useRouter()
const works = ref<RPDBWork[]>([])
const hotWorks = ref<RPDBWork[]>([])
const total = ref(0)
const loading = ref(true)
const hotLoading = ref(true)
const error = ref('')
const styleTags = ref<Tag[]>([])
const pageSize = 12
const filters = reactive<ListRPDBWorksParams>({ search: '', type: '', sort: 'updated_at', page: 1, page_size: pageSize })
type ViewMode = 'card' | 'compact'
const VIEW_MODE_KEY = 'rpdb-view-mode'
const savedViewMode = typeof window !== 'undefined' ? window.localStorage.getItem(VIEW_MODE_KEY) : null
const viewMode = ref<ViewMode>(savedViewMode === 'compact' ? 'compact' : 'card')

const hotSlots = computed(() => Array.from({ length: 3 }, (_, index) => hotWorks.value[index] || null))
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const pageStart = computed(() => total.value ? ((filters.page || 1) - 1) * pageSize + 1 : 0)
const pageEnd = computed(() => Math.min(total.value, (filters.page || 1) * pageSize))
const resultTitle = computed(() => ({
  item_showcase: '魔兽物品',
  transmog: '幻化方案',
  home_showcase: '家宅分享',
  musician_midi: 'Musician MIDI',
}[filters.type as RPDBWorkType] || '最新发布'))
const hotTitle = computed(() => {
  if (!filters.type) return '热度 Top3 · 近7日'
  return `热度 Top3 · ${typeLabel(filters.type as RPDBWorkType)} · 近7日`
})

const channels: Array<{ id: RPDBWorkType | ''; icon: string; label: string }> = [
  { id: '', icon: 'ri-layout-grid-line', label: '全部内容' },
  { id: 'item_showcase', icon: 'ri-magic-line', label: '魔兽物品' },
  { id: 'transmog', icon: 'ri-shirt-line', label: '幻化方案' },
  { id: 'home_showcase', icon: 'ri-home-heart-line', label: '家宅分享' },
  { id: 'musician_midi', icon: 'ri-music-2-line', label: 'Musician MIDI' },
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await listRPDBWorks(filters)
    works.value = result.works || []
    total.value = result.total || 0
    if (total.value && (filters.page || 1) > totalPages.value) {
      filters.page = totalPages.value
      await load()
    }
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

async function loadHotWorks() {
  hotLoading.value = true
  try {
    const result = await listRPDBHotWorks({
      type: filters.type || undefined,
      limit: 3,
    })
    hotWorks.value = result.works || []
  } catch {
    hotWorks.value = []
  } finally {
    hotLoading.value = false
  }
}

function setType(type: ListRPDBWorksParams['type']) {
  filters.type = type
  filters.availability_status = ''
  filters.faction = ''
  filters.armor_type = ''
  filters.tag_id = undefined
  filters.page = 1
  void load()
  void loadHotWorks()
}

function changePage(delta: number) {
  filters.page = Math.min(totalPages.value, Math.max(1, (filters.page || 1) + delta))
  void load()
}

function applySearch() {
  filters.page = 1
  filters.page_size = pageSize
  void load()
}

function openWork(work?: RPDBWork) {
  if (work) void router.push(`/rpdb/${work.id}`)
}

function setViewMode(mode: ViewMode) {
  viewMode.value = mode
  window.localStorage.setItem(VIEW_MODE_KEY, mode)
}

function typeLabel(type?: RPDBWorkType) {
  return ({ item_showcase: '魔兽物品', transmog: '幻化方案', home_showcase: '家宅分享', musician_midi: 'Musician MIDI' } as const)[type || 'item_showcase']
}

function typeIcon(type?: RPDBWorkType) {
  return ({ item_showcase: 'ri-magic-line', transmog: 'ri-shirt-line', home_showcase: 'ri-home-heart-line', musician_midi: 'ri-music-2-line' } as const)[type || 'item_showcase']
}

function formatCount(value?: number) {
  return new Intl.NumberFormat('zh-CN').format(value || 0)
}

function workSummary(work?: RPDBWork | null) {
  if (!work) return '席位空缺，浏览作品后将按近 7 日热度上榜'
  const summary = work.summary?.trim()
  if (summary) return summary
  const effect = work.effect_description?.trim()
  if (effect) return effect
  const useCases = work.rp_use_cases?.trim()
  if (useCases) return useCases
  return '作者尚未填写作品摘要'
}

async function loadStyleTags() {
  try {
    const result = await getPresetTags('rpdb')
    styleTags.value = sortRPDBStyleTags((result.tags || []).filter(isRPDBStyleTag))
  } catch {
    styleTags.value = []
  }
}

onMounted(() => {
  void loadStyleTags()
  void load()
  void loadHotWorks()
})
</script>

<template>
  <div class="rpdb-shell">
    <main class="discovery-main">
      <nav class="content-nav" aria-label="RPDB 内容快捷入口">
        <div class="content-nav__channels">
          <button type="button" :class="{ active: !filters.type }" @click="setType('')"><i class="ri-compass-3-line"></i>推荐</button>
          <button type="button" :class="{ active: filters.type === 'item_showcase' }" @click="setType('item_showcase')"><i class="ri-magic-line"></i>魔兽物品</button>
          <button type="button" :class="{ active: filters.type === 'transmog' }" @click="setType('transmog')"><i class="ri-shirt-line"></i>幻化方案</button>
          <button type="button" :class="{ active: filters.type === 'home_showcase' }" @click="setType('home_showcase')"><i class="ri-home-heart-line"></i>家宅分享</button>
          <button type="button" :class="{ active: filters.type === 'musician_midi' }" @click="setType('musician_midi')"><i class="ri-music-2-line"></i>Musician MIDI</button>
        </div>
        <div class="content-nav__actions">
          <router-link to="/rpdb/my-uploads"><i class="ri-upload-cloud-2-line"></i>我的上传</router-link>
          <router-link to="/rpdb/lists"><i class="ri-star-line"></i>我的收藏</router-link>
          <router-link to="/rpdb/lists"><i class="ri-list-check-3"></i>收集清单</router-link>
          <router-link to="/rpdb/create"><i class="ri-draft-line"></i>发布内容</router-link>
        </div>
      </nav>

      <section class="featured-grid" aria-label="热度 Top3">
        <header class="hot-heading">
          <div>
            <span>近 7 日浏览榜</span>
            <h2>{{ hotTitle }}</h2>
          </div>
          <small v-if="hotLoading">刷新中…</small>
        </header>
        <div class="featured-grid__cards">
          <button
            v-for="(work, index) in hotSlots"
            :key="work?.id || `hot-empty-${index}`"
            type="button"
            class="featured-card"
            :class="{ empty: !work }"
            :disabled="!work"
            data-testid="rpdb-hot-card"
            @click="openWork(work || undefined)"
          >
            <span class="hot-rank">TOP {{ index + 1 }}</span>
            <img v-if="work?.cover_image" :src="resolveRPDBMediaURL(work.cover_image)" :alt="work.title" />
            <div v-else class="featured-placeholder"><i :class="typeIcon(work?.type)"></i></div>
            <div class="featured-shade"></div>
            <div class="featured-copy">
              <span>{{ work ? typeLabel(work.type) : '暂无数据' }}</span>
              <h2>{{ work?.title || `TOP ${index + 1} 空缺` }}</h2>
              <p>{{ workSummary(work) }}</p>
              <div class="featured-meta">
                <span v-if="work" class="featured-author">
                  <span class="featured-author__avatar">
                    <span>{{ (work.author_name || 'R').charAt(0).toUpperCase() }}</span>
                    <img
                      v-if="work.author_avatar"
                      :src="resolveRPDBMediaURL(work.author_avatar)"
                      :alt="`${work.author_name || '发布者'}的头像`"
                      @error="($event.currentTarget as HTMLImageElement).hidden = true"
                    >
                  </span>
                  <b>{{ work.author_name || '匿名贡献者' }}</b>
                </span>
                <b v-else>—</b>
                <div v-if="work" class="featured-metrics" aria-label="作品数据" data-testid="rpdb-featured-metrics">
                  <span title="近7日浏览"><i class="ri-fire-line"></i>{{ formatCount(work.view_count) }}</span>
                  <span title="点赞"><i class="ri-heart-3-line"></i>{{ formatCount(work.like_count) }}</span>
                  <span title="收藏"><i class="ri-bookmark-3-line"></i>{{ formatCount(work.favorite_count) }}</span>
                  <span title="加入清单"><i class="ri-list-check-3"></i>{{ formatCount(work.list_count) }}</span>
                </div>
              </div>
            </div>
          </button>
        </div>
      </section>

      <section class="discovery-toolbar">
        <RPDBFilterBar :model-value="filters" :style-tags="styleTags" @update:model-value="Object.assign(filters, { ...$event, page_size: pageSize })" @search="applySearch" />
        <nav class="channel-tabs" aria-label="内容频道">
          <button v-for="channel in channels" :key="channel.id || 'all'" type="button" :class="{ active: filters.type === channel.id }" @click="setType(channel.id)">
            <i :class="channel.icon"></i>{{ channel.label }}
          </button>
        </nav>
      </section>

      <section class="feed-heading">
        <div>
          <span>{{ total }} 件社区作品 · 每页最多 {{ pageSize }} 个</span>
          <h1>{{ resultTitle }}</h1>
        </div>
        <div class="feed-heading__tools">
          <span v-if="total" class="result-range">显示 {{ pageStart }}-{{ pageEnd }}</span>
          <div class="view-switcher" role="group" aria-label="作品列表布局">
            <button
              type="button"
              data-testid="rpdb-card-view"
              :class="{ active: viewMode === 'card' }"
              :aria-pressed="viewMode === 'card'"
              title="卡片模式"
              @click="setViewMode('card')"
            >
              <i class="ri-layout-grid-fill"></i>
              <span>卡片</span>
            </button>
            <button
              type="button"
              data-testid="rpdb-compact-view"
              :class="{ active: viewMode === 'compact' }"
              :aria-pressed="viewMode === 'compact'"
              title="横向紧凑模式"
              @click="setViewMode('compact')"
            >
              <i class="ri-list-check-2"></i>
              <span>紧凑</span>
            </button>
          </div>
        </div>
      </section>

      <div v-if="loading" class="state"><i class="ri-loader-4-line spin"></i><span>正在整理玩家作品</span></div>
      <div v-else-if="error" class="state error"><i class="ri-error-warning-line"></i><span>{{ error }}</span><button type="button" @click="load">重试</button></div>
      <div v-else-if="works.length" class="discovery-grid" :class="{ compact: viewMode === 'compact' }" data-testid="rpdb-discovery-results">
        <RPDBWorkCard v-for="work in works" :key="work.id" :work="work" :layout="viewMode" @open="openWork(work)" />
      </div>
      <div v-else class="empty-state">
        <i class="ri-gallery-upload-line"></i>
        <h2>这个频道还没有作品</h2>
        <p>上传真实效果、搭配方案或家宅展示，成为第一位贡献者。</p>
        <router-link to="/rpdb/create">发布第一个作品</router-link>
      </div>

      <footer v-if="totalPages > 1" class="pager" data-testid="rpdb-pagination">
        <button type="button" :disabled="filters.page === 1" @click="changePage(-1)"><i class="ri-arrow-left-line"></i></button>
        <span>第 {{ filters.page }} / {{ totalPages }} 页</span>
        <button type="button" :disabled="filters.page === totalPages" @click="changePage(1)"><i class="ri-arrow-right-line"></i></button>
      </footer>
    </main>
  </div>
</template>

<style scoped>
.rpdb-shell{min-height:calc(100vh - 48px);margin:-24px;background:var(--color-main-bg);color:var(--color-text-main)}
.discovery-main{min-width:0;padding:24px}.content-nav{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:16px;padding:10px;border:1px solid var(--color-border);border-radius:var(--radius-md);background:var(--color-panel-bg);box-shadow:var(--shadow-sm)}.content-nav__channels,.content-nav__actions{display:flex;align-items:center;gap:6px}.content-nav button,.content-nav a{display:inline-flex;min-height:38px;align-items:center;gap:7px;padding:0 12px;border:0;border-radius:var(--radius-sm);background:transparent;color:var(--color-text-secondary);font:600 13px/1 inherit;text-decoration:none}.content-nav button{cursor:pointer}.content-nav button:hover,.content-nav a:hover,.content-nav .active{background:var(--btn-secondary-bg);color:var(--btn-secondary-text)}.content-nav i{color:var(--icon-color);font-size:16px}
.featured-grid{display:flex;flex-direction:column;gap:12px}.hot-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:12px}.hot-heading span{display:block;color:var(--color-accent);font-size:11px;font-weight:800;letter-spacing:.04em}.hot-heading h2{margin:4px 0 0;color:var(--color-text-main);font:700 20px/1.25 system-ui,'Microsoft YaHei',sans-serif}.hot-heading small{color:var(--color-text-secondary);font-size:12px}.featured-grid__cards{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}.featured-card{position:relative;min-height:240px;overflow:hidden;padding:0;border:1px solid var(--color-border);border-radius:var(--radius-md);background:var(--color-card-bg);color:#fff;text-align:left;box-shadow:var(--shadow-sm);cursor:pointer}.featured-card:disabled{cursor:default}.hot-rank{position:absolute;top:12px;left:12px;z-index:2;padding:4px 8px;border-radius:999px;background:rgba(0,0,0,.45);color:#fff;font-size:11px;font-weight:800;letter-spacing:.04em;backdrop-filter:blur(6px)}.featured-card img{position:absolute;inset:0;width:100%;height:100%;object-fit:cover;transition:transform .35s ease}.featured-card:not(:disabled):hover img{transform:scale(1.025)}.featured-placeholder{position:absolute;inset:0;display:grid;place-items:center;background:linear-gradient(135deg,color-mix(in srgb,var(--color-card-bg) 78%,var(--color-accent)),var(--color-panel-bg));color:color-mix(in srgb,var(--color-accent) 62%,transparent)}.featured-placeholder i{font-size:52px}.featured-shade{position:absolute;inset:0;background:linear-gradient(180deg,rgba(44,24,16,.03) 20%,rgba(44,24,16,.88) 100%)}.featured-card.empty .featured-shade{background:linear-gradient(180deg,transparent 10%,color-mix(in srgb,var(--color-panel-bg) 92%,transparent) 100%)}.featured-copy{position:absolute;inset:auto 24px 20px;z-index:1}.featured-copy>span{color:var(--color-accent);font-size:12px;font-weight:700}.featured-copy h2{margin:7px 0 5px;font:700 24px/1.2 Georgia,'Microsoft YaHei',serif}.featured-copy p{overflow:hidden;margin:0 0 17px;color:rgba(255,255,255,.76);font-size:12px;white-space:nowrap;text-overflow:ellipsis}.featured-card.empty .featured-copy h2{color:var(--color-text-main)}.featured-card.empty .featured-copy p,.featured-card.empty .featured-meta{color:var(--color-text-secondary)}.featured-meta{display:flex;align-items:center;gap:12px;font-size:11px;color:rgba(255,255,255,.82)}.featured-meta b{min-width:0;overflow:hidden;margin-right:auto;text-overflow:ellipsis;white-space:nowrap}.featured-metrics{display:flex;flex:0 0 auto;align-items:center;gap:8px;font-variant-numeric:tabular-nums}.featured-metrics span{display:inline-flex;align-items:center;gap:3px}.featured-metrics i{color:var(--color-accent)}
.featured-card>img{position:absolute;inset:0;width:100%;height:100%;object-fit:cover;transition:transform .35s ease}.featured-card:not(:disabled):hover>img{transform:scale(1.025)}.featured-author{display:flex!important;min-width:0;align-items:center;gap:7px;margin-right:auto;color:inherit!important}.featured-author b{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.featured-author__avatar{position:relative;display:grid!important;width:24px;height:24px;flex:0 0 24px;overflow:hidden;place-items:center;border:1px solid rgba(255,255,255,.35);border-radius:50%;background:rgba(44,24,16,.45);color:#fff!important;font-size:9px!important}.featured-author__avatar img{position:absolute;inset:0;width:100%;height:100%;object-fit:cover}
.featured-card:not(:disabled):hover .featured-author__avatar img{transform:none}
.discovery-toolbar{margin-top:18px;padding:12px 0 0;background:var(--color-main-bg)}.channel-tabs{display:flex;gap:6px;margin-top:9px;padding-bottom:8px;border-bottom:1px solid var(--color-border)}.channel-tabs button{display:inline-flex;align-items:center;gap:6px;height:36px;padding:0 13px;border:0;border-radius:var(--radius-sm);background:transparent;color:var(--color-text-secondary);font-weight:700}.channel-tabs button:hover,.channel-tabs button.active{background:var(--btn-secondary-bg);color:var(--btn-secondary-text)}.feed-heading{display:flex;align-items:end;justify-content:space-between;gap:16px;padding:18px 2px 12px}.feed-heading>div:first-child>span,.result-range{color:var(--color-text-secondary);font-size:11px}.feed-heading h1{margin:4px 0 0;color:var(--color-text-main);font:700 22px/1.2 Georgia,'Microsoft YaHei',serif}.feed-heading__tools{display:flex;align-items:center;gap:10px}.view-switcher{display:inline-flex;padding:3px;border:1px solid var(--color-border);border-radius:10px;background:var(--color-panel-bg);box-shadow:var(--shadow-sm)}.view-switcher button{display:inline-flex;min-height:32px;align-items:center;gap:5px;padding:0 10px;border:0;border-radius:7px;background:transparent;color:var(--color-text-secondary);font:700 11px/1 inherit}.view-switcher button:hover{color:var(--color-text-main)}.view-switcher button.active{background:var(--color-accent);color:#fff;box-shadow:0 2px 8px color-mix(in srgb,var(--color-accent) 28%,transparent)}.view-switcher i{font-size:14px}.discovery-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:12px}.discovery-grid.compact{grid-template-columns:1fr;gap:8px}
.state,.empty-state{display:grid;min-height:300px;place-items:center;align-content:center;gap:10px;text-align:center;color:var(--color-text-secondary)}.state i,.empty-state>i{font-size:38px;color:var(--icon-color)}.state button{border:0;background:none;color:var(--link-color)}.empty-state h2{color:var(--color-text-main)}.empty-state h2,.empty-state p{margin:0}.empty-state a{display:inline-flex;margin-top:8px;padding:10px 14px;border-radius:var(--radius-sm);background:var(--btn-primary-bg);color:var(--btn-primary-text);text-decoration:none;font-weight:700}.empty-state a:hover{background:var(--btn-primary-hover);color:var(--btn-primary-text)}.pager{display:flex;align-items:center;justify-content:center;gap:12px;padding:24px}.pager button{width:36px;height:36px;border:1px solid var(--color-border);border-radius:var(--radius-sm);background:var(--color-panel-bg);color:var(--color-text-main)}.spin{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:1180px){.content-nav{align-items:flex-start;flex-direction:column}.featured-grid__cards{grid-template-columns:1fr 1fr}.featured-grid__cards .featured-card:last-child{display:none}.discovery-main{padding:18px}}
@media(max-width:820px){.content-nav{overflow:hidden}.content-nav__channels,.content-nav__actions{width:100%;overflow-x:auto}.featured-grid__cards{grid-template-columns:1fr}.featured-card{min-height:260px}.featured-grid__cards .featured-card:nth-child(n+2){display:none}.channel-tabs{overflow:auto}.discovery-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:520px){.discovery-main{padding:12px}.content-nav__actions{display:none}.featured-card{min-height:220px}.featured-copy{inset:auto 18px 16px}.featured-copy h2{font-size:21px}.feed-heading{align-items:flex-start}.result-range{display:none}.view-switcher button{padding:0 9px}.view-switcher button span{display:none}.discovery-grid{grid-template-columns:1fr}}
</style>
