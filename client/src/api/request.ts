const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

async function baseRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')

  const headers: Record<string, string> = {
    ...(options.body ? { 'Content-Type': 'application/json' } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers as Record<string, string> | undefined),
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  // 尝试解析响应体，优先返回后端提供的错误信息
  let data: any = null
  try {
    data = await res.json()
  } catch {
    /* ignore json errors */
  }

  if (!res.ok) {
    const message = data?.error || data?.message || res.statusText || 'Request failed'
    throw new Error(message)
  }

  return data as T
}

function buildRequest(method: HttpMethod) {
  return async function <T>(path: string, body?: any, options: RequestInit = {}): Promise<T> {
    let finalPath = path
    const merged: RequestInit = {
      ...options,
      method,
    }

    // GET 请求：将 params 转换为 URL 查询字符串
    if (method === 'GET' && body?.params) {
      const params = new URLSearchParams()
      for (const [key, value] of Object.entries(body.params)) {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      }
      const queryString = params.toString()
      if (queryString) {
        finalPath = `${path}?${queryString}`
      }
    } else if (body !== undefined && method !== 'GET') {
      // 非 GET 请求：将 body 序列化为 JSON
      merged.body = typeof body === 'string' ? body : JSON.stringify(body)
    }

    return baseRequest<T>(finalPath, merged)
  }
}

const request = Object.assign(baseRequest, {
  get: buildRequest('GET'),
  post: buildRequest('POST'),
  put: buildRequest('PUT'),
  delete: buildRequest('DELETE'),
})

export { request }
export default request
