<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import RCheckbox from '@/components/RCheckbox.vue'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'
import WowIcon from '@/components/WowIcon.vue'

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
  npc?: string   // NPC名字（仅NPC消息）
  nt?: string    // NPC说话类型: say/yell/whisper（仅NPC消息）
  ref_id?: string  // TRP3 profile ref ID
  raw_profile?: string  // 完整的TRP3 profile JSON
  listeners?: Listener[]  // 收听者列表（新增字段，向前兼容）
}

interface AccountChatLogs {
  account_id: string
  last_update: number | null
  record_count: number
  records: ChatRecord[]
}

const emit = defineEmits<{
  archive: [records: ChatRecord[]]
}>()

// 获取所有记录的扁平列表
const allRecords = computed(() => {
  const records: ChatRecord[] = []
  for (const account of accounts.value) {
    records.push(...account.records)
  }
  return records
})

// 获取选中的记录对象
function getSelectedRecords(): ChatRecord[] {
  return allRecords.value.filter(r => selectedRecords.value.has(r.timestamp))
}

// 清除选中状态
function clearSelection() {
  selectedRecords.value.clear()
}

// 移除已归档的记录（添加到已归档集合）
function removeArchivedRecords(timestamps: number[]) {
  for (const ts of timestamps) {
    archivedTimestamps.value.add(ts)
    selectedRecords.value.delete(ts)
  }
  saveArchivedTimestamps()
}

const loading = ref(false)
const accounts = ref<AccountChatLogs[]>([])
const selectedRecords = ref<Set<number>>(new Set())
const expandedDates = ref<Set<string>>(new Set())
const expandedHours = ref<Set<string>>(new Set())

// 筛选条件
const filterStartDate = ref('')
const filterEndDate = ref('')
const filterChannels = ref<Set<string>>(new Set())
const filterListeners = ref<Set<string>>(new Set())  // 收听者筛选（存储 gameID）

// 已归档的 timestamp 集合（持久化到 localStorage）
const archivedTimestamps = ref<Set<number>>(new Set(
  JSON.parse(localStorage.getItem('archived_timestamps') || '[]')
))

function saveArchivedTimestamps() {
  localStorage.setItem('archived_timestamps', JSON.stringify([...archivedTimestamps.value]))
}

// 获取所有可用的收听者列表
const availableListeners = computed(() => {
  const listenersMap = new Map<string, { gameID: string, name: string }>()

  for (const account of accounts.value) {
    for (const record of account.records) {
      // 向前兼容：如果没有 listeners 字段，跳过
      if (!record.listeners || record.listeners.length === 0) continue

      for (const listener of record.listeners) {
        if (!listenersMap.has(listener.gameID)) {
          // 提取角色名（去掉服务器后缀）
          const name = listener.gameID.split('-')[0]
          listenersMap.set(listener.gameID, { gameID: listener.gameID, name })
        }
      }
    }
  }

  return Array.from(listenersMap.values()).sort((a, b) => a.name.localeCompare(b.name))
})

// 按日期和小时分组的记录（过滤已归档 + 应用筛选条件）
const groupedRecords = computed(() => {
  const groups: Record<string, Record<string, ChatRecord[]>> = {}

  for (const account of accounts.value) {
    for (const record of account.records) {
      // 过滤已归档的记录
      if (archivedTimestamps.value.has(record.timestamp)) continue

      const date = new Date(record.timestamp * 1000)
      const dateStr = date.toISOString().split('T')[0]

      // 应用日期筛选
      if (filterStartDate.value && dateStr < filterStartDate.value) continue
      if (filterEndDate.value && dateStr > filterEndDate.value) continue

      // 应用频道筛选（标准化频道名称后比较）
      if (filterChannels.value.size > 0 && !filterChannels.value.has(normalizeChannel(record.channel))) continue

      // 应用收听者筛选（向前兼容：没有 listeners 字段的旧记录不过滤）
      if (filterListeners.value.size > 0 && record.listeners) {
        const hasMatchingListener = record.listeners.some(listener =>
          filterListeners.value.has(listener.gameID)
        )
        if (!hasMatchingListener) continue
      }

      const hourStr = date.getHours().toString().padStart(2, '0')

      if (!groups[dateStr]) groups[dateStr] = {}
      if (!groups[dateStr][hourStr]) groups[dateStr][hourStr] = []
      groups[dateStr][hourStr].push(record)
    }
  }

  // 对每个小时内的记录按时间戳排序（从新到旧）
  for (const dateKey in groups) {
    for (const hourKey in groups[dateKey]) {
      groups[dateKey][hourKey].sort((a, b) => b.timestamp - a.timestamp)
    }
  }

  return groups
})

const totalRecords = computed(() => {
  // 基于过滤后的 groupedRecords 计算
  let count = 0
  for (const date of Object.keys(groupedRecords.value)) {
    for (const hour of Object.keys(groupedRecords.value[date])) {
      count += groupedRecords.value[date][hour].length
    }
  }
  return count
})

const selectedCount = computed(() => selectedRecords.value.size)

// 滚动容器引用
const contentRef = ref<HTMLElement | null>(null)

// 滚动到底部
function scrollToBottom() {
  if (contentRef.value) {
    setTimeout(() => {
      contentRef.value!.scrollTop = contentRef.value!.scrollHeight
    }, 100)
  }
}

// 切换频道筛选
function toggleChannel(channel: string) {
  if (filterChannels.value.has(channel)) {
    filterChannels.value.delete(channel)
  } else {
    filterChannels.value.add(channel)
  }
  // 触发响应式更新
  filterChannels.value = new Set(filterChannels.value)
}

// 切换收听者筛选
function toggleListener(gameID: string) {
  if (filterListeners.value.has(gameID)) {
    filterListeners.value.delete(gameID)
  } else {
    filterListeners.value.add(gameID)
  }
  // 触发响应式更新
  filterListeners.value = new Set(filterListeners.value)
}

// 清除筛选
function clearFilters() {
  filterStartDate.value = ''
  filterEndDate.value = ''
  filterChannels.value.clear()
  filterListeners.value.clear()
}

async function syncFromPlugin() {
  const wowPath = localStorage.getItem('wow_path') || ''
  console.log('[StagingPool] wowPath:', wowPath)
  if (!wowPath) {
    console.log('[StagingPool] wowPath 为空，跳过同步')
    return
  }
  loading.value = true
  try {
    console.log('[StagingPool] 调用 scan_chat_logs...')
    accounts.value = await invoke<AccountChatLogs[]>('scan_chat_logs', {
      wowPath,
    })
    console.log('[StagingPool] 结果:', accounts.value)
    // 默认展开最后一个日期（最新的）
    const dates = Object.keys(groupedRecords.value).sort()
    if (dates.length > 0) {
      expandedDates.value.add(dates[dates.length - 1])
      // 展开该日期下的最后一个小时
      const lastDate = dates[dates.length - 1]
      const hours = Object.keys(groupedRecords.value[lastDate]).sort()
      if (hours.length > 0) {
        expandedHours.value.add(`${lastDate}-${hours[hours.length - 1]}`)
      }
    }
    // 滚动到底部
    scrollToBottom()
  } catch (e) {
    console.error('同步失败:', e)
  } finally {
    loading.value = false
  }
}

function toggleDate(date: string) {
  if (expandedDates.value.has(date)) {
    expandedDates.value.delete(date)
  } else {
    expandedDates.value.add(date)
  }
}

function toggleHour(key: string) {
  if (expandedHours.value.has(key)) {
    expandedHours.value.delete(key)
  } else {
    expandedHours.value.add(key)
  }
}

function toggleRecord(timestamp: number) {
  if (selectedRecords.value.has(timestamp)) {
    selectedRecords.value.delete(timestamp)
  } else {
    selectedRecords.value.add(timestamp)
  }
}

// 获取日期下所有记录
function getDateRecords(date: string): ChatRecord[] {
  const hours = groupedRecords.value[date] || {}
  return Object.values(hours).flat()
}

// 获取小时下所有记录
function getHourRecords(date: string, hour: string): ChatRecord[] {
  return groupedRecords.value[date]?.[hour] || []
}

// 判断日期是否全选
function isDateAllSelected(date: string): boolean {
  const records = getDateRecords(date)
  return records.length > 0 && records.every(r => selectedRecords.value.has(r.timestamp))
}

// 判断日期是否部分选中
function isDatePartialSelected(date: string): boolean {
  const records = getDateRecords(date)
  const selectedCount = records.filter(r => selectedRecords.value.has(r.timestamp)).length
  return selectedCount > 0 && selectedCount < records.length
}

// 判断小时是否全选
function isHourAllSelected(date: string, hour: string): boolean {
  const records = getHourRecords(date, hour)
  return records.length > 0 && records.every(r => selectedRecords.value.has(r.timestamp))
}

// 判断小时是否部分选中
function isHourPartialSelected(date: string, hour: string): boolean {
  const records = getHourRecords(date, hour)
  const selectedCount = records.filter(r => selectedRecords.value.has(r.timestamp)).length
  return selectedCount > 0 && selectedCount < records.length
}

// 切换日期选中状态
function toggleDateSelection(date: string) {
  const records = getDateRecords(date)
  const allSelected = isDateAllSelected(date)
  for (const r of records) {
    if (allSelected) {
      selectedRecords.value.delete(r.timestamp)
    } else {
      selectedRecords.value.add(r.timestamp)
    }
  }
}

// 切换小时选中状态
function toggleHourSelection(date: string, hour: string) {
  const records = getHourRecords(date, hour)
  const allSelected = isHourAllSelected(date, hour)
  for (const r of records) {
    if (allSelected) {
      selectedRecords.value.delete(r.timestamp)
    } else {
      selectedRecords.value.add(r.timestamp)
    }
  }
}

// 清理WoW特殊格式字符
function cleanWowText(text: string): string {
  return text
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

function getSenderName(record: ChatRecord): string {
  // NPC消息显示NPC名字（清理特殊字符）
  if (record.mark === 'N' && record.npc) {
    return cleanWowText(record.npc)
  }
  // 玩家消息优先显示TRP3名字
  const trp3 = record.sender.trp3
  if (trp3?.FN) {
    return trp3.LN ? `${trp3.FN}·${trp3.LN}` : trp3.FN
  }
  return record.sender.gameID.split('-')[0]
}

function getSenderIcon(record: ChatRecord): string {
  return record.sender.trp3?.IC || ''
}

function getSenderColor(record: ChatRecord): string {
  return record.sender.trp3?.CH || ''
}

function getMarkClass(mark?: string): string {
  if (mark === 'N') return 'mark-npc'
  if (mark === 'B') return 'mark-background'
  return ''
}

function getNpcTalkLabel(nt?: string): string {
  const map: Record<string, string> = {
    'say': '说',
    'yell': '喊',
    'whisper': '密语',
  }
  return nt ? map[nt] || nt : ''
}

// 获取NPC说话类型对应的CSS类
function getNpcTalkClass(nt?: string): string {
  if (nt === 'yell') return 'npc-yell'
  if (nt === 'whisper') return 'npc-whisper'
  return 'npc-say'
}

// 获取NPC说话类型对应的文字颜色
function getNpcTalkTextColor(nt?: string): string {
  if (nt === 'yell') return '#FF3333'
  if (nt === 'whisper') return '#B39DDB'
  return ''
}

function formatTime(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

// 标准化频道名称（统一转换为短格式）
function normalizeChannel(channel: string): string {
  // 处理 CHAT_MSG_ 前缀
  if (channel.startsWith('CHAT_MSG_')) {
    channel = channel.replace('CHAT_MSG_', '')
  }
  // 将 TEXT_EMOTE 统一为 EMOTE
  if (channel === 'TEXT_EMOTE') {
    return 'EMOTE'
  }
  return channel
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

// 获取频道对应的文字颜色
function getChannelTextColor(channel: string): string {
  const colorMap: Record<string, string> = {
    'SAY': '',
    'YELL': '#FF3333',
    'WHISPER': '#B39DDB',
    'EMOTE': '#FF8C00',
    'TEXT_EMOTE': '#FF8C00',
    'PARTY': '#AAAAFF',
    'RAID': '#FF7F00',
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

// 获取频道标签的CSS类
function getChannelClass(channel: string): string {
  if (channel === 'YELL' || channel === 'CHAT_MSG_YELL') return 'channel-yell'
  if (channel === 'WHISPER' || channel === 'CHAT_MSG_WHISPER') return 'channel-whisper'
  return ''
}

onMounted(() => {
  const wowPath = localStorage.getItem('wow_path')
  if (wowPath) syncFromPlugin()
})

// 暴露方法供父组件调用
defineExpose({
  getSelectedRecords,
  clearSelection,
  removeArchivedRecords,
  syncFromPlugin
})
</script>

<template>
  <div class="staging-pool">
    <div class="staging-header">
      <div class="staging-info">
        <span>待归档池 ({{ totalRecords }}条)</span>
      </div>
      <div class="staging-actions">
        <RButton :loading="loading" @click="syncFromPlugin">
          <i class="ri-refresh-line"></i> 同步
        </RButton>
      </div>
    </div>

    <!-- 筛选面板 -->
    <div class="filter-panel">
      <div class="filter-row">
        <div class="filter-group">
          <label>开始日期</label>
          <input type="date" v-model="filterStartDate" class="date-input" />
        </div>
        <div class="filter-group">
          <label>结束日期</label>
          <input type="date" v-model="filterEndDate" class="date-input" />
        </div>
        <RButton size="small" @click="clearFilters">
          <i class="ri-close-line"></i> 清除筛选
        </RButton>
      </div>
      <div class="filter-row">
        <label>聊天类型：</label>
        <div class="channel-filters">
          <button
            v-for="channel in ['SAY', 'YELL', 'EMOTE', 'PARTY', 'RAID', 'WHISPER']"
            :key="channel"
            class="channel-filter-btn"
            :class="{ active: filterChannels.has(channel) }"
            @click="toggleChannel(channel)"
          >
            {{ getChannelLabel(channel) }}
          </button>
        </div>
      </div>
      <div class="filter-row">
        <label>收听者人物：</label>
        <div class="channel-filters">
          <button
            v-for="listener in availableListeners"
            :key="listener.gameID"
            class="channel-filter-btn"
            :class="{ active: filterListeners.has(listener.gameID) }"
            @click="toggleListener(listener.gameID)"
          >
            {{ listener.name }}
          </button>
          <span v-if="availableListeners.length === 0" class="filter-empty-hint">
            暂无数据（需要新的聊天记录）
          </span>
        </div>
      </div>
    </div>

    <REmpty v-if="!loading && totalRecords === 0" description="需要安装 RPBox Addon 插件才能采集聊天记录">
      <router-link class="tutorial-link" :to="{ name: 'guide' }">
        <i class="ri-book-open-line"></i> 查看使用教程
      </router-link>
    </REmpty>

    <div v-else class="staging-content" ref="contentRef">
      <div
        v-for="date in Object.keys(groupedRecords).sort().reverse()"
        :key="date"
        class="date-group"
      >
        <div class="date-header">
          <span class="header-checkbox" @click.stop="toggleDateSelection(date)">
            <RCheckbox
              :model-value="isDateAllSelected(date)"
              :indeterminate="isDatePartialSelected(date)"
            />
          </span>
          <div class="date-header-content" @click="toggleDate(date)">
            <i :class="expandedDates.has(date) ? 'ri-arrow-down-s-line' : 'ri-arrow-right-s-line'"></i>
            <span>{{ date }}</span>
            <span class="record-count">
              ({{ Object.values(groupedRecords[date]).flat().length }}条)
            </span>
          </div>
        </div>

        <div v-if="expandedDates.has(date)" class="hour-groups">
          <div
            v-for="hour in Object.keys(groupedRecords[date]).sort().reverse()"
            :key="`${date}-${hour}`"
            class="hour-group"
          >
            <div class="hour-header">
              <span class="header-checkbox" @click.stop="toggleHourSelection(date, hour)">
                <RCheckbox
                  :model-value="isHourAllSelected(date, hour)"
                  :indeterminate="isHourPartialSelected(date, hour)"
                />
              </span>
              <div class="hour-header-content" @click="toggleHour(`${date}-${hour}`)">
                <i :class="expandedHours.has(`${date}-${hour}`) ? 'ri-arrow-down-s-line' : 'ri-arrow-right-s-line'"></i>
                <span>{{ hour }}:00 - {{ hour }}:59</span>
                <span class="record-count">({{ groupedRecords[date][hour].length }}条)</span>
              </div>
            </div>

            <div v-if="expandedHours.has(`${date}-${hour}`)" class="records">
              <div
                v-for="record in groupedRecords[date][hour]"
                :key="record.timestamp"
                class="record-item"
                :class="[
                  { selected: selectedRecords.has(record.timestamp) },
                  getMarkClass(record.mark)
                ]"
                @click="toggleRecord(record.timestamp)"
              >
                <RCheckbox :model-value="selectedRecords.has(record.timestamp)" />
                <span class="record-time">{{ formatTime(record.timestamp) }}</span>
                <span v-if="record.mark === 'N' && record.nt" class="record-channel" :class="getNpcTalkClass(record.nt)">[NPC{{ getNpcTalkLabel(record.nt) }}]</span>
                <span v-else class="record-channel" :class="getChannelClass(record.channel)">[{{ getChannelLabel(record.channel) }}]</span>
                <template v-if="record.mark !== 'B'">
                  <WowIcon v-if="getSenderIcon(record)" :icon="getSenderIcon(record)" :size="18" class="record-avatar" />
                  <span class="record-sender" :style="getSenderColor(record) ? { color: '#' + getSenderColor(record) } : {}">{{ getSenderName(record) }}:</span>
                </template>
                <span
                  class="record-content"
                  :style="(record.mark === 'N' && record.nt)
                    ? (getNpcTalkTextColor(record.nt) ? { color: getNpcTalkTextColor(record.nt) } : {})
                    : (getChannelTextColor(record.channel) ? { color: getChannelTextColor(record.channel) } : {})"
                >{{ record.content }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="selectedCount > 0" class="staging-footer">
      <span>已选 {{ selectedCount }} 条对话</span>
      <RButton type="primary" @click="$emit('archive', getSelectedRecords())">
        归档选中内容
      </RButton>
    </div>
  </div>
</template>

<style scoped>
.staging-pool {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.staging-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--color-border);
}

.staging-info {
  font-weight: 600;
  color: var(--color-primary);
}

.staging-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.date-group {
  margin-bottom: 8px;
}

.date-header,
.hour-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  transition: background 0.2s;
}

.date-header:hover,
.hour-header:hover {
  background: var(--color-bg-secondary);
}

.header-checkbox {
  display: flex;
  cursor: pointer;
}

.header-checkbox :deep(.r-checkbox) {
  pointer-events: none;
}

.date-header-content,
.hour-header-content {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  cursor: pointer;
}

.date-header {
  font-weight: 600;
  color: var(--color-primary);
}

.hour-header {
  margin-left: 24px;
  font-size: 14px;
}

.record-count {
  color: var(--color-secondary);
  font-weight: normal;
}

.hour-groups {
  margin-left: 12px;
}

.records {
  margin-left: 48px;
  border-left: 2px solid var(--color-border);
  padding-left: 12px;
}

.record-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px;
  cursor: pointer;
  border-radius: var(--radius-sm);
  font-size: 14px;
  line-height: 1.5;
}

.record-item:hover {
  background: var(--color-bg-secondary);
}

.record-item.selected {
  background: rgba(128, 64, 48, 0.1);
}

.record-time {
  color: var(--color-secondary);
  flex-shrink: 0;
}

.record-channel {
  color: var(--color-accent);
  flex-shrink: 0;
}

.record-channel.npc-say {
  color: #9b59b6;  /* 紫色表示NPC说 */
}

.record-channel.npc-yell {
  color: #FF3333;  /* 红色表示NPC喊 */
}

.record-channel.npc-whisper {
  color: #B39DDB;  /* 淡紫色表示NPC密语 */
}

.record-channel.channel-yell {
  color: #FF3333;
}

.record-channel.channel-whisper {
  color: #B39DDB;
}

.record-avatar {
  flex-shrink: 0;
  border-radius: 3px;
}

.record-sender {
  color: var(--color-primary);
  font-weight: 500;
  flex-shrink: 0;
}

.record-content {
  color: var(--color-text);
}

/* 消息类型样式 */
.record-item.mark-npc .record-sender {
  color: #9b59b6;  /* 紫色表示NPC */
}

.record-item.mark-background {
  font-style: italic;
  opacity: 0.85;
}

.record-item.mark-background .record-content {
  color: var(--color-secondary);
}

.staging-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-top: 1px solid var(--color-border);
  background: var(--color-bg);
}

.tutorial-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #B87333;
  font-size: 14px;
  text-decoration: none;
  padding: 8px 16px;
  border-radius: 6px;
  transition: all 0.2s;
}

.tutorial-link:hover {
  background: rgba(184, 115, 51, 0.1);
}

/* 筛选面板 */
.filter-panel {
  padding: 16px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  margin-bottom: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.filter-group label {
  font-size: 12px;
  color: var(--color-secondary);
  font-weight: 500;
}

.date-input {
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 14px;
  background: #fff;
  color: var(--color-primary);
}

.date-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.channel-filters {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.channel-filter-btn {
  padding: 6px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: #fff;
  color: var(--color-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.channel-filter-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.channel-filter-btn.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
  font-weight: 500;
}

.filter-empty-hint {
  color: var(--color-secondary);
  font-size: 13px;
  font-style: italic;
  padding: 6px 12px;
}
</style>
