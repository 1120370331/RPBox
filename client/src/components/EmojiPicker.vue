<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useEmoteStore } from '@/stores/emote'
import { buildEmoteToken } from '@/utils/emote'
import type { EmoteItem, EmotePack } from '@/api/emote'

const props = defineProps<{
  show: boolean
  triggerElement?: HTMLElement | null
}>()

const emit = defineEmits<{
  select: [token: string]
  close: []
}>()

const emoteStore = useEmoteStore()
const pickerStyle = ref<Record<string, string>>({})
const searchQuery = ref('')
const activePackId = ref('')

const PICKER_WIDTH = 420
const PICKER_HEIGHT = 460

const packs = computed(() => emoteStore.packs)

const activePack = computed<EmotePack | undefined>(() => {
  if (!packs.value.length) return undefined
  if (activePackId.value) {
    return packs.value.find(pack => pack.id === activePackId.value) || packs.value[0]
  }
  return packs.value[0]
})

const filteredItems = computed<EmoteItem[]>(() => {
  const pack = activePack.value
  if (!pack) return []
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return pack.items
  return pack.items.filter((item) => {
    const name = item.name?.toLowerCase() || ''
    const text = item.text?.toLowerCase() || ''
    return name.includes(query) || text.includes(query)
  })
})

function handleSelect(pack: EmotePack | undefined, item: EmoteItem) {
  if (!pack) return
  emit('select', buildEmoteToken(pack.id, item.id))
  emit('close')
}

function handleClose() {
  emit('close')
}

function updatePosition() {
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight
  const maxHeight = Math.min(PICKER_HEIGHT, viewportHeight - 32)

  if (!props.triggerElement) {
    pickerStyle.value = {
      position: 'fixed',
      top: '50%',
      left: '50%',
      transform: 'translate(-50%, -50%)',
      zIndex: '3000',
      '--picker-max-height': `${maxHeight}px`
    }
    return
  }

  const rect = props.triggerElement.getBoundingClientRect()
  let top = rect.bottom + 8
  let left = rect.left

  if (left + PICKER_WIDTH > viewportWidth) {
    left = viewportWidth - PICKER_WIDTH - 16
  }
  if (left < 16) {
    left = 16
  }

  if (top + maxHeight > viewportHeight) {
    top = rect.top - maxHeight - 8
    if (top < 16) {
      top = 16
    }
  }

  pickerStyle.value = {
    position: 'fixed',
    top: `${top}px`,
    left: `${left}px`,
    zIndex: '3000',
    '--picker-max-height': `${maxHeight}px`
  }
}

watch(packs, (next) => {
  if (!activePackId.value && next.length > 0) {
    activePackId.value = next[0].id
  }
})

watch(() => props.show, async (newShow) => {
  if (!newShow) {
    searchQuery.value = ''
    window.removeEventListener('resize', updatePosition)
    window.removeEventListener('scroll', updatePosition, true)
    return
  }
  await emoteStore.loadPacks()
  if (!activePackId.value && packs.value.length > 0) {
    activePackId.value = packs.value[0].id
  }
  await nextTick()
  updatePosition()
  window.addEventListener('resize', updatePosition)
  window.addEventListener('scroll', updatePosition, true)
})

watch(() => props.triggerElement, async () => {
  if (!props.show) return
  await nextTick()
  updatePosition()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updatePosition)
  window.removeEventListener('scroll', updatePosition, true)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="show" class="emoji-picker-overlay" @click.self="handleClose">
      <div class="emoji-picker-container" :style="pickerStyle">
        <div class="emoji-header">
          <div>
            <div class="emoji-title">表情包</div>
            <div class="emoji-subtitle">点击插入到评论</div>
          </div>
          <button class="emoji-close" type="button" @click="handleClose">
            <i class="ri-close-line"></i>
          </button>
        </div>

        <div class="emoji-search">
          <i class="ri-search-line"></i>
          <input v-model="searchQuery" type="text" placeholder="搜索表情包文案" />
        </div>

        <div class="emoji-body">
          <aside class="emoji-categories custom-scrollbar">
            <button
              v-for="pack in packs"
              :key="pack.id"
              type="button"
              class="emoji-category"
              :class="{ active: activePack?.id === pack.id }"
              @click="activePackId = pack.id"
              :title="pack.name"
              :aria-label="pack.name"
            >
              <img v-if="pack.icon_url" :src="pack.icon_url" alt="" class="pack-icon" />
            </button>
          </aside>

          <section class="emoji-panel">
            <div class="emoji-section-title">
              <span>表情包</span>
              <span v-if="activePack" class="emoji-count">{{ filteredItems.length }} 张</span>
            </div>
            <div v-if="emoteStore.loading" class="emoji-empty">加载中...</div>
            <div v-else-if="filteredItems.length === 0" class="emoji-empty">没有匹配表情</div>
            <div v-else class="emoji-grid custom-scrollbar">
              <button
                v-for="item in filteredItems"
                :key="item.id"
                type="button"
                class="emoji-btn"
                @click="handleSelect(activePack, item)"
                :title="item.text ? `${item.name} · ${item.text}` : item.name"
              >
                <img :src="item.url" :alt="item.name" />
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.emoji-picker-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: rgba(0, 0, 0, 0.08);
}

.emoji-picker-container {
  width: 420px;
  max-height: var(--picker-max-height, 460px);
  display: flex;
  flex-direction: column;
  background: var(--color-panel-bg);
  border-radius: 18px;
  border: 1px solid var(--color-border);
  box-shadow: 0 18px 44px rgba(75, 54, 33, 0.18);
  overflow: hidden;
}

.emoji-picker-container::before {
  content: '';
  display: block;
  height: 4px;
  background: linear-gradient(90deg, transparent, var(--color-accent), transparent);
}

.emoji-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px 10px;
  background: var(--color-panel-bg);
  border-bottom: 1px solid var(--color-border);
}

.emoji-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-main);
}

.emoji-subtitle {
  font-size: 11px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.emoji-close {
  border: none;
  background: none;
  font-size: 18px;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: 2px;
}

.emoji-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--color-border);
}

.emoji-search i {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.emoji-search input {
  flex: 1;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  padding: 6px 10px;
  outline: none;
  font-size: 13px;
  color: var(--color-text-main);
  background: var(--color-panel-bg);
}

.emoji-body {
  display: grid;
  grid-template-columns: 80px minmax(0, 1fr);
  gap: 0;
  flex: 1;
  min-height: 0;
}

.emoji-categories {
  padding: 12px 8px;
  background: var(--color-card-bg);
  border-right: 1px solid var(--color-border);
  overflow-y: auto;
}

.emoji-category {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px 6px;
  border-radius: 12px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  margin-bottom: 8px;
}

.emoji-category.active {
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
  border-color: var(--btn-primary-bg);
}

.pack-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  border: 1px solid var(--color-border);
  background: var(--color-panel-bg);
  object-fit: cover;
}

.emoji-category.active .pack-icon {
  border-color: var(--btn-primary-text);
}

.emoji-panel {
  padding: 12px 14px 16px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.emoji-section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 10px;
}

.emoji-count {
  font-size: 11px;
  color: var(--icon-color);
}

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
  gap: 10px;
  overflow-y: auto;
  padding-right: 4px;
  min-height: 0;
}

.emoji-btn {
  border: none;
  background: var(--color-panel-bg);
  padding: 6px;
  border-radius: 12px;
  cursor: pointer;
  transition: background 0.2s ease, transform 0.2s ease;
  box-shadow: inset 0 0 0 1px var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  height: 76px;
}

.emoji-btn img {
  width: 64px;
  height: 64px;
  display: block;
  object-fit: contain;
}

.emoji-btn:hover {
  background: var(--btn-outline-hover);
  transform: translateY(-1px);
}

.emoji-empty {
  font-size: 12px;
  color: var(--color-text-secondary);
  padding: 12px 0;
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
  border-radius: 3px;
}

@media (max-width: 480px) {
  .emoji-picker-container {
    width: 340px;
  }

  .emoji-body {
    grid-template-columns: 68px minmax(0, 1fr);
  }

  .emoji-grid {
    grid-template-columns: repeat(auto-fill, minmax(64px, 1fr));
  }
}
</style>
