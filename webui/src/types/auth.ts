export interface AuthUser {
  id: number
  username: string
  is_admin: boolean
  enabled: boolean
}

export interface LoginResp {
  token: string
  expired_at: string
  user: AuthUser
}

export interface MeResp {
  user: AuthUser
}
