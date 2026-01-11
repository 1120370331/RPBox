// API 响应格式
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

// 认证
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: import('./types').User
}
