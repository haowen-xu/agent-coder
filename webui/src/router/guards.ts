import type { Pinia } from 'pinia'
import type { Router } from 'vue-router'
import { useSessionStore } from '../stores/session'

export function applyRouteGuards(router: Router, pinia: Pinia) {
  router.beforeEach(async (to) => {
    const session = useSessionStore(pinia)
    const requiresAuth = to.matched.some((record) => record.meta.requiresAuth)
    const requiresAdmin = to.matched.some((record) => record.meta.requiresAdmin)
    const guestOnly = to.matched.some((record) => record.meta.guestOnly)

    if (session.token && !session.initialized) {
      await session.ensureUser()
    }

    if (requiresAuth && !session.isLoggedIn) {
      return {
        name: 'login',
        query: to.fullPath ? { redirect: to.fullPath } : undefined,
      }
    }

    if (requiresAdmin && !session.user?.is_admin) {
      return { name: 'board' }
    }

    if (guestOnly && session.isLoggedIn) {
      return { name: 'board' }
    }

    return true
  })
}
