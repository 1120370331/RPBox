<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import request from '@/api/request'
import AchievementMedal from '@/components/AchievementMedal.vue'
import RModal from '@/components/RModal.vue'
import {
  ACHIEVEMENT_CATEGORY_META,
  ACHIEVEMENT_RARITY_META,
  type AchievementDefinition,
} from '@/data/achievements'
import {
  buildAchievementEntries,
  buildAchievementProgressContext,
  pickFeaturedAchievement,
  summarizeAchievementRarities,
} from '@/utils/achievementProgress'

const route = useRoute()
const router = useRouter()

const userId = computed(() => String(route.params.id || ''))
const userProfile = ref<Record<string, unknown> | null>(null)
const userGuilds = ref<Array<Record<string, unknown>>>([])
const loading = ref(true)
const selectedAchievementId = ref<string | null>(null)
const showAchievementDetail = ref(false)

const sponsorLevel = computed(() => {
  const level = Number(userProfile.value?.sponsor_level)
  if (Number.isFinite(level) && level > 0) return level
  return userProfile.value?.is_sponsor ? 2 : 0
})

const achievementProgressContext = computed(() => buildAchievementProgressContext({
  profile: userProfile.value,
  guilds: userGuilds.value,
  sponsorLevel: sponsorLevel.value,
}))

const achievementEntries = computed(() => buildAchievementEntries(achievementProgressContext.value))
const earnedAchievementCount = computed(() => achievementEntries.value.filter((entry) => entry.progress.earned).length)
const achievementRaritySummary = computed(() => summarizeAchievementRarities(achievementEntries.value))
const featuredAchievementEntry = computed(() => pickFeaturedAchievement(achievementEntries.value))
const nextAchievementEntry = computed(() => (
  achievementEntries.value
    .filter((entry) => !entry.progress.earned)
    .sort((a, b) => b.progress.percent - a.progress.percent || a.definition.threshold - b.definition.threshold)[0] || null
))

const selectedAchievementEntry = computed(() => {
  if (!selectedAchievementId.value) return null
  return achievementEntries.value.find((entry) => entry.definition.id === selectedAchievementId.value) || null
})

const pageTitle = computed(() => {
  const username = userProfile.value?.username
  return typeof username === 'string' && username ? `${username} 的成就` : '成就列表'
})

onMounted(() => {
  void loadAchievements()
})

watch(userId, () => {
  void loadAchievements()
})

async function loadAchievements() {
  if (!userId.value) return
  loading.value = true
  try {
    const [profileResult, guildResult] = await Promise.allSettled([
      request.get<Record<string, unknown>>(`/users/${userId.value}`),
      request.get<{ guilds: Array<Record<string, unknown>> }>(`/users/${userId.value}/guilds`),
    ])

    if (profileResult.status === 'fulfilled') {
      userProfile.value = profileResult.value
    }

    if (guildResult.status === 'fulfilled') {
      userGuilds.value = Array.isArray(guildResult.value.guilds) ? guildResult.value.guilds : []
    } else {
      userGuilds.value = []
    }
  } catch (error) {
    console.error('加载成就列表失败:', error)
  } finally {
    loading.value = false
  }
}

function openAchievementDetail(achievement: AchievementDefinition) {
  selectedAchievementId.value = achievement.id
  showAchievementDetail.value = true
}
</script>

<template>
  <div class="achievement-page">
    <div class="achievement-bg"></div>

    <div class="achievement-shell">
      <button type="button" class="back-btn" @click="router.back()">
        <i class="ri-arrow-left-line"></i>
        返回
      </button>

      <div v-if="loading" class="loading">加载中...</div>

      <template v-else>
        <section class="achievement-hero-panel">
          <div class="hero-copy">
            <span>Achievement Codex</span>
            <h1>{{ pageTitle }}</h1>
            <p>这里汇总所有成就的解锁状态、稀有度和当前进度。勋章轮廓由类型决定，边缘材质由稀有度决定。</p>
          </div>
          <div class="hero-score">
            <strong>{{ earnedAchievementCount }}</strong>
            <span>/ {{ achievementEntries.length }}</span>
            <small>已获得</small>
          </div>
        </section>

        <section class="achievement-feature-row">
          <button
            v-if="featuredAchievementEntry"
            type="button"
            class="feature-card"
            :class="{ earned: featuredAchievementEntry.progress.earned }"
            @click="openAchievementDetail(featuredAchievementEntry.definition)"
          >
            <AchievementMedal
              :achievement="featuredAchievementEntry.definition"
              :earned="featuredAchievementEntry.progress.earned"
              size="lg"
            />
            <span>
              <small>{{ featuredAchievementEntry.progress.earned ? '代表成就' : '起步目标' }}</small>
              <strong>{{ featuredAchievementEntry.definition.title }}</strong>
              <em>{{ featuredAchievementEntry.progress.label }}</em>
            </span>
          </button>

          <button
            v-if="nextAchievementEntry"
            type="button"
            class="feature-card next"
            @click="openAchievementDetail(nextAchievementEntry.definition)"
          >
            <AchievementMedal
              :achievement="nextAchievementEntry.definition"
              :earned="nextAchievementEntry.progress.earned"
              size="lg"
            />
            <span>
              <small>下一个目标</small>
              <strong>{{ nextAchievementEntry.definition.title }}</strong>
              <em>{{ nextAchievementEntry.progress.label }}</em>
            </span>
          </button>
        </section>

        <section class="rarity-strip">
          <div
            v-for="summary in achievementRaritySummary"
            :key="summary.rarity"
            class="rarity-pill"
            :style="{
              '--rarity-edge': ACHIEVEMENT_RARITY_META[summary.rarity].edge,
              '--rarity-glow': ACHIEVEMENT_RARITY_META[summary.rarity].glow,
            }"
          >
            <span>{{ summary.label }}</span>
            <strong>{{ summary.earned }}/{{ summary.total }}</strong>
          </div>
        </section>

        <section class="achievement-grid">
          <button
            v-for="entry in achievementEntries"
            :key="entry.definition.id"
            type="button"
            class="achievement-tile"
            :class="{ earned: entry.progress.earned }"
            @click="openAchievementDetail(entry.definition)"
          >
            <AchievementMedal
              :achievement="entry.definition"
              :earned="entry.progress.earned"
              size="lg"
            />
            <span class="achievement-tile__copy">
              <strong>{{ entry.definition.title }}</strong>
              <span>{{ entry.definition.condition }}</span>
            </span>
            <span class="achievement-tile__meta">
              <em>{{ ACHIEVEMENT_RARITY_META[entry.definition.rarity].label }}</em>
              <b>{{ entry.progress.label }}</b>
            </span>
          </button>
        </section>
      </template>
    </div>

    <RModal
      v-model="showAchievementDetail"
      :title="selectedAchievementEntry ? selectedAchievementEntry.definition.title : '成就详情'"
      width="560px"
    >
      <div v-if="selectedAchievementEntry" class="achievement-detail">
        <AchievementMedal
          :achievement="selectedAchievementEntry.definition"
          :earned="selectedAchievementEntry.progress.earned"
          size="lg"
        />
        <div class="achievement-detail__body">
          <div class="achievement-detail__tags">
            <span
              class="achievement-detail__rarity"
              :style="{ '--rarity-edge': ACHIEVEMENT_RARITY_META[selectedAchievementEntry.definition.rarity].edge }"
            >
              {{ ACHIEVEMENT_RARITY_META[selectedAchievementEntry.definition.rarity].label }}
            </span>
            <span>{{ ACHIEVEMENT_CATEGORY_META[selectedAchievementEntry.definition.category].label }}</span>
          </div>
          <h3>{{ selectedAchievementEntry.definition.title }}</h3>
          <p>{{ selectedAchievementEntry.definition.condition }}</p>
          <div class="achievement-detail__progress">
            <div class="achievement-detail__progress-meta">
              <span>{{ selectedAchievementEntry.progress.earned ? '已获得' : '进度' }}</span>
              <strong>{{ selectedAchievementEntry.progress.label }}</strong>
            </div>
            <div class="achievement-detail__track">
              <div
                class="achievement-detail__fill"
                :style="{ width: `${selectedAchievementEntry.progress.percent}%` }"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </RModal>
  </div>
</template>

<style scoped>
.achievement-page {
  position: relative;
  min-height: 100vh;
  padding: 28px;
  color: var(--color-text-main, #4B3621);
  overflow: hidden;
}

.achievement-bg {
  position: fixed;
  inset: 0;
  pointer-events: none;
  background:
    radial-gradient(circle at 12% 12%, color-mix(in srgb, var(--color-accent, #B87333) 18%, transparent), transparent 30%),
    radial-gradient(circle at 86% 18%, rgba(255, 190, 219, 0.24), transparent 28%),
    radial-gradient(circle at 78% 82%, rgba(90, 183, 255, 0.18), transparent 30%);
  z-index: 0;
}

.achievement-shell {
  position: relative;
  z-index: 1;
  max-width: 1180px;
  margin: 0 auto;
}

.back-btn {
  border: none;
  background: transparent;
  color: var(--color-text-secondary, #8C7B70);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 0;
  margin-bottom: 12px;
  font-weight: 700;
  cursor: pointer;
}

.loading {
  padding: 80px 0;
  text-align: center;
  color: var(--color-text-secondary, #8C7B70);
}

.achievement-hero-panel {
  position: relative;
  overflow: hidden;
  border-radius: 24px;
  padding: 30px;
  border: 1px solid color-mix(in srgb, var(--color-border, #E8DCC8) 72%, transparent);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--color-panel-bg, #FFF9F0) 88%, transparent), color-mix(in srgb, var(--color-card-bg, #F8EEDF) 82%, transparent)),
    radial-gradient(circle at 92% 20%, rgba(255, 178, 62, 0.2), transparent 30%);
  box-shadow: var(--shadow-md, 0 20px 42px -30px rgba(75, 54, 33, 0.32));
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 28px;
}

.hero-copy span {
  display: inline-flex;
  margin-bottom: 8px;
  color: var(--color-accent, #B87333);
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.hero-copy h1 {
  margin: 0;
  color: var(--color-text-main, #4B3621);
  font-size: clamp(32px, 4vw, 52px);
  line-height: 1.04;
  letter-spacing: -0.04em;
}

.hero-copy p {
  max-width: 680px;
  margin: 14px 0 0;
  color: var(--color-text-secondary, #8C7B70);
  line-height: 1.8;
}

.hero-score {
  min-width: 148px;
  padding: 18px;
  border-radius: 20px;
  text-align: right;
  background: rgba(75, 54, 33, 0.08);
  border: 1px solid rgba(184, 115, 51, 0.16);
}

.hero-score strong {
  font-size: 44px;
  line-height: 1;
  color: var(--color-accent, #B87333);
}

.hero-score span {
  color: var(--color-text-secondary, #8C7B70);
  font-size: 18px;
  font-weight: 800;
}

.hero-score small {
  display: block;
  margin-top: 6px;
  color: var(--color-text-muted, #9C8E82);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.achievement-feature-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 16px;
}

.feature-card {
  min-width: 0;
  border: 1px solid rgba(184, 115, 51, 0.14);
  border-radius: 20px;
  padding: 16px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.62), rgba(255, 255, 255, 0.3)),
    radial-gradient(circle at 10% 50%, rgba(255, 178, 62, 0.15), transparent 38%);
  color: var(--color-text-main, #4B3621);
  display: flex;
  align-items: center;
  gap: 16px;
  text-align: left;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    background 0.2s ease;
}

.feature-card.next {
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.52), rgba(255, 255, 255, 0.26)),
    radial-gradient(circle at 10% 50%, rgba(90, 183, 255, 0.15), transparent 38%);
}

.feature-card:hover {
  transform: translateY(-2px);
  border-color: rgba(184, 115, 51, 0.28);
}

.feature-card span {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.feature-card small {
  color: var(--color-accent, #B87333);
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.feature-card strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 19px;
}

.feature-card em {
  color: var(--color-text-secondary, #8C7B70);
  font-size: 13px;
  font-style: normal;
}

.rarity-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}

.rarity-pill {
  --rarity-edge: #B87333;
  --rarity-glow: rgba(184, 115, 51, 0.18);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 11px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--rarity-edge) 12%, rgba(255, 255, 255, 0.72));
  border: 1px solid color-mix(in srgb, var(--rarity-edge) 42%, transparent);
  color: var(--color-text-main, #4B3621);
  box-shadow: 0 8px 18px -16px var(--rarity-glow);
  font-size: 12px;
}

.rarity-pill strong {
  color: var(--rarity-edge);
}

.achievement-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  margin-top: 18px;
}

.achievement-tile {
  min-width: 0;
  border: 1px solid rgba(184, 115, 51, 0.12);
  border-radius: 18px;
  padding: 16px 10px 13px;
  background: rgba(255, 255, 255, 0.38);
  color: var(--color-text-main, #4B3621);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    background 0.2s ease;
}

.achievement-tile:hover {
  transform: translateY(-3px);
  border-color: rgba(184, 115, 51, 0.28);
  background: rgba(255, 255, 255, 0.66);
}

.achievement-tile.earned {
  background:
    radial-gradient(circle at 50% 0%, rgba(255, 214, 135, 0.18), transparent 42%),
    rgba(255, 255, 255, 0.58);
}

.achievement-tile__copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  text-align: center;
}

.achievement-tile__copy strong,
.achievement-tile__copy span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.achievement-tile__copy strong {
  color: var(--color-text-main, #4B3621);
  font-size: 13px;
}

.achievement-tile__copy span {
  color: var(--color-text-secondary, #8C7B70);
  font-size: 11px;
}

.achievement-tile__meta {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid rgba(184, 115, 51, 0.1);
  font-size: 11px;
}

.achievement-tile__meta em {
  color: var(--color-text-muted, #9C8E82);
  font-style: normal;
}

.achievement-tile__meta b {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-accent, #B87333);
  font-weight: 800;
}

.achievement-detail {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 22px;
  align-items: start;
}

.achievement-detail__body {
  min-width: 0;
}

.achievement-detail__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.achievement-detail__tags span {
  padding: 5px 9px;
  border-radius: 999px;
  background: var(--color-card-bg, #F2E6D8);
  color: var(--color-text-secondary, #8C7B70);
  font-size: 11px;
  font-weight: 700;
}

.achievement-detail__rarity {
  --rarity-edge: #B87333;
  background: color-mix(in srgb, var(--rarity-edge) 14%, #fff) !important;
  color: var(--rarity-edge) !important;
}

.achievement-detail h3 {
  margin: 0 0 8px;
  font-size: 22px;
  color: var(--color-text-main, #4B3621);
}

.achievement-detail p {
  margin: 0;
  color: var(--color-text-secondary, #8C7B70);
  line-height: 1.7;
}

.achievement-detail__progress {
  margin: 16px 0;
}

.achievement-detail__progress-meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  color: var(--color-text-secondary, #8C7B70);
  font-size: 12px;
}

.achievement-detail__progress-meta strong {
  color: var(--color-accent, #B87333);
}

.achievement-detail__track {
  height: 9px;
  border-radius: 999px;
  background: rgba(75, 54, 33, 0.1);
  overflow: hidden;
}

.achievement-detail__fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #D4A373, #FFB23E);
}

@media (max-width: 1100px) {
  .achievement-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 860px) {
  .achievement-page {
    padding: 18px;
  }

  .achievement-hero-panel,
  .achievement-feature-row {
    grid-template-columns: 1fr;
  }

  .achievement-hero-panel {
    flex-direction: column;
  }

  .hero-score {
    width: 100%;
    text-align: left;
  }

  .achievement-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .achievement-page {
    padding: 14px;
  }

  .achievement-hero-panel {
    padding: 22px 16px;
  }

  .feature-card {
    align-items: flex-start;
  }

  .achievement-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .achievement-detail {
    grid-template-columns: 1fr;
  }
}
</style>
