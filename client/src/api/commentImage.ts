import request from './request'

interface CommentImageUploadResponse {
  url?: string
  data?: { url?: string }
}

export async function uploadCommentImage(file: File): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append('image', file)
  const response = await request.post<CommentImageUploadResponse>('/upload/comment-image', formData)
  return { url: response.url || response.data?.url || '' }
}
