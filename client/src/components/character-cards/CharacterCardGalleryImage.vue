<script setup lang="ts">
import { computed } from 'vue'
import type { CharacterCard, CharacterCardPortraitImage } from '@/api/characterCard'
import { resolveApiUrl } from '@/api/item'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import CharacterCardPortrait from './CharacterCardPortrait.vue'

const props = withDefaults(defineProps<{
  card: Pick<CharacterCard, 'id' | 'portrait_image_url' | 'portrait_image_updated_at' | 'updated_at'>
  portrait?: CharacterCardPortraitImage | null
  alt: string
  width?: number
  quality?: number
}>(), {
  portrait: null,
  width: 900,
  quality: 88,
})

const gallerySource = computed(() => {
  if (!props.portrait || props.portrait.id <= 0 || !props.portrait.image_url) return ''
  return resolveApiUrl(props.portrait.image_url)
})
</script>

<template>
  <AuthenticatedImage v-if="gallerySource" :src="gallerySource" :alt="alt" />
  <CharacterCardPortrait
    v-else
    :card="card"
    :alt="alt"
    :width="width"
    :quality="quality"
  />
</template>
