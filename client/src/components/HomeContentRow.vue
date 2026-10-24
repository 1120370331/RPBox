<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getImageUrl, resolveApiUrl } from '@/api/item'
import { homeContentKey, type HomeContent } from '@/api/home'

defineProps<{
  featured?: boolean
  entries: HomeContent[]
  loading: boolean
  error: string
  hasMore?: boolean
}>()
defineEmits<{ refresh: []; retry: [] }>()
const { t, locale } = useI18n()
const failedImages = ref(new Set<string>())
const contentTypes = {
  post: { icon: 'ri-article-line', route: '/community/post/' },
  rpdb: { icon: 'ri-archive-drawer-line', route: '/rpdb/' },
  item: { icon: 'ri-sword-line', route: '/market/' },
}

function cover(entry: HomeContent) {
  if (!entry.has_cover || failedImages.value.has(homeContentKey(entry))) return ''
  if (entry.kind === 'rpdb') return resolveApiUrl(entry.cover_url)
  return getImageUrl(entry.kind === 'post' ? 'post-cover' : 'item-preview', entry.id, {
    w: 600, q: 80, v: entry.cover_updated_at || entry.updated_at,
  })
}

function metrics(entry: HomeContent) {
  const values = [
    entry.kind === 'item'
      ? { label: t('home.feed.metrics.downloads'), icon: 'ri-download-2-line', value: entry.downloads }
      : { label: t('home.feed.metrics.views'), icon: 'ri-eye-line', value: entry.view_count },
    { label: t('home.feed.metrics.likes'), icon: 'ri-heart-3-line', value: entry.like_count },
    { label: t('home.feed.metrics.favorites'), icon: 'ri-bookmark-3-line', value: entry.favorite_count },
  ]
  if (entry.kind === 'rpdb') {
    values.push({ label: t('home.feed.metrics.lists'), icon: 'ri-list-check-3', value: entry.list_count })
  } else if (entry.kind === 'post') {
    values.push({ label: t('home.feed.metrics.comments'), icon: 'ri-chat-3-line', value: entry.comment_count })
  }
  return values
}

function formatCount(value?: number) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
}
</script>

<template>
  <section class="home-content-row" :aria-label="t(featured ? 'home.feed.featured' : 'home.feed.recommended')" :aria-busy="loading">
    <header class="feed-heading">
      <div>
        <h2><i :class="featured ? 'ri-fire-line' : 'ri-sparkling-2-line'" aria-hidden="true"></i>{{ t(featured ? 'home.feed.featured' : 'home.feed.recommended') }}</h2>
        <p>{{ t(featured ? 'home.feed.featuredHint' : 'home.feed.recommendedHint') }}</p>
      </div>
      <button v-if="!featured" type="button" class="feed-refresh" :disabled="loading" :title="!hasMore && entries.length ? t('home.feed.noMore') : undefined" @click="$emit('refresh')">
        <i class="ri-refresh-line" :class="{ spinning: loading }" aria-hidden="true"></i>{{ t('home.feed.refresh') }}
      </button>
    </header>
    <p v-if="error" class="feed-error" role="alert">{{ error }} <button type="button" :disabled="loading" @click="$emit('retry')">{{ t('home.feed.retry') }}</button></p>
    <div v-if="loading && entries.length === 0" class="feed-grid" role="status" :aria-label="t('home.feed.loading')">
      <div v-for="index in 3" :key="index" class="feed-skeleton"><div></div><span></span><span></span></div>
    </div>
    <p v-else-if="entries.length === 0 && !error" class="feed-empty">{{ t('home.feed.empty') }}</p>
    <div v-else class="feed-grid">
      <RouterLink v-for="(entry, index) in entries" :key="homeContentKey(entry)" class="work-card feed-card" :to="`${contentTypes[entry.kind].route}${entry.id}`">
        <div class="work-card__media" :class="{ empty: !cover(entry) }">
          <img v-if="cover(entry)" :src="cover(entry)" alt="" loading="lazy" @error="failedImages.add(homeContentKey(entry))" />
          <div v-else class="work-card__placeholder"><i :class="contentTypes[entry.kind].icon" aria-hidden="true"></i></div>
          <div class="work-card__shade"></div>
          <span class="work-card__type"><i :class="contentTypes[entry.kind].icon" aria-hidden="true"></i>{{ t(`home.feed.kinds.${entry.kind}`) }}</span>
          <span v-if="featured" class="work-card__media-count" :aria-label="t('home.feed.rank', { rank: index + 1 })"><i class="ri-fire-line" aria-hidden="true"></i>{{ index + 1 }}</span>
          <div class="work-card__headline"><h3>{{ entry.title }}</h3></div>
        </div>
        <div class="work-card__body">
          <div class="work-card__details">
            <div class="work-card__author">
              <span class="author-mark">{{ entry.author_name?.charAt(0) || '?' }}</span>
              <b>{{ entry.author_name }}</b>
            </div>
          </div>
          <div class="work-card__metrics feed-card__metrics" :style="{ '--metrics-count': metrics(entry).length }" :aria-label="t('home.feed.metrics.label')">
            <span v-for="metric in metrics(entry)" :key="metric.label" :title="metric.label" :aria-label="`${metric.label} ${formatCount(metric.value)}`">
              <i :class="metric.icon" aria-hidden="true"></i><b>{{ formatCount(metric.value) }}</b>
            </span>
          </div>
        </div>
      </RouterLink>
    </div>
  </section>
</template>

<style scoped src="./rpdb/RPDBWorkCard.css"></style>

<style scoped>
.home-content-row { min-width: 0; }
.feed-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.feed-heading h2 { display: flex; align-items: center; gap: 9px; margin: 0; font-size: 19px; font-weight: 700; color: var(--color-text-main); }
.feed-heading h2 i { color: var(--color-accent); }
.feed-heading p { margin: 7px 0 0; color: var(--color-text-secondary); font-size: 12px; line-height: 1.6; }
.feed-refresh, .feed-error button { display: inline-flex; align-items: center; justify-content: center; gap: 7px; flex-shrink: 0; padding: 9px 13px; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-panel-bg); color: var(--color-text-main); font: inherit; font-size: 13px; cursor: pointer; }
.feed-refresh:hover:not(:disabled), .feed-error button:hover { border-color: var(--color-accent); }
.feed-refresh:disabled { opacity: 0.5; cursor: default; }
.feed-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
.feed-card { text-decoration: none; }
.feed-card:focus-visible, button:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 3px; }
.feed-card__metrics { grid-template-columns: repeat(var(--metrics-count, 4), minmax(0, 1fr)); }
.feed-empty, .feed-error { margin: 0; padding: 24px; border: 1px dashed var(--color-border); border-radius: 12px; color: var(--color-text-secondary); font-size: 13px; line-height: 1.7; }
.feed-error { margin-bottom: 12px; }
.feed-error button { margin-left: 8px; padding: 4px 8px; }
.feed-skeleton { overflow: hidden; min-height: 250px; background: var(--color-card-bg); border: 1px solid var(--color-border); border-radius: var(--radius-md); }
.feed-skeleton div { height: 180px; background: var(--color-panel-bg); }
.feed-skeleton span { display: block; height: 12px; margin: 20px 16px; border-radius: 4px; background: var(--color-card-bg); }
.feed-skeleton span:last-child { width: 40%; }
.spinning { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 680px) { .feed-grid { grid-template-columns: 1fr; } .feed-heading { align-items: flex-start; } }
@media (prefers-reduced-motion: reduce) { .feed-card { transition: none; } .feed-card:hover { transform: none; } .spinning { animation: none; } }
</style>
