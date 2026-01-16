<script setup lang="ts">
import { ref } from 'vue'

const activeSection = ref('addon')
</script>

<template>
  <div class="guide-page">
    <div class="guide-header">
      <h1>使用教程</h1>
      <p>快速了解 RPBox 的核心功能</p>
    </div>

    <div class="guide-nav">
      <button
        :class="{ active: activeSection === 'addon' }"
        @click="activeSection = 'addon'"
      >
        <i class="ri-plug-line"></i> 插件安装
      </button>
      <button
        :class="{ active: activeSection === 'archive' }"
        @click="activeSection = 'archive'"
      >
        <i class="ri-archive-line"></i> 剧情归档
      </button>
      <button
        :class="{ active: activeSection === 'sync' }"
        @click="activeSection = 'sync'"
      >
        <i class="ri-refresh-line"></i> 人物卡同步
      </button>
    </div>

    <div class="guide-content">
      <!-- 插件安装教程 -->
      <div v-if="activeSection === 'addon'" class="section">
        <h2><i class="ri-plug-line"></i> RPBox Addon 插件安装</h2>

        <div class="step">
          <div class="step-number">1</div>
          <div class="step-content">
            <h3>下载插件</h3>
            <p>进入「剧情记录」页面，如果未安装插件会自动提示下载。</p>
            <p>点击「安装插件」按钮即可自动下载并安装。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">2</div>
          <div class="step-content">
            <h3>选择游戏版本</h3>
            <p>选择你的魔兽世界版本（正式服 / 怀旧服 / 经典旧世）。</p>
            <p>插件会自动安装到对应的 AddOns 目录。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">3</div>
          <div class="step-content">
            <h3>启用插件</h3>
            <p>进入游戏后，在角色选择界面点击「插件」按钮，确保 RPBox 插件已启用。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">4</div>
          <div class="step-content">
            <h3>查看与采集</h3>
            <p>插件会自动采集 RP 频道的聊天记录，包括 /say、/emote、/yell 等。</p>
            <p>在游戏中输入 <code>/rpbox</code> 可以打开记录查看面板。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">5</div>
          <div class="step-content">
            <h3>导出数据</h3>
            <p>面板右下角有「导出」按钮，点击即可导出数据。</p>
            <p>关闭游戏时也会自动导出。导出后回到 RPBox 即可进行归档。</p>
          </div>
        </div>

        <div class="tip-box">
          <i class="ri-lightbulb-line"></i>
          <div>
            <strong>提示：</strong>插件需要配合 TotalRP3 使用，可以自动获取角色的 RP 名称和信息。
          </div>
        </div>
      </div>

      <!-- 剧情归档教程 -->
      <div v-if="activeSection === 'archive'" class="section">
        <h2><i class="ri-archive-line"></i> 剧情归档功能</h2>

        <div class="step">
          <div class="step-number">1</div>
          <div class="step-content">
            <h3>查看待归档池</h3>
            <p>在「剧情记录」页面，切换到「待归档池」标签。</p>
            <p>这里显示插件采集到的所有聊天记录。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">2</div>
          <div class="step-content">
            <h3>选择记录</h3>
            <p>勾选你想要归档的对话记录。</p>
            <p>可以按日期分组选择，也可以单独选择某条记录。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">3</div>
          <div class="step-content">
            <h3>标记角色类型</h3>
            <p>点击记录可以标记为：玩家(P)、NPC(N)、旁白(B)。</p>
            <p>NPC 消息可以设置 NPC 名称和说话方式。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">4</div>
          <div class="step-content">
            <h3>创建剧情</h3>
            <p>点击「归档选中」按钮，输入剧情标题和描述。</p>
            <p>可以添加标签方便后续筛选。</p>
          </div>
        </div>
      </div>

      <!-- 人物卡同步教程 -->
      <div v-if="activeSection === 'sync'" class="section">
        <h2><i class="ri-refresh-line"></i> 人物卡云同步</h2>

        <div class="step">
          <div class="step-number">1</div>
          <div class="step-content">
            <h3>设置 WoW 路径</h3>
            <p>在「设置」页面中，设置你的魔兽世界安装路径。</p>
            <p>RPBox 会自动扫描 TRP3 的人物卡数据。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">2</div>
          <div class="step-content">
            <h3>上传人物卡</h3>
            <p>在「人物卡同步」页面，选择要同步的人物卡。</p>
            <p>点击「上传」将人物卡备份到云端。</p>
          </div>
        </div>

        <div class="step">
          <div class="step-number">3</div>
          <div class="step-content">
            <h3>跨设备同步</h3>
            <p>在其他设备登录同一账号，即可下载云端的人物卡。</p>
            <p>支持版本管理，可以回滚到历史版本。</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.guide-page {
  max-width: 800px;
  margin: 0 auto;
}

.guide-header {
  text-align: center;
  margin-bottom: 32px;
}

.guide-header h1 {
  font-size: 28px;
  color: #4B3621;
  margin: 0 0 8px 0;
}

.guide-header p {
  font-size: 14px;
  color: #856a52;
  margin: 0;
}

.guide-nav {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  background: #f5f0eb;
  padding: 6px;
  border-radius: 12px;
}

.guide-nav button {
  flex: 1;
  padding: 12px 16px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 14px;
  color: #856a52;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.2s;
}

.guide-nav button:hover {
  color: #4B3621;
}

.guide-nav button.active {
  background: #fff;
  color: #4B3621;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}

.guide-content {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.08);
}

.section h2 {
  font-size: 20px;
  color: #4B3621;
  margin: 0 0 24px 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.section h2 i {
  color: #B87333;
}

.step {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
}

.step-number {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #B87333, #a06028);
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  flex-shrink: 0;
}

.step-content {
  flex: 1;
}

.step-content h3 {
  font-size: 16px;
  color: #4B3621;
  margin: 0 0 8px 0;
}

.step-content p {
  font-size: 14px;
  color: #665242;
  line-height: 1.6;
  margin: 0 0 4px 0;
}

.tip-box {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: rgba(184, 115, 51, 0.1);
  border-radius: 8px;
  margin-top: 24px;
}

.tip-box i {
  font-size: 20px;
  color: #B87333;
}

.tip-box div {
  font-size: 14px;
  color: #665242;
  line-height: 1.6;
}

.tip-box strong {
  color: #4B3621;
}

code {
  background: rgba(184, 115, 51, 0.15);
  color: #804030;
  padding: 2px 8px;
  border-radius: 4px;
  font-family: 'Consolas', monospace;
  font-size: 13px;
}
</style>
