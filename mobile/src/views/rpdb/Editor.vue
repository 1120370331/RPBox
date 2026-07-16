<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  createRPDBDraft,
  deleteRPDBDraft,
  getRPDBDraft,
  getRPDBWork,
  listRPDBDrafts,
  publishRPDBDraft,
  resolveRPDBMediaUrl,
  updateRPDBDraft,
  type RPDBTransmogSlot,
  type RPDBVisibility,
  type RPDBWork,
  type RPDBWorkPayload,
  type RPDBWorkType,
} from '@/api/rpdb'
import { uploadImage } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import MobileRichEditor from '@/components/MobileRichEditor.vue'
import CachedImage from '@/components/CachedImage.vue'
import { getRPDBTypeIcon, getRPDBTypeLabel } from '@/utils/rpdb'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()

const loading = ref(true)
const saving = ref(false)
const publishing = ref(false)
const deleting = ref(false)
const currentDraftId = ref<number | null>(null)
const editingWorkId = ref<number | null>(null)
const activeStep = ref<'basic' | 'content' | 'structure' | 'publish'>('basic')
const previewMode = ref(false)
const coverUploading = ref(false)
const mediaUploading = ref(false)
const coverInput = ref<HTMLInputElement | null>(null)
const mediaInput = ref<HTMLInputElement | null>(null)
const tags = ref<Tag[]>([])
const guilds = ref<Guild[]>([])
const selectedTagIds = ref<number[]>([])
const lastSaved = ref('')
const saveState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const loaded = ref(false)
const deleteDialogOpen = ref(false)
const tomtomImport = ref('')
let autosaveTimer: ReturnType<typeof setTimeout> | null = null

function emptyForm(): RPDBWorkPayload {
  return {
    type: 'item_showcase',
    title: '',
    summary: '',
    content: '',
    content_type: 'html',
    cover_image: '',
    rp_use_cases: '',
    effect_description: '',
    restrictions: {},
    extra: {},
    availability_status: 'available',
    bind_type: 'no',
    faction: 'neutral',
    armor_type: '',
    status: 'draft',
    is_public: true,
    visibility: 'public',
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
      sort_order: 1,
    }],
    media: [],
    transmog_slots: [],
    guide_steps: [],
    tag_ids: [],
  }
}

const form = reactive<RPDBWorkPayload>(emptyForm())
const extra = reactive({
  share_code: '',
  visit_notes: '',
  copy_status: 'copyable',
  visit_status: 'friend_only',
  space_type: 'indoor_outdoor',
})
const stepItems = [
  { id: 'basic' as const, label: '基础资料', icon: 'ri-file-info-line' },
  { id: 'content' as const, label: '正文展示', icon: 'ri-quill-pen-line' },
  { id: 'structure' as const, label: '攻略数据', icon: 'ri-node-tree' },
  { id: 'publish' as const, label: '预览发布', icon: 'ri-send-plane-line' },
]
const slotDefinitions = [
  ['head', '头部'], ['shoulder', '肩部'], ['back', '背部'], ['chest', '胸甲'],
  ['shirt', '衬衣'], ['tabard', '战袍'], ['wrist', '护腕'], ['hands', '手套'],
  ['waist', '腰带'], ['legs', '腿部'], ['feet', '脚部'], ['main_hand', '主手'], ['off_hand', '副手'],
] as const
const isEditing = computed(() => Boolean(editingWorkId.value))
const pageTitle = computed(() => isEditing.value ? '编辑 RP 数据库作品' : '发布 RP 数据库作品')
const coverUrl = computed(() => resolveRPDBMediaUrl(form.cover_image))
const completionChecks = computed(() => [
  { label: '作品标题', done: Boolean(form.title.trim()) },
  { label: '作品摘要', done: Boolean(form.summary?.trim()) },
  { label: '封面或展示媒体', done: Boolean(form.cover_image || form.media?.length) },
  { label: '作品正文', done: Boolean(form.content?.trim()) },
  { label: form.type === 'home_showcase' ? '家宅资料' : '获取攻略', done: form.type === 'home_showcase' ? Boolean(extra.visit_notes || extra.share_code) : Boolean(form.guide_steps?.length) },
])
const completionPercent = computed(() => Math.round((completionChecks.value.filter(item => item.done).length / completionChecks.value.length) * 100))

function parseObject(value: unknown) {
  if (value && typeof value === 'object' && !Array.isArray(value)) return value as Record<string, unknown>
  if (typeof value !== 'string') return {}
  try {
    const parsed = JSON.parse(value)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {}
  } catch {
    return {}
  }
}

function applyPayload(payload: Partial<RPDBWorkPayload>) {
  const next = { ...emptyForm(), ...payload }
  Object.assign(form, next)
  form.references = [...(payload.references || next.references || [])]
  form.media = [...(payload.media || [])]
  form.transmog_slots = [...(payload.transmog_slots || [])]
  form.guide_steps = [...(payload.guide_steps || [])]
  form.guild_ids = [...(payload.guild_ids || [])]
  selectedTagIds.value = [...(payload.tag_ids || [])]
  const details = parseObject(payload.extra)
  extra.share_code = String(details.share_code || '')
  extra.visit_notes = String(details.visit_notes || '')
  extra.copy_status = String(details.copy_status || 'copyable')
  extra.visit_status = String(details.visit_status || 'friend_only')
  extra.space_type = String(details.space_type || 'indoor_outdoor')
  normalizeTypeFields()
}

function workToPayload(work: RPDBWork): RPDBWorkPayload {
  return {
    type: work.type,
    title: work.title,
    summary: work.summary,
    content: work.content,
    content_type: 'html',
    cover_image: work.cover_image,
    rp_use_cases: work.rp_use_cases,
    effect_description: work.effect_description,
    restrictions: parseObject(work.restrictions),
    extra: parseObject(work.extra),
    availability_status: work.availability_status,
    bind_type: work.bind_type,
    faction: work.faction,
    armor_type: work.armor_type,
    status: 'draft',
    is_public: work.is_public,
    visibility: work.visibility,
    guild_id: work.guild_id,
    guild_ids: work.guild_ids || [],
    references: work.references || [],
    media: work.media || [],
    transmog_slots: work.transmog_slots || [],
    guide_steps: work.guide_steps || [],
    tag_ids: (work.tags || []).map(tag => tag.id),
  }
}

function buildPayload(): RPDBWorkPayload {
  return {
    ...form,
    title: form.title.trim(),
    summary: form.summary?.trim(),
    content: form.content?.trim(),
    rp_use_cases: form.rp_use_cases?.trim(),
    effect_description: form.effect_description?.trim(),
    extra: {
      share_code: extra.share_code.trim(),
      visit_notes: extra.visit_notes.trim(),
      copy_status: extra.copy_status,
      visit_status: extra.visit_status,
      space_type: extra.space_type,
    },
    status: 'draft',
    is_public: form.visibility === 'public',
    guild_id: form.visibility === 'guild' ? form.guild_ids?.[0] : undefined,
    guild_ids: form.visibility === 'guild' ? form.guild_ids : [],
    references: (form.references || []).filter(item => item.name?.trim() || item.external_id?.trim()),
    media: (form.media || []).filter(item => item.url?.trim()).map((item, index) => ({ ...item, sort_order: index + 1 })),
    transmog_slots: form.type === 'transmog' ? (form.transmog_slots || []).filter(item => item.role !== 'unused') : [],
    guide_steps: form.type === 'home_showcase' ? [] : (form.guide_steps || []).map((item, index) => ({ ...item, sort_order: index + 1 })),
    tag_ids: selectedTagIds.value,
  }
}

function normalizeTypeFields() {
  if (form.type === 'transmog') ensureSlots()
  if (form.type === 'home_showcase') {
    form.guide_steps = []
    form.transmog_slots = []
    if (!form.references?.length) addReference('furniture')
  } else if (!form.references?.length) {
    addReference(form.type === 'transmog' ? 'transmog' : 'item')
  }
}

function selectType(type: RPDBWorkType) {
  if (isEditing.value && type !== form.type) return
  form.type = type
  normalizeTypeFields()
}

function ensureSlots() {
  const current = new Map((form.transmog_slots || []).map(item => [item.slot, item]))
  form.transmog_slots = slotDefinitions.map(([slot], index) => ({
    slot,
    role: current.get(slot)?.role || 'unused',
    name: current.get(slot)?.name || '',
    description: current.get(slot)?.description || '',
    source: current.get(slot)?.source || '',
    wowhead_url: current.get(slot)?.wowhead_url || '',
    variant: current.get(slot)?.variant || '',
    note: current.get(slot)?.note || '',
    sort_order: index + 1,
  }))
}

function slotLabel(slot: string) {
  return slotDefinitions.find(item => item[0] === slot)?.[1] || slot
}

function toggleSlot(slot: RPDBTransmogSlot) {
  slot.role = slot.role === 'unused' ? 'required' : 'unused'
}

function addReference(type = form.type === 'home_showcase' ? 'furniture' : form.type === 'transmog' ? 'transmog' : 'item') {
  const items = form.references || (form.references = [])
  items.push({
    external_type: type,
    external_id: '',
    name: '',
    icon: '',
    description: '',
    acquisition_method: '',
    source: '',
    url: '',
    is_primary: items.length === 0,
    sort_order: items.length + 1,
  })
}

function removeReference(index: number) {
  form.references?.splice(index, 1)
}

function addMediaUrl() {
  const items = form.media || (form.media = [])
  items.push({ type: 'image', url: '', caption: '', sort_order: items.length + 1 })
}

function removeMedia(index: number) {
  form.media?.splice(index, 1)
}

function addGuideStep() {
  const items = form.guide_steps || (form.guide_steps = [])
  items.push({ sort_order: items.length + 1, title: '', body: '', zone: '', map_id: '', x: undefined, y: undefined, prerequisite: '' })
}

function removeGuideStep(index: number) {
  form.guide_steps?.splice(index, 1)
}

function importTomTom() {
  const lines = tomtomImport.value.split(/\r?\n/).map(line => line.trim()).filter(Boolean)
  let added = 0
  for (const line of lines) {
    const match = line.match(/^\/way\s+(?:(\d+)\s+)?(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s*(.*)$/i)
    if (!match) continue
    const items = form.guide_steps || (form.guide_steps = [])
    items.push({
      sort_order: items.length + 1,
      title: match[4] || `路线点 ${items.length + 1}`,
      body: '',
      map_id: match[1] || '',
      x: Number(match[2]),
      y: Number(match[3]),
      label: match[4] || '',
      zone: '',
      prerequisite: '',
    })
    added++
  }
  if (!added) {
    toast.warning('没有识别到有效的 /way 坐标')
    return
  }
  tomtomImport.value = ''
  toast.success(`已导入 ${added} 个路线点`)
}

function toggleTag(id: number) {
  selectedTagIds.value = selectedTagIds.value.includes(id)
    ? selectedTagIds.value.filter(item => item !== id)
    : [...selectedTagIds.value, id]
}

function toggleGuild(id: number) {
  const next = new Set(form.guild_ids || [])
  if (next.has(id)) next.delete(id)
  else next.add(id)
  form.guild_ids = [...next]
}

async function uploadCover(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  coverUploading.value = true
  try {
    const result = await uploadImage(file)
    form.cover_image = result.url
    toast.success('封面已上传')
  } catch (error) {
    toast.error((error as Error).message || '封面上传失败')
  } finally {
    input.value = ''
    coverUploading.value = false
  }
}

async function uploadMedia(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return
  mediaUploading.value = true
  let added = 0
  try {
    for (const file of files) {
      const result = await uploadImage(file)
      const items = form.media || (form.media = [])
      items.push({ type: file.type === 'image/gif' ? 'gif' : 'image', url: result.url, caption: '', sort_order: items.length + 1 })
      added++
    }
    toast.success(`已添加 ${added} 张展示图`)
  } catch (error) {
    toast.error((error as Error).message || '展示图上传失败')
  } finally {
    input.value = ''
    mediaUploading.value = false
  }
}

async function ensureDraft() {
  const payload = buildPayload()
  if (currentDraftId.value) {
    const result = await updateRPDBDraft(currentDraftId.value, payload)
    return result.draft
  }
  const result = await createRPDBDraft(payload, editingWorkId.value || undefined)
  currentDraftId.value = result.draft.id
  return result.draft
}

async function saveDraft(showToast = true) {
  if (saving.value || !loaded.value) return false
  saving.value = true
  saveState.value = 'saving'
  try {
    await ensureDraft()
    lastSaved.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    saveState.value = 'saved'
    localStorage.removeItem('rpdb-mobile-local-draft')
    if (showToast) toast.success('草稿已保存')
    return true
  } catch (error) {
    saveState.value = 'error'
    localStorage.setItem('rpdb-mobile-local-draft', JSON.stringify(buildPayload()))
    if (showToast) toast.error((error as Error).message || '草稿保存失败')
    return false
  } finally {
    saving.value = false
  }
}

function scheduleAutosave() {
  if (!loaded.value) return
  localStorage.setItem('rpdb-mobile-local-draft', JSON.stringify(buildPayload()))
  if (autosaveTimer) clearTimeout(autosaveTimer)
  autosaveTimer = setTimeout(() => {
    void saveDraft(false)
  }, 1400)
}

async function publish() {
  if (!form.title.trim()) {
    activeStep.value = 'basic'
    toast.warning('请先填写作品标题')
    return
  }
  publishing.value = true
  try {
    const saved = await ensureDraft()
    const result = await publishRPDBDraft(saved.id)
    localStorage.removeItem('rpdb-mobile-local-draft')
    toast.success(result.revision ? '修改已提交审核' : '作品已提交发布')
    await router.replace({ name: 'rpdb-detail', params: { id: result.work.id } })
  } catch (error) {
    toast.error((error as Error).message || '发布失败')
  } finally {
    publishing.value = false
  }
}

async function deleteDraft() {
  if (!currentDraftId.value || deleting.value) return
  deleting.value = true
  try {
    await deleteRPDBDraft(currentDraftId.value)
    localStorage.removeItem('rpdb-mobile-local-draft')
    toast.success('草稿已删除')
    await router.replace({ name: 'rpdb-my-uploads' })
  } catch (error) {
    toast.error((error as Error).message || '删除失败')
  } finally {
    deleting.value = false
    deleteDialogOpen.value = false
  }
}

async function loadInitial() {
  loading.value = true
  try {
    const [tagResult, guildResult] = await Promise.all([
      getPresetTags('rpdb').catch(() => ({ tags: [] })),
      listGuilds().catch(() => ({ guilds: [] })),
    ])
    tags.value = tagResult.tags || []
    guilds.value = guildResult.guilds || []

    const draftId = Number(route.params.draftId)
    const workId = Number(route.params.id)
    if (Number.isFinite(draftId) && draftId > 0) {
      const result = await getRPDBDraft(draftId)
      currentDraftId.value = result.draft.id
      editingWorkId.value = result.draft.work_id || null
      applyPayload(result.draft.payload)
    } else if (Number.isFinite(workId) && workId > 0) {
      editingWorkId.value = workId
      const [detail, draftResult] = await Promise.all([getRPDBWork(workId), listRPDBDrafts({ work_id: workId })])
      const existing = draftResult.drafts?.[0]
      if (existing) {
        currentDraftId.value = existing.id
        applyPayload(existing.payload)
      } else {
        applyPayload(workToPayload(detail.work))
      }
    } else {
      const local = localStorage.getItem('rpdb-mobile-local-draft')
      if (local) {
        try {
          applyPayload(JSON.parse(local))
          toast.info('已恢复本地未保存内容')
        } catch {
          applyPayload(emptyForm())
        }
      } else {
        applyPayload(emptyForm())
      }
    }
  } catch (error) {
    toast.error((error as Error).message || '编辑器加载失败')
    await router.replace({ name: 'rpdb-my-uploads' })
  } finally {
    loading.value = false
    loaded.value = true
  }
}

watch(form, scheduleAutosave, { deep: true })
watch(extra, scheduleAutosave, { deep: true })
watch(selectedTagIds, scheduleAutosave, { deep: true })

onMounted(loadInitial)
onBeforeUnmount(() => {
  if (autosaveTimer) clearTimeout(autosaveTimer)
})
</script>

<template>
  <div class="sub-page editor-page">
    <header class="editor-header">
      <button type="button" aria-label="返回" @click="router.push({ name: 'rpdb-my-uploads' })"><i class="ri-arrow-left-line" /></button>
      <div><small>{{ isEditing ? '修改作品' : '创建作品' }}</small><h1>{{ pageTitle }}</h1></div>
      <button type="button" :class="{ active: previewMode }" aria-label="切换预览" @click="previewMode = !previewMode"><i :class="previewMode ? 'ri-edit-line' : 'ri-eye-line'" /></button>
    </header>

    <div v-if="loading" class="loading-state"><i class="ri-loader-4-line spin" />正在准备编辑器</div>

    <template v-else>
      <nav v-if="!previewMode" class="step-tabs" aria-label="编辑步骤">
        <button v-for="step in stepItems" :key="step.id" type="button" :class="{ active: activeStep === step.id }" @click="activeStep = step.id">
          <i :class="step.icon" /><span>{{ step.label }}</span>
        </button>
      </nav>

      <main v-if="previewMode" class="preview-page">
        <section class="preview-cover">
          <CachedImage v-if="coverUrl" :src="coverUrl" :alt="form.title" />
          <i v-else :class="getRPDBTypeIcon(form.type)" />
        </section>
        <section class="preview-copy">
          <span>{{ getRPDBTypeLabel(form.type) }}</span>
          <h2>{{ form.title || '未命名作品' }}</h2>
          <p>{{ form.summary || '作品摘要将在这里显示。' }}</p>
        </section>
        <section class="preview-content">
          <h3>{{ form.type === 'home_showcase' ? '空间故事' : '作品介绍' }}</h3>
          <p v-if="form.effect_description">{{ form.effect_description }}</p>
          <div v-if="form.content" class="rich-preview" v-html="form.content" />
          <p v-else class="empty">正文内容将在这里显示。</p>
        </section>
      </main>

      <main v-else class="editor-content">
        <section v-if="activeStep === 'basic'" class="editor-section">
          <header><span>01 · 基础资料</span><h2>先决定这是一份什么档案</h2><p>类型会影响后续需要填写的攻略和结构化字段。</p></header>
          <div class="type-selector">
            <button v-for="type in (['item_showcase','transmog','home_showcase'] as RPDBWorkType[])" :key="type" type="button" :class="{ active: form.type === type }" :disabled="isEditing && form.type !== type" @click="selectType(type)">
              <i :class="getRPDBTypeIcon(type)" /><span><b>{{ getRPDBTypeLabel(type) }}</b><small>{{ type === 'item_showcase' ? '特效物品、玩具与装备' : type === 'transmog' ? '整套或散件幻化方案' : '住宅展示与参观资料' }}</small></span>
            </button>
          </div>
          <label class="field required"><span>作品标题</span><input v-model="form.title" maxlength="160" placeholder="填写玩家会搜索的准确名称"></label>
          <label class="field"><span>作品摘要</span><textarea v-model="form.summary" rows="3" maxlength="500" placeholder="用一两句话说明特色、用途与获取难点" /></label>
          <div class="media-grid">
            <div class="field">
              <span>主封面</span>
              <button type="button" class="media-upload" @click="coverInput?.click()">
                <CachedImage v-if="coverUrl" :src="coverUrl" :alt="form.title" />
                <span v-else><i class="ri-image-add-line" /><b>上传主封面</b><small>建议横向 16:10 图片</small></span>
              </button>
              <button v-if="form.cover_image" type="button" class="text-button danger" @click="form.cover_image = ''">移除封面</button>
              <input ref="coverInput" type="file" accept="image/*" hidden @change="uploadCover">
            </div>
            <div class="field">
              <span>展示媒体</span>
              <button type="button" class="media-upload compact" @click="mediaInput?.click()"><span><i class="ri-gallery-upload-line" /><b>{{ mediaUploading ? '上传中' : '添加截图或 GIF' }}</b><small>支持一次选择多张</small></span></button>
              <button type="button" class="text-button" @click="addMediaUrl"><i class="ri-link" />添加外部媒体</button>
              <input ref="mediaInput" type="file" accept="image/*" multiple hidden @change="uploadMedia">
            </div>
          </div>
          <div v-if="form.media?.length" class="media-list">
            <article v-for="(media, index) in form.media" :key="index">
              <span class="media-preview"><CachedImage v-if="media.url" :src="resolveRPDBMediaUrl(media.url)" :alt="media.caption" /><i v-else class="ri-image-line" /></span>
              <div><select v-model="media.type"><option value="image">图片</option><option value="gif">GIF</option><option value="video">视频</option><option value="embed">嵌入链接</option></select><input v-model="media.url" placeholder="媒体 URL"><input v-model="media.caption" placeholder="图片说明"></div>
              <button type="button" aria-label="移除媒体" @click="removeMedia(index)"><i class="ri-delete-bin-line" /></button>
            </article>
          </div>
        </section>

        <section v-if="activeStep === 'content'" class="editor-section">
          <header><span>02 · 正文展示</span><h2>把实际效果讲清楚</h2><p>正文支持图片、标题、链接和排版，适合完整展示作品。</p></header>
          <label class="field"><span>效果说明</span><textarea v-model="form.effect_description" rows="3" placeholder="实际使用时会发生什么，画面与声音效果如何" /></label>
          <label class="field"><span>RP 适用场景</span><textarea v-model="form.rp_use_cases" rows="3" placeholder="适合哪些角色、剧情和互动场景" /></label>
          <div class="field"><span>完整正文</span><MobileRichEditor v-model="form.content!" placeholder="编写作品背景、实测记录、展示图片与注意事项..." /></div>
        </section>

        <section v-if="activeStep === 'structure'" class="editor-section">
          <header><span>03 · 攻略数据</span><h2>补充可检索、可执行的信息</h2><p>这些字段会直接出现在详情页、筛选器与收集助手中。</p></header>
          <div class="metadata-grid">
            <label class="field"><span>{{ form.type === 'home_showcase' ? '开放状态' : '获取状态' }}</span><select v-model="form.availability_status"><option value="available">可获取</option><option value="limited">限时获取</option><option value="removed">已绝版</option><option value="unknown">未知</option><option v-if="form.type === 'home_showcase'" value="friend_only">好友可参观</option><option v-if="form.type === 'home_showcase'" value="closed">暂不开放</option></select></label>
            <label v-if="form.type !== 'home_showcase'" class="field"><span>绑定方式</span><select v-model="form.bind_type"><option value="no">不绑定</option><option value="yes">绑定</option><option value="account">战网绑定</option><option value="pickup">拾取绑定</option><option value="use">使用绑定</option></select></label>
            <label class="field"><span>阵营</span><select v-model="form.faction"><option value="neutral">中立</option><option value="alliance">联盟</option><option value="horde">部落</option></select></label>
            <label v-if="form.type === 'transmog'" class="field"><span>护甲类型</span><select v-model="form.armor_type"><option value="">不限</option><option value="cloth">布甲</option><option value="leather">皮甲</option><option value="mail">锁甲</option><option value="plate">板甲</option><option value="cosmetic">装饰品</option></select></label>
          </div>
          <div v-if="tags.length && form.type !== 'home_showcase'" class="field"><span>扮演风格标签</span><div class="tag-selector"><button v-for="tag in tags" :key="tag.id" type="button" :class="{ active: selectedTagIds.includes(tag.id) }" @click="toggleTag(tag.id)">{{ tag.name }}</button></div></div>

          <template v-if="form.type === 'home_showcase'">
            <div class="subsection"><header><h3>家宅与参观资料</h3><button type="button" @click="addReference('furniture')"><i class="ri-add-line" />添加家具</button></header>
              <div class="metadata-grid"><label class="field"><span>复制状态</span><select v-model="extra.copy_status"><option value="copyable">可复制</option><option value="reference_only">仅供参考</option><option value="private">不公开代码</option></select></label><label class="field"><span>空间类型</span><select v-model="extra.space_type"><option value="indoor">室内</option><option value="outdoor">室外</option><option value="indoor_outdoor">室内与室外</option></select></label></div>
              <label class="field"><span>住宅分享代码</span><textarea v-model="extra.share_code" rows="4" placeholder="可选，粘贴游戏内住宅分享代码" /></label>
              <label class="field"><span>参观说明</span><textarea v-model="extra.visit_notes" rows="4" placeholder="服务器、战网昵称、开放时间与参观注意事项" /></label>
            </div>
          </template>

          <template v-if="form.type === 'transmog'">
            <div class="subsection"><header><h3>幻化分享代码</h3></header><label class="field"><span>导入代码</span><textarea v-model="extra.share_code" rows="4" placeholder="可选，粘贴游戏内幻化分享代码" /></label></div>
            <div class="subsection"><header><h3>幻化部件</h3><small>启用实际使用的部位，再填写名称与来源</small></header>
              <div class="slot-list">
                <details v-for="slot in form.transmog_slots" :key="slot.slot" :open="slot.role !== 'unused'">
                  <summary><button type="button" :class="{ active: slot.role !== 'unused' }" @click.prevent="toggleSlot(slot)"><i :class="slot.role !== 'unused' ? 'ri-checkbox-circle-fill' : 'ri-checkbox-blank-circle-line'" />{{ slotLabel(slot.slot) }}</button><i class="ri-arrow-down-s-line" /></summary>
                  <div v-if="slot.role !== 'unused'" class="slot-fields"><label class="field"><span>用途</span><select v-model="slot.role"><option value="required">必选</option><option value="optional">可选</option><option value="variant">替代方案</option></select></label><label class="field"><span>部件名称</span><input v-model="slot.name"></label><label class="field"><span>来源</span><input v-model="slot.source"></label><label class="field"><span>介绍</span><textarea v-model="slot.description" rows="2" /></label><label class="field"><span>Wowhead 链接</span><input v-model="slot.wowhead_url" type="url"></label></div>
                </details>
              </div>
            </div>
          </template>

          <div class="subsection"><header><h3>{{ form.type === 'home_showcase' ? '家具与引用对象' : '关联物品' }}</h3><button type="button" @click="addReference()"><i class="ri-add-line" />添加</button></header>
            <div class="reference-list">
              <details v-for="(reference, index) in form.references" :key="index">
                <summary><span><i class="ri-archive-2-line" />{{ reference.name || `引用对象 ${index + 1}` }}</span><button type="button" @click.prevent="removeReference(index)"><i class="ri-delete-bin-line" /></button></summary>
                <div class="reference-fields"><div class="metadata-grid"><label class="field"><span>类型</span><select v-model="reference.external_type"><option value="item">物品</option><option value="equipment">装备</option><option value="toy">玩具</option><option value="quest_item">任务道具</option><option value="transmog">幻化</option><option value="furniture">家具</option></select></label><label class="field"><span>外部 ID</span><input v-model="reference.external_id"></label></div><label class="field"><span>名称</span><input v-model="reference.name"></label><label class="field"><span>图标 URL</span><input v-model="reference.icon"></label><label class="field"><span>来源</span><input v-model="reference.source"></label><label class="field"><span>获取方式</span><input v-model="reference.acquisition_method"></label><label class="field"><span>介绍</span><textarea v-model="reference.description" rows="2" /></label><label class="field"><span>外部链接</span><input v-model="reference.url" type="url"></label></div>
              </details>
            </div>
          </div>

          <div v-if="form.type !== 'home_showcase'" class="subsection"><header><h3>获取攻略</h3><button type="button" @click="addGuideStep"><i class="ri-add-line" />添加步骤</button></header>
            <div class="tomtom-import"><textarea v-model="tomtomImport" rows="3" placeholder="/way 47 72.4 46.8 夜色镇入口&#10;每行一个坐标" /><button type="button" @click="importTomTom"><i class="ri-route-line" />批量导入</button></div>
            <div class="guide-editor-list">
              <article v-for="(step, index) in form.guide_steps" :key="index"><span>{{ index + 1 }}</span><div><label class="field"><span>步骤标题</span><input v-model="step.title"></label><label class="field"><span>说明</span><textarea v-model="step.body" rows="3" /></label><div class="coordinate-grid"><label class="field"><span>区域</span><input v-model="step.zone"></label><label class="field"><span>地图 ID</span><input v-model="step.map_id"></label><label class="field"><span>X</span><input v-model.number="step.x" type="number" step="0.01"></label><label class="field"><span>Y</span><input v-model.number="step.y" type="number" step="0.01"></label></div><label class="field"><span>前置条件</span><input v-model="step.prerequisite"></label></div><button type="button" @click="removeGuideStep(index)"><i class="ri-delete-bin-line" /></button></article>
            </div>
          </div>
        </section>

        <section v-if="activeStep === 'publish'" class="editor-section">
          <header><span>04 · 预览发布</span><h2>确认完成度与可见范围</h2><p>发布后，普通用户的修改会进入审核；原线上版本在审核完成前保持不变。</p></header>
          <section class="quality-panel">
            <div class="quality-score"><strong>{{ completionPercent }}%</strong><span>资料完成度</span><div><i :style="{ width: `${completionPercent}%` }" /></div></div>
            <ul><li v-for="item in completionChecks" :key="item.label" :class="{ done: item.done }"><i :class="item.done ? 'ri-checkbox-circle-fill' : 'ri-checkbox-blank-circle-line'" />{{ item.label }}</li></ul>
          </section>
          <div class="visibility-panel">
            <h3>谁可以看到这份作品</h3>
            <div class="visibility-options">
              <button v-for="option in ([['public','公开','所有 RPBox 用户'],['guild','公会可见','仅选中的公会成员'],['private','仅自己','保留为私人档案']] as [RPDBVisibility,string,string][])" :key="option[0]" type="button" :class="{ active: form.visibility === option[0] }" @click="form.visibility = option[0]"><i :class="option[0] === 'public' ? 'ri-global-line' : option[0] === 'guild' ? 'ri-shield-user-line' : 'ri-lock-line'" /><span><b>{{ option[1] }}</b><small>{{ option[2] }}</small></span></button>
            </div>
            <div v-if="form.visibility === 'guild'" class="guild-selector"><button v-for="guild in guilds" :key="guild.id" type="button" :class="{ active: form.guild_ids?.includes(guild.id) }" @click="toggleGuild(guild.id)"><i :class="form.guild_ids?.includes(guild.id) ? 'ri-checkbox-circle-fill' : 'ri-checkbox-blank-circle-line'" />{{ guild.name }}</button><p v-if="!guilds.length">你还没有加入公会。</p></div>
          </div>
          <button type="button" class="large-preview" @click="previewMode = true"><i class="ri-eye-line" />打开移动端预览</button>
          <button v-if="currentDraftId" type="button" class="delete-draft" @click="deleteDialogOpen = true"><i class="ri-delete-bin-line" />删除当前草稿</button>
        </section>
      </main>

      <footer class="submit-bar">
        <span :class="saveState"><i :class="saveState === 'saving' ? 'ri-loader-4-line spin' : saveState === 'error' ? 'ri-error-warning-line' : 'ri-cloud-line'" />{{ saveState === 'saving' ? '保存中' : saveState === 'saved' ? `已保存 ${lastSaved}` : saveState === 'error' ? '等待重试' : '自动保存' }}</span>
        <button type="button" :disabled="saving || publishing" @click="saveDraft()"><i class="ri-save-line" />保存草稿</button>
        <button type="button" class="primary" :disabled="saving || publishing" @click="publish"><i class="ri-send-plane-fill" />{{ publishing ? '发布中' : '提交发布' }}</button>
      </footer>
    </template>

    <div v-if="deleteDialogOpen" class="dialog-mask"><section class="dialog" role="dialog" aria-modal="true"><h2>删除草稿</h2><p>草稿删除后无法恢复，正式作品不会受到影响。</p><footer><button type="button" @click="deleteDialogOpen = false">取消</button><button type="button" class="danger" :disabled="deleting" @click="deleteDraft">确认删除</button></footer></section></div>
  </div>
</template>

<style scoped>
.editor-page{min-height:var(--app-height,100dvh);padding-bottom:calc(86px + var(--safe-bottom,0px))}.editor-header{position:sticky;top:0;z-index:60;display:grid;grid-template-columns:44px minmax(0,1fr) 44px;align-items:center;gap:8px;padding:calc(var(--safe-top,0px) + 7px) var(--page-gutter) 8px;border-bottom:1px solid var(--color-border);background:rgba(238,217,196,.95);backdrop-filter:blur(12px)}.editor-header>button{display:grid;width:44px;height:44px;place-items:center;border:1px solid rgba(75,54,33,.12);border-radius:8px;background:rgba(255,255,255,.65);color:var(--color-text-main);font-size:18px}.editor-header>button.active{background:var(--color-primary);color:#fff}.editor-header>div{min-width:0;text-align:center}.editor-header small{display:block;color:var(--color-accent);font-size:8px;font-weight:800;letter-spacing:.06em}.editor-header h1{overflow:hidden;margin-top:2px;font-size:14px;text-overflow:ellipsis;white-space:nowrap}
.step-tabs{position:sticky;top:60px;z-index:50;display:grid;grid-template-columns:repeat(4,1fr);gap:3px;padding:7px var(--page-gutter);border-bottom:1px solid var(--color-border);background:rgba(253,251,249,.96);backdrop-filter:blur(10px)}.step-tabs button{display:flex;min-width:0;min-height:44px;align-items:center;justify-content:center;flex-direction:column;gap:2px;border:0;border-radius:6px;background:transparent;color:var(--color-text-secondary)}.step-tabs button.active{background:var(--color-primary-light);color:var(--color-secondary)}.step-tabs i{font-size:16px}.step-tabs span{font-size:8px;font-weight:700}
.loading-state{display:grid;min-height:75vh;place-items:center;align-content:center;gap:10px;color:var(--color-text-secondary)}.loading-state i{color:var(--color-secondary);font-size:34px}.editor-content,.preview-page{padding:12px var(--page-gutter)}.editor-section{display:grid;gap:14px}.editor-section>header{padding:4px 2px 10px;border-bottom:1px solid var(--color-border)}.editor-section>header>span{color:var(--color-accent);font-size:9px;font-weight:800;letter-spacing:.06em}.editor-section>header h2{margin-top:4px;font-family:Georgia,'Microsoft YaHei',serif;font-size:21px}.editor-section>header p{margin-top:5px;color:var(--color-text-secondary);font-size:11px;line-height:1.55}
.type-selector{display:grid;grid-template-columns:1fr;gap:7px}.type-selector button{display:grid;grid-template-columns:38px minmax(0,1fr);gap:9px;align-items:center;min-height:62px;padding:9px 11px;border:1px solid var(--color-border);border-radius:7px;background:var(--color-panel-bg);color:var(--color-text-main);text-align:left}.type-selector button>i{display:grid;width:38px;height:38px;place-items:center;border-radius:7px;background:var(--tag-bg);color:var(--color-secondary);font-size:19px}.type-selector button span{display:flex;min-width:0;flex-direction:column}.type-selector small{margin-top:3px;color:var(--color-text-secondary);font-size:9px}.type-selector button.active{border-color:var(--color-accent);box-shadow:inset 3px 0 0 var(--color-accent)}.type-selector button.active>i{background:var(--color-primary);color:#fff}
.field{display:grid;gap:6px}.field>span{color:var(--color-text-secondary);font-size:10px;font-weight:700}.field.required>span::after{margin-left:3px;color:#b6382d;content:'*'}.field input,.field textarea,.field select,.media-list input,.media-list select{width:100%;min-width:0;min-height:40px;padding:9px 10px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);color:var(--color-text-main);font:inherit;font-size:12px}.field textarea{resize:vertical;line-height:1.55}.metadata-grid{display:grid;grid-template-columns:1fr 1fr;gap:10px}
.media-grid{display:grid;grid-template-columns:1fr 1fr;gap:9px}.media-upload{position:relative;display:block;width:100%;aspect-ratio:4/3;overflow:hidden;padding:0;border:1px dashed var(--color-accent);border-radius:7px;background:var(--color-card-bg);color:var(--color-secondary)}.media-upload.compact{aspect-ratio:4/3}.media-upload>span{display:grid;height:100%;place-items:center;align-content:center;gap:4px}.media-upload>span>i{font-size:28px}.media-upload b{font-size:11px}.media-upload small{color:var(--color-text-secondary);font-size:8px}.text-button{display:inline-flex;min-height:34px;align-items:center;justify-content:center;gap:4px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-secondary);font-size:10px}.text-button.danger{color:#b6382d}.media-list{display:grid;gap:8px}.media-list article{display:grid;grid-template-columns:56px minmax(0,1fr) 34px;gap:7px;padding:8px;border:1px solid var(--color-border);border-radius:7px;background:var(--color-panel-bg)}.media-preview{display:grid;width:56px;height:56px;place-items:center;overflow:hidden;border-radius:5px;background:var(--color-primary-light);color:var(--color-secondary)}.media-list article>div{display:grid;gap:5px}.media-list article>button{width:34px;height:34px;border:0;background:transparent;color:#b6382d}
.tag-selector{display:flex;flex-wrap:wrap;gap:6px;max-height:190px;overflow:auto}.tag-selector button{min-height:30px;padding:0 9px;border:1px solid var(--color-border);border-radius:5px;background:var(--color-card-bg);color:var(--color-text-secondary);font-size:9px}.tag-selector button.active{border-color:var(--color-accent);background:var(--tag-bg);color:var(--color-secondary)}
.subsection{display:grid;gap:11px;padding-top:14px;border-top:1px solid var(--color-border)}.subsection>header{display:flex;align-items:center;justify-content:space-between;gap:9px}.subsection>header h3{font-size:15px}.subsection>header small{color:var(--color-text-secondary);font-size:9px}.subsection>header button{display:inline-flex;min-height:32px;align-items:center;gap:4px;padding:0 9px;border:1px solid var(--color-border);border-radius:5px;background:var(--color-card-bg);color:var(--color-secondary);font-size:9px}
.slot-list,.reference-list{display:grid;gap:7px}.slot-list details,.reference-list details{overflow:hidden;border:1px solid var(--color-border);border-radius:7px;background:var(--color-panel-bg)}.slot-list summary,.reference-list summary{display:flex;min-height:42px;align-items:center;justify-content:space-between;gap:8px;padding:0 10px;cursor:pointer;list-style:none}.slot-list summary::-webkit-details-marker,.reference-list summary::-webkit-details-marker{display:none}.slot-list summary button{display:inline-flex;min-height:34px;align-items:center;gap:6px;border:0;background:transparent;color:var(--color-text-secondary);font-size:11px}.slot-list summary button.active{color:var(--color-secondary);font-weight:800}.slot-fields,.reference-fields{display:grid;gap:8px;padding:9px;border-top:1px solid var(--color-border);background:rgba(75,54,33,.025)}.reference-list summary>span{display:inline-flex;min-width:0;align-items:center;gap:6px;overflow:hidden;font-size:11px;text-overflow:ellipsis;white-space:nowrap}.reference-list summary>button{width:32px;height:32px;border:0;background:transparent;color:#b6382d}
.tomtom-import{display:grid;gap:7px;padding:9px;border:1px solid var(--color-border);border-radius:7px;background:var(--color-panel-bg)}.tomtom-import textarea{width:100%;padding:9px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);font:10px/1.5 Consolas,monospace}.tomtom-import button{min-height:36px;border:0;border-radius:6px;background:var(--color-primary);color:#fff}.guide-editor-list{display:grid;gap:9px}.guide-editor-list article{display:grid;grid-template-columns:30px minmax(0,1fr) 32px;gap:7px;padding:9px;border:1px solid var(--color-border);border-radius:7px;background:var(--color-panel-bg)}.guide-editor-list article>span{display:grid;width:30px;height:30px;place-items:center;border-radius:5px;background:var(--color-primary);color:#edbf84;font-size:10px;font-weight:800}.guide-editor-list article>div{display:grid;gap:7px}.guide-editor-list article>button{width:32px;height:32px;border:0;background:transparent;color:#b6382d}.coordinate-grid{display:grid;grid-template-columns:1fr 1fr;gap:7px}
.quality-panel,.visibility-panel{display:grid;gap:13px;padding:13px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-panel-bg)}.quality-score{display:grid;grid-template-columns:1fr auto;gap:4px;align-items:end}.quality-score strong{color:var(--color-secondary);font:800 28px/1 Georgia,serif}.quality-score span{color:var(--color-text-secondary);font-size:10px}.quality-score>div{grid-column:1/-1;height:7px;overflow:hidden;border-radius:4px;background:var(--color-border)}.quality-score i{display:block;height:100%;background:var(--color-accent)}.quality-panel ul{display:grid;grid-template-columns:1fr 1fr;gap:7px;padding:0;list-style:none}.quality-panel li{display:flex;align-items:center;gap:5px;color:var(--color-text-secondary);font-size:10px}.quality-panel li.done{color:var(--color-success)}.visibility-panel h3{font-size:15px}.visibility-options{display:grid;gap:6px}.visibility-options button{display:grid;grid-template-columns:34px minmax(0,1fr);gap:8px;align-items:center;min-height:54px;padding:8px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);text-align:left}.visibility-options button>i{display:grid;width:34px;height:34px;place-items:center;border-radius:6px;background:var(--tag-bg);color:var(--color-secondary)}.visibility-options span{display:flex;flex-direction:column}.visibility-options small{margin-top:3px;color:var(--color-text-secondary);font-size:8px}.visibility-options button.active{border-color:var(--color-accent);background:rgba(184,115,51,.06)}.guild-selector{display:flex;flex-wrap:wrap;gap:6px}.guild-selector button{display:inline-flex;min-height:32px;align-items:center;gap:5px;padding:0 8px;border:1px solid var(--color-border);border-radius:5px;background:var(--color-card-bg);font-size:9px}.guild-selector button.active{border-color:var(--color-accent);color:var(--color-secondary)}.guild-selector p{color:var(--color-text-secondary);font-size:10px}.large-preview,.delete-draft{display:inline-flex;min-height:44px;align-items:center;justify-content:center;gap:6px;border-radius:7px;font-weight:800}.large-preview{border:0;background:var(--color-primary);color:#fff}.delete-draft{border:1px solid #b6382d;background:transparent;color:#b6382d}
.preview-page{display:grid;gap:0}.preview-cover{display:grid;width:100%;aspect-ratio:16/10;place-items:center;overflow:hidden;border-radius:8px 8px 0 0;background:var(--color-primary);color:#edbf84;font-size:42px}.preview-copy,.preview-content{padding:16px;border:1px solid var(--color-border);background:var(--color-panel-bg)}.preview-copy span{color:var(--color-accent);font-size:9px;font-weight:800}.preview-copy h2{margin:6px 0;font-family:Georgia,'Microsoft YaHei',serif;font-size:24px}.preview-copy p,.preview-content p{color:var(--color-text-secondary);font-size:13px;line-height:1.7}.preview-content{border-top:0;border-radius:0 0 8px 8px}.preview-content h3{margin-bottom:10px;font-size:18px}.rich-preview{margin-top:12px;font-size:14px;line-height:1.75}.rich-preview :deep(img){max-width:100%;height:auto}.empty{color:var(--color-text-secondary)}
.submit-bar{position:fixed;right:calc(var(--safe-right,0px) + 8px);bottom:calc(var(--safe-bottom,0px) + 7px);left:calc(var(--safe-left,0px) + 8px);z-index:100;display:grid;grid-template-columns:1fr 1fr;gap:6px;padding:7px;border:1px solid rgba(75,54,33,.13);border-radius:10px;background:rgba(255,255,255,.96);box-shadow:0 10px 28px rgba(44,24,16,.2);backdrop-filter:blur(14px)}.submit-bar>span{display:inline-flex;grid-column:1/-1;align-items:center;gap:5px;color:var(--color-text-secondary);font-size:8px}.submit-bar>span.saved{color:var(--color-success)}.submit-bar>span.error{color:#b6382d}.submit-bar button{display:inline-flex;min-height:40px;align-items:center;justify-content:center;gap:5px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-main);font-size:10px;font-weight:800}.submit-bar button.primary{border-color:var(--color-primary);background:var(--color-primary);color:#fff}
.dialog-mask{position:fixed;inset:0;z-index:2300;display:grid;place-items:center;padding:16px;background:rgba(44,24,16,.5)}.dialog{width:min(100%,360px);padding:16px;border-radius:8px;background:var(--color-panel-bg)}.dialog h2{font-size:18px}.dialog p{margin-top:8px;color:var(--color-text-secondary);font-size:12px;line-height:1.6}.dialog footer{display:flex;justify-content:flex-end;gap:7px;margin-top:16px}.dialog button{min-height:38px;padding:0 12px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg)}.dialog .danger{border-color:#b6382d;background:#b6382d;color:#fff}.spin{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:350px){.media-grid,.metadata-grid,.quality-panel ul{grid-template-columns:1fr}.step-tabs span{font-size:7px}}
</style>
