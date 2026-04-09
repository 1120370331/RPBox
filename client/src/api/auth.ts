import { request } from './request'
import type { UserData } from '@/types/user'

export interface LoginResponse {
  token: string
  user: UserData
}

export interface RegisterAgreementPayload {
  acceptTerms?: boolean
  acceptPrivacy?: boolean
  agreementVersion?: string
}

export function login(username: string, password: string) {
  return request<LoginResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}

export function register(
  username: string,
  email: string,
  password: string,
  verificationCode: string,
  agreement: RegisterAgreementPayload = {},
) {
  return request<{ message: string }>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({
      username,
      email,
      password,
      verification_code: verificationCode,
      accept_terms: agreement.acceptTerms ?? false,
      accept_privacy: agreement.acceptPrivacy ?? false,
      agreement_version: agreement.agreementVersion || '',
    }),
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
