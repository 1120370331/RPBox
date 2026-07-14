<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { getRPDBWorkPreview, resolveRPDBMediaURL, type RPDBWork, type RPDBWorkType } from '@/api/rpdb'
import { resolveApiUrl } from '@/api/item'

interface FallbackPreview {
  title: string
  summary: string
  author: string
  avatar: string
  image: string
  type: RPDBWorkType
  views: number
  likes: number
  favorites: number
  lists: number
}

const CARD_SELECTOR = '[data-jump-type="rpdb_work"]'
const route = useRoute()
const { t, locale } = useI18n()
const previewRef = ref<HTMLElement | null>(null)
const visible = ref(false)
const loading = ref(false)
const unavailable = ref(false)
const work = ref<RPDBWork | null>(null)
const fallback = ref<FallbackPreview | null>(null)
const left = ref(12)
const top = ref(12)
const copyState = ref<'idle' | 'copied' | 'failed'>('idle')
const previewCache = new Map<number, Promise<RPDBWork>>()
const previewTypes: Record<RPDBWorkType, { labelKey: string; icon: string; variant: string }> = {
  item_showcase: { labelKey: 'rpdb.jumpPreview.type.item', icon: 'ri-magic-line', variant: 'item' },
  transmog: { labelKey: 'rpdb.jumpPreview.type.transmog', icon: 'ri-shirt-line', variant: 'transmog' },
  home_showcase: { labelKey: 'rpdb.jumpPreview.type.home', icon: 'ri-home-heart-line', variant: 'home' },
}

let activeCard: HTMLElement | null = null
let hoverTimer: ReturnType<typeof setTimeout> | null = null
let closeTimer: ReturnType<typeof setTimeout> | null = null
let copyResetTimer: ReturnType<typeof setTimeout> | null = null
let pointerX = 0
let pointerY = 0
let requestToken = 0
let positionFrame = 0

const workType = computed<RPDBWorkType>(() => work.value?.type || fallback.value?.type || 'item_showcase')
const typeConfig = computed(() => previewTypes[workType.value])
const title = computed(() => work.value?.title || fallback.value?.title || t('rpdb.jumpPreview.unnamed'))
const summary = computed(() => work.value?.summary || work.value?.effect_description || fallback.value?.summary || t('rpdb.jumpPreview.emptySummary'))
const author = computed(() => work.value?.author_name || fallback.value?.author || t('rpdb.jumpPreview.anonymous'))
const cover = computed(() => work.value
  ? resolveRPDBMediaURL(work.value.cover_image)
  : fallback.value?.image || '')
const avatar = computed(() => work.value
  ? resolveApiUrl(work.value.author_avatar)
  : fallback.value?.avatar || '')
const metrics = computed(() => [
  { icon: 'ri-eye-line', title: t('rpdb.jumpPreview.metrics.views'), value: work.value?.view_count ?? fallback.value?.views ?? 0 },
  { icon: 'ri-heart-3-line', title: t('rpdb.jumpPreview.metrics.likes'), value: work.value?.like_count ?? fallback.value?.likes ?? 0 },
  { icon: 'ri-bookmark-3-line', title: t('rpdb.jumpPreview.metrics.favorites'), value: work.value?.favorite_count ?? fallback.value?.favorites ?? 0 },
  { icon: 'ri-list-check-3', title: t('rpdb.jumpPreview.metrics.lists'), value: work.value?.list_count ?? fallback.value?.lists ?? 0 },
])
const transmogShareCode = computed(() => {
  if (workType.value !== 'transmog' || !work.value?.extra) return ''
  try {
    const extra = JSON.parse(work.value.extra) as { share_code?: unknown }
    return String(extra.share_code || '').trim()
  } catch {
    return ''
  }
})

function parseCount(value: string | null) {
  const count = Number(value || 0)
  return Number.isFinite(count) ? count : 0
}

function resolveWorkType(card: HTMLElement): RPDBWorkType {
  const value = card.getAttribute('data-jump-rpdb-type')
  if (value === 'transmog' || value === 'home_showcase' || value === 'item_showcase') return value
  const variant = card.getAttribute('data-jump-variant') || ''
  if (variant === 'rpdb-transmog') return 'transmog'
  if (variant === 'rpdb-home') return 'home_showcase'
  return 'item_showcase'
}

function resolveWorkId(card: HTMLElement) {
  const href = card.getAttribute('data-jump-href') || card.getAttribute('href') || ''
  const match = href.match(/\/rpdb\/(\d+)/i)
  return match ? Number(match[1]) : 0
}

function readFallback(card: HTMLElement): FallbackPreview {
  return {
    title: card.getAttribute('data-jump-title') || card.querySelector('.jump-card__title')?.textContent?.trim() || '未命名作品',
    summary: card.getAttribute('data-jump-summary') || card.querySelector('.jump-card__rpdb-summary')?.textContent?.trim() || '',
    author: card.getAttribute('data-jump-author') || card.querySelector('.jump-card__rpdb-author')?.textContent?.trim() || '匿名贡献者',
    avatar: card.getAttribute('data-jump-avatar') || '',
    image: card.getAttribute('data-jump-image') || card.querySelector<HTMLImageElement>('.jump-card__rpdb-media img')?.src || '',
    type: resolveWorkType(card),
    views: parseCount(card.getAttribute('data-jump-views')),
    likes: parseCount(card.getAttribute('data-jump-likes')),
    favorites: parseCount(card.getAttribute('data-jump-favorites')),
    lists: parseCount(card.getAttribute('data-jump-lists')),
  }
}

function fetchPreview(id: number) {
  const cached = previewCache.get(id)
  if (cached) return cached
  const request = getRPDBWorkPreview(id)
    .then((result) => result.work)
    .catch((error) => {
      previewCache.delete(id)
      throw error
    })
  previewCache.set(id, request)
  return request
}

function schedulePosition() {
  if (positionFrame) return
  positionFrame = window.requestAnimationFrame(() => {
    positionFrame = 0
    const width = previewRef.value?.offsetWidth || Math.min(368, window.innerWidth - 24)
    const height = previewRef.value?.offsetHeight || 150
    const gap = 16
    const edge = 12
    let nextLeft = pointerX + gap
    let nextTop = pointerY + gap

    if (nextLeft + width > window.innerWidth - edge) nextLeft = pointerX - width - gap
    if (nextTop + height > window.innerHeight - edge) nextTop = pointerY - height - gap
    left.value = Math.max(edge, Math.min(nextLeft, window.innerWidth - width - edge))
    top.value = Math.max(edge, Math.min(nextTop, window.innerHeight - height - edge))
  })
}

function clearHoverTimer() {
  if (!hoverTimer) return
  clearTimeout(hoverTimer)
  hoverTimer = null
}

function clearCloseTimer() {
  if (!closeTimer) return
  clearTimeout(closeTimer)
  closeTimer = null
}

function scheduleClose() {
  clearCloseTimer()
  closeTimer = setTimeout(closePreview, 120)
}

function closePreview() {
  clearHoverTimer()
  clearCloseTimer()
  requestToken += 1
  activeCard = null
  visible.value = false
  loading.value = false
  unavailable.value = false
  work.value = null
  copyState.value = 'idle'
}

function openPreview(card: HTMLElement) {
  const id = resolveWorkId(card)
  if (!id) return
  clearCloseTimer()
  activeCard = card
  fallback.value = readFallback(card)
  clearHoverTimer()
  const token = ++requestToken

  hoverTimer = setTimeout(async () => {
    hoverTimer = null
    if (token !== requestToken || activeCard !== card) return
    work.value = null
    loading.value = true
    unavailable.value = false
    visible.value = true
    await nextTick()
    schedulePosition()

    try {
      const result = await fetchPreview(id)
      if (token !== requestToken || activeCard !== card) return
      work.value = result
    } catch {
      if (token !== requestToken || activeCard !== card) return
      unavailable.value = true
    } finally {
      if (token === requestToken && activeCard === card) {
        loading.value = false
        await nextTick()
        schedulePosition()
      }
    }
  }, 140)
}

function findCard(target: EventTarget | null) {
  const element = target instanceof Element ? target : null
  return element?.closest<HTMLElement>(CARD_SELECTOR) || null
}

function handleMouseOver(event: MouseEvent) {
  const card = findCard(event.target)
  if (!card || card === activeCard) return
  pointerX = event.clientX
  pointerY = event.clientY
  openPreview(card)
}

function handleMouseMove(event: MouseEvent) {
  if (!activeCard) return
  if (event.target instanceof Node && previewRef.value?.contains(event.target)) return
  pointerX = event.clientX
  pointerY = event.clientY
  if (visible.value) schedulePosition()
}

function handleMouseOut(event: MouseEvent) {
  if (!activeCard) return
  const card = findCard(event.target)
  if (card !== activeCard) return
  const related = event.relatedTarget
  if (related instanceof Node && activeCard.contains(related)) return
  scheduleClose()
}

function handleFocusIn(event: FocusEvent) {
  const card = findCard(event.target)
  if (!card || card === activeCard) return
  const rect = card.getBoundingClientRect()
  pointerX = rect.right
  pointerY = rect.top + Math.min(rect.height / 2, 60)
  openPreview(card)
}

function handleFocusOut(event: FocusEvent) {
  if (findCard(event.target) === activeCard) scheduleClose()
}

function handlePreviewEnter() {
  clearCloseTimer()
}

function handlePreviewLeave() {
  scheduleClose()
}

async function copyTransmogShareCode() {
  if (!transmogShareCode.value) return
  if (copyResetTimer) clearTimeout(copyResetTimer)
  try {
    await navigator.clipboard.writeText(transmogShareCode.value)
    copyState.value = 'copied'
  } catch {
    copyState.value = 'failed'
  }
  copyResetTimer = setTimeout(() => {
    copyState.value = 'idle'
    copyResetTimer = null
  }, 1600)
}

function formatCount(value: number) {
  return new Intl.NumberFormat(locale.value, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value || 0)
}

watch(() => route.fullPath, closePreview)

onMounted(() => {
  document.addEventListener('mouseover', handleMouseOver, true)
  document.addEventListener('mousemove', handleMouseMove, true)
  document.addEventListener('mouseout', handleMouseOut, true)
  document.addEventListener('focusin', handleFocusIn, true)
  document.addEventListener('focusout', handleFocusOut, true)
  document.addEventListener('scroll', closePreview, true)
  window.addEventListener('resize', closePreview)
})

onBeforeUnmount(() => {
  closePreview()
  document.removeEventListener('mouseover', handleMouseOver, true)
  document.removeEventListener('mousemove', handleMouseMove, true)
  document.removeEventListener('mouseout', handleMouseOut, true)
  document.removeEventListener('focusin', handleFocusIn, true)
  document.removeEventListener('focusout', handleFocusOut, true)
  document.removeEventListener('scroll', closePreview, true)
  window.removeEventListener('resize', closePreview)
  if (positionFrame) window.cancelAnimationFrame(positionFrame)
  if (copyResetTimer) clearTimeout(copyResetTimer)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="rpdb-jump-preview">
      <aside
        v-if="visible"
        ref="previewRef"
        class="rpdb-jump-preview"
        :class="`rpdb-jump-preview--${typeConfig.variant}`"
        :style="{ left: `${left}px`, top: `${top}px` }"
        role="dialog"
        :aria-label="title"
        data-testid="rpdb-jump-preview"
        @mouseenter="handlePreviewEnter"
        @mouseleave="handlePreviewLeave"
        @focusin="handlePreviewEnter"
        @focusout="handlePreviewLeave"
      >
        <div class="rpdb-jump-preview__media" :class="{ empty: !cover }">
          <img v-if="cover" :src="cover" :alt="title" />
          <i v-else :class="typeConfig.icon"></i>
        </div>

        <div v-if="unavailable" class="rpdb-jump-preview__state">
          <i class="ri-link-unlink-m"></i>
          <b>{{ t('rpdb.jumpPreview.unavailable') }}</b>
          <span>{{ t('rpdb.jumpPreview.unavailableHint') }}</span>
        </div>

        <div v-else class="rpdb-jump-preview__body" :class="{ loading }">
          <div class="rpdb-jump-preview__meta">
            <span><i :class="typeConfig.icon"></i>{{ t(typeConfig.labelKey) }}</span>
            <span class="rpdb-jump-preview__author">
              <img v-if="avatar" :src="avatar" alt="" />
              <i v-else class="ri-user-3-line"></i>
              {{ author }}
            </span>
          </div>
          <h3>{{ title }}</h3>
          <p>{{ loading ? t('rpdb.jumpPreview.loading') : summary }}</p>
          <div class="rpdb-jump-preview__metrics" :aria-label="t('rpdb.jumpPreview.metrics.label')">
            <span v-for="metric in metrics" :key="metric.title" :title="metric.title" :aria-label="`${metric.title} ${formatCount(metric.value)}`">
              <i :class="metric.icon"></i><b>{{ formatCount(metric.value) }}</b>
            </span>
          </div>
          <div v-if="transmogShareCode" class="rpdb-jump-preview__code" data-testid="rpdb-jump-preview-code">
            <span>
              <small>{{ t('rpdb.jumpPreview.transmogCode') }}</small>
              <code :title="transmogShareCode">{{ transmogShareCode }}</code>
            </span>
            <button
              type="button"
              data-testid="copy-transmog-share-code"
              :class="{ copied: copyState === 'copied', failed: copyState === 'failed' }"
              :title="t('rpdb.jumpPreview.copyCode')"
              :aria-label="t('rpdb.jumpPreview.copyCode')"
              @click.stop="copyTransmogShareCode"
            >
              <i :class="copyState === 'copied' ? 'ri-check-line' : copyState === 'failed' ? 'ri-error-warning-line' : 'ri-file-copy-line'"></i>
              <span class="sr-only">{{ copyState === 'copied' ? t('rpdb.jumpPreview.copied') : copyState === 'failed' ? t('rpdb.jumpPreview.copyFailed') : t('rpdb.jumpPreview.copyCode') }}</span>
            </button>
          </div>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.rpdb-jump-preview {
  --preview-accent: #A65F2A;
  --preview-soft: #FBF2E8;
  position: fixed;
  z-index: 2400;
  display: grid;
  width: min(388px, calc(100vw - 24px));
  min-height: 142px;
  grid-template-columns: 126px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--preview-accent) 42%, var(--color-border, #E8DCCF));
  border-radius: 8px;
  background: var(--color-panel-bg, #fff);
  color: var(--color-text-main, #2C1810);
  box-shadow: 0 16px 38px rgba(44, 24, 16, 0.24);
  pointer-events: auto;
}

.rpdb-jump-preview--transmog {
  --preview-accent: #55758B;
  --preview-soft: #EEF3F6;
}

.rpdb-jump-preview--home {
  --preview-accent: #4F7A62;
  --preview-soft: #EEF5F1;
}

.rpdb-jump-preview__media {
  display: grid;
  min-width: 0;
  place-items: center;
  overflow: hidden;
  border-right: 3px solid var(--preview-accent);
  background: var(--preview-soft);
  color: var(--preview-accent);
  font-size: 34px;
}

.rpdb-jump-preview__media img {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: block;
  object-fit: cover;
}

.rpdb-jump-preview__body,
.rpdb-jump-preview__state {
  display: flex;
  min-width: 0;
  flex-direction: column;
  justify-content: center;
  padding: 13px 15px;
}

.rpdb-jump-preview__body.loading {
  opacity: 0.72;
}

.rpdb-jump-preview__meta {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--preview-accent);
  font-size: 10px;
  font-weight: 800;
}

.rpdb-jump-preview__meta > span,
.rpdb-jump-preview__author {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.rpdb-jump-preview__author {
  overflow: hidden;
  color: var(--color-text-secondary, #8C7B70);
  font-weight: 500;
  text-overflow: ellipsis;
}

.rpdb-jump-preview__author img {
  width: 17px;
  height: 17px;
  flex: 0 0 17px;
  border-radius: 50%;
  object-fit: cover;
}

.rpdb-jump-preview h3 {
  overflow: hidden;
  margin: 6px 0 4px;
  color: var(--color-text-main, #2C1810);
  font-size: 15px;
  line-height: 1.3;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.rpdb-jump-preview p {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--color-text-secondary, #8C7B70);
  font-size: 11px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.rpdb-jump-preview__metrics {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 13px;
  margin-top: 9px;
  font-variant-numeric: tabular-nums;
}

.rpdb-jump-preview__metrics span {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 3px;
  color: var(--color-text-secondary, #8C7B70);
  white-space: nowrap;
}

.rpdb-jump-preview__metrics i {
  color: var(--preview-accent);
  font-size: 11px;
}

.rpdb-jump-preview__metrics b {
  overflow: hidden;
  color: var(--color-text-main, #2C1810);
  font-size: 10px;
  text-overflow: ellipsis;
}

.rpdb-jump-preview__code {
  display: grid;
  min-width: 0;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid color-mix(in srgb, var(--preview-accent) 22%, var(--color-border, #E8DCCF));
}

.rpdb-jump-preview__code > span {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.rpdb-jump-preview__code small {
  color: var(--color-text-secondary, #8C7B70);
  font-size: 9px;
  line-height: 1;
}

.rpdb-jump-preview__code code {
  overflow: hidden;
  color: var(--color-text-main, #2C1810);
  font: 10px/1.25 Consolas, 'SFMono-Regular', monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rpdb-jump-preview__code button {
  display: inline-flex;
  min-width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 0 7px;
  border: 1px solid color-mix(in srgb, var(--preview-accent) 40%, var(--color-border, #E8DCCF));
  border-radius: 5px;
  background: color-mix(in srgb, var(--preview-accent) 8%, var(--color-panel-bg, #fff));
  color: var(--preview-accent);
  cursor: pointer;
  font-size: 10px;
  font-weight: 800;
}

.rpdb-jump-preview__code button:hover,
.rpdb-jump-preview__code button:focus-visible {
  border-color: var(--preview-accent);
  outline: none;
}

.rpdb-jump-preview__code button.copied {
  color: var(--color-success, #2F7D50);
}

.rpdb-jump-preview__code button.failed {
  color: var(--color-danger, #B23A3A);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  clip-path: inset(50%);
  white-space: nowrap;
}

.rpdb-jump-preview__state {
  align-items: flex-start;
  gap: 5px;
}

.rpdb-jump-preview__state > i {
  color: var(--preview-accent);
  font-size: 22px;
}

.rpdb-jump-preview__state b {
  font-size: 14px;
}

.rpdb-jump-preview__state span {
  color: var(--color-text-secondary, #8C7B70);
  font-size: 11px;
  line-height: 1.45;
}

.rpdb-jump-preview-enter-active,
.rpdb-jump-preview-leave-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.rpdb-jump-preview-enter-from,
.rpdb-jump-preview-leave-to {
  opacity: 0;
  transform: translateY(3px);
}

@media (max-width: 520px) {
  .rpdb-jump-preview {
    grid-template-columns: 96px minmax(0, 1fr);
  }

}

@media (prefers-reduced-motion: reduce) {
  .rpdb-jump-preview-enter-active,
  .rpdb-jump-preview-leave-active {
    transition: none;
  }
}
</style>
