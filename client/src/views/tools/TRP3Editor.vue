<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { createCharacterCard, getCharacterCard, getCharacterCardTRP3Lua, listMyCharacterCards, saveCharacterCardToCloud, type CharacterCardSaveTarget, type CharacterCard, type CharacterCardSource, type CharacterCardSummary, type CharacterCardPersonalityTrait, type TRP3About, type TRP3AboutSection, type UpdateCharacterCardRequest } from '@/api/characterCard'
import { createCharacterCardDraft, createEmptyCharacterCardDraft } from '../../utils/characterCardDraft'
import { trp3TraitIcons } from '../../data/trp3TraitIcons'
import { normalizeTRP3About } from '../../utils/trp3Markup'
import { normalizeCharacterCardHexForCSS } from '../../utils/characterCardColor'
import { downloadTextFile } from '@/utils/downloadText'
import { useUserStore } from '@/stores/user'
import { useToastStore } from '@/stores/toast'
import { useDialog } from '@/composables/useDialog'
import CharacterSaveTargets from '../../components/tools/CharacterSaveTargets.vue'
import CharacterCardPicker from '../../components/tools/CharacterCardPicker.vue'
import type { CharacterCardChoice, LocalCharacterSource } from '../../utils/characterCardSelection'
import WowIcon from '../../components/WowIcon.vue'
import TRP3IconPicker from '../../components/tools/TRP3IconPicker.vue'
import TRP3MarkupEditor from '../../components/tools/TRP3MarkupEditor.vue'
import TRP3AboutPreview from '../../components/tools/TRP3AboutPreview.vue'

const route = useRoute(), router = useRouter(), user = useUserStore(), toast = useToastStore(), dialog = useDialog()
const { t } = useI18n()
const cards = ref<CharacterCardSummary[]>([]), card = ref<CharacterCard | null>(null)
const form = reactive(createEmptyCharacterCardDraft())
const about = ref<TRP3About>(normalizeTRP3About())
const loading = ref(false), saving = ref(false), creating = ref(false), listLoading = ref(false)
const error = ref(''), listError = ref(''), savedAt = ref(''), snapshot = ref('')
const tab = ref<'characteristics' | 'glance' | 'about'>('characteristics'), preview = ref(false)
const editorBody = ref<HTMLElement | null>(null)
const loadingBodyHeight = ref(0)
interface CreatedCardViewport { id: number; top: number; left: number; height: number }
let createdCardViewport: CreatedCardViewport | null = null
const pickerOpen = ref(false)
const saveTargets = ref<CharacterCardSaveTarget[]>([])
const targetPicker = ref<InstanceType<typeof CharacterSaveTargets> | null>(null)
const iconOpen = ref(false), selectedIcon = ref('')
let iconSetter: ((value: string) => void) | null = null
let loadToken = 0
const currentSnapshot = computed(() => JSON.stringify({ form, about: about.value }))
const dirty = computed(() => !!card.value && !loading.value && snapshot.value !== currentSnapshot.value)
const cardId = computed(() => Number(route.params.id) || 0)
const title = computed(() => [form.first_name, form.last_name].filter(Boolean).join(' ') || form.display_name || '未命名角色')
const selectedTabLabel = computed(() => ({ characteristics: '特征', glance: '第一印象', about: '关于' }[tab.value]))
const physicalFields = [
  ['race', '种族', 64], ['class', '职业', 64], ['age', '年龄', 64], ['residence', '居住地', 256],
  ['eye_color', '眼睛颜色', 64], ['birthplace', '出生地', 256], ['height', '身高', 64], ['weight', '体重', 64],
] as const
const template3Sections = [{ key: 'PH', title: '外貌描写' }, { key: 'PS', title: '性格特征' }, { key: 'HI', title: '历史' }] as const
const relationLabels = ['不显示', '单身', '恋爱中', '已婚', '离异', '丧偶']

async function loadList() {
  listLoading.value = true; listError.value = ''
  try { cards.value = (await listMyCharacterCards()).character_cards } catch (e) { listError.value = e instanceof Error ? e.message : '加载档案失败' } finally { listLoading.value = false }
}
async function loadCard() {
  const token = ++loadToken
  const viewport = createdCardViewport?.id === cardId.value ? createdCardViewport : null
  createdCardViewport = null
  loadingBodyHeight.value = viewport?.height || 0
  saveTargets.value = []; card.value = null; error.value = ''; savedAt.value = ''; preview.value = false
  if (!cardId.value) { loading.value = false; return }
  loading.value = true
  try {
    const loaded = await getCharacterCard(cardId.value)
    if (token !== loadToken) return
    if (loaded.user_id !== user.user?.id) throw new Error('只能编辑自己的 RPBox 人物卡。')
    let sourceAbout: unknown = loaded.trp3_about
    if (!sourceAbout && loaded.source_backup_id && loaded.source_profile_id) {
      const exported = await getCharacterCardTRP3Lua(loaded.id)
      const player = exported.profile.player as Record<string, unknown> | undefined
      sourceAbout = player?.about
    }
    if (token !== loadToken) return
    Object.assign(form, createCharacterCardDraft(loaded))
    about.value = normalizeTRP3About(sourceAbout, loaded.background_story)
    card.value = loaded
    snapshot.value = currentSnapshot.value
  } catch (e) { if (token === loadToken) error.value = e instanceof Error ? e.message : '读取人物卡失败' }
  finally {
    if (token === loadToken) {
      loading.value = false
      await nextTick()
      if (viewport && token === loadToken) {
        editorBody.value?.closest<HTMLElement>('.main-content')?.scrollTo({ top: viewport.top, left: viewport.left, behavior: 'instant' })
      }
      loadingBodyHeight.value = 0
    }
  }
}
watch(cardId, () => void loadCard(), { immediate: true })
onMounted(() => { void loadList(); window.addEventListener('beforeunload', beforeUnload) })
onBeforeUnmount(() => { loadToken++; window.removeEventListener('beforeunload', beforeUnload) })
function beforeUnload(event: BeforeUnloadEvent) { if (dirty.value) event.preventDefault() }
async function canLeave() {
  if (saving.value || creating.value) return false
  return !dirty.value || await dialog.confirm({ title: '尚未保存', message: '当前人物卡有未保存的修改。离开后这些修改将丢失。', confirmText: '放弃修改', cancelText: '继续编辑', type: 'warning' })
}
onBeforeRouteLeave(canLeave)
onBeforeRouteUpdate(async to => to.params.id === route.params.id || await canLeave())
function openPicker() { pickerOpen.value = true }
async function chooseCard(choice: CharacterCardChoice) {
  if (saving.value || creating.value) return
  if (choice.kind === 'archive') {
    if (choice.card.id === cardId.value) { pickerOpen.value = false; return }
    const failure = await router.push(`/tools/trp3/${choice.card.id}`)
    if (!failure) pickerOpen.value = false
  } else if (choice.kind === 'cloud') await create(choice.source)
  else await create(undefined, choice.source)
}
async function create(source?: CharacterCardSource, localSource?: LocalCharacterSource) {
  if (creating.value || saving.value || !(await canLeave())) return
  if (creating.value || saving.value) return
  const scroller = editorBody.value?.closest<HTMLElement>('.main-content')
  const viewport = { top: scroller?.scrollTop || 0, left: scroller?.scrollLeft || 0, height: editorBody.value?.offsetHeight || 0 }
  creating.value = true
  try {
    const localProfile = localSource ? JSON.parse(JSON.stringify(localSource.profile)) as Record<string, unknown> : undefined
    const localPlayer = localProfile?.player as Record<string, unknown> | undefined
    if (localPlayer?.about) localPlayer.about = normalizeTRP3About(localPlayer.about)
    const created = await createCharacterCard(localSource ? {
      source_type: 'local', source_game_flavor: localSource.game_flavor, source_account_id: localSource.account_id,
      source_profile_id: localSource.profile_id, local_profile: localProfile,
    } : source ? { source_type: 'backup', source_backup_id: source.backup_id, source_profile_id: source.profile_id } : { source_type: 'blank' })
    // The previous draft was explicitly discarded above; avoid a second prompt.
    snapshot.value = currentSnapshot.value
    pickerOpen.value = false
    await loadList()
    createdCardViewport = { ...viewport, id: created.id }
    creating.value = false
    const failure = await router.push({ path: `/tools/trp3/${created.id}`, state: { keepToolEditorScroll: true } })
    if (failure) createdCardViewport = null
  } catch (e) { toast.error(e instanceof Error ? e.message : '创建失败') } finally { creating.value = false }
}
function payload(): Partial<UpdateCharacterCardRequest> {
  return {
    first_name: form.first_name, last_name: form.last_name, display_name: form.display_name,
    title: form.title, full_title: form.full_title, race: form.race, class: form.class,
    age: form.age, height: form.height, weight: form.weight, eye_color: form.eye_color,
    eye_color_hex: form.eye_color_hex, birthplace: form.birthplace, residence: form.residence,
    relationship_status: form.relationship_status, icon: form.icon,
    class_color: form.class_color, name_color: form.class_color,
    additional_info: form.additional_info, personality_traits: form.personality_traits,
    impressions: form.impressions, trp3_about: about.value,
  }
}
async function save() {
  if (!card.value || saving.value) return false
  if (!saveTargets.value.length) { toast.warning('请选择至少一个保存子账号'); return false }
  saving.value = true
  try {
    const targetCount = saveTargets.value.length
    const saved = await saveCharacterCardToCloud(card.value.id, payload(), saveTargets.value)
    if (!saved.trp3_about) throw new Error('服务端尚未支持 TRP3 模板保存，请更新后端后重试。')
    form.name_color = form.class_color
    card.value = saved; snapshot.value = currentSnapshot.value
    savedAt.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    toast.success(`已保存人物卡及 ${targetCount} 个子账号的云端存档；本地文件未改动`)
    await targetPicker.value?.refresh(true)
    void loadList()
    return true
  } catch (e) { toast.error(e instanceof Error ? e.message : '保存失败，请重试'); return false } finally { saving.value = false }
}
async function exportLua() {
  if (!card.value || saving.value) return
  if (dirty.value && !(await save())) return
  try {
    const data = await getCharacterCardTRP3Lua(card.value.id)
    await downloadTextFile(`TRP3-${card.value.id}.lua`, data.lua)
  } catch (e) { toast.error(e instanceof Error ? e.message : '导出失败') }
}
function pickIcon(value: string | undefined, setter: (value: string) => void) { selectedIcon.value = value || ''; iconSetter = setter; iconOpen.value = true }
function applyIcon(value: string) { iconSetter?.(value) }
function addTrait() { form.personality_traits.push({ preset_id: 1, left_text: '', right_text: '', left_icon: '', right_icon: '', left_color: null, right_color: null, value: 10 }) }
function traitLabels(trait: CharacterCardPersonalityTrait) { return trait.preset_id ? t(`characterCards.editor.personalityPresets.${trait.preset_id}`).split(' — ') : [trait.left_text || '左侧倾向', trait.right_text || '右侧倾向'] }
function traitColor(value: CharacterCardPersonalityTrait['left_color']) { return value ? '#' + [value.r, value.g, value.b].map(channel => Math.round(channel * 255).toString(16).padStart(2, '0')).join('') : '#b8944e' }
function setTraitColor(trait: CharacterCardPersonalityTrait, side: 'left_color' | 'right_color', event: Event) { const value = (event.target as HTMLInputElement).value.slice(1); trait[side] = { r: parseInt(value.slice(0,2),16)/255, g: parseInt(value.slice(2,4),16)/255, b: parseInt(value.slice(4,6),16)/255 } }
function colorValue(value: string) { return normalizeCharacterCardHexForCSS(value || 'ffffff').slice(0, 7) }
function setColor(field: 'class_color' | 'eye_color_hex', event: Event) { form[field] = (event.target as HTMLInputElement).value.slice(1); if (field === 'class_color') form.name_color = form.class_color }
async function removeEntry(entries: unknown[], index: number) { if (await dialog.confirm({ title: '移除内容', message: '移除这一项？保存后会更新对应的人物卡档案。', confirmText: '移除', type: 'warning' })) entries.splice(index, 1) }
function moveSection(index: number, delta: number) { const target = index + delta; if (target < 0 || target >= about.value.T2.length) return; [about.value.T2[index], about.value.T2[target]] = [about.value.T2[target]!, about.value.T2[index]!] }
function addSection() { if (about.value.T2.length < 50) about.value.T2.push({ TX: '', IC: 'inv_misc_book_09' }) }
function sectionIcon(section: TRP3AboutSection) { pickIcon(section.IC, value => { section.IC = value }) }
</script>

<template>
  <main class="trp3-editor">
    <RouterLink class="trp3-back" to="/tools"><i class="ri-arrow-left-line" aria-hidden="true"></i> 工具箱</RouterLink>
    <header class="online-heading"><div><h1>在线人物卡编辑器</h1><p>Total RP 3 的角色档案工作台，直接连接你的 RPBox 人物卡。</p></div><RouterLink v-if="card" :to="`/character-cards/${card.id}`" class="archive-link">查看 RPBox 档案 <i class="ri-arrow-right-up-line" aria-hidden="true"></i></RouterLink></header>
    <section class="archive-selector" aria-label="选择关联人物卡">
      <div class="archive-choice-control"><span>关联 RPBox 人物卡</span><button type="button" :disabled="saving || creating || listLoading" aria-label="搜索选择人物卡" @click="openPicker()"><i class="ri-search-line" aria-hidden="true"></i><span>{{ card ? (card.display_name || title) + ' · #' + card.id : '按名称搜索 / 选择人物卡…' }}</span><i class="ri-arrow-down-s-line" aria-hidden="true"></i></button></div>
      <button type="button" :disabled="saving || creating" @click="create()"><i class="ri-add-line" aria-hidden="true"></i> {{ creating ? '创建中…' : '新建人物卡' }}</button><button type="button" :disabled="saving || creating" @click="openPicker()">导入人物卡</button>
      <p v-if="listError" role="alert">{{ listError }} <button type="button" @click="loadList">重试</button></p>
      <div v-if="card" class="linked-archive"><i class="ri-link" aria-hidden="true"></i><span>已关联 RPBox 人物卡 #{{ card.id }} · 保存会更新这张档案及所选子账号的云端存档</span><RouterLink :to="`/character-cards/${card.id}`">打开关联档案</RouterLink></div>
    </section>
    <div ref="editorBody" class="trp3-editor-body" :style="loading && loadingBodyHeight ? { minHeight: `${loadingBodyHeight}px` } : undefined">
    <div v-if="loading" class="trp3-state" role="status">读取人物卡…</div>
    <div v-else-if="error" class="trp3-state" role="alert"><p>{{ error }}</p><button type="button" @click="loadCard">重新加载</button></div>
    <div v-else-if="!card" class="trp3-state"><WowIcon icon="inv_misc_book_09" :size="64" /><h2>选择一本角色档案</h2><p>打开已有的 RPBox 人物卡继续编辑，或从空白人物卡、TRP3 云备份开始。</p><button type="button" :disabled="creating" @click="create()">创建第一张人物卡</button></div>
    <template v-else>
      <div class="trp3-window">
        <header class="trp3-titlebar"><img src="../../assets/trp3/ui-sidebar-logo.png" alt="Total RP 3" /><span>Total RP 3 <small>在线编辑</small></span><strong>{{ title }}</strong></header>
        <nav class="trp3-tabs" aria-label="人物卡编辑分区"><button v-for="entry in [{ id: 'characteristics', label: '特征', icon: 'ri-user-line' }, { id: 'glance', label: '第一印象', icon: 'ri-eye-line' }, { id: 'about', label: '关于', icon: 'ri-book-open-line' }]" :key="entry.id" type="button" :class="{ active: tab === entry.id }" :aria-pressed="tab === entry.id" @click="tab = entry.id as typeof tab; preview = false"><i :class="entry.icon" aria-hidden="true"></i>{{ entry.label }}</button><span></span><button v-if="tab === 'about'" type="button" :aria-pressed="preview" @click="preview = !preview">{{ preview ? '继续编辑' : '预览' }}</button></nav>
        <fieldset class="trp3-workspace" :disabled="saving || creating" :aria-label="selectedTabLabel">
          <template v-if="tab === 'characteristics'">
            <fieldset class="trp3-group"><legend>名字与称呼</legend><div class="trp3-name-grid">
              <button class="trp3-icon-button character-icon" type="button" aria-label="选择角色图标" @click="pickIcon(form.icon, value => form.icon = value)"><WowIcon :icon="form.icon || 'inv_misc_grouplooking'" :size="56" /></button>
              <label>称呼<input v-model="form.title" maxlength="128" /></label><label>名字<input v-model="form.first_name" maxlength="128" /></label><label>姓氏<input v-model="form.last_name" maxlength="128" /></label><label class="full-title">完整头衔<input v-model="form.full_title" maxlength="256" /></label>
            </div></fieldset>
            <fieldset class="trp3-group"><legend>特征</legend><div class="trp3-physical-grid">
              <label v-for="field in physicalFields" :key="field[0]">{{ field[1] }}<span class="trp3-field-with-color"><input v-model="form[field[0]]" :maxlength="field[2]" /><input v-if="field[0] === 'class' || field[0] === 'eye_color'" type="color" :aria-label="field[0] === 'class' ? '职业与名字颜色' : '眼睛颜色值'" :value="colorValue(field[0] === 'class' ? form.class_color : form.eye_color_hex)" @input="setColor(field[0] === 'class' ? 'class_color' : 'eye_color_hex', $event)" /></span></label>
              <label>感情状况<select v-model="form.relationship_status"><option value="">未填写</option><option v-for="(label, index) in relationLabels" :key="index" :value="String(index)">{{ label }}</option></select></label>
            </div></fieldset>
            <fieldset class="trp3-group"><legend>附加说明</legend>
              <div v-for="(entry, index) in form.additional_info" :key="index" class="trp3-misc-row"><button class="trp3-icon-button" type="button" :aria-label="`选择附加说明 ${index + 1} 图标`" @click="pickIcon(entry.icon, value => entry.icon = value)"><WowIcon :icon="entry.icon || 'inv_misc_note_06'" :size="32" /></button><select v-model="entry.id" :aria-label="`附加说明 ${index + 1} 类型`" @change="entry.name = entry.id === 1 ? '' : t(`characterCards.editor.additionalInfoTypes.${entry.id}`)"><option v-for="id in 11" :key="id" :value="id">{{ t(`characterCards.editor.additionalInfoTypes.${id}`) }}</option></select><input v-if="entry.id === 1" v-model="entry.name" placeholder="名称" :aria-label="`附加说明 ${index + 1} 名称`" maxlength="80" /><input v-model="entry.value" placeholder="内容" :aria-label="`附加说明 ${index + 1} 内容`" maxlength="500" /><button type="button" :aria-label="`移除附加说明 ${index + 1}`" @click="removeEntry(form.additional_info, index)">×</button></div>
              <button class="trp3-button" type="button" :disabled="form.additional_info.length >= 50" @click="form.additional_info.push({ id: 1, name: '', value: '', icon: 'inv_misc_note_06' })">添加附加说明</button>
            </fieldset>
            <fieldset class="trp3-group"><legend>个性</legend>
              <div v-for="(trait, index) in form.personality_traits" :key="index" class="trp3-trait"><div class="trait-top"><select v-model="trait.preset_id" :aria-label="`个性 ${index + 1} 类型`"><option :value="null">自定义两端</option><option v-for="id in 11" :key="id" :value="id">{{ t(`characterCards.editor.personalityPresets.${id}`) }}</option></select><button type="button" :aria-label="`移除个性 ${index + 1}`" @click="removeEntry(form.personality_traits, index)">×</button></div><div class="trait-range"><WowIcon v-if="trait.preset_id" :icon="trp3TraitIcons[trait.preset_id - 1]?.[0]" :size="32" /><button v-if="!trait.preset_id" class="trp3-icon-button" type="button" aria-label="选择左侧个性图标" @click="pickIcon(trait.left_icon, value => trait.left_icon = value)"><WowIcon :icon="trait.left_icon || 'inv_misc_questionmark'" :size="32" /></button><span>{{ traitLabels(trait)[0] }}</span><input v-model.number="trait.value" type="range" min="0" max="20" step="1" :aria-label="`个性 ${index + 1} 倾向`" /><span>{{ traitLabels(trait)[1] }}</span><button v-if="!trait.preset_id" class="trp3-icon-button" type="button" aria-label="选择右侧个性图标" @click="pickIcon(trait.right_icon, value => trait.right_icon = value)"><WowIcon :icon="trait.right_icon || 'inv_misc_questionmark'" :size="32" /></button><WowIcon v-if="trait.preset_id" :icon="trp3TraitIcons[trait.preset_id - 1]?.[1]" :size="32" /><small>{{ trait.value }} / 20</small></div><div v-if="!trait.preset_id" class="trait-custom"><label>左侧倾向<input v-model="trait.left_text" maxlength="80" /></label><label>右侧倾向<input v-model="trait.right_text" maxlength="80" /></label><label>左侧颜色<input type="color" :value="traitColor(trait.left_color)" @input="setTraitColor(trait, 'left_color', $event)" /></label><label>右侧颜色<input type="color" :value="traitColor(trait.right_color)" @input="setTraitColor(trait, 'right_color', $event)" /></label></div></div>
              <button class="trp3-button" type="button" :disabled="form.personality_traits.length >= 50" @click="addTrait">添加一种个性</button>
            </fieldset>
          </template>
          <template v-else-if="tab === 'glance'">
            <div class="glance-strip"><button v-for="entry in form.impressions" :key="entry.slot" type="button" :class="{ enabled: entry.active }" :aria-label="`第一印象 ${entry.slot} 图标`" @click="pickIcon(entry.trp3_icon, value => entry.trp3_icon = value)"><WowIcon :icon="entry.trp3_icon || 'inv_misc_questionmark'" :size="48" /><span>{{ entry.slot }}</span></button></div>
            <fieldset v-for="entry in form.impressions" :key="entry.slot" class="trp3-group"><legend>第一印象 {{ entry.slot }}</legend><div class="glance-heading"><button class="trp3-icon-button" type="button" :aria-label="`编辑第一印象 ${entry.slot} 图标`" @click="pickIcon(entry.trp3_icon, value => entry.trp3_icon = value)"><WowIcon :icon="entry.trp3_icon || 'inv_misc_questionmark'" :size="42" /></button><label>标题<input v-model="entry.title" maxlength="128" /></label><label class="trp3-check"><input v-model="entry.active" type="checkbox" />启用</label></div><TRP3MarkupEditor v-model="entry.text" :label="`第一印象 ${entry.slot} 内容`" :rows="4" /></fieldset>
          </template>
          <template v-else>
            <div class="about-options"><label>模板<select v-model="about.TE" aria-label="关于模板"><option :value="1">模板 1 · 自由排版</option><option :value="2">模板 2 · 分栏描述</option><option :value="3">模板 3 · 外貌、性格与历史</option></select></label><label>背景<select v-model="about.BK" aria-label="关于背景"><option :value="undefined">默认</option><option :value="1">深色</option><option :value="7">羊皮纸</option><option v-if="about.BK && about.BK !== 1 && about.BK !== 7" :value="about.BK">原背景 {{ about.BK }}</option></select></label></div>
            <TRP3AboutPreview v-if="preview" :about="about" />
            <TRP3MarkupEditor v-else-if="about.TE === 1" v-model="about.T1.TX" label="模板 1 正文" :rows="20" />
            <div v-else-if="about.TE === 2" class="trp3-template2"><fieldset v-for="(section, index) in about.T2" :key="index" class="trp3-group"><legend>段落 {{ index + 1 }}</legend><div class="section-tools"><button class="trp3-icon-button" type="button" :aria-label="`段落 ${index + 1} 图标`" @click="sectionIcon(section)"><WowIcon :icon="section.IC || 'inv_misc_book_09'" :size="42" /></button><select v-model="section.BK" :aria-label="`段落 ${index + 1} 背景`"><option :value="undefined">默认背景</option><option :value="1">深色</option><option :value="7">羊皮纸</option><option v-if="section.BK && section.BK !== 1 && section.BK !== 7" :value="section.BK">原背景 {{ section.BK }}</option></select><button type="button" :disabled="index === 0" @click="moveSection(index, -1)">上移</button><button type="button" :disabled="index === about.T2.length - 1" @click="moveSection(index, 1)">下移</button><button type="button" @click="removeEntry(about.T2, index)">删除</button></div><TRP3MarkupEditor v-model="section.TX" :label="`段落 ${index + 1} 正文`" :rows="5" /></fieldset><button class="trp3-button" type="button" :disabled="about.T2.length >= 50" @click="addSection">添加段落</button></div>
            <div v-else><fieldset v-for="section in template3Sections" :key="section.key" class="trp3-group"><legend>{{ section.title }}</legend><div class="section-tools"><button class="trp3-icon-button" type="button" :aria-label="`${section.title}图标`" @click="sectionIcon(about.T3[section.key])"><WowIcon :icon="about.T3[section.key].IC || 'inv_misc_book_09'" :size="42" /></button><select v-model="about.T3[section.key].BK" :aria-label="`${section.title}背景`"><option :value="undefined">默认背景</option><option :value="1">深色</option><option :value="7">羊皮纸</option><option v-if="about.T3[section.key].BK && about.T3[section.key].BK !== 1 && about.T3[section.key].BK !== 7" :value="about.T3[section.key].BK">原背景 {{ about.T3[section.key].BK }}</option></select></div><TRP3MarkupEditor v-model="about.T3[section.key].TX" :label="`${section.title}正文`" :rows="5" /></fieldset></div>
          </template>
        </fieldset>
        <footer class="trp3-footer"><span role="status">{{ saving ? '正在保存…' : dirty ? '有未保存的修改' : savedAt ? `已保存 · ${savedAt}` : '已与 RPBox 档案关联' }}</span><button class="trp3-button" type="button" :disabled="saving" @click="exportLua">导出 TRP3 Lua</button><button class="trp3-button save-button" type="button" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存到所选云端存档' }}</button></footer>
      </div>
      <div class="archive-meta"><label>RPBox 档案名称<input v-model="form.display_name" :disabled="saving || creating" maxlength="256" placeholder="用于档案列表的名称" /></label><p>保存到勾选的子账号云端存档；需要同步到游戏时，请前往人物卡备份。</p></div>
    </template>
    </div>
    <CharacterSaveTargets v-if="card" ref="targetPicker" :card="card" :busy="saving || creating" @change="saveTargets = $event" @save="save" />
    <footer class="trp3-credit">基于 Total RP 3 的界面布局与格式语法适配。<a href="/licenses/TRP3-NOTICE.txt" target="_blank" rel="noopener">来源与许可</a></footer>
    <TRP3IconPicker v-model="iconOpen" :selected="selectedIcon" @select="applyIcon" />
    <CharacterCardPicker v-model="pickerOpen" :cards="cards" :current-card-id="cardId" :busy="saving || creating" @select="chooseCard" />
  </main>
</template>

<style scoped src="./trp3Editor.css"></style>
