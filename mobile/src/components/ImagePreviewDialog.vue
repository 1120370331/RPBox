<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    open: boolean
    src?: string
    alt?: string
  }>(),
  {
    src: '',
    alt: '',
  },
)

const emit = defineEmits<{
  (e: 'close'): void
}>()
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="preview-mask" @click="emit('close')">
      <button class="preview-close" @click.stop="emit('close')"><i class="ri-close-line" /></button>
      <img v-if="props.src" class="preview-image" :src="props.src" :alt="alt" @click.stop />
    </div>
  </Teleport>
</template>

<style scoped>
.preview-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.86);
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 14px;
}

.preview-close {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 34px;
  height: 34px;
  border: 1px solid rgba(255, 255, 255, 0.35);
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.35);
  color: #fff;
  font-size: 20px;
}

.preview-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
}
</style>

