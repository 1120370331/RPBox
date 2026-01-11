<script setup lang="ts">
interface Props {
  percent: number
  size?: 'sm' | 'md' | 'lg'
  type?: 'line' | 'circle'
  status?: 'normal' | 'success' | 'warning' | 'danger'
  showText?: boolean
  strokeWidth?: number
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  type: 'line',
  status: 'normal',
  showText: true,
  strokeWidth: 6,
})

const circleSize = { sm: 60, md: 80, lg: 120 }
const radius = (circleSize[props.size] - props.strokeWidth) / 2
const circumference = 2 * Math.PI * radius
const offset = circumference * (1 - props.percent / 100)
</script>

<template>
  <div v-if="type === 'line'" class="r-progress" :class="`r-progress--${size}`">
    <div class="r-progress__track">
      <div
        class="r-progress__bar"
        :class="`r-progress--${status}`"
        :style="{ width: `${percent}%` }"
      />
    </div>
    <span v-if="showText" class="r-progress__text">{{ percent }}%</span>
  </div>

  <div v-else class="r-progress-circle" :style="{ width: `${circleSize[size]}px`, height: `${circleSize[size]}px` }">
    <svg :viewBox="`0 0 ${circleSize[size]} ${circleSize[size]}`">
      <circle class="r-progress-circle__track" :cx="circleSize[size]/2" :cy="circleSize[size]/2" :r="radius" :stroke-width="strokeWidth" />
      <circle class="r-progress-circle__bar" :class="`r-progress--${status}`" :cx="circleSize[size]/2" :cy="circleSize[size]/2" :r="radius" :stroke-width="strokeWidth" :stroke-dasharray="circumference" :stroke-dashoffset="offset" />
    </svg>
    <span v-if="showText" class="r-progress-circle__text">{{ percent }}%</span>
  </div>
</template>

<style scoped>
.r-progress { display: flex; align-items: center; gap: 10px; }
.r-progress__track { flex: 1; background: rgba(75,54,33,0.1); border-radius: 4px; overflow: hidden; }
.r-progress--sm .r-progress__track { height: 4px; }
.r-progress--md .r-progress__track { height: 8px; }
.r-progress--lg .r-progress__track { height: 12px; }
.r-progress__bar { height: 100%; transition: width 0.3s; border-radius: 4px; }
.r-progress__text { font-size: 13px; color: var(--color-secondary); min-width: 40px; }

.r-progress--normal { background: var(--color-accent); }
.r-progress--success { background: #2e7d32; }
.r-progress--warning { background: #e65100; }
.r-progress--danger { background: #c41e3a; }

.r-progress-circle { position: relative; }
.r-progress-circle svg { transform: rotate(-90deg); }
.r-progress-circle__track { fill: none; stroke: rgba(75,54,33,0.1); }
.r-progress-circle__bar { fill: none; stroke: var(--color-accent); transition: stroke-dashoffset 0.3s; }
.r-progress-circle__bar.r-progress--success { stroke: #2e7d32; }
.r-progress-circle__bar.r-progress--warning { stroke: #e65100; }
.r-progress-circle__bar.r-progress--danger { stroke: #c41e3a; }
.r-progress-circle__text { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; font-size: 14px; font-weight: 600; }
</style>
