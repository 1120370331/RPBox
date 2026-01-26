<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { login } from '@shared/api/auth'
import { useUserStore } from '@shared/stores/user'
import RButton from '@shared/components/RButton.vue'
import RInput from '@shared/components/RInput.vue'
import RCard from '@shared/components/RCard.vue'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const res = await login(username.value, password.value)
    userStore.setAuth(res.token, res.user)

    const redirect = route.query.redirect as string
    router.push(redirect || '/')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <RCard class="login-card">
      <h1>登录 RPBox</h1>

      <form @submit.prevent="handleLogin" class="form">
        <RInput
          v-model="username"
          placeholder="用户名"
          autocomplete="username"
        />
        <RInput
          v-model="password"
          type="password"
          placeholder="密码"
          autocomplete="current-password"
        />

        <p v-if="error" class="error">{{ error }}</p>

        <RButton
          type="submit"
          variant="primary"
          :loading="loading"
          block
        >
          登录
        </RButton>
      </form>

      <p class="register-link">
        还没有账号？
        <RouterLink to="/register">立即注册</RouterLink>
      </p>
    </RCard>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  padding: 40px;
}

.login-card h1 {
  text-align: center;
  margin-bottom: 32px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.error {
  color: var(--danger-color);
  font-size: 14px;
}

.register-link {
  text-align: center;
  margin-top: 24px;
  color: var(--text-secondary);
}

.register-link a {
  color: var(--primary-color);
}
</style>
