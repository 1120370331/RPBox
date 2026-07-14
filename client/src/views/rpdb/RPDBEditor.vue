<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  createRPDBWork,
  getRPDBWork,
  resolveRPDBMediaURL,
  updateRPDBWork,
  type RPDBGuideStep,
  type RPDBReference,
  type RPDBWorkPayload,
  type RPDBWorkType,
  type RPDBVisibility,
} from '@/api/rpdb'
import { uploadImage } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import ImageViewer from '@/components/ImageViewer.vue'
import PostQuickJump from '@/components/PostQuickJump.vue'
import TiptapEditor from '@/components/TiptapEditor.vue'
import RPDBMediaGallery from '@/components/rpdb/RPDBMediaGallery.vue'
import RPDBSelect from '@/components/rpdb/RPDBSelect.vue'
import { RPDB_STYLE_PRESETS, isRPDBStyleTag, sortRPDBStyleTags } from '@/constants/rpdbStyles'
import { useToastStore } from '@/stores/toast'
import { formatTomTomCommand, hasTomTomCoordinates, parseTomTomCommands } from '@/utils/tomtom'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const saving = ref(false)
const lastSaved = ref('')
const autoSaveState = ref<'idle' | 'local' | 'saving' | 'saved' | 'error'>('idle')
const autoSaveMessage = computed(() => {
  if (autoSaveState.value === 'saving') return t('rpdb.editor.autosave.saving')
  if (autoSaveState.value === 'saved') return t('rpdb.editor.autosave.saved', { time: lastSaved.value })
  if (autoSaveState.value === 'local') return t('rpdb.editor.autosave.local')
  if (autoSaveState.value === 'error') return t('rpdb.editor.autosave.error')
  return t('rpdb.editor.autosave.editing')
})
const hasLoadedInitialData = ref(false)
const autosavedWorkId = ref<number | null>(null)
let autoSaveTimer: ReturnType<typeof window.setTimeout> | null = null
const rpStyleTags = ref<Tag[]>([])
const guilds = ref<Guild[]>([])
const tomtomDraft = ref('')
const editorRef = ref<InstanceType<typeof TiptapEditor> | null>(null)
const quickJumpOpen = ref(false)
const showImageViewer = ref(false)
const viewerImages = ref<string[]>([])
const viewerStartIndex = ref(0)
const topicDraft = ref('')
const showAllStyleTags = ref(false)
const uploadingFurnitureIcon = ref<RPDBReference | null>(null)
const customStyleTags = ref<string[]>([])
const officialSelectedTags = ref<Tag[]>([])
const titleTouched = ref(false)
const form = reactive<RPDBWorkPayload>({
  type: 'item_showcase',
  title: '',
  summary: '',
  content: '',
  cover_image: '',
  rp_use_cases: '',
  effect_description: '',
  availability_status: 'available',
  bind_type: 'no',
  faction: 'neutral',
  armor_type: '',
  visibility: 'public',
  guild_id: undefined,
  guild_ids: [],
  references: [{
    external_type: 'item',
    external_id: '',
    name: '',
    description: '',
    acquisition_method: '',
    source: '',
    url: '',
    is_primary: true,
  }],
  media: [],
  transmog_slots: [],
  guide_steps: [],
  tag_ids: [],
  is_public: true,
  status: 'draft',
})
const guideCoordinateCount = computed(() => form.guide_steps?.filter(hasTomTomCoordinates).length || 0)
const homeDetails = reactive({
  share_code: '',
  visit_notes: '',
  copy_status: 'copyable',
  visit_status: 'friend_only',
  space_type: 'indoor_outdoor',
})
const typeOptions = computed<Array<{ id: RPDBWorkType; icon: string; title: string; description: string }>>(() => [
  {
    id: 'item_showcase',
    icon: 'ri-magic-line',
    title: t('rpdb.editor.workType.item.title'),
    description: t('rpdb.editor.workType.item.description'),
  },
  {
    id: 'transmog',
    icon: 'ri-shirt-line',
    title: t('rpdb.editor.workType.transmog.title'),
    description: t('rpdb.editor.workType.transmog.description'),
  },
  {
    id: 'home_showcase',
    icon: 'ri-home-heart-line',
    title: t('rpdb.editor.workType.home.title'),
    description: t('rpdb.editor.workType.home.description'),
  },
])
const availabilityOptions = computed(() => ['available', 'limited', 'removed', 'unknown'].map(value => ({
  value,
  label: t(`rpdb.editor.options.availability.${value}.label`),
  hint: t(`rpdb.editor.options.availability.${value}.hint`),
})))
const visibilityOptions = computed(() => ['public', 'guild', 'private'].map(value => ({
  value,
  label: t(`rpdb.editor.options.visibility.${value}.label`),
  hint: t(`rpdb.editor.options.visibility.${value}.hint`),
})))
const itemTypeOptions = computed(() => ['item', 'equipment', 'toy', 'quest_item'].map(value => ({
  value,
  label: t(`rpdb.editor.options.itemType.${value}.label`),
  hint: t(`rpdb.editor.options.itemType.${value}.hint`),
})))
const bindOptions = computed(() => ['no', 'yes'].map(value => ({
  value,
  label: t(`rpdb.editor.options.bind.${value}.label`),
  hint: t(`rpdb.editor.options.bind.${value}.hint`),
})))
const factionOptions = computed(() => ['neutral', 'alliance', 'horde'].map(value => ({
  value,
  label: t(`rpdb.editor.options.faction.${value}.label`),
  hint: t(`rpdb.editor.options.faction.${value}.hint`),
})))
const armorTypeOptions = computed(() => ['', 'cloth', 'leather', 'mail', 'plate', 'cosmetic'].map(value => ({
  value,
  label: t(`rpdb.editor.options.armor.${value || 'all'}.label`),
  hint: t(`rpdb.editor.options.armor.${value || 'all'}.hint`),
})))
const slotValues = ['head', 'shoulder', 'back', 'chest', 'shirt', 'tabard', 'wrist', 'hands', 'waist', 'legs', 'feet', 'main_hand', 'off_hand']
const slotOptions = computed(() => slotValues.map(value => ({ value, label: t(`rpdb.editor.options.slot.${value}`) })))
const slotRoleOptions = computed(() => ['unused', 'required', 'optional', 'variant'].map(value => ({
  value,
  label: t(`rpdb.editor.options.slotRole.${value}.label`),
  hint: t(`rpdb.editor.options.slotRole.${value}.hint`),
})))
const visitStatusOptions = computed(() => ['friend_only', 'closed'].map(value => ({
  value,
  label: t(`rpdb.editor.options.visit.${value}.label`),
  hint: t(`rpdb.editor.options.visit.${value}.hint`),
})))
const copyStatusOptions = computed(() => ['copyable', 'reference_only', 'private'].map(value => ({
  value,
  label: t(`rpdb.editor.options.copy.${value}.label`),
  hint: t(`rpdb.editor.options.copy.${value}.hint`),
})))
const spaceTypeOptions = computed(() => ['indoor', 'outdoor', 'indoor_outdoor'].map(value => ({
  value,
  label: t(`rpdb.editor.options.space.${value}.label`),
  hint: t(`rpdb.editor.options.space.${value}.hint`),
})))

const isEdit = computed(() => Boolean(route.params.id))
const isGuideType = computed(() => form.type !== 'home_showcase')
const localDraftKey = computed(() => `rpdb-editor-draft:${route.params.id || 'new'}`)
const coverPreviewURL = computed(() => resolveRPDBMediaURL(form.cover_image))
const previewMedia = computed(() => form.media?.find(item => item.type === 'image' || item.type === 'gif'))
const previewImageURL = computed(() => resolveRPDBMediaURL(previewMedia.value?.url))
const openTransmogSlots = ref<Set<string>>(new Set())
const openItemReferences = ref<Set<number>>(new Set())
const openFurnitureReferences = ref<Set<number>>(new Set())
const primaryReference = computed(() => form.references?.[0])
const primaryItemReference = computed(() => {
  if (form.type !== 'item_showcase') return null
  return primaryReference.value
})
const primaryTransmogReference = computed(() => {
  if (form.type !== 'transmog') return null
  return primaryReference.value
})
const styleOptions = computed(() => {
  return sortRPDBStyleTags(rpStyleTags.value.length ? rpStyleTags.value : RPDB_STYLE_PRESETS)
})
const selectedStyleTags = computed(() => {
  const selectedIds = new Set(form.tag_ids || [])
  const systemTags = styleOptions.value.filter(tag => tag.id && selectedIds.has(tag.id))
  const selectedSystemIds = new Set(systemTags.map(tag => tag.id))
  const officialTags = officialSelectedTags.value.filter(tag => tag.id && selectedIds.has(tag.id) && !selectedSystemIds.has(tag.id))
  const customTags = customStyleTags.value.map(name => ({
    id: 0,
    name,
    color: 'B87333',
    custom: true,
  }))
  return [...systemTags, ...officialTags, ...customTags]
})
const candidateStyleTags = computed(() => {
  const selectedIds = new Set(form.tag_ids || [])
  return styleOptions.value.filter(tag => !tag.id || !selectedIds.has(tag.id))
})
const visibleStyleTags = computed(() => {
  return showAllStyleTags.value ? candidateStyleTags.value : candidateStyleTags.value.slice(0, 8)
})
const typeFormTitle = computed(() => {
  if (form.type === 'home_showcase') return t('rpdb.editor.typeForm.home.title')
  if (form.type === 'transmog') return t('rpdb.editor.typeForm.transmog.title')
  return t('rpdb.editor.typeForm.item.title')
})
const typeFormDescription = computed(() => {
  if (form.type === 'home_showcase') return t('rpdb.editor.typeForm.home.description')
  if (form.type === 'transmog') return t('rpdb.editor.typeForm.transmog.description')
  return t('rpdb.editor.typeForm.item.description')
})
const titleHasError = computed(() => titleTouched.value && !form.title.trim())
function scrollToSection(id: string) {
  const target = document.getElementById(id)
  if (!target || typeof target.scrollIntoView !== 'function') return
  const reduceMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  target.scrollIntoView({ behavior: reduceMotion ? 'auto' : 'smooth', block: 'start' })
}

function selectWorkType(type: RPDBWorkType) {
  form.type = type
  if (type === 'home_showcase') {
    form.references = []
    form.transmog_slots = []
    form.guide_steps = []
    form.armor_type = ''
    ensureHomeFurniture()
    return
  }
  if (type === 'item_showcase') {
    form.armor_type = ''
  }
  ensurePrimaryReference(type)
  if (type === 'transmog') ensureTransmogSlots()
}

function ensurePrimaryReference(type: RPDBWorkType = form.type) {
  if (type === 'home_showcase') return
  if (!form.references?.length) {
    addReference(type === 'transmog' ? 'transmog' : 'item')
    return
  }
  if (type === 'transmog') {
    form.references[0].external_type = 'transmog'
  } else if (!itemTypeOptions.value.some(option => option.value === form.references![0].external_type)) {
    form.references[0].external_type = 'item'
  }
  form.references[0].is_primary = true
  if (type === 'item_showcase' && !form.bind_type) form.bind_type = 'no'
}

function addReference(externalType: 'item' | 'transmog' = form.type === 'transmog' ? 'transmog' : 'item') {
  form.references!.push({
    external_type: externalType,
    external_id: '',
    name: '',
    description: '',
    acquisition_method: '',
    source: externalType === 'transmog' ? 'collection' : '',
    url: '',
    is_primary: form.references!.length === 0,
  })
}

function addItemReference() {
  addReference('item')
}

function addFurnitureReference() {
  const references = form.references || (form.references = [])
  references.push({
    external_type: 'furniture',
    external_id: '',
    name: '',
    icon: '',
    source: 'wowhead',
    url: '',
    description: '',
    acquisition_method: '',
    is_primary: references.length === 0,
  })
}

function ensureHomeFurniture() {
  if (!form.references?.length) addFurnitureReference()
}

function addMedia() {
  form.media!.push({ type: 'image', url: '', caption: '' })
}

function addSlot() {
  const usedSlots = new Set((form.transmog_slots || []).map(slot => slot.slot))
  const nextSlot = slotOptions.value.find(option => !usedSlots.has(option.value)) || slotOptions.value[0]
  form.transmog_slots!.push({ slot: nextSlot.value, role: 'required', name: '', description: '', source: '', wowhead_url: '', variant: '', note: '', sort_order: form.transmog_slots!.length + 1 })
}

function ensureTransmogSlots() {
  const currentSlots = new Map((form.transmog_slots || []).map(slot => [slot.slot, slot]))
  form.transmog_slots = slotOptions.value.map((option, index) => ({
    slot: option.value,
    role: currentSlots.get(option.value)?.role || 'unused',
    name: currentSlots.get(option.value)?.name || '',
    description: currentSlots.get(option.value)?.description || '',
    source: currentSlots.get(option.value)?.source || '',
    wowhead_url: currentSlots.get(option.value)?.wowhead_url || '',
    variant: currentSlots.get(option.value)?.variant || '',
    note: currentSlots.get(option.value)?.note || '',
    sort_order: currentSlots.get(option.value)?.sort_order || index + 1,
  }))
}

function addStep() {
  form.guide_steps!.push({
    sort_order: form.guide_steps!.length + 1,
    title: '',
    body: '',
    zone: '',
    map_id: '',
    x: 0,
    y: 0,
    prerequisite: '',
  })
}

function removeStep(index: number) {
  form.guide_steps!.splice(index, 1)
  form.guide_steps!.forEach((step, stepIndex) => {
    step.sort_order = stepIndex + 1
  })
}

function importTomTomSteps() {
  const { waypoints, rejected } = parseTomTomCommands(tomtomDraft.value)
  for (const waypoint of waypoints) {
    form.guide_steps!.push({
      sort_order: form.guide_steps!.length + 1,
      title: waypoint.label || t('rpdb.editor.guide.routePoint', { number: form.guide_steps!.length + 1 }),
      body: '',
      zone: waypoint.zone,
      map_id: waypoint.map_id,
      x: waypoint.x,
      y: waypoint.y,
      prerequisite: '',
    })
  }
  if (!waypoints.length) {
    toast.error(t('rpdb.editor.toast.noWaypoints'))
    return
  }
  tomtomDraft.value = ''
  if (rejected.length) {
    toast.warning(t('rpdb.editor.toast.waypointsPartial', { count: waypoints.length, rejected: rejected.length }))
  } else {
    toast.success(t('rpdb.editor.toast.waypointsImported', { count: waypoints.length }))
  }
}

function editorTomTomCommand(step: RPDBGuideStep, index: number) {
  if (!hasTomTomCoordinates(step)) return ''
  const sequence = (form.guide_steps || [])
    .slice(0, index + 1)
    .filter(hasTomTomCoordinates)
    .length
  return formatTomTomCommand(step, { sequence, total: guideCoordinateCount.value })
}

function handleQuickInsert(html: string) {
  editorRef.value?.insertContent(html)
  quickJumpOpen.value = false
}

function toggleQuickJump() {
  quickJumpOpen.value = !quickJumpOpen.value
}

function openImageViewer(images: string[], index: number) {
  viewerImages.value = images
  viewerStartIndex.value = index
  showImageViewer.value = images.length > 0
}

async function uploadCover(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  try {
    const result = await uploadImage(file) as { url?: string; data?: { url?: string } }
    form.cover_image = result.url || result.data?.url || ''
    toast.success(t('rpdb.editor.toast.coverUploaded'))
  } catch (error) {
    toast.error((error as Error).message)
  }
}

function removeCover() {
  form.cover_image = ''
}

function toggleTransmogSlot(slot: string) {
  const next = new Set(openTransmogSlots.value)
  if (next.has(slot)) {
    next.delete(slot)
  } else {
    next.add(slot)
  }
  openTransmogSlots.value = next
}

function toggleItemReference(index: number) {
  const next = new Set(openItemReferences.value)
  if (next.has(index)) {
    next.delete(index)
  } else {
    next.add(index)
  }
  openItemReferences.value = next
}

function toggleFurnitureReference(index: number) {
  const next = new Set(openFurnitureReferences.value)
  if (next.has(index)) {
    next.delete(index)
  } else {
    next.add(index)
  }
  openFurnitureReferences.value = next
}

function itemTypeLabel(value?: string) {
  return itemTypeOptions.value.find(option => option.value === value)?.label || t('rpdb.editor.options.itemType.item.label')
}

function appendPreviewImage(url: string) {
  const media = form.media || (form.media = [])
  if (media.some(item => item.url === url)) return
  const previewCount = media.filter(item => item.type === 'image' || item.type === 'gif').length
  media.push({
    type: 'image',
    url,
    thumbnail_url: url,
    caption: t('rpdb.editor.media.previewCaption', { number: previewCount + 1 }),
    sort_order: media.length + 1,
  })
}

async function uploadPreview(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return
  let uploaded = 0
  for (const file of files) {
    try {
      const result = await uploadImage(file) as { url?: string; data?: { url?: string } }
      const url = result.url || result.data?.url || ''
      if (!url) throw new Error(t('rpdb.editor.toast.uploadMissingUrl'))
      appendPreviewImage(url)
      uploaded++
    } catch (error) {
      toast.error(`${file.name}: ${(error as Error).message}`)
    }
  }
  input.value = ''
  if (uploaded) toast.success(t('rpdb.editor.toast.previewsAdded', { count: uploaded }))
}

async function uploadFurnitureIcon(event: Event, reference: RPDBReference) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploadingFurnitureIcon.value = reference
  try {
    const result = await uploadImage(file) as { url?: string; data?: { url?: string } }
    const url = result.url || result.data?.url || ''
    if (!url) throw new Error(t('rpdb.editor.toast.uploadMissingUrl'))
    reference.icon = url
    toast.success(t('rpdb.editor.toast.furnitureIconUploaded'))
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    input.value = ''
    if (uploadingFurnitureIcon.value === reference) uploadingFurnitureIcon.value = null
  }
}

function toggleStyleTag(tag: Pick<Tag, 'id' | 'name'>) {
  if (!tag.id) return
  const selected = new Set(form.tag_ids || [])
  if (selected.has(tag.id)) {
    selected.delete(tag.id)
  } else {
    selected.add(tag.id)
  }
  form.tag_ids = Array.from(selected)
}

function removeCustomStyleTag(name: string) {
  customStyleTags.value = customStyleTags.value.filter(tagName => tagName !== name)
}

function removeSelectedTopic(tag: Pick<Tag, 'id' | 'name'> & { custom?: boolean }) {
  if (tag.custom) {
    removeCustomStyleTag(tag.name)
    return
  }
  toggleStyleTag(tag)
}

function normalizeTopicName(value: string) {
  return value.trim().replace(/^#+/, '').trim()
}

function addTopicFromInput() {
  const name = normalizeTopicName(topicDraft.value)
  if (!name) return
  const existing = styleOptions.value.find(tag => tag.name === name)
  if (existing?.id) {
    toggleStyleTag(existing)
    topicDraft.value = ''
    return
  }
  const lowerName = name.toLowerCase()
  if (!customStyleTags.value.some(tagName => tagName.toLowerCase() === lowerName)) {
    customStyleTags.value = [...customStyleTags.value, name]
  }
  topicDraft.value = ''
}

function referenceHasContent(reference: NonNullable<RPDBWorkPayload['references']>[number]) {
  return [
    reference.external_id,
    reference.name,
    reference.icon,
    reference.url,
    reference.description,
    reference.acquisition_method,
  ].some(value => String(value || '').trim())
}

function buildReferencePayload() {
  return (form.references || [])
    .filter(referenceHasContent)
    .map((reference, index) => ({
      ...reference,
      external_type: reference.external_type || (form.type === 'home_showcase' ? 'furniture' : form.type === 'transmog' ? 'transmog' : 'item'),
      external_id: reference.external_id || `rpbox-${index + 1}`,
      sort_order: index + 1,
      is_primary: index === 0,
    }))
}

function buildTransmogSlotPayload() {
  if (form.type !== 'transmog') return []
  return (form.transmog_slots || [])
    .filter(slot => slot.role && slot.role !== 'unused')
    .map((slot, index) => ({ ...slot, sort_order: index + 1 }))
}

function buildDraftPayload(status: 'draft' | 'published' = 'draft') {
  syncHomeDetails()
  const { game_version: _gameVersion, expansion: _expansion, ...formPayload } = form as RPDBWorkPayload & { game_version?: string; expansion?: string }
  return {
    ...formPayload,
    references: buildReferencePayload(),
    transmog_slots: buildTransmogSlotPayload(),
    tag_names: [...customStyleTags.value],
    status,
    visibility: form.visibility || 'public',
    guild_ids: form.visibility === 'guild' ? form.guild_ids : [],
    guild_id: form.visibility === 'guild' ? form.guild_ids?.[0] : undefined,
    is_public: form.visibility === 'public',
  }
}

function saveLocalDraft() {
  window.localStorage.setItem(localDraftKey.value, JSON.stringify({
    form,
    homeDetails,
    customStyleTags: customStyleTags.value,
    autosavedWorkId: autosavedWorkId.value,
    savedAt: new Date().toISOString(),
  }))
  lastSaved.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  autoSaveState.value = 'local'
}

function loadLocalDraft() {
  if (isEdit.value) return
  const raw = window.localStorage.getItem(localDraftKey.value)
  if (!raw) return
  try {
    const draft = JSON.parse(raw) as { form?: Partial<RPDBWorkPayload>; homeDetails?: Partial<typeof homeDetails>; customStyleTags?: string[]; autosavedWorkId?: number }
    if (draft.form) Object.assign(form, draft.form)
    if (draft.homeDetails) Object.assign(homeDetails, draft.homeDetails)
    customStyleTags.value = draft.customStyleTags || draft.form?.tag_names || []
    autosavedWorkId.value = draft.autosavedWorkId || null
    ensureEditorDefaults()
  } catch {
    window.localStorage.removeItem(localDraftKey.value)
  }
}

async function autoSaveDraft() {
  saveLocalDraft()
  if (!form.title.trim() || !window.localStorage.getItem('token')) return
  autoSaveState.value = 'saving'
  try {
    const payload = buildDraftPayload('draft')
    const targetId = Number(route.params.id) || autosavedWorkId.value
    const result = targetId
      ? await updateRPDBWork(targetId, payload)
      : await createRPDBWork(payload)
    autosavedWorkId.value = result.work?.id || targetId || null
    saveLocalDraft()
    autoSaveState.value = 'saved'
  } catch {
    autoSaveState.value = 'error'
  }
}

function scheduleAutoSave() {
  if (!hasLoadedInitialData.value) return
  if (autoSaveTimer) window.clearTimeout(autoSaveTimer)
  autoSaveTimer = window.setTimeout(() => {
    void autoSaveDraft()
  }, 900)
}

function importHomeShareCode(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    homeDetails.share_code = String(reader.result || '').trim()
    toast.success(t('rpdb.editor.toast.homeCodeImported'))
  }
  reader.onerror = () => toast.error(t('rpdb.editor.toast.homeCodeReadFailed'))
  reader.readAsText(file)
}

async function loadStyleTags() {
  try {
    const result = await getPresetTags('rpdb')
    rpStyleTags.value = sortRPDBStyleTags((result.tags || []).filter(isRPDBStyleTag))
  } catch {
    rpStyleTags.value = []
  }
}

async function loadGuildOptions() {
  if (!window.localStorage.getItem('token')) return
  try {
    const result = await listGuilds()
    guilds.value = result.guilds || []
  } catch {
    guilds.value = []
  }
}

function updateVisibility(value: string) {
  form.visibility = value as RPDBVisibility
  form.is_public = value === 'public'
  if (value !== 'guild') {
    form.guild_id = undefined
    form.guild_ids = []
  }
}

function toggleVisibilityGuild(guildID: number) {
  const selected = new Set(form.guild_ids || [])
  if (selected.has(guildID)) {
    selected.delete(guildID)
  } else {
    selected.add(guildID)
  }
  form.guild_ids = Array.from(selected)
  form.guild_id = form.guild_ids[0]
}

function syncHomeDetails() {
  form.extra = form.type === 'home_showcase' ? {
    share_code: homeDetails.share_code,
    visit_notes: homeDetails.visit_notes,
    copy_status: homeDetails.copy_status,
    visit_status: homeDetails.visit_status,
    space_type: homeDetails.space_type,
  } : {}
}

function ensureEditorDefaults() {
  if (form.visibility !== 'public' && form.visibility !== 'guild' && form.visibility !== 'private') {
    form.visibility = 'public'
  }
  form.is_public = form.visibility === 'public'
  if (form.visibility === 'guild') {
    form.guild_ids = Array.from(new Set((form.guild_ids?.length ? form.guild_ids : form.guild_id ? [form.guild_id] : []).filter(Boolean)))
    form.guild_id = form.guild_ids[0]
  } else {
    form.guild_id = undefined
    form.guild_ids = []
  }
  if (form.type === 'home_showcase') {
    ensureHomeFurniture()
    return
  }
  ensurePrimaryReference()
  if (form.type === 'transmog') ensureTransmogSlots()
}

async function save(status: 'draft' | 'published') {
  if (!form.visibility) {
    toast.error(t('rpdb.editor.validation.visibilityRequired'))
    scrollToSection('section-basics')
    return
  }
  if (form.visibility === 'guild' && !form.guild_ids?.length) {
    toast.error(t('rpdb.editor.validation.guildRequired'))
    scrollToSection('section-basics')
    return
  }
  if (!form.title.trim()) {
    titleTouched.value = true
    toast.error(t('rpdb.editor.validation.titleRequired'))
    scrollToSection('section-basics')
    window.setTimeout(() => document.getElementById('rpdb-title')?.focus(), 250)
    return
  }
  saving.value = true
  try {
    const payload = buildDraftPayload(status)
    const targetId = Number(route.params.id) || autosavedWorkId.value
    const result = targetId
      ? await updateRPDBWork(targetId, payload)
      : await createRPDBWork(payload)
    lastSaved.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    autosavedWorkId.value = result.work?.id || targetId || null
    saveLocalDraft()
    toast.success(status === 'draft' ? t('rpdb.editor.toast.draftSaved') : t('rpdb.editor.toast.published'))
    if (status === 'published') {
      const id = result.work?.id || Number(route.params.id)
      await router.push(id ? `/rpdb/${id}` : '/rpdb')
    }
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    saving.value = false
  }
}

function preview() {
  syncHomeDetails()
  sessionStorage.setItem('rpdb-preview', JSON.stringify(form))
  router.push('/rpdb/preview')
}

onMounted(async () => {
  loadLocalDraft()
  await Promise.all([loadStyleTags(), loadGuildOptions()])
  if (isEdit.value) {
    try {
      const { work } = await getRPDBWork(Number(route.params.id))
      Object.assign(form, {
        type: work.type,
        title: work.title,
        summary: work.summary,
        content: work.content,
        content_type: work.content_type,
        cover_image: work.cover_image,
        rp_use_cases: work.rp_use_cases,
        effect_description: work.effect_description,
        availability_status: work.availability_status,
        bind_type: work.bind_type,
        faction: work.faction,
        armor_type: work.armor_type,
        visibility: work.visibility || (work.is_public ? 'public' : 'private'),
        guild_id: work.guild_id,
        guild_ids: work.guild_ids?.length ? work.guild_ids : work.guild_id ? [work.guild_id] : [],
        references: work.references || [],
        media: work.media || [],
        transmog_slots: work.transmog_slots || [],
        guide_steps: work.guide_steps || [],
        tag_ids: work.tags?.map(tag => tag.id) || [],
      })
      officialSelectedTags.value = work.tags || []
      if (work.type === 'home_showcase') {
        try {
          Object.assign(homeDetails, JSON.parse(work.extra || '{}'))
        } catch {
          Object.assign(homeDetails, {
            share_code: '',
            visit_notes: '',
            copy_status: 'copyable',
            visit_status: 'open',
            space_type: 'indoor',
          })
        }
      }
      ensureEditorDefaults()
    } catch (error) {
      toast.error((error as Error).message)
    }
  }
  ensureEditorDefaults()
  hasLoadedInitialData.value = true
})

onBeforeUnmount(() => {
  if (autoSaveTimer) window.clearTimeout(autoSaveTimer)
  autoSaveTimer = null
})

watch([form, homeDetails, customStyleTags], scheduleAutoSave, { deep: true })
</script>

<template>
  <div class="editor-page minimal-editor-shell">
    <header class="editor-heading">
      <div>
        <span>{{ t('rpdb.editor.header.eyebrow') }}</span>
        <h1>{{ isEdit ? t('rpdb.editor.header.editTitle') : t('rpdb.editor.header.createTitle') }}</h1>
        <p>{{ t('rpdb.editor.header.description') }}</p>
      </div>
      <div class="heading-actions">
        <span v-if="lastSaved" class="saved-status"><i class="ri-checkbox-circle-fill"></i>{{ t('rpdb.editor.header.saved', { time: lastSaved }) }}</span>
        <button type="button" :aria-label="t('rpdb.editor.action.close')" :title="t('rpdb.editor.action.close')" @click="router.push('/rpdb')"><i class="ri-close-line"></i></button>
      </div>
    </header>

    <section id="section-basics" class="editor-upper section-anchor" data-testid="editor-upper">
      <div class="metadata-panel">
        <div class="panel-heading">
          <div><span>01</span><h2>{{ t('rpdb.editor.basics.title') }}</h2></div>
          <small><b class="required-mark" aria-hidden="true">*</b> {{ t('rpdb.editor.basics.required') }} · {{ form.status === 'published' ? t('rpdb.editor.status.pending') : t('rpdb.editor.status.draft') }}</small>
        </div>

        <div class="visibility-setup" data-testid="visibility-field">
          <div class="visibility-setup__intro">
            <i class="ri-eye-line"></i>
            <span>
              <b>{{ t('rpdb.editor.help.visibilityTitle') }}</b>
              <small>{{ t('rpdb.editor.help.visibility') }}</small>
            </span>
          </div>
          <label data-testid="visibility-select">
            <span>{{ t('rpdb.editor.field.visibility') }} <b class="required-mark" aria-hidden="true">*</b></span>
            <RPDBSelect
              :model-value="form.visibility"
              :options="visibilityOptions"
              @update:model-value="updateVisibility"
            />
          </label>
          <label v-if="form.visibility === 'guild'" data-testid="visibility-guild-select">
            <span>{{ t('rpdb.editor.field.visibilityGuild') }} <b class="required-mark" aria-hidden="true">*</b> <small>{{ t('rpdb.editor.help.visibilityGuildMultiple') }}</small></span>
            <div v-if="guilds.length" class="visibility-guild-options">
              <label
                v-for="guild in guilds"
                :key="guild.id"
                class="visibility-guild-option"
                :class="{ selected: form.guild_ids?.includes(guild.id) }"
              >
                <input
                  type="checkbox"
                  :checked="form.guild_ids?.includes(guild.id)"
                  :value="guild.id"
                  @change="toggleVisibilityGuild(guild.id)"
                >
                <span><b>{{ guild.name }}</b><small>{{ t(`rpdb.editor.options.guildRole.${guild.my_role || 'member'}`) }}</small></span>
                <i v-if="form.guild_ids?.includes(guild.id)" class="ri-check-line"></i>
              </label>
            </div>
            <small v-else class="visibility-empty">{{ t('rpdb.editor.help.noGuilds') }}</small>
          </label>
        </div>

        <div class="type-cards">
          <button
            v-for="option in typeOptions"
            :key="option.id"
            type="button"
            :class="{ active: form.type === option.id }"
            @click="selectWorkType(option.id)"
          >
            <i :class="option.icon"></i>
            <span><b>{{ option.title }}</b><small>{{ option.description }}</small></span>
          </button>
        </div>

        <div class="metadata-grid">
          <label class="span-2 field-control" :class="{ invalid: titleHasError }">
            <span>{{ t('rpdb.editor.field.title') }} <b class="required-mark" aria-hidden="true">*</b></span>
            <input
              id="rpdb-title"
              v-model="form.title"
              maxlength="256"
              required
              :aria-invalid="titleHasError"
              aria-describedby="rpdb-title-help"
              :placeholder="t('rpdb.editor.placeholder.title')"
              @blur="titleTouched = true"
            >
            <small id="rpdb-title-help" :class="{ 'field-error': titleHasError }">
              {{ titleHasError ? t('rpdb.editor.validation.titleRequired') : t('rpdb.editor.help.title') }}
            </small>
          </label>
          <label class="span-2">
            <span>{{ t('rpdb.editor.field.summary') }} <em class="optional-label">{{ t('rpdb.editor.common.optional') }}</em></span>
            <textarea v-model="form.summary" maxlength="512" :placeholder="t('rpdb.editor.placeholder.summary')"></textarea>
          </label>
          <label v-if="form.type !== 'home_showcase'">
            <span>{{ t('rpdb.editor.field.availability') }}</span>
            <RPDBSelect v-model="form.availability_status" :options="availabilityOptions" />
          </label>
          <div class="style-picker span-2">
            <span>{{ t('rpdb.editor.field.styleTags') }} <em class="optional-label">{{ t('rpdb.editor.common.optional') }}</em></span>
            <div class="selected-topics" data-testid="rpdb-selected-topics">
              <button
                v-for="tag in selectedStyleTags"
                :key="tag.custom ? `custom-${tag.name}` : `selected-${tag.id}`"
                type="button"
                data-testid="rpdb-selected-topic"
                :style="{ '--tag-color': `#${tag.color || 'B87333'}` }"
                @click="removeSelectedTopic(tag)"
              >
                <span>#{{ tag.name }}</span>
                <i class="ri-close-line" data-testid="remove-rpdb-style-tag"></i>
              </button>
              <span v-if="!selectedStyleTags.length" class="topic-empty">{{ t('rpdb.editor.style.empty') }}</span>
            </div>
            <div class="style-options" data-testid="rpdb-style-candidates">
              <button
                v-for="tag in visibleStyleTags"
                :key="`${tag.name}-${tag.id}`"
                type="button"
                data-testid="rpdb-style-option"
                :class="{ disabled: !tag.id }"
                :style="{ '--tag-color': `#${tag.color || 'B87333'}` }"
                @click="toggleStyleTag(tag)"
              >
                #{{ tag.name }}
              </button>
              <button
                v-if="candidateStyleTags.length > 8"
                type="button"
                class="style-options-toggle"
                :aria-expanded="showAllStyleTags"
                @click="showAllStyleTags = !showAllStyleTags"
              >
                <i :class="showAllStyleTags ? 'ri-arrow-up-s-line' : 'ri-arrow-down-s-line'"></i>
                {{ showAllStyleTags ? t('rpdb.editor.style.collapse') : t('rpdb.editor.style.expand', { count: candidateStyleTags.length - 8 }) }}
              </button>
            </div>
            <label class="topic-input" data-testid="rpdb-topic-custom">
              <input
                v-model="topicDraft"
                data-testid="rpdb-topic-input"
                :placeholder="t('rpdb.editor.placeholder.customStyle')"
                @keydown.enter.prevent="addTopicFromInput"
              >
            </label>
            <small v-if="!rpStyleTags.length">{{ t('rpdb.editor.style.loading') }}</small>
          </div>
        </div>

        <div id="section-details" class="panel-heading section-anchor">
          <div><span>02</span><h2>{{ typeFormTitle }}</h2></div>
          <small>{{ typeFormDescription }}</small>
        </div>

        <section v-if="form.type === 'item_showcase' && primaryItemReference" class="type-form-panel item-type-form" data-testid="item-editor-fields">
          <div class="metadata-grid">
            <label><span>{{ t('rpdb.editor.field.itemName') }}</span><input v-model="primaryItemReference.name" :placeholder="t('rpdb.editor.placeholder.itemName')"></label>
            <label><span>{{ t('rpdb.editor.field.itemType') }}</span><RPDBSelect v-model="primaryItemReference.external_type" :options="itemTypeOptions" /></label>
            <label class="span-2"><span>{{ t('rpdb.editor.field.itemDescription') }}</span><textarea v-model="primaryItemReference.description" :placeholder="t('rpdb.editor.placeholder.itemDescription')"></textarea></label>
            <label class="span-2"><span>{{ t('rpdb.editor.field.itemSource') }}</span><input v-model="primaryItemReference.acquisition_method" :placeholder="t('rpdb.editor.placeholder.itemSource')"></label>
            <label><span>{{ t('rpdb.editor.field.faction') }}</span><RPDBSelect v-model="form.faction" :options="factionOptions" /></label>
            <label><span>{{ t('rpdb.editor.field.bound') }}</span><RPDBSelect v-model="form.bind_type" :options="bindOptions" /></label>
          </div>
        </section>

        <section v-else-if="form.type === 'transmog' && primaryTransmogReference" class="type-form-panel transmog-type-form" data-testid="transmog-editor-fields">
          <div class="metadata-grid">
            <label><span>{{ t('rpdb.editor.field.armorType') }}</span><RPDBSelect v-model="form.armor_type" :options="armorTypeOptions" /></label>
            <label><span>{{ t('rpdb.editor.field.faction') }}</span><RPDBSelect v-model="form.faction" :options="factionOptions" /></label>
            <label><span>{{ t('rpdb.editor.field.availability') }}</span><RPDBSelect v-model="form.availability_status" :options="availabilityOptions" /></label>
          </div>
          <div class="slot-helper-panel">
            <i class="ri-shirt-line"></i>
            <span><b>{{ t('rpdb.editor.transmog.slotsTitle') }}</b><small>{{ t('rpdb.editor.transmog.slotsHelp') }}</small></span>
          </div>
        </section>

        <section v-else class="type-form-panel home-type-form" data-testid="home-editor-fields">
          <div class="home-grid">
            <label><span>{{ t('rpdb.editor.field.visitStatus') }}</span><RPDBSelect v-model="homeDetails.visit_status" :options="visitStatusOptions" /></label>
            <label><span>{{ t('rpdb.editor.field.codeStatus') }}</span><RPDBSelect v-model="homeDetails.copy_status" :options="copyStatusOptions" /></label>
            <label><span>{{ t('rpdb.editor.field.spaceType') }}</span><RPDBSelect v-model="homeDetails.space_type" :options="spaceTypeOptions" /></label>
            <div class="home-code-row span-2" data-testid="home-code-upload-row">
              <label>
                <span>{{ t('rpdb.editor.field.homeCode') }}</span>
                <textarea v-model="homeDetails.share_code" data-testid="home-share-code-input" :placeholder="t('rpdb.editor.placeholder.homeCode')"></textarea>
              </label>
              <label class="home-code-upload">
                <input type="file" accept=".txt,.json,.lua" data-testid="home-code-file-input" @change="importHomeShareCode">
                <span><i class="ri-file-upload-line"></i><b>{{ t('rpdb.editor.action.uploadHomeCode') }}</b><small>.txt / .json / .lua</small></span>
              </label>
            </div>
            <label class="span-2"><span>{{ t('rpdb.editor.field.visitNotes') }}</span><textarea v-model="homeDetails.visit_notes" :placeholder="t('rpdb.editor.placeholder.visitNotes')"></textarea></label>
          </div>
        </section>

        <label v-if="isEdit" class="change-summary">
          <span>{{ t('rpdb.editor.field.changeSummary') }}</span>
          <textarea v-model="form.change_summary" :placeholder="t('rpdb.editor.placeholder.changeSummary')"></textarea>
        </label>
      </div>
    </section>

    <section id="section-media" class="media-strip section-anchor" data-testid="rpdb-media-strip">
      <div class="media-strip__heading">
        <span>03 · {{ t('rpdb.editor.common.optional') }}</span>
        <h2>{{ t('rpdb.editor.media.title') }}</h2>
        <p>{{ t('rpdb.editor.media.description') }}</p>
      </div>

      <div class="media-upload-card">
        <div class="panel-heading">
          <div><span>{{ t('rpdb.editor.media.cover') }}</span><h2>{{ t('rpdb.editor.media.coverTitle') }}</h2></div>
          <small>{{ t('rpdb.editor.media.coverHint') }}</small>
        </div>
        <div class="media-upload media-upload--cover" data-testid="cover-upload">
          <input id="rpdb-cover-upload" type="file" accept="image/*" @change="uploadCover">
          <img v-if="coverPreviewURL" :src="coverPreviewURL" :alt="t('rpdb.editor.media.coverAlt')">
          <span v-else>
            <i class="ri-upload-cloud-2-line"></i>
            <b>{{ t('rpdb.editor.action.uploadCover') }}</b>
            <small>{{ t('rpdb.editor.media.coverOptional') }}</small>
          </span>
          <label class="media-upload__action" for="rpdb-cover-upload">
            <i class="ri-upload-cloud-2-line"></i>
            {{ coverPreviewURL ? t('rpdb.editor.action.replaceCover') : t('rpdb.editor.action.customCover') }}
          </label>
          <button
            v-if="coverPreviewURL"
            type="button"
            class="media-remove"
            data-testid="cover-remove"
            :aria-label="t('rpdb.editor.action.removeCover')"
            @click="removeCover"
          >
            <i class="ri-close-line"></i>
          </button>
        </div>
      </div>

      <div class="media-upload-card">
        <div class="panel-heading">
          <div><span>{{ t('rpdb.editor.media.preview') }}</span><h2>{{ t('rpdb.editor.media.previewTitle') }}</h2></div>
          <small>{{ t('rpdb.editor.media.previewHint') }}</small>
        </div>
        <label class="media-upload" data-testid="preview-upload">
          <input type="file" accept="image/*" multiple @change="uploadPreview">
          <img v-if="previewImageURL" :src="previewImageURL" :alt="t('rpdb.editor.media.previewAlt')">
          <span v-else>
            <i class="ri-image-add-line"></i>
            <b>{{ t('rpdb.editor.action.uploadPreview') }}</b>
            <small>{{ t('rpdb.editor.media.previewOptional') }}</small>
          </span>
          <em v-if="previewImageURL">{{ t('rpdb.editor.action.addMorePreviews') }}</em>
        </label>
      </div>

      <div v-if="previewImageURL" class="preview-gallery-panel" data-testid="preview-gallery">
        <div class="panel-heading">
          <div><span>{{ t('rpdb.editor.media.gallery') }}</span><h2>{{ t('rpdb.editor.media.galleryTitle') }}</h2></div>
          <small>{{ t('rpdb.editor.media.galleryHint') }}</small>
        </div>
        <RPDBMediaGallery
          :cover="form.cover_image"
          :media="form.media"
          :title="form.title || t('rpdb.editor.media.workPreview')"
          @open-image="openImageViewer"
        />
      </div>
    </section>

    <section id="section-content" class="editor-lower section-anchor" data-testid="editor-lower">
      <main class="writing-workspace">
        <div class="writing-heading">
          <div>
            <span>04 · {{ t('rpdb.editor.writing.eyebrow') }}</span>
            <h2>{{ t('rpdb.editor.writing.title') }}</h2>
            <p>{{ t('rpdb.editor.writing.description') }}</p>
          </div>
          <div class="outline-chips">
            <span>{{ t('rpdb.editor.writing.body') }}</span>
            <span v-if="form.type === 'item_showcase'">{{ t('rpdb.editor.writing.itemGuide') }}</span>
            <span v-else-if="form.type === 'transmog'">{{ t('rpdb.editor.writing.transmogGuide') }}</span>
            <span v-else>{{ t('rpdb.editor.writing.homeNotes') }}</span>
            <span>{{ form.type === 'home_showcase' ? t('rpdb.editor.writing.shareCode') : t('rpdb.editor.writing.versionNotes') }}</span>
          </div>
        </div>

        <div class="rich-editor-shell">
          <TiptapEditor
            ref="editorRef"
            :model-value="form.content || ''"
            data-testid="rpdb-rich-editor"
            :placeholder="t('rpdb.editor.placeholder.content')"
            @update:model-value="form.content = $event"
          >
            <template #toolbar>
              <button
                type="button"
                class="toolbar-slot toolbar-slot--featured"
                :class="{ active: quickJumpOpen }"
                :title="t('rpdb.editor.action.internalLink')"
                :aria-label="t('rpdb.editor.action.internalLink')"
                data-testid="rpdb-internal-link-button"
                @mousedown.prevent
                @click="toggleQuickJump"
              >
                <i class="ri-links-line"></i>
                <span>{{ t('rpdb.editor.action.internalLink') }}</span>
              </button>
            </template>
          </TiptapEditor>
        </div>

        <section v-if="isGuideType" class="guide-editor">
          <div class="section-heading">
            <div>
              <span>{{ t('rpdb.editor.guide.eyebrow') }}</span>
              <h2>{{ form.type === 'transmog' ? t('rpdb.editor.guide.transmogTitle') : t('rpdb.editor.guide.title') }}</h2>
              <p>{{ t('rpdb.editor.guide.description') }}</p>
            </div>
            <button type="button" @click="addStep"><i class="ri-add-line"></i>{{ t('rpdb.editor.action.addGuideStep') }}</button>
          </div>
          <div class="tomtom-import" data-testid="tomtom-import-panel">
            <div class="tomtom-import__mark">
              <i class="ri-route-line"></i>
              <span><b>{{ t('rpdb.editor.guide.tomtomTitle') }}</b><small>{{ t('rpdb.editor.guide.tomtomFormat') }}</small></span>
            </div>
            <label>
              <span>{{ t('rpdb.editor.guide.bulkCoordinates') }}</span>
              <textarea
                v-model="tomtomDraft"
                data-testid="tomtom-import-input"
                :placeholder="t('rpdb.editor.placeholder.tomtom')"
              ></textarea>
              <small>{{ t('rpdb.editor.guide.tomtomHelp') }}</small>
            </label>
            <button type="button" data-testid="tomtom-import-button" @click="importTomTomSteps"><i class="ri-map-pin-add-line"></i>{{ t('rpdb.editor.action.importRoute') }}</button>
          </div>
          <div v-if="form.guide_steps?.length" class="guide-step-list">
            <article v-for="(step, index) in form.guide_steps" :key="`${step.sort_order}-${index}`">
              <div class="step-number">{{ index + 1 }}</div>
              <div class="step-main">
                <div class="step-grid">
                <label><span>{{ t('rpdb.editor.field.stepName') }}</span><input v-model="step.title" :placeholder="t('rpdb.editor.placeholder.stepName')"></label>
                <label><span>{{ t('rpdb.editor.field.zone') }}</span><input v-model="step.zone" :placeholder="t('rpdb.editor.placeholder.zone')"></label>
              </div>
              <p v-if="step.title" class="step-title-preview">{{ step.title }}</p>
              <label><span>{{ t('rpdb.editor.field.stepDescription') }}</span><textarea v-model="step.body" :placeholder="t('rpdb.editor.placeholder.stepDescription')"></textarea></label>
                <label><span>{{ t('rpdb.editor.field.prerequisite') }}</span><input v-model="step.prerequisite" :placeholder="t('rpdb.editor.placeholder.prerequisite')"></label>
              </div>
              <div class="step-coordinate">
                <label><span>{{ t('rpdb.editor.field.mapId') }}</span><input v-model="step.map_id" placeholder="47"></label>
                <div>
                  <label><span>X</span><input v-model.number="step.x" type="number" min="0" max="100"></label>
                  <label><span>Y</span><input v-model.number="step.y" type="number" min="0" max="100"></label>
                </div>
                <code v-if="editorTomTomCommand(step, index)">{{ editorTomTomCommand(step, index) }}</code>
                <button type="button" class="remove" @click="removeStep(index)"><i class="ri-delete-bin-line"></i>{{ t('rpdb.editor.action.deleteStep') }}</button>
              </div>
            </article>
          </div>
          <div v-else class="empty-guide">
            <i class="ri-route-line"></i>
            <p>{{ t('rpdb.editor.guide.empty') }}</p>
          </div>
          <div class="guide-bottom-actions">
            <button type="button" data-testid="add-guide-step-bottom" @click="addStep">
              <i class="ri-add-line"></i>
              {{ t('rpdb.editor.action.addGuideStep') }}
            </button>
          </div>
        </section>

        <section v-else class="home-editor">
          <div class="section-heading">
            <div>
              <span>{{ t('rpdb.editor.workType.home.title') }}</span>
              <h2>{{ t('rpdb.editor.homeSupplement.title') }}</h2>
              <p>{{ t('rpdb.editor.homeSupplement.description') }}</p>
            </div>
          </div>
        </section>
      </main>

      <aside class="content-inspector">
        <details open data-testid="rpdb-content-checklist">
          <summary><span><i class="ri-list-check-3"></i>{{ t('rpdb.editor.checklist.title') }}</span><b>{{ form.type === 'transmog' ? form.transmog_slots?.filter(slot => slot.role !== 'unused').length || 0 : form.references?.length || 0 }}</b></summary>
          <div class="inspector-body">
            <div v-if="form.type === 'item_showcase'" class="checklist-stack" data-testid="item-content-checklist">
              <details
                v-for="(item, index) in form.references"
                :key="index"
                class="compact-card slot-row item-reference-row"
                data-testid="item-reference-panel"
                :open="openItemReferences.has(index)"
              >
                <summary class="slot-row__head" @click.prevent="toggleItemReference(index)">
                  <b>{{ item.name || t('rpdb.editor.checklist.itemNumber', { number: index + 1 }) }}</b>
                  <span><small>{{ itemTypeLabel(item.external_type) }}</small><i class="ri-arrow-down-s-line"></i></span>
                </summary>
                <div class="slot-row__body">
                  <label><span>{{ t('rpdb.editor.field.itemName') }}</span><input v-model="item.name" :placeholder="t('rpdb.editor.placeholder.itemNameShort')"></label>
                  <label><span>{{ t('rpdb.editor.field.itemDescription') }}</span><textarea v-model="item.description" :placeholder="t('rpdb.editor.placeholder.itemDescription')"></textarea></label>
                  <label><span>{{ t('rpdb.editor.field.itemType') }}</span><RPDBSelect v-model="item.external_type" :options="itemTypeOptions" /></label>
                  <label><span>{{ t('rpdb.editor.field.itemSource') }}</span><textarea v-model="item.acquisition_method" :placeholder="t('rpdb.editor.placeholder.itemSourceShort')"></textarea></label>
                  <button type="button" class="remove" @click="form.references!.splice(index, 1)"><i class="ri-delete-bin-line"></i>{{ t('rpdb.editor.action.remove') }}</button>
                </div>
              </details>
              <button type="button" class="add-button" @click="addItemReference"><i class="ri-add-line"></i>{{ t('rpdb.editor.action.addItem') }}</button>
            </div>

            <div v-else-if="form.type === 'transmog'" class="transmog-slot-checklist" data-testid="transmog-content-checklist">
              <details
                v-for="slot in form.transmog_slots"
                :key="slot.slot"
                class="compact-card slot-row"
                data-testid="transmog-slot-panel"
                :open="openTransmogSlots.has(slot.slot)"
              >
                <summary class="slot-row__head" @click.prevent="toggleTransmogSlot(slot.slot)">
                  <b>{{ slotOptions.find(option => option.value === slot.slot)?.label || slot.slot }}</b>
                  <span><small>{{ slot.role === 'unused' ? t('rpdb.editor.status.unused') : t('rpdb.editor.status.filled') }}</small><i class="ri-arrow-down-s-line"></i></span>
                </summary>
                <div class="slot-row__body">
                  <label>
                    <span>{{ t('rpdb.editor.field.slotStatus') }}</span>
                    <RPDBSelect v-model="slot.role" :options="slotRoleOptions" />
                  </label>
                  <label><span>{{ t('rpdb.editor.field.componentName') }}</span><input v-model="slot.name" :placeholder="t('rpdb.editor.placeholder.componentName')"></label>
                  <label><span>{{ t('rpdb.editor.field.componentDescription') }}</span><textarea v-model="slot.description" :placeholder="t('rpdb.editor.placeholder.componentDescription')"></textarea></label>
                  <label><span>{{ t('rpdb.editor.field.acquisitionSource') }}</span><input v-model="slot.source" :placeholder="t('rpdb.editor.placeholder.componentSource')"></label>
                  <details class="slot-extra-options" data-testid="transmog-slot-more-options">
                    <summary>
                      <span><i class="ri-equalizer-line"></i>{{ t('rpdb.editor.checklist.moreOptions') }}</span>
                      <small>{{ t('rpdb.editor.checklist.moreOptionsHint') }}</small>
                    </summary>
                    <div class="slot-extra-grid">
                      <label><span>{{ t('rpdb.editor.field.wowheadUrl') }}</span><input v-model="slot.wowhead_url" placeholder="https://www.wowhead.com/item=..."></label>
                      <label><span>{{ t('rpdb.editor.field.variant') }}</span><input v-model="slot.variant" :placeholder="t('rpdb.editor.placeholder.variant')"></label>
                    </div>
                  </details>
                </div>
              </details>
            </div>

            <div v-else class="checklist-stack" data-testid="home-content-checklist">
              <details
                v-for="(item, index) in form.references"
                :key="index"
                class="compact-card slot-row furniture-reference-row"
                data-testid="furniture-reference-panel"
                :open="openFurnitureReferences.has(index)"
              >
                <summary class="slot-row__head" @click.prevent="toggleFurnitureReference(index)">
                  <b>{{ item.name || t('rpdb.editor.checklist.furnitureNumber', { number: index + 1 }) }}</b>
                  <span><small>{{ item.name ? t('rpdb.editor.status.filled') : t('rpdb.editor.status.unfilled') }}</small><i class="ri-arrow-down-s-line"></i></span>
                </summary>
                <div class="slot-row__body">
                  <label><span>{{ t('rpdb.editor.field.furnitureName') }}</span><input v-model="item.name" :placeholder="t('rpdb.editor.placeholder.furnitureName')"></label>
                  <div class="furniture-icon-field" data-testid="furniture-icon-field">
                    <span>{{ t('rpdb.editor.field.icon') }}</span>
                    <div class="furniture-icon-control">
                      <span class="furniture-icon-preview" :class="{ empty: !item.icon }">
                        <img v-if="item.icon" :src="resolveRPDBMediaURL(item.icon)" :alt="t('rpdb.editor.media.furnitureIconAlt', { name: item.name || t('rpdb.editor.checklist.furniture') })">
                        <i v-else class="ri-image-line"></i>
                      </span>
                      <input v-model="item.icon" data-testid="furniture-icon-url" :placeholder="t('rpdb.editor.placeholder.iconUrl')">
                      <label class="furniture-icon-upload" :class="{ disabled: uploadingFurnitureIcon === item }" :title="t('rpdb.editor.action.uploadIcon')">
                        <input
                          type="file"
                          accept="image/*"
                          data-testid="furniture-icon-upload"
                          :disabled="uploadingFurnitureIcon === item"
                          @change="uploadFurnitureIcon($event, item)"
                        >
                        <i :class="uploadingFurnitureIcon === item ? 'ri-loader-4-line spin' : 'ri-upload-2-line'"></i>
                        <span>{{ uploadingFurnitureIcon === item ? t('rpdb.editor.status.uploading') : t('rpdb.editor.action.upload') }}</span>
                      </label>
                      <button v-if="item.icon" type="button" class="furniture-icon-clear" :title="t('rpdb.editor.action.clearIcon')" :aria-label="t('rpdb.editor.action.clearIcon')" @click="item.icon = ''"><i class="ri-close-line"></i></button>
                    </div>
                  </div>
                  <label><span>{{ t('rpdb.editor.field.wowheadUrl') }}</span><input v-model="item.url" placeholder="https://www.wowhead.com/item=..."></label>
                  <label><span>{{ t('rpdb.editor.field.acquisitionMethod') }}</span><textarea v-model="item.acquisition_method" :placeholder="t('rpdb.editor.placeholder.furnitureSource')"></textarea></label>
                  <label><span>{{ t('rpdb.editor.field.description') }}</span><textarea v-model="item.description" :placeholder="t('rpdb.editor.placeholder.furnitureDescription')"></textarea></label>
                  <button type="button" class="remove" @click="form.references!.splice(index, 1)"><i class="ri-delete-bin-line"></i>{{ t('rpdb.editor.action.remove') }}</button>
                </div>
              </details>
              <button type="button" class="add-button" @click="addFurnitureReference"><i class="ri-add-line"></i>{{ t('rpdb.editor.action.addFurniture') }}</button>
            </div>
              </div>
        </details>

      </aside>
    </section>

    <div class="floating-submit-toolbar" data-testid="floating-submit-toolbar">
      <div class="auto-save-state" :class="autoSaveState">
        <i class="ri-cloud-line"></i>
        <span>{{ autoSaveMessage }}</span>
      </div>
      <button type="button" :title="t('rpdb.editor.action.internalPreview')" @click="preview"><i class="ri-eye-line"></i><span>{{ t('rpdb.editor.action.internalPreview') }}</span></button>
      <button type="button" class="primary" data-testid="publish-work" :disabled="saving" @click="save('published')"><i class="ri-send-plane-2-line"></i><span>{{ saving ? t('rpdb.editor.status.publishing') : t('rpdb.editor.action.publish') }}</span></button>
    </div>

    <PostQuickJump v-model="quickJumpOpen" :on-insert="handleQuickInsert" />
    <ImageViewer v-model="showImageViewer" :images="viewerImages" :start-index="viewerStartIndex" />
  </div>
</template>

<style scoped>
.editor-page{max-width:1380px;margin:auto;color:var(--color-text-main)}
.minimal-editor-shell{--rpdb-surface:color-mix(in srgb,var(--color-panel-bg) 88%,var(--color-main-bg) 12%);--rpdb-muted:color-mix(in srgb,var(--color-card-bg) 82%,var(--color-main-bg) 18%);--rpdb-line:color-mix(in srgb,var(--color-border) 72%,transparent);--rpdb-soft:color-mix(in srgb,var(--color-accent) 8%,transparent)}
.editor-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:18px;margin-bottom:18px;padding-bottom:16px;border-bottom:1px solid var(--rpdb-line)}
.editor-heading>div:first-child>span,.writing-heading>div>span,.section-heading>div>span{color:var(--color-accent);font-size:11px;font-weight:800;letter-spacing:.06em}
.editor-heading h1{margin:6px 0 4px;color:var(--color-text-main);font:700 30px/1.2 system-ui,'Microsoft YaHei',sans-serif}
.editor-heading p{margin:0;color:var(--color-text-secondary)}
.heading-actions{display:flex;align-items:center;gap:10px}
.heading-actions>button{display:grid;width:36px;height:36px;place-items:center;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--rpdb-surface);color:var(--color-text-main)}
.saved-status{display:inline-flex;align-items:center;gap:6px;color:var(--color-success);font-size:12px}
.media-strip{display:grid;grid-template-columns:220px minmax(0,1fr) minmax(0,1fr);gap:14px;margin-bottom:14px;padding:16px;border:1px solid var(--rpdb-line);border-radius:14px;background:var(--rpdb-surface)}
.media-strip__heading{display:flex;min-width:0;flex-direction:column;justify-content:center}
.media-strip__heading span{color:var(--color-accent);font-size:11px;font-weight:800}
.media-strip__heading h2{margin:6px 0;color:var(--color-text-main);font-size:18px}
.media-strip__heading p{margin:0;color:var(--color-text-secondary);font-size:12px;line-height:1.55}
.media-upload-card{min-width:0;padding:12px;border:1px solid var(--rpdb-line);border-radius:13px;background:var(--rpdb-muted)}
.preview-gallery-panel{grid-column:1/-1;min-width:0;padding:12px;border:1px solid var(--rpdb-line);border-radius:13px;background:var(--rpdb-muted)}
.preview-gallery-panel .panel-heading{margin-bottom:10px}
.preview-gallery-panel :deep(.stage){min-height:220px;border-color:var(--rpdb-line);background:var(--color-panel-bg)}
.preview-gallery-panel :deep(.stage img),.preview-gallery-panel :deep(.stage video),.preview-gallery-panel :deep(.stage iframe){max-height:300px}
.preview-gallery-panel :deep(.empty){min-height:180px;color:var(--color-text-secondary)}
.preview-gallery-panel :deep(.thumbs){margin-top:10px}
.preview-gallery-panel :deep(.thumbs button){border-color:var(--rpdb-line);background:var(--color-panel-bg)}
.preview-gallery-panel :deep(.thumbs button.active){border-color:var(--color-accent)}
.editor-upper{display:grid;grid-template-columns:minmax(0,1fr);overflow:hidden;border:1px solid var(--rpdb-line);border-radius:14px;background:var(--rpdb-surface)}
.metadata-panel,.publish-panel{min-width:0;padding:16px}
.publish-panel{background:var(--rpdb-muted)}
.publish-panel{border-left:1px solid var(--rpdb-line)}
.panel-heading{display:flex;align-items:flex-start;justify-content:space-between;gap:10px;margin-bottom:12px}
.panel-heading>div{display:flex;align-items:center;gap:8px}
.panel-heading>div>span{color:var(--color-accent);font-size:11px;font-weight:800}
.panel-heading h2{margin:0;color:var(--color-text-main);font-size:15px}
.panel-heading small{color:var(--color-text-secondary);font-size:11px;text-align:right}
.media-upload{position:relative;display:grid;height:184px;place-items:center;overflow:hidden;border:1px dashed color-mix(in srgb,var(--color-accent) 48%,var(--rpdb-line));border-radius:12px;background:var(--rpdb-soft);cursor:pointer}
.media-upload input{display:none}
.media-upload img{width:100%;height:100%;object-fit:cover}
.media-upload>span{display:flex;flex-direction:column;align-items:center;gap:7px;color:var(--color-accent);text-align:center}
.media-upload>span i{font-size:30px}
.media-upload>span small{max-width:210px;color:var(--color-text-secondary);line-height:1.5}
.media-upload em{position:absolute;inset:auto 8px 8px;padding:5px 8px;border-radius:999px;background:rgba(25,17,12,.68);color:#fff;font-size:10px;font-style:normal}
.type-cards{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin-bottom:14px}
.visibility-setup{display:grid;grid-template-columns:minmax(210px,.75fr) minmax(240px,1fr);align-items:end;gap:12px;margin-bottom:14px;padding:0 0 14px;border-bottom:1px solid var(--rpdb-line)}
.visibility-setup__intro{display:flex;min-width:0;align-items:center;gap:9px;padding-bottom:4px}.visibility-setup__intro>i{display:grid;width:30px;height:30px;flex:0 0 auto;place-items:center;border-radius:7px;background:var(--rpdb-soft);color:var(--color-accent);font-size:17px}.visibility-setup__intro>span{display:grid;min-width:0;gap:3px}.visibility-setup__intro b{font-size:12px}.visibility-setup__intro small,.visibility-empty{color:var(--color-text-secondary);font-size:10px;font-weight:500;line-height:1.45}.visibility-empty{display:flex;min-height:34px;align-items:center;padding:7px 9px;border:1px dashed var(--rpdb-line);border-radius:7px;background:var(--rpdb-muted)}
.visibility-setup>[data-testid="visibility-guild-select"]{grid-column:2}
.visibility-setup>[data-testid="visibility-guild-select"]>span{display:flex;align-items:baseline;gap:5px}.visibility-setup>[data-testid="visibility-guild-select"]>span small{color:var(--color-text-secondary);font-size:10px;font-weight:500}.visibility-guild-options{display:flex;flex-wrap:wrap;gap:7px}.visibility-guild-option{display:flex;min-width:0;align-items:center;gap:7px;padding:7px 9px;border:1px solid var(--rpdb-line);border-radius:7px;background:var(--color-panel-bg);cursor:pointer}.visibility-guild-option.selected{border-color:var(--rpdb-focus);background:var(--rpdb-soft)}.visibility-guild-option input{position:absolute;width:1px;height:1px;opacity:0;pointer-events:none}.visibility-guild-option>span{display:grid;min-width:0;gap:2px}.visibility-guild-option>span b{font-size:11px}.visibility-guild-option>span small{color:var(--color-text-secondary);font-size:9px;font-weight:500}.visibility-guild-option>i{color:var(--color-accent);font-size:15px}
.type-cards button{display:flex;min-width:0;align-items:center;gap:8px;padding:10px;border:1px solid var(--rpdb-line);border-radius:12px;background:transparent;color:var(--color-text-main);text-align:left}
.type-cards button.active{border-color:color-mix(in srgb,var(--color-accent) 68%,var(--rpdb-line));background:var(--rpdb-soft)}
.type-cards i{flex:0 0 auto;color:var(--color-accent);font-size:20px}
.type-cards button>span{display:flex;min-width:0;flex-direction:column}
.type-cards small{margin-top:2px;overflow:hidden;color:var(--color-text-secondary);font-size:10px;text-overflow:ellipsis;white-space:nowrap}
  .metadata-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:10px}
  .span-2{grid-column:1/-1}
  .style-picker{display:grid;gap:8px;color:var(--color-text-main);font-weight:700}
  .style-picker>span{font-size:12px}
  .style-picker small{color:var(--color-text-secondary);font-size:11px;font-weight:500}
  .selected-topics{display:flex;min-height:38px;flex-wrap:wrap;gap:7px;align-items:center;padding:8px;border:1px solid var(--rpdb-line);border-radius:9px;background:var(--color-panel-bg)}
  .selected-topics button{display:inline-flex;align-items:center;gap:6px;min-height:28px;padding:0 8px;border:1px solid color-mix(in srgb,var(--tag-color) 58%,var(--rpdb-line));border-radius:999px;background:color-mix(in srgb,var(--tag-color) 16%,transparent);color:var(--color-text-main);font-size:12px;font-weight:800}
  .selected-topics i{color:var(--color-text-secondary);font-size:14px}
  .topic-empty{color:var(--color-text-secondary);font-size:12px;font-weight:500}
  .topic-input{display:block}
  .topic-input input{min-height:36px}
  .style-options{display:flex;flex-wrap:wrap;gap:7px}
  .style-options button{display:inline-flex;align-items:center;min-height:30px;padding:0 10px;border:1px solid color-mix(in srgb,var(--tag-color) 52%,var(--rpdb-line));border-radius:999px;background:color-mix(in srgb,var(--tag-color) 9%,transparent);color:var(--color-text-main);font-size:12px;font-weight:700}
  .style-options button.selected{background:var(--tag-color);border-color:var(--tag-color);color:var(--btn-primary-text)}
  .style-options button.disabled{opacity:.55;cursor:not-allowed}
label{display:grid;gap:6px;color:var(--color-text-main);font-weight:700}
label>span{font-size:12px}
input,textarea,select{width:100%;box-sizing:border-box;padding:10px 11px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg);color:var(--color-text-main);font:inherit}
textarea{min-height:70px;resize:vertical}
.check-list{display:grid;gap:4px;margin-bottom:12px}
.check-list>div{display:grid;grid-template-columns:20px 1fr auto;gap:7px;align-items:center;padding:8px 0;color:var(--color-text-secondary)}
.check-list i,.check-list b{color:var(--btn-danger-bg)}
.check-list b{font-size:10px}
.check-list .done i,.check-list .done b{color:var(--color-success)}
.change-summary textarea{min-height:58px}
.publish-actions{display:grid;gap:8px;margin-top:12px}
.publish-actions button,.section-heading button,.add-button,.remove{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-height:36px;padding:0 12px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg);color:var(--color-text-main)}
.publish-actions .primary{border-color:var(--color-accent);background:var(--color-accent);color:var(--btn-primary-text)}
.editor-lower{display:grid;grid-template-columns:minmax(0,1fr) 300px;gap:14px;margin-top:14px;align-items:start}
.writing-workspace,.content-inspector details{min-width:0;overflow:hidden;border:1px solid var(--rpdb-line);border-radius:14px;background:var(--rpdb-surface)}
.writing-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:14px;padding:18px 20px;border-bottom:1px solid var(--rpdb-line)}
.writing-heading h2,.section-heading h2{margin:4px 0;color:var(--color-text-main);font:700 22px/1.25 system-ui,'Microsoft YaHei',sans-serif}
.writing-heading p,.section-heading p{margin:0;color:var(--color-text-secondary);font-size:12px}
.outline-chips{display:flex;flex-wrap:wrap;justify-content:flex-end;gap:6px}
.outline-chips span{padding:5px 9px;border:1px solid var(--rpdb-line);border-radius:999px;color:var(--color-text-secondary);font-size:10px}
.rich-editor-shell{padding:16px 20px;border-bottom:1px solid var(--rpdb-line)}
.rich-editor-shell :deep(.tiptap-editor){min-height:390px}
.context-fields{display:grid;grid-template-columns:1fr 1fr;gap:10px;padding:18px 20px;border-bottom:1px solid var(--rpdb-line);background:var(--rpdb-muted)}
.context-fields textarea{min-height:92px;background:var(--color-panel-bg)}
.guide-editor,.home-editor{padding:22px 20px}
.section-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:14px;margin-bottom:14px}
.section-heading button{border-color:color-mix(in srgb,var(--color-accent) 70%,var(--rpdb-line));color:var(--color-accent)}
.tomtom-import{display:grid;grid-template-columns:150px minmax(0,1fr) auto;align-items:end;gap:12px;margin-bottom:14px;padding:12px;border:1px solid color-mix(in srgb,var(--color-accent) 32%,var(--rpdb-line));border-radius:10px;background:color-mix(in srgb,var(--color-accent) 6%,var(--color-panel-bg))}
.tomtom-import__mark{display:flex;align-self:stretch;align-items:center;gap:9px;padding-right:12px;border-right:1px solid var(--rpdb-line)}
.tomtom-import__mark>i{color:var(--color-accent);font-size:23px}.tomtom-import__mark>span{display:flex;min-width:0;flex-direction:column}.tomtom-import__mark b{font-size:12px}.tomtom-import__mark small{margin-top:4px;color:var(--color-text-secondary);font:10px/1.35 Consolas,monospace}
.tomtom-import label{display:grid;min-width:0;gap:5px}.tomtom-import label>span{font-size:11px;font-weight:800}.tomtom-import label>small{color:var(--color-text-secondary);font-size:10px;font-weight:500;line-height:1.5}.tomtom-import textarea{min-height:82px;font:11px/1.55 Consolas,'SFMono-Regular',monospace;resize:vertical}
.tomtom-import>button{display:inline-flex;min-height:38px;align-items:center;justify-content:center;gap:6px;padding:0 13px;border:1px solid var(--color-accent);border-radius:9px;background:var(--color-accent);color:var(--btn-primary-text);font-weight:800;white-space:nowrap}
.guide-step-list{display:grid;gap:10px}
.guide-step-list article{display:grid;grid-template-columns:34px minmax(0,1fr) 200px;gap:12px;padding:13px;border:1px solid var(--rpdb-line);border-radius:12px;background:var(--rpdb-muted)}
.step-number{display:grid;width:30px;height:30px;place-items:center;border-radius:50%;background:var(--rpdb-soft);color:var(--color-accent);font-weight:800}
.step-main{display:grid;gap:8px}
.step-grid{display:grid;grid-template-columns:1fr 180px;gap:8px}
.step-coordinate{display:flex;flex-direction:column;gap:7px;padding-left:12px;border-left:1px solid var(--rpdb-line)}
.step-coordinate>div{display:grid;grid-template-columns:1fr 1fr;gap:7px}
.step-coordinate code{overflow-wrap:anywhere;padding:8px;border-radius:9px;background:var(--rpdb-soft);color:var(--color-accent);font:10px/1.5 Consolas,monospace}
.remove{min-height:30px;color:var(--btn-danger-bg)}
.empty-guide{display:grid;min-height:140px;place-items:center;align-content:center;color:var(--color-text-secondary);text-align:center}
.empty-guide i{font-size:34px;color:var(--color-accent)}
.guide-bottom-actions{display:flex;justify-content:flex-end;margin-top:12px;padding-top:12px;border-top:1px solid var(--rpdb-line)}
.guide-bottom-actions button{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-height:36px;padding:0 14px;border:1px solid color-mix(in srgb,var(--color-accent) 70%,var(--rpdb-line));border-radius:10px;background:var(--color-panel-bg);color:var(--color-accent);font-weight:700}
.home-grid{display:grid;grid-template-columns:1fr 1fr;gap:10px}
.content-inspector{display:grid;gap:10px}
.content-inspector summary{display:flex;align-items:center;justify-content:space-between;gap:8px;padding:13px 14px;cursor:pointer;list-style:none;color:var(--color-text-main);font-weight:800}
.content-inspector summary::-webkit-details-marker{display:none}
.content-inspector summary>span{display:inline-flex;align-items:center;gap:7px}
.content-inspector summary i{color:var(--color-accent)}
.content-inspector summary b{color:var(--color-text-secondary);font-size:11px}
.inspector-body{display:grid;gap:8px;padding:10px;border-top:1px solid var(--rpdb-line);background:var(--rpdb-muted)}
.compact-card{display:grid;gap:7px;padding:9px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg)}
.add-button{border-style:dashed;border-color:color-mix(in srgb,var(--color-accent) 70%,var(--rpdb-line));background:transparent;color:var(--color-accent)}
.restriction-grid{display:grid;gap:9px}
.checklist-stack{display:grid;gap:8px}
.transmog-slot-checklist{display:grid;grid-template-columns:1fr;gap:8px;max-height:680px;overflow:auto;padding-right:2px}
.slot-row{grid-template-columns:1fr}
.slot-row__head{display:flex;align-items:center;justify-content:space-between;gap:8px}
.slot-row__head b{color:var(--color-text-main);font-size:12px}
.slot-row__head small{color:var(--color-text-secondary);font-size:10px}
.slot-extra-options{overflow:hidden;border:1px solid var(--rpdb-line);border-radius:8px;background:var(--rpdb-muted)}
.slot-extra-options summary{display:flex;align-items:center;justify-content:space-between;gap:8px;padding:9px 10px;cursor:pointer;list-style:none;color:var(--color-text-main);font-size:12px;font-weight:800}
.slot-extra-options summary::-webkit-details-marker{display:none}
.slot-extra-options summary>span{display:inline-flex;align-items:center;gap:6px}
.slot-extra-options summary i{color:var(--color-accent)}
.slot-extra-options summary small{color:var(--color-text-secondary);font-size:10px;font-weight:600}
.slot-extra-grid{display:grid;gap:7px;padding:9px;border-top:1px solid var(--rpdb-line);background:var(--color-panel-bg)}
@media(max-width:1160px){.media-strip{grid-template-columns:1fr 1fr}.media-strip__heading{grid-column:1/-1}.publish-panel{border-top:1px solid var(--rpdb-line);border-left:0}.publish-panel .check-list{grid-template-columns:1fr 1fr;gap:0 18px}.publish-actions{grid-template-columns:repeat(3,1fr)}.editor-lower{grid-template-columns:1fr}.content-inspector{grid-template-columns:repeat(2,1fr)}}
@media(max-width:760px){.editor-heading,.writing-heading,.section-heading{align-items:flex-start;flex-direction:column}.media-strip,.editor-upper,.tomtom-import{grid-template-columns:1fr}.tomtom-import__mark{padding:0 0 10px;border-right:0;border-bottom:1px solid var(--rpdb-line)}.tomtom-import>button{width:100%}.type-cards,.metadata-grid,.context-fields,.home-grid,.content-inspector{grid-template-columns:1fr}.span-2{grid-column:auto}.publish-panel .check-list{grid-template-columns:1fr}.publish-actions{grid-template-columns:1fr}.guide-step-list article{grid-template-columns:32px 1fr}.step-coordinate{grid-column:2;padding:10px 0 0;border-top:1px dashed var(--rpdb-line);border-left:0}.step-grid{grid-template-columns:1fr}.outline-chips{justify-content:flex-start}}
@media(max-width:760px){.visibility-setup{grid-template-columns:1fr}.visibility-setup>[data-testid="visibility-guild-select"]{grid-column:auto}}
@media(max-width:480px){.editor-heading h1{font-size:26px}.media-upload{height:165px}.step-coordinate>div{grid-template-columns:1fr 1fr}}

/* Focused authoring workflow */
.editor-page{
  max-width:1380px;
  padding:0 12px 96px;
}
.minimal-editor-shell{
  --rpdb-surface:color-mix(in srgb,var(--color-panel-bg) 94%,var(--color-main-bg) 6%);
  --rpdb-muted:color-mix(in srgb,var(--color-card-bg) 88%,var(--color-main-bg) 12%);
  --rpdb-line:color-mix(in srgb,var(--color-border) 64%,transparent);
  --rpdb-soft:color-mix(in srgb,var(--color-accent) 9%,transparent);
  --rpdb-focus:color-mix(in srgb,var(--color-accent) 68%,var(--rpdb-line));
  --rpdb-success:var(--color-success);
  --rpdb-danger:var(--btn-danger-bg);
  --rpdb-shadow:0 8px 24px rgba(var(--shadow-base),.055);
}
.editor-heading{
  align-items:center;
  margin-bottom:10px;
  padding:4px 2px 14px;
}
.editor-heading>div:first-child>span{
  display:none;
}
.editor-heading h1{
  margin:0;
  font-size:24px;
  letter-spacing:0;
}
.editor-heading p{
  margin-top:5px;
  font-size:12px;
}
.heading-actions>button{
  width:32px;
  height:32px;
  border-radius:8px;
  box-shadow:none;
}
.saved-status{
  padding:4px 8px;
  border:1px solid color-mix(in srgb,var(--color-success) 20%,transparent);
  border-radius:999px;
  background:var(--color-success-light);
}
.section-anchor{
  scroll-margin-top:92px;
}
.media-strip{
  grid-template-columns:190px repeat(2,minmax(0,1fr));
  gap:14px;
  margin-bottom:12px;
  padding:16px;
  border-radius:8px;
  box-shadow:var(--rpdb-shadow);
}
.media-strip__heading h2{
  margin:5px 0;
  font-size:17px;
}
.media-strip__heading p{
  max-width:180px;
  font-size:12px;
}
.media-upload-card{
  display:grid;
  grid-template-columns:1fr;
  gap:8px;
  align-items:start;
  padding-left:14px;
  border-left:1px solid var(--rpdb-line);
  border-radius:0;
  background:transparent;
}
.media-upload-card .panel-heading{
  margin:0;
  align-content:start;
}
.media-upload-card .panel-heading>div{
  display:block;
}
.media-upload-card .panel-heading h2{
  margin-top:4px;
  font-size:14px;
}
.media-upload-card .panel-heading small{
  display:block;
  margin-top:6px;
  text-align:left;
}
.media-upload{
  height:132px;
  border-radius:8px;
}
.media-upload>span{
  gap:3px;
}
.media-upload>span i{
  font-size:20px;
}
.media-upload>span small{
  padding:0 56px;
  font-size:10px;
}
.media-upload__action{
  position:absolute;
  right:8px;
  bottom:8px;
  z-index:2;
  display:inline-flex;
  min-height:28px;
  align-items:center;
  gap:5px;
  padding:0 9px;
  border:1px solid color-mix(in srgb,var(--color-accent) 55%,var(--rpdb-line));
  border-radius:6px;
  background:color-mix(in srgb,var(--color-panel-bg) 92%,transparent);
  color:var(--color-accent);
  font-size:10px;
  font-weight:800;
  cursor:pointer;
  backdrop-filter:blur(8px);
}
.media-remove{
  position:absolute;
  top:8px;
  right:8px;
  z-index:3;
  display:grid;
  width:28px;
  height:28px;
  place-items:center;
  border:1px solid rgba(255,255,255,.52);
  border-radius:6px;
  background:rgba(37,27,20,.66);
  color:#fff;
}
.editor-upper{
  margin:0 0 12px;
  border-radius:8px;
  box-shadow:var(--rpdb-shadow);
}
.metadata-panel{
  padding:18px 20px 20px;
}
.metadata-panel>.panel-heading:not(:first-child){
  margin-top:22px;
  padding-top:18px;
  border-top:1px solid var(--rpdb-line);
}
.panel-heading{
  margin-bottom:12px;
}
.panel-heading h2{
  font-size:16px;
}
.panel-heading>div>span,
.media-strip__heading>span,
.writing-heading>div>span,
.section-heading>div>span{
  letter-spacing:0;
}
.panel-heading>small b{
  margin-right:2px;
}
.type-cards{
  gap:8px;
  margin:8px 0 18px;
}
.type-cards button{
  min-height:64px;
  padding:10px 12px;
  border-radius:8px;
  background:var(--color-panel-bg);
  transition:border-color .16s ease,background-color .16s ease,box-shadow .16s ease;
}
.type-cards button.active{
  border-color:var(--rpdb-focus);
  background:color-mix(in srgb,var(--color-accent) 8%,var(--color-panel-bg));
  box-shadow:inset 3px 0 0 var(--color-accent);
}
.type-cards i{
  display:grid;
  width:32px;
  height:32px;
  place-items:center;
  border-radius:999px;
  background:var(--rpdb-soft);
  font-size:18px;
}
.type-cards b{
  font-size:14px;
}
.type-cards small{
  font-size:11px;
}
.metadata-grid{
  grid-template-columns:repeat(3,minmax(0,1fr));
  gap:14px 16px;
}
.metadata-grid>label.span-2{
  grid-column:span 2;
}
.metadata-grid>.style-picker.span-2{
  grid-column:1/-1;
}
.style-picker{
  padding:12px;
  border:1px solid var(--rpdb-line);
  border-radius:8px;
  background:var(--rpdb-muted);
}
.style-options{
  gap:8px;
}
.style-options button{
  min-height:28px;
  gap:6px;
  border-radius:6px;
  background:var(--color-panel-bg);
}
.style-options button::before{
  width:7px;
  height:7px;
  flex:0 0 auto;
  border-radius:50%;
  background:var(--tag-color);
  box-shadow:0 0 0 2px color-mix(in srgb,var(--tag-color) 18%,transparent);
  content:'';
}
.style-options button.style-options-toggle{
  border-style:dashed;
  border-color:var(--rpdb-line);
  color:var(--color-accent);
}
.style-options button.style-options-toggle::before{
  display:none;
}
.required-mark{
  color:var(--rpdb-danger);
  font-size:12px;
  font-style:normal;
}
.optional-label{
  margin-left:4px;
  color:var(--color-text-secondary);
  font-size:10px;
  font-style:normal;
  font-weight:500;
}
.field-control>small{
  color:var(--color-text-secondary);
  font-size:10px;
  font-weight:500;
}
.field-control>small.field-error{
  color:var(--rpdb-danger);
}
.field-control.invalid input{
  border-color:var(--rpdb-danger);
  background:color-mix(in srgb,var(--rpdb-danger) 4%,var(--color-panel-bg));
}
.type-form-panel{
  padding:14px;
  border:1px solid var(--rpdb-line);
  border-radius:8px;
  background:color-mix(in srgb,var(--rpdb-muted) 72%,transparent);
}
.home-grid{
  grid-template-columns:repeat(6,minmax(0,1fr));
  gap:10px 12px;
}
.home-grid>label{
  grid-column:span 2;
}
.home-grid>.span-2{
  grid-column:1/-1;
}
.home-code-row{
  display:grid;
  grid-template-columns:minmax(0,1.2fr) minmax(270px,.8fr);
  gap:12px;
  align-items:stretch;
  padding:12px;
  border:1px solid var(--rpdb-line);
  border-radius:10px;
  background:var(--color-panel-bg);
}
.home-code-row textarea{
  min-height:118px;
  font-family:Consolas,'SFMono-Regular',monospace;
  line-height:1.55;
}
.home-code-upload{
  min-height:118px;
  place-content:center;
  justify-items:center;
  border:1px dashed var(--rpdb-focus);
  border-radius:8px;
  background:var(--rpdb-soft);
  cursor:pointer;
}
.home-code-upload input{
  display:none;
}
.home-code-upload>span{
  display:grid;
  justify-items:center;
  gap:5px;
  color:var(--color-accent);
  text-align:center;
}
.home-code-upload i{
  font-size:22px;
}
.home-code-upload small{
  color:var(--color-text-secondary);
  font-weight:500;
}
input,textarea,select{
  min-height:34px;
  padding:8px 10px;
  border-radius:7px;
}
input:focus,textarea:focus{
  outline:none;
  border-color:var(--rpdb-focus);
  box-shadow:0 0 0 3px color-mix(in srgb,var(--color-accent) 12%,transparent);
}
button:focus-visible,
label[for]:focus-visible,
summary:focus-visible{
  outline:2px solid color-mix(in srgb,var(--color-accent) 72%,var(--color-main-bg));
  outline-offset:2px;
}
select{
  appearance:none;
  -webkit-appearance:none;
  padding-right:34px;
  cursor:pointer;
  background:
    linear-gradient(45deg,transparent 50%,var(--color-text-secondary) 50%) calc(100% - 18px) 50%/5px 5px no-repeat,
    linear-gradient(135deg,var(--color-text-secondary) 50%,transparent 50%) calc(100% - 13px) 50%/5px 5px no-repeat,
    linear-gradient(180deg,color-mix(in srgb,var(--color-panel-bg) 96%,var(--color-main-bg) 4%),color-mix(in srgb,var(--color-card-bg) 82%,var(--color-main-bg) 18%));
  box-shadow:inset 0 1px 0 color-mix(in srgb,var(--color-text-main) 12%,transparent);
}
select:hover{
  border-color:color-mix(in srgb,var(--color-accent) 38%,var(--rpdb-line));
  background:
    linear-gradient(45deg,transparent 50%,var(--color-accent) 50%) calc(100% - 18px) 50%/5px 5px no-repeat,
    linear-gradient(135deg,var(--color-accent) 50%,transparent 50%) calc(100% - 13px) 50%/5px 5px no-repeat,
    linear-gradient(180deg,color-mix(in srgb,var(--color-panel-bg) 98%,var(--color-main-bg) 2%),color-mix(in srgb,var(--color-accent) 6%,var(--color-card-bg)));
}
select:focus{
  outline:none;
  border-color:var(--rpdb-focus);
  box-shadow:0 0 0 3px color-mix(in srgb,var(--color-accent) 14%,transparent),inset 0 1px 0 color-mix(in srgb,var(--color-text-main) 12%,transparent);
}
select:disabled{
  cursor:not-allowed;
  opacity:.62;
}
select option{
  background:var(--color-panel-bg);
  color:var(--color-text-main);
}
textarea{
  min-height:64px;
}
label{
  gap:5px;
}
label>span{
  font-size:11px;
}
.editor-lower{
  grid-template-columns:minmax(0,1fr) 320px;
  gap:12px;
  margin-top:0;
}
.writing-workspace,.content-inspector details{
  border-radius:8px;
  box-shadow:var(--rpdb-shadow);
}
.writing-heading{
  padding:17px 20px;
}
.writing-heading h2,.section-heading h2{
  font-size:19px;
}
.rich-editor-shell{
  padding:16px 20px;
}
.rich-editor-shell :deep(.tiptap-editor){
  min-height:360px;
  border-radius:8px;
}
.context-fields{
  padding:12px 16px;
}
.context-fields textarea{
  min-height:74px;
}
.guide-editor,.home-editor{
  padding:18px 20px 20px;
}
.home-code-preview{
  display:grid;
  gap:8px;
  padding:12px;
  border:1px solid var(--rpdb-line);
  border-radius:9px;
  background:var(--rpdb-muted);
}
.home-code-preview>span{
  color:var(--color-text-main);
  font-size:12px;
  font-weight:800;
}
.home-code-preview pre{
  max-height:160px;
  margin:0;
  overflow:auto;
  white-space:pre-wrap;
  word-break:break-word;
  padding:10px;
  border:1px solid var(--rpdb-line);
  border-radius:8px;
  background:var(--color-panel-bg);
  color:var(--color-text-secondary);
  font:11px/1.55 Consolas,'SFMono-Regular',monospace;
}
.content-inspector{
  position:sticky;
  top:82px;
  gap:10px;
}
.content-inspector details{
  max-height:calc(100vh - 104px);
  overflow:auto;
  border-radius:8px;
}
.content-inspector summary{
  padding:11px 12px;
  font-size:13px;
}
.inspector-body{
  padding:10px;
}
.compact-card{
  border-radius:8px;
}
.slot-row.compact-card{
  display:block;
  overflow:visible;
  padding:0;
}
.content-inspector .slot-row,
.content-inspector .slot-extra-options{
  overflow:visible;
}
.content-inspector .slot-row:not([open]) .slot-row__body{
  display:none;
}
.content-inspector .slot-row[open] .slot-row__body{
  display:grid;
}
.slot-row__head{
  display:flex;
  min-height:42px;
  align-items:center;
  justify-content:space-between;
  gap:10px;
  padding:0 12px;
  cursor:pointer;
  line-height:1;
  list-style:none;
}
.slot-row__head::-webkit-details-marker{
  display:none;
}
.slot-row__head b{
  display:inline-flex;
  min-height:22px;
  align-items:center;
  color:var(--color-text-main);
  font-size:12px;
}
.slot-row__head>span{
  display:inline-flex;
  min-height:22px;
  align-items:center;
  gap:6px;
  color:var(--color-text-secondary);
}
.slot-row__head small{
  font-size:10px;
  font-weight:700;
}
.slot-row__head i{
  color:var(--color-accent);
  font-size:16px;
  transition:transform .16s ease;
}
.slot-row[open] .slot-row__head i{
  transform:rotate(180deg);
}
.slot-row__body{
  display:grid;
  gap:7px;
  padding:9px;
  border-top:1px solid var(--rpdb-line);
}
.furniture-icon-field{
  display:grid;
  gap:6px;
}
.furniture-icon-field>span{
  font-size:11px;
  font-weight:800;
}
.furniture-icon-control{
  display:grid;
  grid-template-columns:34px minmax(0,1fr) auto auto;
  align-items:center;
  gap:6px;
}
.furniture-icon-preview{
  display:grid;
  width:34px;
  height:34px;
  place-items:center;
  overflow:hidden;
  border:1px solid var(--rpdb-line);
  border-radius:7px;
  background:var(--rpdb-muted);
  color:var(--color-text-secondary);
}
.furniture-icon-preview img{
  width:100%;
  height:100%;
  object-fit:cover;
}
.furniture-icon-control>input{
  min-width:0;
  height:34px;
  padding:7px 8px;
}
.furniture-icon-upload{
  display:inline-flex;
  min-height:34px;
  align-items:center;
  justify-content:center;
  gap:5px;
  padding:0 9px;
  border:1px solid var(--rpdb-line);
  border-radius:7px;
  background:var(--color-panel-bg);
  color:var(--color-text-main);
  font-size:11px;
  cursor:pointer;
}
.furniture-icon-upload input{
  display:none;
}
.furniture-icon-upload.disabled{
  cursor:wait;
  opacity:.65;
}
.furniture-icon-clear{
  display:grid;
  width:34px;
  height:34px;
  place-items:center;
  padding:0;
  border:1px solid var(--rpdb-line);
  border-radius:7px;
  background:var(--color-panel-bg);
  color:var(--color-text-secondary);
}
.spin{
  animation:spin .9s linear infinite;
}
@keyframes spin{
  to{transform:rotate(360deg)}
}
.floating-submit-toolbar{
  position:fixed;
  right:20px;
  bottom:18px;
  z-index:80;
  display:flex;
  align-items:center;
  justify-content:flex-end;
  gap:7px;
  padding:8px;
  border:1px solid var(--rpdb-line);
  border-radius:8px;
  background:color-mix(in srgb,var(--color-panel-bg) 94%,transparent);
  box-shadow:0 12px 32px rgba(var(--shadow-base),.16);
  backdrop-filter:blur(16px);
}
.floating-submit-toolbar .auto-save-state{
  display:inline-flex;
  min-width:0;
  align-items:center;
  gap:7px;
  color:var(--color-text-secondary);
  font-size:11px;
  white-space:nowrap;
}
.floating-submit-toolbar .auto-save-state.saved,
.floating-submit-toolbar .auto-save-state.local{
  color:var(--color-success);
}
.floating-submit-toolbar .auto-save-state.error{
  color:var(--btn-danger-bg);
}
.floating-submit-toolbar button{
  display:inline-flex;
  align-items:center;
  justify-content:center;
  gap:7px;
  min-height:34px;
  padding:0 12px;
  border:1px solid var(--rpdb-line);
  border-radius:7px;
  background:var(--color-panel-bg);
  color:var(--color-text-main);
  font-size:12px;
  font-weight:700;
}
.floating-submit-toolbar button.primary{
  border-color:var(--color-accent);
  background:var(--color-accent);
  color:var(--btn-primary-text);
  box-shadow:0 5px 12px color-mix(in srgb,var(--color-accent) 22%,transparent);
}
.floating-submit-toolbar button:disabled{
  cursor:wait;
  opacity:.68;
}
@media(max-width:1100px){
  .media-strip{grid-template-columns:repeat(2,minmax(0,1fr))}
  .media-strip__heading{grid-column:1/-1}
  .media-strip__heading p{max-width:none}
  .media-upload-card:first-of-type{padding-left:0;border-left:0}
  .metadata-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
  .metadata-grid>label.span-2{grid-column:1/-1}
  .home-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
  .home-grid>label{grid-column:auto}
  .home-code-row{grid-template-columns:1fr}
  .editor-lower{grid-template-columns:1fr}
  .content-inspector{position:static}
  .content-inspector details{max-height:none}
}
@media(max-width:760px){
  .editor-page{padding:0 4px 132px}
  .editor-heading{align-items:flex-start;flex-direction:row}
  .editor-heading h1{font-size:21px}
  .editor-heading p{max-width:290px}
  .saved-status{display:none}
  .floating-submit-toolbar{right:10px;bottom:10px;left:10px;display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr)}
  .floating-submit-toolbar .auto-save-state{grid-column:1/-1;margin:0}
  .floating-submit-toolbar button{width:100%;padding:0 8px}
  .media-strip{grid-template-columns:1fr;padding:14px}
  .media-strip__heading{grid-column:auto}
  .media-upload-card{
    padding:14px 0 0;
    border-top:1px solid var(--rpdb-line);
    border-left:0;
  }
  .media-upload>span small{padding:0 20px}
  .metadata-panel{padding:16px 14px 18px}
  .type-cards,.metadata-grid,.home-grid{grid-template-columns:1fr}
  .metadata-grid>label.span-2,.metadata-grid>.style-picker.span-2,.home-grid>.span-2{grid-column:auto}
  .type-cards button{min-height:54px}
  .home-code-row{min-width:0;padding:10px}
  .writing-heading,.section-heading{align-items:flex-start;flex-direction:column}
  .writing-heading{padding:15px 14px}
  .rich-editor-shell,.guide-editor,.home-editor{padding-right:14px;padding-left:14px}
  .outline-chips{justify-content:flex-start}
}
@media(max-width:480px){
  .editor-heading p{font-size:11px}
  .floating-submit-toolbar button{font-size:11px}
  .step-coordinate>div{grid-template-columns:1fr}
}
@media(prefers-reduced-motion:reduce){
  .type-cards button,.slot-row__head i{transition:none}
}
</style>
