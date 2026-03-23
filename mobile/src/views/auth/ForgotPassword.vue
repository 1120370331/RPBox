<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { forgotPassword, resetPassword } from '@shared/api/auth'
import { useToastStore } from '@shared/stores/toast'

const { t } = useI18n()
const router = useRouter()
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
    countdown.value = 60
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0 && countdownTimer) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000) as unknown as number
  } catch (e: any) {
    toast.error(e.message || t('auth.forgotPassword.sendFailed'))
  } finally {
    sendingCode.value = false
  }
}

async function handleResetPassword() {
  if (!verificationCode.value) { toast.error(t('auth.forgotPassword.codeRequired')); return }
  if (!newPassword.value || newPassword.value.length < 6) { toast.error(t('auth.forgotPassword.newPasswordTooShort')); return }
  if (newPassword.value !== confirmPassword.value) { toast.error(t('auth.forgotPassword.passwordMismatch')); return }

  try {
    await resetPassword(email.value, verificationCode.value, newPassword.value)
    toast.success(t('auth.forgotPassword.resetSuccess'))
    router.push('/login')
  } catch (e: any) {
    toast.error(e.message || t('auth.forgotPassword.resetFailed'))
  }
}
</script>

<template>
  <div class="forgot-page">
    <div class="forgot-card">
      <div class="forgot-header">
        <button class="back-btn" @click="router.push('/login')">
          <i class="ri-arrow-left-line" />
        </button>
        <h1>{{ $t('auth.forgotPassword.title') }}</h1>
      </div>

      <div v-if="step === 'email'" class="step-content">
        <div class="form-group">
          <label>{{ $t('auth.forgotPassword.emailLabel') }}</label>
          <input
            v-model="email"
            type="email"
            :placeholder="$t('auth.forgotPassword.emailPlaceholder')"
            @keyup.enter="handleSendCode"
            required
          />
        </div>
        <button class="btn-primary" @click="handleSendCode" :disabled="sendingCode">
          {{ sendingCode ? $t('auth.forgotPassword.sending') : $t('auth.forgotPassword.sendCode') }}
        </button>
        <p class="hint">{{ $t('auth.forgotPassword.hint') }}</p>
      </div>

      <div v-else class="step-content">
        <div class="form-group">
          <label>{{ $t('auth.forgotPassword.emailDisplayLabel') }}</label>
          <div class="email-display">{{ email }}</div>
        </div>
        <div class="form-group">
          <label>{{ $t('auth.forgotPassword.codeLabel') }}</label>
          <div class="verification-group">
            <input v-model="verificationCode" :placeholder="$t('auth.forgotPassword.codePlaceholder')" maxlength="6" required />
            <button class="btn-resend" @click="handleSendCode" :disabled="countdown > 0">
              <span v-if="countdown > 0">{{ countdown }}s</span>
              <span v-else>{{ $t('auth.forgotPassword.resend') }}</span>
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>{{ $t('auth.forgotPassword.newPasswordLabel') }}</label>
          <input v-model="newPassword" type="password" :placeholder="$t('auth.forgotPassword.newPasswordPlaceholder')" required />
        </div>
        <div class="form-group">
          <label>{{ $t('auth.forgotPassword.confirmPasswordLabel') }}</label>
          <input v-model="confirmPassword" type="password" :placeholder="$t('auth.forgotPassword.confirmPasswordPlaceholder')" @keyup.enter="handleResetPassword" required />
        </div>
        <button class="btn-primary" @click="handleResetPassword">{{ $t('auth.forgotPassword.resetButton') }}</button>
        <button class="btn-secondary" @click="step = 'email'">
          <i class="ri-arrow-left-line" /> {{ $t('auth.forgotPassword.backToPrevious') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.forgot-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.forgot-card {
  width: 100%;
  max-width: 400px;
  background: var(--color-panel-bg);
  border-radius: var(--radius-lg);
  padding: 32px 24px;
  box-shadow: var(--shadow-md);
}

.forgot-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 28px;
}

.forgot-header h1 {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-primary);
}

.back-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  background: var(--color-primary-light);
  color: var(--color-primary);
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.step-content { display: flex; flex-direction: column; gap: 16px; }

.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { font-size: 13px; font-weight: 600; color: var(--color-text-secondary); }

.form-group input {
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  font-size: 16px;
  background: var(--input-bg);
  color: var(--color-primary);
}

.form-group input:focus { outline: none; border-color: var(--color-accent); }

.email-display {
  padding: 14px 16px;
  background: var(--color-primary-light);
  border-radius: var(--radius-sm);
  font-size: 14px;
  color: var(--color-primary);
}

.verification-group { display: flex; gap: 8px; }
.verification-group input { flex: 1; }

.btn-resend {
  padding: 14px 16px;
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
  color: var(--color-accent);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.btn-resend:disabled { opacity: 0.5; }

.btn-primary {
  width: 100%;
  padding: 14px;
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 16px;
  font-weight: 600;
}

.btn-primary:disabled { opacity: 0.6; }
.btn-primary:active:not(:disabled) { transform: scale(0.98); }

.btn-secondary {
  width: 100%;
  padding: 12px;
  background: var(--color-card-bg);
  color: var(--color-text-secondary);
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.hint { font-size: 12px; color: var(--color-text-secondary); text-align: center; }
</style>
