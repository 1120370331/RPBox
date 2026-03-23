const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'

export function normalizeApiOrigin(apiBase: string): string {
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    return apiBase.replace(/\/api\/v1\/?$/, '')
  }
  return ''
}

const API_ORIGIN = normalizeApiOrigin(API_BASE)

// 处理后端返回的图片 URL。
// 后端常返回 /api/v1/... 相对路径；当 API_BASE 是绝对地址时需要补上主机。
export function resolveApiUrl(url: string | undefined | null): string {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('data:')) {
    return url
  }
  if (url.startsWith('/')) {
    return API_ORIGIN ? `${API_ORIGIN}${url}` : url
  }
  return API_ORIGIN ? `${API_ORIGIN}/${url}` : url
}
