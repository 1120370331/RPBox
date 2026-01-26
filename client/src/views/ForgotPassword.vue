<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { forgotPassword, resetPassword } from '../api/auth'
import { useToastStore } from '../stores/toast'

const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()

const step = ref<'email' | 'reset'>('email')
const email = ref('')
const verificationCode = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const countdown = ref(0)
const sendingCode = ref(false)

let countdownTimer: number | null = null

async function handleSendCode() {
  if (!email.value) {
    toast.error(t('auth.forgotPassword.emailRequired'))
    return
  }

  sendingCode.value = true
  try {
    await forgotPassword(email.value)
    toast.success(t('auth.forgotPassword.codeSent'))
    step.value = 'reset'

    // 开始60秒倒计时
    countdown.value = 60
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0 && countdownTimer) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000) as unknown as number
  } catch (error: any) {
    console.error('找回密码错误:', error)
    toast.error(error.message || t('auth.forgotPassword.sendFailed'))
  } finally {
    sendingCode.value = false
  }
}

async function handleResetPassword() {
  if (!verificationCode.value) {
    toast.error(t('auth.forgotPassword.codeRequired'))
    return
  }
  if (!newPassword.value) {
    toast.error(t('auth.forgotPassword.newPasswordRequired'))
    return
  }
  if (newPassword.value.length < 6) {
    toast.error(t('auth.forgotPassword.passwordTooShort'))
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    toast.error(t('auth.register.passwordMismatch'))
    return
  }

  try {
    await resetPassword(email.value, verificationCode.value, newPassword.value)
    toast.success(t('auth.forgotPassword.resetSuccess'))
    router.push('/login')
  } catch (error: any) {
    toast.error(error.message || t('auth.forgotPassword.resetFailed'))
  }
}

function goBack() {
  router.push('/login')
}
</script>

<template>
  <div class="forgot-password-page">
    <div class="bg-pattern"></div>
    <div class="forgot-password-container">
      <div class="header">
        <h1>{{ t('auth.forgotPassword.title') }}</h1>
        <button class="back-btn" @click="goBack">
          <i class="ri-arrow-left-line"></i>
          {{ t('auth.forgotPassword.backToLogin') }}
        </button>
      </div>

      <!-- 步骤1：输入邮箱 -->
      <div v-if="step === 'email'" class="step-email">
        <div class="form-group">
          <label>{{ t('auth.forgotPassword.emailLabel') }}</label>
          <input
            v-model="email"
            type="email"
            :placeholder="t('auth.forgotPassword.emailPlaceholder')"
            @keyup.enter="handleSendCode"
            required
          />
        </div>

        <button class="btn-primary" @click="handleSendCode" :disabled="sendingCode">
          {{ sendingCode ? t('auth.register.sending') : t('auth.forgotPassword.sendCode') }}
        </button>

        <p class="hint">{{ t('auth.forgotPassword.hint') }}</p>
      </div>

      <!-- 步骤2：重置密码 -->
      <div v-else class="step-reset">
        <div class="form-group">
          <label>{{ t('auth.forgotPassword.emailLabel') }}</label>
          <div class="email-display">{{ email }}</div>
        </div>

        <div class="form-group">
          <label>{{ t('auth.forgotPassword.codeLabel') }}</label>
          <div class="verification-group">
            <input
              v-model="verificationCode"
              :placeholder="t('auth.forgotPassword.codePlaceholder')"
              maxlength="6"
              required
            />
            <button
              class="btn-resend"
              @click="handleSendCode"
              :disabled="countdown > 0"
            >
              <span v-if="countdown > 0">{{ countdown }}s</span>
              <span v-else>{{ t('auth.register.resend') }}</span>
            </button>
          </div>
        </div>

        <div class="form-group">
          <label>{{ t('auth.forgotPassword.newPassword') }}</label>
          <input
            v-model="newPassword"
            type="password"
            :placeholder="t('auth.forgotPassword.newPasswordPlaceholder')"
            required
          />
        </div>

        <div class="form-group">
          <label>{{ t('auth.forgotPassword.confirmPassword') }}</label>
          <input
            v-model="confirmPassword"
            type="password"
            :placeholder="t('auth.forgotPassword.confirmPasswordPlaceholder')"
            @keyup.enter="handleResetPassword"
            required
          />
        </div>

        <button class="btn-primary" @click="handleResetPassword">
          {{ t('auth.forgotPassword.resetButton') }}
        </button>

        <button class="btn-secondary" @click="step = 'email'">
          <i class="ri-arrow-left-line"></i>
          {{ t('auth.forgotPassword.backToPrevious') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.forgot-password-page {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #FBF5EF 0%, #F2E6D8 100%);
  padding: 24px;
}

.bg-pattern {
  position: fixed;
  inset: 0;
  pointer-events: none;
  opacity: 0.3;
  background-image: radial-gradient(#D4A373 1px, transparent 1px);
  background-size: 24px 24px;
}

.forgot-password-container {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 420px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(75, 54, 33, 0.12);
  border: 1px solid #E8DCC8;
  padding: 32px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.header h1 {
  font-size: 24px;
  font-weight: 700;
  color: #4B3621;
  margin: 0;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  color: #B87333;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.2s;
  padding: 4px 8px;
}

.back-btn:hover {
  color: #4B3621;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: #8C7B70;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.form-group input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid #E8DCC8;
  border-radius: 6px;
  font-size: 14px;
  background: #FBF5EF;
  color: #4B3621;
  transition: all 0.2s;
  box-sizing: border-box;
}

.form-group input:focus {
  outline: none;
  border-color: #B87333;
  background: #fff;
}

.email-display {
  padding: 12px 16px;
  background: #F2E6D8;
  border-radius: 6px;
  font-size: 14px;
  color: #4B3621;
  font-weight: 500;
}

.verification-group {
  display: flex;
  gap: 8px;
}

.verification-group input {
  flex: 1;
}

.btn-resend {
  padding: 12px 16px;
  border: 1px solid #D4A373;
  border-radius: 6px;
  background: #fff;
  color: #B87333;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}

.btn-resend:hover:not(:disabled) {
  background: #B87333;
  color: #fff;
}

.btn-resend:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  width: 100%;
  padding: 14px;
  background: linear-gradient(135deg, #D4A373, #B87333);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 4px 12px rgba(212, 163, 115, 0.3);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(212, 163, 115, 0.4);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.btn-secondary {
  width: 100%;
  padding: 12px;
  background: #FBF5EF;
  color: #8C7B70;
  border: 1px solid #E8DCC8;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  margin-top: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.btn-secondary:hover {
  background: #F2E6D8;
  border-color: #D4A373;
}

.hint {
  margin-top: 16px;
  font-size: 12px;
  color: #8C7B70;
  text-align: center;
  line-height: 1.5;
}
</style>
