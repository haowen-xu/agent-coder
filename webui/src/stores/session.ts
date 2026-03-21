import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { loginApi, meApi } from '../api/auth'
import type { AuthUser } from '../types/auth'

const TOKEN_KEY = 'agent_coder_token'

export const useSessionStore = defineStore('session', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) ?? '')
  const user = ref<AuthUser | null>(null)
  const loading = ref(false)
  const error = ref('')
  const initialized = ref(false)

  let ensureUserPromise: Promise<void> | null = null

  const isLoggedIn = computed(() => token.value.length > 0)

  async function login(username: string, password: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await loginApi(username, password)
      token.value = resp.token
      user.value = resp.user
      localStorage.setItem(TOKEN_KEY, token.value)
      initialized.value = true
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchMe() {
    if (!token.value) {
      initialized.value = true
      return
    }
    try {
      const resp = await meApi(token.value)
      user.value = resp.user
      error.value = ''
    } catch (err) {
      error.value = (err as Error).message
      logout()
    } finally {
      initialized.value = true
    }
  }

  async function ensureUser() {
    if (!token.value || user.value) {
      initialized.value = true
      return
    }
    if (!ensureUserPromise) {
      ensureUserPromise = fetchMe().finally(() => {
        ensureUserPromise = null
      })
    }
    await ensureUserPromise
  }

  function logout() {
    token.value = ''
    user.value = null
    initialized.value = true
    localStorage.removeItem(TOKEN_KEY)
  }

  return {
    token,
    user,
    loading,
    error,
    initialized,
    isLoggedIn,
    login,
    fetchMe,
    ensureUser,
    logout,
  }
})
