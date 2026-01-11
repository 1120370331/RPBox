<script setup lang="ts">
import { ref } from 'vue'

const profiles = ref<any[]>([])
</script>

<template>
  <div class="profiles">
    <h2>人物卡管理</h2>
    <router-link to="/">返回首页</router-link>
    <div v-if="profiles.length === 0" class="empty">
      暂无人物卡数据
    </div>
  </div>
</template>

<style scoped>
.profiles {
  padding: 2rem;
}
.empty {
  color: #888;
  margin-top: 2rem;
}
</style>
