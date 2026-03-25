<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@shared/stores/user'
import { useRouter } from 'vue-router'
import { getUserInfo, type UserInfo } from '@/api/user'
import { resolveApiUrl } from '@/api/image'

const { t } = useI18n()
const userStore = useUserStore()
const router = useRouter()
const userInfo = ref<UserInfo | null>(null)

const displayUser = computed(() => userInfo.value || userStore.user)
const displayEmail = computed(() => userInfo.value?.email || '')

const roleLabel = computed(() => {
  const r = displayUser.value?.role
  if (r === 'admin') return t('profile.role.admin')
  if (r === 'moderator') return t('profile.role.moderator')
  return t('profile.role.user')
})

async function loadProfile() {
  try {
    const res = await getUserInfo()
    userInfo.value = res
  } catch (e) {
    console.error('Failed to load profile', e)
  }
}

function handleLogout() {
  userStore.logout()
  router.replace({ name: 'login' })
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
          <span
            class="username"
            :style="{
              color: displayUser.name_color || undefined,
              fontWeight: displayUser.name_bold ? 'bold' : undefined,
            }"
          >{{ displayUser.username }}</span>
          <div class="badges">
            <span class="role-badge">{{ roleLabel }}</span>
            <span v-if="displayUser.is_sponsor" class="sponsor-badge">
              <i class="ri-vip-crown-line" /> {{ $t('profile.sponsor') }}
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
.username { font-size: 17px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
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

  .menu-item {
    padding: 11px 12px;
    font-size: 12px;
  }
}
</style>
