import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api'

interface AuthUser {
  id: number
  username: string
  is_admin: boolean
  enabled: boolean
}

interface LoginResp {
  token: string
  expired_at: string
  user: AuthUser
}

interface MeResp {
  user: AuthUser
}

const TOKEN_KEY = 'agent_coder_token'

export const useSessionStore = defineStore('session', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) ?? '')
  const user = ref<AuthUser | null>(null)
  const loading = ref(false)
  const error = ref('')

  const isLoggedIn = computed(() => token.value.length > 0)

  async function login(username: string, password: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await apiRequest<LoginResp>('/api/v1/auth/login', {
        method: 'POST',
        body: { username, password },
      })
      token.value = resp.token
      user.value = resp.user
      localStorage.setItem(TOKEN_KEY, token.value)
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchMe() {
    if (!token.value) {
      return
    }
    try {
      const resp = await apiRequest<MeResp>('/api/v1/auth/me', {
        token: token.value,
      })
      user.value = resp.user
      error.value = ''
    } catch (err) {
      error.value = (err as Error).message
      logout()
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
  }

  return {
    token,
    user,
    loading,
    error,
    isLoggedIn,
    login,
    fetchMe,
    logout,
  }
})
