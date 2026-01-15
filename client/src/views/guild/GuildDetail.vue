<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGuild, leaveGuild, deleteGuild, listGuildMembers, type Guild, type GuildMember } from '@/api/guild'
import RButton from '@/components/RButton.vue'
import RCard from '@/components/RCard.vue'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const guild = ref<Guild | null>(null)
const myRole = ref('')
const members = ref<GuildMember[]>([])

const guildId = Number(route.params.id)

async function loadGuild() {
  loading.value = true
  try {
    const res = await getGuild(guildId)
    guild.value = res.guild
    myRole.value = res.my_role
    const membersRes = await listGuildMembers(guildId)
    members.value = membersRes.members || []
  } catch (e) {
    console.error('加载失败:', e)
  } finally {
    loading.value = false
  }
}

async function handleLeave() {
  if (!confirm('确定要退出公会吗？')) return
  try {
    await leaveGuild(guildId)
    router.push('/guild')
  } catch (e: any) {
    alert(e.message || '退出失败')
  }
}

async function handleDelete() {
  if (!confirm('确定要解散公会吗？此操作不可恢复！')) return
  try {
    await deleteGuild(guildId)
    router.push('/guild')
  } catch (e: any) {
    alert(e.message || '解散失败')
  }
}

function getRoleLabel(role: string): string {
  const map: Record<string, string> = { owner: '会长', admin: '管理员', member: '成员' }
  return map[role] || role
}

onMounted(loadGuild)
</script>

<template>
  <div class="detail-page">
    <div v-if="loading" class="loading">加载中...</div>
    <template v-else-if="guild">
      <div class="guild-header">
        <div class="guild-icon" :style="{ background: '#' + (guild.color || 'B87333') }">
          {{ guild.name.charAt(0) }}
        </div>
        <div class="guild-info">
          <h1>{{ guild.name }}</h1>
          <p>{{ guild.description || '暂无描述' }}</p>
          <div class="guild-meta">
            <span>成员: {{ guild.member_count }}</span>
            <span>剧情: {{ guild.story_count }}</span>
            <span>邀请码: {{ guild.invite_code }}</span>
          </div>
        </div>
        <div class="guild-actions">
          <RButton v-if="myRole !== 'owner'" type="danger" @click="handleLeave">退出</RButton>
          <RButton v-if="myRole === 'owner'" type="danger" @click="handleDelete">解散</RButton>
        </div>
      </div>

      <RCard title="成员列表" class="members-card">
        <div class="member-list">
          <div v-for="m in members" :key="m.id" class="member-item">
            <span class="member-name">{{ m.username }}</span>
            <span class="member-role">{{ getRoleLabel(m.role) }}</span>
          </div>
        </div>
      </RCard>
    </template>
  </div>
</template>

<style scoped>
.detail-page { padding: 24px; }
.loading { text-align: center; padding: 40px; color: #856a52; }

.guild-header {
  display: flex;
  gap: 20px;
  align-items: flex-start;
  margin-bottom: 24px;
  background: #fff;
  padding: 24px;
  border-radius: 16px;
}

.guild-icon {
  width: 80px;
  height: 80px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36px;
  font-weight: 600;
  color: #fff;
}

.guild-info { flex: 1; }
.guild-info h1 { font-size: 24px; color: #4B3621; margin: 0 0 8px 0; }
.guild-info p { font-size: 14px; color: #856a52; margin: 0 0 12px 0; }
.guild-meta { display: flex; gap: 16px; font-size: 13px; color: #856a52; }

.members-card { margin-top: 16px; }
.member-list { display: flex; flex-direction: column; gap: 8px; }
.member-item { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f0e6dc; }
.member-name { color: #4B3621; }
.member-role { font-size: 12px; color: #B87333; }
</style>
