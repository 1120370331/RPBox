<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login } from '@shared/api/auth'
import { useUserStore } from '@shared/stores/user'

const router = useRouter()
const route = useRoute()
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
    const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
      ? route.query.redirect
      : '/'
    router.replace(redirect)
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
      <div class="login-header">
        <div class="logo">RPBOX</div>
        <p class="subtitle">{{ $t('auth.login.subtitle') }}</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <input
            v-model="username"
            class="input"
            :placeholder="$t('auth.login.usernamePlaceholder')"
            autocomplete="username"
            required
          />
        </div>
        <div class="form-group">
          <input
            v-model="password"
            type="password"
            class="input"
            :placeholder="$t('auth.login.passwordPlaceholder')"
            autocomplete="current-password"
            required
          />
        </div>

        <div class="form-actions">
          <router-link to="/forgot-password" class="forgot-link">{{ $t('auth.login.forgotPassword') }}</router-link>
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="btn-login" :disabled="loading">
          {{ loading ? $t('auth.login.submitting') : $t('auth.login.submit') }}
        </button>
      </form>

      <div class="login-footer">
        <router-link to="/register">{{ $t('auth.login.noAccount') }}</router-link>
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
  padding: 24px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: var(--color-panel-bg);
  border-radius: var(--radius-lg);
  padding: 32px 24px;
  box-shadow: var(--shadow-md);
  opacity: 0;
  transform: translateY(20px);
  transition: opacity 0.4s ease, transform 0.4s ease;
}

.login-card.animate-in {
  opacity: 1;
  transform: translateY(0);
}

.login-header { text-align: center; margin-bottom: 28px; }
.logo { font-size: 28px; font-weight: 700; color: var(--color-primary); margin-bottom: 6px; }
.subtitle { font-size: 13px; color: var(--color-secondary); }

.login-form { display: flex; flex-direction: column; gap: 14px; }

.form-group .input {
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  font-size: 16px;
  background: var(--input-bg);
  color: var(--color-text-main);
}

.form-group .input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.form-actions { display: flex; justify-content: flex-end; margin-top: -6px; }
.forgot-link { font-size: 13px; color: var(--color-accent); }

.error-msg { color: var(--btn-danger-bg); font-size: 13px; text-align: center; }

.btn-login {
  width: 100%;
  padding: 14px;
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 16px;
  font-weight: 600;
  margin-top: 4px;
}

.btn-login:disabled { opacity: 0.6; }
.btn-login:active:not(:disabled) { transform: scale(0.98); }

.login-footer { text-align: center; margin-top: 20px; font-size: 14px; }
.login-footer a { color: var(--color-accent); text-decoration: none; }
</style>
