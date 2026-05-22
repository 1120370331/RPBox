<script setup lang="ts">
import { computed, nextTick, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { login, switchLogin, type LoginResponse } from '@shared/api/auth'
import { useUserStore, type AccountHistoryItem } from '@shared/stores/user'
import { resolveApiUrl } from '@/api/image'

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

function removeRecentAccount(id: number) {
  if (selectedAccountId.value === id) {
    useAnotherAccount()
  }
  userStore.removeAccountHistoryItem(id)
}

function nameStyle(account: AccountHistoryItem) {
  return {
    color: account.name_color || undefined,
    fontWeight: account.name_bold ? '700' : undefined,
  }
}

function getAccountHint(account: AccountHistoryItem) {
  return userStore.hasValidAccountSwitchSession(account.id)
    ? t('auth.login.passwordlessSwitchAvailable')
    : t('auth.login.passwordRequiredForSwitch')
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
      <div class="login-header">
        <div class="logo">RPBOX</div>
        <p class="subtitle">{{ $t('auth.login.subtitle') }}</p>
      </div>

      <section v-if="hasRecentAccounts" class="account-switcher" :aria-label="$t('auth.login.recentAccounts')">
        <div class="account-switcher-head">
          <span>{{ $t('auth.login.recentAccounts') }}</span>
          <button type="button" class="use-other-btn" @click="useAnotherAccount">
            {{ $t('auth.login.useAnotherAccount') }}
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
              <span class="account-avatar">
                <img v-if="account.avatar" :src="resolveApiUrl(account.avatar)" alt="">
                <span v-else>{{ account.username.charAt(0).toUpperCase() }}</span>
              </span>
              <span class="account-meta">
                <span class="account-name" :style="nameStyle(account)">
                  {{ account.username }}
                </span>
                <span class="account-hint">
                  {{ switchingAccountId === account.id ? $t('auth.login.passwordlessSwitching') : getAccountHint(account) }}
                </span>
              </span>
            </button>
            <button
              type="button"
              class="remove-account-btn"
              :title="$t('auth.login.removeRecentAccount')"
              :aria-label="$t('auth.login.removeRecentAccount')"
              :disabled="loading"
              @click="removeRecentAccount(account.id)"
            >
              <i class="ri-close-line" />
            </button>
          </div>
        </div>
      </section>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <input
            ref="usernameInputRef"
            v-model="username"
            class="input"
            :placeholder="$t('auth.login.usernamePlaceholder')"
            autocomplete="username"
            required
          />
        </div>
        <div class="form-group">
          <input
            ref="passwordInputRef"
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
  height: 100%;
  min-height: 100vh;
  min-height: var(--app-height, 100dvh);
  overflow-y: auto;
  padding: calc(var(--safe-top, 0px) + clamp(12px, 2.6vh, 22px)) 16px calc(clamp(16px, 3vh, 24px) + var(--safe-bottom, 0px));
}

.login-card {
  width: 100%;
  max-width: 420px;
  margin: max(8px, 3vh) auto;
  background: var(--color-panel-bg);
  border-radius: var(--radius-lg);
  padding: clamp(20px, 3.2vh, 30px) clamp(16px, 4.2vw, 24px);
  border: 1px solid rgba(75, 54, 33, 0.08);
  box-shadow: var(--shadow-md);
  opacity: 0;
  transform: translateY(20px);
  transition: opacity 0.4s ease, transform 0.4s ease;
}

.login-card.animate-in {
  opacity: 1;
  transform: translateY(0);
}

.login-header { text-align: center; margin-bottom: clamp(16px, 2.6vh, 26px); }
.logo { font-size: clamp(24px, 4.8vw, 30px); font-weight: 700; color: var(--color-primary); margin-bottom: 6px; }
.subtitle { font-size: 13px; color: var(--color-secondary); line-height: 1.45; }

.account-switcher {
  margin-bottom: clamp(14px, 2.2vh, 20px);
}

.account-switcher-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.use-other-btn {
  border: none;
  background: transparent;
  color: var(--color-accent);
  font-size: 12px;
  padding: 2px 0;
  white-space: nowrap;
}

.account-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: min(232px, 30vh);
  overflow-y: auto;
  padding-right: 2px;
  overscroll-behavior: contain;
}

.account-item {
  position: relative;
  min-height: 56px;
  border: 1px solid rgba(75, 54, 33, 0.1);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.72);
  transition: border-color 0.2s ease, background-color 0.2s ease, box-shadow 0.2s ease;
}

.account-item.active,
.account-item:active {
  border-color: var(--color-secondary);
  background: rgba(255, 244, 232, 0.94);
  box-shadow: 0 8px 18px rgba(44, 24, 16, 0.08);
}

.account-item.passwordless {
  border-color: var(--color-border-hover);
}

.account-select {
  width: 100%;
  min-height: 54px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 40px 8px 10px;
  border: none;
  background: transparent;
  color: inherit;
  text-align: left;
}

.account-select:disabled,
.remove-account-btn:disabled {
  opacity: 0.72;
}

.account-avatar {
  width: 38px;
  height: 38px;
  flex: 0 0 38px;
  border-radius: 50%;
  overflow: hidden;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--icon-bg);
  color: var(--icon-color);
  font-size: 16px;
  font-weight: 700;
}

.account-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.account-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.account-name {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-main);
  font-size: 14px;
  font-weight: 700;
}

.account-hint {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-secondary);
  font-size: 11px;
}

.remove-account-btn {
  position: absolute;
  right: 8px;
  top: 50%;
  width: 28px;
  height: 28px;
  transform: translateY(-50%);
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--color-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.remove-account-btn:active {
  background: rgba(75, 54, 33, 0.1);
  color: var(--color-primary);
}

.login-form { display: flex; flex-direction: column; gap: clamp(10px, 1.9vh, 14px); }

.form-group .input {
  width: 100%;
  padding: clamp(12px, 1.9vh, 14px) 14px;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  font-size: 15px;
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
  padding: clamp(12px, 1.9vh, 14px);
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 15px;
  font-weight: 600;
  margin-top: 4px;
}

.btn-login:disabled { opacity: 0.6; }
.btn-login:active:not(:disabled) { transform: scale(0.98); }

.login-footer { text-align: center; margin-top: 20px; font-size: 14px; }
.login-footer a { color: var(--color-accent); text-decoration: none; }

@media (max-height: 700px) {
  .login-page {
    padding-top: 12px;
  }

  .login-card {
    margin: 8px auto;
    border-radius: var(--radius-md);
  }

  .login-header {
    margin-bottom: 14px;
  }

  .login-form {
    gap: 9px;
  }

  .account-list {
    max-height: 176px;
  }

  .login-footer {
    margin-top: 14px;
  }
}
</style>
