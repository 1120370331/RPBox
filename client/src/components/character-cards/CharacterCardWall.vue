<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  listMyCharacterCards,
  listUserCharacterCards,
  type CharacterCardSummary,
} from '@/api/characterCard'
import { getCharacterCardDisplayName } from '@/utils/characterCardDraft'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import { buildNameStyle } from '@/utils/userNameStyle'
import CharacterCardPortrait from './CharacterCardPortrait.vue'

const props = defineProps<{
  userId: number
  isOwnProfile: boolean
}>()
const { t } = useI18n()

const cards = ref<CharacterCardSummary[]>([])
const loading = ref(false)
const errorMessage = ref('')
let requestToken = 0
const viewport = ref<HTMLElement | null>(null)
const columns = ref(4)
const page = ref(0)
let resizeObserver: ResizeObserver | undefined
const entries = computed(() => props.isOwnProfile ? [...cards.value, null] : cards.value)
const pages = computed(() => Array.from(
  { length: Math.ceil(entries.value.length / columns.value) },
  (_, index) => entries.value.slice(index * columns.value, (index + 1) * columns.value),
))

function measure() {
  if (!viewport.value) return
  const next = Math.max(1, Math.floor((viewport.value.clientWidth + 16) / 216))
  if (columns.value !== next) {
    columns.value = next
    page.value = 0
    viewport.value.scrollLeft = 0
  }
}

watch(viewport, (element) => {
  resizeObserver?.disconnect()
  if (element) {
    measure()
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(measure)
      resizeObserver.observe(element)
    }
  }
})
onMounted(() => window.addEventListener('resize', measure))
onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  window.removeEventListener('resize', measure)
  requestToken++
})

function goToPage(index: number) {
  if (!viewport.value) return
  page.value = Math.max(0, Math.min(pages.value.length - 1, index))
  viewport.value.scrollTo({
    left: page.value * viewport.value.clientWidth,
    behavior: window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth',
  })
}

function syncPage() {
  if (!viewport.value?.clientWidth) return
  page.value = Math.round(viewport.value.scrollLeft / viewport.value.clientWidth)
}

const shouldRender = computed(() => (
  props.isOwnProfile || loading.value || Boolean(errorMessage.value) || cards.value.length > 0
))

watch(
  () => [props.userId, props.isOwnProfile] as const,
  () => void loadCards(),
  { immediate: true },
)

async function loadCards() {
  const token = ++requestToken
  loading.value = true
  page.value = 0
  errorMessage.value = ''
  try {
    const response = props.isOwnProfile
      ? await listMyCharacterCards()
      : await listUserCharacterCards(props.userId)
    if (token !== requestToken) return
    cards.value = response.character_cards || []
  } catch (error: unknown) {
    if (token !== requestToken) return
    cards.value = []
    errorMessage.value = error instanceof Error ? error.message : t('characterCards.wall.loadFailed')
  } finally {
    if (token === requestToken) {
      loading.value = false
      await nextTick()
      measure()
    }
  }
}

function displayName(card: CharacterCardSummary) {
  return getCharacterCardDisplayName(card)
}

function secondaryLine(card: CharacterCardSummary) {
  return card.title || card.full_title || [card.race, card.class].filter(Boolean).join(' · ') || t('characterCards.wall.noIdentity')
}

function displayNameColor(card: CharacterCardSummary) {
  return getCharacterCardDisplayColor(card)
}

function statusLabel(card: CharacterCardSummary) {
  if (card.status === 'draft') return t('characterCards.common.status.draft')
  if (card.visibility === 'private') return t('characterCards.common.status.private')
  if (card.review_status === 'pending') return t('characterCards.common.status.pending')
  if (card.review_status === 'rejected') return t('characterCards.common.status.rejected')
  return ''
}

function statusIcon(card: CharacterCardSummary) {
  if (card.review_status === 'pending' && card.status === 'published' && card.visibility === 'public') return 'ri-time-line'
  if (card.review_status === 'rejected' && card.status === 'published' && card.visibility === 'public') return 'ri-close-circle-line'
  return card.status === 'draft' ? 'ri-draft-line' : 'ri-lock-line'
}
</script>

<template>
  <section v-if="shouldRender" class="character-wall" aria-labelledby="character-wall-title">
    <header class="character-wall__header">
      <div class="character-wall__heading">
        <h2 id="character-wall-title">{{ t('characterCards.wall.title') }}</h2>
        <span v-if="cards.length" class="character-wall__count">{{ cards.length }}</span>
      </div>
      <RouterLink v-if="isOwnProfile && cards.length" class="character-wall__new" to="/character-cards/new">
        <i class="ri-add-line" aria-hidden="true"></i>
        {{ t('characterCards.wall.create') }}
      </RouterLink>
    </header>

    <div v-if="loading" class="character-wall__state" role="status">
      <i class="ri-loader-4-line character-wall__spinner" aria-hidden="true"></i>
      <span>{{ t('characterCards.common.loading') }}</span>
    </div>

    <div v-else-if="errorMessage" class="character-wall__state character-wall__state--error" role="alert">
      <i class="ri-file-warning-line" aria-hidden="true"></i>
      <div>
        <strong>{{ t('characterCards.wall.errorTitle') }}</strong>
        <span>{{ errorMessage }}</span>
      </div>
      <button type="button" @click="loadCards">{{ t('characterCards.common.reload') }}</button>
    </div>

    <div v-else-if="isOwnProfile && cards.length === 0" class="character-wall__empty">
      <div class="character-wall__empty-mark" aria-hidden="true">
        <i class="ri-compass-3-line"></i>
        <span class="character-wall__empty-plus"><i class="ri-add-line"></i></span>
      </div>
      <div class="character-wall__empty-copy">
        <span>{{ t('characterCards.wall.emptyKicker') }}</span>
        <h3>{{ t('characterCards.wall.emptyTitle') }}</h3>
        <p>{{ t('characterCards.wall.emptyBody') }}</p>
      </div>
      <RouterLink class="character-wall__primary" to="/character-cards/new">
        {{ t('characterCards.wall.create') }}
        <i class="ri-arrow-right-line" aria-hidden="true"></i>
      </RouterLink>
    </div>

    <template v-else>
      <div class="character-wall__carousel">
        <button class="character-wall__arrow character-wall__arrow--prev" type="button" :disabled="page === 0" :aria-label="t('characterCards.wall.previous')" aria-controls="character-wall-viewport" @click="goToPage(page - 1)">
          <i class="ri-arrow-left-s-line" aria-hidden="true"></i>
        </button>
        <div id="character-wall-viewport" ref="viewport" class="character-wall__viewport" role="region" :aria-label="t('characterCards.wall.title')" tabindex="0" @scroll.passive="syncPage" @keydown.left.prevent="goToPage(page - 1)" @keydown.right.prevent="goToPage(page + 1)">
          <div v-for="(group, pageIndex) in pages" :key="pageIndex" class="character-wall__grid" :style="{ '--columns': columns }" :inert="pageIndex !== page || undefined">
            <template v-for="card in group" :key="card?.id || 'add'">
              <RouterLink
                v-if="card"
                class="portrait-card"
                :to="`/character-cards/${card.id}`"
                :aria-label="t('characterCards.wall.viewAria', { name: displayName(card) })"
              >
                <span class="portrait-card__frame">
                  <CharacterCardPortrait
                    v-if="card.portrait_image_url"
                    class="portrait-card__image"
                    :card="card"
                    :alt="t('characterCards.common.portraitAlt', { name: displayName(card) })"
                    :width="560"
                    :quality="86"
                  />
                  <span v-else class="portrait-card__placeholder" aria-hidden="true">
                    <i class="ri-user-star-line"></i>
                    <small>{{ t('characterCards.wall.portraitPending') }}</small>
                  </span>
                  <span class="portrait-card__shade"></span>
                  <span v-if="isOwnProfile && statusLabel(card)" class="portrait-card__status">
                    <i :class="statusIcon(card)" aria-hidden="true"></i>
                    {{ statusLabel(card) }}
                  </span>
                  <span class="portrait-card__reveal">
                    <span>{{ card.summary || t('characterCards.wall.summaryEmpty') }}</span>
                    <b>{{ t('characterCards.wall.readFull') }} <i class="ri-arrow-right-line" aria-hidden="true"></i></b>
                  </span>
                </span>
                <span class="portrait-card__plaque">
                  <strong :style="buildNameStyle(displayNameColor(card), false, 'cardBg')">{{ displayName(card) }}</strong>
                  <span>{{ secondaryLine(card) }}</span>
                </span>
              </RouterLink>
              <RouterLink v-else class="portrait-card portrait-card--add" to="/character-cards/new">
                <span class="portrait-card__add-mark" aria-hidden="true"><i class="ri-add-line"></i></span>
                <strong>{{ t('characterCards.wall.addTitle') }}</strong>
                <span>{{ t('characterCards.wall.addBody') }}</span>
              </RouterLink>
            </template>
          </div>
        </div>
        <button class="character-wall__arrow character-wall__arrow--next" type="button" :disabled="page >= pages.length - 1" :aria-label="t('characterCards.wall.next')" aria-controls="character-wall-viewport" @click="goToPage(page + 1)">
          <i class="ri-arrow-right-s-line" aria-hidden="true"></i>
        </button>
      </div>
      <nav class="character-wall__pagination" :aria-label="t('characterCards.wall.pagination')">
        <button v-for="(_, index) in pages" :key="index" type="button" class="character-wall__dot" :class="{ active: page === index }" :aria-label="t('characterCards.wall.page', { page: index + 1 })" :aria-current="page === index ? 'page' : undefined" @click="goToPage(index)"></button>
        <span class="character-wall__page-status" aria-live="polite">{{ page + 1 }} / {{ pages.length }}</span>
      </nav>
    </template>
  </section>
</template>

<style scoped>
.character-wall { grid-column: 1 / -1; min-width: 0; padding: 24px; border: 1px solid var(--color-border); border-radius: 22px; background: var(--color-panel-bg); box-shadow: var(--shadow-sm); }
.character-wall__header, .character-wall__heading { display: flex; align-items: center; gap: 12px; }
.character-wall__header { justify-content: space-between; margin-bottom: 20px; }
.character-wall h2 { margin: 0; color: var(--color-text-main); font-size: 20px; font-weight: 650; letter-spacing: -.02em; }
.character-wall__count { color: var(--color-text-secondary); font-size: 13px; font-variant-numeric: tabular-nums; }
.character-wall__new, .character-wall__primary { display: inline-flex; min-height: 36px; align-items: center; justify-content: center; gap: 6px; padding: 0 14px; border: 1px solid var(--color-border); border-radius: 999px; background: var(--color-card-bg); color: var(--color-text-main); font-size: 12px; font-weight: 600; text-decoration: none; }
.character-wall__new:hover, .character-wall__primary:hover { background: var(--color-card-bg-hover); }
.character-wall__carousel { position: relative; padding: 0 22px; }
.character-wall__viewport { display: flex; overflow-x: auto; scroll-snap-type: x mandatory; scrollbar-width: none; border-radius: 16px; }
.character-wall__viewport::-webkit-scrollbar { display: none; }
.character-wall__grid { display: grid; flex: 0 0 100%; min-width: 0; grid-template-columns: repeat(var(--columns), minmax(0, 1fr)); gap: 16px; scroll-snap-align: start; scroll-snap-stop: always; padding: 4px 0 10px; box-sizing: border-box; }
.character-wall__arrow { position: absolute; z-index: 5; top: 50%; display: grid; width: 38px; height: 38px; place-items: center; padding: 0; border: 1px solid var(--color-border); border-radius: 50%; background: color-mix(in srgb, var(--color-panel-bg) 90%, transparent); color: var(--color-text-main); box-shadow: var(--shadow-sm); backdrop-filter: blur(12px); font-size: 24px; cursor: pointer; transform: translateY(-50%); transition: background 160ms ease, opacity 160ms ease; }
.character-wall__arrow--prev { left: -6px; }
.character-wall__arrow--next { right: -6px; }
.character-wall__arrow:hover:not(:disabled) { background: var(--color-card-bg-hover); }
.character-wall__arrow:disabled { opacity: .3; cursor: default; }
.character-wall__pagination { position: relative; display: flex; min-height: 24px; align-items: center; justify-content: center; gap: 2px; margin-top: 6px; flex-wrap: wrap; }
.character-wall__dot { display: grid; width: 24px; height: 24px; place-items: center; padding: 0; border: 0; background: transparent; cursor: pointer; }
.character-wall__dot::after { width: 6px; height: 6px; border-radius: 50%; background: var(--color-text-secondary); opacity: .35; content: ''; transition: opacity 180ms ease, transform 180ms ease; }
.character-wall__dot.active::after { background: var(--color-text-main); opacity: 1; transform: scale(1.25); }
.character-wall__page-status { position: absolute; right: 22px; color: var(--color-text-secondary); font-size: 11px; font-variant-numeric: tabular-nums; }
.portrait-card { position: relative; display: flex; min-width: 0; flex-direction: column; overflow: hidden; border: 1px solid var(--color-border); border-radius: 16px; background: var(--color-card-bg); color: inherit; text-decoration: none; transition: border-color 180ms ease, box-shadow 180ms ease; }
.portrait-card:hover { border-color: var(--color-border-hover); box-shadow: var(--shadow-sm); }
.portrait-card__frame { position: relative; display: block; overflow: hidden; height: 210px; background: var(--gradient-end); }
.portrait-card__image { display: block; width: 100%; height: 100%; object-fit: cover; transition: transform 300ms ease; }
.portrait-card:hover .portrait-card__image { transform: scale(1.035); }
.portrait-card__placeholder { display: grid; width: 100%; height: 100%; place-content: center; gap: 10px; background: linear-gradient(145deg, var(--gradient-start), var(--gradient-end)); color: var(--gradient-text); text-align: center; }
.portrait-card__placeholder i { font-size: 42px; }
.portrait-card__placeholder small { color: var(--gradient-text-muted); font-size: 11px; }
.portrait-card__shade { position: absolute; inset: 0; background: linear-gradient(to top, color-mix(in srgb, var(--gradient-end) 85%, transparent), transparent 55%); pointer-events: none; }
.portrait-card__status { position: absolute; z-index: 3; top: 12px; right: 12px; display: inline-flex; align-items: center; gap: 4px; padding: 4px 8px; border: 1px solid var(--gradient-border); border-radius: 999px; background: color-mix(in srgb, var(--gradient-end) 78%, transparent); color: var(--gradient-text); font-size: 10px; backdrop-filter: blur(6px); }
.portrait-card__reveal { position: absolute; right: 14px; bottom: 14px; left: 14px; display: grid; gap: 8px; color: var(--gradient-text); opacity: 0; transform: translateY(6px); transition: opacity 180ms ease, transform 180ms ease; }
.portrait-card__reveal > span { display: -webkit-box; overflow: hidden; font-size: 12px; line-height: 1.6; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }
.portrait-card__reveal b { color: var(--gradient-text); font-size: 11px; }
.portrait-card:hover .portrait-card__reveal, .portrait-card:focus-visible .portrait-card__reveal { opacity: 1; transform: translateY(0); }
.portrait-card__plaque { display: grid; min-width: 0; gap: 5px; padding: 14px; }
.portrait-card__plaque strong { overflow: hidden; color: var(--color-text-main); font-size: 16px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.portrait-card__plaque > span { overflow: hidden; color: var(--color-text-secondary); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.portrait-card--add { min-height: 280px; align-items: center; justify-content: center; gap: 10px; border-style: dashed; background: var(--color-card-bg); color: var(--color-text-main); text-align: center; }
.portrait-card--add > strong { font-size: 14px; }
.portrait-card--add > span:last-child { color: var(--color-text-secondary); font-size: 11px; }
.portrait-card__add-mark { display: grid; width: 48px; height: 48px; place-items: center; margin-bottom: 4px; border-radius: 50%; background: var(--color-card-bg-hover); color: var(--color-text-secondary); font-size: 24px; }
.character-wall__state { display: flex; min-height: 160px; align-items: center; justify-content: center; gap: 10px; color: var(--color-text-secondary); }
.character-wall__state--error { flex-wrap: wrap; }
.character-wall__state--error div { display: grid; gap: 4px; }
.character-wall__state--error strong { color: var(--color-text-main); }
.character-wall__state--error span { font-size: 12px; }
.character-wall__state button { padding: 8px 14px; border: 1px solid var(--color-border); border-radius: 999px; background: var(--color-card-bg); color: var(--color-text-main); cursor: pointer; }
.character-wall__empty { display: flex; min-height: 180px; align-items: center; gap: 24px; padding: 24px; border: 1px dashed var(--color-border); border-radius: 16px; }
.character-wall__empty-mark { position: relative; color: var(--color-accent); font-size: 44px; }
.character-wall__empty-plus { position: absolute; right: -4px; bottom: 0; font-size: 20px; }
.character-wall__empty-copy { flex: 1; }
.character-wall__empty-copy > span { color: var(--color-text-secondary); font-size: 11px; }
.character-wall__empty-copy h3 { margin: 6px 0; color: var(--color-text-main); font-size: 18px; }
.character-wall__empty-copy p { margin: 0; color: var(--color-text-secondary); font-size: 12px; line-height: 1.7; }
.character-wall :is(a, button, [tabindex]):focus-visible { outline: 3px solid var(--color-accent); outline-offset: 3px; }
.character-wall__spinner { animation: archive-spin 900ms linear infinite; }
@keyframes archive-spin { to { transform: rotate(360deg); } }
@media (max-width: 760px) { .character-wall { padding: 18px 14px; } .character-wall__empty { flex-direction: column; align-items: flex-start; padding: 20px; } .character-wall__page-status { right: 0; } }
@media (prefers-reduced-motion: reduce) { .character-wall *, .character-wall *::after { transition: none; animation: none; scroll-behavior: auto; } }
</style>
