<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  deleteCharacterCard,
  getCharacterCard,
  getCharacterCardSharePath,
  updateCharacterCard,
  type CharacterCard,
  type CharacterCardPortraitImage,
} from '@/api/characterCard'
import { resolveApiUrl } from '@/api/item'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import ImageViewer from '@/components/ImageViewer.vue'
import CharacterCardImpressionMark from '@/components/character-cards/CharacterCardImpressionMark.vue'
import CharacterCardGalleryImage from '@/components/character-cards/CharacterCardGalleryImage.vue'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import { hydrateJumpCards, sanitizeJumpLinks } from '@/utils/jumpLink'
import { attachImagePreview } from '@/utils/imagePreview'
import {
  getCharacterCardDisplayName,
  normalizeCharacterCardImpressions,
  type CharacterCardEditorTab,
} from '@/utils/characterCardDraft'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import { getCharacterCardCoverPortrait, normalizeCharacterCardPortraits } from '@/utils/characterCardPortraits'
import { buildPublicSitePathUrl } from '@/utils/desktopDeepLink'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const dialog = useDialog()
const userStore = useUserStore()

const cardId = computed(() => Number(route.params.id))
const card = ref<CharacterCard | null>(null)
const loading = ref(true)
const actionLoading = ref(false)
const shareLoading = ref(false)
const errorMessage = ref('')
const activeTab = ref<CharacterCardEditorTab>('basic')
const backgroundRef = ref<HTMLElement | null>(null)
const impressionRef = ref<HTMLElement | null>(null)
const otherRef = ref<HTMLElement | null>(null)
const portraitFrameRef = ref<HTMLElement | null>(null)
const portraitFilmRef = ref<HTMLElement | null>(null)
const selectedPortraitId = ref<number | null>(null)
const showImageViewer = ref(false)
const viewerImages = ref<string[]>([])
const viewerStartIndex = ref(0)

const tabs = computed<Array<{ id: CharacterCardEditorTab; label: string }>>(() => [
  { id: 'basic', label: t('characterCards.tabs.basic') },
  { id: 'background', label: t('characterCards.tabs.background') },
  { id: 'impression', label: t('characterCards.tabs.impression') },
  { id: 'other', label: t('characterCards.tabs.other') },
])

const isOwner = computed(() => Boolean(card.value && userStore.user?.id === card.value.user_id))
const wantsPublic = computed(() => card.value?.status === 'published' && card.value?.visibility === 'public')
const isPublic = computed(() => wantsPublic.value && (!card.value?.review_status || card.value.review_status === 'approved'))
const canShare = computed(() => Boolean(
  card.value?.status === 'published'
  && card.value?.visibility === 'public'
  && card.value?.review_status === 'approved',
))
const shareUnavailableReason = computed(() => {
  if (!card.value || canShare.value) return ''
  if (card.value.status !== 'published') return t('characterCards.detail.shareDraftUnavailable')
  if (card.value.visibility !== 'public') return t('characterCards.detail.sharePrivateUnavailable')
  if (card.value.review_status === 'pending') return t('characterCards.detail.sharePendingUnavailable')
  if (card.value.review_status === 'rejected') return t('characterCards.detail.shareRejectedUnavailable')
  return t('characterCards.detail.shareReviewUnavailable')
})
const displayName = computed(() => card.value ? getCharacterCardDisplayName(card.value) : t('characterCards.detail.displayFallback'))
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
    [t('characterCards.detail.fields.fullTitle'), card.value.full_title],
    [t('characterCards.detail.fields.age'), card.value.age],
    [t('characterCards.detail.fields.height'), card.value.height],
    [t('characterCards.detail.fields.weight'), card.value.weight],
    [t('characterCards.detail.fields.eyes'), card.value.eye_color],
    [t('characterCards.detail.fields.birthplace'), card.value.birthplace],
    [t('characterCards.detail.fields.residence'), card.value.residence],
    [t('characterCards.detail.fields.relationship'), card.value.relationship_status],
  ].filter((entry) => Boolean(entry[1]))
})

function selectTab(tab: CharacterCardEditorTab) {
  activeTab.value = tab
}

function handleTabKeydown(event: KeyboardEvent, currentTab: CharacterCardEditorTab) {
  const currentIndex = tabs.value.findIndex((tab) => tab.id === currentTab)
  if (currentIndex < 0) return

  let nextIndex: number | null = null
  switch (event.key) {
    case 'ArrowRight':
    case 'ArrowDown':
      nextIndex = (currentIndex + 1) % tabs.value.length
      break
    case 'ArrowLeft':
    case 'ArrowUp':
      nextIndex = (currentIndex - 1 + tabs.value.length) % tabs.value.length
      break
    case 'Home':
      nextIndex = 0
      break
    case 'End':
      nextIndex = tabs.value.length - 1
      break
    default:
      return
  }

  event.preventDefault()
  const nextTab = tabs.value[nextIndex]
  activeTab.value = nextTab.id
  void nextTick(() => {
    document.getElementById(`character-detail-tab-${nextTab.id}`)?.focus()
  })
}

watch(cardId, () => void loadCard(), { immediate: true })

async function loadCard() {
  if (!Number.isFinite(cardId.value) || cardId.value <= 0) {
    errorMessage.value = t('characterCards.common.unavailable')
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
    errorMessage.value = t('characterCards.common.unavailable')
  } finally {
    loading.value = false
  }
  if (card.value) await hydrateRichContent()
}

async function hydrateRichContent() {
  await nextTick()
  for (const container of [backgroundRef.value, impressionRef.value, otherRef.value]) {
    attachImagePreview(container, openImageViewer, t('characterCards.detail.viewImage'))
    sanitizeJumpLinks(container)
    hydrateJumpCards(container)
  }
}

function imageSourceWithin(container: HTMLElement | null): string {
  const image = container?.querySelector<HTMLImageElement>('img')
  return image?.currentSrc || image?.src || image?.getAttribute('src') || ''
}

function openImageViewer(images: string[], index = 0) {
  const selectedSource = images[index] || ''
  const uniqueImages = images.filter((source, sourceIndex) => (
    Boolean(source) && images.indexOf(source) === sourceIndex
  ))
  if (!uniqueImages.length) return

  viewerImages.value = uniqueImages
  viewerStartIndex.value = Math.max(0, uniqueImages.indexOf(selectedSource))
  showImageViewer.value = true
}

function openPortraitViewer() {
  const frameSource = imageSourceWithin(portraitFrameRef.value)
  const buttons = Array.from(portraitFilmRef.value?.querySelectorAll<HTMLButtonElement>('button') || [])
  if (!buttons.length) {
    openImageViewer([frameSource])
    return
  }

  const activeIndex = buttons.findIndex((button) => button.classList.contains('active'))
  const sources = buttons.map((button) => imageSourceWithin(button))
  if (activeIndex >= 0 && frameSource) sources[activeIndex] = frameSource
  openImageViewer(sources, activeIndex >= 0 ? activeIndex : 0)
}

function openImageFromEvent(event: Event) {
  openImageViewer([imageSourceWithin(event.currentTarget as HTMLElement)])
}

async function togglePublicAccess() {
  if (!card.value || actionLoading.value) return
  actionLoading.value = true
  try {
    const next = wantsPublic.value
      ? { visibility: 'private' as const }
      : { status: 'published' as const, visibility: 'public' as const }
    card.value = await updateCharacterCard(card.value.id, next)
    toast.success(t(wantsPublic.value
      ? (card.value.review_status === 'pending' ? 'characterCards.detail.submittedReview' : 'characterCards.detail.publishedPublic')
      : 'characterCards.detail.madePrivate'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.detail.statusFailed'))
  } finally {
    actionLoading.value = false
  }
}

async function removeCard() {
  if (!card.value || actionLoading.value) return
  const confirmed = await dialog.confirm({
    title: t('characterCards.detail.deleteTitle'),
    message: t('characterCards.detail.deleteMessage', { name: displayName.value }),
    type: 'error',
    confirmText: t('characterCards.detail.deleteConfirm'),
  })
  if (!confirmed) return

  actionLoading.value = true
  try {
    await deleteCharacterCard(card.value.id)
    toast.success(t('characterCards.detail.deleted'))
    await router.replace(`/user/${userStore.user?.id || card.value.user_id}`)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.detail.deleteFailed'))
  } finally {
    actionLoading.value = false
  }
}

async function shareCard() {
  if (!card.value || !canShare.value || shareLoading.value) return
  shareLoading.value = true
  try {
    const sharePath = await getCharacterCardSharePath(card.value.id)
    const url = buildPublicSitePathUrl(sharePath)
    if (navigator.share) {
      await navigator.share({
        title: displayName.value,
        text: t('characterCards.detail.shareText', { name: displayName.value }),
        url,
      })
      toast.success(t('characterCards.detail.shareSuccess'))
      return
    }

    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = url
      textarea.setAttribute('readonly', 'true')
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      const copied = document.execCommand('copy')
      document.body.removeChild(textarea)
      if (!copied) throw new Error('Clipboard copy failed')
    }
    toast.success(t('characterCards.detail.shareCopied'))
  } catch (error: any) {
    if (error?.name === 'AbortError') return
    console.error('分享人物卡失败:', error)
    toast.error(t('characterCards.detail.shareFailed'))
  } finally {
    shareLoading.value = false
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
      <span>{{ t('characterCards.common.loading') }}</span>
    </div>

    <div v-else-if="errorMessage" class="detail-state detail-state--error" role="alert">
      <div class="unavailable-seal"><i class="ri-link-unlink-m" aria-hidden="true"></i></div>
      <h1>{{ t('characterCards.common.unavailable') }}</h1>
      <p>{{ t('characterCards.detail.unavailableBody') }}</p>
      <button type="button" @click="goBack"><i class="ri-arrow-left-line" aria-hidden="true"></i>{{ t('characterCards.common.back') }}</button>
    </div>

    <template v-else-if="card">
      <header class="detail-toolbar">
        <button type="button" class="detail-toolbar__back" @click="goBack">
          <i class="ri-arrow-left-line" aria-hidden="true"></i>{{ t('characterCards.detail.backWall') }}
        </button>
        <div v-if="isOwner || canShare" class="detail-toolbar__actions">
          <button v-if="canShare" type="button" class="toolbar-button toolbar-button--share" :disabled="shareLoading" @click="shareCard">
            <i :class="shareLoading ? 'ri-loader-4-line spin' : 'ri-share-forward-line'" aria-hidden="true"></i>
            {{ t('characterCards.detail.share') }}
          </button>
          <span v-else-if="isOwner" class="share-unavailable" :title="shareUnavailableReason">
            <i class="ri-information-line" aria-hidden="true"></i>{{ shareUnavailableReason }}
          </span>
          <template v-if="isOwner">
            <button type="button" class="toolbar-button" :disabled="actionLoading" @click="togglePublicAccess">
              <i :class="wantsPublic ? 'ri-lock-line' : 'ri-global-line'" aria-hidden="true"></i>
              {{ t(wantsPublic ? 'characterCards.detail.makePrivate' : 'characterCards.detail.publishPublic') }}
            </button>
            <RouterLink class="toolbar-button toolbar-button--primary" :to="`/character-cards/${card.id}/edit`">
              <i class="ri-edit-line" aria-hidden="true"></i>{{ t('characterCards.detail.edit') }}
            </RouterLink>
            <button type="button" class="toolbar-button toolbar-button--danger" :disabled="actionLoading" @click="removeCard">
              <i class="ri-delete-bin-line" aria-hidden="true"></i><span class="sr-only">{{ t('characterCards.detail.deleteTitle') }}</span>
            </button>
          </template>
        </div>
      </header>

      <div class="detail-shell">
        <aside class="character-portrait">
          <span class="character-portrait__index">RPBOX · CHARACTER {{ card.id }}</span>
          <div
            ref="portraitFrameRef"
            class="character-portrait__frame"
            :class="{ 'is-previewable': selectedPortrait }"
            :role="selectedPortrait ? 'button' : undefined"
            :tabindex="selectedPortrait ? 0 : undefined"
            :aria-label="selectedPortrait ? t('characterCards.detail.previewPortrait', { name: displayName }) : undefined"
            @click="openPortraitViewer"
            @keydown.enter.prevent="openPortraitViewer"
            @keydown.space.prevent="openPortraitViewer"
          >
            <CharacterCardGalleryImage
              v-if="selectedPortrait"
              class="character-portrait__image"
              :card="card"
              :portrait="selectedPortrait"
              :alt="t('characterCards.common.portraitAlt', { name: displayName })"
              :width="1000"
              :quality="90"
            />
            <div v-else class="character-portrait__empty" aria-hidden="true">
              <i class="ri-user-star-line"></i>
              <span>{{ t('characterCards.detail.portraitMissing') }}</span>
            </div>
            <span class="character-portrait__shade"></span>
            <span v-if="selectedPortrait" class="character-portrait__zoom" aria-hidden="true">
              <i class="ri-zoom-in-line"></i>
              {{ t('characterCards.detail.viewImage') }}
            </span>
            <span v-if="isOwner && (!isPublic || card.review_status === 'pending')" class="character-portrait__privacy" :class="{ pending: card.review_status === 'pending' && wantsPublic }">
              <i :class="card.review_status === 'pending' && wantsPublic ? 'ri-time-line' : card.status === 'draft' ? 'ri-draft-line' : 'ri-lock-line'" aria-hidden="true"></i>
              {{ t(card.review_status === 'pending' && wantsPublic ? 'characterCards.common.status.pending' : card.status === 'draft' ? 'characterCards.common.status.draft' : 'characterCards.common.status.private') }}
            </span>
          </div>
          <div v-if="portraits.length > 1" ref="portraitFilmRef" class="character-portrait__film" :aria-label="t('characterCards.detail.galleryAria')">
            <button
              v-for="(portrait, index) in portraits"
              :key="portrait.id"
              type="button"
              :class="{ active: selectedPortrait?.id === portrait.id }"
              :aria-label="t('characterCards.detail.galleryItemAria', { index: index + 1, cover: portrait.is_cover ? t('characterCards.detail.coverSuffix') : '' })"
              :aria-pressed="selectedPortrait?.id === portrait.id"
              @click="selectedPortraitId = portrait.id"
            >
              <CharacterCardGalleryImage :card="card" :portrait="portrait" alt="" :width="180" :quality="70" />
              <span v-if="portrait.is_cover">{{ t('characterCards.detail.coverShort') }}</span>
            </button>
          </div>
          <div class="character-portrait__plaque">
            <h1 :style="displayNameColor ? { color: displayNameColor } : undefined">{{ displayName }}</h1>
            <strong>{{ card.title || card.full_title || t('characterCards.detail.titleMissing') }}</strong>
            <span>{{ identityLine || t('characterCards.detail.identityMissing') }}</span>
          </div>
        </aside>

        <article class="character-file">
          <header class="character-file__header">
            <div>
              <span class="character-file__kicker">{{ t('characterCards.detail.kicker') }}</span>
              <h2 :style="displayNameColor ? { color: displayNameColor } : undefined">{{ displayName }}</h2>
              <p>{{ card.summary || t('characterCards.detail.summaryMissing') }}</p>
            </div>
            <span v-if="isPublic" class="public-mark"><i class="ri-global-line" aria-hidden="true"></i>{{ t('characterCards.detail.publicRecord') }}</span>
          </header>

          <section v-if="isOwner && wantsPublic && card.review_status && card.review_status !== 'approved'" class="review-banner" :class="`review-banner--${card.review_status}`">
            <i :class="card.review_status === 'rejected' ? 'ri-close-circle-line' : 'ri-time-line'" aria-hidden="true"></i>
            <div>
              <strong>{{ t(card.review_status === 'rejected' ? 'characterCards.detail.reviewRejected' : 'characterCards.detail.reviewPending') }}</strong>
              <span v-if="card.review_status === 'rejected' && card.review_comment">{{ t('characterCards.detail.moderatorComment', { comment: card.review_comment }) }}</span>
              <span v-else>{{ t('characterCards.detail.reviewBody') }}</span>
            </div>
          </section>

          <nav class="detail-tabs" role="tablist" :aria-label="t('characterCards.tabs.aria')">
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
              <div><span>{{ t('characterCards.detail.fields.firstName') }}</span><strong>{{ card.first_name || '—' }}</strong></div>
              <div><span>{{ t('characterCards.detail.fields.lastName') }}</span><strong>{{ card.last_name || '—' }}</strong></div>
              <div><span>{{ t('characterCards.detail.fields.race') }}</span><strong>{{ card.race || '—' }}</strong></div>
              <div><span>{{ t('characterCards.detail.fields.class') }}</span><strong>{{ card.class || '—' }}</strong></div>
            </div>

            <div v-if="basicFields.length" class="basic-ledger">
              <div v-for="entry in basicFields" :key="entry[0]">
                <span>{{ entry[0] }}</span><strong>{{ entry[1] }}</strong>
              </div>
            </div>
            <div v-else class="empty-chapter">
              <i class="ri-file-list-3-line" aria-hidden="true"></i>
              <span>{{ t('characterCards.detail.moreBasicMissing') }}</span>
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
            <div v-else class="empty-chapter"><i class="ri-book-open-line" aria-hidden="true"></i><span>{{ t('characterCards.detail.backgroundMissing') }}</span></div>
          </section>

          <section
            v-show="activeTab === 'impression'"
            id="character-detail-panel-impression"
            class="detail-panel"
            role="tabpanel"
            aria-labelledby="character-detail-tab-impression"
          >
            <template v-if="hasImpressionContent">
              <div v-if="activeImpressions.length" class="impression-readout" :aria-label="t('characterCards.detail.impressionAria')">
                <article
                  v-for="impression in activeImpressions"
                  :key="impression.slot"
                  class="impression-entry"
                  :class="{ 'impression-entry--with-image': impression.image_url }"
                >
                  <div
                    class="impression-entry__mark"
                    :class="{ 'is-previewable': impression.icon_image_url }"
                    :role="impression.icon_image_url ? 'button' : undefined"
                    :tabindex="impression.icon_image_url ? 0 : undefined"
                    :aria-label="impression.icon_image_url ? t('characterCards.detail.previewImpressionIcon', { title: impression.title || t('characterCards.common.impressionFallbackTitle', { slot: impression.slot }) }) : undefined"
                    @click="impression.icon_image_url && openImageFromEvent($event)"
                    @keydown.enter.prevent="impression.icon_image_url && openImageFromEvent($event)"
                    @keydown.space.prevent="impression.icon_image_url && openImageFromEvent($event)"
                  >
                    <CharacterCardImpressionMark
                      :icon-image-url="impression.icon_image_url"
                      :trp3-icon="impression.trp3_icon"
                      :fallback-label="String(impression.slot)"
                      :size="72"
                    />
                    <span v-if="impression.trp3_icon && !impression.icon_image_url" :title="impression.trp3_icon">
                      TRP3 · {{ impression.trp3_icon }}
                    </span>
                    <span v-else-if="!impression.icon_image_url">{{ t('characterCards.detail.defaultMark') }}</span>
                    <i v-if="impression.icon_image_url" class="impression-entry__mark-zoom ri-zoom-in-line" aria-hidden="true"></i>
                  </div>

                  <div class="impression-entry__copy">
                    <span class="impression-entry__index">{{ t('characterCards.detail.observation', { slot: String(impression.slot).padStart(2, '0') }) }}</span>
                    <h3>{{ impression.title || t('characterCards.detail.observationUnnamed') }}</h3>
                    <p>{{ impression.text || t('characterCards.detail.observationEmpty') }}</p>
                  </div>

                  <figure
                    v-if="impression.image_url"
                    class="impression-entry__image"
                    role="button"
                    tabindex="0"
                    :aria-label="t('characterCards.detail.previewImpressionImage', { title: impression.title || t('characterCards.common.impressionFallbackTitle', { slot: impression.slot }) })"
                    @click="openImageFromEvent"
                    @keydown.enter.prevent="openImageFromEvent"
                    @keydown.space.prevent="openImageFromEvent"
                  >
                    <AuthenticatedImage
                      class="impression-entry__protected-image"
                      :src="resolveApiUrl(impression.image_url)"
                      :alt="t('characterCards.common.impressionImageAlt', { title: impression.title || t('characterCards.common.impressionFallbackTitle', { slot: impression.slot }) })"
                    />
                    <span class="impression-entry__image-zoom" aria-hidden="true"><i class="ri-zoom-in-line"></i></span>
                  </figure>
                </article>
              </div>

              <section v-if="card.first_impression" class="impression-supplement">
                <header>
                  <span>{{ t('characterCards.detail.supplement') }}</span>
                  <h3>{{ t('characterCards.detail.otherNotes') }}</h3>
                </header>
                <div ref="impressionRef" class="character-rich" v-html="card.first_impression"></div>
              </section>
            </template>
            <div v-else class="empty-chapter"><i class="ri-eye-2-line" aria-hidden="true"></i><span>{{ t('characterCards.detail.impressionsMissing') }}</span></div>
          </section>

          <section
            v-show="activeTab === 'other'"
            id="character-detail-panel-other"
            class="detail-panel"
            role="tabpanel"
            aria-labelledby="character-detail-tab-other"
          >
            <div v-if="card.other_content" ref="otherRef" class="character-rich" v-html="card.other_content"></div>
            <div v-else class="empty-chapter"><i class="ri-archive-stack-line" aria-hidden="true"></i><span>{{ t('characterCards.detail.otherMissing') }}</span></div>
          </section>

        </article>
      </div>
    </template>

    <ImageViewer
      v-model="showImageViewer"
      :images="viewerImages"
      :start-index="viewerStartIndex"
    />
  </main>
</template>

<style scoped>
.detail-page {
  --ink: var(--color-text-main);
  --walnut: var(--color-primary);
  --copper: var(--color-accent);
  --rust: var(--color-secondary);
  --muted: var(--color-text-secondary);
  --line: var(--color-border);
  width: min(1240px, calc(100% - 40px));
  margin: 0 auto;
  padding: 26px 0 54px;
  color: var(--ink);
}

.detail-state { display: grid; min-height: 70vh; place-content: center; justify-items: center; gap: 12px; color: var(--muted); text-align: center; }
.detail-state > i { color: var(--copper); font-size: 38px; }
.detail-state h1 { margin: 5px 0 0; color: var(--ink); font-family: Georgia, 'Noto Serif SC', serif; font-size: 30px; }
.detail-state p { max-width: 430px; margin: 0; line-height: 1.65; }
.detail-state button { display: inline-flex; align-items: center; gap: 6px; margin-top: 8px; padding: 9px 15px; border: 1px solid var(--btn-primary-bg); border-radius: 7px; background: var(--btn-primary-bg); color: var(--btn-primary-text); cursor: pointer; }
.unavailable-seal { display: grid; width: 78px; height: 78px; place-items: center; border: 1px solid var(--color-border-hover); border-radius: 50%; color: var(--rust); font-size: 31px; outline: 1px dashed var(--color-border); outline-offset: 7px; }

.detail-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.detail-toolbar__back { display: inline-flex; align-items: center; gap: 7px; padding: 8px 0; border: 0; background: transparent; color: var(--muted); cursor: pointer; font: inherit; }
.detail-toolbar__actions { display: flex; align-items: center; gap: 7px; }
.toolbar-button { display: inline-flex; min-height: 38px; align-items: center; justify-content: center; gap: 6px; padding: 0 13px; border: 1px solid var(--btn-outline-border); border-radius: 7px; background: var(--color-panel-bg); color: var(--btn-outline-text); cursor: pointer; font: inherit; font-size: 11px; font-weight: 700; text-decoration: none; }
.toolbar-button--primary { border-color: var(--btn-primary-bg); background: var(--btn-primary-bg); color: var(--btn-primary-text); }
.toolbar-button--share { border-color: var(--color-border-hover); background: var(--tag-bg); color: var(--tag-text); }
.toolbar-button--danger { width: 38px; padding: 0; color: var(--btn-danger-bg); }
.share-unavailable { display: inline-flex; max-width: 270px; min-height: 38px; align-items: center; gap: 6px; padding: 0 10px; border: 1px dashed var(--color-border); border-radius: 7px; color: var(--color-text-secondary); font-size: 10px; line-height: 1.35; }
.share-unavailable i { flex: 0 0 auto; color: var(--color-warning-dark); font-size: 14px; }

.detail-shell { display: grid; grid-template-columns: minmax(280px, 350px) minmax(0, 1fr); gap: 22px; align-items: start; }

.character-portrait { position: sticky; top: 18px; min-width: 0; padding: 9px; border: 1px solid var(--gradient-border); border-radius: 8px; background: var(--gradient-end); box-shadow: 0 16px 36px color-mix(in srgb, var(--color-primary) 28%, transparent); }
.character-portrait__index { display: block; padding: 5px 8px 11px; color: var(--gradient-text-muted); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: 0.14em; }
.character-portrait__frame { position: relative; overflow: hidden; aspect-ratio: 3 / 4; border: 1px solid var(--gradient-border); }
.character-portrait__frame.is-previewable { cursor: zoom-in; }
.character-portrait__image { width: 100%; height: 100%; display: block; object-fit: cover; }
.character-portrait__empty { display: grid; width: 100%; height: 100%; place-content: center; gap: 10px; background: radial-gradient(circle, color-mix(in srgb, var(--color-accent) 22%, transparent), transparent 48%), var(--gradient-end); color: var(--gradient-text); text-align: center; }
.character-portrait__empty i { font-size: 56px; }
.character-portrait__empty span { color: var(--gradient-text-muted); font-size: 11px; }
.character-portrait__shade { position: absolute; inset: 0; background: linear-gradient(to top, color-mix(in srgb, var(--gradient-end) 58%, transparent), transparent 42%); pointer-events: none; }
.character-portrait__zoom { position: absolute; right: 10px; bottom: 10px; display: inline-flex; align-items: center; gap: 5px; padding: 6px 8px; border: 1px solid var(--gradient-border); border-radius: 999px; background: color-mix(in srgb, var(--gradient-end) 82%, transparent); color: var(--gradient-text); font-size: 9px; opacity: 0; pointer-events: none; transform: translateY(4px); transition: opacity .18s ease, transform .18s ease; backdrop-filter: blur(7px); }
.character-portrait__frame:hover .character-portrait__zoom,
.character-portrait__frame:focus-visible .character-portrait__zoom { opacity: 1; transform: translateY(0); }
.character-portrait__privacy { position: absolute; top: 10px; right: 10px; display: inline-flex; align-items: center; gap: 5px; padding: 5px 8px; border: 1px solid var(--gradient-border); border-radius: 999px; background: color-mix(in srgb, var(--gradient-end) 82%, transparent); color: var(--gradient-text); font-size: 10px; backdrop-filter: blur(7px); }
.character-portrait__privacy.pending { border-color: var(--color-warning-border); background: color-mix(in srgb, var(--color-warning) 30%, var(--gradient-end)); color: var(--gradient-text); }
.character-portrait__film { display: flex; gap: 7px; padding: 9px 9px 5px; overflow-x: auto; background: repeating-linear-gradient(90deg, var(--gradient-surface) 0 9px, transparent 9px 18px); }
.character-portrait__film button { position: relative; flex: 0 0 54px; height: 68px; overflow: hidden; padding: 0; border: 2px solid var(--gradient-border); border-radius: 4px; background: var(--gradient-end); cursor: pointer; }
.character-portrait__film button.active { border-color: var(--color-accent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 24%, transparent); }
.character-portrait__film button :deep(img), .character-portrait__film button :deep(.authenticated-image) { width: 100%; height: 100%; object-fit: cover; }
.character-portrait__film button > span { position: absolute; right: 2px; bottom: 2px; display: grid; width: 16px; height: 16px; place-items: center; border-radius: 50%; background: var(--color-accent); color: var(--color-accent-contrast); font-size: 8px; }
.character-portrait__plaque { display: grid; gap: 4px; padding: 16px 12px 13px; text-align: center; }
.character-portrait__plaque h1 { overflow: hidden; margin: 0; color: var(--gradient-text); font-family: Georgia, 'Noto Serif SC', serif; font-size: 25px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.character-portrait__plaque strong { color: var(--gradient-text); font-size: 12px; font-weight: 500; }
.character-portrait__plaque span { color: var(--gradient-text-muted); font-size: 10px; }

.character-file { min-width: 0; overflow: hidden; border: 1px solid var(--line); border-radius: 14px; background: var(--color-panel-bg); box-shadow: var(--shadow-md); }
.character-file__header { display: grid; grid-template-columns: minmax(0,1fr) auto; align-items: start; gap: 24px; padding: 30px; border-bottom: 1px solid var(--line); background: linear-gradient(135deg, var(--color-panel-bg), var(--color-card-bg)); }
.character-file__kicker { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .17em; text-transform: uppercase; }
.character-file__header h2 { margin: 5px 0 8px; font-family: Georgia, 'Noto Serif SC', serif; font-size: clamp(28px, 4vw, 42px); font-weight: 600; }
.character-file__header p { max-width: 720px; margin: 0; color: var(--muted); font-size: 13px; line-height: 1.75; }
.public-mark { display: inline-flex; align-items: center; gap: 5px; padding: 5px 8px; border: 1px solid var(--color-border-hover); border-radius: 999px; color: var(--rust); font-size: 10px; white-space: nowrap; }
.review-banner { display: flex; gap: 10px; align-items: flex-start; margin: 18px 30px 0; padding: 12px 14px; border: 1px solid var(--color-warning-border); border-radius: 9px; background: var(--color-warning-light); color: var(--color-warning-dark); }
.review-banner > i { margin-top: 1px; font-size: 20px; }
.review-banner > div { display: grid; gap: 3px; }
.review-banner strong { font-size: 12px; }
.review-banner span { font-size: 10px; line-height: 1.55; }
.review-banner--rejected { border-color: color-mix(in srgb, var(--btn-danger-bg) 42%, var(--color-border)); background: color-mix(in srgb, var(--btn-danger-bg) 12%, var(--color-panel-bg)); color: var(--btn-danger-bg); }

.detail-tabs { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); padding: 0 24px; border-bottom: 1px solid var(--line); background: var(--color-card-bg); }
.detail-tabs button { position: relative; min-height: 55px; border: 0; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 12px; font-weight: 700; }
.detail-tabs button::after { position: absolute; right: 23%; bottom: -1px; left: 23%; height: 3px; border-radius: 3px 3px 0 0; background: var(--copper); content: ''; opacity: 0; }
.detail-tabs button.active { color: var(--rust); }
.detail-tabs button.active::after { opacity: 1; }

.detail-panel { min-height: 470px; padding: 30px; }
.identity-summary { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); gap: 1px; overflow: hidden; margin-bottom: 24px; border: 1px solid var(--line); border-radius: 9px; background: var(--line); }
.identity-summary div { display: grid; gap: 5px; padding: 15px; background: var(--color-panel-bg); }
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
  background: linear-gradient(var(--copper), var(--color-border-hover) 50%, var(--copper));
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
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background:
    linear-gradient(105deg, color-mix(in srgb, var(--color-accent) 11%, transparent), transparent 34%),
    var(--color-card-bg);
  box-shadow: var(--shadow-sm);
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

.impression-entry__mark.is-previewable { cursor: zoom-in; }
.impression-entry__mark-zoom { position: absolute; top: -2px; right: 1px; display: grid; width: 21px; height: 21px; place-items: center; border: 1px solid var(--gradient-border); border-radius: 50%; background: var(--gradient-end); color: var(--gradient-text); font-size: 10px; opacity: 0; pointer-events: none; transition: opacity .18s ease; }
.impression-entry__mark.is-previewable:hover .impression-entry__mark-zoom,
.impression-entry__mark.is-previewable:focus-visible .impression-entry__mark-zoom { opacity: 1; }

.impression-entry__mark > span {
  overflow: hidden;
  width: 82px;
  color: var(--color-text-muted);
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
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.impression-entry__image {
  position: relative;
  align-self: stretch;
  min-height: 112px;
  overflow: hidden;
  margin: 0;
  border: 1px solid var(--color-border-hover);
  border-radius: 7px;
  background: var(--gradient-end);
  box-shadow: inset 0 0 0 3px color-mix(in srgb, var(--gradient-end) 82%, black);
  cursor: zoom-in;
}

.impression-entry__image-zoom { position: absolute; right: 8px; bottom: 8px; display: grid; width: 28px; height: 28px; place-items: center; border: 1px solid var(--gradient-border); border-radius: 50%; background: color-mix(in srgb, var(--gradient-end) 82%, transparent); color: var(--gradient-text); opacity: 0; pointer-events: none; transition: opacity .18s ease; backdrop-filter: blur(6px); }
.impression-entry__image:hover .impression-entry__image-zoom,
.impression-entry__image:focus-visible .impression-entry__image-zoom { opacity: 1; }

.impression-entry__protected-image {
  width: 100%;
  height: 100%;
  min-height: 112px;
  max-height: 180px;
  display: block;
  object-fit: cover;
  color: var(--gradient-text);
}

.impression-entry__protected-image :deep(.authenticated-image__state) { background: var(--gradient-end); }

.impression-supplement {
  margin-top: 28px;
  padding-top: 24px;
  border-top: 1px dashed var(--color-border-hover);
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

.character-rich { color: var(--color-text-main); font-size: 14px; line-height: 1.85; }
.character-rich :deep(h1), .character-rich :deep(h2), .character-rich :deep(h3) { color: var(--ink); font-family: Georgia, 'Noto Serif SC', serif; }
.character-rich :deep(img) { max-width: 100%; height: auto; border-radius: 8px; }
.character-rich :deep(.image-preview) { position: relative; display: inline-block; max-width: 100%; overflow: hidden; border-radius: 8px; cursor: zoom-in; }
.character-rich :deep(.image-preview img) { display: block; }
.character-rich :deep(.image-preview-overlay) { position: absolute; inset: auto 10px 10px auto; display: inline-flex; align-items: center; padding: 5px 8px; border: 1px solid var(--gradient-border); border-radius: 999px; background: color-mix(in srgb, var(--gradient-end) 82%, transparent); color: var(--gradient-text); font-size: 9px; opacity: 0; pointer-events: none; transform: translateY(4px); transition: opacity .18s ease, transform .18s ease; backdrop-filter: blur(6px); }
.character-rich :deep(.image-preview:hover .image-preview-overlay),
.character-rich :deep(.image-preview:focus-within .image-preview-overlay) { opacity: 1; transform: translateY(0); }
.character-rich :deep(a) { color: var(--rust); }
.empty-chapter { display: grid; min-height: 290px; place-content: center; justify-items: center; gap: 10px; color: var(--muted); }
.empty-chapter i { color: var(--color-accent); font-size: 36px; }
.empty-chapter span { font-size: 12px; }

.detail-toolbar button:focus-visible,
.toolbar-button:focus-visible,
.detail-tabs button:focus-visible,
.character-portrait__film button:focus-visible,
.character-portrait__frame:focus-visible,
.impression-entry__mark:focus-visible,
.impression-entry__image:focus-visible,
.detail-state button:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-accent) 32%, transparent); outline-offset: 2px; }
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
  .character-portrait__zoom,
  .impression-entry__mark-zoom,
  .impression-entry__image-zoom,
  .character-rich :deep(.image-preview-overlay) { transition: none; }
}
</style>
