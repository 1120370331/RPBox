<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import { searchUsers, type UserMentionItem } from '@/api/user'
import { createUserBlock, deleteUserBlock, listUserBlocks, type UserBlockItem } from '@/api/safety'
import { resolveApiUrl } from '@/api/item'
import { buildNameStyle } from '@/utils/userNameStyle'

const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const blockedUsers = ref<UserBlockItem[]>([])
const blocksLoading = ref(false)
const unblockLoadingId = ref<number | null>(null)
const blockActionUserId = ref<number | null>(null)
const blockSearchKeyword = ref('')
const blockSearchLoading = ref(false)
const blockSearchTouched = ref(false)
const blockSearchResults = ref<UserMentionItem[]>([])

function formatBlockTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

function isBlockedUser(userId: number) {
  return blockedUsers.value.some(item => item.blocked_user_id === userId)
}

async function loadBlockedUsers(silent = false) {
  blocksLoading.value = true
  try {
    const res = await listUserBlocks()
    blockedUsers.value = res.blocks || []
  } catch (error) {
    console.error('加载屏蔽列表失败:', error)
    if (!silent) {
      toast.error((error as Error)?.message || t('settings.blocks.loadFailed'))
    }
  } finally {
    blocksLoading.value = false
  }
}

async function handleSearchBlockedUsers() {
  const keyword = blockSearchKeyword.value.trim()

  if (!keyword) {
    blockSearchTouched.value = false
    blockSearchResults.value = []
    return
  }

  blockSearchTouched.value = true
  blockSearchLoading.value = true
  try {
    const res = await searchUsers(keyword, 10)
    const currentUserId = userStore.user?.id
    blockSearchResults.value = (res.users || []).filter(user => user.id !== currentUserId)
  } catch (error) {
    console.error('搜索用户失败:', error)
    toast.error((error as Error)?.message || t('settings.blocks.loadFailed'))
    blockSearchResults.value = []
  } finally {
    blockSearchLoading.value = false
  }
}

async function handleBlockUser(user: UserMentionItem) {
  if (blockActionUserId.value) return

  blockActionUserId.value = user.id
  try {
    await createUserBlock(user.id)
    toast.success(t('settings.blocks.added'))
    await loadBlockedUsers(true)
  } catch (error) {
    console.error('添加屏蔽失败:', error)
    toast.error((error as Error)?.message || t('settings.blocks.addFailed'))
  } finally {
    blockActionUserId.value = null
  }
}

async function handleRemoveBlock(blockedUserId: number) {
  if (unblockLoadingId.value) return

  unblockLoadingId.value = blockedUserId
  try {
    await deleteUserBlock(blockedUserId)
    blockedUsers.value = blockedUsers.value.filter(item => item.blocked_user_id !== blockedUserId)
    toast.success(t('settings.blocks.removed'))
  } catch (error) {
    console.error('移除屏蔽失败:', error)
    toast.error((error as Error)?.message || t('settings.blocks.removeFailed'))
  } finally {
    unblockLoadingId.value = null
  }
}

onMounted(() => {
  void loadBlockedUsers(true)
})
</script>

<template>
  <div class="blocked-users-page">
    <div class="top-toolbar">
      <div class="breadcrumbs">
        <i class="ri-forbid-2-line"></i>
        <span class="crumb-link" @click="router.push({ name: 'settings' })">{{ $t('settings.title') }}</span>
        <i class="ri-arrow-right-s-line sep"></i>
        <span class="current">{{ $t('settings.blocks.title') }}</span>
      </div>
      <div class="toolbar-actions">
        <button class="btn btn-secondary" @click="router.back()">
          <i class="ri-arrow-left-line"></i> {{ $t('settings.back') }}
        </button>
      </div>
    </div>

    <div class="setting-card">
      <div class="card-header">
        <div class="card-icon">
          <i class="ri-user-search-line"></i>
        </div>
        <div class="card-title">
          <h3>{{ $t('settings.blocks.title') }}</h3>
          <p>{{ $t('settings.blocks.description') }}</p>
        </div>
      </div>

      <div class="card-body">
        <div class="block-search-bar">
          <input
            v-model="blockSearchKeyword"
            type="text"
            class="path-input"
            :placeholder="$t('settings.blocks.searchPlaceholder')"
            @keyup.enter="handleSearchBlockedUsers"
          />
          <button class="btn btn-primary" :disabled="blockSearchLoading" @click="handleSearchBlockedUsers">
            <i :class="blockSearchLoading ? 'ri-loader-4-line spin' : 'ri-search-line'"></i>
            {{ blockSearchLoading ? $t('settings.blocks.searching') : $t('settings.blocks.searchAction') }}
          </button>
        </div>

        <div v-if="blockSearchTouched" class="block-search-results">
          <div
            v-for="user in blockSearchResults"
            :key="user.id"
            class="block-user-item"
          >
            <div class="block-user-main">
              <img v-if="user.avatar" :src="resolveApiUrl(user.avatar)" alt="" class="block-user-avatar" />
              <div v-else class="block-user-avatar placeholder">
                {{ user.username?.charAt(0)?.toUpperCase() || 'U' }}
              </div>
              <div class="block-user-meta">
                <span class="block-user-name" :style="buildNameStyle(user.name_color, user.name_bold)">
                  {{ user.username }}
                </span>
              </div>
            </div>
            <button
              class="btn"
              :class="isBlockedUser(user.id) ? 'btn-secondary' : 'btn-outline'"
              :disabled="isBlockedUser(user.id) || blockActionUserId === user.id"
              @click="handleBlockUser(user)"
            >
              <i :class="blockActionUserId === user.id ? 'ri-loader-4-line spin' : (isBlockedUser(user.id) ? 'ri-check-line' : 'ri-forbid-line')"></i>
              {{ isBlockedUser(user.id) ? $t('settings.blocks.alreadyBlocked') : $t('settings.blocks.addAction') }}
            </button>
          </div>
          <div v-if="!blockSearchLoading && blockSearchResults.length === 0" class="block-empty">
            {{ $t('settings.blocks.searchEmpty') }}
          </div>
        </div>

        <div v-if="blocksLoading" class="block-empty">
          {{ $t('settings.blocks.loading') }}
        </div>
        <div v-else-if="blockedUsers.length === 0" class="block-empty">
          {{ $t('settings.blocks.empty') }}
        </div>
        <div v-else class="block-user-list">
          <div
            v-for="block in blockedUsers"
            :key="block.id"
            class="block-user-item"
          >
            <div class="block-user-main">
              <img v-if="block.avatar" :src="resolveApiUrl(block.avatar)" alt="" class="block-user-avatar" />
              <div v-else class="block-user-avatar placeholder">
                {{ block.username?.charAt(0)?.toUpperCase() || 'U' }}
              </div>
              <div class="block-user-meta">
                <span class="block-user-name">{{ block.username }}</span>
                <span v-if="block.reason" class="block-user-subtle">{{ block.reason }}</span>
                <span class="block-user-subtle">{{ formatBlockTime(block.created_at) }}</span>
              </div>
            </div>
            <button
              class="btn btn-outline"
              :disabled="unblockLoadingId === block.blocked_user_id"
              @click="handleRemoveBlock(block.blocked_user_id)"
            >
              <i :class="unblockLoadingId === block.blocked_user_id ? 'ri-loader-4-line spin' : 'ri-user-unfollow-line'"></i>
              {{ $t('settings.blocks.removeAction') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.blocked-users-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.top-toolbar {
  background-color: var(--color-panel-bg);
  border-radius: 16px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: var(--shadow-md);
}

.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.breadcrumbs i {
  font-size: 18px;
  color: var(--icon-color);
}

.crumb-link {
  cursor: pointer;
}

.sep {
  font-size: 16px;
}

.current {
  color: var(--icon-color);
  font-weight: 600;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
}

.setting-card {
  background: var(--color-panel-bg);
  border-radius: 16px;
  padding: 24px;
  box-shadow: var(--shadow-md);
  max-width: 860px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-border);
}

.card-icon {
  width: 48px;
  height: 48px;
  background: var(--icon-bg);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon i {
  font-size: 24px;
  color: var(--icon-color);
}

.card-title h3 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main);
}

.card-title p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.block-search-bar {
  display: flex;
  gap: 12px;
}

.path-input {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid var(--input-border);
  border-radius: 10px;
  background: var(--input-bg);
  color: var(--color-text-main);
  font-size: 14px;
}

.path-input:focus {
  outline: none;
  border-color: var(--input-focus);
  box-shadow: 0 0 0 3px rgba(var(--shadow-base), 0.1);
}

.block-search-results,
.block-user-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.block-user-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 12px;
}

.block-user-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex: 1;
}

.block-user-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.block-user-avatar.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--icon-bg);
  color: var(--icon-color);
  font-weight: 700;
}

.block-user-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.block-user-name {
  color: var(--color-text-main);
  font-size: 14px;
  font-weight: 600;
}

.block-user-subtle {
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.block-empty {
  padding: 18px 12px;
  color: var(--color-text-secondary);
  font-size: 13px;
  text-align: center;
  background: var(--color-card-bg);
  border: 1px dashed var(--color-border);
  border-radius: 12px;
}

.btn {
  padding: 10px 18px;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.btn:hover {
  transform: translateY(-2px);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.btn-primary {
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
}

.btn-primary:hover:not(:disabled) {
  background: var(--btn-primary-hover);
  box-shadow: 0 4px 12px rgba(var(--shadow-base), 0.3);
}

.btn-secondary {
  background: var(--btn-secondary-bg);
  color: var(--btn-secondary-text);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--btn-secondary-hover);
}

.btn-outline {
  background: transparent;
  border: 1px solid var(--btn-outline-border);
  color: var(--btn-outline-text);
}

.btn-outline:hover:not(:disabled) {
  background: var(--btn-outline-hover);
  border-color: var(--color-border-hover);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
