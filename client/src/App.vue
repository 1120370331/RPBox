<script setup lang="ts">
import UpdateNotification from '@/components/UpdateNotification.vue'
import RDialog from '@/components/RDialog.vue'
</script>

<template>
  <router-view />
  <UpdateNotification />
  <RDialog />
</template>

<style>
/* ========== RPBox 设计系统 ========== */
:root {
  /* 主色板 */
  --color-primary: #4B3621;      /* 深咖啡 */
  --color-secondary: #804030;    /* 中棕色 */
  --color-accent: #B87333;       /* 铜橙色 */
  --color-background: #EED9C4;   /* 浅米色 */
  --color-highlight: #D2691E;    /* 巧克力色-标记用 */

  /* 文字颜色 */
  --text-light: #EED9C4;
  --text-dark: #4B3621;

  /* 圆角 */
  --radius-sm: 8px;
  --radius-md: 16px;
  --radius-lg: 20px;

  /* 字体 */
  font-family: 'Microsoft YaHei', sans-serif;
  font-weight: 500;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  background-color: var(--color-background);
  color: var(--text-dark);
  min-height: 100vh;
}

/* 通用按钮 */
.btn-primary {
  padding: 12px 24px;
  background: var(--color-accent);
  color: var(--color-primary);
  border: none;
  border-radius: var(--radius-sm);
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(184, 115, 51, 0.4);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

/* 通用输入框 */
.input {
  padding: 12px 16px;
  background: #fff;
  border: 2px solid rgba(75, 54, 33, 0.2);
  border-radius: var(--radius-sm);
  font-size: 14px;
  color: var(--text-dark);
  transition: border-color 0.3s;
}

.input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.input::placeholder {
  color: rgba(75, 54, 33, 0.5);
}

/* 玻璃效果 */
.glass {
  background: rgba(128, 64, 48, 0.4);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(128, 64, 48, 0.2);
}

/* 链接 */
a {
  color: var(--color-accent);
  text-decoration: none;
  transition: color 0.3s;
}

a:hover {
  color: var(--color-highlight);
}
</style>
