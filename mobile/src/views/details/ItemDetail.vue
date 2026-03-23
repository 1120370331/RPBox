<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { resolveApiUrl } from '@/api/image'
import {
  createItemComment,
  favoriteItem,
  getItem,
  likeItem,
  listItemComments,
  type Item,
  type ItemAuthor,
  type ItemComment,
  unfavoriteItem,
  unlikeItem,
} from '@/api/item'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const submitting = ref(false)
const item = ref<Item | null>(null)
const author = ref<ItemAuthor | null>(null)
const comments = ref<ItemComment[]>([])
const liked = ref(false)
const favorited = ref(false)
const commentText = ref('')
const rating = ref(0)

const itemId = computed(() => Number(route.params.id))
const previewUrl = computed(() => {
  if (!item.value) return ''
  if (item.value.preview_image_url) return resolveApiUrl(item.value.preview_image_url)
  if (item.value.type === 'artwork') return `/api/v1/images/item-preview/${item.value.id}?w=900&q=80`
  return `/api/v1/images/item-preview/${item.value.id}?w=900&q=80`
})

async function loadItemDetail() {
  if (!itemId.value) return
  loading.value = true
  try {
    const [itemRes, commentRes] = await Promise.all([
      getItem(itemId.value),
      listItemComments(itemId.value),
    ])
    item.value = itemRes.item
    author.value = itemRes.author
    liked.value = itemRes.liked
    favorited.value = itemRes.favorited
    comments.value = commentRes || []
  } catch (error) {
    console.error('Failed to load item detail', error)
  } finally {
    loading.value = false
  }
}

async function toggleLike() {
  if (!item.value) return
  try {
    if (liked.value) {
      await unlikeItem(item.value.id)
      liked.value = false
      item.value.like_count = Math.max(0, item.value.like_count - 1)
      return
    }
    await likeItem(item.value.id)
    liked.value = true
    item.value.like_count += 1
  } catch (error) {
    console.error('Failed to toggle item like', error)
  }
}

async function toggleFavorite() {
  if (!item.value) return
  try {
    if (favorited.value) {
      await unfavoriteItem(item.value.id)
      favorited.value = false
      return
    }
    await favoriteItem(item.value.id)
    favorited.value = true
  } catch (error) {
    console.error('Failed to toggle item favorite', error)
  }
}

async function submitComment() {
  if (!item.value || !commentText.value.trim()) return
  submitting.value = true
  try {
    await createItemComment(item.value.id, commentText.value.trim(), rating.value)
    commentText.value = ''
    rating.value = 0
    await loadItemDetail()
  } catch (error) {
    console.error('Failed to create item comment', error)
  } finally {
    submitting.value = false
  }
}

function formatTime(value: string) {
  const date = new Date(value)
  return `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

onMounted(loadItemDetail)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('market.detailTitle') }}</h1>
    </header>
    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="!item" class="hint">{{ $t('market.empty') }}</div>

      <template v-else>
        <article class="item-main">
          <img :src="previewUrl" class="preview" alt="" />
          <h2>{{ item.name }}</h2>
          <div v-if="author" class="author-row">
            <span :style="{ color: author.name_color || undefined, fontWeight: author.name_bold ? 'bold' : undefined }">
              {{ author.username }}
            </span>
            <span class="type-tag">{{ $t('market.typeBadge.' + item.type) }}</span>
          </div>
          <p>{{ item.description }}</p>
          <div class="stat-row">
            <span><i class="ri-download-line" /> {{ item.downloads }}</span>
            <span>★ {{ item.rating.toFixed(1) }} ({{ item.rating_count }})</span>
          </div>
        </article>

        <section class="action-row">
          <button @click="toggleLike">
            <i :class="liked ? 'ri-heart-fill' : 'ri-heart-line'" /> {{ item.like_count }}
          </button>
          <button @click="toggleFavorite">
            <i :class="favorited ? 'ri-star-fill' : 'ri-star-line'" />
            {{ favorited ? $t('common.action.favorited') : $t('common.action.favorite') }}
          </button>
        </section>

        <section class="comment-box">
          <h3>{{ $t('market.comments') }} ({{ comments.length }})</h3>
          <select v-model.number="rating">
            <option :value="0">{{ $t('market.ratingOptional') }}</option>
            <option :value="5">5</option>
            <option :value="4">4</option>
            <option :value="3">3</option>
            <option :value="2">2</option>
            <option :value="1">1</option>
          </select>
          <textarea v-model="commentText" :placeholder="$t('market.commentPlaceholder')" rows="3" />
          <button :disabled="submitting || !commentText.trim()" @click="submitComment">
            {{ submitting ? $t('common.action.submitting') : $t('market.submitComment') }}
          </button>
        </section>

        <section v-if="comments.length" class="comment-list">
          <article v-for="comment in comments" :key="comment.id" class="comment-item">
            <header>
              <span :style="{ color: comment.name_color || undefined, fontWeight: comment.name_bold ? 'bold' : undefined }">
                {{ comment.username }}
              </span>
              <time>{{ formatTime(comment.created_at) }}</time>
            </header>
            <p>{{ comment.content }}</p>
            <div class="rate" v-if="comment.rating > 0">★ {{ comment.rating }}</div>
          </article>
        </section>
      </template>
    </div>
  </div>
</template>

<style scoped>
.item-main {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 14px;
  margin-bottom: 12px;
}

.preview {
  width: 100%;
  height: 180px;
  border-radius: var(--radius-sm);
  object-fit: cover;
  margin-bottom: 10px;
}

.item-main h2 {
  font-size: 18px;
  margin-bottom: 8px;
}

.author-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.type-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 8px;
  background: var(--tag-bg);
  color: var(--tag-text);
}

.item-main p {
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.action-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 12px;
}

.action-row button {
  border: 1px solid var(--color-border);
  background: var(--color-card-bg);
  border-radius: var(--radius-sm);
  padding: 10px 0;
  font-size: 13px;
  cursor: pointer;
}

.comment-box {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 14px;
  margin-bottom: 12px;
}

.comment-box h3 {
  font-size: 14px;
  margin-bottom: 8px;
}

.comment-box select,
.comment-box textarea {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  margin-bottom: 8px;
  background: var(--input-bg);
}

.comment-box button {
  width: 100%;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  padding: 10px 0;
  cursor: pointer;
}

.comment-box button:disabled {
  opacity: 0.6;
  cursor: default;
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.comment-item {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 10px 12px;
}

.comment-item header {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-bottom: 6px;
}

.comment-item p {
  font-size: 14px;
  line-height: 1.6;
}

.rate {
  margin-top: 6px;
  color: var(--color-accent);
  font-size: 12px;
}
</style>
