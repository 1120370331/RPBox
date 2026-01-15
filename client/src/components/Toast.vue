<script setup lang="ts">
import { ref, onMounted } from 'vue'

const props = defineProps<{
  message: string
  type?: 'success' | 'error' | 'info'
  duration?: number
}>()

const emit = defineEmits(['close'])
const visible = ref(false)

onMounted(() => {
  visible.value = true
  setTimeout(() => {
    visible.value = false
    setTimeout(() => emit('close'), 300)
  }, props.duration || 2000)
})
</script>

<template>
  <Transition name="toast">
    <div v-if="visible" class="toast" :class="type || 'success'">
      <i :class="{
        'ri-checkbox-circle-fill': type === 'success' || !type,
        'ri-error-warning-fill': type === 'error',
        'ri-information-fill': type === 'info'
      }"></i>
      <span>{{ message }}</span>
    </div>
  </Transition>
</template>

<style scoped>
.toast {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 12px 20px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  z-index: 9999;
}

.toast.success {
  background: #E8F5E9;
  color: #2E7D32;
}

.toast.error {
  background: #FFEBEE;
  color: #C62828;
}

.toast.info {
  background: #E3F2FD;
  color: #1565C0;
}

.toast i {
  font-size: 18px;
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(100px);
}
</style>
