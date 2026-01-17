<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import RTabs from '@/components/RTabs.vue'
import RTabPane from '@/components/RTabPane.vue'
import RModal from '@/components/RModal.vue'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'
import AddonInstaller from '@/components/AddonInstaller.vue'
import AddonUpdateDialog from '@/components/AddonUpdateDialog.vue'
import StagingPool from './StagingPool.vue'
import StoryList from './StoryList.vue'
import { createStory, addStoryEntries, listStories, type CreateStoryEntryRequest, type Story } from '@/api/story'
import { listTags, addStoryTag, type Tag } from '@/api/tag'
import { getAddonManifest } from '@/api/addon'

// ChatRecord 类型定义
interface TRP3Info {
  FN?: string
  LN?: string
  TI?: string
  IC?: string
  CH?: string  // 名字颜色
}

interface Listener {
  gameID: string
  profileID?: string
}

interface ChatRecord {
  timestamp: number
  channel: string
  sender: {
    gameID: string
    trp3?: TRP3Info
  }
  content: string
  mark?: string  // P(Player), N(NPC), B(Background)
  npc?: string   // NPC名字
  nt?: string    // NPC说话类型
  ref_id?: string  // TRP3 profile ref ID
  raw_profile?: string  // 完整的TRP3 profile JSON
  listeners?: Listener[]  // 收听者列表（新增字段，向前兼容）
}

const mounted = ref(false)
const router = useRouter()
const activeTab = ref('staging')
const wowPath = ref(localStorage.getItem('wow_path') || '')

// 创建剧情对话框
const showCreateModal = ref(false)
const newStoryTitle = ref('')
const newStoryDesc = ref('')
const creating = ref(false)
const storyListRef = ref<InstanceType<typeof StoryList> | null>(null)
const stagingPoolRef = ref<InstanceType<typeof StagingPool> | null>(null)

// 标签选择
const allTags = ref<Tag[]>([])
const selectedTagIds = ref<number[]>([])

// 归档模式：create（创建新剧情）或 append（追加到已有剧情）
const archiveMode = ref<'create' | 'append'>('create')
const userStories = ref<Story[]>([])
const selectedStoryId = ref<number | null>(null)
const loadingStories = ref(false)

// 待归档的记录
const pendingRecords = ref<ChatRecord[]>([])

// 插件状态
const showAddonInstaller = ref(false)
const addonInstalled = ref(false)
const selectedFlavor = ref('_retail_')
const addonUpdateDialogRef = ref<InstanceType<typeof AddonUpdateDialog> | null>(null)

async function checkAddonStatus() {
  if (!wowPath.value) return
  try {
    const info = await invoke<{ installed: boolean }>('check_addon_installed', {
      wowPath: wowPath.value,
      flavor: selectedFlavor.value,
    })
    addonInstalled.value = info.installed
  } catch (e) {
    console.error('检测插件失败:', e)
  }
}

// 检查插件更新
async function checkAddonUpdate() {
  try {
    const manifest = await getAddonManifest()
    const latestVersion = manifest.latest

    // 从 localStorage 读取上次检查的版本
    const lastCheckedVersion = localStorage.getItem('addon_last_checked_version')

    // 如果有新版本，显示更新提示
    if (!lastCheckedVersion || lastCheckedVersion !== latestVersion) {
      // 使用上次检查的版本作为"当前版本"，如果没有则使用 "未知"
      const currentVersion = lastCheckedVersion || '未知'

      // 查找最新版本的详细信息（包括 changelog）
      const latestVersionInfo = manifest.versions.find(v => v.version === latestVersion)
      const changelog = latestVersionInfo?.changelog || '暂无更新说明'

      addonUpdateDialogRef.value?.show(currentVersion, latestVersion, changelog)

      // 记录本次检查的版本
      localStorage.setItem('addon_last_checked_version', latestVersion)
    }
  } catch (e) {
    console.error('检查插件更新失败:', e)
  }
}

onMounted(() => {
  // 检查是否已设置魔兽路径，未设置则跳转到设置向导
  const savedPath = localStorage.getItem('wow_path')
  if (!savedPath) {
    router.push('/sync/setup')
    return
  }
  wowPath.value = savedPath
  setTimeout(() => mounted.value = true, 50)
  checkAddonStatus()
  checkAddonUpdate()  // 检查插件更新
  loadTags()
})

// 监听标签页切换，每次打开时检查插件更新
watch(activeTab, (newTab) => {
  if (newTab === 'staging' || newTab === 'stories') {
    checkAddonUpdate()
  }
})

async function loadTags() {
  try {
    const res = await listTags('story')
    allTags.value = res.tags || []
  } catch (e) {
    console.error('加载标签失败:', e)
  }
}

async function loadUserStories() {
  loadingStories.value = true
  try {
    const res = await listStories({ sort: 'updated_at', order: 'desc' })
    userStories.value = res.stories || []
  } catch (e) {
    console.error('加载剧情列表失败:', e)
  } finally {
    loadingStories.value = false
  }
}

// 清理TRP3特殊格式字符
function cleanTRP3Content(content: string): string {
  return content
    .replace(/\{[^}]+\}/g, '') // 移除 {icon:xxx} {col:xxx} 等标记
    .replace(/\|c[0-9a-fA-F]{8}/g, '') // 移除颜色开始标记 |cFFFFFFFF
    .replace(/\|r/g, '') // 移除颜色结束标记 |r
    .replace(/\|T[^|]+\|t/g, '') // 移除纹理标记 |Txxx|t
    .replace(/\|H[^|]+\|h/g, '') // 移除超链接标记
    .replace(/\|h/g, '')
    .replace(/[\uE000-\uF8FF]/g, '') // 移除私用区Unicode字符
    .replace(/\uFFFD/g, '') // 移除替换字符 �
    .replace(/[\u0000-\u001F]/g, '') // 移除控制字符
    .trim()
}

async function handleArchive(records: ChatRecord[]) {
  pendingRecords.value = records
  archiveMode.value = 'create'
  selectedStoryId.value = null
  showCreateModal.value = true
  // 加载用户剧情列表供追加选择
  loadUserStories()
}

// 将待归档记录转换为条目请求
function buildEntriesFromRecords(records: ChatRecord[]): CreateStoryEntryRequest[] {
  return records.map(record => {
    const trp3 = record.sender.trp3
    let speaker: string
    let type: string = 'dialogue'
    let channel: string = record.channel
    let isNpc: boolean = false

    if (record.mark === 'N' && record.npc) {
      speaker = record.npc
      isNpc = true
      if (record.nt) {
        channel = record.nt.toUpperCase()
      }
    } else if (record.mark === 'B' || (record.mark === 'N' && !record.npc)) {
      speaker = ''
      type = 'narration'
    } else {
      speaker = trp3?.FN
        ? (trp3.LN ? `${trp3.FN}·${trp3.LN}` : trp3.FN)
        : record.sender.gameID.split('-')[0]
    }

    return {
      source_id: `chat_${record.timestamp}`,
      type: type,
      speaker: speaker,
      content: cleanTRP3Content(record.content),
      channel: channel,
      timestamp: new Date(record.timestamp * 1000).toISOString(),
      ref_id: record.ref_id,
      game_id: record.sender.gameID,
      trp3_data: record.raw_profile,
      is_npc: isNpc,
    }
  })
}

async function handleCreateStory() {
  // 创建模式需要标题，追加模式需要选择剧情
  if (archiveMode.value === 'create' && !newStoryTitle.value.trim()) return
  if (archiveMode.value === 'append' && !selectedStoryId.value) return

  creating.value = true
  try {
    let storyId: number

    if (archiveMode.value === 'create') {
      // 创建新剧情
      const story = await createStory({
        title: newStoryTitle.value,
        description: newStoryDesc.value,
      })
      storyId = story.id

      // 添加选中的标签
      if (selectedTagIds.value.length > 0) {
        for (const tagId of selectedTagIds.value) {
          try {
            await addStoryTag(storyId, tagId)
          } catch (e) {
            console.error('添加标签失败:', e)
          }
        }
      }
    } else {
      // 追加到已有剧情
      storyId = selectedStoryId.value!
    }

    // 添加待归档记录到剧情
    if (pendingRecords.value.length > 0) {
      const entries = buildEntriesFromRecords(pendingRecords.value)
      await addStoryEntries(storyId, entries)

      // 从待归档池移除已归档的记录
      const archivedTimestamps = pendingRecords.value.map(r => r.timestamp)
      stagingPoolRef.value?.removeArchivedRecords?.(archivedTimestamps)
      pendingRecords.value = []
    }

    // 重置表单
    showCreateModal.value = false
    newStoryTitle.value = ''
    newStoryDesc.value = ''
    selectedTagIds.value = []
    selectedStoryId.value = null
    archiveMode.value = 'create'
    activeTab.value = 'stories'

    // 刷新剧情列表
    storyListRef.value?.loadStories?.()
  } catch (e) {
    console.error('归档失败:', e)
  } finally {
    creating.value = false
  }
}

function handleViewStory(id: number) {
  router.push({ name: 'story-detail', params: { id } })
}
</script>

<template>
  <div class="archives-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar anim-item" style="--delay: 0">
      <div class="page-title">
        <h1>剧情记录</h1>
        <p>记录每一个精彩瞬间，编织属于你的冒险史诗</p>
      </div>
      <button class="btn-create" @click="showCreateModal = true">
        <i class="ri-add-line"></i> 新建剧情
      </button>
    </div>

    <!-- 插件状态提示 -->
    <div v-if="wowPath && !addonInstalled" class="addon-notice anim-item" style="--delay: 0.5">
      <i class="ri-plug-line"></i>
      <span>需要安装 RPBox 插件才能采集聊天记录</span>
      <RButton size="small" type="primary" @click="showAddonInstaller = true">
        安装插件
      </RButton>
    </div>

    <!-- Tab切换 -->
    <RTabs v-model="activeTab" class="anim-item" style="--delay: 1">
      <RTabPane name="staging" label="待归档池">
        <StagingPool ref="stagingPoolRef" @archive="handleArchive" />
      </RTabPane>
      <RTabPane name="stories" label="我的剧情">
        <StoryList ref="storyListRef" @create="showCreateModal = true" @view="handleViewStory" />
      </RTabPane>
    </RTabs>

    <!-- 创建/追加剧情对话框 -->
    <RModal v-model="showCreateModal" :title="pendingRecords.length > 0 ? '归档对话记录' : '新建剧情'" width="480px">
      <div class="create-form">
        <!-- 模式切换（仅在有待归档记录时显示） -->
        <div v-if="pendingRecords.length > 0" class="mode-switcher">
          <button
            class="mode-btn"
            :class="{ active: archiveMode === 'create' }"
            @click="archiveMode = 'create'"
          >
            <i class="ri-add-line"></i> 创建新剧情
          </button>
          <button
            class="mode-btn"
            :class="{ active: archiveMode === 'append' }"
            @click="archiveMode = 'append'"
          >
            <i class="ri-file-add-line"></i> 追加到已有剧情
          </button>
        </div>

        <!-- 创建模式：显示标题、描述、标签 -->
        <template v-if="archiveMode === 'create'">
          <div class="form-field">
            <label>剧情标题</label>
            <RInput v-model="newStoryTitle" placeholder="输入剧情标题" />
          </div>
          <div class="form-field">
            <label>剧情描述</label>
            <textarea v-model="newStoryDesc" placeholder="简要描述这个剧情..." rows="3"></textarea>
          </div>
          <div class="form-field">
            <label>添加标签</label>
            <div class="tag-selector">
              <span
                v-for="tag in allTags"
                :key="tag.id"
                class="tag-option"
                :class="{ selected: selectedTagIds.includes(tag.id) }"
                :style="selectedTagIds.includes(tag.id) ? { background: `#${tag.color}`, color: '#fff' } : { borderColor: `#${tag.color}`, color: `#${tag.color}` }"
                @click="selectedTagIds.includes(tag.id) ? selectedTagIds = selectedTagIds.filter(id => id !== tag.id) : selectedTagIds.push(tag.id)"
              >
                {{ tag.name }}
              </span>
            </div>
          </div>
        </template>

        <!-- 追加模式：显示剧情选择器 -->
        <template v-else>
          <div class="form-field">
            <label>选择剧情</label>
            <div v-if="loadingStories" class="loading-stories">
              <i class="ri-loader-4-line spinning"></i> 加载中...
            </div>
            <div v-else-if="userStories.length === 0" class="no-stories">
              暂无剧情，请先创建一个剧情
            </div>
            <div v-else class="story-selector">
              <div
                v-for="story in userStories"
                :key="story.id"
                class="story-option"
                :class="{ selected: selectedStoryId === story.id }"
                @click="selectedStoryId = story.id"
              >
                <div class="story-option-title">{{ story.title }}</div>
                <div class="story-option-meta">
                  更新于 {{ new Date(story.updated_at).toLocaleDateString() }}
                </div>
              </div>
            </div>
          </div>
        </template>

        <p v-if="pendingRecords.length > 0" class="pending-info">
          将归档 {{ pendingRecords.length }} 条对话记录
        </p>
      </div>
      <template #footer>
        <RButton @click="showCreateModal = false">取消</RButton>
        <RButton
          type="primary"
          :loading="creating"
          :disabled="archiveMode === 'append' && !selectedStoryId"
          @click="handleCreateStory"
        >
          {{ archiveMode === 'create' ? '创建' : '追加' }}
        </RButton>
      </template>
    </RModal>

    <!-- 插件安装器 -->
    <AddonInstaller
      v-model="showAddonInstaller"
      :wow-path="wowPath"
      @installed="checkAddonStatus"
    />

    <!-- 插件更新提示 -->
    <AddonUpdateDialog ref="addonUpdateDialogRef" />
  </div>
</template>

<style scoped>
.archives-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.top-toolbar {
  background: #fff;
  border-radius: 16px;
  padding: 24px 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.page-title h1 {
  font-size: 28px;
  color: #4B3621;
  margin: 0 0 4px 0;
}

.page-title p {
  font-size: 14px;
  color: #856a52;
  margin: 0;
}

.btn-create {
  background: #804030;
  color: #fff;
  border: none;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-bar {
  display: flex;
  gap: 12px;
}

.filter-item {
  background: rgba(255,255,255,0.6);
  padding: 10px 16px;
  border-radius: 20px;
  border: 1px solid #d1bfa8;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.timeline-section {
  position: relative;
  padding: 40px 0;
}

.timeline-line {
  position: absolute;
  left: 50%;
  top: 0;
  bottom: 0;
  width: 3px;
  background: #4B3621;
  transform: translateX(-50%);
  opacity: 0.3;
}

.timeline-item {
  display: flex;
  margin-bottom: 40px;
  position: relative;
}

.timeline-item.left { justify-content: flex-start; padding-right: 52%; }
.timeline-item.right { justify-content: flex-end; padding-left: 52%; }

.timeline-dot {
  width: 16px;
  height: 16px;
  background: #EED9C4;
  border: 3px solid #4B3621;
  border-radius: 50%;
  position: absolute;
  left: 50%;
  top: 24px;
  transform: translateX(-50%);
  z-index: 2;
}

.timeline-item.highlight .timeline-dot {
  border-color: #B87333;
  box-shadow: 0 0 0 4px rgba(184,115,51,0.2);
}

.story-card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 8px 24px rgba(75,54,33,0.08);
  max-width: 400px;
}

.card-date {
  display: inline-block;
  background: rgba(184,115,51,0.1);
  color: #B87333;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}

.card-title {
  font-size: 18px;
  color: #2c1e12;
  font-weight: 600;
  margin-bottom: 12px;
}

.card-body {
  font-size: 14px;
  color: #665242;
  line-height: 1.7;
  margin-bottom: 16px;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid #f0e6dc;
  padding-top: 16px;
}

.avatars { display: flex; }
.avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  border: 2px solid #fff;
  margin-left: -8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #fff;
}
.avatar:nth-child(1) { background: #D4A373; margin-left: 0; }
.avatar:nth-child(2) { background: #A98467; }
.avatar:nth-child(3) { background: #ADC178; }
.avatar:nth-child(4) { background: #A9D6E5; }

.view-detail {
  color: #B87333;
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }

.create-form {
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
  color: #4B3621;
}

.form-field textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #d1bfa8;
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
  background: #fff;
  color: #4B3621;
}

.form-field textarea:focus {
  outline: none;
  border-color: #B87333;
  box-shadow: 0 0 0 2px rgba(184, 115, 51, 0.1);
}

.pending-info {
  font-size: 13px;
  color: #B87333;
  background: rgba(184, 115, 51, 0.1);
  padding: 8px 12px;
  border-radius: 6px;
  margin: 0;
}

.addon-notice {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(184, 115, 51, 0.1);
  border: 1px solid rgba(184, 115, 51, 0.2);
  border-radius: 8px;
  color: #804030;
  font-size: 14px;
}

.addon-notice i {
  font-size: 18px;
  color: #B87333;
}

.addon-notice span {
  flex: 1;
}

.tag-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-option {
  padding: 6px 12px;
  border: 1.5px solid;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.tag-option:hover {
  opacity: 0.8;
}

.tag-option.selected {
  font-weight: 600;
}

/* 模式切换器 */
.mode-switcher {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.mode-btn {
  flex: 1;
  padding: 10px 16px;
  border: 1.5px solid #d1bfa8;
  border-radius: 8px;
  background: #fff;
  color: #665242;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.2s;
}

.mode-btn:hover {
  border-color: #B87333;
  color: #B87333;
}

.mode-btn.active {
  background: #804030;
  border-color: #804030;
  color: #fff;
}

/* 剧情选择器 */
.story-selector {
  max-height: 240px;
  overflow-y: auto;
  border: 1px solid #d1bfa8;
  border-radius: 8px;
}

.story-option {
  padding: 12px 14px;
  cursor: pointer;
  border-bottom: 1px solid #f0e6dc;
  transition: background 0.2s;
}

.story-option:last-child {
  border-bottom: none;
}

.story-option:hover {
  background: rgba(184, 115, 51, 0.05);
}

.story-option.selected {
  background: rgba(184, 115, 51, 0.1);
  border-left: 3px solid #B87333;
}

.story-option-title {
  font-size: 14px;
  font-weight: 500;
  color: #4B3621;
  margin-bottom: 4px;
}

.story-option-meta {
  font-size: 12px;
  color: #856a52;
}

.loading-stories,
.no-stories {
  padding: 24px;
  text-align: center;
  color: #856a52;
  font-size: 14px;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
