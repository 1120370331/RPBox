<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { resolveRPDBMediaURL, type RPDBWork, type RPDBWorkPayload } from '@/api/rpdb'
import { hydrateJumpCards, sanitizeJumpLinks } from '@/utils/jumpLink'
import { sanitizeRichHtml } from '@/utils/sanitizeHtml'
import RPDBGuideSection from './RPDBGuideSection.vue'

const props = defineProps<{
  work: RPDBWork | RPDBWorkPayload
  homeDetails?: Record<string, string>
  transmogShareCode?: string
  copiedTransmogShareCode?: boolean
}>()
defineEmits<{ copyTransmogShareCode: [] }>()

const transmogSlotOrder = [
  { value: 'head', label: '头部' },
  { value: 'shoulder', label: '肩部' },
  { value: 'back', label: '背部' },
  { value: 'chest', label: '胸甲' },
  { value: 'shirt', label: '衬衣' },
  { value: 'tabard', label: '战袍' },
  { value: 'wrist', label: '护腕' },
  { value: 'hands', label: '手套' },
  { value: 'waist', label: '腰带' },
  { value: 'legs', label: '腿部' },
  { value: 'feet', label: '脚部' },
  { value: 'main_hand', label: '主手' },
  { value: 'off_hand', label: '副手' },
]
const transmogSlotRank = new Map(transmogSlotOrder.map((slot, index) => [slot.value, index]))
const transmogSlotLabel = new Map(transmogSlotOrder.map(slot => [slot.value, slot.label]))
const isHome = computed(() => props.work.type === 'home_showcase')
const isMusicianMIDI = computed(() => props.work.type === 'musician_midi')
const workExtra = computed<Record<string, unknown>>(() => {
  const value = props.work.extra
  if (value && typeof value === 'object' && !Array.isArray(value)) return value as Record<string, unknown>
  try {
    const parsed = JSON.parse(String(value || '{}'))
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {}
  } catch {
    return {}
  }
})
const musicianMIDIURL = computed(() => resolveRPDBMediaURL(String(workExtra.value.midi_url || '')))
const musicianMIDIName = computed(() => String(workExtra.value.midi_name || 'Musician MIDI'))
const musicianCode = computed(() => String(workExtra.value.musician_code || '').replace(/\s+/g, ''))
const copiedMusicianCode = ref(false)
const musicianMIDISize = computed(() => {
  const size = Number(workExtra.value.midi_size || 0)
  if (!size) return ''
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
})
const description = computed(() => props.work.effect_description || props.work.rp_use_cases || props.work.summary || '')
const sanitizedWorkContent = computed(() => sanitizeRichHtml(props.work.content || ''))
const orderedTransmogSlots = computed(() => {
  return [...(props.work.transmog_slots || [])]
    .filter(slot => slot.role !== 'unused')
    .sort((a, b) => {
      const aRank = transmogSlotRank.get(a.slot) ?? 999
      const bRank = transmogSlotRank.get(b.slot) ?? 999
      if (aRank !== bRank) return aRank - bRank
      return (a.sort_order || 0) - (b.sort_order || 0)
    })
})

const richContentRef = ref<HTMLElement | null>(null)

async function hydrateRichContent() {
  await nextTick()
  sanitizeJumpLinks(richContentRef.value)
  hydrateJumpCards(richContentRef.value)
}

watch(() => props.work.content, () => {
  void hydrateRichContent()
}, { immediate: true })

function formatSlotLabel(slot: string) {
  return transmogSlotLabel.get(slot) || slot
}

async function copyMusicianCode() {
  if (!musicianCode.value) return
  try {
    await navigator.clipboard.writeText(musicianCode.value)
    copiedMusicianCode.value = true
    window.setTimeout(() => copiedMusicianCode.value = false, 1800)
  } catch {
    copiedMusicianCode.value = false
  }
}
</script>

<template>
  <div class="work-content" data-testid="work-content">
    <section id="rpdb-section-overview" class="editorial-section">
      <header class="section-heading">
        <span>{{ isHome ? '空间故事' : '作品介绍' }}</span>
        <h2>{{ isHome ? '空间故事与参观亮点' : '实际效果与 RP 用途' }}</h2>
      </header>
      <p v-if="description" class="lead">{{ description }}</p>
      <p v-if="work.rp_use_cases && work.rp_use_cases !== description" class="use-cases">
        <b>适用场景</b>
        {{ work.rp_use_cases }}
      </p>
      <div v-if="work.content" ref="richContentRef" class="rich-content" v-html="sanitizedWorkContent"></div>
      <p v-else class="empty-copy">作者尚未补充完整正文。</p>
      <div v-if="work.type === 'transmog' && transmogShareCode" class="inline-share-code" data-testid="inline-transmog-share-code">
        <span>
          <i class="ri-shirt-line"></i>
          <span>
            <b>幻化分享代码</b>
            <small>复制后可在游戏内导入这套幻化方案</small>
          </span>
        </span>
        <button type="button" data-testid="copy-transmog-share-code-inline" @click="$emit('copyTransmogShareCode')">
          <i :class="copiedTransmogShareCode ? 'ri-check-line' : 'ri-file-copy-line'"></i>
          {{ copiedTransmogShareCode ? '已复制' : '复制代码' }}
        </button>
      </div>
      <div v-if="isMusicianMIDI && (musicianCode || musicianMIDIURL)" id="rpdb-section-musician-midi" class="inline-share-code musician-midi-download" data-testid="musician-midi-download">
        <span>
          <i class="ri-file-music-line"></i>
          <span>
            <b>{{ musicianCode ? 'Musician 音乐代码' : musicianMIDIName }}</b>
            <small>{{ musicianCode ? '复制后粘贴到 Musician 的歌曲导入框' : `${musicianMIDISize || 'Standard MIDI'} · 原始 MIDI 文件` }}</small>
          </span>
        </span>
        <div class="musician-midi-actions">
          <button v-if="musicianCode" type="button" data-testid="copy-musician-code" @click="copyMusicianCode">
            <i :class="copiedMusicianCode ? 'ri-check-line' : 'ri-file-copy-line'"></i>{{ copiedMusicianCode ? '已复制' : '复制音乐代码' }}
          </button>
          <a v-if="musicianMIDIURL" :href="musicianMIDIURL" :download="musicianMIDIName" target="_blank" rel="noopener">
            <i class="ri-download-cloud-2-line"></i>下载 MIDI
          </a>
        </div>
      </div>
    </section>

    <section v-if="orderedTransmogSlots.length" id="rpdb-section-transmog" class="editorial-section">
      <header class="section-heading">
        <span>幻化部件</span>
        <h2>幻化部件与替代方案</h2>
      </header>
      <div class="slot-grid">
        <article v-for="slot in orderedTransmogSlots" :key="slot.id || `${slot.slot}-${slot.sort_order}`" data-testid="transmog-slot-card">
          <i class="ri-shirt-line"></i>
          <div>
            <b data-testid="transmog-slot-label">{{ formatSlotLabel(slot.slot) }}</b>
            <dl class="slot-fields">
              <div v-if="slot.name || slot.note">
                <dt>名称</dt>
                <dd>{{ slot.name || slot.note }}</dd>
              </div>
              <div v-if="slot.description">
                <dt>介绍</dt>
                <dd>{{ slot.description }}</dd>
              </div>
              <div v-if="slot.source">
                <dt>来源</dt>
                <dd>{{ slot.source }}</dd>
              </div>
              <div v-if="slot.wowhead_url">
                <dt>Wowhead</dt>
                <dd><a :href="slot.wowhead_url" target="_blank" rel="noopener">{{ slot.wowhead_url }}</a></dd>
              </div>
              <div v-if="slot.variant">
                <dt>替代</dt>
                <dd>{{ slot.variant }}</dd>
              </div>
            </dl>
            <p v-if="!slot.name && !slot.note && !slot.description && !slot.source && !slot.variant && !slot.wowhead_url">
              {{ slot.role === 'variant' ? '替代部件' : slot.role === 'optional' ? '可选部件' : '必选部件' }}
            </p>
          </div>
          <small>{{ slot.role || 'required' }}</small>
        </article>
      </div>
    </section>

    <RPDBGuideSection v-if="!isHome && !isMusicianMIDI" :steps="work.guide_steps || []" />

    <section v-if="isHome" id="rpdb-section-home" class="editorial-section home-profile">
      <header class="section-heading">
        <span>家宅资料</span>
        <h2>家宅资料与参观方式</h2>
      </header>
      <p v-if="homeDetails?.visit_notes" class="visit-notes">{{ homeDetails.visit_notes }}</p>
    </section>
  </div>
</template>

<style scoped>
.work-content{overflow:hidden;background:transparent}
.editorial-section{padding:28px 30px;border-top:1px solid color-mix(in srgb,var(--color-border) 72%,transparent)}
.editorial-section:first-child{border-top:0}
.section-heading{margin:0 0 14px}
.section-heading span{display:block;color:var(--color-accent);font-size:10px;font-weight:800;letter-spacing:.06em}
.section-heading h2{margin:5px 0 0;color:var(--color-text-main);font:700 20px/1.3 system-ui,'Microsoft YaHei',sans-serif}
.lead{margin:0;color:var(--color-text-main);font-size:15px;line-height:1.9}
.use-cases{margin:16px 0;padding:12px 14px;border-radius:12px;background:color-mix(in srgb,var(--color-accent) 8%,transparent);color:var(--color-text-secondary);line-height:1.75}
.use-cases b{margin-right:8px;color:var(--color-text-main)}
.rich-content{color:var(--color-text-main);font-size:14px;line-height:1.9}
.rich-content :deep(img){max-width:100%;height:auto;border-radius:10px}
.empty-copy{color:var(--color-text-secondary)}
.inline-share-code{display:flex;align-items:center;justify-content:space-between;gap:18px;margin-top:20px;padding:14px 16px;border:1px solid color-mix(in srgb,var(--color-accent) 42%,var(--color-border));border-radius:12px;background:color-mix(in srgb,var(--color-accent) 7%,var(--color-card-bg))}
.inline-share-code>span{display:flex;min-width:0;align-items:center;gap:11px}.inline-share-code>span>i{display:grid;width:36px;height:36px;flex:0 0 36px;place-items:center;border-radius:9px;background:var(--color-accent);color:#fff;font-size:18px}.inline-share-code>span>span{display:flex;min-width:0;flex-direction:column;gap:3px}.inline-share-code b{color:var(--color-text-main)}.inline-share-code small{color:var(--color-text-secondary);line-height:1.45}.inline-share-code button,.inline-share-code a{display:inline-flex;min-height:38px;flex:0 0 auto;align-items:center;justify-content:center;gap:6px;padding:0 13px;border:1px solid var(--color-accent);border-radius:9px;background:var(--color-accent);color:#fff;font-weight:800;text-decoration:none}.musician-midi-actions{display:flex;flex:0 0 auto;gap:7px}
.slot-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(210px,1fr));gap:8px}
.slot-grid article{display:grid;grid-template-columns:30px minmax(0,1fr) auto;gap:8px;align-items:start;padding:12px;border:1px solid color-mix(in srgb,var(--color-border) 72%,transparent);border-radius:12px;background:color-mix(in srgb,var(--color-card-bg) 84%,#fff 16%)}
.slot-grid i{color:var(--color-accent);font-size:20px}
.slot-grid b{color:var(--color-text-main)}
.slot-grid p{margin:5px 0 0;color:var(--color-text-secondary);font-size:12px}
.slot-grid small{color:var(--color-text-secondary)}
.slot-fields{display:grid;gap:5px;margin:7px 0 0}
.slot-fields div{display:grid;grid-template-columns:38px minmax(0,1fr);gap:8px}
.slot-fields dt{color:var(--color-text-secondary);font-size:11px}
.slot-fields dd{margin:0;color:var(--color-text-main);font-size:12px;line-height:1.45}
.slot-fields a{color:var(--link-color);overflow-wrap:anywhere;text-decoration:none}
.slot-fields a:hover{color:var(--link-hover);text-decoration:underline}
.home-profile dl{display:grid;grid-template-columns:1fr 1fr;gap:0 20px;margin:0;border-top:1px solid color-mix(in srgb,var(--color-border) 72%,transparent)}
.home-profile dl div{display:flex;justify-content:space-between;gap:12px;padding:11px 0;border-bottom:1px solid color-mix(in srgb,var(--color-border) 72%,transparent)}
.home-profile dt{color:var(--color-text-secondary)}
.home-profile dd{margin:0;color:var(--color-text-main);text-align:right}
.visit-notes{margin:16px 0 0;padding:13px;border-radius:12px;background:color-mix(in srgb,var(--color-card-bg) 84%,#fff 16%);color:var(--color-text-secondary);line-height:1.75}
@media(max-width:680px){.editorial-section{padding:22px 18px}.inline-share-code{align-items:stretch;flex-direction:column}.musician-midi-actions{flex-direction:column}.inline-share-code button,.inline-share-code a{width:100%}.home-profile dl{grid-template-columns:1fr}}
</style>
