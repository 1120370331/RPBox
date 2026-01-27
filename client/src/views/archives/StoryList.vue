<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listStories, deleteStory, deleteStories, moveStoriesToStory, updateStory, updateStoriesBackgroundColor, type Story, type StoryFilterParams } from '@/api/story'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import { useDialog } from '@/composables/useDialog'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'
import RCard from '@/components/RCard.vue'
import RModal from '@/components/RModal.vue'
import RInput from '@/components/RInput.vue'
import StoryFilter from '@/components/StoryFilter.vue'

const { t } = useI18n()
const { confirm, alert } = useDialog()

const props = defineProps<{
  initialFilter?: StoryFilterParams
}>()

const emit = defineEmits<{
  create: []
  view: [id: number]
}>()

const viewMode = ref<'timeline' | 'grid'>('timeline')

const loading = ref(false)
const stories = ref<Story[]>([])
const tags = ref<Tag[]>([])
const guilds = ref<Guild[]>([])
const showFilter = ref(false)
const filterParams = ref<StoryFilterParams>(props.initialFilter || {})

// 改名相关
const showRenameModal = ref(false)
const renamingStory = ref<Story | null>(null)
const newStoryTitle = ref('')

// 管理模式
const manageMode = ref(false)
const selectedIds = ref<number[]>([])
const batchDeleting = ref(false)

// 移动到其它剧情
const showMoveModal = ref(false)
const moveTargetId = ref<number | null>(null)
const batchMoving = ref(false)

// 编组/背景色
const showColorModal = ref(false)
const selectedColor = ref('')
const batchUpdatingColor = ref(false)
const defaultColors = [
  '#E57373', '#F06292', '#BA68C8', '#9575CD', '#7986CB',
  '#64B5F6', '#4FC3F7', '#4DD0E1', '#4DB6AC', '#81C784',
  '#AED581', '#DCE775', '#FFF176', '#FFD54F', '#FFB74D',
  '#FF8A65', '#A1887F', '#90A4AE'
]

function enterManageMode() {
  manageMode.value = true
  selectedIds.value = []
}

function exitManageMode() {
  manageMode.value = false
  selectedIds.value = []
}

function toggleSelect(id: number) {
  const index = selectedIds.value.indexOf(id)
  if (index === -1) {
    selectedIds.value.push(id)
  } else {
    selectedIds.value.splice(index, 1)
  }
}

function selectAll() {
  if (selectedIds.value.length === stories.value.length) {
    selectedIds.value = []
  } else {
    selectedIds.value = stories.value.map(s => s.id)
  }
}

async function handleBatchDelete() {
  if (selectedIds.value.length === 0) return

  const confirmed = await confirm({
    title: t('archives.batch.deleteConfirmTitle'),
    message: t('archives.batch.deleteConfirmMessage', { count: selectedIds.value.length }),
    type: 'error'
  })
  if (!confirmed) return

  batchDeleting.value = true
  try {
    await deleteStories(selectedIds.value)
    stories.value = stories.value.filter(s => !selectedIds.value.includes(s.id))
    exitManageMode()
  } catch (e: any) {
    await alert({
      title: t('archives.batch.deleteFailed'),
      message: e.message || t('archives.batch.batchDeleteFailed'),
      type: 'error'
    })
  } finally {
    batchDeleting.value = false
  }
}

// 可选的目标剧情（排除已选中的）
const availableTargetStories = computed(() => {
  return stories.value.filter(s => !selectedIds.value.includes(s.id))
})

function openMoveModal() {
  if (selectedIds.value.length === 0) return
  moveTargetId.value = null
  showMoveModal.value = true
}

async function handleBatchMove() {
  if (selectedIds.value.length === 0 || !moveTargetId.value) return

  batchMoving.value = true
  try {
    await moveStoriesToStory(selectedIds.value, moveTargetId.value)
    // 移动成功后，删除源剧情并刷新列表
    stories.value = stories.value.filter(s => !selectedIds.value.includes(s.id))
    showMoveModal.value = false
    exitManageMode()
  } catch (e: any) {
    await alert({
      title: t('archives.batch.moveFailed'),
      message: e.message || t('archives.batch.batchMoveFailed'),
      type: 'error'
    })
  } finally {
    batchMoving.value = false
  }
}

function openColorModal() {
  if (selectedIds.value.length === 0) return
  selectedColor.value = ''
  showColorModal.value = true
}

async function handleBatchUpdateColor() {
  if (selectedIds.value.length === 0) return

  batchUpdatingColor.value = true
  try {
    await updateStoriesBackgroundColor(selectedIds.value, selectedColor.value)
    // 更新本地数据
    stories.value = stories.value.map(s => {
      if (selectedIds.value.includes(s.id)) {
        return { ...s, background_color: selectedColor.value }
      }
      return s
    })
    showColorModal.value = false
    exitManageMode()
  } catch (e: any) {
    await alert({
      title: t('archives.batch.updateFailed'),
      message: e.message || t('archives.batch.batchUpdateFailed'),
      type: 'error'
    })
  } finally {
    batchUpdatingColor.value = false
  }
}

async function loadStories() {
  loading.value = true
  try {
    const res = await listStories(filterParams.value)
    stories.value = res.stories || []
  } catch (e) {
    console.error('加载失败:', e)
  } finally {
    loading.value = false
  }
}

async function loadTags() {
  try {
    const res = await listTags('story')
    tags.value = res.tags || []
  } catch (e) {
    console.error('加载标签失败:', e)
  }
}

async function loadGuilds() {
  try {
    const res = await listGuilds()
    guilds.value = res.guilds || []
  } catch (e) {
    console.error('加载公会失败:', e)
  }
}

function handleFilter(values: any) {
  filterParams.value = {
    tag_ids: values.tagIds.length > 0 ? values.tagIds.join(',') : undefined,
    guild_id: values.guildId ? String(values.guildId) : undefined,
    search: values.search || undefined,
    start_date: values.startDate || undefined,
    end_date: values.endDate || undefined,
    sort: values.sort,
    order: values.order
  }
  loadStories()
}

function handleResetFilter() {
  filterParams.value = {}
  loadStories()
}

async function handleDelete(id: number) {
  const confirmed = await confirm({
    title: t('archives.confirm.deleteTitle'),
    message: t('archives.confirm.deleteMessage'),
    type: 'error'
  })
  if (!confirmed) return

  try {
    await deleteStory(id)
    stories.value = stories.value.filter(s => s.id !== id)
  } catch (e: any) {
    await alert({
      title: t('archives.batch.deleteFailed'),
      message: e.message || t('archives.batch.deleteFailed'),
      type: 'error'
    })
  }
}

function openRenameModal(story: Story) {
  renamingStory.value = story
  newStoryTitle.value = story.title
  showRenameModal.value = true
}

async function handleRename() {
  if (!renamingStory.value || !newStoryTitle.value.trim()) return

  try {
    await updateStory(renamingStory.value.id, { title: newStoryTitle.value })
    const index = stories.value.findIndex(s => s.id === renamingStory.value!.id)
    if (index !== -1) {
      stories.value[index].title = newStoryTitle.value
    }
    showRenameModal.value = false
  } catch (e: any) {
    await alert({
      title: t('archives.modal.renameFailed'),
      message: e.message || t('archives.modal.renameFailed'),
      type: 'error'
    })
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

function getParticipants(story: Story): string[] {
  if (!story.participants) return []
  try {
    return JSON.parse(story.participants)
  } catch {
    return []
  }
}

function getTags(story: Story): string[] {
  if (!story.tags) return []
  return story.tags.split(',').map(t => t.trim()).filter(Boolean)
}

const tagMap = computed(() => {
  const map = new Map<string, Tag>()
  for (const tag of tags.value) {
    map.set(tag.name, tag)
  }
  return map
})

function getTagChips(story: Story): { name: string; color?: string }[] {
  return getTags(story).map((name) => {
    const tag = tagMap.value.get(name)
    return { name, color: tag?.color }
  })
}

// 按月份分组剧情用于时间线视图
const groupedStories = computed(() => {
  const groups: { month: string; stories: Story[] }[] = []
  const monthMap = new Map<string, Story[]>()

  for (const story of stories.value) {
    const date = new Date(story.created_at)
    const year = date.getFullYear()
    const month = date.getMonth() + 1
    const monthKey = `${year}-${month}`
    if (!monthMap.has(monthKey)) {
      monthMap.set(monthKey, [])
    }
    monthMap.get(monthKey)!.push(story)
  }

  for (const [key, items] of monthMap) {
    const [year, month] = key.split('-').map(Number)
    const monthLabel = t('common.calendar.yearMonth', { year, month })
    groups.push({ month: monthLabel, stories: items })
  }

  return groups
})

onMounted(() => {
  loadStories()
  loadTags()
  loadGuilds()
})

// 监听外部传入的筛选参数变化
watch(() => props.initialFilter, (newFilter) => {
  if (newFilter) {
    filterParams.value = { ...newFilter }
    loadStories()
  }
}, { deep: true })

// 暴露方法供父组件调用
defineExpose({
  loadStories
})
</script>

<template>
  <div class="story-list">
    <!-- 筛选工具栏 -->
    <div class="filter-toolbar">
      <div class="view-toggle">
        <button
          class="toggle-btn"
          :class="{ active: viewMode === 'timeline' }"
          @click="viewMode = 'timeline'"
          :disabled="manageMode"
        >
          <i class="ri-time-line"></i> {{ t('archives.view.timeline') }}
        </button>
        <button
          class="toggle-btn"
          :class="{ active: viewMode === 'grid' }"
          @click="viewMode = 'grid'"
          :disabled="manageMode"
        >
          <i class="ri-grid-line"></i> {{ t('archives.view.grid') }}
        </button>
      </div>
      <div class="toolbar-actions">
        <RButton v-if="!manageMode" @click="showFilter = !showFilter">
          <i class="ri-filter-3-line"></i>
          {{ showFilter ? t('archives.filter.hideFilter') : t('archives.filter.showFilter') }}
        </RButton>
        <RButton v-if="!manageMode" :disabled="stories.length === 0" @click="enterManageMode">
          <i class="ri-checkbox-multiple-line"></i> {{ t('archives.action.manage') }}
        </RButton>
        <template v-if="manageMode">
          <RButton @click="selectAll">
            <i class="ri-checkbox-line"></i>
            {{ selectedIds.length === stories.length ? t('archives.action.deselectAll') : t('archives.action.selectAll') }}
          </RButton>
          <RButton @click="exitManageMode">
            <i class="ri-close-line"></i> {{ t('archives.action.exitManage') }}
          </RButton>
        </template>
      </div>
    </div>

    <!-- 筛选面板 -->
    <StoryFilter
      v-if="showFilter"
      :tags="tags"
      :guilds="guilds"
      @filter="handleFilter"
      @reset="handleResetFilter"
    />

    <REmpty v-if="!loading && stories.length === 0" :description="t('archives.empty.noRecords')">
      <RButton type="primary" @click="$emit('create')">{{ t('archives.action.createFirst') }}</RButton>
    </REmpty>

    <!-- 时间线视图 -->
    <div v-else-if="viewMode === 'timeline'" class="timeline-view">
      <div class="timeline-line"></div>
      <div v-for="group in groupedStories" :key="group.month" class="timeline-group">
        <div class="timeline-month">{{ group.month }}</div>
        <div
          v-for="(story, index) in group.stories"
          :key="story.id"
          class="timeline-item"
          :class="[index % 2 === 0 ? 'left' : 'right', { selected: selectedIds.includes(story.id) }]"
        >
          <div class="timeline-connector"></div>
          <div class="timeline-dot"></div>
          <div
            class="timeline-card"
            :class="{ 'manage-mode': manageMode }"
            :style="story.background_color ? { backgroundColor: story.background_color + '20', borderLeft: `4px solid ${story.background_color}` } : {}"
            @click="manageMode ? toggleSelect(story.id) : $emit('view', story.id)"
          >
            <!-- 管理模式复选框 -->
            <div v-if="manageMode" class="card-checkbox" @click.stop="toggleSelect(story.id)">
              <i :class="selectedIds.includes(story.id) ? 'ri-checkbox-fill' : 'ri-checkbox-blank-line'"></i>
            </div>
            <div class="card-date">{{ formatDate(story.created_at) }}</div>
            <h3 class="card-title">{{ story.title }}</h3>
            <p class="card-desc">{{ story.description || t('archives.story.noDescription') }}</p>
            <div v-if="getTagChips(story).length" class="card-tags">
              <span
                v-for="tag in getTagChips(story)"
                :key="tag.name"
                class="tag"
                :style="tag.color ? { background: `#${tag.color}20`, color: `#${tag.color}` } : {}"
              >
                {{ tag.name }}
              </span>
            </div>
            <div v-if="!manageMode" class="card-footer">
              <div class="participants">
                <span v-for="(p, i) in getParticipants(story).slice(0, 3)" :key="i" class="avatar">
                  {{ p.charAt(0) }}
                </span>
              </div>
              <div class="card-actions">
                <button class="action-btn" @click.stop="openRenameModal(story)">
                  <i class="ri-edit-line"></i>
                </button>
                <button class="action-btn danger" @click.stop="handleDelete(story.id)">
                  <i class="ri-delete-bin-line"></i>
                </button>
                <span class="view-link">{{ t('archives.action.viewDetail') }} <i class="ri-arrow-right-s-line"></i></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 网格视图 -->
    <div v-else class="stories-grid">
      <RCard
        v-for="story in stories"
        :key="story.id"
        class="story-card"
        :class="{ selected: selectedIds.includes(story.id), 'manage-mode': manageMode }"
        :style="story.background_color ? { backgroundColor: story.background_color + '20', borderLeft: `4px solid ${story.background_color}` } : {}"
        hoverable
        @click="manageMode ? toggleSelect(story.id) : undefined"
      >
        <!-- 管理模式复选框 -->
        <div v-if="manageMode" class="grid-checkbox" @click.stop="toggleSelect(story.id)">
          <i :class="selectedIds.includes(story.id) ? 'ri-checkbox-fill' : 'ri-checkbox-blank-line'"></i>
        </div>
        <div class="story-header">
          <span class="story-date">{{ formatDate(story.created_at) }}</span>
          <span class="story-status" :class="story.status">
            {{ story.status === 'published' ? t('archives.story.published') : t('archives.story.draft') }}
          </span>
        </div>
        <h3 class="story-title">{{ story.title }}</h3>
        <p class="story-desc">{{ story.description || t('archives.story.noDescription') }}</p>
        <div v-if="getTagChips(story).length" class="story-tags">
          <span
            v-for="tag in getTagChips(story)"
            :key="tag.name"
            class="tag"
            :style="tag.color ? { background: `#${tag.color}20`, color: `#${tag.color}` } : {}"
          >
            {{ tag.name }}
          </span>
        </div>
        <div v-if="!manageMode" class="story-footer">
          <div class="participants">
            <span v-for="(p, i) in getParticipants(story).slice(0, 3)" :key="i" class="participant">
              {{ p.charAt(0) }}
            </span>
            <span v-if="getParticipants(story).length > 3" class="more">
              +{{ getParticipants(story).length - 3 }}
            </span>
          </div>
          <div class="actions">
            <RButton size="small" @click="$emit('view', story.id)">{{ t('archives.action.view') }}</RButton>
            <RButton size="small" @click="openRenameModal(story)">{{ t('archives.action.rename') }}</RButton>
            <RButton size="small" type="danger" @click="handleDelete(story.id)">{{ t('archives.action.delete') }}</RButton>
          </div>
        </div>
      </RCard>
    </div>

    <!-- 改名弹窗 -->
    <RModal v-model="showRenameModal" :title="t('archives.modal.renameTitle')" @confirm="handleRename">
      <RInput
        v-model="newStoryTitle"
        :placeholder="t('archives.modal.renamePlaceholder')"
        maxlength="100"
      />
    </RModal>

    <!-- 批量操作栏 -->
    <Transition name="slide-up">
      <div v-if="manageMode && selectedIds.length > 0" class="batch-action-bar">
        <span class="selected-count">{{ t('archives.batch.selected', { count: selectedIds.length }) }}</span>
        <div class="batch-actions">
          <RButton @click="openColorModal">
            <i class="ri-palette-line"></i> {{ t('archives.action.setColor') }}
          </RButton>
          <RButton :disabled="availableTargetStories.length === 0" @click="openMoveModal">
            <i class="ri-folder-transfer-line"></i> {{ t('archives.action.moveToStory') }}
          </RButton>
          <RButton type="danger" :loading="batchDeleting" @click="handleBatchDelete">
            <i class="ri-delete-bin-line"></i> {{ t('archives.action.delete') }}
          </RButton>
        </div>
      </div>
    </Transition>

    <!-- 移动到其它剧情弹窗 -->
    <RModal v-model="showMoveModal" :title="t('archives.modal.moveTitle')" width="480px">
      <div class="move-form">
        <p class="move-tip">{{ t('archives.modal.moveTip', { count: selectedIds.length }) }}</p>
        <div class="form-field">
          <label>{{ t('archives.modal.selectTarget') }}</label>
          <div v-if="availableTargetStories.length === 0" class="no-target">
            {{ t('archives.modal.noTarget') }}
          </div>
          <div v-else class="target-selector">
            <div
              v-for="story in availableTargetStories"
              :key="story.id"
              class="target-option"
              :class="{ selected: moveTargetId === story.id }"
              @click="moveTargetId = story.id"
            >
              <i :class="moveTargetId === story.id ? 'ri-checkbox-circle-fill' : 'ri-checkbox-blank-circle-line'"></i>
              <div class="target-info">
                <div class="target-title">{{ story.title }}</div>
                <div class="target-meta">{{ t('archives.story.records', { count: story.entry_count || 0 }) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <RButton @click="showMoveModal = false">{{ t('archives.action.cancel') }}</RButton>
        <RButton type="primary" :loading="batchMoving" :disabled="!moveTargetId" @click="handleBatchMove">
          {{ t('archives.action.confirmMove') }}
        </RButton>
      </template>
    </RModal>

    <!-- 设置背景色弹窗 -->
    <RModal v-model="showColorModal" :title="t('archives.modal.colorTitle')" width="400px">
      <div class="color-form">
        <p class="color-tip">{{ t('archives.modal.colorTip', { count: selectedIds.length }) }}</p>
        <div class="color-grid">
          <button
            v-for="color in defaultColors"
            :key="color"
            class="color-swatch"
            :class="{ selected: selectedColor === color }"
            :style="{ backgroundColor: color }"
            @click="selectedColor = color"
          />
          <button
            class="color-swatch clear-color"
            :class="{ selected: selectedColor === '' }"
            @click="selectedColor = ''"
          >
            <i class="ri-close-line"></i>
          </button>
        </div>
        <div class="custom-color">
          <label>{{ t('archives.modal.customColor') }}</label>
          <input type="color" v-model="selectedColor" class="color-input" />
        </div>
      </div>
      <template #footer>
        <RButton @click="showColorModal = false">{{ t('archives.action.cancel') }}</RButton>
        <RButton type="primary" :loading="batchUpdatingColor" @click="handleBatchUpdateColor">
          {{ t('archives.action.confirm') }}
        </RButton>
      </template>
    </RModal>
  </div>
</template>

<style scoped>
.story-list {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: sticky;
  top: 0;
  z-index: 50;
  background: var(--color-bg, #faf6f1);
  padding: 12px 0;
  margin: -12px 0 0 0;
}

.view-toggle {
  display: flex;
  gap: 4px;
  background: var(--color-card-bg, #f5f0eb);
  padding: 4px;
  border-radius: 8px;
}

.toggle-btn {
  padding: 8px 16px;
  border: none;
  background: transparent;
  border-radius: 6px;
  font-size: 13px;
  color: var(--color-text-secondary, #856a52);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.toggle-btn:hover {
  color: var(--color-primary, #4B3621);
}

.toggle-btn.active {
  background: var(--color-panel-bg, #fff);
  color: var(--color-primary, #4B3621);
  font-weight: 600;
  box-shadow: var(--shadow-sm, 0 2px 4px rgba(0,0,0,0.05));
}

.stories-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.story-card {
  cursor: pointer;
}

.story-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.story-date {
  font-size: 13px;
  color: var(--color-secondary);
}

.story-status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
}

.story-status.draft {
  background: var(--color-bg-secondary);
  color: var(--color-secondary);
}

.story-status.published {
  background: rgba(40, 167, 69, 0.1);
  color: #28a745;
}

.story-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-primary);
  margin: 0 0 8px 0;
}

.story-desc {
  font-size: 14px;
  color: var(--color-secondary);
  margin: 0 0 12px 0;
  line-height: 1.5;
}

.story-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.story-tags .tag {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  background: rgba(184, 115, 51, 0.1);
  color: #B87333;
}

.story-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.participants {
  display: flex;
  gap: 4px;
}

.participant {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-accent);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
}

.more {
  font-size: 12px;
  color: var(--color-secondary);
  margin-left: 4px;
}

.actions {
  display: flex;
  gap: 8px;
}

/* 时间线视图样式 */
.timeline-view {
  position: relative;
  padding: 40px 20px;
  min-height: 200px;
  width: 100%;
  background: linear-gradient(135deg, var(--color-card-bg, #f5f0eb) 0%, var(--color-background, #ebe4dc) 100%);
  border-radius: 16px;
}

.timeline-line {
  position: absolute;
  left: 50%;
  top: 0;
  bottom: 0;
  width: 4px;
  background: var(--color-primary, #4B3621);
  transform: translateX(-50%);
  opacity: 0.3;
  z-index: 1;
}

.timeline-group {
  position: relative;
  margin-bottom: 40px;
}

.timeline-month {
  text-align: center;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-primary, #4B3621);
  background: var(--color-card-bg, #f5f0eb);
  padding: 8px 20px;
  border-radius: 20px;
  display: block;
  width: fit-content;
  margin: 0 auto 32px auto;
  position: relative;
  z-index: 3;
}

.timeline-item {
  position: relative;
  width: 100%;
  min-height: 120px;
  margin-bottom: 40px;
}

/* 左侧卡片 */
.timeline-item.left .timeline-card {
  position: absolute;
  right: calc(50% + 40px);
  width: calc(50% - 60px);
}

/* 右侧卡片 */
.timeline-item.right .timeline-card {
  position: absolute;
  left: calc(50% + 40px);
  width: calc(50% - 60px);
}

.timeline-dot {
  width: 18px;
  height: 18px;
  background: var(--color-background, #EED9C4);
  border: 4px solid var(--color-primary, #4B3621);
  border-radius: 50%;
  position: absolute;
  left: 50%;
  top: 20px;
  transform: translateX(-50%);
  z-index: 2;
}

.timeline-item:hover .timeline-dot {
  background: var(--color-accent, #B87333);
  border-color: var(--color-accent, #B87333);
  box-shadow: 0 0 0 4px var(--color-primary-light, rgba(184, 115, 51, 0.2));
}

/* 连接线 - 从卡片到时间轴 */
.timeline-connector {
  position: absolute;
  top: 28px;
  height: 3px;
  background: var(--color-primary, #4B3621);
  opacity: 0.3;
  z-index: 1;
}

.timeline-item.left .timeline-connector {
  left: calc(50% - 40px);
  right: calc(50% + 9px);
  width: auto;
}

.timeline-item.right .timeline-connector {
  left: calc(50% + 9px);
  right: calc(50% - 40px);
  width: auto;
}

.timeline-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  padding: 20px;
  box-shadow: var(--shadow-md, 0 4px 16px rgba(75, 54, 33, 0.08));
  cursor: pointer;
  transition: all 0.3s;
  max-width: 100%;
}

.timeline-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg, 0 8px 24px rgba(75, 54, 33, 0.12));
}

.card-date {
  display: inline-block;
  background: var(--color-primary-light, rgba(184, 115, 51, 0.1));
  color: var(--color-accent, #B87333);
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 8px;
}

.card-title {
  font-size: 16px;
  color: var(--color-text-main, #2c1e12);
  font-weight: 600;
  margin: 0 0 8px 0;
}

.card-desc {
  font-size: 14px;
  color: var(--color-text-secondary, #665242);
  line-height: 1.6;
  margin: 0 0 12px 0;
}

.card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.card-tags .tag {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  background: var(--color-primary-light, rgba(184, 115, 51, 0.1));
  color: var(--color-accent, #B87333);
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid var(--color-border-light, #f0e6dc);
  padding-top: 12px;
}

.avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-accent, #D4A373);
  color: var(--color-text-light, #fff);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  margin-left: -8px;
  border: 2px solid var(--color-panel-bg, #fff);
}

.avatar:first-child {
  margin-left: 0;
}

.view-link {
  color: var(--color-accent, #B87333);
  font-size: 13px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 2px;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  background: var(--color-card-bg, #f5f0eb);
  color: var(--color-text-secondary, #856a52);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--color-border, #e5d4c1);
  color: var(--color-primary, #4B3621);
  transform: scale(1.1);
}

.action-btn.danger {
  color: #d32f2f;
}

.action-btn.danger:hover {
  background: #ffebee;
  color: #c62828;
}

/* 工具栏操作区 */
.toolbar-actions {
  display: flex;
  gap: 8px;
}

/* 管理模式样式 */
.card-checkbox,
.grid-checkbox {
  position: absolute;
  top: 12px;
  left: 12px;
  font-size: 22px;
  color: var(--color-secondary, #804030);
  z-index: 10;
  cursor: pointer;
}

.card-checkbox i,
.grid-checkbox i {
  transition: transform 0.2s;
}

.card-checkbox:hover i,
.grid-checkbox:hover i {
  transform: scale(1.1);
}

.timeline-card.manage-mode {
  position: relative;
  padding-left: 40px;
}

.story-card.manage-mode {
  position: relative;
}

.story-card.manage-mode .grid-checkbox {
  top: 16px;
  left: 16px;
}

/* 选中状态 */
.timeline-item.selected .timeline-card,
.story-card.selected {
  border: 2px solid var(--color-secondary, #804030);
  background: var(--color-primary-light, rgba(128, 64, 48, 0.05));
}

.timeline-item.selected .timeline-dot {
  background: var(--color-secondary, #804030);
  border-color: var(--color-secondary, #804030);
}

/* 批量操作栏 */
.batch-action-bar {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  padding: 12px 24px;
  display: flex;
  align-items: center;
  gap: 20px;
  box-shadow: var(--shadow-lg, 0 4px 24px rgba(75, 54, 33, 0.15));
  z-index: 100;
}

.selected-count {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary, #4B3621);
}

.batch-actions {
  display: flex;
  gap: 12px;
}

/* 滑入动画 */
.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.3s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}

/* 禁用状态 */
.toggle-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 移动弹窗样式 */
.move-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.move-tip {
  font-size: 14px;
  color: var(--color-text-secondary, #856a52);
  margin: 0;
  padding: 12px;
  background: var(--color-primary-light, rgba(184, 115, 51, 0.1));
  border-radius: 8px;
}

.target-selector {
  max-height: 280px;
  overflow-y: auto;
  border: 1px solid var(--color-border, #d1bfa8);
  border-radius: 8px;
}

.target-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  cursor: pointer;
  border-bottom: 1px solid var(--color-border-light, #f0e6dc);
  transition: background 0.2s;
}

.target-option:last-child {
  border-bottom: none;
}

.target-option:hover {
  background: var(--color-primary-light, rgba(184, 115, 51, 0.05));
}

.target-option.selected {
  background: var(--color-primary-light, rgba(184, 115, 51, 0.1));
}

.target-option i {
  font-size: 20px;
  color: var(--color-text-secondary, #856a52);
}

.target-option.selected i {
  color: var(--color-secondary, #804030);
}

.target-info {
  flex: 1;
  min-width: 0;
}

.target-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary, #4B3621);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.target-meta {
  font-size: 12px;
  color: var(--color-text-secondary, #856a52);
  margin-top: 2px;
}

.no-target {
  padding: 24px;
  text-align: center;
  color: var(--color-text-secondary, #856a52);
  font-size: 14px;
}

/* 颜色选择弹窗样式 */
.color-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.color-tip {
  font-size: 14px;
  color: var(--color-text-secondary, #856a52);
  margin: 0;
}

.color-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 8px;
}

.color-swatch {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  border: 2px solid transparent;
  cursor: pointer;
  transition: all 0.2s;
}

.color-swatch:hover {
  transform: scale(1.1);
}

.color-swatch.selected {
  border-color: var(--color-primary, #4B3621);
  box-shadow: 0 0 0 2px var(--color-panel-bg, #fff), 0 0 0 4px var(--color-primary, #4B3621);
}

.color-swatch.clear-color {
  background: var(--color-card-bg, #f5f0eb);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-secondary, #856a52);
  font-size: 18px;
}

.custom-color {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--color-border-light, #f0e6dc);
}

.custom-color label {
  font-size: 14px;
  color: var(--color-text-secondary, #856a52);
}

.color-input {
  width: 48px;
  height: 32px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  background: transparent;
}

.color-input::-webkit-color-swatch-wrapper {
  padding: 0;
}

.color-input::-webkit-color-swatch {
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 6px;
}
</style>
