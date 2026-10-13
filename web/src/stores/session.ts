import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as authApi from '@/api/auth'
import type { CurrentUser } from '@/api/auth'
import { ApiError } from '@/api/client'

// Holds only the session/current user; server state lives in vue-query.
export const useSessionStore = defineStore('session', () => {
  const user = ref<CurrentUser | null>(null)
  const loaded = ref(false)

  async function load() {
    try {
      user.value = await authApi.me()
    } catch (e) {
      if (!(e instanceof ApiError) || (e.status !== 401 && e.status !== 403)) throw e
      user.value = null
    }
    loaded.value = true
  }

  async function login(username: string, password: string) {
    user.value = await authApi.login(username, password)
    loaded.value = true
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      user.value = null
    }
  }

  function setUser(u: CurrentUser | null) {
    user.value = u
  }

  return { user, loaded, load, login, logout, setUser }
})
