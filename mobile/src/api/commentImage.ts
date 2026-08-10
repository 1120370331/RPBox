import { request } from '@shared/api/request'

export function uploadCommentImage(file: File) {
  const formData = new FormData()
  formData.append('image', file)
  return request.post<{ url: string }>('/upload/comment-image', formData)
}
