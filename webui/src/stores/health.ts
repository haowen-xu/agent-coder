import { defineStore } from 'pinia'

interface MetaPayload {
  app: {
    name: string
    env: string
  }
  server: {
    addr: string
  }
  db: {
    enabled: boolean
    dialect: string
  }
  now: string
}

export const useHealthStore = defineStore('health', {
  state: () => ({
    loading: false,
    status: 'unknown' as string,
    db: 'unknown' as string,
    meta: null as MetaPayload | null,
    error: '' as string,
  }),
  actions: {
    async refresh() {
      this.loading = true
      this.error = ''
      try {
        const healthResp = await fetch('/healthz')
        const healthJson = (await healthResp.json()) as { status?: string; db?: string }
        this.status = healthJson.status ?? 'unknown'
        this.db = healthJson.db ?? 'unknown'

        const metaResp = await fetch('/api/v1/meta')
        this.meta = (await metaResp.json()) as MetaPayload
      } catch (e) {
        this.error = e instanceof Error ? e.message : String(e)
      } finally {
        this.loading = false
      }
    },
  },
})
