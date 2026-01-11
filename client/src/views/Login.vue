<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api/auth'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 100)
})

async function handleLogin() {
  error.value = ''
  loading.value = true

  try {
    const res = await login(username.value, password.value)
    userStore.setAuth(res.token, res.user)
    router.push('/')
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card" :class="{ 'animate-in': mounted }">
      <div class="login-header anim-item" style="--delay: 0">
        <div class="logo">RPBOX</div>
        <p class="subtitle">魔兽世界 RP 玩家工具箱</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group anim-item" style="--delay: 1">
          <input
            v-model="username"
            class="input"
            placeholder="用户名"
            required
          />
        </div>
        <div class="form-group anim-item" style="--delay: 2">
          <input
            v-model="password"
            type="password"
            class="input"
            placeholder="密码"
            required
          />
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="btn-primary login-btn anim-item" style="--delay: 3" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>

      <div class="login-footer anim-item" style="--delay: 4">
        <router-link to="/register">没有账号？立即注册</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: #fff;
  border-radius: var(--radius-lg);
  padding: 40px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.1);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  font-size: 32px;
  font-weight: 700;
  color: var(--color-primary);
  margin-bottom: 8px;
}

.subtitle {
  font-size: 14px;
  color: var(--color-secondary);
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group .input {
  width: 100%;
}

.error-msg {
  color: #c41e3a;
  font-size: 13px;
  text-align: center;
}

.login-btn {
  width: 100%;
  margin-top: 8px;
}

.login-footer {
  text-align: center;
  margin-top: 24px;
  font-size: 14px;
}

/* 向上键入动画 */
.anim-item {
  opacity: 0;
  transform: translateY(30px);
}

.animate-in .anim-item {
  animation: slideUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(var(--delay) * 0.1s);
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 卡片入场动画 */
.login-card {
  opacity: 0;
  transform: scale(0.95);
  transition: opacity 0.4s ease, transform 0.4s ease;
}

.login-card.animate-in {
  opacity: 1;
  transform: scale(1);
}

/* 输入框聚焦动画 */
.input {
  transition: border-color 0.3s, box-shadow 0.3s, transform 0.2s;
}

.input:focus {
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(184, 115, 51, 0.15);
}

/* 按钮悬浮动画 */
.login-btn {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.login-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(184, 115, 51, 0.3);
}

.login-btn:active:not(:disabled) {
  transform: translateY(0) scale(0.98);
}
</style>
