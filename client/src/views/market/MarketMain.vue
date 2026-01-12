<script setup lang="ts">
import { ref, onMounted } from 'vue'

const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

const categories = ['全部', '武器', '护甲', '消耗品', '环境', '其他']
const activeCategory = ref('全部')

const items = [
  { id: 1, name: '霓虹武士刀', creator: 'NeonMaster', desc: '一把充满赛博朋克风格的武士刀，挥动时带有发光的粒子轨迹。', tags: ['武器', '科幻'], downloads: 12405, rating: 4.8 },
  { id: 2, name: '龙血药剂', creator: 'AlchemyQueen', desc: '瞬间恢复大量生命值，并提供短暂的火焰抗性。', tags: ['消耗品', '魔法'], downloads: 8332, rating: 5.0 },
  { id: 3, name: '维多利亚煤气灯', creator: 'SteamPunkJoe', desc: '经典的街头煤气灯模型，拥有真实的闪烁火光效果。', tags: ['环境', '装饰'], downloads: 5621, rating: 4.2 },
  { id: 4, name: '圣光塔盾', creator: 'PaladinX', desc: '坚不可摧的重型盾牌，表面镀金，能在黑暗中发出微弱的光芒。', tags: ['护甲', '重装'], downloads: 10998, rating: 4.7 },
  { id: 5, name: '禁忌卷轴', creator: 'MageElder', desc: '记载着古老咒语的羊皮卷，展开时会有漂浮的符文特效。', tags: ['消耗品', '剧情'], downloads: 6700, rating: 4.9 },
  { id: 6, name: '暗影匕首', creator: 'RogueOne', desc: '淬毒的匕首，攻击时有一定几率造成持续伤害。', tags: ['武器', '近战'], downloads: 9870, rating: 4.6 },
]
</script>

<template>
  <div class="market-page" :class="{ 'animate-in': mounted }">
    <!-- 头部 -->
    <div class="header anim-item" style="--delay: 0">
      <h1>道具市场</h1>
      <div class="search-box">
        <input type="text" placeholder="搜索道具名称、类型或标签..." />
        <i class="ri-search-line"></i>
      </div>
    </div>

    <!-- 筛选 -->
    <div class="filter-bar anim-item" style="--delay: 1">
      <span
        v-for="cat in categories"
        :key="cat"
        class="tag"
        :class="{ active: activeCategory === cat }"
        @click="activeCategory = cat"
      >{{ cat }}</span>
    </div>

    <!-- 卡片网格 -->
    <div class="card-grid anim-item" style="--delay: 2">
      <div v-for="item in items" :key="item.id" class="card">
        <div class="card-image"></div>
        <div class="card-content">
          <h3>{{ item.name }}</h3>
          <p class="creator">by {{ item.creator }}</p>
          <p class="desc">{{ item.desc }}</p>
          <div class="tags">
            <span v-for="t in item.tags" :key="t" class="mini-tag">{{ t }}</span>
          </div>
          <div class="card-footer">
            <span class="stat"><i class="ri-download-line"></i> {{ item.downloads }}</span>
            <span class="rating"><i class="ri-star-fill"></i> {{ item.rating }}</span>
          </div>
          <button class="import-btn"><i class="ri-download-line"></i> 一键导入</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.market-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header {
  text-align: center;
  padding: 20px 0;
}

.header h1 {
  font-size: 36px;
  color: #3E2723;
  margin-bottom: 20px;
}

.search-box {
  position: relative;
  max-width: 500px;
  margin: 0 auto;
}

.search-box input {
  width: 100%;
  height: 48px;
  border-radius: 24px;
  border: none;
  padding: 0 48px 0 20px;
  font-size: 15px;
  box-shadow: 0 4px 12px rgba(184,115,51,0.15);
}

.search-box i {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 20px;
  color: #B87333;
}

.filter-bar {
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: wrap;
}

.tag {
  padding: 8px 18px;
  border-radius: 20px;
  background: rgba(255,255,255,0.6);
  cursor: pointer;
  font-size: 14px;
  color: #5D4037;
  border: 1px solid transparent;
}

.tag.active {
  background: #B87333;
  color: #fff;
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}

.card {
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 8px 20px rgba(93,64,55,0.05);
  transition: transform 0.3s;
}

.card:hover {
  transform: translateY(-6px);
}

.card-image {
  height: 140px;
  background: linear-gradient(135deg, #D4A373 0%, #8C7B70 100%);
}

.card-content {
  padding: 20px;
}

.card-content h3 {
  font-size: 18px;
  color: #3E2723;
  margin-bottom: 4px;
}

.creator {
  font-size: 13px;
  color: #999;
  margin-bottom: 8px;
}

.desc {
  font-size: 14px;
  color: #666;
  line-height: 1.5;
  margin-bottom: 12px;
}

.tags { display: flex; gap: 6px; margin-bottom: 12px; }
.mini-tag {
  font-size: 12px;
  padding: 3px 8px;
  background: #F5F0EB;
  color: #795548;
  border-radius: 4px;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #999;
  margin-bottom: 12px;
}

.rating { color: #FFB300; }

.import-btn {
  width: 100%;
  height: 40px;
  border: none;
  border-radius: 10px;
  background: #B87333;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
