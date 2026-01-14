<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getStory, updateStory, addStoryEntries, publishStory, type Story, type StoryEntry } from '@/api/story'
import { getCharacter, updateCharacter, type Character } from '@/api/character'
import RButton from '@/components/RButton.vue'
import RCard from '@/components/RCard.vue'
import RInput from '@/components/RInput.vue'
import RModal from '@/components/RModal.vue'
import WowIcon from '@/components/WowIcon.vue'
import RichEditor from '@/components/RichEditor.vue'
import CharacterCard from '@/components/CharacterCard.vue'
import RColorPicker from '@/components/RColorPicker.vue'
import RAvatarPicker from '@/components/RAvatarPicker.vue'

const route = useRoute()
const router = useRouter()

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
const adding = ref(false)

// 发布分享
const publishing = ref(false)
const showShareModal = ref(false)

// 角色信息弹窗
const showCharacterModal = ref(false)
const selectedCharacter = ref<StoryEntry | null>(null)
const characterCardPosition = ref({ x: 0, y: 0 })

// 角色数据缓存 (character_id -> Character)
const charactersMap = ref<Map<number, Character>>(new Map())

const storyId = computed(() => Number(route.params.id))

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

async function handleAddEntry() {
  if (!newEntryContent.value.trim()) return
  adding.value = true
  try {
    await addStoryEntries(storyId.value, [{
      content: newEntryContent.value,
      speaker: newEntrySpeaker.value,
      type: newEntryType.value,
    }])
    showAddModal.value = false
    newEntryContent.value = ''
    newEntrySpeaker.value = ''
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

function goBack() {
  router.push({ name: 'archives' })
}

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
  if (entry.type === 'narration') return
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
    'SAY': '说',
    'YELL': '喊',
    'EMOTE': '表情',
    'PARTY': '小队',
    'RAID': '团队',
    'WHISPER': '密语',
  }
  return map[channel] || channel
}

function getChannelClass(channel: string): string {
  if (channel === 'YELL') return 'channel-yell'
  if (channel === 'WHISPER') return 'channel-whisper'
  return ''
}

// 获取频道对应的文字颜色
function getChannelTextColor(channel: string): string {
  const colorMap: Record<string, string> = {
    'SAY': '',  // 默认颜色
    'YELL': '#FF3333',  // 红色
    'WHISPER': '#B39DDB',  // 紫色
    'EMOTE': '#FF8C00',  // 橙色
    'PARTY': '#AAAAFF',  // 蓝色
    'RAID': '#FF7F00',  // 橙色
  }
  return colorMap[channel] || ''
}

onMounted(loadStory)
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
                <span class="meta-item status" :class="story.status">
                  {{ story.status === 'published' ? '已发布' : '草稿' }}
                </span>
              </div>
            </div>
            <div class="header-actions">
              <RButton @click="startEdit">编辑</RButton>
              <RButton @click="showAddModal = true">添加条目</RButton>
              <RButton
                :type="story.is_public ? 'default' : 'primary'"
                :loading="publishing"
                @click="togglePublish"
              >
                {{ story.is_public ? '取消公开' : '公开分享' }}
              </RButton>
              <RButton v-if="story.is_public" @click="showShareModal = true">
                <i class="ri-share-line"></i> 分享
              </RButton>
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
              class="entry-avatar"
              :class="{ clickable: entry.type !== 'narration' }"
              @click="showCharacterInfo(entry, $event)"
            >
              <span v-if="isNpcEntry(entry)" class="avatar-npc">NPC</span>
              <span v-else-if="isNarrationEntry(entry)" class="avatar-narration">旁白</span>
              <WowIcon v-else-if="getEntryIcon(entry)" :icon="getEntryIcon(entry)" :size="40" :fallback="entry.speaker?.charAt(0) || '?'" />
              <span v-else class="avatar-fallback">{{ entry.speaker?.charAt(0) || '?' }}</span>
            </div>
            <div class="entry-content">
              <div class="entry-header">
                <span class="speaker" :style="getEntryColor(entry) ? { color: '#' + getEntryColor(entry) } : {}">{{ entry.speaker || '旁白' }}</span>
                <span v-if="entry.channel" class="channel" :class="getChannelClass(entry.channel)">[{{ getChannelLabel(entry.channel) }}]</span>
                <span class="timestamp">{{ formatDate(entry.timestamp) }}</span>
              </div>
              <div class="entry-text" :style="getChannelTextColor(entry.channel) ? { color: getChannelTextColor(entry.channel) } : {}">{{ entry.content }}</div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 添加条目对话框 -->
    <RModal v-model="showAddModal" title="添加条目" width="500px">
      <div class="add-form">
        <div class="form-field">
          <label>类型</label>
          <select v-model="newEntryType">
            <option value="dialogue">对话</option>
            <option value="narration">旁白</option>
          </select>
        </div>
        <div class="form-field">
          <label>说话者</label>
          <RInput v-model="newEntrySpeaker" placeholder="角色名称" />
        </div>
        <div class="form-field">
          <label>内容</label>
          <RichEditor v-model="newEntryContent" placeholder="输入内容..." min-height="120px" />
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
      :speaker="selectedCharacter?.speaker"
      :position="characterCardPosition"
      @edit="handleEditCharacter"
    />

    <!-- 编辑角色对话框 -->
    <RModal v-model="showEditModal" title="编辑角色信息（本剧情）" width="480px">
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
            <label>自定义显示名</label>
            <RInput v-model="editingCharacter.custom_name" :placeholder="editingCharacter.first_name || ''" />
          </div>
          <div class="form-field">
            <label>自定义颜色</label>
            <RColorPicker v-model="editingCharacter.custom_color" />
          </div>
        </div>

        <div class="original-info">
          <h4>原始TRP3信息</h4>
          <div class="info-list">
            <div v-if="editingCharacter.first_name" class="info-row">
              <span class="label">名字:</span>
              <span>{{ editingCharacter.first_name }} {{ editingCharacter.last_name }}</span>
            </div>
            <div v-if="editingCharacter.title" class="info-row">
              <span class="label">头衔:</span>
              <span>{{ editingCharacter.title }}</span>
            </div>
            <div v-if="editingCharacter.race" class="info-row">
              <span class="label">种族:</span>
              <span>{{ editingCharacter.race }}</span>
            </div>
            <div v-if="editingCharacter.class" class="info-row">
              <span class="label">职业:</span>
              <span>{{ editingCharacter.class }}</span>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <RButton @click="showEditModal = false">取消</RButton>
        <RButton type="primary" :loading="savingCharacter" @click="saveCharacterEdit">保存</RButton>
      </template>
    </RModal>
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
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
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
  gap: 8px;
  flex-shrink: 0;
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

.edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}

.entries-section {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
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
  gap: 12px;
  padding: 16px;
  background: var(--color-bg-secondary);
  border-radius: 8px;
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

.entry-item.narration {
  background: rgba(184, 115, 51, 0.05);
  border-left: 3px solid var(--color-accent);
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
  min-width: 48px;
}
</style>
