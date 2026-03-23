<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { request } from '@shared/api/request'
import { resolveApiUrl } from '@/api/image'
import type { Item } from '@/api/item'

const router = useRouter()
const items = ref<Item[]>([])
const loading = ref(false)

async function loadMyItems() {
  loading.value = true
  try {
    const res = await request.get<{ items: Item[]; total: number }>('/items/favorites')
    items.value = res.items || []
  } catch (e) {
    console.error('Failed to load my items', e)
  } finally {
    loading.value = false
  }
}

onMounted(loadMyItems)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.myItems.title') }}</h1>
    </header>
    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="items.length === 0" class="hint">{{ $t('profile.myItems.empty') }}</div>
      <div v-else class="item-list">
        <button
          v-for="item in items"
          :key="item.id"
          class="item-row"
          @click="router.push({ name: 'item-detail', params: { id: item.id } })"
        >
          <div class="item-icon">
            <img v-if="item.preview_image_url" :src="resolveApiUrl(item.preview_image_url)" alt="" />
            <i v-else class="ri-box-3-line" />
          </div>
          <div class="item-info">
            <h3>{{ item.name }}</h3>
            <span class="item-type">{{ $t('market.types.' + item.type) }}</span>
          </div>
          <div class="item-stats">
            <span>★ {{ item.rating.toFixed(1) }}</span>
            <span><i class="ri-download-line" /> {{ item.downloads }}</span>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>
