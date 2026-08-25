<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import {
  addCharacterCardPortrait,
  deleteCharacterCardPortrait,
  getCharacterCard,
  publishCharacterCard,
  setCharacterCardPortraitCover,
  updateCharacterCard,
  uploadCharacterCardPortrait,
  type CharacterCard,
  type CharacterCardAdditionalInfo,
  type CharacterCardPersonalityTrait,
  type CharacterCardPortrait,
  type UpdateCharacterCardRequest,
} from '@/api/characterCard'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'
import MobileRichEditor from '@/components/MobileRichEditor.vue'
import NativeImageSourceDialog from '@/components/NativeImageSourceDialog.vue'
import {
  canUseNativeImagePicker,
  pickSingleNativeImageFile,
  type NativeImageSource,
} from '@/utils/nativeImagePicker'
import {
  createCharacterCardDraft,
  createEmptyCharacterCardDraft,
  getCharacterCardDisplayName,
  type CharacterCardDraft,
  type CharacterCardEditorTab,
} from '@/utils/characterCardDraft'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const cardId = computed(() => Number(route.params.id))
const card = ref<CharacterCard | null>(null)
const form = reactive<CharacterCardDraft>(createEmptyCharacterCardDraft())
const portraits = ref<CharacterCardPortrait[]>([])
const activeTab = ref<CharacterCardEditorTab>('basic')
const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const publishing = ref(false)
const portraitUploading = ref(false)
const showImageSourceDialog = ref(false)
const portraitInput = ref<HTMLInputElement | null>(null)
const originalSnapshot = ref('')
const showLeaveDialog = ref(false)
const portraitToDelete = ref<CharacterCardPortrait | null>(null)

const useNativeImagePicker = canUseNativeImagePicker()

type CharacterCardSavePayload = Omit<UpdateCharacterCardRequest, 'portrait_image_url'>
type CharacterCardEditorStatus = 'draft' | 'private' | 'pending' | 'rejected' | 'published' | 'unsubmitted'

const tabs = computed<Array<{ id: CharacterCardEditorTab; label: string; icon: string }>>(() => [
  { id: 'basic', label: t('characterCards.tabs.basic'), icon: 'ri-id-card-line' },
  { id: 'traits', label: t('characterCards.tabs.traits'), icon: 'ri-scales-3-line' },
  { id: 'background', label: t('characterCards.tabs.background'), icon: 'ri-book-open-line' },
  { id: 'impression', label: t('characterCards.tabs.impression'), icon: 'ri-eye-2-line' },
  { id: 'other', label: t('characterCards.tabs.other'), icon: 'ri-archive-stack-line' },
])

const additionalInfoTypes = computed(() => Array.from({ length: 11 }, (_, index) => ({
  id: index + 1,
  label: t(`characterCards.editor.additionalInfoTypes.${index + 1}`),
})))

const personalityPresets = computed(() => Array.from({ length: 11 }, (_, index) => ({
  id: index + 1,
  label: t(`characterCards.editor.personalityPresets.${index + 1}`),
})))

const displayName = computed(() => getCharacterCardDisplayName(
  form,
  t('characterCards.common.unnamed'),
))
const formSnapshot = computed(() => JSON.stringify(form))
const isDirty = computed(() => originalSnapshot.value !== formSnapshot.value)
const busy = computed(() => saving.value || publishing.value || portraitUploading.value)
const sortedPortraits = computed(() => [...portraits.value].sort((left, right) => (
  left.sort_order - right.sort_order || left.id - right.id
)))
const coverPortrait = computed(() => (
  sortedPortraits.value.find((portrait) => portrait.is_cover) || sortedPortraits.value[0] || null
))
const coverPortraitUrl = computed(() => resolveApiUrl(
  coverPortrait.value?.image_url || form.portrait_image_url,
))
const publishActionText = computed(() => {
  if (publishing.value) return t('characterCards.editor.publishing')
  if (card.value?.review_status === 'pending') return t('characterCards.editor.updateSubmission')
  if (card.value?.review_status === 'rejected') return t('characterCards.editor.resubmit')
  if (card.value?.review_status === 'approved') return t('characterCards.editor.publishUpdate')
  return t('characterCards.editor.publish')
})
const editorStatus = computed<CharacterCardEditorStatus>(() => {
  const loadedCard = card.value
  if (!loadedCard || loadedCard.status === 'draft') return 'draft'
  if (loadedCard.review_status === 'rejected') return 'rejected'
  if (loadedCard.review_status === 'pending') return 'pending'
  if (loadedCard.visibility === 'private') return 'private'
  if (loadedCard.review_status === 'approved') return 'published'
  return 'unsubmitted'
})

watch(cardId, () => void loadCard(), { immediate: true })
watch(() => form.class_color, (value) => {
  if (form.name_color !== value) form.name_color = value
})
watch(() => form.name_color, (value) => {
  if (form.class_color !== value) form.class_color = value
})

async function loadCard() {
  card.value = null
  loading.value = true
  loadError.value = ''
  if (!Number.isSafeInteger(cardId.value) || cardId.value <= 0) {
    loadError.value = t('characterCards.editor.invalidAddress')
    loading.value = false
    return
  }

  try {
    const result = await getCharacterCard(cardId.value)
    const ownerId = userStore.user?.id
    if (!ownerId || result.user_id !== ownerId) {
      loadError.value = t('characterCards.editor.noPermission')
      return
    }
    applyCard(result)
  } catch (error: unknown) {
    loadError.value = error instanceof Error ? error.message : t('characterCards.editor.unavailable')
  } finally {
    loading.value = false
  }
}

function applyCard(nextCard: CharacterCard) {
  card.value = nextCard
  Object.assign(form, createCharacterCardDraft(nextCard))
  portraits.value = Array.isArray(nextCard.portraits) ? nextCard.portraits.map((item) => ({ ...item })) : []
  originalSnapshot.value = JSON.stringify(form)
}

function applyPortraitCard(nextCard: CharacterCard) {
  card.value = { ...nextCard }
  portraits.value = Array.isArray(nextCard.portraits) ? nextCard.portraits.map((item) => ({ ...item })) : []
  form.portrait_image_url = nextCard.portrait_image_url || ''
  patchPersistedPortraitSnapshot(form.portrait_image_url)
}

function patchPersistedPortraitSnapshot(value: string) {
  try {
    const snapshot = JSON.parse(originalSnapshot.value) as CharacterCardDraft
    snapshot.portrait_image_url = value
    originalSnapshot.value = JSON.stringify(snapshot)
  } catch {
    // Keep the editor dirty when its local baseline cannot be decoded safely.
  }
}

function addAdditionalInfo() {
  if (form.additional_info.length >= 50) return
  const item: CharacterCardAdditionalInfo = { id: 1, name: '', value: '', icon: '' }
  form.additional_info.push(item)
}

function removeAdditionalInfo(index: number) {
  form.additional_info.splice(index, 1)
}

function fillAdditionalInfoName(item: CharacterCardAdditionalInfo) {
  if (item.name.trim()) return
  item.name = additionalInfoTypes.value.find((type) => type.id === item.id)?.label || ''
}

function addPersonalityTrait() {
  if (form.personality_traits.length >= 50) return
  const trait: CharacterCardPersonalityTrait = {
    preset_id: 1,
    left_text: '',
    right_text: '',
    left_icon: '',
    right_icon: '',
    left_color: null,
    right_color: null,
    value: 10,
  }
  form.personality_traits.push(trait)
}

function removePersonalityTrait(index: number) {
  form.personality_traits.splice(index, 1)
}

function buildPayload(overrides: Partial<Pick<UpdateCharacterCardRequest, 'status' | 'visibility'>> = {}): CharacterCardSavePayload {
  return {
    character_id: form.character_id,
    first_name: form.first_name,
    last_name: form.last_name,
    display_name: form.display_name,
    title: form.title,
    full_title: form.full_title,
    race: form.race,
    class: form.class,
    eye_color: form.eye_color,
    eye_color_hex: form.eye_color_hex,
    age: form.age,
    height: form.height,
    weight: form.weight,
    birthplace: form.birthplace,
    residence: form.residence,
    relationship_status: form.relationship_status,
    icon: form.icon,
    class_color: form.class_color,
    name_color: form.name_color,
    additional_info: form.additional_info.map((item) => ({ ...item })),
    personality_traits: form.personality_traits.map((item) => ({
      ...item,
      left_color: item.left_color ? { ...item.left_color } : null,
      right_color: item.right_color ? { ...item.right_color } : null,
    })),
    summary: form.summary,
    background_story: form.background_story,
    first_impression: form.first_impression,
    impressions: form.impressions.map((impression) => ({
      slot: impression.slot,
      active: impression.active,
      title: impression.title,
      text: impression.text,
      trp3_icon: impression.trp3_icon,
      icon_image_url: impression.icon_image_url,
      image_url: impression.image_url,
    })),
    other_content: form.other_content,
    status: overrides.status ?? form.status,
    visibility: overrides.visibility ?? form.visibility,
    sort_order: form.sort_order,
  }
}

async function saveCard() {
  if (!card.value || busy.value) return
  saving.value = true
  try {
    const result = await updateCharacterCard(card.value.id, buildPayload())
    applyCard(result)
    toast.success(t(result.review_status === 'pending'
      ? 'characterCards.editor.savedPending'
      : 'characterCards.editor.saved'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function publishCard() {
  if (!card.value || busy.value) return
  publishing.value = true
  try {
    const saved = await updateCharacterCard(card.value.id, buildPayload({
      status: 'published',
      visibility: 'public',
    }))
    const published = await publishCharacterCard(saved.id)
    applyCard(published)
    toast.success(t('characterCards.editor.submittedReview'), 6000)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.publishFailed'))
  } finally {
    publishing.value = false
  }
}

function validatePortrait(file: File) {
  if (!file.type.startsWith('image/')) {
    toast.warning(t('characterCards.editor.invalidImage', { name: file.name }))
    return false
  }
  if (file.size > 20 * 1024 * 1024) {
    toast.warning(t('characterCards.editor.imageTooLarge', { name: file.name }))
    return false
  }
  return true
}

async function uploadPortrait(file: File | null) {
  if (!card.value || !file || portraitUploading.value || !validatePortrait(file)) return
  portraitUploading.value = true
  const previousIds = new Set(portraits.value.map((portrait) => portrait.id))
  try {
    const imageRef = await uploadCharacterCardPortrait(file)
    let result = await addCharacterCardPortrait(card.value.id, imageRef)
    const addedPortrait = (result.portraits || []).find((portrait) => !previousIds.has(portrait.id))
    if (addedPortrait && !addedPortrait.is_cover) {
      result = await setCharacterCardPortraitCover(card.value.id, addedPortrait.id)
    }
    applyPortraitCard(result)
    toast.success(t('characterCards.editor.portraitsUploaded', { count: 1 }))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.portraitUploadFailed'))
  } finally {
    portraitUploading.value = false
  }
}

function requestPortraitSource() {
  if (useNativeImagePicker) {
    showImageSourceDialog.value = true
    return
  }
  portraitInput.value?.click()
}

async function handleNativeImageSource(source: NativeImageSource) {
  showImageSourceDialog.value = false
  try {
    await uploadPortrait(await pickSingleNativeImageFile(source))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.portraitUploadFailed'))
  }
}

async function handlePortraitInput(event: Event) {
  const input = event.target as HTMLInputElement
  try {
    await uploadPortrait(input.files?.[0] || null)
  } finally {
    input.value = ''
  }
}

async function chooseCover(portrait: CharacterCardPortrait) {
  if (!card.value || portrait.is_cover || busy.value) return
  portraitUploading.value = true
  try {
    applyPortraitCard(await setCharacterCardPortraitCover(card.value.id, portrait.id))
    toast.success(t('characterCards.editor.coverUpdated'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.coverUpdateFailed'))
  } finally {
    portraitUploading.value = false
  }
}

async function confirmPortraitDelete() {
  const portrait = portraitToDelete.value
  if (!card.value || !portrait || busy.value) return
  portraitUploading.value = true
  try {
    applyPortraitCard(await deleteCharacterCardPortrait(card.value.id, portrait.id))
    portraitToDelete.value = null
    toast.success(t('characterCards.editor.portraitRemoved'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.portraitRemoveFailed'))
  } finally {
    portraitUploading.value = false
  }
}

function navigateBack() {
  showLeaveDialog.value = false
  if (card.value) {
    void router.push({ name: 'character-card-detail', params: { id: card.value.id } })
    return
  }
  void router.push({ name: 'my-character-cards' })
}

function requestBack() {
  if (card.value && isDirty.value) {
    showLeaveDialog.value = true
    return
  }
  navigateBack()
}

onBeforeUnmount(() => {
  showImageSourceDialog.value = false
})
</script>

<template>
  <main class="sub-page character-editor-page">
    <header class="sub-header editor-header">
      <button
        type="button"
        class="back-btn"
        :aria-label="t('characterCards.common.back')"
        @click="requestBack"
      >
        <i class="ri-arrow-left-line" aria-hidden="true" />
      </button>
      <div class="editor-header__identity">
        <span>{{ card ? t('characterCards.editor.fileKicker', { id: card.id }) : t('characterCards.common.card') }}</span>
        <h1>{{ displayName }}</h1>
      </div>
      <span v-if="card" class="status-pill" :class="`status-pill--${editorStatus}`">
        {{ t(`characterCards.common.status.${editorStatus}`) }}
      </span>
    </header>

    <div class="sub-body editor-body">
      <div v-if="loading" class="editor-state" role="status">
        <i class="ri-loader-4-line spin" aria-hidden="true" />
        {{ t('characterCards.editor.loading') }}
      </div>

      <div v-else-if="loadError || !card" class="editor-state editor-state--error" role="alert">
        <i class="ri-file-warning-line" aria-hidden="true" />
        <h2>{{ t('characterCards.editor.unavailableTitle') }}</h2>
        <p>{{ loadError || t('characterCards.editor.unavailable') }}</p>
        <button type="button" @click="router.push({ name: 'my-character-cards' })">
          {{ t('characterCards.editor.backCard') }}
        </button>
      </div>

      <template v-else>
        <section class="portrait-dossier" :aria-label="t('characterCards.editor.portraitEditorAria')">
          <div class="portrait-frame">
            <CachedImage
              v-if="coverPortraitUrl"
              :src="coverPortraitUrl"
              :alt="t('characterCards.common.portraitAlt', { name: displayName })"
              :auth-fetch="true"
              loading="eager"
            />
            <div v-else class="portrait-empty">
              <i class="ri-user-add-line" aria-hidden="true" />
              <span>{{ t('characterCards.editor.portraitHint') }}</span>
            </div>
          </div>
          <div class="portrait-copy">
            <span>{{ t('characterCards.editor.basicKicker') }}</span>
            <h2>{{ t('characterCards.editor.portraitTitle') }}</h2>
            <p>{{ t('characterCards.editor.portraitHint') }}</p>
            <button
              type="button"
              class="secondary-action"
              data-testid="portrait-upload"
              :disabled="busy"
              @click="requestPortraitSource"
            >
              <i :class="portraitUploading ? 'ri-loader-4-line spin' : 'ri-image-add-line'" aria-hidden="true" />
              {{ portraitUploading ? t('characterCards.common.uploading') : t('characterCards.editor.upload') }}
            </button>
            <input
              ref="portraitInput"
              data-testid="portrait-file-input"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              hidden
              @change="handlePortraitInput"
            >
          </div>
        </section>

        <div v-if="sortedPortraits.length > 0" class="portrait-film" :aria-label="t('characterCards.editor.portraitFilmAria')">
          <div v-for="(portrait, index) in sortedPortraits" :key="portrait.id" class="portrait-film-item">
            <button
              type="button"
              class="portrait-thumb"
              :class="{ cover: portrait.is_cover }"
              :aria-label="t('characterCards.editor.portraitItemAria', {
                index: index + 1,
                cover: portrait.is_cover ? t('characterCards.editor.currentCoverSuffix') : '',
              })"
              :disabled="busy"
              @click="chooseCover(portrait)"
            >
              <CachedImage :src="resolveApiUrl(portrait.image_url)" alt="" :auth-fetch="true" />
              <span v-if="portrait.is_cover">{{ t('characterCards.editor.cover') }}</span>
            </button>
            <button
              type="button"
              class="portrait-delete"
              :aria-label="t('characterCards.editor.deleteImage')"
              :disabled="busy"
              @click="portraitToDelete = portrait"
            >
              <i class="ri-delete-bin-line" aria-hidden="true" />
            </button>
          </div>
        </div>

        <nav class="editor-tabs" role="tablist" :aria-label="t('characterCards.tabs.editorAria')">
          <button
            v-for="tab in tabs"
            :id="`character-card-tab-${tab.id}`"
            :key="tab.id"
            type="button"
            role="tab"
            :aria-selected="activeTab === tab.id"
            :aria-controls="`character-card-panel-${tab.id}`"
            :tabindex="activeTab === tab.id ? 0 : -1"
            :class="{ active: activeTab === tab.id }"
            @click="activeTab = tab.id"
          >
            <i :class="tab.icon" aria-hidden="true" />
            <span>{{ tab.label }}</span>
          </button>
        </nav>

        <section
          v-show="activeTab === 'basic'"
          id="character-card-panel-basic"
          class="editor-panel"
          role="tabpanel"
          aria-labelledby="character-card-tab-basic"
        >
          <header class="panel-heading">
            <span>{{ t('characterCards.editor.basicKicker') }}</span>
            <h2>{{ t('characterCards.editor.basicTitle') }}</h2>
            <p>{{ t('characterCards.editor.basicBody') }}</p>
          </header>

          <fieldset class="form-section">
            <legend>{{ t('characterCards.editor.nameTitle') }}</legend>
            <div class="field-grid">
              <label><span>{{ t('characterCards.editor.fields.firstName') }}</span><input v-model="form.first_name" maxlength="128"></label>
              <label><span>{{ t('characterCards.editor.fields.lastName') }}</span><input v-model="form.last_name" maxlength="128"></label>
              <label class="field-wide"><span>{{ t('characterCards.editor.fields.displayName') }}</span><input v-model="form.display_name" maxlength="256" :placeholder="t('characterCards.editor.fields.displayNamePlaceholder')" data-testid="display-name-input"></label>
              <label><span>{{ t('characterCards.editor.fields.title') }}</span><input v-model="form.title" maxlength="128"></label>
              <label><span>{{ t('characterCards.editor.fields.fullTitle') }}</span><input v-model="form.full_title" maxlength="256"></label>
            </div>
          </fieldset>

          <fieldset class="form-section">
            <legend>{{ t('characterCards.editor.identityTitle') }}</legend>
            <div class="field-grid">
              <label><span>{{ t('characterCards.editor.fields.race') }}</span><input v-model="form.race" maxlength="64"></label>
              <label><span>{{ t('characterCards.editor.fields.class') }}</span><input v-model="form.class" maxlength="64"></label>
              <label><span>{{ t('characterCards.editor.fields.age') }}</span><input v-model="form.age" maxlength="64"></label>
              <label><span>{{ t('characterCards.editor.fields.height') }}</span><input v-model="form.height" maxlength="64"></label>
              <label><span>{{ t('characterCards.editor.fields.weight') }}</span><input v-model="form.weight" maxlength="64"></label>
              <label><span>{{ t('characterCards.editor.fields.relationship') }}</span><input v-model="form.relationship_status" maxlength="64" :placeholder="t('characterCards.editor.fields.relationshipPlaceholder')"></label>
              <label><span>{{ t('characterCards.editor.fields.birthplace') }}</span><input v-model="form.birthplace" maxlength="256"></label>
              <label><span>{{ t('characterCards.editor.fields.residence') }}</span><input v-model="form.residence" maxlength="256"></label>
              <label><span>{{ t('characterCards.editor.fields.eyeColor') }}</span><input v-model="form.eye_color" maxlength="64"></label>
              <label><span>{{ t('characterCards.editor.eyeColorValue') }}</span><input v-model="form.eye_color_hex" maxlength="16" placeholder="#RRGGBB"></label>
              <label><span>{{ t('characterCards.editor.fields.trp3Icon') }}</span><input v-model="form.icon" maxlength="128"></label>
              <label><span>{{ t('characterCards.editor.classColor') }}</span><input v-model="form.class_color" data-testid="class-color-input" maxlength="16" placeholder="#RRGGBB"></label>
              <label><span>{{ t('characterCards.editor.nameColor') }}</span><input v-model="form.name_color" data-testid="name-color-input" maxlength="16" placeholder="#RRGGBB"></label>
            </div>
          </fieldset>

          <fieldset class="form-section">
            <legend>{{ t('characterCards.editor.displayTitle') }}</legend>
            <label class="stack-field">
              <span>{{ t('characterCards.editor.summary') }}</span>
              <textarea v-model="form.summary" rows="5" maxlength="1000" :placeholder="t('characterCards.editor.summaryPlaceholder')" />
            </label>
            <div class="review-note" :class="`review-note--${card.review_status || 'none'}`">
              <i class="ri-shield-check-line" aria-hidden="true" />
              <div v-if="card.review_status === 'pending'">
                <strong>{{ t('characterCards.editor.reviewPending') }}</strong>
                <p>{{ t('characterCards.editor.pendingSubmissionBody') }}</p>
              </div>
              <div v-else-if="card.review_status === 'approved'">
                <strong>{{ t('characterCards.editor.reviewApproved') }}</strong>
                <p>{{ t('characterCards.editor.workingCopyReviewBody') }}</p>
              </div>
              <div v-else-if="card.review_status === 'rejected'">
                <strong>{{ t('characterCards.editor.reviewRejected') }}</strong>
                <p v-if="card.review_comment">{{ t('characterCards.editor.moderatorComment', { comment: card.review_comment }) }}</p>
              </div>
              <div v-else>
                <strong>{{ t('characterCards.editor.reviewOnPublish') }}</strong>
                <p>{{ t('characterCards.editor.publicPublishHint') }}</p>
              </div>
            </div>
          </fieldset>
        </section>

        <section
          v-show="activeTab === 'traits'"
          id="character-card-panel-traits"
          class="editor-panel"
          role="tabpanel"
          aria-labelledby="character-card-tab-traits"
        >
          <header class="panel-heading">
            <span>{{ t('characterCards.editor.traitsKicker') }}</span>
            <h2>{{ t('characterCards.editor.traitsTitle') }}</h2>
            <p>{{ t('characterCards.editor.traitsBody') }}</p>
          </header>

          <section class="dynamic-section">
            <header>
              <div><h3>{{ t('characterCards.editor.additionalInfoTitle') }}</h3><p>{{ t('characterCards.editor.additionalInfoBody') }}</p></div>
              <button type="button" :disabled="form.additional_info.length >= 50" @click="addAdditionalInfo"><i class="ri-add-line" aria-hidden="true" />{{ t('characterCards.editor.addAdditionalInfo') }}</button>
            </header>
            <p v-if="form.additional_info.length === 0" class="empty-note">{{ t('characterCards.editor.additionalInfoEmpty') }}</p>
            <div v-for="(item, index) in form.additional_info" :key="`additional-${index}`" class="dynamic-row">
              <div class="dynamic-row__heading"><strong>#{{ index + 1 }}</strong><button type="button" :aria-label="t('characterCards.editor.removeAdditionalInfo')" @click="removeAdditionalInfo(index)"><i class="ri-delete-bin-line" aria-hidden="true" /></button></div>
              <label><span>{{ t('characterCards.editor.additionalInfoType') }}</span><select v-model.number="item.id" @change="fillAdditionalInfoName(item)"><option v-for="type in additionalInfoTypes" :key="type.id" :value="type.id">{{ type.label }}</option></select></label>
              <label><span>{{ t('characterCards.editor.additionalInfoName') }}</span><input v-model="item.name" maxlength="80"></label>
              <label><span>{{ t('characterCards.editor.additionalInfoValue') }}</span><textarea v-model="item.value" rows="3" maxlength="500" /></label>
              <label><span>{{ t('characterCards.editor.additionalInfoIcon') }}</span><input v-model="item.icon" maxlength="128"></label>
            </div>
          </section>

          <section class="dynamic-section">
            <header>
              <div><h3>{{ t('characterCards.editor.personalityTitle') }}</h3><p>{{ t('characterCards.editor.personalityBody') }}</p></div>
              <button type="button" :disabled="form.personality_traits.length >= 50" @click="addPersonalityTrait"><i class="ri-add-line" aria-hidden="true" />{{ t('characterCards.editor.addPersonality') }}</button>
            </header>
            <p v-if="form.personality_traits.length === 0" class="empty-note">{{ t('characterCards.editor.personalityEmpty') }}</p>
            <div v-for="(trait, index) in form.personality_traits" :key="`personality-${index}`" class="dynamic-row">
              <div class="dynamic-row__heading"><strong>#{{ index + 1 }}</strong><button type="button" :aria-label="t('characterCards.editor.removePersonality')" @click="removePersonalityTrait(index)"><i class="ri-delete-bin-line" aria-hidden="true" /></button></div>
              <label><span>{{ t('characterCards.editor.personalityType') }}</span><select v-model="trait.preset_id"><option :value="null">{{ t('characterCards.editor.customPersonality') }}</option><option v-for="preset in personalityPresets" :key="preset.id" :value="preset.id">{{ preset.label }}</option></select></label>
              <label><span>{{ t('characterCards.editor.leftTrait') }}</span><input v-model="trait.left_text" maxlength="80"></label>
              <label><span>{{ t('characterCards.editor.rightTrait') }}</span><input v-model="trait.right_text" maxlength="80"></label>
              <label class="range-field"><span>{{ t('characterCards.editor.personalityValue') }} · {{ trait.value }} / 20</span><input v-model.number="trait.value" type="range" min="0" max="20" step="1"></label>
            </div>
          </section>
        </section>

        <section
          v-show="activeTab === 'background'"
          id="character-card-panel-background"
          class="editor-panel rich-panel"
          role="tabpanel"
          aria-labelledby="character-card-tab-background"
        >
          <header class="panel-heading"><span>{{ t('characterCards.editor.backgroundKicker') }}</span><h2>{{ t('characterCards.editor.backgroundTitle') }}</h2><p>{{ t('characterCards.editor.backgroundBody') }}</p></header>
          <MobileRichEditor v-model="form.background_story" :placeholder="t('characterCards.editor.backgroundPlaceholder')" />
        </section>

        <section
          v-show="activeTab === 'impression'"
          id="character-card-panel-impression"
          class="editor-panel"
          role="tabpanel"
          aria-labelledby="character-card-tab-impression"
        >
          <header class="panel-heading"><span>{{ t('characterCards.editor.impressionKicker') }}</span><h2>{{ t('characterCards.editor.impressionTitle') }}</h2><p>{{ t('characterCards.editor.impressionBody') }}</p></header>
          <div class="impression-slots" :aria-label="t('characterCards.editor.impressionAria')">
            <article v-for="impression in form.impressions" :key="impression.slot" class="impression-slot" :class="{ active: impression.active }">
              <header>
                <strong>{{ t('characterCards.editor.observationLabel') }} {{ impression.slot }}</strong>
                <label class="toggle-field"><input v-model="impression.active" type="checkbox"><span aria-hidden="true" />{{ impression.active ? t('characterCards.editor.impressionEnabled') : t('characterCards.editor.impressionDisabled') }}</label>
              </header>
              <label><span>{{ t('characterCards.editor.attributeName') }}</span><input v-model="impression.title" maxlength="80" :placeholder="t('characterCards.editor.attributePlaceholder')"></label>
              <label><span>{{ t('characterCards.editor.shortDescription') }}</span><textarea v-model="impression.text" rows="4" maxlength="500" :placeholder="t('characterCards.editor.descriptionPlaceholder')" /></label>
              <label><span>{{ t('characterCards.editor.trp3IconFallback') }}</span><input v-model="impression.trp3_icon" maxlength="128" :placeholder="t('characterCards.editor.trp3IconPlaceholder')"></label>
            </article>
          </div>
          <div class="supplement-editor">
            <h3>{{ t('characterCards.editor.otherNotes') }}</h3>
            <p>{{ t('characterCards.editor.supplementBody') }}</p>
            <MobileRichEditor v-model="form.first_impression" :placeholder="t('characterCards.editor.supplementPlaceholder')" />
          </div>
        </section>

        <section
          v-show="activeTab === 'other'"
          id="character-card-panel-other"
          class="editor-panel rich-panel"
          role="tabpanel"
          aria-labelledby="character-card-tab-other"
        >
          <header class="panel-heading"><span>{{ t('characterCards.editor.otherKicker') }}</span><h2>{{ t('characterCards.editor.otherTitle') }}</h2><p>{{ t('characterCards.editor.otherBody') }}</p></header>
          <MobileRichEditor v-model="form.other_content" :placeholder="t('characterCards.editor.otherPlaceholder')" />
        </section>

        <footer class="save-dock" aria-live="polite">
          <div class="save-state">
            <i :class="isDirty ? 'ri-edit-circle-line' : 'ri-cloud-line'" aria-hidden="true" />
            <span>{{ isDirty ? t('characterCards.editor.unsavedChanges') : t('characterCards.editor.allSaved') }}</span>
          </div>
          <div class="save-actions">
            <button type="button" class="secondary-action" data-testid="save-character-card" :disabled="busy" @click="saveCard">
              <i :class="saving ? 'ri-loader-4-line spin' : 'ri-save-3-line'" aria-hidden="true" />
              {{ saving ? t('characterCards.common.saving') : t('characterCards.common.save') }}
            </button>
            <button type="button" class="primary-action" data-testid="publish-character-card" :disabled="busy" @click="publishCard">
              <i :class="publishing ? 'ri-loader-4-line spin' : 'ri-send-plane-2-line'" aria-hidden="true" />
              {{ publishActionText }}
            </button>
          </div>
        </footer>
      </template>
    </div>

    <NativeImageSourceDialog
      :model-value="showImageSourceDialog"
      @update:modelValue="showImageSourceDialog = $event"
      @select="handleNativeImageSource"
    />

    <div v-if="showLeaveDialog" class="dialog-mask" @click.self="showLeaveDialog = false">
      <div class="leave-dialog" role="dialog" aria-modal="true" :aria-labelledby="'character-card-leave-title'">
        <h2 id="character-card-leave-title">{{ t('characterCards.editor.leaveTitle') }}</h2>
        <p>{{ t('characterCards.editor.leaveMessage') }}</p>
        <div>
          <button type="button" @click="showLeaveDialog = false">{{ t('characterCards.common.cancel') }}</button>
          <button type="button" class="danger-action" @click="navigateBack">{{ t('characterCards.editor.discard') }}</button>
        </div>
      </div>
    </div>

    <div v-if="portraitToDelete" class="dialog-mask" @click.self="portraitToDelete = null">
      <div class="leave-dialog" role="dialog" aria-modal="true" aria-labelledby="portrait-delete-title">
        <h2 id="portrait-delete-title">{{ t('characterCards.editor.removePortraitTitle') }}</h2>
        <p>{{ t(portraitToDelete.is_cover
          ? 'characterCards.editor.removeCoverMessage'
          : 'characterCards.editor.removePortraitMessage') }}</p>
        <div>
          <button type="button" :disabled="portraitUploading" @click="portraitToDelete = null">{{ t('characterCards.common.cancel') }}</button>
          <button type="button" class="danger-action" data-testid="confirm-portrait-delete" :disabled="portraitUploading" @click="confirmPortraitDelete">{{ t('characterCards.editor.removePortraitConfirm') }}</button>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.character-editor-page { min-height: 100dvh; padding-bottom: calc(110px + var(--safe-bottom)); }
.editor-header { align-items: center; }
.editor-header__identity { display: grid; min-width: 0; flex: 1; gap: 2px; }
.editor-header__identity > span { color: var(--color-accent); font: 800 8px/1.2 ui-monospace, monospace; letter-spacing: .12em; text-transform: uppercase; }
.editor-header__identity h1 { overflow: hidden; color: var(--color-primary); font-family: Georgia, 'Noto Serif SC', serif; font-size: 17px; text-overflow: ellipsis; white-space: nowrap; }
.status-pill { flex: 0 0 auto; padding: 5px 8px; border-radius: 999px; background: var(--color-primary-light); color: var(--color-primary); font-size: 9px; font-weight: 800; }
.status-pill--pending { background: #fff4d6; color: #7b5200; }
.status-pill--approved { background: var(--color-success-light); color: var(--color-success); }
.status-pill--published { background: var(--color-success-light); color: var(--color-success); }
.status-pill--rejected { background: #fde8e8; color: var(--btn-danger-bg); }
.status-pill--draft, .status-pill--private { color: var(--color-text-secondary); }
.status-pill--unsubmitted { background: var(--tag-bg); color: var(--color-accent); }
.editor-body { display: grid; max-width: 820px; margin: 0 auto; gap: 12px; }
.editor-state { display: grid; min-height: 62vh; place-items: center; align-content: center; gap: 10px; color: var(--color-text-secondary); text-align: center; }
.editor-state > i { font-size: 32px; }
.editor-state h2 { color: var(--color-primary); font-family: Georgia, 'Noto Serif SC', serif; }
.editor-state p { max-width: 430px; font-size: 13px; line-height: 1.6; }
.editor-state button { min-height: 44px; padding: 0 16px; border: 1px solid var(--color-primary); border-radius: 11px; background: var(--color-primary); color: var(--text-light); }
.editor-state--error > i { color: var(--btn-danger-bg); }
.portrait-dossier { display: grid; grid-template-columns: 120px minmax(0, 1fr); gap: 15px; padding: 14px; border: 1px solid var(--color-border); border-radius: 18px; background: var(--color-card-bg); box-shadow: var(--shadow-sm); }
.portrait-frame { overflow: hidden; aspect-ratio: 3 / 4; border: 1px solid color-mix(in srgb, var(--color-accent) 40%, var(--color-border)); border-radius: 12px; background: var(--color-primary-light); }
.portrait-frame :deep(.cached-image) { height: 100%; }
.portrait-empty { display: grid; height: 100%; place-items: center; align-content: center; gap: 6px; padding: 8px; color: var(--color-text-secondary); text-align: center; }
.portrait-empty i { font-size: 28px; }
.portrait-empty span { font-size: 9px; line-height: 1.4; }
.portrait-copy { display: grid; min-width: 0; align-content: center; gap: 6px; }
.portrait-copy > span, .panel-heading > span { color: var(--color-accent); font: 800 8px/1.2 ui-monospace, monospace; letter-spacing: .13em; text-transform: uppercase; }
.portrait-copy h2 { color: var(--color-primary); font-family: Georgia, 'Noto Serif SC', serif; font-size: 18px; }
.portrait-copy p { color: var(--color-text-secondary); font-size: 11px; line-height: 1.5; }
.primary-action, .secondary-action { display: inline-flex; min-height: 44px; align-items: center; justify-content: center; gap: 6px; padding: 0 13px; border: 1px solid var(--color-primary); border-radius: 11px; font-weight: 800; }
.primary-action { background: var(--color-primary); color: var(--text-light); }
.secondary-action { background: var(--input-bg); color: var(--color-primary); }
.primary-action:disabled, .secondary-action:disabled { opacity: .55; }
.portrait-copy .secondary-action { justify-self: start; margin-top: 3px; }
.portrait-film { display: flex; gap: 8px; padding: 3px 1px; overflow-x: auto; }
.portrait-film-item { display: grid; width: 64px; flex: 0 0 64px; gap: 4px; }
.portrait-thumb { position: relative; width: 64px; height: 80px; overflow: hidden; padding: 0; border: 2px solid transparent; border-radius: 10px; background: var(--color-card-bg); }
.portrait-thumb.cover { border-color: var(--color-accent); }
.portrait-thumb :deep(.cached-image) { height: 100%; }
.portrait-thumb > span { position: absolute; right: 3px; bottom: 3px; padding: 2px 5px; border-radius: 999px; background: var(--color-primary); color: var(--text-light); font-size: 8px; }
.portrait-delete { display: grid; width: 100%; min-height: 44px; place-items: center; border: 1px solid color-mix(in srgb, var(--btn-danger-bg) 45%, var(--color-border)); border-radius: 9px; background: var(--input-bg); color: var(--btn-danger-bg); font-size: 16px; }
.editor-tabs { display: grid; position: sticky; z-index: 8; top: calc(var(--safe-top) + 60px); grid-auto-flow: column; grid-auto-columns: minmax(82px, 1fr); overflow-x: auto; border: 1px solid var(--color-border); border-radius: 13px; background: color-mix(in srgb, var(--color-card-bg) 94%, transparent); box-shadow: var(--shadow-sm); backdrop-filter: blur(8px); }
.editor-tabs button { display: grid; min-width: 82px; min-height: 54px; place-items: center; align-content: center; gap: 3px; padding: 5px; border: 0; border-bottom: 3px solid transparent; background: transparent; color: var(--color-text-secondary); font-size: 9px; font-weight: 800; }
.editor-tabs button i { font-size: 18px; }
.editor-tabs button.active { border-bottom-color: var(--color-accent); color: var(--color-primary); background: var(--color-primary-light); }
.editor-panel { display: grid; gap: 20px; padding: 17px 14px; border: 1px solid var(--color-border); border-radius: 18px; background: var(--color-card-bg); box-shadow: var(--shadow-sm); }
.panel-heading { display: grid; gap: 5px; padding-bottom: 14px; border-bottom: 1px solid var(--color-border); }
.panel-heading h2 { color: var(--color-primary); font-family: Georgia, 'Noto Serif SC', serif; font-size: 20px; }
.panel-heading p { color: var(--color-text-secondary); font-size: 11px; line-height: 1.55; }
.form-section { display: grid; min-width: 0; gap: 12px; padding: 0; border: 0; }
.form-section legend { margin-bottom: 10px; color: var(--color-primary); font-family: Georgia, 'Noto Serif SC', serif; font-size: 16px; font-weight: 700; }
.field-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.field-wide { grid-column: 1 / -1; }
.field-grid label, .stack-field, .dynamic-row label, .impression-slot label { display: grid; min-width: 0; gap: 6px; color: var(--color-text-secondary); font-size: 10px; font-weight: 800; }
.field-grid input, .stack-field textarea, .dynamic-row input, .dynamic-row select, .dynamic-row textarea, .impression-slot input, .impression-slot textarea { width: 100%; min-height: 44px; padding: 10px 11px; border: 1px solid var(--input-border); border-radius: 10px; outline: none; background: var(--input-bg); color: var(--color-text-main); font: inherit; font-size: 13px; }
.stack-field textarea, .dynamic-row textarea, .impression-slot textarea { min-height: 92px; resize: vertical; line-height: 1.55; }
.field-grid input:focus, .stack-field textarea:focus, .dynamic-row input:focus, .dynamic-row select:focus, .dynamic-row textarea:focus, .impression-slot input:focus, .impression-slot textarea:focus { border-color: var(--color-accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-accent) 14%, transparent); }
.review-note { display: flex; gap: 9px; padding: 12px; border: 1px solid var(--color-border); border-radius: 11px; background: var(--color-primary-light); color: var(--color-primary); }
.review-note > i { font-size: 20px; }
.review-note > div { display: grid; gap: 4px; }
.review-note strong { font-size: 12px; }
.review-note p { color: var(--color-text-secondary); font-size: 10px; line-height: 1.5; }
.review-note--pending { background: #fff8e6; color: #7b5200; }
.review-note--approved { background: var(--color-success-light); color: var(--color-success); }
.review-note--rejected { background: #fdecec; color: var(--btn-danger-bg); }
.dynamic-section { display: grid; gap: 12px; }
.dynamic-section > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
.dynamic-section > header h3, .supplement-editor h3 { color: var(--color-primary); font-family: Georgia, 'Noto Serif SC', serif; font-size: 16px; }
.dynamic-section > header p, .supplement-editor > p { margin-top: 4px; color: var(--color-text-secondary); font-size: 10px; line-height: 1.5; }
.dynamic-section > header button { display: inline-flex; min-height: 44px; flex: 0 0 auto; align-items: center; gap: 5px; padding: 0 10px; border: 1px solid var(--color-primary); border-radius: 10px; background: var(--input-bg); color: var(--color-primary); font-size: 10px; font-weight: 800; }
.dynamic-row { display: grid; gap: 10px; padding: 12px; border: 1px solid var(--color-border); border-radius: 12px; background: color-mix(in srgb, var(--color-background) 35%, var(--color-card-bg)); }
.dynamic-row__heading { display: flex; align-items: center; justify-content: space-between; color: var(--color-accent); font: 800 10px/1 ui-monospace, monospace; }
.dynamic-row__heading button { display: grid; width: 44px; height: 44px; place-items: center; border: 0; border-radius: 10px; background: transparent; color: var(--btn-danger-bg); font-size: 18px; }
.range-field input { min-height: 44px; padding: 0; accent-color: var(--color-accent); }
.empty-note { padding: 18px; border: 1px dashed var(--color-border); border-radius: 11px; color: var(--color-text-secondary); font-size: 11px; text-align: center; }
.impression-slots { display: grid; gap: 11px; }
.impression-slot { display: grid; gap: 11px; padding: 12px; border: 1px solid var(--color-border); border-radius: 13px; background: color-mix(in srgb, var(--color-background) 32%, var(--color-card-bg)); opacity: .78; }
.impression-slot.active { border-color: color-mix(in srgb, var(--color-accent) 45%, var(--color-border)); opacity: 1; }
.impression-slot > header { display: flex; min-height: 44px; align-items: center; justify-content: space-between; gap: 10px; }
.impression-slot > header > strong { color: var(--color-primary); font: 800 10px/1 ui-monospace, monospace; letter-spacing: .08em; text-transform: uppercase; }
.toggle-field { display: inline-flex !important; min-height: 44px; grid-template-columns: auto auto auto; align-items: center; gap: 6px !important; }
.toggle-field input { position: absolute; width: 1px; min-height: 1px; opacity: 0; }
.toggle-field > span { position: relative; width: 34px; height: 20px; border-radius: 999px; background: var(--color-border); }
.toggle-field > span::after { position: absolute; top: 3px; left: 3px; width: 14px; height: 14px; border-radius: 50%; background: white; box-shadow: var(--shadow-sm); content: ''; transition: transform .15s ease; }
.toggle-field input:checked + span { background: var(--color-accent); }
.toggle-field input:checked + span::after { transform: translateX(14px); }
.supplement-editor { display: grid; gap: 8px; padding-top: 16px; border-top: 1px dashed var(--color-border); }
.rich-panel :deep(.editor-content), .supplement-editor :deep(.editor-content) { min-height: 260px; }
.save-dock { display: grid; position: fixed; z-index: 30; right: 0; bottom: 0; left: 0; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 10px; padding: 10px calc(var(--safe-right) + var(--page-gutter)) calc(10px + var(--safe-bottom)) calc(var(--safe-left) + var(--page-gutter)); border-top: 1px solid var(--color-border); background: color-mix(in srgb, var(--color-card-bg) 94%, transparent); box-shadow: 0 -6px 22px rgba(44, 24, 16, .1); backdrop-filter: blur(12px); }
.save-state { display: flex; min-width: 0; align-items: center; gap: 6px; color: var(--color-text-secondary); font-size: 10px; }
.save-state span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.save-actions { display: flex; gap: 8px; }
.dialog-mask { display: grid; position: fixed; z-index: 1200; inset: 0; place-items: end center; padding: 16px; background: rgba(0, 0, 0, .44); }
.leave-dialog { display: grid; width: min(100%, 430px); gap: 10px; padding: 18px; border-radius: 18px; background: var(--color-panel-bg); box-shadow: 0 18px 40px rgba(0, 0, 0, .2); }
.leave-dialog h2 { color: var(--color-primary); font-size: 17px; }
.leave-dialog p { color: var(--color-text-secondary); font-size: 13px; line-height: 1.6; }
.leave-dialog > div { display: grid; grid-template-columns: 1fr 1fr; gap: 9px; }
.leave-dialog button { min-height: 44px; border: 1px solid var(--color-border); border-radius: 11px; background: var(--input-bg); color: var(--color-primary); font-weight: 800; }
.leave-dialog .danger-action { border-color: var(--btn-danger-bg); background: var(--btn-danger-bg); color: white; }
.spin { animation: editor-spin .9s linear infinite; }
@keyframes editor-spin { to { transform: rotate(360deg); } }
@media (max-width: 430px) {
  .portrait-dossier { grid-template-columns: 104px minmax(0, 1fr); }
  .field-grid { grid-template-columns: 1fr; }
  .field-wide { grid-column: auto; }
  .save-state { display: none; }
  .save-dock { grid-template-columns: 1fr; }
  .save-actions { display: grid; grid-template-columns: minmax(0, .8fr) minmax(0, 1.2fr); }
}
@media (prefers-reduced-motion: reduce) {
  .spin { animation: none; }
  .toggle-field > span::after { transition: none; }
}
</style>
