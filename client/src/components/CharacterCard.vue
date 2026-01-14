<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { type Character } from '@/api/character'
import WowIcon from './WowIcon.vue'
import RButton from './RButton.vue'

interface Props {
  visible: boolean
  character?: Character
  speaker?: string
  position?: { x: number; y: number }
  editable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  editable: true
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'edit': [character: Character]
}>()

const isVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

// 计算卡片位置（确保不超出屏幕）
const cardStyle = computed(() => {
  if (!props.position) return {}

  const cardHeight = 400  // 估算卡片高度
  const cardWidth = 320   // 卡片宽度
  const padding = 20      // 边距

  let top = props.position.y
  let left = props.position.x + padding

  // 检查是否超出底部
  if (top + cardHeight > window.innerHeight - padding) {
    top = window.innerHeight - cardHeight - padding
  }

  // 检查是否超出顶部
  if (top < padding) {
    top = padding
  }

  // 检查是否超出右侧
  if (left + cardWidth > window.innerWidth - padding) {
    left = props.position.x - cardWidth - padding
  }

  return {
    top: `${top}px`,
    left: `${left}px`,
  }
})

// 获取显示名称
const displayName = computed(() => {
  if (!props.character) return props.speaker || '未知'
  return props.character.custom_name ||
    (props.character.first_name
      ? (props.character.last_name
        ? `${props.character.first_name} ${props.character.last_name}`
        : props.character.first_name)
      : props.speaker || '未知')
})

// 获取显示颜色
const displayColor = computed(() => {
  if (!props.character) return ''
  return props.character.custom_color || props.character.color || ''
})

// 获取显示图标
const displayIcon = computed(() => {
  if (!props.character) return ''
  return props.character.custom_avatar || props.character.icon || ''
})

// 解析第一印象（PE/Glance）数据 - 来自misc_info.PE
interface GlanceSlot {
  active: boolean
  icon: string
  title: string
  text: string
}

const glanceSlots = computed<GlanceSlot[]>(() => {
  if (!props.character?.misc_info) return []
  try {
    const misc = JSON.parse(props.character.misc_info)
    if (!misc.PE) return []
    const slots: GlanceSlot[] = []
    for (let i = 1; i <= 5; i++) {
      const slot = misc.PE[String(i)]
      if (slot && slot.AC) {
        slots.push({
          active: slot.AC,
          icon: slot.IC || '',
          title: slot.TI || '',
          text: slot.TX || ''
        })
      }
    }
    return slots
  } catch {
    return []
  }
})

function close() {
  isVisible.value = false
}

function handleEdit() {
  if (props.character) {
    emit('edit', props.character)
  }
}

// 点击外部关闭
function handleClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.character-card')) {
    close()
  }
}

watch(isVisible, (val) => {
  if (val) {
    setTimeout(() => {
      document.addEventListener('click', handleClickOutside)
    }, 100)
  } else {
    document.removeEventListener('click', handleClickOutside)
  }
})
</script>

<template>
  <Teleport to="body">
    <Transition name="card-slide">
      <div v-if="isVisible" class="character-card" :style="cardStyle">
        <div class="card-arrow"></div>

        <!-- 头部 -->
        <div class="card-header">
          <div class="avatar-section">
            <div class="avatar">
              <WowIcon v-if="displayIcon" :icon="displayIcon" :size="64" :fallback="displayName.charAt(0)" />
              <span v-else class="avatar-text">{{ displayName.charAt(0) }}</span>
            </div>
          </div>
          <div class="info-section">
            <div class="name" :style="displayColor ? { color: '#' + displayColor } : {}">
              {{ displayName }}
            </div>
            <div v-if="character?.title" class="title">{{ character.title }}</div>
            <div v-if="character?.full_title" class="full-title">{{ character.full_title }}</div>
          </div>
          <button class="btn-close" @click="close">
            <i class="ri-close-line"></i>
          </button>
        </div>

        <!-- 基本信息 -->
        <div v-if="character" class="card-body">
          <div class="info-grid">
            <div v-if="character.race" class="info-item">
              <span class="label">种族</span>
              <span class="value">{{ character.race }}</span>
            </div>
            <div v-if="character.class" class="info-item">
              <span class="label">职业</span>
              <span class="value">{{ character.class }}</span>
            </div>
            <div v-if="character.age" class="info-item">
              <span class="label">年龄</span>
              <span class="value">{{ character.age }}</span>
            </div>
            <div v-if="character.height" class="info-item">
              <span class="label">身高</span>
              <span class="value">{{ character.height }}</span>
            </div>
            <div v-if="character.eye_color" class="info-item">
              <span class="label">瞳色</span>
              <span class="value">{{ character.eye_color }}</span>
            </div>
          </div>

          <!-- 第一印象 (Glance/PE) -->
          <div v-if="glanceSlots.length > 0" class="glance-section">
            <div class="section-title">第一印象</div>
            <div class="glance-list">
              <div v-for="(slot, idx) in glanceSlots" :key="idx" class="glance-item">
                <WowIcon v-if="slot.icon" :icon="slot.icon" :size="24" class="glance-icon" />
                <div class="glance-content">
                  <div class="glance-title">{{ slot.title }}</div>
                  <div v-if="slot.text" class="glance-text">{{ slot.text }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 底部操作 -->
        <div v-if="editable && character" class="card-footer">
          <RButton size="small" @click="handleEdit">
            <i class="ri-edit-line"></i> 编辑本剧情角色
          </RButton>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.character-card {
  position: fixed;
  z-index: 1000;
  width: 320px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.card-arrow {
  position: absolute;
  left: -8px;
  top: 24px;
  width: 0;
  height: 0;
  border-top: 8px solid transparent;
  border-bottom: 8px solid transparent;
  border-right: 8px solid #fff;
}

.card-header {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: linear-gradient(135deg, #f8f4f0 0%, #efe8e0 100%);
  position: relative;
}

.avatar-section .avatar {
  width: 64px;
  height: 64px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-accent);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.avatar-text {
  font-size: 24px;
  font-weight: 600;
  color: #fff;
}

.info-section {
  flex: 1;
  min-width: 0;
}

.info-section .name {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-primary);
  margin-bottom: 4px;
}

.info-section .title {
  font-size: 13px;
  color: var(--color-accent);
  font-style: italic;
}

.info-section .full-title {
  font-size: 12px;
  color: var(--color-secondary);
  margin-top: 2px;
}

.btn-close {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 24px;
  height: 24px;
  border: none;
  background: rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-secondary);
  transition: all 0.2s;
}

.btn-close:hover {
  background: rgba(0, 0, 0, 0.2);
  color: var(--color-primary);
}

.card-body {
  padding: 16px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-item.full {
  grid-column: 1 / -1;
}

.info-item .label {
  font-size: 11px;
  color: var(--color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-item .value {
  font-size: 13px;
  color: var(--color-primary);
}

/* 第一印象 (Glance) */
.glance-section {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
}

.section-title {
  font-size: 11px;
  color: var(--color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 10px;
}

.glance-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.glance-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.glance-icon {
  flex-shrink: 0;
  border-radius: 4px;
}

.glance-content {
  flex: 1;
  min-width: 0;
}

.glance-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary);
}

.glance-text {
  font-size: 12px;
  color: var(--color-secondary);
  margin-top: 2px;
  line-height: 1.4;
}

.card-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--color-border);
  display: flex;
  justify-content: flex-end;
}

/* 动画 */
.card-slide-enter-active {
  animation: cardSlideIn 0.25s ease-out;
}

.card-slide-leave-active {
  animation: cardSlideOut 0.2s ease-in;
}

@keyframes cardSlideIn {
  from {
    opacity: 0;
    transform: translateX(-10px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes cardSlideOut {
  from {
    opacity: 1;
    transform: translateX(0);
  }
  to {
    opacity: 0;
    transform: translateX(-10px);
  }
}
</style>
