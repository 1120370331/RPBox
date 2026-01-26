<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '@shared/api/auth'
import { useUserStore } from '@shared/stores/user'
import RButton from '@shared/components/RButton.vue'
import RInput from '@shared/components/RInput.vue'
import RCard from '@shared/components/RCard.vue'

const router = useRouter()
const userStore = useUserStore()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')

async function handleRegister() {
  if (!username.value || !password.value) {
    error.value = '请填写完整信息'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = '两次密码不一致'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const res = await register(username.value, password.value)
    userStore.setAuth(res.token, res.user)
    router.push('/')
  } catch (e: any) {
    error.value = e.message || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="register-page">
    <RCard class="register-card">
      <h1>注册 RPBox</h1>

      <form @submit.prevent="handleRegister" class="form">
        <RInput v-model="username" placeholder="用户名" />
        <RInput v-model="password" type="password" placeholder="密码" />
        <RInput v-model="confirmPassword" type="password" placeholder="确认密码" />

        <p v-if="error" class="error">{{ error }}</p>

        <RButton type="submit" variant="primary" :loading="loading" block>
          注册
        </RButton>
      </form>

      <p class="login-link">
        已有账号？<RouterLink to="/login">立即登录</RouterLink>
      </p>
    </RCard>
  </div>
</template>

<style scoped>
.register-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.register-card {
  width: 100%;
  max-width: 400px;
  padding: 40px;
}

.register-card h1 {
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

.login-link {
  text-align: center;
  margin-top: 24px;
  color: var(--text-secondary);
}

.login-link a {
  color: var(--primary-color);
}
</style>
