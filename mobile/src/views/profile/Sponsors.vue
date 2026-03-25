<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { resolveApiUrl } from '@/api/image'
import { listSponsors, type SponsorUser } from '@/api/user'

const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const sponsors = ref<SponsorUser[]>([])

async function loadSponsors() {
  loading.value = true
  try {
    const res = await listSponsors()
    sponsors.value = Array.isArray(res.users) ? res.users : []
  } finally {
    loading.value = false
  }
}

function getInitial(name: string) {
  const trimmed = String(name || '').trim()
  return trimmed ? trimmed.slice(0, 1).toUpperCase() : '?'
}

onMounted(loadSponsors)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.about.sponsors.title') }}</h1>
    </header>
    <div class="sub-body">
      <p class="list-desc">{{ $t('profile.about.sponsors.desc') }}</p>

      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="sponsors.length === 0" class="hint">{{ $t('profile.about.sponsors.empty') }}</div>

      <div v-else class="sponsor-list">
        <article v-for="user in sponsors" :key="user.id" class="sponsor-item">
          <div class="avatar-wrap">
            <img v-if="user.avatar" :src="resolveApiUrl(user.avatar)" alt="" class="avatar" />
            <div v-else class="avatar-fallback">{{ getInitial(user.username) }}</div>
          </div>
          <div class="meta">
            <div class="name-row">
              <strong :style="{ color: user.name_color || undefined, fontWeight: user.name_bold ? '700' : '600' }">
                {{ user.username }}
              </strong>
              <span class="level-badge">Lv{{ user.sponsor_level || 1 }}</span>
            </div>
          </div>
        </article>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sub-body {
  padding-bottom: calc(24px + var(--safe-bottom, 0px));
}

.list-desc {
  color: var(--color-text-secondary);
  font-size: 13px;
  margin-bottom: 10px;
}

.sponsor-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sponsor-item {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.avatar-wrap {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
}

.avatar {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  font-size: 13px;
  font-weight: 700;
}

.meta {
  min-width: 0;
  flex: 1;
}

.name-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.name-row strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.level-badge {
  flex-shrink: 0;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  border: 1px solid var(--color-border-light);
}
</style>
