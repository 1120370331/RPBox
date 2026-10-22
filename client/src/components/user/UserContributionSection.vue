<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  listUserContributions,
  type ContributionKind,
  type ContributionSort,
  type UserContribution,
} from '@/api/userContribution'
import { getImageUrl, resolveApiUrl } from '@/api/item'

const props = defineProps<{ userId: number; isOwnProfile: boolean }>()
const section = ref<HTMLElement | null>(null)
const entries = ref<UserContribution[]>([])
const total = ref(0)
const page = ref(1)
const columns = ref(3)
const pageSize = computed(() => columns.value * 2)
const kind = ref<ContributionKind | 'all'>('all')
const sort = ref<ContributionSort>('created_at')
const loading = ref(false)
const errorMessage = ref('')
const failedImages = ref(new Set<string>())
let requestToken = 0
let resizeObserver: ResizeObserver | undefined

const kinds = [
  { value: 'all', label: '全部' },
  { value: 'post', label: '社区帖子' },
  { value: 'rpdb', label: 'RP 数据库' },
  { value: 'item', label: 'TRP3 道具' },
] as const
const metadata = {
  post: { label: '社区帖子', icon: 'ri-article-line', route: '/community/post/', create: '/community/create', manage: '/community/my-posts' },
  rpdb: { label: 'RP 数据库', icon: 'ri-archive-drawer-line', route: '/rpdb/', create: '/rpdb/create', manage: '/rpdb/my-uploads' },
  item: { label: 'TRP3 道具', icon: 'ri-box-3-line', route: '/market/', create: '/market/upload', manage: '/market/my-items' },
}
const selectedMetadata = computed(() => metadata[kind.value === 'all' ? 'post' : kind.value])
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const visiblePages = computed(() => {
  const start = Math.max(1, Math.min(page.value - 2, totalPages.value - 4))
  return Array.from({ length: Math.min(5, totalPages.value) }, (_, index) => start + index)
})

watch(() => [props.userId, props.isOwnProfile, kind.value, sort.value, columns.value] as const, () => {
  page.value = 1
  void loadEntries()
}, { immediate: true })

function measure() {
  if (!section.value?.clientWidth) return
  columns.value = section.value.clientWidth < 520 ? 1 : section.value.clientWidth < 800 ? 2 : 3
}

onMounted(() => {
  measure()
  if (section.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(measure)
    resizeObserver.observe(section.value)
  }
  window.addEventListener('resize', measure)
})
onBeforeUnmount(() => {
  requestToken++
  resizeObserver?.disconnect()
  window.removeEventListener('resize', measure)
})

async function loadEntries() {
  const token = ++requestToken
  entries.value = []
  errorMessage.value = ''
  failedImages.value = new Set()
  if (!props.userId) {
    total.value = 0
    loading.value = false
    return
  }
  loading.value = true
  try {
    const response = await listUserContributions(props.userId, {
      page: page.value, page_size: pageSize.value, kind: kind.value, sort: sort.value,
    })
    if (token !== requestToken) return
    entries.value = response.entries || []
    total.value = response.total || 0
    if (page.value > totalPages.value) {
      page.value = totalPages.value
      void loadEntries()
    }
  } catch (error: unknown) {
    if (token !== requestToken) return
    errorMessage.value = error instanceof Error ? error.message : '加载帖子失败，请重试。'
  } finally {
    if (token === requestToken) loading.value = false
  }
}

function changePage(next: number) {
  if (loading.value || next < 1 || next > totalPages.value || next === page.value) return
  page.value = next
  void loadEntries()
}

function coverURL(entry: UserContribution) {
  if (failedImages.value.has(`${entry.kind}-${entry.id}`)) return ''
  if (entry.kind === 'rpdb') return resolveApiUrl(entry.cover_url)
  return entry.has_cover ? getImageUrl(entry.kind === 'post' ? 'post-cover' : 'item-preview', entry.id, {
    w: 520, q: 80, v: entry.cover_updated_at || entry.updated_at,
  }) : ''
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function formatCount(value: number) {
  return new Intl.NumberFormat('zh-CN', { notation: 'compact', maximumFractionDigits: 1 }).format(value || 0)
}
</script>

<template>
  <section ref="section" class="user-contributions" aria-labelledby="user-contributions-title">
    <header class="contributions-header">
      <div>
        <div class="contributions-heading">
          <h2 id="user-contributions-title">{{ isOwnProfile ? '我的帖子' : 'TA 的帖子' }}</h2>
          <span v-if="!loading && !errorMessage" class="contributions-count">{{ total }} 篇</span>
        </div>
        <p>社区帖子、RP 数据库与 TRP3 道具</p>
      </div>
      <RouterLink v-if="isOwnProfile" class="contributions-manage" :to="selectedMetadata.manage">
        管理{{ selectedMetadata.label }}<i class="ri-arrow-right-up-line" aria-hidden="true"></i>
      </RouterLink>
    </header>

    <div class="contributions-toolbar">
      <div class="contributions-filters" role="group" aria-label="内容类型">
        <button v-for="filter in kinds" :key="filter.value" type="button" :class="{ active: kind === filter.value }" :aria-pressed="kind === filter.value" @click="kind = filter.value">{{ filter.label }}</button>
      </div>
      <div class="contributions-sort" role="group" aria-label="帖子排序">
        <button type="button" :class="{ active: sort === 'created_at' }" :aria-pressed="sort === 'created_at'" @click="sort = 'created_at'">最新发布</button>
        <button type="button" :class="{ active: sort === 'popular' }" :aria-pressed="sort === 'popular'" title="按点赞数排序，相同时按浏览或下载量排序" @click="sort = 'popular'">热度优先</button>
      </div>
    </div>

    <div class="contributions-body" :style="{ '--columns': columns }" :aria-busy="loading">
      <div v-if="loading" class="contributions-grid" role="status" aria-label="正在加载帖子">
        <div v-for="index in pageSize" :key="index" class="contribution-skeleton"><div></div><span></span><span></span></div>
      </div>
      <div v-else-if="errorMessage" class="contributions-state" role="alert">
        <i class="ri-cloud-off-line" aria-hidden="true"></i>
        <strong>帖子加载失败</strong><p>{{ errorMessage }}</p>
        <button type="button" @click="loadEntries">重新加载</button>
      </div>
      <div v-else-if="entries.length === 0" class="contributions-state">
        <i :class="selectedMetadata.icon" aria-hidden="true"></i>
        <strong>{{ isOwnProfile ? '还没有发布的内容' : '这位玩家还没有公开内容' }}</strong>
        <p>{{ kind === 'all' ? '公开发布的帖子、数据库作品和道具会一起展示在这里。' : `这里会展示公开发布的${selectedMetadata.label}。` }}</p>
        <RouterLink v-if="isOwnProfile" :to="selectedMetadata.create">发布{{ selectedMetadata.label }}<i class="ri-arrow-right-line" aria-hidden="true"></i></RouterLink>
      </div>
      <div v-else class="contributions-grid">
        <RouterLink v-for="entry in entries" :key="`${entry.kind}-${entry.id}`" class="contribution-card" :to="`${metadata[entry.kind].route}${entry.id}`">
          <div class="contribution-card__media">
            <img v-if="coverURL(entry)" :src="coverURL(entry)" alt="" loading="lazy" @error="failedImages.add(`${entry.kind}-${entry.id}`)" />
            <div v-else class="contribution-card__placeholder" :class="`contribution-card__placeholder--${entry.kind}`"><i :class="metadata[entry.kind].icon" aria-hidden="true"></i></div>
            <span class="contribution-card__kind"><i :class="metadata[entry.kind].icon" aria-hidden="true"></i>{{ metadata[entry.kind].label }}</span>
          </div>
          <div class="contribution-card__body">
            <h3>{{ entry.title }}</h3>
            <div class="contribution-card__meta">
              <time :datetime="entry.created_at">{{ formatDate(entry.created_at) }}</time>
              <span :title="`${entry.like_count} 次点赞`"><i class="ri-heart-3-line" aria-hidden="true"></i>{{ formatCount(entry.like_count) }}</span>
              <span :title="`${entry.view_count} 次${entry.kind === 'item' ? '下载' : '浏览'}`"><i :class="entry.kind === 'item' ? 'ri-download-2-line' : 'ri-eye-line'" aria-hidden="true"></i>{{ formatCount(entry.view_count) }}</span>
            </div>
          </div>
        </RouterLink>
      </div>
    </div>

    <footer v-if="total > 0 && !errorMessage" class="contributions-footer">
      <span aria-live="polite">第 {{ page }} / {{ totalPages }} 页</span>
      <nav class="contributions-pagination" aria-label="帖子分页">
        <button type="button" :disabled="loading || page === 1" aria-label="上一页帖子" @click="changePage(page - 1)"><i class="ri-arrow-left-s-line" aria-hidden="true"></i></button>
        <button v-if="visiblePages[0]! > 1" type="button" :disabled="loading" aria-label="第 1 页帖子" @click="changePage(1)">1</button>
        <span v-if="visiblePages[0]! > 2">…</span>
        <button v-for="index in visiblePages" :key="index" type="button" :disabled="loading" :class="{ active: index === page }" :aria-label="`第 ${index} 页帖子`" :aria-current="index === page ? 'page' : undefined" @click="changePage(index)">{{ index }}</button>
        <span v-if="visiblePages[visiblePages.length - 1]! < totalPages - 1">…</span>
        <button v-if="visiblePages[visiblePages.length - 1]! < totalPages" type="button" :disabled="loading" :aria-label="`第 ${totalPages} 页帖子`" @click="changePage(totalPages)">{{ totalPages }}</button>
        <button type="button" :disabled="loading || page === totalPages" aria-label="下一页帖子" @click="changePage(page + 1)"><i class="ri-arrow-right-s-line" aria-hidden="true"></i></button>
      </nav>
    </footer>
  </section>
</template>

<style scoped>
.user-contributions { grid-column: 1 / -1; min-width: 0; padding: 24px; border: 1px solid var(--color-border); border-radius: 22px; background: var(--color-panel-bg); box-shadow: var(--shadow-sm); }
.contributions-header, .contributions-heading { display: flex; align-items: center; gap: 12px; }
.contributions-header { justify-content: space-between; gap: 20px; }
.contributions-heading h2 { margin: 0; color: var(--color-text-main); font-size: 20px; font-weight: 650; letter-spacing: -.02em; }
.contributions-count { color: var(--color-text-secondary); font-size: 12px; font-variant-numeric: tabular-nums; }
.contributions-header p { margin: 7px 0 0; color: var(--color-text-secondary); font-size: 12px; }
.contributions-manage { display: inline-flex; flex-shrink: 0; align-items: center; gap: 5px; color: var(--color-text-secondary); font-size: 12px; text-decoration: none; }
.contributions-manage:hover { color: var(--color-text-main); }
.contributions-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin: 20px 0; }
.contributions-filters, .contributions-sort { display: flex; align-items: center; gap: 3px; }
.contributions-filters { flex-wrap: wrap; }
.contributions-filters button { min-height: 34px; padding: 0 13px; border: 0; border-radius: 999px; background: transparent; color: var(--color-text-secondary); font-size: 12px; cursor: pointer; }
.contributions-filters button.active { background: var(--btn-primary-bg); color: var(--btn-primary-text); font-weight: 600; }
.contributions-filters button:hover:not(.active) { background: var(--color-card-bg-hover); }
.contributions-sort { flex-shrink: 0; padding: 3px; border-radius: 10px; background: var(--color-card-bg-hover); }
.contributions-sort button { min-height: 28px; padding: 0 10px; border: 0; border-radius: 7px; background: transparent; color: var(--color-text-secondary); font-size: 11px; cursor: pointer; transition: background 160ms ease, box-shadow 160ms ease; }
.contributions-sort button.active { background: var(--color-panel-bg); color: var(--color-text-main); box-shadow: var(--shadow-sm); font-weight: 600; }
.contributions-body { min-height: 468px; }
.contributions-grid { display: grid; grid-template-columns: repeat(var(--columns), minmax(0, 1fr)); gap: 16px; }
.contribution-card { display: block; min-width: 0; overflow: hidden; border: 1px solid var(--color-border); border-radius: 16px; background: var(--color-card-bg); color: var(--color-text-main); text-decoration: none; transition: border-color 160ms ease, box-shadow 160ms ease; }
.contribution-card:hover { border-color: var(--color-border-hover); box-shadow: var(--shadow-sm); }
.contribution-card__media { position: relative; height: 130px; overflow: hidden; background: var(--color-card-bg-hover); }
.contribution-card__media img { width: 100%; height: 100%; object-fit: cover; transition: transform 250ms ease; }
.contribution-card:hover img { transform: scale(1.035); }
.contribution-card__placeholder { display: grid; width: 100%; height: 100%; place-items: center; color: var(--color-text-secondary); background: linear-gradient(135deg, var(--color-card-bg), var(--color-card-bg-hover)); }
.contribution-card__placeholder > i { opacity: .35; font-size: 42px; }
.contribution-card__kind { position: absolute; top: 10px; left: 10px; display: inline-flex; align-items: center; gap: 4px; padding: 4px 8px; border: 1px solid var(--color-border); border-radius: 7px; background: color-mix(in srgb, var(--color-panel-bg) 94%, transparent); color: var(--color-text-main); backdrop-filter: blur(8px); font-size: 10px; }
.contribution-card__body { padding: 12px 14px; }
.contribution-card__body h3 { display: -webkit-box; height: 40px; margin: 0 0 10px; overflow: hidden; font-size: 14px; font-weight: 600; line-height: 20px; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.contribution-card__meta { display: flex; align-items: center; gap: 10px; color: var(--color-text-secondary); font-size: 10px; font-variant-numeric: tabular-nums; }
.contribution-card__meta time { margin-right: auto; }
.contribution-card__meta > span { display: inline-flex; align-items: center; gap: 3px; }
.contributions-state { display: flex; min-height: 468px; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 24px; box-sizing: border-box; text-align: center; }
.contributions-state > i { margin-bottom: 4px; color: var(--color-text-secondary); font-size: 36px; }
.contributions-state strong { color: var(--color-text-main); font-size: 15px; }
.contributions-state p { max-width: 360px; margin: 0; color: var(--color-text-secondary); font-size: 12px; line-height: 1.8; }
.contributions-state > a, .contributions-state > button { display: inline-flex; align-items: center; gap: 6px; min-height: 36px; margin-top: 6px; padding: 0 16px; border: 1px solid var(--color-border); border-radius: 999px; background: var(--color-card-bg); color: var(--color-text-main); font-size: 12px; text-decoration: none; cursor: pointer; }
.contribution-skeleton { height: 226px; overflow: hidden; border: 1px solid var(--color-border); border-radius: 16px; animation: contribution-pulse 1.4s ease-in-out infinite alternate; }
.contribution-skeleton > div { height: 130px; background: var(--color-card-bg-hover); }
.contribution-skeleton > span { display: block; width: 65%; height: 12px; margin: 16px 14px 0; border-radius: 4px; background: var(--color-card-bg-hover); }
.contribution-skeleton > span:last-child { width: 40%; height: 8px; }
.contributions-footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 22px; padding-top: 18px; border-top: 1px solid var(--color-border); }
.contributions-footer > span { color: var(--color-text-secondary); font-size: 11px; font-variant-numeric: tabular-nums; }
.contributions-pagination { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; }
.contributions-pagination button { display: grid; width: 32px; height: 32px; place-items: center; padding: 0; border: 0; border-radius: 50%; background: transparent; color: var(--color-text-secondary); font-size: 12px; cursor: pointer; }
.contributions-pagination button:hover:not(:disabled) { background: var(--color-card-bg-hover); }
.contributions-pagination button.active { background: var(--btn-primary-bg); color: var(--btn-primary-text); font-weight: 600; }
.contributions-pagination button:disabled { opacity: .35; cursor: default; }
.user-contributions :is(a, button):focus-visible { outline: 3px solid var(--color-accent); outline-offset: 3px; }
@keyframes contribution-pulse { to { opacity: .45; } }
@media (max-width: 760px) { .user-contributions { padding: 18px 14px; } .contributions-toolbar { align-items: flex-start; flex-direction: column; gap: 12px; } .contributions-filters button { padding: 0 10px; } .contributions-manage { font-size: 11px; } .contributions-footer { align-items: flex-start; flex-direction: column; } }
@media (prefers-reduced-motion: reduce) { .user-contributions * { transition: none; animation: none; } }
</style>
