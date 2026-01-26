import request from './request'

export interface Collection {
  id: number
  author_id: number
  name: string
  description: string
  cover_image?: string
  content_type: 'post' | 'item' | 'mixed'
  item_count: number
  is_public: boolean
  created_at: string
  updated_at: string
}

export interface CollectionWithAuthor extends Collection {
  author_name: string
}

export interface CollectionDetail extends CollectionWithAuthor {
  is_favorited?: boolean
  posts?: CollectionPost[]
  items?: CollectionItem[]
}

export interface CollectionPost {
  id: number
  title: string
  sort_order: number
}

export interface CollectionItem {
  id: number
  name: string
  sort_order: number
}

export interface CreateCollectionRequest {
  name: string
  description?: string
  content_type?: 'post' | 'item' | 'mixed'
  is_public?: boolean
}

export interface UpdateCollectionRequest {
  name?: string
  description?: string
  content_type?: 'post' | 'item' | 'mixed'
  is_public?: boolean
}

export interface CollectionInfo {
  collection: Collection | null
  post_ids?: number[]
  item_ids?: number[]
  current_index: number
}

// ========== 合集管理 ==========

export async function listCollections(params?: {
  author_id?: number
  content_type?: string
}): Promise<{ collections: CollectionWithAuthor[] }> {
  return request.get('/collections', { params })
}

export async function listUserCollections(contentType?: string): Promise<{ collections: Collection[] }> {
  const params = contentType ? { content_type: contentType } : undefined
  return request.get('/user/collections', { params })
}

export async function createCollection(data: CreateCollectionRequest): Promise<Collection> {
  return request.post('/collections', data)
}

export async function getCollection(id: number): Promise<CollectionDetail> {
  return request.get(`/collections/${id}`)
}

export async function updateCollection(id: number, data: UpdateCollectionRequest): Promise<Collection> {
  return request.put(`/collections/${id}`, data)
}

export async function deleteCollection(id: number): Promise<void> {
  return request.delete(`/collections/${id}`)
}

// ========== 合集内容管理 ==========

export async function getCollectionPosts(id: number): Promise<{ posts: any[] }> {
  return request.get(`/collections/${id}/posts`)
}

export async function addPostToCollection(collectionId: number, postId: number): Promise<void> {
  return request.post(`/collections/${collectionId}/posts`, { post_id: postId })
}

export async function removePostFromCollection(collectionId: number, postId: number): Promise<void> {
  return request.delete(`/collections/${collectionId}/posts/${postId}`)
}

export async function getCollectionItems(id: number): Promise<{ items: any[] }> {
  return request.get(`/collections/${id}/items`)
}

export async function addItemToCollection(collectionId: number, itemId: number): Promise<void> {
  return request.post(`/collections/${collectionId}/items`, { item_id: itemId })
}

export async function removeItemFromCollection(collectionId: number, itemId: number): Promise<void> {
  return request.delete(`/collections/${collectionId}/items/${itemId}`)
}

// ========== 获取内容所属合集 ==========

export async function getPostCollection(postId: number): Promise<CollectionInfo> {
  return request.get(`/posts/${postId}/collection`)
}

export async function getItemCollection(itemId: number): Promise<CollectionInfo> {
  return request.get(`/items/${itemId}/collection`)
}

// ========== 合集收藏 ==========

export async function favoriteCollection(id: number): Promise<void> {
  return request.post(`/collections/${id}/favorite`)
}

export async function unfavoriteCollection(id: number): Promise<void> {
  return request.delete(`/collections/${id}/favorite`)
}

export async function listMyCollectionFavorites(): Promise<{ collections: CollectionWithAuthor[] }> {
  return request.get('/user/favorite-collections')
}
