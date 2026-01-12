<script setup lang="ts">
import { ref, onMounted } from 'vue'

const mounted = ref(false)
const activeTab = ref('guilds')

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

const tabs = [
  { id: 'guilds', icon: 'ri-team-line', label: '公会' },
  { id: 'events', icon: 'ri-calendar-event-line', label: '活动' },
  { id: 'posts', icon: 'ri-article-line', label: '帖子' },
]

const guilds = [
  { id: 1, name: '银月骑士团', members: 128, desc: '守护艾泽拉斯的荣耀骑士', online: 24 },
  { id: 2, name: '暗夜精灵议会', members: 86, desc: '月神艾露恩的虔诚信徒', online: 12 },
  { id: 3, name: '铁炉堡矿工协会', members: 64, desc: '挖掘艾泽拉斯的宝藏', online: 8 },
]
</script>

<template>
  <div class="community-page" :class="{ 'animate-in': mounted }">
    <!-- 标题 -->
    <h1 class="page-title anim-item" style="--delay: 0">社区广场</h1>

    <!-- 标签页 -->
    <div class="tab-container anim-item" style="--delay: 1">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="tab-item"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        <i :class="tab.icon"></i>
        <span>{{ tab.label }}</span>
      </div>
    </div>

    <!-- 公会列表 -->
    <div class="guild-list anim-item" style="--delay: 2">
      <div v-for="guild in guilds" :key="guild.id" class="guild-card">
        <div class="guild-avatar">{{ guild.name.charAt(0) }}</div>
        <div class="guild-info">
          <h3>{{ guild.name }}</h3>
          <p>{{ guild.desc }}</p>
          <div class="guild-stats">
            <span><i class="ri-group-line"></i> {{ guild.members }} 成员</span>
            <span class="online"><i class="ri-checkbox-blank-circle-fill"></i> {{ guild.online }} 在线</span>
          </div>
        </div>
        <button class="join-btn">加入</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.community-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-title {
  font-size: 42px;
  color: #4B3621;
  margin-bottom: 8px;
}

.tab-container {
  background: #4B3621;
  border-radius: 16px;
  padding: 8px;
  display: flex;
  gap: 8px;
}

.tab-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  border-radius: 12px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 600;
  color: #EED9C4;
  transition: all 0.3s;
}

.tab-item.active {
  background: #EED9C4;
  color: #4B3621;
}

.guild-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.guild-card {
  background: #fff;
  border-radius: 16px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.guild-avatar {
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #B87333, #804030);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 700;
  color: #fff;
}

.guild-info { flex: 1; }
.guild-info h3 { font-size: 18px; color: #2C1810; margin-bottom: 4px; }
.guild-info p { font-size: 14px; color: #8D7B68; margin-bottom: 8px; }

.guild-stats {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #8D7B68;
}

.guild-stats .online { color: #5B8C5A; }
.guild-stats .online i { font-size: 8px; margin-right: 4px; }

.join-btn {
  padding: 10px 24px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
