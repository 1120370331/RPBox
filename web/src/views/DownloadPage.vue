<script setup lang="ts">
import { ref, onMounted } from 'vue'
import RButton from '@shared/components/RButton.vue'
import RCard from '@shared/components/RCard.vue'

const clientVersion = ref('0.2.7')
const addonVersion = ref('1.0.0')

onMounted(async () => {
  // 可从 API 获取最新版本
})
</script>

<template>
  <div class="download-page">
    <h1 class="page-title">下载 RPBox</h1>

    <div class="download-grid">
      <RCard class="download-card">
        <h2>桌面客户端</h2>
        <p class="version">版本 {{ clientVersion }}</p>
        <p class="desc">Windows 桌面应用，提供完整功能</p>
        <RButton variant="primary" size="large">
          <i class="ri-windows-fill"></i> Windows 下载
        </RButton>
      </RCard>

      <RCard class="download-card">
        <h2>配套插件</h2>
        <p class="version">RPBox_Addon v{{ addonVersion }}</p>
        <p class="desc">WoW 游戏内插件，用于聊天记录</p>
        <RButton variant="outline" size="large">
          <i class="ri-download-line"></i> 下载插件
        </RButton>
      </RCard>
    </div>
  </div>
</template>

<style scoped>
.download-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 60px 24px;
}

.page-title {
  text-align: center;
  margin-bottom: 48px;
}

.download-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 24px;
}

.download-card {
  text-align: center;
  padding: 40px 24px;
}

.download-card h2 {
  margin-bottom: 8px;
}

.version {
  color: var(--primary-color);
  margin-bottom: 16px;
}

.desc {
  color: var(--text-secondary);
  margin-bottom: 24px;
}

@media (max-width: 600px) {
  .download-grid {
    grid-template-columns: 1fr;
  }
}
</style>
