import request from './request'

export interface Post {
  id: number
  author_id: number
  title: string
  content: string
  content_type: string
  category: PostCategory
  guild_id?: number
  story_id?: number
  status: 'draft' | 'published'
  is_public: boolean
  is_pinned: boolean      // 置顶
  is_featured: boolean    // 精华
  cover_image?: string    // 封面图
  cover_image_updated_at?: string
  view_count: number
  like_count: number
  comment_count: number
  favorite_count: number
  event_type?: 'server' | 'guild'
  event_start_time?: string
  event_end_time?: string
  event_color?: string        // 活动标记颜色（十六进制）
  created_at: string
  updated_at: string
}

// 帖子分区
export type PostCategory = 'profile' | 'guild' | 'report' | 'novel' | 'item' | 'event' | 'other'

// 分区配置
export const POST_CATEGORIES: { value: PostCategory; label: string; icon: string }[] = [
  { value: 'profile', label: '人物卡', icon: 'ri-user-line' },
  { value: 'guild', label: '公会卡', icon: 'ri-team-line' },
  { value: 'report', label: '战报', icon: 'ri-sword-line' },
  { value: 'novel', label: '小说', icon: 'ri-book-open-line' },
  { value: 'item', label: 'TRP3道具', icon: 'ri-box-3-line' },
  { value: 'event', label: '活动', icon: 'ri-calendar-event-line' },
  { value: 'other', label: '其他', icon: 'ri-more-line' },
]

export interface PostWithAuthor extends Post {
  author_name: string
  author_avatar?: string
  author_role?: string
  author_name_color?: string
  author_name_bold?: boolean
  cover_image_url?: string  // 封面图缩略图 URL（列表页使用）
}

export interface Comment {
  id: number
  post_id: number
  author_id: number
  content: string
  parent_id?: number
  like_count: number
  created_at: string
  updated_at: string
}

export interface CommentWithAuthor extends Comment {
  author_name: string
  author_name_color?: string
  author_name_bold?: boolean
}

export interface CreatePostRequest {
  title: string
  content: string
  content_type?: string
  category?: PostCategory
  guild_id?: number
  story_id?: number
  tag_ids?: number[]
  status?: 'draft' | 'published'
  cover_image?: string
  is_public?: boolean  // 公会外成员可见（关联公会时有效）
  event_type?: 'server' | 'guild'
  event_start_time?: string
  event_end_time?: string
  event_color?: string
}

export interface UpdatePostRequest {
  title?: string
  content?: string
  content_type?: string
  category?: PostCategory
  guild_id?: number
  story_id?: number
  status?: 'draft' | 'published'
  cover_image?: string
  is_public?: boolean  // 公会外成员可见（关联公会时有效）
  event_type?: 'server' | 'guild'
  event_start_time?: string
  event_end_time?: string
  event_color?: string
}

export interface ListPostsParams {
  page?: number
  page_size?: number
  sort?: 'created_at' | 'view_count' | 'like_count'
  order?: 'asc' | 'desc'
  guild_id?: number
  tag_id?: number
  author_id?: number
  status?: 'draft' | 'published' | 'all'
  category?: PostCategory
  is_pinned?: boolean
}

// ========== 帖子管理 ==========

export async function listPosts(params?: ListPostsParams): Promise<{ posts: PostWithAuthor[]; total: number }> {
  return request.get('/posts', { params })
}

export async function createPost(data: CreatePostRequest): Promise<Post> {
  return request.post('/posts', data)
}

export async function getPost(id: number): Promise<{
  post: Post
  author_name: string
  author_name_color?: string
  author_name_bold?: boolean
  tags: any[]
  liked: boolean
  favorited: boolean
}> {
  return request.get(`/posts/${id}`)
}

export async function updatePost(id: number, data: UpdatePostRequest): Promise<Post> {
  return request.put(`/posts/${id}`, data)
}

export async function deletePost(id: number): Promise<void> {
  return request.delete(`/posts/${id}`)
}

// ========== 点赞和收藏 ==========

export async function likePost(id: number): Promise<void> {
  return request.post(`/posts/${id}/like`)
}

export async function unlikePost(id: number): Promise<void> {
  return request.delete(`/posts/${id}/like`)
}

export async function favoritePost(id: number): Promise<void> {
  return request.post(`/posts/${id}/favorite`)
}

export async function unfavoritePost(id: number): Promise<void> {
  return request.delete(`/posts/${id}/favorite`)
}

// ========== 评论管理 ==========

export async function listComments(postId: number): Promise<{ comments: CommentWithAuthor[] }> {
  return request.get(`/posts/${postId}/comments`)
}

export async function createComment(postId: number, content: string, parentId?: number): Promise<Comment> {
  return request.post(`/posts/${postId}/comments`, { content, parent_id: parentId })
}

export async function deleteComment(postId: number, commentId: number): Promise<void> {
  return request.delete(`/posts/${postId}/comments/${commentId}`)
}

export async function likeComment(commentId: number): Promise<void> {
  return request.post(`/comments/${commentId}/like`)
}

export async function unlikeComment(commentId: number): Promise<void> {
  return request.delete(`/comments/${commentId}/like`)
}

// ========== 帖子标签管理 ==========

export async function getPostTags(postId: number): Promise<{ tags: any[] }> {
  return request.get(`/posts/${postId}/tags`)
}

export async function addPostTag(postId: number, tagId: number): Promise<void> {
  return request.post(`/posts/${postId}/tags`, { tag_id: tagId })
}

export async function removePostTag(postId: number, tagId: number): Promise<void> {
  return request.delete(`/posts/${postId}/tags/${tagId}`)
}

// ========== 我的收藏 ==========

export async function listMyFavorites(): Promise<{ posts: PostWithAuthor[]; total?: number }> {
  return request.get('/posts/favorites')
}

export async function listMyPostLikes(): Promise<{ posts: PostWithAuthor[]; total?: number }> {
  return request.get('/posts/likes')
}

export async function listMyPostViews(): Promise<{ posts: PostWithAuthor[]; total?: number }> {
  return request.get('/posts/views')
}

// ========== 活动日历 ==========

export interface EventItem extends Post {
  author_name: string
  guild_name?: string
}

export async function listEvents(start?: string, end?: string): Promise<{ events: EventItem[] }> {
  const params: Record<string, string> = {}
  if (start) params.start = start
  if (end) params.end = end
  return request.get('/posts/events', { params })
}
