<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listStories, deleteStory, type Story } from '@/api/story'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'
import RCard from '@/components/RCard.vue'

const emit = defineEmits<{
  create: []
  view: [id: number]
}>()

const loading = ref(false)
const stories = ref<Story[]>([])

async function loadStories() {
  loading.value = true
  try {
    const res = await listStories()
    stories.value = res.stories || []
  } catch (e) {
    console.error('加载失败:', e)
  } finally {
    loading.value = false
  }
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

onMounted(loadStories)

// 暴露方法供父组件调用
defineExpose({
  loadStories
})
</script>

<template>
  <div class="story-list">
    <REmpty v-if="!loading && stories.length === 0" description="暂无剧情记录">
      <RButton type="primary" @click="$emit('create')">创建第一个剧情</RButton>
    </REmpty>

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
  margin: 0 0 16px 0;
  line-height: 1.5;
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
</style>
