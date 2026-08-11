<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  addCharacterCardPortrait,
  deleteCharacterCardPortrait,
  getCharacterCard,
  getCharacterCardTRP3Lua,
  reorderCharacterCardPortraits,
  setCharacterCardPortraitCover,
  syncCharacterCardFromTRP3,
  updateCharacterCard,
  uploadCharacterCardImpressionImage,
  uploadCharacterCardPortrait,
  writeBackCharacterCardToTRP3,
  type CharacterCard,
  type CharacterCardImpressionImageKind,
  type CharacterCardPortraitImage,
} from '@/api/characterCard'
import { listAccountBackups, type AccountBackup } from '@/api/accountBackup'
import { resolveApiUrl } from '@/api/item'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import ImageCropperDialog from '@/components/ImageCropperDialog.vue'
import CharacterCardColorField from '@/components/character-cards/CharacterCardColorField.vue'
import CharacterCardGalleryImage from '@/components/character-cards/CharacterCardGalleryImage.vue'
import CharacterCardImpressionMark from '@/components/character-cards/CharacterCardImpressionMark.vue'
import LocalTRP3VersionHistory from '@/components/sync/LocalTRP3VersionHistory.vue'
import PostQuickJump from '@/components/PostQuickJump.vue'
import RModal from '@/components/RModal.vue'
import TiptapEditor from '@/components/TiptapEditor.vue'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import {
  createCharacterCardDraft,
  createEmptyCharacterCardDraft,
  getCharacterCardDisplayName,
  type CharacterCardEditorTab,
} from '@/utils/characterCardDraft'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import { getCharacterCardCoverPortrait, normalizeCharacterCardPortraits } from '@/utils/characterCardPortraits'

interface EditorHandle {
  insertContent: (html: string) => void
}

interface ImpressionUploadTarget {
  index: number
  kind: CharacterCardImpressionImageKind
}

interface LocalAccountOption {
  account_id: string
}

type RichTextTab = Exclude<CharacterCardEditorTab, 'basic'>

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const dialog = useDialog()
const userStore = useUserStore()

const cardId = computed(() => Number(route.params.id))
const card = ref<CharacterCard | null>(null)
const form = reactive(createEmptyCharacterCardDraft())
const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const activeTab = ref<CharacterCardEditorTab>('basic')
const originalSnapshot = ref('')

const portraitInput = ref<HTMLInputElement | null>(null)
const cropperOpen = ref(false)
const cropperFile = ref<File | null>(null)
const portraitUploading = ref(false)
const syncing = ref(false)
const writingBack = ref(false)
const uploadedPortraitPreview = ref('')
const portraitPreviewObjectUrls = new Set<string>()
const portraitPreviewOpen = ref(false)
const portraits = ref<CharacterCardPortraitImage[]>([])
const selectedPortraitId = ref<number | null>(null)

const writeBackSetupOpen = ref(false)
const writeBackHistoryOpen = ref(false)
const luaPreviewOpen = ref(false)
const luaPreviewLoading = ref(false)
const luaPreview = ref('')
const localAccounts = ref<LocalAccountOption[]>([])
const cloudBackups = ref<AccountBackup[]>([])
const writeBackAccountId = ref('')
const writeBackProfileId = ref('')
const writeBackSnapshotName = ref('')
const syncCloudAfterWriteBack = ref(false)

const impressionImageInput = ref<HTMLInputElement | null>(null)
const impressionUploadTarget = ref<ImpressionUploadTarget | null>(null)
const impressionUploading = reactive<Record<string, boolean>>({})
const impressionImagePreviews = reactive<Record<string, string>>({})
const impressionPreviewObjectUrls = new Set<string>()

const quickJumpOpen = ref(false)
const quickJumpTarget = ref<RichTextTab>('background')
const backgroundEditor = ref<EditorHandle | null>(null)
const impressionEditor = ref<EditorHandle | null>(null)
const otherEditor = ref<EditorHandle | null>(null)

const tabs = computed<Array<{ id: CharacterCardEditorTab; label: string; icon: string }>>(() => [
  { id: 'basic', label: t('characterCards.tabs.basic'), icon: 'ri-id-card-line' },
  { id: 'background', label: t('characterCards.tabs.background'), icon: 'ri-book-open-line' },
  { id: 'impression', label: t('characterCards.tabs.impression'), icon: 'ri-eye-2-line' },
  { id: 'other', label: t('characterCards.tabs.other'), icon: 'ri-archive-stack-line' },
])

const displayName = computed(() => getCharacterCardDisplayName(form))
const displayNameColor = computed(() => getCharacterCardDisplayColor(form))
const isDirty = computed(() => originalSnapshot.value !== JSON.stringify(form))
const hasPortrait = computed(() => portraits.value.length > 0 || Boolean(form.portrait_image_url))
const selectedPortrait = computed(() => (
  portraits.value.find((portrait) => portrait.id === selectedPortraitId.value)
  || getCharacterCardCoverPortrait(portraits.value)
))
const writeBackBackup = computed(() => cloudBackups.value.find((backup) => backup.account_id === writeBackAccountId.value))
const historyAccountId = computed(() => writeBackAccountId.value || card.value?.source_account_id || '')
const publicReviewPending = computed(() => (
  form.status === 'published'
  && form.visibility === 'public'
  && card.value?.review_status === 'pending'
))
const imageUploadInProgress = computed(() => (
  portraitUploading.value || Object.values(impressionUploading).some(Boolean)
))

const isTauriRuntime = computed(() => typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window)
const wowPath = computed(() => localStorage.getItem('wow_path')?.trim() || '')
const writeBackDisabledReason = computed(() => {
  if (!isTauriRuntime.value) return t('characterCards.editor.desktopOnly')
  if (!wowPath.value) return t('characterCards.editor.wowPathRequired')
  return ''
})
const canSyncFromBackup = computed(() => Boolean(card.value?.source_backup_id && card.value?.source_profile_id))

watch(cardId, () => void loadCard(), { immediate: true })
watch(() => form.class_color, (value) => {
  if (form.name_color !== value) form.name_color = value
})
watch(() => form.name_color, (value) => {
  if (form.class_color !== value) form.class_color = value
})
watch(writeBackAccountId, (accountId) => {
  syncCloudAfterWriteBack.value = cloudBackups.value.some((backup) => backup.account_id === accountId)
})
onBeforeUnmount(() => {
  revokeAllPortraitPreviewObjectUrls()
  revokeAllImpressionPreviewObjectUrls()
})

async function loadCard() {
  if (!Number.isFinite(cardId.value) || cardId.value <= 0) {
    loadError.value = t('characterCards.editor.invalidAddress')
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    const result = await getCharacterCard(cardId.value)
    if (!userStore.user?.id || userStore.user.id !== result.user_id) {
      card.value = null
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
  portraits.value = normalizeCharacterCardPortraits(nextCard)
  selectedPortraitId.value = getCharacterCardCoverPortrait(portraits.value)?.id ?? null
  revokeUploadedPortraitPreview()
  revokeAllImpressionPreviewObjectUrls()
  originalSnapshot.value = JSON.stringify(form)
}

function applySyncedCard(nextCard: CharacterCard) {
  const preserved = {
    display_name: form.display_name,
    summary: form.summary,
    background_story: form.background_story,
    first_impression: form.first_impression,
    impressions: form.impressions.map((impression) => ({ ...impression })),
    other_content: form.other_content,
    portrait_image_url: form.portrait_image_url,
    status: form.status,
    visibility: form.visibility,
    sort_order: form.sort_order,
  }
  const preservedPortraitPreview = uploadedPortraitPreview.value
  const preservedPortraits = portraits.value.map((portrait) => ({ ...portrait }))
  const preservedSelectedPortraitId = selectedPortraitId.value

  card.value = {
    ...nextCard,
    portraits: preservedPortraits,
    portrait_image_url: card.value?.portrait_image_url || nextCard.portrait_image_url,
    portrait_image_updated_at: card.value?.portrait_image_updated_at || nextCard.portrait_image_updated_at,
  }
  Object.assign(form, createCharacterCardDraft(nextCard))
  originalSnapshot.value = JSON.stringify(form)
  Object.assign(form, preserved)
  portraits.value = preservedPortraits
  selectedPortraitId.value = preservedSelectedPortraitId
  uploadedPortraitPreview.value = preservedPortraitPreview
}

function selectTab(tab: CharacterCardEditorTab) {
  activeTab.value = tab
}

function handleTabKeydown(event: KeyboardEvent, currentTab: CharacterCardEditorTab) {
  const tabItems = tabs.value
  const currentIndex = tabItems.findIndex((tab) => tab.id === currentTab)
  if (currentIndex < 0) return

  let nextIndex: number | null = null
  switch (event.key) {
    case 'ArrowRight':
    case 'ArrowDown':
      nextIndex = (currentIndex + 1) % tabItems.length
      break
    case 'ArrowLeft':
    case 'ArrowUp':
      nextIndex = (currentIndex - 1 + tabItems.length) % tabItems.length
      break
    case 'Home':
      nextIndex = 0
      break
    case 'End':
      nextIndex = tabItems.length - 1
      break
    default:
      return
  }

  event.preventDefault()
  const nextTab = tabItems[nextIndex]
  activeTab.value = nextTab.id
  void nextTick(() => {
    document.getElementById(`character-tab-${nextTab.id}`)?.focus()
  })
}

function openQuickJump(tab: RichTextTab) {
  quickJumpTarget.value = tab
  quickJumpOpen.value = true
}

function insertQuickJump(html: string) {
  const editors: Record<RichTextTab, EditorHandle | null> = {
    background: backgroundEditor.value,
    impression: impressionEditor.value,
    other: otherEditor.value,
  }
  editors[quickJumpTarget.value]?.insertContent(html)
}

function triggerPortraitUpload() {
  portraitInput.value?.click()
}

async function handlePortraitFile(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  const validFiles = files.filter(validatePortraitFile)
  if (!validFiles.length) return
  if (validFiles.length === 1) {
    cropperFile.value = validFiles[0]
    cropperOpen.value = true
    return
  }
  await uploadPortraitFiles(validFiles)
}

function validatePortraitFile(file: File) {
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

async function uploadPortraitFiles(files: File[]) {
  if (!card.value || portraitUploading.value) return
  portraitUploading.value = true
  let uploaded = 0
  try {
    for (const file of files) {
      const nextPreview = URL.createObjectURL(file)
      portraitPreviewObjectUrls.add(nextPreview)
      uploadedPortraitPreview.value = nextPreview
      try {
        const imageRef = await uploadCharacterCardPortrait(file)
        const result = await addCharacterCardPortrait(card.value.id, imageRef)
        applyPortraitMutationCard(result)
        uploaded += 1
      } finally {
        revokeObjectUrl(nextPreview)
        if (uploadedPortraitPreview.value === nextPreview) uploadedPortraitPreview.value = ''
      }
    }
    if (uploaded) toast.success(t('characterCards.editor.portraitsUploaded', { count: uploaded }))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.portraitUploadFailed'))
  } finally {
    portraitUploading.value = false
  }
}

async function handlePortraitCropped(file: File) {
  try {
    await uploadPortraitFiles([file])
  } finally {
    cropperFile.value = null
  }
}

function handleCropperError(error: Error) {
  toast.error(error.message || t('characterCards.editor.imageProcessFailed'))
}

function patchOriginalPortraitSnapshot(value: string) {
  try {
    const snapshot = JSON.parse(originalSnapshot.value) as Record<string, unknown>
    snapshot.portrait_image_url = value
    originalSnapshot.value = JSON.stringify(snapshot)
  } catch {
    // A malformed local snapshot should make the editor dirty instead of hiding changes.
  }
}

function applyPortraitMutationCard(nextCard: CharacterCard) {
  card.value = nextCard
  portraits.value = normalizeCharacterCardPortraits(nextCard)
  const cover = getCharacterCardCoverPortrait(portraits.value)
  selectedPortraitId.value = cover?.id ?? null
  form.portrait_image_url = nextCard.portrait_image_url || ''
  patchOriginalPortraitSnapshot(form.portrait_image_url)
}

async function setPortraitCover(portrait: CharacterCardPortraitImage) {
  if (!card.value || portrait.id <= 0 || portrait.is_cover || portraitUploading.value) return
  portraitUploading.value = true
  try {
    applyPortraitMutationCard(await setCharacterCardPortraitCover(card.value.id, portrait.id))
    toast.success(t('characterCards.editor.coverUpdated'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.coverUpdateFailed'))
  } finally {
    portraitUploading.value = false
  }
}

async function movePortrait(index: number, delta: -1 | 1) {
  if (!card.value || portraitUploading.value) return
  const target = index + delta
  if (target < 0 || target >= portraits.value.length || portraits.value.some((portrait) => portrait.id <= 0)) return
  const previous = portraits.value.map((portrait) => ({ ...portrait }))
  const next = [...portraits.value]
  ;[next[index], next[target]] = [next[target], next[index]]
  portraits.value = next.map((portrait, sortOrder) => ({ ...portrait, sort_order: sortOrder }))
  portraitUploading.value = true
  try {
    applyPortraitMutationCard(await reorderCharacterCardPortraits(card.value.id, next.map((portrait) => portrait.id)))
  } catch (error: unknown) {
    portraits.value = previous
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.reorderFailed'))
  } finally {
    portraitUploading.value = false
  }
}

async function removePortrait(portrait: CharacterCardPortraitImage) {
  if (!card.value || portraitUploading.value) return
  const confirmed = await dialog.confirm({
    title: t('characterCards.editor.removePortraitTitle'),
    message: portrait.is_cover
      ? t('characterCards.editor.removeCoverMessage')
      : t('characterCards.editor.removePortraitMessage'),
    type: 'error',
    confirmText: t('characterCards.editor.removePortraitConfirm'),
  })
  if (!confirmed) return

  if (portrait.id <= 0) {
    form.portrait_image_url = ''
    portraits.value = []
    selectedPortraitId.value = null
    portraitPreviewOpen.value = false
    return
  }

  portraitUploading.value = true
  try {
    applyPortraitMutationCard(await deleteCharacterCardPortrait(card.value.id, portrait.id))
    portraitPreviewOpen.value = false
    toast.success(t('characterCards.editor.portraitRemoved'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.portraitRemoveFailed'))
  } finally {
    portraitUploading.value = false
  }
}

function openPortraitPreview(portrait: CharacterCardPortraitImage) {
  selectedPortraitId.value = portrait.id
  portraitPreviewOpen.value = true
}

function revokeObjectUrl(url: string) {
  if (!url.startsWith('blob:')) return
  URL.revokeObjectURL(url)
  portraitPreviewObjectUrls.delete(url)
}

function revokeUploadedPortraitPreview() {
  revokeObjectUrl(uploadedPortraitPreview.value)
  uploadedPortraitPreview.value = ''
}

function revokeAllPortraitPreviewObjectUrls() {
  for (const url of portraitPreviewObjectUrls) URL.revokeObjectURL(url)
  portraitPreviewObjectUrls.clear()
  uploadedPortraitPreview.value = ''
}

function impressionUploadKey(index: number, kind: CharacterCardImpressionImageKind) {
  return `${index}:${kind}`
}

function getImpressionPreview(index: number, kind: CharacterCardImpressionImageKind) {
  return impressionImagePreviews[impressionUploadKey(index, kind)] || ''
}

function getImpressionImageUrl(index: number) {
  const impression = form.impressions[index]
  if (!impression) return ''
  return getImpressionPreview(index, 'image') || resolveApiUrl(impression.image_url)
}

function isImpressionUploading(index: number, kind: CharacterCardImpressionImageKind) {
  return impressionUploading[impressionUploadKey(index, kind)] === true
}

function triggerImpressionUpload(index: number, kind: CharacterCardImpressionImageKind) {
  impressionUploadTarget.value = { index, kind }
  impressionImageInput.value?.click()
}

async function handleImpressionImageFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  const target = impressionUploadTarget.value
  impressionUploadTarget.value = null
  if (!file || !target || !form.impressions[target.index]) return
  if (!file.type.startsWith('image/')) {
    toast.warning(t('characterCards.editor.impressionImageType'))
    return
  }

  if (file.size > 20 * 1024 * 1024) {
    toast.warning(t(target.kind === 'icon' ? 'characterCards.editor.impressionIconTooLarge' : 'characterCards.editor.impressionImageTooLarge'))
    return
  }

  const key = impressionUploadKey(target.index, target.kind)
  const previousPreview = impressionImagePreviews[key] || ''
  const nextPreview = URL.createObjectURL(file)
  impressionPreviewObjectUrls.add(nextPreview)
  impressionImagePreviews[key] = nextPreview
  impressionUploading[key] = true
  const uploadingCardId = card.value?.id

  try {
    const imageRef = await uploadCharacterCardImpressionImage(file, target.kind)
    if (!card.value || card.value.id !== uploadingCardId) return
    const impression = form.impressions[target.index]
    if (target.kind === 'icon') {
      impression.icon_image_url = imageRef
    } else {
      impression.image_url = imageRef
    }
    revokeImpressionPreviewUrl(previousPreview)
    toast.success(t(target.kind === 'icon' ? 'characterCards.editor.impressionIconUploaded' : 'characterCards.editor.impressionImageUploaded'))
  } catch (error: unknown) {
    revokeImpressionPreviewUrl(nextPreview)
    if (previousPreview) impressionImagePreviews[key] = previousPreview
    else delete impressionImagePreviews[key]
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.impressionUploadFailed'))
  } finally {
    impressionUploading[key] = false
  }
}

function removeImpressionImage(index: number, kind: CharacterCardImpressionImageKind) {
  const impression = form.impressions[index]
  if (!impression) return
  const key = impressionUploadKey(index, kind)
  revokeImpressionPreviewUrl(impressionImagePreviews[key] || '')
  delete impressionImagePreviews[key]
  if (kind === 'icon') {
    impression.icon_image_url = ''
  } else {
    impression.image_url = ''
  }
}

function revokeImpressionPreviewUrl(url: string) {
  if (!url.startsWith('blob:')) return
  URL.revokeObjectURL(url)
  impressionPreviewObjectUrls.delete(url)
}

function revokeAllImpressionPreviewObjectUrls() {
  for (const url of impressionPreviewObjectUrls) URL.revokeObjectURL(url)
  impressionPreviewObjectUrls.clear()
  for (const key of Object.keys(impressionImagePreviews)) delete impressionImagePreviews[key]
}

function createImpressionUpdatePayload() {
  return form.impressions.map((impression) => ({
    slot: impression.slot,
    active: impression.active,
    title: impression.title,
    text: impression.text,
    trp3_icon: impression.trp3_icon,
    icon_image_url: impression.icon_image_url,
    image_url: impression.image_url,
  }))
}

async function saveCard(returnToDetail = false): Promise<CharacterCard | null> {
  if (saving.value || imageUploadInProgress.value || !card.value) return null
  saving.value = true
  try {
    const result = await updateCharacterCard(card.value.id, {
      ...form,
      impressions: createImpressionUpdatePayload(),
    })
    applyCard(result)
    toast.success(
      result.status === 'published' && result.visibility === 'public' && result.review_status === 'pending'
        ? t('characterCards.editor.savedPending')
        : t('characterCards.editor.saved'),
      result.review_status === 'pending' ? 6000 : 3000,
    )
    if (returnToDetail) await router.push(`/character-cards/${result.id}`)
    return result
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.saveFailed'))
    return null
  } finally {
    saving.value = false
  }
}

async function syncFromBackup() {
  if (!card.value || !canSyncFromBackup.value || syncing.value) return
  const confirmed = await dialog.confirm({
    title: t('characterCards.editor.refreshTitle'),
    message: t('characterCards.editor.refreshMessage'),
    type: 'warning',
    confirmText: t('characterCards.editor.refreshConfirm'),
  })
  if (!confirmed) return

  syncing.value = true
  try {
    const result = await syncCharacterCardFromTRP3(card.value.id)
    applySyncedCard(result)
    toast.success(t('characterCards.editor.refreshed'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.refreshFailed'))
  } finally {
    syncing.value = false
  }
}

function createDefaultProfileId() {
  return `RPBOX_${card.value?.id || 'NEW'}_${Date.now().toString(36).toUpperCase()}`
}

async function openWriteBackSetup() {
  if (!card.value || writeBackDisabledReason.value || writingBack.value) return
  writingBack.value = true
  try {
    const [localResult, backups] = await Promise.all([
      invoke<{ accounts?: LocalAccountOption[] }>('scan_profiles', { wowPath: wowPath.value }),
      listAccountBackups().catch(() => []),
    ])
    localAccounts.value = localResult.accounts || []
    cloudBackups.value = backups
    writeBackAccountId.value = (
      localAccounts.value.find((account) => account.account_id === card.value?.source_account_id)?.account_id
      || localAccounts.value[0]?.account_id
      || ''
    )
    syncCloudAfterWriteBack.value = cloudBackups.value.some(
      (backup) => backup.account_id === writeBackAccountId.value,
    )
    writeBackProfileId.value = card.value.source_profile_id || createDefaultProfileId()
    writeBackSnapshotName.value = t('characterCards.editor.snapshotDefault', { name: displayName.value })
    writeBackSetupOpen.value = true
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.accountsFailed'))
  } finally {
    writingBack.value = false
  }
}

async function previewTRP3Lua() {
  if (!card.value || luaPreviewLoading.value) return
  luaPreviewLoading.value = true
  luaPreviewOpen.value = true
  luaPreview.value = ''
  try {
    const result = await getCharacterCardTRP3Lua(card.value.id)
    luaPreview.value = result.lua
  } catch (error: unknown) {
    luaPreviewOpen.value = false
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.luaFailed'))
  } finally {
    luaPreviewLoading.value = false
  }
}

async function writeBackToLocalTRP3() {
  if (!card.value || writeBackDisabledReason.value || writingBack.value) return
  if (!writeBackAccountId.value || !writeBackProfileId.value.trim()) {
    toast.warning(t('characterCards.editor.writeTargetRequired'))
    return
  }
  const confirmed = await dialog.confirm({
    title: t('characterCards.editor.writeConfirmTitle'),
    message: t('characterCards.editor.writeConfirmMessage', {
      account: writeBackAccountId.value,
      profile: writeBackProfileId.value.trim(),
      cloudAction: t(writeBackBackup.value && syncCloudAfterWriteBack.value
        ? 'characterCards.editor.cloudWillSync'
        : 'characterCards.editor.cloudWillNotSync'),
    }),
    type: 'error',
    confirmText: t('characterCards.editor.writeConfirm'),
  })
  if (!confirmed) return

  writingBack.value = true
  try {
    const savedCard = isDirty.value ? await saveCard(false) : card.value
    if (!savedCard) return
    const targetProfileId = writeBackProfileId.value.trim()
    const exported = await getCharacterCardTRP3Lua(savedCard.id)
    await invoke('write_character_card_profile', {
      wowPath: wowPath.value,
      accountId: writeBackAccountId.value,
      profileId: targetProfileId,
      profile: exported.profile,
      snapshotName: writeBackSnapshotName.value.trim() || null,
    })
    writeBackProfileId.value = targetProfileId
    writeBackSetupOpen.value = false

    const backup = writeBackBackup.value
    if (backup && syncCloudAfterWriteBack.value) {
      try {
        await writeBackCharacterCardToTRP3(savedCard.id, {
          backup_id: backup.id,
          profile_id: targetProfileId,
          snapshot_name: writeBackSnapshotName.value.trim() || undefined,
        })
        if (card.value?.id === savedCard.id) {
          card.value = {
            ...card.value,
            source_backup_id: backup.id,
            source_account_id: backup.account_id,
            source_profile_id: targetProfileId,
          }
        }
        toast.success(t('characterCards.editor.writeCloudSuccess'), 7000)
      } catch (error: unknown) {
        const reason = error instanceof Error ? error.message : t('characterCards.editor.cloudSyncFailed')
        toast.warning(t('characterCards.editor.localSuccessCloudFailed', { reason }), 8000)
      }
      return
    }

    toast.success(t('characterCards.editor.localSuccess'), 7000)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.editor.localFailed'))
  } finally {
    writingBack.value = false
  }
}

async function goBack() {
  if (isDirty.value) {
    const confirmed = await dialog.confirm({
      title: t('characterCards.editor.leaveTitle'),
      message: t('characterCards.editor.leaveMessage'),
      type: 'warning',
      confirmText: t('characterCards.editor.discard'),
    })
    if (!confirmed) return
  }
  if (card.value) {
    await router.push(`/character-cards/${card.value.id}`)
    return
  }
  router.back()
}
</script>

<template>
  <main class="editor-page">
    <div v-if="loading" class="editor-state" role="status">
      <i class="ri-loader-4-line spin" aria-hidden="true"></i>
      <span>{{ t('characterCards.editor.loading') }}</span>
    </div>

    <div v-else-if="loadError" class="editor-state editor-state--error" role="alert">
      <i class="ri-file-warning-line" aria-hidden="true"></i>
      <h1>{{ t('characterCards.editor.unavailableTitle') }}</h1>
      <p>{{ loadError }}</p>
      <button type="button" @click="goBack">{{ t('characterCards.common.back') }}</button>
    </div>

    <template v-else-if="card">
      <header class="editor-header">
        <button type="button" class="editor-header__back" @click="goBack">
          <i class="ri-arrow-left-line" aria-hidden="true"></i>
          {{ t('characterCards.editor.backCard') }}
        </button>
        <div class="editor-header__identity">
          <span>{{ t('characterCards.editor.fileKicker', { id: card.id }) }}</span>
          <h1 :style="displayNameColor ? { color: displayNameColor } : undefined">{{ displayName }}</h1>
        </div>
        <div class="editor-header__actions">
          <span v-if="isDirty" class="unsaved-mark"><i class="ri-circle-fill" aria-hidden="true"></i>{{ t('characterCards.editor.unsaved') }}</span>
          <button type="button" class="button button--quiet" :disabled="saving || imageUploadInProgress" @click="saveCard(false)">
            {{ saving ? t('characterCards.common.saving') : t('characterCards.common.save') }}
          </button>
          <button type="button" class="button button--primary" :disabled="saving || imageUploadInProgress" @click="saveCard(true)">
            {{ t('characterCards.editor.saveAndView') }}
          </button>
        </div>
      </header>

      <div class="editor-layout">
        <aside class="portrait-editor" :aria-label="t('characterCards.editor.portraitEditorAria')">
          <div class="portrait-editor__rail" aria-hidden="true"></div>
          <div class="portrait-editor__frame">
            <img
              v-if="uploadedPortraitPreview"
              class="portrait-editor__image"
              :src="uploadedPortraitPreview"
              :alt="t('characterCards.common.portraitPreviewAlt', { name: displayName })"
            />
            <CharacterCardGalleryImage
              v-else-if="selectedPortrait"
              class="portrait-editor__image"
              :card="card"
              :portrait="selectedPortrait"
              :alt="t('characterCards.common.portraitPreviewAlt', { name: displayName })"
              :width="900"
              :quality="90"
            />
            <div v-else class="portrait-editor__empty">
              <i class="ri-user-star-line" aria-hidden="true"></i>
              <strong>{{ t('characterCards.editor.portraitTitle') }}</strong>
              <span>{{ t('characterCards.editor.portraitHint') }}</span>
            </div>
            <div class="portrait-editor__tools">
              <button type="button" :disabled="portraitUploading" @click="triggerPortraitUpload">
                <i :class="portraitUploading ? 'ri-loader-4-line spin' : 'ri-camera-line'" aria-hidden="true"></i>
                {{ hasPortrait ? t('characterCards.editor.add') : t('characterCards.editor.upload') }}
              </button>
              <button v-if="selectedPortrait" type="button" @click="openPortraitPreview(selectedPortrait)">
                <i class="ri-focus-mode" aria-hidden="true"></i>{{ t('characterCards.editor.preview') }}
              </button>
              <button v-if="selectedPortrait" type="button" class="danger" @click="removePortrait(selectedPortrait)">
                <i class="ri-delete-bin-line" aria-hidden="true"></i>{{ t('characterCards.common.remove') }}
              </button>
            </div>
          </div>
          <input ref="portraitInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" multiple hidden @change="handlePortraitFile" />

          <div v-if="portraits.length" class="portrait-film" :aria-label="t('characterCards.editor.portraitFilmAria')">
            <article
              v-for="(portrait, index) in portraits"
              :key="portrait.id"
              class="portrait-film__cell"
              :class="{ active: selectedPortrait?.id === portrait.id }"
            >
              <button
                type="button"
                class="portrait-film__thumb"
                :aria-label="t('characterCards.editor.portraitItemAria', { index: index + 1, cover: portrait.is_cover ? t('characterCards.editor.currentCoverSuffix') : '' })"
                :aria-pressed="selectedPortrait?.id === portrait.id"
                @click="selectedPortraitId = portrait.id"
              >
                <CharacterCardGalleryImage :card="card" :portrait="portrait" alt="" :width="220" :quality="72" />
                <span v-if="portrait.is_cover">{{ t('characterCards.editor.cover') }}</span>
              </button>
              <div class="portrait-film__actions">
                <button type="button" :disabled="index === 0 || portraitUploading" :aria-label="t('characterCards.editor.movePrevious')" @click="movePortrait(index, -1)"><i class="ri-arrow-left-line"></i></button>
                <button type="button" :disabled="index === portraits.length - 1 || portraitUploading" :aria-label="t('characterCards.editor.moveNext')" @click="movePortrait(index, 1)"><i class="ri-arrow-right-line"></i></button>
                <button type="button" :disabled="portrait.is_cover || portrait.id <= 0 || portraitUploading" :aria-label="t('characterCards.editor.setCover')" @click="setPortraitCover(portrait)"><i class="ri-bookmark-line"></i></button>
                <button type="button" :disabled="portraitUploading" :aria-label="t('characterCards.editor.deleteImage')" @click="removePortrait(portrait)"><i class="ri-delete-bin-line"></i></button>
              </div>
            </article>
          </div>
          <p class="portrait-film__hint">{{ t('characterCards.editor.filmHint') }}</p>

          <div class="portrait-editor__plaque">
            <strong :style="displayNameColor ? { color: displayNameColor } : undefined">{{ displayName }}</strong>
            <span>{{ form.title || form.full_title || t('characterCards.editor.titlePending') }}</span>
            <small>{{ [form.race, form.class].filter(Boolean).join(' · ') || t('characterCards.editor.identityPending') }}</small>
          </div>

          <div class="source-info">
            <span><i class="ri-link-m" aria-hidden="true"></i>{{ t('characterCards.editor.source') }}</span>
            <strong v-if="card.source_profile_id">{{ card.source_account_id || t('characterCards.common.sourceCloud') }} · {{ card.source_profile_id }}</strong>
            <strong v-else>{{ t('characterCards.editor.standalone') }}</strong>
            <button v-if="canSyncFromBackup" type="button" :disabled="syncing" @click="syncFromBackup">
              <i :class="syncing ? 'ri-loader-4-line spin' : 'ri-refresh-line'" aria-hidden="true"></i>
              {{ syncing ? t('characterCards.editor.refreshing') : t('characterCards.editor.refreshAction') }}
            </button>
          </div>

          <div class="local-writeback" :class="{ disabled: Boolean(writeBackDisabledReason) }">
            <span><i class="ri-hard-drive-3-line" aria-hidden="true"></i>{{ t('characterCards.editor.desktopInterop') }}</span>
            <p>{{ writeBackDisabledReason || t('characterCards.editor.writeSafetyHint') }}</p>
            <div class="local-writeback__actions">
              <button type="button" :disabled="luaPreviewLoading" @click="previewTRP3Lua">{{ t('characterCards.editor.previewLua') }}</button>
              <button type="button" :disabled="Boolean(writeBackDisabledReason) || writingBack" @click="openWriteBackSetup">
                {{ writingBack ? t('characterCards.editor.readingAccount') : t('characterCards.editor.writeLocal') }}
              </button>
              <button type="button" :disabled="Boolean(writeBackDisabledReason)" @click="writeBackHistoryOpen = true">{{ t('characterCards.editor.localVersions') }}</button>
            </div>
          </div>
        </aside>

        <section class="editor-ledger">
          <nav class="editor-tabs" role="tablist" :aria-label="t('characterCards.tabs.editorAria')">
            <button
              v-for="tab in tabs"
              :id="`character-tab-${tab.id}`"
              :key="tab.id"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.id"
              :aria-controls="`character-panel-${tab.id}`"
              :tabindex="activeTab === tab.id ? 0 : -1"
              :class="{ active: activeTab === tab.id }"
              @click="selectTab(tab.id)"
              @keydown="handleTabKeydown($event, tab.id)"
            >
              <i :class="tab.icon" aria-hidden="true"></i>
              {{ tab.label }}
            </button>
          </nav>

          <section
            v-show="activeTab === 'basic'"
            id="character-panel-basic"
            class="editor-panel"
            role="tabpanel"
            aria-labelledby="character-tab-basic"
          >
            <header class="panel-heading">
              <div><span>{{ t('characterCards.editor.basicKicker') }}</span><h2>{{ t('characterCards.editor.basicTitle') }}</h2></div>
              <p>{{ t('characterCards.editor.basicBody') }}</p>
            </header>

            <div class="form-section">
              <h3>{{ t('characterCards.editor.nameTitle') }}</h3>
              <div class="form-grid form-grid--three">
                <label><span>{{ t('characterCards.editor.fields.firstName') }}</span><input v-model="form.first_name" maxlength="128" autocomplete="off" /></label>
                <label><span>{{ t('characterCards.editor.fields.lastName') }}</span><input v-model="form.last_name" maxlength="128" autocomplete="off" /></label>
                <label><span>{{ t('characterCards.editor.fields.displayName') }}</span><input v-model="form.display_name" maxlength="256" :placeholder="t('characterCards.editor.fields.displayNamePlaceholder')" /></label>
                <label><span>{{ t('characterCards.editor.fields.title') }}</span><input v-model="form.title" maxlength="128" /></label>
                <label class="form-span-two"><span>{{ t('characterCards.editor.fields.fullTitle') }}</span><input v-model="form.full_title" maxlength="256" /></label>
              </div>
            </div>

            <div class="form-section">
              <h3>{{ t('characterCards.editor.identityTitle') }}</h3>
              <div class="form-grid form-grid--three">
                <label><span>{{ t('characterCards.editor.fields.race') }}</span><input v-model="form.race" maxlength="64" /></label>
                <label><span>{{ t('characterCards.editor.fields.class') }}</span><input v-model="form.class" maxlength="64" /></label>
                <label><span>{{ t('characterCards.editor.fields.age') }}</span><input v-model="form.age" maxlength="64" /></label>
                <label><span>{{ t('characterCards.editor.fields.height') }}</span><input v-model="form.height" maxlength="64" /></label>
                <label><span>{{ t('characterCards.editor.fields.weight') }}</span><input v-model="form.weight" maxlength="64" /></label>
                <label><span>{{ t('characterCards.editor.fields.relationship') }}</span><input v-model="form.relationship_status" maxlength="64" :placeholder="t('characterCards.editor.fields.relationshipPlaceholder')" /></label>
                <label><span>{{ t('characterCards.editor.fields.birthplace') }}</span><input v-model="form.birthplace" maxlength="256" /></label>
                <label><span>{{ t('characterCards.editor.fields.residence') }}</span><input v-model="form.residence" maxlength="256" /></label>
                <label><span>{{ t('characterCards.editor.fields.trp3Icon') }}</span><input v-model="form.icon" maxlength="128" /></label>
              </div>
            </div>

            <div class="form-section">
              <h3>{{ t('characterCards.editor.colorsTitle') }}</h3>
              <div class="form-grid form-grid--three">
                <label><span>{{ t('characterCards.editor.fields.eyeColor') }}</span><input v-model="form.eye_color" maxlength="64" /></label>
                <CharacterCardColorField
                  v-model="form.eye_color_hex"
                  field-id="character-eye-color"
                  :label="t('characterCards.editor.eyeColorValue')"
                  :hint="t('characterCards.editor.eyeColorHint')"
                />
                <CharacterCardColorField
                  v-model="form.class_color"
                  field-id="character-class-color"
                  :label="t('characterCards.editor.classColor')"
                  :hint="t('characterCards.editor.classColorHint')"
                />
                <CharacterCardColorField
                  v-model="form.name_color"
                  field-id="character-name-color"
                  :label="t('characterCards.editor.nameColor')"
                  :hint="t('characterCards.editor.nameColorHint')"
                />
              </div>
            </div>

            <div class="form-section form-section--rpbox">
              <h3>{{ t('characterCards.editor.displayTitle') }}</h3>
              <label class="summary-field">
                <span>{{ t('characterCards.editor.summary') }}</span>
                <textarea v-model="form.summary" rows="4" maxlength="1000" :placeholder="t('characterCards.editor.summaryPlaceholder')"></textarea>
                <small>{{ form.summary.length }} / 1000</small>
              </label>
              <div class="visibility-grid">
                <label>
                  <span>{{ t('characterCards.editor.productionStatus') }}</span>
                  <select v-model="form.status">
                    <option value="draft">{{ t('characterCards.common.status.draft') }}</option>
                    <option value="published">{{ t('characterCards.common.status.published') }}</option>
                  </select>
                  <small>{{ t('characterCards.editor.draftHint') }}</small>
                </label>
                <label>
                  <span>{{ t('characterCards.editor.visibility') }}</span>
                  <select v-model="form.visibility">
                    <option value="private">{{ t('characterCards.common.status.private') }}</option>
                    <option value="public">{{ t('characterCards.common.status.public') }}</option>
                  </select>
                  <small>{{ t('characterCards.editor.publicHint') }}</small>
                </label>
              </div>
              <div v-if="form.status === 'published' && form.visibility === 'public'" class="review-notice" :class="`review-notice--${card.review_status || 'pending'}`">
                <i :class="card.review_status === 'rejected' ? 'ri-close-circle-line' : card.review_status === 'approved' ? 'ri-checkbox-circle-line' : 'ri-time-line'" aria-hidden="true"></i>
                <div>
                  <strong>{{ card.review_status === 'rejected' ? t('characterCards.editor.reviewRejected') : card.review_status === 'approved' ? t('characterCards.editor.reviewApproved') : t('characterCards.editor.reviewOnSave') }}</strong>
                  <span v-if="card.review_status === 'rejected' && card.review_comment">{{ t('characterCards.editor.moderatorComment', { comment: card.review_comment }) }}</span>
                  <span v-else-if="publicReviewPending">{{ t('characterCards.editor.pendingBody') }}</span>
                  <span v-else>{{ t('characterCards.editor.reviewBody') }}</span>
                </div>
              </div>
            </div>
          </section>

          <section
            v-show="activeTab === 'background'"
            id="character-panel-background"
            class="editor-panel editor-panel--rich"
            role="tabpanel"
            aria-labelledby="character-tab-background"
          >
            <header class="panel-heading">
              <div><span>{{ t('characterCards.editor.backgroundKicker') }}</span><h2>{{ t('characterCards.editor.backgroundTitle') }}</h2></div>
              <p>{{ t('characterCards.editor.backgroundBody') }}</p>
            </header>
            <TiptapEditor ref="backgroundEditor" v-model="form.background_story" :placeholder="t('characterCards.editor.backgroundPlaceholder')">
              <template #toolbar>
                <button type="button" class="internal-link-button" :title="t('characterCards.editor.insertInternal')" @mousedown.prevent @click="openQuickJump('background')">
                  <i class="ri-links-line"></i><span>{{ t('characterCards.editor.internalLink') }}</span>
                </button>
              </template>
            </TiptapEditor>
          </section>

          <section
            v-show="activeTab === 'impression'"
            id="character-panel-impression"
            class="editor-panel editor-panel--impression"
            role="tabpanel"
            aria-labelledby="character-tab-impression"
          >
            <header class="panel-heading">
              <div><span>{{ t('characterCards.editor.impressionKicker') }}</span><h2>{{ t('characterCards.editor.impressionTitle') }}</h2></div>
              <p>{{ t('characterCards.editor.impressionBody') }}</p>
            </header>

            <div class="observation-register" :aria-label="t('characterCards.editor.impressionAria')">
              <article
                v-for="(impression, index) in form.impressions"
                :key="impression.slot"
                class="observation-record"
                :class="{ 'observation-record--active': impression.active }"
              >
                <header class="observation-record__header">
                  <div class="observation-record__index" aria-hidden="true">
                    <span>{{ t('characterCards.editor.observationLabel') }}</span>
                    <strong>{{ String(impression.slot).padStart(2, '0') }}</strong>
                  </div>
                  <label class="observation-toggle">
                    <input v-model="impression.active" type="checkbox" />
                    <span class="observation-toggle__track" aria-hidden="true"><i></i></span>
                    <span>{{ impression.active ? t('characterCards.editor.impressionEnabled') : t('characterCards.editor.impressionDisabled') }}</span>
                  </label>
                </header>

                <div class="observation-record__body">
                  <section class="observation-icon-station" :aria-label="t('characterCards.editor.iconSlotAria', { slot: impression.slot })">
                    <CharacterCardImpressionMark
                      :icon-image-url="impression.icon_image_url"
                      :preview-url="getImpressionPreview(index, 'icon')"
                      :trp3-icon="impression.trp3_icon"
                      :fallback-label="String(impression.slot)"
                      :size="76"
                    />
                    <div class="observation-media-actions">
                      <button
                        type="button"
                        :disabled="isImpressionUploading(index, 'icon')"
                        @click="triggerImpressionUpload(index, 'icon')"
                      >
                        <i :class="isImpressionUploading(index, 'icon') ? 'ri-loader-4-line spin' : 'ri-upload-2-line'" aria-hidden="true"></i>
                        {{ isImpressionUploading(index, 'icon') ? t('characterCards.common.uploading') : (impression.icon_image_url ? t('characterCards.editor.replaceIcon') : t('characterCards.editor.customIcon')) }}
                      </button>
                      <button
                        v-if="impression.icon_image_url || getImpressionPreview(index, 'icon')"
                        type="button"
                        class="observation-media-actions__remove"
                        :disabled="isImpressionUploading(index, 'icon')"
                        @click="removeImpressionImage(index, 'icon')"
                      >{{ t('characterCards.common.remove') }}</button>
                    </div>
                  </section>

                  <div class="observation-fields">
                    <label>
                      <span>{{ t('characterCards.editor.attributeName') }}</span>
                      <input v-model="impression.title" maxlength="80" :placeholder="t('characterCards.editor.attributePlaceholder')" />
                    </label>
                    <label>
                      <span>{{ t('characterCards.editor.shortDescription') }}</span>
                      <textarea
                        v-model="impression.text"
                        rows="3"
                        maxlength="500"
                        :placeholder="t('characterCards.editor.descriptionPlaceholder')"
                      ></textarea>
                      <small>{{ impression.text.length }} / 500</small>
                    </label>
                    <label>
                      <span>{{ t('characterCards.editor.trp3IconFallback') }}</span>
                      <input v-model="impression.trp3_icon" maxlength="128" :placeholder="t('characterCards.editor.trp3IconPlaceholder')" />
                    </label>
                  </div>

                  <section class="observation-image-station" :aria-label="t('characterCards.editor.impressionImageSlotAria', { slot: impression.slot })">
                    <div class="observation-image-station__frame">
                      <AuthenticatedImage
                        v-if="getImpressionImageUrl(index)"
                        class="observation-image-station__media"
                        :src="getImpressionImageUrl(index)"
                        :alt="t('characterCards.common.impressionImageAlt', { title: impression.title || t('characterCards.common.impressionFallbackTitle', { slot: impression.slot }) })"
                      />
                      <div v-else class="observation-image-station__empty">
                        <i class="ri-landscape-line" aria-hidden="true"></i>
                        <span>{{ t('characterCards.editor.optionalImage') }}</span>
                      </div>
                    </div>
                    <div class="observation-media-actions">
                      <button
                        type="button"
                        :disabled="isImpressionUploading(index, 'image')"
                        @click="triggerImpressionUpload(index, 'image')"
                      >
                        <i :class="isImpressionUploading(index, 'image') ? 'ri-loader-4-line spin' : 'ri-image-add-line'" aria-hidden="true"></i>
                        {{ isImpressionUploading(index, 'image') ? t('characterCards.common.uploading') : (impression.image_url ? t('characterCards.editor.replaceImage') : t('characterCards.editor.uploadImage')) }}
                      </button>
                      <button
                        v-if="impression.image_url || getImpressionPreview(index, 'image')"
                        type="button"
                        class="observation-media-actions__remove"
                        :disabled="isImpressionUploading(index, 'image')"
                        @click="removeImpressionImage(index, 'image')"
                      >{{ t('characterCards.common.remove') }}</button>
                    </div>
                  </section>
                </div>
              </article>
            </div>

            <input
              ref="impressionImageInput"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              hidden
              @change="handleImpressionImageFile"
            />

            <section class="impression-notes">
              <header>
                <div><span>{{ t('characterCards.editor.supplementKicker') }}</span><h3>{{ t('characterCards.editor.otherNotes') }}</h3></div>
                <p>{{ t('characterCards.editor.supplementBody') }}</p>
              </header>
              <TiptapEditor ref="impressionEditor" v-model="form.first_impression" :placeholder="t('characterCards.editor.supplementPlaceholder')">
                <template #toolbar>
                  <button type="button" class="internal-link-button" :title="t('characterCards.editor.insertInternal')" @mousedown.prevent @click="openQuickJump('impression')">
                    <i class="ri-links-line"></i><span>{{ t('characterCards.editor.internalLink') }}</span>
                  </button>
                </template>
              </TiptapEditor>
            </section>
          </section>

          <section
            v-show="activeTab === 'other'"
            id="character-panel-other"
            class="editor-panel editor-panel--rich"
            role="tabpanel"
            aria-labelledby="character-tab-other"
          >
            <header class="panel-heading">
              <div><span>{{ t('characterCards.editor.otherKicker') }}</span><h2>{{ t('characterCards.editor.otherTitle') }}</h2></div>
              <p>{{ t('characterCards.editor.otherBody') }}</p>
            </header>
            <TiptapEditor ref="otherEditor" v-model="form.other_content" :placeholder="t('characterCards.editor.otherPlaceholder')">
              <template #toolbar>
                <button type="button" class="internal-link-button" :title="t('characterCards.editor.insertInternal')" @mousedown.prevent @click="openQuickJump('other')">
                  <i class="ri-links-line"></i><span>{{ t('characterCards.editor.internalLink') }}</span>
                </button>
              </template>
            </TiptapEditor>
          </section>
        </section>
      </div>

      <footer class="save-dock">
        <div>
          <strong>{{ imageUploadInProgress ? t('characterCards.editor.uploadPending') : (isDirty ? t('characterCards.editor.unsavedChanges') : t('characterCards.editor.allSaved')) }}</strong>
          <span>{{ imageUploadInProgress ? t('characterCards.editor.uploadPendingBody') : t('characterCards.editor.saveBody') }}</span>
        </div>
        <button type="button" class="button button--primary" :disabled="saving || imageUploadInProgress || !isDirty" @click="saveCard(false)">
          <i class="ri-save-3-line" aria-hidden="true"></i>{{ saving ? t('characterCards.common.saving') : t('characterCards.editor.saveWhole') }}
        </button>
      </footer>

      <ImageCropperDialog
        v-model="cropperOpen"
        :file="cropperFile"
        :aspect-ratio="3 / 4"
        :output-width="1200"
        :output-height="1600"
        :max-size-k-b="2048"
        :title="t('characterCards.editor.adjustPortrait')"
        @cropped="handlePortraitCropped"
        @error="handleCropperError"
      />

      <RModal v-model="portraitPreviewOpen" :title="t('characterCards.editor.portraitModalTitle', { name: displayName })" width="680px">
        <div class="portrait-lightbox">
          <img v-if="uploadedPortraitPreview" :src="uploadedPortraitPreview" :alt="t('characterCards.common.portraitLargeAlt', { name: displayName })" />
          <CharacterCardGalleryImage
            v-else-if="selectedPortrait"
            :card="card"
            :portrait="selectedPortrait"
            :alt="t('characterCards.common.portraitLargeAlt', { name: displayName })"
            :width="1200"
            :quality="92"
          />
        </div>
      </RModal>

      <RModal v-model="writeBackSetupOpen" :title="t('characterCards.editor.writeSetupTitle')" width="620px">
        <div class="writeback-sheet">
          <header>
            <span>{{ t('characterCards.editor.writeKicker') }}</span>
            <h3>{{ t('characterCards.editor.writeLocation') }}</h3>
            <p>{{ t('characterCards.editor.writeLocationBody') }}</p>
          </header>
          <label>
            <span>{{ t('characterCards.editor.localAccount') }}</span>
            <select v-model="writeBackAccountId">
              <option value="" disabled>{{ t('characterCards.editor.selectAccount') }}</option>
              <option v-for="account in localAccounts" :key="account.account_id" :value="account.account_id">{{ account.account_id }}</option>
            </select>
          </label>
          <label>
            <span>TRP3 profile ID</span>
            <input v-model.trim="writeBackProfileId" maxlength="128" autocomplete="off" />
            <small>{{ card.source_profile_id ? t('characterCards.editor.sourceIdHint') : t('characterCards.editor.generatedIdHint') }}</small>
          </label>
          <label>
            <span>{{ t('characterCards.editor.snapshotName') }}</span>
            <input v-model.trim="writeBackSnapshotName" maxlength="120" autocomplete="off" />
            <small>{{ t('characterCards.editor.snapshotHint') }}</small>
          </label>
          <div class="writeback-sheet__account-note">
            <i :class="writeBackBackup ? 'ri-cloud-line' : 'ri-hard-drive-2-line'" aria-hidden="true"></i>
            {{ writeBackBackup ? t('characterCards.editor.cloudBackupFound', { version: writeBackBackup.version }) : t('characterCards.editor.cloudBackupMissing') }}
          </div>
          <label v-if="writeBackBackup" class="writeback-sheet__cloud-option">
            <input v-model="syncCloudAfterWriteBack" type="checkbox" />
            <span>{{ t('characterCards.editor.syncCloud') }}</span>
            <small>{{ t('characterCards.editor.syncCloudHint') }}</small>
          </label>
          <footer>
            <button type="button" class="button button--quiet" @click="writeBackSetupOpen = false">{{ t('characterCards.common.cancel') }}</button>
            <button type="button" class="button button--primary" :disabled="writingBack || !writeBackAccountId || !writeBackProfileId.trim()" @click="writeBackToLocalTRP3">
              <i :class="writingBack ? 'ri-loader-4-line spin' : 'ri-file-transfer-line'" aria-hidden="true"></i>
              {{ writingBack ? t('characterCards.editor.writing') : t('characterCards.editor.continueWrite') }}
            </button>
          </footer>
        </div>
      </RModal>

      <RModal v-model="luaPreviewOpen" :title="t('characterCards.editor.luaPreviewTitle')" width="760px">
        <div class="lua-preview">
          <p v-if="isDirty"><i class="ri-information-line"></i> {{ t('characterCards.editor.unsavedPreviewHint') }}</p>
          <div v-if="luaPreviewLoading" class="lua-preview__loading"><i class="ri-loader-4-line spin"></i>{{ t('characterCards.editor.generatingLua') }}</div>
          <pre v-else>{{ luaPreview }}</pre>
        </div>
      </RModal>

      <RModal v-model="writeBackHistoryOpen" :title="t('characterCards.editor.localHistoryTitle')" width="820px">
        <LocalTRP3VersionHistory
          v-if="historyAccountId && wowPath"
          :wow-path="wowPath"
          :account-id="historyAccountId"
        />
        <div v-else class="lua-preview__loading">{{ t('characterCards.editor.historySelectHint') }}</div>
      </RModal>

      <PostQuickJump v-model="quickJumpOpen" :on-insert="insertQuickJump" />
    </template>
  </main>
</template>

<style scoped>
.editor-page {
  --ink: var(--color-text-main);
  --walnut: var(--color-primary);
  --copper: var(--color-accent);
  --rust: var(--color-secondary);
  --paper: var(--color-panel-bg);
  --line: var(--color-border);
  --muted: var(--color-text-secondary);
  width: min(1320px, calc(100% - 36px));
  margin: 0 auto;
  padding: 20px 0 86px;
  color: var(--ink);
  container-type: inline-size;
}

.editor-state {
  display: grid;
  min-height: 60vh;
  place-content: center;
  gap: 12px;
  color: var(--muted);
  text-align: center;
}
.editor-state > i { color: var(--copper); font-size: 36px; }
.editor-state h1 { margin: 0; color: var(--ink); font-family: Georgia, 'Noto Serif SC', serif; }
.editor-state p { margin: 0; }
.editor-state button { justify-self: center; padding: 9px 18px; border: 1px solid var(--btn-primary-bg); border-radius: 7px; background: var(--btn-primary-bg); color: var(--btn-primary-text); cursor: pointer; }

.editor-header {
  display: grid;
  grid-template-columns: minmax(170px, 1fr) auto minmax(270px, 1fr);
  align-items: center;
  gap: 18px;
  margin-bottom: 16px;
  padding: 12px 16px;
  border: 1px solid var(--line);
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-panel-bg) 92%, transparent);
  box-shadow: var(--shadow-sm);
  backdrop-filter: blur(12px);
}

.editor-header__back {
  display: inline-flex;
  justify-self: start;
  align-items: center;
  gap: 7px;
  padding: 8px 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.editor-header__identity { min-width: 0; text-align: center; }
.editor-header__identity span { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: 0.16em; text-transform: uppercase; }
.editor-header__identity h1 { overflow: hidden; margin: 3px 0 0; font-family: Georgia, 'Noto Serif SC', serif; font-size: 21px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.editor-header__actions { display: flex; justify-content: flex-end; align-items: center; gap: 8px; }

.unsaved-mark { display: inline-flex; align-items: center; gap: 5px; color: var(--color-warning-dark); font-size: 10px; }
.unsaved-mark i { font-size: 6px; }

.button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 15px;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: var(--btn-outline-hover);
  color: var(--walnut);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  font-weight: 700;
}
.button--primary { border-color: var(--btn-primary-bg); background: var(--btn-primary-bg); color: var(--btn-primary-text); }
.button:disabled { cursor: not-allowed; opacity: 0.5; }

.editor-layout {
  display: grid;
  grid-template-columns: minmax(240px, 300px) minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.portrait-editor,
.editor-ledger {
  border: 1px solid var(--line);
  border-radius: 14px;
  background: var(--paper);
  box-shadow: var(--shadow-md);
}

.portrait-editor {
  position: sticky;
  top: 14px;
  overflow: hidden;
  padding: 18px;
}

.portrait-editor__rail {
  position: absolute;
  top: 0;
  left: 18px;
  right: 18px;
  height: 5px;
  border-radius: 0 0 5px 5px;
  background: linear-gradient(90deg, var(--gradient-start), var(--color-accent), var(--gradient-end));
}

.portrait-editor__frame {
  position: relative;
  overflow: hidden;
  aspect-ratio: 3 / 4;
  border: 6px solid var(--gradient-end);
  border-radius: 6px 6px 0 0;
  background: var(--gradient-end);
  box-shadow: var(--shadow-lg);
}
.portrait-editor__image { width: 100%; height: 100%; display: block; object-fit: cover; }
.portrait-editor__empty { display: grid; width: 100%; height: 100%; place-content: center; gap: 7px; background: radial-gradient(circle at center, color-mix(in srgb, var(--color-accent) 28%, transparent), transparent 48%), var(--gradient-end); color: var(--gradient-text); text-align: center; }
.portrait-editor__empty i { font-size: 50px; }
.portrait-editor__empty strong { font-family: Georgia, 'Noto Serif SC', serif; }
.portrait-editor__empty span { color: var(--gradient-text-muted); font-size: 10px; }

.portrait-editor__tools {
  position: absolute;
  right: 8px;
  bottom: 8px;
  left: 8px;
  display: flex;
  justify-content: center;
  gap: 5px;
  padding: 7px;
  border: 1px solid var(--gradient-border);
  border-radius: 7px;
  background: color-mix(in srgb, var(--gradient-end) 82%, transparent);
  backdrop-filter: blur(8px);
}
.portrait-editor__tools button { display: inline-flex; align-items: center; gap: 4px; padding: 6px 7px; border: 0; border-radius: 5px; background: var(--gradient-surface); color: var(--gradient-text); cursor: pointer; font-size: 10px; }
.portrait-editor__tools button.danger { color: color-mix(in srgb, var(--btn-danger-bg) 45%, var(--gradient-text)); }

.portrait-film {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  padding: 9px 8px;
  overflow-x: auto;
  border: 1px solid var(--gradient-border);
  border-radius: 8px;
  background:
    linear-gradient(90deg, transparent 6px, var(--gradient-border) 7px, transparent 8px) 0 0 / 18px 100%,
    var(--gradient-end);
  scrollbar-width: thin;
}
.portrait-film__cell { flex: 0 0 78px; min-width: 0; }
.portrait-film__thumb {
  position: relative;
  display: block;
  width: 78px;
  height: 98px;
  overflow: hidden;
  padding: 0;
  border: 2px solid var(--gradient-border);
  border-radius: 4px;
  background: var(--gradient-start);
  color: var(--gradient-text);
  cursor: pointer;
}
.portrait-film__thumb :deep(img),
.portrait-film__thumb :deep(.authenticated-image) { width: 100%; height: 100%; object-fit: cover; }
.portrait-film__thumb > span { position: absolute; right: 3px; bottom: 3px; padding: 2px 4px; border-radius: 3px; background: var(--color-accent); color: var(--color-accent-contrast); font-size: 8px; font-weight: 800; }
.portrait-film__cell.active .portrait-film__thumb { border-color: var(--color-accent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 24%, transparent); }
.portrait-film__actions { display: grid; grid-template-columns: repeat(4, 1fr); gap: 2px; margin-top: 4px; }
.portrait-film__actions button { display: grid; min-height: 22px; place-items: center; padding: 0; border: 0; border-radius: 3px; background: var(--gradient-surface); color: var(--gradient-text-muted); cursor: pointer; }
.portrait-film__actions button:last-child { color: color-mix(in srgb, var(--btn-danger-bg) 45%, var(--gradient-text)); }
.portrait-film__actions button:disabled { cursor: not-allowed; opacity: .3; }
.portrait-film__hint { margin: 5px 2px 0; color: var(--muted); font-size: 9px; line-height: 1.45; }

.portrait-editor__plaque {
  display: grid;
  gap: 3px;
  padding: 13px 12px;
  border-top: 1px solid var(--color-accent);
  border-radius: 0 0 6px 6px;
  background: linear-gradient(90deg, var(--gradient-end), var(--gradient-start) 50%, var(--gradient-end));
  color: var(--gradient-text);
  text-align: center;
}
.portrait-editor__plaque strong { font-family: Georgia, 'Noto Serif SC', serif; font-size: 19px; font-weight: 600; }
.portrait-editor__plaque span { color: var(--gradient-text); font-size: 11px; }
.portrait-editor__plaque small { color: var(--gradient-text-muted); font-size: 9px; }

.source-info,
.local-writeback { display: grid; gap: 6px; margin-top: 14px; padding: 12px; border: 1px solid var(--line); border-radius: 8px; background: var(--color-card-bg); }
.source-info > span,
.local-writeback > span { display: inline-flex; align-items: center; gap: 5px; color: var(--copper); font-size: 10px; font-weight: 800; text-transform: uppercase; letter-spacing: 0.08em; }
.source-info strong { overflow: hidden; color: var(--walnut); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.source-info button,
.local-writeback button { padding: 7px 9px; border: 1px solid var(--btn-outline-border); border-radius: 6px; background: var(--btn-outline-hover); color: var(--btn-outline-text); cursor: pointer; font-size: 10px; font-weight: 700; }
.local-writeback p { margin: 0; color: var(--muted); font-size: 10px; line-height: 1.55; }
.local-writeback.disabled { background: color-mix(in srgb, var(--color-card-bg) 72%, var(--color-main-bg)); }
.local-writeback button:disabled { cursor: not-allowed; opacity: 0.45; }
.local-writeback__actions { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 5px; }

.editor-ledger { min-width: 0; overflow: hidden; }

.editor-tabs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  padding: 0 18px;
  border-bottom: 1px solid var(--line);
  background: var(--color-card-bg);
}
.editor-tabs button { position: relative; display: inline-flex; min-height: 58px; align-items: center; justify-content: center; gap: 7px; border: 0; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 12px; font-weight: 700; }
.editor-tabs button::after { position: absolute; right: 18%; bottom: -1px; left: 18%; height: 3px; border-radius: 3px 3px 0 0; background: var(--copper); content: ''; opacity: 0; transform: scaleX(0.4); transition: opacity 150ms ease, transform 150ms ease; }
.editor-tabs button.active { color: var(--rust); }
.editor-tabs button.active::after { opacity: 1; transform: scaleX(1); }

.editor-panel { min-height: 650px; padding: 26px; }
.editor-panel--rich :deep(.rich-editor) { min-height: 500px; }
.editor-panel--rich :deep(.editor-content) { min-height: 390px; }

.panel-heading { display: grid; grid-template-columns: minmax(0, 1fr) minmax(220px, 0.8fr); align-items: end; gap: 24px; margin-bottom: 26px; padding-bottom: 16px; border-bottom: 1px solid var(--line); }
.panel-heading span { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: 0.16em; text-transform: uppercase; }
.panel-heading h2 { margin: 4px 0 0; font-family: Georgia, 'Noto Serif SC', serif; font-size: 25px; font-weight: 600; }
.panel-heading p { margin: 0; color: var(--muted); font-size: 11px; line-height: 1.65; }

.form-section { margin-bottom: 26px; }
.form-section h3 { margin: 0 0 12px; color: var(--walnut); font-size: 12px; letter-spacing: 0.06em; }
.form-section--rpbox { margin-bottom: 0; padding: 18px; border: 1px solid var(--color-border-hover); border-radius: 10px; background: var(--color-card-bg); }
.form-grid { display: grid; gap: 12px; }
.form-grid--three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.form-span-two { grid-column: span 2; }
.form-grid label,
.summary-field,
.visibility-grid label { display: grid; min-width: 0; gap: 6px; color: var(--muted); font-size: 10px; font-weight: 700; }
.form-grid input,
.summary-field textarea,
.visibility-grid select { width: 100%; box-sizing: border-box; padding: 10px 11px; border: 1px solid var(--input-border); border-radius: 7px; outline: none; background: var(--input-bg); color: var(--ink); font: inherit; font-size: 12px; }
.form-grid input:focus,
.summary-field textarea:focus,
.visibility-grid select:focus { border-color: var(--input-focus); box-shadow: 0 0 0 3px color-mix(in srgb, var(--input-focus) 14%, transparent); }
.summary-field { position: relative; }
.summary-field textarea { resize: vertical; line-height: 1.65; }
.summary-field small { position: absolute; right: 9px; bottom: 8px; color: var(--color-text-muted); font-weight: 400; }
.visibility-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 14px; }
.visibility-grid small { color: var(--muted); font-weight: 400; line-height: 1.45; }
.review-notice { display: flex; gap: 9px; align-items: flex-start; margin-top: 12px; padding: 11px 12px; border: 1px solid var(--color-warning-border); border-radius: 8px; background: var(--color-warning-light); color: var(--color-warning-dark); }
.review-notice > i { margin-top: 1px; font-size: 18px; }
.review-notice > div { display: grid; gap: 3px; }
.review-notice strong { font-size: 11px; }
.review-notice span { font-size: 9px; line-height: 1.5; }
.review-notice--approved { border-color: color-mix(in srgb, var(--color-success) 42%, var(--color-border)); background: var(--color-success-light); color: var(--color-success); }
.review-notice--rejected { border-color: color-mix(in srgb, var(--btn-danger-bg) 42%, var(--color-border)); background: color-mix(in srgb, var(--btn-danger-bg) 10%, var(--color-panel-bg)); color: var(--btn-danger-bg); }

.observation-register {
  position: relative;
  display: grid;
  gap: 14px;
}

.observation-register::before {
  position: absolute;
  z-index: 0;
  top: 28px;
  bottom: 28px;
  left: 22px;
  width: 1px;
  background: linear-gradient(var(--copper), var(--color-border-hover) 50%, var(--copper));
  content: '';
}

.observation-record {
  position: relative;
  z-index: 1;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
  transition: border-color 160ms ease, box-shadow 160ms ease, background-color 160ms ease;
}

.observation-record--active {
  border-color: var(--color-border-hover);
  background: var(--color-panel-bg);
  box-shadow: var(--shadow-md);
}

.observation-record__header {
  display: flex;
  min-height: 47px;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 0 14px 0 12px;
  border-bottom: 1px solid var(--color-border);
  background:
    linear-gradient(90deg, color-mix(in srgb, var(--color-accent) 12%, transparent), transparent 38%),
    var(--color-card-bg);
}

.observation-record__index {
  display: flex;
  align-items: baseline;
  gap: 8px;
  color: var(--color-accent);
  font: 800 8px/1 ui-monospace, Consolas, monospace;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.observation-record__index strong {
  display: grid;
  width: 24px;
  height: 24px;
  place-items: center;
  border: 1px solid var(--gradient-border);
  border-radius: 50%;
  background: var(--gradient-start);
  color: var(--gradient-text);
  font-size: 9px;
  letter-spacing: 0;
  box-shadow: 0 0 0 3px var(--color-card-bg);
}

.observation-toggle {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  cursor: pointer;
  font-size: 10px;
  font-weight: 700;
}

.observation-toggle input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
}

.observation-toggle__track {
  position: relative;
  width: 31px;
  height: 17px;
  border: 1px solid var(--color-border-hover);
  border-radius: 999px;
  background: var(--switch-inactive);
  transition: background-color 150ms ease, border-color 150ms ease;
}

.observation-toggle__track i {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--input-bg);
  box-shadow: var(--shadow-sm);
  transition: transform 150ms ease;
}

.observation-toggle input:checked + .observation-toggle__track {
  border-color: var(--rust);
  background: var(--rust);
}

.observation-toggle input:checked + .observation-toggle__track i { transform: translateX(14px); }
.observation-toggle input:focus-visible + .observation-toggle__track { outline: 3px solid color-mix(in srgb, var(--input-focus) 30%, transparent); outline-offset: 2px; }

.observation-record__body {
  display: grid;
  grid-template-columns: 104px minmax(0, 1fr) minmax(140px, 174px);
  gap: 16px;
  align-items: start;
  padding: 16px;
}

.observation-icon-station,
.observation-image-station {
  display: grid;
  min-width: 0;
  justify-items: center;
  gap: 8px;
}

.observation-fields {
  display: grid;
  min-width: 0;
  gap: 11px;
}

.observation-fields label {
  position: relative;
  display: grid;
  min-width: 0;
  gap: 5px;
  color: var(--muted);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.observation-fields input,
.observation-fields textarea {
  width: 100%;
  box-sizing: border-box;
  padding: 9px 10px;
  border: 1px solid var(--input-border);
  border-radius: 6px;
  outline: none;
  background: var(--input-bg);
  color: var(--ink);
  font: inherit;
  font-size: 11px;
}

.observation-fields textarea {
  min-height: 70px;
  padding-bottom: 20px;
  resize: vertical;
  line-height: 1.55;
}

.observation-fields input:focus,
.observation-fields textarea:focus {
  border-color: var(--input-focus);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--input-focus) 14%, transparent);
}

.observation-fields small {
  position: absolute;
  right: 8px;
  bottom: 6px;
  color: var(--color-text-muted);
  font-size: 8px;
  font-weight: 400;
}

.observation-image-station__frame {
  display: grid;
  width: 100%;
  aspect-ratio: 4 / 3;
  place-items: center;
  overflow: hidden;
  box-sizing: border-box;
  border: 1px solid var(--gradient-border);
  border-radius: 7px;
  background: var(--gradient-end);
  box-shadow: inset 0 0 0 3px color-mix(in srgb, var(--gradient-end) 82%, var(--color-text-main));
}

.observation-image-station__media {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  color: var(--gradient-text-muted);
}

.observation-image-station__media :deep(.authenticated-image__state) { background: var(--gradient-end); }

.observation-image-station__empty {
  display: grid;
  justify-items: center;
  gap: 5px;
  color: var(--gradient-text-muted);
  text-align: center;
}

.observation-image-station__empty i { font-size: 24px; }
.observation-image-station__empty span { color: var(--gradient-text-muted); font-size: 8px; }

.observation-media-actions {
  display: flex;
  width: 100%;
  flex-wrap: wrap;
  justify-content: center;
  gap: 5px;
}

.observation-media-actions button {
  display: inline-flex;
  min-height: 29px;
  flex: 1 1 auto;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 4px 7px;
  border: 1px solid var(--btn-outline-border);
  border-radius: 5px;
  background: var(--btn-outline-hover);
  color: var(--btn-outline-text);
  cursor: pointer;
  font: inherit;
  font-size: 8px;
  font-weight: 700;
}

.observation-media-actions button:disabled { cursor: not-allowed; opacity: 0.55; }
.observation-media-actions .observation-media-actions__remove { flex: 0 0 auto; color: var(--btn-danger-bg); }

.impression-notes {
  margin-top: 30px;
  padding-top: 24px;
  border-top: 1px dashed var(--color-border-hover);
}

.impression-notes > header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(220px, 0.8fr);
  align-items: end;
  gap: 24px;
  margin-bottom: 14px;
}

.impression-notes > header span {
  color: var(--copper);
  font: 800 8px/1.2 ui-monospace, Consolas, monospace;
  letter-spacing: 0.15em;
  text-transform: uppercase;
}

.impression-notes > header h3 {
  margin: 4px 0 0;
  color: var(--walnut);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 18px;
  font-weight: 600;
}

.impression-notes > header p { margin: 0; color: var(--muted); font-size: 10px; line-height: 1.55; }
.impression-notes :deep(.rich-editor) { min-height: 330px; }
.impression-notes :deep(.editor-content) { min-height: 220px; }

.internal-link-button { display: inline-flex; align-items: center; gap: 5px; width: auto !important; padding: 0 8px !important; color: var(--rust) !important; }

.save-dock {
  position: fixed;
  z-index: 40;
  right: 24px;
  bottom: 18px;
  display: flex;
  max-width: min(600px, calc(100vw - 48px));
  align-items: center;
  gap: 22px;
  padding: 11px 12px 11px 16px;
  border: 1px solid var(--color-border-hover);
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-panel-bg) 94%, transparent);
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(14px);
}
.save-dock > div { display: grid; gap: 2px; }
.save-dock strong { color: var(--walnut); font-size: 11px; }
.save-dock span { color: var(--muted); font-size: 9px; }

.portrait-lightbox { display: grid; max-height: 72vh; place-items: center; overflow: hidden; border-radius: 8px; background: var(--gradient-end); }
.portrait-lightbox :deep(img) { max-width: 100%; max-height: 72vh; object-fit: contain; }
.portrait-lightbox :deep(.authenticated-image) { display: grid; max-height: 72vh; place-items: center; }

.writeback-sheet { display: grid; gap: 14px; color: var(--ink); }
.writeback-sheet header { display: grid; gap: 4px; padding-bottom: 12px; border-bottom: 1px solid var(--line); }
.writeback-sheet header span { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .14em; text-transform: uppercase; }
.writeback-sheet header h3 { margin: 0; font-family: Georgia, 'Noto Serif SC', serif; }
.writeback-sheet header p { margin: 0; color: var(--muted); font-size: 10px; line-height: 1.55; }
.writeback-sheet label { display: grid; gap: 6px; color: var(--muted); font-size: 10px; font-weight: 700; }
.writeback-sheet input, .writeback-sheet select { width: 100%; box-sizing: border-box; padding: 10px 11px; border: 1px solid var(--input-border); border-radius: 7px; outline: none; background: var(--input-bg); color: var(--ink); font: inherit; }
.writeback-sheet input:focus, .writeback-sheet select:focus { border-color: var(--input-focus); box-shadow: 0 0 0 3px color-mix(in srgb, var(--input-focus) 14%, transparent); }
.writeback-sheet label small { font-weight: 400; line-height: 1.45; }
.writeback-sheet__account-note { display: flex; gap: 7px; align-items: center; padding: 9px 10px; border-radius: 7px; background: var(--btn-secondary-bg); color: var(--btn-secondary-text); font-size: 10px; }
.writeback-sheet .writeback-sheet__cloud-option { grid-template-columns: auto minmax(0, 1fr); align-items: center; padding: 10px; border: 1px solid var(--color-border-hover); border-radius: 7px; background: var(--color-card-bg); }
.writeback-sheet .writeback-sheet__cloud-option input { width: 16px; height: 16px; padding: 0; accent-color: var(--rust); }
.writeback-sheet__cloud-option small { grid-column: 2; }
.writeback-sheet footer { display: flex; justify-content: flex-end; gap: 8px; }
.lua-preview { display: grid; gap: 10px; }
.lua-preview > p { margin: 0; padding: 8px 10px; border-radius: 6px; background: var(--color-warning-light); color: var(--color-warning-dark); font-size: 10px; }
.lua-preview pre { max-height: 60vh; margin: 0; padding: 14px; overflow: auto; border-radius: 7px; background: var(--gradient-end); color: var(--gradient-text); font: 10px/1.65 ui-monospace, Consolas, monospace; white-space: pre-wrap; word-break: break-word; }
.lua-preview__loading { display: flex; min-height: 180px; align-items: center; justify-content: center; gap: 7px; color: var(--muted); }

.editor-header__back:focus-visible,
.button:focus-visible,
.editor-tabs button:focus-visible,
.portrait-editor button:focus-visible,
.observation-media-actions button:focus-visible { outline: 3px solid color-mix(in srgb, var(--input-focus) 30%, transparent); outline-offset: 2px; }

.spin { animation: editor-spin 900ms linear infinite; }
@keyframes editor-spin { to { transform: rotate(360deg); } }

@container (max-width: 1180px) {
  .editor-layout { grid-template-columns: minmax(240px, 280px) minmax(0, 1fr); }
  .observation-record__body { grid-template-columns: 92px minmax(0, 1fr); }
  .observation-image-station {
    grid-column: 1 / -1;
    grid-template-columns: minmax(180px, 260px) minmax(120px, auto);
    align-items: center;
    justify-content: start;
  }
}

@media (max-width: 980px) {
  .editor-header { grid-template-columns: 1fr auto; }
  .editor-header__identity { grid-row: 2; grid-column: 1 / -1; text-align: left; }
  .editor-layout { grid-template-columns: 240px minmax(0, 1fr); }
  .form-grid--three { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .observation-record__body { grid-template-columns: 92px minmax(0, 1fr); }
  .observation-image-station { grid-column: 1 / -1; grid-template-columns: minmax(180px, 280px) minmax(120px, auto); align-items: center; justify-content: start; }
}

@media (max-width: 760px) {
  .editor-page { width: min(100% - 20px, 1320px); padding-top: 10px; }
  .editor-header { grid-template-columns: 1fr; }
  .editor-header__actions { justify-content: flex-start; flex-wrap: wrap; }
  .editor-header__identity { grid-row: auto; grid-column: auto; }
  .editor-layout { grid-template-columns: 1fr; }
  .portrait-editor { position: relative; top: auto; }
  .portrait-editor__frame { max-width: 360px; margin: 0 auto; }
  .portrait-editor__plaque { max-width: 336px; margin: 0 auto; }
  .editor-tabs { padding: 0 5px; overflow-x: auto; }
  .editor-tabs button { min-width: 105px; }
  .editor-panel { min-height: 540px; padding: 20px 14px; }
  .panel-heading { grid-template-columns: 1fr; gap: 8px; }
  .observation-record__body { grid-template-columns: 1fr; }
  .observation-icon-station { grid-template-columns: auto minmax(130px, 1fr); align-items: center; justify-content: start; }
  .observation-image-station { grid-column: auto; grid-template-columns: 1fr; }
  .observation-image-station__frame { max-height: 220px; }
  .impression-notes > header { grid-template-columns: 1fr; gap: 6px; }
  .form-grid--three,
  .visibility-grid { grid-template-columns: 1fr; }
  .form-span-two { grid-column: auto; }
  .save-dock { right: 10px; bottom: 10px; left: 10px; max-width: none; justify-content: space-between; }
}

@media (max-width: 480px) {
  .observation-record__header { align-items: flex-start; flex-direction: column; gap: 8px; padding-top: 10px; padding-bottom: 10px; }
  .observation-record__body { padding: 12px; }
  .observation-icon-station { grid-template-columns: 1fr; }
  .save-dock > div { display: none; }
  .save-dock .button { width: 100%; }
}

@media (prefers-reduced-motion: reduce) {
  .editor-tabs button::after,
  .observation-record,
  .observation-toggle__track,
  .observation-toggle__track i { transition: none; }
  .spin { animation: none; }
}
</style>
