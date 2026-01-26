<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getPostCollection, getItemCollection, type CollectionInfo } from '../api/collection'

const props = defineProps<{
  type: 'post' | 'item'
  contentId: number
}>()

const router = useRouter()
const { t } = useI18n()

const collectionInfo = ref<CollectionInfo | null>(null)
const loading = ref(false)

const collection = computed(() => collectionInfo.value?.collection)

async function loadCollection() {
  loading.value = true
  try {
    if (props.type === 'post') {
      collectionInfo.value = await getPostCollection(props.contentId)
    } else {
      collectionInfo.value = await getItemCollection(props.contentId)
    }
  } catch (e) {
    console.error('Failed to load collection:', e)
    collectionInfo.value = null
  } finally {
    loading.value = false
  }
}

function goToCollection() {
  if (collection.value) {
    router.push({ name: 'collection-detail', params: { id: collection.value.id } })
  }
}

watch(() => props.contentId, loadCollection)
onMounted(loadCollection)
</script>

<template>
  <div v-if="collection" class="collection-banner" @click="goToCollection">
    <div class="banner-icon">
      <i class="ri-book-2-line"></i>
    </div>
    <div class="banner-content">
      <span class="banner-label">{{ t('collection.banner.belongsTo') }}：</span>
      <span class="collection-name">《{{ collection.name }}》</span>
      <span class="item-count">{{ t('collection.banner.totalCount', { count: collection.item_count }) }}</span>
    </div>
    <div class="banner-arrow">
      <i class="ri-arrow-right-s-line"></i>
    </div>
  </div>
</template>

<style scoped>
.collection-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  margin: 16px 32px;
  cursor: pointer;
  transition: all 0.2s;
}

.collection-banner:hover {
  border-color: var(--color-accent);
  background: var(--color-primary-light);
}

.banner-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-accent);
  border-radius: 8px;
  color: var(--color-text-light);
  font-size: 18px;
  flex-shrink: 0;
}

.banner-content {
  flex: 1;
  display: flex;
  align-items: baseline;
  gap: 4px;
  flex-wrap: wrap;
}

.banner-label {
  font-size: 14px;
  color: var(--color-text-muted);
}

.collection-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-secondary);
}

.item-count {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-left: 4px;
}

.banner-arrow {
  color: var(--color-text-muted);
  font-size: 20px;
  flex-shrink: 0;
}

.collection-banner:hover .banner-arrow {
  color: var(--color-accent);
}
</style>
