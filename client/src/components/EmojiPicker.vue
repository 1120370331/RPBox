<script setup lang="ts">
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  'select': [emoji: string]
  'close': []
}>()

function handleSelect(emoji: any) {
  emit('select', emoji.i)
  emit('close')
}

function handleClose() {
  emit('close')
}
</script>

<template>
  <div v-if="show" class="emoji-picker-overlay" @click.self="handleClose">
    <div class="emoji-picker-container">
      <EmojiPicker
        :native="true"
        @select="handleSelect"
      />
    </div>
  </div>
</template>

<style scoped>
.emoji-picker-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.emoji-picker-container {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}
</style>
