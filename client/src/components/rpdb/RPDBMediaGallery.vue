<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { RPDBMedia } from '@/api/rpdb'
import { resolveRPDBMediaURL } from '@/api/rpdb'

const props = defineProps<{ cover?: string; media?: RPDBMedia[]; title: string }>()
const emit = defineEmits<{ openImage: [images: string[], index: number] }>()
const active = ref(0)
const thumbsRef = ref<HTMLElement | null>(null)

const items = computed(() => {
  const result = (props.media || []).filter(item => item.url?.trim())
  if (props.cover && !result.some(item => item.url === props.cover)) {
    result.unshift({ type: 'image', url: props.cover, caption: '封面' })
  }
  return result
})
const current = computed(() => items.value[active.value])
const imageItems = computed(() => items.value.filter(item => item.type !== 'video' && item.type !== 'embed'))
const imageURLs = computed(() => imageItems.value.map(item => resolveRPDBMediaURL(item.url)))
const currentPosition = computed(() => items.value.length ? `${active.value + 1} / ${items.value.length}` : '')

watch(items, (next, previous) => {
  const previousURL = previous?.[active.value]?.url
  const retainedIndex = previousURL ? next.findIndex(item => item.url === previousURL) : -1
  active.value = retainedIndex >= 0 ? retainedIndex : 0
})

function selectItem(index: number) {
  active.value = Math.min(Math.max(index, 0), items.value.length - 1)
  void nextTick(() => {
    const target = thumbsRef.value?.children[active.value] as HTMLElement | undefined
    target?.scrollIntoView?.({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
  })
}

function moveSelection(offset: number) {
  const count = items.value.length
  if (count < 2) return
  selectItem(active.value + offset)
}

function mediaLabel(item: RPDBMedia, index: number) {
  if (item.caption) return item.caption
  if (item.type === 'video') return `视频 ${index + 1}`
  if (item.type === 'embed') return `嵌入内容 ${index + 1}`
  return `预览图 ${index + 1}`
}

function openCurrentImage() {
  if (!current.value || current.value.type === 'video' || current.value.type === 'embed') return
  const url = resolveRPDBMediaURL(current.value.url)
  const index = Math.max(0, imageURLs.value.indexOf(url))
  emit('openImage', imageURLs.value, index)
}
</script>

<template>
  <section class="gallery" aria-label="作品预览图集" data-testid="rpdb-media-gallery">
    <div class="stage" data-testid="gallery-stage">
      <template v-if="current">
        <video v-if="current.type === 'video'" controls :poster="resolveRPDBMediaURL(current.thumbnail_url)"><source :src="resolveRPDBMediaURL(current.url)" /></video>
        <iframe v-else-if="current.type === 'embed'" :src="current.url" title="作品媒体" allowfullscreen></iframe>
        <button v-else type="button" class="stage-image" :aria-label="`查看大图：${current.caption || title}`" @click="openCurrentImage">
          <img :src="resolveRPDBMediaURL(current.url)" :alt="current.caption || title" />
        </button>
        <p v-if="current.caption">{{ current.caption }}</p>
        <span v-if="items.length > 1" class="position" aria-live="polite">{{ currentPosition }}</span>
        <button
          v-if="items.length > 1"
          type="button"
          class="stage-nav previous"
          aria-label="上一张预览"
          title="上一张"
          @click="moveSelection(-1)"
        >
          <i class="ri-arrow-left-s-line"></i>
        </button>
        <button
          v-if="items.length > 1"
          type="button"
          class="stage-nav next"
          aria-label="下一张预览"
          title="下一张"
          @click="moveSelection(1)"
        >
          <i class="ri-arrow-right-s-line"></i>
        </button>
      </template>
      <div v-else class="empty"><i class="ri-image-add-line"></i><span>作者还没有上传效果展示</span></div>
    </div>
    <footer v-if="items.length" class="media-reel" data-testid="gallery-reel">
      <div class="reel-track">
        <button
          type="button"
          class="reel-nav"
          :disabled="active === 0"
          aria-label="选择上一项素材"
          title="上一项"
          @click="moveSelection(-1)"
        >
          <i class="ri-arrow-left-s-line"></i>
        </button>
        <div ref="thumbsRef" class="thumbs" aria-label="预览缩略图列表" data-testid="gallery-thumbnails">
          <button
            v-for="(item,index) in items"
            :key="`${item.url}-${index}`"
            type="button"
            :class="{ active: index === active }"
            :aria-label="`切换到${mediaLabel(item, index)}`"
            :aria-pressed="index === active"
            :title="mediaLabel(item, index)"
            @click="selectItem(index)"
          >
            <img v-if="item.type === 'image' || item.type === 'gif'" :src="resolveRPDBMediaURL(item.thumbnail_url || item.url)" alt="" />
            <img v-else-if="item.type === 'video' && item.thumbnail_url" :src="resolveRPDBMediaURL(item.thumbnail_url)" alt="" />
            <i v-else :class="item.type === 'video' ? 'ri-play-circle-line' : 'ri-links-line'"></i>
            <span>{{ index + 1 }}</span>
          </button>
        </div>
        <button
          type="button"
          class="reel-nav"
          :disabled="active === items.length - 1"
          aria-label="选择下一项素材"
          title="下一项"
          @click="moveSelection(1)"
        >
          <i class="ri-arrow-right-s-line"></i>
        </button>
      </div>
    </footer>
  </section>
</template>

<style scoped>
.gallery{min-width:0}.stage{position:relative;display:grid;min-height:420px;place-items:center;overflow:hidden;border:1px solid rgba(255,255,255,.08);border-radius:8px;background:#18130f}
.stage>video,.stage>iframe{width:100%;height:100%;max-height:640px;object-fit:contain;border:0}.stage-image{display:grid;width:100%;height:100%;min-height:inherit;place-items:center;padding:0;border:0;background:transparent;cursor:zoom-in}.stage-image img{width:100%;height:100%;max-height:640px;object-fit:contain}.stage-image:focus-visible{outline:2px solid var(--color-accent);outline-offset:-3px}.stage p{position:absolute;z-index:2;inset:auto 56px 12px 56px;margin:0;padding:7px 10px;border-radius:4px;background:rgba(20,14,10,.82);color:#fff;font-size:12px;text-align:center}
.position{position:absolute;z-index:2;top:12px;right:12px;padding:4px 8px;border:1px solid rgba(255,255,255,.14);border-radius:4px;background:rgba(20,14,10,.76);color:#fff;font-size:11px;font-variant-numeric:tabular-nums}
.stage-nav{position:absolute;top:50%;display:grid;width:38px;height:46px;place-items:center;transform:translateY(-50%);border:1px solid rgba(255,255,255,.16);border-radius:5px;background:rgba(20,14,10,.76);color:#fff;cursor:pointer}.stage-nav:hover,.stage-nav:focus-visible{border-color:var(--color-accent);background:rgba(20,14,10,.94);outline:none}.stage-nav i{font-size:24px}.stage-nav.previous{left:10px}.stage-nav.next{right:10px}
.empty{display:flex;flex-direction:column;align-items:center;gap:10px;color:#bba99a}.empty i{font-size:42px}.media-reel{margin-top:8px;padding:8px;border:1px solid rgba(255,255,255,.08);border-radius:6px;background:#211914}.reel-track{display:grid;grid-template-columns:28px minmax(0,1fr) 28px;align-items:center;gap:6px}.reel-nav{display:grid;width:28px;height:56px;place-items:center;border:1px solid rgba(255,255,255,.1);border-radius:4px;background:#18130f;color:#fff;cursor:pointer}.reel-nav:hover:not(:disabled),.reel-nav:focus-visible{border-color:var(--color-accent);outline:none}.reel-nav:disabled{cursor:default;opacity:.3}.reel-nav i{font-size:18px}.thumbs{display:flex;min-width:0;gap:7px;overflow-x:auto;scrollbar-width:none}.thumbs::-webkit-scrollbar{display:none}.thumbs button{position:relative;width:88px;height:56px;flex:0 0 auto;overflow:hidden;border:2px solid transparent;border-radius:4px;background:#18130f;color:#fff;cursor:pointer}.thumbs button:hover,.thumbs button:focus-visible{border-color:color-mix(in srgb,var(--color-accent) 58%,transparent);outline:none}.thumbs button.active{border-color:var(--color-accent);box-shadow:0 0 0 1px color-mix(in srgb,var(--color-accent) 28%,transparent)}.thumbs button.active::before{position:absolute;z-index:2;top:0;left:50%;width:18px;height:3px;transform:translateX(-50%);background:var(--color-accent);content:''}.thumbs img{width:100%;height:100%;object-fit:cover}.thumbs i{display:grid;height:100%;place-items:center;font-size:24px}.thumbs span{position:absolute;right:3px;bottom:3px;display:grid;min-width:17px;height:17px;place-items:center;border-radius:3px;background:rgba(20,14,10,.84);font-size:9px;font-variant-numeric:tabular-nums}
@media(max-width:700px){.stage{min-height:260px}.thumbs button{width:78px}.reel-track{grid-template-columns:26px minmax(0,1fr) 26px}.reel-nav{width:26px}}
@media(prefers-reduced-motion:reduce){.thumbs{scroll-behavior:auto}}
</style>
