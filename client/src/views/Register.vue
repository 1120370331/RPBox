<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { register, sendVerificationCode } from '../api/auth'

const router = useRouter()
const { t } = useI18n()
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
const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 100)
})

const canSendCode = computed(() => {
  return email.value.includes('@') && !sendingCode.value && countdown.value === 0
})

let countdownTimer: number | null = null

async function handleSendCode() {
  if (!email.value || !email.value.includes('@')) {
    error.value = t('common.validation.invalidEmail')
    return
  }

  sendingCode.value = true
  error.value = ''

  try {
    const response = await sendVerificationCode(email.value)
    codeSent.value = true

    // 开始60秒倒计时
    countdown.value = 60
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0 && countdownTimer) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000) as unknown as number

    // 显示成功消息
    const successMsg = response.message || t('auth.register.codeSent')
    error.value = '' // 清除错误，用成功消息替代
    setTimeout(() => {
      if (error.value === '') {
        error.value = successMsg
        setTimeout(() => {
          if (error.value === successMsg) {
            error.value = ''
          }
        }, 3000)
      }
    }, 0)
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
    <div class="register-card" :class="{ 'animate-in': mounted }">
      <div class="register-header anim-item" style="--delay: 0">
        <div class="logo">RPBOX</div>
        <p class="subtitle">{{ t('auth.register.subtitle') }}</p>
      </div>

      <form class="register-form" @submit.prevent="handleRegister">
        <div class="form-group anim-item" style="--delay: 1">
          <input
            v-model="username"
            class="input"
            :placeholder="t('auth.register.usernamePlaceholder')"
            required
          />
        </div>
        <div class="form-group anim-item" style="--delay: 2">
          <input
            v-model="email"
            type="email"
            class="input"
            :placeholder="t('auth.register.emailPlaceholder')"
            required
          />
        </div>
        <div class="form-group verification-group anim-item" style="--delay: 3">
          <input
            v-model="verificationCode"
            class="input verification-input"
            :placeholder="t('auth.register.codePlaceholder')"
            maxlength="6"
            required
          />
          <button
            type="button"
            class="btn-send-code"
            @click="handleSendCode"
            :disabled="!canSendCode"
          >
            <span v-if="countdown > 0">{{ countdown }}s</span>
            <span v-else-if="sendingCode">{{ t('auth.register.sending') }}</span>
            <span v-else-if="codeSent">{{ t('auth.register.resend') }}</span>
            <span v-else>{{ t('auth.register.getCode') }}</span>
          </button>
        </div>
        <div class="form-group anim-item" style="--delay: 4">
          <input
            v-model="password"
            type="password"
            class="input"
            :placeholder="t('auth.register.passwordPlaceholder')"
            required
          />
        </div>
        <div class="form-group anim-item" style="--delay: 5">
          <input
            v-model="confirmPassword"
            type="password"
            class="input"
            :placeholder="t('auth.register.confirmPasswordPlaceholder')"
            required
          />
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="btn-primary register-btn anim-item" style="--delay: 6" :disabled="loading">
          {{ loading ? t('auth.register.submitting') : t('auth.register.submit') }}
        </button>
      </form>

      <div class="register-footer anim-item" style="--delay: 7">
        <router-link to="/login">{{ t('auth.register.hasAccount') }} {{ t('auth.register.login') }}</router-link>
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

.verification-group {
  display: flex;
  gap: 8px;
}

.verification-input {
  flex: 1;
}

.btn-send-code {
  padding: 12px 20px;
  border: 1px solid #B87333;
  border-radius: 8px;
  background: #fff;
  color: #B87333;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.3s;
}

.btn-send-code:hover:not(:disabled) {
  background: #B87333;
  color: #fff;
}

.btn-send-code:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
