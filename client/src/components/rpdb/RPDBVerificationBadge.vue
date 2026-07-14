<script setup lang="ts">
import { computed } from 'vue'
import type { RPDBVerificationStatus } from '@/api/rpdb'

const props = defineProps<{
  status: RPDBVerificationStatus
}>()

const config = computed(() => ({
  unverified: { label: '未验证', icon: 'ri-question-line' },
  verified: { label: '社区已验证', icon: 'ri-verified-badge-line' },
  stale: { label: '可能过期', icon: 'ri-time-line' },
  disputed: { label: '存在争议', icon: 'ri-error-warning-line' },
}[props.status]))
</script>

<template>
  <span class="verification-badge" :class="`is-${status}`">
    <i :class="config.icon"></i>
    {{ config.label }}
  </span>
</template>

<style scoped>
.verification-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 24px;
  padding: 3px 8px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-panel-bg);
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.is-verified {
  color: #256b4a;
  border-color: rgba(37, 107, 74, 0.28);
  background: #edf8f1;
}

.is-stale {
  color: #9a5b00;
  border-color: rgba(154, 91, 0, 0.28);
  background: #fff6e6;
}

.is-disputed {
  color: #a33a35;
  border-color: rgba(163, 58, 53, 0.28);
  background: #fff0ef;
}
</style>
