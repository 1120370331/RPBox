import { request } from '@shared/api/request'

export interface Post {
  id: number
  author_id: number
  title: string
  content: string
  content_type: string
  category: PostCategory
  region?: string
  address?: string
  guild_id?: number
  story_id?: number
  status: 'draft' | 'pending' | 'published'
  review_status?: 'pending' | 'approved' | 'rejected' | ''
  is_public: boolean
  is_pinned: boolean
  is_featured: boolean
  cover_image?: string
  cover_image_updated_at?: string
  view_count: number
  like_count: number
  comment_count: number
  favorite_count: number
  event_type?: 'server' | 'guild'
  event_start_time?: string
  event_end_time?: string
  event_color?: string
  created_at: string
  updated_at: string
}

export interface PostWithAuthor extends Post {
  author_name: string
  author_avatar?: string
  author_name_color?: string
  author_name_bold?: boolean
  author_forum_level?: number
  author_forum_level_name?: string
  author_forum_level_color?: string
  author_forum_level_bold?: boolean
  cover_image_url?: string
}

export type PostCategory = 'profile' | 'guild' | 'report' | 'novel' | 'item' | 'event' | 'other'

export const POST_CATEGORIES: { value: PostCategory; label: string }[] = [
  { value: 'profile', label: '人物卡' },
  { value: 'guild', label: '公会卡' },
  { value: 'report', label: '战报' },
  { value: 'novel', label: '小说' },
  { value: 'item', label: 'TRP3道具' },
  { value: 'event', label: '活动' },
  { value: 'other', label: '其他' },
]

export interface ListPostsParams {
  page?: number
  page_size?: number
  sort?: 'created_at' | 'view_count' | 'like_count'
  order?: 'asc' | 'desc'
  search?: string
  author_name?: string
  region?: string
  address?: string
  category?: PostCategory
  guild_id?: number
  author_id?: number
  tag_id?: number
  status?: 'draft' | 'published' | 'all'
  is_pinned?: boolean
}

export interface CreatePostRequest {
  title: string
  content: string
  content_type?: string
  category?: PostCategory
  region?: string
  address?: string
  guild_id?: number
  story_id?: number
  tag_ids?: number[]
  status?: 'draft' | 'published'
  cover_image?: string
  is_public?: boolean
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
  region?: string
  address?: string
  guild_id?: number
  story_id?: number
  status?: 'draft' | 'published'
  cover_image?: string
  is_public?: boolean
  event_type?: 'server' | 'guild'
  event_start_time?: string
  event_end_time?: string
  event_color?: string
}

export interface PostComment {
  id: number
  post_id: number
  author_id: number
  content: string
  parent_id?: number
  like_count: number
  created_at: string
  updated_at: string
  author_name: string
  author_avatar?: string
  author_name_color?: string
  author_name_bold?: boolean
  author_forum_level?: number
  author_forum_level_name?: string
  author_forum_level_color?: string
  author_forum_level_bold?: boolean
  liked?: boolean
}

export interface PostDetailResponse {
  post: Post
  author_name: string
  author_avatar?: string
  author_name_color?: string
  author_name_bold?: boolean
  author_forum_level?: number
  author_forum_level_name?: string
  author_forum_level_color?: string
  author_forum_level_bold?: boolean
  tags?: Array<{ id: number; name: string; color?: string }>
  liked: boolean
  favorited: boolean
}

export function listPosts(params?: ListPostsParams) {
  return request.get<{ posts: PostWithAuthor[]; total: number }>('/posts', { params })
}

export function getPost(id: number) {
  return request.get<PostDetailResponse>(`/posts/${id}`)
}

export function createPost(data: CreatePostRequest) {
  return request.post<Post>('/posts', data)
}

export function updatePost(id: number, data: UpdatePostRequest) {
  return request.put<Post>(`/posts/${id}`, data)
}

export function deletePost(id: number) {
  return request.delete<void>(`/posts/${id}`)
}

export function likePost(id: number) { return request.post(`/posts/${id}/like`) }
export function unlikePost(id: number) { return request.delete(`/posts/${id}/like`) }
export function favoritePost(id: number) { return request.post(`/posts/${id}/favorite`) }
export function unfavoritePost(id: number) { return request.delete(`/posts/${id}/favorite`) }

export function listPostComments(postId: number) {
  return request.get<{ comments: PostComment[] }>(`/posts/${postId}/comments`)
}

export function createPostComment(postId: number, content: string, parentId?: number) {
  return request.post<PostComment>(`/posts/${postId}/comments`, {
    content,
    parent_id: parentId,
  })
}
