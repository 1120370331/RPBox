<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { getCharacterCard, type CharacterCard } from '@/api/characterCard'
import { resolveApiUrl } from '@/api/image'
import ImagePreviewDialog from '@/components/ImagePreviewDialog.vue'
import { handleJumpLinkClick, sanitizeJumpLinks } from '@/utils/jumpLink'

type DetailTab = 'basic' | 'background' | 'impression' | 'other'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const card = ref<CharacterCard | null>(null)
const loading = ref(true)
const loadFailed = ref(false)
const activeTab = ref<DetailTab>('basic')
const selectedPortraitIndex = ref(0)
const previewOpen = ref(false)
const previewSrc = ref('')
const previewAlt = ref('')
const backgroundRef = ref<HTMLElement | null>(null)
const impressionRef = ref<HTMLElement | null>(null)
const otherRef = ref<HTMLElement | null>(null)

const cardId = computed(() => Number(route.params.id))
const displayName = computed(() => {
  if (!card.value) return ''
  return card.value.display_name.trim()
    || [card.value.first_name, card.value.last_name].filter(Boolean).join(' ')
    || `#${card.value.id}`
})
const displayColor = computed(() => normalizeColor(card.value?.name_color || card.value?.class_color || ''))
const identityLine = computed(() => [card.value?.race, card.value?.class].filter(Boolean).join(' · '))
const tabs = computed<Array<{ id: DetailTab; label: string }>>(() => [
  { id: 'basic', label: t('characterCards.detail.tabs.basic') },
  { id: 'background', label: t('characterCards.detail.tabs.background') },
  { id: 'impression', label: t('characterCards.detail.tabs.impression') },
  { id: 'other', label: t('characterCards.detail.tabs.other') },
])
const portraitSources = computed(() => {
  if (!card.value) return []
  const sources = [...(card.value.portraits || [])]
    .sort((left, right) => left.sort_order - right.sort_order)
    .map(portrait => ({
      id: portrait.id,
      src: resolveApiUrl(portrait.image_url),
      isCover: portrait.is_cover,
    }))
    .filter(portrait => portrait.src)

  const legacySource = resolveApiUrl(card.value.portrait_image_url)
  if (legacySource && !sources.some(portrait => portrait.src === legacySource)) {
    sources.unshift({ id: 0, src: legacySource, isCover: sources.length === 0 })
  }
  return sources
})
const selectedPortrait = computed(() => portraitSources.value[selectedPortraitIndex.value] || null)
const activeImpressions = computed(() => (card.value?.impressions || [])
  .filter(impression => impression.active)
  .sort((left, right) => left.slot - right.slot)
  .slice(0, 5))
const basicFields = computed(() => {
  if (!card.value) return []
  return [
    [t('characterCards.detail.fields.name'), displayName.value],
    [t('characterCards.detail.fields.race'), card.value.race],
    [t('characterCards.detail.fields.class'), card.value.class],
    [t('characterCards.detail.fields.eyes'), card.value.eye_color],
    [t('characterCards.detail.fields.age'), card.value.age],
    [t('characterCards.detail.fields.height'), card.value.height],
    [t('characterCards.detail.fields.weight'), card.value.weight],
    [t('characterCards.detail.fields.birthplace'), card.value.birthplace],
    [t('characterCards.detail.fields.residence'), card.value.residence],
    [t('characterCards.detail.fields.relationship'), card.value.relationship_status],
  ].filter((entry): entry is [string, string] => Boolean(entry[1]))
})
const backgroundHtml = computed(() => normalizeRichContent(card.value?.background_story || ''))
const impressionHtml = computed(() => normalizeRichContent(card.value?.first_impression || ''))
const otherHtml = computed(() => normalizeRichContent(card.value?.other_content || ''))

function normalizeColor(value: string) {
  const normalized = value.trim().replace(/^#/, '')
  return /^(?:[\da-f]{6}|[\da-f]{8})$/i.test(normalized) ? `#${normalized}` : ''
}

function normalizeRichContent(input: string) {
  if (!input) return ''
  const doc = new DOMParser().parseFromString(input, 'text/html')
  doc.querySelectorAll('script, style, iframe, object, embed').forEach(node => node.remove())
  doc.querySelectorAll<HTMLElement>('*').forEach(node => {
    Array.from(node.attributes).forEach(attribute => {
      if (attribute.name.toLowerCase().startsWith('on')) node.removeAttribute(attribute.name)
    })
  })
  doc.querySelectorAll('img').forEach(image => {
    const src = image.getAttribute('src') || ''
    if (!/^(https?:|data:image\/|\/|\.\/|\.\.\/)/i.test(src)) {
      image.remove()
      return
    }
    image.removeAttribute('srcset')
    image.setAttribute('src', resolveApiUrl(src))
    image.setAttribute('loading', 'lazy')
  })
  doc.querySelectorAll('a').forEach(link => {
    const href = link.getAttribute('href') || ''
    if (!/^(https?:|mailto:|\/|#)/i.test(href)) {
      link.removeAttribute('href')
      return
    }
    if (/^https?:\/\//i.test(href)) {
      link.setAttribute('target', '_blank')
      link.setAttribute('rel', 'noopener noreferrer')
    }
  })
  sanitizeJumpLinks(doc.body)
  return doc.body.innerHTML
}

async function hydrateRichLinks() {
  await nextTick()
  for (const container of [backgroundRef.value, impressionRef.value, otherRef.value]) {
    sanitizeJumpLinks(container)
  }
}

async function loadCard() {
  if (!Number.isInteger(cardId.value) || cardId.value <= 0) {
    loading.value = false
    loadFailed.value = true
    return
  }
  loading.value = true
  loadFailed.value = false
  try {
    card.value = await getCharacterCard(cardId.value)
    const coverIndex = portraitSources.value.findIndex(portrait => portrait.isCover)
    selectedPortraitIndex.value = Math.max(0, coverIndex)
    await hydrateRichLinks()
  } catch (error) {
    console.error('Failed to load public character card', error)
    card.value = null
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function goBack() {
  if (window.history.length > 1) {
    router.back()
    return
  }
  void router.replace('/community')
}

function openPreview(src: string, alt = '') {
  if (!src) return
  previewSrc.value = resolveApiUrl(src)
  previewAlt.value = alt
  previewOpen.value = true
}

function handleRichContentClick(event: MouseEvent) {
  if (handleJumpLinkClick(event, router)) return
  const target = event.target instanceof Element ? event.target : null
  const image = target?.closest('img') as HTMLImageElement | null
  if (!image?.src) return
  event.preventDefault()
  openPreview(image.currentSrc || image.src, image.alt)
}

watch(cardId, loadCard, { immediate: true })
watch(activeTab, hydrateRichLinks)
</script>

<template>
  <div class="sub-page character-card-page">
    <header class="sub-header character-card-header">
      <button type="button" class="back-btn" :aria-label="t('common.button.back')" @click="goBack">
        <i class="ri-arrow-left-line" aria-hidden="true" />
      </button>
      <div>
        <span>RPBOX · CHARACTER</span>
        <h1>{{ t('characterCards.detail.title') }}</h1>
      </div>
    </header>

    <main class="sub-body character-card-body">
      <div v-if="loading" class="page-state">
        <i class="ri-loader-4-line spin" aria-hidden="true" />
        <p>{{ t('characterCards.detail.loading') }}</p>
      </div>

      <div v-else-if="loadFailed || !card" class="page-state page-state--error">
        <i class="ri-file-warning-line" aria-hidden="true" />
        <p>{{ t('characterCards.detail.unavailable') }}</p>
        <button type="button" @click="loadCard">{{ t('characterCards.detail.retry') }}</button>
      </div>

      <template v-else>
        <section class="portrait-dossier">
          <button
            v-if="selectedPortrait"
            type="button"
            class="portrait-stage"
            :aria-label="t('characterCards.detail.previewImage')"
            @click="openPreview(selectedPortrait.src, t('characterCards.detail.portraitAlt', { name: displayName }))"
          >
            <img :src="selectedPortrait.src" :alt="t('characterCards.detail.portraitAlt', { name: displayName })" />
            <span class="portrait-stage__zoom"><i class="ri-zoom-in-line" aria-hidden="true" />{{ t('characterCards.detail.previewImage') }}</span>
          </button>
          <div v-else class="portrait-empty">
            <i class="ri-user-star-line" aria-hidden="true" />
            <span>{{ t('characterCards.detail.portraitMissing') }}</span>
          </div>

          <div v-if="portraitSources.length > 1" class="portrait-film">
            <button
              v-for="(portrait, index) in portraitSources"
              :key="portrait.id || portrait.src"
              type="button"
              :class="{ active: selectedPortraitIndex === index }"
              :aria-pressed="selectedPortraitIndex === index"
              @click="selectedPortraitIndex = index"
            >
              <img :src="portrait.src" alt="" />
            </button>
          </div>

          <div class="character-plaque">
            <span class="public-mark"><i class="ri-shield-check-line" aria-hidden="true" />{{ t('characterCards.detail.publicMark') }}</span>
            <h2 :style="displayColor ? { color: displayColor } : undefined">{{ displayName }}</h2>
            <strong>{{ card.title || card.full_title || t('characterCards.detail.titleMissing') }}</strong>
            <p>{{ identityLine || t('characterCards.detail.identityMissing') }}</p>
          </div>
        </section>

        <p class="character-summary">{{ card.summary || t('characterCards.detail.summaryMissing') }}</p>

        <nav class="detail-tabs" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            type="button"
            role="tab"
            :aria-selected="activeTab === tab.id"
            :class="{ active: activeTab === tab.id }"
            @click="activeTab = tab.id"
          >{{ tab.label }}</button>
        </nav>

        <section v-show="activeTab === 'basic'" class="detail-sheet" role="tabpanel">
          <dl v-if="basicFields.length" class="identity-grid">
            <div v-for="entry in basicFields" :key="entry[0]">
              <dt>{{ entry[0] }}</dt>
              <dd :style="entry[0] === t('characterCards.detail.fields.name') && displayColor ? { color: displayColor } : undefined">{{ entry[1] }}</dd>
            </div>
          </dl>
          <p v-else class="empty-section">{{ t('characterCards.detail.emptySection') }}</p>
        </section>

        <section v-show="activeTab === 'background'" class="detail-sheet" role="tabpanel">
          <div v-if="backgroundHtml" ref="backgroundRef" class="rich-content" v-html="backgroundHtml" @click="handleRichContentClick" />
          <p v-else class="empty-section">{{ t('characterCards.detail.emptySection') }}</p>
        </section>

        <section v-show="activeTab === 'impression'" class="detail-sheet impression-sheet" role="tabpanel">
          <article v-for="impression in activeImpressions" :key="impression.slot" class="impression-card">
            <button
              v-if="impression.icon_image_url"
              type="button"
              class="impression-mark impression-mark--image"
              :aria-label="t('characterCards.detail.previewImage')"
              @click="openPreview(impression.icon_image_url, impression.title)"
            >
              <img :src="resolveApiUrl(impression.icon_image_url)" alt="" />
              <i class="ri-zoom-in-line" aria-hidden="true" />
            </button>
            <div v-else class="impression-mark" :title="impression.trp3_icon">
              <span>{{ String(impression.slot).padStart(2, '0') }}</span>
            </div>
            <div class="impression-copy">
              <small>{{ t('characterCards.detail.observation', { slot: String(impression.slot).padStart(2, '0') }) }}</small>
              <h3>{{ impression.title || t('characterCards.detail.unnamedObservation') }}</h3>
              <p>{{ impression.text }}</p>
            </div>
            <button
              v-if="impression.image_url"
              type="button"
              class="impression-image"
              :aria-label="t('characterCards.detail.previewImage')"
              @click="openPreview(impression.image_url, impression.title)"
            >
              <img :src="resolveApiUrl(impression.image_url)" :alt="impression.title" />
              <i class="ri-zoom-in-line" aria-hidden="true" />
            </button>
          </article>

          <section v-if="impressionHtml" class="impression-notes">
            <h3>{{ t('characterCards.detail.notes') }}</h3>
            <div ref="impressionRef" class="rich-content" v-html="impressionHtml" @click="handleRichContentClick" />
          </section>
          <p v-if="!activeImpressions.length && !impressionHtml" class="empty-section">{{ t('characterCards.detail.emptySection') }}</p>
        </section>

        <section v-show="activeTab === 'other'" class="detail-sheet" role="tabpanel">
          <div v-if="otherHtml" ref="otherRef" class="rich-content" v-html="otherHtml" @click="handleRichContentClick" />
          <p v-else class="empty-section">{{ t('characterCards.detail.emptySection') }}</p>
        </section>
      </template>
    </main>

    <ImagePreviewDialog
      :open="previewOpen"
      :src="previewSrc"
      :alt="previewAlt"
      @close="previewOpen = false"
    />
  </div>
</template>

<style scoped>
.character-card-page {
  color: var(--color-text-main);
}

.character-card-header > div {
  min-width: 0;
}

.character-card-header span {
  display: block;
  color: var(--color-text-secondary);
  font: 700 9px/1.2 ui-monospace, SFMono-Regular, Consolas, monospace;
  letter-spacing: 0.14em;
}

.character-card-header h1 {
  margin-top: 4px;
  color: var(--color-text-main);
}

.character-card-body {
  display: grid;
  gap: 14px;
  padding-top: var(--content-top-gap);
}

.page-state {
  display: grid;
  min-height: 58vh;
  place-content: center;
  justify-items: center;
  gap: 12px;
  color: var(--color-text-secondary);
  text-align: center;
}

.page-state > i {
  color: var(--color-accent);
  font-size: 38px;
}

.page-state p {
  max-width: 310px;
  line-height: 1.65;
}

.page-state button {
  min-height: 42px;
  padding: 0 17px;
  border: 0;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--btn-primary-text);
  font: inherit;
}

.portrait-dossier {
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-card-bg);
  box-shadow: var(--shadow-md);
}

.portrait-stage {
  position: relative;
  display: block;
  width: 100%;
  aspect-ratio: 4 / 5;
  overflow: hidden;
  padding: 0;
  border: 0;
  background: var(--color-primary-light);
}

.portrait-stage img,
.portrait-empty {
  width: 100%;
  height: 100%;
}

.portrait-stage img {
  display: block;
  object-fit: cover;
}

.portrait-stage::after {
  position: absolute;
  inset: 45% 0 0;
  background: linear-gradient(transparent, color-mix(in srgb, var(--color-primary) 72%, transparent));
  content: '';
  pointer-events: none;
}

.portrait-stage__zoom {
  position: absolute;
  right: 12px;
  bottom: 12px;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 7px 10px;
  border: 1px solid color-mix(in srgb, var(--color-text-light) 35%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 75%, transparent);
  color: var(--color-text-light);
  font-size: 11px;
  backdrop-filter: blur(7px);
}

.portrait-empty {
  display: grid;
  min-height: 320px;
  place-content: center;
  justify-items: center;
  gap: 10px;
  background: var(--color-primary-light);
  color: var(--color-text-secondary);
}

.portrait-empty i {
  color: var(--color-accent);
  font-size: 54px;
}

.portrait-film {
  display: flex;
  gap: 8px;
  padding: 10px 12px 0;
  overflow-x: auto;
}

.portrait-film button {
  flex: 0 0 58px;
  width: 58px;
  height: 70px;
  overflow: hidden;
  padding: 0;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
}

.portrait-film button.active {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 22%, transparent);
}

.portrait-film img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.character-plaque {
  display: grid;
  justify-items: center;
  gap: 5px;
  padding: 16px;
  text-align: center;
}

.public-mark {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--color-success-light);
  color: var(--color-success);
  font-size: 10px;
  font-weight: 700;
}

.character-plaque h2 {
  max-width: 100%;
  overflow: hidden;
  margin: 4px 0 0;
  color: var(--color-text-main);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 26px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.character-plaque strong {
  color: var(--color-text-main);
  font-size: 13px;
}

.character-plaque p {
  color: var(--color-text-secondary);
  font-size: 12px;
}

.character-summary {
  margin: 0;
  padding: 14px 16px;
  border-left: 3px solid var(--color-accent);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  background: var(--color-card-bg);
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.75;
}

.detail-tabs {
  display: flex;
  gap: 7px;
  padding-bottom: 2px;
  overflow-x: auto;
  scrollbar-width: none;
}

.detail-tabs::-webkit-scrollbar {
  display: none;
}

.detail-tabs button {
  flex: 0 0 auto;
  min-height: 39px;
  padding: 0 14px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-card-bg);
  color: var(--color-text-secondary);
  font: inherit;
  font-size: 12px;
}

.detail-tabs button.active {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--btn-primary-text);
}

.detail-sheet {
  min-height: 260px;
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
}

.identity-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.identity-grid > div {
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
}

.identity-grid dt {
  color: var(--color-text-secondary);
  font-size: 10px;
}

.identity-grid dd {
  overflow-wrap: anywhere;
  margin: 5px 0 0;
  color: var(--color-text-main);
  font-size: 14px;
  font-weight: 700;
}

.empty-section {
  display: grid;
  min-height: 220px;
  place-content: center;
  color: var(--color-text-secondary);
  text-align: center;
}

.impression-sheet {
  display: grid;
  align-content: start;
  gap: 12px;
}

.impression-card {
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
}

.impression-mark {
  position: relative;
  display: grid;
  width: 58px;
  height: 58px;
  place-items: center;
  overflow: hidden;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-primary-light);
  color: var(--color-primary);
  font: 700 13px/1 ui-monospace, SFMono-Regular, Consolas, monospace;
}

.impression-mark img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.impression-mark i,
.impression-image i {
  position: absolute;
  right: 4px;
  bottom: 4px;
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-primary) 78%, transparent);
  color: var(--color-text-light);
  font-size: 12px;
}

.impression-copy {
  min-width: 0;
}

.impression-copy small {
  color: var(--color-accent);
  font: 700 9px/1.2 ui-monospace, SFMono-Regular, Consolas, monospace;
  letter-spacing: 0.08em;
}

.impression-copy h3 {
  margin: 5px 0;
  color: var(--color-text-main);
  font-size: 14px;
}

.impression-copy p {
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.65;
  white-space: pre-wrap;
}

.impression-image {
  position: relative;
  grid-column: 1 / -1;
  width: 100%;
  max-height: 230px;
  overflow: hidden;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: var(--color-primary-light);
}

.impression-image img {
  display: block;
  width: 100%;
  max-height: 230px;
  object-fit: cover;
}

.impression-notes {
  padding-top: 4px;
}

.impression-notes > h3 {
  margin: 0 0 10px;
  color: var(--color-text-main);
  font-size: 14px;
}

.rich-content {
  overflow-wrap: anywhere;
  color: var(--color-text-main);
  font-size: 14px;
  line-height: 1.8;
}

.rich-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: var(--radius-sm);
  cursor: zoom-in;
}

.rich-content :deep(a) {
  color: var(--color-secondary);
}

.rich-content :deep(p + p) {
  margin-top: 0.8em;
}

.spin {
  animation: character-card-spin 900ms linear infinite;
}

@keyframes character-card-spin {
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .spin { animation: none; }
}
</style>
