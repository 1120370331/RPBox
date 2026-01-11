<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../api/auth'

const router = useRouter()
const username = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleRegister() {
  error.value = ''
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
  <div class="register">
    <h2>注册 RPBox</h2>
    <form @submit.prevent="handleRegister">
      <input v-model="username" placeholder="用户名" required />
      <input v-model="email" type="email" placeholder="邮箱" required />
      <input v-model="password" type="password" placeholder="密码" required />
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit" :disabled="loading">
        {{ loading ? '注册中...' : '注册' }}
      </button>
    </form>
    <router-link to="/login">已有账号？登录</router-link>
  </div>
</template>

<style scoped>
.register { max-width: 320px; margin: 100px auto; padding: 2rem; }
form { display: flex; flex-direction: column; gap: 1rem; }
input { padding: 0.75rem; border: 1px solid #444; border-radius: 4px; background: #2a2a2a; color: #fff; }
button { padding: 0.75rem; background: #42b883; border: none; border-radius: 4px; color: #fff; cursor: pointer; }
button:disabled { opacity: 0.6; }
.error { color: #f66; }
</style>
