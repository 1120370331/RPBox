<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  listMyCharacterCards,
  listUserCharacterCards,
  type CharacterCardSummary,
} from '@/api/characterCard'
import { getCharacterCardDisplayName } from '@/utils/characterCardDraft'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import CharacterCardPortrait from './CharacterCardPortrait.vue'

const props = defineProps<{
  userId: number
  isOwnProfile: boolean
}>()

const cards = ref<CharacterCardSummary[]>([])
const loading = ref(false)
const errorMessage = ref('')
let requestToken = 0

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
    errorMessage.value = error instanceof Error ? error.message : '人物卡展示墙加载失败'
  } finally {
    if (token === requestToken) loading.value = false
  }
}

function displayName(card: CharacterCardSummary) {
  return getCharacterCardDisplayName(card)
}

function secondaryLine(card: CharacterCardSummary) {
  return card.title || card.full_title || [card.race, card.class].filter(Boolean).join(' · ') || '身份尚待记录'
}

function displayNameColor(card: CharacterCardSummary) {
  return getCharacterCardDisplayColor(card)
}

function statusLabel(card: CharacterCardSummary) {
  if (card.status === 'draft') return '草稿'
  if (card.visibility === 'private') return '仅自己可见'
  if (card.review_status === 'pending') return '审核中'
  if (card.review_status === 'rejected') return '未通过审核'
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
    <div class="character-wall__rail" aria-hidden="true">
      <span></span><span></span><span></span><span></span><span></span>
    </div>

    <header class="character-wall__header">
      <div>
        <span class="character-wall__kicker">Adventurer archive</span>
        <h2 id="character-wall-title">人物卡展示墙</h2>
        <p>{{ isOwnProfile ? '整理你的角色设定，并选择哪些档案向社区公开。' : '这位冒险者公开陈列的角色档案。' }}</p>
      </div>
      <RouterLink v-if="isOwnProfile && cards.length" class="character-wall__new" to="/character-cards/new">
        <i class="ri-add-line" aria-hidden="true"></i>
        新建人物卡
      </RouterLink>
    </header>

    <div v-if="loading" class="character-wall__state" role="status">
      <i class="ri-loader-4-line character-wall__spinner" aria-hidden="true"></i>
      <span>正在查阅人物档案…</span>
    </div>

    <div v-else-if="errorMessage" class="character-wall__state character-wall__state--error" role="alert">
      <i class="ri-file-warning-line" aria-hidden="true"></i>
      <div>
        <strong>暂时无法打开展示墙</strong>
        <span>{{ errorMessage }}</span>
      </div>
      <button type="button" @click="loadCards">重新加载</button>
    </div>

    <div v-else-if="isOwnProfile && cards.length === 0" class="character-wall__empty">
      <div class="character-wall__empty-mark" aria-hidden="true">
        <i class="ri-compass-3-line"></i>
        <span class="character-wall__empty-plus"><i class="ri-add-line"></i></span>
      </div>
      <div class="character-wall__empty-copy">
        <span>档案架仍是空的</span>
        <h3>为你的第一个角色留下正式档案</h3>
        <p>可以从已有 TRP3 云备份带入基础身份信息，也可以从一张空白人物卡开始创作。</p>
      </div>
      <RouterLink class="character-wall__primary" to="/character-cards/new">
        新建人物卡
        <i class="ri-arrow-right-line" aria-hidden="true"></i>
      </RouterLink>
    </div>

    <div v-else class="character-wall__grid">
      <RouterLink
        v-for="(card, index) in cards"
        :key="card.id"
        class="portrait-card"
        :to="`/character-cards/${card.id}`"
        :aria-label="`查看人物卡：${displayName(card)}`"
      >
        <span class="portrait-card__folio" aria-hidden="true">CC · {{ String(index + 1).padStart(2, '0') }}</span>
        <span class="portrait-card__frame">
          <CharacterCardPortrait
            v-if="card.portrait_image_url"
            class="portrait-card__image"
            :card="card"
            :alt="`${displayName(card)}的角色肖像`"
            :width="560"
            :quality="86"
          />
          <span v-else class="portrait-card__placeholder" aria-hidden="true">
            <i class="ri-user-star-line"></i>
            <small>肖像待归档</small>
          </span>
          <span class="portrait-card__shade"></span>
          <span v-if="isOwnProfile && statusLabel(card)" class="portrait-card__status">
            <i :class="statusIcon(card)" aria-hidden="true"></i>
            {{ statusLabel(card) }}
          </span>
          <span class="portrait-card__reveal">
            <span>{{ card.summary || '这份人物档案还没有写下摘要。' }}</span>
            <b>查阅完整档案 <i class="ri-arrow-right-line" aria-hidden="true"></i></b>
          </span>
        </span>
        <span class="portrait-card__plaque">
          <strong :style="displayNameColor(card) ? { color: displayNameColor(card) } : undefined">{{ displayName(card) }}</strong>
          <span>{{ secondaryLine(card) }}</span>
        </span>
      </RouterLink>

      <RouterLink v-if="isOwnProfile" class="portrait-card portrait-card--add" to="/character-cards/new">
        <span class="portrait-card__add-mark" aria-hidden="true"><i class="ri-add-line"></i></span>
        <strong>登记新人物</strong>
        <span>从备份或空白档案开始</span>
      </RouterLink>
    </div>
  </section>
</template>

<style scoped>
.character-wall {
  position: relative;
  grid-column: 1 / -1;
  min-width: 0;
  overflow: hidden;
  padding: 28px;
  border: 1px solid #E2D2C2;
  border-radius: 18px;
  background:
    linear-gradient(90deg, rgba(184, 115, 51, 0.035) 1px, transparent 1px) 0 0 / 42px 42px,
    linear-gradient(#FDFBF9, #FAF5EF);
  box-shadow: 0 10px 26px rgba(75, 54, 33, 0.07);
}

.character-wall__rail {
  position: absolute;
  top: 0;
  left: 28px;
  right: 28px;
  display: flex;
  height: 8px;
  align-items: center;
  justify-content: space-around;
  border-radius: 0 0 8px 8px;
  background: linear-gradient(90deg, #804030, #C88B51 45%, #804030);
  box-shadow: 0 3px 8px rgba(75, 54, 33, 0.18);
}

.character-wall__rail span {
  width: 3px;
  height: 12px;
  border-radius: 2px;
  background: #EED9C4;
  transform: translateY(4px);
}

.character-wall__header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;
}

.character-wall__kicker,
.portrait-card__folio {
  color: #9B6B43;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.character-wall h2 {
  margin: 4px 0 5px;
  color: #2C1810;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: clamp(24px, 3vw, 34px);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.character-wall__header p {
  margin: 0;
  color: #8C7B70;
  font-size: 13px;
}

.character-wall__new,
.character-wall__primary {
  display: inline-flex;
  min-height: 40px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 16px;
  border: 1px solid #804030;
  border-radius: 8px;
  background: #804030;
  color: #FFF8F1;
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
  transition: transform 160ms ease, box-shadow 160ms ease, background 160ms ease;
}

.character-wall__new:hover,
.character-wall__primary:hover {
  background: #6E3427;
  box-shadow: 0 7px 16px rgba(75, 54, 33, 0.18);
  transform: translateY(-1px);
}

.character-wall__new:focus-visible,
.character-wall__primary:focus-visible,
.portrait-card:focus-visible,
.character-wall__state button:focus-visible {
  outline: 3px solid rgba(184, 115, 51, 0.38);
  outline-offset: 3px;
}

.character-wall__state {
  display: flex;
  min-height: 180px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #8C7B70;
}

.character-wall__spinner {
  animation: archive-spin 900ms linear infinite;
}

.character-wall__state--error {
  flex-wrap: wrap;
  align-content: center;
}

.character-wall__state--error > i {
  color: #9A4D3C;
  font-size: 28px;
}

.character-wall__state--error div {
  display: grid;
  gap: 2px;
}

.character-wall__state--error strong {
  color: #4B3621;
}

.character-wall__state--error span {
  font-size: 12px;
}

.character-wall__state button {
  padding: 8px 12px;
  border: 1px solid #B87333;
  border-radius: 7px;
  background: transparent;
  color: #804030;
  cursor: pointer;
}

.character-wall__empty {
  display: grid;
  min-height: 230px;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 28px;
  padding: 28px clamp(24px, 5vw, 56px);
  border: 1px dashed #CDB49C;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.58);
}

.character-wall__empty-mark {
  position: relative;
  display: grid;
  width: 96px;
  height: 96px;
  place-items: center;
  border: 1px solid #D5C0AB;
  border-radius: 50%;
  color: #B87333;
  font-size: 48px;
}

.character-wall__empty-mark::before,
.character-wall__empty-mark::after {
  position: absolute;
  width: 118px;
  height: 1px;
  background: #DDCBB9;
  content: '';
}

.character-wall__empty-mark::after { transform: rotate(90deg); }

.character-wall__empty-plus {
  position: absolute;
  right: -2px;
  bottom: 4px;
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 3px solid #FDFBF9;
  border-radius: 50%;
  background: #804030;
  color: white;
  font-size: 16px;
}

.character-wall__empty-copy > span {
  color: #B87333;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.character-wall__empty-copy h3 {
  margin: 5px 0 8px;
  color: #2C1810;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 22px;
}

.character-wall__empty-copy p {
  max-width: 580px;
  margin: 0;
  color: #8C7B70;
  font-size: 13px;
  line-height: 1.75;
}

.character-wall__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 20px;
}

.portrait-card {
  position: relative;
  display: flex;
  min-width: 0;
  flex-direction: column;
  padding: 7px;
  border: 1px solid #BFA58B;
  border-radius: 7px;
  background: #37231A;
  box-shadow: 0 7px 17px rgba(44, 24, 16, 0.17);
  color: inherit;
  text-decoration: none;
  transition: transform 180ms ease, box-shadow 180ms ease, border-color 180ms ease;
}

.portrait-card:hover,
.portrait-card:focus-visible {
  border-color: #D3975D;
  box-shadow: 0 13px 27px rgba(44, 24, 16, 0.24), 0 0 0 2px rgba(184, 115, 51, 0.14);
  transform: translateY(-4px);
}

.portrait-card__folio {
  position: absolute;
  z-index: 4;
  top: 15px;
  left: 15px;
  color: rgba(255, 244, 231, 0.74);
}

.portrait-card__frame {
  position: relative;
  display: block;
  overflow: hidden;
  aspect-ratio: 3 / 4;
  border: 1px solid rgba(238, 217, 196, 0.25);
  border-radius: 3px 3px 0 0;
  background: #231711;
}

.portrait-card__image {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
  transition: transform 300ms ease, filter 300ms ease;
}

.portrait-card:hover .portrait-card__image,
.portrait-card:focus-visible .portrait-card__image {
  filter: saturate(0.92) contrast(1.03);
  transform: scale(1.025);
}

.portrait-card__placeholder {
  display: grid;
  width: 100%;
  height: 100%;
  place-content: center;
  gap: 10px;
  background:
    radial-gradient(circle, rgba(184, 115, 51, 0.17), transparent 48%),
    linear-gradient(145deg, #3D2A20, #1E1511);
  color: #C58D5D;
  text-align: center;
}

.portrait-card__placeholder i { font-size: 48px; }
.portrait-card__placeholder small { color: #CDB7A3; font-size: 11px; }

.portrait-card__shade {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(25, 13, 8, 0.92), transparent 52%);
  pointer-events: none;
}

.portrait-card__status {
  position: absolute;
  z-index: 3;
  top: 13px;
  right: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 7px;
  border: 1px solid rgba(255, 246, 235, 0.22);
  border-radius: 999px;
  background: rgba(35, 22, 15, 0.78);
  color: #F2DFC9;
  font-size: 10px;
  backdrop-filter: blur(6px);
}

.portrait-card__reveal {
  position: absolute;
  z-index: 2;
  right: 14px;
  bottom: 14px;
  left: 14px;
  display: grid;
  gap: 10px;
  color: rgba(255, 246, 235, 0.88);
  opacity: 0;
  transform: translateY(8px);
  transition: opacity 180ms ease, transform 180ms ease;
}

.portrait-card__reveal > span {
  display: -webkit-box;
  overflow: hidden;
  font-size: 12px;
  line-height: 1.55;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.portrait-card__reveal b {
  color: #F0BE86;
  font-size: 11px;
}

.portrait-card:hover .portrait-card__reveal,
.portrait-card:focus-visible .portrait-card__reveal {
  opacity: 1;
  transform: translateY(0);
}

.portrait-card__plaque {
  display: grid;
  min-height: 66px;
  place-content: center;
  gap: 3px;
  padding: 10px 12px;
  border-top: 1px solid #A56738;
  background: linear-gradient(90deg, #271812, #3B251A 50%, #271812);
  text-align: center;
}

.portrait-card__plaque strong {
  overflow: hidden;
  color: #F1D5B7;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 18px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.portrait-card__plaque span {
  overflow: hidden;
  color: #BCA895;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.portrait-card--add {
  min-height: 330px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-style: dashed;
  background: rgba(75, 54, 33, 0.035);
  box-shadow: none;
  color: #804030;
  text-align: center;
}

.portrait-card--add > span:last-child { color: #8C7B70; font-size: 11px; }

.portrait-card__add-mark {
  display: grid;
  width: 58px;
  height: 58px;
  margin-bottom: 6px;
  place-items: center;
  border: 1px solid #B87333;
  border-radius: 50%;
  font-size: 26px;
}

@keyframes archive-spin { to { transform: rotate(360deg); } }

@media (max-width: 760px) {
  .character-wall { padding: 24px 18px; }
  .character-wall__rail { right: 18px; left: 18px; }
  .character-wall__header { align-items: flex-start; flex-direction: column; }
  .character-wall__empty { grid-template-columns: 1fr; text-align: center; }
  .character-wall__empty-mark { margin: 0 auto; }
  .character-wall__primary { justify-self: center; }
  .character-wall__grid { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
  .portrait-card__plaque strong { font-size: 15px; }
}

@media (max-width: 460px) {
  .character-wall__grid { grid-template-columns: 1fr; }
  .portrait-card--add { min-height: 180px; }
}

@media (prefers-reduced-motion: reduce) {
  .portrait-card,
  .portrait-card__image,
  .portrait-card__reveal,
  .character-wall__new,
  .character-wall__primary { transition: none; }
  .character-wall__spinner { animation: none; }
}
</style>
