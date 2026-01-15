<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listGuilds, joinGuild, type Guild } from '@/api/guild'
import RButton from '@/components/RButton.vue'
import RCard from '@/components/RCard.vue'
import REmpty from '@/components/REmpty.vue'
import RModal from '@/components/RModal.vue'
import RInput from '@/components/RInput.vue'

const router = useRouter()
const loading = ref(false)
const guilds = ref<Guild[]>([])
const showJoinModal = ref(false)
const inviteCode = ref('')
const joining = ref(false)

async function loadGuilds() {
  loading.value = true
  try {
    const res = await listGuilds()
    guilds.value = res.guilds || []
  } finally {
    loading.value = false
  }
}

async function handleJoin() {
  if (!inviteCode.value.trim()) return
  joining.value = true
  try {
    await joinGuild(inviteCode.value)
    showJoinModal.value = false
    inviteCode.value = ''
    loadGuilds()
  } catch (e) {
    console.error('加入失败:', e)
  } finally {
    joining.value = false
  }
}

function getRoleLabel(role?: string): string {
  const map: Record<string, string> = {
    owner: '会长',
    admin: '管理员',
    member: '成员'
  }
  return map[role || ''] || ''
}

onMounted(loadGuilds)
</script>

<template>
  <div class="guild-page">
    <div class="page-header">
      <h1>我的公会</h1>
      <div class="header-actions">
        <RButton @click="showJoinModal = true">加入公会</RButton>
        <RButton type="primary" @click="router.push('/guild/create')">创建公会</RButton>
      </div>
    </div>

    <REmpty v-if="!loading && guilds.length === 0" description="暂未加入任何公会" />

    <div v-else class="guild-grid">
      <RCard v-for="guild in guilds" :key="guild.id" class="guild-card" hoverable @click="router.push(`/guild/${guild.id}`)">
        <div class="guild-icon" :style="{ background: '#' + (guild.color || 'B87333') }">
          {{ guild.name.charAt(0) }}
        </div>
        <div class="guild-info">
          <h3>{{ guild.name }}</h3>
          <p>{{ guild.description || '暂无描述' }}</p>
          <div class="guild-meta">
            <span><i class="ri-user-line"></i> {{ guild.member_count }}</span>
            <span><i class="ri-book-line"></i> {{ guild.story_count }}</span>
            <span class="role-badge">{{ getRoleLabel(guild.my_role) }}</span>
          </div>
        </div>
      </RCard>
    </div>

    <RModal v-model="showJoinModal" title="加入公会" width="400px">
      <RInput v-model="inviteCode" placeholder="输入邀请码" />
      <template #footer>
        <RButton @click="showJoinModal = false">取消</RButton>
        <RButton type="primary" :loading="joining" @click="handleJoin">加入</RButton>
      </template>
    </RModal>
  </div>
</template>

<style scoped>
.guild-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-header h1 {
  font-size: 24px;
  color: #4B3621;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.guild-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.guild-card {
  display: flex;
  gap: 16px;
  cursor: pointer;
}

.guild-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
}

.guild-info {
  flex: 1;
  min-width: 0;
}

.guild-info h3 {
  font-size: 16px;
  color: #4B3621;
  margin: 0 0 4px 0;
}

.guild-info p {
  font-size: 13px;
  color: #856a52;
  margin: 0 0 8px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.guild-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #856a52;
}

.guild-meta i {
  margin-right: 4px;
}

.role-badge {
  background: rgba(184, 115, 51, 0.1);
  color: #B87333;
  padding: 2px 8px;
  border-radius: 10px;
}
</style>
