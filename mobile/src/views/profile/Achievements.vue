<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getUserInfo, type UserInfo } from '@/api/user'
import AchievementMedal from '@/components/AchievementMedal.vue'
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

const router = useRouter()

const userInfo = ref<UserInfo | null>(null)
const loading = ref(true)
const selectedAchievementId = ref<string | null>(null)

const achievementContext = computed(() => buildAchievementProgressContext(userInfo.value as Record<string, unknown> | null))
const achievementEntries = computed(() => buildAchievementEntries(achievementContext.value))
const earnedAchievementCount = computed(() => achievementEntries.value.filter((entry) => entry.progress.earned).length)
const raritySummary = computed(() => summarizeAchievementRarities(achievementEntries.value))
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

onMounted(() => {
  void loadAchievements()
})

async function loadAchievements() {
  loading.value = true
  try {
    userInfo.value = await getUserInfo()
  } catch (error) {
    console.error('Failed to load achievements', error)
  } finally {
    loading.value = false
  }
}

function openAchievementDetail(achievement: AchievementDefinition) {
  selectedAchievementId.value = achievement.id
}

function closeAchievementDetail() {
  selectedAchievementId.value = null
}
</script>

<template>
  <div class="page achievements-page">
    <header class="page-header">
      <button type="button" class="back-btn" @click="router.back()">
        <i class="ri-arrow-left-line" />
      </button>
      <div>
        <h1>{{ $t('profile.achievements.listTitle') }}</h1>
        <p>{{ $t('profile.achievements.subtitle') }}</p>
      </div>
    </header>

    <div v-if="loading" class="loading-text">{{ $t('common.status.loading') }}</div>

    <template v-else>
      <section class="achievement-hero">
        <div class="hero-copy">
          <span>{{ $t('profile.achievements.kicker') }}</span>
          <h2>{{ earnedAchievementCount }} / {{ achievementEntries.length }}</h2>
          <p>{{ $t('profile.achievements.earned') }}</p>
        </div>
        <AchievementMedal
          v-if="featuredAchievementEntry"
          :achievement="featuredAchievementEntry.definition"
          :earned="featuredAchievementEntry.progress.earned"
          size="lg"
        />
      </section>

      <section class="feature-stack">
        <button
          v-if="featuredAchievementEntry"
          type="button"
          class="feature-row"
          @click="openAchievementDetail(featuredAchievementEntry.definition)"
        >
          <AchievementMedal
            :achievement="featuredAchievementEntry.definition"
            :earned="featuredAchievementEntry.progress.earned"
            size="md"
          />
          <span>
            <small>{{ featuredAchievementEntry.progress.earned ? $t('profile.achievements.featured') : $t('profile.achievements.firstGoal') }}</small>
            <strong>{{ featuredAchievementEntry.definition.title }}</strong>
            <em>{{ featuredAchievementEntry.progress.label }}</em>
          </span>
          <i class="ri-arrow-right-s-line" />
        </button>

        <button
          v-if="nextAchievementEntry"
          type="button"
          class="feature-row next"
          @click="openAchievementDetail(nextAchievementEntry.definition)"
        >
          <AchievementMedal
            :achievement="nextAchievementEntry.definition"
            :earned="nextAchievementEntry.progress.earned"
            size="md"
          />
          <span>
            <small>{{ $t('profile.achievements.nextGoal') }}</small>
            <strong>{{ nextAchievementEntry.definition.title }}</strong>
            <em>{{ nextAchievementEntry.progress.label }}</em>
          </span>
          <i class="ri-arrow-right-s-line" />
        </button>
      </section>

      <section class="rarity-strip">
        <span
          v-for="summary in raritySummary"
          :key="summary.rarity"
          class="rarity-pill"
          :style="{
            '--rarity-edge': ACHIEVEMENT_RARITY_META[summary.rarity].edge,
          }"
        >
          {{ summary.label }} {{ summary.earned }}/{{ summary.total }}
        </span>
      </section>

      <section class="achievement-list">
        <button
          v-for="entry in achievementEntries"
          :key="entry.definition.id"
          type="button"
          class="achievement-item"
          :class="{ earned: entry.progress.earned }"
          @click="openAchievementDetail(entry.definition)"
        >
          <AchievementMedal
            :achievement="entry.definition"
            :earned="entry.progress.earned"
            size="md"
          />
          <span class="achievement-copy">
            <strong>{{ entry.definition.title }}</strong>
            <small>{{ entry.definition.condition }}</small>
            <span class="progress-line">
              <span class="progress-track">
                <span
                  class="progress-fill"
                  :style="{ width: `${entry.progress.percent}%` }"
                ></span>
              </span>
              <em>{{ entry.progress.label }}</em>
            </span>
          </span>
        </button>
      </section>
    </template>

    <div v-if="selectedAchievementEntry" class="dialog-mask" @click.self="closeAchievementDetail">
      <div class="dialog achievement-dialog">
        <AchievementMedal
          :achievement="selectedAchievementEntry.definition"
          :earned="selectedAchievementEntry.progress.earned"
          size="lg"
        />
        <div class="dialog-tags">
          <span
            :style="{ '--rarity-edge': ACHIEVEMENT_RARITY_META[selectedAchievementEntry.definition.rarity].edge }"
            class="rarity-tag"
          >
            {{ ACHIEVEMENT_RARITY_META[selectedAchievementEntry.definition.rarity].label }}
          </span>
          <span>{{ ACHIEVEMENT_CATEGORY_META[selectedAchievementEntry.definition.category].label }}</span>
        </div>
        <h3>{{ selectedAchievementEntry.definition.title }}</h3>
        <p>{{ selectedAchievementEntry.definition.condition }}</p>
        <div class="dialog-progress">
          <span>{{ selectedAchievementEntry.progress.earned ? $t('profile.achievements.earned') : $t('profile.achievements.progress') }}</span>
          <strong>{{ selectedAchievementEntry.progress.label }}</strong>
        </div>
        <div class="progress-track dialog-track">
          <span
            class="progress-fill"
            :style="{ width: `${selectedAchievementEntry.progress.percent}%` }"
          ></span>
        </div>
        <button type="button" class="close-btn" @click="closeAchievementDetail">
          {{ $t('profile.achievements.close') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding: calc(var(--safe-top, 0px) + 2px) var(--page-gutter) calc(26px + var(--safe-bottom, 0px));
}

.achievements-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 6px 0 0;
}

.back-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 999px;
  background: rgba(75, 54, 33, 0.08);
  color: var(--color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.page-header h1 {
  color: var(--color-text-main);
  font-size: 24px;
  line-height: 1.1;
}

.page-header p {
  margin-top: 5px;
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.loading-text {
  padding: 60px 0;
  color: var(--color-text-secondary);
  text-align: center;
  font-size: 13px;
}

.achievement-hero {
  overflow: hidden;
  border-radius: var(--radius-md);
  border: 1px solid rgba(184, 115, 51, 0.14);
  padding: 16px;
  background:
    radial-gradient(circle at 8% 8%, rgba(255, 178, 62, 0.2), transparent 34%),
    radial-gradient(circle at 92% 10%, rgba(255, 190, 219, 0.26), transparent 34%),
    var(--color-card-bg);
  box-shadow: var(--shadow-sm);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.hero-copy span {
  color: var(--color-accent);
  font-size: 10px;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.hero-copy h2 {
  margin-top: 5px;
  color: var(--color-text-main);
  font-size: 34px;
  line-height: 1;
}

.hero-copy p {
  margin-top: 4px;
  color: var(--color-text-secondary);
  font-size: 12px;
}

.feature-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.feature-row,
.achievement-item {
  width: 100%;
  border: 1px solid rgba(184, 115, 51, 0.1);
  border-radius: 16px;
  background: var(--color-card-bg);
  color: var(--color-text-main);
  box-shadow: var(--shadow-sm);
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  text-align: left;
}

.feature-row {
  padding: 10px;
}

.feature-row.next {
  background:
    radial-gradient(circle at 0% 50%, rgba(90, 183, 255, 0.12), transparent 34%),
    var(--color-card-bg);
}

.feature-row span,
.achievement-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.feature-row small {
  color: var(--color-accent);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.feature-row strong,
.achievement-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-main);
  font-size: 14px;
}

.feature-row em,
.achievement-copy small,
.progress-line em {
  color: var(--color-text-secondary);
  font-size: 11px;
  font-style: normal;
}

.feature-row > i {
  color: var(--color-text-muted);
  font-size: 18px;
}

.rarity-strip {
  display: flex;
  gap: 7px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.rarity-pill {
  --rarity-edge: #B87333;
  flex-shrink: 0;
  border: 1px solid color-mix(in srgb, var(--rarity-edge) 36%, transparent);
  border-radius: 999px;
  padding: 5px 9px;
  background: color-mix(in srgb, var(--rarity-edge) 10%, var(--color-card-bg));
  color: var(--rarity-edge);
  font-size: 11px;
  font-weight: 800;
}

.achievement-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.achievement-item {
  padding: 10px;
}

.achievement-item.earned {
  background:
    radial-gradient(circle at 0% 50%, rgba(255, 178, 62, 0.12), transparent 36%),
    var(--color-card-bg);
}

.progress-line {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
}

.progress-track {
  display: block;
  height: 6px;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(75, 54, 33, 0.1);
}

.progress-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #D4A373, #FFB23E);
}

.dialog-mask {
  position: fixed;
  inset: 0;
  padding: 16px;
  background: rgba(0, 0, 0, 0.48);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.achievement-dialog {
  width: 100%;
  max-width: 360px;
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  padding: 18px;
  box-shadow: 0 14px 36px rgba(44, 24, 16, 0.2);
  text-align: center;
}

.dialog-tags {
  display: flex;
  justify-content: center;
  gap: 6px;
  margin: 12px 0 8px;
}

.dialog-tags span {
  border-radius: 999px;
  padding: 4px 8px;
  background: rgba(75, 54, 33, 0.07);
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.dialog-tags .rarity-tag {
  --rarity-edge: #B87333;
  background: color-mix(in srgb, var(--rarity-edge) 12%, var(--color-card-bg));
  color: var(--rarity-edge);
}

.achievement-dialog h3 {
  color: var(--color-text-main);
  font-size: 18px;
}

.achievement-dialog p {
  margin-top: 7px;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.dialog-progress {
  margin-top: 14px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--color-text-secondary);
  font-size: 12px;
}

.dialog-progress strong {
  color: var(--color-accent);
}

.dialog-track {
  margin-top: 8px;
}

.close-btn {
  width: 100%;
  margin-top: 16px;
  border: none;
  border-radius: 12px;
  padding: 11px;
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  font-size: 13px;
  font-weight: 700;
}

@media (max-width: 380px) {
  .page-header h1 {
    font-size: 22px;
  }

  .achievement-hero {
    padding: 14px;
  }

  .hero-copy h2 {
    font-size: 30px;
  }
}
</style>
