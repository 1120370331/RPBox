<script setup lang="ts">
import type { ConflictInfo, ConflictResolution } from '../../utils/conflict'

defineProps<{
  conflict: ConflictInfo
}>()

const emit = defineEmits<{
  resolve: [resolution: ConflictResolution]
  cancel: []
}>()


function useLocal() {
  emit('resolve', 'local')
}

function useCloud() {
  emit('resolve', 'cloud')
}
</script>

<template>
  <div class="conflict-dialog">
    <div class="dialog-header">
      <h3>检测到数据冲突</h3>
    </div>

    <div class="dialog-body">
      <p class="profile-name">{{ conflict.profileName }}</p>

      <div class="versions">
        <div class="version local">
          <h4>本地版本</h4>
          <p>{{ conflict.localModifiedAt }}</p>
        </div>
        <div class="version cloud">
          <h4>云端版本</h4>
          <p>{{ conflict.cloudModifiedAt }}</p>
        </div>
      </div>
    </div>

    <div class="dialog-actions">
      <button class="btn" @click="useLocal">使用本地</button>
      <button class="btn" @click="useCloud">使用云端</button>
      <button class="btn secondary" @click="emit('cancel')">取消</button>
    </div>
  </div>
</template>

<style scoped>
.conflict-dialog {
  background: var(--color-bg);
  border-radius: 8px;
  padding: 1.5rem;
  max-width: 500px;
}

.dialog-header h3 {
  margin: 0 0 1rem;
  color: #c00;
}

.profile-name {
  font-weight: bold;
  margin-bottom: 1rem;
}

.versions {
  display: flex;
  gap: 1rem;
}

.version {
  flex: 1;
  padding: 1rem;
  border-radius: 4px;
  background: var(--color-bg-secondary);
}

.dialog-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1.5rem;
  justify-content: flex-end;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
}

.btn.secondary {
  background: var(--color-bg-secondary);
  color: var(--color-text);
}
</style>
