import { request } from '@shared/api/request'

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

export function listUserCollections(contentType?: 'post' | 'item' | 'mixed') {
  const params = contentType ? { content_type: contentType } : undefined
  return request.get<{ collections: Collection[] }>('/user/collections', { params })
}

export function listCollections(params?: { author_id?: number; content_type?: string }) {
  return request.get<{ collections: CollectionWithAuthor[] }>('/collections', { params })
}

export function createCollection(data: CreateCollectionRequest) {
  return request.post<Collection>('/collections', data)
}

export function updateCollection(id: number, data: UpdateCollectionRequest) {
  return request.put<Collection>(`/collections/${id}`, data)
}

export function deleteCollection(id: number) {
  return request.delete<void>(`/collections/${id}`)
}

export function getPostCollection(postId: number) {
  return request.get<CollectionInfo>(`/posts/${postId}/collection`)
}

export function getItemCollection(itemId: number) {
  return request.get<CollectionInfo>(`/items/${itemId}/collection`)
}

export function addPostToCollection(collectionId: number, postId: number) {
  return request.post<void>(`/collections/${collectionId}/posts`, { post_id: postId })
}

export function removePostFromCollection(collectionId: number, postId: number) {
  return request.delete<void>(`/collections/${collectionId}/posts/${postId}`)
}

export function addItemToCollection(collectionId: number, itemId: number) {
  return request.post<void>(`/collections/${collectionId}/items`, { item_id: itemId })
}

export function removeItemFromCollection(collectionId: number, itemId: number) {
  return request.delete<void>(`/collections/${collectionId}/items/${itemId}`)
}
