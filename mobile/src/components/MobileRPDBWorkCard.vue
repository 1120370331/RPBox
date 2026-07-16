<script setup lang="ts">
import { computed } from 'vue'
import type { RPDBWork } from '@/api/rpdb'
import { resolveRPDBMediaUrl } from '@/api/rpdb'
import { getRPDBSummary, getRPDBTypeIcon, getRPDBTypeLabel } from '@/utils/rpdb'
import CachedImage from './CachedImage.vue'

const props = withDefaults(defineProps<{
  work: RPDBWork
  compact?: boolean
}>(), {
  compact: false,
})

defineEmits<{
  (event: 'open'): void
}>()

const styleTags = computed(() => (props.work.tags || []).filter(tag => tag.name.endsWith('风格')).slice(0, 2))
</script>

<template>
  <button class="work-card" :class="{ compact }" type="button" @click="$emit('open')">
    <div class="work-cover">
      <CachedImage
        v-if="work.cover_image"
        :src="resolveRPDBMediaUrl(work.cover_image)"
        :alt="work.title"
        loading="lazy"
      />
      <div v-else class="cover-placeholder">
        <i :class="getRPDBTypeIcon(work.type)" />
      </div>
      <span class="type-mark"><i :class="getRPDBTypeIcon(work.type)" />{{ getRPDBTypeLabel(work.type) }}</span>
    </div>

    <div class="work-copy">
      <div v-if="styleTags.length" class="tag-row">
        <span v-for="tag in styleTags" :key="tag.id">{{ tag.name }}</span>
      </div>
      <h3>{{ work.title }}</h3>
      <p>{{ getRPDBSummary(work) }}</p>
      <footer>
        <span class="author">{{ work.author_name || '匿名贡献者' }}</span>
        <span><i class="ri-eye-line" />{{ work.view_count || 0 }}</span>
        <span><i class="ri-heart-3-line" />{{ work.like_count || 0 }}</span>
      </footer>
    </div>
  </button>
</template>

<style scoped>
.work-card {
  display: flex;
  width: 100%;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(75, 54, 33, 0.1);
  border-radius: 8px;
  background: var(--color-card-bg);
  color: var(--color-text-main);
  text-align: left;
  box-shadow: var(--shadow-sm);
}

.work-cover {
  position: relative;
  width: 100%;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  background: #d8c8b8;
}

.cover-placeholder {
  display: grid;
  width: 100%;
  height: 100%;
  place-items: center;
  background:
    linear-gradient(145deg, rgba(75, 54, 33, 0.12), rgba(184, 115, 51, 0.2)),
    var(--color-border);
  color: var(--color-secondary);
}

.cover-placeholder i {
  font-size: 34px;
}

.type-mark {
  position: absolute;
  top: 8px;
  left: 8px;
  display: inline-flex;
  min-height: 24px;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  border-radius: 4px;
  background: rgba(44, 24, 16, 0.78);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  backdrop-filter: blur(5px);
}

.work-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  padding: 10px;
}

.tag-row {
  display: flex;
  min-width: 0;
  gap: 5px;
  margin-bottom: 6px;
  overflow: hidden;
}

.tag-row span {
  overflow: hidden;
  padding: 2px 5px;
  border-radius: 3px;
  background: var(--tag-bg);
  color: var(--tag-text);
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

h3 {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--color-text-main);
  font-size: 14px;
  font-weight: 750;
  line-height: 1.35;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

p {
  display: -webkit-box;
  overflow: hidden;
  margin: 6px 0 10px;
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.5;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

footer {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  margin-top: auto;
  color: var(--color-text-secondary);
  font-size: 10px;
}

footer span {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.author {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compact {
  display: grid;
  grid-template-columns: 116px minmax(0, 1fr);
}

.compact .work-cover {
  height: 100%;
  min-height: 116px;
  aspect-ratio: auto;
}

.compact .work-copy {
  padding: 11px;
}

.compact p {
  -webkit-line-clamp: 2;
}

@media (max-width: 350px) {
  .compact {
    grid-template-columns: 102px minmax(0, 1fr);
  }
}
</style>
