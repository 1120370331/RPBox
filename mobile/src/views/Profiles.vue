<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listCloudProfiles, type CloudProfile } from '@/api/profile'

const router = useRouter()

const loading = ref(false)
const keyword = ref('')
const profiles = ref<CloudProfile[]>([])

const filteredProfiles = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  if (!query) return profiles.value
  return profiles.value.filter((profile) => {
    return (
      profile.profile_name.toLowerCase().includes(query) ||
      profile.id.toLowerCase().includes(query) ||
      profile.account_id.toLowerCase().includes(query)
    )
  })
})

async function loadProfiles() {
  loading.value = true
  try {
    const res = await listCloudProfiles({ page: 1, page_size: 200 })
    profiles.value = res.profiles || []
  } catch (error) {
    console.error('Failed to load cloud profiles', error)
  } finally {
    loading.value = false
  }
}

function formatDate(value: string) {
  const date = new Date(value)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

onMounted(loadProfiles)
</script>

<template>
  <div class="page profiles-page">
    <header class="page-header">
      <h1>{{ $t('profiles.title') }}</h1>
    </header>

    <div class="search-bar">
      <i class="ri-search-line" />
      <input v-model="keyword" :placeholder="$t('profiles.searchPlaceholder')" />
    </div>

    <div class="page-body">
      <div v-if="loading && profiles.length === 0" class="empty-hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="filteredProfiles.length === 0" class="empty-hint">{{ $t('profiles.empty') }}</div>

      <div v-else class="profile-list">
        <button
          v-for="profile in filteredProfiles"
          :key="profile.id"
          class="profile-card"
          @click="router.push({ name: 'profile-detail', params: { id: profile.id } })"
        >
          <div class="profile-main">
            <h3>{{ profile.profile_name || profile.id }}</h3>
            <p class="profile-id">{{ profile.id }}</p>
          </div>
          <div class="profile-meta">
            <span>{{ $t('profiles.version', { n: profile.version }) }}</span>
            <span>{{ formatDate(profile.updated_at) }}</span>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 0 16px 16px; }
.page-header { padding: 12px 0 8px; }
.page-header h1 { font-size: 22px; }

.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--input-bg);
  border-radius: 20px;
  padding: 8px 14px;
  margin-bottom: 10px;
}
.search-bar i { color: var(--input-placeholder); font-size: 16px; }
.search-bar input {
  flex: 1;
  border: none;
  background: transparent;
  outline: none;
  font-size: 14px;
  color: var(--text-dark);
}

.empty-hint {
  text-align: center;
  padding: 60px 0;
  color: var(--color-accent);
  font-size: 14px;
}

.profile-list { display: flex; flex-direction: column; gap: 10px; }

.profile-card {
  width: 100%;
  border: none;
  border-radius: var(--radius-md);
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
  padding: 14px;
  text-align: left;
  cursor: pointer;
}

.profile-main h3 {
  font-size: 15px;
  margin-bottom: 4px;
}

.profile-id {
  font-size: 12px;
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-meta {
  margin-top: 10px;
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--color-text-secondary);
}
</style>
