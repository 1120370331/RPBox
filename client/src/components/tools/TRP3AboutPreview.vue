<script setup lang="ts">
import { computed } from 'vue'
import type { TRP3About } from '@/api/characterCard'
import { normalizeTRP3About, renderTRP3Markup } from '@/utils/trp3Markup'
import WowIcon from '@/components/WowIcon.vue'
const props = defineProps<{ about: TRP3About }>()
const normalized = computed(() => normalizeTRP3About(props.about))
const sections = computed(() => normalized.value.TE === 1 ? [{ ...normalized.value.T1, title: '' }] : normalized.value.TE === 2 ? normalized.value.T2.map(value => ({ ...value, title: '' })) : [
  { ...normalized.value.T3.PH, title: '外貌描写' }, { ...normalized.value.T3.PS, title: '性格特征' }, { ...normalized.value.T3.HI, title: '历史' },
])
</script>
<template>
  <div class="trp3-about-preview" :class="{ parchment: normalized.BK === 7 }">
    <section v-for="(section, index) in sections" :key="index" class="trp3-about-section" :class="{ parchment: section.BK === 7, 'trp3-about-block': normalized.TE === 2 }">
      <header v-if="section.IC || section.title"><WowIcon v-if="section.IC" :icon="section.IC" :size="48" /><h3 v-if="section.title">{{ section.title }}</h3></header>
      <div class="trp3-markup" v-html="renderTRP3Markup(section.TX)"></div>
    </section>
    <p v-if="sections.every(section => !section.TX)" class="empty">还没有填写关于内容。</p>
  </div>
</template>
<style scoped>
.trp3-about-block{display:flex;align-items:center;gap:20px;border:1px solid #65573d;margin-bottom:12px}.trp3-about-block:nth-child(even){flex-direction:row-reverse}.trp3-about-block header{flex-shrink:0;margin:0!important}.trp3-about-block .trp3-markup{flex:1;min-width:0}
.trp3-about-preview{background:#181716 url('../../assets/trp3/ui-frame-neutral-background.png') repeat;color:#efeadb;padding:24px;border:1px solid #75674b;border-radius:4px;min-height:220px;font-family:Georgia,'Noto Serif SC',serif}.trp3-about-section{padding:18px 12px;overflow-wrap:anywhere}.trp3-about-section+.trp3-about-section{border-top:1px solid #65573d}.trp3-about-section header{display:flex;gap:16px;align-items:center;margin-bottom:16px}.trp3-about-section h3{color:#ffd46b;margin:0;font-size:20px}.trp3-markup{font-size:14px;line-height:1.8;white-space:normal}.trp3-markup :deep(h1){font-size:26px}.trp3-markup :deep(h2){font-size:22px}.trp3-markup :deep(h3){font-size:18px}.trp3-markup :deep(p){margin:.7em 0}.trp3-markup :deep(img){max-width:100%;object-fit:contain}.trp3-markup :deep(.trp3-inline-icon){display:inline-block;vertical-align:middle;border-radius:2px}.trp3-markup :deep(a){color:#65df65;text-decoration:underline}.parchment{background:#d3b987 url('../../assets/trp3/ui-frame-parchment-background.png') repeat;color:#38281a}.empty{color:inherit;opacity:.6;text-align:center}
</style>
