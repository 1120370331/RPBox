<script setup lang="ts">
import { computed, watch, nextTick, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
const route = useRoute(), router = useRouter()
const scroller = ref<HTMLElement | null>(null)
const title = computed(() => route.path.startsWith('/sync') ? '人物卡备份' : route.path.includes('/trp3') ? '在线人物卡编辑器' : route.path.endsWith('/install') ? '插件安装指引' : route.path.endsWith('/rpbox') ? 'RPBox' : '工具箱')
function back() {
  if (typeof router.options.history.state.back === 'string') router.back()
  else router.push(route.path === '/tools' ? '/profile' : route.path.startsWith('/sync') ? '/tools/rpbox' : '/tools')
}
watch(() => route.path, async (path, previous) => {
  // The editor owns scroll restoration when creating or switching a card.
  if (path.startsWith('/tools/trp3') && previous.startsWith('/tools/trp3')) return
  await nextTick(); scroller.value?.scrollTo({ top: 0 })
})
</script>
<template>
  <div class="mobile-tools">
    <header class="tools-topbar"><button type="button" aria-label="返回上一级" @click="back"><i class="ri-arrow-left-line" /></button><strong>{{ title }}</strong><RouterLink to="/tools" aria-label="工具箱首页"><i class="ri-apps-2-line" /></RouterLink></header>
    <main ref="scroller" class="main-content"><RouterView /></main>
  </div>
</template>
<style scoped>
.mobile-tools{height:var(--app-height,100dvh);display:flex;flex-direction:column;padding-top:var(--safe-top,env(safe-area-inset-top,0px));background:var(--color-background);overflow:hidden}.tools-topbar{flex-shrink:0;display:flex;align-items:center;gap:8px;padding:4px max(8px,env(safe-area-inset-right));border-bottom:1px solid var(--color-border);background:var(--color-panel-bg)}.tools-topbar strong{flex:1;min-width:0;font-size:16px;text-align:center;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.tools-topbar button,.tools-topbar a{display:grid;place-items:center;width:44px;height:44px;border:0;background:transparent;color:var(--color-text-main);font-size:22px;text-decoration:none}.main-content{flex:1;min-height:0;overflow-y:auto;overscroll-behavior-y:contain;padding-bottom:max(20px,env(safe-area-inset-bottom));-webkit-overflow-scrolling:touch;scroll-padding:16px}
</style>
