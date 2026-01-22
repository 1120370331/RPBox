<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps<{
  show: boolean
  triggerElement?: HTMLElement | null
}>()

const emit = defineEmits<{
  select: [emoji: string]
  close: []
}>()

const pickerStyle = ref<Record<string, string>>({})
const searchQuery = ref('')
const activeCategory = ref('recent')
const recentEmojis = ref<string[]>([])

const RECENT_KEY = 'rpbox:emoji-recent'
const MAX_RECENT = 24
const PICKER_WIDTH = 360
const PICKER_HEIGHT = 420

const fallbackRecent = [
  '😂', '🤣', '😊', '😉', '😍', '😘', '😋', '🤔',
  '😐', '🙄', '😴', '😭', '😡', '😇', '😎', '🥳',
  '👍', '👏', '🙏', '💪', '✨', '🔥', '💯', '🎉'
]

const baseCategories = [
  {
    id: 'face',
    label: '表情',
    tags: ['表情', '笑', '开心', '生气', '难过', '冷静'],
    emojis: [
      '😀', '😃', '😄', '😁', '😆', '😅', '🤣', '😂',
      '🙂', '😉', '😊', '😍', '😘', '😚', '😙', '😋',
      '😛', '😜', '🤪', '🤔', '🤐', '😐', '😑', '😶',
      '🙄', '😏', '😴', '😪', '😷', '🤒', '🤕', '😵',
      '🥴', '😵‍💫', '🤯', '😤', '😠', '😡', '🤬', '😢',
      '😭', '😮', '😲', '😳', '🥺', '🥹', '😇', '😎',
      '🤓', '🥳', '🤡'
    ]
  },
  {
    id: 'gesture',
    label: '手势',
    tags: ['手势', '动作', '支持', '鼓掌', '加油'],
    emojis: [
      '👍', '👎', '👌', '🤌', '🤏', '✌️', '🤞', '🤟',
      '🤘', '🤙', '👏', '🙌', '🫶', '🤲', '🙏', '💪',
      '👊', '✊', '🤛', '🤜', '🫡'
    ]
  },
  {
    id: 'celebrate',
    label: '庆祝',
    tags: ['庆祝', '派对', '祝贺', '礼物', '节日'],
    emojis: [
      '🎉', '🎊', '🥳', '🎈', '🎂', '🍰', '🎁', '🏆',
      '🥇', '🎯', '🧨', '✨', '🔥', '💥'
    ]
  },
  {
    id: 'heart',
    label: '爱心',
    tags: ['爱心', '喜欢', '心', '感情'],
    emojis: [
      '❤️', '🧡', '💛', '💚', '💙', '💜', '🤎', '🖤',
      '🤍', '💖', '💗', '💓', '💞', '💕', '💔', '❤️‍🔥',
      '❤️‍🩹'
    ]
  },
  {
    id: 'animal',
    label: '动物',
    tags: ['动物', '宠物', '萌'],
    emojis: [
      '🐶', '🐱', '🐭', '🐹', '🐰', '🦊', '🐻', '🐼',
      '🐨', '🐯', '🦁', '🐮', '🐷', '🐸', '🐵', '🐔',
      '🐧', '🐦', '🦉', '🐺'
    ]
  },
  {
    id: 'food',
    label: '食物',
    tags: ['食物', '饮料', '好吃'],
    emojis: [
      '🍎', '🍊', '🍋', '🍉', '🍇', '🍓', '🍑', '🍒',
      '🥝', '🍍', '🥭', '🍌', '🍔', '🍟', '🍕', '🍜',
      '🍣', '🍱', '🥟', '🍗', '🥗', '🍰', '🍪', '☕',
      '🧋', '🍺', '🍻'
    ]
  },
  {
    id: 'symbol',
    label: '符号',
    tags: ['符号', '标记', '提醒'],
    emojis: [
      '✅', '❌', '⚠️', '🚫', '⭐', '🌟', '💯', '✔️',
      '➕', '➖', '➗', '✖️', '🔔', '📌', '📝', '📣'
    ]
  }
]

const categories = computed(() => {
  const recent = recentEmojis.value.length > 0 ? recentEmojis.value : fallbackRecent
  return [
    { id: 'recent', label: '常用', tags: ['常用', '最近'], emojis: recent },
    ...baseCategories
  ]
})

const emojiIndex = computed(() => {
  const map = new Map<string, { emoji: string; tags: string[] }>()
  baseCategories.forEach((category) => {
    category.emojis.forEach((emoji) => {
      if (!map.has(emoji)) {
        map.set(emoji, { emoji, tags: [category.label, ...category.tags] })
      }
    })
  })
  recentEmojis.value.forEach((emoji) => {
    if (!map.has(emoji)) {
      map.set(emoji, { emoji, tags: ['常用', '最近'] })
    }
  })
  return Array.from(map.values())
})

const activeList = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) {
    const category = categories.value.find(item => item.id === activeCategory.value) || categories.value[0]
    return { title: category.label, emojis: category.emojis, isSearch: false }
  }

  const hits = emojiIndex.value
    .filter(item => item.tags.some(tag => tag.toLowerCase().includes(query)))
    .map(item => item.emoji)

  return { title: '搜索结果', emojis: hits, isSearch: true }
})

function loadRecentEmojis() {
  try {
    const raw = localStorage.getItem(RECENT_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    if (Array.isArray(parsed)) {
      recentEmojis.value = parsed.filter(item => typeof item === 'string')
    }
  } catch (error) {
    recentEmojis.value = []
  }
}

function saveRecentEmojis(emoji: string) {
  const next = [emoji, ...recentEmojis.value.filter(item => item !== emoji)]
  const trimmed = next.slice(0, MAX_RECENT)
  recentEmojis.value = trimmed
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(trimmed))
  } catch (error) {
    // ignore storage errors
  }
}

function handleSelect(emoji: string) {
  saveRecentEmojis(emoji)
  emit('select', emoji)
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
  let left = rect.left + rect.width / 2 - PICKER_WIDTH / 2

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

watch(() => props.show, async (newShow) => {
  if (!newShow) {
    searchQuery.value = ''
    window.removeEventListener('resize', updatePosition)
    window.removeEventListener('scroll', updatePosition, true)
    return
  }
  loadRecentEmojis()
  activeCategory.value = recentEmojis.value.length > 0 ? 'recent' : 'face'
  await nextTick()
  updatePosition()
  window.addEventListener('resize', updatePosition)
  window.addEventListener('scroll', updatePosition, true)
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
            <div class="emoji-title">表情选择</div>
            <div class="emoji-subtitle">偏好社区常用表情</div>
          </div>
          <button class="emoji-close" type="button" @click="handleClose">
            <i class="ri-close-line"></i>
          </button>
        </div>

        <div class="emoji-search">
          <i class="ri-search-line"></i>
          <input v-model="searchQuery" type="text" placeholder="搜索表情或分类" />
        </div>

        <div class="emoji-body">
          <aside class="emoji-categories custom-scrollbar">
            <button
              v-for="category in categories"
              :key="category.id"
              type="button"
              class="emoji-category"
              :class="{ active: activeCategory === category.id }"
              @click="activeCategory = category.id"
            >
              <span>{{ category.label }}</span>
              <i class="ri-arrow-right-s-line"></i>
            </button>
          </aside>

          <section class="emoji-panel">
            <div class="emoji-section-title">{{ activeList.title }}</div>
            <div v-if="activeList.emojis.length === 0" class="emoji-empty">没有匹配表情</div>
            <div v-else class="emoji-grid custom-scrollbar">
              <button
                v-for="emoji in activeList.emojis"
                :key="emoji"
                type="button"
                class="emoji-btn"
                @click="handleSelect(emoji)"
              >
                {{ emoji }}
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
  width: 360px;
  max-height: var(--picker-max-height, 420px);
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, #FFFDF9 0%, #FFF8F0 100%);
  border-radius: 18px;
  border: 1px solid #E5D4C1;
  box-shadow: 0 18px 44px rgba(75, 54, 33, 0.18);
  overflow: hidden;
}

.emoji-picker-container::before {
  content: '';
  display: block;
  height: 4px;
  background: linear-gradient(90deg, transparent, #B87333, transparent);
}

.emoji-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px 10px;
  background: rgba(255, 255, 255, 0.8);
  border-bottom: 1px solid #F5EFE7;
}

.emoji-title {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
}

.emoji-subtitle {
  font-size: 11px;
  color: #8D7B68;
  margin-top: 2px;
}

.emoji-close {
  border: none;
  background: none;
  font-size: 18px;
  color: #8D7B68;
  cursor: pointer;
  padding: 2px;
}

.emoji-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid #F5EFE7;
}

.emoji-search i {
  font-size: 14px;
  color: #8D7B68;
}

.emoji-search input {
  flex: 1;
  border: 1px solid #E5D4C1;
  border-radius: 999px;
  padding: 6px 10px;
  outline: none;
  font-size: 13px;
  color: #4B3621;
  background: #fff;
}

.emoji-body {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  gap: 0;
  flex: 1;
  min-height: 0;
}

.emoji-categories {
  padding: 12px 10px;
  background: rgba(250, 245, 238, 0.9);
  border-right: 1px solid #F5EFE7;
  overflow-y: auto;
}

.emoji-category {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 4px;
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: transparent;
  color: #8D7B68;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  margin-bottom: 6px;
}

.emoji-category i {
  font-size: 14px;
  opacity: 0.6;
}

.emoji-category.active {
  background: #2C1810;
  color: #fff;
  border-color: #2C1810;
}

.emoji-category.active i {
  opacity: 1;
  color: #fff;
}

.emoji-panel {
  padding: 12px 14px 16px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.emoji-section-title {
  font-size: 12px;
  font-weight: 600;
  color: #8D7B68;
  margin-bottom: 10px;
}

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 6px;
  overflow-y: auto;
  padding-right: 4px;
  min-height: 0;
}

.emoji-btn {
  border: none;
  background: transparent;
  font-size: 20px;
  line-height: 1;
  padding: 8px 6px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.2s ease, transform 0.2s ease;
}

.emoji-btn:hover {
  background: rgba(184, 115, 51, 0.14);
  transform: translateY(-1px);
}

.emoji-empty {
  font-size: 12px;
  color: #8D7B68;
  padding: 12px 0;
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(184, 115, 51, 0.4);
  border-radius: 3px;
}

@media (max-width: 480px) {
  .emoji-picker-container {
    width: 320px;
  }

  .emoji-grid {
    grid-template-columns: repeat(6, 1fr);
  }
}
</style>
