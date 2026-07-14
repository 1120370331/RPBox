<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listRPDBWorks, type RPDBWork } from '@/api/rpdb'
import RPDBWorkCard from './RPDBWorkCard.vue'

const props = defineProps<{
  userId: number
  isOwnProfile: boolean
}>()

const router = useRouter()
const loading = ref(false)
const works = ref<RPDBWork[]>([])
const total = ref(0)

watch(
  () => props.userId,
  () => loadWorks(),
  { immediate: true },
)

async function loadWorks() {
  if (!props.userId) {
    works.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const response = await listRPDBWorks({
      author_id: props.userId,
      page: 1,
      page_size: 6,
      sort: 'updated_at',
    })
    works.value = response.works || []
    total.value = response.total || 0
  } catch (error) {
    console.error('加载用户 RP 数据库贡献失败:', error)
    works.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function openWork(id: number) {
  router.push({ name: 'rpdb-detail', params: { id } })
}
</script>

<template>
  <section class="rpdb-contributions">
    <header>
      <div>
        <span>RP 数据库贡献</span>
        <h2>{{ isOwnProfile ? '我的 RP 数据库贡献' : 'RP 数据库贡献' }}</h2>
        <p>玩家发布的魔兽物品、幻化方案和家宅分享。</p>
      </div>
      <div class="contribution-actions">
        <strong>{{ total }}<small>篇公开作品</small></strong>
        <router-link v-if="isOwnProfile" to="/rpdb/create"><i class="ri-add-line"></i>发布作品</router-link>
        <router-link to="/rpdb"><i class="ri-compass-3-line"></i>浏览 RPDB</router-link>
      </div>
    </header>

    <div v-if="loading" class="contribution-state">
      <i class="ri-loader-4-line"></i>
      <span>加载贡献作品...</span>
    </div>
    <div v-else-if="works.length === 0" class="contribution-state">
      <i class="ri-archive-drawer-line"></i>
      <span>{{ isOwnProfile ? '还没有公开作品，从发布第一个作品开始。' : '这位玩家还没有公开 RP 数据库作品。' }}</span>
    </div>
    <div v-else class="contribution-grid">
      <RPDBWorkCard v-for="work in works" :key="work.id" :work="work" @open="openWork(work.id)" />
    </div>
  </section>
</template>

<style scoped>
.rpdb-contributions{grid-column:span 12;padding:24px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-panel-bg);box-shadow:var(--shadow-md)}header{display:flex;align-items:flex-end;justify-content:space-between;gap:20px;margin-bottom:18px}header span{color:var(--color-accent);font-size:10px;font-weight:800;letter-spacing:.14em}header h2{margin:6px 0 5px;color:var(--color-text-main);font-size:24px}header p{margin:0;color:var(--color-text-secondary);font-size:13px}.contribution-actions{display:flex;align-items:center;justify-content:flex-end;gap:10px;flex-wrap:wrap}.contribution-actions strong{display:flex;align-items:baseline;gap:5px;margin-right:4px;color:var(--color-accent);font-size:24px}.contribution-actions small{color:var(--color-text-secondary);font-size:11px;font-weight:600}.contribution-actions a{display:inline-flex;align-items:center;gap:5px;min-height:36px;padding:0 11px;border:1px solid var(--color-border);border-radius:7px;background:var(--color-card-bg);color:var(--color-text-main);font-size:12px;font-weight:700;text-decoration:none}.contribution-actions a:first-of-type{border-color:var(--color-accent);background:var(--color-accent);color:var(--color-accent-contrast)}.contribution-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}.contribution-state{display:flex;min-height:180px;align-items:center;justify-content:center;gap:8px;border:1px dashed var(--color-border);border-radius:10px;color:var(--color-text-secondary)}.contribution-state i{font-size:24px;color:var(--color-accent)}@media(max-width:1100px){.contribution-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:700px){.rpdb-contributions{padding:18px 14px}header{align-items:flex-start;flex-direction:column}.contribution-actions{justify-content:flex-start}.contribution-grid{grid-template-columns:1fr}}
</style>
