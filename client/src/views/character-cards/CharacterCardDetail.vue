<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  deleteCharacterCard,
  getCharacterCard,
  updateCharacterCard,
  type CharacterCard,
  type CharacterCardPortraitImage,
} from '@/api/characterCard'
import { resolveApiUrl } from '@/api/item'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import CharacterCardImpressionMark from '@/components/character-cards/CharacterCardImpressionMark.vue'
import CharacterCardGalleryImage from '@/components/character-cards/CharacterCardGalleryImage.vue'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import { hydrateJumpCards, sanitizeJumpLinks } from '@/utils/jumpLink'
import {
  getCharacterCardDisplayName,
  normalizeCharacterCardImpressions,
  type CharacterCardEditorTab,
} from '@/utils/characterCardDraft'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import { getCharacterCardCoverPortrait, normalizeCharacterCardPortraits } from '@/utils/characterCardPortraits'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()
const dialog = useDialog()
const userStore = useUserStore()

const cardId = computed(() => Number(route.params.id))
const card = ref<CharacterCard | null>(null)
const loading = ref(true)
const actionLoading = ref(false)
const errorMessage = ref('')
const activeTab = ref<CharacterCardEditorTab>('basic')
const backgroundRef = ref<HTMLElement | null>(null)
const impressionRef = ref<HTMLElement | null>(null)
const otherRef = ref<HTMLElement | null>(null)
const selectedPortraitId = ref<number | null>(null)

const tabs: Array<{ id: CharacterCardEditorTab; label: string }> = [
  { id: 'basic', label: '基础信息' },
  { id: 'background', label: '背景故事' },
  { id: 'impression', label: '第一印象' },
  { id: 'other', label: '其他' },
]

const isOwner = computed(() => Boolean(card.value && userStore.user?.id === card.value.user_id))
const wantsPublic = computed(() => card.value?.status === 'published' && card.value?.visibility === 'public')
const isPublic = computed(() => wantsPublic.value && (!card.value?.review_status || card.value.review_status === 'approved'))
const displayName = computed(() => card.value ? getCharacterCardDisplayName(card.value) : '人物卡')
const displayNameColor = computed(() => getCharacterCardDisplayColor(card.value))
const identityLine = computed(() => [card.value?.race, card.value?.class].filter(Boolean).join(' · '))
const portraits = computed(() => normalizeCharacterCardPortraits(card.value))
const selectedPortrait = computed<CharacterCardPortraitImage | null>(() => (
  portraits.value.find((portrait) => portrait.id === selectedPortraitId.value)
  || getCharacterCardCoverPortrait(portraits.value)
))
const activeImpressions = computed(() => normalizeCharacterCardImpressions(card.value?.impressions).filter((item) => item.active))
const hasImpressionContent = computed(() => activeImpressions.value.length > 0 || Boolean(card.value?.first_impression))
const basicFields = computed(() => {
  if (!card.value) return []
  return [
    ['完整头衔', card.value.full_title],
    ['年龄', card.value.age],
    ['身高', card.value.height],
    ['体重', card.value.weight],
    ['眼睛', card.value.eye_color],
    ['出生地', card.value.birthplace],
    ['居所', card.value.residence],
    ['关系状态', card.value.relationship_status],
  ].filter((entry) => Boolean(entry[1]))
})

function selectTab(tab: CharacterCardEditorTab) {
  activeTab.value = tab
}

function handleTabKeydown(event: KeyboardEvent, currentTab: CharacterCardEditorTab) {
  const currentIndex = tabs.findIndex((tab) => tab.id === currentTab)
  if (currentIndex < 0) return

  let nextIndex: number | null = null
  switch (event.key) {
    case 'ArrowRight':
    case 'ArrowDown':
      nextIndex = (currentIndex + 1) % tabs.length
      break
    case 'ArrowLeft':
    case 'ArrowUp':
      nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
      break
    case 'Home':
      nextIndex = 0
      break
    case 'End':
      nextIndex = tabs.length - 1
      break
    default:
      return
  }

  event.preventDefault()
  const nextTab = tabs[nextIndex]
  activeTab.value = nextTab.id
  void nextTick(() => {
    document.getElementById(`character-detail-tab-${nextTab.id}`)?.focus()
  })
}

watch(cardId, () => void loadCard(), { immediate: true })

async function loadCard() {
  if (!Number.isFinite(cardId.value) || cardId.value <= 0) {
    errorMessage.value = '人物卡暂不可用'
    loading.value = false
    return
  }
  loading.value = true
  activeTab.value = 'basic'
  errorMessage.value = ''
  try {
    card.value = await getCharacterCard(cardId.value)
    selectedPortraitId.value = getCharacterCardCoverPortrait(normalizeCharacterCardPortraits(card.value))?.id ?? null
  } catch {
    card.value = null
    errorMessage.value = '人物卡暂不可用'
  } finally {
    loading.value = false
  }
  if (card.value) await hydrateRichContent()
}

async function hydrateRichContent() {
  await nextTick()
  for (const container of [backgroundRef.value, impressionRef.value, otherRef.value]) {
    sanitizeJumpLinks(container)
    hydrateJumpCards(container)
  }
}

async function togglePublicAccess() {
  if (!card.value || actionLoading.value) return
  actionLoading.value = true
  try {
    const next = wantsPublic.value
      ? { visibility: 'private' as const }
      : { status: 'published' as const, visibility: 'public' as const }
    card.value = await updateCharacterCard(card.value.id, next)
    toast.success(wantsPublic.value
      ? (card.value.review_status === 'pending' ? '人物卡已提交公开审核' : '人物卡已发布并公开')
      : '人物卡已设为仅自己可见')
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '人物卡状态更新失败')
  } finally {
    actionLoading.value = false
  }
}

async function removeCard() {
  if (!card.value || actionLoading.value) return
  const confirmed = await dialog.confirm({
    title: '删除人物卡',
    message: `确定删除「${displayName.value}」吗？帖子中的历史引用会显示为“人物卡暂不可用”，此操作无法撤销。`,
    type: 'error',
    confirmText: '删除人物卡',
  })
  if (!confirmed) return

  actionLoading.value = true
  try {
    await deleteCharacterCard(card.value.id)
    toast.success('人物卡已删除')
    await router.replace(`/user/${userStore.user?.id || card.value.user_id}`)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '人物卡删除失败')
  } finally {
    actionLoading.value = false
  }
}

function goBack() {
  if (card.value?.user_id) {
    void router.push(`/user/${card.value.user_id}`)
    return
  }
  router.back()
}
</script>

<template>
  <main class="detail-page">
    <div v-if="loading" class="detail-state" role="status">
      <i class="ri-loader-4-line spin" aria-hidden="true"></i>
      <span>正在查阅人物档案…</span>
    </div>

    <div v-else-if="errorMessage" class="detail-state detail-state--error" role="alert">
      <div class="unavailable-seal"><i class="ri-link-unlink-m" aria-hidden="true"></i></div>
      <h1>人物卡暂不可用</h1>
      <p>这份档案可能不存在、尚未公开，或你没有查看权限。</p>
      <button type="button" @click="goBack"><i class="ri-arrow-left-line" aria-hidden="true"></i>返回</button>
    </div>

    <template v-else-if="card">
      <header class="detail-toolbar">
        <button type="button" class="detail-toolbar__back" @click="goBack">
          <i class="ri-arrow-left-line" aria-hidden="true"></i>返回展示墙
        </button>
        <div v-if="isOwner" class="detail-toolbar__actions">
          <button type="button" class="toolbar-button" :disabled="actionLoading" @click="togglePublicAccess">
            <i :class="wantsPublic ? 'ri-lock-line' : 'ri-global-line'" aria-hidden="true"></i>
            {{ wantsPublic ? '设为仅自己可见' : '发布并公开' }}
          </button>
          <RouterLink class="toolbar-button toolbar-button--primary" :to="`/character-cards/${card.id}/edit`">
            <i class="ri-edit-line" aria-hidden="true"></i>编辑人物卡
          </RouterLink>
          <button type="button" class="toolbar-button toolbar-button--danger" :disabled="actionLoading" @click="removeCard">
            <i class="ri-delete-bin-line" aria-hidden="true"></i><span class="sr-only">删除人物卡</span>
          </button>
        </div>
      </header>

      <div class="detail-shell">
        <aside class="character-portrait">
          <span class="character-portrait__index">RPBOX · CHARACTER {{ card.id }}</span>
          <div class="character-portrait__frame">
            <CharacterCardGalleryImage
              v-if="selectedPortrait"
              class="character-portrait__image"
              :card="card"
              :portrait="selectedPortrait"
              :alt="`${displayName}的角色肖像`"
              :width="1000"
              :quality="90"
            />
            <div v-else class="character-portrait__empty" aria-hidden="true">
              <i class="ri-user-star-line"></i>
              <span>未收录角色肖像</span>
            </div>
            <span class="character-portrait__shade"></span>
            <span v-if="isOwner && (!isPublic || card.review_status === 'pending')" class="character-portrait__privacy" :class="{ pending: card.review_status === 'pending' && wantsPublic }">
              <i :class="card.review_status === 'pending' && wantsPublic ? 'ri-time-line' : card.status === 'draft' ? 'ri-draft-line' : 'ri-lock-line'" aria-hidden="true"></i>
              {{ card.review_status === 'pending' && wantsPublic ? '审核中' : card.status === 'draft' ? '草稿' : '仅自己可见' }}
            </span>
          </div>
          <div v-if="portraits.length > 1" class="character-portrait__film" aria-label="角色画廊">
            <button
              v-for="(portrait, index) in portraits"
              :key="portrait.id"
              type="button"
              :class="{ active: selectedPortrait?.id === portrait.id }"
              :aria-label="`查看角色大图 ${index + 1}${portrait.is_cover ? '，封面' : ''}`"
              :aria-pressed="selectedPortrait?.id === portrait.id"
              @click="selectedPortraitId = portrait.id"
            >
              <CharacterCardGalleryImage :card="card" :portrait="portrait" alt="" :width="180" :quality="70" />
              <span v-if="portrait.is_cover">封</span>
            </button>
          </div>
          <div class="character-portrait__plaque">
            <h1 :style="displayNameColor ? { color: displayNameColor } : undefined">{{ displayName }}</h1>
            <strong>{{ card.title || card.full_title || '无称号记录' }}</strong>
            <span>{{ identityLine || '身份尚待记录' }}</span>
          </div>
        </aside>

        <article class="character-file">
          <header class="character-file__header">
            <div>
              <span class="character-file__kicker">Adventurer dossier</span>
              <h2 :style="displayNameColor ? { color: displayNameColor } : undefined">{{ displayName }}</h2>
              <p>{{ card.summary || '这位角色尚未留下展示摘要。' }}</p>
            </div>
            <span v-if="isPublic" class="public-mark"><i class="ri-global-line" aria-hidden="true"></i>公开档案</span>
          </header>

          <section v-if="isOwner && wantsPublic && card.review_status && card.review_status !== 'approved'" class="review-banner" :class="`review-banner--${card.review_status}`">
            <i :class="card.review_status === 'rejected' ? 'ri-close-circle-line' : 'ri-time-line'" aria-hidden="true"></i>
            <div>
              <strong>{{ card.review_status === 'rejected' ? '公开版本未通过审核' : '公开版本正在审核' }}</strong>
              <span v-if="card.review_status === 'rejected' && card.review_comment">版主说明：{{ card.review_comment }}</span>
              <span v-else>你可以继续编辑工作副本；访客在审核完成前不会看到这次修改。</span>
            </div>
          </section>

          <nav class="detail-tabs" role="tablist" aria-label="人物卡资料分栏">
            <button
              v-for="tab in tabs"
              :id="`character-detail-tab-${tab.id}`"
              :key="tab.id"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.id"
              :aria-controls="`character-detail-panel-${tab.id}`"
              :tabindex="activeTab === tab.id ? 0 : -1"
              :class="{ active: activeTab === tab.id }"
              @click="selectTab(tab.id)"
              @keydown="handleTabKeydown($event, tab.id)"
            >{{ tab.label }}</button>
          </nav>

          <section
            v-show="activeTab === 'basic'"
            id="character-detail-panel-basic"
            class="detail-panel"
            role="tabpanel"
            aria-labelledby="character-detail-tab-basic"
          >
            <div class="identity-summary">
              <div><span>名</span><strong>{{ card.first_name || '—' }}</strong></div>
              <div><span>姓</span><strong>{{ card.last_name || '—' }}</strong></div>
              <div><span>种族</span><strong>{{ card.race || '—' }}</strong></div>
              <div><span>职业</span><strong>{{ card.class || '—' }}</strong></div>
            </div>

            <div v-if="basicFields.length" class="basic-ledger">
              <div v-for="entry in basicFields" :key="entry[0]">
                <span>{{ entry[0] }}</span><strong>{{ entry[1] }}</strong>
              </div>
            </div>
            <div v-else class="empty-chapter">
              <i class="ri-file-list-3-line" aria-hidden="true"></i>
              <span>更多基础资料尚未记录。</span>
            </div>
          </section>

          <section
            v-show="activeTab === 'background'"
            id="character-detail-panel-background"
            class="detail-panel"
            role="tabpanel"
            aria-labelledby="character-detail-tab-background"
          >
            <div v-if="card.background_story" ref="backgroundRef" class="character-rich" v-html="card.background_story"></div>
            <div v-else class="empty-chapter"><i class="ri-book-open-line" aria-hidden="true"></i><span>背景故事尚未写下。</span></div>
          </section>

          <section
            v-show="activeTab === 'impression'"
            id="character-detail-panel-impression"
            class="detail-panel"
            role="tabpanel"
            aria-labelledby="character-detail-tab-impression"
          >
            <template v-if="hasImpressionContent">
              <div v-if="activeImpressions.length" class="impression-readout" aria-label="已启用的第一印象">
                <article
                  v-for="impression in activeImpressions"
                  :key="impression.slot"
                  class="impression-entry"
                  :class="{ 'impression-entry--with-image': impression.image_url }"
                >
                  <div class="impression-entry__mark">
                    <CharacterCardImpressionMark
                      :icon-image-url="impression.icon_image_url"
                      :trp3-icon="impression.trp3_icon"
                      :fallback-label="String(impression.slot)"
                      :size="72"
                    />
                    <span v-if="impression.trp3_icon && !impression.icon_image_url" :title="impression.trp3_icon">
                      TRP3 · {{ impression.trp3_icon }}
                    </span>
                    <span v-else-if="!impression.icon_image_url">默认观察印记</span>
                  </div>

                  <div class="impression-entry__copy">
                    <span class="impression-entry__index">Observation {{ String(impression.slot).padStart(2, '0') }}</span>
                    <h3>{{ impression.title || '未命名观察' }}</h3>
                    <p>{{ impression.text || '这条观察只留下了一枚印记。' }}</p>
                  </div>

                  <figure v-if="impression.image_url" class="impression-entry__image">
                    <AuthenticatedImage
                      class="impression-entry__protected-image"
                      :src="resolveApiUrl(impression.image_url)"
                      :alt="`${impression.title || `第 ${impression.slot} 条印象`}的配图`"
                    />
                  </figure>
                </article>
              </div>

              <section v-if="card.first_impression" class="impression-supplement">
                <header>
                  <span>Supplement</span>
                  <h3>其他备注</h3>
                </header>
                <div ref="impressionRef" class="character-rich" v-html="card.first_impression"></div>
              </section>
            </template>
            <div v-else class="empty-chapter"><i class="ri-eye-2-line" aria-hidden="true"></i><span>尚未启用第一印象观察记录。</span></div>
          </section>

          <section
            v-show="activeTab === 'other'"
            id="character-detail-panel-other"
            class="detail-panel"
            role="tabpanel"
            aria-labelledby="character-detail-tab-other"
          >
            <div v-if="card.other_content" ref="otherRef" class="character-rich" v-html="card.other_content"></div>
            <div v-else class="empty-chapter"><i class="ri-archive-stack-line" aria-hidden="true"></i><span>其他资料尚未写下。</span></div>
          </section>

          <footer v-if="isOwner && card.source_profile_id" class="source-footnote">
            <i class="ri-link-m" aria-hidden="true"></i>
            基础身份来源：{{ card.source_account_id || '云备份' }} · {{ card.source_profile_id }}。关联不代表自动同步。
          </footer>
        </article>
      </div>
    </template>
  </main>
</template>

<style scoped>
.detail-page {
  --ink: #2C1810;
  --walnut: #4B3621;
  --copper: #B87333;
  --rust: #804030;
  --muted: #8C7B70;
  --line: #E2D1C0;
  width: min(1240px, calc(100% - 40px));
  margin: 0 auto;
  padding: 26px 0 54px;
  color: var(--ink);
}

.detail-state { display: grid; min-height: 70vh; place-content: center; justify-items: center; gap: 12px; color: var(--muted); text-align: center; }
.detail-state > i { color: var(--copper); font-size: 38px; }
.detail-state h1 { margin: 5px 0 0; color: var(--ink); font-family: Georgia, 'Noto Serif SC', serif; font-size: 30px; }
.detail-state p { max-width: 430px; margin: 0; line-height: 1.65; }
.detail-state button { display: inline-flex; align-items: center; gap: 6px; margin-top: 8px; padding: 9px 15px; border: 1px solid var(--rust); border-radius: 7px; background: var(--rust); color: #FFF; cursor: pointer; }
.unavailable-seal { display: grid; width: 78px; height: 78px; place-items: center; border: 1px solid #C89B70; border-radius: 50%; color: var(--rust); font-size: 31px; outline: 1px dashed #D9C2AA; outline-offset: 7px; }

.detail-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.detail-toolbar__back { display: inline-flex; align-items: center; gap: 7px; padding: 8px 0; border: 0; background: transparent; color: var(--muted); cursor: pointer; font: inherit; }
.detail-toolbar__actions { display: flex; align-items: center; gap: 7px; }
.toolbar-button { display: inline-flex; min-height: 38px; align-items: center; justify-content: center; gap: 6px; padding: 0 13px; border: 1px solid var(--line); border-radius: 7px; background: #FFF; color: var(--walnut); cursor: pointer; font: inherit; font-size: 11px; font-weight: 700; text-decoration: none; }
.toolbar-button--primary { border-color: var(--rust); background: var(--rust); color: #FFF8F1; }
.toolbar-button--danger { width: 38px; padding: 0; color: #A24737; }

.detail-shell { display: grid; grid-template-columns: minmax(280px, 350px) minmax(0, 1fr); gap: 22px; align-items: start; }

.character-portrait { position: sticky; top: 18px; min-width: 0; padding: 9px; border: 1px solid #B79370; border-radius: 8px; background: #342219; box-shadow: 0 16px 36px rgba(44, 24, 16, 0.25); }
.character-portrait__index { display: block; padding: 5px 8px 11px; color: #C8AA8D; font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: 0.14em; }
.character-portrait__frame { position: relative; overflow: hidden; aspect-ratio: 3 / 4; border: 1px solid rgba(238, 217, 196, 0.28); }
.character-portrait__image { width: 100%; height: 100%; display: block; object-fit: cover; }
.character-portrait__empty { display: grid; width: 100%; height: 100%; place-content: center; gap: 10px; background: radial-gradient(circle, rgba(184, 115, 51, 0.18), transparent 48%), #241711; color: #C99B70; text-align: center; }
.character-portrait__empty i { font-size: 56px; }
.character-portrait__empty span { color: #B6A08E; font-size: 11px; }
.character-portrait__shade { position: absolute; inset: 0; background: linear-gradient(to top, rgba(26, 14, 9, 0.56), transparent 42%); pointer-events: none; }
.character-portrait__privacy { position: absolute; top: 10px; right: 10px; display: inline-flex; align-items: center; gap: 5px; padding: 5px 8px; border: 1px solid rgba(255,255,255,.16); border-radius: 999px; background: rgba(35, 21, 14, .8); color: #F0DCC7; font-size: 10px; backdrop-filter: blur(7px); }
.character-portrait__privacy.pending { border-color: rgba(229,180,127,.5); background: rgba(113,69,30,.9); color: #FFE2B6; }
.character-portrait__film { display: flex; gap: 7px; padding: 9px 9px 5px; overflow-x: auto; background: repeating-linear-gradient(90deg, rgba(255,255,255,.04) 0 9px, transparent 9px 18px); }
.character-portrait__film button { position: relative; flex: 0 0 54px; height: 68px; overflow: hidden; padding: 0; border: 2px solid #573927; border-radius: 4px; background: #23160F; cursor: pointer; }
.character-portrait__film button.active { border-color: #D9A66F; box-shadow: 0 0 0 2px rgba(217,166,111,.2); }
.character-portrait__film button :deep(img), .character-portrait__film button :deep(.authenticated-image) { width: 100%; height: 100%; object-fit: cover; }
.character-portrait__film button > span { position: absolute; right: 2px; bottom: 2px; display: grid; width: 16px; height: 16px; place-items: center; border-radius: 50%; background: #B87333; color: #FFF; font-size: 8px; }
.character-portrait__plaque { display: grid; gap: 4px; padding: 16px 12px 13px; text-align: center; }
.character-portrait__plaque h1 { overflow: hidden; margin: 0; color: #F1D5B7; font-family: Georgia, 'Noto Serif SC', serif; font-size: 25px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.character-portrait__plaque strong { color: #D2B08F; font-size: 12px; font-weight: 500; }
.character-portrait__plaque span { color: #AFA092; font-size: 10px; }

.character-file { min-width: 0; overflow: hidden; border: 1px solid var(--line); border-radius: 14px; background: #FDFBF9; box-shadow: 0 11px 30px rgba(75,54,33,.08); }
.character-file__header { display: grid; grid-template-columns: minmax(0,1fr) auto; align-items: start; gap: 24px; padding: 30px; border-bottom: 1px solid var(--line); background: linear-gradient(135deg, #FFFEFC, #F8F0E8); }
.character-file__kicker { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .17em; text-transform: uppercase; }
.character-file__header h2 { margin: 5px 0 8px; font-family: Georgia, 'Noto Serif SC', serif; font-size: clamp(28px, 4vw, 42px); font-weight: 600; }
.character-file__header p { max-width: 720px; margin: 0; color: var(--muted); font-size: 13px; line-height: 1.75; }
.public-mark { display: inline-flex; align-items: center; gap: 5px; padding: 5px 8px; border: 1px solid #CDA67F; border-radius: 999px; color: var(--rust); font-size: 10px; white-space: nowrap; }
.review-banner { display: flex; gap: 10px; align-items: flex-start; margin: 18px 30px 0; padding: 12px 14px; border: 1px solid #D9B57F; border-radius: 9px; background: #FFF8E8; color: #765126; }
.review-banner > i { margin-top: 1px; font-size: 20px; }
.review-banner > div { display: grid; gap: 3px; }
.review-banner strong { font-size: 12px; }
.review-banner span { font-size: 10px; line-height: 1.55; }
.review-banner--rejected { border-color: #D8A39A; background: #FFF0ED; color: #8C3D33; }

.detail-tabs { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); padding: 0 24px; border-bottom: 1px solid var(--line); background: #FAF5EF; }
.detail-tabs button { position: relative; min-height: 55px; border: 0; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 12px; font-weight: 700; }
.detail-tabs button::after { position: absolute; right: 23%; bottom: -1px; left: 23%; height: 3px; border-radius: 3px 3px 0 0; background: var(--copper); content: ''; opacity: 0; }
.detail-tabs button.active { color: var(--rust); }
.detail-tabs button.active::after { opacity: 1; }

.detail-panel { min-height: 470px; padding: 30px; }
.identity-summary { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); gap: 1px; overflow: hidden; margin-bottom: 24px; border: 1px solid var(--line); border-radius: 9px; background: var(--line); }
.identity-summary div { display: grid; gap: 5px; padding: 15px; background: #FFF; }
.identity-summary span,
.basic-ledger span { color: var(--muted); font-size: 9px; font-weight: 700; letter-spacing: .08em; }
.identity-summary strong { overflow: hidden; color: var(--walnut); font-family: Georgia, 'Noto Serif SC', serif; font-size: 17px; text-overflow: ellipsis; white-space: nowrap; }
.basic-ledger { display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); border-top: 1px solid var(--line); }
.basic-ledger div { display: grid; grid-template-columns: minmax(80px,.7fr) minmax(0,1fr); gap: 16px; padding: 13px 5px; border-bottom: 1px solid var(--line); }
.basic-ledger div:nth-child(odd) { margin-right: 18px; }
.basic-ledger div:nth-child(even) { margin-left: 18px; }
.basic-ledger strong { overflow-wrap: anywhere; color: var(--walnut); font-size: 12px; }

.impression-readout {
  position: relative;
  display: grid;
  gap: 14px;
}

.impression-readout::before {
  position: absolute;
  top: 34px;
  bottom: 34px;
  left: 37px;
  width: 1px;
  background: linear-gradient(var(--copper), #d6b18d 50%, var(--copper));
  content: '';
}

.impression-entry {
  position: relative;
  display: grid;
  grid-template-columns: 86px minmax(0, 1fr);
  gap: 17px;
  align-items: center;
  overflow: hidden;
  padding: 15px;
  border: 1px solid #dbc8b5;
  border-radius: 10px;
  background:
    linear-gradient(105deg, rgba(184, 115, 51, 0.09), transparent 34%),
    #fffdf9;
  box-shadow: 0 5px 14px rgba(75, 54, 33, 0.06);
}

.impression-entry--with-image { grid-template-columns: 86px minmax(160px, 1fr) minmax(150px, 210px); }

.impression-entry__mark {
  position: relative;
  z-index: 1;
  display: grid;
  min-width: 0;
  justify-items: center;
  gap: 7px;
}

.impression-entry__mark > span {
  overflow: hidden;
  width: 82px;
  color: #9b7e67;
  font: 700 7px/1.25 ui-monospace, Consolas, monospace;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.impression-entry__copy {
  min-width: 0;
  padding: 4px 0;
}

.impression-entry__index {
  color: var(--copper);
  font: 800 8px/1.2 ui-monospace, Consolas, monospace;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.impression-entry__copy h3 {
  margin: 5px 0 7px;
  color: var(--ink);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 19px;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.impression-entry__copy p {
  margin: 0;
  color: #665348;
  font-size: 12px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.impression-entry__image {
  align-self: stretch;
  min-height: 112px;
  overflow: hidden;
  margin: 0;
  border: 1px solid #b88c65;
  border-radius: 7px;
  background: #2a1a13;
  box-shadow: inset 0 0 0 3px #1f130e;
}

.impression-entry__protected-image {
  width: 100%;
  height: 100%;
  min-height: 112px;
  max-height: 180px;
  display: block;
  object-fit: cover;
  color: #d8b28d;
}

.impression-entry__protected-image :deep(.authenticated-image__state) { background: #2a1a13; }

.impression-supplement {
  margin-top: 28px;
  padding-top: 24px;
  border-top: 1px dashed #cdb49b;
}

.impression-supplement > header {
  margin-bottom: 14px;
}

.impression-supplement > header span {
  color: var(--copper);
  font: 800 8px/1.2 ui-monospace, Consolas, monospace;
  letter-spacing: 0.15em;
  text-transform: uppercase;
}

.impression-supplement > header h3 {
  margin: 4px 0 0;
  color: var(--walnut);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 19px;
  font-weight: 600;
}

.character-rich { color: #4B3A30; font-size: 14px; line-height: 1.85; }
.character-rich :deep(h1), .character-rich :deep(h2), .character-rich :deep(h3) { color: var(--ink); font-family: Georgia, 'Noto Serif SC', serif; }
.character-rich :deep(img) { max-width: 100%; height: auto; border-radius: 8px; }
.character-rich :deep(a) { color: var(--rust); }
.empty-chapter { display: grid; min-height: 290px; place-content: center; justify-items: center; gap: 10px; color: var(--muted); }
.empty-chapter i { color: #C69A70; font-size: 36px; }
.empty-chapter span { font-size: 12px; }

.source-footnote { display: flex; align-items: center; gap: 7px; padding: 12px 30px; border-top: 1px solid var(--line); background: #FAF6F1; color: var(--muted); font-size: 10px; }
.source-footnote i { color: var(--copper); }

.detail-toolbar button:focus-visible,
.toolbar-button:focus-visible,
.detail-tabs button:focus-visible,
.character-portrait__film button:focus-visible,
.detail-state button:focus-visible { outline: 3px solid rgba(184,115,51,.3); outline-offset: 2px; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0,0,0,0); clip-path: inset(50%); white-space: nowrap; }
.spin { animation: detail-spin 900ms linear infinite; }
@keyframes detail-spin { to { transform: rotate(360deg); } }

@media (max-width: 840px) {
  .detail-shell { grid-template-columns: 1fr; }
  .character-portrait { position: relative; top: auto; width: min(360px, 100%); box-sizing: border-box; margin: 0 auto; }
}

@media (max-width: 600px) {
  .detail-page { width: min(100% - 20px, 1240px); padding-top: 16px; }
  .detail-toolbar { align-items: flex-start; flex-direction: column; }
  .detail-toolbar__actions { width: 100%; flex-wrap: wrap; }
  .toolbar-button { flex: 1; }
  .toolbar-button--danger { flex: 0 0 38px; }
  .character-file__header { grid-template-columns: 1fr; padding: 22px 18px; }
  .review-banner { margin-right: 18px; margin-left: 18px; }
  .public-mark { justify-self: start; }
  .detail-tabs { padding: 0; overflow-x: auto; }
  .detail-tabs button { min-width: 104px; }
  .detail-panel { padding: 20px 16px; }
  .impression-entry,
  .impression-entry--with-image { grid-template-columns: 72px minmax(0, 1fr); gap: 12px; padding: 12px; }
  .impression-entry__image { grid-column: 1 / -1; min-height: 150px; }
  .impression-entry__protected-image { min-height: 150px; max-height: 230px; }
  .impression-entry__mark > span { width: 72px; }
  .impression-readout::before { left: 48px; }
  .identity-summary { grid-template-columns: repeat(2, minmax(0,1fr)); }
  .basic-ledger { grid-template-columns: 1fr; }
  .basic-ledger div:nth-child(n) { margin: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .spin { animation: none; }
}
</style>
