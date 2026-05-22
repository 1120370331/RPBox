<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { login, switchLogin, type LoginResponse } from '../api/auth'
import { useUserStore } from '../stores/user'
import type { AccountHistoryItem } from '../stores/user'
import { buildNameStyle } from '@/utils/userNameStyle'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const userStore = useUserStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const mounted = ref(false)
const selectedAccountId = ref<number | null>(null)
const switchingAccountId = ref<number | null>(null)
const usernameInputRef = ref<HTMLInputElement | null>(null)
const passwordInputRef = ref<HTMLInputElement | null>(null)

const hasRecentAccounts = computed(() => userStore.accountHistory.length > 0)

onMounted(() => {
  setTimeout(() => mounted.value = true, 100)
})

function finishLogin(res: LoginResponse) {
  userStore.setAuth(res.token, res.user, {
    switchToken: res.switch_token,
    switchTokenExpiresAt: res.switch_token_expires_at,
  })
  const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
    ? route.query.redirect
    : '/'
  router.replace(redirect)
}

async function selectAccount(account: AccountHistoryItem) {
  selectedAccountId.value = account.id
  username.value = account.username
  password.value = ''
  error.value = ''

  const switchToken = userStore.getAccountSwitchToken(account.id)
  if (switchToken) {
    switchingAccountId.value = account.id
    loading.value = true
    try {
      const res = await switchLogin(switchToken)
      finishLogin(res)
      return
    } catch (e: any) {
      userStore.clearAccountSwitchSession(account.id)
      error.value = e.message
    } finally {
      loading.value = false
      switchingAccountId.value = null
    }
  }

  nextTick(() => passwordInputRef.value?.focus())
}

function useAnotherAccount() {
  selectedAccountId.value = null
  username.value = ''
  password.value = ''
  error.value = ''
  nextTick(() => usernameInputRef.value?.focus())
}

function getAccountHint(account: AccountHistoryItem) {
  return userStore.hasValidAccountSwitchSession(account.id)
    ? t('auth.login.passwordlessSwitchAvailable')
    : t('auth.login.passwordRequiredForSwitch')
}

function removeRecentAccount(id: number) {
  if (selectedAccountId.value === id) {
    useAnotherAccount()
  }
  userStore.removeAccountHistoryItem(id)
}

async function handleLogin() {
  error.value = ''
  loading.value = true

  try {
    const res = await login(username.value, password.value)
    finishLogin(res)
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
        <p class="subtitle">{{ t('auth.login.subtitle') }}</p>
      </div>

      <div v-if="hasRecentAccounts" class="account-switcher anim-item" style="--delay: 0.5">
        <div class="account-switcher-header">
          <span>{{ t('auth.login.recentAccounts') }}</span>
          <button type="button" class="use-other-btn" @click="useAnotherAccount">
            {{ t('auth.login.useAnotherAccount') }}
          </button>
        </div>
        <div class="account-list">
          <div
            v-for="account in userStore.accountHistory"
            :key="account.id"
            class="account-item"
            :class="{
              active: selectedAccountId === account.id || username === account.username,
              passwordless: userStore.hasValidAccountSwitchSession(account.id),
            }"
          >
            <button type="button" class="account-select" :disabled="loading" @click="selectAccount(account)">
              <div class="account-avatar">
                <img v-if="account.avatar" :src="account.avatar" alt="头像" />
                <span v-else>{{ account.username.charAt(0).toUpperCase() }}</span>
              </div>
              <div class="account-meta">
                <span class="account-name" :style="buildNameStyle(account.name_color, account.name_bold)">
                  {{ account.username }}
                </span>
                <span class="account-hint">
                  {{ switchingAccountId === account.id ? t('auth.login.passwordlessSwitching') : getAccountHint(account) }}
                </span>
              </div>
            </button>
            <button
              type="button"
              class="remove-account-btn"
              :title="t('auth.login.removeRecentAccount')"
              :aria-label="t('auth.login.removeRecentAccount')"
              :disabled="loading"
              @click="removeRecentAccount(account.id)"
            >
              <i class="ri-close-line"></i>
            </button>
          </div>
        </div>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group anim-item" style="--delay: 1">
          <input
            ref="usernameInputRef"
            v-model="username"
            class="input"
            :placeholder="t('auth.login.usernamePlaceholder')"
            required
          />
        </div>
        <div class="form-group anim-item" style="--delay: 2">
          <input
            ref="passwordInputRef"
            v-model="password"
            type="password"
            class="input"
            :placeholder="t('auth.login.passwordPlaceholder')"
            required
          />
        </div>

        <div class="form-actions anim-item" style="--delay: 2.5">
          <router-link to="/forgot-password" class="forgot-password-link">{{ t('auth.login.forgotPassword') }}</router-link>
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="btn-primary login-btn anim-item" style="--delay: 3" :disabled="loading">
          {{ loading ? t('auth.login.submitting') : t('auth.login.submit') }}
        </button>
      </form>

      <div class="login-footer anim-item" style="--delay: 4">
        <router-link to="/register">{{ t('auth.login.noAccount') }} {{ t('auth.login.register') }}</router-link>
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

.account-switcher {
  margin-bottom: 22px;
}

.account-switcher-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  color: #7A5C46;
  font-size: 13px;
  font-weight: 600;
}

.use-other-btn {
  border: 0;
  background: transparent;
  color: var(--color-primary);
  font-size: 12px;
  cursor: pointer;
  padding: 2px 0;
  white-space: nowrap;
}

.use-other-btn:hover {
  color: #4B3621;
  text-decoration: underline;
}

.account-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.account-item {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 56px;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  background: #FFF9F2;
  transition: border-color 0.2s, background-color 0.2s, box-shadow 0.2s;
}

.account-item:hover,
.account-item.active {
  border-color: #B87333;
  background: #FFF4E8;
  box-shadow: 0 6px 18px rgba(75, 54, 33, 0.08);
}

.account-item.passwordless {
  border-color: var(--color-border-hover);
}

.account-select {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 54px;
  padding: 8px 40px 8px 10px;
  border: 0;
  background: transparent;
  cursor: pointer;
  text-align: left;
}

.account-select:disabled,
.remove-account-btn:disabled {
  cursor: wait;
  opacity: 0.72;
}

.account-avatar {
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  border-radius: 50%;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at 30% 24%, rgba(255, 255, 255, 0.72), transparent 34%),
    linear-gradient(135deg, var(--gradient-start, #D4A373), var(--gradient-end, #8C7B70));
  color: #fff;
  font-weight: 700;
}

.account-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.account-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.account-name {
  font-size: 14px;
  font-weight: 700;
  color: #3B2418;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 230px;
}

.account-hint {
  font-size: 12px;
  color: #8C7B70;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.remove-account-btn {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: #A68A79;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.2s, color 0.2s;
}

.remove-account-btn:hover {
  background: rgba(184, 115, 51, 0.12);
  color: #7A3E1D;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group .input {
  width: 100%;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: -8px;
}

.forgot-password-link {
  font-size: 13px;
  color: var(--color-primary);
  text-decoration: none;
  transition: color 0.2s;
}

.forgot-password-link:hover {
  color: #4B3621;
  text-decoration: underline;
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

@media (max-width: 480px) {
  .login-page {
    align-items: flex-start;
    justify-content: flex-start;
    padding: 24px 16px;
  }

  .login-card {
    padding: 28px 22px;
  }

  .account-name {
    max-width: 180px;
  }
}
</style>
