import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '@/services/api'
import type { LoginPayload, User } from '@/types'

const readUser = (): User | null => {
  try {
    const value = localStorage.getItem('shikan_user')
    return value ? (JSON.parse(value) as User) : null
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('shikan_token') ?? '')
  const user = ref<User | null>(readUser())
  const loading = ref(false)
  const isAuthenticated = computed(() => Boolean(token.value))

  async function login(payload: LoginPayload) {
    loading.value = true
    try {
      const result = await api.auth.login(payload)
      token.value = result.token
      user.value = result.user
      localStorage.setItem('shikan_token', result.token)
      localStorage.setItem('shikan_user', JSON.stringify(result.user))
    } finally {
      loading.value = false
    }
  }

  async function refreshUser() {
    if (!token.value) return
    const current = await api.auth.me()
    user.value = current
    localStorage.setItem('shikan_user', JSON.stringify(current))
  }

  function setDisplayName(name: string) {
    if (!user.value) return
    user.value = { ...user.value, name }
    localStorage.setItem('shikan_user', JSON.stringify(user.value))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('shikan_token')
    localStorage.removeItem('shikan_user')
  }

  return {
    token,
    user,
    loading,
    isAuthenticated,
    login,
    logout,
    refreshUser,
    setDisplayName,
  }
})
