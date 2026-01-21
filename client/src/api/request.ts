import router from '../router'
import { useUserStore } from '../stores/user'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

// 错误信息翻译映射表
const ERROR_MESSAGES: Record<string, string> = {
  'invalid credentials': '用户名或密码错误',
  'invalid username or password': '用户名或密码错误',
  'user not found': '用户不存在',
  'username already exists': '用户名已存在',
  'email already exists': '邮箱已被注册',
  'unauthorized': '未授权，请先登录',
  'forbidden': '没有权限执行此操作',
  'not found': '请求的资源不存在',
  'internal server error': '服务器内部错误',
  'bad request': '请求参数错误',
  'request failed': '请求失败',
}

// 翻译错误信息
function translateError(message: string): string {
  const lowerMessage = message.toLowerCase()
  for (const [key, value] of Object.entries(ERROR_MESSAGES)) {
    if (lowerMessage.includes(key.toLowerCase())) {
      return value
    }
  }
  return message
}

let isHandlingUnauthorized = false

function clearAuthState() {
  try {
    const userStore = useUserStore()
    userStore.logout()
  } catch {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }
}

function handleUnauthorized() {
  if (isHandlingUnauthorized) return
  isHandlingUnauthorized = true
  clearAuthState()

  const currentRoute = router.currentRoute.value
  if (currentRoute?.name === 'login') {
    isHandlingUnauthorized = false
    return
  }

  const redirect = currentRoute?.fullPath || `${window.location.pathname}${window.location.search}${window.location.hash}`
  router.replace({ name: 'login', query: { redirect } }).finally(() => {
    isHandlingUnauthorized = false
  })
}

async function baseRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')

  // 检查是否是 FormData（不需要设置 Content-Type）
  const isFormData = options.body instanceof FormData

  const headers: Record<string, string> = {
    ...(options.body && !isFormData ? { 'Content-Type': 'application/json' } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers as Record<string, string> | undefined),
  }

  // 如果是 FormData，删除 Content-Type 让浏览器自动设置
  if (isFormData) {
    delete headers['Content-Type']
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
    if (res.status === 401) {
      handleUnauthorized()
    }
    const message = data?.error || data?.message || res.statusText || 'Request failed'
    const translatedMessage = translateError(message)
    throw new Error(translatedMessage)
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
      // 非 GET 请求：FormData 直接使用，其他序列化为 JSON
      if (body instanceof FormData) {
        merged.body = body
      } else {
        merged.body = typeof body === 'string' ? body : JSON.stringify(body)
      }
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
