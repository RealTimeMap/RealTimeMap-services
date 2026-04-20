import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'

export const useAuthStore = defineStore('auth', () => {
  // TODO: auth guard — заглушка, позже заменится реальной логикой
  const isAuthenticated = true

  const tokens = useStorage<Record<string, string>>('docs-auth-tokens', {})

  function setToken(env: string, token: string) {
    tokens.value[env] = token
  }

  function getToken(env: string): string {
    return tokens.value[env] ?? ''
  }

  return { isAuthenticated, tokens, setToken, getToken }
})
