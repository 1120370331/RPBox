<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  getCharacterCard,
  type CharacterCard,
} from '@/api/characterCard'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import CharacterCardPortrait from './CharacterCardPortrait.vue'

const CARD_SELECTOR = '[data-jump-type="character_card"]'
const route = useRoute()
const previewRef = ref<HTMLElement | null>(null)
const visible = ref(false)
const loading = ref(false)
const unavailable = ref(false)
const characterCard = ref<CharacterCard | null>(null)
const left = ref(12)
const top = ref(12)
const previewRequests = new Map<number, Promise<CharacterCard>>()

let activeCard: HTMLElement | null = null
let hoverTimer: ReturnType<typeof setTimeout> | null = null
let closeTimer: ReturnType<typeof setTimeout> | null = null
let pointerX = 0
let pointerY = 0
let requestToken = 0
let positionFrame = 0

const name = computed(() => {
  const card = characterCard.value
  if (!card) return loading.value ? '正在查阅人物档案…' : '人物卡暂不可用'
  return card.display_name?.trim()
    || [card.first_name, card.last_name].map((part) => part?.trim()).filter(Boolean).join(' ')
    || '未命名人物'
})
const title = computed(() => characterCard.value?.title || characterCard.value?.full_title || '人物档案')
const summary = computed(() => characterCard.value?.summary
  || [characterCard.value?.race, characterCard.value?.class].filter(Boolean).join(' · ')
  || '角色摘要尚未填写。')
const identity = computed(() => [characterCard.value?.race, characterCard.value?.class].filter(Boolean).join(' · '))
const nameColor = computed(() => getCharacterCardDisplayColor(characterCard.value))

function resolveId(card: HTMLElement) {
  const direct = Number(card.getAttribute('data-jump-id'))
  if (Number.isFinite(direct) && direct > 0) return direct
  const href = card.getAttribute('data-jump-href') || card.getAttribute('href') || ''
  const match = href.match(/\/character-cards\/(\d+)/i)
  return match ? Number(match[1]) : 0
}

function fetchPreview(id: number) {
  const pending = previewRequests.get(id)
  if (pending) return pending
  const request = getCharacterCard(id)
    .then((result) => {
      if (result.status !== 'published'
        || result.visibility !== 'public'
        || (result.review_status && result.review_status !== 'approved')) {
        throw Object.assign(new Error('character card is not public'), { status: 404 })
      }
      return result
    })
    .catch((error) => {
      throw error
    })
    .finally(() => {
      if (previewRequests.get(id) === request) previewRequests.delete(id)
    })
  previewRequests.set(id, request)
  return request
}

function schedulePosition() {
  if (positionFrame) return
  positionFrame = window.requestAnimationFrame(() => {
    positionFrame = 0
    const width = previewRef.value?.offsetWidth || Math.min(382, window.innerWidth - 24)
    const height = previewRef.value?.offsetHeight || 190
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
  characterCard.value = null
}

function redactUnavailableCard(card: HTMLElement) {
  card.setAttribute('data-jump-unavailable', 'true')
  card.setAttribute('aria-disabled', 'true')
  card.setAttribute('tabindex', '-1')
  card.setAttribute('data-jump-title', '人物卡暂不可用')
  for (const attr of ['data-jump-image', 'data-jump-summary', 'data-jump-author', 'data-jump-status', 'data-jump-visibility']) {
    card.removeAttribute(attr)
  }
}

function openPreview(card: HTMLElement) {
  const id = resolveId(card)
  if (!id) return
  clearCloseTimer()
  activeCard = card
  clearHoverTimer()
  const token = ++requestToken

  hoverTimer = setTimeout(async () => {
    hoverTimer = null
    if (token !== requestToken || activeCard !== card) return
    characterCard.value = null
    loading.value = card.getAttribute('data-jump-unavailable') !== 'true'
    unavailable.value = card.getAttribute('data-jump-unavailable') === 'true'
    visible.value = true
    await nextTick()
    schedulePosition()
    if (unavailable.value) return

    try {
      const result = await fetchPreview(id)
      if (token !== requestToken || activeCard !== card) return
      characterCard.value = result
    } catch (error: unknown) {
      if (token !== requestToken || activeCard !== card) return
      unavailable.value = true
      const status = typeof error === 'object' && error ? (error as { status?: number }).status : undefined
      if (status === 403 || status === 404) redactUnavailableCard(card)
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
  if (!activeCard || findCard(event.target) !== activeCard) return
  const related = event.relatedTarget
  if (related instanceof Node && activeCard.contains(related)) return
  scheduleClose()
}

function handleFocusIn(event: FocusEvent) {
  const card = findCard(event.target)
  if (!card || card === activeCard) return
  const rect = card.getBoundingClientRect()
  pointerX = rect.right
  pointerY = rect.top + Math.min(rect.height / 2, 70)
  openPreview(card)
}

function handleFocusOut(event: FocusEvent) {
  if (findCard(event.target) === activeCard) scheduleClose()
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
})
</script>

<template>
  <Teleport to="body">
    <Transition name="character-preview">
      <aside
        v-if="visible"
        ref="previewRef"
        class="character-preview"
        :style="{ left: `${left}px`, top: `${top}px` }"
        role="dialog"
        :aria-label="name"
        data-testid="character-card-jump-preview"
        @mouseenter="clearCloseTimer"
        @mouseleave="scheduleClose"
        @focusin="clearCloseTimer"
        @focusout="scheduleClose"
      >
        <div class="character-preview__portrait" :class="{ empty: !characterCard?.portrait_image_url }">
          <CharacterCardPortrait
            v-if="characterCard?.portrait_image_url && !unavailable"
            class="character-preview__image"
            :card="characterCard"
            :alt="`${name}的角色肖像`"
            :width="420"
            :quality="86"
          />
          <i v-else :class="unavailable ? 'ri-link-unlink-m' : loading ? 'ri-loader-4-line character-preview__spin' : 'ri-user-star-line'" aria-hidden="true"></i>
          <span class="character-preview__rail" aria-hidden="true"></span>
        </div>

        <div v-if="unavailable" class="character-preview__state">
          <span>Character file</span>
          <h3>人物卡暂不可用</h3>
          <p>这份档案可能已删除、设为私密或尚未发布。</p>
        </div>

        <div v-else class="character-preview__body" :class="{ loading }">
          <div class="character-preview__meta">
            <span><i class="ri-id-card-line" aria-hidden="true"></i>人物卡</span>
            <small>{{ loading ? '查阅中' : '公开档案' }}</small>
          </div>
          <h3 :style="nameColor ? { color: nameColor } : undefined">{{ name }}</h3>
          <strong>{{ loading ? '正在读取最新资料…' : title }}</strong>
          <span v-if="!loading && identity" class="character-preview__identity">{{ identity }}</span>
          <p>{{ loading ? '请稍候，正在确认这份档案仍可访问。' : summary }}</p>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.character-preview {
  position: fixed;
  z-index: 2450;
  display: grid;
  width: min(390px, calc(100vw - 24px));
  min-height: 184px;
  grid-template-columns: 116px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid #C89669;
  border-radius: 8px;
  background: #FDFBF9;
  color: #2C1810;
  box-shadow: 0 18px 42px rgba(44, 24, 16, 0.27);
  pointer-events: auto;
}

.character-preview__portrait {
  position: relative;
  display: grid;
  min-width: 0;
  place-items: center;
  overflow: hidden;
  border-right: 1px solid #CFA57E;
  background:
    radial-gradient(circle, rgba(184, 115, 51, 0.2), transparent 48%),
    #302019;
  color: #C99462;
  font-size: 34px;
}

.character-preview__image {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}

.character-preview__spin { animation: character-preview-spin 900ms linear infinite; }
@keyframes character-preview-spin { to { transform: rotate(360deg); } }

.character-preview__rail {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 5px;
  background: linear-gradient(#804030, #D09660 50%, #804030);
}

.character-preview__body,
.character-preview__state {
  display: flex;
  min-width: 0;
  flex-direction: column;
  justify-content: center;
  padding: 16px 18px;
}

.character-preview__body.loading { opacity: 0.7; }

.character-preview__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: #9A5A2D;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.character-preview__meta span { display: inline-flex; align-items: center; gap: 4px; }
.character-preview__meta small { color: #9B8879; font-size: 8px; letter-spacing: .08em; }

.character-preview h3 {
  overflow: hidden;
  margin: 7px 0 2px;
  color: #2C1810;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 21px;
  font-weight: 600;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.character-preview__body > strong { overflow: hidden; color: #804030; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.character-preview__identity { margin-top: 3px; color: #8C7B70; font-size: 9px; }
.character-preview p { display: -webkit-box; overflow: hidden; margin: 9px 0 0; color: #8C7B70; font-size: 10px; line-height: 1.55; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }

.character-preview__state > span { color: #B87333; font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .14em; text-transform: uppercase; }
.character-preview__state h3 { font-size: 18px; }

.character-preview-enter-active,
.character-preview-leave-active { transition: opacity 120ms ease, transform 120ms ease; }
.character-preview-enter-from,
.character-preview-leave-to { opacity: 0; transform: translateY(3px); }

@media (max-width: 520px) {
  .character-preview { grid-template-columns: 92px minmax(0, 1fr); min-height: 164px; }
}

@media (prefers-reduced-motion: reduce) {
  .character-preview-enter-active,
  .character-preview-leave-active { transition: none; }
  .character-preview__spin { animation: none; }
}
</style>
