<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getAddonLatest, getAddonDownloadUrl } from '@/api/addon'
import { gameFlavors, gameFlavorLabel } from '../../../../client/src/utils/gameInstallations'
const flavor = ref('_retail_'), version = ref(''), error = ref('')
async function load() { error.value = ''; try { version.value = (await getAddonLatest()).version } catch { error.value = '暂时无法获取插件版本，请重试。' } }
onMounted(load)
</script>
<template>
  <div class="plugin-guide"><h1>插件安装指引</h1><p>手机端可以编辑、保存云端人物卡。插件需要在安装魔兽世界的电脑上使用。</p><label>游戏版本<select v-model="flavor" aria-label="插件游戏版本"><option v-for="item in gameFlavors" :key="item.value" :value="item.value">{{ item.label }}</option></select></label><h2>{{ gameFlavorLabel(flavor) }}</h2><a href="https://www.curseforge.com/wow/addons/total-rp-3/files/all" target="_blank" rel="noopener">Total RP 3 下载页 <i class="ri-external-link-line" /></a><a href="https://www.curseforge.com/wow/addons/total-rp-3-extended/files/all" target="_blank" rel="noopener">TRP3 Extended 下载页 <i class="ri-external-link-line" /></a><a v-if="flavor === '_retail_' && version" :href="getAddonDownloadUrl(version)" target="_blank" rel="noopener">RPBox Addon {{ version }} · 正式服下载 <i class="ri-download-line" /></a><p v-if="flavor !== '_retail_'">RPBox 当前仅提供正式服安装包，请勿将正式服包安装到其他版本目录。第三方插件请在下载页核对适用版本。</p><p v-if="error" role="alert">{{ error }} <button @click="load">重试</button></p><ol><li>在电脑上打开 RPBox，添加对应游戏版本目录。</li><li>退出游戏后安装兼容版本的插件。</li><li>在人物卡备份中选择相同版本和子账号，将云端存档写回游戏。</li></ol><RouterLink to="/sync">查看云端人物卡备份 →</RouterLink></div>
</template>
<style scoped>
.plugin-guide{padding:20px 16px 32px;color:var(--color-text-main)}h1{font-size:24px;margin:0}h2{font-size:18px;margin:24px 0 12px}p,li{font-size:13px;line-height:1.8;color:var(--color-text-secondary)}label{display:grid;gap:8px;margin:20px 0;font-size:13px}select{min-height:46px;min-width:0;width:100%;padding:8px;font-size:16px;background:var(--input-bg);color:var(--color-text-main);border:1px solid var(--color-border);border-radius:8px}a{display:flex;justify-content:space-between;align-items:center;gap:8px;min-height:48px;margin:8px 0;padding:12px;border:1px solid var(--color-border);border-radius:10px;background:var(--color-panel-bg);color:var(--color-accent);text-decoration:none;overflow-wrap:anywhere}ol{padding-left:20px}li{margin:12px 0}button{min-height:44px}
</style>
