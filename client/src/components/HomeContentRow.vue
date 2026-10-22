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
const { t } = useI18n()
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
      <RouterLink v-for="(entry, index) in entries" :key="homeContentKey(entry)" class="feed-card" :to="`${contentTypes[entry.kind].route}${entry.id}`">
        <div class="feed-card__media">
          <img v-if="cover(entry)" :src="cover(entry)" alt="" loading="lazy" @error="failedImages.add(homeContentKey(entry))" />
          <div v-else class="feed-card__placeholder"><i :class="contentTypes[entry.kind].icon" aria-hidden="true"></i></div>
          <span v-if="featured" class="feed-card__rank" :aria-label="t('home.feed.rank', { rank: index + 1 })">{{ index + 1 }}</span>
          <span class="feed-card__kind">{{ t(`home.feed.kinds.${entry.kind}`) }}</span>
        </div>
        <div class="feed-card__body">
          <h3>{{ entry.title }}</h3>
          <span class="feed-card__author">{{ entry.author_name }}</span>
          <div class="feed-card__footer">
            <span v-if="featured"><i class="ri-fire-line" aria-hidden="true"></i>{{ t('home.feed.weeklyScore', { score: entry.weekly_score }) }}</span>
            <span v-else>{{ t(entry.reason === 'interest' ? 'home.feed.interestReason' : 'home.feed.popularReason') }}</span>
            <i class="ri-arrow-right-line" aria-hidden="true"></i>
          </div>
        </div>
      </RouterLink>
    </div>
  </section>
</template>

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
.feed-card { min-width: 0; overflow: hidden; border: 1px solid var(--color-border); border-radius: 14px; background: var(--color-panel-bg); color: var(--color-text-main); text-decoration: none; box-shadow: var(--shadow-sm); transition: transform 0.18s ease, border-color 0.18s ease; }
.feed-card:hover { transform: translateY(-3px); border-color: var(--color-accent); }
.feed-card:focus-visible, button:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 3px; }
.feed-card__media { position: relative; height: 144px; overflow: hidden; background: var(--icon-bg); }
.feed-card__media img { width: 100%; height: 100%; object-fit: cover; }
.feed-card__placeholder { display: grid; place-items: center; width: 100%; height: 100%; color: var(--icon-color); background: linear-gradient(140deg, var(--icon-bg), var(--color-card-bg)); }
.feed-card__placeholder > i { font-size: 40px; opacity: 0.7; }
.feed-card__rank { position: absolute; left: 12px; top: 12px; display: grid; place-items: center; width: 30px; height: 34px; background: var(--btn-primary-bg); color: var(--btn-primary-text); border-radius: 4px 4px 10px 10px; font-family: 'Consolas', monospace; font-weight: 700; }
.feed-card__kind { position: absolute; right: 10px; bottom: 10px; padding: 4px 8px; border: 1px solid var(--color-border); border-radius: 6px; background: var(--color-panel-bg); color: var(--color-text-main); font-size: 11px; }
.feed-card__body { display: flex; flex-direction: column; gap: 10px; padding: 16px; }
.feed-card h3 { min-height: 44px; margin: 0; font-size: 15px; line-height: 22px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; overflow-wrap: anywhere; }
.feed-card__author { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--color-text-secondary); font-size: 12px; }
.feed-card__footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding-top: 12px; border-top: 1px solid var(--color-border); color: var(--color-text-secondary); font-size: 11px; }
.feed-card__footer span { display: inline-flex; align-items: center; gap: 5px; }
.feed-card__footer > i { color: var(--color-accent); }
.feed-empty, .feed-error { margin: 0; padding: 24px; border: 1px dashed var(--color-border); border-radius: 12px; color: var(--color-text-secondary); font-size: 13px; line-height: 1.7; }
.feed-error { margin-bottom: 12px; }
.feed-error button { margin-left: 8px; padding: 4px 8px; }
.feed-skeleton { overflow: hidden; min-height: 265px; background: var(--color-panel-bg); border: 1px solid var(--color-border); border-radius: 14px; }
.feed-skeleton div { height: 144px; background: var(--color-card-bg); }
.feed-skeleton span { display: block; height: 12px; margin: 20px 16px; border-radius: 4px; background: var(--color-card-bg); }
.feed-skeleton span:last-child { width: 40%; }
.spinning { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 680px) { .feed-grid { grid-template-columns: 1fr; } .feed-card__media { height: 180px; } .feed-heading { align-items: flex-start; } }
@media (prefers-reduced-motion: reduce) { .feed-card { transition: none; } .feed-card:hover { transform: none; } .spinning { animation: none; } }
</style>
