<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listSponsors, type SponsorUser } from '@/api/user'

const router = useRouter()
const mounted = ref(false)

const sponsors = ref<SponsorUser[]>([])
const loadingSponsors = ref(false)
const specialSponsors = computed(() => sponsors.value.filter((s) => String(s.role || '').trim() === '特别赞助'))
const visibleSponsors = computed(() => sponsors.value.filter((s) => String(s.role || '').trim() !== '特别赞助'))

const openSourceThanks = [
  {
    name: 'Total RP 3',
    desc: '感谢 Total RP 3 作者的开源。',
    link: 'https://github.com/Total-RP/Total-RP-3',
  },
]

function getInitial(name: string) {
  const trimmed = name.trim()
  return trimmed ? trimmed.charAt(0) : '?'
}

async function loadSponsors() {
  loadingSponsors.value = true
  try {
    const res = await listSponsors()
    sponsors.value = Array.isArray(res.users) ? res.users : []
  } finally {
    loadingSponsors.value = false
  }
}

onMounted(() => {
  loadSponsors()
  setTimeout(() => mounted.value = true, 50)
})
</script>

<template>
  <div class="thanks-page" :class="{ 'animate-in': mounted }">
    <div class="thanks-shell">
      <header class="thanks-hero anim-item" style="--delay: 0">
        <div class="hero-badge">
          <span class="badge-dot"></span>
          Acknowledgments
        </div>
        <h1 class="hero-title">
          感谢那些让美好得以发生的伙伴们。
        </h1>
        <p class="hero-subtitle">
          每一行代码、每一次反馈、每一笔赞助，都是推动我们前行的燃料。这个页面致力于记录和感谢所有为本项目做出贡献的朋友。
        </p>
      </header>

      <section class="thanks-section anim-item" style="--delay: 1">
        <div class="section-header">
          <h2 class="section-title">
            <i class="ri-star-smile-line"></i>
            特别赞助
          </h2>
          <span class="section-pill">至尊合作伙伴</span>
        </div>
        <div v-if="loadingSponsors" class="section-note">加载中...</div>
        <div v-else-if="specialSponsors.length === 0" class="section-note">暂无特别赞助数据</div>
        <div v-else class="featured-grid">
          <div v-for="sponsor in specialSponsors" :key="`special-${sponsor.id}`" class="featured-card">
            <div class="featured-decoration"></div>
            <div class="featured-content">
              <div class="featured-avatar" aria-hidden="true">
                {{ getInitial(sponsor.username) }}
              </div>
              <div class="featured-body">
                <div class="featured-title">
                  <h3 :style="{ color: sponsor.name_color || undefined, fontWeight: sponsor.name_bold ? '700' : undefined }">{{ sponsor.username }}</h3>
                  <span class="featured-badge">特别赞助</span>
                </div>
                <p class="featured-desc">对 RPBox 的赞助与宣发支持，特别感谢。</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="thanks-section anim-item" style="--delay: 2">
        <div class="section-header">
          <h2 class="section-title">赞助支持</h2>
          <span class="section-note">来自云端名单</span>
        </div>
        <div v-if="loadingSponsors" class="section-note">加载中...</div>
        <div v-else-if="visibleSponsors.length === 0" class="section-note">暂无赞助者数据</div>
        <div v-else class="sponsor-grid">
          <div v-for="sponsor in visibleSponsors" :key="sponsor.id" class="sponsor-card">
            <div class="sponsor-avatar">{{ getInitial(sponsor.username) }}</div>
            <div class="sponsor-body">
              <div class="sponsor-title">
                <h4 :style="{ color: sponsor.name_color || undefined, fontWeight: sponsor.name_bold ? '700' : '600' }">
                  {{ sponsor.username }}
                </h4>
                <span class="sponsor-level">Lv{{ sponsor.sponsor_level || 1 }}</span>
              </div>
              <p class="sponsor-desc">{{ sponsor.role || '赞助支持' }}</p>
            </div>
          </div>
        </div>
      </section>

      <section class="thanks-section anim-item" style="--delay: 3">
        <div class="section-header">
          <h2 class="section-title">开源致谢</h2>
          <span class="section-note">Code Contributors</span>
        </div>
        <div class="opensource-list">
          <div v-for="item in openSourceThanks" :key="item.name" class="opensource-item">
            <div class="opensource-avatar">{{ getInitial(item.name) }}</div>
            <div class="opensource-body">
              <h4>{{ item.name }}</h4>
              <p>{{ item.desc }}</p>
            </div>
            <a :href="item.link" target="_blank" rel="noopener" class="opensource-link">
              <i class="ri-external-link-line"></i>
            </a>
          </div>
        </div>
      </section>

      <footer class="thanks-footer anim-item" style="--delay: 4">
        <p class="footer-quote">"Alone we can do so little; together we can do so much."</p>
        <button class="footer-btn" type="button" @click="router.push('/settings')">
          成为赞助者
        </button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.thanks-page {
  position: relative;
  min-height: 100%;
}

.thanks-page::before {
  content: '';
  position: absolute;
  inset: 0 0 auto 0;
  height: 260px;
  background: linear-gradient(180deg, #E6D0BA 0%, var(--color-main-bg, #EED9C4) 100%);
  pointer-events: none;
  z-index: 0;
}

.thanks-shell {
  position: relative;
  z-index: 1;
  max-width: 960px;
  margin: 0 auto;
  padding: 8px 0 32px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.thanks-hero {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 999px;
  background: rgba(128, 64, 48, 0.12);
  color: var(--color-primary, #804030);
  font-size: 11px;
  letter-spacing: 0.12em;
  font-weight: 700;
  text-transform: uppercase;
  width: fit-content;
}

.badge-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-highlight, #D2691E);
  box-shadow: 0 0 0 4px rgba(210, 105, 30, 0.2);
  animation: pulse 1.8s ease-in-out infinite;
}

.hero-title {
  font-size: 36px;
  line-height: 1.2;
  color: var(--color-text-main, #2C1810);
  font-weight: 700;
  margin: 0;
  font-family: 'Merriweather', 'Georgia', serif;
}

.hero-subtitle {
  font-size: 16px;
  line-height: 1.7;
  color: var(--color-text-secondary, #8C7B70);
  margin: 0;
  max-width: 640px;
}

.thanks-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid rgba(128, 64, 48, 0.1);
  padding-bottom: 8px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 700;
  color: var(--color-primary, #804030);
  margin: 0;
}

.section-title i {
  font-size: 18px;
}

.section-pill {
  background: rgba(210, 105, 30, 0.12);
  color: var(--color-highlight, #D2691E);
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}

.section-note {
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.featured-card {
  position: relative;
  background: var(--color-panel-bg, #FFFFFF);
  border-radius: 18px;
  padding: 28px;
  border: 1px solid #fff;
  box-shadow: var(--shadow-lg, 0 8px 32px rgba(75, 54, 33, 0.12));
  overflow: hidden;
}

.featured-decoration {
  position: absolute;
  top: -80px;
  right: -80px;
  width: 220px;
  height: 220px;
  background: radial-gradient(circle, rgba(212, 163, 115, 0.35) 0%, rgba(212, 163, 115, 0) 65%);
}

.featured-content {
  display: flex;
  align-items: flex-start;
  gap: 24px;
  position: relative;
  z-index: 1;
}

.featured-avatar {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-primary, #804030), var(--color-highlight, #D2691E));
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 700;
  flex-shrink: 0;
  box-shadow: 0 10px 24px rgba(128, 64, 48, 0.25);
}

.featured-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.featured-title h3 {
  font-size: 22px;
  margin: 0;
  color: var(--color-text-main, #2C1810);
  font-weight: 700;
  font-family: 'Merriweather', 'Georgia', serif;
}

.featured-badge {
  background: var(--color-highlight, #D2691E);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 999px;
}

.featured-desc {
  margin: 10px 0 0;
  color: var(--color-text-secondary, #8C7B70);
  line-height: 1.6;
}

.featured-meta {
  margin-top: 14px;
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.meta-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--color-primary, #804030);
  font-weight: 600;
  text-decoration: none;
}

.meta-link:hover {
  color: var(--color-highlight, #D2691E);
}

.sponsor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.sponsor-card {
  background: var(--color-panel-bg, #FFFFFF);
  border-radius: 14px;
  padding: 18px;
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(75, 54, 33, 0.05));
  border: 1px solid transparent;
  display: flex;
  gap: 14px;
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.sponsor-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md, 0 4px 20px rgba(75, 54, 33, 0.08));
  border-color: rgba(212, 163, 115, 0.4);
}

.sponsor-avatar {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: rgba(128, 64, 48, 0.12);
  color: var(--color-primary, #804030);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
}

.sponsor-title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.sponsor-title h4 {
  margin: 0;
  font-size: 15px;
  color: var(--color-text-main, #2C1810);
  font-weight: 600;
}

.sponsor-level {
  font-size: 10px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 999px;
  background: rgba(212, 163, 115, 0.2);
  color: var(--color-primary, #804030);
}

.sponsor-desc {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--color-text-secondary, #8C7B70);
  line-height: 1.5;
}

.sponsor-meta {
  margin-top: 10px;
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.opensource-list {
  background: var(--color-panel-bg, #FFFFFF);
  border-radius: 16px;
  border: 1px solid var(--color-border, #E8DCCF);
  overflow: hidden;
}

.opensource-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 18px;
  border-bottom: 1px solid rgba(232, 220, 207, 0.6);
}

.opensource-item:last-child {
  border-bottom: none;
}

.opensource-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(212, 163, 115, 0.2);
  color: var(--color-primary, #804030);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 12px;
}

.opensource-body h4 {
  margin: 0 0 4px;
  font-size: 14px;
  color: var(--color-text-main, #2C1810);
}

.opensource-body p {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.opensource-link {
  margin-left: auto;
  color: var(--color-text-secondary, #8C7B70);
  text-decoration: none;
  padding: 6px;
  border-radius: 50%;
}

.opensource-link:hover {
  color: var(--color-primary, #804030);
  background: rgba(255, 255, 255, 0.6);
}

.thanks-footer {
  padding-top: 24px;
  border-top: 1px solid rgba(128, 64, 48, 0.1);
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.footer-quote {
  margin: 0;
  font-style: italic;
  color: var(--color-text-secondary, #8C7B70);
  font-family: 'Merriweather', 'Georgia', serif;
}

.footer-btn {
  align-self: center;
  padding: 10px 22px;
  border-radius: 999px;
  border: none;
  background: var(--color-primary, #804030);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.footer-btn:hover {
  transform: translateY(-2px);
  background: var(--color-highlight, #D2691E);
  box-shadow: 0 12px 24px rgba(128, 64, 48, 0.2);
}

.anim-item {
  opacity: 0;
  transform: translateY(12px);
}

.animate-in .anim-item {
  animation: fadeUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(var(--delay) * 0.12s);
}

@keyframes fadeUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes pulse {
  0% {
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(210, 105, 30, 0.2);
  }
  70% {
    transform: scale(1.05);
    box-shadow: 0 0 0 8px rgba(210, 105, 30, 0);
  }
  100% {
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(210, 105, 30, 0);
  }
}

@media (max-width: 768px) {
  .thanks-shell {
    padding: 0 0 28px;
  }

  .hero-title {
    font-size: 28px;
  }

  .featured-content {
    flex-direction: column;
  }

  .section-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .sponsor-grid {
    grid-template-columns: 1fr;
  }

  .opensource-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .opensource-link {
    margin-left: 0;
  }
}
</style>
