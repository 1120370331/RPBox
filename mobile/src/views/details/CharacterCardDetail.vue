<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  getCharacterCard,
  getCharacterCardShare,
  updateCharacterCard,
  type CharacterCard,
  type CharacterCardPersonalityTrait,
  type CharacterCardTRP3Color,
} from '@/api/characterCard'
import { resolveApiUrl } from '@/api/image'
import { createContentReport } from '@/api/safety'
import CachedImage from '@/components/CachedImage.vue'
import ImagePreviewDialog from '@/components/ImagePreviewDialog.vue'
import SafetyReportSheet from '@/components/SafetyReportSheet.vue'
import { handleJumpLinkClick, sanitizeJumpLinks } from '@/utils/jumpLink'
import { shareRouteLink } from '@/utils/mobileShare'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'

type DetailTab = 'basic' | 'background' | 'impression' | 'other'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const card = ref<CharacterCard | null>(null)
const loading = ref(true)
const loadFailed = ref(false)
const sharing = ref(false)
const safetySheetOpen = ref(false)
const safetySubmitting = ref(false)
const makePrivateDialogOpen = ref(false)
const makingPrivate = ref(false)
const activeTab = ref<DetailTab>('basic')
const selectedPortraitIndex = ref(0)
const previewOpen = ref(false)
const previewSrc = ref('')
const previewAlt = ref('')
const backgroundRef = ref<HTMLElement | null>(null)
const impressionRef = ref<HTMLElement | null>(null)
const otherRef = ref<HTMLElement | null>(null)

const cardId = computed(() => Number(route.params.id))
const isOwner = computed(() => {
  const currentUserId = Number(userStore.user?.id)
  const ownerId = Number(card.value?.user_id)
  return Number.isInteger(currentUserId) && currentUserId > 0 && currentUserId === ownerId
})
const canShare = computed(() => card.value?.status === 'published'
  && card.value.visibility === 'public'
  && card.value.review_status === 'approved')
const canMakePrivate = computed(() => isOwner.value
  && card.value?.status === 'published'
  && card.value.visibility === 'public')
const canUseSafetyAction = computed(() => {
  const currentUserId = userStore.user?.id
  const ownerId = card.value?.user_id
  return Boolean(currentUserId && ownerId && !isOwner.value)
})
const ownerStatus = computed(() => {
  const loadedCard = card.value
  if (!loadedCard) return 'draft'
  if (loadedCard.status === 'draft') return 'draft'
  if (loadedCard.review_status === 'rejected') return 'rejected'
  if (loadedCard.review_status === 'pending') return 'pending'
  if (loadedCard.visibility === 'private') return 'private'
  if (loadedCard.status === 'published'
    && loadedCard.visibility === 'public'
    && loadedCard.review_status === 'approved') return 'published'
  return 'unsubmitted'
})
const statusMark = computed(() => {
  if (!isOwner.value) {
    return {
      className: 'published',
      icon: 'ri-shield-check-line',
      label: t('characterCards.detail.publicMark'),
    }
  }
  const iconByStatus = {
    draft: 'ri-draft-line',
    private: 'ri-lock-2-line',
    pending: 'ri-time-line',
    rejected: 'ri-close-circle-line',
    published: 'ri-shield-check-line',
    unsubmitted: 'ri-send-plane-line',
  } as const
  return {
    className: ownerStatus.value,
    icon: iconByStatus[ownerStatus.value],
    label: t(`characterCards.detail.ownerStatus.${ownerStatus.value}`),
  }
})
const displayName = computed(() => {
  if (!card.value) return ''
  return card.value.display_name.trim()
    || [card.value.first_name, card.value.last_name].filter(Boolean).join(' ')
    || `#${card.value.id}`
})
const safetyTargetLabel = computed(() => card.value
  ? t('characterCards.detail.safety.targetLabel', { id: card.value.id, name: displayName.value })
  : '')
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
const additionalInfoEntries = computed(() => {
  if (!Array.isArray(card.value?.additional_info)) return []
  return card.value.additional_info.flatMap((item, index) => {
    const name = normalizePlainText(item?.name)
    const value = normalizePlainText(item?.value)
    if (!name || !value) return []
    return [{
      key: `${Number.isInteger(item.id) ? item.id : 0}-${index}`,
      name,
      value,
      iconUrl: resolveTRP3IconUrl(item.icon),
    }]
  })
})
const personalityTraitEntries = computed(() => {
  if (!Array.isArray(card.value?.personality_traits)) return []
  return card.value.personality_traits.flatMap((trait, index) => {
    const normalized = normalizePersonalityTrait(trait, index)
    return normalized ? [normalized] : []
  })
})
const backgroundHtml = computed(() => normalizeRichContent(card.value?.background_story || ''))
const impressionHtml = computed(() => normalizeRichContent(card.value?.first_impression || ''))
const otherHtml = computed(() => normalizeRichContent(card.value?.other_content || ''))

function normalizeColor(value: string) {
  const normalized = value.trim().replace(/^#/, '')
  return /^(?:[\da-f]{6}|[\da-f]{8})$/i.test(normalized) ? `#${normalized}` : ''
}

function normalizePlainText(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function normalizeTRP3IconName(value: unknown) {
  const trimmed = normalizePlainText(value)
  if (!trimmed || trimmed.length > 256 || /^(?:https?|data|blob|file|javascript):/i.test(trimmed)) return ''
  const textureMatch = trimmed.match(/\|T([^|]+)\|t/i)
  const source = textureMatch ? textureMatch[1].split(':')[0] : trimmed
  if (/^(?:https?|data|blob|file|javascript):/i.test(source)) return ''
  let name = source.replace(/\\/g, '/')
  if (name.toLowerCase().startsWith('interface/icons/')) name = name.slice('interface/icons/'.length)
  name = (name.split('/').pop() || name).replace(/\.(?:blp|tga|png|jpe?g)$/i, '').toLowerCase().trim()
  return name.length <= 128 && /^[a-z0-9_-]+$/.test(name) ? name : ''
}

function resolveTRP3IconUrl(value: unknown) {
  const iconName = normalizeTRP3IconName(value)
  return iconName ? resolveApiUrl(`/api/v1/icons/${iconName}`) : ''
}

function normalizeTraitColor(color: CharacterCardTRP3Color | null | undefined) {
  if (!color) return ''
  const components = [Number(color.r), Number(color.g), Number(color.b)]
  if (components.some(component => !Number.isFinite(component))) return ''
  const [r, g, b] = components.map(component => Math.round(Math.min(1, Math.max(0, component)) * 255))
  return `rgb(${r}, ${g}, ${b})`
}

function normalizeTraitValue(value: unknown) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return 10
  return Math.min(20, Math.max(0, numeric))
}

function normalizePersonalityTrait(trait: CharacterCardPersonalityTrait | null | undefined, index: number) {
  if (!trait) return null
  const presetId = Number.isInteger(trait.preset_id) && Number(trait.preset_id) >= 1 && Number(trait.preset_id) <= 11
    ? Number(trait.preset_id)
    : null
  const explicitLeft = normalizePlainText(trait.left_text)
  const explicitRight = normalizePlainText(trait.right_text)
  if (!presetId && !explicitLeft && !explicitRight) return null

  const presetPath = presetId
    ? `characterCards.detail.personalityPresets.${presetId}`
    : ''
  const left = explicitLeft
    || (presetPath ? t(`${presetPath}.left`) : t('characterCards.detail.personalityLeftFallback'))
  const right = explicitRight
    || (presetPath ? t(`${presetPath}.right`) : t('characterCards.detail.personalityRightFallback'))
  const value = normalizeTraitValue(trait.value)

  return {
    key: `${presetId || 'custom'}-${index}`,
    left,
    right,
    leftIconUrl: resolveTRP3IconUrl(trait.left_icon),
    rightIconUrl: resolveTRP3IconUrl(trait.right_icon),
    leftColor: normalizeTraitColor(trait.left_color),
    rightColor: normalizeTraitColor(trait.right_color),
    value,
  }
}

function getTraitTrackStyle(trait: { leftColor: string; rightColor: string }) {
  return {
    '--trait-left-color': trait.leftColor || 'var(--color-primary-light)',
    '--trait-right-color': trait.rightColor || 'var(--color-accent)',
  }
}

function getTraitMarkerStyle(value: number) {
  const percent = value * 5
  const transform = percent <= 0
    ? 'translateX(0)'
    : (percent >= 100 ? 'translateX(-100%)' : 'translateX(-50%)')
  return { left: `${percent}%`, transform }
}

function handleTRP3IconError(event: Event) {
  const image = event.currentTarget
  if (image instanceof HTMLImageElement) image.hidden = true
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
  card.value = null
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

async function shareCard() {
  const loadedCard = card.value
  if (!loadedCard || !canShare.value || sharing.value) return

  sharing.value = true
  try {
    const share = await getCharacterCardShare(loadedCard.id)
    if (card.value?.id !== loadedCard.id) return
    await shareRouteLink({
      path: share.path,
      title: share.title,
      text: share.summary,
      dialogTitle: t('characterCards.detail.shareDialogTitle'),
    })
    toast.success(t('characterCards.detail.shareSuccess'))
  } catch (error) {
    console.error('Failed to share public character card', error)
    toast.error(t('characterCards.detail.shareFailed'))
  } finally {
    sharing.value = false
  }
}

function openMakePrivateDialog() {
  if (!canMakePrivate.value || makingPrivate.value) return
  makePrivateDialogOpen.value = true
}

function closeMakePrivateDialog() {
  if (makingPrivate.value) return
  makePrivateDialogOpen.value = false
}

async function confirmMakePrivate() {
  const loadedCard = card.value
  if (!loadedCard || !canMakePrivate.value || makingPrivate.value) return
  makingPrivate.value = true
  try {
    const updated = await updateCharacterCard(loadedCard.id, { visibility: 'private' })
    if (card.value?.id !== loadedCard.id) return
    card.value = updated
    makePrivateDialogOpen.value = false
    toast.success(t('characterCards.detail.makePrivateSuccess'))
  } catch (error) {
    console.error('Failed to make character card private', error)
    toast.error((error as Error)?.message || t('characterCards.detail.makePrivateFailed'))
  } finally {
    makingPrivate.value = false
  }
}

function openSafetySheet() {
  if (!canUseSafetyAction.value || safetySubmitting.value) return
  safetySheetOpen.value = true
}

function closeSafetySheet() {
  if (safetySubmitting.value) return
  safetySheetOpen.value = false
}

async function submitCharacterCardSafety(payload: {
  reason: string
  detail: string
  hideTarget: boolean
  blockAuthor: boolean
  submitReport: boolean
}) {
  if (!card.value || !canUseSafetyAction.value || safetySubmitting.value) return

  const currentCard = card.value
  const currentUserId = userStore.user?.id
  if (!currentUserId || currentCard.user_id === currentUserId) return

  const leavesDetail = payload.hideTarget || payload.blockAuthor
  safetySubmitting.value = true
  try {
    await createContentReport({
      target_type: 'character_card',
      target_id: currentCard.id,
      reason: payload.reason,
      detail: payload.detail,
      hide_target: payload.hideTarget,
      block_author: payload.blockAuthor,
      submit_report: payload.submitReport,
    })
  } catch (error) {
    toast.error((error as Error)?.message || t('characterCards.detail.safety.failed'))
    return
  } finally {
    safetySubmitting.value = false
  }

  safetySheetOpen.value = false
  const successKey = payload.blockAuthor
    ? 'characterCards.detail.safety.blockedSuccess'
    : (payload.hideTarget
      ? 'characterCards.detail.safety.hiddenSuccess'
      : 'characterCards.detail.safety.reportSuccess')
  toast.success(t(successKey))
  if (leavesDetail) {
    card.value = null
    loadFailed.value = true
    void router.replace({ name: 'community' })
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
  previewSrc.value = /^(?:blob:|data:)/i.test(src) ? src : resolveApiUrl(src)
  previewAlt.value = alt
  previewOpen.value = true
}

function openRenderedImagePreview(event: MouseEvent, fallbackSrc: string, alt = '') {
  const trigger = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
  const renderedImage = trigger?.querySelector<HTMLImageElement>('img')
  openPreview(renderedImage?.currentSrc || renderedImage?.src || fallbackSrc, alt)
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
      <div class="character-card-heading">
        <span>RPBOX · CHARACTER</span>
        <h1>{{ isOwner ? t('characterCards.detail.ownerTitle') : t('characterCards.detail.title') }}</h1>
      </div>
      <button
        v-if="canShare"
        type="button"
        class="character-share-btn"
        :disabled="sharing"
        :aria-label="sharing ? t('characterCards.detail.sharing') : t('characterCards.detail.share')"
        :aria-busy="sharing"
        @click="shareCard"
      >
        <i :class="sharing ? 'ri-loader-4-line spin' : 'ri-share-forward-line'" aria-hidden="true" />
      </button>
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
            @click="openRenderedImagePreview($event, selectedPortrait.src, t('characterCards.detail.portraitAlt', { name: displayName }))"
          >
            <CachedImage
              class="portrait-stage-image"
              :src="selectedPortrait.src"
              :alt="t('characterCards.detail.portraitAlt', { name: displayName })"
              :auth-fetch="isOwner"
              loading="eager"
            />
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
              <CachedImage :src="portrait.src" alt="" :auth-fetch="isOwner" />
            </button>
          </div>

          <div class="character-plaque">
            <span class="public-mark" :class="`public-mark--${statusMark.className}`">
              <i :class="statusMark.icon" aria-hidden="true" />{{ statusMark.label }}
            </span>
            <h2 :style="displayColor ? { color: displayColor } : undefined">{{ displayName }}</h2>
            <strong>{{ card.title || card.full_title || t('characterCards.detail.titleMissing') }}</strong>
            <p>{{ identityLine || t('characterCards.detail.identityMissing') }}</p>
          </div>
        </section>

        <p class="character-summary">{{ card.summary || t('characterCards.detail.summaryMissing') }}</p>

        <section v-if="isOwner" class="owner-card-action" data-testid="character-card-owner-tools">
          <div>
            <span>{{ t(`characterCards.detail.ownerStatus.${ownerStatus}`) }}</span>
            <p>{{ t(`characterCards.detail.ownerStatusBody.${ownerStatus}`) }}</p>
          </div>
          <div class="owner-card-buttons">
            <button
              type="button"
              data-testid="character-card-owner-edit"
              @click="router.push({ name: 'character-card-edit', params: { id: card.id } })"
            >
              <i class="ri-quill-pen-line" aria-hidden="true" />
              {{ t('characterCards.detail.edit') }}
            </button>
            <button
              v-if="canMakePrivate"
              type="button"
              class="make-private-button"
              data-testid="character-card-owner-make-private"
              @click="openMakePrivateDialog"
            >
              <i class="ri-lock-2-line" aria-hidden="true" />
              {{ t('characterCards.detail.makePrivate') }}
            </button>
          </div>
        </section>

        <button
          v-if="canUseSafetyAction"
          type="button"
          class="character-safety-action"
          data-testid="character-card-safety-open"
          :disabled="safetySubmitting"
          @click="openSafetySheet"
        >
          <i class="ri-alarm-warning-line" aria-hidden="true" />
          <span>
            <strong>{{ t('characterCards.detail.safety.action') }}</strong>
            <small>{{ t('characterCards.detail.safety.description') }}</small>
          </span>
          <i class="ri-arrow-right-s-line" aria-hidden="true" />
        </button>

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

          <section v-if="additionalInfoEntries.length" class="basic-supplement additional-info-section">
            <h3>{{ t('characterCards.detail.additionalInfoTitle') }}</h3>
            <div class="additional-info-list">
              <div v-for="entry in additionalInfoEntries" :key="entry.key" class="additional-info-row">
                <span class="trp3-field-icon" aria-hidden="true">
                  <i class="ri-information-line" />
                  <img v-if="entry.iconUrl" :src="entry.iconUrl" alt="" loading="lazy" @error="handleTRP3IconError" />
                </span>
                <div>
                  <small>{{ entry.name }}</small>
                  <strong>{{ entry.value }}</strong>
                </div>
              </div>
            </div>
          </section>

          <section v-if="personalityTraitEntries.length" class="basic-supplement personality-section">
            <h3>{{ t('characterCards.detail.personalityTitle') }}</h3>
            <div class="personality-list">
              <article v-for="trait in personalityTraitEntries" :key="trait.key" class="personality-trait">
                <div class="personality-endpoints">
                  <span :style="trait.leftColor ? { color: trait.leftColor } : undefined">
                    <span class="trait-endpoint-icon" aria-hidden="true">
                      <i class="ri-circle-line" />
                      <img v-if="trait.leftIconUrl" :src="trait.leftIconUrl" alt="" loading="lazy" @error="handleTRP3IconError" />
                    </span>
                    <b>{{ trait.left }}</b>
                  </span>
                  <small>{{ t('characterCards.detail.personalityValueShort', { value: trait.value }) }}</small>
                  <span :style="trait.rightColor ? { color: trait.rightColor } : undefined">
                    <b>{{ trait.right }}</b>
                    <span class="trait-endpoint-icon" aria-hidden="true">
                      <i class="ri-circle-line" />
                      <img v-if="trait.rightIconUrl" :src="trait.rightIconUrl" alt="" loading="lazy" @error="handleTRP3IconError" />
                    </span>
                  </span>
                </div>
                <div
                  class="personality-track"
                  role="meter"
                  aria-valuemin="0"
                  aria-valuemax="20"
                  :aria-valuenow="trait.value"
                  :aria-label="t('characterCards.detail.personalityAria', { left: trait.left, right: trait.right })"
                  :aria-valuetext="t('characterCards.detail.personalityValue', { value: trait.value, left: trait.left, right: trait.right })"
                  :style="getTraitTrackStyle(trait)"
                >
                  <span class="personality-track__center" aria-hidden="true" />
                  <span
                    class="personality-track__marker"
                    data-testid="personality-trait-marker"
                    :data-trait-value="trait.value"
                    :style="getTraitMarkerStyle(trait.value)"
                    aria-hidden="true"
                  />
                </div>
              </article>
            </div>
          </section>
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
              @click="openRenderedImagePreview($event, impression.icon_image_url, impression.title)"
            >
              <CachedImage
                :src="resolveApiUrl(impression.icon_image_url)"
                alt=""
                :auth-fetch="isOwner"
              />
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
              @click="openRenderedImagePreview($event, impression.image_url, impression.title)"
            >
              <CachedImage
                class="impression-cached-image"
                :src="resolveApiUrl(impression.image_url)"
                :alt="impression.title"
                :auth-fetch="isOwner"
              />
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

    <div
      v-if="makePrivateDialogOpen && card"
      class="owner-dialog-mask"
      data-testid="character-card-make-private-dialog"
      @click.self="closeMakePrivateDialog"
    >
      <section
        class="owner-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="character-card-make-private-title"
      >
        <span>VISIBILITY · CONTROL</span>
        <h2 id="character-card-make-private-title">{{ t('characterCards.detail.makePrivateTitle') }}</h2>
        <p>{{ t('characterCards.detail.makePrivateMessage', { name: displayName }) }}</p>
        <div>
          <button type="button" :disabled="makingPrivate" @click="closeMakePrivateDialog">
            {{ t('characterCards.common.cancel') }}
          </button>
          <button
            type="button"
            class="owner-dialog-confirm"
            data-testid="character-card-make-private-confirm"
            :disabled="makingPrivate"
            @click="confirmMakePrivate"
          >
            {{ makingPrivate ? t('characterCards.detail.makingPrivate') : t('characterCards.detail.makePrivateConfirm') }}
          </button>
        </div>
      </section>
    </div>

    <ImagePreviewDialog
      :open="previewOpen"
      :src="previewSrc"
      :alt="previewAlt"
      @close="previewOpen = false"
    />
    <SafetyReportSheet
      :open="safetySheetOpen"
      :submitting="safetySubmitting"
      :title="t('characterCards.detail.safety.sheetTitle')"
      :target-label="safetyTargetLabel"
      target-type="character_card"
      initial-action="report"
      @close="closeSafetySheet"
      @submit="submitCharacterCardSafety"
    />
  </div>
</template>

<style scoped>
.character-card-page {
  color: var(--color-text-main);
}

.character-card-heading {
  flex: 1;
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

.character-share-btn {
  display: grid;
  flex: 0 0 44px;
  width: 44px;
  height: 44px;
  place-items: center;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-card-bg);
  color: var(--color-primary);
  font: inherit;
  font-size: 20px;
  touch-action: manipulation;
}

.character-share-btn:disabled {
  cursor: wait;
  opacity: 0.65;
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
  min-height: 44px;
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

.portrait-stage-image {
  width: 100%;
  height: 100%;
  display: block;
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
  width: 100%;
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

.portrait-film :deep(.cached-image) {
  width: 100%;
  height: 100%;
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

.public-mark--draft,
.public-mark--private {
  background: var(--color-primary-light);
  color: var(--color-text-secondary);
}

.public-mark--pending,
.public-mark--unsubmitted {
  background: var(--tag-bg);
  color: var(--color-accent);
}

.public-mark--rejected {
  background: var(--color-primary-light);
  color: var(--btn-danger-bg);
}

.public-mark--published {
  background: var(--color-success-light);
  color: var(--color-success);
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

.owner-card-action {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--color-border);
  border-left: 3px solid var(--color-accent);
  border-radius: var(--radius-sm);
  background: var(--color-card-bg);
}

.owner-card-action > div {
  min-width: 0;
}

.owner-card-action span {
  color: var(--color-accent);
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.owner-card-action p {
  margin-top: 4px;
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.45;
}

.owner-card-action button {
  min-height: 44px;
  padding: 0 12px;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: var(--text-light);
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font: inherit;
  font-size: 12px;
  font-weight: 700;
}

.owner-card-buttons {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.owner-card-action button.make-private-button {
  border-color: var(--color-border);
  background: var(--input-bg);
  color: var(--color-secondary);
}

.owner-dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  padding: 16px;
  background: var(--overlay-bg);
  display: flex;
  align-items: center;
  justify-content: center;
}

.owner-dialog {
  width: 100%;
  max-width: 380px;
  padding: 18px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  box-shadow: var(--shadow-md);
}

.owner-dialog > span {
  color: var(--color-accent);
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.1em;
}

.owner-dialog h2 {
  margin-top: 6px;
  color: var(--color-text-main);
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 20px;
}

.owner-dialog p {
  margin-top: 10px;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.owner-dialog > div {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
}

.owner-dialog button {
  min-height: 44px;
  padding: 0 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--color-primary);
  font: inherit;
  font-size: 12px;
  font-weight: 700;
}

.owner-dialog button.owner-dialog-confirm {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}

.owner-dialog button:disabled {
  opacity: 0.55;
}

.character-safety-action {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) 24px;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 58px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--color-warning) 38%, var(--color-border));
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--color-warning-light) 72%, var(--color-card-bg));
  color: var(--color-text-main);
  font: inherit;
  text-align: left;
  touch-action: manipulation;
}

.character-safety-action > i:first-child {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: 10px;
  background: var(--color-warning-light);
  color: var(--color-warning-dark);
  font-size: 19px;
}

.character-safety-action > span {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.character-safety-action strong,
.character-safety-action small {
  overflow-wrap: anywhere;
}

.character-safety-action strong {
  font-size: 13px;
}

.character-safety-action small {
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.4;
}

.character-safety-action > i:last-child {
  color: var(--color-text-secondary);
  font-size: 20px;
}

.character-safety-action:disabled {
  opacity: 0.6;
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

.basic-supplement {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border-light);
}

.basic-supplement > h3 {
  margin: 0 0 11px;
  color: var(--color-text-main);
  font-size: 14px;
}

.additional-info-list,
.personality-list {
  display: grid;
  gap: 9px;
}

.additional-info-row {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 11px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
}

.trp3-field-icon,
.trait-endpoint-icon {
  position: relative;
  display: grid;
  place-items: center;
  overflow: hidden;
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.trp3-field-icon {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  font-size: 18px;
}

.trp3-field-icon img,
.trait-endpoint-icon img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.additional-info-row > div {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.additional-info-row small {
  color: var(--color-text-secondary);
  font-size: 10px;
}

.additional-info-row strong {
  overflow-wrap: anywhere;
  color: var(--color-text-main);
  font-size: 13px;
}

.personality-trait {
  display: grid;
  gap: 9px;
  min-width: 0;
  padding: 11px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
}

.personality-endpoints {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: center;
  gap: 7px;
}

.personality-endpoints > span {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.personality-endpoints > span:last-child {
  justify-content: flex-end;
  text-align: right;
}

.personality-endpoints b {
  overflow: hidden;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.personality-endpoints > small {
  padding: 3px 6px;
  border-radius: 999px;
  background: var(--color-card-bg);
  color: var(--color-text-secondary);
  font: 700 9px/1 ui-monospace, SFMono-Regular, Consolas, monospace;
  white-space: nowrap;
}

.trait-endpoint-icon {
  flex: 0 0 24px;
  width: 24px;
  height: 24px;
  border-radius: 7px;
  font-size: 12px;
}

.personality-track {
  position: relative;
  height: 14px;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--color-border) 72%, transparent);
  border-radius: 999px;
  background: linear-gradient(90deg, var(--trait-left-color), var(--trait-right-color));
}

.personality-track__center {
  position: absolute;
  top: 2px;
  bottom: 2px;
  left: 50%;
  width: 1px;
  background: color-mix(in srgb, var(--color-text-main) 28%, transparent);
}

.personality-track__marker {
  position: absolute;
  top: 1px;
  width: 10px;
  height: 10px;
  border: 2px solid var(--color-card-bg);
  border-radius: 50%;
  background: var(--color-text-main);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--color-text-main) 28%, transparent);
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

.impression-mark :deep(.cached-image) {
  width: 100%;
  height: 100%;
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

.impression-cached-image {
  display: block;
  width: 100%;
  height: 230px;
  max-height: 230px;
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
