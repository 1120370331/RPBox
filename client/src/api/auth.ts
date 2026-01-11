import { request } from './request'

export interface LoginResponse {
  token: string
  user: {
    id: number
    username: string
    email: string
  }
}

export function login(username: string, password: string) {
  return request<LoginResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}

export function register(username: string, email: string, password: string) {
  return request<{ message: string }>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password }),
  })
}
