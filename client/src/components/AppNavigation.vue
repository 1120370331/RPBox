<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { advanceNavigationTrail, defaultNavigationTrail, navigationLabel, safeNavigationPath, type NavigationStep } from '@/utils/navigationTrail'

const route = useRoute(), router = useRouter(), user = useUserStore()
const trail = ref<NavigationStep[]>([])
const parent = computed(() => trail.value.at(-2))
let storageKey = ''
function knownPath(path: unknown): path is string { return safeNavigationPath(path) && router.resolve(path).matched.length > 0 }
watch(() => [route.fullPath, user.user?.id] as const, ([path, id]) => {
  const key = `rpbox.navigation.${id || 'guest'}`
  if (storageKey !== key) {
    storageKey = key
    trail.value = []
    try {
      const saved: unknown = JSON.parse(sessionStorage.getItem(key) || 'null')
      if (Array.isArray(saved) && saved.length <= 10 && saved.at(-1) === path && saved.every(knownPath)) {
        trail.value = saved.map(value => ({ path: value, label: navigationLabel(value) }))
      }
    } catch { /* A blocked or expired session store does not prevent navigation. */ }
  }
  trail.value = advanceNavigationTrail(trail.value, path).filter(step => knownPath(step.path))
  if (!trail.value.length) trail.value = defaultNavigationTrail(path).filter(step => knownPath(step.path))
  try { sessionStorage.setItem(key, JSON.stringify(trail.value.map(step => step.path))) } catch { /* Memory navigation remains available. */ }
}, { immediate: true })
</script>

<template>
  <div v-if="trail.length > 1" class="app-navigation">
    <RouterLink v-if="parent" class="navigation-up" :to="parent.path" replace :aria-label="`返回上一级：${parent.label}`"><i class="ri-arrow-left-line" aria-hidden="true"></i>返回上一级</RouterLink>
    <nav aria-label="页面层级"><ol><li v-for="(step, index) in trail" :key="step.path"><i v-if="index" class="ri-arrow-right-s-line" aria-hidden="true"></i><span v-if="index === trail.length - 1" aria-current="page">{{ step.label }}</span><RouterLink v-else :to="step.path" replace>{{ step.label }}</RouterLink></li></ol></nav>
  </div>
</template>

<style scoped>
.app-navigation{position:sticky;top:0;z-index:20;display:flex;align-items:center;gap:18px;flex-wrap:wrap;margin:0 0 12px;padding:10px 12px;border:1px solid var(--color-border);border-radius:10px;background:var(--color-panel-bg);font-size:12px;color:var(--color-text-secondary)}.navigation-up{display:inline-flex;align-items:center;gap:6px;flex-shrink:0;padding-right:16px;border-right:1px solid var(--color-border);font-weight:600}.app-navigation a{color:var(--color-text-secondary);text-decoration:none}.app-navigation a:hover{color:var(--color-accent)}nav{min-width:0}ol{display:flex;align-items:center;flex-wrap:wrap;gap:6px;list-style:none;margin:0;padding:0}li{display:flex;align-items:center;gap:6px}li>span{color:var(--color-text-main);font-weight:600}.app-navigation a:focus-visible{outline:2px solid var(--color-accent);outline-offset:4px}
</style>
