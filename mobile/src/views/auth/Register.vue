<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { register, sendVerificationCode } from '@shared/api/auth'

const { t } = useI18n()
const router = useRouter()
const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const verificationCode = ref('')
const error = ref('')
const loading = ref(false)
const sendingCode = ref(false)
const codeSent = ref(false)
const countdown = ref(0)

const canSendCode = computed(() => {
  return email.value.includes('@') && !sendingCode.value && countdown.value === 0
})

let countdownTimer: number | null = null

async function handleSendCode() {
  if (!email.value || !email.value.includes('@')) {
    error.value = t('auth.register.invalidEmail')
    return
  }
  sendingCode.value = true
  error.value = ''
  try {
    await sendVerificationCode(email.value)
    codeSent.value = true
    countdown.value = 60
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0 && countdownTimer) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000) as unknown as number
  } catch (e: any) {
    error.value = e.message || t('auth.register.sendCodeFailed')
  } finally {
    sendingCode.value = false
  }
}

async function handleRegister() {
  error.value = ''
  if (password.value !== confirmPassword.value) {
    error.value = t('auth.register.passwordMismatch')
    return
  }
  if (!verificationCode.value) {
    error.value = t('auth.register.codeRequired')
    return
  }
  loading.value = true
  try {
    await register(username.value, email.value, password.value, verificationCode.value)
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
    <div class="register-card">
      <div class="register-header">
        <div class="logo">RPBOX</div>
        <p class="subtitle">{{ $t('auth.register.subtitle') }}</p>
      </div>

      <form class="register-form" @submit.prevent="handleRegister">
        <div class="form-group">
          <input v-model="username" class="input" :placeholder="$t('auth.register.usernamePlaceholder')" autocomplete="username" required />
        </div>
        <div class="form-group">
          <input v-model="email" type="email" class="input" :placeholder="$t('auth.register.emailPlaceholder')" autocomplete="email" required />
        </div>
        <div class="form-group verification-group">
          <input v-model="verificationCode" class="input verification-input" :placeholder="$t('auth.register.codePlaceholder')" maxlength="6" required />
          <button type="button" class="btn-send-code" @click="handleSendCode" :disabled="!canSendCode">
            <span v-if="countdown > 0">{{ countdown }}s</span>
            <span v-else-if="sendingCode">{{ $t('auth.register.sending') }}</span>
            <span v-else-if="codeSent">{{ $t('auth.register.resend') }}</span>
            <span v-else>{{ $t('auth.register.sendCode') }}</span>
          </button>
        </div>
        <div class="form-group">
          <input v-model="password" type="password" class="input" :placeholder="$t('auth.register.passwordPlaceholder')" autocomplete="new-password" required />
        </div>
        <div class="form-group">
          <input v-model="confirmPassword" type="password" class="input" :placeholder="$t('auth.register.confirmPasswordPlaceholder')" autocomplete="new-password" required />
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="btn-register" :disabled="loading">
          {{ loading ? $t('auth.register.submitting') : $t('auth.register.submit') }}
        </button>
      </form>

      <div class="register-footer">
        <router-link to="/login">{{ $t('auth.register.hasAccount') }}</router-link>
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
  padding: 24px;
}

.register-card {
  width: 100%;
  max-width: 400px;
  background: var(--color-panel-bg);
  border-radius: var(--radius-lg);
  padding: 32px 24px;
  box-shadow: var(--shadow-md);
}

.register-header { text-align: center; margin-bottom: 24px; }
.logo { font-size: 28px; font-weight: 700; color: var(--color-primary); margin-bottom: 6px; }
.subtitle { font-size: 13px; color: var(--color-text-secondary); }

.register-form { display: flex; flex-direction: column; gap: 14px; }

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

.verification-group { display: flex; gap: 8px; }
.verification-input { flex: 1; }

.btn-send-code {
  padding: 14px 16px;
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
  color: var(--color-accent);
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
}

.btn-send-code:disabled { opacity: 0.5; }

.error-msg { color: var(--btn-danger-bg); font-size: 13px; text-align: center; }

.btn-register {
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

.btn-register:disabled { opacity: 0.6; }
.btn-register:active:not(:disabled) { transform: scale(0.98); }

.register-footer { text-align: center; margin-top: 20px; font-size: 14px; }
.register-footer a { color: var(--color-accent); text-decoration: none; }
</style>
