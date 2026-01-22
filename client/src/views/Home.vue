<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import RModal from '@/components/RModal.vue'

const router = useRouter()
const mounted = ref(false)
const showThanksModal = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

const quickActions = [
  { icon: 'ri-user-star-line', label: '人物卡同步', desc: '管理你的RP角色', route: '/sync' },
  { icon: 'ri-book-open-line', label: '剧情档案', desc: '记录精彩故事', route: '/archives' },
  { icon: 'ri-sword-line', label: '道具物品', desc: '浏览道具市场', route: '/market' },
  { icon: 'ri-settings-3-line', label: '系统设置', desc: '配置应用选项', route: '/settings' },
]
</script>

<template>
  <div class="home-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar anim-item" style="--delay: 0">
      <div class="breadcrumbs">
        <i class="ri-home-4-line"></i>
        <span class="current">工作台</span>
      </div>
      <div class="toolbar-actions">
        <button class="btn btn-secondary">
          <i class="ri-question-line"></i> 帮助
        </button>
        <button class="btn btn-secondary" @click="showThanksModal = true">
          <i class="ri-heart-3-line"></i> 特别鸣谢
        </button>
      </div>
    </div>

    <!-- 欢迎面板 -->
    <div class="welcome-panel anim-item" style="--delay: 1">
      <div class="welcome-content">
        <h1>欢迎回来</h1>
        <p>艾泽拉斯的故事，由你书写</p>
      </div>
      <div class="welcome-decoration">
        <i class="ri-quill-pen-line"></i>
      </div>
    </div>

    <!-- 快捷入口 -->
    <div class="quick-grid anim-item" style="--delay: 2">
      <div
        v-for="(action, index) in quickActions"
        :key="action.route"
        class="quick-card"
        @click="router.push(action.route)"
      >
        <div class="card-icon">
          <i :class="action.icon"></i>
        </div>
        <div class="card-info">
          <span class="card-label">{{ action.label }}</span>
          <span class="card-desc">{{ action.desc }}</span>
        </div>
        <i class="ri-arrow-right-s-line card-arrow"></i>
      </div>
    </div>

    <RModal v-model="showThanksModal" title="特别鸣谢" width="520px">
      <div class="thanks-modal">
        <p class="thanks-intro">感谢以下伙伴对 RPBox 的支持与贡献：</p>
        <div class="thanks-item featured">
          <div class="thanks-title">
            <i class="ri-heart-3-line"></i>
            厘米特·绿宝石
          </div>
          <p class="thanks-desc">对 RPBox 的赞助与宣发支持，特别感谢。</p>
        </div>
        <div class="thanks-item">
          <div class="thanks-title">
            <i class="ri-heart-3-line"></i>
            摩迪斯特雷德
          </div>
          <p class="thanks-desc">在初期给予大力宣发支持。</p>
        </div>
        <div class="thanks-item">
          <div class="thanks-title">
            <i class="ri-heart-3-line"></i>
            海人
          </div>
          <p class="thanks-desc">赞助（1月18日）。</p>
        </div>
        <div class="thanks-item">
          <div class="thanks-title">
            <i class="ri-heart-3-line"></i>
            <a href="https://github.com/Total-RP/Total-RP-3" target="_blank" rel="noopener">Total RP 3</a>
          </div>
          <p class="thanks-desc">感谢 Total RP 3 作者的开源。</p>
        </div>
      </div>
    </RModal>
  </div>
</template>

<style scoped>
.home-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 顶部工具栏 */
.top-toolbar {
  background-color: #FFFFFF;
  border-radius: 16px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #8C7B70;
}

.breadcrumbs i {
  font-size: 18px;
}

.breadcrumbs .current {
  color: #804030;
  font-weight: 600;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
}

.btn {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: transform 0.2s;
}

.btn:hover {
  transform: translateY(-2px);
}

.btn-secondary {
  background-color: rgba(128, 64, 48, 0.1);
  color: #804030;
}

/* 欢迎面板 */
.welcome-panel {
  background: linear-gradient(135deg, #804030 0%, #4B3621 100%);
  border-radius: 16px;
  padding: 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.15);
}

.welcome-content h1 {
  font-size: 28px;
  color: #FBF5EF;
  margin: 0 0 8px 0;
}

.welcome-content p {
  color: rgba(251, 245, 239, 0.7);
  font-size: 15px;
  margin: 0;
}

.welcome-decoration i {
  font-size: 64px;
  color: rgba(212, 163, 115, 0.3);
}

/* 快捷入口 */
.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.quick-card {
  background: #FFFFFF;
  border-radius: 16px;
  padding: 20px 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.quick-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 30px rgba(75, 54, 33, 0.1);
}

.card-icon {
  width: 48px;
  height: 48px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon i {
  font-size: 24px;
  color: #804030;
}

.card-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.card-label {
  font-size: 15px;
  font-weight: 600;
  color: #2C1810;
}

.card-desc {
  font-size: 13px;
  color: #8C7B70;
}

.card-arrow {
  font-size: 20px;
  color: #D4A373;
}

/* 特别鸣谢弹窗 */
.thanks-modal {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.thanks-intro {
  margin: 0;
  font-size: 14px;
  color: #4B3621;
}

.thanks-item {
  background: #FDFBF9;
  border: 1px solid #E8DCCF;
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.thanks-item.featured {
  background: #F6E6D6;
  border-color: #D8B58A;
  border-left: 4px solid #C44536;
  box-shadow: 0 6px 16px rgba(128, 64, 48, 0.12);
}

.thanks-item.featured .thanks-title {
  color: #5B2E1E;
  font-weight: 700;
}

.thanks-item.featured .thanks-title i {
  color: #A63D2E;
}

.thanks-item.featured .thanks-desc {
  color: #6F4A2C;
}

.thanks-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #2C1810;
}

.thanks-title i {
  color: #C44536;
  font-size: 18px;
}

.thanks-title a {
  color: #804030;
  text-decoration: none;
}

.thanks-title a:hover {
  text-decoration: underline;
}

.thanks-desc {
  margin: 0;
  font-size: 13px;
  color: #8C7B70;
}

.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}

@keyframes fadeUp {
  to { opacity: 1; transform: translateY(0); }
}
</style>
