<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listStories, deleteStory, type Story, type StoryFilterParams } from '@/api/story'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'
import StoryFilter from '@/components/StoryFilter.vue'

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
const filterParams = ref<StoryFilterParams>({})

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
  if (!confirm('确定要删除这个剧情吗？')) return
  try {
    await deleteStory(id)
    stories.value = stories.value.filter(s => s.id !== id)
  } catch (e) {
    console.error('删除失败:', e)
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

// 按月份分组剧情用于时间线视图
const groupedStories = computed(() => {
  const groups: { month: string; stories: Story[] }[] = []
  const monthMap = new Map<string, Story[]>()

  for (const story of stories.value) {
    const date = new Date(story.created_at)
    const monthKey = `${date.getFullYear()}年${date.getMonth() + 1}月`
    if (!monthMap.has(monthKey)) {
      monthMap.set(monthKey, [])
    }
    monthMap.get(monthKey)!.push(story)
  }

  for (const [month, items] of monthMap) {
    groups.push({ month, stories: items })
  }

  return groups
})

onMounted(() => {
  loadStories()
  loadTags()
  loadGuilds()
})

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
        >
          <i class="ri-time-line"></i> 时间线
        </button>
        <button
          class="toggle-btn"
          :class="{ active: viewMode === 'grid' }"
          @click="viewMode = 'grid'"
        >
          <i class="ri-grid-line"></i> 网格
        </button>
      </div>
      <RButton @click="showFilter = !showFilter">
        <i class="ri-filter-3-line"></i>
        {{ showFilter ? '隐藏筛选' : '显示筛选' }}
      </RButton>
    </div>

    <!-- 筛选面板 -->
    <StoryFilter
      v-if="showFilter"
      :tags="tags"
      :guilds="guilds"
      @filter="handleFilter"
      @reset="handleResetFilter"
    />

    <REmpty v-if="!loading && stories.length === 0" description="暂无剧情记录">
      <RButton type="primary" @click="$emit('create')">创建第一个剧情</RButton>
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
          :class="index % 2 === 0 ? 'left' : 'right'"
        >
          <div class="timeline-connector"></div>
          <div class="timeline-dot"></div>
          <div class="timeline-card" @click="$emit('view', story.id)">
            <div class="card-date">{{ formatDate(story.created_at) }}</div>
            <h3 class="card-title">{{ story.title }}</h3>
            <p class="card-desc">{{ story.description || '暂无描述' }}</p>
            <div v-if="getTags(story).length" class="card-tags">
              <span v-for="tag in getTags(story)" :key="tag" class="tag">{{ tag }}</span>
            </div>
            <div class="card-footer">
              <div class="participants">
                <span v-for="(p, i) in getParticipants(story).slice(0, 3)" :key="i" class="avatar">
                  {{ p.charAt(0) }}
                </span>
              </div>
              <div class="card-actions">
                <span class="view-link">查看详情 <i class="ri-arrow-right-s-line"></i></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 网格视图 -->
    <div v-else class="stories-grid">
      <RCard v-for="story in stories" :key="story.id" class="story-card" hoverable>
        <div class="story-header">
          <span class="story-date">{{ formatDate(story.created_at) }}</span>
          <span class="story-status" :class="story.status">
            {{ story.status === 'published' ? '已发布' : '草稿' }}
          </span>
        </div>
        <h3 class="story-title">{{ story.title }}</h3>
        <p class="story-desc">{{ story.description || '暂无描述' }}</p>
        <div v-if="getTags(story).length" class="story-tags">
          <span v-for="tag in getTags(story)" :key="tag" class="tag">{{ tag }}</span>
        </div>
        <div class="story-footer">
          <div class="participants">
            <span v-for="(p, i) in getParticipants(story).slice(0, 3)" :key="i" class="participant">
              {{ p.charAt(0) }}
            </span>
            <span v-if="getParticipants(story).length > 3" class="more">
              +{{ getParticipants(story).length - 3 }}
            </span>
          </div>
          <div class="actions">
            <RButton size="small" @click="$emit('view', story.id)">查看</RButton>
            <RButton size="small" type="danger" @click="handleDelete(story.id)">删除</RButton>
          </div>
        </div>
      </RCard>
    </div>
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
}

.view-toggle {
  display: flex;
  gap: 4px;
  background: #f5f0eb;
  padding: 4px;
  border-radius: 8px;
}

.toggle-btn {
  padding: 8px 16px;
  border: none;
  background: transparent;
  border-radius: 6px;
  font-size: 13px;
  color: #856a52;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.toggle-btn:hover {
  color: #4B3621;
}

.toggle-btn.active {
  background: #fff;
  color: #4B3621;
  font-weight: 600;
  box-shadow: 0 2px 4px rgba(0,0,0,0.05);
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
  background: linear-gradient(135deg, #f5f0eb 0%, #ebe4dc 100%);
  border-radius: 16px;
}

.timeline-line {
  position: absolute;
  left: 50%;
  top: 0;
  bottom: 0;
  width: 4px;
  background: #4B3621;
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
  color: #4B3621;
  background: #f5f0eb;
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
  background: #EED9C4;
  border: 4px solid #4B3621;
  border-radius: 50%;
  position: absolute;
  left: 50%;
  top: 20px;
  transform: translateX(-50%);
  z-index: 2;
}

.timeline-item:hover .timeline-dot {
  background: #B87333;
  border-color: #B87333;
  box-shadow: 0 0 0 4px rgba(184, 115, 51, 0.2);
}

/* 连接线 - 从卡片到时间轴 */
.timeline-connector {
  position: absolute;
  top: 28px;
  height: 3px;
  background: #4B3621;
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
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 4px 16px rgba(75, 54, 33, 0.08);
  cursor: pointer;
  transition: all 0.3s;
  max-width: 100%;
}

.timeline-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(75, 54, 33, 0.12);
}

.card-date {
  display: inline-block;
  background: rgba(184, 115, 51, 0.1);
  color: #B87333;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 8px;
}

.card-title {
  font-size: 16px;
  color: #2c1e12;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.card-desc {
  font-size: 14px;
  color: #665242;
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
  background: rgba(184, 115, 51, 0.1);
  color: #B87333;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid #f0e6dc;
  padding-top: 12px;
}

.avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #D4A373;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  margin-left: -8px;
  border: 2px solid #fff;
}

.avatar:first-child {
  margin-left: 0;
}

.view-link {
  color: #B87333;
  font-size: 13px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 2px;
}
</style>
