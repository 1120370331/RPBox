<script setup lang="ts">
import { computed } from 'vue'
import type { RPDBWork } from '@/api/rpdb'
import { resolveRPDBMediaURL } from '@/api/rpdb'
import { sortRPDBStyleTags } from '@/constants/rpdbStyles'

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
          <b :style="{ color: work.author_name_color ? `#${work.author_name_color}` : undefined }">{{ work.author_name || '匿名贡献者' }}</b>
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

<style scoped>
.work-card{min-width:0;overflow:hidden;border:1px solid var(--color-border);border-radius:var(--radius-md);background:var(--color-card-bg);color:var(--color-text-main);box-shadow:var(--shadow-sm);cursor:pointer;transition:transform .18s ease,border-color .18s ease,box-shadow .18s ease}.work-card:hover,.work-card:focus-visible{transform:translateY(-2px);border-color:var(--color-border-hover);box-shadow:var(--shadow-md);outline:none}.work-card__media{position:relative;height:180px;overflow:hidden;background:var(--color-panel-bg)}.work-card__media img{width:100%;height:100%;object-fit:cover;transition:transform .3s ease}.work-card:hover .work-card__media img{transform:scale(1.035)}.work-card__media.empty{background:linear-gradient(135deg,color-mix(in srgb,var(--color-card-bg) 78%,var(--color-accent)),var(--color-panel-bg))}.work-card__placeholder{position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:7px;color:var(--color-text-secondary);font-size:11px}.work-card__placeholder i{color:var(--icon-color);font-size:34px}.work-card__shade{position:absolute;inset:0;background:linear-gradient(180deg,rgba(44,24,16,0) 30%,rgba(44,24,16,.88) 100%)}.work-card__media.empty .work-card__shade{background:linear-gradient(180deg,transparent 24%,color-mix(in srgb,var(--color-panel-bg) 92%,transparent) 100%)}.work-card__type,.work-card__media-count{position:absolute;z-index:1;top:10px;display:inline-flex;align-items:center;gap:5px;min-height:24px;padding:0 8px;border:1px solid var(--color-border);border-radius:4px;background:color-mix(in srgb,var(--color-panel-bg) 88%,transparent);color:var(--tag-text);font-size:10px;font-weight:800;backdrop-filter:blur(6px)}.work-card__type{left:10px}.work-card__media-count{right:10px}.work-card__headline{position:absolute;z-index:1;right:12px;bottom:11px;left:12px}.work-card__headline h3{overflow:hidden;margin:0;color:#fff;font:700 17px/1.25 Georgia,'Microsoft YaHei',serif;white-space:nowrap;text-overflow:ellipsis}.work-card__headline p{display:-webkit-box;min-height:30px;overflow:hidden;margin:5px 0 0;color:rgba(255,255,255,.78);font-size:11px;line-height:1.45;-webkit-box-orient:vertical;-webkit-line-clamp:2}.work-card__media.empty .work-card__headline h3{color:var(--color-text-main)}.work-card__media.empty .work-card__headline p{color:var(--color-text-secondary)}.work-card__body{display:grid;gap:9px;padding:10px 11px}.work-card__details{display:grid;gap:9px;min-width:0}.work-card__flags{display:flex;align-items:center;gap:6px;min-height:24px;overflow:hidden}.style-tag,.availability{padding:3px 6px;border-radius:4px;background:var(--tag-bg);color:var(--tag-text);font-size:10px;white-space:nowrap}.style-tag{border:1px solid color-mix(in srgb,var(--tag-color) 55%,var(--color-border));background:color-mix(in srgb,var(--tag-color) 11%,var(--tag-bg));color:var(--color-text-main)}.work-card__author{display:flex;min-width:0;align-items:center;gap:7px;color:var(--color-text-secondary);font-size:11px}.work-card__author b{overflow:hidden;white-space:nowrap;text-overflow:ellipsis}.author-mark{display:grid;width:23px;height:23px;flex:0 0 23px;place-items:center;border:1px solid var(--color-border);border-radius:50%;background:var(--icon-bg);color:var(--icon-color);font-size:9px}.work-card__metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));margin:1px -11px -10px;padding:7px 8px 8px;border-top:1px solid var(--color-border);background:color-mix(in srgb,var(--color-panel-bg) 62%,transparent)}.work-card__metrics>span{display:inline-flex;min-width:0;align-items:center;justify-content:center;gap:3px;color:var(--color-text-secondary);font-variant-numeric:tabular-nums;white-space:nowrap}.work-card__metrics i{flex:0 0 auto;color:var(--icon-color);font-size:11px}.work-card__metrics b{min-width:0;overflow:hidden;color:var(--color-text-main);font-size:10px;line-height:1;text-overflow:ellipsis}
.work-card__item-traits{display:flex;min-width:0;align-items:center;gap:6px}.item-trait{display:inline-flex;min-width:0;align-items:center;gap:4px;padding:4px 7px;border:1px solid var(--color-border);border-radius:4px;background:var(--tag-bg);color:var(--color-text-secondary);font-size:10px;font-weight:800;white-space:nowrap}.item-trait i{flex:0 0 auto;color:var(--icon-color);font-size:11px}.item-trait.type{border-color:color-mix(in srgb,var(--color-accent) 42%,var(--color-border));background:color-mix(in srgb,var(--color-accent) 8%,var(--tag-bg));color:var(--color-text-main)}.item-trait.bound{color:var(--color-text-main)}.item-trait.unbound{border-style:dashed}.item-trait.unknown{opacity:.72}
.author-mark{position:relative;overflow:hidden}.author-mark img{position:absolute;inset:0;width:100%;height:100%;object-fit:cover}
.work-card__recommendation{display:flex;min-width:0;align-items:center;gap:5px;overflow:hidden}.work-card__recommendation span{overflow:hidden;padding:3px 6px;border:1px solid color-mix(in srgb,var(--color-accent) 32%,var(--color-border));border-radius:4px;background:color-mix(in srgb,var(--color-accent) 7%,var(--tag-bg));color:var(--color-text-secondary);font-size:9px;text-overflow:ellipsis;white-space:nowrap}.work-card__recommendation b{display:inline-flex;flex:0 0 auto;align-items:center;gap:3px;margin-left:auto;color:var(--color-accent);font-size:10px}.work-card__recommendation i{font-size:11px}
.work-card__compact-headline{min-width:0}.work-card__compact-headline>span{display:inline-flex;align-items:center;gap:5px;color:var(--color-accent);font-size:10px;font-weight:800}.work-card__compact-headline h3{overflow:hidden;margin:4px 0 3px;color:var(--color-text-main);font:700 17px/1.25 Georgia,'Microsoft YaHei',serif;text-overflow:ellipsis;white-space:nowrap}.work-card__compact-headline p{display:-webkit-box;overflow:hidden;margin:0;color:var(--color-text-secondary);font-size:11px;line-height:1.45;-webkit-box-orient:vertical;-webkit-line-clamp:2}
.work-card--compact{display:grid;grid-template-columns:176px minmax(0,1fr);min-height:138px}.work-card--compact:hover,.work-card--compact:focus-visible{transform:translateX(2px)}.work-card--compact .work-card__media{height:100%;min-height:138px}.work-card--compact .work-card__headline{display:none}.work-card--compact .work-card__body{grid-template-columns:minmax(0,1fr) minmax(180px,230px);gap:14px;padding:12px 14px}.work-card--compact .work-card__details{align-content:center}.work-card--compact .work-card__metrics{grid-template-columns:repeat(2,minmax(0,1fr));margin:0;padding:9px;border:0;border-left:1px solid var(--color-border);border-radius:0;background:transparent}.work-card--compact .work-card__metrics>span{border-radius:7px;background:color-mix(in srgb,var(--color-panel-bg) 72%,transparent)}.work-card--compact .work-card__metrics i{font-size:13px}.work-card--compact .work-card__metrics b{font-size:11px}
.work-card--mini{display:grid;grid-template-columns:96px minmax(0,1fr);min-height:118px;border-radius:6px;box-shadow:none}.work-card--mini:hover,.work-card--mini:focus-visible{transform:translateY(-1px)}.work-card--mini .work-card__media{height:100%;min-height:118px}.work-card--mini .work-card__headline,.work-card--mini .work-card__media-count{display:none}.work-card--mini .work-card__type{top:7px;left:7px;min-height:20px;padding:0 5px;font-size:8px}.work-card--mini .work-card__placeholder{gap:3px}.work-card--mini .work-card__placeholder i{font-size:25px}.work-card--mini .work-card__placeholder span{display:none}.work-card--mini .work-card__body{display:grid;grid-template-columns:1fr;gap:5px;padding:8px 8px 6px}.work-card--mini .work-card__details{align-content:start;gap:5px}.work-card--mini .work-card__compact-headline>span{font-size:8px}.work-card--mini .work-card__compact-headline h3{margin:2px 0;color:var(--color-text-main);font-size:13px;line-height:1.25}.work-card--mini .work-card__compact-headline p{font-size:9px;line-height:1.35;-webkit-line-clamp:1}.work-card--mini .work-card__item-traits,.work-card--mini .work-card__flags,.work-card--mini .work-card__author{display:none}.work-card--mini .work-card__recommendation{gap:3px}.work-card--mini .work-card__recommendation span{display:none;padding:2px 4px;font-size:8px}.work-card--mini .work-card__recommendation span:first-child{display:block}.work-card--mini .work-card__recommendation b{font-size:9px}.work-card--mini .work-card__metrics{grid-template-columns:repeat(4,minmax(0,1fr));margin:0;padding:5px 0 0;border-top:1px solid var(--color-border);background:transparent}.work-card--mini .work-card__metrics>span{gap:2px}.work-card--mini .work-card__metrics i{font-size:9px}.work-card--mini .work-card__metrics b{font-size:8px}
@media(max-width:760px){.work-card--compact{grid-template-columns:126px minmax(0,1fr)}.work-card--compact .work-card__body{grid-template-columns:1fr;gap:8px;padding:10px 11px}.work-card--compact .work-card__metrics{grid-template-columns:repeat(4,minmax(0,1fr));padding:7px 0 0;border-top:1px solid var(--color-border);border-left:0}.work-card--compact .work-card__metrics>span{min-height:26px}.work-card--compact .work-card__flags{display:none}.work-card--compact .work-card__placeholder span{display:none}}
@media(prefers-reduced-motion:reduce){.work-card,.work-card__media img{transition:none}}
</style>
