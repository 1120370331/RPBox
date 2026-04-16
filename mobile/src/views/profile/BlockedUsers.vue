<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import { createUserBlock, deleteUserBlock, listUserBlocks, type UserBlockItem } from '@/api/safety'
import { resolveApiUrl } from '@/api/image'
import { searchUsers, type UserMentionItem } from '@/api/user'

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
    console.error('Failed to load blocked users', error)
    if (!silent) {
      toast.error((error as Error)?.message || t('profile.blocks.loadFailed'))
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
    console.error('Failed to search users', error)
    toast.error((error as Error)?.message || t('profile.blocks.loadFailed'))
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
    toast.success(t('profile.blocks.added'))
    await loadBlockedUsers(true)
  } catch (error) {
    console.error('Failed to block user', error)
    toast.error((error as Error)?.message || t('profile.blocks.addFailed'))
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
    toast.success(t('profile.blocks.removed'))
  } catch (error) {
    console.error('Failed to unblock user', error)
    toast.error((error as Error)?.message || t('profile.blocks.removeFailed'))
  } finally {
    unblockLoadingId.value = null
  }
}

onMounted(() => {
  void loadBlockedUsers(true)
})
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.blocks.title') }}</h1>
    </header>

    <div class="sub-body blocked-body">
      <section class="blocked-card">
        <p class="blocked-desc">{{ $t('profile.blocks.description') }}</p>

        <div class="block-search-bar">
          <input
            v-model="blockSearchKeyword"
            class="block-search-input"
            type="text"
            :placeholder="$t('profile.blocks.searchPlaceholder')"
            @keyup.enter="handleSearchBlockedUsers"
          >
          <button
            type="button"
            class="block-search-btn"
            :disabled="blockSearchLoading"
            @click="handleSearchBlockedUsers"
          >
            <i :class="blockSearchLoading ? 'ri-loader-4-line spin' : 'ri-search-line'" />
            <span>{{ blockSearchLoading ? $t('profile.blocks.searching') : $t('profile.blocks.searchAction') }}</span>
          </button>
        </div>

        <div v-if="blockSearchTouched" class="block-list search-list">
          <div v-for="user in blockSearchResults" :key="user.id" class="block-item">
            <div class="block-user-info">
              <img v-if="user.avatar" :src="resolveApiUrl(user.avatar)" class="block-avatar" alt="">
              <div v-else class="block-avatar placeholder">{{ user.username?.charAt(0)?.toUpperCase() || 'U' }}</div>
              <div class="block-user-text">
                <strong>{{ user.username }}</strong>
              </div>
            </div>
            <button
              type="button"
              class="mini-action"
              :class="{ secondary: isBlockedUser(user.id) }"
              :disabled="isBlockedUser(user.id) || blockActionUserId === user.id"
              @click="handleBlockUser(user)"
            >
              {{ isBlockedUser(user.id) ? $t('profile.blocks.alreadyBlocked') : $t('profile.blocks.addAction') }}
            </button>
          </div>
          <div v-if="!blockSearchLoading && blockSearchResults.length === 0" class="block-empty">
            {{ $t('profile.blocks.searchEmpty') }}
          </div>
        </div>

        <div v-if="blocksLoading" class="block-empty">
          {{ $t('profile.blocks.loading') }}
        </div>
        <div v-else-if="blockedUsers.length === 0" class="block-empty">
          {{ $t('profile.blocks.empty') }}
        </div>
        <div v-else class="block-list">
          <div v-for="block in blockedUsers" :key="block.id" class="block-item">
            <div class="block-user-info">
              <img v-if="block.avatar" :src="resolveApiUrl(block.avatar)" class="block-avatar" alt="">
              <div v-else class="block-avatar placeholder">{{ block.username?.charAt(0)?.toUpperCase() || 'U' }}</div>
              <div class="block-user-text">
                <strong>{{ block.username }}</strong>
                <span v-if="block.reason">{{ block.reason }}</span>
                <span>{{ formatBlockTime(block.created_at) }}</span>
              </div>
            </div>
            <button
              type="button"
              class="mini-action"
              :disabled="unblockLoadingId === block.blocked_user_id"
              @click="handleRemoveBlock(block.blocked_user_id)"
            >
              {{ $t('profile.blocks.removeAction') }}
            </button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.blocked-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.blocked-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  border: 1px solid rgba(75, 54, 33, 0.08);
  padding: 14px 12px;
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.blocked-desc {
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-secondary);
}

.block-search-bar {
  display: flex;
  gap: 8px;
}

.block-search-input {
  flex: 1;
  min-width: 0;
  border: 1px solid var(--input-border);
  border-radius: 10px;
  padding: 10px 12px;
  background: var(--input-bg);
  color: var(--color-text-main);
}

.block-search-btn,
.mini-action {
  border: none;
  border-radius: 10px;
  padding: 0 12px;
  min-height: 38px;
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}

.block-search-btn:disabled,
.mini-action:disabled {
  opacity: 0.6;
}

.mini-action {
  background: rgba(75, 54, 33, 0.12);
  color: var(--color-primary);
}

.mini-action.secondary {
  background: rgba(75, 54, 33, 0.08);
}

.block-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.search-list {
  padding-bottom: 4px;
  border-bottom: 1px solid var(--color-border-light);
}

.block-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px;
  border-radius: 12px;
  background: rgba(75, 54, 33, 0.05);
}

.block-user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.block-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.block-avatar.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--icon-bg);
  color: var(--icon-color);
  font-weight: 700;
}

.block-user-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.block-user-text strong,
.block-user-text span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.block-user-text strong {
  font-size: 13px;
  color: var(--color-text-main);
}

.block-user-text span {
  font-size: 11px;
  color: var(--color-text-secondary);
}

.block-empty {
  padding: 12px;
  border-radius: 12px;
  background: rgba(75, 54, 33, 0.04);
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
