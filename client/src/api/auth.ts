import { request } from './request'

export interface LoginResponse {
  token: string
  user: {
    id: number
    username: string
    email: string
    avatar?: string
    role?: string
    is_sponsor?: boolean
    sponsor_level?: number
    sponsor_color?: string
    sponsor_bold?: boolean
    name_color?: string
    name_bold?: boolean
  }
}

export function login(username: string, password: string) {
  return request<LoginResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}

export function register(username: string, email: string, password: string, verificationCode?: string) {
  return request<{ message: string }>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password, verification_code: verificationCode }),
  })
}

export function sendVerificationCode(email: string) {
  return request<{ message: string }>('/auth/send-code', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export function forgotPassword(email: string) {
  return request<{ message: string }>('/auth/forgot-password', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export function resetPassword(email: string, verificationCode: string, newPassword: string) {
  return request<{ message: string }>('/auth/reset-password', {
    method: 'POST',
    body: JSON.stringify({ email, verification_code: verificationCode, new_password: newPassword }),
  })
}
