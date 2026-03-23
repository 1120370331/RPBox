import { useUserStore } from '../stores/user'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

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

interface ApiEnvelope<T> {
  code?: number
  message?: string
  error?: string
  data?: T
}

function translateError(message: string): string {
  const lowerMessage = message.toLowerCase()
  for (const [key, value] of Object.entries(ERROR_MESSAGES)) {
    if (lowerMessage.includes(key.toLowerCase())) {
      return value
    }
  }
  return message
}

// 可由各端注入的 401 处理回调
let onUnauthorized: (() => void) | null = null

export function setUnauthorizedHandler(handler: () => void) {
  onUnauthorized = handler
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
  onUnauthorized?.()
  setTimeout(() => { isHandlingUnauthorized = false }, 500)
}

export function unwrapApiPayload<T>(payload: unknown): T {
  if (!payload || typeof payload !== 'object') {
    return payload as T
  }

  const envelope = payload as ApiEnvelope<T>
  if (typeof envelope.code === 'number') {
    if (envelope.code !== 0) {
      const message = envelope.error || envelope.message || 'Request failed'
      throw new Error(translateError(message))
    }
    return (envelope.data ?? payload) as T
  }

  return payload as T
}

async function baseRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')
  const isFormData = options.body instanceof FormData

  const headers: Record<string, string> = {
    ...(options.body && !isFormData ? { 'Content-Type': 'application/json' } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers as Record<string, string> | undefined),
  }

  if (isFormData) {
    delete headers['Content-Type']
  }

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers })

  let data: any = null
  try {
    data = await res.json()
  } catch { /* ignore */ }

  if (!res.ok) {
    if (res.status === 401) handleUnauthorized()
    const message = data?.error || data?.message || res.statusText || 'Request failed'
    throw new Error(translateError(message))
  }

  return unwrapApiPayload<T>(data)
}

function buildRequest(method: HttpMethod) {
  return async function <T>(path: string, body?: any, options: RequestInit = {}): Promise<T> {
    let finalPath = path
    const merged: RequestInit = { ...options, method }

    if (method === 'GET' && body?.params) {
      const params = new URLSearchParams()
      for (const [key, value] of Object.entries(body.params)) {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      }
      const qs = params.toString()
      if (qs) finalPath = `${path}?${qs}`
    } else if (body !== undefined && method !== 'GET') {
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
