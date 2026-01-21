<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import { save } from '@tauri-apps/plugin-dialog'
import { getStory, updateStory, addStoryEntries, publishStory, updateStoryEntry, deleteStoryEntry, type Story, type StoryEntry } from '@/api/story'
import { getCharacter, updateCharacter, listCharacters, type Character } from '@/api/character'
import { listTags, getStoryTags, addStoryTag, removeStoryTag, type Tag } from '@/api/tag'
import { listGuilds, getStoryGuilds, archiveStoryToGuild, removeStoryFromGuild, type Guild } from '@/api/guild'
import { useUserStore } from '@/stores/user'
import RButton from '@/components/RButton.vue'
import RCard from '@/components/RCard.vue'
import RInput from '@/components/RInput.vue'
import RModal from '@/components/RModal.vue'
import WowIcon from '@/components/WowIcon.vue'
import RichEditor from '@/components/RichEditor.vue'
import CharacterCard from '@/components/CharacterCard.vue'
import RColorPicker from '@/components/RColorPicker.vue'
import RAvatarPicker from '@/components/RAvatarPicker.vue'
import TagSelector from '@/components/TagSelector.vue'
import ImageViewer from '@/components/ImageViewer.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const loading = ref(true)
const story = ref<Story | null>(null)
const entries = ref<StoryEntry[]>([])
const editing = ref(false)
const editTitle = ref('')
const editDesc = ref('')
const saving = ref(false)

// 添加条目对话框
const showAddModal = ref(false)
const newEntryContent = ref('')
const newEntrySpeaker = ref('')
const newEntryType = ref('dialogue')
const newEntryChannel = ref('SAY')
const newEntryTimestamp = ref('')
const newEntryCharacterId = ref<number | null>(null)
const adding = ref(false)

// 图片条目相关
const newEntryImageFile = ref<File | null>(null)
const newEntryImagePreview = ref('')
const newEntryImageDesc = ref('')
const newEntryImageOffset = ref(0)

// 可选人物卡列表
const availableCharacters = ref<Character[]>([])

// 发布分享
const publishing = ref(false)
const showShareModal = ref(false)

// 角色信息弹窗
const showCharacterModal = ref(false)
const selectedCharacter = ref<StoryEntry | null>(null)
const characterCardPosition = ref({ x: 0, y: 0 })

// 角色数据缓存 (character_id -> Character)
const charactersMap = ref<Map<number, Character>>(new Map())

// 标签管理
const storyTags = ref<Tag[]>([])
const allTags = ref<Tag[]>([])
const showTagModal = ref(false)

// 公会归档
const storyGuilds = ref<Guild[]>([])
const myGuilds = ref<Guild[]>([])
const showGuildModal = ref(false)

// 导入导出
const showImportModal = ref(false)
const importFile = ref<File | null>(null)
const importing = ref(false)

// 条目编辑
const showEditEntryModal = ref(false)
const editingEntry = ref<StoryEntry | null>(null)
const editEntryContent = ref('')
const editEntrySpeaker = ref('')
const editEntryChannel = ref('SAY')
const editEntryType = ref('dialogue')
const editEntryCharacterId = ref<number | null>(null)
const editEntryTimestamp = ref('')
const editEntryImageOffset = ref(0)
const savingEntry = ref(false)

const storyId = computed(() => Number(route.params.id))

const showImageViewer = ref(false)
const viewerImages = ref<string[]>([])
const viewerStartIndex = ref(0)

const imageEntries = computed(() => {
  const result: { id: number; image: string }[] = []
  for (const entry of entries.value) {
    if (entry.type !== 'image') continue
    const parsed = parseImageEntry(entry)
    if (parsed?.image) {
      result.push({ id: entry.id, image: parsed.image })
    }
  }
  return result
})

// 权限检查：只有剧情上传者、管理员、版主可以编辑
const canEdit = computed(() => {
  if (!userStore.user || !story.value) return false
  return story.value.user_id === userStore.user.id ||
         userStore.isAdmin ||
         userStore.isModerator
})

// WoW 主题颜色预设
const wowColorPresets = [
  // 职业颜色
  'C41E3A', // 死亡骑士
  'FF7C0A', // 德鲁伊
  'AAD372', // 猎人
  '3FC7EB', // 法师
  '00FF98', // 武僧
  'F48CBA', // 圣骑士
  'FFFFFF', // 牧师
  'FFF468', // 盗贼
  '0070DD', // 萨满
  '8788EE', // 术士
  'C69B6D', // 战士
  'A330C9', // 恶魔猎手
  // 阵营颜色
  'FF3333', // 部落红
  '0066FF', // 联盟蓝
  // 常用颜色
  'FFD700', // 金色
  'B87333', // 铜色
  '9B59B6', // 紫色
  '2ECC71', // 绿色
  'E74C3C', // 红色
  '95A5A6', // 灰色
]

async function loadStory() {
  loading.value = true
  try {
    const res = await getStory(storyId.value)
    story.value = res.story
    entries.value = res.entries || []
    console.log('[StoryDetail] entries:', entries.value)
    console.log('[StoryDetail] 第一条entry:', entries.value[0])
    editTitle.value = res.story.title
    editDesc.value = res.story.description

    // 加载所有关联的角色信息
    const characterIds = new Set<number>()
    for (const entry of entries.value) {
      if (entry.character_id) {
        characterIds.add(entry.character_id)
      }
    }

    // 并行获取所有角色信息
    const characterPromises = Array.from(characterIds).map(async (id) => {
      try {
        const character = await getCharacter(id)
        charactersMap.value.set(id, character)
      } catch (e) {
        console.error(`加载角色 ${id} 失败:`, e)
      }
    })
    await Promise.all(characterPromises)
    console.log('[StoryDetail] charactersMap:', charactersMap.value)
  } catch (e) {
    console.error('加载失败:', e)
  } finally {
    loading.value = false
  }
}

function startEdit() {
  editing.value = true
  editTitle.value = story.value?.title || ''
  editDesc.value = story.value?.description || ''
}

async function saveEdit() {
  if (!editTitle.value.trim()) return
  saving.value = true
  try {
    const updated = await updateStory(storyId.value, {
      title: editTitle.value,
      description: editDesc.value,
    })
    story.value = updated
    editing.value = false
  } catch (e) {
    console.error('保存失败:', e)
  } finally {
    saving.value = false
  }
}

// 处理图片选择
function handleImageSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  // 检查文件类型
  if (!file.type.startsWith('image/')) {
    alert('请选择图片文件')
    return
  }

  // 检查文件大小（最大10MB）
  if (file.size > 10 * 1024 * 1024) {
    alert('图片文件不能超过10MB')
    return
  }

  newEntryImageFile.value = file

  // 生成预览
  const reader = new FileReader()
  reader.onload = (e) => {
    newEntryImagePreview.value = e.target?.result as string
  }
  reader.readAsDataURL(file)
}

// 清除图片
function clearImage() {
  newEntryImageFile.value = null
  newEntryImagePreview.value = ''
  newEntryImageDesc.value = ''
}

function normalizeOffset(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.min(Math.max(Math.trunc(value), 0), 999)
}

async function handleAddEntry() {
  // 验证：图片类型需要有图片文件，其他类型需要有内容
  if (newEntryType.value === 'image') {
    if (!newEntryImageFile.value) {
      alert('请选择图片')
      return
    }
  } else {
    if (!newEntryContent.value.trim()) return
  }
  adding.value = true
  try {
    // 转换时间格式为 ISO 8601
    const entryTime = newEntryTimestamp.value ? new Date(newEntryTimestamp.value) : new Date()
    if (newEntryType.value === 'image') {
      entryTime.setMilliseconds(normalizeOffset(newEntryImageOffset.value))
    } else {
      entryTime.setMilliseconds(0)
    }
    const timestamp = entryTime.toISOString()

    // 准备内容
    let content = newEntryContent.value

    // 如果是图片类型，将图片转换为 base64 并与描述一起保存
    if (newEntryType.value === 'image' && newEntryImageFile.value) {
      const reader = new FileReader()
      const base64Promise = new Promise<string>((resolve) => {
        reader.onload = (e) => {
          resolve(e.target?.result as string)
        }
        reader.readAsDataURL(newEntryImageFile.value!)
      })
      const base64Data = await base64Promise

      // 将图片数据和描述保存为 JSON 格式
      content = JSON.stringify({
        image: base64Data,
        description: newEntryImageDesc.value
      })
    }

    const speaker = newEntryType.value === 'image' ? '' : newEntrySpeaker.value
    const channel = newEntryType.value === 'image' ? '' : newEntryChannel.value

    await addStoryEntries(storyId.value, [{
      content: content,
      speaker: speaker,
      type: newEntryType.value,
      channel: channel,
      timestamp: timestamp,
    }])

    // 清理表单
    showAddModal.value = false
    newEntryContent.value = ''
    newEntrySpeaker.value = ''
    newEntryType.value = 'dialogue'
    newEntryChannel.value = 'SAY'
    newEntryTimestamp.value = ''
    newEntryCharacterId.value = null
    newEntryImageOffset.value = 0
    clearImage()
    await loadStory()
  } catch (e) {
    console.error('添加失败:', e)
  } finally {
    adding.value = false
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 解析图片条目内容
function parseImageEntry(entry: StoryEntry): { image: string; description: string } | null {
  if (entry.type !== 'image') return null
  try {
    return JSON.parse(entry.content)
  } catch {
    return null
  }
}

function openImageViewer(entryId: number) {
  const images = imageEntries.value
  if (!images.length) return
  const index = images.findIndex((image) => image.id === entryId)
  if (index < 0) return
  viewerImages.value = images.map((image) => image.image)
  viewerStartIndex.value = index
  showImageViewer.value = true
}

function goBack() {
  // 检查是否从公会剧情页面进入
  const from = route.query.from as string
  const guildId = route.query.guildId as string

  if (from === 'guild' && guildId) {
    router.push({ name: 'guild-stories', params: { id: guildId } })
  } else {
    router.push({ name: 'archives' })
  }
}

// ========== 标签管理 ==========
async function loadTags() {
  try {
    const [tagsRes, storyTagsRes] = await Promise.all([
      listTags('story'),
      getStoryTags(storyId.value)
    ])
    // 前端也过滤一下，确保只显示 story 类型的标签
    allTags.value = (tagsRes.tags || []).filter(t => !t.category || t.category === 'story')
    storyTags.value = storyTagsRes.tags || []
  } catch (e) {
    console.error('加载标签失败:', e)
  }
}

async function handleAddTag(tagId: number) {
  try {
    await addStoryTag(storyId.value, tagId)
    await loadTags()
  } catch (e) {
    console.error('添加标签失败:', e)
  }
}

async function handleRemoveTag(tagId: number) {
  try {
    await removeStoryTag(storyId.value, tagId)
    await loadTags()
  } catch (e) {
    console.error('移除标签失败:', e)
  }
}

// ========== 公会归档 ==========
async function loadGuilds() {
  try {
    const [guildsRes, storyGuildsRes] = await Promise.all([
      listGuilds(),
      getStoryGuilds(storyId.value)
    ])
    myGuilds.value = guildsRes.guilds || []
    storyGuilds.value = storyGuildsRes.guilds || []
  } catch (e) {
    console.error('加载公会失败:', e)
  }
}

// 加载可选人物卡列表
async function loadAvailableCharacters() {
  try {
    const res = await listCharacters()
    availableCharacters.value = res.characters || []
  } catch (e) {
    console.error('加载人物卡失败:', e)
  }
}

// 选择人物卡时自动填充说话者名称
function handleCharacterSelect(characterId: number | null) {
  newEntryCharacterId.value = characterId
  if (characterId) {
    const character = availableCharacters.value.find(c => c.id === characterId)
    if (character) {
      const name = character.custom_name ||
        (character.first_name ?
          (character.last_name ? `${character.first_name} ${character.last_name}` : character.first_name)
          : character.game_id?.split('-')[0] || '')
      newEntrySpeaker.value = name
    }
  }
}

// 获取人物卡显示名称
function getCharacterDisplayName(character: Character): string {
  if (character.custom_name) return character.custom_name
  if (character.first_name) {
    return character.last_name
      ? `${character.first_name} ${character.last_name}`
      : character.first_name
  }
  return character.game_id?.split('-')[0] || '未知角色'
}

function getEntrySpeakerName(entry: StoryEntry): string {
  if (entry.type === 'narration') return '旁白'
  const character = getEntryCharacter(entry)
  if (character) {
    return getCharacterDisplayName(character)
  }
  return entry.speaker || '旁白'
}

function getEntrySpeakerInitial(entry: StoryEntry): string {
  const name = getEntrySpeakerName(entry)
  return name ? name.charAt(0) : '?'
}

async function handleArchiveToGuild(guildId: number) {
  try {
    await archiveStoryToGuild(guildId, storyId.value)
    await loadGuilds()
    showGuildModal.value = false
  } catch (e: any) {
    alert(e.message || '归档失败')
  }
}

async function handleRemoveFromGuild(guildId: number) {
  if (!confirm('确定要从该公会移除归档吗？')) return
  try {
    await removeStoryFromGuild(guildId, storyId.value)
    await loadGuilds()
  } catch (e) {
    console.error('移除归档失败:', e)
  }
}

// 获取未归档的公会列表
const availableGuilds = computed(() => {
  const archivedIds = new Set(storyGuilds.value.map(g => g.id))
  return myGuilds.value.filter(g => !archivedIds.has(g.id))
})

// 分享链接
const shareUrl = computed(() => {
  if (!story.value?.share_code) return ''
  return `${window.location.origin}/story/${story.value.share_code}`
})

async function togglePublish() {
  if (!story.value) return
  publishing.value = true
  try {
    const updated = await publishStory(storyId.value, !story.value.is_public)
    story.value = updated
    if (updated.is_public) {
      showShareModal.value = true
    }
  } catch (e) {
    console.error('发布失败:', e)
  } finally {
    publishing.value = false
  }
}

function copyShareLink() {
  navigator.clipboard.writeText(shareUrl.value)
}

function showCharacterInfo(entry: StoryEntry, event: MouseEvent) {
  if (entry.type === 'narration' || entry.type === 'image') return
  selectedCharacter.value = entry
  // 记录点击位置
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  characterCardPosition.value = {
    x: rect.right,
    y: rect.top
  }
  showCharacterModal.value = true
}

// 获取条目对应的角色信息
function getEntryCharacter(entry: StoryEntry): Character | undefined {
  if (entry.character_id) {
    return charactersMap.value.get(entry.character_id)
  }
  return undefined
}

// 获取条目的头像图标
function getEntryIcon(entry: StoryEntry): string {
  const character = getEntryCharacter(entry)
  if (character) {
    return character.custom_avatar || character.icon || ''
  }
  return ''
}

// 获取条目的名字颜色
function getEntryColor(entry: StoryEntry): string {
  const character = getEntryCharacter(entry)
  if (character) {
    return character.custom_color || character.color || ''
  }
  return ''
}

// 判断是否是NPC
function isNpcEntry(entry: StoryEntry): boolean {
  const character = getEntryCharacter(entry)
  return character?.is_npc || false
}

// 判断是否是旁白
function isNarrationEntry(entry: StoryEntry): boolean {
  return entry.type === 'narration'
}

// 编辑角色信息
const showEditModal = ref(false)
const editingCharacter = ref<Character | null>(null)

function handleEditCharacter(character: Character) {
  editingCharacter.value = { ...character }
  showCharacterModal.value = false
  showEditModal.value = true
}

const savingCharacter = ref(false)

async function saveCharacterEdit() {
  if (!editingCharacter.value) return
  savingCharacter.value = true
  try {
    const updated = await updateCharacter(editingCharacter.value.id, {
      custom_name: editingCharacter.value.custom_name,
      custom_color: editingCharacter.value.custom_color,
      custom_avatar: editingCharacter.value.custom_avatar,
      title: editingCharacter.value.title,
      full_title: editingCharacter.value.full_title,
      race: editingCharacter.value.race,
      class: editingCharacter.value.class,
      age: editingCharacter.value.age,
      height: editingCharacter.value.height,
      eye_color: editingCharacter.value.eye_color,
      residence: editingCharacter.value.residence,
      birthplace: editingCharacter.value.birthplace,
    })
    // 更新缓存
    charactersMap.value.set(updated.id, updated)
    showEditModal.value = false
  } catch (e) {
    console.error('保存失败:', e)
  } finally {
    savingCharacter.value = false
  }
}

function getChannelLabel(channel: string): string {
  const map: Record<string, string> = {
    // 新格式（简写）
    'SAY': '说',
    'YELL': '喊',
    'EMOTE': '表情',
    'TEXT_EMOTE': '表情',
    'PARTY': '小队',
    'RAID': '团队',
    'WHISPER': '密语',
    // 旧格式（完整事件名）
    'CHAT_MSG_SAY': '说',
    'CHAT_MSG_YELL': '喊',
    'CHAT_MSG_EMOTE': '表情',
    'CHAT_MSG_TEXT_EMOTE': '表情',
    'CHAT_MSG_PARTY': '小队',
    'CHAT_MSG_RAID': '团队',
    'CHAT_MSG_WHISPER': '密语',
  }
  return map[channel] || channel
}

function getChannelClass(channel: string): string {
  if (channel === 'YELL' || channel === 'CHAT_MSG_YELL') return 'channel-yell'
  if (channel === 'WHISPER' || channel === 'CHAT_MSG_WHISPER') return 'channel-whisper'
  return ''
}

// 获取频道对应的文字颜色
function getChannelTextColor(channel: string): string {
  const colorMap: Record<string, string> = {
    'SAY': '',  // 默认颜色
    'YELL': '#FF3333',  // 红色
    'WHISPER': '#B39DDB',  // 紫色
    'EMOTE': '#FF8C00',  // 橙色
    'TEXT_EMOTE': '#FF8C00',  // 橙色
    'PARTY': '#AAAAFF',  // 蓝色
    'RAID': '#FF7F00',  // 橙色
    'CHAT_MSG_SAY': '',
    'CHAT_MSG_YELL': '#FF3333',
    'CHAT_MSG_WHISPER': '#B39DDB',
    'CHAT_MSG_EMOTE': '#FF8C00',
    'CHAT_MSG_TEXT_EMOTE': '#FF8C00',
    'CHAT_MSG_PARTY': '#AAAAFF',
    'CHAT_MSG_RAID': '#FF7F00',
  }
  return colorMap[channel] || ''
}

// ========== 导出/导入功能 ==========
interface StoryExportData {
  version: string
  exported_at: string
  story: {
    title: string
    description: string
    status: string
    tags: string[]
  }
  entries: Array<{
    type: string
    speaker: string
    content: string
    channel: string
    timestamp: string
    sort_order: number
  }>
  characters: Record<number, {
    first_name: string
    last_name: string
    title: string
    race: string
    class: string
    icon: string
    color: string
    custom_name: string
    custom_color: string
    custom_avatar: string
  }>
}

async function exportStory() {
  if (!story.value) return

  const exportData: StoryExportData = {
    version: '1.0',
    exported_at: new Date().toISOString(),
    story: {
      title: story.value.title,
      description: story.value.description,
      status: story.value.status,
      tags: storyTags.value.map(t => t.name),
    },
    entries: entries.value.map(e => ({
      type: e.type,
      speaker: e.speaker,
      content: e.content,
      channel: e.channel,
      timestamp: e.timestamp,
      sort_order: e.sort_order,
    })),
    characters: {},
  }

  // 导出角色信息
  charactersMap.value.forEach((char, id) => {
    exportData.characters[id] = {
      first_name: char.first_name,
      last_name: char.last_name,
      title: char.title,
      race: char.race,
      class: char.class,
      icon: char.icon,
      color: char.color,
      custom_name: char.custom_name,
      custom_color: char.custom_color,
      custom_avatar: char.custom_avatar,
    }
  })

  // 使用 Tauri 保存对话框
  const defaultName = `${story.value.title}_${new Date().toISOString().split('T')[0]}.json`
  const filePath = await save({
    defaultPath: defaultName,
    filters: [{ name: 'JSON', extensions: ['json'] }]
  })

  if (filePath) {
    try {
      await invoke('save_text_file', {
        path: filePath,
        content: JSON.stringify(exportData, null, 2)
      })
    } catch (e) {
      console.error('导出失败:', e)
      alert('导出失败: ' + e)
    }
  }
}

function handleImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    importFile.value = input.files[0]
  }
}

async function importStoryFromFile() {
  if (!importFile.value) return

  importing.value = true
  try {
    const text = await importFile.value.text()
    const data = JSON.parse(text) as StoryExportData

    if (!data.version || !data.entries) {
      throw new Error('无效的剧情文件格式')
    }

    // 导入条目到当前剧情
    const entriesToAdd = data.entries.map(e => ({
      content: e.content,
      speaker: e.speaker,
      type: e.type,
      channel: e.channel,
      timestamp: e.timestamp,
    }))

    await addStoryEntries(storyId.value, entriesToAdd)

    showImportModal.value = false
    importFile.value = null
    await loadStory()
  } catch (e: any) {
    console.error('导入失败:', e)
    alert('导入失败: ' + (e.message || '未知错误'))
  } finally {
    importing.value = false
  }
}

// ========== 条目编辑 ==========
function startEditEntry(entry: StoryEntry) {
  editingEntry.value = entry
  editEntryContent.value = entry.content
  editEntrySpeaker.value = entry.speaker || ''
  // 将 TEXT_EMOTE 统一转换为 EMOTE，因为编辑表单中只有 EMOTE 选项
  editEntryChannel.value = entry.channel === 'TEXT_EMOTE' ? 'EMOTE' : (entry.channel || 'SAY')
  editEntryType.value = entry.type || 'dialogue'
  editEntryCharacterId.value = entry.character_id || null
  // 初始化时间（格式化为 datetime-local）
  const timestamp = new Date(entry.timestamp)
  const year = timestamp.getFullYear()
  const month = String(timestamp.getMonth() + 1).padStart(2, '0')
  const day = String(timestamp.getDate()).padStart(2, '0')
  const hours = String(timestamp.getHours()).padStart(2, '0')
  const minutes = String(timestamp.getMinutes()).padStart(2, '0')
  const seconds = String(timestamp.getSeconds()).padStart(2, '0')
  editEntryTimestamp.value = `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`
  editEntryImageOffset.value = entry.type === 'image' ? timestamp.getMilliseconds() : 0
  showEditEntryModal.value = true
}

async function saveEntryEdit() {
  if (!editingEntry.value) return
  savingEntry.value = true
  try {
    const entryTime = new Date(editEntryTimestamp.value)
    if (editEntryType.value === 'image') {
      entryTime.setMilliseconds(normalizeOffset(editEntryImageOffset.value))
    } else {
      entryTime.setMilliseconds(0)
    }

    const isImage = editEntryType.value === 'image'
    const speaker = isImage ? '' : editEntrySpeaker.value
    const channel = isImage ? '' : editEntryChannel.value
    const characterId = isImage ? null : editEntryCharacterId.value

    await updateStoryEntry(storyId.value, editingEntry.value.id, {
      content: editEntryContent.value,
      speaker: speaker,
      channel: channel,
      type: editEntryType.value,
      character_id: characterId,
      timestamp: entryTime.toISOString(),
    })

    // 如果关联了新角色，确保角色信息在 charactersMap 中
    if (editEntryCharacterId.value) {
      const char = availableCharacters.value.find(c => c.id === editEntryCharacterId.value)
      if (char && !charactersMap.value.has(editEntryCharacterId.value)) {
        charactersMap.value.set(editEntryCharacterId.value, char)
      }
    }

    showEditEntryModal.value = false
    await loadStory()
  } catch (e) {
    console.error('保存失败:', e)
  } finally {
    savingEntry.value = false
  }
}

// 编辑时选择人物卡
function handleEditCharacterSelect(characterId: number | null) {
  editEntryCharacterId.value = characterId
  if (characterId) {
    const character = availableCharacters.value.find(c => c.id === characterId)
    if (character) {
      const name = character.custom_name ||
        (character.first_name ?
          (character.last_name ? `${character.first_name} ${character.last_name}` : character.first_name)
          : character.game_id?.split('-')[0] || '')
      editEntrySpeaker.value = name
    }
  }
}

async function handleDeleteEntry(entry: StoryEntry) {
  if (!confirm('确定要删除这条记录吗？')) return
  try {
    await deleteStoryEntry(storyId.value, entry.id)
    await loadStory()
  } catch (e) {
    console.error('删除失败:', e)
  }
}

// 在指定条目之前插入
function insertEntryBefore(entry: StoryEntry) {
  // 计算插入时间：当前条目时间减去1秒
  const currentTime = new Date(entry.timestamp)
  const insertTime = new Date(currentTime.getTime() - 1000) // 减去1秒

  // 格式化为 datetime-local 格式
  const year = insertTime.getFullYear()
  const month = String(insertTime.getMonth() + 1).padStart(2, '0')
  const day = String(insertTime.getDate()).padStart(2, '0')
  const hours = String(insertTime.getHours()).padStart(2, '0')
  const minutes = String(insertTime.getMinutes()).padStart(2, '0')
  const seconds = String(insertTime.getSeconds()).padStart(2, '0')

  newEntryTimestamp.value = `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`
  showAddModal.value = true
}

// 在指定条目之后插入
function insertEntryAfter(entry: StoryEntry) {
  // 计算插入时间：当前条目时间加上1秒
  const currentTime = new Date(entry.timestamp)
  const insertTime = new Date(currentTime.getTime() + 1000) // 加上1秒

  // 格式化为 datetime-local 格式
  const year = insertTime.getFullYear()
  const month = String(insertTime.getMonth() + 1).padStart(2, '0')
  const day = String(insertTime.getDate()).padStart(2, '0')
  const hours = String(insertTime.getHours()).padStart(2, '0')
  const minutes = String(insertTime.getMinutes()).padStart(2, '0')
  const seconds = String(insertTime.getSeconds()).padStart(2, '0')

  newEntryTimestamp.value = `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`
  showAddModal.value = true
}

onMounted(() => {
  loadStory()
  loadTags()
  loadGuilds()
  loadAvailableCharacters()
})
</script>

<template>
  <div class="story-detail">
    <!-- 返回按钮 -->
    <div class="back-bar">
      <button class="btn-back" @click="goBack">
        <i class="ri-arrow-left-line"></i> 返回剧情列表
      </button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <i class="ri-loader-4-line spinning"></i> 加载中...
    </div>

    <!-- 剧情内容 -->
    <template v-else-if="story">
      <!-- 剧情头部 -->
      <RCard class="story-header">
        <template v-if="!editing">
          <div class="header-content">
            <div class="header-info">
              <h1>{{ story.title }}</h1>
              <p class="description">{{ story.description || '暂无描述' }}</p>
              <div class="meta">
                <span class="meta-item">
                  <i class="ri-time-line"></i> {{ formatDate(story.created_at) }}
                </span>
              </div>
              <!-- 标签显示 -->
              <div class="tags-section">
                <div class="tags-list">
                  <span
                    v-for="tag in storyTags"
                    :key="tag.id"
                    class="tag-item"
                    :style="{ background: `#${tag.color}20`, color: `#${tag.color}` }"
                  >
                    {{ tag.name }}
                    <i v-if="canEdit" class="ri-close-line" @click.stop="handleRemoveTag(tag.id)"></i>
                  </span>
                  <button v-if="canEdit" class="add-tag-btn" @click="showTagModal = true">
                    <i class="ri-add-line"></i> 添加标签
                  </button>
                </div>
              </div>
              <!-- 公会归档显示 -->
              <div v-if="storyGuilds.length > 0" class="guilds-section">
                <span class="guilds-label">已归档到:</span>
                <div class="guilds-list">
                  <span
                    v-for="guild in storyGuilds"
                    :key="guild.id"
                    class="guild-badge"
                    :style="{ borderColor: `#${guild.color || 'B87333'}` }"
                  >
                    {{ guild.name }}
                    <i v-if="canEdit" class="ri-close-line" @click.stop="handleRemoveFromGuild(guild.id)"></i>
                  </span>
                </div>
              </div>
            </div>
            <div class="header-actions">
              <!-- 分享功能暂时禁用（需要网页端支持）
              <button
                class="action-btn action-btn--primary"
                :disabled="publishing"
                @click="togglePublish"
              >
                <i class="ri-loader-4-line spinning" v-if="publishing"></i>
                <i class="ri-share-line" v-else></i>
                {{ story.is_public ? '取消公开' : '公开分享' }}
              </button>
              -->

              <!-- Secondary action buttons -->
              <button v-if="canEdit" class="action-btn action-btn--secondary" @click="startEdit">
                <i class="ri-edit-line"></i> 编辑信息
              </button>

              <button v-if="canEdit" class="action-btn action-btn--secondary" @click="showGuildModal = true">
                <i class="ri-archive-line"></i> 归档到公会
              </button>

              <!-- Icon button group -->
              <div class="icon-button-group">
                <button v-if="canEdit" class="icon-btn" @click="showAddModal = true" title="添加条目">
                  <i class="ri-chat-new-line"></i>
                </button>
                <button v-if="canEdit" class="icon-btn" @click="exportStory" title="导出">
                  <i class="ri-download-line"></i>
                </button>
                <button v-if="canEdit" class="icon-btn" @click="showImportModal = true" title="导入">
                  <i class="ri-upload-line"></i>
                </button>
                <!-- 分享链接按钮暂时禁用（需要网页端支持）
                <button v-if="story.is_public" class="icon-btn" @click="showShareModal = true" title="分享链接">
                  <i class="ri-share-forward-line"></i>
                </button>
                -->
              </div>
            </div>
          </div>
        </template>
        <template v-else>
          <div class="edit-form">
            <div class="form-field">
              <label>标题</label>
              <RInput v-model="editTitle" placeholder="剧情标题" />
            </div>
            <div class="form-field">
              <label>描述</label>
              <textarea v-model="editDesc" placeholder="剧情描述" rows="3"></textarea>
            </div>
            <div class="edit-actions">
              <RButton @click="editing = false">取消</RButton>
              <RButton type="primary" :loading="saving" @click="saveEdit">保存</RButton>
            </div>
          </div>
        </template>
      </RCard>

      <!-- 剧情条目列表 -->
      <div class="entries-section">
        <h2>剧情内容 ({{ entries.length }} 条)</h2>
        <div v-if="entries.length === 0" class="empty-entries">
          <p>暂无内容，点击上方"添加条目"开始记录</p>
        </div>
        <div v-else class="entries-list">
          <div v-for="entry in entries" :key="entry.id" class="entry-item" :class="entry.type">
            <div
              v-if="entry.type !== 'image'"
              class="entry-avatar"
              :class="{ clickable: entry.type !== 'narration' }"
              @click="showCharacterInfo(entry, $event)"
            >
              <span v-if="isNpcEntry(entry)" class="avatar-npc">NPC</span>
              <span v-else-if="isNarrationEntry(entry)" class="avatar-narration">旁白</span>
              <WowIcon v-else-if="getEntryIcon(entry)" :icon="getEntryIcon(entry)" :size="40" :fallback="getEntrySpeakerInitial(entry)" />
              <span v-else class="avatar-fallback">{{ getEntrySpeakerInitial(entry) }}</span>
            </div>
            <div class="entry-content">
              <div class="entry-header">
                <span v-if="entry.type !== 'image'" class="speaker" :style="getEntryColor(entry) ? { color: '#' + getEntryColor(entry) } : {}">
                  {{ getEntrySpeakerName(entry) }}
                </span>
                <span v-if="entry.channel && entry.type !== 'narration' && entry.type !== 'image'" class="channel" :class="getChannelClass(entry.channel)">[{{ getChannelLabel(entry.channel) }}]</span>
                <span class="timestamp">{{ formatDate(entry.timestamp) }}</span>
              </div>

              <!-- 普通文本内容 -->
              <div v-if="entry.type !== 'image'" class="entry-text" :style="getChannelTextColor(entry.channel) ? { color: getChannelTextColor(entry.channel) } : {}">{{ entry.content }}</div>

              <!-- 图片内容 -->
              <div v-else-if="parseImageEntry(entry)" class="entry-image-content">
                <div class="entry-image-wrapper" @click="openImageViewer(entry.id)" title="查看图像">
                  <img :src="parseImageEntry(entry)!.image" alt="剧情图片" class="entry-image" />
                  <div class="entry-image-hover">
                    <i class="ri-zoom-in-line"></i>
                    <span>查看图像</span>
                  </div>
                </div>
                <p v-if="parseImageEntry(entry)!.description" class="image-description">
                  {{ parseImageEntry(entry)!.description }}
                </p>
              </div>
            </div>
            <div v-if="canEdit" class="entry-actions">
              <button class="entry-action-btn" @click="insertEntryBefore(entry)" title="在此之前插入">
                <i class="ri-arrow-up-line"></i>
              </button>
              <button class="entry-action-btn" @click="insertEntryAfter(entry)" title="在此之后插入">
                <i class="ri-arrow-down-line"></i>
              </button>
              <button class="entry-action-btn" @click="startEditEntry(entry)" title="编辑">
                <i class="ri-edit-line"></i>
              </button>
              <button class="entry-action-btn delete" @click="handleDeleteEntry(entry)" title="删除">
                <i class="ri-delete-bin-line"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 添加条目对话框 -->
    <RModal v-model="showAddModal" title="添加条目" width="500px">
      <div class="add-form">
        <div class="form-row">
          <div class="form-field">
            <label>类型</label>
            <select v-model="newEntryType">
              <option value="dialogue">对话</option>
              <option value="narration">旁白</option>
              <option value="image">图片</option>
            </select>
          </div>
          <div v-if="newEntryType !== 'image'" class="form-field">
            <label>频道</label>
            <select v-model="newEntryChannel">
              <option value="SAY">说</option>
              <option value="YELL">喊</option>
              <option value="WHISPER">密语</option>
              <option value="EMOTE">表情</option>
              <option value="PARTY">小队</option>
              <option value="RAID">团队</option>
            </select>
          </div>
        </div>
        <div class="form-field">
          <label>时间</label>
          <input
            type="datetime-local"
            v-model="newEntryTimestamp"
            step="1"
            class="datetime-input"
          />
          <span class="field-hint">留空则使用当前时间，精确到秒</span>
        </div>
        <div v-if="newEntryType === 'image'" class="form-field">
          <label>相对定位</label>
          <input
            type="number"
            v-model.number="newEntryImageOffset"
            min="0"
            max="999"
            step="1"
            class="datetime-input"
          />
          <span class="field-hint">同一秒内排序用，0-999，数值越大越靠后</span>
        </div>
        <div v-if="newEntryType !== 'image'" class="form-field">
          <label>选择人物卡</label>
          <select
            :value="newEntryCharacterId"
            @change="handleCharacterSelect(($event.target as HTMLSelectElement).value ? Number(($event.target as HTMLSelectElement).value) : null)"
            class="character-select"
          >
            <option :value="null">-- 不关联人物卡 --</option>
            <option v-for="char in availableCharacters" :key="char.id" :value="char.id">
              {{ getCharacterDisplayName(char) }}
              <template v-if="char.is_npc"> (NPC)</template>
            </option>
          </select>
          <span class="field-hint">选择后自动填充说话者名称</span>
        </div>
        <div v-if="newEntryType !== 'image'" class="form-field">
          <label>说话者</label>
          <RInput v-model="newEntrySpeaker" placeholder="角色名称" />
        </div>
        <div v-if="newEntryType !== 'image'" class="form-field">
          <label>内容</label>
          <RichEditor v-model="newEntryContent" placeholder="输入内容..." min-height="120px" simple />
        </div>

        <!-- 图片上传 -->
        <div v-if="newEntryType === 'image'" class="form-field">
          <label>选择图片</label>
          <input
            type="file"
            accept="image/*"
            @change="handleImageSelect"
            class="image-input"
          />
          <span class="field-hint">支持 JPG、PNG、GIF 等格式，最大 10MB</span>
        </div>
        <div v-if="newEntryType === 'image' && newEntryImagePreview" class="form-field">
          <label>图片预览</label>
          <div class="image-preview-box">
            <img :src="newEntryImagePreview" alt="预览" class="image-preview" />
            <button type="button" class="clear-image-btn" @click="clearImage">
              <i class="ri-close-line"></i>
            </button>
          </div>
        </div>
        <div v-if="newEntryType === 'image'" class="form-field">
          <label>图片描述</label>
          <RInput v-model="newEntryImageDesc" placeholder="为图片添加描述（可选）" />
        </div>
      </div>
      <template #footer>
        <RButton @click="showAddModal = false">取消</RButton>
        <RButton type="primary" :loading="adding" @click="handleAddEntry">添加</RButton>
      </template>
    </RModal>

    <!-- 分享对话框 -->
    <RModal v-model="showShareModal" title="分享剧情" width="450px">
      <div class="share-content">
        <p class="share-tip">剧情已公开，任何人都可以通过以下链接查看</p>
        <div class="share-link-box">
          <input type="text" :value="shareUrl" readonly class="share-link-input" />
          <RButton type="primary" @click="copyShareLink">复制链接</RButton>
        </div>
        <div class="share-stats">
          <span><i class="ri-eye-line"></i> {{ story?.view_count || 0 }} 次浏览</span>
        </div>
      </div>
      <template #footer>
        <RButton @click="showShareModal = false">关闭</RButton>
      </template>
    </RModal>

    <!-- 角色信息卡片 -->
    <CharacterCard
      v-model:visible="showCharacterModal"
      :character="selectedCharacter ? getEntryCharacter(selectedCharacter) : undefined"
      :speaker="selectedCharacter ? getEntrySpeakerName(selectedCharacter) : undefined"
      :position="characterCardPosition"
      @edit="handleEditCharacter"
    />

    <!-- 编辑角色对话框 -->
    <RModal v-model="showEditModal" title="编辑角色信息" width="560px" :mask-closable="false">
      <div v-if="editingCharacter" class="edit-character-form">
        <div class="form-field">
          <label>自定义头像</label>
          <RAvatarPicker
            v-model="editingCharacter.custom_avatar"
            :fallback-icon="editingCharacter.first_name?.charAt(0) || '?'"
          />
        </div>

        <div class="form-row">
          <div class="form-field">
            <label>显示名称</label>
            <RInput v-model="editingCharacter.custom_name" :placeholder="editingCharacter.first_name || ''" />
          </div>
          <div class="form-field">
            <label>名字颜色</label>
            <RColorPicker
              v-model="editingCharacter.custom_color"
              :presets="wowColorPresets"
            />
          </div>
        </div>

        <div class="form-row">
          <div class="form-field">
            <label>头衔</label>
            <RInput v-model="editingCharacter.title" placeholder="角色头衔" />
          </div>
          <div class="form-field">
            <label>全名头衔</label>
            <RInput v-model="editingCharacter.full_title" placeholder="完整头衔" />
          </div>
        </div>

        <div class="form-row">
          <div class="form-field">
            <label>种族</label>
            <RInput v-model="editingCharacter.race" placeholder="种族" />
          </div>
          <div class="form-field">
            <label>职业</label>
            <RInput v-model="editingCharacter.class" placeholder="职业" />
          </div>
        </div>

        <div class="form-row">
          <div class="form-field">
            <label>年龄</label>
            <RInput v-model="editingCharacter.age" placeholder="年龄" />
          </div>
          <div class="form-field">
            <label>身高</label>
            <RInput v-model="editingCharacter.height" placeholder="身高" />
          </div>
        </div>

        <div class="form-row">
          <div class="form-field">
            <label>眼睛颜色</label>
            <RInput v-model="editingCharacter.eye_color" placeholder="眼睛颜色" />
          </div>
          <div class="form-field">
            <label>居住地</label>
            <RInput v-model="editingCharacter.residence" placeholder="居住地" />
          </div>
        </div>

        <div class="form-field">
          <label>出生地</label>
          <RInput v-model="editingCharacter.birthplace" placeholder="出生地" />
        </div>

        <div class="original-info">
          <h4>原始TRP3信息</h4>
          <div class="info-list">
            <div v-if="editingCharacter.first_name" class="info-row">
              <span class="label">名字:</span>
              <span>{{ editingCharacter.first_name }} {{ editingCharacter.last_name }}</span>
            </div>
            <div v-if="editingCharacter.game_id" class="info-row">
              <span class="label">游戏ID:</span>
              <span>{{ editingCharacter.game_id }}</span>
            </div>
            <div v-if="editingCharacter.ref_id" class="info-row">
              <span class="label">TRP3 ID:</span>
              <span class="ref-id">{{ editingCharacter.ref_id }}</span>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <RButton @click="showEditModal = false">取消</RButton>
        <RButton type="primary" :loading="savingCharacter" @click="saveCharacterEdit">保存</RButton>
      </template>
    </RModal>

    <!-- 标签选择对话框 -->
    <RModal v-model="showTagModal" title="管理标签" width="500px">
      <TagSelector
        :selected-tags="storyTags"
        :all-tags="allTags"
        @add="handleAddTag"
        @remove="handleRemoveTag"
        @create="(tag) => allTags.push(tag)"
      />
      <template #footer>
        <RButton @click="showTagModal = false">关闭</RButton>
      </template>
    </RModal>

    <!-- 公会归档对话框 -->
    <RModal v-model="showGuildModal" title="归档到公会" width="500px">
      <div class="guild-archive-content">
        <p v-if="availableGuilds.length === 0" class="empty-tip">
          暂无可归档的公会，请先加入公会
        </p>
        <div v-else class="guild-list">
          <div
            v-for="guild in availableGuilds"
            :key="guild.id"
            class="guild-item"
            @click="handleArchiveToGuild(guild.id)"
          >
            <div class="guild-info">
              <div class="guild-name" :style="{ color: `#${guild.color || 'B87333'}` }">
                {{ guild.name }}
              </div>
              <div class="guild-desc">{{ guild.description || '暂无描述' }}</div>
            </div>
            <i class="ri-arrow-right-line"></i>
          </div>
        </div>
      </div>
      <template #footer>
        <RButton @click="showGuildModal = false">取消</RButton>
      </template>
    </RModal>

    <!-- 导入对话框 -->
    <RModal v-model="showImportModal" title="导入剧情数据" width="480px">
      <div class="import-content">
        <p class="import-tip">
          选择一个 JSON 文件导入剧情条目。导入的条目将追加到当前剧情中。
        </p>
        <div class="file-input-wrapper">
          <input
            type="file"
            accept=".json"
            @change="handleImportFileChange"
            class="file-input"
          />
          <div class="file-input-display">
            <i class="ri-file-upload-line"></i>
            <span v-if="importFile">{{ importFile.name }}</span>
            <span v-else>点击选择文件或拖拽文件到此处</span>
          </div>
        </div>
      </div>
      <template #footer>
        <RButton @click="showImportModal = false; importFile = null">取消</RButton>
        <RButton
          type="primary"
          :loading="importing"
          :disabled="!importFile"
          @click="importStoryFromFile"
        >
          导入
        </RButton>
      </template>
    </RModal>

    <!-- 编辑条目对话框 -->
    <RModal v-model="showEditEntryModal" title="编辑条目" width="500px" :mask-closable="false">
      <div class="add-form">
        <div class="form-row">
          <div class="form-field">
            <label>类型</label>
            <select v-model="editEntryType">
              <option value="dialogue">对话</option>
              <option value="narration">旁白</option>
              <option value="image">图片</option>
            </select>
          </div>
          <div v-if="editEntryType !== 'image'" class="form-field">
            <label>频道</label>
            <select v-model="editEntryChannel">
              <option value="SAY">说</option>
              <option value="YELL">喊</option>
              <option value="WHISPER">密语</option>
              <option value="EMOTE">表情</option>
              <option value="PARTY">小队</option>
              <option value="RAID">团队</option>
            </select>
          </div>
        </div>
        <div v-if="editEntryType !== 'image'" class="form-field">
          <label>选择人物卡</label>
          <select
            :value="editEntryCharacterId"
            @change="handleEditCharacterSelect(($event.target as HTMLSelectElement).value ? Number(($event.target as HTMLSelectElement).value) : null)"
            class="character-select"
          >
            <option :value="null">-- 不关联人物卡 --</option>
            <option v-for="char in availableCharacters" :key="char.id" :value="char.id">
              {{ getCharacterDisplayName(char) }}
              <template v-if="char.is_npc"> (NPC)</template>
            </option>
          </select>
          <span class="field-hint">选择后自动填充说话者名称</span>
        </div>
        <div v-if="editEntryType !== 'image'" class="form-field">
          <label>说话者</label>
          <RInput v-model="editEntrySpeaker" placeholder="角色名称" />
        </div>
        <div class="form-field">
          <label>记录时间</label>
          <input type="datetime-local" v-model="editEntryTimestamp" step="1" class="datetime-input" />
          <span class="field-hint">修改时间会影响条目排序顺序</span>
        </div>
        <div v-if="editEntryType === 'image'" class="form-field">
          <label>相对定位</label>
          <input
            type="number"
            v-model.number="editEntryImageOffset"
            min="0"
            max="999"
            step="1"
            class="datetime-input"
          />
          <span class="field-hint">同一秒内排序用，0-999，数值越大越靠后</span>
        </div>
        <div v-if="editEntryType !== 'image'" class="form-field">
          <label>内容</label>
          <RichEditor v-model="editEntryContent" placeholder="输入内容..." min-height="120px" simple />
        </div>
      </div>
      <template #footer>
        <RButton @click="showEditEntryModal = false">取消</RButton>
        <RButton type="primary" :loading="savingEntry" @click="saveEntryEdit">保存</RButton>
      </template>
    </RModal>

    <ImageViewer
      v-model="showImageViewer"
      :images="viewerImages"
      :start-index="viewerStartIndex"
    />
  </div>
</template>

<style scoped>
.story-detail {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.back-bar {
  margin-bottom: 8px;
}

.btn-back {
  background: transparent;
  border: none;
  color: var(--color-secondary);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 0;
}

.btn-back:hover {
  color: var(--color-primary);
}

.loading-state {
  text-align: center;
  padding: 60px;
  color: var(--color-secondary);
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.story-header {
  padding: 24px;
  position: relative;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(229, 212, 193, 0.6);
  border-radius: 16px;
  box-shadow: 0 4px 20px rgba(44, 24, 16, 0.08);
}

@media (min-width: 1024px) {
  .story-header {
    padding: 32px;
  }
}

/* 装饰性圆角元素 */
.story-header::before {
  content: '';
  position: absolute;
  top: -32px;
  right: -32px;
  width: 128px;
  height: 128px;
  background: rgba(184, 115, 51, 0.05);
  border-radius: 0 0 0 100px;
  pointer-events: none;
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

@media (min-width: 1024px) {
  .header-content {
    flex-direction: row;
    justify-content: space-between;
    align-items: flex-start;
  }
}

.header-info h1 {
  font-size: 24px;
  color: var(--color-primary);
  margin: 0 0 8px 0;
}

.header-info .description {
  font-size: 14px;
  color: var(--color-secondary);
  margin: 0 0 12px 0;
  line-height: 1.6;
}

.meta {
  display: flex;
  gap: 16px;
  align-items: center;
}

.meta-item {
  font-size: 13px;
  color: var(--color-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}

.meta-item.status {
  padding: 2px 8px;
  border-radius: 4px;
}

.meta-item.status.draft {
  background: var(--color-bg-secondary);
}

.meta-item.status.published {
  background: rgba(40, 167, 69, 0.1);
  color: #28a745;
}

.header-actions {
  display: flex;
  flex-direction: row; /* 小屏幕水平排列 */
  justify-content: flex-start;
  gap: 12px;
  flex-shrink: 0;
  border-top: 1px solid rgba(229, 212, 193, 0.5);
  padding-top: 24px;
}

/* 大屏幕垂直排列 */
@media (min-width: 1024px) {
  .header-actions {
    flex-direction: column;
    align-items: flex-end;
    border-top: none;
    border-left: 1px solid rgba(229, 212, 193, 0.5);
    padding-top: 0;
    padding-left: 32px;
    min-width: 200px;
  }
}

/* Action buttons - 按照 story_info.html 设计 */
.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 500;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid;
  gap: 8px;
}

@media (min-width: 1024px) {
  .action-btn {
    width: 100%;
    min-width: 180px;
  }
}

.action-btn--primary {
  background: #2C1810;
  border-color: #2C1810;
  color: #fff;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.action-btn--primary:hover {
  background: #1a0e09;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
  transform: scale(0.98);
}

.action-btn--primary:active {
  transform: scale(0.95);
}

.action-btn--primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.action-btn--secondary {
  background: #F5EFE7;
  border-color: #E5D4C1;
  color: #2C1810;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.action-btn--secondary:hover {
  background: #fff;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

/* Icon button group */
.icon-button-group {
  display: flex;
  gap: 8px;
  width: 100%;
}

@media (min-width: 1024px) {
  .icon-button-group {
    width: auto;
  }
}

.icon-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px 12px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #8D7B68;
  cursor: pointer;
  transition: all 0.2s;
}

.icon-btn:hover {
  background: #fff;
  color: #B87333;
}

.icon-btn i {
  font-size: 18px;
}

@media (min-width: 1024px) {
  .icon-btn {
    flex: initial;
    min-width: 44px;
  }
}

.edit-form, .add-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-field label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary);
}

.form-field textarea,
.form-field select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  background: #fff;
  color: var(--color-primary);
}

.form-field textarea:focus,
.form-field select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.datetime-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  background: #fff;
  color: var(--color-primary);
}

.datetime-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.field-hint {
  font-size: 12px;
  color: var(--color-secondary);
  margin-top: 4px;
}

.character-select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  background: #fff;
  color: var(--color-primary);
  cursor: pointer;
}

.character-select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}

.entries-section {
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(229, 212, 193, 0.6);
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 4px 20px rgba(44, 24, 16, 0.08);
  min-height: 400px;
}

@media (min-width: 1024px) {
  .entries-section {
    padding: 32px;
  }
}

.entries-section h2 {
  font-size: 18px;
  color: var(--color-primary);
  margin: 0 0 20px 0;
}

.empty-entries {
  text-align: center;
  padding: 40px;
  color: var(--color-secondary);
}

.entries-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.entry-item {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: #F5EFE7;
  border-radius: 8px;
  border: 1px solid transparent;
  position: relative;
  transition: all 0.2s;
}

.entry-item:hover {
  border-color: rgba(184, 115, 51, 0.3);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.entry-item.image {
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.entry-item.image .entry-content {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.entry-item.image .entry-header {
  justify-content: center;
}

.entry-item.image .entry-header .timestamp {
  margin-left: 0;
}

.entry-item.image .entry-image-content {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.entry-avatar {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: var(--color-accent);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  flex-shrink: 0;
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.entry-avatar :deep(.wow-icon) {
  width: 100%;
  height: 100%;
  border-radius: 0;
}

.entry-avatar :deep(.wow-icon img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.entry-content {
  flex: 1;
  min-width: 0;
}

.entry-header {
  display: flex;
  align-items: center;
  margin-bottom: 6px;
  gap: 6px;
}

.entry-header .speaker {
  font-weight: 600;
  color: var(--color-primary);
}

.entry-header .timestamp {
  font-size: 12px;
  color: var(--color-secondary);
  margin-left: auto;
}

.entry-text {
  font-size: 14px;
  color: var(--color-text);
  line-height: 1.6;
  white-space: pre-wrap;
}

/* 图片条目样式 */
.entry-image-content {
  margin-top: 8px;
}

.entry-image-wrapper {
  position: relative;
  display: inline-flex;
  max-width: 100%;
  cursor: zoom-in;
  margin-bottom: 8px;
  border-radius: 8px;
  overflow: hidden;
}

.entry-image-hover {
  position: absolute;
  inset: 0;
  background: rgba(44, 24, 16, 0.45);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  opacity: 0;
  transition: opacity 0.2s ease;
  border-radius: 8px;
  pointer-events: none;
  font-size: 13px;
}

.entry-image-wrapper:hover .entry-image-hover {
  opacity: 1;
}

.entry-image {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
  border: 2px solid #e5d4c1;
  display: block;
}

.image-description {
  font-size: 14px;
  color: #665242;
  line-height: 1.6;
  margin: 0;
  padding: 8px 12px;
  background: #f5f0eb;
  border-radius: 6px;
  border-left: 3px solid #d4a373;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.entry-item.narration {
  background: linear-gradient(to right, rgba(184, 115, 51, 0.08), rgba(184, 115, 51, 0.02));
  border: none;
  border-left: 3px solid #B87333;
}

.entry-item.narration .entry-avatar {
  background: #a98467;
}

.share-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.share-tip {
  color: var(--color-secondary);
  font-size: 14px;
  margin: 0;
}

.share-link-box {
  display: flex;
  gap: 8px;
}

.share-link-input {
  flex: 1;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  font-size: 14px;
  background: var(--color-bg-secondary);
  color: var(--color-primary);
}

.share-stats {
  font-size: 13px;
  color: var(--color-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 头像可点击样式 */
.entry-avatar.clickable {
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.entry-avatar.clickable:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.avatar-fallback {
  font-size: 16px;
  font-weight: 600;
}

.avatar-npc {
  font-size: 12px;
  font-weight: 600;
  color: #666;
  background: #e0e0e0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.avatar-narration {
  font-size: 10px;
  font-weight: 600;
  color: #666;
  background: #e0e0e0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

/* 频道标签 */
.entry-header .channel {
  font-size: 12px;
  color: var(--color-accent);
}

.entry-header .channel.channel-yell {
  color: #e74c3c;
  font-weight: bold;
}

.entry-header .channel.channel-whisper {
  color: #b39ddb;
}

/* 角色信息弹窗 */
.character-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 20px 0;
}

.character-avatar {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-bg-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-large {
  font-size: 32px;
  font-weight: 600;
  color: var(--color-secondary);
}

.character-name {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-primary);
}

.character-channel {
  font-size: 14px;
  color: var(--color-secondary);
}

.character-title {
  font-size: 14px;
  color: var(--color-accent);
  font-style: italic;
}

.character-race-class {
  font-size: 13px;
  color: var(--color-secondary);
}

/* 编辑角色表单 */
.edit-character-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.original-info {
  background: var(--color-bg-secondary);
  border-radius: 8px;
  padding: 12px 16px;
}

.original-info h4 {
  font-size: 13px;
  color: var(--color-secondary);
  margin: 0 0 8px 0;
  font-weight: 500;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-row {
  font-size: 13px;
  display: flex;
  gap: 8px;
}

.info-row .label {
  color: var(--color-secondary);
  min-width: 60px;
}

.info-row .ref-id {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-secondary);
  word-break: break-all;
}

/* 标签区域 */
.tags-section {
  margin-top: 12px;
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.tag-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 500;
  cursor: default;
  transition: opacity 0.2s;
}

.tag-item i {
  cursor: pointer;
  font-size: 14px;
  opacity: 0.6;
  transition: opacity 0.2s;
}

.tag-item i:hover {
  opacity: 1;
}

.add-tag-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px dashed var(--color-border);
  border-radius: 12px;
  background: transparent;
  color: var(--color-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.add-tag-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: rgba(184, 115, 51, 0.05);
}

/* 公会归档区域 */
.guilds-section {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.guilds-label {
  font-size: 13px;
  color: var(--color-secondary);
}

.guilds-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.guild-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1.5px solid;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 500;
  background: rgba(184, 115, 51, 0.05);
  cursor: default;
}

.guild-badge i {
  cursor: pointer;
  font-size: 14px;
  opacity: 0.6;
  transition: opacity 0.2s;
}

.guild-badge i:hover {
  opacity: 1;
}

/* 公会归档对话框 */
.guild-archive-content {
  min-height: 200px;
}

.empty-tip {
  text-align: center;
  padding: 60px 20px;
  color: var(--color-secondary);
  font-size: 14px;
}

.guild-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.guild-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.guild-item:hover {
  border-color: var(--color-accent);
  background: rgba(184, 115, 51, 0.05);
}

.guild-info {
  flex: 1;
}

.guild-name {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}

.guild-desc {
  font-size: 13px;
  color: var(--color-secondary);
}

.guild-item i {
  color: var(--color-secondary);
  font-size: 18px;
}

/* 导入对话框 */
.import-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.import-tip {
  font-size: 14px;
  color: var(--color-secondary);
  margin: 0;
  line-height: 1.6;
}

.file-input-wrapper {
  position: relative;
}

.file-input {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  opacity: 0;
  cursor: pointer;
}

.file-input-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 32px;
  border: 2px dashed var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-secondary);
  transition: all 0.2s;
}

.file-input-wrapper:hover .file-input-display {
  border-color: var(--color-accent);
  background: rgba(184, 115, 51, 0.05);
}

.file-input-display i {
  font-size: 32px;
  color: var(--color-accent);
}

.file-input-display span {
  font-size: 14px;
  color: var(--color-secondary);
}

/* 条目操作按钮 */
.entry-item {
  position: relative;
}

.entry-actions {
  position: absolute;
  right: 12px;
  bottom: 12px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.entry-item:hover .entry-actions {
  opacity: 1;
}

.entry-action-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  backdrop-filter: blur(4px);
}

.entry-action-btn:hover {
  background: #B87333;
}

.entry-action-btn.delete:hover {
  background: #e74c3c;
}

/* 图片上传相关样式 */
.image-input {
  width: 100%;
  padding: 10px;
  border: 2px dashed #d4a373;
  border-radius: 8px;
  background: #f5f0eb;
  cursor: pointer;
  transition: all 0.2s;
}

.image-input:hover {
  border-color: #b87333;
  background: #ebe4dc;
}

.image-preview-box {
  position: relative;
  width: 100%;
  max-width: 400px;
  border-radius: 8px;
  overflow: hidden;
  border: 2px solid #e5d4c1;
}

.image-preview {
  width: 100%;
  height: auto;
  display: block;
}

.clear-image-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  transition: all 0.2s;
  backdrop-filter: blur(4px);
}

.clear-image-btn:hover {
  background: #e74c3c;
  transform: scale(1.1);
}
</style>
