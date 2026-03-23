<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getCloudProfile, type CloudProfile } from '@/api/profile'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const profile = ref<CloudProfile | null>(null)
const showRawLua = ref(false)

const profileId = computed(() => String(route.params.id || ''))

async function loadDetail() {
  if (!profileId.value) return
  loading.value = true
  try {
    profile.value = await getCloudProfile(profileId.value)
  } catch (error) {
    console.error('Failed to load profile detail', error)
  } finally {
    loading.value = false
  }
}

function formatDate(value: string) {
  const date = new Date(value)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

onMounted(loadDetail)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profiles.detailTitle') }}</h1>
    </header>

    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="!profile" class="hint">{{ $t('profiles.empty') }}</div>

      <template v-else>
        <section class="detail-panel">
          <h2>{{ profile.profile_name || profile.id }}</h2>
          <p>{{ $t('profiles.field.id') }}: {{ profile.id }}</p>
          <p>{{ $t('profiles.field.account') }}: {{ profile.account_id }}</p>
          <p>{{ $t('profiles.field.version') }}: {{ profile.version }}</p>
          <p>{{ $t('profiles.field.checksum') }}: {{ profile.checksum }}</p>
          <p>{{ $t('profiles.field.updatedAt') }}: {{ formatDate(profile.updated_at) }}</p>
        </section>

        <section class="detail-panel">
          <button class="toggle-btn" @click="showRawLua = !showRawLua">
            <span>{{ $t('profiles.field.rawLua') }}</span>
            <i :class="showRawLua ? 'ri-arrow-up-s-line' : 'ri-arrow-down-s-line'" />
          </button>
          <pre v-if="showRawLua" class="lua-view">{{ profile.raw_lua || '-' }}</pre>
        </section>
      </template>
    </div>
  </div>
</template>

<style scoped>
.detail-panel {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 14px;
  margin-bottom: 12px;
}

.detail-panel h2 {
  font-size: 16px;
  margin-bottom: 8px;
}

.detail-panel p {
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-secondary);
  word-break: break-all;
}

.toggle-btn {
  width: 100%;
  border: none;
  background: transparent;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  color: var(--text-dark);
  cursor: pointer;
}

.lua-view {
  margin-top: 10px;
  border-radius: var(--radius-sm);
  padding: 10px;
  background: var(--input-bg);
  max-height: 320px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
