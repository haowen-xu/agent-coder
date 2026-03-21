import { apiRequest } from './client'
import type { LoginResp, MeResp } from '../types/auth'

export function loginApi(username: string, password: string) {
  return apiRequest<LoginResp>('/api/v1/auth/login', {
    method: 'POST',
    body: { username, password },
  })
}

export function meApi(token: string) {
  return apiRequest<MeResp>('/api/v1/auth/me', { token })
}

export type { AuthUser } from '../types/auth'
