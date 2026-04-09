<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import { useRouter } from 'vue-router'
import { getUserInfo, signInDaily, type UserInfo } from '@/api/user'
import { resolveApiUrl } from '@/api/image'
import UserLevelBadge from '@/components/UserLevelBadge.vue'
import { buildForumLevelGuide, computeLevelProgressPercent } from '@/utils/forumLevel'

const { t } = useI18n()
const userStore = useUserStore()
const router = useRouter()
const toast = useToastStore()
const userInfo = ref<UserInfo | null>(null)
const showLevelGuide = ref(false)
const signingIn = ref(false)

const displayUser = computed(() => userInfo.value || userStore.user)
const displayEmail = computed(() => userInfo.value?.email || '')

const roleLabel = computed(() => {
  const r = displayUser.value?.role
  if (r === 'admin') return t('profile.role.admin')
  if (r === 'moderator') return t('profile.role.moderator')
  return t('profile.role.user')
})

const forumLevelDefinitions = computed(() => [
  { level: 1, name: t('profile.activity.levels.lv1'), color: '#403B33', bold: false },
  { level: 2, name: t('profile.activity.levels.lv2'), color: '#808080', bold: false },
  { level: 3, name: t('profile.activity.levels.lv3'), color: '#FFFFFF', bold: false },
  { level: 4, name: t('profile.activity.levels.lv4'), color: '#00C100', bold: false },
  { level: 5, name: t('profile.activity.levels.lv5'), color: '#0080FF', bold: false },
  { level: 6, name: t('profile.activity.levels.lv6'), color: '#800080', bold: false },
  { level: 7, name: t('profile.activity.levels.lv7'), color: '#F59B00', bold: true },
  { level: 8, name: t('profile.activity.levels.lv8'), color: '#0080C0', bold: true },
  { level: 9, name: t('profile.activity.levels.lv9'), color: '#EBD7A7', bold: true },
  { level: 10, name: t('profile.activity.levels.lv10'), color: '#8E1027', bold: true },
])

const activityExpRules = computed(() => [
  t('profile.activity.rules.dailySignIn'),
  t('profile.activity.rules.firstLike'),
  t('profile.activity.rules.comment'),
  t('profile.activity.rules.liked'),
  t('profile.activity.rules.approved'),
  t('profile.activity.rules.download'),
  t('profile.activity.rules.story'),
])

const forumLevelGuide = computed(() => buildForumLevelGuide(forumLevelDefinitions.value))

const currentForumLevel = computed(() => {
  const level = Number(displayUser.value?.forum_level || 1)
  return Number.isFinite(level) && level > 0 ? Math.floor(level) : 1
})

const currentForumLevelName = computed(() => {
  const matched = forumLevelDefinitions.value.find((definition) => definition.level === currentForumLevel.value)
  return matched?.name || displayUser.value?.forum_level_name || t('profile.activity.levels.lv1')
})

const activityProgressPercent = computed(() => {
  return computeLevelProgressPercent(displayUser.value?.current_level_exp, displayUser.value?.next_level_exp)
})

const signInHint = computed(() => (
  displayUser.value?.signed_in_today
    ? t('profile.activity.signInDone')
    : t('profile.activity.signInReady')
))

async function loadProfile() {
  try {
    const res = await getUserInfo()
    userInfo.value = res
    if (userStore.user?.id === res.id) {
      userStore.mergeUser({
        username: res.username,
        avatar: res.avatar,
        role: res.role,
        is_sponsor: res.is_sponsor,
        sponsor_level: res.sponsor_level,
        sponsor_color: res.sponsor_color,
        sponsor_bold: res.sponsor_bold,
        name_color: res.name_color,
        name_bold: res.name_bold,
        activity_points: res.activity_points,
        activity_experience: res.activity_experience,
        forum_level: res.forum_level,
        forum_level_name: res.forum_level_name,
        forum_level_color: res.forum_level_color,
        forum_level_bold: res.forum_level_bold,
        current_level_exp: res.current_level_exp,
        next_level_exp: res.next_level_exp,
        signed_in_today: res.signed_in_today,
      })
    }
  } catch (e) {
    console.error('Failed to load profile', e)
  }
}

async function handleDailySignIn() {
  if (signingIn.value || displayUser.value?.signed_in_today) return
  signingIn.value = true
  try {
    const result = await signInDaily()
    userInfo.value = userInfo.value ? { ...userInfo.value, ...result } : userInfo.value
    userStore.mergeUser(result)
    if (result.granted) {
      toast.success(t('profile.activity.signInSuccess', {
        points: result.points_delta,
        exp: result.experience_delta,
      }))
    } else {
      toast.info(result.message)
    }
  } catch (error) {
    console.error('Failed to sign in daily', error)
    toast.error((error as Error)?.message || t('profile.activity.signInFailed'))
  } finally {
    signingIn.value = false
  }
}

function handleLogout() {
  userStore.logout()
  router.replace({ name: 'login' })
}

function formatLevelRange(level: { currentBase: number; nextBase: number | null }) {
  if (level.nextBase == null) {
    return t('profile.activity.rangeMax', { start: level.currentBase })
  }
  return t('profile.activity.rangeToNext', { start: level.currentBase, end: level.nextBase - 1 })
}

onMounted(loadProfile)
</script>

<template>
  <div class="page profile-page">
    <header class="page-header">
      <h1>{{ $t('profile.title') }}</h1>
    </header>

    <div class="page-body">
      <div class="profile-card" v-if="displayUser">
        <div class="avatar-wrap">
          <img
            v-if="displayUser.avatar"
            :src="resolveApiUrl(displayUser.avatar)"
            class="avatar-img" alt=""
          />
          <div v-else class="avatar-placeholder">
            <i class="ri-user-3-fill" />
          </div>
        </div>
        <div class="user-main">
          <div class="name-row">
            <span
              class="username"
              :style="{
                color: displayUser.name_color || undefined,
                fontWeight: displayUser.name_bold ? 'bold' : undefined,
              }"
            >{{ displayUser.username }}</span>
            <UserLevelBadge
              v-if="displayUser.forum_level"
              class="inline-level-badge"
              compact
              :level="displayUser.forum_level"
              :name="currentForumLevelName"
              :color="displayUser.forum_level_color"
              :bold="displayUser.forum_level_bold"
            />
          </div>
          <div class="badges">
            <span class="role-badge">{{ roleLabel }}</span>
            <span v-if="displayUser.is_sponsor" class="sponsor-badge">
              <i class="ri-vip-crown-line" /> {{ $t('profile.sponsor') }}
            </span>
            <span v-if="typeof displayUser.activity_points === 'number'" class="points-badge">
              {{ $t('profile.activity.pointsText', { n: displayUser.activity_points }) }}
            </span>
          </div>
        </div>
      </div>

      <div v-if="displayEmail" class="info-section">
        <div class="info-row">
          <i class="ri-mail-line" />
          <span>{{ displayEmail }}</span>
        </div>
      </div>

      <section v-if="displayUser?.forum_level" class="activity-card">
        <div class="activity-card-head">
          <div class="activity-card-identity">
            <p class="activity-card-label">{{ $t('profile.activity.title') }}</p>
            <UserLevelBadge
              compact
              :level="displayUser.forum_level"
              :name="currentForumLevelName"
              :color="displayUser.forum_level_color"
              :bold="displayUser.forum_level_bold"
            />
          </div>
          <button
            type="button"
            class="guide-btn"
            :aria-label="$t('profile.activity.help')"
            @click="showLevelGuide = true"
          >
            <i class="ri-question-line" />
          </button>
        </div>
        <div class="sign-in-row">
          <div class="sign-in-copy">
            <span class="sign-in-kicker">{{ $t('profile.activity.signInKicker') }}</span>
            <p>{{ signInHint }}</p>
          </div>
          <button
            type="button"
            class="sign-in-btn"
            :class="{ done: displayUser.signed_in_today }"
            :disabled="signingIn || displayUser.signed_in_today"
            @click="handleDailySignIn"
          >
            <i :class="displayUser.signed_in_today ? 'ri-checkbox-circle-fill' : 'ri-rocket-line'" />
            <span v-if="displayUser.signed_in_today">{{ $t('profile.activity.signInCompleted') }}</span>
            <span v-else-if="signingIn">{{ $t('profile.activity.signInLoading') }}</span>
            <span v-else>{{ $t('profile.activity.signInAction') }}</span>
          </button>
        </div>
        <div class="activity-summary-row">
          <span class="activity-summary-item">
            <span class="activity-summary-label">{{ $t('profile.activity.pointsLabel') }}</span>
            <strong>{{ displayUser.activity_points || 0 }}</strong>
          </span>
          <span class="activity-summary-divider" aria-hidden="true"></span>
          <span class="activity-summary-item">
            <span class="activity-summary-label">{{ $t('profile.activity.expLabel') }}</span>
            <strong>{{ displayUser.activity_experience || 0 }}</strong>
          </span>
          <span class="activity-summary-divider" aria-hidden="true"></span>
          <span class="activity-summary-item progress">
            <span class="activity-summary-label">{{ $t('profile.activity.progress') }}</span>
            <strong>{{ displayUser.current_level_exp || 0 }} / {{ displayUser.next_level_exp || displayUser.current_level_exp || 0 }}</strong>
          </span>
        </div>
        <div class="activity-progress-track">
          <span
            class="activity-progress-fill"
            :style="{
              width: `${activityProgressPercent}%`,
              background: displayUser.forum_level_color || '#4B3621',
            }"
          />
        </div>
      </section>

      <div class="menu-section">
        <button class="menu-item" @click="router.push('/profiles')">
          <i class="ri-id-card-line" />
          <span>{{ $t('profile.menu.cloudProfiles') }}</span>
          <i class="ri-arrow-right-s-line arrow" />
        </button>
        <button class="menu-item" @click="router.push('/my-favorites')">
          <i class="ri-heart-line" />
          <span>{{ $t('profile.menu.favorites') }}</span>
          <i class="ri-arrow-right-s-line arrow" />
        </button>
        <button class="menu-item" @click="router.push('/my-posts')">
          <i class="ri-file-list-line" />
          <span>{{ $t('profile.menu.posts') }}</span>
          <i class="ri-arrow-right-s-line arrow" />
        </button>
        <button class="menu-item" @click="router.push('/my-items')">
          <i class="ri-box-3-line" />
          <span>{{ $t('profile.menu.items') }}</span>
          <i class="ri-arrow-right-s-line arrow" />
        </button>
        <button class="menu-item" @click="router.push('/my-collections')">
          <i class="ri-folder-line" />
          <span>{{ $t('profile.menu.collections') }}</span>
          <i class="ri-arrow-right-s-line arrow" />
        </button>
      </div>

      <div class="menu-section">
        <button class="menu-item" @click="router.push('/about')">
          <i class="ri-information-line" />
          <span>{{ $t('profile.menu.about') }}</span>
          <i class="ri-arrow-right-s-line arrow" />
        </button>
      </div>

      <button class="logout-btn" @click="handleLogout">
        <i class="ri-logout-box-r-line" /> {{ $t('profile.logout') }}
      </button>

      <div v-if="showLevelGuide" class="dialog-mask" @click.self="showLevelGuide = false">
        <div class="dialog level-guide-dialog">
          <h3>{{ $t('profile.activity.guideTitle') }}</h3>
          <p class="guide-current">{{ $t('profile.activity.guideCurrent', { level: currentForumLevel, name: currentForumLevelName }) }}</p>

          <section class="guide-section">
            <h4>{{ $t('profile.activity.rulesTitle') }}</h4>
            <div class="guide-rule-list">
              <div v-for="rule in activityExpRules" :key="rule" class="guide-rule-item">
                {{ rule }}
              </div>
            </div>
          </section>

          <section class="guide-section">
            <h4>{{ $t('profile.activity.levelsTitle') }}</h4>
            <div class="level-guide-list">
              <div
                v-for="level in forumLevelGuide"
                :key="level.level"
                class="level-guide-item"
                :class="{ current: currentForumLevel === level.level }"
              >
                <div class="level-guide-top">
                  <UserLevelBadge
                    compact
                    :level="level.level"
                    :name="level.name"
                    :color="level.color"
                    :bold="level.bold"
                  />
                  <span v-if="currentForumLevel === level.level" class="current-chip">{{ $t('profile.activity.current') }}</span>
                </div>
                <p class="level-guide-range">{{ formatLevelRange(level) }}</p>
              </div>
            </div>
          </section>

          <div class="dialog-actions">
            <button type="button" class="action-btn primary" @click="showLevelGuide = false">
              {{ $t('profile.activity.close') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding: calc(var(--safe-top, 0px) + 2px) var(--page-gutter) calc(26px + var(--safe-bottom, 0px));
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.page-header { padding: 6px 0 0; }
.page-header h1 { font-size: 24px; line-height: 1.1; letter-spacing: 0.01em; }

.profile-card {
  display: flex; align-items: center; gap: 16px;
  padding: 18px 16px;
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  border: 1px solid rgba(75, 54, 33, 0.08);
  box-shadow: var(--shadow-sm);
}

.avatar-wrap { flex-shrink: 0; }
.avatar-img { width: 58px; height: 58px; border-radius: 50%; object-fit: cover; }
.avatar-placeholder {
  width: 58px; height: 58px; border-radius: 50%;
  background: var(--icon-bg); display: flex;
  align-items: center; justify-content: center;
}
.avatar-placeholder i { font-size: 29px; color: var(--icon-color); }

.user-main { display: flex; flex-direction: column; gap: 6px; min-width: 0; flex: 1; }
.name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.username { font-size: 17px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.inline-level-badge { flex-shrink: 0; }
.badges { display: flex; gap: 6px; flex-wrap: wrap; }
.role-badge {
  font-size: 11px; padding: 2px 7px; border-radius: 8px;
  background: var(--color-primary-light); color: var(--color-text-secondary);
}
.sponsor-badge {
  font-size: 11px; padding: 2px 7px; border-radius: 8px;
  background: var(--tag-bg); color: var(--color-accent);
}
.sponsor-badge i { margin-right: 2px; }
.points-badge {
  font-size: 11px;
  padding: 2px 7px;
  border-radius: 8px;
  background: rgba(75, 54, 33, 0.08);
  color: var(--color-primary);
}

.info-section {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  border: 1px solid rgba(75, 54, 33, 0.08);
  padding: 13px 14px;
  box-shadow: var(--shadow-sm);
}
.info-row { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--color-text-secondary); min-width: 0; }
.info-row span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.info-row i { font-size: 18px; color: var(--icon-color); }

.activity-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  border: 1px solid rgba(75, 54, 33, 0.08);
  padding: 10px 12px;
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.activity-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.activity-card-identity {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.activity-card-label {
  font-size: 10px;
  color: var(--color-text-secondary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.guide-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 999px;
  background: rgba(75, 54, 33, 0.08);
  color: var(--color-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.guide-btn i {
  font-size: 13px;
}

.sign-in-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 12px;
  background: rgba(75, 54, 33, 0.04);
}

.sign-in-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sign-in-kicker {
  font-size: 10px;
  color: var(--color-text-secondary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sign-in-copy p {
  font-size: 12px;
  line-height: 1.4;
  color: var(--color-text-main);
}

.sign-in-btn {
  flex-shrink: 0;
  min-height: 34px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
}

.sign-in-btn.done {
  background: rgba(75, 54, 33, 0.12);
  color: var(--color-primary);
}

.sign-in-btn:disabled {
  opacity: 1;
}

.activity-summary-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.activity-summary-item {
  display: inline-flex;
  align-items: baseline;
  gap: 5px;
  min-width: 0;
}

.activity-summary-item.progress {
  margin-left: auto;
}

.activity-summary-label {
  font-size: 11px;
  color: var(--color-text-secondary);
}

.activity-summary-item strong {
  color: var(--color-text-main);
  font-size: 12px;
  font-weight: 600;
}

.activity-summary-divider {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: rgba(75, 54, 33, 0.22);
  flex-shrink: 0;
}

.activity-progress-track {
  width: 100%;
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: #E4D9CC;
}

.activity-progress-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  box-shadow: inset 0 0 0 1px rgba(75, 54, 33, 0.08);
}

.menu-section {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  border: 1px solid rgba(75, 54, 33, 0.08);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}
.menu-item {
  display: flex; align-items: center; gap: 10px; padding: 13px 14px;
  font-size: 13px; color: var(--text-dark); cursor: pointer;
  background: none; border: none; width: 100%; text-align: left;
  border-bottom: 1px solid var(--color-border-light);
}
.menu-item:last-child { border-bottom: none; }
.menu-item i { font-size: 17px; color: var(--icon-color); }
.menu-item span { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.menu-item .arrow { color: var(--color-text-muted); font-size: 16px; }

.logout-btn {
  width: 100%; padding: 13px; border: none; border-radius: var(--radius-sm);
  background: var(--btn-danger-bg); color: var(--btn-primary-text); font-size: 14px; font-weight: 500;
  cursor: pointer; display: flex; align-items: center; justify-content: center; gap: 6px;
  margin-top: 4px;
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

.dialog.level-guide-dialog {
  width: 100%;
  max-width: 420px;
  max-height: min(78vh, 680px);
  overflow-y: auto;
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  padding: 16px;
  box-shadow: 0 14px 36px rgba(44, 24, 16, 0.2);
}

.dialog.level-guide-dialog h3 {
  font-size: 17px;
  color: var(--color-text-main);
}

.guide-current {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-secondary);
}

.guide-section {
  margin-top: 16px;
}

.guide-section h4 {
  font-size: 13px;
  color: var(--color-text-main);
}

.guide-rule-list,
.level-guide-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 10px;
}

.guide-rule-item,
.level-guide-item {
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(75, 54, 33, 0.05);
}

.guide-rule-item {
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-main);
}

.level-guide-item.current {
  background: rgba(75, 54, 33, 0.1);
}

.level-guide-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.current-chip {
  flex-shrink: 0;
  padding: 3px 8px;
  border-radius: 999px;
  background: rgba(75, 54, 33, 0.12);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 600;
}

.level-guide-range {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.dialog-actions {
  margin-top: 18px;
  display: flex;
  justify-content: flex-end;
}

.action-btn {
  min-width: 96px;
  padding: 10px 14px;
  border: none;
  border-radius: 10px;
  background: rgba(75, 54, 33, 0.08);
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 600;
}

.action-btn.primary {
  background: var(--color-secondary);
  color: var(--btn-primary-text);
}

@media (max-width: 380px) {
  .page-header h1 { font-size: 22px; }
  .profile-card {
    padding: 14px 12px;
    gap: 12px;
  }

  .avatar-img,
  .avatar-placeholder {
    width: 50px;
    height: 50px;
  }

  .avatar-placeholder i {
    font-size: 24px;
  }

  .username {
    font-size: 15px;
  }

  .name-row {
    gap: 6px;
  }

  .activity-card {
    padding: 9px 10px;
  }

  .activity-card-identity {
    gap: 6px;
  }

  .activity-summary-row {
    gap: 5px;
  }

  .sign-in-row {
    align-items: flex-start;
  }

  .sign-in-btn {
    min-width: 92px;
    padding: 0 10px;
  }

  .activity-summary-item.progress {
    width: 100%;
    margin-left: 0;
    justify-content: space-between;
  }

  .activity-summary-label,
  .activity-summary-item strong {
    font-size: 11px;
  }

  .menu-item {
    padding: 11px 12px;
    font-size: 12px;
  }
}
</style>
