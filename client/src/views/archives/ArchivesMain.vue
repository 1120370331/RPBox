<script setup lang="ts">
import { ref, onMounted } from 'vue'

const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

const stories = [
  { id: 1, date: '2023.11.24', title: '遗忘之森的低语：终章决战', content: '探险队终于抵达了森林深处的祭坛，古老的符文在月光下闪烁。随着仪式的启动，沉睡百年的守护者苏醒了。', avatars: ['L', 'A', 'K'], highlight: true },
  { id: 2, date: '2023.11.18', title: '银月港湾的午后茶会', content: '难得的休闲时光，公会成员聚集在银月港湾的露天酒馆。交换着最近的情报，也分享着各自旅途中的趣闻。', avatars: ['S', 'M', 'R', 'T'] },
  { id: 3, date: '2023.11.10', title: '暗夜公会：潜入作战', content: '为了获取敌对势力的情报，我们伪装成商队潜入了地下黑市。紧张的氛围，每一句话都可能是陷阱。', avatars: ['Z', 'B'] },
  { id: 4, date: '2023.11.02', title: '初遇：命运的齿轮', content: '在一个风雨交加的夜晚，流浪的骑士与寻找身世的魔法师在破旧的教堂相遇。', avatars: ['L', 'F'] },
]
</script>

<template>
  <div class="archives-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar anim-item" style="--delay: 0">
      <div class="page-title">
        <h1>我的剧情记录</h1>
        <p>记录每一个精彩瞬间，编织属于你的冒险史诗</p>
      </div>
      <button class="btn-create">
        <i class="ri-add-line"></i> 新建记录
      </button>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar anim-item" style="--delay: 1">
      <div class="filter-item">
        <i class="ri-community-line"></i> 全部公会
        <i class="ri-arrow-down-s-line"></i>
      </div>
      <div class="filter-item">
        <i class="ri-user-3-line"></i> 全部角色
        <i class="ri-arrow-down-s-line"></i>
      </div>
      <div class="filter-item">
        <i class="ri-search-line"></i> 搜索剧情...
      </div>
    </div>

    <!-- 时间轴 -->
    <div class="timeline-section anim-item" style="--delay: 2">
      <div class="timeline-line"></div>
      <div
        v-for="(story, index) in stories"
        :key="story.id"
        class="timeline-item"
        :class="{ highlight: story.highlight, left: index % 2 === 0, right: index % 2 === 1 }"
      >
        <div class="timeline-dot"></div>
        <div class="story-card">
          <div class="card-date">{{ story.date }}</div>
          <div class="card-title">{{ story.title }}</div>
          <div class="card-body">{{ story.content }}</div>
          <div class="card-footer">
            <div class="avatars">
              <div v-for="(a, i) in story.avatars" :key="i" class="avatar">{{ a }}</div>
            </div>
            <a class="view-detail">查看详情 <i class="ri-arrow-right-line"></i></a>
          </div>
        </div>
      </div>
    </div>
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
</style>
