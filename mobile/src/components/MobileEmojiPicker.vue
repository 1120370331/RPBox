<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { listEmotePacks, type EmoteItem, type EmotePack } from '@/api/emote'
import { buildEmoteToken } from '@/utils/emote'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'select', token: string): void
}>()

const loading = ref(false)
const loaded = ref(false)
const errorMessage = ref('')
const packs = ref<EmotePack[]>([])
const activePackId = ref('')
const searchQuery = ref('')

const activePack = computed<EmotePack | undefined>(() => {
  if (!packs.value.length) return undefined
  if (!activePackId.value) return packs.value[0]
  return packs.value.find(pack => pack.id === activePackId.value) || packs.value[0]
})

const filteredItems = computed<EmoteItem[]>(() => {
  const pack = activePack.value
  if (!pack) return []
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return pack.items
  return pack.items.filter((item) => {
    const name = (item.name || '').toLowerCase()
    const text = (item.text || '').toLowerCase()
    return name.includes(query) || text.includes(query)
  })
})

async function loadPacks(force = false) {
  if (loading.value) return
  if (loaded.value && !force) return
  loading.value = true
  errorMessage.value = ''
  try {
    const next = await listEmotePacks()
    packs.value = next
    if (!activePackId.value && next.length > 0) {
      activePackId.value = next[0].id
    }
    loaded.value = true
  } catch (error) {
    errorMessage.value = (error as Error)?.message || '表情加载失败'
  } finally {
    loading.value = false
  }
}

function closePicker() {
  emit('close')
}

function selectItem(item: EmoteItem) {
  const pack = activePack.value
  if (!pack) return
  emit('select', buildEmoteToken(pack.id, item.id))
  emit('close')
}

function lockBodyScroll(locked: boolean) {
  if (typeof document === 'undefined') return
  document.body.style.overflow = locked ? 'hidden' : ''
}

watch(
  () => props.open,
  (open) => {
    if (!open) {
      searchQuery.value = ''
      lockBodyScroll(false)
      return
    }
    lockBodyScroll(true)
    void loadPacks()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  lockBodyScroll(false)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="emoji-sheet-mask" @click.self="closePicker">
      <section class="emoji-sheet" role="dialog" aria-modal="true" aria-label="表情选择器">
        <div class="sheet-handle" />
        <header class="emoji-header">
          <div class="title-group">
            <h3>表情</h3>
            <p v-if="activePack">{{ activePack.name }}</p>
          </div>
          <button class="close-btn" type="button" @click="closePicker" aria-label="关闭表情面板">
            <i class="ri-close-line" />
          </button>
        </header>

        <label class="emoji-search">
          <i class="ri-search-line" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索表情"
            autocomplete="off"
          >
        </label>

        <div v-if="packs.length" class="pack-strip">
          <button
            v-for="pack in packs"
            :key="pack.id"
            type="button"
            class="pack-btn"
            :class="{ active: activePack?.id === pack.id }"
            :title="pack.name"
            :aria-label="pack.name"
            @click="activePackId = pack.id"
          >
            <img v-if="pack.icon_url" :src="pack.icon_url" :alt="pack.name" loading="lazy">
            <span v-else>{{ pack.name.slice(0, 2) || '🙂' }}</span>
          </button>
        </div>

        <div class="emoji-content">
          <div v-if="loading" class="state-row">加载中...</div>
          <div v-else-if="errorMessage" class="state-row">
            <span>{{ errorMessage }}</span>
            <button type="button" @click="loadPacks(true)">重试</button>
          </div>
          <div v-else-if="filteredItems.length === 0" class="state-row">没有匹配表情</div>
          <div v-else class="emoji-grid">
            <button
              v-for="item in filteredItems"
              :key="item.id"
              type="button"
              class="emoji-item"
              :title="item.text ? `${item.name} · ${item.text}` : item.name"
              @click="selectItem(item)"
            >
              <img :src="item.url" :alt="item.name || ''" loading="lazy">
            </button>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.emoji-sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 2500;
  background: rgba(0, 0, 0, 0.44);
  display: flex;
  align-items: flex-end;
}

.emoji-sheet {
  width: 100%;
  min-height: 280px;
  max-height: min(78vh, 640px);
  background: linear-gradient(180deg, #fffefe 0%, #fcf4eb 100%);
  border-radius: 18px 18px 0 0;
  border-top: 1px solid rgba(75, 54, 33, 0.15);
  box-shadow: 0 -12px 32px rgba(28, 17, 10, 0.22);
  padding: 8px 14px calc(12px + var(--safe-bottom, 0px));
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sheet-handle {
  width: 44px;
  height: 4px;
  border-radius: 999px;
  background: rgba(75, 54, 33, 0.24);
  margin: 2px auto 0;
}

.emoji-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.title-group h3 {
  font-size: 16px;
  color: var(--color-text-main);
}

.title-group p {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.close-btn {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 1px solid var(--color-border);
  background: rgba(255, 255, 255, 0.8);
  color: var(--color-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.emoji-search {
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: #fff;
  padding: 8px 12px;
}

.emoji-search i {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.emoji-search input {
  flex: 1;
  border: none;
  background: transparent;
  outline: none;
  font-size: 14px;
  color: var(--color-text-main);
}

.pack-strip {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.pack-btn {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  border: 1px solid var(--color-border);
  background: #fff;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-main);
  font-size: 12px;
  font-weight: 600;
}

.pack-btn.active {
  border-color: var(--color-accent);
  background: rgba(184, 115, 51, 0.12);
  box-shadow: inset 0 0 0 1px rgba(184, 115, 51, 0.22);
}

.pack-btn img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 11px;
}

.emoji-content {
  min-height: 0;
  flex: 1;
}

.emoji-grid {
  height: 100%;
  overflow-y: auto;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(66px, 1fr));
  gap: 8px;
  padding: 2px 2px 2px 0;
}

.emoji-item {
  border: 1px solid rgba(75, 54, 33, 0.12);
  border-radius: 12px;
  background: #fff;
  height: 72px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.emoji-item img {
  width: 64px;
  height: 64px;
  object-fit: contain;
}

.state-row {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.state-row button {
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: #fff;
  color: var(--color-text-main);
  padding: 4px 10px;
}
</style>
