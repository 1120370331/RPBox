<script setup lang="ts">
import { computed, ref } from 'vue'
import type { RPDBGuideStep } from '@/api/rpdb'
import { formatTomTomCommand, hasTomTomCoordinates } from '@/utils/tomtom'

const props = withDefaults(defineProps<{
  steps?: RPDBGuideStep[]
}>(), {
  steps: () => [],
})

const copied = ref('')
const commandEntries = computed(() => {
  const available = props.steps
    .map((step, sourceIndex) => ({ step, sourceIndex }))
    .filter(({ step }) => hasTomTomCoordinates(step))
  return available.map(({ step, sourceIndex }, routeIndex) => ({
    sourceIndex,
    command: formatTomTomCommand(step, {
      sequence: routeIndex + 1,
      total: available.length,
    }),
  }))
})
const commands = computed(() => commandEntries.value.map(entry => entry.command))
const commandByStep = computed(() => new Map(commandEntries.value.map(entry => [entry.sourceIndex, entry.command])))

function commandForStep(index: number) {
  return commandByStep.value.get(index) || ''
}

async function copyText(value: string, key: string) {
  if (!value) return
  await navigator.clipboard?.writeText(value)
  copied.value = key
  window.setTimeout(() => {
    if (copied.value === key) copied.value = ''
  }, 1400)
}
</script>

<template>
  <section v-if="steps.length" id="rpdb-section-guide" class="guide-section" data-testid="guide-reading">
    <header class="guide-heading">
      <div class="section-heading">
        <span>获取攻略</span>
        <h2>路线与坐标步骤</h2>
      </div>
      <div v-if="commands.length" class="tomtom-bulk" data-testid="tomtom-bulk-panel">
        <span><i class="ri-route-line"></i>TomTom 多点路线</span>
        <code>/ttpaste</code>
        <button
          type="button"
          class="copy-all"
          data-testid="tomtom-copy-all"
          @click="copyText(commands.join('\n'), 'all')"
        >
          <i class="ri-file-copy-line"></i>
          {{ copied === 'all' ? '已复制' : `复制 ${commands.length} 个坐标` }}
        </button>
        <small>游戏内打开 /ttpaste 后批量粘贴。</small>
      </div>
    </header>

    <ol class="guide-steps">
      <li
        v-for="(step, index) in steps"
        :key="step.id || `${step.sort_order}-${index}`"
        data-testid="guide-step"
      >
        <div class="step-number">{{ step.sort_order || index + 1 }}</div>
        <div class="step-copy">
          <div class="step-title">
            <h3>{{ step.title || `步骤 ${index + 1}` }}</h3>
            <span v-if="step.zone"><i class="ri-map-pin-2-line"></i>{{ step.zone }}</span>
          </div>
          <p v-if="step.body">{{ step.body }}</p>
          <p v-if="step.prerequisite" class="prerequisite">
            <i class="ri-git-branch-line"></i>
            前置条件：{{ step.prerequisite }}
          </p>
        </div>
        <div class="step-tools">
          <template v-if="commandForStep(index)">
            <span>地图坐标</span>
            <code>{{ commandForStep(index) }}</code>
            <button type="button" @click="copyText(commandForStep(index), `step-${index}`)">
              <i class="ri-clipboard-line"></i>
              {{ copied === `step-${index}` ? '已复制' : '复制坐标' }}
            </button>
          </template>
          <span v-else class="no-coordinate">作者未填写坐标</span>
        </div>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.guide-section{padding:26px 30px;border-top:1px solid color-mix(in srgb,var(--color-border) 72%,transparent);background:transparent}
.guide-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:16px;padding-bottom:16px}
.section-heading{min-width:0}
.section-heading span{display:block;color:var(--color-accent);font-size:10px;font-weight:800;letter-spacing:.06em}
.section-heading h2{margin:5px 0 0;color:var(--color-text-main);font:700 20px/1.3 system-ui,'Microsoft YaHei',sans-serif}
.tomtom-bulk{display:grid;grid-template-columns:auto auto auto;align-items:center;gap:6px 9px;max-width:430px;padding:9px 10px;border:1px solid color-mix(in srgb,var(--color-accent) 34%,var(--color-border));border-radius:10px;background:color-mix(in srgb,var(--color-accent) 7%,var(--color-panel-bg));color:var(--color-text-main)}
.tomtom-bulk>span{display:inline-flex;align-items:center;gap:5px;font-size:11px;font-weight:800;letter-spacing:0}.tomtom-bulk>span i{color:var(--color-accent);font-size:15px}.tomtom-bulk>code{padding:4px 7px;border-radius:6px;background:var(--color-panel-bg);color:var(--color-accent);font:700 11px/1.2 Consolas,monospace}.tomtom-bulk>small{grid-column:1/-1;color:var(--color-text-secondary);font-size:10px;line-height:1.5}
.copy-all,.step-tools button{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-height:34px;padding:0 12px;border:1px solid var(--color-accent);border-radius:10px;background:var(--color-accent);color:#fff}
.copy-all:disabled{cursor:not-allowed;opacity:.45}
.guide-steps{display:grid;gap:10px;margin:0;padding:0;list-style:none}
.guide-steps li{display:grid;grid-template-columns:40px minmax(0,1fr) 220px;gap:14px;padding:14px;border:1px solid color-mix(in srgb,var(--color-border) 72%,transparent);border-radius:12px;background:color-mix(in srgb,var(--color-card-bg) 84%,#fff 16%)}
.step-number{display:grid;width:34px;height:34px;place-items:center;border-radius:50%;background:color-mix(in srgb,var(--color-accent) 8%,transparent);color:var(--color-accent);font-weight:800}
.step-copy{min-width:0}
.step-title{display:flex;align-items:center;justify-content:space-between;gap:12px}
.step-title h3{margin:0;color:var(--color-text-main);font-size:15px}
.step-title span{display:inline-flex;align-items:center;gap:4px;color:var(--color-text-secondary);font:inherit;font-size:11px;letter-spacing:0}
.step-copy p{margin:8px 0 0;color:var(--color-text-secondary);line-height:1.75}
.step-copy .prerequisite{display:flex;align-items:center;gap:6px;color:var(--color-text-main);font-size:12px}
.step-tools{display:flex;min-width:0;flex-direction:column;align-items:stretch;gap:7px;padding-left:13px;border-left:1px solid color-mix(in srgb,var(--color-border) 72%,transparent)}
.step-tools>span{color:var(--color-text-secondary);font-size:11px}
.step-tools code{overflow-wrap:anywhere;padding:8px;border-radius:9px;background:color-mix(in srgb,var(--color-accent) 8%,transparent);color:var(--color-accent);font:11px/1.5 Consolas,monospace}
.step-tools button{min-height:31px;background:transparent;color:var(--color-accent)}
.step-tools .no-coordinate{margin:auto 0;text-align:center}
@media(max-width:760px){.guide-section{padding:20px}.guide-heading{align-items:flex-start;flex-direction:column}.tomtom-bulk{width:100%;max-width:none;grid-template-columns:auto auto minmax(0,1fr)}.tomtom-bulk .copy-all{justify-self:end}.guide-steps li{grid-template-columns:38px 1fr}.step-tools{grid-column:2;padding:10px 0 0;border-left:0;border-top:1px dashed color-mix(in srgb,var(--color-border) 72%,transparent)}}
</style>
