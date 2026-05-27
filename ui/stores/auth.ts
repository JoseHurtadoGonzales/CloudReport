import { defineStore } from 'pinia'

export interface User {
  username: string
  shortid: string
  isAdmin: boolean
}

export interface LoginResponse {
  user: User
  token: string
}

// Persist the JWT to localStorage so refreshes keep the session.
const TOKEN_KEY = 'cr.jwt'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    token: null as string | null,
    loading: false,
    error: '' as string,
  }),
  getters: {
    isAuthenticated: (s) => !!s.token,
  },
  actions: {
    hydrate() {
      if (typeof window === 'undefined') return
      const t = localStorage.getItem(TOKEN_KEY)
      if (t) this.token = t
    },

    /** Re-fetch the current user from the token (used after a hard reload,
     *  where `user` is null but `token` was restored from localStorage). */
    async fetchCurrentUser(): Promise<void> {
      if (!this.token) return
      try {
        const cfg = useRuntimeConfig()
        const u = await $fetch<User>(`${cfg.public.apiBase}/api/current-user`, {
          headers: { Authorization: `Bearer ${this.token}` },
        })
        this.user = u
      } catch {
        // Token invalid/expired — drop it so the guard redirects to login.
        this.logout()
      }
    },

    /** Slide the session forward: swap the current token for a fresh one.
     *  Called on app load, on window focus, and on an interval so an active
     *  user is never logged out. Silent — failures just leave the old token
     *  in place (it may still be valid). */
    async refresh(): Promise<void> {
      if (!this.token) return
      try {
        const cfg = useRuntimeConfig()
        const data = await $fetch<LoginResponse>(`${cfg.public.apiBase}/api/auth/refresh`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${this.token}` },
        })
        this.user = data.user
        this.token = data.token
        if (typeof window !== 'undefined') {
          localStorage.setItem(TOKEN_KEY, data.token)
        }
      } catch (err: any) {
        // 401 → token is dead; force re-login. Network/5xx → keep current
        // token and try again on the next tick.
        if (err?.response?.status === 401) this.logout()
      }
    },

    async login(username: string, password: string): Promise<boolean> {
      this.loading = true
      this.error = ''
      try {
        const cfg = useRuntimeConfig()
        const data = await $fetch<LoginResponse>(
          `${cfg.public.apiBase}/api/auth/login`,
          {
            method: 'POST',
            body: { username, password },
            headers: { 'content-type': 'application/json' },
          },
        )
        this.user = data.user
        this.token = data.token
        if (typeof window !== 'undefined') {
          localStorage.setItem(TOKEN_KEY, data.token)
        }
        return true
      } catch (err: any) {
        // Friendly error messages
        const status = err?.response?.status
        if (status === 401) {
          this.error = 'Usuario o contraseña incorrectos.'
        } else if (status === 0 || err?.message?.includes('fetch')) {
          this.error = 'No se pudo conectar con el servidor.'
        } else {
          this.error = err?.data?.error || err?.message || 'Error de conexión.'
        }
        return false
      } finally {
        this.loading = false
      }
    },

    logout() {
      this.user = null
      this.token = null
      if (typeof window !== 'undefined') {
        localStorage.removeItem(TOKEN_KEY)
      }
    },
  },
})
