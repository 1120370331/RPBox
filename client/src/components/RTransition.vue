<script setup lang="ts">
interface Props {
  name?: 'fade' | 'zoom' | 'bounce' | 'slide-up' | 'slide-down' | 'slide-left' | 'slide-right' | 'flip'
  mode?: 'in-out' | 'out-in'
  appear?: boolean
}

withDefaults(defineProps<Props>(), {
  name: 'fade',
  mode: 'out-in',
  appear: false,
})
</script>

<template>
  <Transition :name="`r-${name}`" :mode="mode" :appear="appear">
    <slot />
  </Transition>
</template>
