<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  createRPDBWork,
  getRPDBWork,
  resolveRPDBMediaURL,
  updateRPDBWork,
  type RPDBGuideStep,
  type RPDBReference,
  type RPDBWorkPayload,
  type RPDBWorkType,
} from '@/api/rpdb'
import { uploadImage } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
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
const toast = useToastStore()
const saving = ref(false)
const lastSaved = ref('')
const autoSaveState = ref<'idle' | 'local' | 'saving' | 'saved' | 'error'>('idle')
const autoSaveMessage = computed(() => {
  if (autoSaveState.value === 'saving') return '自动保存中'
  if (autoSaveState.value === 'saved') return `已自动保存 ${lastSaved.value}`
  if (autoSaveState.value === 'local') return '已本地保存'
  if (autoSaveState.value === 'error') return '自动保存失败'
  return '正在编辑'
})
const hasLoadedInitialData = ref(false)
const autosavedWorkId = ref<number | null>(null)
let autoSaveTimer: ReturnType<typeof window.setTimeout> | null = null
const rpStyleTags = ref<Tag[]>([])
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
  is_public: false,
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
const typeOptions: Array<{ id: RPDBWorkType; icon: string; title: string; description: string }> = [
  {
    id: 'item_showcase',
    icon: 'ri-magic-line',
    title: '魔兽物品',
    description: '分享实际效果、使用场景和获取攻略',
  },
  {
    id: 'transmog',
    icon: 'ri-shirt-line',
    title: '幻化方案',
    description: '整理整套搭配、替代件和部件攻略',
  },
  {
    id: 'home_showcase',
    icon: 'ri-home-heart-line',
    title: '家宅分享',
    description: '展示住宅空间、参观信息和分享代码',
  },
]
const availabilityOptions = [
  { value: 'available', label: '可获取', hint: '当前版本仍可取得' },
  { value: 'limited', label: '限时获取', hint: '节日、活动或轮换内容' },
  { value: 'removed', label: '已绝版', hint: '当前无法正常取得' },
  { value: 'unknown', label: '未知', hint: '等待作者补充确认' },
]
const itemTypeOptions = [
  { value: 'item', label: '物品', hint: '一般消耗品、材料或剧情物件' },
  { value: 'equipment', label: '装备', hint: '角色可以装备的物品' },
  { value: 'toy', label: '玩具', hint: '收藏或可重复使用的玩具' },
  { value: 'quest_item', label: '任务道具', hint: '任务、事件或剧情流程使用' },
]
const bindOptions = [
  { value: 'no', label: '否', hint: '物品不绑定角色或账号' },
  { value: 'yes', label: '是', hint: '物品会绑定角色或账号' },
]
const factionOptions = [
  { value: 'neutral', label: '不限', hint: '联盟和部落均可用' },
  { value: 'alliance', label: '联盟', hint: '联盟角色适用' },
  { value: 'horde', label: '部落', hint: '部落角色适用' },
]
const armorTypeOptions = [
  { value: '', label: '不限', hint: '不限制护甲类型' },
  { value: 'cloth', label: '布甲', hint: '布甲职业外观' },
  { value: 'leather', label: '皮甲', hint: '皮甲职业外观' },
  { value: 'mail', label: '锁甲', hint: '锁甲职业外观' },
  { value: 'plate', label: '板甲', hint: '板甲职业外观' },
  { value: 'cosmetic', label: '装饰外观', hint: '通用装饰或节日外观' },
]
const transmogSourceOptions = [
  { value: 'collection', label: '收藏外观', hint: '已有收藏中组合' },
  { value: 'dungeon', label: '副本', hint: '副本或团队副本掉落' },
  { value: 'pvp', label: 'PVP', hint: '荣誉、征服或评级奖励' },
  { value: 'trading-post', label: '商栈', hint: '商栈轮换奖励' },
  { value: 'vendor', label: '商人', hint: '商人购买或兑换' },
  { value: 'quest', label: '任务', hint: '任务线奖励' },
]
const slotOptions = [
  { value: 'head', label: '头部' },
  { value: 'shoulder', label: '肩部' },
  { value: 'back', label: '背部' },
  { value: 'chest', label: '胸甲' },
  { value: 'shirt', label: '衬衣' },
  { value: 'tabard', label: '战袍' },
  { value: 'wrist', label: '护腕' },
  { value: 'hands', label: '手套' },
  { value: 'waist', label: '腰带' },
  { value: 'legs', label: '腿部' },
  { value: 'feet', label: '脚部' },
  { value: 'main_hand', label: '主手' },
  { value: 'off_hand', label: '副手' },
]
const slotRoleOptions = [
  { value: 'unused', label: '不使用', hint: '这一槽位不纳入方案' },
  { value: 'required', label: '必选', hint: '核心部件' },
  { value: 'optional', label: '可选', hint: '可按角色调整' },
  { value: 'variant', label: '替代', hint: '同风格替代件' },
]
const visitStatusOptions = [
  { value: 'friend_only', label: '加好友后参观', hint: '需要先添加好友或联系作者' },
  { value: 'closed', label: '不可参观', hint: '仅展示，不开放参观' },
]
const copyStatusOptions = [
  { value: 'copyable', label: '可复制导入', hint: '公开住宅分享代码' },
  { value: 'reference_only', label: '仅供参考', hint: '展示设计但不开放复制' },
  { value: 'private', label: '暂不公开', hint: '隐藏代码内容' },
]
const spaceTypeOptions = [
  { value: 'indoor', label: '室内', hint: '房间、酒馆、工坊等' },
  { value: 'outdoor', label: '室外', hint: '庭院、营地、街区等' },
  { value: 'indoor_outdoor', label: '室内外', hint: '室内和室外连续空间' },
]
const mediaTypeOptions = [
  { value: 'image', label: '图片', hint: '静态截图' },
  { value: 'gif', label: 'GIF', hint: '动态效果' },
  { value: 'video', label: '视频', hint: '视频链接' },
  { value: 'embed', label: '嵌入', hint: '外部嵌入内容' },
]

const isEdit = computed(() => Boolean(route.params.id))
const isGuideType = computed(() => form.type !== 'home_showcase')
const localDraftKey = computed(() => `rpdb-editor-draft:${route.params.id || 'new'}`)
const coverPreviewURL = computed(() => resolveRPDBMediaURL(form.cover_image))
const previewMedia = computed(() => form.media?.find(item => item.type === 'image' || item.type === 'gif'))
const previewImageURL = computed(() => resolveRPDBMediaURL(previewMedia.value?.url))
const openTransmogSlots = ref<Set<string>>(new Set())
const openItemReferences = ref<Set<number>>(new Set())
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
  if (form.type === 'home_showcase') return '家宅资料'
  if (form.type === 'transmog') return '幻化资料'
  return '道具资料'
})
const typeFormDescription = computed(() => {
  if (form.type === 'home_showcase') return '填写参观状态、空间类型和可导入的家宅分享代码。'
  if (form.type === 'transmog') return '填写护甲类型、阵营限制、部件清单和外观获取方式。'
  return '填写道具来源、绑定状态、阵营限制和实际 RP 用途。'
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
  } else if (!itemTypeOptions.some(option => option.value === form.references![0].external_type)) {
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
  const nextSlot = slotOptions.find(option => !usedSlots.has(option.value)) || slotOptions[0]
  form.transmog_slots!.push({ slot: nextSlot.value, role: 'required', name: '', description: '', source: '', wowhead_url: '', variant: '', note: '', sort_order: form.transmog_slots!.length + 1 })
}

function ensureTransmogSlots() {
  const currentSlots = new Map((form.transmog_slots || []).map(slot => [slot.slot, slot]))
  form.transmog_slots = slotOptions.map((option, index) => ({
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
      title: waypoint.label || `路线点 ${form.guide_steps!.length + 1}`,
      body: '',
      zone: waypoint.zone,
      map_id: waypoint.map_id,
      x: waypoint.x,
      y: waypoint.y,
      prerequisite: '',
    })
  }
  if (!waypoints.length) {
    toast.error('未识别到可用的 TomTom /way 坐标')
    return
  }
  tomtomDraft.value = ''
  if (rejected.length) {
    toast.warning(`已导入 ${waypoints.length} 个路线点，跳过 ${rejected.length} 行`)
  } else {
    toast.success(`已按顺序导入 ${waypoints.length} 个路线点`)
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
    toast.success('封面已上传')
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

function itemTypeLabel(value?: string) {
  return itemTypeOptions.find(option => option.value === value)?.label || '物品'
}

function appendPreviewImage(url: string) {
  const media = form.media || (form.media = [])
  if (media.some(item => item.url === url)) return
  const previewCount = media.filter(item => item.type === 'image' || item.type === 'gif').length
  media.push({
    type: 'image',
    url,
    thumbnail_url: url,
    caption: `效果预览 ${previewCount + 1}`,
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
      if (!url) throw new Error('上传失败，未返回图片地址')
      appendPreviewImage(url)
      uploaded++
    } catch (error) {
      toast.error(`${file.name}: ${(error as Error).message}`)
    }
  }
  input.value = ''
  if (uploaded) toast.success(`已添加 ${uploaded} 张预览图`)
}

async function uploadFurnitureIcon(event: Event, reference: RPDBReference) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploadingFurnitureIcon.value = reference
  try {
    const result = await uploadImage(file) as { url?: string; data?: { url?: string } }
    const url = result.url || result.data?.url || ''
    if (!url) throw new Error('上传失败，未返回图片地址')
    reference.icon = url
    toast.success('家具图标已上传')
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
    is_public: status === 'published',
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
    toast.success('家宅代码已导入')
  }
  reader.onerror = () => toast.error('读取家宅代码失败')
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
  if (form.type === 'home_showcase') {
    ensureHomeFurniture()
    return
  }
  ensurePrimaryReference()
  if (form.type === 'transmog') ensureTransmogSlots()
}

async function save(status: 'draft' | 'published') {
  if (!form.title.trim()) {
    titleTouched.value = true
    toast.error('请先填写作品标题')
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
    toast.success(status === 'draft' ? '草稿已自动保存' : '发布成功')
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
  await loadStyleTags()
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
        <span>RP 数据库发布台</span>
        <h1>{{ isEdit ? '修订玩家作品' : '发布一份玩家作品' }}</h1>
        <p>按创作流程完善资料，未完成的内容会自动保存在本地。</p>
      </div>
      <div class="heading-actions">
        <span v-if="lastSaved" class="saved-status"><i class="ri-checkbox-circle-fill"></i>已保存 {{ lastSaved }}</span>
        <button type="button" aria-label="关闭编辑器" title="关闭编辑器" @click="router.push('/rpdb')"><i class="ri-close-line"></i></button>
      </div>
    </header>

    <section id="section-basics" class="editor-upper section-anchor" data-testid="editor-upper">
      <div class="metadata-panel">
        <div class="panel-heading">
          <div><span>01</span><h2>基础资料</h2></div>
          <small><b class="required-mark" aria-hidden="true">*</b> 为必填项 · {{ form.status === 'published' ? '待审核' : '草稿' }}</small>
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
            <span>作品标题 <b class="required-mark" aria-hidden="true">*</b></span>
            <input
              id="rpdb-title"
              v-model="form.title"
              maxlength="256"
              required
              :aria-invalid="titleHasError"
              aria-describedby="rpdb-title-help"
              placeholder="例如：月光灯笼的巡夜用法"
              @blur="titleTouched = true"
            >
            <small id="rpdb-title-help" :class="{ 'field-error': titleHasError }">
              {{ titleHasError ? '请填写作品标题后再发布' : '建议写清物品、主题或空间特色' }}
            </small>
          </label>
          <label class="span-2">
            <span>一句话摘要 <em class="optional-label">选填</em></span>
            <textarea v-model="form.summary" maxlength="512" placeholder="让浏览者快速理解实际效果和获取价值"></textarea>
          </label>
          <label v-if="form.type !== 'home_showcase'">
            <span>获取状态</span>
            <RPDBSelect v-model="form.availability_status" :options="availabilityOptions" />
          </label>
          <div class="style-picker span-2">
            <span>RP 风格标签 <em class="optional-label">选填</em></span>
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
              <span v-if="!selectedStyleTags.length" class="topic-empty">还没有选择风格话题</span>
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
                {{ showAllStyleTags ? '收起标签' : `展开更多 ${candidateStyleTags.length - 8} 个` }}
              </button>
            </div>
            <label class="topic-input" data-testid="rpdb-topic-custom">
              <input
                v-model="topicDraft"
                data-testid="rpdb-topic-input"
                placeholder="# 输入自定义 RP 风格话题"
                @keydown.enter.prevent="addTopicFromInput"
              >
            </label>
            <small v-if="!rpStyleTags.length">等待后端预设标签加载完成后可保存选择。</small>
          </div>
        </div>

        <div id="section-details" class="panel-heading section-anchor">
          <div><span>02</span><h2>{{ typeFormTitle }}</h2></div>
          <small>{{ typeFormDescription }}</small>
        </div>

        <section v-if="form.type === 'item_showcase' && primaryItemReference" class="type-form-panel item-type-form" data-testid="item-editor-fields">
          <div class="metadata-grid">
            <label><span>物品名称</span><input v-model="primaryItemReference.name" placeholder="例如：月光灯笼"></label>
            <label><span>物品类型</span><RPDBSelect v-model="primaryItemReference.external_type" :options="itemTypeOptions" /></label>
            <label class="span-2"><span>物品描述</span><textarea v-model="primaryItemReference.description" placeholder="描述外观、效果或 RP 使用方式"></textarea></label>
            <label class="span-2"><span>物品来源</span><input v-model="primaryItemReference.acquisition_method" placeholder="例如：任务奖励、商人购买或公会活动产出"></label>
            <label><span>阵营</span><RPDBSelect v-model="form.faction" :options="factionOptions" /></label>
            <label><span>是否绑定</span><RPDBSelect v-model="form.bind_type" :options="bindOptions" /></label>
          </div>
        </section>

        <section v-else-if="form.type === 'transmog' && primaryTransmogReference" class="type-form-panel transmog-type-form" data-testid="transmog-editor-fields">
          <div class="metadata-grid">
            <label><span>护甲类型</span><RPDBSelect v-model="form.armor_type" :options="armorTypeOptions" /></label>
            <label><span>阵营</span><RPDBSelect v-model="form.faction" :options="factionOptions" /></label>
            <label><span>外观主题</span><input v-model="primaryTransmogReference.name" placeholder="银白骑士 / 暗月旅人 / 港务军官"></label>
            <label><span>主体来源</span><RPDBSelect v-model="primaryTransmogReference.source" :options="transmogSourceOptions" /></label>
            <label><span>套装链接</span><input v-model="primaryTransmogReference.url" placeholder="Wowhead dressing room 或截图链接"></label>
            <label><span>获取状态</span><RPDBSelect v-model="form.availability_status" :options="availabilityOptions" /></label>
          </div>
          <div class="slot-helper-panel">
            <i class="ri-shirt-line"></i>
            <span><b>幻化部位</b><small>右侧内容清单按固定槽位填写，未使用的槽位保持“不使用”。</small></span>
          </div>
        </section>

        <section v-else class="type-form-panel home-type-form" data-testid="home-editor-fields">
          <div class="home-grid">
            <label><span>参观状态</span><RPDBSelect v-model="homeDetails.visit_status" :options="visitStatusOptions" /></label>
            <label><span>代码状态</span><RPDBSelect v-model="homeDetails.copy_status" :options="copyStatusOptions" /></label>
            <label><span>空间类型</span><RPDBSelect v-model="homeDetails.space_type" :options="spaceTypeOptions" /></label>
            <div class="home-code-row span-2" data-testid="home-code-upload-row">
              <label>
                <span>住宅分享代码</span>
                <textarea v-model="homeDetails.share_code" data-testid="home-share-code-input" placeholder="粘贴家宅导入代码，或上传 .txt / .json / .lua 文件自动填入"></textarea>
              </label>
              <label class="home-code-upload">
                <input type="file" accept=".txt,.json,.lua" data-testid="home-code-file-input" @change="importHomeShareCode">
                <span><i class="ri-file-upload-line"></i><b>上传家宅代码</b><small>.txt / .json / .lua</small></span>
              </label>
            </div>
            <label class="span-2"><span>参观说明</span><textarea v-model="homeDetails.visit_notes" placeholder="开放时间、访问方式、战网昵称或拍摄须知"></textarea></label>
          </div>
        </section>

        <label v-if="isEdit" class="change-summary">
          <span>修订说明</span>
          <textarea v-model="form.change_summary" placeholder="说明这次修改了什么"></textarea>
        </label>
      </div>
    </section>

    <section id="section-media" class="media-strip section-anchor" data-testid="rpdb-media-strip">
      <div class="media-strip__heading">
        <span>03 · 选填</span>
        <h2>图片展示</h2>
        <p>封面用于列表与首屏，预览图用于展示实际效果。两者都可以稍后补充。</p>
      </div>

      <div class="media-upload-card">
        <div class="panel-heading">
          <div><span>封面图</span><h2>列表主视觉</h2></div>
          <small>推荐 16:9，JPG 或 PNG</small>
        </div>
        <div class="media-upload media-upload--cover" data-testid="cover-upload">
          <input id="rpdb-cover-upload" type="file" accept="image/*" @change="uploadCover">
          <img v-if="coverPreviewURL" :src="coverPreviewURL" alt="作品封面预览">
          <span v-else>
            <i class="ri-upload-cloud-2-line"></i>
            <b>上传封面图</b>
            <small>可不填，发布后会根据标题自动生成默认封面</small>
          </span>
          <label class="media-upload__action" for="rpdb-cover-upload">
            <i class="ri-upload-cloud-2-line"></i>
            {{ coverPreviewURL ? '更换封面图' : '自定义封面图' }}
          </label>
          <button
            v-if="coverPreviewURL"
            type="button"
            class="media-remove"
            data-testid="cover-remove"
            aria-label="移除封面图"
            @click="removeCover"
          >
            <i class="ri-close-line"></i>
          </button>
        </div>
      </div>

      <div class="media-upload-card">
        <div class="panel-heading">
          <div><span>预览图</span><h2>效果展示</h2></div>
          <small>支持多选，推荐 1 至 6 张</small>
        </div>
        <label class="media-upload" data-testid="preview-upload">
          <input type="file" accept="image/*" multiple @change="uploadPreview">
          <img v-if="previewImageURL" :src="previewImageURL" alt="作品预览图">
          <span v-else>
            <i class="ri-image-add-line"></i>
            <b>上传预览图</b>
            <small>展示实际使用、成套幻化或家宅全景</small>
          </span>
          <em v-if="previewImageURL">继续添加预览图</em>
        </label>
      </div>

      <div v-if="previewImageURL" class="preview-gallery-panel" data-testid="preview-gallery">
        <div class="panel-heading">
          <div><span>相册预览</span><h2>预览图相册</h2></div>
          <small>点击图片可查看大图</small>
        </div>
        <RPDBMediaGallery
          :cover="form.cover_image"
          :media="form.media"
          :title="form.title || '作品预览'"
          @open-image="openImageViewer"
        />
      </div>
    </section>

    <section id="section-content" class="editor-lower section-anchor" data-testid="editor-lower">
      <main class="writing-workspace">
        <div class="writing-heading">
          <div>
            <span>04 · 帖子编辑</span>
            <h2>正文与获取攻略</h2>
            <p>先说明实际体验，再补充可执行的获取路线和内容清单。</p>
          </div>
          <div class="outline-chips">
            <span>正文</span>
            <span v-if="form.type === 'item_showcase'">道具获取攻略</span>
            <span v-else-if="form.type === 'transmog'">幻化部件攻略</span>
            <span v-else>家宅展示说明</span>
            <span>{{ form.type === 'home_showcase' ? '分享代码' : '版本说明' }}</span>
          </div>
        </div>

        <div class="rich-editor-shell">
          <TiptapEditor
            ref="editorRef"
            :model-value="form.content || ''"
            data-testid="rpdb-rich-editor"
            placeholder="从实际效果、适用角色和剧情场景开始写作。可以插入图片、链接和分段标题。"
            @update:model-value="form.content = $event"
          >
            <template #toolbar>
              <button
                type="button"
                class="toolbar-slot toolbar-slot--featured"
                :class="{ active: quickJumpOpen }"
                title="内部链接"
                aria-label="内部链接"
                data-testid="rpdb-internal-link-button"
                @mousedown.prevent
                @click="toggleQuickJump"
              >
                <i class="ri-links-line"></i>
                <span>内部链接</span>
              </button>
            </template>
          </TiptapEditor>
        </div>

        <section v-if="isGuideType" class="guide-editor">
          <div class="section-heading">
            <div>
              <span>获取攻略</span>
              <h2>{{ form.type === 'transmog' ? '部件获取攻略' : '获取攻略' }}</h2>
              <p>攻略属于当前作品，可填写文字路线、前置条件、区域和 TomTom 坐标。</p>
            </div>
            <button type="button" @click="addStep"><i class="ri-add-line"></i>添加攻略步骤</button>
          </div>
          <div class="tomtom-import" data-testid="tomtom-import-panel">
            <div class="tomtom-import__mark">
              <i class="ri-route-line"></i>
              <span><b>TomTom 多点路线</b><small>/ttpaste 兼容格式</small></span>
            </div>
            <label>
              <span>批量坐标</span>
              <textarea
                v-model="tomtomDraft"
                data-testid="tomtom-import-input"
                placeholder="/way #47 73.80 44.50 夜色镇集合&#10;/way #47 68.20 51.40 林地入口"
              ></textarea>
              <small>支持 /way、/tway、#地图ID 和区域名称；多行顺序会成为攻略步骤顺序。</small>
            </label>
            <button type="button" data-testid="tomtom-import-button" @click="importTomTomSteps"><i class="ri-map-pin-add-line"></i>导入路线</button>
          </div>
          <div v-if="form.guide_steps?.length" class="guide-step-list">
            <article v-for="(step, index) in form.guide_steps" :key="`${step.sort_order}-${index}`">
              <div class="step-number">{{ index + 1 }}</div>
              <div class="step-main">
                <div class="step-grid">
                <label><span>步骤名称</span><input v-model="step.title" placeholder="例如：前往守夜营地"></label>
                <label><span>区域</span><input v-model="step.zone" placeholder="暮色森林"></label>
              </div>
              <p v-if="step.title" class="step-title-preview">{{ step.title }}</p>
              <label><span>步骤说明</span><textarea v-model="step.body" placeholder="说明路线、目标、掉落方式或注意事项"></textarea></label>
                <label><span>前置条件</span><input v-model="step.prerequisite" placeholder="任务、声望、职业或其他前置"></label>
              </div>
              <div class="step-coordinate">
                <label><span>地图 ID</span><input v-model="step.map_id" placeholder="47"></label>
                <div>
                  <label><span>X</span><input v-model.number="step.x" type="number" min="0" max="100"></label>
                  <label><span>Y</span><input v-model.number="step.y" type="number" min="0" max="100"></label>
                </div>
                <code v-if="editorTomTomCommand(step, index)">{{ editorTomTomCommand(step, index) }}</code>
                <button type="button" class="remove" @click="removeStep(index)"><i class="ri-delete-bin-line"></i>删除步骤</button>
              </div>
            </article>
          </div>
          <div v-else class="empty-guide">
            <i class="ri-route-line"></i>
            <p>尚未填写攻略。添加步骤后，公开页面会生成获取路线和 TomTom 操作。</p>
          </div>
          <div class="guide-bottom-actions">
            <button type="button" data-testid="add-guide-step-bottom" @click="addStep">
              <i class="ri-add-line"></i>
              添加攻略步骤
            </button>
          </div>
        </section>

        <section v-else class="home-editor">
          <div class="section-heading">
            <div>
              <span>家宅分享</span>
              <h2>家宅展示补充</h2>
              <p>家宅分享不显示获取攻略，重点补充参观说明和可导入代码。</p>
            </div>
          </div>
        </section>
      </main>

      <aside class="content-inspector">
        <details open data-testid="rpdb-content-checklist">
          <summary><span><i class="ri-list-check-3"></i>内容清单</span><b>{{ form.type === 'transmog' ? form.transmog_slots?.filter(slot => slot.role !== 'unused').length || 0 : form.references?.length || 0 }}</b></summary>
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
                  <b>{{ item.name || `物品 ${index + 1}` }}</b>
                  <span><small>{{ itemTypeLabel(item.external_type) }}</small><i class="ri-arrow-down-s-line"></i></span>
                </summary>
                <div class="slot-row__body">
                  <label><span>物品名称</span><input v-model="item.name" placeholder="例如：月光灯笼"></label>
                  <label><span>物品描述</span><textarea v-model="item.description" placeholder="描述外观、效果或 RP 使用方式"></textarea></label>
                  <label><span>物品类型</span><RPDBSelect v-model="item.external_type" :options="itemTypeOptions" /></label>
                  <label><span>物品来源</span><textarea v-model="item.acquisition_method" placeholder="任务、商人、专业或活动来源"></textarea></label>
                  <button type="button" class="remove" @click="form.references!.splice(index, 1)"><i class="ri-delete-bin-line"></i>移除</button>
                </div>
              </details>
              <button type="button" class="add-button" @click="addItemReference"><i class="ri-add-line"></i>添加道具</button>
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
                  <span><small>{{ slot.role === 'unused' ? '未使用' : '已填写' }}</small><i class="ri-arrow-down-s-line"></i></span>
                </summary>
                <div class="slot-row__body">
                  <label>
                    <span>槽位状态</span>
                    <RPDBSelect v-model="slot.role" :options="slotRoleOptions" />
                  </label>
                  <label><span>部件名称</span><input v-model="slot.name" placeholder="例如：海潮卫士头盔"></label>
                  <label><span>部件介绍</span><textarea v-model="slot.description" placeholder="外观特点、替代搭配或 RP 用途"></textarea></label>
                  <label><span>获取来源</span><input v-model="slot.source" placeholder="副本 / 商人 / 任务 / 成就 / 商栈"></label>
                  <details class="slot-extra-options" data-testid="transmog-slot-more-options">
                    <summary>
                      <span><i class="ri-equalizer-line"></i>其他选项</span>
                      <small>Wowhead / 替代件</small>
                    </summary>
                    <div class="slot-extra-grid">
                      <label><span>Wowhead 地址</span><input v-model="slot.wowhead_url" placeholder="https://www.wowhead.com/item=..."></label>
                      <label><span>替代件</span><input v-model="slot.variant" placeholder="可替代部件名称或链接"></label>
                    </div>
                  </details>
                </div>
              </details>
            </div>

            <div v-else class="checklist-stack" data-testid="home-content-checklist">
              <article v-for="(item, index) in form.references" :key="index" class="compact-card">
                <label><span>家具名称</span><input v-model="item.name" placeholder="例如：木质长桌"></label>
                <div class="furniture-icon-field" data-testid="furniture-icon-field">
                  <span>图标</span>
                  <div class="furniture-icon-control">
                    <span class="furniture-icon-preview" :class="{ empty: !item.icon }">
                      <img v-if="item.icon" :src="resolveRPDBMediaURL(item.icon)" :alt="`${item.name || '家具'}图标`">
                      <i v-else class="ri-image-line"></i>
                    </span>
                    <input v-model="item.icon" data-testid="furniture-icon-url" placeholder="粘贴图片链接">
                    <label class="furniture-icon-upload" :class="{ disabled: uploadingFurnitureIcon === item }" title="上传图标">
                      <input
                        type="file"
                        accept="image/*"
                        data-testid="furniture-icon-upload"
                        :disabled="uploadingFurnitureIcon === item"
                        @change="uploadFurnitureIcon($event, item)"
                      >
                      <i :class="uploadingFurnitureIcon === item ? 'ri-loader-4-line spin' : 'ri-upload-2-line'"></i>
                      <span>{{ uploadingFurnitureIcon === item ? '上传中' : '上传' }}</span>
                    </label>
                    <button v-if="item.icon" type="button" class="furniture-icon-clear" title="清除图标" aria-label="清除图标" @click="item.icon = ''"><i class="ri-close-line"></i></button>
                  </div>
                </div>
                <label><span>Wowhead 地址</span><input v-model="item.url" placeholder="https://www.wowhead.com/item=..."></label>
                <label><span>获取途径</span><textarea v-model="item.acquisition_method" placeholder="商人、任务、成就、专业或活动来源"></textarea></label>
                <label><span>描述</span><textarea v-model="item.description" placeholder="用途、摆放效果或组合建议"></textarea></label>
                <button type="button" class="remove" @click="form.references!.splice(index, 1)"><i class="ri-delete-bin-line"></i>移除</button>
              </article>
              <button type="button" class="add-button" @click="addFurnitureReference"><i class="ri-add-line"></i>添加家具</button>
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
      <button type="button" title="内部预览" @click="preview"><i class="ri-eye-line"></i><span>内部预览</span></button>
      <button type="button" class="primary" data-testid="publish-work" :disabled="saving" @click="save('published')"><i class="ri-send-plane-2-line"></i><span>{{ saving ? '发布中' : '发布' }}</span></button>
    </div>

    <PostQuickJump v-model="quickJumpOpen" :on-insert="handleQuickInsert" />
    <ImageViewer v-model="showImageViewer" :images="viewerImages" :start-index="viewerStartIndex" />
  </div>
</template>

<style scoped>
.editor-page{max-width:1380px;margin:auto;color:var(--color-text-main)}
.minimal-editor-shell{--rpdb-surface:color-mix(in srgb,var(--color-panel-bg) 88%,#fff 12%);--rpdb-muted:color-mix(in srgb,var(--color-card-bg) 82%,#fff 18%);--rpdb-line:color-mix(in srgb,var(--color-border) 72%,transparent);--rpdb-soft:color-mix(in srgb,var(--color-accent) 8%,transparent)}
.editor-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:18px;margin-bottom:18px;padding-bottom:16px;border-bottom:1px solid var(--rpdb-line)}
.editor-heading>div:first-child>span,.writing-heading>div>span,.section-heading>div>span{color:var(--color-accent);font-size:11px;font-weight:800;letter-spacing:.06em}
.editor-heading h1{margin:6px 0 4px;color:var(--color-text-main);font:700 30px/1.2 system-ui,'Microsoft YaHei',sans-serif}
.editor-heading p{margin:0;color:var(--color-text-secondary)}
.heading-actions{display:flex;align-items:center;gap:10px}
.heading-actions>button{display:grid;width:36px;height:36px;place-items:center;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--rpdb-surface);color:var(--color-text-main)}
.saved-status{display:inline-flex;align-items:center;gap:6px;color:#4d7a4c;font-size:12px}
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
  .style-options button.selected{background:var(--tag-color);border-color:var(--tag-color);color:#fff}
  .style-options button.disabled{opacity:.55;cursor:not-allowed}
label{display:grid;gap:6px;color:var(--color-text-main);font-weight:700}
label>span{font-size:12px}
input,textarea,select{width:100%;box-sizing:border-box;padding:10px 11px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg);color:var(--color-text-main);font:inherit}
textarea{min-height:70px;resize:vertical}
.check-list{display:grid;gap:4px;margin-bottom:12px}
.check-list>div{display:grid;grid-template-columns:20px 1fr auto;gap:7px;align-items:center;padding:8px 0;color:var(--color-text-secondary)}
.check-list i,.check-list b{color:#b65a4f}
.check-list b{font-size:10px}
.check-list .done i,.check-list .done b{color:#4d7a4c}
.change-summary textarea{min-height:58px}
.publish-actions{display:grid;gap:8px;margin-top:12px}
.publish-actions button,.section-heading button,.add-button,.remove{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-height:36px;padding:0 12px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg);color:var(--color-text-main)}
.publish-actions .primary{border-color:var(--color-accent);background:var(--color-accent);color:#fff}
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
.tomtom-import>button{display:inline-flex;min-height:38px;align-items:center;justify-content:center;gap:6px;padding:0 13px;border:1px solid var(--color-accent);border-radius:9px;background:var(--color-accent);color:#fff;font-weight:800;white-space:nowrap}
.guide-step-list{display:grid;gap:10px}
.guide-step-list article{display:grid;grid-template-columns:34px minmax(0,1fr) 200px;gap:12px;padding:13px;border:1px solid var(--rpdb-line);border-radius:12px;background:var(--rpdb-muted)}
.step-number{display:grid;width:30px;height:30px;place-items:center;border-radius:50%;background:var(--rpdb-soft);color:var(--color-accent);font-weight:800}
.step-main{display:grid;gap:8px}
.step-grid{display:grid;grid-template-columns:1fr 180px;gap:8px}
.step-coordinate{display:flex;flex-direction:column;gap:7px;padding-left:12px;border-left:1px solid var(--rpdb-line)}
.step-coordinate>div{display:grid;grid-template-columns:1fr 1fr;gap:7px}
.step-coordinate code{overflow-wrap:anywhere;padding:8px;border-radius:9px;background:var(--rpdb-soft);color:var(--color-accent);font:10px/1.5 Consolas,monospace}
.remove{min-height:30px;color:#b65a4f}
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
@media(max-width:480px){.editor-heading h1{font-size:26px}.media-upload{height:165px}.step-coordinate>div{grid-template-columns:1fr 1fr}}

/* Focused authoring workflow */
.editor-page{
  max-width:1380px;
  padding:0 12px 96px;
}
.minimal-editor-shell{
  --rpdb-surface:color-mix(in srgb,var(--color-panel-bg) 94%,#fff 6%);
  --rpdb-muted:color-mix(in srgb,var(--color-card-bg) 88%,#fff 12%);
  --rpdb-line:color-mix(in srgb,var(--color-border) 64%,transparent);
  --rpdb-soft:color-mix(in srgb,var(--color-accent) 9%,transparent);
  --rpdb-focus:color-mix(in srgb,var(--color-accent) 68%,var(--rpdb-line));
  --rpdb-success:#3f7d52;
  --rpdb-danger:#b34f45;
  --rpdb-shadow:0 8px 24px rgba(37,27,20,.055);
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
  border:1px solid color-mix(in srgb,#4d7a4c 20%,transparent);
  border-radius:999px;
  background:color-mix(in srgb,#4d7a4c 8%,transparent);
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
  outline:2px solid color-mix(in srgb,var(--color-accent) 72%,#fff);
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
    linear-gradient(180deg,color-mix(in srgb,var(--color-panel-bg) 96%,#fff 4%),color-mix(in srgb,var(--color-card-bg) 82%,#fff 18%));
  box-shadow:inset 0 1px 0 rgba(255,255,255,.35);
}
select:hover{
  border-color:color-mix(in srgb,var(--color-accent) 38%,var(--rpdb-line));
  background:
    linear-gradient(45deg,transparent 50%,var(--color-accent) 50%) calc(100% - 18px) 50%/5px 5px no-repeat,
    linear-gradient(135deg,var(--color-accent) 50%,transparent 50%) calc(100% - 13px) 50%/5px 5px no-repeat,
    linear-gradient(180deg,color-mix(in srgb,var(--color-panel-bg) 98%,#fff 2%),color-mix(in srgb,var(--color-accent) 6%,var(--color-card-bg)));
}
select:focus{
  outline:none;
  border-color:var(--rpdb-focus);
  box-shadow:0 0 0 3px color-mix(in srgb,var(--color-accent) 14%,transparent),inset 0 1px 0 rgba(255,255,255,.35);
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
  box-shadow:0 12px 32px rgba(37,27,20,.16);
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
  color:#3f7d52;
}
.floating-submit-toolbar .auto-save-state.error{
  color:#b65a4f;
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
  color:#fff;
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
