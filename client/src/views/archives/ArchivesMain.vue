<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import RTabs from '@/components/RTabs.vue'
import RTabPane from '@/components/RTabPane.vue'
import RModal from '@/components/RModal.vue'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'
import AddonInstaller from '@/components/AddonInstaller.vue'
import StagingPool from './StagingPool.vue'
import StoryList from './StoryList.vue'
import { createStory, addStoryEntries, type CreateStoryEntryRequest } from '@/api/story'

// ChatRecord 类型定义
interface TRP3Info {
  FN?: string
  LN?: string
  TI?: string
  IC?: string
  CH?: string  // 名字颜色
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

// 待归档的记录
const pendingRecords = ref<ChatRecord[]>([])

// 插件状态
const showAddonInstaller = ref(false)
const addonInstalled = ref(false)
const selectedFlavor = ref('_retail_')

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

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
  checkAddonStatus()
})

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
  showCreateModal.value = true
}

async function handleCreateStory() {
  if (!newStoryTitle.value.trim()) return
  creating.value = true
  try {
    const story = await createStory({
      title: newStoryTitle.value,
      description: newStoryDesc.value,
    })
    // 如果有待归档记录，添加到剧情
    if (pendingRecords.value.length > 0 && story.id) {
      console.log('[Archive] pendingRecords:', pendingRecords.value)
      const entries: CreateStoryEntryRequest[] = pendingRecords.value.map(record => {
        const trp3 = record.sender.trp3
        console.log('[Archive] record.sender:', record.sender)
        console.log('[Archive] trp3:', trp3)

        // 根据mark类型确定speaker和type
        let speaker: string
        let type: string = 'dialogue'
        let channel: string = record.channel
        let speakerIc: string = trp3?.IC || ''

        console.log('[Archive] mark:', record.mark, 'channel:', record.channel, 'nt:', record.nt, 'npc:', record.npc)

        if (record.mark === 'N' && record.npc) {
          // NPC消息，使用nt字段作为频道，设置NPC图标
          speaker = record.npc
          speakerIc = '_NPC_' // NPC专用标记
          if (record.nt) {
            channel = record.nt.toUpperCase() // say -> SAY, yell -> YELL, whisper -> WHISPER
          }
        } else if (record.mark === 'B' || (record.mark === 'N' && !record.npc)) {
          // 旁白/背景，或者没有NPC名字的NPC消息也当作旁白
          speaker = ''
          speakerIc = '_NARRATION_' // 旁白专用标记
          type = 'narration'
        } else {
          // 玩家消息
          speaker = trp3?.FN
            ? (trp3.LN ? `${trp3.FN} ${trp3.LN}` : trp3.FN)
            : record.sender.gameID.split('-')[0]
        }

        return {
          source_id: `chat_${record.timestamp}`,
          type: type,
          speaker: speaker,
          speaker_ic: speakerIc,
          speaker_color: trp3?.CH || '',
          content: cleanTRP3Content(record.content),
          channel: channel,
          timestamp: new Date(record.timestamp * 1000).toISOString(),
        }
      })
      await addStoryEntries(story.id, entries)
      // 从待归档池移除已归档的记录
      const archivedTimestamps = pendingRecords.value.map(r => r.timestamp)
      stagingPoolRef.value?.removeArchivedRecords?.(archivedTimestamps)
      pendingRecords.value = []
    }
    showCreateModal.value = false
    newStoryTitle.value = ''
    newStoryDesc.value = ''
    activeTab.value = 'stories'
    // 刷新剧情列表
    storyListRef.value?.loadStories?.()
  } catch (e) {
    console.error('创建失败:', e)
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

    <!-- 创建剧情对话框 -->
    <RModal v-model="showCreateModal" title="新建剧情" width="480px">
      <div class="create-form">
        <div class="form-field">
          <label>剧情标题</label>
          <RInput v-model="newStoryTitle" placeholder="输入剧情标题" />
        </div>
        <div class="form-field">
          <label>剧情描述</label>
          <textarea v-model="newStoryDesc" placeholder="简要描述这个剧情..." rows="3"></textarea>
        </div>
        <p v-if="pendingRecords.length > 0" class="pending-info">
          将归档 {{ pendingRecords.length }} 条对话记录
        </p>
      </div>
      <template #footer>
        <RButton @click="showCreateModal = false">取消</RButton>
        <RButton type="primary" :loading="creating" @click="handleCreateStory">创建</RButton>
      </template>
    </RModal>

    <!-- 插件安装器 -->
    <AddonInstaller
      v-model="showAddonInstaller"
      :wow-path="wowPath"
      @installed="checkAddonStatus"
    />
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
</style>
