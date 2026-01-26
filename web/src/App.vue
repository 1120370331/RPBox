<script setup lang="ts">
import { RouterView } from 'vue-router'
import RToast from '@shared/components/RToast.vue'
import RDialog from '@shared/components/RDialog.vue'
</script>

<template>
  <RouterView />
  <RToast />
  <RDialog />
</template>
