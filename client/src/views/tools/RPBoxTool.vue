<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getAddonDownloadUrl, getAddonLatest } from '@/api/addon'
import { useUserStore } from '@/stores/user'
const userStore = useUserStore()
const characterArchivePath = computed(() => userStore.user ? `/user/${userStore.user.id}` : '/tools/trp3')
const desktop = typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
const version = ref('')
const downloadError = ref('')
async function loadVersion() {
  downloadError.value = ''
  try { version.value = (await getAddonLatest()).version } catch { downloadError.value = '暂时无法获取插件版本，请重试。' }
}
onMounted(() => { if (!desktop) void loadVersion() })
</script>

<template>
  <main class="rpbox-tool">
    <RouterLink class="back" to="/tools"><i class="ri-arrow-left-line" aria-hidden="true"></i> 工具箱</RouterLink>
    <header><span class="tool-mark"><i class="ri-box-3-line" aria-hidden="true"></i></span><div><h1>RPBox</h1><p>把角色留在档案里，把相遇留在故事里。</p></div></header>
    <section class="shortcuts" aria-label="RPBox 快捷入口">
      <RouterLink :to="characterArchivePath"><i class="ri-id-card-line" aria-hidden="true"></i><div><h2>RPBox 人物卡档案</h2><p>查看已有角色档案，并联动在线编辑器继续编辑。</p></div><i class="ri-arrow-right-line" aria-hidden="true"></i></RouterLink>
      <RouterLink to="/sync"><i class="ri-cloud-line" aria-hidden="true"></i><div><h2>人物卡备份</h2><p>备份与同步 TRP3 人物卡，找回每一次修改。</p></div><i class="ri-arrow-right-line" aria-hidden="true"></i></RouterLink>
      <RouterLink to="/archives"><i class="ri-book-open-line" aria-hidden="true"></i><div><h2>剧情归档</h2><p>导入聊天记录，整理、回顾与分享 RP 剧情。</p></div><i class="ri-arrow-right-line" aria-hidden="true"></i></RouterLink>
    </section>
    <section class="install-guide">
      <div class="guide-heading"><div><span>从游戏内开始</span><h2>安装 RPBox 配套插件</h2></div><RouterLink v-if="desktop" class="primary" to="/tools/rpbox/install">打开插件安装器 <i class="ri-arrow-right-line" aria-hidden="true"></i></RouterLink><a v-else-if="version" class="primary" :href="getAddonDownloadUrl(version)" download>下载 RPBox 插件 {{ version }} <i class="ri-download-2-line" aria-hidden="true"></i></a><button v-else-if="downloadError" type="button" @click="loadVersion">{{ downloadError }}</button><span v-else role="status">获取插件版本…</span></div>
      <ol><li><strong>选择你的魔兽世界版本</strong><p>找到游戏版本目录，例如 <code>World of Warcraft/_retail_</code>。经典服请选择对应的版本目录。</p></li><li><strong>安装并启用插件</strong><p>退出游戏后，将压缩包中的 <code>RPBox_Addon</code> 文件夹放入 <code>Interface/AddOns</code>。进入角色选择界面，在插件列表启用 RPBox Addon 和 Total RP 3。</p></li><li><strong>开始记录，然后同步</strong><p>进入游戏使用插件记录 RP 对话；退出游戏或执行 <code>/reload</code> 使数据写入磁盘，再通过人物卡备份和剧情归档入口导入。</p></li></ol>
      <p class="guide-note">{{ desktop ? '桌面版可以扫描游戏目录并一键安装、更新插件。' : '网页版可编辑云端档案；自动扫描游戏目录、安装插件和读取本地游戏数据需要 RPBox 桌面客户端。' }}</p>
      <RouterLink v-if="!desktop" class="installer-link" to="/tools/rpbox/install">查看桌面版安装流程 <i class="ri-arrow-right-up-line" aria-hidden="true"></i></RouterLink>
    </section>
    <RouterLink class="editor-shortcut" to="/tools/trp3"><i class="ri-id-card-line" aria-hidden="true"></i><span>想继续完善角色设定？打开在线人物卡编辑器</span><i class="ri-arrow-right-line" aria-hidden="true"></i></RouterLink>
  </main>
</template>

<style scoped>
.rpbox-tool{max-width:1120px;margin:auto;padding:24px 16px 60px;color:var(--color-text-main)}.back{display:inline-flex;gap:6px;align-items:center;color:var(--color-text-secondary);text-decoration:none;font-size:13px}.rpbox-tool>header{display:flex;align-items:center;gap:20px;margin:32px 0}.tool-mark{display:grid;place-items:center;width:76px;height:76px;border-radius:20px;background:var(--gradient-start);color:var(--gradient-text);font-size:38px}h1{margin:0 0 8px;font-size:34px}.rpbox-tool>header p{margin:0;color:var(--color-text-secondary);font-size:14px}.shortcuts{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.shortcuts>a{display:flex;align-items:center;gap:12px;padding:20px;border:1px solid var(--color-border);border-radius:16px;background:var(--color-panel-bg);color:inherit;text-decoration:none}.shortcuts>a>i:first-child{font-size:30px;color:var(--color-accent)}.shortcuts>a>i:last-child{margin-left:auto}.shortcuts h2{font-size:18px;margin:0 0 8px}.shortcuts p{margin:0;font-size:12px;line-height:1.7;color:var(--color-text-secondary)}.install-guide{margin-top:24px;padding:30px;border:1px solid var(--color-border);border-radius:18px;background:var(--color-panel-bg)}.guide-heading{display:flex;justify-content:space-between;align-items:center;gap:20px}.guide-heading span{font-size:11px;color:var(--color-accent)}.guide-heading h2{font-size:23px;margin:8px 0 0}.primary{display:inline-flex;align-items:center;gap:8px;padding:12px 18px;border-radius:9px;background:var(--btn-primary-bg);color:var(--btn-primary-text);font-size:12px;font-weight:600;text-decoration:none}.install-guide ol{counter-reset:steps;list-style:none;padding:0;margin:30px 0}.install-guide li{position:relative;padding:0 0 24px 46px;counter-increment:steps}.install-guide li::before{content:counter(steps);position:absolute;left:0;top:0;display:grid;place-items:center;width:28px;height:28px;border-radius:50%;border:1px solid var(--color-border);color:var(--color-accent);font-size:12px}.install-guide strong{font-size:14px}.install-guide li p{font-size:13px;color:var(--color-text-secondary);line-height:1.9;margin:8px 0 0}.install-guide code{color:var(--color-text-main);background:var(--color-card-bg);padding:2px 5px;border-radius:4px;overflow-wrap:anywhere}.guide-note{border-top:1px solid var(--color-border);padding-top:18px;font-size:12px;line-height:1.8;color:var(--color-text-secondary)}.installer-link{font-size:12px;color:var(--link-color)}.editor-shortcut{display:flex;align-items:center;gap:12px;margin-top:24px;padding:20px;border-radius:12px;background:var(--color-card-bg);color:var(--color-text-main);text-decoration:none;font-size:13px}.editor-shortcut>i:last-child{margin-left:auto}.rpbox-tool a:focus-visible{outline:3px solid var(--color-accent);outline-offset:3px}@media(max-width:900px){.shortcuts{grid-template-columns:1fr}.guide-heading{align-items:flex-start;flex-direction:column}.install-guide{padding:20px}}
</style>
