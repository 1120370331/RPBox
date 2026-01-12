<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as profileApi from '../../api/profile'
import type { ProfileVersion } from '../../api/profile'

const props = defineProps<{
  profileId: string
  profileName: string
}>()

const emit = defineEmits<{
  close: []
  rollback: [version: number]
}>()

const versions = ref<ProfileVersion[]>([])
const isLoading = ref(false)
const selectedVersion = ref<number | null>(null)

onMounted(async () => {
  await loadVersions()
})

async function loadVersions() {
  isLoading.value = true
  try {
    versions.value = await profileApi.getVersions(props.profileId)
  } finally {
    isLoading.value = false
  }
}

async function confirmRollback() {
  if (selectedVersion.value === null) return
  emit('rollback', selectedVersion.value)
}
</script>

<template>
  <div class="version-history">
    <div class="header">
      <h3>{{ profileName }} - 版本历史</h3>
      <button class="close-btn" @click="emit('close')">×</button>
    </div>

    <div v-if="isLoading" class="loading">加载中...</div>

    <div v-else class="version-list">
      <div
        v-for="v in versions"
        :key="v.id"
        class="version-item"
        :class="{ selected: selectedVersion === v.version }"
        @click="selectedVersion = v.version"
      >
        <span class="version-num">v{{ v.version }}</span>
        <span class="version-date">{{ v.created_at }}</span>
        <span class="version-log">{{ v.change_log || '无备注' }}</span>
      </div>
    </div>

    <div class="actions">
      <button
        class="btn"
        :disabled="selectedVersion === null"
        @click="confirmRollback"
      >
        恢复到此版本
      </button>
    </div>
  </div>
</template>

<style scoped>
.version-history {
  background: var(--color-bg);
  border-radius: 8px;
  padding: 1.5rem;
  min-width: 400px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
}

.version-list {
  max-height: 300px;
  overflow-y: auto;
}

.version-item {
  display: flex;
  gap: 1rem;
  padding: 0.75rem;
  border-radius: 4px;
  cursor: pointer;
  margin-bottom: 0.5rem;
  background: var(--color-bg-secondary);
}

.version-item.selected {
  background: var(--color-primary);
  color: white;
}

.version-num {
  font-weight: bold;
}

.actions {
  margin-top: 1rem;
  text-align: right;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  background: var(--color-primary);
  color: white;
  cursor: pointer;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
