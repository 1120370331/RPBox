<script setup lang="ts">
defineProps<{
  id: string
  name: string
  icon?: string
  status: 'synced' | 'pending' | 'conflict'
  selected?: boolean
}>()

const emit = defineEmits<{
  click: []
  contextmenu: [e: MouseEvent]
}>()
</script>

<template>
  <div
    class="profile-card"
    :class="{ selected }"
    @click="emit('click')"
    @contextmenu.prevent="emit('contextmenu', $event)"
  >
    <div class="icon">{{ icon || '👤' }}</div>
    <div class="name">{{ name }}</div>
    <div class="status" :class="status">
      <span v-if="status === 'synced'">✓ 已同步</span>
      <span v-else-if="status === 'pending'">⟳ 待同步</span>
      <span v-else>⚠ 冲突</span>
    </div>
  </div>
</template>

<style scoped>
.profile-card {
  background: var(--color-bg-secondary);
  border-radius: 8px;
  padding: 1rem;
  cursor: pointer;
  text-align: center;
  transition: transform 0.2s;
}

.profile-card:hover {
  transform: translateY(-2px);
}

.profile-card.selected {
  outline: 2px solid var(--color-primary);
}

.icon { font-size: 2rem; margin-bottom: 0.5rem; }
.name { font-weight: bold; margin-bottom: 0.5rem; }

.status.synced { color: #0a0; }
.status.pending { color: #f80; }
.status.conflict { color: #c00; }
</style>
