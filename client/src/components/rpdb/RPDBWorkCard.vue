<script setup lang="ts">
import { computed } from 'vue'
import type { RPDBWork } from '@/api/rpdb'
import { resolveRPDBMediaURL } from '@/api/rpdb'
import { sortRPDBStyleTags } from '@/constants/rpdbStyles'
import { buildNameStyle } from '@/utils/userNameStyle'
import { normalizeCharacterCardHexForCSS } from '@/utils/characterCardColor'

const props = withDefaults(defineProps<{
  work: RPDBWork
  layout?: 'card' | 'compact' | 'mini'
}>(), {
  layout: 'card',
})
defineEmits<{ open: [] }>()

const itemTypeLabels: Record<string, string> = {
  item: '物品',
  equipment: '装备',
  toy: '玩具',
  quest_item: '任务道具',
}
const typeLabel = computed(() => ({ item_showcase: '魔兽物品', transmog: '幻化方案', home_showcase: '家宅分享', musician_midi: 'Musician MIDI' }[props.work.type]))
const typeIcon = computed(() => ({ item_showcase: 'ri-magic-line', transmog: 'ri-shirt-line', home_showcase: 'ri-home-heart-line', musician_midi: 'ri-music-2-line' }[props.work.type]))
const coverURL = computed(() => resolveRPDBMediaURL(props.work.cover_image))
const styleTags = computed(() => sortRPDBStyleTags((props.work.tags || []).filter(tag => tag.name.endsWith('风格'))).slice(0, 3))
const availabilityLabel = computed(() => {
  if (!props.work.availability_status) return ''
  if (props.work.type === 'home_showcase') return props.work.availability_status === 'available' ? '可参观' : '仅展示'
  if (props.work.type === 'musician_midi') return '可下载'
  return props.work.availability_status === 'removed' ? '已绝版' : props.work.availability_status === 'limited' ? '限时获取' : '可获取'
})
const itemTypeLabel = computed(() => itemTypeLabels[props.work.item_type || 'item'] || '物品')
const bindTrait = computed(() => {
  if (props.work.bind_type === 'no') return { label: '不绑定', icon: 'ri-link-unlink', tone: 'unbound' }
  if (props.work.bind_type === 'yes') return { label: '已绑定', icon: 'ri-link', tone: 'bound' }
  if (props.work.bind_type === 'account') return { label: '账号绑定', icon: 'ri-link', tone: 'bound' }
  if (props.work.bind_type === 'pickup') return { label: '拾取绑定', icon: 'ri-link', tone: 'bound' }
  if (props.work.bind_type === 'use') return { label: '使用绑定', icon: 'ri-link', tone: 'bound' }
  return { label: '绑定未知', icon: 'ri-question-line', tone: 'unknown' }
})

function formatCount(value?: number) {
  return new Intl.NumberFormat('zh-CN', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value || 0)
}
</script>

<template>
  <article
    data-testid="rpdb-work-card"
    class="work-card"
    :class="{
      'work-card--compact': layout === 'compact',
      'work-card--mini': layout === 'mini',
    }"
    role="button"
    tabindex="0"
    @click="$emit('open')"
    @keydown.enter="$emit('open')"
    @keydown.space.prevent="$emit('open')"
  >
    <div class="work-card__media" :class="{ empty: !coverURL }">
      <img v-if="coverURL" :src="coverURL" :alt="work.title" loading="lazy" />
      <div v-else class="work-card__placeholder"><i :class="typeIcon"></i><span>等待作者上传封面</span></div>
      <div class="work-card__shade"></div>
      <span class="work-card__type"><i :class="typeIcon"></i>{{ typeLabel }}</span>
      <span v-if="work.media_count" class="work-card__media-count"><i class="ri-image-2-line"></i>{{ work.media_count }}</span>
      <div class="work-card__headline">
        <h3>{{ work.title }}</h3>
        <p>{{ work.summary || work.effect_description || '作者尚未填写作品摘要。' }}</p>
      </div>
    </div>

    <div class="work-card__body">
      <div class="work-card__details">
        <div v-if="layout !== 'card'" class="work-card__compact-headline">
          <span><i :class="typeIcon"></i>{{ typeLabel }}</span>
          <h3>{{ work.title }}</h3>
          <p>{{ work.summary || work.effect_description || '作者尚未填写作品摘要。' }}</p>
        </div>
        <div v-if="work.type === 'item_showcase'" class="work-card__item-traits" data-testid="rpdb-item-traits">
          <span class="item-trait type" :aria-label="`物品类型 ${itemTypeLabel}`" title="物品类型"><i class="ri-archive-2-line"></i>{{ itemTypeLabel }}</span>
          <span class="item-trait" :class="bindTrait.tone" :aria-label="`是否绑定 ${bindTrait.label}`" title="是否绑定"><i :class="bindTrait.icon"></i>{{ bindTrait.label }}</span>
        </div>
        <div class="work-card__flags">
          <span v-for="tag in styleTags" :key="tag.id" class="style-tag" :style="{ '--tag-color': `#${tag.color || 'B87333'}` }">{{ tag.name }}</span>
          <span v-if="availabilityLabel" class="availability">{{ availabilityLabel }}</span>
        </div>
        <div class="work-card__author">
          <span class="author-mark">
            <span>{{ (work.author_name || 'U').charAt(0).toUpperCase() }}</span>
            <img
              v-if="work.author_avatar"
              :src="resolveRPDBMediaURL(work.author_avatar)"
              :alt="`${work.author_name || '发布者'}的头像`"
              loading="lazy"
              @error="($event.currentTarget as HTMLImageElement).hidden = true"
            >
          </span>
          <b :style="buildNameStyle(normalizeCharacterCardHexForCSS(work.author_name_color))">{{ work.author_name || '匿名贡献者' }}</b>
        </div>
        <div v-if="work.recommendation_reasons?.length" class="work-card__recommendation" data-testid="rpdb-recommendation-reasons">
          <span v-for="reason in work.recommendation_reasons.slice(0, 2)" :key="reason">{{ reason }}</span>
          <b title="推荐匹配分"><i class="ri-sparkling-line"></i>{{ work.recommendation_score }}</b>
        </div>
      </div>
      <div class="work-card__metrics" aria-label="作品数据" data-testid="rpdb-work-metrics">
        <span title="浏览（用户每日计1次）" :aria-label="`浏览 ${formatCount(work.view_count)}`"><i class="ri-eye-line"></i><b>{{ formatCount(work.view_count) }}</b></span>
        <span title="点赞" :aria-label="`点赞 ${formatCount(work.like_count)}`"><i class="ri-heart-3-line"></i><b>{{ formatCount(work.like_count) }}</b></span>
        <span title="收藏" :aria-label="`收藏 ${formatCount(work.favorite_count)}`"><i class="ri-bookmark-3-line"></i><b>{{ formatCount(work.favorite_count) }}</b></span>
        <span title="加入清单" :aria-label="`加入清单 ${formatCount(work.list_count)}`"><i class="ri-list-check-3"></i><b>{{ formatCount(work.list_count) }}</b></span>
      </div>
    </div>
  </article>
</template>

<style scoped src="./RPDBWorkCard.css"></style>
