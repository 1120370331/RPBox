<script setup lang="ts">
import { RouterLink } from 'vue-router'
import RButton from '@shared/components/RButton.vue'

const features = [
  { icon: 'ri-user-line', title: '人物卡云同步', desc: '跨设备备份同步你的 TRP3 人物卡' },
  { icon: 'ri-book-line', title: '剧情归档', desc: 'RP 剧情记录归档与回放' },
  { icon: 'ri-group-line', title: '社区分享', desc: '与其他 RP 玩家交流分享' },
  { icon: 'ri-shopping-bag-line', title: '道具市场', desc: 'TRP3 Extended 道具分享' }
]
</script>

<template>
  <div class="home">
    <!-- Hero -->
    <section class="hero">
      <h1 class="hero-title">打破封闭的 RP 生态</h1>
      <p class="hero-subtitle">迈向开放、互联的新时代</p>
      <p class="hero-desc">
        连接游戏内外，让数据自由流动，让创作不再孤立
      </p>
      <div class="hero-actions">
        <RouterLink to="/download">
          <RButton variant="primary" size="large">立即下载</RButton>
        </RouterLink>
        <RouterLink to="/community">
          <RButton variant="outline" size="large">进入社区</RButton>
        </RouterLink>
      </div>
    </section>

    <!-- Features -->
    <section class="features">
      <div class="features-grid">
        <div v-for="f in features" :key="f.title" class="feature-card">
          <i :class="f.icon" class="feature-icon"></i>
          <h3 class="feature-title">{{ f.title }}</h3>
          <p class="feature-desc">{{ f.desc }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.home {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
}

.hero {
  text-align: center;
  padding: 80px 0;
}

.hero-title {
  font-size: 48px;
  font-weight: bold;
  margin-bottom: 16px;
}

.hero-subtitle {
  font-size: 24px;
  color: var(--primary-color);
  margin-bottom: 24px;
}

.hero-desc {
  font-size: 18px;
  color: var(--text-secondary);
  margin-bottom: 40px;
}

.hero-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.features {
  padding: 60px 0;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.feature-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 32px 24px;
  text-align: center;
}

.feature-icon {
  font-size: 40px;
  color: var(--primary-color);
  margin-bottom: 16px;
}

.feature-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 8px;
}

.feature-desc {
  font-size: 14px;
  color: var(--text-secondary);
}

@media (max-width: 768px) {
  .features-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .hero-title {
    font-size: 32px;
  }
}
</style>
