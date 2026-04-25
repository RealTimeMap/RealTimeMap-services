import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'
import { useEnvironmentStore } from './environment'
import { getServiceUrl } from '@/utils/env'

export const useAuthStore = defineStore('auth', () => {
  const tokens = useStorage<Record<string, string>>('docs-auth-tokens', {})
  const userInfo = ref<{ userId: string; userName: string } | null>(null)
  const checking = ref(false)

  const envStore = useEnvironmentStore()

  const isAuthenticated = computed(() => userInfo.value !== null)

  function setToken(env: string, token: string) {
    tokens.value[env] = token
  }

  function getToken(env: string): string {
    return tokens.value[env] ?? ''
  }

  function currentToken(): string {
    return getToken(envStore.current)
  }

  async function login(username: string, password: string): Promise<{ ok: boolean; error?: string }> {
    const baseUrl = getServiceUrl('auth-service', envStore.current)
    if (!baseUrl) {
      return { ok: false, error: 'URL сервиса авторизации не настроен для текущего окружения' }
    }

    checking.value = true
    try {
      const body = new URLSearchParams({ username, password })
      const res = await fetch(`${baseUrl}/api/v2/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body,
        signal: AbortSignal.timeout(10000),
      })

      if (res.ok) {
        const data = await res.json()
        const token = data.accessToken ?? data.access_token ?? ''
        if (!token) {
          return { ok: false, error: 'Сервер не вернул токен' }
        }
        tokens.value[envStore.current] = token
        // Валидируем токен, чтобы получить информацию о пользователе
        const valid = await validate()
        if (!valid) {
          // Токен получен, но валидация не прошла — всё равно считаем успехом,
          // заполним userInfo из username
          userInfo.value = { userId: '', userName: username }
        }
        return { ok: true }
      }

      if (res.status === 401 || res.status === 403) {
        return { ok: false, error: 'Неверный логин или пароль' }
      }

      return { ok: false, error: `Ошибка сервера (${res.status})` }
    } catch {
      return { ok: false, error: 'Сервис авторизации недоступен' }
    } finally {
      checking.value = false
    }
  }

  async function validate(): Promise<boolean> {
    const token = currentToken()
    if (!token) {
      userInfo.value = null
      return false
    }

    checking.value = true
    try {
      const baseUrl = getServiceUrl('auth-service', envStore.current)
      if (!baseUrl) {
        userInfo.value = null
        return false
      }

      const res = await fetch(`${baseUrl}/api/v2/auth/token-validate`, {
        method: 'GET',
        headers: { Authorization: `Bearer ${token}` },
        signal: AbortSignal.timeout(5000),
      })

      if (res.ok) {
        const userId = res.headers.get('X-User-Id')
        const userName = res.headers.get('X-User-Name')
        userInfo.value = {
          userId: userId ?? '',
          userName: userName ?? '',
        }
        return true
      }

      userInfo.value = null
      return false
    } catch {
      userInfo.value = null
      return false
    } finally {
      checking.value = false
    }
  }

  function logout() {
    tokens.value[envStore.current] = ''
    userInfo.value = null
  }

  return { tokens, userInfo, checking, isAuthenticated, setToken, getToken, currentToken, login, validate, logout }
})
