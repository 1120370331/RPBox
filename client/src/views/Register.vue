<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../api/auth'

const router = useRouter()
const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)
const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 100)
})

async function handleRegister() {
  error.value = ''

  if (password.value !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }

  loading.value = true
  try {
    await register(username.value, email.value, password.value)
    router.push('/login')
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="register-page">
    <div class="register-card" :class="{ 'animate-in': mounted }">
      <div class="register-header anim-item" style="--delay: 0">
        <div class="logo">RPBOX</div>
        <p class="subtitle">创建你的冒险者账号</p>
      </div>

      <form class="register-form" @submit.prevent="handleRegister">
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
            v-model="email"
            type="email"
            class="input"
            placeholder="邮箱"
            required
          />
        </div>
        <div class="form-group anim-item" style="--delay: 3">
          <input
            v-model="password"
            type="password"
            class="input"
            placeholder="密码"
            required
          />
        </div>
        <div class="form-group anim-item" style="--delay: 4">
          <input
            v-model="confirmPassword"
            type="password"
            class="input"
            placeholder="确认密码"
            required
          />
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="btn-primary register-btn anim-item" style="--delay: 5" :disabled="loading">
          {{ loading ? '注册中...' : '注册' }}
        </button>
      </form>

      <div class="register-footer anim-item" style="--delay: 6">
        <router-link to="/login">已有账号？立即登录</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.register-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: #EED9C4;
}

.register-card {
  width: 100%;
  max-width: 400px;
  background: #fff;
  border-radius: 16px;
  padding: 40px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.1);
  opacity: 0;
  transform: scale(0.95);
  transition: opacity 0.4s ease, transform 0.4s ease;
}

.register-card.animate-in {
  opacity: 1;
  transform: scale(1);
}

.register-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  font-size: 32px;
  font-weight: 700;
  color: #804030;
  margin-bottom: 8px;
}

.subtitle {
  font-size: 14px;
  color: #8C7B70;
}

.register-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group .input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid #E8DCCF;
  border-radius: 8px;
  font-size: 15px;
  background: #FFFCF9;
  color: #2C1810;
  transition: border-color 0.3s, box-shadow 0.3s, transform 0.2s;
}

.form-group .input:focus {
  outline: none;
  border-color: #B87333;
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(184, 115, 51, 0.15);
}

.error-msg {
  color: #c41e3a;
  font-size: 13px;
  text-align: center;
}

.register-btn {
  width: 100%;
  margin-top: 8px;
  padding: 12px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.register-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(184, 115, 51, 0.3);
}

.register-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.register-footer {
  text-align: center;
  margin-top: 24px;
  font-size: 14px;
}

.register-footer a {
  color: #B87333;
  text-decoration: none;
}

.register-footer a:hover {
  text-decoration: underline;
}

/* 动画 */
.anim-item {
  opacity: 0;
  transform: translateY(30px);
}

.animate-in .anim-item {
  animation: slideUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(var(--delay) * 0.1s);
}

@keyframes slideUp {
  to { opacity: 1; transform: translateY(0); }
}
</style>
